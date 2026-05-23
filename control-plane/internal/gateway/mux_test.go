package gateway

import (
	"io"
	"log"
	"testing"
)

func TestEncodeDecodeFrameRoundTrip(t *testing.T) {
	frame := Frame{
		ChannelID:   TerminalChannelID,
		MessageType: MessageTypeData,
		Payload:     []byte("hello"),
	}

	raw, err := EncodeFrame(frame)
	if err != nil {
		t.Fatalf("encode frame: %v", err)
	}

	decoded, err := DecodeFrame(raw)
	if err != nil {
		t.Fatalf("decode frame: %v", err)
	}

	if decoded.ChannelID != frame.ChannelID {
		t.Fatalf("unexpected channel id: got %d want %d", decoded.ChannelID, frame.ChannelID)
	}
	if decoded.MessageType != frame.MessageType {
		t.Fatalf("unexpected message type: got %d want %d", decoded.MessageType, frame.MessageType)
	}
	if string(decoded.Payload) != string(frame.Payload) {
		t.Fatalf("unexpected payload: got %q want %q", decoded.Payload, frame.Payload)
	}
}

func TestDecodeFrameRejectsPayloadLengthMismatch(t *testing.T) {
	raw, err := EncodeFrame(Frame{
		ChannelID:   TerminalChannelID,
		MessageType: MessageTypeOpen,
		Payload:     []byte("abc"),
	})
	if err != nil {
		t.Fatalf("encode frame: %v", err)
	}

	if _, err := DecodeFrame(raw[:len(raw)-1]); err == nil {
		t.Fatal("expected decode error for truncated frame")
	}
}

func TestEncodeDecodeTerminalResizePayloadRoundTrip(t *testing.T) {
	payload := EncodeTerminalResizePayload(TerminalResize{
		Cols: 120,
		Rows: 40,
	})

	size, err := DecodeTerminalResizePayload(payload)
	if err != nil {
		t.Fatalf("decode resize payload: %v", err)
	}

	if size.Cols != 120 {
		t.Fatalf("unexpected cols: got %d want %d", size.Cols, 120)
	}
	if size.Rows != 40 {
		t.Fatalf("unexpected rows: got %d want %d", size.Rows, 40)
	}
}

func TestMuxConnEnqueueDropsWhenQueueIsFull(t *testing.T) {
	conn := &MuxConn{
		logger:  log.New(io.Discard, "", 0),
		writeCh: make(chan []byte, 1),
		done:    make(chan struct{}),
	}

	if ok := conn.enqueue([]byte("first")); !ok {
		t.Fatal("expected first enqueue to succeed")
	}
	if ok := conn.enqueue([]byte("second")); ok {
		t.Fatal("expected second enqueue to be dropped")
	}
}
