package hybrid

import (
	"errors"
	"fmt"

	"github.com/shadowmesh/shadowmesh/pkg/crypto/classical"
	"github.com/shadowmesh/shadowmesh/pkg/crypto/mldsa"
)

const (
	// HybridSignatureSize is the total size of hybrid signature (ML-DSA + Ed25519)
	// ML-DSA-87: 4595 bytes (Dilithium Round 3) + Ed25519: 64 bytes = 4659 bytes
	HybridSignatureSize = mldsa.SignatureSize + classical.Ed25519SignatureSize // 4659 bytes
)

var (
	// ErrInvalidSignature indicates the signature format is invalid
	ErrInvalidSignature = errors.New("invalid signature format")
	// ErrVerificationFailed indicates signature verification failed
	ErrVerificationFailed = errors.New("signature verification failed")
	// ErrSigningFailed indicates signature generation failed
	ErrSigningFailed = errors.New("signature generation failed")
	// ErrInvalidKeypair indicates the keypair is missing required keys
	ErrInvalidKeypair = errors.New("invalid keypair: missing signature keys")
)

// HybridSign creates a hybrid signature combining ML-DSA-87 and Ed25519
// Returns: signature (4595 + 64 = 4659 bytes), error
// Signature format: ML-DSA-signature || Ed25519-signature
// Performance target: <10ms on commodity hardware (4 GHz CPU)
func HybridSign(message []byte, privateKey *HybridKeypair) ([]byte, error) {
	if privateKey == nil {
		return nil, fmt.Errorf("%w: private key cannot be nil", ErrSigningFailed)
	}

	// Validate that signature keys are present
	if len(privateKey.MLDSAPrivateKey) != mldsa.PrivateKeySize {
		return nil, fmt.Errorf("%w: ML-DSA private key missing or invalid size", ErrInvalidKeypair)
	}
	if len(privateKey.Ed25519PrivateKey) != classical.Ed25519PrivateKeySize {
		return nil, fmt.Errorf("%w: Ed25519 private key missing or invalid size", ErrInvalidKeypair)
	}

	// Sign with ML-DSA-87
	mldsaSig, err := mldsa.Sign(message, privateKey.MLDSAPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: ML-DSA signing failed: %v", ErrSigningFailed, err)
	}

	// Sign with Ed25519
	ed25519Sig, err := classical.Ed25519Sign(message, privateKey.Ed25519PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: Ed25519 signing failed: %v", ErrSigningFailed, err)
	}

	// Concatenate signatures: ML-DSA || Ed25519
	hybridSignature := make([]byte, HybridSignatureSize)
	copy(hybridSignature, mldsaSig)
	copy(hybridSignature[len(mldsaSig):], ed25519Sig)

	return hybridSignature, nil
}

// HybridVerify verifies both ML-DSA-87 and Ed25519 signatures independently
// Returns: true if BOTH signatures are valid, false otherwise
// Performance target: <5ms on commodity hardware (4 GHz CPU)
// Security property: Both signatures must pass - if either fails, verification fails
func HybridVerify(message []byte, signature []byte, publicKey *HybridKeypair) bool {
	if publicKey == nil {
		return false
	}

	// Validate signature size
	if len(signature) != HybridSignatureSize {
		return false
	}

	// Validate that public keys are present
	if len(publicKey.MLDSAPublicKey) != mldsa.PublicKeySize {
		return false
	}
	if len(publicKey.Ed25519PublicKey) != classical.Ed25519PublicKeySize {
		return false
	}

	// Split signature into ML-DSA and Ed25519 components
	mldsaSig := signature[:mldsa.SignatureSize]
	ed25519Sig := signature[mldsa.SignatureSize:]

	// Verify ML-DSA-87 signature
	mldsaValid := mldsa.Verify(message, mldsaSig, publicKey.MLDSAPublicKey)
	if !mldsaValid {
		return false
	}

	// Verify Ed25519 signature
	ed25519Valid := classical.Ed25519Verify(message, ed25519Sig, publicKey.Ed25519PublicKey)
	if !ed25519Valid {
		return false
	}

	// Both signatures must be valid
	return true
}
