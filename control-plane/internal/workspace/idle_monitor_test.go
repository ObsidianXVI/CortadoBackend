package workspace

import (
	"context"
	"log"
	"testing"
	"time"
)

func TestIdleMonitorStopsWorkspaceWhenAgentReportsIdle(t *testing.T) {
	now := time.Date(2026, time.May, 23, 20, 0, 0, 0, time.UTC)
	repository := newMemoryRepository(
		Workspace{
			ID:         "ws-123",
			TenantID:   "tenant-1",
			Status:     StatusRunning,
			LastActive: now.Add(-25 * time.Minute),
		},
	)
	provisioner := &provisionerStub{}
	service := NewService(ServiceConfig{
		Provisioner: provisioner,
		Repository:  repository,
		Now:         func() time.Time { return now },
	})
	monitor := NewIdleMonitor(IdleMonitorConfig{
		IdleInspector: idleInspectorStub{
			statuses: map[string]IdleStatus{
				"ws-123": {
					CPUPercent:   1,
					IdleDuration: 25 * time.Minute,
				},
			},
		},
		IdleTimeout:  20 * time.Minute,
		Logger:       log.New(ioDiscard{}, "", 0),
		Now:          func() time.Time { return now },
		PollInterval: time.Minute,
		Service:      service,
		StaleTimeout: 30 * time.Minute,
	})

	monitor.pollOnce(context.Background())

	ws, err := repository.Get(context.Background(), "ws-123")
	if err != nil {
		t.Fatalf("get workspace after idle poll: %v", err)
	}
	if ws.Status != StatusStopping {
		t.Fatalf("unexpected workspace status after idle stop: %q", ws.Status)
	}
	if provisioner.stoppedWorkspaceID != "ws-123" {
		t.Fatalf("unexpected stopped workspace id: %q", provisioner.stoppedWorkspaceID)
	}
}

func TestIdleMonitorStopsStaleWorkspaceWithoutAgentSignal(t *testing.T) {
	now := time.Date(2026, time.May, 23, 20, 10, 0, 0, time.UTC)
	repository := newMemoryRepository(
		Workspace{
			ID:         "ws-456",
			TenantID:   "tenant-2",
			Status:     StatusRunning,
			LastActive: now.Add(-35 * time.Minute),
		},
	)
	provisioner := &provisionerStub{}
	service := NewService(ServiceConfig{
		Provisioner: provisioner,
		Repository:  repository,
		Now:         func() time.Time { return now },
	})
	monitor := NewIdleMonitor(IdleMonitorConfig{
		Logger:       log.New(ioDiscard{}, "", 0),
		Now:          func() time.Time { return now },
		PollInterval: time.Minute,
		Service:      service,
		StaleTimeout: 30 * time.Minute,
	})

	monitor.pollOnce(context.Background())

	ws, err := repository.Get(context.Background(), "ws-456")
	if err != nil {
		t.Fatalf("get workspace after stale poll: %v", err)
	}
	if ws.Status != StatusStopping {
		t.Fatalf("unexpected workspace status after stale stop: %q", ws.Status)
	}
	if provisioner.stoppedWorkspaceID != "ws-456" {
		t.Fatalf("unexpected stopped workspace id: %q", provisioner.stoppedWorkspaceID)
	}
}

type idleInspectorStub struct {
	statuses map[string]IdleStatus
}

func (s idleInspectorStub) GetIdleStatus(_ context.Context, workspaceID string) (IdleStatus, error) {
	return s.statuses[workspaceID], nil
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
