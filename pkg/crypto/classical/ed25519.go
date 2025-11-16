// Package classical provides classical cryptography implementations (X25519 ECDH + Ed25519 signatures).
package classical

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
)

// Ed25519 key sizes from RFC 8032
const (
	Ed25519PublicKeySize  = ed25519.PublicKeySize  // 32 bytes
	Ed25519PrivateKeySize = ed25519.PrivateKeySize // 64 bytes
	Ed25519SignatureSize  = ed25519.SignatureSize  // 64 bytes
)

// Ed25519Keypair represents an Ed25519 keypair
type Ed25519Keypair struct {
	PublicKey  []byte // 32 bytes
	PrivateKey []byte // 64 bytes
}

// Error types for Ed25519 operations
var (
	ErrEd25519KeyGenerationFailed = errors.New("Ed25519 keypair generation failed")
	ErrEd25519SigningFailed       = errors.New("Ed25519 signing failed")
	ErrEd25519InvalidPublicKey    = errors.New("invalid Ed25519 public key")
	ErrEd25519InvalidPrivateKey   = errors.New("invalid Ed25519 private key")
)

// GenerateEd25519Keypair generates a new Ed25519 keypair using Go standard library
func GenerateEd25519Keypair() (*Ed25519Keypair, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEd25519KeyGenerationFailed, err)
	}

	return &Ed25519Keypair{
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}, nil
}

// Ed25519Sign signs a message with Ed25519 private key
func Ed25519Sign(message []byte, privateKey []byte) ([]byte, error) {
	if len(privateKey) != Ed25519PrivateKeySize {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrEd25519InvalidPrivateKey, Ed25519PrivateKeySize, len(privateKey))
	}

	signature := ed25519.Sign(ed25519.PrivateKey(privateKey), message)
	if len(signature) != Ed25519SignatureSize {
		return nil, fmt.Errorf("%w: unexpected signature size %d", ErrEd25519SigningFailed, len(signature))
	}

	return signature, nil
}

// Ed25519Verify verifies an Ed25519 signature
func Ed25519Verify(message []byte, signature []byte, publicKey []byte) bool {
	if len(publicKey) != Ed25519PublicKeySize {
		return false
	}
	if len(signature) != Ed25519SignatureSize {
		return false
	}

	return ed25519.Verify(ed25519.PublicKey(publicKey), message, signature)
}
