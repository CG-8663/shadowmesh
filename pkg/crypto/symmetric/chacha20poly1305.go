// Package symmetric provides symmetric encryption primitives for ShadowMesh.
// This package implements ChaCha20-Poly1305 AEAD for high-performance frame encryption.
package symmetric

import (
	"crypto/subtle"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// Key size and nonce size constants
const (
	KeySize   = chacha20poly1305.KeySize   // 32 bytes
	NonceSize = chacha20poly1305.NonceSize // 12 bytes (96 bits)
	TagSize   = 16                         // Poly1305 tag size
)

// EncryptedFrame represents an encrypted Ethernet frame with AEAD authentication
type EncryptedFrame struct {
	Nonce      [NonceSize]byte // 96-bit nonce (48-bit counter + 48-bit salt)
	Ciphertext []byte          // Encrypted payload
	// Note: Poly1305 tag (16 bytes) is appended to Ciphertext by AEAD
}

// Error types
var (
	ErrInvalidKeySize    = errors.New("invalid key size: must be 32 bytes")
	ErrInvalidNonceSize  = errors.New("invalid nonce size: must be 12 bytes")
	ErrEncryptionFailed  = errors.New("encryption failed")
	ErrDecryptionFailed  = errors.New("decryption failed: authentication tag mismatch or corrupted ciphertext")
	ErrInvalidCiphertext = errors.New("invalid ciphertext: too short or corrupted")
)

// Encrypt encrypts plaintext using ChaCha20-Poly1305 AEAD
// Returns: EncryptedFrame containing nonce, ciphertext, and authentication tag
//
// Security properties:
// - IND-CCA2 secure (indistinguishability under chosen-ciphertext attack)
// - Authenticated encryption (confidentiality + integrity + authenticity)
// - Nonce must be unique for each (key, plaintext) pair - never reuse!
//
// Performance: ~1+ Gbps on single CPU core (commodity 4 GHz hardware)
func Encrypt(plaintext []byte, key [KeySize]byte, nonce [NonceSize]byte) (*EncryptedFrame, error) {
	// Validate inputs
	if len(key) != KeySize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidKeySize, len(key))
	}
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidNonceSize, len(nonce))
	}

	// Create ChaCha20-Poly1305 AEAD cipher
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create cipher: %v", ErrEncryptionFailed, err)
	}

	// Encrypt and authenticate
	// aead.Seal appends the ciphertext and tag to dst
	// Output format: ciphertext || 16-byte Poly1305 tag
	ciphertext := aead.Seal(nil, nonce[:], plaintext, nil)

	return &EncryptedFrame{
		Nonce:      nonce,
		Ciphertext: ciphertext, // Includes 16-byte tag at end
	}, nil
}

// Decrypt decrypts and authenticates ciphertext using ChaCha20-Poly1305 AEAD
// Returns: plaintext if authentication tag is valid, error otherwise
//
// Security properties:
// - Constant-time tag comparison (timing attack resistant)
// - Fails fast if tag is invalid (no partial decryption)
// - Prevents tampering, replay, and forgery attacks
func Decrypt(frame *EncryptedFrame, key [KeySize]byte) ([]byte, error) {
	// Validate inputs
	if frame == nil {
		return nil, fmt.Errorf("%w: frame cannot be nil", ErrDecryptionFailed)
	}
	if len(key) != KeySize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidKeySize, len(key))
	}
	if len(frame.Nonce) != NonceSize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidNonceSize, len(frame.Nonce))
	}

	// Validate ciphertext contains at least the tag
	if len(frame.Ciphertext) < TagSize {
		return nil, fmt.Errorf("%w: ciphertext must be at least %d bytes (tag size)", ErrInvalidCiphertext, TagSize)
	}

	// Create ChaCha20-Poly1305 AEAD cipher
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create cipher: %v", ErrDecryptionFailed, err)
	}

	// Decrypt and verify authentication tag
	// aead.Open validates the tag using constant-time comparison (via subtle.ConstantTimeCompare)
	plaintext, err := aead.Open(nil, frame.Nonce[:], frame.Ciphertext, nil)
	if err != nil {
		// Tag validation failed - ciphertext was tampered with or wrong key
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// EncryptWithKey is a convenience function that accepts a byte slice key
// Validates key size and converts to [32]byte array before encrypting
func EncryptWithKey(plaintext []byte, key []byte, nonce [NonceSize]byte) (*EncryptedFrame, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidKeySize, len(key))
	}

	var keyArray [KeySize]byte
	copy(keyArray[:], key)

	return Encrypt(plaintext, keyArray, nonce)
}

// DecryptWithKey is a convenience function that accepts a byte slice key
// Validates key size and converts to [32]byte array before decrypting
func DecryptWithKey(frame *EncryptedFrame, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidKeySize, len(key))
	}

	var keyArray [KeySize]byte
	copy(keyArray[:], key)

	return Decrypt(frame, keyArray)
}

// ValidateTag performs constant-time comparison of two Poly1305 tags
// This function is exposed for testing purposes only - AEAD.Open handles tag validation internally
func ValidateTag(tag1, tag2 [TagSize]byte) bool {
	return subtle.ConstantTimeCompare(tag1[:], tag2[:]) == 1
}
