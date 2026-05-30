package workspace

import (
	"context"
	"errors"
	"log"
	"time"
)

const (
	defaultIdlePollInterval = 5 * time.Minute
	defaultIdleTimeout      = 20 * time.Minute
	defaultStaleTimeout     = 30 * time.Minute
)

type IdleStatus struct {
	CPUPercent     float64
	IdleDuration   time.Duration
	LastActivityAt time.Time
}

type IdleInspector interface {
	GetIdleStatus(ctx context.Context, workspaceID string) (IdleStatus, error)
}

type IdleMonitorConfig struct {
	IdleInspector IdleInspector
	IdleTimeout   time.Duration
	Logger        *log.Logger
	Now           func() time.Time
	PollInterval  time.Duration
	Service       *Service
	StaleTimeout  time.Duration
}

type IdleMonitor struct {
	idleInspector IdleInspector
	idleTimeout   time.Duration
	logger        *log.Logger
	now           func() time.Time
	pollInterval  time.Duration
	service       *Service
	staleTimeout  time.Duration
}

func NewIdleMonitor(cfg IdleMonitorConfig) *IdleMonitor {
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultIdlePollInterval
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = defaultIdleTimeout
	}
	if cfg.StaleTimeout <= 0 {
		cfg.StaleTimeout = defaultStaleTimeout
	}

	return &IdleMonitor{
		idleInspector: cfg.IdleInspector,
		idleTimeout:   cfg.IdleTimeout,
		logger:        cfg.Logger,
		now:           cfg.Now,
		pollInterval:  cfg.PollInterval,
		service:       cfg.Service,
		staleTimeout:  cfg.StaleTimeout,
	}
}

func (m *IdleMonitor) Run(ctx context.Context) {
	if m == nil || m.service == nil {
		return
	}

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	m.pollOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.pollOnce(ctx)
		}
	}
}

func (m *IdleMonitor) pollOnce(ctx context.Context) {
	now := m.now().UTC()
	stopped := make(map[string]struct{})

	if m.idleInspector != nil {
		running, err := m.service.ListWorkspacesByStatus(ctx, StatusRunning)
		if err != nil {
			m.logger.Printf("list running workspaces for idle monitor: %v", err)
		} else {
			for _, ws := range running {
				idleStatus, idleErr := m.idleInspector.GetIdleStatus(ctx, ws.ID)
				if idleErr != nil {
					if errors.Is(idleErr, ErrIdleStatusUnsupported) {
						continue
					}
					m.logger.Printf("get idle status workspace=%s: %v", ws.ID, idleErr)
					continue
				}
				if !idleStatus.LastActivityAt.IsZero() {
					if activityErr := m.service.RecordActivity(ctx, ws.ID, idleStatus.LastActivityAt); activityErr != nil {
						m.logger.Printf("record agent activity workspace=%s: %v", ws.ID, activityErr)
					}
				}
				if idleStatus.CPUPercent > 5 {
					continue
				}
				if idleStatus.IdleDuration < m.idleTimeout {
					continue
				}
				if _, stopErr := m.service.StopWorkspace(ctx, ws.TenantID, ws.ID); stopErr != nil {
					m.logger.Printf("stop idle workspace=%s: %v", ws.ID, stopErr)
					continue
				}
				stopped[ws.ID] = struct{}{}
			}
		}
	}

	staleWorkspaces, err := m.service.ListInactiveWorkspaces(ctx, now.Add(-m.staleTimeout))
	if err != nil {
		m.logger.Printf("list stale workspaces for idle monitor: %v", err)
		return
	}

	for _, ws := range staleWorkspaces {
		if _, ok := stopped[ws.ID]; ok {
			continue
		}
		switch ws.Status {
		case StatusDeleted, StatusStopped, StatusStopping:
			continue
		}
		if _, stopErr := m.service.StopWorkspace(ctx, ws.TenantID, ws.ID); stopErr != nil {
			m.logger.Printf("stop stale workspace=%s: %v", ws.ID, stopErr)
		}
	}
}
