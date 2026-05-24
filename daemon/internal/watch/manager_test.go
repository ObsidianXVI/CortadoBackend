package watch_test

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/your-org/cortado/daemon/internal/state"
	"github.com/your-org/cortado/daemon/internal/watch"
)

func TestManagerEmitsStableModifiedEventAndUpdatesState(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.dart")
	if err := os.WriteFile(path, []byte("first"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	store := mustOpenStore(t)
	defer store.Close()

	manager, err := watch.NewManager(watch.ManagerConfig{
		Debounce:   50 * time.Millisecond,
		Logger:     log.New(io.Discard, "", 0),
		Roots:      []string{root},
		StateStore: store,
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- manager.Run(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(path, []byte("second"), 0o644); err != nil {
		t.Fatalf("write updated file: %v", err)
	}

	select {
	case event := <-manager.Events():
		if event.Path != path {
			t.Fatalf("unexpected event path: got %q want %q", event.Path, path)
		}
		if event.Type != watch.EventModified {
			t.Fatalf("unexpected event type: got %q want %q", event.Type, watch.EventModified)
		}
		if event.Checksum == "" {
			t.Fatal("expected checksum to be populated")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for watcher event")
	}

	fileState, found, err := store.LookupFileState(path)
	if err != nil {
		t.Fatalf("lookup file state: %v", err)
	}
	if !found {
		t.Fatal("expected file state to be recorded")
	}
	if fileState.Checksum == "" {
		t.Fatal("expected non-empty persisted checksum")
	}
	if fileState.LocalClock != 1 {
		t.Fatalf("unexpected local clock: got %d want %d", fileState.LocalClock, 1)
	}
	if fileState.RemoteClock != 0 || fileState.SyncedClock != 0 {
		t.Fatalf("unexpected remote/synced clocks: %#v", fileState)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run watcher: %v", err)
	}
}

func TestManagerSkipsExcludedPaths(t *testing.T) {
	root := t.TempDir()
	excludedDir := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(excludedDir, 0o755); err != nil {
		t.Fatalf("mkdir excluded dir: %v", err)
	}

	store := mustOpenStore(t)
	defer store.Close()

	manager, err := watch.NewManager(watch.ManagerConfig{
		Debounce:   50 * time.Millisecond,
		Logger:     log.New(io.Discard, "", 0),
		Roots:      []string{root},
		StateStore: store,
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- manager.Run(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(excludedDir, "ignored.js"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write excluded file: %v", err)
	}

	select {
	case event := <-manager.Events():
		t.Fatalf("unexpected event for excluded path: %#v", event)
	case <-time.After(300 * time.Millisecond):
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run watcher: %v", err)
	}
}

func mustOpenStore(t *testing.T) *state.Store {
	t.Helper()

	store, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}
