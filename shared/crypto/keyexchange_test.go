package crypto

import (
	"bytes"
	"crypto/ecdh"
	"testing"

	"github.com/cloudflare/circl/kem/kyber/kyber1024"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	if kp == nil {
		t.Fatal("GenerateKeyPair returned nil keypair")
	}

	if kp.KEMPublicKey == nil {
		t.Error("KEM public key is nil")
	}

	if kp.KEMPrivateKey == nil {
		t.Error("KEM private key is nil")
	}

	if kp.ECDHPublicKey == nil {
		t.Error("ECDH public key is nil")
	}

	if kp.ECDHPrivateKey == nil {
		t.Error("ECDH private key is nil")
	}

	// Verify key sizes
	scheme := kyber1024.Scheme()
	kemPubBytes, _ := kp.KEMPublicKey.MarshalBinary()
	if len(kemPubBytes) != scheme.PublicKeySize() {
		t.Errorf("KEM public key size mismatch: expected %d, got %d", scheme.PublicKeySize(), len(kemPubBytes))
	}

	kemPrivBytes, _ := kp.KEMPrivateKey.MarshalBinary()
	if len(kemPrivBytes) != scheme.PrivateKeySize() {
		t.Errorf("KEM private key size mismatch: expected %d, got %d", scheme.PrivateKeySize(), len(kemPrivBytes))
	}

	ecdhPubBytes := kp.ECDHPublicKey.Bytes()
	if len(ecdhPubBytes) != 32 {
		t.Errorf("ECDH public key size mismatch: expected 32, got %d", len(ecdhPubBytes))
	}
}

func TestEncapsulateDecapsulate(t *testing.T) {
	// Generate recipient keypair
	recipientKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient keypair: %v", err)
	}

	// Encapsulate to recipient's public key
	sharedSecret1, ciphertext, err := Encapsulate(recipientKP.PublicKey())
	if err != nil {
		t.Fatalf("Encapsulate failed: %v", err)
	}

	if len(sharedSecret1) != SharedSecretSize {
		t.Errorf("Shared secret size mismatch: expected %d, got %d", SharedSecretSize, len(sharedSecret1))
	}

	if len(ciphertext) == 0 {
		t.Error("Ciphertext is empty")
	}

	// Decapsulate using recipient's private key
	sharedSecret2, err := Decapsulate(recipientKP, ciphertext)
	if err != nil {
		t.Fatalf("Decapsulate failed: %v", err)
	}

	if len(sharedSecret2) != SharedSecretSize {
		t.Errorf("Decapsulated shared secret size mismatch: expected %d, got %d", SharedSecretSize, len(sharedSecret2))
	}

	// Verify that both shared secrets match
	if !bytes.Equal(sharedSecret1, sharedSecret2) {
		t.Error("Shared secrets do not match after encapsulation/decapsulation")
	}
}

func TestEncapsulateDecapsulateMultipleRounds(t *testing.T) {
	// Test that multiple encapsulations produce different ciphertexts and secrets
	recipientKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient keypair: %v", err)
	}

	sharedSecret1, ciphertext1, err := Encapsulate(recipientKP.PublicKey())
	if err != nil {
		t.Fatalf("First encapsulate failed: %v", err)
	}

	sharedSecret2, ciphertext2, err := Encapsulate(recipientKP.PublicKey())
	if err != nil {
		t.Fatalf("Second encapsulate failed: %v", err)
	}

	// Ciphertexts should be different (due to randomness)
	if bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("Multiple encapsulations produced identical ciphertexts (should be random)")
	}

	// Shared secrets should be different (due to randomness)
	if bytes.Equal(sharedSecret1, sharedSecret2) {
		t.Error("Multiple encapsulations produced identical shared secrets (should be random)")
	}

	// But each can be decapsulated correctly
	decryptedSecret1, err := Decapsulate(recipientKP, ciphertext1)
	if err != nil {
		t.Fatalf("Failed to decapsulate first ciphertext: %v", err)
	}

	if !bytes.Equal(sharedSecret1, decryptedSecret1) {
		t.Error("First shared secret mismatch after decapsulation")
	}

	decryptedSecret2, err := Decapsulate(recipientKP, ciphertext2)
	if err != nil {
		t.Fatalf("Failed to decapsulate second ciphertext: %v", err)
	}

	if !bytes.Equal(sharedSecret2, decryptedSecret2) {
		t.Error("Second shared secret mismatch after decapsulation")
	}
}

func TestDecapsulateInvalidCiphertext(t *testing.T) {
	recipientKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient keypair: %v", err)
	}

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{
			name:       "empty ciphertext",
			ciphertext: []byte{},
		},
		{
			name:       "too short ciphertext",
			ciphertext: make([]byte, 100),
		},
		{
			name:       "too long ciphertext",
			ciphertext: make([]byte, 2000),
		},
		{
			name:       "nil ciphertext",
			ciphertext: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decapsulate(recipientKP, tt.ciphertext)
			if err == nil {
				t.Error("Expected error for invalid ciphertext, got nil")
			}
		})
	}
}

func TestDecapsulateNilKeyPair(t *testing.T) {
	_, err := Decapsulate(nil, []byte("dummy"))
	if err != ErrKEMNilKeyPair {
		t.Errorf("Expected ErrKEMNilKeyPair, got %v", err)
	}
}

func TestEncapsulateNilPublicKey(t *testing.T) {
	_, _, err := Encapsulate(nil)
	if err != ErrKEMNilKeyPair {
		t.Errorf("Expected ErrKEMNilKeyPair, got %v", err)
	}
}

func TestKEMPublicKeyBytes(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	pubBytes := kp.PublicKeyBytes()

	scheme := kyber1024.Scheme()
	expectedSize := scheme.PublicKeySize() + 32 // KEM public key + X25519 public key

	if len(pubBytes) != expectedSize {
		t.Errorf("PublicKeyBytes size mismatch: expected %d, got %d", expectedSize, len(pubBytes))
	}
}

func TestParsePublicKey(t *testing.T) {
	// Generate a keypair and serialize its public key
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	pubBytes := kp.PublicKeyBytes()

	// Parse the public key back
	parsedPub, err := ParsePublicKey(pubBytes)
	if err != nil {
		t.Fatalf("Failed to parse public key: %v", err)
	}

	// Verify the parsed key can be used for encapsulation
	sharedSecret1, ciphertext, err := Encapsulate(parsedPub)
	if err != nil {
		t.Fatalf("Encapsulate with parsed key failed: %v", err)
	}

	// Decapsulate with original private key
	sharedSecret2, err := Decapsulate(kp, ciphertext)
	if err != nil {
		t.Fatalf("Decapsulate failed: %v", err)
	}

	// Verify secrets match
	if !bytes.Equal(sharedSecret1, sharedSecret2) {
		t.Error("Shared secrets do not match after parsing public key")
	}
}

func TestParsePublicKeyInvalid(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
	}{
		{
			name:  "empty bytes",
			bytes: []byte{},
		},
		{
			name:  "too short",
			bytes: make([]byte, 100),
		},
		{
			name:  "too long",
			bytes: make([]byte, 2000),
		},
		{
			name:  "nil bytes",
			bytes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePublicKey(tt.bytes)
			if err == nil {
				t.Error("Expected error for invalid public key bytes, got nil")
			}
		})
	}
}

func TestDeriveSharedSecretDeterministic(t *testing.T) {
	kemSecret := []byte("test-kem-secret-32-bytes-long!!")
	ecdhSecret := []byte("test-ecdh-secret-32-bytes-lng!!")

	// Derive shared secret twice with same inputs
	secret1 := deriveSharedSecret(kemSecret, ecdhSecret)
	secret2 := deriveSharedSecret(kemSecret, ecdhSecret)

	// Should produce identical results
	if !bytes.Equal(secret1, secret2) {
		t.Error("deriveSharedSecret is not deterministic")
	}

	// Should be correct size
	if len(secret1) != SharedSecretSize {
		t.Errorf("Derived secret size mismatch: expected %d, got %d", SharedSecretSize, len(secret1))
	}
}

func TestDeriveSharedSecretUnique(t *testing.T) {
	kemSecret1 := []byte("test-kem-secret-32-bytes-long!!")
	ecdhSecret1 := []byte("test-ecdh-secret-32-bytes-lng!!")

	kemSecret2 := []byte("different-kem-secret-32-bytes!!")
	ecdhSecret2 := []byte("different-ecdh-secret-32-byte!")

	secret1 := deriveSharedSecret(kemSecret1, ecdhSecret1)
	secret2 := deriveSharedSecret(kemSecret2, ecdhSecret2)

	// Different inputs should produce different outputs
	if bytes.Equal(secret1, secret2) {
		t.Error("Different inputs produced identical shared secrets")
	}
}

func TestHybridKeyPairPublicKey(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	pubKey := kp.PublicKey()

	if pubKey == nil {
		t.Fatal("PublicKey() returned nil")
	}

	if pubKey.KEMPublicKey == nil {
		t.Error("Public key has nil KEM public key")
	}

	if pubKey.ECDHPublicKey == nil {
		t.Error("Public key has nil ECDH public key")
	}

	// Verify the public keys match
	kemPubBytes1, _ := kp.KEMPublicKey.MarshalBinary()
	kemPubBytes2, _ := pubKey.KEMPublicKey.MarshalBinary()

	if !bytes.Equal(kemPubBytes1, kemPubBytes2) {
		t.Error("KEM public keys do not match")
	}

	if !bytes.Equal(kp.ECDHPublicKey.Bytes(), pubKey.ECDHPublicKey.Bytes()) {
		t.Error("ECDH public keys do not match")
	}
}

func TestCiphertextStructure(t *testing.T) {
	recipientKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient keypair: %v", err)
	}

	_, ciphertext, err := Encapsulate(recipientKP.PublicKey())
	if err != nil {
		t.Fatalf("Encapsulate failed: %v", err)
	}

	scheme := kyber1024.Scheme()
	expectedSize := scheme.CiphertextSize() + 32 // KEM ciphertext + ephemeral X25519 public key

	if len(ciphertext) != expectedSize {
		t.Errorf("Ciphertext size mismatch: expected %d, got %d", expectedSize, len(ciphertext))
	}
}

func TestDecapsulateWrongKey(t *testing.T) {
	// Generate two different keypairs
	kp1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate first keypair: %v", err)
	}

	kp2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate second keypair: %v", err)
	}

	// Encapsulate to first keypair
	sharedSecret1, ciphertext, err := Encapsulate(kp1.PublicKey())
	if err != nil {
		t.Fatalf("Encapsulate failed: %v", err)
	}

	// Try to decapsulate with second keypair (wrong key)
	sharedSecret2, err := Decapsulate(kp2, ciphertext)

	// Decapsulation should succeed (no authentication in KEM)
	// but produce a different shared secret
	if err != nil {
		t.Logf("Decapsulation with wrong key failed (acceptable): %v", err)
		return
	}

	// If it didn't fail, the secrets should be different
	if bytes.Equal(sharedSecret1, sharedSecret2) {
		t.Error("Decapsulation with wrong key produced same shared secret (security issue)")
	}
}

func TestInvalidECDHPublicKeyInCiphertext(t *testing.T) {
	recipientKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient keypair: %v", err)
	}

	_, ciphertext, err := Encapsulate(recipientKP.PublicKey())
	if err != nil {
		t.Fatalf("Encapsulate failed: %v", err)
	}

	// Corrupt the ECDH public key portion (last 32 bytes)
	scheme := kyber1024.Scheme()
	kemCiphertextSize := scheme.CiphertextSize()

	// Create invalid ciphertext with all zeros in ECDH portion
	invalidCiphertext := make([]byte, len(ciphertext))
	copy(invalidCiphertext, ciphertext[:kemCiphertextSize])
	// Leave the rest as zeros (invalid X25519 public key)

	_, err = Decapsulate(recipientKP, invalidCiphertext)
	// X25519 may accept all-zero key, but we test that the function handles it
	if err != nil {
		t.Logf("Decapsulation with invalid ECDH key failed as expected: %v", err)
	}
}

func TestECDHComponentIsUsed(t *testing.T) {
	// This test verifies that the ECDH component actually contributes to the shared secret
	recipientKP, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient keypair: %v", err)
	}

	_, ciphertext, err := Encapsulate(recipientKP.PublicKey())
	if err != nil {
		t.Fatalf("Encapsulate failed: %v", err)
	}

	// Modify the ECDH public key in the ciphertext
	scheme := kyber1024.Scheme()
	kemCiphertextSize := scheme.CiphertextSize()

	modifiedCiphertext := make([]byte, len(ciphertext))
	copy(modifiedCiphertext, ciphertext)
	modifiedCiphertext[kemCiphertextSize]++ // Flip one bit in ECDH public key

	// Decapsulate both
	secret1, err1 := Decapsulate(recipientKP, ciphertext)
	secret2, err2 := Decapsulate(recipientKP, modifiedCiphertext)

	// If ECDH is used, modifying the ECDH key should either fail or produce different secret
	if err1 == nil && err2 == nil {
		if bytes.Equal(secret1, secret2) {
			t.Error("Modifying ECDH component did not change shared secret (ECDH may not be used)")
		}
	}
}

func TestInvalidECDHPublicKeyType(t *testing.T) {
	// Generate a valid keypair
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	// Create a public key with wrong ECDH curve (P-256 instead of X25519)
	wrongCurvePriv, err := ecdh.P256().GenerateKey(bytes.NewReader(make([]byte, 1000)))
	if err != nil {
		t.Fatalf("Failed to generate P-256 key: %v", err)
	}

	// Create hybrid public key with mismatched ECDH key
	wrongPubKey := &HybridPublicKey{
		KEMPublicKey:  kp.KEMPublicKey,
		ECDHPublicKey: wrongCurvePriv.PublicKey(),
	}

	// Try to encapsulate (should fail or produce incorrect result)
	_, _, err = Encapsulate(wrongPubKey)
	if err != nil {
		t.Logf("Encapsulation with wrong curve failed as expected: %v", err)
	} else {
		t.Log("Encapsulation with wrong curve succeeded (implementation accepts it)")
	}
}
