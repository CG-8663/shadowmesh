package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// EncodeMessage encodes a complete message (header + payload) to binary
func EncodeMessage(msg *Message) ([]byte, error) {
	// Encode payload first to get length
	payload, err := encodePayload(msg.Header.Type, msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode payload: %w", err)
	}

	// Update header length
	msg.Header.Length = uint32(len(payload))

	// Encode header
	header := EncodeHeader(msg.Header)

	// Combine header + payload
	buf := make([]byte, 0, len(header)+len(payload))
	buf = append(buf, header...)
	buf = append(buf, payload...)

	return buf, nil
}

// DecodeMessage decodes a complete message from binary
func DecodeMessage(data []byte) (*Message, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("insufficient data: got %d bytes, need at least %d", len(data), HeaderSize)
	}

	// Decode header
	header, err := DecodeHeader(data[:HeaderSize])
	if err != nil {
		return nil, fmt.Errorf("failed to decode header: %w", err)
	}

	// Validate total message size
	expectedSize := HeaderSize + int(header.Length)
	if len(data) < expectedSize {
		return nil, fmt.Errorf("incomplete message: got %d bytes, expected %d", len(data), expectedSize)
	}

	// Decode payload
	payload, err := decodePayload(header.Type, data[HeaderSize:HeaderSize+int(header.Length)])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	return &Message{
		Header:  header,
		Payload: payload,
	}, nil
}

// ReadMessage reads a complete message from an io.Reader
func ReadMessage(r io.Reader) (*Message, error) {
	// Read header
	header, err := ReadHeader(r)
	if err != nil {
		return nil, err
	}

	// Read payload
	payload := make([]byte, header.Length)
	if header.Length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, fmt.Errorf("failed to read payload: %w", err)
		}
	}

	// Decode payload
	decodedPayload, err := decodePayload(header.Type, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	return &Message{
		Header:  header,
		Payload: decodedPayload,
	}, nil
}

// WriteMessage writes a complete message to an io.Writer
func WriteMessage(w io.Writer, msg *Message) error {
	data, err := EncodeMessage(msg)
	if err != nil {
		return err
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// encodePayload encodes a payload based on message type
func encodePayload(msgType byte, payload interface{}) ([]byte, error) {
	switch msgType {
	case MsgTypeHello:
		return encodeHello(payload.(*HelloMessage))
	case MsgTypeChallenge:
		return encodeChallenge(payload.(*ChallengeMessage))
	case MsgTypeResponse:
		return encodeResponse(payload.(*ResponseMessage))
	case MsgTypeEstablished:
		return encodeEstablished(payload.(*EstablishedMessage))
	case MsgTypeHeartbeat:
		return []byte{}, nil // Empty payload
	case MsgTypeHeartbeatAck:
		return []byte{}, nil // Empty payload
	case MsgTypeDataFrame:
		return encodeDataFrame(payload.(*DataFrame))
	case MsgTypeError:
		return encodeError(payload.(*ErrorMessage))
	case MsgTypeClose:
		return encodeClose(payload.(*CloseMessage))
	case MsgTypeRehandshakeRequest:
		return encodeRehandshakeRequest(payload.(*RehandshakeRequestMessage))
	case MsgTypeRehandshakeResponse:
		return encodeRehandshakeResponse(payload.(*RehandshakeResponseMessage))
	case MsgTypeRehandshakeComplete:
		return encodeRehandshakeComplete(payload.(*RehandshakeCompleteMessage))
	default:
		return nil, fmt.Errorf("unsupported message type for encoding: 0x%02x", msgType)
	}
}

// decodePayload decodes a payload based on message type
func decodePayload(msgType byte, data []byte) (interface{}, error) {
	switch msgType {
	case MsgTypeHello:
		return decodeHello(data)
	case MsgTypeChallenge:
		return decodeChallenge(data)
	case MsgTypeResponse:
		return decodeResponse(data)
	case MsgTypeEstablished:
		return decodeEstablished(data)
	case MsgTypeHeartbeat:
		return &HeartbeatMessage{}, nil
	case MsgTypeHeartbeatAck:
		return &HeartbeatAckMessage{}, nil
	case MsgTypeDataFrame:
		return decodeDataFrame(data)
	case MsgTypeError:
		return decodeError(data)
	case MsgTypeClose:
		return decodeClose(data)
	case MsgTypeRehandshakeRequest:
		return decodeRehandshakeRequest(data)
	case MsgTypeRehandshakeResponse:
		return decodeRehandshakeResponse(data)
	case MsgTypeRehandshakeComplete:
		return decodeRehandshakeComplete(data)
	default:
		return nil, fmt.Errorf("unsupported message type for decoding: 0x%02x", msgType)
	}
}

// HELLO message encoding/decoding
func encodeHello(msg *HelloMessage) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write fixed-size fields
	buf.Write(msg.ClientID[:])
	buf.Write(msg.KEMPublicKey[:])
	buf.Write(msg.ECDHPublicKey[:])
	buf.Write(msg.Signature[:])
	buf.Write(msg.ClassicalSignature[:])

	// Write timestamp (8 bytes, big-endian)
	if err := binary.Write(buf, binary.BigEndian, msg.Timestamp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeHello(data []byte) (*HelloMessage, error) {
	expectedSize := ClientIDSize + KEMPublicKeySize + ECDHPublicKeySize +
		SignatureSize + Ed25519SignatureSize + 8

	if len(data) < expectedSize {
		return nil, fmt.Errorf("invalid HELLO message size: got %d, expected %d", len(data), expectedSize)
	}

	msg := &HelloMessage{}
	offset := 0

	copy(msg.ClientID[:], data[offset:offset+ClientIDSize])
	offset += ClientIDSize

	copy(msg.KEMPublicKey[:], data[offset:offset+KEMPublicKeySize])
	offset += KEMPublicKeySize

	copy(msg.ECDHPublicKey[:], data[offset:offset+ECDHPublicKeySize])
	offset += ECDHPublicKeySize

	copy(msg.Signature[:], data[offset:offset+SignatureSize])
	offset += SignatureSize

	copy(msg.ClassicalSignature[:], data[offset:offset+Ed25519SignatureSize])
	offset += Ed25519SignatureSize

	msg.Timestamp = int64(binary.BigEndian.Uint64(data[offset : offset+8]))

	return msg, nil
}

// CHALLENGE message encoding/decoding
func encodeChallenge(msg *ChallengeMessage) ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.Write(msg.RelayID[:])
	buf.Write(msg.SessionID[:])
	buf.Write(msg.KEMCiphertext[:])
	buf.Write(msg.ECDHPublicKey[:])
	buf.Write(msg.Nonce[:])
	buf.Write(msg.Signature[:])
	buf.Write(msg.ClassicalSignature[:])

	if err := binary.Write(buf, binary.BigEndian, msg.Timestamp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeChallenge(data []byte) (*ChallengeMessage, error) {
	expectedSize := RelayIDSize + SessionIDSize + KEMCiphertextSize +
		ECDHPublicKeySize + NonceSize + SignatureSize + Ed25519SignatureSize + 8

	if len(data) < expectedSize {
		return nil, fmt.Errorf("invalid CHALLENGE message size: got %d, expected %d", len(data), expectedSize)
	}

	msg := &ChallengeMessage{}
	offset := 0

	copy(msg.RelayID[:], data[offset:offset+RelayIDSize])
	offset += RelayIDSize

	copy(msg.SessionID[:], data[offset:offset+SessionIDSize])
	offset += SessionIDSize

	copy(msg.KEMCiphertext[:], data[offset:offset+KEMCiphertextSize])
	offset += KEMCiphertextSize

	copy(msg.ECDHPublicKey[:], data[offset:offset+ECDHPublicKeySize])
	offset += ECDHPublicKeySize

	copy(msg.Nonce[:], data[offset:offset+NonceSize])
	offset += NonceSize

	copy(msg.Signature[:], data[offset:offset+SignatureSize])
	offset += SignatureSize

	copy(msg.ClassicalSignature[:], data[offset:offset+Ed25519SignatureSize])
	offset += Ed25519SignatureSize

	msg.Timestamp = int64(binary.BigEndian.Uint64(data[offset : offset+8]))

	return msg, nil
}

// RESPONSE message encoding/decoding
func encodeResponse(msg *ResponseMessage) ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.Write(msg.SessionID[:])
	buf.Write(msg.Proof[:])

	if err := binary.Write(buf, binary.BigEndian, msg.Capabilities); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeResponse(data []byte) (*ResponseMessage, error) {
	expectedSize := SessionIDSize + ProofSize + 4

	if len(data) < expectedSize {
		return nil, fmt.Errorf("invalid RESPONSE message size: got %d, expected %d", len(data), expectedSize)
	}

	msg := &ResponseMessage{}
	offset := 0

	copy(msg.SessionID[:], data[offset:offset+SessionIDSize])
	offset += SessionIDSize

	copy(msg.Proof[:], data[offset:offset+ProofSize])
	offset += ProofSize

	msg.Capabilities = binary.BigEndian.Uint32(data[offset : offset+4])

	return msg, nil
}

// ESTABLISHED message encoding/decoding
func encodeEstablished(msg *EstablishedMessage) ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.Write(msg.SessionID[:])

	if err := binary.Write(buf, binary.BigEndian, msg.ServerCapabilities); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, msg.HeartbeatInterval); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, msg.MTU); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, msg.KeyRotationInterval); err != nil {
		return nil, err
	}

	// Write new peer address fields
	buf.Write(msg.PeerPublicIP[:])
	if err := binary.Write(buf, binary.BigEndian, msg.PeerPublicPort); err != nil {
		return nil, err
	}

	// Write boolean as a single byte
	var peerSupportsByte byte
	if msg.PeerSupportsDirectP2P {
		peerSupportsByte = 1
	}
	if err := binary.Write(buf, binary.BigEndian, peerSupportsByte); err != nil {
		return nil, err
	}

	// Write TLS certificate (variable length)
	certLen := uint32(len(msg.PeerTLSCertificate))
	if err := binary.Write(buf, binary.BigEndian, certLen); err != nil {
		return nil, err
	}
	if certLen > 0 {
		buf.Write(msg.PeerTLSCertificate)
	}

	// Write TLS certificate signature (variable length)
	sigLen := uint32(len(msg.PeerTLSCertSignature))
	if err := binary.Write(buf, binary.BigEndian, sigLen); err != nil {
		return nil, err
	}
	if sigLen > 0 {
		buf.Write(msg.PeerTLSCertSignature)
	}

	return buf.Bytes(), nil
}

func decodeEstablished(data []byte) (*EstablishedMessage, error) {
	// Minimum size without variable-length fields: 49 bytes + 4 (certLen) + 4 (sigLen) = 57 bytes
	minSize := SessionIDSize + 4 + 4 + 2 + 4 + 16 + 2 + 1 + 4 + 4

	if len(data) < minSize {
		return nil, fmt.Errorf("invalid ESTABLISHED message size: got %d, expected at least %d", len(data), minSize)
	}

	msg := &EstablishedMessage{}
	offset := 0

	copy(msg.SessionID[:], data[offset:offset+SessionIDSize])
	offset += SessionIDSize

	msg.ServerCapabilities = binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	msg.HeartbeatInterval = binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	msg.MTU = binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	msg.KeyRotationInterval = binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	// Read peer address fields
	copy(msg.PeerPublicIP[:], data[offset:offset+16])
	offset += 16

	msg.PeerPublicPort = binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	peerSupportsByte := data[offset]
	msg.PeerSupportsDirectP2P = peerSupportsByte != 0
	offset += 1

	// Read TLS certificate (variable length)
	if offset+4 > len(data) {
		return nil, fmt.Errorf("insufficient data for certificate length")
	}
	certLen := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	if certLen > 0 {
		if offset+int(certLen) > len(data) {
			return nil, fmt.Errorf("insufficient data for certificate: expected %d bytes, got %d", certLen, len(data)-offset)
		}
		msg.PeerTLSCertificate = make([]byte, certLen)
		copy(msg.PeerTLSCertificate, data[offset:offset+int(certLen)])
		offset += int(certLen)
	}

	// Read TLS certificate signature (variable length)
	if offset+4 > len(data) {
		return nil, fmt.Errorf("insufficient data for signature length")
	}
	sigLen := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	if sigLen > 0 {
		if offset+int(sigLen) > len(data) {
			return nil, fmt.Errorf("insufficient data for signature: expected %d bytes, got %d", sigLen, len(data)-offset)
		}
		msg.PeerTLSCertSignature = make([]byte, sigLen)
		copy(msg.PeerTLSCertSignature, data[offset:offset+int(sigLen)])
	}

	return msg, nil
}

// DATA_FRAME message encoding/decoding
func encodeDataFrame(msg *DataFrame) ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, msg.Counter); err != nil {
		return nil, err
	}

	buf.Write(msg.EncryptedData)

	return buf.Bytes(), nil
}

func decodeDataFrame(data []byte) (*DataFrame, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("invalid DATA_FRAME message: too short")
	}

	msg := &DataFrame{
		Counter:       binary.BigEndian.Uint64(data[0:8]),
		EncryptedData: make([]byte, len(data)-8),
	}

	copy(msg.EncryptedData, data[8:])

	return msg, nil
}

// ERROR message encoding/decoding
func encodeError(msg *ErrorMessage) ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, msg.ErrorCode); err != nil {
		return nil, err
	}

	buf.WriteString(msg.ErrorMessage)

	return buf.Bytes(), nil
}

func decodeError(data []byte) (*ErrorMessage, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("invalid ERROR message: too short")
	}

	msg := &ErrorMessage{
		ErrorCode:    binary.BigEndian.Uint16(data[0:2]),
		ErrorMessage: string(data[2:]),
	}

	return msg, nil
}

// CLOSE message encoding/decoding
func encodeClose(msg *CloseMessage) ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, msg.ReasonCode); err != nil {
		return nil, err
	}

	buf.WriteString(msg.ReasonString)

	return buf.Bytes(), nil
}

func decodeClose(data []byte) (*CloseMessage, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("invalid CLOSE message: too short")
	}

	msg := &CloseMessage{
		ReasonCode:   binary.BigEndian.Uint16(data[0:2]),
		ReasonString: string(data[2:]),
	}

	return msg, nil
}

// NewHelloMessage creates a new HELLO message
func NewHelloMessage(clientID [32]byte, kemPubKey [1568]byte, ecdhPubKey [32]byte,
	sig [4595]byte, classicalSig [64]byte, timestamp int64) *Message {
	return &Message{
		Header: NewHeader(MsgTypeHello, FlagNone, 0),
		Payload: &HelloMessage{
			ClientID:           clientID,
			KEMPublicKey:       kemPubKey,
			ECDHPublicKey:      ecdhPubKey,
			Signature:          sig,
			ClassicalSignature: classicalSig,
			Timestamp:          timestamp,
		},
	}
}

// NewChallengeMessage creates a new CHALLENGE message
func NewChallengeMessage(relayID [32]byte, sessionID [16]byte, kemCT [1568]byte, ecdhPubKey [32]byte,
	nonce [24]byte, sig [4595]byte, classicalSig [64]byte, timestamp int64) *Message {

	return &Message{
		Header: NewHeader(MsgTypeChallenge, FlagNone, 0),
		Payload: &ChallengeMessage{
			RelayID:            relayID,
			SessionID:          sessionID,
			KEMCiphertext:      kemCT,
			ECDHPublicKey:      ecdhPubKey,
			Nonce:              nonce,
			Signature:          sig,
			ClassicalSignature: classicalSig,
			Timestamp:          timestamp,
		},
	}
}

// NewResponseMessage creates a new RESPONSE message
func NewResponseMessage(sessionID [16]byte, proof [32]byte, capabilities uint32) *Message {
	return &Message{
		Header: NewHeader(MsgTypeResponse, FlagNone, 0),
		Payload: &ResponseMessage{
			SessionID:    sessionID,
			Proof:        proof,
			Capabilities: capabilities,
		},
	}
}

// NewEstablishedMessage creates a new ESTABLISHED message
func NewEstablishedMessage(sessionID [16]byte, serverCaps uint32, heartbeatInterval uint32,
	mtu uint16, keyRotationInterval uint32, peerIP [16]byte, peerPort uint16, peerSupportsDirectP2P bool,
	peerTLSCert []byte, peerTLSCertSig []byte) *Message {
	return &Message{
		Header: NewHeader(MsgTypeEstablished, FlagNone, 0),
		Payload: &EstablishedMessage{
			SessionID:             sessionID,
			ServerCapabilities:    serverCaps,
			HeartbeatInterval:     heartbeatInterval,
			MTU:                   mtu,
			KeyRotationInterval:   keyRotationInterval,
			PeerPublicIP:          peerIP,
			PeerPublicPort:        peerPort,
			PeerSupportsDirectP2P: peerSupportsDirectP2P,
			PeerTLSCertificate:    peerTLSCert,
			PeerTLSCertSignature:  peerTLSCertSig,
		},
	}
}

// NewHeartbeatMessage creates a new HEARTBEAT message
func NewHeartbeatMessage() *Message {
	return &Message{
		Header:  NewHeader(MsgTypeHeartbeat, FlagNone, 0),
		Payload: &HeartbeatMessage{},
	}
}

// NewHeartbeatAckMessage creates a new HEARTBEAT_ACK message
func NewHeartbeatAckMessage() *Message {
	return &Message{
		Header:  NewHeader(MsgTypeHeartbeatAck, FlagNone, 0),
		Payload: &HeartbeatAckMessage{},
	}
}

// NewDataFrameMessage creates a new DATA_FRAME message
func NewDataFrameMessage(counter uint64, encryptedData []byte) *Message {
	return &Message{
		Header: NewHeader(MsgTypeDataFrame, FlagNone, 0),
		Payload: &DataFrame{
			Counter:       counter,
			EncryptedData: encryptedData,
		},
	}
}

// NewErrorMessage creates a new ERROR message
func NewErrorMessage(code uint16, message string) *Message {
	return &Message{
		Header: NewHeader(MsgTypeError, FlagNone, 0),
		Payload: &ErrorMessage{
			ErrorCode:    code,
			ErrorMessage: message,
		},
	}
}

// NewCloseMessage creates a new CLOSE message
func NewCloseMessage(code uint16, reason string) *Message {
	return &Message{
		Header: NewHeader(MsgTypeClose, FlagNone, 0),
		Payload: &CloseMessage{
			ReasonCode:   code,
			ReasonString: reason,
		},
	}
}

// NewRehandshakeRequestMessage creates a new REHANDSHAKE_REQUEST message
func NewRehandshakeRequestMessage(sessionID [SessionIDSize]byte, challenge [32]byte, timestamp uint64) *Message {
	return &Message{
		Header: NewHeader(MsgTypeRehandshakeRequest, FlagNone, 0),
		Payload: &RehandshakeRequestMessage{
			SessionID: sessionID,
			Challenge: challenge,
			Timestamp: timestamp,
		},
	}
}

// NewRehandshakeResponseMessage creates a new REHANDSHAKE_RESPONSE message
func NewRehandshakeResponseMessage(sessionID [SessionIDSize]byte, challengeResponse [32]byte, challenge [32]byte, timestamp uint64) *Message {
	return &Message{
		Header: NewHeader(MsgTypeRehandshakeResponse, FlagNone, 0),
		Payload: &RehandshakeResponseMessage{
			SessionID:         sessionID,
			ChallengeResponse: challengeResponse,
			Challenge:         challenge,
			Timestamp:         timestamp,
		},
	}
}

// NewRehandshakeCompleteMessage creates a new REHANDSHAKE_COMPLETE message
func NewRehandshakeCompleteMessage(sessionID [SessionIDSize]byte, challengeResponse [32]byte, timestamp uint64) *Message {
	return &Message{
		Header: NewHeader(MsgTypeRehandshakeComplete, FlagNone, 0),
		Payload: &RehandshakeCompleteMessage{
			SessionID:         sessionID,
			ChallengeResponse: challengeResponse,
			Timestamp:         timestamp,
		},
	}
}

// REHANDSHAKE_REQUEST message encoding/decoding
func encodeRehandshakeRequest(msg *RehandshakeRequestMessage) ([]byte, error) {
	buf := new(bytes.Buffer)

	// SessionID (16 bytes)
	if err := binary.Write(buf, binary.BigEndian, msg.SessionID); err != nil {
		return nil, err
	}

	// Challenge (32 bytes)
	if err := binary.Write(buf, binary.BigEndian, msg.Challenge); err != nil {
		return nil, err
	}

	// Timestamp (8 bytes)
	if err := binary.Write(buf, binary.BigEndian, msg.Timestamp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeRehandshakeRequest(data []byte) (*RehandshakeRequestMessage, error) {
	if len(data) < SessionIDSize+32+8 {
		return nil, fmt.Errorf("insufficient data for REHANDSHAKE_REQUEST")
	}

	msg := &RehandshakeRequestMessage{}

	offset := 0

	// SessionID
	copy(msg.SessionID[:], data[offset:offset+SessionIDSize])
	offset += SessionIDSize

	// Challenge
	copy(msg.Challenge[:], data[offset:offset+32])
	offset += 32

	// Timestamp
	msg.Timestamp = binary.BigEndian.Uint64(data[offset : offset+8])

	return msg, nil
}

// REHANDSHAKE_RESPONSE message encoding/decoding
func encodeRehandshakeResponse(msg *RehandshakeResponseMessage) ([]byte, error) {
	buf := new(bytes.Buffer)

	// SessionID (16 bytes)
	if err := binary.Write(buf, binary.BigEndian, msg.SessionID); err != nil {
		return nil, err
	}

	// ChallengeResponse (32 bytes)
	if err := binary.Write(buf, binary.BigEndian, msg.ChallengeResponse); err != nil {
		return nil, err
	}

	// Challenge (32 bytes)
	if err := binary.Write(buf, binary.BigEndian, msg.Challenge); err != nil {
		return nil, err
	}

	// Timestamp (8 bytes)
	if err := binary.Write(buf, binary.BigEndian, msg.Timestamp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeRehandshakeResponse(data []byte) (*RehandshakeResponseMessage, error) {
	if len(data) < SessionIDSize+32+32+8 {
		return nil, fmt.Errorf("insufficient data for REHANDSHAKE_RESPONSE")
	}

	msg := &RehandshakeResponseMessage{}

	offset := 0

	// SessionID
	copy(msg.SessionID[:], data[offset:offset+SessionIDSize])
	offset += SessionIDSize

	// ChallengeResponse
	copy(msg.ChallengeResponse[:], data[offset:offset+32])
	offset += 32

	// Challenge
	copy(msg.Challenge[:], data[offset:offset+32])
	offset += 32

	// Timestamp
	msg.Timestamp = binary.BigEndian.Uint64(data[offset : offset+8])

	return msg, nil
}

// REHANDSHAKE_COMPLETE message encoding/decoding
func encodeRehandshakeComplete(msg *RehandshakeCompleteMessage) ([]byte, error) {
	buf := new(bytes.Buffer)

	// SessionID (16 bytes)
	if err := binary.Write(buf, binary.BigEndian, msg.SessionID); err != nil {
		return nil, err
	}

	// ChallengeResponse (32 bytes)
	if err := binary.Write(buf, binary.BigEndian, msg.ChallengeResponse); err != nil {
		return nil, err
	}

	// Timestamp (8 bytes)
	if err := binary.Write(buf, binary.BigEndian, msg.Timestamp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeRehandshakeComplete(data []byte) (*RehandshakeCompleteMessage, error) {
	if len(data) < SessionIDSize+32+8 {
		return nil, fmt.Errorf("insufficient data for REHANDSHAKE_COMPLETE")
	}

	msg := &RehandshakeCompleteMessage{}

	offset := 0

	// SessionID
	copy(msg.SessionID[:], data[offset:offset+SessionIDSize])
	offset += SessionIDSize

	// ChallengeResponse
	copy(msg.ChallengeResponse[:], data[offset:offset+32])
	offset += 32

	// Timestamp
	msg.Timestamp = binary.BigEndian.Uint64(data[offset : offset+8])

	return msg, nil
}
