// Package keystore provides encrypted storage for ShadowMesh hybrid keypairs.
package keystore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

var (
	// ErrEncryptionFailed indicates encryption operation failed
	ErrEncryptionFailed = errors.New("encryption failed")
	// ErrDecryptionFailed indicates decryption operation failed
	ErrDecryptionFailed = errors.New("decryption failed")
	// ErrInvalidIVSize indicates IV is not the correct size
	ErrInvalidIVSize = errors.New("IV must be 12 bytes for AES-GCM")
	// ErrInvalidKeyLength indicates key is not 32 bytes
	ErrInvalidKeyLength = errors.New("key must be 32 bytes for AES-256-GCM")
	// ErrEmptyPlaintext indicates plaintext is empty
	ErrEmptyPlaintext = errors.New("plaintext cannot be empty")
	// ErrEmptyCiphertext indicates ciphertext is empty
	ErrEmptyCiphertext = errors.New("ciphertext cannot be empty")
)

// EncryptedData holds encrypted data and its IV (nonce).
// The ciphertext includes the GCM authentication tag (appended by GCM).
type EncryptedData struct {
	// Ciphertext contains the encrypted data + 16-byte GCM tag
	Ciphertext []byte
	// IV is the 12-byte initialization vector (nonce) used for encryption
	IV [IVSize]byte
}

// Encrypt encrypts plaintext using AES-256-GCM with the provided key.
//
// Parameters:
//   - plaintext: Data to encrypt (must not be empty)
//   - key: 32-byte AES-256 key (derived from PBKDF2)
//
// Returns:
//   - *EncryptedData containing ciphertext (with GCM tag) and IV
//   - error if encryption fails
//
// Security Notes:
//   - Uses AES-256-GCM (AEAD cipher providing confidentiality + authenticity)
//   - IV is randomly generated for each encryption (12 bytes)
//   - GCM tag (16 bytes) is automatically appended to ciphertext
//   - Never reuse the same (key, IV) pair for different plaintexts
//   - Constant-time operations prevent timing attacks
func Encrypt(plaintext []byte, key [32]byte) (*EncryptedData, error) {
	// Validate inputs
	if len(plaintext) == 0 {
		return nil, ErrEmptyPlaintext
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create cipher: %v", ErrEncryptionFailed, err)
	}

	// Create GCM mode cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create GCM: %v", ErrEncryptionFailed, err)
	}

	// Generate random IV (nonce)
	var iv [IVSize]byte
	if _, err := rand.Read(iv[:]); err != nil {
		return nil, fmt.Errorf("%w: failed to generate IV: %v", ErrEncryptionFailed, err)
	}

	// Encrypt plaintext
	// GCM.Seal appends the ciphertext and tag to dst (we pass nil as dst)
	// Output: ciphertext || tag (16 bytes)
	ciphertext := gcm.Seal(nil, iv[:], plaintext, nil)

	return &EncryptedData{
		Ciphertext: ciphertext,
		IV:         iv,
	}, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with the provided key.
//
// Parameters:
//   - encrypted: EncryptedData containing ciphertext (with GCM tag) and IV
//   - key: 32-byte AES-256 key (same key used for encryption)
//
// Returns:
//   - Decrypted plaintext
//   - error if decryption or authentication fails
//
// Security Notes:
//   - GCM automatically verifies authentication tag during decryption
//   - Returns error if ciphertext has been tampered with
//   - Returns error if wrong key is used
//   - Constant-time comparison prevents timing attacks
func Decrypt(encrypted *EncryptedData, key [32]byte) ([]byte, error) {
	// Validate inputs
	if encrypted == nil {
		return nil, fmt.Errorf("%w: encrypted data is nil", ErrDecryptionFailed)
	}
	if len(encrypted.Ciphertext) == 0 {
		return nil, ErrEmptyCiphertext
	}

	// GCM tag is 16 bytes, so ciphertext must be at least 16 bytes
	if len(encrypted.Ciphertext) < 16 {
		return nil, fmt.Errorf("%w: ciphertext too short (must be at least 16 bytes)", ErrDecryptionFailed)
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create cipher: %v", ErrDecryptionFailed, err)
	}

	// Create GCM mode cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create GCM: %v", ErrDecryptionFailed, err)
	}

	// Decrypt ciphertext
	// GCM.Open verifies the tag and decrypts in one operation
	// Returns error if tag is invalid (tampering detected or wrong key)
	plaintext, err := gcm.Open(nil, encrypted.IV[:], encrypted.Ciphertext, nil)
	if err != nil {
		// This could be wrong passphrase or corrupted data
		return nil, fmt.Errorf("%w: authentication failed or wrong key", ErrDecryptionFailed)
	}

	return plaintext, nil
}

// EncryptWithExternalIV encrypts plaintext with a provided IV (for testing).
// PRODUCTION CODE SHOULD USE Encrypt() WHICH GENERATES RANDOM IV.
//
// This function exists only for deterministic testing and should not be
// used in production code. Always use Encrypt() which generates random IVs.
func EncryptWithExternalIV(plaintext []byte, key [32]byte, iv [IVSize]byte) (*EncryptedData, error) {
	// Validate inputs
	if len(plaintext) == 0 {
		return nil, ErrEmptyPlaintext
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create cipher: %v", ErrEncryptionFailed, err)
	}

	// Create GCM mode cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create GCM: %v", ErrEncryptionFailed, err)
	}

	// Encrypt plaintext with provided IV
	ciphertext := gcm.Seal(nil, iv[:], plaintext, nil)

	return &EncryptedData{
		Ciphertext: ciphertext,
		IV:         iv,
	}, nil
}
