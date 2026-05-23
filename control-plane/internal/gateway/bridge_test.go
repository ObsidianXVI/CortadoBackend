package gateway_test

import (
	"context"
	"net"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	"github.com/your-org/cortado/control-plane/internal/api"
	"github.com/your-org/cortado/control-plane/internal/gateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestConnectRouteBridgesTerminalFramesToAgentStream(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	createReqCh := make(chan *agentpb.CreatePtyRequest, 1)
	streamReadyCh := make(chan string, 1)
	inboundDataCh := make(chan []byte, 1)
	targetCh := make(chan string, 1)

	dialCount := int32(0)
	dialer, cleanup := newAgentDialer(t, &stubAgentServer{
		createPty: func(_ context.Context, req *agentpb.CreatePtyRequest) (*agentpb.CreatePtyResponse, error) {
			createReqCh <- req
			return &agentpb.CreatePtyResponse{PtyId: "pty-123"}, nil
		},
		streamPty: func(stream agentpb.WorkspaceAgentService_StreamPtyServer) error {
			first, err := stream.Recv()
			if err != nil {
				return err
			}
			streamReadyCh <- first.GetPtyId()

			second, err := stream.Recv()
			if err != nil {
				return err
			}
			inboundDataCh <- append([]byte(nil), second.GetData()...)

			if err := stream.Send(&agentpb.StreamPtyResponse{
				Payload: &agentpb.StreamPtyResponse_Data{Data: []byte("echo:" + string(second.GetData()))},
			}); err != nil {
				return err
			}

			return stream.Send(&agentpb.StreamPtyResponse{
				Payload: &agentpb.StreamPtyResponse_ExitCode{ExitCode: 0},
			})
		},
	}, func(target string) {
		atomic.AddInt32(&dialCount, 1)
		targetCh <- target
	})
	defer cleanup()

	server := httptest.NewServer(api.NewRouter(api.RouterConfig{
		ConnectHandler: gateway.NewConnectHandler(gateway.ConnectHandlerConfig{
			Logger: newDiscardLogger(),
			MuxConnConfig: gateway.MuxConnConfig{
				Logger:       newDiscardLogger(),
				PingInterval: time.Hour,
			},
			GRPCDialer: dialer,
			WorkspaceResolver: testWorkspaceResolver{
				dns: "workspace.test.svc.cluster.local",
			},
		}),
	}))
	defer server.Close()

	ws := mustDial(t, server.URL+"/v1/workspaces/ws-123/connect?dev_token=dev-bypass")
	defer ws.Close()

	writeFrame(t, ws, gateway.Frame{
		ChannelID:   gateway.TerminalChannelID,
		MessageType: gateway.MessageTypeOpen,
		Payload:     []byte("/bin/bash"),
	})

	select {
	case req := <-createReqCh:
		if req.GetCols() != 80 || req.GetRows() != 24 {
			t.Fatalf("unexpected PTY size: cols=%d rows=%d", req.GetCols(), req.GetRows())
		}
		if req.GetShell() != "/bin/bash" {
			t.Fatalf("unexpected shell: %q", req.GetShell())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for CreatePty request")
	}

	select {
	case ptyID := <-streamReadyCh:
		if ptyID != "pty-123" {
			t.Fatalf("unexpected PTY id: %q", ptyID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for StreamPty setup")
	}

	select {
	case target := <-targetCh:
		if target != "workspace.test.svc.cluster.local:9090" {
			t.Fatalf("unexpected dial target: %q", target)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for gRPC dial")
	}

	writeFrame(t, ws, gateway.Frame{
		ChannelID:   gateway.TerminalChannelID,
		MessageType: gateway.MessageTypeData,
		Payload:     []byte("pwd\n"),
	})

	select {
	case got := <-inboundDataCh:
		if string(got) != "pwd\n" {
			t.Fatalf("unexpected gRPC stream data: %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for data to reach agent stream")
	}

	dataFrame := readFrame(t, ws)
	if dataFrame.MessageType != gateway.MessageTypeData {
		t.Fatalf("unexpected first frame type: got %d want %d", dataFrame.MessageType, gateway.MessageTypeData)
	}
	if string(dataFrame.Payload) != "echo:pwd\n" {
		t.Fatalf("unexpected echoed payload: %q", dataFrame.Payload)
	}

	closeFrame := readFrame(t, ws)
	if closeFrame.MessageType != gateway.MessageTypeClose {
		t.Fatalf("unexpected close frame type: got %d want %d", closeFrame.MessageType, gateway.MessageTypeClose)
	}
	if string(closeFrame.Payload) != "0" {
		t.Fatalf("unexpected close payload: %q", closeFrame.Payload)
	}

	if got := atomic.LoadInt32(&dialCount); got != 1 {
		t.Fatalf("unexpected dial count: got %d want %d", got, 1)
	}
}

func TestConnectRouteReusesCachedGRPCConnectionPerWorkspace(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	var createCount int32
	var dialCount int32

	dialer, cleanup := newAgentDialer(t, &stubAgentServer{
		createPty: func(_ context.Context, _ *agentpb.CreatePtyRequest) (*agentpb.CreatePtyResponse, error) {
			id := atomic.AddInt32(&createCount, 1)
			return &agentpb.CreatePtyResponse{PtyId: "pty-" + string(rune('0'+id))}, nil
		},
		streamPty: func(stream agentpb.WorkspaceAgentService_StreamPtyServer) error {
			if _, err := stream.Recv(); err != nil {
				return err
			}
			return stream.Send(&agentpb.StreamPtyResponse{
				Payload: &agentpb.StreamPtyResponse_ExitCode{ExitCode: 0},
			})
		},
	}, func(string) {
		atomic.AddInt32(&dialCount, 1)
	})
	defer cleanup()

	connectHandler := gateway.NewConnectHandler(gateway.ConnectHandlerConfig{
		Logger: newDiscardLogger(),
		MuxConnConfig: gateway.MuxConnConfig{
			Logger:       newDiscardLogger(),
			PingInterval: time.Hour,
		},
		GRPCDialer: dialer,
		WorkspaceResolver: testWorkspaceResolver{
			dns: "workspace.test.svc.cluster.local",
		},
	})

	server := httptest.NewServer(api.NewRouter(api.RouterConfig{ConnectHandler: connectHandler}))
	defer server.Close()

	for i := 0; i < 2; i++ {
		ws := mustDial(t, server.URL+"/v1/workspaces/ws-123/connect?dev_token=dev-bypass")
		writeFrame(t, ws, gateway.Frame{
			ChannelID:   gateway.TerminalChannelID,
			MessageType: gateway.MessageTypeOpen,
		})

		closeFrame := readFrame(t, ws)
		if closeFrame.MessageType != gateway.MessageTypeClose {
			t.Fatalf("unexpected close frame type: got %d want %d", closeFrame.MessageType, gateway.MessageTypeClose)
		}
		_ = ws.Close()
	}

	if got := atomic.LoadInt32(&dialCount); got != 1 {
		t.Fatalf("unexpected dial count: got %d want %d", got, 1)
	}
	if got := atomic.LoadInt32(&createCount); got != 2 {
		t.Fatalf("unexpected CreatePty count: got %d want %d", got, 2)
	}
}

func TestConnectRouteBridgesTerminalResizeFramesToAgentStream(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	resizeCh := make(chan *agentpb.WindowSize, 1)

	dialer, cleanup := newAgentDialer(t, &stubAgentServer{
		createPty: func(_ context.Context, _ *agentpb.CreatePtyRequest) (*agentpb.CreatePtyResponse, error) {
			return &agentpb.CreatePtyResponse{PtyId: "pty-123"}, nil
		},
		streamPty: func(stream agentpb.WorkspaceAgentService_StreamPtyServer) error {
			first, err := stream.Recv()
			if err != nil {
				return err
			}
			if first.GetPtyId() != "pty-123" {
				t.Fatalf("unexpected PTY id: %q", first.GetPtyId())
			}

			second, err := stream.Recv()
			if err != nil {
				return err
			}
			resizeCh <- second.GetResize()

			return stream.Send(&agentpb.StreamPtyResponse{
				Payload: &agentpb.StreamPtyResponse_ExitCode{ExitCode: 0},
			})
		},
	}, nil)
	defer cleanup()

	server := httptest.NewServer(api.NewRouter(api.RouterConfig{
		ConnectHandler: gateway.NewConnectHandler(gateway.ConnectHandlerConfig{
			Logger: newDiscardLogger(),
			MuxConnConfig: gateway.MuxConnConfig{
				Logger:       newDiscardLogger(),
				PingInterval: time.Hour,
			},
			GRPCDialer: dialer,
			WorkspaceResolver: testWorkspaceResolver{
				dns: "workspace.test.svc.cluster.local",
			},
		}),
	}))
	defer server.Close()

	ws := mustDial(t, server.URL+"/v1/workspaces/ws-123/connect?dev_token=dev-bypass")
	defer ws.Close()

	writeFrame(t, ws, gateway.Frame{
		ChannelID:   gateway.TerminalChannelID,
		MessageType: gateway.MessageTypeOpen,
		Payload:     []byte("/bin/bash"),
	})

	writeFrame(t, ws, gateway.Frame{
		ChannelID:   gateway.TerminalChannelID,
		MessageType: gateway.MessageTypeResize,
		Payload: gateway.EncodeTerminalResizePayload(gateway.TerminalResize{
			Cols: 132,
			Rows: 43,
		}),
	})

	select {
	case size := <-resizeCh:
		if size == nil {
			t.Fatal("expected resize payload")
		}
		if size.GetCols() != 132 || size.GetRows() != 43 {
			t.Fatalf("unexpected resize payload: cols=%d rows=%d", size.GetCols(), size.GetRows())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for resize to reach agent stream")
	}

	closeFrame := readFrame(t, ws)
	if closeFrame.MessageType != gateway.MessageTypeClose {
		t.Fatalf("unexpected close frame type: got %d want %d", closeFrame.MessageType, gateway.MessageTypeClose)
	}
	if string(closeFrame.Payload) != "0" {
		t.Fatalf("unexpected close payload: %q", closeFrame.Payload)
	}
}

func TestConnectRouteReturnsCloseFrameWhenAgentCreateFails(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	dialer, cleanup := newAgentDialer(t, &stubAgentServer{
		createPty: func(_ context.Context, _ *agentpb.CreatePtyRequest) (*agentpb.CreatePtyResponse, error) {
			return nil, status.Error(codes.Unavailable, "workspace starting")
		},
	}, nil)
	defer cleanup()

	server := httptest.NewServer(api.NewRouter(api.RouterConfig{
		ConnectHandler: gateway.NewConnectHandler(gateway.ConnectHandlerConfig{
			Logger: newDiscardLogger(),
			MuxConnConfig: gateway.MuxConnConfig{
				Logger:       newDiscardLogger(),
				PingInterval: time.Hour,
			},
			GRPCDialer: dialer,
			WorkspaceResolver: testWorkspaceResolver{
				dns: "workspace.test.svc.cluster.local",
			},
		}),
	}))
	defer server.Close()

	ws := mustDial(t, server.URL+"/v1/workspaces/ws-123/connect?dev_token=dev-bypass")
	defer ws.Close()

	writeFrame(t, ws, gateway.Frame{
		ChannelID:   gateway.TerminalChannelID,
		MessageType: gateway.MessageTypeOpen,
	})

	closeFrame := readFrame(t, ws)
	if closeFrame.MessageType != gateway.MessageTypeClose {
		t.Fatalf("unexpected close frame type: got %d want %d", closeFrame.MessageType, gateway.MessageTypeClose)
	}
	if !strings.Contains(string(closeFrame.Payload), "workspace starting") {
		t.Fatalf("unexpected close payload: %q", closeFrame.Payload)
	}
}

type stubAgentServer struct {
	agentpb.UnimplementedWorkspaceAgentServiceServer

	createPty func(context.Context, *agentpb.CreatePtyRequest) (*agentpb.CreatePtyResponse, error)
	streamPty func(agentpb.WorkspaceAgentService_StreamPtyServer) error
}

func (s *stubAgentServer) CreatePty(ctx context.Context, req *agentpb.CreatePtyRequest) (*agentpb.CreatePtyResponse, error) {
	if s.createPty == nil {
		return &agentpb.CreatePtyResponse{PtyId: "pty-default"}, nil
	}
	return s.createPty(ctx, req)
}

func (s *stubAgentServer) StreamPty(stream agentpb.WorkspaceAgentService_StreamPtyServer) error {
	if s.streamPty == nil {
		return nil
	}
	return s.streamPty(stream)
}

type testWorkspaceResolver struct {
	dns string
}

func (r testWorkspaceResolver) GetServiceDNS(string) string {
	return r.dns
}

func newAgentDialer(t *testing.T, server agentpb.WorkspaceAgentServiceServer, onDial func(target string)) (gateway.GRPCDialFunc, func()) {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	agentpb.RegisterWorkspaceAgentServiceServer(grpcServer, server)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			t.Errorf("serve bufconn grpc server: %v", err)
		}
	}()

	dialer := func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
		if onDial != nil {
			onDial(target)
		}
		opts = append(opts, grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}))
		return grpc.NewClient("passthrough:///bufconn", opts...)
	}

	cleanup := func() {
		grpcServer.Stop()
		_ = listener.Close()
	}

	return dialer, cleanup
}

func writeFrame(t *testing.T, ws interface{ WriteMessage(int, []byte) error }, frame gateway.Frame) {
	t.Helper()

	raw, err := gateway.EncodeFrame(frame)
	if err != nil {
		t.Fatalf("encode frame: %v", err)
	}
	if err := ws.WriteMessage(2, raw); err != nil {
		t.Fatalf("write frame: %v", err)
	}
}

func readFrame(t *testing.T, ws interface{ ReadMessage() (int, []byte, error) }) gateway.Frame {
	t.Helper()

	_, raw, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	frame, err := gateway.DecodeFrame(raw)
	if err != nil {
		t.Fatalf("decode frame: %v", err)
	}
	return frame
}
