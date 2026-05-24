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
	"github.com/your-org/cortado/daemon/internal/filesync"
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

func TestWebSocketUpgradeHandlesSyncCommands(t *testing.T) {
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

	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read hello message: %v", err)
	}

	commandPayload, err := json.Marshal(map[string]string{
		"type":        "start_sync",
		"requestId":   "req-1",
		"localPath":   "/tmp/workspace",
		"workspaceId": "ws-123",
	})
	if err != nil {
		t.Fatalf("marshal start sync command: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, commandPayload); err != nil {
		t.Fatalf("write start sync command: %v", err)
	}

	_, responsePayload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read start sync response: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal(responsePayload, &response); err != nil {
		t.Fatalf("decode start sync response: %v", err)
	}
	if response["type"] != "sync_status" {
		t.Fatalf("unexpected response type: got %#v want %q", response["type"], "sync_status")
	}
	if response["state"] != string(app.SyncStateSyncing) {
		t.Fatalf("unexpected response state: got %#v want %q", response["state"], app.SyncStateSyncing)
	}
	if response["workspaceId"] != "ws-123" {
		t.Fatalf("unexpected workspace id: %#v", response["workspaceId"])
	}
	if response["localPath"] != "/tmp/workspace" {
		t.Fatalf("unexpected local path: %#v", response["localPath"])
	}

	commandPayload, err = json.Marshal(map[string]string{
		"type":        "get_sync_status",
		"requestId":   "req-2",
		"localPath":   "/tmp/workspace",
		"workspaceId": "ws-123",
	})
	if err != nil {
		t.Fatalf("marshal get sync status command: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, commandPayload); err != nil {
		t.Fatalf("write get sync status command: %v", err)
	}

	_, responsePayload, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("read get sync status response: %v", err)
	}
	if err := json.Unmarshal(responsePayload, &response); err != nil {
		t.Fatalf("decode get sync status response: %v", err)
	}
	if response["state"] != string(app.SyncStateSyncing) {
		t.Fatalf("unexpected tracked state: got %#v want %q", response["state"], app.SyncStateSyncing)
	}
}

func TestWebSocketUpgradeForwardsConflictNoticeFrame(t *testing.T) {
	broadcaster := app.NewConflictBroadcaster()
	registry := app.NewSyncRegistry()
	if _, err := registry.StartSync("/tmp/workspace", "ws-123"); err != nil {
		t.Fatalf("start sync in registry: %v", err)
	}
	server := newHTTPTestServerWithDependencies(t, broadcaster, registry)
	defer server.Close()

	dialer := websocket.Dialer{
		Subprotocols: []string{"cortado-daemon-v1"},
	}
	conn, _, err := dialer.Dial(websocketURL(server.URL+"/"), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read hello message: %v", err)
	}

	broadcaster.PublishConflict(filesync.ConflictNotice{
		Path:            "/tmp/workspace/main.txt",
		Reason:          "text conflict requires manual resolution",
		LocalClock:      2,
		RemoteClock:     3,
		LastSyncedClock: 1,
	})

	messageType, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read conflict frame: %v", err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("unexpected websocket message type: got %d want %d", messageType, websocket.BinaryMessage)
	}

	frame, err := app.DecodeFrame(payload)
	if err != nil {
		t.Fatalf("decode conflict frame: %v", err)
	}
	if frame.ChannelID != 0x0600 {
		t.Fatalf("unexpected conflict channel: got %d want %d", frame.ChannelID, 0x0600)
	}
	if frame.MessageType != app.MessageTypeData {
		t.Fatalf("unexpected frame message type: got %d want %d", frame.MessageType, app.MessageTypeData)
	}

	var notice filesync.ConflictNotice
	if err := json.Unmarshal(frame.Payload, &notice); err != nil {
		t.Fatalf("unmarshal conflict payload: %v", err)
	}
	if notice.Path != "/tmp/workspace/main.txt" || notice.RemoteClock != 3 {
		t.Fatalf("unexpected conflict notice payload: %#v", notice)
	}

	status, err := registry.GetSyncStatus("/tmp/workspace", "ws-123")
	if err != nil {
		t.Fatalf("get sync status after conflict: %v", err)
	}
	if status.State != app.SyncStateConflicted {
		t.Fatalf("unexpected sync state after conflict: got %q want %q", status.State, app.SyncStateConflicted)
	}
	if status.WorkspacePath != "/main.txt" {
		t.Fatalf("unexpected workspace path after conflict: got %q want %q", status.WorkspacePath, "/main.txt")
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
	return newHTTPTestServerWithDependencies(t, nil, app.NewSyncRegistry())
}

func newHTTPTestServerWithBroadcaster(t *testing.T, broadcaster *app.ConflictBroadcaster) *httptest.Server {
	t.Helper()
	return newHTTPTestServerWithDependencies(t, broadcaster, app.NewSyncRegistry())
}

func newHTTPTestServerWithDependencies(
	t *testing.T,
	broadcaster *app.ConflictBroadcaster,
	registry *app.SyncRegistry,
) *httptest.Server {
	t.Helper()

	store := mustOpenStore(t)
	t.Cleanup(func() {
		_ = store.Close()
	})

	server, err := app.NewServer(app.ServerConfig{
		ConflictBroadcaster: broadcaster,
		ListenAddr:          "127.0.0.1:9731",
		Logger:              log.New(io.Discard, "", 0),
		StateStore:          store,
		SyncRegistry:        registry,
		Version:             version.Info(),
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
