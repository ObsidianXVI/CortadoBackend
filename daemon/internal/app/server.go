package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/your-org/cortado/daemon/internal/filesync"
	"github.com/your-org/cortado/daemon/internal/state"
	"github.com/your-org/cortado/daemon/internal/version"
)

const daemonSubprotocol = "cortado-daemon-v1"

type daemonCommand struct {
	LocalPath   string `json:"localPath"`
	RequestID   string `json:"requestId"`
	Type        string `json:"type"`
	WorkspaceID string `json:"workspaceId"`
}

type daemonErrorResponse struct {
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
	Type      string `json:"type"`
}

type daemonSyncStatusResponse struct {
	SyncStatus
	RequestID string `json:"requestId,omitempty"`
	Type      string `json:"type"`
}

type ServerConfig struct {
	ConflictBroadcaster *ConflictBroadcaster
	ListenAddr          string
	Logger              *log.Logger
	StateStore          *state.Store
	SyncRegistry        *SyncRegistry
	Version             version.BuildInfo
}

type Server struct {
	conflictBroadcaster *ConflictBroadcaster
	listenAddr          string
	logger              *log.Logger
	server              *http.Server
	stateStore          *state.Store
	syncRegistry        *SyncRegistry
	version             version.BuildInfo
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
		conflictBroadcaster: cfg.ConflictBroadcaster,
		listenAddr:          cfg.ListenAddr,
		logger:              logger,
		stateStore:          cfg.StateStore,
		syncRegistry:        cfg.SyncRegistry,
		version:             cfg.Version,
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

	var writeMu sync.Mutex
	writeMessage := func(messageType int, payload []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()

		if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
			return fmt.Errorf("set daemon websocket write deadline: %w", err)
		}
		return conn.WriteMessage(messageType, payload)
	}

	helloPayload, err := json.Marshal(map[string]any{
		"type":          "hello",
		"schemaVersion": s.stateStore.SchemaVersion(),
		"statePath":     s.stateStore.Path(),
		"version":       s.version,
	})
	if err != nil {
		s.logger.Printf("marshal daemon hello frame: %v", err)
		return
	}
	if err := writeMessage(websocket.TextMessage, helloPayload); err != nil {
		s.logger.Printf("write daemon hello frame: %v", err)
		return
	}

	done := make(chan struct{})
	defer close(done)

	if s.conflictBroadcaster != nil {
		conflicts, unsubscribe := s.conflictBroadcaster.Subscribe()
		defer unsubscribe()

		go s.forwardConflicts(done, conflicts, writeMessage)
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

		if handled, err := s.handleWebSocketMessage(messageType, payload, writeMessage); err != nil {
			s.logger.Printf("handle local daemon websocket payload: %v", err)
			return
		} else if handled {
			continue
		}

		if err := writeMessage(messageType, payload); err != nil {
			s.logger.Printf("echo local daemon websocket payload: %v", err)
			return
		}
	}
}

func (s *Server) handleWebSocketMessage(
	messageType int,
	payload []byte,
	writeMessage func(int, []byte) error,
) (bool, error) {
	if messageType != websocket.TextMessage {
		return false, nil
	}

	var command daemonCommand
	if err := json.Unmarshal(payload, &command); err != nil {
		return false, nil
	}

	if command.Type != "start_sync" && command.Type != "stop_sync" && command.Type != "get_sync_status" {
		return false, nil
	}
	if s.syncRegistry == nil {
		return true, s.writeErrorResponse(
			writeMessage,
			command.RequestID,
			"sync registry is unavailable",
		)
	}

	var (
		status SyncStatus
		err    error
	)
	switch command.Type {
	case "start_sync":
		status, err = s.syncRegistry.StartSync(command.LocalPath, command.WorkspaceID)
	case "stop_sync":
		status, err = s.syncRegistry.StopSync(command.LocalPath, command.WorkspaceID)
	case "get_sync_status":
		status, err = s.syncRegistry.GetSyncStatus(command.LocalPath, command.WorkspaceID)
	}
	if err != nil {
		return true, s.writeErrorResponse(writeMessage, command.RequestID, err.Error())
	}

	responsePayload, err := json.Marshal(daemonSyncStatusResponse{
		SyncStatus: status,
		RequestID:  command.RequestID,
		Type:       "sync_status",
	})
	if err != nil {
		return true, fmt.Errorf("marshal daemon sync status response: %w", err)
	}
	if err := writeMessage(websocket.TextMessage, responsePayload); err != nil {
		return true, fmt.Errorf("write daemon sync status response: %w", err)
	}
	return true, nil
}

func (s *Server) writeErrorResponse(
	writeMessage func(int, []byte) error,
	requestID, message string,
) error {
	payload, err := json.Marshal(daemonErrorResponse{
		Message:   message,
		RequestID: requestID,
		Type:      "error",
	})
	if err != nil {
		return fmt.Errorf("marshal daemon error response: %w", err)
	}
	if err := writeMessage(websocket.TextMessage, payload); err != nil {
		return fmt.Errorf("write daemon error response: %w", err)
	}
	return nil
}

func (s *Server) forwardConflicts(
	done <-chan struct{},
	conflicts <-chan filesync.ConflictNotice,
	writeMessage func(int, []byte) error,
) {
	for {
		select {
		case <-done:
			return
		case notice, ok := <-conflicts:
			if !ok {
				return
			}
			if s.syncRegistry != nil {
				s.syncRegistry.MarkConflict(notice)
			}

			payload, err := json.Marshal(notice)
			if err != nil {
				s.logger.Printf("marshal conflict notice: %v", err)
				continue
			}
			frame, err := EncodeFrame(Frame{
				ChannelID:   conflictNoticeChannelID,
				MessageType: MessageTypeData,
				Payload:     payload,
			})
			if err != nil {
				s.logger.Printf("encode conflict notice frame: %v", err)
				continue
			}
			if err := writeMessage(websocket.BinaryMessage, frame); err != nil {
				s.logger.Printf("write conflict notice frame: %v", err)
				return
			}
		}
	}
}
