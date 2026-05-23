package usage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTrackerFlushPublishesPendingRecordsAndMarksThemPublished(t *testing.T) {
	t.Parallel()

	publisher := &publisherStub{}
	tracker, err := NewTracker(context.Background(), Config{
		CPU:         2,
		MemoryGB:    4,
		ProjectID:   "cortado-ide",
		Publisher:   publisher,
		Region:      "us-central1",
		StorageGB:   10,
		TopicID:     "cortado-usage-events-dev",
		UserID:      "user-1",
		WALPath:     filepath.Join(t.TempDir(), "usage.wal"),
		WorkspaceID: "ws-123",
		TenantID:    "tenant-1",
	})
	if err != nil {
		t.Fatalf("new tracker: %v", err)
	}

	record := eventRecord{
		CPUVCPUSeconds:   20,
		DurationSeconds:  10,
		EventID:          "evt-1",
		EventTime:        time.Date(2026, time.May, 23, 13, 0, 0, 0, time.UTC),
		MemoryGBSeconds:  40,
		Published:        false,
		Region:           "us-central1",
		StorageGBSeconds: 100,
		TenantID:         "tenant-1",
		UserID:           "user-1",
		WorkspaceID:      "ws-123",
	}
	if err := tracker.appendRecord(record); err != nil {
		t.Fatalf("append WAL record: %v", err)
	}

	if err := tracker.Flush(context.Background()); err != nil {
		t.Fatalf("flush usage WAL: %v", err)
	}

	if len(publisher.payloads) != 1 {
		t.Fatalf("unexpected publish count: got %d want 1", len(publisher.payloads))
	}

	records, err := tracker.loadRecords()
	if err != nil {
		t.Fatalf("load records: %v", err)
	}
	if len(records) != 1 || !records[0].Published {
		t.Fatalf("expected record to be marked published, got %#v", records)
	}
}

func TestTrackerStartSessionPublishesOnTicker(t *testing.T) {
	t.Parallel()

	publisher := &publisherStub{}
	now := time.Date(2026, time.May, 23, 13, 0, 0, 0, time.UTC)
	tracker, err := NewTracker(context.Background(), Config{
		CPU:             1.5,
		MemoryGB:        2,
		Now:             func() time.Time { return now },
		ProjectID:       "cortado-ide",
		Publisher:       publisher,
		PublishInterval: 5 * time.Millisecond,
		Region:          "us-central1",
		StorageGB:       10,
		TopicID:         "cortado-usage-events-dev",
		UserID:          "user-1",
		WALPath:         filepath.Join(t.TempDir(), "usage.wal"),
		WorkspaceID:     "ws-123",
		TenantID:        "tenant-1",
	})
	if err != nil {
		t.Fatalf("new tracker: %v", err)
	}
	defer tracker.Close()

	tracker.StartSession("pty-1")
	time.Sleep(12 * time.Millisecond)
	tracker.EndSession("pty-1")

	if len(publisher.payloads) == 0 {
		t.Fatal("expected at least one published payload")
	}

	var published eventRecord
	if err := json.Unmarshal(publisher.payloads[0], &published); err != nil {
		t.Fatalf("decode published payload: %v", err)
	}
	if published.WorkspaceID != "ws-123" {
		t.Fatalf("unexpected workspace id: %q", published.WorkspaceID)
	}
	if published.EventID == "" {
		t.Fatal("expected published event id")
	}
}

func TestTrackerNewRecordUsesConfiguredInterval(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 23, 13, 0, 0, 0, time.UTC)
	tracker, err := NewTracker(context.Background(), Config{
		CPU:             1.5,
		MemoryGB:        2,
		Now:             func() time.Time { return now },
		ProjectID:       "cortado-ide",
		Publisher:       &publisherStub{},
		PublishInterval: 5 * time.Second,
		Region:          "us-central1",
		StorageGB:       10,
		TopicID:         "cortado-usage-events-dev",
		UserID:          "user-1",
		WALPath:         filepath.Join(t.TempDir(), "usage.wal"),
		WorkspaceID:     "ws-123",
		TenantID:        "tenant-1",
	})
	if err != nil {
		t.Fatalf("new tracker: %v", err)
	}

	record := tracker.newRecord()

	if record.DurationSeconds != 5 {
		t.Fatalf("unexpected duration seconds: got %d want 5", record.DurationSeconds)
	}
	if record.CPUVCPUSeconds != 7.5 {
		t.Fatalf("unexpected cpu seconds: got %v want 7.5", record.CPUVCPUSeconds)
	}
}

func TestNewTrackerFromEnvDisablesWhenTopicIsMissing(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "cortado-ide")
	t.Setenv("CORTADO_WORKSPACE_ID", "ws-123")

	tracker, err := NewTrackerFromEnv(context.Background())
	if err != nil {
		t.Fatalf("new tracker from env: %v", err)
	}
	if tracker.enabled {
		t.Fatal("expected tracker to be disabled without a topic")
	}
}

func TestTrackerCloseClosesPublisher(t *testing.T) {
	publisher := &publisherStub{}
	tracker, err := NewTracker(context.Background(), Config{
		ProjectID:   "cortado-ide",
		Publisher:   publisher,
		TopicID:     "cortado-usage-events-dev",
		WorkspaceID: "ws-123",
		TenantID:    "tenant-1",
		WALPath:     filepath.Join(t.TempDir(), "usage.wal"),
	})
	if err != nil {
		t.Fatalf("new tracker: %v", err)
	}

	if err := tracker.Close(); err != nil {
		t.Fatalf("close tracker: %v", err)
	}
	if !publisher.closed {
		t.Fatal("expected publisher to be closed")
	}
}

type publisherStub struct {
	closeErr error
	closed   bool
	payloads [][]byte
	pubErr   error
}

func (p *publisherStub) Close() error {
	p.closed = true
	return p.closeErr
}

func (p *publisherStub) Publish(_ context.Context, payload []byte) error {
	if p.pubErr != nil {
		return p.pubErr
	}
	p.payloads = append(p.payloads, append([]byte(nil), payload...))
	return nil
}

func TestTrackerRewriteRecordsCreatesWalDirectory(t *testing.T) {
	tracker, err := NewTracker(context.Background(), Config{
		ProjectID:   "cortado-ide",
		Publisher:   &publisherStub{},
		TopicID:     "cortado-usage-events-dev",
		WorkspaceID: "ws-123",
		TenantID:    "tenant-1",
		WALPath:     filepath.Join(t.TempDir(), "nested", "usage.wal"),
	})
	if err != nil {
		t.Fatalf("new tracker: %v", err)
	}

	if err := tracker.rewriteRecords(nil); err != nil {
		t.Fatalf("rewrite records: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(tracker.walPath)); err != nil {
		t.Fatalf("stat wal dir: %v", err)
	}
}

func TestTrackerFlushReturnsPublishError(t *testing.T) {
	publisher := &publisherStub{pubErr: errors.New("publish failed")}
	tracker, err := NewTracker(context.Background(), Config{
		ProjectID:   "cortado-ide",
		Publisher:   publisher,
		TopicID:     "cortado-usage-events-dev",
		WorkspaceID: "ws-123",
		TenantID:    "tenant-1",
		WALPath:     filepath.Join(t.TempDir(), "usage.wal"),
	})
	if err != nil {
		t.Fatalf("new tracker: %v", err)
	}
	if err := tracker.appendRecord(eventRecord{EventID: "evt-1"}); err != nil {
		t.Fatalf("append record: %v", err)
	}

	if err := tracker.Flush(context.Background()); err == nil {
		t.Fatal("expected flush to fail")
	}
}
