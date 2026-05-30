package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/your-org/cortado/control-plane/internal/gateway"
	"github.com/your-org/cortado/control-plane/internal/workspace"
)

func TestConnectRouteRejectsUnknownWorkspace(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	authService, accessToken := mustIssueAccessToken(t, "tenant-1", "user-1")
	router := NewRouter(RouterConfig{
		JWKSProvider: authService,
		WorkspaceSvc: &workspaceServiceStub{getErr: workspace.ErrNotFound},
		ConnectHandler: gateway.NewConnectHandler(gateway.ConnectHandlerConfig{
			MuxConnConfig: gateway.MuxConnConfig{PingInterval: time.Hour},
		}),
	})
	server := httptest.NewServer(router)
	defer server.Close()

	conn, response, err := websocket.DefaultDialer.Dial(
		websocketURL(server.URL+"/v1/workspaces/ws-404/connect?token="+url.QueryEscape(accessToken)),
		nil,
	)
	if conn != nil {
		conn.Close()
	}
	if err == nil {
		t.Fatal("expected unauthorized workspace websocket handshake to fail")
	}
	if response == nil || response.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status code: %#v", response)
	}
}

func websocketURL(rawURL string) string {
	return "ws" + rawURL[len("http"):]
}
