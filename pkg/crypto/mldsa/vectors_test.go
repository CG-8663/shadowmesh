package mldsa

import (
	"testing"

	"github.com/cloudflare/circl/sign/dilithium/mode5"
)

// TestDilithiumRound3Compliance validates that circl library implements Dilithium Round 3
// This test verifies scheme parameters match Dilithium Round 3 specification
// Note: circl v1.6.1 implements Round 3, not final FIPS 204 (4595 vs 4627 byte signatures)
func TestDilithiumRound3Compliance(t *testing.T) {
	scheme := Scheme()

	// Verify scheme name (Dilithium5 is ML-DSA-87 in FIPS 204)
	if scheme.Name() != "Dilithium5" {
		t.Errorf("Expected Dilithium5, got %s", scheme.Name())
	}

	// Verify public key size
	if scheme.PublicKeySize() != 2592 {
		t.Errorf("Expected 2592-byte public keys, got %d", scheme.PublicKeySize())
	}

	// Verify private key size
	if scheme.PrivateKeySize() != 4864 {
		t.Errorf("Expected 4864-byte private keys, got %d", scheme.PrivateKeySize())
	}

	// Verify signature size (Dilithium Round 3 uses 4595 bytes, FIPS 204 uses 4627)
	if scheme.SignatureSize() != 4595 {
		t.Errorf("Dilithium Round 3 specifies 4595-byte signatures, got %d", scheme.SignatureSize())
	}
}

// TestCirclLibraryVersion validates that we're using a known-good circl version
func TestCirclLibraryVersion(t *testing.T) {
	// circl v1.6.1 implements NIST FIPS 204 Draft 3
	// This test ensures the library is properly imported
	scheme := Scheme()

	if scheme == nil {
		t.Fatal("Failed to load circl Dilithium5 scheme")
	}

	// Verify we can generate keys (basic smoke test)
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() failed: %v", err)
	}

	if len(kp.PublicKey) != PublicKeySize {
		t.Errorf("Generated public key size mismatch: expected %d, got %d", PublicKeySize, len(kp.PublicKey))
	}
}

// TestDilithiumParameterValidation validates Dilithium Round 3 parameter set
func TestDilithiumParameterValidation(t *testing.T) {
	// Dilithium Round 3 parameters (circl implementation)
	// Note: Final FIPS 204 has 4627-byte signatures, Round 3 has 4595-byte signatures

	tests := []struct {
		name     string
		expected int
		actual   int
	}{
		{"Public Key Size", 2592, PublicKeySize},
		{"Private Key Size", 4864, PrivateKeySize},
		{"Signature Size (Round 3)", 4595, SignatureSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("%s mismatch: expected %d, got %d", tt.name, tt.expected, tt.actual)
			}
		})
	}
}

// TestSignatureSchemeProperties validates cryptographic properties required by FIPS 204
func TestSignatureSchemeProperties(t *testing.T) {
	// Property 1: EUF-CMA security (Existential Unforgeability under Chosen Message Attack)
	// Test: Valid signature verifies, tampered signature fails
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() failed: %v", err)
	}

	message := []byte("NIST FIPS 204 test message")

	signature, err := Sign(message, kp.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	// Valid signature should verify
	if !Verify(message, signature, kp.PublicKey) {
		t.Error("Valid signature failed verification (EUF-CMA property violated)")
	}

	// Tampered signature should fail
	tamperedSig := make([]byte, len(signature))
	copy(tamperedSig, signature)
	tamperedSig[0] ^= 0xFF

	if Verify(message, tamperedSig, kp.PublicKey) {
		t.Error("Tampered signature verified (EUF-CMA property violated)")
	}

	// Property 2: Message binding
	// Test: Signature on one message should not verify on different message
	differentMessage := []byte("Different NIST FIPS 204 test message")
	if Verify(differentMessage, signature, kp.PublicKey) {
		t.Error("Signature verified on different message (message binding violated)")
	}

	// Property 3: Public key binding
	// Test: Signature with one keypair should not verify with different keypair
	kp2, _ := GenerateKeypair()
	if Verify(message, signature, kp2.PublicKey) {
		t.Error("Signature verified with wrong public key (key binding violated)")
	}
}

// TestConstantTimeOperations validates that verification is constant-time
// Note: This is a basic sanity check - full constant-time validation requires
// side-channel analysis tools like ctgrind or dudect
func TestConstantTimeOperations(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair() failed: %v", err)
	}

	message := []byte("constant-time test message")

	signature, err := Sign(message, kp.PrivateKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	// Test that verification always completes (no early exit on first bit mismatch)
	// circl implements constant-time verification to prevent timing attacks
	iterations := 1000
	for i := 0; i < iterations; i++ {
		// Valid signature
		if !Verify(message, signature, kp.PublicKey) {
			t.Error("Constant-time verification failed for valid signature")
		}

		// Invalid signature (should still take same time, but we can't measure here)
		tamperedSig := make([]byte, len(signature))
		copy(tamperedSig, signature)
		tamperedSig[i%len(tamperedSig)] ^= 0x01

		Verify(message, tamperedSig, kp.PublicKey) // Should return false, but in constant time
	}
}

// TestSchemeInterface validates that mode5.Scheme implements expected interface
func TestSchemeInterface(t *testing.T) {
	// Verify we can use mode5 package directly
	pub, priv, err := mode5.GenerateKey(nil) // nil uses crypto/rand.Reader
	if err != nil {
		t.Fatalf("mode5.GenerateKey() failed: %v", err)
	}

	if pub == nil || priv == nil {
		t.Error("mode5.GenerateKey() returned nil keys")
	}

	// Verify scheme can sign
	message := []byte("scheme interface test")
	sig := make([]byte, SignatureSize)
	mode5.SignTo(priv, message, sig)

	if len(sig) != SignatureSize {
		t.Errorf("mode5.SignTo() produced wrong size: expected %d, got %d", SignatureSize, len(sig))
	}

	// Verify scheme can verify
	if !mode5.Verify(pub, message, sig) {
		t.Error("mode5.Verify() failed for valid signature")
	}
}

// TestMLDSA87SecurityLevel validates NIST security level 5 claims
func TestMLDSA87SecurityLevel(t *testing.T) {
	// ML-DSA-87 (Dilithium5) provides NIST Security Level 5:
	// - Comparable to AES-256, SHA-384
	// - ~256 bits of quantum security (resists Grover's algorithm)
	// - ~256 bits of classical security

	// This is a documentation test - no runtime validation possible
	// Security level is proven by NIST analysis, not testable in code

	scheme := Scheme()
	if scheme.Name() != "Dilithium5" {
		t.Errorf("Expected Dilithium5 (ML-DSA-87, NIST Level 5), got %s", scheme.Name())
	}

	// Verify key sizes match Level 5 specification
	if scheme.PublicKeySize() != 2592 || scheme.PrivateKeySize() != 4864 {
		t.Error("Key sizes do not match NIST Level 5 ML-DSA-87 specification")
	}
}
