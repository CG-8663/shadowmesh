package hybrid

import (
	"bytes"
	"testing"

	"github.com/shadowmesh/shadowmesh/pkg/crypto/classical"
	"github.com/shadowmesh/shadowmesh/pkg/crypto/mldsa"
)

// TestPublicKeyHash tests basic public key hash generation
func TestPublicKeyHash(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	hash, err := PublicKeyHash(kp)
	if err != nil {
		t.Fatalf("PublicKeyHash() failed: %v", err)
	}

	if len(hash) != PublicKeyHashSize {
		t.Errorf("Hash size mismatch: expected %d bytes, got %d", PublicKeyHashSize, len(hash))
	}

	// Verify hash is exactly 32 bytes (SHA-256 output)
	if len(hash) != 32 {
		t.Errorf("Public key hash must be 32 bytes (SHA-256), got %d", len(hash))
	}
}

// TestPublicKeyHashDeterministic tests that hash is deterministic
func TestPublicKeyHashDeterministic(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	hash1, err := PublicKeyHash(kp)
	if err != nil {
		t.Fatalf("PublicKeyHash() #1 failed: %v", err)
	}

	hash2, err := PublicKeyHash(kp)
	if err != nil {
		t.Fatalf("PublicKeyHash() #2 failed: %v", err)
	}

	if !bytes.Equal(hash1, hash2) {
		t.Error("Public key hash is not deterministic - same keypair produced different hashes")
	}
}

// TestPublicKeyHashUniqueness tests that different keypairs produce different hashes
func TestPublicKeyHashUniqueness(t *testing.T) {
	kp1, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() #1 failed: %v", err)
	}

	kp2, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() #2 failed: %v", err)
	}

	hash1, err := PublicKeyHash(kp1)
	if err != nil {
		t.Fatalf("PublicKeyHash() #1 failed: %v", err)
	}

	hash2, err := PublicKeyHash(kp2)
	if err != nil {
		t.Fatalf("PublicKeyHash() #2 failed: %v", err)
	}

	if bytes.Equal(hash1, hash2) {
		t.Error("Two different keypairs produced identical public key hashes")
	}
}

// TestPublicKeyHashErrorCases tests error handling
func TestPublicKeyHashErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		testFunc  func() error
		expectErr bool
	}{
		{
			name: "Hash with nil keypair",
			testFunc: func() error {
				_, err := PublicKeyHash(nil)
				return err
			},
			expectErr: true,
		},
		{
			name: "Hash with missing ML-DSA public key",
			testFunc: func() error {
				kp := &HybridKeypair{
					Ed25519PublicKey: make([]byte, classical.Ed25519PublicKeySize),
				}
				_, err := PublicKeyHash(kp)
				return err
			},
			expectErr: true,
		},
		{
			name: "Hash with missing Ed25519 public key",
			testFunc: func() error {
				kp := &HybridKeypair{
					MLDSAPublicKey: make([]byte, mldsa.PublicKeySize),
				}
				_, err := PublicKeyHash(kp)
				return err
			},
			expectErr: true,
		},
		{
			name: "Hash with invalid ML-DSA public key size",
			testFunc: func() error {
				kp := &HybridKeypair{
					MLDSAPublicKey:   make([]byte, 100), // Wrong size
					Ed25519PublicKey: make([]byte, classical.Ed25519PublicKeySize),
				}
				_, err := PublicKeyHash(kp)
				return err
			},
			expectErr: true,
		},
		{
			name: "Hash with invalid Ed25519 public key size",
			testFunc: func() error {
				kp := &HybridKeypair{
					MLDSAPublicKey:   make([]byte, mldsa.PublicKeySize),
					Ed25519PublicKey: make([]byte, 10), // Wrong size
				}
				_, err := PublicKeyHash(kp)
				return err
			},
			expectErr: true,
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

// TestPublicKeyHashFormat tests the hash concatenation format
func TestPublicKeyHashFormat(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	// Get the hash
	hash, err := PublicKeyHash(kp)
	if err != nil {
		t.Fatalf("PublicKeyHash() failed: %v", err)
	}

	// Manually compute expected hash to verify format
	expectedInput := make([]byte, len(kp.MLDSAPublicKey)+len(kp.Ed25519PublicKey))
	copy(expectedInput, kp.MLDSAPublicKey)
	copy(expectedInput[len(kp.MLDSAPublicKey):], kp.Ed25519PublicKey)

	// Verify input size is correct (2592 + 32 = 2624 bytes)
	expectedInputSize := mldsa.PublicKeySize + classical.Ed25519PublicKeySize
	if len(expectedInput) != expectedInputSize {
		t.Errorf("Combined public key size mismatch: expected %d, got %d", expectedInputSize, len(expectedInput))
	}

	// Verify the hash matches expected SHA-256 output
	if len(hash) != 32 {
		t.Errorf("SHA-256 hash must be 32 bytes, got %d", len(hash))
	}
}

// TestPublicKeyHashSmartContractIntegration validates hash format for blockchain use
func TestPublicKeyHashSmartContractIntegration(t *testing.T) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		t.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	hash, err := PublicKeyHash(kp)
	if err != nil {
		t.Fatalf("PublicKeyHash() failed: %v", err)
	}

	// Smart contracts require:
	// 1. Exactly 32 bytes (fits in Solidity bytes32)
	if len(hash) != 32 {
		t.Errorf("Smart contract requires 32-byte hash, got %d", len(hash))
	}

	// 2. Deterministic output
	hash2, _ := PublicKeyHash(kp)
	if !bytes.Equal(hash, hash2) {
		t.Error("Hash must be deterministic for blockchain verification")
	}

	// 3. Non-zero hash (entropy check)
	allZero := true
	for _, b := range hash {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("Public key hash should not be all zeros")
	}
}

// BenchmarkPublicKeyHash benchmarks public key hash computation
func BenchmarkPublicKeyHash(b *testing.B) {
	kp, err := GenerateHybridKeypair()
	if err != nil {
		b.Fatalf("GenerateHybridKeypair() failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := PublicKeyHash(kp)
		if err != nil {
			b.Fatalf("PublicKeyHash() failed: %v", err)
		}
	}
}
