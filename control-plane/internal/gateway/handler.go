package gateway

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

const cortadoWebSocketProtocol = "cortado-v1"

type Session struct {
	Conn        *MuxConn
	WorkspaceID string
}

type TerminalFrameHandler func(ctx context.Context, session Session, frame Frame, receivedAt time.Time) error
type LSPFrameHandler func(ctx context.Context, session Session, frame Frame, receivedAt time.Time) error
type FileFrameHandler func(ctx context.Context, session Session, frame Frame, receivedAt time.Time) error

type ConnectHandlerConfig struct {
	FileHandler        FileFrameHandler
	LSPHandler         LSPFrameHandler
	Logger             *log.Logger
	MuxConnConfig      MuxConnConfig
	TerminalHandler    TerminalFrameHandler
	GRPCDialer         GRPCDialFunc
	Upgrader           websocket.Upgrader
	WorkspaceNamespace string
	WorkspaceResolver  WorkspaceResolver
}

func NewConnectHandler(cfg ConnectHandlerConfig) http.Handler {
	cfg = withConnectHandlerDefaults(cfg)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workspaceID := chi.URLParam(r, "id")
		if workspaceID == "" {
			http.Error(w, "missing workspace id", http.StatusBadRequest)
			return
		}

		ws, err := cfg.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			cfg.Logger.Printf("upgrade workspace websocket for %s: %v", workspaceID, err)
			return
		}

		conn, err := NewMuxConn(ws, cfg.MuxConnConfig)
		if err != nil {
			cfg.Logger.Printf("initialize mux connection for %s: %v", workspaceID, err)
			_ = ws.Close()
			return
		}
		defer conn.Close()

		go func() {
			<-r.Context().Done()
			conn.Close()
		}()
		go conn.StartWritePump()

		session := Session{
			Conn:        conn,
			WorkspaceID: workspaceID,
		}
		if err := readLoop(r.Context(), session, cfg.TerminalHandler, cfg.LSPHandler, cfg.FileHandler); err != nil {
			cfg.Logger.Printf("workspace websocket closed for %s: %v", workspaceID, err)
		}
	})
}

func withConnectHandlerDefaults(cfg ConnectHandlerConfig) ConnectHandlerConfig {
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}

	if cfg.MuxConnConfig.Logger == nil {
		cfg.MuxConnConfig.Logger = cfg.Logger
	}
	cfg.MuxConnConfig = withMuxConnDefaults(cfg.MuxConnConfig)

	if cfg.TerminalHandler == nil {
		resolver := cfg.WorkspaceResolver
		if resolver == nil {
			resolver = StaticWorkspaceResolver{Namespace: cfg.WorkspaceNamespace}
		}
		bridge := NewTerminalBridge(TerminalBridgeConfig{
			Dialer:            cfg.GRPCDialer,
			Logger:            cfg.Logger,
			WorkspaceResolver: resolver,
		})
		cfg.TerminalHandler = bridge.HandleFrame
	}
	if cfg.FileHandler == nil {
		resolver := cfg.WorkspaceResolver
		if resolver == nil {
			resolver = StaticWorkspaceResolver{Namespace: cfg.WorkspaceNamespace}
		}
		cfg.FileHandler = NewFileBridge(FileBridgeConfig{
			Dialer:            cfg.GRPCDialer,
			Logger:            cfg.Logger,
			WorkspaceResolver: resolver,
		}).HandleFrame
	}
	if cfg.LSPHandler == nil {
		resolver := cfg.WorkspaceResolver
		if resolver == nil {
			resolver = StaticWorkspaceResolver{Namespace: cfg.WorkspaceNamespace}
		}
		cfg.LSPHandler = NewLSPBridge(LSPBridgeConfig{
			Dialer:            cfg.GRPCDialer,
			Logger:            cfg.Logger,
			WorkspaceResolver: resolver,
		}).HandleFrame
	}

	if cfg.Upgrader.CheckOrigin == nil {
		cfg.Upgrader.CheckOrigin = func(*http.Request) bool { return true }
	}
	if len(cfg.Upgrader.Subprotocols) == 0 {
		cfg.Upgrader.Subprotocols = []string{cortadoWebSocketProtocol}
	}

	return cfg
}

func readLoop(ctx context.Context, session Session, terminalHandler TerminalFrameHandler, lspHandler LSPFrameHandler, fileHandler FileFrameHandler) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-session.Conn.done:
			return nil
		default:
		}

		messageType, payload, err := session.Conn.ws.ReadMessage()
		if err != nil {
			if isExpectedCloseError(err) {
				return nil
			}
			return fmt.Errorf("read websocket message: %w", err)
		}
		if messageType != websocket.BinaryMessage {
			continue
		}
		receivedAt := time.Now()

		frame, err := DecodeFrame(payload)
		if err != nil {
			return fmt.Errorf("decode mux frame: %w", err)
		}
		if len(frame.Payload) > MaxPayloadSize(frame.ChannelID) {
			session.Conn.SendError(frame.ChannelID, fmt.Sprintf("payload exceeds %d bytes", MaxPayloadSize(frame.ChannelID)))
			continue
		}
		if frame.MessageType == MessageTypePing {
			session.Conn.SendFrame(frame)
			continue
		}

		var handlerErr error
		switch frame.ChannelID {
		case TerminalChannelID:
			handlerErr = terminalHandler(ctx, session, frame, receivedAt)
		case FileSyncChannelID:
			handlerErr = fileHandler(ctx, session, frame, receivedAt)
		default:
			if IsLSPChannel(frame.ChannelID) {
				handlerErr = lspHandler(ctx, session, frame, receivedAt)
				break
			}
			session.Conn.SendError(frame.ChannelID, "unsupported channel")
			continue
		}
		if handlerErr != nil {
			session.Conn.SendError(frame.ChannelID, handlerErr.Error())
		}
	}
}

func isExpectedCloseError(err error) bool {
	return websocket.IsCloseError(
		err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
	) || errors.Is(err, net.ErrClosed)
}
