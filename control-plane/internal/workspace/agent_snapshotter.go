package workspace

import (
	"context"
	"fmt"
	"log"
	"time"

	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const defaultCreateSnapshotTimeout = 30 * time.Second

type Snapshotter interface {
	CreateSnapshot(ctx context.Context, workspaceID string) error
}

type AgentSnapshotterConfig struct {
	Dialer            func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
	Logger            *log.Logger
	Timeout           time.Duration
	WorkspaceResolver ServiceResolver
}

type AgentSnapshotter struct {
	dialer            func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
	logger            *log.Logger
	timeout           time.Duration
	workspaceResolver ServiceResolver
}

func NewAgentSnapshotter(cfg AgentSnapshotterConfig) *AgentSnapshotter {
	if cfg.Dialer == nil {
		cfg.Dialer = grpc.NewClient
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultCreateSnapshotTimeout
	}

	return &AgentSnapshotter{
		dialer:            cfg.Dialer,
		logger:            cfg.Logger,
		timeout:           cfg.Timeout,
		workspaceResolver: cfg.WorkspaceResolver,
	}
}

func (s *AgentSnapshotter) CreateSnapshot(ctx context.Context, workspaceID string) error {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	target := fmt.Sprintf("%s:%d", s.workspaceResolver.GetServiceDNS(workspaceID), defaultAgentIdleAddressPort)
	conn, err := s.dialer(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dial workspace agent %q: %w", target, err)
	}
	defer conn.Close()

	if _, err := agentpb.NewWorkspaceAgentServiceClient(conn).CreateSnapshot(
		callCtx,
		&agentpb.CreateSnapshotRequest{},
	); err != nil {
		if status.Code(err) == codes.DeadlineExceeded {
			return context.DeadlineExceeded
		}
		return fmt.Errorf("create snapshot for workspace %q: %w", workspaceID, err)
	}

	return nil
}
