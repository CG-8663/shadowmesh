package main

import (
	"time"
)

// SessionKeys contains the derived session keys and parameters
// This structure is extended with Epic 2 fields for Direct P2P support
type SessionKeys struct {
	// Original fields (Epic 1)
	SessionID           [16]byte
	TXKey               [32]byte // Transmission key
	RXKey               [32]byte // Reception key
	HeartbeatInterval   time.Duration
	MTU                 uint16
	KeyRotationInterval time.Duration
	ServerCapabilities  uint32

	// Epic 2: Direct P2P fields
	PeerPublicIP          [16]byte // IPv4 or IPv6 address (IPv4 mapped in first 4 bytes)
	PeerPublicPort        uint16   // Peer's public port
	PeerSupportsDirectP2P bool     // Can this peer accept direct connections?

	// Epic 2: TLS certificate fields for secure direct P2P
	PeerTLSCert    []byte // DER-encoded X.509 certificate
	PeerTLSCertSig []byte // ML-DSA-87 signature of certificate
}
