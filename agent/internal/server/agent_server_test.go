package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	pb "github.com/your-org/cortado/agent/gen/agent/v1"
	lspmanager "github.com/your-org/cortado/agent/internal/lsp"
	portmonitor "github.com/your-org/cortado/agent/internal/ports"
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

func TestAgentServerOpenAndStreamLSP(t *testing.T) {
	t.Parallel()

	manager := lspmanager.NewManagerWithConfig(lspmanager.ManagerConfig{
		CommandFactory: helperLSPCommandFactory(t, "echo", ""),
		WorkspaceRoot:  t.TempDir(),
	})
	client, cleanup := newTestClientWithConfig(t, nil, AgentServerConfig{
		LSPManager:    manager,
		WorkspaceRoot: t.TempDir(),
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.OpenLSP(ctx, &pb.OpenLSPRequest{Language: "dart"}); err != nil {
		t.Fatalf("open lsp: %v", err)
	}

	stream, err := client.StreamLSP(ctx)
	if err != nil {
		t.Fatalf("stream lsp: %v", err)
	}
	if err := stream.Send(&pb.LSPMessage{Data: []byte(`{"id":1}`)}); err != nil {
		t.Fatalf("send lsp message: %v", err)
	}

	response, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv lsp message: %v", err)
	}
	if got, want := string(response.GetData()), `echo:{"id":1}`; got != want {
		t.Fatalf("unexpected lsp response: got %q want %q", got, want)
	}
}

func TestAgentServerRejectsStreamLSPBeforeOpen(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, nil)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.StreamLSP(ctx)
	if err != nil {
		t.Fatalf("stream lsp: %v", err)
	}
	if _, err := stream.Recv(); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected failed precondition, got %v", err)
	}
}

func TestAgentServerRejectsUnsupportedLSP(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClient(t, nil)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.OpenLSP(ctx, &pb.OpenLSPRequest{Language: "python"}); status.Code(err) != codes.InvalidArgument {
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

func TestAgentServerListPorts(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClientWithConfig(t, nil, AgentServerConfig{
		PortMonitor: &portMonitorStub{
			ports: [][]portmonitor.Port{
				{
					{Host: "127.0.0.1", Network: "tcp4", Port: 3000},
					{Host: "::1", Network: "tcp6", Port: 8080},
				},
			},
			pollInterval: 5 * time.Millisecond,
		},
		WorkspaceRoot: t.TempDir(),
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.ListPorts(ctx, &pb.ListPortsRequest{})
	if err != nil {
		t.Fatalf("list ports: %v", err)
	}
	if len(response.GetPorts()) != 2 {
		t.Fatalf("unexpected port count: got %d want 2", len(response.GetPorts()))
	}
	if response.GetPorts()[0].GetPort() != 3000 || response.GetPorts()[1].GetPort() != 8080 {
		t.Fatalf("unexpected ports: %#v", response.GetPorts())
	}
}

func TestAgentServerWatchPorts(t *testing.T) {
	t.Parallel()

	client, cleanup := newTestClientWithConfig(t, nil, AgentServerConfig{
		PortMonitor: &portMonitorStub{
			ports: [][]portmonitor.Port{
				{
					{Host: "127.0.0.1", Network: "tcp4", Port: 3000},
				},
				{
					{Host: "127.0.0.1", Network: "tcp4", Port: 3000},
					{Host: "127.0.0.1", Network: "tcp4", Port: 8080},
				},
				{
					{Host: "127.0.0.1", Network: "tcp4", Port: 8080},
				},
			},
			pollInterval: 10 * time.Millisecond,
		},
		PortPollInterval: 10 * time.Millisecond,
		WorkspaceRoot:    t.TempDir(),
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	stream, err := client.WatchPorts(ctx, &pb.WatchPortsRequest{})
	if err != nil {
		t.Fatalf("watch ports: %v", err)
	}

	first, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv first port event: %v", err)
	}
	if first.GetType() != pb.PortEventType_PORT_EVENT_TYPE_ADDED || first.GetPort().GetPort() != 8080 {
		t.Fatalf("unexpected first port event: %#v", first)
	}

	second, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv second port event: %v", err)
	}
	if second.GetType() != pb.PortEventType_PORT_EVENT_TYPE_REMOVED || second.GetPort().GetPort() != 3000 {
		t.Fatalf("unexpected second port event: %#v", second)
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

type portMonitorStub struct {
	calls        int
	pollInterval time.Duration
	ports        [][]portmonitor.Port
}

func (s *portMonitorStub) List() ([]portmonitor.Port, error) {
	if len(s.ports) == 0 {
		return nil, nil
	}
	index := s.calls
	if index >= len(s.ports) {
		index = len(s.ports) - 1
	}
	s.calls++
	return append([]portmonitor.Port(nil), s.ports[index]...), nil
}

func (s *portMonitorStub) PollInterval() time.Duration {
	if s.pollInterval > 0 {
		return s.pollInterval
	}
	return 5 * time.Second
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

func helperLSPCommandFactory(t *testing.T, mode, stateFile string) lspmanager.CommandFactory {
	t.Helper()

	return func(language, workspaceRoot string) (lspmanager.CommandConfig, error) {
		return lspmanager.CommandConfig{
			Args: []string{"-test.run=TestHelperProcess"},
			Dir:  workspaceRoot,
			Env: append(os.Environ(),
				"GO_WANT_HELPER_PROCESS=1",
				"LSP_HELPER_MODE="+mode,
				"LSP_HELPER_STATE_FILE="+stateFile,
			),
			Path: os.Args[0],
		}, nil
	}
}

func TestHelperProcess(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	switch os.Getenv("LSP_HELPER_MODE") {
	case "echo":
		runEchoHelper()
	default:
		os.Exit(2)
	}
}

func runEchoHelper() {
	reader := bufio.NewReader(os.Stdin)
	for {
		body, err := readHelperFrame(reader)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				os.Exit(0)
			}
			os.Exit(7)
		}
		if err := writeHelperFrame(os.Stdout, []byte(fmt.Sprintf("echo:%s", body))); err != nil {
			os.Exit(8)
		}
	}
}

func readHelperFrame(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			continue
		}

		contentLength, err = strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	return body, nil
}

func writeHelperFrame(writer io.Writer, data []byte) error {
	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	_, err := writer.Write(data)
	return err
}
