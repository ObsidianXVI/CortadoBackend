package workspace

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
)

var (
	ErrAlreadyExists = errors.New("workspace resource already exists")
	ErrConflict      = errors.New("workspace conflict")
	ErrInvalid       = errors.New("invalid workspace request")
	ErrNotFound      = errors.New("workspace not found")
	ErrTenantID      = errors.New("tenant id is required")
	ErrWorkspace     = errors.New("workspace id is required")
)

type Repository interface {
	Create(ctx context.Context, workspace Workspace) error
	Delete(ctx context.Context, workspaceID string) error
	Get(ctx context.Context, workspaceID string) (Workspace, error)
	ListByStatus(ctx context.Context, status Status) ([]Workspace, error)
	ListByTenant(ctx context.Context, tenantID string) ([]Workspace, error)
	ListInactiveSince(ctx context.Context, threshold time.Time) ([]Workspace, error)
	UpdateLastActive(ctx context.Context, workspaceID string, observedAt time.Time) (Workspace, error)
	UpdateStatus(ctx context.Context, workspaceID string, status Status, updatedAt time.Time) (Workspace, error)
}

type Provisioner interface {
	Create(workspace Workspace) error
	Delete(workspaceID string) error
	Stop(workspaceID string) error
}

type ServiceConfig struct {
	DefaultResources Resources
	IDGenerator      func() string
	Now              func() time.Time
	Provisioner      Provisioner
	Repository       Repository
}

type Service struct {
	activityMu       sync.Mutex
	lastActivitySync map[string]time.Time
	defaultResources Resources
	idGenerator      func() string
	now              func() time.Time
	provisioner      Provisioner
	repository       Repository
}

type CreateParams struct {
	Image     string
	Resources Resources
	TenantID  string
	UserID    string
}

func NewService(cfg ServiceConfig) *Service {
	if cfg.DefaultResources.CPU <= 0 {
		cfg.DefaultResources.CPU = 1
	}
	if cfg.DefaultResources.MemoryGB <= 0 {
		cfg.DefaultResources.MemoryGB = 2
	}
	if cfg.IDGenerator == nil {
		cfg.IDGenerator = func() string {
			return "ws-" + uuid.NewString()
		}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	return &Service{
		lastActivitySync: map[string]time.Time{},
		defaultResources: cfg.DefaultResources,
		idGenerator:      cfg.IDGenerator,
		now:              cfg.Now,
		provisioner:      cfg.Provisioner,
		repository:       cfg.Repository,
	}
}

func (s *Service) CreateWorkspace(ctx context.Context, params CreateParams) (Workspace, error) {
	if params.TenantID == "" {
		return Workspace{}, ErrTenantID
	}
	if params.Image == "" {
		return Workspace{}, fmt.Errorf("%w: image is required", ErrInvalid)
	}
	if params.Resources.CPU == 0 {
		params.Resources.CPU = s.defaultResources.CPU
	}
	if params.Resources.MemoryGB == 0 {
		params.Resources.MemoryGB = s.defaultResources.MemoryGB
	}
	if params.Resources.CPU < 0 || params.Resources.MemoryGB < 0 {
		return Workspace{}, fmt.Errorf("%w: workspace resources must be positive", ErrInvalid)
	}

	now := s.now().UTC()
	workspace := Workspace{
		ID:         s.idGenerator(),
		TenantID:   params.TenantID,
		UserID:     params.UserID,
		Image:      params.Image,
		Resources:  params.Resources,
		Status:     StatusCreating,
		CreatedAt:  now,
		LastActive: now,
		UpdatedAt:  now,
	}

	if err := s.repository.Create(ctx, workspace); err != nil {
		return Workspace{}, fmt.Errorf("create workspace record: %w", err)
	}

	if err := s.provisioner.Create(workspace); err != nil {
		if deleteErr := s.repository.Delete(ctx, workspace.ID); deleteErr != nil {
			return Workspace{}, fmt.Errorf("provision workspace: %w (cleanup record: %v)", err, deleteErr)
		}
		return Workspace{}, fmt.Errorf("provision workspace: %w", err)
	}

	return workspace, nil
}

func (s *Service) GetWorkspace(ctx context.Context, tenantID, workspaceID string) (Workspace, error) {
	workspace, err := s.lookupTenantWorkspace(ctx, tenantID, workspaceID)
	if err != nil {
		return Workspace{}, err
	}
	return workspace, nil
}

func (s *Service) ListWorkspaces(ctx context.Context, tenantID string) ([]Workspace, error) {
	if tenantID == "" {
		return nil, ErrTenantID
	}

	workspaces, err := s.repository.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	return workspaces, nil
}

func (s *Service) ListWorkspacesByStatus(ctx context.Context, status Status) ([]Workspace, error) {
	workspaces, err := s.repository.ListByStatus(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("list workspaces by status: %w", err)
	}
	return workspaces, nil
}

func (s *Service) ListInactiveWorkspaces(ctx context.Context, threshold time.Time) ([]Workspace, error) {
	workspaces, err := s.repository.ListInactiveSince(ctx, threshold.UTC())
	if err != nil {
		return nil, fmt.Errorf("list inactive workspaces: %w", err)
	}
	return workspaces, nil
}

func (s *Service) RecordActivity(ctx context.Context, workspaceID string, observedAt time.Time) error {
	if workspaceID == "" {
		return ErrWorkspace
	}
	if observedAt.IsZero() {
		observedAt = s.now().UTC()
	} else {
		observedAt = observedAt.UTC()
	}

	s.activityMu.Lock()
	lastObserved, ok := s.lastActivitySync[workspaceID]
	if ok && observedAt.Sub(lastObserved) < time.Minute {
		s.activityMu.Unlock()
		return nil
	}
	s.lastActivitySync[workspaceID] = observedAt
	s.activityMu.Unlock()

	if _, err := s.repository.UpdateLastActive(ctx, workspaceID, observedAt); err != nil {
		s.activityMu.Lock()
		delete(s.lastActivitySync, workspaceID)
		s.activityMu.Unlock()

		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return fmt.Errorf("update workspace activity: %w", err)
	}

	return nil
}

func (s *Service) StartWorkspace(ctx context.Context, tenantID, workspaceID string) (Workspace, error) {
	workspace, err := s.lookupTenantWorkspace(ctx, tenantID, workspaceID)
	if err != nil {
		return Workspace{}, err
	}

	switch workspace.Status {
	case StatusCreating, StatusStarting, StatusRunning:
		return workspace, nil
	case StatusDeleted:
		return Workspace{}, ErrNotFound
	case StatusStopping:
		return Workspace{}, ErrConflict
	}

	now := s.now().UTC()
	workspace, err = s.repository.UpdateStatus(ctx, workspaceID, StatusStarting, now)
	if err != nil {
		return Workspace{}, fmt.Errorf("set workspace starting: %w", err)
	}

	if err := s.provisioner.Create(workspace); err != nil {
		if _, rollbackErr := s.repository.UpdateStatus(ctx, workspaceID, StatusStopped, s.now().UTC()); rollbackErr != nil {
			return Workspace{}, fmt.Errorf("start workspace: %w (rollback status: %v)", err, rollbackErr)
		}
		return Workspace{}, fmt.Errorf("start workspace: %w", err)
	}

	return workspace, nil
}

func (s *Service) StopWorkspace(ctx context.Context, tenantID, workspaceID string) (Workspace, error) {
	workspace, err := s.lookupTenantWorkspace(ctx, tenantID, workspaceID)
	if err != nil {
		return Workspace{}, err
	}

	switch workspace.Status {
	case StatusStopped, StatusStopping, StatusDeleted:
		return workspace, nil
	}

	previousStatus := workspace.Status
	now := s.now().UTC()
	workspace, err = s.repository.UpdateStatus(ctx, workspaceID, StatusStopping, now)
	if err != nil {
		return Workspace{}, fmt.Errorf("set workspace stopping: %w", err)
	}

	if err := s.provisioner.Stop(workspaceID); err != nil {
		if _, rollbackErr := s.repository.UpdateStatus(ctx, workspaceID, previousStatus, s.now().UTC()); rollbackErr != nil {
			return Workspace{}, fmt.Errorf("stop workspace: %w (rollback status: %v)", err, rollbackErr)
		}
		return Workspace{}, fmt.Errorf("stop workspace: %w", err)
	}

	return workspace, nil
}

func (s *Service) DeleteWorkspace(ctx context.Context, tenantID, workspaceID string) (Workspace, error) {
	workspace, err := s.lookupTenantWorkspace(ctx, tenantID, workspaceID)
	if err != nil {
		return Workspace{}, err
	}
	if workspace.Status == StatusDeleted {
		return workspace, nil
	}

	previousStatus := workspace.Status
	now := s.now().UTC()
	workspace, err = s.repository.UpdateStatus(ctx, workspaceID, StatusDeleted, now)
	if err != nil {
		return Workspace{}, fmt.Errorf("set workspace deleted: %w", err)
	}

	if err := s.provisioner.Delete(workspaceID); err != nil {
		if _, rollbackErr := s.repository.UpdateStatus(ctx, workspaceID, previousStatus, s.now().UTC()); rollbackErr != nil {
			return Workspace{}, fmt.Errorf("delete workspace: %w (rollback status: %v)", err, rollbackErr)
		}
		return Workspace{}, fmt.Errorf("delete workspace: %w", err)
	}

	return workspace, nil
}

func (s *Service) OnPodDeleted(ctx context.Context, workspaceID string) error {
	workspace, err := s.repository.Get(ctx, workspaceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return fmt.Errorf("lookup workspace for deleted pod: %w", err)
	}
	if workspace.Status == StatusDeleted {
		return nil
	}

	if _, err := s.repository.UpdateStatus(ctx, workspaceID, StatusStopped, s.now().UTC()); err != nil {
		return fmt.Errorf("mark workspace stopped: %w", err)
	}
	return nil
}

func (s *Service) OnPodStatus(ctx context.Context, workspaceID string, phase corev1.PodPhase, ready bool, deleting bool) error {
	workspace, err := s.repository.Get(ctx, workspaceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return fmt.Errorf("lookup workspace for pod status: %w", err)
	}
	if workspace.Status == StatusDeleted {
		return nil
	}

	switch {
	case deleting:
		if workspace.Status == StatusStopping {
			return nil
		}
		_, err = s.repository.UpdateStatus(ctx, workspaceID, StatusStopping, s.now().UTC())
	case phase == corev1.PodRunning && ready:
		if workspace.Status == StatusRunning {
			return nil
		}
		_, err = s.repository.UpdateStatus(ctx, workspaceID, StatusRunning, s.now().UTC())
	case phase == corev1.PodFailed || phase == corev1.PodSucceeded:
		_, err = s.repository.UpdateStatus(ctx, workspaceID, StatusStopped, s.now().UTC())
	default:
		return nil
	}
	if err != nil {
		return fmt.Errorf("update workspace status from pod state: %w", err)
	}
	return nil
}

func (s *Service) lookupTenantWorkspace(ctx context.Context, tenantID, workspaceID string) (Workspace, error) {
	if tenantID == "" {
		return Workspace{}, ErrTenantID
	}
	if workspaceID == "" {
		return Workspace{}, ErrWorkspace
	}

	workspace, err := s.repository.Get(ctx, workspaceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Workspace{}, ErrNotFound
		}
		return Workspace{}, fmt.Errorf("get workspace: %w", err)
	}
	if workspace.TenantID != tenantID {
		return Workspace{}, ErrNotFound
	}
	return workspace, nil
}
