package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"

	"github.com/cloudflare/circl/sign/dilithium/mode5"
)

// Signature size constants
const (
	// ML-DSA-87 (Dilithium5/mode5) signature size
	MLDSASignatureSize = mode5.SignatureSize // 4,595 bytes
	// Ed25519 signature size
	Ed25519SignatureSize = ed25519.SignatureSize // 64 bytes
	// Total hybrid signature size
	HybridSignatureSize = MLDSASignatureSize + Ed25519SignatureSize // 4,659 bytes
)

var (
	// ErrInvalidSignature indicates the signature verification failed
	ErrInvalidSignature = errors.New("signature verification failed")
	// ErrInvalidSignatureLength indicates the signature has incorrect length
	ErrInvalidSignatureLength = errors.New("invalid signature length")
	// ErrInvalidVerifyKey indicates the verify key is invalid
	ErrInvalidVerifyKey = errors.New("invalid verify key")
)

// HybridSigningKey contains both ML-DSA-87 and Ed25519 private keys
type HybridSigningKey struct {
	MLDSAPrivateKey    *mode5.PrivateKey
	Ed25519PrivateKey  ed25519.PrivateKey
}

// HybridVerifyKey contains both ML-DSA-87 and Ed25519 public keys
type HybridVerifyKey struct {
	MLDSAPublicKey    *mode5.PublicKey
	Ed25519PublicKey  ed25519.PublicKey
}

// GenerateSigningKey generates a new hybrid signing key pair
// Returns a HybridSigningKey containing both ML-DSA-87 and Ed25519 private keys
func GenerateSigningKey() (*HybridSigningKey, error) {
	// Generate ML-DSA-87 key pair
	mldsaPub, mldsaPriv, err := mode5.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ML-DSA-87 key pair: %w", err)
	}

	// Generate Ed25519 key pair
	ed25519Pub, ed25519Priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 key pair: %w", err)
	}

	// Store public keys for later use
	_ = mldsaPub
	_ = ed25519Pub

	return &HybridSigningKey{
		MLDSAPrivateKey:   mldsaPriv,
		Ed25519PrivateKey: ed25519Priv,
	}, nil
}

// PublicKey extracts the public verification key from the signing key
func (hsk *HybridSigningKey) PublicKey() *HybridVerifyKey {
	mldsaPub := hsk.MLDSAPrivateKey.Public().(*mode5.PublicKey)
	return &HybridVerifyKey{
		MLDSAPublicKey:   mldsaPub,
		Ed25519PublicKey: hsk.Ed25519PrivateKey.Public().(ed25519.PublicKey),
	}
}

// Sign creates a hybrid signature over a message
// Returns a signature in the format: [ML-DSA-87 signature][Ed25519 signature]
// The combined signature is 4,659 bytes (4,595 + 64)
func Sign(privateKey *HybridSigningKey, message []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("private key is nil")
	}
	if message == nil {
		return nil, errors.New("message is nil")
	}

	// Sign with ML-DSA-87
	log.Println("crypto.Sign: Starting ML-DSA-87 (Dilithium) signature...")
	mldsaSig := make([]byte, MLDSASignatureSize)
	mode5.SignTo(privateKey.MLDSAPrivateKey, message, mldsaSig)
	log.Println("crypto.Sign: ML-DSA-87 signature complete")

	// Sign with Ed25519
	log.Println("crypto.Sign: Starting Ed25519 signature...")
	ed25519Sig := ed25519.Sign(privateKey.Ed25519PrivateKey, message)
	log.Println("crypto.Sign: Ed25519 signature complete")
	if len(ed25519Sig) != Ed25519SignatureSize {
		return nil, fmt.Errorf("Ed25519 signature has unexpected length: %d", len(ed25519Sig))
	}

	// Concatenate signatures: [ML-DSA-87][Ed25519]
	hybridSig := make([]byte, 0, HybridSignatureSize)
	hybridSig = append(hybridSig, mldsaSig...)
	hybridSig = append(hybridSig, ed25519Sig...)

	return hybridSig, nil
}

// Verify validates a hybrid signature
// Both ML-DSA-87 and Ed25519 signatures must be valid for verification to succeed
// Returns nil on success, error on failure (fail-fast on first invalid signature)
func Verify(publicKey *HybridVerifyKey, message, signature []byte) error {
	if publicKey == nil {
		return ErrInvalidVerifyKey
	}
	if message == nil {
		return errors.New("message is nil")
	}
	if signature == nil {
		return ErrInvalidSignature
	}

	// Verify signature length
	if len(signature) != HybridSignatureSize {
		return fmt.Errorf("%w: expected %d bytes, got %d bytes",
			ErrInvalidSignatureLength, HybridSignatureSize, len(signature))
	}

	// Split signature into components
	mldsaSig := signature[:MLDSASignatureSize]
	ed25519Sig := signature[MLDSASignatureSize:]

	// Verify ML-DSA-87 signature (fail fast)
	if !mode5.Verify(publicKey.MLDSAPublicKey, message, mldsaSig) {
		return fmt.Errorf("%w: ML-DSA-87 verification failed", ErrInvalidSignature)
	}

	// Verify Ed25519 signature (fail fast)
	if !ed25519.Verify(publicKey.Ed25519PublicKey, message, ed25519Sig) {
		return fmt.Errorf("%w: Ed25519 verification failed", ErrInvalidSignature)
	}

	return nil
}

// PublicKeyHash returns a SHA-256 hash of the hybrid public key
// Format: SHA256(ML-DSA-87 PublicKey || Ed25519 PublicKey)
// This hash is used for smart contract storage and validation
func PublicKeyHash(publicKey *HybridVerifyKey) [32]byte {
	if publicKey == nil {
		return [32]byte{}
	}

	// Serialize ML-DSA-87 public key
	mldsaPubBytes, err := publicKey.MLDSAPublicKey.MarshalBinary()
	if err != nil {
		// This should never happen with a valid public key
		return [32]byte{}
	}

	// Ed25519 public key is already in byte slice format
	ed25519PubBytes := []byte(publicKey.Ed25519PublicKey)

	// Concatenate and hash
	combined := append(mldsaPubBytes, ed25519PubBytes...)
	return sha256.Sum256(combined)
}

// Bytes returns the concatenated bytes of the hybrid public key
// Format: [ML-DSA-87 PublicKey][Ed25519 PublicKey]
func (hvk *HybridVerifyKey) Bytes() ([]byte, error) {
	if hvk == nil {
		return nil, ErrInvalidVerifyKey
	}

	mldsaPubBytes, err := hvk.MLDSAPublicKey.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ML-DSA-87 public key: %w", err)
	}

	ed25519PubBytes := []byte(hvk.Ed25519PublicKey)

	combined := make([]byte, 0, len(mldsaPubBytes)+len(ed25519PubBytes))
	combined = append(combined, mldsaPubBytes...)
	combined = append(combined, ed25519PubBytes...)

	return combined, nil
}
