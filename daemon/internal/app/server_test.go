package app_test

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/your-org/cortado/daemon/internal/app"
	"github.com/your-org/cortado/daemon/internal/state"
	"github.com/your-org/cortado/daemon/internal/version"
)

func TestNewServerRejectsNonLoopbackAddress(t *testing.T) {
	store := mustOpenStore(t)
	defer store.Close()

	_, err := app.NewServer(app.ServerConfig{
		ListenAddr: "0.0.0.0:9731",
		Logger:     log.New(io.Discard, "", 0),
		StateStore: store,
		Version:    version.Info(),
	})
	if err == nil {
		t.Fatal("expected non-loopback listen address to be rejected")
	}
}

func TestPreflightReturnsCORSHeaders(t *testing.T) {
	server := newHTTPTestServer(t)
	defer server.Close()

	req, err := http.NewRequest(http.MethodOptions, server.URL+"/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Origin", "https://example.test")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)

	response, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("perform preflight request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected preflight status: got %d want %d", response.StatusCode, http.StatusNoContent)
	}
	if got := response.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("unexpected allow-origin header: got %q want %q", got, "*")
	}
}

func TestWebSocketUpgradeSendsHelloAndEchoesPayload(t *testing.T) {
	server := newHTTPTestServer(t)
	defer server.Close()

	dialer := websocket.Dialer{
		Subprotocols: []string{"cortado-daemon-v1"},
	}
	conn, _, err := dialer.Dial(websocketURL(server.URL+"/"), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	_, helloPayload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read hello message: %v", err)
	}

	var hello map[string]any
	if err := json.Unmarshal(helloPayload, &hello); err != nil {
		t.Fatalf("decode hello payload: %v", err)
	}
	if hello["type"] != "hello" {
		t.Fatalf("unexpected hello type: got %#v want %q", hello["type"], "hello")
	}
	if _, ok := hello["schemaVersion"]; !ok {
		t.Fatal("expected schemaVersion in hello payload")
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}

	_, echoPayload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read echo message: %v", err)
	}
	if string(echoPayload) != "ping" {
		t.Fatalf("unexpected echo payload: got %q want %q", echoPayload, "ping")
	}
}

func TestRunShutsDownWhenContextCancels(t *testing.T) {
	store := mustOpenStore(t)
	defer store.Close()

	server, err := app.NewServer(app.ServerConfig{
		ListenAddr: "127.0.0.1:0",
		Logger:     log.New(io.Discard, "", 0),
		StateStore: store,
		Version:    version.Info(),
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error after shutdown: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server shutdown")
	}
}

func newHTTPTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	store := mustOpenStore(t)
	t.Cleanup(func() {
		_ = store.Close()
	})

	server, err := app.NewServer(app.ServerConfig{
		ListenAddr: "127.0.0.1:9731",
		Logger:     log.New(io.Discard, "", 0),
		StateStore: store,
		Version:    version.Info(),
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	return httptest.NewServer(server.Routes())
}

func mustOpenStore(t *testing.T) *state.Store {
	t.Helper()

	store, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}

func websocketURL(rawURL string) string {
	return "ws" + strings.TrimPrefix(rawURL, "http")
}
