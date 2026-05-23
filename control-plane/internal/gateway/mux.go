package gateway

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	frameHeaderLen      = 7
	terminalResizeLen   = 8
	TerminalChannelID   = 0x0001
	defaultPingInterval = 20 * time.Second
	defaultPongWait     = 60 * time.Second
	defaultWriteTimeout = 10 * time.Second
	defaultWriteQueue   = 64
)

var errInvalidFrame = errors.New("invalid mux frame")

type MessageType uint8

const (
	MessageTypeData   MessageType = 0x01
	MessageTypeOpen   MessageType = 0x02
	MessageTypeClose  MessageType = 0x03
	MessageTypeError  MessageType = 0x04
	MessageTypeResize MessageType = 0x05
	MessageTypePing   MessageType = 0xFF
)

type Frame struct {
	ChannelID   uint16
	MessageType MessageType
	Payload     []byte
}

type TerminalResize struct {
	Cols uint32
	Rows uint32
}

type MuxConnConfig struct {
	Logger             *log.Logger
	PingInterval       time.Duration
	PongWait           time.Duration
	WriteQueueCapacity int
	WriteTimeout       time.Duration
}

type MuxConn struct {
	ws           *websocket.Conn
	logger       *log.Logger
	writeCh      chan []byte
	done         chan struct{}
	pingInterval time.Duration
	pongWait     time.Duration
	writeTimeout time.Duration
	closeOnce    sync.Once
}

func NewMuxConn(ws *websocket.Conn, cfg MuxConnConfig) (*MuxConn, error) {
	cfg = withMuxConnDefaults(cfg)

	conn := &MuxConn{
		ws:           ws,
		logger:       cfg.Logger,
		writeCh:      make(chan []byte, cfg.WriteQueueCapacity),
		done:         make(chan struct{}),
		pingInterval: cfg.PingInterval,
		pongWait:     cfg.PongWait,
		writeTimeout: cfg.WriteTimeout,
	}

	if err := conn.ws.SetReadDeadline(time.Now().Add(conn.pongWait)); err != nil {
		return nil, fmt.Errorf("set initial read deadline: %w", err)
	}
	conn.ws.SetPongHandler(func(string) error {
		return conn.ws.SetReadDeadline(time.Now().Add(conn.pongWait))
	})

	return conn, nil
}

func withMuxConnDefaults(cfg MuxConnConfig) MuxConnConfig {
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.PingInterval <= 0 {
		cfg.PingInterval = defaultPingInterval
	}
	if cfg.PongWait <= 0 {
		cfg.PongWait = defaultPongWait
	}
	if cfg.WriteQueueCapacity <= 0 {
		cfg.WriteQueueCapacity = defaultWriteQueue
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = defaultWriteTimeout
	}
	return cfg
}

func EncodeFrame(frame Frame) ([]byte, error) {
	if len(frame.Payload) > math.MaxUint32 {
		return nil, fmt.Errorf("payload too large: %d", len(frame.Payload))
	}

	buf := make([]byte, frameHeaderLen+len(frame.Payload))
	binary.BigEndian.PutUint16(buf[0:2], frame.ChannelID)
	buf[2] = byte(frame.MessageType)
	binary.BigEndian.PutUint32(buf[3:7], uint32(len(frame.Payload)))
	copy(buf[frameHeaderLen:], frame.Payload)

	return buf, nil
}

func DecodeFrame(raw []byte) (Frame, error) {
	if len(raw) < frameHeaderLen {
		return Frame{}, fmt.Errorf("%w: frame shorter than %d-byte header", errInvalidFrame, frameHeaderLen)
	}

	payloadLen := binary.BigEndian.Uint32(raw[3:7])
	if len(raw) != frameHeaderLen+int(payloadLen) {
		return Frame{}, fmt.Errorf("%w: payload length mismatch", errInvalidFrame)
	}

	frame := Frame{
		ChannelID:   binary.BigEndian.Uint16(raw[0:2]),
		MessageType: MessageType(raw[2]),
		Payload:     make([]byte, payloadLen),
	}
	copy(frame.Payload, raw[frameHeaderLen:])

	return frame, nil
}

func ErrorFrame(channelID uint16, message string) Frame {
	return Frame{
		ChannelID:   channelID,
		MessageType: MessageTypeError,
		Payload:     []byte(message),
	}
}

func EncodeTerminalResizePayload(size TerminalResize) []byte {
	payload := make([]byte, terminalResizeLen)
	binary.BigEndian.PutUint32(payload[0:4], size.Cols)
	binary.BigEndian.PutUint32(payload[4:8], size.Rows)
	return payload
}

func DecodeTerminalResizePayload(payload []byte) (TerminalResize, error) {
	if len(payload) != terminalResizeLen {
		return TerminalResize{}, fmt.Errorf(
			"%w: resize payload must be %d bytes",
			errInvalidFrame,
			terminalResizeLen,
		)
	}

	return TerminalResize{
		Cols: binary.BigEndian.Uint32(payload[0:4]),
		Rows: binary.BigEndian.Uint32(payload[4:8]),
	}, nil
}

func (c *MuxConn) StartWritePump() {
	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()
	defer c.Close()

	for {
		select {
		case frame := <-c.writeCh:
			if err := c.ws.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
				c.logger.Printf("mux write pump: set binary write deadline: %v", err)
				return
			}
			if err := c.ws.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				c.logger.Printf("mux write pump: write binary frame: %v", err)
				return
			}
		case <-ticker.C:
			if err := c.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(c.writeTimeout)); err != nil {
				c.logger.Printf("mux write pump: write ping: %v", err)
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *MuxConn) SendFrame(frame Frame) bool {
	encoded, err := EncodeFrame(frame)
	if err != nil {
		c.logger.Printf("mux enqueue: encode frame: %v", err)
		return false
	}

	return c.enqueue(encoded)
}

func (c *MuxConn) SendError(channelID uint16, message string) bool {
	return c.SendFrame(ErrorFrame(channelID, message))
}

func (c *MuxConn) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.ws.Close()
	})
}

func (c *MuxConn) enqueue(frame []byte) bool {
	select {
	case <-c.done:
		return false
	default:
	}

	select {
	case c.writeCh <- frame:
		return true
	default:
		c.logger.Printf("mux enqueue: dropping frame because write queue is full")
		return false
	}
}
