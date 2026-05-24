package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	defaultLSPQueueSize     = 64
	lspLanguageMetadataName = "x-cortado-lsp-language"
)

type LSPBridgeConfig struct {
	Dialer            GRPCDialFunc
	Logger            *log.Logger
	WorkspaceResolver WorkspaceResolver
}

type LSPBridge struct {
	bindings          map[*MuxConn]map[uint16]*lspBinding
	connCache         map[string]*grpc.ClientConn
	dialer            GRPCDialFunc
	logger            *log.Logger
	mu                sync.Mutex
	workspaceResolver WorkspaceResolver
}

type lspBinding struct {
	cancel      context.CancelFunc
	channelID   uint16
	conn        *MuxConn
	ctx         context.Context
	inbound     chan inboundFrame
	language    string
	release     func()
	releaseOnce sync.Once
	stream      agentpb.WorkspaceAgentService_StreamLSPClient
	workspaceID string
}

func NewLSPBridge(cfg LSPBridgeConfig) *LSPBridge {
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.Dialer == nil {
		cfg.Dialer = grpc.NewClient
	}
	if cfg.WorkspaceResolver == nil {
		cfg.WorkspaceResolver = StaticWorkspaceResolver{}
	}

	return &LSPBridge{
		bindings:          map[*MuxConn]map[uint16]*lspBinding{},
		connCache:         map[string]*grpc.ClientConn{},
		dialer:            cfg.Dialer,
		logger:            cfg.Logger,
		workspaceResolver: cfg.WorkspaceResolver,
	}
}

func (b *LSPBridge) HandleFrame(ctx context.Context, session Session, frame Frame, receivedAt time.Time) error {
	switch frame.MessageType {
	case MessageTypeOpen:
		return b.handleOpen(ctx, session, frame)
	case MessageTypeData, MessageTypeClose:
		binding, ok := b.bindingFor(session.Conn, frame.ChannelID)
		if !ok {
			return fmt.Errorf("lsp channel %d is not open", frame.ChannelID)
		}
		return binding.enqueue(inboundFrame{frame: frame, receivedAt: receivedAt})
	default:
		return fmt.Errorf("unsupported lsp message type %d", frame.MessageType)
	}
}

func (b *LSPBridge) handleOpen(ctx context.Context, session Session, frame Frame) error {
	language := string(frame.Payload)
	if language == "" {
		return errors.New("lsp open payload must contain a language")
	}
	if _, exists := b.bindingFor(session.Conn, frame.ChannelID); exists {
		return fmt.Errorf("lsp channel %d is already open", frame.ChannelID)
	}

	conn, err := b.clientConn(session.WorkspaceID, false)
	if err != nil {
		b.sendClose(session.Conn, frame.ChannelID, fmt.Sprintf("resolve workspace agent: %v", err))
		return nil
	}

	conn, err = b.openLSP(ctx, session.WorkspaceID, conn, language)
	if err != nil {
		b.sendClose(session.Conn, frame.ChannelID, fmt.Sprintf("open lsp: %v", err))
		return nil
	}

	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := b.streamLSP(streamCtx, session.WorkspaceID, conn, language)
	if err != nil {
		cancel()
		b.sendClose(session.Conn, frame.ChannelID, fmt.Sprintf("open lsp stream: %v", err))
		return nil
	}

	binding := &lspBinding{
		cancel:      cancel,
		channelID:   frame.ChannelID,
		conn:        session.Conn,
		ctx:         streamCtx,
		inbound:     make(chan inboundFrame, defaultLSPQueueSize),
		language:    language,
		stream:      stream,
		workspaceID: session.WorkspaceID,
	}
	if err := b.registerBinding(session.Conn, binding); err != nil {
		cancel()
		_ = stream.CloseSend()
		return err
	}

	go binding.forwardInbound(b.logger)
	go binding.forwardOutbound(b.logger)
	go binding.watchConnDone()
	return nil
}

func (b *LSPBridge) openLSP(ctx context.Context, workspaceID string, conn *grpc.ClientConn, language string) (*grpc.ClientConn, error) {
	client := agentpb.NewWorkspaceAgentServiceClient(conn)
	_, err := client.OpenLSP(ctx, &agentpb.OpenLSPRequest{Language: language})
	if err == nil || !shouldRedialWorkspaceConn(err) {
		return conn, err
	}

	refreshedConn, refreshErr := b.clientConn(workspaceID, true)
	if refreshErr != nil {
		return nil, refreshErr
	}

	client = agentpb.NewWorkspaceAgentServiceClient(refreshedConn)
	_, err = client.OpenLSP(ctx, &agentpb.OpenLSPRequest{Language: language})
	return refreshedConn, err
}

func (b *LSPBridge) streamLSP(ctx context.Context, workspaceID string, conn *grpc.ClientConn, language string) (agentpb.WorkspaceAgentService_StreamLSPClient, error) {
	client := agentpb.NewWorkspaceAgentServiceClient(conn)
	streamCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(lspLanguageMetadataName, language))
	stream, err := client.StreamLSP(streamCtx)
	if err == nil || !shouldRedialWorkspaceConn(err) {
		return stream, err
	}

	refreshedConn, refreshErr := b.clientConn(workspaceID, true)
	if refreshErr != nil {
		return nil, refreshErr
	}

	client = agentpb.NewWorkspaceAgentServiceClient(refreshedConn)
	return client.StreamLSP(metadata.NewOutgoingContext(ctx, metadata.Pairs(lspLanguageMetadataName, language)))
}

func (b *LSPBridge) clientConn(workspaceID string, refresh bool) (*grpc.ClientConn, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if refresh {
		b.evictConnLocked(workspaceID)
	}
	if conn, ok := b.connCache[workspaceID]; ok {
		return conn, nil
	}

	target := fmt.Sprintf("%s:%d", b.workspaceResolver.GetServiceDNS(workspaceID), defaultAgentAddressPort)
	conn, err := b.dialer(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial workspace agent %q: %w", target, err)
	}

	b.connCache[workspaceID] = conn
	return conn, nil
}

func (b *LSPBridge) evictConnLocked(workspaceID string) {
	conn, ok := b.connCache[workspaceID]
	if !ok {
		return
	}
	delete(b.connCache, workspaceID)
	_ = conn.Close()
}

func (b *LSPBridge) registerBinding(conn *MuxConn, binding *lspBinding) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.bindings[conn]; !ok {
		b.bindings[conn] = map[uint16]*lspBinding{}
	}
	if _, exists := b.bindings[conn][binding.channelID]; exists {
		return fmt.Errorf("lsp channel %d is already open", binding.channelID)
	}

	binding.release = func() {
		b.mu.Lock()
		if current, ok := b.bindings[conn][binding.channelID]; ok && current == binding {
			delete(b.bindings[conn], binding.channelID)
			if len(b.bindings[conn]) == 0 {
				delete(b.bindings, conn)
			}
		}
		b.mu.Unlock()

		binding.cancel()
		_ = binding.stream.CloseSend()
	}

	b.bindings[conn][binding.channelID] = binding
	return nil
}

func (b *LSPBridge) bindingFor(conn *MuxConn, channelID uint16) (*lspBinding, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	channelBindings, ok := b.bindings[conn]
	if !ok {
		return nil, false
	}
	binding, ok := channelBindings[channelID]
	return binding, ok
}

func (b *LSPBridge) sendClose(conn *MuxConn, channelID uint16, message string) {
	conn.SendFrame(Frame{
		ChannelID:   channelID,
		MessageType: MessageTypeClose,
		Payload:     []byte(message),
	})
}

func (b *lspBinding) enqueue(frame inboundFrame) error {
	select {
	case <-b.ctx.Done():
		return errors.New("lsp stream is closed")
	default:
	}

	select {
	case b.inbound <- frame:
		return nil
	case <-b.ctx.Done():
		return errors.New("lsp stream is closed")
	}
}

func (b *lspBinding) forwardInbound(logger *log.Logger) {
	defer b.close()

	for {
		select {
		case <-b.ctx.Done():
			return
		case item := <-b.inbound:
			switch item.frame.MessageType {
			case MessageTypeData:
				if err := b.stream.Send(&agentpb.LSPMessage{
					Data: append([]byte(nil), item.frame.Payload...),
				}); err != nil {
					b.conn.SendFrame(ErrorFrame(b.channelID, fmt.Sprintf("send lsp data: %v", err)))
					return
				}
				logger.Printf(
					"lsp grpc send workspace=%s channel=%d language=%s latency=%s",
					b.workspaceID,
					b.channelID,
					b.language,
					time.Since(item.receivedAt),
				)
			case MessageTypeClose:
				return
			default:
				b.conn.SendError(b.channelID, fmt.Sprintf("unsupported lsp message type %d", item.frame.MessageType))
				return
			}
		}
	}
}

func (b *lspBinding) forwardOutbound(logger *log.Logger) {
	defer b.close()

	for {
		resp, err := b.stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(b.ctx.Err(), context.Canceled) {
				b.conn.SendFrame(Frame{
					ChannelID:   b.channelID,
					MessageType: MessageTypeClose,
				})
				return
			}
			b.conn.SendFrame(Frame{
				ChannelID:   b.channelID,
				MessageType: MessageTypeClose,
				Payload:     []byte(fmt.Sprintf("lsp stream closed: %v", err)),
			})
			return
		}

		logger.Printf("lsp grpc recv workspace=%s channel=%d language=%s", b.workspaceID, b.channelID, b.language)
		if !b.conn.SendFrame(Frame{
			ChannelID:   b.channelID,
			MessageType: MessageTypeData,
			Payload:     append([]byte(nil), resp.GetData()...),
		}) {
			return
		}
	}
}

func (b *lspBinding) watchConnDone() {
	select {
	case <-b.ctx.Done():
	case <-b.conn.done:
	}
	b.close()
}

func (b *lspBinding) close() {
	b.releaseOnce.Do(func() {
		if b.release != nil {
			b.release()
		}
	})
}
