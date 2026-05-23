package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

const (
	defaultAgentAddressPort  = 9090
	defaultPtyCols           = 80
	defaultPtyRows           = 24
	defaultTerminalQueueSize = 64
	defaultCreatePtyTimeout  = 10 * time.Second
	defaultClusterDNSDomain  = "cluster.local"
	defaultWorkspaceNS       = "cortado-workspaces"
)

type WorkspaceResolver interface {
	GetServiceDNS(workspaceID string) string
}

type StaticWorkspaceResolver struct {
	Namespace string
	DNSDomain string
}

func (r StaticWorkspaceResolver) GetServiceDNS(workspaceID string) string {
	namespace := r.Namespace
	if namespace == "" {
		namespace = defaultWorkspaceNS
	}

	dnsDomain := r.DNSDomain
	if dnsDomain == "" {
		dnsDomain = defaultClusterDNSDomain
	}

	return fmt.Sprintf("%s.%s.svc.%s", workspaceID, namespace, dnsDomain)
}

type GRPCDialFunc func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)

type TerminalBridgeConfig struct {
	CreatePtyTimeout  time.Duration
	Dialer            GRPCDialFunc
	Logger            *log.Logger
	WorkspaceResolver WorkspaceResolver
}

type TerminalBridge struct {
	connCache         map[string]*grpc.ClientConn
	createPtyTimeout  time.Duration
	dialer            GRPCDialFunc
	logger            *log.Logger
	mu                sync.Mutex
	workspaceResolver WorkspaceResolver
	bindings          map[*MuxConn]*terminalBinding
}

type inboundFrame struct {
	frame      Frame
	receivedAt time.Time
}

type terminalBinding struct {
	cancel      context.CancelFunc
	channelID   uint16
	conn        *MuxConn
	ctx         context.Context
	inbound     chan inboundFrame
	release     func()
	releaseOnce sync.Once
	stream      agentpb.WorkspaceAgentService_StreamPtyClient
	workspaceID string
}

func NewTerminalBridge(cfg TerminalBridgeConfig) *TerminalBridge {
	cfg = withTerminalBridgeDefaults(cfg)

	return &TerminalBridge{
		connCache:         map[string]*grpc.ClientConn{},
		createPtyTimeout:  cfg.CreatePtyTimeout,
		dialer:            cfg.Dialer,
		logger:            cfg.Logger,
		workspaceResolver: cfg.WorkspaceResolver,
		bindings:          map[*MuxConn]*terminalBinding{},
	}
}

func withTerminalBridgeDefaults(cfg TerminalBridgeConfig) TerminalBridgeConfig {
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.Dialer == nil {
		cfg.Dialer = grpc.NewClient
	}
	if cfg.CreatePtyTimeout <= 0 {
		cfg.CreatePtyTimeout = defaultCreatePtyTimeout
	}
	if cfg.WorkspaceResolver == nil {
		cfg.WorkspaceResolver = StaticWorkspaceResolver{}
	}
	return cfg
}

func (b *TerminalBridge) HandleFrame(ctx context.Context, session Session, frame Frame, receivedAt time.Time) error {
	switch frame.MessageType {
	case MessageTypeOpen:
		return b.handleOpen(ctx, session, frame)
	case MessageTypeData, MessageTypeClose, MessageTypeResize:
		binding, ok := b.bindingFor(session.Conn)
		if !ok {
			return errors.New("terminal channel is not open")
		}
		if binding.channelID != frame.ChannelID {
			return fmt.Errorf("terminal channel %d is not open", frame.ChannelID)
		}
		return binding.enqueue(inboundFrame{frame: frame, receivedAt: receivedAt})
	default:
		return fmt.Errorf("unsupported terminal message type %d", frame.MessageType)
	}
}

func (b *TerminalBridge) handleOpen(ctx context.Context, session Session, frame Frame) error {
	if _, exists := b.bindingFor(session.Conn); exists {
		return errors.New("terminal channel is already open")
	}

	conn, err := b.clientConn(session.WorkspaceID, false)
	if err != nil {
		b.sendClose(session.Conn, frame.ChannelID, fmt.Sprintf("resolve workspace agent: %v", err))
		return nil
	}

	createCtx, cancel := context.WithTimeout(ctx, b.createPtyTimeout)
	defer cancel()

	conn, createResp, err := b.createPty(createCtx, session.WorkspaceID, conn, string(frame.Payload))
	if err != nil {
		b.sendClose(session.Conn, frame.ChannelID, fmt.Sprintf("open terminal: %v", err))
		return nil
	}

	client := agentpb.NewWorkspaceAgentServiceClient(conn)
	streamCtx, streamCancel := context.WithCancel(ctx)
	stream, err := client.StreamPty(streamCtx)
	if err != nil {
		streamCancel()
		b.sendClose(session.Conn, frame.ChannelID, fmt.Sprintf("open terminal stream: %v", err))
		return nil
	}
	if err := stream.Send(&agentpb.StreamPtyRequest{PtyId: createResp.GetPtyId()}); err != nil {
		streamCancel()
		_ = stream.CloseSend()
		b.sendClose(session.Conn, frame.ChannelID, fmt.Sprintf("bind terminal stream: %v", err))
		return nil
	}

	binding := &terminalBinding{
		cancel:      streamCancel,
		channelID:   frame.ChannelID,
		conn:        session.Conn,
		ctx:         streamCtx,
		inbound:     make(chan inboundFrame, defaultTerminalQueueSize),
		stream:      stream,
		workspaceID: session.WorkspaceID,
	}
	if err := b.registerBinding(session.Conn, binding); err != nil {
		streamCancel()
		_ = stream.CloseSend()
		return err
	}

	go binding.forwardInbound(b.logger)
	go binding.forwardOutbound(b.logger)
	go binding.watchConnDone()

	return nil
}

func (b *TerminalBridge) createPty(ctx context.Context, workspaceID string, conn *grpc.ClientConn, shell string) (*grpc.ClientConn, *agentpb.CreatePtyResponse, error) {
	request := &agentpb.CreatePtyRequest{
		Cols:  defaultPtyCols,
		Rows:  defaultPtyRows,
		Shell: shell,
	}

	client := agentpb.NewWorkspaceAgentServiceClient(conn)
	response, err := client.CreatePty(ctx, request)
	if err == nil || !shouldRedialWorkspaceConn(err) {
		return conn, response, err
	}

	refreshedConn, refreshErr := b.clientConn(workspaceID, true)
	if refreshErr != nil {
		return nil, nil, refreshErr
	}

	client = agentpb.NewWorkspaceAgentServiceClient(refreshedConn)
	response, err = client.CreatePty(ctx, request)
	return refreshedConn, response, err
}

func shouldRedialWorkspaceConn(err error) bool {
	switch status.Code(err) {
	case codes.DeadlineExceeded, codes.Unavailable:
		return true
	default:
		return false
	}
}

func (b *TerminalBridge) clientConn(workspaceID string, refresh bool) (*grpc.ClientConn, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if refresh {
		b.evictConnLocked(workspaceID)
	}

	if conn, ok := b.connCache[workspaceID]; ok {
		return conn, nil
	}

	target := fmt.Sprintf("%s:%d", b.workspaceResolver.GetServiceDNS(workspaceID), defaultAgentAddressPort)
	conn, err := b.dialer(
		target,
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial workspace agent %q: %w", target, err)
	}

	b.connCache[workspaceID] = conn
	return conn, nil
}

func (b *TerminalBridge) evictConnLocked(workspaceID string) {
	conn, ok := b.connCache[workspaceID]
	if !ok {
		return
	}
	delete(b.connCache, workspaceID)
	_ = conn.Close()
}

func (b *TerminalBridge) registerBinding(conn *MuxConn, binding *terminalBinding) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.bindings[conn]; exists {
		return errors.New("terminal channel is already open")
	}

	binding.release = func() {
		b.mu.Lock()
		if current, ok := b.bindings[conn]; ok && current == binding {
			delete(b.bindings, conn)
		}
		b.mu.Unlock()

		binding.cancel()
		_ = binding.stream.CloseSend()
	}

	b.bindings[conn] = binding
	return nil
}

func (b *TerminalBridge) bindingFor(conn *MuxConn) (*terminalBinding, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	binding, ok := b.bindings[conn]
	return binding, ok
}

func (b *TerminalBridge) sendClose(conn *MuxConn, channelID uint16, message string) {
	conn.SendFrame(Frame{
		ChannelID:   channelID,
		MessageType: MessageTypeClose,
		Payload:     []byte(message),
	})
}

func (b *terminalBinding) enqueue(frame inboundFrame) error {
	select {
	case <-b.ctx.Done():
		return errors.New("terminal stream is closed")
	default:
	}

	select {
	case b.inbound <- frame:
		return nil
	case <-b.ctx.Done():
		return errors.New("terminal stream is closed")
	}
}

func (b *terminalBinding) forwardInbound(logger *log.Logger) {
	defer b.close()

	for {
		select {
		case <-b.ctx.Done():
			return
		case item := <-b.inbound:
			switch item.frame.MessageType {
			case MessageTypeData:
				if err := b.stream.Send(&agentpb.StreamPtyRequest{
					Payload: &agentpb.StreamPtyRequest_Data{
						Data: append([]byte(nil), item.frame.Payload...),
					},
				}); err != nil {
					b.conn.SendFrame(ErrorFrame(b.channelID, fmt.Sprintf("send terminal data: %v", err)))
					return
				}
				logger.Printf(
					"terminal grpc send workspace=%s channel=%d latency=%s",
					b.workspaceID,
					b.channelID,
					time.Since(item.receivedAt),
				)
			case MessageTypeResize:
				size, err := DecodeTerminalResizePayload(item.frame.Payload)
				if err != nil {
					b.conn.SendError(b.channelID, fmt.Sprintf("decode terminal resize: %v", err))
					return
				}
				if size.Cols == 0 || size.Rows == 0 {
					b.conn.SendError(b.channelID, "terminal resize cols and rows must be greater than zero")
					return
				}
				if size.Cols > math.MaxUint16 || size.Rows > math.MaxUint16 {
					b.conn.SendError(b.channelID, "terminal resize cols and rows exceed PTY limits")
					return
				}
				if err := b.stream.Send(&agentpb.StreamPtyRequest{
					Payload: &agentpb.StreamPtyRequest_Resize{
						Resize: &agentpb.WindowSize{
							Cols: size.Cols,
							Rows: size.Rows,
						},
					},
				}); err != nil {
					b.conn.SendFrame(ErrorFrame(b.channelID, fmt.Sprintf("send terminal resize: %v", err)))
					return
				}
			case MessageTypeClose:
				return
			default:
				b.conn.SendError(b.channelID, fmt.Sprintf("unsupported terminal message type %d", item.frame.MessageType))
				return
			}
		}
	}
}

func (b *terminalBinding) forwardOutbound(logger *log.Logger) {
	defer b.close()

	for {
		resp, err := b.stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(b.ctx.Err(), context.Canceled) || status.Code(err) == codes.Canceled {
				return
			}
			b.conn.SendFrame(Frame{
				ChannelID:   b.channelID,
				MessageType: MessageTypeClose,
				Payload:     []byte(fmt.Sprintf("terminal stream closed: %v", err)),
			})
			return
		}

		switch payload := resp.GetPayload().(type) {
		case *agentpb.StreamPtyResponse_Data:
			b.conn.SendFrame(Frame{
				ChannelID:   b.channelID,
				MessageType: MessageTypeData,
				Payload:     append([]byte(nil), payload.Data...),
			})
		case *agentpb.StreamPtyResponse_ExitCode:
			b.conn.SendFrame(Frame{
				ChannelID:   b.channelID,
				MessageType: MessageTypeClose,
				Payload:     []byte(strconv.FormatInt(int64(payload.ExitCode), 10)),
			})
			return
		default:
			logger.Printf("terminal stream received unknown payload for workspace=%s channel=%d", b.workspaceID, b.channelID)
			b.conn.SendError(b.channelID, "unknown terminal stream payload")
			return
		}
	}
}

func (b *terminalBinding) watchConnDone() {
	select {
	case <-b.ctx.Done():
	case <-b.conn.done:
	}
	b.close()
}

func (b *terminalBinding) close() {
	b.releaseOnce.Do(func() {
		if b.release != nil {
			b.release()
		}
	})
}
