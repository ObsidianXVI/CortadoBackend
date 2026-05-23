package usage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
)

const (
	defaultPublishInterval = 10 * time.Second
	defaultWALPath         = "/workspace/.cortado/usage.wal"
)

type Publisher interface {
	Close() error
	Publish(ctx context.Context, payload []byte) error
}

type Config struct {
	CPU             float64
	MemoryGB        float64
	Now             func() time.Time
	ProjectID       string
	Publisher       Publisher
	PublishInterval time.Duration
	Region          string
	StorageGB       float64
	TopicID         string
	UserID          string
	WALPath         string
	WorkspaceID     string
	TenantID        string
}

type Tracker struct {
	activeMu        sync.Mutex
	activeSessions  map[string]struct{}
	cancelLoop      context.CancelFunc
	enabled         bool
	metadata        eventRecord
	now             func() time.Time
	publisher       Publisher
	publishInterval time.Duration
	publishMu       sync.Mutex
	walPath         string
}

type eventRecord struct {
	Published        bool      `json:"published"`
	CPUVCPUSeconds   float64   `json:"cpu_vcpu_seconds"`
	DurationSeconds  int64     `json:"duration_seconds"`
	EventID          string    `json:"event_id"`
	EventTime        time.Time `json:"event_time"`
	GPUSeconds       float64   `json:"gpu_seconds"`
	MemoryGBSeconds  float64   `json:"memory_gb_seconds"`
	Region           string    `json:"region,omitempty"`
	StorageGBSeconds float64   `json:"storage_gb_seconds"`
	TenantID         string    `json:"tenant_id"`
	UserID           string    `json:"user_id,omitempty"`
	WorkspaceID      string    `json:"workspace_id"`
}

func NewTracker(ctx context.Context, cfg Config) (*Tracker, error) {
	if cfg.PublishInterval <= 0 {
		cfg.PublishInterval = defaultPublishInterval
	}
	if cfg.WALPath == "" {
		cfg.WALPath = defaultWALPath
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	enabled := cfg.WorkspaceID != "" && cfg.ProjectID != "" && cfg.TopicID != ""
	if !enabled {
		return &Tracker{
			activeSessions:  map[string]struct{}{},
			enabled:         false,
			now:             cfg.Now,
			publishInterval: cfg.PublishInterval,
			walPath:         cfg.WALPath,
		}, nil
	}

	publisher := cfg.Publisher
	if publisher == nil {
		var err error
		publisher, err = newPubSubPublisher(ctx, cfg.ProjectID, cfg.TopicID)
		if err != nil {
			return nil, fmt.Errorf("create usage publisher: %w", err)
		}
	}

	return &Tracker{
		activeSessions: map[string]struct{}{},
		enabled:        true,
		metadata: eventRecord{
			CPUVCPUSeconds:   cfg.CPU,
			MemoryGBSeconds:  cfg.MemoryGB,
			Region:           cfg.Region,
			StorageGBSeconds: cfg.StorageGB,
			TenantID:         cfg.TenantID,
			UserID:           cfg.UserID,
			WorkspaceID:      cfg.WorkspaceID,
		},
		now:             cfg.Now,
		publisher:       publisher,
		publishInterval: cfg.PublishInterval,
		walPath:         cfg.WALPath,
	}, nil
}

func NewTrackerFromEnv(ctx context.Context) (*Tracker, error) {
	projectID := firstNonEmpty(
		os.Getenv("GOOGLE_CLOUD_PROJECT"),
		os.Getenv("GCP_PROJECT"),
		os.Getenv("GCLOUD_PROJECT"),
	)
	cfg := Config{
		CPU:             parseEnvFloat("CORTADO_WORKSPACE_CPU"),
		MemoryGB:        parseEnvFloat("CORTADO_WORKSPACE_MEMORY_GB"),
		ProjectID:       projectID,
		PublishInterval: parseEnvDuration("CORTADO_USAGE_PUBLISH_INTERVAL_SECONDS"),
		Region:          os.Getenv("CORTADO_GCP_REGION"),
		StorageGB:       parseEnvFloat("CORTADO_WORKSPACE_STORAGE_GB"),
		TopicID:         os.Getenv("CORTADO_USAGE_EVENTS_TOPIC"),
		UserID:          os.Getenv("CORTADO_USER_ID"),
		WALPath:         os.Getenv("CORTADO_USAGE_WAL_PATH"),
		WorkspaceID:     os.Getenv("CORTADO_WORKSPACE_ID"),
		TenantID:        os.Getenv("CORTADO_TENANT_ID"),
	}

	return NewTracker(ctx, cfg)
}

func (t *Tracker) Close() error {
	if t.cancelLoop != nil {
		t.cancelLoop()
	}
	if t.publisher == nil {
		return nil
	}
	return t.publisher.Close()
}

func (t *Tracker) Flush(ctx context.Context) error {
	if !t.enabled {
		return nil
	}
	return t.flushPending(ctx)
}

func (t *Tracker) ReplayPending(ctx context.Context) error {
	if !t.enabled {
		return nil
	}
	return t.flushPending(ctx)
}

func (t *Tracker) StartSession(sessionID string) {
	if !t.enabled || strings.TrimSpace(sessionID) == "" {
		return
	}

	t.activeMu.Lock()
	defer t.activeMu.Unlock()

	if _, exists := t.activeSessions[sessionID]; exists {
		return
	}

	t.activeSessions[sessionID] = struct{}{}
	if len(t.activeSessions) == 1 {
		t.startTickerLoopLocked()
	}
}

func (t *Tracker) EndSession(sessionID string) {
	if !t.enabled || strings.TrimSpace(sessionID) == "" {
		return
	}

	t.activeMu.Lock()
	defer t.activeMu.Unlock()

	delete(t.activeSessions, sessionID)
	if len(t.activeSessions) == 0 && t.cancelLoop != nil {
		t.cancelLoop()
		t.cancelLoop = nil
	}
}

func (t *Tracker) startTickerLoopLocked() {
	if t.cancelLoop != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.cancelLoop = cancel

	go func() {
		ticker := time.NewTicker(t.publishInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := t.publishTick(ctx); err != nil && !errors.Is(err, context.Canceled) {
					log.Printf("publish usage event: %v", err)
				}
			}
		}
	}()
}

func (t *Tracker) publishTick(ctx context.Context) error {
	record := t.newRecord()
	if err := t.appendRecord(record); err != nil {
		return fmt.Errorf("append usage WAL record: %w", err)
	}

	if err := t.publishRecord(ctx, record); err != nil {
		return err
	}

	return t.markPublished(record.EventID)
}

func (t *Tracker) publishRecord(ctx context.Context, record eventRecord) error {
	t.publishMu.Lock()
	defer t.publishMu.Unlock()

	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal usage event: %w", err)
	}

	if err := t.publisher.Publish(ctx, payload); err != nil {
		return fmt.Errorf("publish usage event %q: %w", record.EventID, err)
	}
	return nil
}

func (t *Tracker) flushPending(ctx context.Context) error {
	records, err := t.loadRecords()
	if err != nil {
		return fmt.Errorf("load usage WAL: %w", err)
	}

	for _, record := range records {
		if record.Published {
			continue
		}
		if err := t.publishRecord(ctx, record); err != nil {
			return err
		}
		if err := t.markPublished(record.EventID); err != nil {
			return err
		}
	}

	return nil
}

func (t *Tracker) newRecord() eventRecord {
	durationSeconds := int64(t.publishInterval / time.Second)
	return eventRecord{
		CPUVCPUSeconds:   t.metadata.CPUVCPUSeconds * float64(durationSeconds),
		DurationSeconds:  durationSeconds,
		EventID:          uuid.NewString(),
		EventTime:        t.now().UTC(),
		GPUSeconds:       0,
		MemoryGBSeconds:  t.metadata.MemoryGBSeconds * float64(durationSeconds),
		Published:        false,
		Region:           t.metadata.Region,
		StorageGBSeconds: t.metadata.StorageGBSeconds * float64(durationSeconds),
		TenantID:         t.metadata.TenantID,
		UserID:           t.metadata.UserID,
		WorkspaceID:      t.metadata.WorkspaceID,
	}
}

func (t *Tracker) appendRecord(record eventRecord) error {
	if err := os.MkdirAll(filepath.Dir(t.walPath), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(t.walPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(record)
}

func (t *Tracker) loadRecords() ([]eventRecord, error) {
	file, err := os.Open(t.walPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	records := []eventRecord{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var record eventRecord
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, fmt.Errorf("decode usage WAL record: %w", err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (t *Tracker) markPublished(eventID string) error {
	records, err := t.loadRecords()
	if err != nil {
		return fmt.Errorf("load usage WAL for publish mark: %w", err)
	}

	updated := false
	for index := range records {
		if records[index].EventID != eventID {
			continue
		}
		records[index].Published = true
		updated = true
		break
	}
	if !updated {
		return nil
	}

	return t.rewriteRecords(records)
}

func (t *Tracker) rewriteRecords(records []eventRecord) error {
	if err := os.MkdirAll(filepath.Dir(t.walPath), 0o755); err != nil {
		return err
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return err
		}
	}

	tmpPath := t.walPath + ".tmp"
	if err := os.WriteFile(tmpPath, buffer.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, t.walPath)
}

type pubSubPublisher struct {
	client *pubsub.Client
	topic  *pubsub.Topic
}

func newPubSubPublisher(ctx context.Context, projectID, topicID string) (*pubSubPublisher, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &pubSubPublisher{
		client: client,
		topic:  client.Topic(topicID),
	}, nil
}

func (p *pubSubPublisher) Close() error {
	if p.topic != nil {
		p.topic.Stop()
	}
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

func (p *pubSubPublisher) Publish(ctx context.Context, payload []byte) error {
	result := p.topic.Publish(ctx, &pubsub.Message{Data: payload})
	if _, err := result.Get(ctx); err != nil {
		return err
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseEnvDuration(name string) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func parseEnvFloat(name string) float64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}
