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

const defaultFlushUsageWALTimeout = 10 * time.Second

type UsageFlusher interface {
	FlushUsageWAL(ctx context.Context, workspaceID string) error
}

type AgentUsageFlusherConfig struct {
	Dialer            func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
	Logger            *log.Logger
	Timeout           time.Duration
	WorkspaceResolver ServiceResolver
}

type AgentUsageFlusher struct {
	dialer            func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
	logger            *log.Logger
	timeout           time.Duration
	workspaceResolver ServiceResolver
}

func NewAgentUsageFlusher(cfg AgentUsageFlusherConfig) *AgentUsageFlusher {
	if cfg.Dialer == nil {
		cfg.Dialer = grpc.NewClient
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultFlushUsageWALTimeout
	}

	return &AgentUsageFlusher{
		dialer:            cfg.Dialer,
		logger:            cfg.Logger,
		timeout:           cfg.Timeout,
		workspaceResolver: cfg.WorkspaceResolver,
	}
}

func (f *AgentUsageFlusher) FlushUsageWAL(ctx context.Context, workspaceID string) error {
	callCtx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	target := fmt.Sprintf("%s:%d", f.workspaceResolver.GetServiceDNS(workspaceID), defaultAgentIdleAddressPort)
	conn, err := f.dialer(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dial workspace agent %q: %w", target, err)
	}
	defer conn.Close()

	if _, err := agentpb.NewWorkspaceAgentServiceClient(conn).FlushUsageWAL(
		callCtx,
		&agentpb.FlushUsageWALRequest{},
	); err != nil {
		return fmt.Errorf("flush usage WAL for workspace %q: %w", workspaceID, err)
	}

	return nil
}
