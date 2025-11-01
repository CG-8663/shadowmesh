package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// EncodeHeader encodes a message header to binary format
// Format: [Version:1][Type:1][Flags:2][Length:4] = 8 bytes
func EncodeHeader(h Header) []byte {
	buf := make([]byte, HeaderSize)
	buf[0] = h.Version
	buf[1] = h.Type
	binary.BigEndian.PutUint16(buf[2:4], h.Flags)
	binary.BigEndian.PutUint32(buf[4:8], h.Length)
	return buf
}

// DecodeHeader decodes a message header from binary format
func DecodeHeader(data []byte) (Header, error) {
	if len(data) < HeaderSize {
		return Header{}, fmt.Errorf("insufficient data for header: got %d bytes, need %d", len(data), HeaderSize)
	}

	h := Header{
		Version: data[0],
		Type:    data[1],
		Flags:   binary.BigEndian.Uint16(data[2:4]),
		Length:  binary.BigEndian.Uint32(data[4:8]),
	}

	// Validate protocol version
	if h.Version != ProtocolVersion {
		return h, fmt.Errorf("unsupported protocol version: got 0x%02x, expected 0x%02x", h.Version, ProtocolVersion)
	}

	// Validate message length
	if h.Length > MaxMessageSize {
		return h, fmt.Errorf("message too large: %d bytes (max %d)", h.Length, MaxMessageSize)
	}

	return h, nil
}

// ReadHeader reads a header from an io.Reader
func ReadHeader(r io.Reader) (Header, error) {
	buf := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return Header{}, fmt.Errorf("failed to read header: %w", err)
	}
	return DecodeHeader(buf)
}

// WriteHeader writes a header to an io.Writer
func WriteHeader(w io.Writer, h Header) error {
	buf := EncodeHeader(h)
	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	return nil
}

// NewHeader creates a new header with the protocol version set
func NewHeader(msgType byte, flags uint16, payloadLen uint32) Header {
	return Header{
		Version: ProtocolVersion,
		Type:    msgType,
		Flags:   flags,
		Length:  payloadLen,
	}
}

// ValidateMessageType checks if the message type is valid
func ValidateMessageType(msgType byte) error {
	switch msgType {
	case MsgTypeHello, MsgTypeChallenge, MsgTypeResponse, MsgTypeEstablished,
		MsgTypeHeartbeat, MsgTypeHeartbeatAck, MsgTypeError, MsgTypeClose,
		MsgTypeDataFrame, MsgTypeMultiHop,
		MsgTypeConfigUpdate, MsgTypeStatsRequest, MsgTypeStatsResponse:
		return nil
	default:
		return fmt.Errorf("invalid message type: 0x%02x", msgType)
	}
}

// String returns a human-readable representation of the header
func (h Header) String() string {
	return fmt.Sprintf("Header{Version: 0x%02x, Type: %s (0x%02x), Flags: 0x%04x, Length: %d}",
		h.Version, MessageTypeName(h.Type), h.Type, h.Flags, h.Length)
}
