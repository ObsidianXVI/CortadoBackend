package server

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	pb "github.com/your-org/cortado/agent/gen/agent/v1"
	ptymanager "github.com/your-org/cortado/agent/internal/pty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestAgentServerHealth(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, nil)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if resp.GetStatus() != "ok" {
		t.Fatalf("unexpected health status: %q", resp.GetStatus())
	}
}

func TestAgentServerGetIdleStatus(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, nil)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetIdleStatus(ctx, &pb.GetIdleStatusRequest{})
	if err != nil {
		t.Fatalf("get idle status: %v", err)
	}
	if resp.GetCpuPercentOverSixtySeconds() < 0 {
		t.Fatalf("unexpected cpu percent: %f", resp.GetCpuPercentOverSixtySeconds())
	}
}

func TestAgentServerCreateAndStreamPty(t *testing.T) {
	t.Parallel()

	tracker := &usageTrackerStub{}
	client, cleanup := newTestClient(t, tracker)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	createResp, err := client.CreatePty(ctx, &pb.CreatePtyRequest{
		Cols:  80,
		Rows:  24,
		Shell: "/bin/bash",
	})
	if err != nil {
		t.Fatalf("create pty: %v", err)
	}
	if len(tracker.started) != 1 || tracker.started[0] != createResp.GetPtyId() {
		t.Fatalf("unexpected tracked sessions: %#v", tracker.started)
	}

	stream, err := client.StreamPty(ctx)
	if err != nil {
		t.Fatalf("stream pty: %v", err)
	}

	if err := stream.Send(&pb.StreamPtyRequest{PtyId: createResp.GetPtyId()}); err != nil {
		t.Fatalf("send handshake: %v", err)
	}
	if err := stream.Send(&pb.StreamPtyRequest{
		Payload: &pb.StreamPtyRequest_Data{Data: []byte("echo hello_grpc\nexit\n")},
	}); err != nil {
		t.Fatalf("send PTY input: %v", err)
	}

	var output strings.Builder
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("recv PTY output: %v", err)
		}

		switch payload := resp.GetPayload().(type) {
		case *pb.StreamPtyResponse_Data:
			output.Write(payload.Data)
			if strings.Contains(output.String(), "hello_grpc") {
				return
			}
		case *pb.StreamPtyResponse_ExitCode:
			if payload.ExitCode != 0 {
				t.Fatalf("unexpected exit code: %d", payload.ExitCode)
			}
		}
	}

	t.Fatalf("stream output missing expected marker: %q", output.String())
}

func TestAgentServerRejectsMissingStreamPtySessionID(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, nil)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.StreamPty(ctx)
	if err != nil {
		t.Fatalf("stream pty: %v", err)
	}

	if err := stream.Send(&pb.StreamPtyRequest{}); err != nil {
		t.Fatalf("send handshake: %v", err)
	}

	_, err = stream.Recv()
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestAgentServerFlushUsageWAL(t *testing.T) {
	t.Parallel()

	tracker := &usageTrackerStub{}
	client, cleanup := newTestClient(t, tracker)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.FlushUsageWAL(ctx, &pb.FlushUsageWALRequest{}); err != nil {
		t.Fatalf("flush usage WAL: %v", err)
	}
	if tracker.flushCalls != 1 {
		t.Fatalf("unexpected flush call count: got %d want 1", tracker.flushCalls)
	}
}

func TestAgentServerCreateSnapshot(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	var calls []snapshotCommandCall
	client, cleanup := newTestClientWithConfig(t, nil, AgentServerConfig{
		CommandRunner: func(_ context.Context, env []string, name string, args ...string) ([]byte, error) {
			calls = append(calls, snapshotCommandCall{
				args: append([]string{name}, args...),
				env:  append([]string(nil), env...),
			})
			return []byte("ok"), nil
		},
		SnapshotBucket:   "cortado-snapshots-cortado-ide-dev",
		SnapshotPassword: "snapshot-secret",
		WorkspaceID:      "ws-123",
		WorkspaceRoot:    workspaceRoot,
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.CreateSnapshot(ctx, &pb.CreateSnapshotRequest{})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if response.GetRepository() != "gs:cortado-snapshots-cortado-ide-dev:/ws-123" {
		t.Fatalf("unexpected repository: %q", response.GetRepository())
	}
	if len(calls) != 2 {
		t.Fatalf("expected init and backup calls, got %d", len(calls))
	}
	if got := strings.Join(calls[0].args, " "); !strings.Contains(got, "restic -r gs:cortado-snapshots-cortado-ide-dev:/ws-123 init") {
		t.Fatalf("unexpected init command: %q", got)
	}
	if got := strings.Join(calls[1].args, " "); !strings.Contains(got, "restic -r gs:cortado-snapshots-cortado-ide-dev:/ws-123 backup "+workspaceRoot) {
		t.Fatalf("unexpected backup command: %q", got)
	}
	if !containsEnv(calls[0].env, "RESTIC_PASSWORD=snapshot-secret") {
		t.Fatalf("missing restic password env: %#v", calls[0].env)
	}
}

func TestAgentServerCreateSnapshotRequiresConfiguration(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClientWithConfig(t, nil, AgentServerConfig{WorkspaceRoot: t.TempDir()})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.CreateSnapshot(ctx, &pb.CreateSnapshotRequest{})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected failed precondition, got %v", err)
	}
}

func TestAgentServerCreateSnapshotMapsTimeout(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClientWithConfig(t, nil, AgentServerConfig{
		CommandRunner: func(_ context.Context, _ []string, _ string, _ ...string) ([]byte, error) {
			return nil, context.DeadlineExceeded
		},
		SnapshotBucket:   "cortado-snapshots-cortado-ide-dev",
		SnapshotPassword: "snapshot-secret",
		WorkspaceID:      "ws-123",
		WorkspaceRoot:    t.TempDir(),
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.CreateSnapshot(ctx, &pb.CreateSnapshotRequest{})
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func newTestClient(t *testing.T, tracker usageTracker) (pb.WorkspaceAgentServiceClient, func()) {
	t.Helper()
	return newTestClientWithWorkspaceRoot(t, tracker, t.TempDir())
}

func newTestClientWithWorkspaceRoot(t *testing.T, tracker usageTracker, workspaceRoot string) (pb.WorkspaceAgentServiceClient, func()) {
	t.Helper()

	return newTestClientWithConfig(t, tracker, AgentServerConfig{WorkspaceRoot: workspaceRoot})
}

func newTestClientWithConfig(t *testing.T, tracker usageTracker, cfg AgentServerConfig) (pb.WorkspaceAgentServiceClient, func()) {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pb.RegisterWorkspaceAgentServiceServer(
		grpcServer,
		NewAgentServerWithConfig(&ptymanager.Manager{}, tracker, cfg),
	)

	go func() {
		if serveErr := grpcServer.Serve(listener); serveErr != nil {
			t.Logf("grpc server stopped: %v", serveErr)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	cancel()
	if err != nil {
		grpcServer.Stop()
		_ = listener.Close()
		t.Fatalf("dial bufconn: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return pb.NewWorkspaceAgentServiceClient(conn), cleanup
}

type usageTrackerStub struct {
	flushCalls int
	started    []string
}

type snapshotCommandCall struct {
	args []string
	env  []string
}

func (u *usageTrackerStub) EndSession(string) {}

func (u *usageTrackerStub) Flush(context.Context) error {
	u.flushCalls++
	return nil
}

func (u *usageTrackerStub) StartSession(sessionID string) {
	u.started = append(u.started, sessionID)
}

func containsEnv(env []string, target string) bool {
	for _, entry := range env {
		if entry == target {
			return true
		}
	}
	return false
}
