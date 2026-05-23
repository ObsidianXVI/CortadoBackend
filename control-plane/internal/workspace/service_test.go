package workspace

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
)

func TestServiceCreateWorkspacePersistsAndProvisions(t *testing.T) {
	repository := newMemoryRepository()
	provisioner := &provisionerStub{}
	now := time.Date(2026, time.May, 23, 13, 0, 0, 0, time.UTC)
	service := NewService(ServiceConfig{
		IDGenerator: func() string { return "ws-123" },
		Now:         func() time.Time { return now },
		Provisioner: provisioner,
		Repository:  repository,
	})

	ws, err := service.CreateWorkspace(context.Background(), CreateParams{
		Image:    "example.com/cortado/workspace:test",
		TenantID: "tenant-1",
		UserID:   "user-1",
	})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	if ws.Status != StatusCreating {
		t.Fatalf("unexpected create status: %q", ws.Status)
	}
	if provisioner.createdWorkspaceID != "ws-123" {
		t.Fatalf("unexpected provisioned workspace: %q", provisioner.createdWorkspaceID)
	}
	if provisioner.createdCPU != 1 || provisioner.createdMemGB != 2 {
		t.Fatalf("unexpected default resources: cpu=%v mem=%v", provisioner.createdCPU, provisioner.createdMemGB)
	}
}

func TestServiceStartWorkspaceTransitionsToStarting(t *testing.T) {
	repository := newMemoryRepository(
		Workspace{
			ID:       "ws-123",
			TenantID: "tenant-1",
			Image:    "example.com/cortado/workspace:test",
			Resources: Resources{
				CPU:      2,
				MemoryGB: 4,
			},
			Status: StatusStopped,
		},
	)
	provisioner := &provisionerStub{}
	service := NewService(ServiceConfig{
		Provisioner: provisioner,
		Repository:  repository,
		Now:         func() time.Time { return time.Date(2026, time.May, 23, 14, 0, 0, 0, time.UTC) },
	})

	ws, err := service.StartWorkspace(context.Background(), "tenant-1", "ws-123")
	if err != nil {
		t.Fatalf("start workspace: %v", err)
	}

	if ws.Status != StatusStarting {
		t.Fatalf("unexpected workspace status: %q", ws.Status)
	}
	if provisioner.createdWorkspaceID != "ws-123" {
		t.Fatalf("unexpected provisioned workspace: %q", provisioner.createdWorkspaceID)
	}
}

func TestServiceStopWorkspaceTransitionsToStopping(t *testing.T) {
	repository := newMemoryRepository(
		Workspace{
			ID:       "ws-123",
			TenantID: "tenant-1",
			Status:   StatusRunning,
		},
	)
	provisioner := &provisionerStub{}
	service := NewService(ServiceConfig{
		Provisioner: provisioner,
		Repository:  repository,
		Now:         func() time.Time { return time.Date(2026, time.May, 23, 15, 0, 0, 0, time.UTC) },
	})

	ws, err := service.StopWorkspace(context.Background(), "tenant-1", "ws-123")
	if err != nil {
		t.Fatalf("stop workspace: %v", err)
	}

	if ws.Status != StatusStopping {
		t.Fatalf("unexpected workspace status: %q", ws.Status)
	}
	if provisioner.stoppedWorkspaceID != "ws-123" {
		t.Fatalf("unexpected stopped workspace: %q", provisioner.stoppedWorkspaceID)
	}
}

func TestServiceDeleteWorkspaceTransitionsToDeleted(t *testing.T) {
	repository := newMemoryRepository(
		Workspace{
			ID:       "ws-123",
			TenantID: "tenant-1",
			Status:   StatusStopped,
		},
	)
	provisioner := &provisionerStub{}
	service := NewService(ServiceConfig{
		Provisioner: provisioner,
		Repository:  repository,
		Now:         func() time.Time { return time.Date(2026, time.May, 23, 16, 0, 0, 0, time.UTC) },
	})

	ws, err := service.DeleteWorkspace(context.Background(), "tenant-1", "ws-123")
	if err != nil {
		t.Fatalf("delete workspace: %v", err)
	}

	if ws.Status != StatusDeleted {
		t.Fatalf("unexpected workspace status: %q", ws.Status)
	}
	if provisioner.deletedWorkspaceID != "ws-123" {
		t.Fatalf("unexpected deleted workspace: %q", provisioner.deletedWorkspaceID)
	}
}

func TestServiceOnPodStatusTracksRunningAndDeleting(t *testing.T) {
	repository := newMemoryRepository(
		Workspace{
			ID:       "ws-123",
			TenantID: "tenant-1",
			Status:   StatusCreating,
		},
	)
	service := NewService(ServiceConfig{
		Provisioner: &provisionerStub{},
		Repository:  repository,
		Now:         func() time.Time { return time.Date(2026, time.May, 23, 17, 0, 0, 0, time.UTC) },
	})

	if err := service.OnPodStatus(context.Background(), "ws-123", corev1.PodRunning, false); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	ws, err := repository.Get(context.Background(), "ws-123")
	if err != nil {
		t.Fatalf("get workspace after running: %v", err)
	}
	if ws.Status != StatusRunning {
		t.Fatalf("unexpected running status: %q", ws.Status)
	}

	if err := service.OnPodStatus(context.Background(), "ws-123", corev1.PodRunning, true); err != nil {
		t.Fatalf("mark stopping: %v", err)
	}
	ws, err = repository.Get(context.Background(), "ws-123")
	if err != nil {
		t.Fatalf("get workspace after deleting: %v", err)
	}
	if ws.Status != StatusStopping {
		t.Fatalf("unexpected stopping status: %q", ws.Status)
	}

	if err := service.OnPodDeleted(context.Background(), "ws-123"); err != nil {
		t.Fatalf("mark stopped from delete: %v", err)
	}
	ws, err = repository.Get(context.Background(), "ws-123")
	if err != nil {
		t.Fatalf("get workspace after delete event: %v", err)
	}
	if ws.Status != StatusStopped {
		t.Fatalf("unexpected stopped status: %q", ws.Status)
	}
}

func TestServiceOnPodDeletedPreservesDeletedWorkspace(t *testing.T) {
	repository := newMemoryRepository(
		Workspace{
			ID:       "ws-123",
			TenantID: "tenant-1",
			Status:   StatusDeleted,
		},
	)
	service := NewService(ServiceConfig{
		Provisioner: &provisionerStub{},
		Repository:  repository,
		Now:         func() time.Time { return time.Date(2026, time.May, 23, 18, 0, 0, 0, time.UTC) },
	})

	if err := service.OnPodDeleted(context.Background(), "ws-123"); err != nil {
		t.Fatalf("handle deleted pod for deleted workspace: %v", err)
	}

	ws, err := repository.Get(context.Background(), "ws-123")
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	if ws.Status != StatusDeleted {
		t.Fatalf("unexpected workspace status: %q", ws.Status)
	}
}

type memoryRepository struct {
	workspaces map[string]Workspace
}

func newMemoryRepository(workspaces ...Workspace) *memoryRepository {
	repository := &memoryRepository{
		workspaces: make(map[string]Workspace, len(workspaces)),
	}
	for _, workspace := range workspaces {
		repository.workspaces[workspace.ID] = workspace
	}
	return repository
}

func (r *memoryRepository) Create(_ context.Context, workspace Workspace) error {
	if _, exists := r.workspaces[workspace.ID]; exists {
		return errors.New("workspace already exists")
	}
	r.workspaces[workspace.ID] = workspace
	return nil
}

func (r *memoryRepository) Delete(_ context.Context, workspaceID string) error {
	delete(r.workspaces, workspaceID)
	return nil
}

func (r *memoryRepository) Get(_ context.Context, workspaceID string) (Workspace, error) {
	workspace, ok := r.workspaces[workspaceID]
	if !ok {
		return Workspace{}, ErrNotFound
	}
	return workspace, nil
}

func (r *memoryRepository) ListByTenant(_ context.Context, tenantID string) ([]Workspace, error) {
	result := make([]Workspace, 0)
	for _, workspace := range r.workspaces {
		if workspace.TenantID == tenantID {
			result = append(result, workspace)
		}
	}
	return result, nil
}

func (r *memoryRepository) UpdateStatus(_ context.Context, workspaceID string, status Status, updatedAt time.Time) (Workspace, error) {
	workspace, ok := r.workspaces[workspaceID]
	if !ok {
		return Workspace{}, ErrNotFound
	}
	workspace.Status = status
	workspace.UpdatedAt = updatedAt
	r.workspaces[workspaceID] = workspace
	return workspace, nil
}

type provisionerStub struct {
	createErr          error
	deleteErr          error
	stopErr            error
	createdCPU         float64
	createdMemGB       float64
	createdWorkspaceID string
	deletedWorkspaceID string
	stoppedWorkspaceID string
}

func (p *provisionerStub) Create(workspaceID, _ string, cpu, memGB float64) error {
	p.createdWorkspaceID = workspaceID
	p.createdCPU = cpu
	p.createdMemGB = memGB
	return p.createErr
}

func (p *provisionerStub) Delete(workspaceID string) error {
	p.deletedWorkspaceID = workspaceID
	return p.deleteErr
}

func (p *provisionerStub) Stop(workspaceID string) error {
	p.stoppedWorkspaceID = workspaceID
	return p.stopErr
}
