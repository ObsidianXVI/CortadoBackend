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
	"google.golang.org/protobuf/proto"
)

type FileBridgeConfig struct {
	Dialer            GRPCDialFunc
	Logger            *log.Logger
	WorkspaceResolver WorkspaceResolver
}

type FileBridge struct {
	bindings          map[*MuxConn]*fileBinding
	connCache         map[string]*grpc.ClientConn
	dialer            GRPCDialFunc
	logger            *log.Logger
	mu                sync.Mutex
	workspaceResolver WorkspaceResolver
}

type fileBinding struct {
	cancel      context.CancelFunc
	channelID   uint16
	conn        *MuxConn
	release     func()
	releaseOnce sync.Once
	workspaceID string
}

func NewFileBridge(cfg FileBridgeConfig) *FileBridge {
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.Dialer == nil {
		cfg.Dialer = grpc.NewClient
	}
	if cfg.WorkspaceResolver == nil {
		cfg.WorkspaceResolver = StaticWorkspaceResolver{}
	}

	return &FileBridge{
		bindings:          map[*MuxConn]*fileBinding{},
		connCache:         map[string]*grpc.ClientConn{},
		dialer:            cfg.Dialer,
		logger:            cfg.Logger,
		workspaceResolver: cfg.WorkspaceResolver,
	}
}

func (b *FileBridge) HandleFrame(ctx context.Context, session Session, frame Frame, _ time.Time) error {
	switch frame.MessageType {
	case MessageTypeOpen:
		return b.handleOpen(ctx, session, frame)
	case MessageTypeClose:
		binding, ok := b.bindingFor(session.Conn)
		if !ok || binding.channelID != frame.ChannelID {
			return fmt.Errorf("file sync channel %d is not open", frame.ChannelID)
		}
		binding.releaseWithClose("")
		return nil
	default:
		return fmt.Errorf("unsupported file sync message type %d", frame.MessageType)
	}
}

func (b *FileBridge) handleOpen(ctx context.Context, session Session, frame Frame) error {
	if _, exists := b.bindingFor(session.Conn); exists {
		return errors.New("file sync channel is already open")
	}

	conn, err := b.clientConn(session.WorkspaceID, false)
	if err != nil {
		b.sendClose(session.Conn, frame.ChannelID, fmt.Sprintf("resolve workspace agent: %v", err))
		return nil
	}

	streamCtx, cancel := context.WithCancel(ctx)
	_, stream, err := b.watchFiles(streamCtx, session.WorkspaceID, conn)
	if err != nil {
		cancel()
		b.sendClose(session.Conn, frame.ChannelID, fmt.Sprintf("open file watch: %v", err))
		return nil
	}

	binding := &fileBinding{
		cancel:      cancel,
		channelID:   frame.ChannelID,
		conn:        session.Conn,
		workspaceID: session.WorkspaceID,
	}
	if err := b.registerBinding(session.Conn, binding); err != nil {
		cancel()
		return err
	}

	go b.forwardOutbound(binding, stream)
	go binding.watchConnDone()

	return nil
}

func (b *FileBridge) watchFiles(ctx context.Context, workspaceID string, conn *grpc.ClientConn) (*grpc.ClientConn, agentpb.WorkspaceAgentService_WatchFilesClient, error) {
	client := agentpb.NewWorkspaceAgentServiceClient(conn)
	stream, err := client.WatchFiles(ctx, &agentpb.WatchFilesRequest{})
	if err == nil || !shouldRedialWorkspaceConn(err) {
		return conn, stream, err
	}

	refreshedConn, refreshErr := b.clientConn(workspaceID, true)
	if refreshErr != nil {
		return nil, nil, refreshErr
	}

	client = agentpb.NewWorkspaceAgentServiceClient(refreshedConn)
	stream, err = client.WatchFiles(ctx, &agentpb.WatchFilesRequest{})
	return refreshedConn, stream, err
}

func (b *FileBridge) clientConn(workspaceID string, refresh bool) (*grpc.ClientConn, error) {
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

func (b *FileBridge) evictConnLocked(workspaceID string) {
	conn, ok := b.connCache[workspaceID]
	if !ok {
		return
	}
	delete(b.connCache, workspaceID)
	_ = conn.Close()
}

func (b *FileBridge) registerBinding(conn *MuxConn, binding *fileBinding) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.bindings[conn]; exists {
		return errors.New("file sync connection is already bound")
	}

	binding.release = func() {
		b.mu.Lock()
		delete(b.bindings, conn)
		b.mu.Unlock()
		binding.cancel()
	}
	b.bindings[conn] = binding
	return nil
}

func (b *FileBridge) bindingFor(conn *MuxConn) (*fileBinding, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	binding, ok := b.bindings[conn]
	return binding, ok
}

func (b *FileBridge) forwardOutbound(binding *fileBinding, stream agentpb.WorkspaceAgentService_WatchFilesClient) {
	defer binding.releaseOnly()

	for {
		response, err := stream.Recv()
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
				b.sendClose(binding.conn, binding.channelID, "")
				return
			}
			binding.conn.SendError(binding.channelID, fmt.Sprintf("file watch recv: %v", err))
			b.sendClose(binding.conn, binding.channelID, "")
			return
		}

		event := response.GetEvent()
		if event == nil {
			continue
		}

		payload, err := proto.Marshal(event)
		if err != nil {
			binding.conn.SendError(binding.channelID, fmt.Sprintf("marshal file event: %v", err))
			b.sendClose(binding.conn, binding.channelID, "")
			return
		}

		if !binding.conn.SendFrame(Frame{
			ChannelID:   binding.channelID,
			MessageType: MessageTypeData,
			Payload:     payload,
		}) {
			return
		}
	}
}

func (b *FileBridge) sendClose(conn *MuxConn, channelID uint16, message string) {
	conn.SendFrame(Frame{
		ChannelID:   channelID,
		MessageType: MessageTypeClose,
		Payload:     []byte(message),
	})
}

func (b *fileBinding) releaseOnly() {
	b.releaseOnce.Do(func() {
		if b.release != nil {
			b.release()
		}
	})
}

func (b *fileBinding) releaseWithClose(message string) {
	b.releaseOnly()
	b.conn.SendFrame(Frame{
		ChannelID:   b.channelID,
		MessageType: MessageTypeClose,
		Payload:     []byte(message),
	})
}

func (b *fileBinding) watchConnDone() {
	<-b.conn.done
	b.releaseOnly()
}
