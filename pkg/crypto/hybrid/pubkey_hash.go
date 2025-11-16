package hybrid

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/shadowmesh/shadowmesh/pkg/crypto/classical"
	"github.com/shadowmesh/shadowmesh/pkg/crypto/mldsa"
)

const (
	// PublicKeyHashSize is the output size of PublicKeyHash (SHA-256 = 32 bytes)
	PublicKeyHashSize = 32
)

var (
	// ErrInvalidPublicKey indicates the public key is missing or invalid
	ErrInvalidPublicKey = errors.New("invalid public key")
)

// PublicKeyHash computes a SHA-256 hash of the hybrid public key for smart contract integration
// Hash format: SHA256(MLDSAPublicKey || Ed25519PublicKey)
// Returns: 32-byte deterministic hash suitable for blockchain storage
//
// This hash is used in Epic 3 for smart contract public key verification:
// - Smart contracts store this hash for relay node verification
// - Clients can verify relay nodes by comparing hash against blockchain registry
// - Deterministic output ensures consistent verification across network
func PublicKeyHash(publicKey *HybridKeypair) ([]byte, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("%w: public key cannot be nil", ErrInvalidPublicKey)
	}

	// Validate ML-DSA public key
	if len(publicKey.MLDSAPublicKey) != mldsa.PublicKeySize {
		return nil, fmt.Errorf("%w: ML-DSA public key must be %d bytes, got %d",
			ErrInvalidPublicKey, mldsa.PublicKeySize, len(publicKey.MLDSAPublicKey))
	}

	// Validate Ed25519 public key
	if len(publicKey.Ed25519PublicKey) != classical.Ed25519PublicKeySize {
		return nil, fmt.Errorf("%w: Ed25519 public key must be %d bytes, got %d",
			ErrInvalidPublicKey, classical.Ed25519PublicKeySize, len(publicKey.Ed25519PublicKey))
	}

	// Concatenate public keys: MLDSAPublicKey || Ed25519PublicKey
	// Total size: 2592 + 32 = 2624 bytes
	combinedPublicKey := make([]byte, len(publicKey.MLDSAPublicKey)+len(publicKey.Ed25519PublicKey))
	copy(combinedPublicKey, publicKey.MLDSAPublicKey)
	copy(combinedPublicKey[len(publicKey.MLDSAPublicKey):], publicKey.Ed25519PublicKey)

	// Compute SHA-256 hash
	hash := sha256.Sum256(combinedPublicKey)

	return hash[:], nil
}
