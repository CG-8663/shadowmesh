package mlkem

import (
	"bytes"
	"testing"

	"github.com/cloudflare/circl/kem/kyber/kyber1024"
)

// TestMLKEMKeypairGeneration tests ML-KEM-1024 keypair generation
func TestMLKEMKeypairGeneration(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() failed: %v", err)
	}

	// Verify public key size (1568 bytes for ML-KEM-1024)
	expectedPKSize := kyber1024.Scheme().PublicKeySize()
	if len(kp.PublicKey) != expectedPKSize {
		t.Errorf("Public key size mismatch: expected %d, got %d", expectedPKSize, len(kp.PublicKey))
	}

	// Verify private key size (3168 bytes for ML-KEM-1024)
	expectedSKSize := kyber1024.Scheme().PrivateKeySize()
	if len(kp.PrivateKey) != expectedSKSize {
		t.Errorf("Private key size mismatch: expected %d, got %d", expectedSKSize, len(kp.PrivateKey))
	}

	// Verify keys are not all zeros (entropy check)
	allZerosPK := true
	for _, b := range kp.PublicKey {
		if b != 0 {
			allZerosPK = false
			break
		}
	}
	if allZerosPK {
		t.Error("Public key is all zeros - likely entropy failure")
	}

	allZerosSK := true
	for _, b := range kp.PrivateKey {
		if b != 0 {
			allZerosSK = false
			break
		}
	}
	if allZerosSK {
		t.Error("Private key is all zeros - likely entropy failure")
	}
}

// TestMLKEMEncapsulationDecapsulation tests round-trip encapsulation/decapsulation
func TestMLKEMEncapsulationDecapsulation(t *testing.T) {
	// Generate keypair
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() failed: %v", err)
	}

	// Encapsulate
	ct, ss1, err := Encapsulate(kp.PublicKey)
	if err != nil {
		t.Fatalf("Encapsulate() failed: %v", err)
	}

	// Verify ciphertext size (1568 bytes for ML-KEM-1024)
	expectedCTSize := kyber1024.Scheme().CiphertextSize()
	if len(ct) != expectedCTSize {
		t.Errorf("Ciphertext size mismatch: expected %d, got %d", expectedCTSize, len(ct))
	}

	// Verify shared secret size (32 bytes)
	if len(ss1) != 32 {
		t.Errorf("Shared secret size mismatch: expected 32, got %d", len(ss1))
	}

	// Decapsulate
	ss2, err := Decapsulate(ct, kp.PrivateKey)
	if err != nil {
		t.Fatalf("Decapsulate() failed: %v", err)
	}

	// Verify shared secrets match
	if !bytes.Equal(ss1, ss2) {
		t.Error("Shared secrets do not match after round-trip")
	}
}

// TestMLKEMMultipleRoundTrips tests multiple encapsulation/decapsulation cycles
func TestMLKEMMultipleRoundTrips(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() failed: %v", err)
	}

	// Perform 10 round-trips
	for i := 0; i < 10; i++ {
		ct, ss1, err := Encapsulate(kp.PublicKey)
		if err != nil {
			t.Fatalf("Round-trip %d: Encapsulate() failed: %v", i, err)
		}

		ss2, err := Decapsulate(ct, kp.PrivateKey)
		if err != nil {
			t.Fatalf("Round-trip %d: Decapsulate() failed: %v", i, err)
		}

		if !bytes.Equal(ss1, ss2) {
			t.Errorf("Round-trip %d: Shared secrets do not match", i)
		}
	}
}

// TestMLKEMInvalidCiphertext tests error handling for invalid ciphertext
func TestMLKEMInvalidCiphertext(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() failed: %v", err)
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
			ciphertext: make([]byte, 10),
			wantErr:    ErrInvalidCiphertext,
		},
		{
			name:       "too long ciphertext",
			ciphertext: make([]byte, 2000),
			wantErr:    ErrInvalidCiphertext,
		},
		{
			name:       "corrupted ciphertext (wrong size)",
			ciphertext: make([]byte, kyber1024.Scheme().CiphertextSize()-1),
			wantErr:    ErrInvalidCiphertext,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decapsulate(tc.ciphertext, kp.PrivateKey)
			if err == nil {
				t.Error("Expected error but got nil")
			}
			if !bytes.Contains([]byte(err.Error()), []byte(tc.wantErr.Error())) {
				t.Errorf("Expected error containing %q, got %q", tc.wantErr, err)
			}
		})
	}
}

// TestMLKEMInvalidPrivateKey tests error handling for invalid private keys
func TestMLKEMInvalidPrivateKey(t *testing.T) {
	// Generate valid ciphertext
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() failed: %v", err)
	}

	ct, _, err := Encapsulate(kp.PublicKey)
	if err != nil {
		t.Fatalf("Encapsulate() failed: %v", err)
	}

	testCases := []struct {
		name       string
		privateKey []byte
		wantErr    error
	}{
		{
			name:       "nil private key",
			privateKey: nil,
			wantErr:    ErrDecapsulationFailed,
		},
		{
			name:       "empty private key",
			privateKey: []byte{},
			wantErr:    ErrDecapsulationFailed,
		},
		{
			name:       "too short private key",
			privateKey: make([]byte, 10),
			wantErr:    ErrDecapsulationFailed,
		},
		{
			name:       "too long private key",
			privateKey: make([]byte, 4000),
			wantErr:    ErrDecapsulationFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decapsulate(ct, tc.privateKey)
			if err == nil {
				t.Error("Expected error but got nil")
			}
			if !bytes.Contains([]byte(err.Error()), []byte(tc.wantErr.Error())) {
				t.Errorf("Expected error containing %q, got %q", tc.wantErr, err)
			}
		})
	}
}

// TestMLKEMInvalidPublicKey tests error handling for invalid public keys during encapsulation
func TestMLKEMInvalidPublicKey(t *testing.T) {
	testCases := []struct {
		name      string
		publicKey []byte
		wantErr   error
	}{
		{
			name:      "nil public key",
			publicKey: nil,
			wantErr:   ErrInvalidCiphertext,
		},
		{
			name:      "empty public key",
			publicKey: []byte{},
			wantErr:   ErrInvalidCiphertext,
		},
		{
			name:      "too short public key",
			publicKey: make([]byte, 10),
			wantErr:   ErrInvalidCiphertext,
		},
		{
			name:      "too long public key",
			publicKey: make([]byte, 2000),
			wantErr:   ErrInvalidCiphertext,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Encapsulate(tc.publicKey)
			if err == nil {
				t.Error("Expected error but got nil")
			}
			if !bytes.Contains([]byte(err.Error()), []byte(tc.wantErr.Error())) {
				t.Errorf("Expected error containing %q, got %q", tc.wantErr, err)
			}
		})
	}
}

// TestMLKEMDifferentKeypairs tests that different keypairs produce different shared secrets
func TestMLKEMDifferentKeypairs(t *testing.T) {
	// Generate two different keypairs
	kp1, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() #1 failed: %v", err)
	}

	kp2, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() #2 failed: %v", err)
	}

	// Ensure keys are different
	if bytes.Equal(kp1.PublicKey, kp2.PublicKey) {
		t.Error("Two generated keypairs have identical public keys (extremely unlikely)")
	}

	// Encapsulate with first keypair
	ct1, ss1, err := Encapsulate(kp1.PublicKey)
	if err != nil {
		t.Fatalf("Encapsulate() with kp1 failed: %v", err)
	}

	// Attempt to decapsulate with wrong private key (should fail or produce wrong secret)
	ss2, err := Decapsulate(ct1, kp2.PrivateKey)
	if err == nil {
		// If no error, shared secrets should be different
		if bytes.Equal(ss1, ss2) {
			t.Error("Wrong private key produced same shared secret (security violation)")
		}
	}
	// Note: CIRCL may return error or different secret depending on implementation
}

// TestMLKEMScheme tests the Scheme() helper function
func TestMLKEMScheme(t *testing.T) {
	scheme := Scheme()

	if scheme.Name() != "Kyber1024" {
		t.Errorf("Scheme name mismatch: expected Kyber1024, got %s", scheme.Name())
	}

	if scheme.PublicKeySize() != 1568 {
		t.Errorf("Public key size mismatch: expected 1568, got %d", scheme.PublicKeySize())
	}

	if scheme.PrivateKeySize() != 3168 {
		t.Errorf("Private key size mismatch: expected 3168, got %d", scheme.PrivateKeySize())
	}

	if scheme.CiphertextSize() != 1568 {
		t.Errorf("Ciphertext size mismatch: expected 1568, got %d", scheme.CiphertextSize())
	}

	if scheme.SharedKeySize() != 32 {
		t.Errorf("Shared key size mismatch: expected 32, got %d", scheme.SharedKeySize())
	}
}

// BenchmarkMLKEMKeypairGeneration benchmarks ML-KEM-1024 keypair generation
func BenchmarkMLKEMKeypairGeneration(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := GenerateKeypair()
		if err != nil {
			b.Fatalf("GenerateKeypair() failed: %v", err)
		}
	}
}

// BenchmarkMLKEMEncapsulate benchmarks ML-KEM-1024 encapsulation
func BenchmarkMLKEMEncapsulate(b *testing.B) {
	kp, err := GenerateKeypair()
	if err != nil {
		b.Fatalf("GenerateKeypair() failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, err := Encapsulate(kp.PublicKey)
		if err != nil {
			b.Fatalf("Encapsulate() failed: %v", err)
		}
	}
}

// BenchmarkMLKEMDecapsulate benchmarks ML-KEM-1024 decapsulation
func BenchmarkMLKEMDecapsulate(b *testing.B) {
	kp, err := GenerateKeypair()
	if err != nil {
		b.Fatalf("GenerateKeypair() failed: %v", err)
	}

	ct, _, err := Encapsulate(kp.PublicKey)
	if err != nil {
		b.Fatalf("Encapsulate() failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Decapsulate(ct, kp.PrivateKey)
		if err != nil {
			b.Fatalf("Decapsulate() failed: %v", err)
		}
	}
}

// BenchmarkMLKEMRoundTrip benchmarks full ML-KEM-1024 encapsulation + decapsulation
func BenchmarkMLKEMRoundTrip(b *testing.B) {
	kp, err := GenerateKeypair()
	if err != nil {
		b.Fatalf("GenerateKeypair() failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ct, ss1, err := Encapsulate(kp.PublicKey)
		if err != nil {
			b.Fatalf("Encapsulate() failed: %v", err)
		}

		ss2, err := Decapsulate(ct, kp.PrivateKey)
		if err != nil {
			b.Fatalf("Decapsulate() failed: %v", err)
		}

		if !bytes.Equal(ss1, ss2) {
			b.Fatal("Shared secrets do not match")
		}
	}
}

// TestMLKEMCorruptedCiphertext tests decapsulation with bit-flipped ciphertext
func TestMLKEMCorruptedCiphertext(b *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		b.Fatalf("GenerateKeypair() failed: %v", err)
	}

	ct, ss1, err := Encapsulate(kp.PublicKey)
	if err != nil {
		b.Fatalf("Encapsulate() failed: %v", err)
	}

	// Corrupt ciphertext by flipping a bit in the middle
	corruptedCT := make([]byte, len(ct))
	copy(corruptedCT, ct)
	corruptedCT[len(ct)/2] ^= 0x01

	// Decapsulation should either fail or produce different secret
	ss2, err := Decapsulate(corruptedCT, kp.PrivateKey)
	if err == nil {
		// If no error, secrets should be different (IND-CCA2 property)
		if bytes.Equal(ss1, ss2) {
			b.Error("Corrupted ciphertext produced same shared secret (IND-CCA2 violation)")
		}
	}
}

// TestMLKEMKeypairUniqueness tests that generated keypairs are unique
func TestMLKEMKeypairUniqueness(b *testing.T) {
	kp1, err := GenerateKeypair()
	if err != nil {
		b.Fatalf("GenerateKeypair() #1 failed: %v", err)
	}

	kp2, err := GenerateKeypair()
	if err != nil {
		b.Fatalf("GenerateKeypair() #2 failed: %v", err)
	}

	// Public keys should be different
	if bytes.Equal(kp1.PublicKey, kp2.PublicKey) {
		b.Error("Two keypairs have identical public keys (entropy failure)")
	}

	// Private keys should be different
	if bytes.Equal(kp1.PrivateKey, kp2.PrivateKey) {
		b.Error("Two keypairs have identical private keys (entropy failure)")
	}
}
