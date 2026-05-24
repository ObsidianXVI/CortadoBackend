package server

import (
	"context"
	"errors"
	"io"
	"math"
	"strings"
	"syscall"
	"time"

	pb "github.com/your-org/cortado/agent/gen/agent/v1"
	ptymanager "github.com/your-org/cortado/agent/internal/pty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultWorkspaceRoot = "/workspace"

type AgentServer struct {
	pb.UnimplementedWorkspaceAgentServiceServer

	commandRunner    snapshotCommandRunner
	ptyMgr           *ptymanager.Manager
	snapshotBucket   string
	snapshotPassword string
	usageTracker     usageTracker
	workspaceID      string
	workspaceRoot    string
}

type AgentServerConfig struct {
	CommandRunner    snapshotCommandRunner
	SnapshotBucket   string
	SnapshotPassword string
	WorkspaceID      string
	WorkspaceRoot    string
}

type usageTracker interface {
	EndSession(sessionID string)
	Flush(ctx context.Context) error
	StartSession(sessionID string)
}

func NewAgentServer(ptyMgr *ptymanager.Manager, tracker usageTracker) *AgentServer {
	return NewAgentServerWithConfig(ptyMgr, tracker, AgentServerConfig{})
}

func NewAgentServerWithWorkspaceRoot(ptyMgr *ptymanager.Manager, tracker usageTracker, workspaceRoot string) *AgentServer {
	return NewAgentServerWithConfig(ptyMgr, tracker, AgentServerConfig{WorkspaceRoot: workspaceRoot})
}

func NewAgentServerWithConfig(ptyMgr *ptymanager.Manager, tracker usageTracker, cfg AgentServerConfig) *AgentServer {
	if ptyMgr == nil {
		ptyMgr = &ptymanager.Manager{}
	}
	if strings.TrimSpace(cfg.WorkspaceRoot) == "" {
		cfg.WorkspaceRoot = defaultWorkspaceRoot
	}
	if cfg.CommandRunner == nil {
		cfg.CommandRunner = runSnapshotCommand
	}

	return &AgentServer{
		commandRunner:    cfg.CommandRunner,
		ptyMgr:           ptyMgr,
		snapshotBucket:   strings.TrimSpace(cfg.SnapshotBucket),
		snapshotPassword: cfg.SnapshotPassword,
		usageTracker:     tracker,
		workspaceID:      strings.TrimSpace(cfg.WorkspaceID),
		workspaceRoot:    cfg.WorkspaceRoot,
	}
}

func (s *AgentServer) CreatePty(ctx context.Context, req *pb.CreatePtyRequest) (*pb.CreatePtyResponse, error) {
	if req.GetCols() == 0 || req.GetRows() == 0 {
		return nil, status.Error(codes.InvalidArgument, "cols and rows must be greater than zero")
	}
	if req.GetCols() > math.MaxUint16 || req.GetRows() > math.MaxUint16 {
		return nil, status.Error(codes.InvalidArgument, "cols and rows exceed PTY limits")
	}

	session, err := s.ptyMgr.Create(req.GetShell(), uint16(req.GetCols()), uint16(req.GetRows()), req.GetEnv())
	if err != nil {
		if strings.Contains(err.Error(), "not found in image") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "create pty: %v", err)
	}

	s.trackSessionLifetime(session.ID)

	return &pb.CreatePtyResponse{PtyId: session.ID}, nil
}

func (s *AgentServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Status: "ok"}, nil
}

func (s *AgentServer) GetIdleStatus(ctx context.Context, req *pb.GetIdleStatusRequest) (*pb.GetIdleStatusResponse, error) {
	lastActivity, cpuPercent, err := s.ptyMgr.IdleStatus(time.Now())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get idle status: %v", err)
	}

	response := &pb.GetIdleStatusResponse{
		CpuPercentOverSixtySeconds: cpuPercent,
	}
	if !lastActivity.IsZero() {
		response.LastPtyActivityTime = timestamppb.New(lastActivity)
	}

	return response, nil
}

func (s *AgentServer) FlushUsageWAL(ctx context.Context, req *pb.FlushUsageWALRequest) (*pb.FlushUsageWALResponse, error) {
	if s.usageTracker != nil {
		if err := s.usageTracker.Flush(ctx); err != nil {
			return nil, status.Errorf(codes.Internal, "flush usage WAL: %v", err)
		}
	}

	return &pb.FlushUsageWALResponse{}, nil
}

func (s *AgentServer) StreamPty(stream pb.WorkspaceAgentService_StreamPtyServer) error {
	first, err := stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return status.Error(codes.InvalidArgument, "first stream message must include pty_id")
		}
		return err
	}

	sessionID := strings.TrimSpace(first.GetPtyId())
	if sessionID == "" {
		return status.Error(codes.InvalidArgument, "pty_id is required")
	}
	if first.GetPayload() != nil {
		return status.Error(codes.InvalidArgument, "first stream message must only identify the PTY session")
	}

	exitCh, err := s.ptyMgr.OnExit(sessionID)
	if err != nil {
		return mapManagerError(err)
	}

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	sendCh := make(chan *pb.StreamPtyResponse, 16)
	errCh := make(chan error, 2)

	go s.pipePTYToStream(ctx, sessionID, sendCh, errCh)
	go s.pipeStreamToPTY(ctx, stream, sessionID, errCh)

	for {
		select {
		case <-ctx.Done():
			if err := stream.Context().Err(); err != nil && status.Code(err) == codes.Canceled {
				return nil
			}
			return nil
		case msg := <-sendCh:
			if msg == nil {
				continue
			}
			if err := stream.Send(msg); err != nil {
				return err
			}
		case exitCode, ok := <-exitCh:
			if !ok {
				exitCh = nil
				continue
			}
			if err := stream.Send(&pb.StreamPtyResponse{
				Payload: &pb.StreamPtyResponse_ExitCode{ExitCode: exitCode},
			}); err != nil {
				return err
			}
			return nil
		case err := <-errCh:
			if err == nil {
				return nil
			}
			return err
		}
	}
}

func (s *AgentServer) pipePTYToStream(ctx context.Context, sessionID string, sendCh chan<- *pb.StreamPtyResponse, errCh chan<- error) {
	buf := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := s.ptyMgr.Read(sessionID, buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			select {
			case sendCh <- &pb.StreamPtyResponse{
				Payload: &pb.StreamPtyResponse_Data{Data: data},
			}:
			case <-ctx.Done():
				return
			}
		}

		if err != nil {
			if errors.Is(err, syscall.EIO) || errors.Is(err, ptymanager.ErrSessionNotFound) {
				return
			}
			select {
			case errCh <- status.Errorf(codes.Internal, "read PTY: %v", err):
			case <-ctx.Done():
			}
			return
		}
	}
}

func (s *AgentServer) pipeStreamToPTY(ctx context.Context, stream pb.WorkspaceAgentService_StreamPtyServer, sessionID string, errCh chan<- error) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				s.ptyMgr.Kill(sessionID)
				select {
				case errCh <- nil:
				case <-ctx.Done():
				}
				return
			}
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
			return
		}

		if msg.GetPtyId() != "" && msg.GetPtyId() != sessionID {
			select {
			case errCh <- status.Error(codes.InvalidArgument, "pty_id cannot change within a stream"):
			case <-ctx.Done():
			}
			return
		}

		if err := s.handleStreamPayload(sessionID, msg); err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
			return
		}
	}
}

func (s *AgentServer) handleStreamPayload(sessionID string, msg *pb.StreamPtyRequest) error {
	switch payload := msg.GetPayload().(type) {
	case nil:
		return nil
	case *pb.StreamPtyRequest_Data:
		if err := s.ptyMgr.Write(sessionID, payload.Data); err != nil {
			return mapManagerError(err)
		}
		return nil
	case *pb.StreamPtyRequest_Resize:
		if payload.Resize == nil {
			return status.Error(codes.InvalidArgument, "resize payload is required")
		}
		if payload.Resize.GetCols() == 0 || payload.Resize.GetRows() == 0 {
			return status.Error(codes.InvalidArgument, "resize cols and rows must be greater than zero")
		}
		if payload.Resize.GetCols() > math.MaxUint16 || payload.Resize.GetRows() > math.MaxUint16 {
			return status.Error(codes.InvalidArgument, "resize cols and rows exceed PTY limits")
		}
		if err := s.ptyMgr.Resize(sessionID, uint16(payload.Resize.GetCols()), uint16(payload.Resize.GetRows())); err != nil {
			return mapManagerError(err)
		}
		return nil
	case *pb.StreamPtyRequest_Signal:
		signal, err := parseSignal(payload.Signal)
		if err != nil {
			return err
		}
		if err := s.ptyMgr.Signal(sessionID, signal); err != nil {
			return mapManagerError(err)
		}
		return nil
	default:
		return status.Error(codes.InvalidArgument, "unknown PTY payload")
	}
}

func mapManagerError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ptymanager.ErrSessionNotFound) {
		return status.Error(codes.NotFound, "pty session not found")
	}
	return status.Errorf(codes.Internal, "pty operation failed: %v", err)
}

func parseSignal(value int32) (syscall.Signal, error) {
	signal := syscall.Signal(value)
	switch signal {
	case syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL:
		return signal, nil
	default:
		return 0, status.Errorf(codes.InvalidArgument, "unsupported signal %d", value)
	}
}

func (s *AgentServer) trackSessionLifetime(sessionID string) {
	if s.usageTracker == nil || strings.TrimSpace(sessionID) == "" {
		return
	}

	s.usageTracker.StartSession(sessionID)

	exitCh, err := s.ptyMgr.OnExit(sessionID)
	if err != nil {
		s.usageTracker.EndSession(sessionID)
		return
	}

	go func() {
		defer s.usageTracker.EndSession(sessionID)
		for range exitCh {
		}
	}()
}
