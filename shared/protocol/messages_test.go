package protocol

import (
	"bytes"
	"testing"
)

func TestHeaderEncodeDecode(t *testing.T) {
	tests := []struct {
		name    string
		header  Header
		wantErr bool
	}{
		{
			name: "valid HELLO header",
			header: Header{
				Version: ProtocolVersion,
				Type:    MsgTypeHello,
				Flags:   FlagNone,
				Length:  1234,
			},
			wantErr: false,
		},
		{
			name: "valid DATA_FRAME header with flags",
			header: Header{
				Version: ProtocolVersion,
				Type:    MsgTypeDataFrame,
				Flags:   FlagKeyRotation,
				Length:  1524,
			},
			wantErr: false,
		},
		{
			name: "maximum message size",
			header: Header{
				Version: ProtocolVersion,
				Type:    MsgTypeEstablished,
				Flags:   FlagNone,
				Length:  MaxMessageSize,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode header
			encoded := EncodeHeader(tt.header)

			if len(encoded) != HeaderSize {
				t.Errorf("encoded header size = %d, want %d", len(encoded), HeaderSize)
			}

			// Decode header
			decoded, err := DecodeHeader(encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if decoded.Version != tt.header.Version {
					t.Errorf("Version = %d, want %d", decoded.Version, tt.header.Version)
				}
				if decoded.Type != tt.header.Type {
					t.Errorf("Type = %d, want %d", decoded.Type, tt.header.Type)
				}
				if decoded.Flags != tt.header.Flags {
					t.Errorf("Flags = %d, want %d", decoded.Flags, tt.header.Flags)
				}
				if decoded.Length != tt.header.Length {
					t.Errorf("Length = %d, want %d", decoded.Length, tt.header.Length)
				}
			}
		})
	}
}

func TestMessageTypeName(t *testing.T) {
	tests := []struct {
		msgType byte
		want    string
	}{
		{MsgTypeHello, "HELLO"},
		{MsgTypeChallenge, "CHALLENGE"},
		{MsgTypeResponse, "RESPONSE"},
		{MsgTypeEstablished, "ESTABLISHED"},
		{MsgTypeHeartbeat, "HEARTBEAT"},
		{MsgTypeDataFrame, "DATA_FRAME"},
		{MsgTypeError, "ERROR"},
		{MsgTypeClose, "CLOSE"},
		{0xFF, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := MessageTypeName(tt.msgType)
			if got != tt.want {
				t.Errorf("MessageTypeName(%d) = %s, want %s", tt.msgType, got, tt.want)
			}
		})
	}
}

func TestResponseMessageEncodeDecode(t *testing.T) {
	var sessionID [16]byte
	copy(sessionID[:], []byte("test-session-id0"))

	var proof [32]byte
	copy(proof[:], []byte("test-proof-data-0123456789012345"))

	responseMsg := NewResponseMessage(sessionID, proof, CapMultiHop|CapIPv6)

	// Encode message
	encoded, err := EncodeMessage(responseMsg)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}

	// Decode message
	decoded, err := DecodeMessage(encoded)
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}

	// Verify header
	if decoded.Header.Type != MsgTypeResponse {
		t.Errorf("decoded message type = %d, want %d", decoded.Header.Type, MsgTypeResponse)
	}

	// Verify payload
	payload, ok := decoded.Payload.(*ResponseMessage)
	if !ok {
		t.Fatalf("decoded payload is not *ResponseMessage")
	}

	if payload.SessionID != sessionID {
		t.Errorf("SessionID mismatch")
	}

	if payload.Proof != proof {
		t.Errorf("Proof mismatch")
	}

	if payload.Capabilities != (CapMultiHop | CapIPv6) {
		t.Errorf("Capabilities = %d, want %d", payload.Capabilities, CapMultiHop|CapIPv6)
	}
}

func TestEstablishedMessageEncodeDecode(t *testing.T) {
	var sessionID [16]byte
	copy(sessionID[:], []byte("test-session-id0"))

	var peerIP [16]byte
	copy(peerIP[:4], []byte{192, 168, 1, 100}) // Example IPv4: 192.168.1.100

	// Example TLS certificate and signature (empty for test)
	var peerTLSCert []byte
	var peerTLSCertSig []byte

	establishedMsg := NewEstablishedMessage(
		sessionID,
		CapMultiHop|CapObfuscation,
		30,    // heartbeat interval
		1500,  // MTU
		3600,  // key rotation interval
		peerIP,
		45678, // peer port
		true,  // supports direct P2P
		peerTLSCert,    // TLS certificate
		peerTLSCertSig, // TLS certificate signature
	)

	// Encode message
	encoded, err := EncodeMessage(establishedMsg)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}

	// Decode message
	decoded, err := DecodeMessage(encoded)
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}

	// Verify payload
	payload, ok := decoded.Payload.(*EstablishedMessage)
	if !ok {
		t.Fatalf("decoded payload is not *EstablishedMessage")
	}

	if payload.SessionID != sessionID {
		t.Errorf("SessionID mismatch")
	}

	if payload.ServerCapabilities != (CapMultiHop | CapObfuscation) {
		t.Errorf("ServerCapabilities = %d, want %d", payload.ServerCapabilities, CapMultiHop|CapObfuscation)
	}

	if payload.HeartbeatInterval != 30 {
		t.Errorf("HeartbeatInterval = %d, want 30", payload.HeartbeatInterval)
	}

	if payload.MTU != 1500 {
		t.Errorf("MTU = %d, want 1500", payload.MTU)
	}

	if payload.KeyRotationInterval != 3600 {
		t.Errorf("KeyRotationInterval = %d, want 3600", payload.KeyRotationInterval)
	}
}

func TestDataFrameEncodeDecode(t *testing.T) {
	counter := uint64(12345)
	encryptedData := []byte("encrypted-frame-data-test-1234567890")

	dataMsg := NewDataFrameMessage(counter, encryptedData)

	// Encode message
	encoded, err := EncodeMessage(dataMsg)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}

	// Decode message
	decoded, err := DecodeMessage(encoded)
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}

	// Verify payload
	payload, ok := decoded.Payload.(*DataFrame)
	if !ok {
		t.Fatalf("decoded payload is not *DataFrame")
	}

	if payload.Counter != counter {
		t.Errorf("Counter = %d, want %d", payload.Counter, counter)
	}

	if !bytes.Equal(payload.EncryptedData, encryptedData) {
		t.Errorf("EncryptedData mismatch")
	}
}

func TestErrorMessageEncodeDecode(t *testing.T) {
	errorMsg := NewErrorMessage(ErrHandshakeTimeout, "Handshake timed out after 30 seconds")

	// Encode message
	encoded, err := EncodeMessage(errorMsg)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}

	// Decode message
	decoded, err := DecodeMessage(encoded)
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}

	// Verify payload
	payload, ok := decoded.Payload.(*ErrorMessage)
	if !ok {
		t.Fatalf("decoded payload is not *ErrorMessage")
	}

	if payload.ErrorCode != ErrHandshakeTimeout {
		t.Errorf("ErrorCode = %d, want %d", payload.ErrorCode, ErrHandshakeTimeout)
	}

	if payload.ErrorMessage != "Handshake timed out after 30 seconds" {
		t.Errorf("ErrorMessage = %s, want 'Handshake timed out after 30 seconds'", payload.ErrorMessage)
	}
}

func TestCloseMessageEncodeDecode(t *testing.T) {
	closeMsg := NewCloseMessage(CloseNormalShutdown, "Client shutdown")

	// Encode message
	encoded, err := EncodeMessage(closeMsg)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}

	// Decode message
	decoded, err := DecodeMessage(encoded)
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}

	// Verify payload
	payload, ok := decoded.Payload.(*CloseMessage)
	if !ok {
		t.Fatalf("decoded payload is not *CloseMessage")
	}

	if payload.ReasonCode != CloseNormalShutdown {
		t.Errorf("ReasonCode = %d, want %d", payload.ReasonCode, CloseNormalShutdown)
	}

	if payload.ReasonString != "Client shutdown" {
		t.Errorf("ReasonString = %s, want 'Client shutdown'", payload.ReasonString)
	}
}

func TestHeartbeatMessageEncodeDecode(t *testing.T) {
	heartbeatMsg := NewHeartbeatMessage()

	// Encode message
	encoded, err := EncodeMessage(heartbeatMsg)
	if err != nil {
		t.Fatalf("EncodeMessage() error = %v", err)
	}

	// Decode message
	decoded, err := DecodeMessage(encoded)
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}

	// Verify header
	if decoded.Header.Type != MsgTypeHeartbeat {
		t.Errorf("decoded message type = %d, want %d", decoded.Header.Type, MsgTypeHeartbeat)
	}

	// Verify payload
	_, ok := decoded.Payload.(*HeartbeatMessage)
	if !ok {
		t.Fatalf("decoded payload is not *HeartbeatMessage")
	}
}

func TestInvalidMessageDecode(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "truncated header",
			data: []byte{0x01, 0x02, 0x03},
		},
		{
			name: "invalid version",
			data: []byte{0xFF, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeMessage(tt.data)
			if err == nil {
				t.Errorf("DecodeMessage() expected error, got nil")
			}
		})
	}
}

func BenchmarkEncodeResponseMessage(b *testing.B) {
	var sessionID [16]byte
	var proof [32]byte
	copy(sessionID[:], []byte("test-session-id0"))
	copy(proof[:], []byte("test-proof-data-0123456789012345"))

	msg := NewResponseMessage(sessionID, proof, CapMultiHop)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeMessage(msg)
	}
}

func BenchmarkDecodeResponseMessage(b *testing.B) {
	var sessionID [16]byte
	var proof [32]byte
	copy(sessionID[:], []byte("test-session-id0"))
	copy(proof[:], []byte("test-proof-data-0123456789012345"))

	msg := NewResponseMessage(sessionID, proof, CapMultiHop)
	encoded, _ := EncodeMessage(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeMessage(encoded)
	}
}

func BenchmarkEncodeDataFrame(b *testing.B) {
	encryptedData := make([]byte, 1500)
	msg := NewDataFrameMessage(1, encryptedData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeMessage(msg)
	}
}

func BenchmarkDecodeDataFrame(b *testing.B) {
	encryptedData := make([]byte, 1500)
	msg := NewDataFrameMessage(1, encryptedData)
	encoded, _ := EncodeMessage(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeMessage(encoded)
	}
}
