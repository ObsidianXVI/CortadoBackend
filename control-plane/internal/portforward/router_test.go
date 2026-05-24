package portforward

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	"github.com/your-org/cortado/control-plane/internal/workspace"
)

func TestHealth(t *testing.T) {
	router := NewRouter(RouterConfig{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusOK)
	}
}

func TestHTTPProxy(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/preview/index.html" {
			t.Fatalf("unexpected proxied path: %q", r.URL.Path)
		}
		if r.URL.RawQuery != "theme=dark" {
			t.Fatalf("unexpected proxied query: %q", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("proxied"))
	}))
	defer upstream.Close()

	parsedUpstream, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream url: %v", err)
	}
	port, err := strconv.Atoi(parsedUpstream.Port())
	if err != nil {
		t.Fatalf("parse upstream port: %v", err)
	}

	handler := NewHandler(HandlerConfig{
		PortService: portServiceStub{ports: []*agentpb.PortInfo{
			{Port: uint32(port)},
		}},
		WorkspaceResolver: staticResolver{serviceDNS: parsedUpstream.Hostname()},
		WorkspaceService: workspaceServiceStub{
			workspace: workspace.Workspace{
				ID:       "ws-123",
				TenantID: "dev-tenant",
				Status:   workspace.StatusRunning,
			},
		},
	})
	router := NewRouter(RouterConfig{Handler: handler})
	server := httptest.NewServer(router)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/ws-123/"+parsedUpstream.Port()+"/preview/index.html?theme=dark", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-Cortado-Dev-Token", "dev-bypass")

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("unexpected status: got %d want %d", resp.StatusCode, http.StatusCreated)
	}
	if string(body) != "proxied" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestWebSocketProxy(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade upstream websocket: %v", err)
		}
		defer conn.Close()

		mt, payload, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read upstream websocket message: %v", err)
		}
		if err := conn.WriteMessage(mt, append([]byte("echo:"), payload...)); err != nil {
			t.Fatalf("write upstream websocket message: %v", err)
		}
	}))
	defer upstream.Close()

	parsedUpstream, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream url: %v", err)
	}
	port, err := strconv.Atoi(parsedUpstream.Port())
	if err != nil {
		t.Fatalf("parse upstream port: %v", err)
	}

	handler := NewHandler(HandlerConfig{
		PortService: portServiceStub{ports: []*agentpb.PortInfo{
			{Port: uint32(port)},
		}},
		WorkspaceResolver: staticResolver{serviceDNS: parsedUpstream.Hostname()},
		WorkspaceService: workspaceServiceStub{
			workspace: workspace.Workspace{
				ID:       "ws-123",
				TenantID: "dev-tenant",
				Status:   workspace.StatusRunning,
			},
		},
	})
	server := httptest.NewServer(NewRouter(RouterConfig{Handler: handler}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws-123/" + parsedUpstream.Port() + "/socket"
	headers := http.Header{}
	headers.Set("X-Cortado-Dev-Token", "dev-bypass")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}

	messageType, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}
	if messageType != websocket.TextMessage {
		t.Fatalf("unexpected websocket message type: got %d want %d", messageType, websocket.TextMessage)
	}
	if string(payload) != "echo:hello" {
		t.Fatalf("unexpected websocket payload: %q", payload)
	}
}

func TestUnauthorized(t *testing.T) {
	router := NewRouter(RouterConfig{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	})

	req := httptest.NewRequest(http.MethodGet, "/ws-123/8080/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d want %d", rec.Code, http.StatusUnauthorized)
	}
}

type staticResolver struct {
	serviceDNS string
}

func (r staticResolver) GetServiceDNS(string) string {
	return r.serviceDNS
}

type portServiceStub struct {
	ports []*agentpb.PortInfo
	err   error
}

func (s portServiceStub) ListPorts(context.Context, string) ([]*agentpb.PortInfo, error) {
	return s.ports, s.err
}

type workspaceServiceStub struct {
	err       error
	workspace workspace.Workspace
}

func (s workspaceServiceStub) GetWorkspace(context.Context, string, string) (workspace.Workspace, error) {
	return s.workspace, s.err
}
