package portforward

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	cpmiddleware "github.com/your-org/cortado/control-plane/internal/middleware"
	"github.com/your-org/cortado/control-plane/internal/workspace"
)

const (
	defaultProxyDialTimeout = 10 * time.Second
	maxPortNumber           = 65535
	minPortNumber           = 1024
)

type WorkspaceService interface {
	GetWorkspace(ctx context.Context, tenantID, workspaceID string) (workspace.Workspace, error)
}

type PortService interface {
	ListPorts(ctx context.Context, workspaceID string) ([]*agentpb.PortInfo, error)
}

type WorkspaceResolver interface {
	GetServiceDNS(workspaceID string) string
}

type HandlerConfig struct {
	DialContext       func(ctx context.Context, network, address string) (net.Conn, error)
	ProxyDialTimeout  time.Duration
	PortService       PortService
	WorkspaceResolver WorkspaceResolver
	WorkspaceService  WorkspaceService
}

type Handler struct {
	dialContext       func(ctx context.Context, network, address string) (net.Conn, error)
	portService       PortService
	proxyDialTimeout  time.Duration
	workspaceResolver WorkspaceResolver
	workspaceService  WorkspaceService
}

type target struct {
	address     string
	path        string
	rawQuery    string
	workspaceID string
}

func NewHandler(cfg HandlerConfig) *Handler {
	if cfg.ProxyDialTimeout <= 0 {
		cfg.ProxyDialTimeout = defaultProxyDialTimeout
	}

	dialContext := cfg.DialContext
	if dialContext == nil {
		dialer := &net.Dialer{Timeout: cfg.ProxyDialTimeout}
		dialContext = dialer.DialContext
	}

	return &Handler{
		dialContext:       dialContext,
		portService:       cfg.PortService,
		proxyDialTimeout:  cfg.ProxyDialTimeout,
		workspaceResolver: cfg.WorkspaceResolver,
		workspaceService:  cfg.WorkspaceService,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target, err := h.resolveTarget(r.Context(), r)
	if err != nil {
		writeProxyError(w, err)
		return
	}

	if isWebSocketRequest(r) {
		if err := h.proxyWebSocket(w, r, target); err != nil {
			writeProxyError(w, err)
		}
		return
	}

	h.proxyHTTP(w, r, target)
}

func (h *Handler) proxyHTTP(w http.ResponseWriter, r *http.Request, target target) {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL = &url.URL{
				Scheme:   "http",
				Host:     target.address,
				Path:     target.path,
				RawQuery: target.rawQuery,
			}
			req.Host = target.address
			req.RequestURI = ""
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeProxyError(w, fmt.Errorf("proxy http request: %w", err))
		},
	}
	proxy.ServeHTTP(w, r)
}

func (h *Handler) proxyWebSocket(w http.ResponseWriter, r *http.Request, target target) error {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return fmt.Errorf("websocket proxy requires hijack support")
	}

	upstreamConn, err := h.dialContext(r.Context(), "tcp", target.address)
	if err != nil {
		return fmt.Errorf("dial upstream websocket target %q: %w", target.address, err)
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		_ = upstreamConn.Close()
		return fmt.Errorf("hijack client connection: %w", err)
	}

	request := r.Clone(r.Context())
	request.URL.Scheme = "http"
	request.URL.Host = target.address
	request.URL.Path = target.path
	request.URL.RawQuery = target.rawQuery
	request.RequestURI = ""
	request.Host = target.address

	if err := request.Write(upstreamConn); err != nil {
		_ = clientConn.Close()
		_ = upstreamConn.Close()
		return nil
	}

	copyErr := make(chan error, 2)
	go func() {
		_, err := io.Copy(upstreamConn, clientConn)
		copyErr <- err
	}()
	go func() {
		_, err := io.Copy(clientConn, upstreamConn)
		copyErr <- err
	}()

	err = <-copyErr
	_ = clientConn.Close()
	_ = upstreamConn.Close()
	return nil
}

func (h *Handler) resolveTarget(ctx context.Context, r *http.Request) (target, error) {
	workspaceID, port, forwardPath, err := parseRequestPath(r.URL.Path)
	if err != nil {
		return target{}, err
	}

	tenantID, ok := cpmiddleware.TenantID(ctx)
	if !ok || strings.TrimSpace(tenantID) == "" {
		return target{}, fmt.Errorf("missing tenant context")
	}

	ws, err := h.workspaceService.GetWorkspace(ctx, tenantID, workspaceID)
	if err != nil {
		return target{}, err
	}
	if ws.Status != workspace.StatusRunning {
		return target{}, fmt.Errorf("%w: workspace is not running", workspace.ErrConflict)
	}

	ports, err := h.portService.ListPorts(ctx, workspaceID)
	if err != nil {
		return target{}, err
	}
	if !containsPort(ports, uint32(port)) {
		return target{}, workspace.ErrNotFound
	}

	return target{
		address:     net.JoinHostPort(h.workspaceResolver.GetServiceDNS(workspaceID), strconv.Itoa(port)),
		path:        forwardPath,
		rawQuery:    r.URL.RawQuery,
		workspaceID: workspaceID,
	}, nil
}

func containsPort(ports []*agentpb.PortInfo, want uint32) bool {
	for _, port := range ports {
		if port.GetPort() == want {
			return true
		}
	}
	return false
}

func isWebSocketRequest(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("Upgrade")), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

func parseRequestPath(rawPath string) (workspaceID string, port int, forwardPath string, err error) {
	trimmed := strings.Trim(rawPath, "/")
	if trimmed == "" {
		return "", 0, "", fmt.Errorf("%w: missing workspace and port path segments", workspace.ErrInvalid)
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return "", 0, "", fmt.Errorf("%w: expected /{workspaceId}/{port}/{path...}", workspace.ErrInvalid)
	}

	workspaceID = strings.TrimSpace(parts[0])
	if workspaceID == "" {
		return "", 0, "", fmt.Errorf("%w: workspace id is required", workspace.ErrWorkspace)
	}

	port, err = strconv.Atoi(parts[1])
	if err != nil || port < minPortNumber || port > maxPortNumber {
		return "", 0, "", fmt.Errorf("%w: invalid port", workspace.ErrInvalid)
	}

	forwardPath = "/"
	if len(parts) > 2 {
		forwardPath = "/" + strings.Join(parts[2:], "/")
	}

	return workspaceID, port, forwardPath, nil
}

func writeProxyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, workspace.ErrNotFound):
		http.Error(w, "workspace or port not found", http.StatusNotFound)
	case errors.Is(err, workspace.ErrConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, workspace.ErrInvalid), errors.Is(err, workspace.ErrWorkspace):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusBadGateway)
	}
}
