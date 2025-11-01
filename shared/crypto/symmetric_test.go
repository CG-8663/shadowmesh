package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// generateTestKey creates a random 256-bit key for testing
func generateTestKey() [32]byte {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		panic(err)
	}
	return key
}

// TestEncryptDecryptRoundtrip verifies successful encryption/decryption
func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		t.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	// Test with various plaintext sizes
	testCases := []struct {
		name      string
		plaintext []byte
	}{
		{"Empty frame", []byte{}},
		{"Small frame", []byte("Hello, ShadowMesh!")},
		{"Typical Ethernet frame (1500 bytes)", make([]byte, 1500)},
		{"Jumbo frame (9000 bytes)", make([]byte, 9000)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Fill with test data
			plaintext := make([]byte, len(tc.plaintext))
			copy(plaintext, tc.plaintext)
			if len(plaintext) > len(tc.plaintext) {
				rand.Read(plaintext)
			}

			// Encrypt
			ciphertext, err := encryptor.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			// Verify ciphertext length
			expectedLen := len(plaintext) + OverheadSize
			if len(ciphertext) != expectedLen {
				t.Errorf("Ciphertext length mismatch: got %d, want %d", len(ciphertext), expectedLen)
			}

			// Decrypt
			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			// Verify plaintext matches
			if !bytes.Equal(decrypted, plaintext) {
				t.Errorf("Decrypted plaintext doesn't match original")
			}
		})
	}
}

// TestMultipleFramesUniqueNonces verifies nonce uniqueness across multiple frames
func TestMultipleFramesUniqueNonces(t *testing.T) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		t.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	const numFrames = 10000
	nonces := make(map[[NonceSize]byte]bool)
	plaintext := []byte("Test frame")

	for i := 0; i < numFrames; i++ {
		ciphertext, err := encryptor.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt failed on frame %d: %v", i, err)
		}

		// Extract nonce (first 12 bytes)
		var nonce [NonceSize]byte
		copy(nonce[:], ciphertext[:NonceSize])

		// Check for duplicates
		if nonces[nonce] {
			t.Fatalf("Duplicate nonce detected on frame %d", i)
		}
		nonces[nonce] = true
	}

	// Verify counter incremented correctly
	expectedCounter := uint64(numFrames)
	if encryptor.GetCounter() != expectedCounter {
		t.Errorf("Counter mismatch: got %d, want %d", encryptor.GetCounter(), expectedCounter)
	}
}

// TestDecryptCorruptedCiphertext verifies decryption fails with corrupted data
func TestDecryptCorruptedCiphertext(t *testing.T) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		t.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	plaintext := []byte("Test frame for corruption")
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Test corruption at various positions
	testCases := []struct {
		name     string
		position int
		desc     string
	}{
		{"Corrupt nonce", 5, "nonce corruption"},
		{"Corrupt ciphertext", NonceSize + 5, "ciphertext corruption"},
		{"Corrupt tag", len(ciphertext) - 5, "tag corruption"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Make a copy and corrupt one byte
			corrupted := make([]byte, len(ciphertext))
			copy(corrupted, ciphertext)
			corrupted[tc.position] ^= 0xFF

			// Decryption should fail
			_, err := encryptor.Decrypt(corrupted)
			if err != ErrAuthenticationFailed {
				t.Errorf("Expected ErrAuthenticationFailed for %s, got: %v", tc.desc, err)
			}
		})
	}
}

// TestDecryptCorruptedTag specifically verifies tag validation
func TestDecryptCorruptedTag(t *testing.T) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		t.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	plaintext := []byte("Test frame")
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Corrupt the last byte of the tag
	corrupted := make([]byte, len(ciphertext))
	copy(corrupted, ciphertext)
	corrupted[len(corrupted)-1] ^= 0x01

	// Decryption must fail with authentication error
	_, err = encryptor.Decrypt(corrupted)
	if err != ErrAuthenticationFailed {
		t.Errorf("Expected ErrAuthenticationFailed for tag corruption, got: %v", err)
	}
}

// TestDecryptWrongKey verifies decryption fails with wrong key
func TestDecryptWrongKey(t *testing.T) {
	key1 := generateTestKey()
	key2 := generateTestKey()

	encryptor1, err := NewFrameEncryptor(key1)
	if err != nil {
		t.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	encryptor2, err := NewFrameEncryptor(key2)
	if err != nil {
		t.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	plaintext := []byte("Test frame")

	// Encrypt with key1
	ciphertext, err := encryptor1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Try to decrypt with key2 (should fail)
	_, err = encryptor2.Decrypt(ciphertext)
	if err != ErrAuthenticationFailed {
		t.Errorf("Expected ErrAuthenticationFailed when using wrong key, got: %v", err)
	}
}

// TestDecryptInvalidCiphertext verifies handling of malformed input
func TestDecryptInvalidCiphertext(t *testing.T) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		t.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	testCases := []struct {
		name       string
		ciphertext []byte
		wantErr    error
		desc       string
	}{
		{"Empty ciphertext", []byte{}, ErrInvalidCiphertext, "empty data"},
		{"Too short (only nonce)", make([]byte, NonceSize), ErrInvalidCiphertext, "only nonce"},
		{"Too short (nonce + partial tag)", make([]byte, NonceSize+10), ErrInvalidCiphertext, "partial tag"},
		{"Minimum valid (nonce + tag, empty plaintext)", make([]byte, OverheadSize), ErrAuthenticationFailed, "empty plaintext"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := encryptor.Decrypt(tc.ciphertext)
			if err != tc.wantErr {
				t.Errorf("Expected error %v, got: %v", tc.wantErr, err)
			}
		})
	}
}

// TestNonceUniquenessAcrossInstances verifies different instances generate different nonces
func TestNonceUniquenessAcrossInstances(t *testing.T) {
	key := generateTestKey()
	plaintext := []byte("Test frame")

	// Create two encryptors with same key
	enc1, err := NewFrameEncryptor(key)
	if err != nil {
		t.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	enc2, err := NewFrameEncryptor(key)
	if err != nil {
		t.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	// Encrypt same plaintext with both
	ct1, err := enc1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	ct2, err := enc2.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Extract nonces
	nonce1 := ct1[:NonceSize]
	nonce2 := ct2[:NonceSize]

	// Nonces should be different due to different random prefixes
	if bytes.Equal(nonce1, nonce2) {
		t.Error("Nonces are identical across different encryptor instances (very unlikely)")
	}

	// Ciphertexts should be different (different nonces)
	if bytes.Equal(ct1, ct2) {
		t.Error("Ciphertexts are identical across different encryptor instances")
	}
}

// TestCounterIncrement verifies the counter increments correctly
func TestCounterIncrement(t *testing.T) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		t.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	plaintext := []byte("Test frame")

	// Initial counter should be 0
	if encryptor.GetCounter() != 0 {
		t.Errorf("Initial counter should be 0, got: %d", encryptor.GetCounter())
	}

	// Encrypt 5 frames
	for i := 1; i <= 5; i++ {
		_, err := encryptor.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt failed: %v", err)
		}

		if encryptor.GetCounter() != uint64(i) {
			t.Errorf("Counter after %d encryptions: got %d, want %d", i, encryptor.GetCounter(), i)
		}
	}
}

// TestConstantTimeComparison verifies tag validation is constant-time
// Note: This is a behavioral test, not a timing-based test
func TestConstantTimeComparison(t *testing.T) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		t.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	plaintext := []byte("Test frame for timing attack resistance")
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Test that all tag corruption patterns fail consistently
	// (ChaCha20-Poly1305 implementation should use constant-time comparison)
	for i := 0; i < 16; i++ {
		corrupted := make([]byte, len(ciphertext))
		copy(corrupted, ciphertext)

		// Flip each bit of the tag
		tagStart := len(corrupted) - TagSize
		corrupted[tagStart+i] ^= 0x01

		_, err := encryptor.Decrypt(corrupted)
		if err != ErrAuthenticationFailed {
			t.Errorf("Tag corruption at position %d should fail with ErrAuthenticationFailed, got: %v", i, err)
		}
	}
}

// TestLargeFrameEncryption verifies handling of large frames
func TestLargeFrameEncryption(t *testing.T) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		t.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	// Test with 64KB frame (larger than typical MTU)
	plaintext := make([]byte, 65536)
	rand.Read(plaintext)

	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Decrypted large frame doesn't match original")
	}
}
