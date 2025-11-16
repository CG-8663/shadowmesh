package symmetric

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// TestEncryptDecryptRoundTrip tests basic encryption and decryption
func TestEncryptDecryptRoundTrip(t *testing.T) {
	testCases := []struct {
		name      string
		plaintext []byte
	}{
		{"Empty plaintext", []byte{}},
		{"Small plaintext (16 bytes)", []byte("Hello, ShadowMesh")},
		{"Medium plaintext (1 KB)", make([]byte, 1024)},
		{"Large plaintext (10 KB)", make([]byte, 10*1024)},
		{"Jumbo frame (9000 bytes)", make([]byte, 9000)},
		{"Very large (1 MB)", make([]byte, 1024*1024)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate random plaintext if empty
			plaintext := tc.plaintext
			if len(plaintext) > 0 && plaintext[0] == 0 {
				_, err := rand.Read(plaintext)
				if err != nil {
					t.Fatalf("Failed to generate random plaintext: %v", err)
				}
			}

			// Generate key and nonce
			var key [KeySize]byte
			var nonce [NonceSize]byte
			_, err := rand.Read(key[:])
			if err != nil {
				t.Fatalf("Failed to generate key: %v", err)
			}
			_, err = rand.Read(nonce[:])
			if err != nil {
				t.Fatalf("Failed to generate nonce: %v", err)
			}

			// Encrypt
			frame, err := Encrypt(plaintext, key, nonce)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// Verify nonce is preserved
			if frame.Nonce != nonce {
				t.Errorf("Nonce mismatch: got %x, want %x", frame.Nonce, nonce)
			}

			// Verify ciphertext length (plaintext + 16-byte tag)
			expectedLen := len(plaintext) + TagSize
			if len(frame.Ciphertext) != expectedLen {
				t.Errorf("Ciphertext length mismatch: got %d, want %d", len(frame.Ciphertext), expectedLen)
			}

			// Decrypt
			decrypted, err := Decrypt(frame, key)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			// Verify plaintext matches
			if !bytes.Equal(decrypted, plaintext) {
				t.Errorf("Decrypted plaintext doesn't match original (length: %d vs %d)", len(decrypted), len(plaintext))
			}
		})
	}
}

// TestInvalidKeySize tests error handling for invalid key sizes
func TestInvalidKeySize(t *testing.T) {
	testCases := []struct {
		name    string
		keySize int
	}{
		{"Empty key", 0},
		{"Too small (16 bytes)", 16},
		{"Too large (64 bytes)", 64},
		{"Wrong size (31 bytes)", 31},
		{"Wrong size (33 bytes)", 33},
	}

	plaintext := []byte("test message")
	var nonce [NonceSize]byte

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			invalidKey := make([]byte, tc.keySize)
			_, err := EncryptWithKey(plaintext, invalidKey, nonce)
			if err == nil {
				t.Error("Expected error for invalid key size, got nil")
			}
			// Check if error wraps ErrInvalidKeySize
			if !isErrorType(err, ErrInvalidKeySize) {
				t.Errorf("Expected ErrInvalidKeySize, got: %v", err)
			}
		})
	}
}

// Helper function to check if error contains a specific error type
func isErrorType(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}
	// Check if error message contains target error message
	return err.Error()[:len(target.Error())] == target.Error()
}

// TestTamperedCiphertext tests authentication tag validation
func TestTamperedCiphertext(t *testing.T) {
	plaintext := []byte("Secret message that must not be tampered with")
	var key [KeySize]byte
	var nonce [NonceSize]byte
	rand.Read(key[:])
	rand.Read(nonce[:])

	// Encrypt
	frame, err := Encrypt(plaintext, key, nonce)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Tamper with ciphertext (flip one bit)
	frame.Ciphertext[len(frame.Ciphertext)/2] ^= 0x01

	// Attempt to decrypt - should fail due to tag mismatch
	_, err = Decrypt(frame, key)
	if err == nil {
		t.Error("Expected decryption to fail for tampered ciphertext")
	}
	if !isErrorType(err, ErrDecryptionFailed) {
		t.Errorf("Expected ErrDecryptionFailed, got: %v", err)
	}
}

// TestWrongKey tests decryption with incorrect key
func TestWrongKey(t *testing.T) {
	plaintext := []byte("Top secret data")
	var key1, key2 [KeySize]byte
	var nonce [NonceSize]byte
	rand.Read(key1[:])
	rand.Read(key2[:])
	rand.Read(nonce[:])

	// Encrypt with key1
	frame, err := Encrypt(plaintext, key1, nonce)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Attempt to decrypt with key2 - should fail
	_, err = Decrypt(frame, key2)
	if err == nil {
		t.Error("Expected decryption to fail with wrong key")
	}
	if !isErrorType(err, ErrDecryptionFailed) {
		t.Errorf("Expected ErrDecryptionFailed, got: %v", err)
	}
}

// TestTamperedTag tests tampering with the authentication tag
func TestTamperedTag(t *testing.T) {
	plaintext := []byte("Authenticated data")
	var key [KeySize]byte
	var nonce [NonceSize]byte
	rand.Read(key[:])
	rand.Read(nonce[:])

	// Encrypt
	frame, err := Encrypt(plaintext, key, nonce)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Tamper with tag (last 16 bytes of ciphertext)
	frame.Ciphertext[len(frame.Ciphertext)-1] ^= 0x80

	// Attempt to decrypt - should fail
	_, err = Decrypt(frame, key)
	if err == nil {
		t.Error("Expected decryption to fail for tampered tag")
	}
}

// TestTamperedNonce tests tampering with the nonce
func TestTamperedNonce(t *testing.T) {
	plaintext := []byte("Nonce-protected data")
	var key [KeySize]byte
	var nonce [NonceSize]byte
	rand.Read(key[:])
	rand.Read(nonce[:])

	// Encrypt
	frame, err := Encrypt(plaintext, key, nonce)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Tamper with nonce
	frame.Nonce[0] ^= 0x01

	// Attempt to decrypt - should fail due to nonce mismatch
	_, err = Decrypt(frame, key)
	if err == nil {
		t.Error("Expected decryption to fail for tampered nonce")
	}
}

// TestInvalidCiphertext tests error handling for invalid ciphertext
func TestInvalidCiphertext(t *testing.T) {
	var key [KeySize]byte
	rand.Read(key[:])

	testCases := []struct {
		name       string
		ciphertext []byte
	}{
		{"Empty ciphertext", []byte{}},
		{"Too short (15 bytes - less than tag size)", make([]byte, 15)},
		{"Just tag size (16 bytes)", make([]byte, 16)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var nonce [NonceSize]byte
			frame := &EncryptedFrame{
				Nonce:      nonce,
				Ciphertext: tc.ciphertext,
			}

			_, err := Decrypt(frame, key)
			if err == nil {
				t.Error("Expected error for invalid ciphertext")
			}
		})
	}
}

// TestNilFrame tests error handling for nil frame
func TestNilFrame(t *testing.T) {
	var key [KeySize]byte
	_, err := Decrypt(nil, key)
	if err == nil {
		t.Error("Expected error for nil frame")
	}
}

// TestEncryptWithKeyConvenience tests the convenience function
func TestEncryptWithKeyConvenience(t *testing.T) {
	plaintext := []byte("Convenience function test")
	key := make([]byte, KeySize)
	var nonce [NonceSize]byte
	rand.Read(key)
	rand.Read(nonce[:])

	frame, err := EncryptWithKey(plaintext, key, nonce)
	if err != nil {
		t.Fatalf("EncryptWithKey failed: %v", err)
	}

	decrypted, err := DecryptWithKey(frame, key)
	if err != nil {
		t.Fatalf("DecryptWithKey failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Decrypted plaintext doesn't match original")
	}
}

// TestDifferentNoncesSameKey tests that different nonces produce different ciphertexts
func TestDifferentNoncesSameKey(t *testing.T) {
	plaintext := []byte("Same plaintext, different nonces")
	var key [KeySize]byte
	rand.Read(key[:])

	var nonce1, nonce2 [NonceSize]byte
	rand.Read(nonce1[:])
	rand.Read(nonce2[:])

	frame1, _ := Encrypt(plaintext, key, nonce1)
	frame2, _ := Encrypt(plaintext, key, nonce2)

	// Ciphertexts should be different (even for same plaintext + key)
	if bytes.Equal(frame1.Ciphertext, frame2.Ciphertext) {
		t.Error("Different nonces should produce different ciphertexts")
	}

	// Both should decrypt correctly
	decrypted1, _ := Decrypt(frame1, key)
	decrypted2, _ := Decrypt(frame2, key)

	if !bytes.Equal(decrypted1, plaintext) || !bytes.Equal(decrypted2, plaintext) {
		t.Error("Decryption failed for different nonces")
	}
}

// Benchmark: Encrypt small frame (1 KB)
func BenchmarkEncrypt1KB(b *testing.B) {
	plaintext := make([]byte, 1024)
	var key [KeySize]byte
	var nonce [NonceSize]byte
	rand.Read(plaintext)
	rand.Read(key[:])

	b.ResetTimer()
	b.SetBytes(1024)

	for i := 0; i < b.N; i++ {
		nonce[0] = byte(i) // Vary nonce
		_, err := Encrypt(plaintext, key, nonce)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark: Encrypt medium frame (10 KB)
func BenchmarkEncrypt10KB(b *testing.B) {
	plaintext := make([]byte, 10*1024)
	var key [KeySize]byte
	var nonce [NonceSize]byte
	rand.Read(plaintext)
	rand.Read(key[:])

	b.ResetTimer()
	b.SetBytes(10 * 1024)

	for i := 0; i < b.N; i++ {
		nonce[0] = byte(i)
		_, err := Encrypt(plaintext, key, nonce)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark: Encrypt large frame (100 KB)
func BenchmarkEncrypt100KB(b *testing.B) {
	plaintext := make([]byte, 100*1024)
	var key [KeySize]byte
	var nonce [NonceSize]byte
	rand.Read(plaintext)
	rand.Read(key[:])

	b.ResetTimer()
	b.SetBytes(100 * 1024)

	for i := 0; i < b.N; i++ {
		nonce[0] = byte(i)
		_, err := Encrypt(plaintext, key, nonce)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark: Encrypt very large frame (1 MB)
func BenchmarkEncrypt1MB(b *testing.B) {
	plaintext := make([]byte, 1024*1024)
	var key [KeySize]byte
	var nonce [NonceSize]byte
	rand.Read(plaintext)
	rand.Read(key[:])

	b.ResetTimer()
	b.SetBytes(1024 * 1024)

	for i := 0; i < b.N; i++ {
		nonce[0] = byte(i)
		_, err := Encrypt(plaintext, key, nonce)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark: Decrypt small frame (1 KB)
func BenchmarkDecrypt1KB(b *testing.B) {
	plaintext := make([]byte, 1024)
	var key [KeySize]byte
	var nonce [NonceSize]byte
	rand.Read(plaintext)
	rand.Read(key[:])
	rand.Read(nonce[:])

	frame, _ := Encrypt(plaintext, key, nonce)

	b.ResetTimer()
	b.SetBytes(1024)

	for i := 0; i < b.N; i++ {
		_, err := Decrypt(frame, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark: Decrypt medium frame (10 KB)
func BenchmarkDecrypt10KB(b *testing.B) {
	plaintext := make([]byte, 10*1024)
	var key [KeySize]byte
	var nonce [NonceSize]byte
	rand.Read(plaintext)
	rand.Read(key[:])
	rand.Read(nonce[:])

	frame, _ := Encrypt(plaintext, key, nonce)

	b.ResetTimer()
	b.SetBytes(10 * 1024)

	for i := 0; i < b.N; i++ {
		_, err := Decrypt(frame, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark: Decrypt large frame (100 KB)
func BenchmarkDecrypt100KB(b *testing.B) {
	plaintext := make([]byte, 100*1024)
	var key [KeySize]byte
	var nonce [NonceSize]byte
	rand.Read(plaintext)
	rand.Read(key[:])
	rand.Read(nonce[:])

	frame, _ := Encrypt(plaintext, key, nonce)

	b.ResetTimer()
	b.SetBytes(100 * 1024)

	for i := 0; i < b.N; i++ {
		_, err := Decrypt(frame, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark: Decrypt very large frame (1 MB)
func BenchmarkDecrypt1MB(b *testing.B) {
	plaintext := make([]byte, 1024*1024)
	var key [KeySize]byte
	var nonce [NonceSize]byte
	rand.Read(plaintext)
	rand.Read(key[:])
	rand.Read(nonce[:])

	frame, _ := Encrypt(plaintext, key, nonce)

	b.ResetTimer()
	b.SetBytes(1024 * 1024)

	for i := 0; i < b.N; i++ {
		_, err := Decrypt(frame, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}
