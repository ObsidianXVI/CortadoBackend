package gateway_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/your-org/cortado/control-plane/internal/api"
	"github.com/your-org/cortado/control-plane/internal/gateway"
)

func TestConnectRouteRejectsUnauthorizedUpgrade(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	server := httptest.NewServer(api.NewRouter(api.RouterConfig{
		ConnectHandler: gateway.NewConnectHandler(gateway.ConnectHandlerConfig{
			Logger: newDiscardLogger(),
			MuxConnConfig: gateway.MuxConnConfig{
				Logger:       newDiscardLogger(),
				PingInterval: time.Hour,
			},
		}),
	}))
	defer server.Close()

	wsURL := websocketURL(server.URL + "/v1/workspaces/ws-123/connect")
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if conn != nil {
		conn.Close()
	}
	if err == nil {
		t.Fatal("expected unauthorized websocket handshake to fail")
	}
	if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: %#v", response)
	}
}

func TestConnectRouteDispatchesTerminalFrames(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	received := make(chan gateway.Frame, 1)
	workspaceIDs := make(chan string, 1)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{
		ConnectHandler: gateway.NewConnectHandler(gateway.ConnectHandlerConfig{
			Logger: newDiscardLogger(),
			MuxConnConfig: gateway.MuxConnConfig{
				Logger:       newDiscardLogger(),
				PingInterval: time.Hour,
			},
			TerminalHandler: func(_ context.Context, session gateway.Session, frame gateway.Frame, _ time.Time) error {
				workspaceIDs <- session.WorkspaceID
				received <- frame
				session.Conn.SendFrame(gateway.Frame{
					ChannelID:   frame.ChannelID,
					MessageType: gateway.MessageTypeData,
					Payload:     []byte("ack"),
				})
				return nil
			},
		}),
	}))
	defer server.Close()

	ws := mustDial(t, server.URL+"/v1/workspaces/ws-123/connect?dev_token=dev-bypass")
	defer ws.Close()

	outbound := gateway.Frame{
		ChannelID:   gateway.TerminalChannelID,
		MessageType: gateway.MessageTypeOpen,
		Payload:     []byte("shell"),
	}
	raw, err := gateway.EncodeFrame(outbound)
	if err != nil {
		t.Fatalf("encode outbound frame: %v", err)
	}
	if err := ws.WriteMessage(websocket.BinaryMessage, raw); err != nil {
		t.Fatalf("write frame: %v", err)
	}

	select {
	case gotWorkspaceID := <-workspaceIDs:
		if gotWorkspaceID != "ws-123" {
			t.Fatalf("unexpected workspace id: %q", gotWorkspaceID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for terminal handler workspace id")
	}

	select {
	case gotFrame := <-received:
		if gotFrame.ChannelID != outbound.ChannelID {
			t.Fatalf("unexpected channel id: got %d want %d", gotFrame.ChannelID, outbound.ChannelID)
		}
		if gotFrame.MessageType != outbound.MessageType {
			t.Fatalf("unexpected message type: got %d want %d", gotFrame.MessageType, outbound.MessageType)
		}
		if string(gotFrame.Payload) != string(outbound.Payload) {
			t.Fatalf("unexpected payload: got %q want %q", gotFrame.Payload, outbound.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for terminal handler frame")
	}

	_, inboundRaw, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read ack frame: %v", err)
	}
	ack, err := gateway.DecodeFrame(inboundRaw)
	if err != nil {
		t.Fatalf("decode ack frame: %v", err)
	}
	if ack.MessageType != gateway.MessageTypeData {
		t.Fatalf("unexpected ack message type: got %d want %d", ack.MessageType, gateway.MessageTypeData)
	}
	if string(ack.Payload) != "ack" {
		t.Fatalf("unexpected ack payload: %q", ack.Payload)
	}
}

func TestConnectRouteNegotiatesCortadoSubprotocol(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	server := httptest.NewServer(api.NewRouter(api.RouterConfig{
		ConnectHandler: gateway.NewConnectHandler(gateway.ConnectHandlerConfig{
			Logger: newDiscardLogger(),
			MuxConnConfig: gateway.MuxConnConfig{
				Logger:       newDiscardLogger(),
				PingInterval: time.Hour,
			},
		}),
	}))
	defer server.Close()

	dialer := websocket.Dialer{
		Subprotocols: []string{"cortado-v1"},
	}
	conn, response, err := dialer.Dial(websocketURL(server.URL+"/v1/workspaces/ws-123/connect?dev_token=dev-bypass"), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v (response=%v)", err, response)
	}
	defer conn.Close()

	if got := conn.Subprotocol(); got != "cortado-v1" {
		t.Fatalf("unexpected negotiated subprotocol: got %q want %q", got, "cortado-v1")
	}
}

func TestConnectRouteReturnsErrorForUnsupportedChannel(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	server := httptest.NewServer(api.NewRouter(api.RouterConfig{
		ConnectHandler: gateway.NewConnectHandler(gateway.ConnectHandlerConfig{
			Logger: newDiscardLogger(),
			MuxConnConfig: gateway.MuxConnConfig{
				Logger:       newDiscardLogger(),
				PingInterval: time.Hour,
			},
		}),
	}))
	defer server.Close()

	ws := mustDial(t, server.URL+"/v1/workspaces/ws-123/connect?dev_token=dev-bypass")
	defer ws.Close()

	raw, err := gateway.EncodeFrame(gateway.Frame{
		ChannelID:   0x0200,
		MessageType: gateway.MessageTypeOpen,
		Payload:     []byte("ignored"),
	})
	if err != nil {
		t.Fatalf("encode outbound frame: %v", err)
	}
	if err := ws.WriteMessage(websocket.BinaryMessage, raw); err != nil {
		t.Fatalf("write unsupported channel frame: %v", err)
	}

	_, inboundRaw, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read error frame: %v", err)
	}
	frame, err := gateway.DecodeFrame(inboundRaw)
	if err != nil {
		t.Fatalf("decode error frame: %v", err)
	}
	if frame.ChannelID != 0x0200 {
		t.Fatalf("unexpected error frame channel: got %d want %d", frame.ChannelID, 0x0200)
	}
	if frame.MessageType != gateway.MessageTypeError {
		t.Fatalf("unexpected error frame type: got %d want %d", frame.MessageType, gateway.MessageTypeError)
	}
	if !strings.Contains(string(frame.Payload), "unsupported channel") {
		t.Fatalf("unexpected error payload: %q", frame.Payload)
	}
}

func mustDial(t *testing.T, rawURL string) *websocket.Conn {
	t.Helper()

	ws := websocketURL(rawURL)
	conn, _, err := websocket.DefaultDialer.Dial(ws, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}

	return conn
}

func websocketURL(rawURL string) string {
	return "ws" + strings.TrimPrefix(rawURL, "http")
}

func newDiscardLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}
