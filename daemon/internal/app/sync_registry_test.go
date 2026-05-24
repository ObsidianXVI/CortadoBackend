package app

import (
	"testing"

	"github.com/your-org/cortado/daemon/internal/filesync"
)

func TestSyncRegistryTracksLifecycleAndConflicts(t *testing.T) {
	registry := NewSyncRegistry()

	started, err := registry.StartSync("/tmp/workspace", "ws-123")
	if err != nil {
		t.Fatalf("start sync: %v", err)
	}
	if started.State != SyncStateSyncing {
		t.Fatalf("unexpected start state: got %q want %q", started.State, SyncStateSyncing)
	}
	if started.WorkspacePath != "/" {
		t.Fatalf("unexpected start workspace path: got %q want %q", started.WorkspacePath, "/")
	}

	registry.MarkConflict(filesync.ConflictNotice{
		Path:   "/tmp/workspace/lib/main.dart",
		Reason: "manual merge required",
	})

	conflicted, err := registry.GetSyncStatus("/tmp/workspace", "ws-123")
	if err != nil {
		t.Fatalf("get conflicted sync status: %v", err)
	}
	if conflicted.State != SyncStateConflicted {
		t.Fatalf("unexpected conflict state: got %q want %q", conflicted.State, SyncStateConflicted)
	}
	if conflicted.WorkspacePath != "/lib/main.dart" {
		t.Fatalf("unexpected conflict workspace path: got %q want %q", conflicted.WorkspacePath, "/lib/main.dart")
	}
	if conflicted.Message != "manual merge required" {
		t.Fatalf("unexpected conflict message: got %q", conflicted.Message)
	}

	stopped, err := registry.StopSync("/tmp/workspace", "ws-123")
	if err != nil {
		t.Fatalf("stop sync: %v", err)
	}
	if stopped.State != SyncStateIdle {
		t.Fatalf("unexpected stop state: got %q want %q", stopped.State, SyncStateIdle)
	}
	if stopped.Message != "" {
		t.Fatalf("expected stop to clear conflict message, got %q", stopped.Message)
	}
}
