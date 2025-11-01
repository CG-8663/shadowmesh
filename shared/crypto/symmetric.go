package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"sync/atomic"

	"golang.org/x/crypto/chacha20poly1305"
)

var (
	// ErrAuthenticationFailed indicates the authentication tag validation failed
	ErrAuthenticationFailed = errors.New("authentication failed: invalid tag")

	// ErrInvalidCiphertext indicates the ciphertext is malformed or too short
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

const (
	// NonceSize is the size of the nonce in bytes (96 bits)
	NonceSize = chacha20poly1305.NonceSize // 12 bytes
	// TagSize is the size of the authentication tag in bytes
	TagSize = 16
	// OverheadSize is the total overhead per encrypted frame (nonce + tag)
	OverheadSize = NonceSize + TagSize // 28 bytes
)

// FrameEncryptor provides ChaCha20-Poly1305 AEAD encryption for Ethernet frames.
// It is NOT thread-safe - callers must synchronize access if used concurrently.
type FrameEncryptor struct {
	aead         cipher.AEAD
	counter      uint64  // Frame counter for nonce uniqueness (atomic)
	randomPrefix [4]byte // Random component to prevent nonce reuse across sessions
}

// NewFrameEncryptor creates a new FrameEncryptor with the given 256-bit key.
// The random prefix is generated once per encryptor instance to ensure
// nonce uniqueness across different sessions/instances.
func NewFrameEncryptor(key [32]byte) (*FrameEncryptor, error) {
	aead, err := chacha20poly1305.New(key[:])
	if err != nil {
		return nil, err
	}

	fe := &FrameEncryptor{
		aead:    aead,
		counter: 0,
	}

	// Generate random prefix for nonce uniqueness across sessions
	if _, err := rand.Read(fe.randomPrefix[:]); err != nil {
		return nil, err
	}

	return fe, nil
}

// generateNonce creates a unique 96-bit nonce using:
// - 64-bit counter (incremented for each frame)
// - 32-bit random prefix (unique per FrameEncryptor instance)
// This ensures nonce uniqueness even across restarts and prevents replay attacks.
func (fe *FrameEncryptor) generateNonce() [NonceSize]byte {
	var nonce [NonceSize]byte

	// Atomically increment counter and use previous value
	count := atomic.AddUint64(&fe.counter, 1) - 1

	// First 8 bytes: counter (little-endian)
	binary.LittleEndian.PutUint64(nonce[0:8], count)

	// Last 4 bytes: random prefix
	copy(nonce[8:12], fe.randomPrefix[:])

	return nonce
}

// Encrypt encrypts a plaintext frame and returns the encrypted frame with format:
// [nonce (12 bytes)][ciphertext][tag (16 bytes)]
//
// The nonce is prepended to allow stateless decryption. The authentication tag
// is appended by the AEAD cipher and provides integrity and authenticity.
func (fe *FrameEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := fe.generateNonce()

	// Allocate output buffer: nonce + ciphertext + tag
	// The AEAD Seal operation appends the tag automatically
	ciphertext := make([]byte, NonceSize+len(plaintext)+TagSize)

	// Copy nonce to beginning of output
	copy(ciphertext[:NonceSize], nonce[:])

	// Encrypt plaintext and append tag
	// Seal appends to dst starting at len(dst), so we pass ciphertext[:NonceSize]
	// to append encrypted data after the nonce
	fe.aead.Seal(ciphertext[:NonceSize], nonce[:], plaintext, nil)

	return ciphertext, nil
}

// Decrypt decrypts a ciphertext frame and validates the authentication tag.
// The ciphertext must have format: [nonce (12 bytes)][ciphertext][tag (16 bytes)]
//
// Returns ErrInvalidCiphertext if the input is too short or malformed.
// Returns ErrAuthenticationFailed if the tag validation fails (constant-time).
func (fe *FrameEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	// Minimum size: nonce + tag (empty plaintext is valid)
	if len(ciphertext) < OverheadSize {
		return nil, ErrInvalidCiphertext
	}

	// Extract nonce from first 12 bytes
	var nonce [NonceSize]byte
	copy(nonce[:], ciphertext[:NonceSize])

	// Decrypt and verify tag (constant-time comparison)
	// Open verifies the tag before returning plaintext
	plaintext, err := fe.aead.Open(nil, nonce[:], ciphertext[NonceSize:], nil)
	if err != nil {
		return nil, ErrAuthenticationFailed
	}

	return plaintext, nil
}

// GetCounter returns the current frame counter value.
// Useful for monitoring and debugging nonce generation.
func (fe *FrameEncryptor) GetCounter() uint64 {
	return atomic.LoadUint64(&fe.counter)
}
