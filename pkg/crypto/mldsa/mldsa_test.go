package mldsa

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// TestMLDSAKeypairGeneration tests ML-DSA-87 keypair generation
func TestMLDSAKeypairGeneration(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() failed: %v", err)
	}

	if len(kp.PublicKey) != PublicKeySize {
		t.Errorf("Public key size mismatch: expected %d, got %d", PublicKeySize, len(kp.PublicKey))
	}

	if len(kp.PrivateKey) != PrivateKeySize {
		t.Errorf("Private key size mismatch: expected %d, got %d", PrivateKeySize, len(kp.PrivateKey))
	}
}

// TestMLDSASignatureRoundTrip tests signing and verification
func TestMLDSASignatureRoundTrip(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() failed: %v", err)
	}

	message := []byte("test message for ML-DSA-87 signature")

	signature, err := Sign(message, kp.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	if len(signature) != SignatureSize {
		t.Errorf("Signature size mismatch: expected %d, got %d", SignatureSize, len(signature))
	}

	if !Verify(message, signature, kp.PublicKey) {
		t.Error("Verify() failed for valid signature")
	}
}

// TestMLDSAInvalidSignature tests verification with invalid signatures
func TestMLDSAInvalidSignature(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() failed: %v", err)
	}

	message := []byte("test message")
	signature, err := Sign(message, kp.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	// Tamper with signature
	tamperedSig := make([]byte, len(signature))
	copy(tamperedSig, signature)
	tamperedSig[0] ^= 0xFF

	if Verify(message, tamperedSig, kp.PublicKey) {
		t.Error("Verify() succeeded for tampered signature")
	}
}

// TestMLDSATamperedMessage tests verification with tampered message
func TestMLDSATamperedMessage(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() failed: %v", err)
	}

	message := []byte("original message")
	signature, err := Sign(message, kp.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	tamperedMessage := []byte("tampered message")
	if Verify(tamperedMessage, signature, kp.PublicKey) {
		t.Error("Verify() succeeded for tampered message")
	}
}

// TestMLDSAWrongPublicKey tests verification with wrong public key
func TestMLDSAWrongPublicKey(t *testing.T) {
	kp1, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() #1 failed: %v", err)
	}

	kp2, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() #2 failed: %v", err)
	}

	message := []byte("test message")
	signature, err := Sign(message, kp1.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	if Verify(message, signature, kp2.PublicKey) {
		t.Error("Verify() succeeded with wrong public key")
	}
}

// TestMLDSAErrorCases tests error handling
func TestMLDSAErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		testFunc  func() error
		expectErr bool
	}{
		{
			name: "Sign with invalid private key size",
			testFunc: func() error {
				_, err := Sign([]byte("test"), make([]byte, 10))
				return err
			},
			expectErr: true,
		},
		{
			name: "Verify with invalid public key size",
			testFunc: func() error {
				valid := Verify([]byte("test"), make([]byte, SignatureSize), make([]byte, 10))
				if valid {
					return ErrInvalidPublicKey
				}
				return nil
			},
			expectErr: false,
		},
		{
			name: "Verify with invalid signature size",
			testFunc: func() error {
				valid := Verify([]byte("test"), make([]byte, 10), make([]byte, PublicKeySize))
				if valid {
					return ErrInvalidSignature
				}
				return nil
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

// TestMLDSAKeypairUniqueness tests that generated keypairs are unique
func TestMLDSAKeypairUniqueness(t *testing.T) {
	kp1, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() #1 failed: %v", err)
	}

	kp2, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() #2 failed: %v", err)
	}

	if bytes.Equal(kp1.PublicKey, kp2.PublicKey) {
		t.Error("Two keypairs have identical public keys (entropy failure)")
	}

	if bytes.Equal(kp1.PrivateKey, kp2.PrivateKey) {
		t.Error("Two keypairs have identical private keys (entropy failure)")
	}
}

// TestMLDSAScheme tests the Scheme() helper function
func TestMLDSAScheme(t *testing.T) {
	scheme := Scheme()

	if scheme.Name() != "Dilithium5" {
		t.Errorf("Scheme name mismatch: expected Dilithium5, got %s", scheme.Name())
	}

	if scheme.PublicKeySize() != PublicKeySize {
		t.Errorf("Public key size mismatch: expected %d, got %d", PublicKeySize, scheme.PublicKeySize())
	}

	if scheme.PrivateKeySize() != PrivateKeySize {
		t.Errorf("Private key size mismatch: expected %d, got %d", PrivateKeySize, scheme.PrivateKeySize())
	}

	if scheme.SignatureSize() != SignatureSize {
		t.Errorf("Signature size mismatch: expected %d, got %d", SignatureSize, scheme.SignatureSize())
	}
}

// BenchmarkMLDSAKeypairGeneration benchmarks ML-DSA-87 keypair generation
func BenchmarkMLDSAKeypairGeneration(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := GenerateKeypair()
		if err != nil {
			b.Fatalf("GenerateKeypair() failed: %v", err)
		}
	}
}

// BenchmarkMLDSASign benchmarks ML-DSA-87 signing
func BenchmarkMLDSASign(b *testing.B) {
	kp, err := GenerateKeypair()
	if err != nil {
		b.Fatalf("GenerateKeypair() failed: %v", err)
	}

	message := make([]byte, 1024)
	rand.Read(message)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Sign(message, kp.PrivateKey)
		if err != nil {
			b.Fatalf("Sign() failed: %v", err)
		}
	}
}

// BenchmarkMLDSAVerify benchmarks ML-DSA-87 verification
func BenchmarkMLDSAVerify(b *testing.B) {
	kp, err := GenerateKeypair()
	if err != nil {
		b.Fatalf("GenerateKeypair() failed: %v", err)
	}

	message := make([]byte, 1024)
	rand.Read(message)

	signature, err := Sign(message, kp.PrivateKey)
	if err != nil {
		b.Fatalf("Sign() failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !Verify(message, signature, kp.PublicKey) {
			b.Fatal("Verify() failed")
		}
	}
}

// BenchmarkMLDSARoundTrip benchmarks full ML-DSA-87 sign + verify
func BenchmarkMLDSARoundTrip(b *testing.B) {
	kp, err := GenerateKeypair()
	if err != nil {
		b.Fatalf("GenerateKeypair() failed: %v", err)
	}

	message := make([]byte, 1024)
	rand.Read(message)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sig, err := Sign(message, kp.PrivateKey)
		if err != nil {
			b.Fatalf("Sign() failed: %v", err)
		}

		if !Verify(message, sig, kp.PublicKey) {
			b.Fatal("Verify() failed")
		}
	}
}
