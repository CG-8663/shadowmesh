package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// ChaCha20Cipher handles encryption/decryption using ChaCha20-Poly1305
type ChaCha20Cipher struct {
	aead cipher.AEAD
}

// NewChaCha20Cipher creates a new ChaCha20-Poly1305 cipher with the given 32-byte key
func NewChaCha20Cipher(key []byte) (*ChaCha20Cipher, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid key size: got %d, want %d", len(key), chacha20poly1305.KeySize)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 cipher: %w", err)
	}

	return &ChaCha20Cipher{aead: aead}, nil
}

// Encrypt encrypts plaintext and returns ciphertext with authentication tag
// Format: [12 bytes nonce][N bytes encrypted data][16 bytes auth tag]
func (c *ChaCha20Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	// Generate random nonce (96 bits = 12 bytes for XChaCha20-Poly1305)
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	// Output: nonce || ciphertext || tag
	ciphertext := c.aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext and verifies authentication tag
func (c *ChaCha20Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := chacha20poly1305.NonceSizeX
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce from beginning
	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]

	// Decrypt and verify
	plaintext, err := c.aead.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// GenerateKey generates a random 32-byte key for ChaCha20-Poly1305
func GenerateKey() ([]byte, error) {
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}
