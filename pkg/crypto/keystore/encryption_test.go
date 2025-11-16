package keystore

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// TestEncryptDecryptRoundTrip tests basic encryption and decryption
func TestEncryptDecryptRoundTrip(t *testing.T) {
	plaintext := []byte("This is a secret message that needs encryption!")
	var key [32]byte
	rand.Read(key[:])

	// Encrypt
	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// Verify ciphertext is not equal to plaintext
	if bytes.Equal(encrypted.Ciphertext, plaintext) {
		t.Error("Ciphertext should not equal plaintext")
	}

	// Verify ciphertext is longer (includes 16-byte tag)
	if len(encrypted.Ciphertext) != len(plaintext)+16 {
		t.Errorf("Ciphertext length = %d, want %d (plaintext + 16-byte tag)",
			len(encrypted.Ciphertext), len(plaintext)+16)
	}

	// Decrypt
	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}

	// Verify decrypted matches original plaintext
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypted text does not match original.\nGot:  %s\nWant: %s",
			decrypted, plaintext)
	}
}

// TestEncryptDifferentCiphertexts tests that same plaintext produces different ciphertexts
func TestEncryptDifferentCiphertexts(t *testing.T) {
	plaintext := []byte("Same plaintext encrypted twice")
	var key [32]byte
	rand.Read(key[:])

	// Encrypt twice
	encrypted1, err1 := Encrypt(plaintext, key)
	encrypted2, err2 := Encrypt(plaintext, key)

	if err1 != nil || err2 != nil {
		t.Fatalf("Encrypt() failed: %v, %v", err1, err2)
	}

	// IVs should be different (random)
	if encrypted1.IV == encrypted2.IV {
		t.Error("Two encryptions produced same IV (should be random)")
	}

	// Ciphertexts should be different (due to different IVs)
	if bytes.Equal(encrypted1.Ciphertext, encrypted2.Ciphertext) {
		t.Error("Two encryptions produced same ciphertext (should differ due to random IV)")
	}

	// Both should decrypt to same plaintext
	decrypted1, _ := Decrypt(encrypted1, key)
	decrypted2, _ := Decrypt(encrypted2, key)

	if !bytes.Equal(decrypted1, plaintext) || !bytes.Equal(decrypted2, plaintext) {
		t.Error("Decryption failed for one or both ciphertexts")
	}
}

// TestDecryptWithWrongKey tests that decryption fails with wrong key
func TestDecryptWithWrongKey(t *testing.T) {
	plaintext := []byte("Secret message")
	var key1, key2 [32]byte
	rand.Read(key1[:])
	rand.Read(key2[:])

	// Encrypt with key1
	encrypted, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// Try to decrypt with key2
	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Error("Decrypt() should fail with wrong key")
	}
}

// TestDecryptTamperedCiphertext tests that decryption fails with tampered data
func TestDecryptTamperedCiphertext(t *testing.T) {
	plaintext := []byte("Secret message")
	var key [32]byte
	rand.Read(key[:])

	// Encrypt
	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// Tamper with ciphertext (flip one bit)
	encrypted.Ciphertext[0] ^= 0x01

	// Try to decrypt
	_, err = Decrypt(encrypted, key)
	if err == nil {
		t.Error("Decrypt() should fail with tampered ciphertext")
	}
}

// TestDecryptTamperedIV tests that decryption fails with tampered IV
func TestDecryptTamperedIV(t *testing.T) {
	plaintext := []byte("Secret message")
	var key [32]byte
	rand.Read(key[:])

	// Encrypt
	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// Tamper with IV (flip one bit)
	encrypted.IV[0] ^= 0x01

	// Try to decrypt
	_, err = Decrypt(encrypted, key)
	if err == nil {
		t.Error("Decrypt() should fail with tampered IV")
	}
}

// TestEncryptEmptyPlaintext tests error handling for empty plaintext
func TestEncryptEmptyPlaintext(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	_, err := Encrypt([]byte{}, key)
	if err == nil {
		t.Error("Encrypt() should fail with empty plaintext")
	}
	if err != ErrEmptyPlaintext {
		t.Errorf("Expected ErrEmptyPlaintext, got %v", err)
	}
}

// TestDecryptEmptyCiphertext tests error handling for empty ciphertext
func TestDecryptEmptyCiphertext(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	encrypted := &EncryptedData{
		Ciphertext: []byte{},
		IV:         [IVSize]byte{},
	}

	_, err := Decrypt(encrypted, key)
	if err == nil {
		t.Error("Decrypt() should fail with empty ciphertext")
	}
}

// TestDecryptNilEncryptedData tests error handling for nil encrypted data
func TestDecryptNilEncryptedData(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	_, err := Decrypt(nil, key)
	if err == nil {
		t.Error("Decrypt() should fail with nil encrypted data")
	}
}

// TestDecryptShortCiphertext tests error handling for ciphertext shorter than tag
func TestDecryptShortCiphertext(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	// Ciphertext must be at least 16 bytes (GCM tag size)
	encrypted := &EncryptedData{
		Ciphertext: []byte{0x01, 0x02, 0x03}, // Only 3 bytes
		IV:         [IVSize]byte{},
	}

	_, err := Decrypt(encrypted, key)
	if err == nil {
		t.Error("Decrypt() should fail with ciphertext shorter than GCM tag (16 bytes)")
	}
}

// TestEncryptWithExternalIV tests deterministic encryption with provided IV
func TestEncryptWithExternalIV(t *testing.T) {
	plaintext := []byte("Test message")
	var key [32]byte
	var iv [IVSize]byte
	rand.Read(key[:])
	rand.Read(iv[:])

	// Encrypt twice with same IV
	encrypted1, err1 := EncryptWithExternalIV(plaintext, key, iv)
	encrypted2, err2 := EncryptWithExternalIV(plaintext, key, iv)

	if err1 != nil || err2 != nil {
		t.Fatalf("EncryptWithExternalIV() failed: %v, %v", err1, err2)
	}

	// Ciphertexts should be identical (same key, IV, plaintext)
	if !bytes.Equal(encrypted1.Ciphertext, encrypted2.Ciphertext) {
		t.Error("Same (key, IV, plaintext) should produce identical ciphertext")
	}

	// Decrypt
	decrypted, err := Decrypt(encrypted1, key)
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Decrypted text does not match original")
	}
}

// TestEncryptLargePlaintext tests encryption of large data (simulating keystore data)
func TestEncryptLargePlaintext(t *testing.T) {
	// Simulate keystore JSON (~12KB for hybrid keypair)
	plaintext := make([]byte, 12000)
	rand.Read(plaintext)

	var key [32]byte
	rand.Read(key[:])

	// Encrypt
	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() failed for large plaintext: %v", err)
	}

	// Verify size (plaintext + 16-byte tag)
	if len(encrypted.Ciphertext) != len(plaintext)+16 {
		t.Errorf("Ciphertext length = %d, want %d",
			len(encrypted.Ciphertext), len(plaintext)+16)
	}

	// Decrypt
	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt() failed for large plaintext: %v", err)
	}

	// Verify
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Decrypted large plaintext does not match original")
	}
}

// BenchmarkEncrypt benchmarks encryption performance
func BenchmarkEncrypt(b *testing.B) {
	plaintext := make([]byte, 1024) // 1KB
	rand.Read(plaintext)
	var key [32]byte
	rand.Read(key[:])

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := Encrypt(plaintext, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecrypt benchmarks decryption performance
func BenchmarkDecrypt(b *testing.B) {
	plaintext := make([]byte, 1024) // 1KB
	rand.Read(plaintext)
	var key [32]byte
	rand.Read(key[:])

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := Decrypt(encrypted, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncryptLarge benchmarks encryption of large data (12KB keystore)
func BenchmarkEncryptLarge(b *testing.B) {
	plaintext := make([]byte, 12000) // ~12KB (typical keystore size)
	rand.Read(plaintext)
	var key [32]byte
	rand.Read(key[:])

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := Encrypt(plaintext, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecryptLarge benchmarks decryption of large data (12KB keystore)
func BenchmarkDecryptLarge(b *testing.B) {
	plaintext := make([]byte, 12000) // ~12KB (typical keystore size)
	rand.Read(plaintext)
	var key [32]byte
	rand.Read(key[:])

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := Decrypt(encrypted, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}
