package workspace

import (
	"context"
	"fmt"
	"log"
	"time"

	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultAgentIdleAddressPort = 9090
	defaultIdleStatusTimeout    = 10 * time.Second
)

type ServiceResolver interface {
	GetServiceDNS(workspaceID string) string
}

type AgentIdleInspectorConfig struct {
	Dialer            func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
	Logger            *log.Logger
	Now               func() time.Time
	Timeout           time.Duration
	WorkspaceResolver ServiceResolver
}

type AgentIdleInspector struct {
	dialer            func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
	logger            *log.Logger
	now               func() time.Time
	timeout           time.Duration
	workspaceResolver ServiceResolver
}

func NewAgentIdleInspector(cfg AgentIdleInspectorConfig) *AgentIdleInspector {
	if cfg.Dialer == nil {
		cfg.Dialer = grpc.NewClient
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultIdleStatusTimeout
	}

	return &AgentIdleInspector{
		dialer:            cfg.Dialer,
		logger:            cfg.Logger,
		now:               cfg.Now,
		timeout:           cfg.Timeout,
		workspaceResolver: cfg.WorkspaceResolver,
	}
}

func (i *AgentIdleInspector) GetIdleStatus(ctx context.Context, workspaceID string) (IdleStatus, error) {
	callCtx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	target := fmt.Sprintf("%s:%d", i.workspaceResolver.GetServiceDNS(workspaceID), defaultAgentIdleAddressPort)
	conn, err := i.dialer(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return IdleStatus{}, fmt.Errorf("dial workspace agent %q: %w", target, err)
	}
	defer conn.Close()

	response, err := agentpb.NewWorkspaceAgentServiceClient(conn).GetIdleStatus(callCtx, &agentpb.GetIdleStatusRequest{})
	if err != nil {
		return IdleStatus{}, fmt.Errorf("get idle status from workspace %q: %w", workspaceID, err)
	}

	idleStatus := IdleStatus{
		CPUPercent: response.GetCpuPercentOverSixtySeconds(),
	}
	if lastActivity := response.GetLastPtyActivityTime(); lastActivity != nil {
		idleStatus.LastActivityAt = lastActivity.AsTime().UTC()
		idleStatus.IdleDuration = i.now().UTC().Sub(idleStatus.LastActivityAt)
	}

	return idleStatus, nil
}
