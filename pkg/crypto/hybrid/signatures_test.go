package hybrid

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/shadowmesh/shadowmesh/pkg/crypto/classical"
	"github.com/shadowmesh/shadowmesh/pkg/crypto/mldsa"
)

// TestHybridSignatureRoundTrip tests basic hybrid signature creation and verification
func TestHybridSignatureRoundTrip(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	message := []byte("test message for hybrid signature")

	signature, err := HybridSign(message, kp)
	if err != nil {
		t.Fatalf("HybridSign() failed: %v", err)
	}

	if len(signature) != HybridSignatureSize {
		t.Errorf("Signature size mismatch: expected %d, got %d", HybridSignatureSize, len(signature))
	}

	// Verify signature size is exactly 4659 bytes (AC 1.3.3)
	// Note: circl implements Dilithium Round 3 with 4595-byte signatures
	expectedSize := 4659
	if len(signature) != expectedSize {
		t.Errorf("Hybrid signature must be %d bytes (4595 ML-DSA + 64 Ed25519), got %d", expectedSize, len(signature))
	}

	if !HybridVerify(message, signature, kp) {
		t.Error("HybridVerify() failed for valid signature")
	}
}

// TestHybridSignatureFormat validates the signature format (ML-DSA || Ed25519)
func TestHybridSignatureFormat(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	message := []byte("format test message")

	signature, err := HybridSign(message, kp)
	if err != nil {
		t.Fatalf("HybridSign() failed: %v", err)
	}

	// Verify total size
	if len(signature) != HybridSignatureSize {
		t.Errorf("Expected %d bytes, got %d", HybridSignatureSize, len(signature))
	}

	// Verify can split into ML-DSA and Ed25519 components
	mldsaSig := signature[:mldsa.SignatureSize]
	ed25519Sig := signature[mldsa.SignatureSize:]

	if len(mldsaSig) != mldsa.SignatureSize {
		t.Errorf("ML-DSA signature size mismatch: expected %d, got %d", mldsa.SignatureSize, len(mldsaSig))
	}

	if len(ed25519Sig) != classical.Ed25519SignatureSize {
		t.Errorf("Ed25519 signature size mismatch: expected %d, got %d", classical.Ed25519SignatureSize, len(ed25519Sig))
	}

	// Verify each component independently
	if !mldsa.Verify(message, mldsaSig, kp.MLDSAPublicKey) {
		t.Error("ML-DSA component verification failed")
	}

	if !classical.Ed25519Verify(message, ed25519Sig, kp.Ed25519PublicKey) {
		t.Error("Ed25519 component verification failed")
	}
}

// TestHybridSignatureTamperedMLDSA tests that tampering ML-DSA signature fails verification
func TestHybridSignatureTamperedMLDSA(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	message := []byte("test message")
	signature, err := HybridSign(message, kp)
	if err != nil {
		t.Fatalf("HybridSign() failed: %v", err)
	}

	// Tamper with ML-DSA portion (first byte)
	tamperedSig := make([]byte, len(signature))
	copy(tamperedSig, signature)
	tamperedSig[0] ^= 0xFF

	// Verification must fail (AC 1.3.4: both must be valid)
	if HybridVerify(message, tamperedSig, kp) {
		t.Error("HybridVerify() succeeded with tampered ML-DSA signature")
	}
}

// TestHybridSignatureTamperedEd25519 tests that tampering Ed25519 signature fails verification
func TestHybridSignatureTamperedEd25519(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	message := []byte("test message")
	signature, err := HybridSign(message, kp)
	if err != nil {
		t.Fatalf("HybridSign() failed: %v", err)
	}

	// Tamper with Ed25519 portion (last byte)
	tamperedSig := make([]byte, len(signature))
	copy(tamperedSig, signature)
	tamperedSig[len(tamperedSig)-1] ^= 0xFF

	// Verification must fail (AC 1.3.4: both must be valid)
	if HybridVerify(message, tamperedSig, kp) {
		t.Error("HybridVerify() succeeded with tampered Ed25519 signature")
	}
}

// TestHybridSignatureTamperedMessage tests verification with tampered message
func TestHybridSignatureTamperedMessage(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	message := []byte("original message")
	signature, err := HybridSign(message, kp)
	if err != nil {
		t.Fatalf("HybridSign() failed: %v", err)
	}

	tamperedMessage := []byte("tampered message")
	if HybridVerify(tamperedMessage, signature, kp) {
		t.Error("HybridVerify() succeeded for tampered message")
	}
}

// TestHybridSignatureWrongPublicKey tests verification with wrong public key
func TestHybridSignatureWrongPublicKey(t *testing.T) {
	kp1, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() #1 failed: %v", err)
	}

	kp2, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() #2 failed: %v", err)
	}

	message := []byte("test message")
	signature, err := HybridSign(message, kp1)
	if err != nil {
		t.Fatalf("HybridSign() failed: %v", err)
	}

	if HybridVerify(message, signature, kp2) {
		t.Error("HybridVerify() succeeded with wrong public key")
	}
}

// TestHybridSignatureErrorCases tests error handling
func TestHybridSignatureErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		testFunc  func() error
		expectErr bool
	}{
		{
			name: "Sign with nil keypair",
			testFunc: func() error {
				_, err := HybridSign([]byte("test"), nil)
				return err
			},
			expectErr: true,
		},
		{
			name: "Sign with missing ML-DSA private key",
			testFunc: func() error {
				kp := &HybridKeypair{
					Ed25519PrivateKey: make([]byte, classical.Ed25519PrivateKeySize),
				}
				_, err := HybridSign([]byte("test"), kp)
				return err
			},
			expectErr: true,
		},
		{
			name: "Sign with missing Ed25519 private key",
			testFunc: func() error {
				kp := &HybridKeypair{
					MLDSAPrivateKey: make([]byte, mldsa.PrivateKeySize),
				}
				_, err := HybridSign([]byte("test"), kp)
				return err
			},
			expectErr: true,
		},
		{
			name: "Verify with nil keypair",
			testFunc: func() error {
				valid := HybridVerify([]byte("test"), make([]byte, HybridSignatureSize), nil)
				if valid {
					return ErrVerificationFailed
				}
				return nil
			},
			expectErr: false,
		},
		{
			name: "Verify with invalid signature size",
			testFunc: func() error {
				kp, _ := GenerateHybridKeypair()
				valid := HybridVerify([]byte("test"), make([]byte, 100), kp)
				if valid {
					return ErrInvalidSignature
				}
				return nil
			},
			expectErr: false,
		},
		{
			name: "Verify with missing ML-DSA public key",
			testFunc: func() error {
				kp := &HybridKeypair{
					Ed25519PublicKey: make([]byte, classical.Ed25519PublicKeySize),
				}
				valid := HybridVerify([]byte("test"), make([]byte, HybridSignatureSize), kp)
				if valid {
					return ErrVerificationFailed
				}
				return nil
			},
			expectErr: false,
		},
		{
			name: "Verify with missing Ed25519 public key",
			testFunc: func() error {
				kp := &HybridKeypair{
					MLDSAPublicKey: make([]byte, mldsa.PublicKeySize),
				}
				valid := HybridVerify([]byte("test"), make([]byte, HybridSignatureSize), kp)
				if valid {
					return ErrVerificationFailed
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

// TestHybridSignatureIndependentVerification tests that both signatures are verified independently
func TestHybridSignatureIndependentVerification(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	message := []byte("independent verification test")
	signature, err := HybridSign(message, kp)
	if err != nil {
		t.Fatalf("HybridSign() failed: %v", err)
	}

	// Test 1: Replace ML-DSA signature with random bytes, keep Ed25519 valid
	tamperedSig1 := make([]byte, len(signature))
	copy(tamperedSig1, signature)
	rand.Read(tamperedSig1[:mldsa.SignatureSize]) // Randomize ML-DSA portion

	if HybridVerify(message, tamperedSig1, kp) {
		t.Error("Verification succeeded with invalid ML-DSA but valid Ed25519")
	}

	// Test 2: Keep ML-DSA valid, replace Ed25519 with random bytes
	tamperedSig2 := make([]byte, len(signature))
	copy(tamperedSig2, signature)
	rand.Read(tamperedSig2[mldsa.SignatureSize:]) // Randomize Ed25519 portion

	if HybridVerify(message, tamperedSig2, kp) {
		t.Error("Verification succeeded with valid ML-DSA but invalid Ed25519")
	}
}

// TestHybridSignatureKeypairUniqueness tests that signatures from different keypairs differ
func TestHybridSignatureKeypairUniqueness(t *testing.T) {
	kp1, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() #1 failed: %v", err)
	}

	kp2, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() #2 failed: %v", err)
	}

	message := []byte("uniqueness test")

	sig1, err := HybridSign(message, kp1)
	if err != nil {
		t.Fatalf("HybridSign() #1 failed: %v", err)
	}

	sig2, err := HybridSign(message, kp2)
	if err != nil {
		t.Fatalf("HybridSign() #2 failed: %v", err)
	}

	if bytes.Equal(sig1, sig2) {
		t.Error("Two signatures from different keypairs are identical")
	}
}

// BenchmarkHybridSign benchmarks hybrid signature generation
func BenchmarkHybridSign(b *testing.B) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		b.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	message := make([]byte, 1024)
	rand.Read(message)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := HybridSign(message, kp)
		if err != nil {
			b.Fatalf("HybridSign() failed: %v", err)
		}
	}
}

// BenchmarkHybridVerify benchmarks hybrid signature verification
func BenchmarkHybridVerify(b *testing.B) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		b.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	message := make([]byte, 1024)
	rand.Read(message)

	signature, err := HybridSign(message, kp)
	if err != nil {
		b.Fatalf("HybridSign() failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !HybridVerify(message, signature, kp) {
			b.Fatal("HybridVerify() failed")
		}
	}
}

// BenchmarkHybridRoundTrip benchmarks full hybrid sign + verify
func BenchmarkHybridRoundTrip(b *testing.B) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		b.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	message := make([]byte, 1024)
	rand.Read(message)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sig, err := HybridSign(message, kp)
		if err != nil {
			b.Fatalf("HybridSign() failed: %v", err)
		}

		if !HybridVerify(message, sig, kp) {
			b.Fatal("HybridVerify() failed")
		}
	}
}

// BenchmarkGenerateHybridKeypair benchmarks full keypair generation (KEX + signatures)
func BenchmarkGenerateHybridKeypair(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := GenerateHybridKeypair()
		if err != nil {
			b.Fatalf("GenerateHybridKeypair() failed: %v", err)
		}
	}
}
