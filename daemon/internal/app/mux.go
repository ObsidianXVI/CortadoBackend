package app

import (
	"encoding/binary"
	"fmt"
	"math"
)

const (
	frameHeaderLen          = 7
	conflictNoticeChannelID = 0x0600
)

type MessageType uint8

const (
	MessageTypeData MessageType = 0x01
)

type Frame struct {
	ChannelID   uint16
	MessageType MessageType
	Payload     []byte
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
		return Frame{}, fmt.Errorf("frame shorter than %d-byte header", frameHeaderLen)
	}

	payloadLen := binary.BigEndian.Uint32(raw[3:7])
	if len(raw) != frameHeaderLen+int(payloadLen) {
		return Frame{}, fmt.Errorf("payload length mismatch")
	}

	frame := Frame{
		ChannelID:   binary.BigEndian.Uint16(raw[0:2]),
		MessageType: MessageType(raw[2]),
		Payload:     make([]byte, payloadLen),
	}
	copy(frame.Payload, raw[frameHeaderLen:])
	return frame, nil
}
