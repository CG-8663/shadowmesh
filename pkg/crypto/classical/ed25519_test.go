package classical

import (
	"bytes"
	"crypto/rand"
	"testing"
)

// TestEd25519KeypairGeneration tests Ed25519 keypair generation
func TestEd25519KeypairGeneration(t *testing.T) {
	kp, err := GenerateEd25519Keypair()
	if err != nil {
		t.Fatalf("GenerateEd25519Keypair() failed: %v", err)
	}

	if len(kp.PublicKey) != Ed25519PublicKeySize {
		t.Errorf("Public key size mismatch: expected %d, got %d", Ed25519PublicKeySize, len(kp.PublicKey))
	}

	if len(kp.PrivateKey) != Ed25519PrivateKeySize {
		t.Errorf("Private key size mismatch: expected %d, got %d", Ed25519PrivateKeySize, len(kp.PrivateKey))
	}
}

// TestEd25519SignatureRoundTrip tests signing and verification
func TestEd25519SignatureRoundTrip(t *testing.T) {
	kp, err := GenerateEd25519Keypair()
	if err != nil {
		t.Fatalf("GenerateEd25519Keypair() failed: %v", err)
	}

	message := []byte("test message for Ed25519 signature")

	signature, err := Ed25519Sign(message, kp.PrivateKey)
	if err != nil {
		t.Fatalf("Ed25519Sign() failed: %v", err)
	}

	if len(signature) != Ed25519SignatureSize {
		t.Errorf("Signature size mismatch: expected %d, got %d", Ed25519SignatureSize, len(signature))
	}

	if !Ed25519Verify(message, signature, kp.PublicKey) {
		t.Error("Ed25519Verify() failed for valid signature")
	}
}

// TestEd25519InvalidSignature tests verification with invalid signatures
func TestEd25519InvalidSignature(t *testing.T) {
	kp, err := GenerateEd25519Keypair()
	if err != nil {
		t.Fatalf("GenerateEd25519Keypair() failed: %v", err)
	}

	message := []byte("test message")
	signature, err := Ed25519Sign(message, kp.PrivateKey)
	if err != nil {
		t.Fatalf("Ed25519Sign() failed: %v", err)
	}

	// Tamper with signature
	tamperedSig := make([]byte, len(signature))
	copy(tamperedSig, signature)
	tamperedSig[0] ^= 0xFF

	if Ed25519Verify(message, tamperedSig, kp.PublicKey) {
		t.Error("Ed25519Verify() succeeded for tampered signature")
	}
}

// TestEd25519TamperedMessage tests verification with tampered message
func TestEd25519TamperedMessage(t *testing.T) {
	kp, err := GenerateEd25519Keypair()
	if err != nil {
		t.Fatalf("GenerateEd25519Keypair() failed: %v", err)
	}

	message := []byte("original message")
	signature, err := Ed25519Sign(message, kp.PrivateKey)
	if err != nil {
		t.Fatalf("Ed25519Sign() failed: %v", err)
	}

	tamperedMessage := []byte("tampered message")
	if Ed25519Verify(tamperedMessage, signature, kp.PublicKey) {
		t.Error("Ed25519Verify() succeeded for tampered message")
	}
}

// TestEd25519WrongPublicKey tests verification with wrong public key
func TestEd25519WrongPublicKey(t *testing.T) {
	kp1, err := GenerateEd25519Keypair()
	if err != nil {
		t.Fatalf("GenerateEd25519Keypair() #1 failed: %v", err)
	}

	kp2, err := GenerateEd25519Keypair()
	if err != nil {
		t.Fatalf("GenerateEd25519Keypair() #2 failed: %v", err)
	}

	message := []byte("test message")
	signature, err := Ed25519Sign(message, kp1.PrivateKey)
	if err != nil {
		t.Fatalf("Ed25519Sign() failed: %v", err)
	}

	if Ed25519Verify(message, signature, kp2.PublicKey) {
		t.Error("Ed25519Verify() succeeded with wrong public key")
	}
}

// TestEd25519ErrorCases tests error handling
func TestEd25519ErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		testFunc  func() error
		expectErr bool
	}{
		{
			name: "Sign with invalid private key size",
			testFunc: func() error {
				_, err := Ed25519Sign([]byte("test"), make([]byte, 10))
				return err
			},
			expectErr: true,
		},
		{
			name: "Verify with invalid public key size",
			testFunc: func() error {
				valid := Ed25519Verify([]byte("test"), make([]byte, Ed25519SignatureSize), make([]byte, 10))
				if valid {
					return ErrEd25519InvalidPublicKey
				}
				return nil
			},
			expectErr: false,
		},
		{
			name: "Verify with invalid signature size",
			testFunc: func() error {
				valid := Ed25519Verify([]byte("test"), make([]byte, 10), make([]byte, Ed25519PublicKeySize))
				if valid {
					return ErrEd25519SigningFailed
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

// TestEd25519KeypairUniqueness tests that generated keypairs are unique
func TestEd25519KeypairUniqueness(t *testing.T) {
	kp1, err := GenerateEd25519Keypair()
	if err != nil {
		t.Fatalf("GenerateEd25519Keypair() #1 failed: %v", err)
	}

	kp2, err := GenerateEd25519Keypair()
	if err != nil {
		t.Fatalf("GenerateEd25519Keypair() #2 failed: %v", err)
	}

	if bytes.Equal(kp1.PublicKey, kp2.PublicKey) {
		t.Error("Two keypairs have identical public keys (entropy failure)")
	}

	if bytes.Equal(kp1.PrivateKey, kp2.PrivateKey) {
		t.Error("Two keypairs have identical private keys (entropy failure)")
	}
}

// TestEd25519RFC8032Compliance tests Ed25519 properties from RFC 8032
func TestEd25519RFC8032Compliance(t *testing.T) {
	// Verify key sizes match RFC 8032 specification
	if Ed25519PublicKeySize != 32 {
		t.Errorf("RFC 8032 specifies 32-byte public keys, got %d", Ed25519PublicKeySize)
	}

	if Ed25519PrivateKeySize != 64 {
		t.Errorf("RFC 8032 specifies 64-byte private keys, got %d", Ed25519PrivateKeySize)
	}

	if Ed25519SignatureSize != 64 {
		t.Errorf("RFC 8032 specifies 64-byte signatures, got %d", Ed25519SignatureSize)
	}
}

// BenchmarkEd25519KeypairGeneration benchmarks Ed25519 keypair generation
func BenchmarkEd25519KeypairGeneration(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := GenerateEd25519Keypair()
		if err != nil {
			b.Fatalf("GenerateEd25519Keypair() failed: %v", err)
		}
	}
}

// BenchmarkEd25519Sign benchmarks Ed25519 signing
func BenchmarkEd25519Sign(b *testing.B) {
	kp, err := GenerateEd25519Keypair()
	if err != nil {
		b.Fatalf("GenerateEd25519Keypair() failed: %v", err)
	}

	message := make([]byte, 1024)
	rand.Read(message)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Ed25519Sign(message, kp.PrivateKey)
		if err != nil {
			b.Fatalf("Ed25519Sign() failed: %v", err)
		}
	}
}

// BenchmarkEd25519Verify benchmarks Ed25519 verification
func BenchmarkEd25519Verify(b *testing.B) {
	kp, err := GenerateEd25519Keypair()
	if err != nil {
		b.Fatalf("GenerateEd25519Keypair() failed: %v", err)
	}

	message := make([]byte, 1024)
	rand.Read(message)

	signature, err := Ed25519Sign(message, kp.PrivateKey)
	if err != nil {
		b.Fatalf("Ed25519Sign() failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !Ed25519Verify(message, signature, kp.PublicKey) {
			b.Fatal("Ed25519Verify() failed")
		}
	}
}

// BenchmarkEd25519RoundTrip benchmarks full Ed25519 sign + verify
func BenchmarkEd25519RoundTrip(b *testing.B) {
	kp, err := GenerateEd25519Keypair()
	if err != nil {
		b.Fatalf("GenerateEd25519Keypair() failed: %v", err)
	}

	message := make([]byte, 1024)
	rand.Read(message)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sig, err := Ed25519Sign(message, kp.PrivateKey)
		if err != nil {
			b.Fatalf("Ed25519Sign() failed: %v", err)
		}

		if !Ed25519Verify(message, sig, kp.PublicKey) {
			b.Fatal("Ed25519Verify() failed")
		}
	}
}
