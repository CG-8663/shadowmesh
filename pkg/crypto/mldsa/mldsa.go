// Package mldsa implements ML-DSA-87 (Dilithium5) digital signatures using NIST FIPS 204 standard.
// ML-DSA provides post-quantum security against quantum adversaries with EUF-CMA security.
package mldsa

import (
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/cloudflare/circl/sign"
	"github.com/cloudflare/circl/sign/dilithium/mode5"
)

// Key sizes for ML-DSA-87 (Dilithium5) from circl implementation
// Note: circl implements Dilithium Round 3, signature size differs slightly from final FIPS 204
const (
	PublicKeySize  = mode5.PublicKeySize  // 2592 bytes
	PrivateKeySize = mode5.PrivateKeySize // 4864 bytes
	SignatureSize  = mode5.SignatureSize  // 4595 bytes (Dilithium Round 3)
)

// MLDSAKeypair represents an ML-DSA-87 keypair
type MLDSAKeypair struct {
	PublicKey  []byte // 2592 bytes
	PrivateKey []byte // 4864 bytes
}

// Error types
var (
	ErrKeyGenerationFailed = errors.New("ML-DSA keypair generation failed")
	ErrSigningFailed       = errors.New("ML-DSA signing failed")
	ErrInvalidPublicKey    = errors.New("invalid ML-DSA public key")
	ErrInvalidPrivateKey   = errors.New("invalid ML-DSA private key")
	ErrInvalidSignature    = errors.New("invalid ML-DSA signature")
)

// GenerateKeypair generates a new ML-DSA-87 (Dilithium5) keypair
func GenerateKeypair() (*MLDSAKeypair, error) {
	publicKey, privateKey, err := mode5.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
	}

	// Pack keys to byte slices
	pubKeyBytes, err := publicKey.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal public key: %v", ErrKeyGenerationFailed, err)
	}

	privKeyBytes, err := privateKey.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal private key: %v", ErrKeyGenerationFailed, err)
	}

	return &MLDSAKeypair{
		PublicKey:  pubKeyBytes,
		PrivateKey: privKeyBytes,
	}, nil
}

// Sign creates an ML-DSA-87 signature for the given message
func Sign(message []byte, privateKey []byte) ([]byte, error) {
	if len(privateKey) != PrivateKeySize {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidPrivateKey, PrivateKeySize, len(privateKey))
	}

	// Unmarshal private key
	var privKey mode5.PrivateKey
	if err := privKey.UnmarshalBinary(privateKey); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPrivateKey, err)
	}

	// Allocate signature buffer and sign
	signature := make([]byte, SignatureSize)
	mode5.SignTo(&privKey, message, signature)

	return signature, nil
}

// Verify verifies an ML-DSA-87 signature for the given message using constant-time operations
func Verify(message []byte, signature []byte, publicKey []byte) bool {
	if len(publicKey) != PublicKeySize {
		return false
	}
	if len(signature) != SignatureSize {
		return false
	}

	// Unmarshal public key
	var pubKey mode5.PublicKey
	if err := pubKey.UnmarshalBinary(publicKey); err != nil {
		return false
	}

	// Verify signature (constant-time operation)
	return mode5.Verify(&pubKey, message, signature)
}

// Scheme returns the ML-DSA-87 (Dilithium5) scheme for validation
func Scheme() sign.Scheme {
	return mode5.Scheme()
}
