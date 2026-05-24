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
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
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

func TestConnectRouteBridgesFileWatchEventsToMuxChannel(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	dialer, cleanup := newAgentDialer(t, &stubAgentServer{
		watchFiles: func(_ *agentpb.WatchFilesRequest, stream agentpb.WorkspaceAgentService_WatchFilesServer) error {
			if err := stream.Send(&agentpb.WatchFilesResponse{
				Event: &agentpb.FileEvent{
					Path:     "src/main.go",
					Type:     agentpb.FileEventType_FILE_EVENT_TYPE_MODIFIED,
					Checksum: []byte{0xde, 0xad, 0xbe, 0xef},
				},
			}); err != nil {
				return err
			}
			return nil
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
		ChannelID:   gateway.FileSyncChannelID,
		MessageType: gateway.MessageTypeOpen,
	})

	dataFrame := readFrame(t, ws)
	if dataFrame.ChannelID != gateway.FileSyncChannelID {
		t.Fatalf("unexpected file channel id: got %d want %d", dataFrame.ChannelID, gateway.FileSyncChannelID)
	}
	if dataFrame.MessageType != gateway.MessageTypeData {
		t.Fatalf("unexpected file frame type: got %d want %d", dataFrame.MessageType, gateway.MessageTypeData)
	}

	var event agentpb.FileEvent
	if err := proto.Unmarshal(dataFrame.Payload, &event); err != nil {
		t.Fatalf("unmarshal file event payload: %v", err)
	}
	if event.GetPath() != "src/main.go" || event.GetType() != agentpb.FileEventType_FILE_EVENT_TYPE_MODIFIED {
		t.Fatalf("unexpected file event: %#v", event)
	}

	closeFrame := readFrame(t, ws)
	if closeFrame.ChannelID != gateway.FileSyncChannelID || closeFrame.MessageType != gateway.MessageTypeClose {
		t.Fatalf("unexpected close frame: %#v", closeFrame)
	}
}

func TestConnectRouteBridgesLSPFramesToAgentStream(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	openReqCh := make(chan *agentpb.OpenLSPRequest, 1)
	languageMDCh := make(chan string, 1)
	inboundDataCh := make(chan []byte, 1)

	dialer, cleanup := newAgentDialer(t, &stubAgentServer{
		openLSP: func(_ context.Context, req *agentpb.OpenLSPRequest) (*agentpb.OpenLSPResponse, error) {
			openReqCh <- req
			return &agentpb.OpenLSPResponse{}, nil
		},
		streamLSP: func(stream agentpb.WorkspaceAgentService_StreamLSPServer) error {
			if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
				values := md.Get("x-cortado-lsp-language")
				if len(values) > 0 {
					languageMDCh <- values[0]
				}
			}

			msg, err := stream.Recv()
			if err != nil {
				return err
			}
			inboundDataCh <- append([]byte(nil), msg.GetData()...)

			if err := stream.Send(&agentpb.LSPMessage{Data: []byte("echo:" + string(msg.GetData()))}); err != nil {
				return err
			}
			return nil
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
		ChannelID:   gateway.LSPChannelStartID,
		MessageType: gateway.MessageTypeOpen,
		Payload:     []byte("dart"),
	})

	select {
	case req := <-openReqCh:
		if req.GetLanguage() != "dart" {
			t.Fatalf("unexpected lsp language: %q", req.GetLanguage())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for OpenLSP request")
	}

	select {
	case language := <-languageMDCh:
		if language != "dart" {
			t.Fatalf("unexpected lsp metadata language: %q", language)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for StreamLSP metadata")
	}

	writeFrame(t, ws, gateway.Frame{
		ChannelID:   gateway.LSPChannelStartID,
		MessageType: gateway.MessageTypeData,
		Payload:     []byte(`{"id":1}`),
	})

	select {
	case got := <-inboundDataCh:
		if string(got) != `{"id":1}` {
			t.Fatalf("unexpected lsp stream data: %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for data to reach lsp stream")
	}

	dataFrame := readFrame(t, ws)
	if dataFrame.ChannelID != gateway.LSPChannelStartID {
		t.Fatalf("unexpected lsp channel id: got %d want %d", dataFrame.ChannelID, gateway.LSPChannelStartID)
	}
	if dataFrame.MessageType != gateway.MessageTypeData {
		t.Fatalf("unexpected lsp frame type: got %d want %d", dataFrame.MessageType, gateway.MessageTypeData)
	}
	if string(dataFrame.Payload) != `echo:{"id":1}` {
		t.Fatalf("unexpected lsp payload: %q", dataFrame.Payload)
	}

	closeFrame := readFrame(t, ws)
	if closeFrame.ChannelID != gateway.LSPChannelStartID || closeFrame.MessageType != gateway.MessageTypeClose {
		t.Fatalf("unexpected lsp close frame: %#v", closeFrame)
	}
}

func TestStaticWorkspaceResolverUsesConfiguredDomain(t *testing.T) {
	resolver := gateway.StaticWorkspaceResolver{
		Namespace: "cortado-workspaces",
		DNSDomain: "cortado-dev.internal",
	}

	if got := resolver.GetServiceDNS("ws-123"); got != "ws-123.cortado-workspaces.svc.cortado-dev.internal" {
		t.Fatalf("unexpected service dns: %q", got)
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

func TestConnectRouteRedialsWorkspaceConnAfterCreateDeadlineExceeded(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	var dialCount int32

	dialer, cleanup := newSequentialAgentDialer(
		t,
		[]agentpb.WorkspaceAgentServiceServer{
			&stubAgentServer{
				createPty: func(_ context.Context, _ *agentpb.CreatePtyRequest) (*agentpb.CreatePtyResponse, error) {
					return nil, status.Error(codes.DeadlineExceeded, "context deadline exceeded while waiting for connections to become ready")
				},
			},
			&stubAgentServer{
				createPty: func(_ context.Context, _ *agentpb.CreatePtyRequest) (*agentpb.CreatePtyResponse, error) {
					return &agentpb.CreatePtyResponse{PtyId: "pty-retried"}, nil
				},
				streamPty: func(stream agentpb.WorkspaceAgentService_StreamPtyServer) error {
					first, err := stream.Recv()
					if err != nil {
						return err
					}
					if first.GetPtyId() != "pty-retried" {
						t.Fatalf("unexpected PTY id after redial: %q", first.GetPtyId())
					}
					return stream.Send(&agentpb.StreamPtyResponse{
						Payload: &agentpb.StreamPtyResponse_ExitCode{ExitCode: 0},
					})
				},
			},
		},
		func(string) {
			atomic.AddInt32(&dialCount, 1)
		},
	)
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
	if string(closeFrame.Payload) != "0" {
		t.Fatalf("unexpected close payload: %q", closeFrame.Payload)
	}
	if got := atomic.LoadInt32(&dialCount); got != 2 {
		t.Fatalf("unexpected dial count: got %d want %d", got, 2)
	}
}

func TestConnectRouteDoesNotSendGRPCKeepalivesByDefault(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	dialer, cleanup := newTCPAgentDialerWithServerOptions(
		t,
		&stubAgentServer{
			createPty: func(_ context.Context, _ *agentpb.CreatePtyRequest) (*agentpb.CreatePtyResponse, error) {
				return &agentpb.CreatePtyResponse{PtyId: "pty-idle"}, nil
			},
			streamPty: func(stream agentpb.WorkspaceAgentService_StreamPtyServer) error {
				if _, err := stream.Recv(); err != nil {
					return err
				}
				<-stream.Context().Done()
				return stream.Context().Err()
			},
		},
		[]grpc.ServerOption{
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             time.Hour,
				PermitWithoutStream: true,
			}),
		},
		nil,
	)
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

	if err := ws.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	_, _, err := ws.ReadMessage()
	if err == nil {
		t.Fatal("expected idle connection to remain open without emitting a frame")
	}
	netErr, ok := err.(net.Error)
	if !ok || !netErr.Timeout() {
		t.Fatalf("expected read timeout, got %v", err)
	}
}

type stubAgentServer struct {
	agentpb.UnimplementedWorkspaceAgentServiceServer

	createPty  func(context.Context, *agentpb.CreatePtyRequest) (*agentpb.CreatePtyResponse, error)
	openLSP    func(context.Context, *agentpb.OpenLSPRequest) (*agentpb.OpenLSPResponse, error)
	streamLSP  func(agentpb.WorkspaceAgentService_StreamLSPServer) error
	streamPty  func(agentpb.WorkspaceAgentService_StreamPtyServer) error
	watchFiles func(*agentpb.WatchFilesRequest, agentpb.WorkspaceAgentService_WatchFilesServer) error
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

func (s *stubAgentServer) OpenLSP(ctx context.Context, req *agentpb.OpenLSPRequest) (*agentpb.OpenLSPResponse, error) {
	if s.openLSP == nil {
		return &agentpb.OpenLSPResponse{}, nil
	}
	return s.openLSP(ctx, req)
}

func (s *stubAgentServer) StreamLSP(stream agentpb.WorkspaceAgentService_StreamLSPServer) error {
	if s.streamLSP == nil {
		return nil
	}
	return s.streamLSP(stream)
}

func (s *stubAgentServer) WatchFiles(req *agentpb.WatchFilesRequest, stream agentpb.WorkspaceAgentService_WatchFilesServer) error {
	if s.watchFiles == nil {
		return nil
	}
	return s.watchFiles(req, stream)
}

type testWorkspaceResolver struct {
	dns string
}

func (r testWorkspaceResolver) GetServiceDNS(string) string {
	return r.dns
}

func newAgentDialer(t *testing.T, server agentpb.WorkspaceAgentServiceServer, onDial func(target string)) (gateway.GRPCDialFunc, func()) {
	t.Helper()
	return newAgentDialerWithServerOptions(t, server, nil, onDial)
}

func newAgentDialerWithServerOptions(t *testing.T, server agentpb.WorkspaceAgentServiceServer, serverOpts []grpc.ServerOption, onDial func(target string)) (gateway.GRPCDialFunc, func()) {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer(serverOpts...)
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

func newTCPAgentDialerWithServerOptions(t *testing.T, server agentpb.WorkspaceAgentServiceServer, serverOpts []grpc.ServerOption, onDial func(target string)) (gateway.GRPCDialFunc, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}

	grpcServer := grpc.NewServer(serverOpts...)
	agentpb.RegisterWorkspaceAgentServiceServer(grpcServer, server)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			t.Errorf("serve tcp grpc server: %v", err)
		}
	}()

	dialer := func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
		if onDial != nil {
			onDial(target)
		}
		return grpc.NewClient(listener.Addr().String(), opts...)
	}

	cleanup := func() {
		grpcServer.Stop()
		_ = listener.Close()
	}

	return dialer, cleanup
}

func newSequentialAgentDialer(t *testing.T, servers []agentpb.WorkspaceAgentServiceServer, onDial func(target string)) (gateway.GRPCDialFunc, func()) {
	t.Helper()

	listeners := make([]*bufconn.Listener, 0, len(servers))
	grpcServers := make([]*grpc.Server, 0, len(servers))

	for _, server := range servers {
		listener := bufconn.Listen(1024 * 1024)
		grpcServer := grpc.NewServer()
		agentpb.RegisterWorkspaceAgentServiceServer(grpcServer, server)

		go func(s *grpc.Server, l *bufconn.Listener) {
			if err := s.Serve(l); err != nil {
				t.Errorf("serve bufconn grpc server: %v", err)
			}
		}(grpcServer, listener)

		listeners = append(listeners, listener)
		grpcServers = append(grpcServers, grpcServer)
	}

	var dialIndex int32
	dialer := func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
		if onDial != nil {
			onDial(target)
		}

		index := int(atomic.AddInt32(&dialIndex, 1)) - 1
		if index >= len(listeners) {
			index = len(listeners) - 1
		}
		listener := listeners[index]

		opts = append(opts, grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}))
		return grpc.NewClient("passthrough:///bufconn", opts...)
	}

	cleanup := func() {
		for _, grpcServer := range grpcServers {
			grpcServer.Stop()
		}
		for _, listener := range listeners {
			_ = listener.Close()
		}
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
