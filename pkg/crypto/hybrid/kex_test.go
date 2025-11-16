package hybrid

import (
	"bytes"
	"testing"
	"time"
)

// TestHybridKeypairGeneration tests hybrid keypair generation
func TestHybridKeypairGeneration(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	// Verify ML-KEM key sizes
	if len(kp.MLKEMPublicKey) != 1568 {
		t.Errorf("ML-KEM public key size mismatch: expected 1568, got %d", len(kp.MLKEMPublicKey))
	}
	if len(kp.MLKEMPrivateKey) != 3168 {
		t.Errorf("ML-KEM private key size mismatch: expected 3168, got %d", len(kp.MLKEMPrivateKey))
	}

	// Verify X25519 key sizes
	if len(kp.X25519PublicKey) != 32 {
		t.Errorf("X25519 public key size mismatch: expected 32, got %d", len(kp.X25519PublicKey))
	}
	if len(kp.X25519PrivateKey) != 32 {
		t.Errorf("X25519 private key size mismatch: expected 32, got %d", len(kp.X25519PrivateKey))
	}

	// Verify metadata
	if kp.CreatedAt.IsZero() {
		t.Error("CreatedAt timestamp is zero")
	}
	if kp.ExpiresAt.IsZero() {
		t.Error("ExpiresAt timestamp is zero")
	}
	if !kp.ExpiresAt.After(kp.CreatedAt) {
		t.Error("ExpiresAt should be after CreatedAt")
	}
}

// TestHybridEncapsulationDecapsulation tests hybrid encapsulation/decapsulation round-trip
func TestHybridEncapsulationDecapsulation(t *testing.T) {
	// Generate keypair
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	// Encapsulate
	ct, ss1, err := HybridEncapsulate(kp)
	if err != nil {
		t.Fatalf("HybridEncapsulate() failed: %v", err)
	}

	// Verify ciphertext size (1568 + 32 = 1600 bytes)
	expectedCTSize := 1568 + 32
	if len(ct) != expectedCTSize {
		t.Errorf("Ciphertext size mismatch: expected %d, got %d", expectedCTSize, len(ct))
	}

	// Verify shared secret size (32 bytes)
	if len(ss1) != 32 {
		t.Errorf("Shared secret size mismatch: expected 32, got %d", len(ss1))
	}

	// Decapsulate
	ss2, err := HybridDecapsulate(ct, kp)
	if err != nil {
		t.Fatalf("HybridDecapsulate() failed: %v", err)
	}

	// Verify shared secrets match
	if !bytes.Equal(ss1, ss2) {
		t.Error("Shared secrets do not match after round-trip")
	}
}

// TestHybridMultipleRoundTrips tests multiple hybrid encapsulation/decapsulation cycles
func TestHybridMultipleRoundTrips(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	// Perform 10 round-trips
	for i := 0; i < 10; i++ {
		ct, ss1, err := HybridEncapsulate(kp)
		if err != nil {
			t.Fatalf("Round-trip %d: HybridEncapsulate() failed: %v", i, err)
		}

		ss2, err := HybridDecapsulate(ct, kp)
		if err != nil {
			t.Fatalf("Round-trip %d: HybridDecapsulate() failed: %v", i, err)
		}

		if !bytes.Equal(ss1, ss2) {
			t.Errorf("Round-trip %d: Shared secrets do not match", i)
		}
	}
}

// TestHybridKDF tests HKDF-SHA256 combination of secrets
func TestHybridKDF(t *testing.T) {
	// Test with known inputs
	kemSecret := make([]byte, 32)
	ecdhSecret := make([]byte, 32)

	// Fill with test data
	for i := range kemSecret {
		kemSecret[i] = byte(i)
		ecdhSecret[i] = byte(i + 32)
	}

	// Derive shared secret
	ss1, err := deriveSharedSecret(kemSecret, ecdhSecret)
	if err != nil {
		t.Fatalf("deriveSharedSecret() failed: %v", err)
	}

	// Verify output size
	if len(ss1) != 32 {
		t.Errorf("Derived secret size mismatch: expected 32, got %d", len(ss1))
	}

	// Derive again with same inputs - should be deterministic
	ss2, err := deriveSharedSecret(kemSecret, ecdhSecret)
	if err != nil {
		t.Fatalf("deriveSharedSecret() second call failed: %v", err)
	}

	if !bytes.Equal(ss1, ss2) {
		t.Error("HKDF is not deterministic - same inputs produced different outputs")
	}

	// Different inputs should produce different outputs
	kemSecret2 := make([]byte, 32)
	for i := range kemSecret2 {
		kemSecret2[i] = byte(i + 64)
	}

	ss3, err := deriveSharedSecret(kemSecret2, ecdhSecret)
	if err != nil {
		t.Fatalf("deriveSharedSecret() with different input failed: %v", err)
	}

	if bytes.Equal(ss1, ss3) {
		t.Error("HKDF produced same output for different inputs (collision)")
	}
}

// TestHybridInvalidCiphertext tests error handling for invalid ciphertext
func TestHybridInvalidCiphertext(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	testCases := []struct {
		name       string
		ciphertext []byte
		wantErr    error
	}{
		{
			name:       "nil ciphertext",
			ciphertext: nil,
			wantErr:    ErrInvalidCiphertext,
		},
		{
			name:       "empty ciphertext",
			ciphertext: []byte{},
			wantErr:    ErrInvalidCiphertext,
		},
		{
			name:       "too short ciphertext",
			ciphertext: make([]byte, 100),
			wantErr:    ErrInvalidCiphertext,
		},
		{
			name:       "too long ciphertext",
			ciphertext: make([]byte, 2000),
			wantErr:    ErrInvalidCiphertext,
		},
		{
			name:       "ciphertext missing ECDH public key",
			ciphertext: make([]byte, 1568), // Only KEM CT, missing ECDH
			wantErr:    ErrInvalidCiphertext,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := HybridDecapsulate(tc.ciphertext, kp)
			if err == nil {
				t.Error("Expected error but got nil")
			}
			if !bytes.Contains([]byte(err.Error()), []byte(tc.wantErr.Error())) {
				t.Errorf("Expected error containing %q, got %q", tc.wantErr, err)
			}
		})
	}
}

// TestHybridNilKeys tests error handling for nil keys
func TestHybridNilKeys(t *testing.T) {
	// Test nil public key in encapsulation
	_, _, err := HybridEncapsulate(nil)
	if err == nil {
		t.Error("HybridEncapsulate() with nil key should fail")
	}
	if !bytes.Contains([]byte(err.Error()), []byte(ErrEncapsulationFailed.Error())) {
		t.Errorf("Expected error containing %q, got %q", ErrEncapsulationFailed, err)
	}

	// Test nil private key in decapsulation
	validCT := make([]byte, 1600)
	_, err = HybridDecapsulate(validCT, nil)
	if err == nil {
		t.Error("HybridDecapsulate() with nil key should fail")
	}
	if !bytes.Contains([]byte(err.Error()), []byte(ErrDecapsulationFailed.Error())) {
		t.Errorf("Expected error containing %q, got %q", ErrDecapsulationFailed, err)
	}
}

// TestHybridDifferentKeypairs tests that different keypairs produce different shared secrets
func TestHybridDifferentKeypairs(t *testing.T) {
	// Generate two keypairs
	kp1, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() #1 failed: %v", err)
	}

	kp2, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() #2 failed: %v", err)
	}

	// Encapsulate with first keypair
	ct1, ss1, err := HybridEncapsulate(kp1)
	if err != nil {
		t.Fatalf("HybridEncapsulate() with kp1 failed: %v", err)
	}

	// Attempt to decapsulate with wrong private key (should fail or produce wrong secret)
	ss2, err := HybridDecapsulate(ct1, kp2)
	if err == nil {
		// If no error, shared secrets should be different
		if bytes.Equal(ss1, ss2) {
			t.Error("Wrong private key produced same shared secret (security violation)")
		}
	}
	// Note: Implementation may return error or different secret
}

// TestHybridEncapsulationUniqueness tests that multiple encapsulations produce unique ciphertexts
func TestHybridEncapsulationUniqueness(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	// Perform 5 encapsulations
	ciphertexts := make([][]byte, 5)
	secrets := make([][]byte, 5)

	for i := 0; i < 5; i++ {
		ct, ss, err := HybridEncapsulate(kp)
		if err != nil {
			t.Fatalf("Encapsulation %d failed: %v", i, err)
		}
		ciphertexts[i] = ct
		secrets[i] = ss
	}

	// Verify all ciphertexts are unique (due to ephemeral X25519 keys)
	for i := 0; i < 5; i++ {
		for j := i + 1; j < 5; j++ {
			if bytes.Equal(ciphertexts[i], ciphertexts[j]) {
				t.Errorf("Ciphertexts %d and %d are identical (should be unique)", i, j)
			}
		}
	}

	// Verify all secrets are unique (due to different ephemeral ECDH)
	for i := 0; i < 5; i++ {
		for j := i + 1; j < 5; j++ {
			if bytes.Equal(secrets[i], secrets[j]) {
				t.Errorf("Secrets %d and %d are identical (should be unique due to ephemeral keys)", i, j)
			}
		}
	}
}

// BenchmarkHybridKeypairGen benchmarks hybrid keypair generation
// Target: <100ms on commodity hardware
func BenchmarkHybridKeypairGen(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := GenerateHybridKeypair()
		if err != nil {
			b.Fatalf("GenerateHybridKeypair() failed: %v", err)
		}
	}
}

// BenchmarkHybridEncapsulate benchmarks hybrid encapsulation
func BenchmarkHybridEncapsulate(b *testing.B) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		b.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, err := HybridEncapsulate(kp)
		if err != nil {
			b.Fatalf("HybridEncapsulate() failed: %v", err)
		}
	}
}

// BenchmarkHybridDecapsulate benchmarks hybrid decapsulation
func BenchmarkHybridDecapsulate(b *testing.B) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		b.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	ct, _, err := HybridEncapsulate(kp)
	if err != nil {
		b.Fatalf("HybridEncapsulate() failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := HybridDecapsulate(ct, kp)
		if err != nil {
			b.Fatalf("HybridDecapsulate() failed: %v", err)
		}
	}
}

// BenchmarkHybridKEX benchmarks full hybrid key exchange (encapsulation + decapsulation)
// Target: <50ms roundtrip on commodity hardware
func BenchmarkHybridKEX(b *testing.B) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		b.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ct, ss1, err := HybridEncapsulate(kp)
		if err != nil {
			b.Fatalf("HybridEncapsulate() failed: %v", err)
		}

		ss2, err := HybridDecapsulate(ct, kp)
		if err != nil {
			b.Fatalf("HybridDecapsulate() failed: %v", err)
		}

		if !bytes.Equal(ss1, ss2) {
			b.Fatal("Shared secrets do not match")
		}
	}
}

// TestHybridDeriveSharedSecretEdgeCases tests edge cases for HKDF derivation
func TestHybridDeriveSharedSecretEdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		kemSecret   []byte
		ecdhSecret  []byte
		expectError bool
	}{
		{
			name:        "normal case",
			kemSecret:   make([]byte, 32),
			ecdhSecret:  make([]byte, 32),
			expectError: false,
		},
		{
			name:        "all zero secrets",
			kemSecret:   make([]byte, 32),
			ecdhSecret:  make([]byte, 32),
			expectError: false, // Valid, though not recommended
		},
		{
			name:        "different size secrets",
			kemSecret:   make([]byte, 32),
			ecdhSecret:  make([]byte, 16),
			expectError: false, // HKDF supports variable length
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Fill with test data if not all zeros
			for i := range tc.kemSecret {
				tc.kemSecret[i] = byte(i)
			}
			for i := range tc.ecdhSecret {
				tc.ecdhSecret[i] = byte(i + 32)
			}

			ss, err := deriveSharedSecret(tc.kemSecret, tc.ecdhSecret)
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tc.expectError && len(ss) != SharedSecretSize {
				t.Errorf("Shared secret size mismatch: expected %d, got %d", SharedSecretSize, len(ss))
			}
		})
	}
}

// TestHybridKeypairExpiration tests CreatedAt and ExpiresAt metadata
func TestHybridKeypairExpiration(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	// Verify timestamps are set
	if kp.CreatedAt.IsZero() {
		t.Error("CreatedAt timestamp is zero")
	}
	if kp.ExpiresAt.IsZero() {
		t.Error("ExpiresAt timestamp is zero")
	}

	// Verify expiration is after creation
	if !kp.ExpiresAt.After(kp.CreatedAt) {
		t.Error("ExpiresAt should be after CreatedAt")
	}

	// Verify expiration is approximately 5 minutes after creation
	expectedDuration := 5 * time.Minute
	actualDuration := kp.ExpiresAt.Sub(kp.CreatedAt)
	if actualDuration < expectedDuration-time.Second || actualDuration > expectedDuration+time.Second {
		t.Errorf("Expiration duration mismatch: expected ~%v, got %v", expectedDuration, actualDuration)
	}
}

// TestHybridCorruptedCiphertext tests decapsulation with bit-flipped ciphertext
func TestHybridCorruptedCiphertext(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	ct, ss1, err := HybridEncapsulate(kp)
	if err != nil {
		t.Fatalf("HybridEncapsulate() failed: %v", err)
	}

	// Corrupt ciphertext by flipping a bit
	corruptedCT := make([]byte, len(ct))
	copy(corruptedCT, ct)
	corruptedCT[len(ct)/2] ^= 0x01

	// Decapsulation should either fail or produce different secret
	ss2, err := HybridDecapsulate(corruptedCT, kp)
	if err == nil {
		// If no error, secrets should be different (IND-CCA2 property)
		if bytes.Equal(ss1, ss2) {
			t.Error("Corrupted ciphertext produced same shared secret (IND-CCA2 violation)")
		}
	}
}

// TestHybridCiphertextFormat tests the ciphertext structure (KEM CT || ECDH ephemeral pubkey)
func TestHybridCiphertextFormat(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	ct, _, err := HybridEncapsulate(kp)
	if err != nil {
		t.Fatalf("HybridEncapsulate() failed: %v", err)
	}

	// Verify ciphertext size (1568 KEM CT + 32 ECDH pubkey = 1600)
	expectedSize := 1568 + 32
	if len(ct) != expectedSize {
		t.Errorf("Ciphertext size mismatch: expected %d, got %d", expectedSize, len(ct))
	}

	// Extract components
	kemCT := ct[:1568]
	ecdhPubKey := ct[1568:]

	// Verify components are not all zeros
	allZerosKEM := true
	for _, b := range kemCT {
		if b != 0 {
			allZerosKEM = false
			break
		}
	}
	if allZerosKEM {
		t.Error("KEM ciphertext is all zeros")
	}

	allZerosECDH := true
	for _, b := range ecdhPubKey {
		if b != 0 {
			allZerosECDH = false
			break
		}
	}
	if allZerosECDH {
		t.Error("ECDH ephemeral public key is all zeros")
	}
}
