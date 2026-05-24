package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/your-org/cortado/daemon/internal/state"
	"github.com/your-org/cortado/daemon/internal/version"
)

const daemonSubprotocol = "cortado-daemon-v1"

type ServerConfig struct {
	ListenAddr string
	Logger     *log.Logger
	StateStore *state.Store
	Version    version.BuildInfo
}

type Server struct {
	listenAddr string
	logger     *log.Logger
	server     *http.Server
	stateStore *state.Store
	version    version.BuildInfo
}

func NewServer(cfg ServerConfig) (*Server, error) {
	if cfg.ListenAddr == "" {
		return nil, fmt.Errorf("listen address must not be empty")
	}
	if host, _, err := net.SplitHostPort(cfg.ListenAddr); err != nil {
		return nil, fmt.Errorf("parse listen address %q: %w", cfg.ListenAddr, err)
	} else if host != "127.0.0.1" {
		return nil, fmt.Errorf("listen address host must be 127.0.0.1, got %q", host)
	}
	if cfg.StateStore == nil {
		return nil, fmt.Errorf("state store must not be nil")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}

	server := &Server{
		listenAddr: cfg.ListenAddr,
		logger:     logger,
		stateStore: cfg.StateStore,
		version:    cfg.Version,
	}
	server.server = &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return server, nil
}

func (s *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.listenAddr, err)
	}
	defer listener.Close()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil && ctx.Err() == nil {
			s.logger.Printf("shutdown daemon server: %v", err)
		}
	}()

	if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve daemon http server: %w", err)
	}
	return nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleProxy)
	mux.HandleFunc("/healthz", s.handleHealth)
	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Upgrade, Sec-WebSocket-Key, Sec-WebSocket-Version, Sec-WebSocket-Protocol")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"listenAddr":    s.listenAddr,
		"schemaVersion": s.stateStore.SchemaVersion(),
		"statePath":     s.stateStore.Path(),
		"status":        "ok",
		"version":       s.version,
	})
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	if !websocket.IsWebSocketUpgrade(r) {
		s.handleHealth(w, r)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool {
			return true
		},
		Subprotocols: []string{daemonSubprotocol},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Printf("upgrade local daemon websocket: %v", err)
		return
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type":          "hello",
		"schemaVersion": s.stateStore.SchemaVersion(),
		"statePath":     s.stateStore.Path(),
		"version":       s.version,
	}); err != nil {
		s.logger.Printf("write daemon hello frame: %v", err)
		return
	}

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
			) {
				return
			}
			s.logger.Printf("read local daemon websocket: %v", err)
			return
		}

		if err := conn.WriteMessage(messageType, payload); err != nil {
			s.logger.Printf("echo local daemon websocket payload: %v", err)
			return
		}
	}
}
