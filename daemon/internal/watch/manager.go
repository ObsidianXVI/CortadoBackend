package watch

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/fsnotify/fsnotify"
	"github.com/your-org/cortado/daemon/internal/state"
)

const (
	DefaultDebounce        = 50 * time.Millisecond
	linuxWatchLimitWarning = "fs.inotify.max_user_watches is approaching exhaustion; increase it with: echo fs.inotify.max_user_watches=524288 | sudo tee /etc/sysctl.d/40-inotify.conf && sudo sysctl -p"
)

var defaultExcludes = []string{
	"node_modules",
	".git",
	".dart_tool",
	"build",
	"Pods",
	".gradle",
	"__pycache__",
	"*.pyc",
}

type EventType string

const (
	EventCreated  EventType = "created"
	EventModified EventType = "modified"
	EventDeleted  EventType = "deleted"
	EventRenamed  EventType = "renamed"
)

type Event struct {
	Path        string
	Type        EventType
	Checksum    string
	ModTimeUnix int64
}

type ManagerConfig struct {
	Debounce   time.Duration
	Excludes   []string
	Logger     *log.Logger
	Roots      []string
	StateStore *state.Store
}

type Manager struct {
	debounce   time.Duration
	events     chan Event
	excludes   []string
	logger     *log.Logger
	roots      []string
	stateStore *state.Store
	warnings   chan string
}

func NewManager(cfg ManagerConfig) (*Manager, error) {
	if len(cfg.Roots) == 0 {
		return nil, fmt.Errorf("watch roots must not be empty")
	}
	if cfg.StateStore == nil {
		return nil, fmt.Errorf("state store must not be nil")
	}

	debounce := cfg.Debounce
	if debounce <= 0 {
		debounce = DefaultDebounce
	}

	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}

	excludes := cfg.Excludes
	if len(excludes) == 0 {
		excludes = append([]string(nil), defaultExcludes...)
	}

	return &Manager{
		debounce:   debounce,
		events:     make(chan Event, 64),
		excludes:   excludes,
		logger:     logger,
		roots:      append([]string(nil), cfg.Roots...),
		stateStore: cfg.StateStore,
		warnings:   make(chan string, 8),
	}, nil
}

func (m *Manager) Events() <-chan Event {
	return m.events
}

func (m *Manager) Warnings() <-chan string {
	return m.warnings
}

func (m *Manager) Run(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create fsnotify watcher: %w", err)
	}
	defer watcher.Close()
	defer close(m.events)
	defer close(m.warnings)

	watchedDirs := make(map[string]struct{})
	var watchedDirsMu sync.Mutex
	watchCount := 0

	addWatch := func(path string) error {
		path = filepath.Clean(path)
		if m.shouldExclude(path) {
			return nil
		}

		watchedDirsMu.Lock()
		if _, exists := watchedDirs[path]; exists {
			watchedDirsMu.Unlock()
			return nil
		}
		watchedDirs[path] = struct{}{}
		watchedDirsMu.Unlock()

		if err := watcher.Add(path); err != nil {
			return fmt.Errorf("watch %s: %w", path, err)
		}

		watchCount++
		if warning := maybeLinuxWatchLimitWarning(watchCount); warning != "" {
			m.emitWarning(warning)
		}
		return nil
	}

	addRecursive := func(root string) error {
		return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() {
				return nil
			}
			if path != root && m.shouldExclude(path) {
				return filepath.SkipDir
			}
			return addWatch(path)
		})
	}

	for _, root := range m.roots {
		if err := addRecursive(root); err != nil {
			return fmt.Errorf("add root %s: %w", root, err)
		}
	}

	type pending struct {
		eventType EventType
		existed   bool
		timer     *time.Timer
	}
	pendingByPath := make(map[string]pending)
	var pendingMu sync.Mutex

	schedule := func(path string, eventType EventType) {
		_, _, initialExists, err := readFileChecksum(path)
		if err != nil {
			m.logger.Printf("snapshot watched path %s: %v", path, err)
			return
		}

		pendingMu.Lock()
		defer pendingMu.Unlock()

		if existing, ok := pendingByPath[path]; ok {
			existing.timer.Stop()
			if existing.existed {
				initialExists = existing.existed
			}
		}

		pendingByPath[path] = pending{
			eventType: eventType,
			existed:   initialExists,
			timer: time.AfterFunc(m.debounce, func() {
				if err := m.processPath(path, eventType, initialExists); err != nil {
					m.logger.Printf("process watched path %s: %v", path, err)
				}
				pendingMu.Lock()
				delete(pendingByPath, path)
				pendingMu.Unlock()
			}),
		}
	}

	for {
		select {
		case <-ctx.Done():
			pendingMu.Lock()
			for _, item := range pendingByPath {
				item.timer.Stop()
			}
			pendingMu.Unlock()
			return nil
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			m.logger.Printf("fsnotify error: %v", err)
		case rawEvent, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			path := filepath.Clean(rawEvent.Name)
			if m.shouldExclude(path) {
				continue
			}

			if rawEvent.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(path); err == nil && info.IsDir() {
					if err := addRecursive(path); err != nil {
						m.logger.Printf("add recursive watch for %s: %v", path, err)
					}
				}
				schedule(path, EventCreated)
				continue
			}
			if rawEvent.Op&fsnotify.Write != 0 {
				schedule(path, EventModified)
				continue
			}
			if rawEvent.Op&fsnotify.Remove != 0 {
				schedule(path, EventDeleted)
				continue
			}
			if rawEvent.Op&fsnotify.Rename != 0 {
				schedule(path, EventRenamed)
			}
		}
	}
}

func (m *Manager) processPath(
	path string,
	eventType EventType,
	initialExists bool,
) error {
	time.Sleep(10 * time.Millisecond)
	finalChecksum, info, finalExists, err := readFileChecksum(path)
	if err != nil {
		return err
	}

	if !finalExists {
		previous, found, err := m.stateStore.LookupFileState(path)
		if err != nil {
			return err
		}
		if !found && !initialExists {
			return nil
		}
		if err := m.stateStore.DeleteFileState(path); err != nil {
			return err
		}
		m.emitEvent(Event{
			Path:        path,
			Type:        coalesceDeleteType(eventType),
			Checksum:    previous.Checksum,
			ModTimeUnix: previous.ModTimeUnix,
		})
		return nil
	}

	if info == nil || info.IsDir() {
		return nil
	}

	previous, found, err := m.stateStore.LookupFileState(path)
	if err != nil {
		return err
	}
	if found && previous.Checksum == finalChecksum {
		return nil
	}

	nextState := state.FileState{
		Path:        path,
		Checksum:    finalChecksum,
		ModTimeUnix: info.ModTime().UnixMilli(),
		SyncedClock: previous.SyncedClock,
		LocalClock:  state.NextLogicalClock(previous),
		RemoteClock: previous.RemoteClock,
	}
	if err := m.stateStore.UpsertFileState(nextState); err != nil {
		return err
	}

	if eventType == EventCreated && found {
		eventType = EventModified
	}
	m.emitEvent(Event{
		Path:        path,
		Type:        eventType,
		Checksum:    finalChecksum,
		ModTimeUnix: nextState.ModTimeUnix,
	})
	return nil
}

func (m *Manager) shouldExclude(path string) bool {
	path = filepath.Clean(path)
	base := filepath.Base(path)

	for _, exclude := range m.excludes {
		if strings.HasPrefix(exclude, "*.") {
			if strings.HasSuffix(base, strings.TrimPrefix(exclude, "*")) {
				return true
			}
			continue
		}
		if base == exclude {
			return true
		}
	}

	return false
}

func (m *Manager) emitEvent(event Event) {
	select {
	case m.events <- event:
	default:
		m.logger.Printf("dropping watcher event for %s", event.Path)
	}
}

func (m *Manager) emitWarning(warning string) {
	m.logger.Print(warning)
	select {
	case m.warnings <- warning:
	default:
	}
}

func readFileChecksum(path string) (string, os.FileInfo, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, false, nil
		}
		return "", nil, false, fmt.Errorf("stat %s: %w", path, err)
	}
	if info.IsDir() {
		return "", info, true, nil
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, false, nil
		}
		return "", nil, false, fmt.Errorf("read %s: %w", path, err)
	}
	return strconv.FormatUint(xxhash.Sum64(bytes), 16), info, true, nil
}

func coalesceDeleteType(eventType EventType) EventType {
	if eventType == EventRenamed {
		return EventRenamed
	}
	return EventDeleted
}

func maybeLinuxWatchLimitWarning(watchCount int) string {
	if runtime.GOOS != "linux" {
		return ""
	}

	rawLimit, err := os.ReadFile("/proc/sys/fs/inotify/max_user_watches")
	if err != nil {
		return ""
	}

	limit, err := strconv.Atoi(strings.TrimSpace(string(rawLimit)))
	if err != nil || limit <= 0 {
		return ""
	}
	if watchCount*100 < limit*80 {
		return ""
	}

	return fmt.Sprintf(
		"fsnotify watch usage is at %d/%d (>=80%%). %s",
		watchCount,
		limit,
		linuxWatchLimitWarning,
	)
}
