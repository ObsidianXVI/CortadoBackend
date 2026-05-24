package workspace

import (
	"context"
	"fmt"
	"time"

	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultPortListTimeout = 10 * time.Second

type AgentPortServiceConfig struct {
	Dialer            func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
	Timeout           time.Duration
	WorkspaceResolver ServiceResolver
}

type AgentPortService struct {
	dialer            func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
	timeout           time.Duration
	workspaceResolver ServiceResolver
}

func NewAgentPortService(cfg AgentPortServiceConfig) *AgentPortService {
	if cfg.Dialer == nil {
		cfg.Dialer = grpc.NewClient
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultPortListTimeout
	}

	return &AgentPortService{
		dialer:            cfg.Dialer,
		timeout:           cfg.Timeout,
		workspaceResolver: cfg.WorkspaceResolver,
	}
}

func (s *AgentPortService) ListPorts(ctx context.Context, workspaceID string) ([]*agentpb.PortInfo, error) {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	target := fmt.Sprintf("%s:%d", s.workspaceResolver.GetServiceDNS(workspaceID), defaultAgentIdleAddressPort)
	conn, err := s.dialer(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial workspace agent %q: %w", target, err)
	}
	defer conn.Close()

	response, err := agentpb.NewWorkspaceAgentServiceClient(conn).ListPorts(callCtx, &agentpb.ListPortsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list ports from workspace %q: %w", workspaceID, err)
	}
	return response.GetPorts(), nil
}
