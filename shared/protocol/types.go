package protocol

import (
	"time"
)

// Protocol version
const (
	ProtocolVersion byte = 0x01
)

// Message types
const (
	// Control messages (0x00-0x0F)
	MsgTypeHello         byte = 0x01
	MsgTypeChallenge     byte = 0x02
	MsgTypeResponse      byte = 0x03
	MsgTypeEstablished   byte = 0x04
	MsgTypeHeartbeat     byte = 0x05
	MsgTypeHeartbeatAck  byte = 0x06
	MsgTypeError         byte = 0x0E
	MsgTypeClose         byte = 0x0F

	// Data messages (0x10-0x1F)
	MsgTypeDataFrame     byte = 0x10
	MsgTypeMultiHop      byte = 0x11

	// Management messages (0x20-0x2F)
	MsgTypeConfigUpdate  byte = 0x20
	MsgTypeStatsRequest  byte = 0x21
	MsgTypeStatsResponse byte = 0x22
)

// Message flags (2 bytes)
const (
	FlagNone          uint16 = 0x0000
	FlagKeyRotation   uint16 = 0x0001 // Set in HELLO for key rotation
)

// Error codes
const (
	ErrInvalidVersion       uint16 = 0x0001
	ErrInvalidMessageType   uint16 = 0x0002
	ErrInvalidSignature     uint16 = 0x0003
	ErrHandshakeTimeout     uint16 = 0x0004
	ErrDecryptionFailure    uint16 = 0x0005
	ErrReplayAttack         uint16 = 0x0006
	ErrUnsupportedFeature   uint16 = 0x0007
	ErrRateLimitExceeded    uint16 = 0x0008
	ErrInternalServerError  uint16 = 0x00FF
)

// Close reason codes
const (
	CloseNormalShutdown       uint16 = 0x0000
	CloseIdleTimeout          uint16 = 0x0001
	CloseAdministrativeShutdown uint16 = 0x0002
	CloseProtocolViolation    uint16 = 0x0003
)

// Capability flags (4 bytes)
const (
	CapMultiHop     uint32 = 1 << 0
	CapObfuscation  uint32 = 1 << 1
	CapIPv6         uint32 = 1 << 2
)

// Protocol constants
const (
	MaxMessageSize      = 65535            // 64 KB
	SessionIDSize       = 16               // bytes
	NonceSize           = 24               // bytes
	ProofSize           = 32               // bytes (HMAC-SHA256)
	ClientIDSize        = 32               // bytes (Ed25519 pubkey hash)
	RelayIDSize         = 32               // bytes (Ed25519 pubkey hash)
	HeaderSize          = 8                // bytes (version + type + flags + length)

	// Crypto sizes (from shared/crypto)
	KEMPublicKeySize    = 1568             // ML-KEM-1024 public key
	KEMCiphertextSize   = 1568             // ML-KEM-1024 ciphertext
	ECDHPublicKeySize   = 32               // X25519 public key
	SignatureSize       = 4627             // ML-DSA-87 signature
	Ed25519SignatureSize = 64              // Ed25519 signature

	// Timeouts and intervals
	HandshakeTimeout    = 30 * time.Second
	DefaultHeartbeatInterval = 30 * time.Second
	HeartbeatResponseTimeout = 5 * time.Second
	MaxMissedHeartbeats = 3
	KeyRotationGracePeriod = 30 * time.Second

	// Defaults
	DefaultMTU          = 1500             // Standard Ethernet MTU
	DefaultKeyRotationInterval = 3600      // 1 hour in seconds
)

// Header represents the common message header
type Header struct {
	Version byte
	Type    byte
	Flags   uint16
	Length  uint32 // Payload length
}

// HelloMessage represents the initial handshake message from client
type HelloMessage struct {
	ClientID           [ClientIDSize]byte       // Ed25519 public key hash
	KEMPublicKey       [KEMPublicKeySize]byte   // ML-KEM-1024 public key
	ECDHPublicKey      [ECDHPublicKeySize]byte  // X25519 public key
	Signature          [SignatureSize]byte      // ML-DSA-87 signature
	ClassicalSignature [Ed25519SignatureSize]byte // Ed25519 signature
	Timestamp          int64                     // Unix nanoseconds
}

// ChallengeMessage represents the relay's response with encapsulated secret
type ChallengeMessage struct {
	RelayID            [RelayIDSize]byte        // Ed25519 public key hash
	SessionID          [SessionIDSize]byte      // Random session identifier
	KEMCiphertext      [KEMCiphertextSize]byte  // ML-KEM-1024 ciphertext
	ECDHPublicKey      [ECDHPublicKeySize]byte  // X25519 public key
	Nonce              [NonceSize]byte          // For proof MAC
	Signature          [SignatureSize]byte      // ML-DSA-87 signature
	ClassicalSignature [Ed25519SignatureSize]byte // Ed25519 signature
	Timestamp          int64                     // Unix nanoseconds
}

// ResponseMessage proves client has the shared secret
type ResponseMessage struct {
	SessionID    [SessionIDSize]byte // Echoed from CHALLENGE
	Proof        [ProofSize]byte     // HMAC-SHA256(shared_secret, nonce)
	Capabilities uint32              // Client capability flags
}

// EstablishedMessage confirms handshake completion
type EstablishedMessage struct {
	SessionID             [SessionIDSize]byte
	ServerCapabilities    uint32
	HeartbeatInterval     uint32 // seconds
	MTU                   uint16 // Maximum Transmission Unit
	KeyRotationInterval   uint32 // seconds
}

// HeartbeatMessage is an empty keepalive message
type HeartbeatMessage struct {
	// Empty - just header
}

// HeartbeatAckMessage acknowledges a heartbeat
type HeartbeatAckMessage struct {
	// Empty - just header
}

// DataFrame carries an encrypted Ethernet frame
type DataFrame struct {
	Counter       uint64 // Monotonic frame counter
	EncryptedData []byte // Encrypted Ethernet frame + Poly1305 tag
}

// ErrorMessage reports an error condition
type ErrorMessage struct {
	ErrorCode    uint16
	ErrorMessage string
}

// CloseMessage gracefully terminates the connection
type CloseMessage struct {
	ReasonCode   uint16
	ReasonString string
}

// ConfigUpdateMessage pushes configuration changes (future use)
type ConfigUpdateMessage struct {
	ConfigData []byte // JSON-encoded configuration
}

// StatsRequestMessage requests statistics from peer (future use)
type StatsRequestMessage struct {
	RequestedMetrics []string
}

// StatsResponseMessage reports statistics (future use)
type StatsResponseMessage struct {
	StatsData []byte // JSON-encoded statistics
}

// Message is a generic container for all message types
type Message struct {
	Header  Header
	Payload interface{} // One of the *Message types above
}

// String returns a human-readable message type name
func MessageTypeName(msgType byte) string {
	switch msgType {
	case MsgTypeHello:
		return "HELLO"
	case MsgTypeChallenge:
		return "CHALLENGE"
	case MsgTypeResponse:
		return "RESPONSE"
	case MsgTypeEstablished:
		return "ESTABLISHED"
	case MsgTypeHeartbeat:
		return "HEARTBEAT"
	case MsgTypeHeartbeatAck:
		return "HEARTBEAT_ACK"
	case MsgTypeDataFrame:
		return "DATA_FRAME"
	case MsgTypeMultiHop:
		return "MULTI_HOP"
	case MsgTypeConfigUpdate:
		return "CONFIG_UPDATE"
	case MsgTypeStatsRequest:
		return "STATS_REQUEST"
	case MsgTypeStatsResponse:
		return "STATS_RESPONSE"
	case MsgTypeError:
		return "ERROR"
	case MsgTypeClose:
		return "CLOSE"
	default:
		return "UNKNOWN"
	}
}

// ErrorCodeName returns a human-readable error code name
func ErrorCodeName(code uint16) string {
	switch code {
	case ErrInvalidVersion:
		return "Invalid Protocol Version"
	case ErrInvalidMessageType:
		return "Invalid Message Type"
	case ErrInvalidSignature:
		return "Invalid Signature"
	case ErrHandshakeTimeout:
		return "Handshake Timeout"
	case ErrDecryptionFailure:
		return "Decryption Failure"
	case ErrReplayAttack:
		return "Replay Attack Detected"
	case ErrUnsupportedFeature:
		return "Unsupported Feature"
	case ErrRateLimitExceeded:
		return "Rate Limit Exceeded"
	case ErrInternalServerError:
		return "Internal Server Error"
	default:
		return "Unknown Error"
	}
}
