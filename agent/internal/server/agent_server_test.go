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

	client, cleanup := newTestClient(t)
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

	client, cleanup := newTestClient(t)
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

	client, cleanup := newTestClient(t)
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

	client, cleanup := newTestClient(t)
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

func newTestClient(t *testing.T) (pb.WorkspaceAgentServiceClient, func()) {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pb.RegisterWorkspaceAgentServiceServer(grpcServer, NewAgentServer(&ptymanager.Manager{}))

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
