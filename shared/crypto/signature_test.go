package crypto

import (
	"bytes"
	"crypto/ed25519"
	"testing"
)

// TestGenerateSigningKey tests key generation
func TestGenerateSigningKey(t *testing.T) {
	key, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	if key == nil {
		t.Fatal("Generated key is nil")
	}

	// Verify ML-DSA-87 key is valid
	if key.MLDSAPrivateKey == nil {
		t.Error("ML-DSA-87 private key is nil")
	}

	// Verify Ed25519 key is valid
	if len(key.Ed25519PrivateKey) != ed25519.PrivateKeySize {
		t.Errorf("Ed25519 private key has wrong size: got %d, want %d",
			len(key.Ed25519PrivateKey), ed25519.PrivateKeySize)
	}
}

// TestPublicKeyExtraction tests extracting public key from signing key
func TestPublicKeyExtraction(t *testing.T) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	pubKey := signingKey.PublicKey()
	if pubKey == nil {
		t.Fatal("Public key is nil")
	}

	// Verify ML-DSA-87 public key is valid
	if pubKey.MLDSAPublicKey == nil {
		t.Error("ML-DSA-87 public key is nil")
	}

	// Verify Ed25519 public key is valid
	if len(pubKey.Ed25519PublicKey) != ed25519.PublicKeySize {
		t.Errorf("Ed25519 public key has wrong size: got %d, want %d",
			len(pubKey.Ed25519PublicKey), ed25519.PublicKeySize)
	}
}

// TestSignAndVerify tests successful signing and verification
func TestSignAndVerify(t *testing.T) {
	// Generate key pair
	signingKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()

	// Test message
	message := []byte("Hello, ShadowMesh! This is a test message for hybrid signatures.")

	// Sign the message
	signature, err := Sign(signingKey, message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Verify signature length
	if len(signature) != HybridSignatureSize {
		t.Errorf("Signature has wrong length: got %d, want %d",
			len(signature), HybridSignatureSize)
	}

	// Verify the signature
	err = Verify(verifyKey, message, signature)
	if err != nil {
		t.Fatalf("Failed to verify valid signature: %v", err)
	}
}

// TestVerifyWrongMessage tests that verification fails with wrong message
func TestVerifyWrongMessage(t *testing.T) {
	// Generate key pair
	signingKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()

	// Original message
	originalMessage := []byte("Original message")
	signature, err := Sign(signingKey, originalMessage)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Try to verify with different message
	wrongMessage := []byte("Wrong message")
	err = Verify(verifyKey, wrongMessage, signature)
	if err == nil {
		t.Fatal("Verification should fail with wrong message")
	}

	if err != ErrInvalidSignature && !bytes.Contains([]byte(err.Error()), []byte("signature verification failed")) {
		t.Errorf("Expected signature verification error, got: %v", err)
	}
}

// TestVerifyWrongPublicKey tests that verification fails with wrong public key
func TestVerifyWrongPublicKey(t *testing.T) {
	// Generate first key pair
	signingKey1, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key 1: %v", err)
	}

	// Generate second key pair
	signingKey2, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key 2: %v", err)
	}

	verifyKey2 := signingKey2.PublicKey()

	// Sign with first key
	message := []byte("Test message")
	signature, err := Sign(signingKey1, message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Try to verify with second (wrong) public key
	err = Verify(verifyKey2, message, signature)
	if err == nil {
		t.Fatal("Verification should fail with wrong public key")
	}

	if err != ErrInvalidSignature && !bytes.Contains([]byte(err.Error()), []byte("signature verification failed")) {
		t.Errorf("Expected signature verification error, got: %v", err)
	}
}

// TestVerifyCorruptedSignature tests verification with corrupted signatures
func TestVerifyCorruptedSignature(t *testing.T) {
	// Generate key pair
	signingKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()
	message := []byte("Test message")

	// Sign the message
	signature, err := Sign(signingKey, message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	t.Run("CorruptMLDSASignature", func(t *testing.T) {
		corruptedSig := make([]byte, len(signature))
		copy(corruptedSig, signature)
		// Corrupt first byte of ML-DSA-87 signature
		corruptedSig[0] ^= 0xFF

		err = Verify(verifyKey, message, corruptedSig)
		if err == nil {
			t.Fatal("Verification should fail with corrupted ML-DSA-87 signature")
		}
	})

	t.Run("CorruptEd25519Signature", func(t *testing.T) {
		corruptedSig := make([]byte, len(signature))
		copy(corruptedSig, signature)
		// Corrupt first byte of Ed25519 signature
		corruptedSig[MLDSASignatureSize] ^= 0xFF

		err = Verify(verifyKey, message, corruptedSig)
		if err == nil {
			t.Fatal("Verification should fail with corrupted Ed25519 signature")
		}
	})

	t.Run("TruncatedSignature", func(t *testing.T) {
		truncatedSig := signature[:len(signature)-10]

		err = Verify(verifyKey, message, truncatedSig)
		if err != ErrInvalidSignatureLength && !bytes.Contains([]byte(err.Error()), []byte("invalid signature length")) {
			t.Errorf("Expected invalid signature length error, got: %v", err)
		}
	})

	t.Run("ExtendedSignature", func(t *testing.T) {
		extendedSig := append(signature, []byte{0x00, 0x00}...)

		err = Verify(verifyKey, message, extendedSig)
		if err != ErrInvalidSignatureLength && !bytes.Contains([]byte(err.Error()), []byte("invalid signature length")) {
			t.Errorf("Expected invalid signature length error, got: %v", err)
		}
	})
}

// TestPublicKeyHash tests public key hash generation
func TestPublicKeyHash(t *testing.T) {
	// Generate key pair
	signingKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()

	// Generate hash
	hash1 := PublicKeyHash(verifyKey)

	// Verify hash is not empty
	emptyHash := [32]byte{}
	if hash1 == emptyHash {
		t.Error("Public key hash is empty")
	}

	// Verify hash is deterministic
	hash2 := PublicKeyHash(verifyKey)
	if hash1 != hash2 {
		t.Error("Public key hash is not deterministic")
	}

	// Verify different keys produce different hashes
	signingKey2, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate second signing key: %v", err)
	}
	verifyKey2 := signingKey2.PublicKey()

	hash3 := PublicKeyHash(verifyKey2)
	if hash1 == hash3 {
		t.Error("Different public keys produced same hash (collision)")
	}
}

// TestPublicKeyHashNil tests that nil public key returns empty hash
func TestPublicKeyHashNil(t *testing.T) {
	hash := PublicKeyHash(nil)
	emptyHash := [32]byte{}
	if hash != emptyHash {
		t.Error("Nil public key should return empty hash")
	}
}

// TestSignatureFormat tests signature format and structure
func TestSignatureFormat(t *testing.T) {
	// Generate key pair
	signingKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	message := []byte("Test message for format validation")

	// Sign the message
	signature, err := Sign(signingKey, message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Verify total length
	if len(signature) != HybridSignatureSize {
		t.Errorf("Signature length incorrect: got %d, want %d",
			len(signature), HybridSignatureSize)
	}

	// Verify expected size is correct
	expectedSize := MLDSASignatureSize + Ed25519SignatureSize
	if HybridSignatureSize != expectedSize {
		t.Errorf("HybridSignatureSize constant incorrect: got %d, want %d",
			HybridSignatureSize, expectedSize)
	}

	// Verify we can extract both components
	mldsaSig := signature[:MLDSASignatureSize]
	ed25519Sig := signature[MLDSASignatureSize:]

	if len(mldsaSig) != MLDSASignatureSize {
		t.Errorf("ML-DSA-87 signature component has wrong length: got %d, want %d",
			len(mldsaSig), MLDSASignatureSize)
	}

	if len(ed25519Sig) != Ed25519SignatureSize {
		t.Errorf("Ed25519 signature component has wrong length: got %d, want %d",
			len(ed25519Sig), Ed25519SignatureSize)
	}
}

// TestSignNilInputs tests signing with nil inputs
func TestSignNilInputs(t *testing.T) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	message := []byte("Test message")

	t.Run("NilPrivateKey", func(t *testing.T) {
		_, err := Sign(nil, message)
		if err == nil {
			t.Error("Signing with nil private key should fail")
		}
	})

	t.Run("NilMessage", func(t *testing.T) {
		_, err := Sign(signingKey, nil)
		if err == nil {
			t.Error("Signing nil message should fail")
		}
	})
}

// TestVerifyNilInputs tests verification with nil inputs
func TestVerifyNilInputs(t *testing.T) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()
	message := []byte("Test message")
	signature, err := Sign(signingKey, message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	t.Run("NilPublicKey", func(t *testing.T) {
		err := Verify(nil, message, signature)
		if err != ErrInvalidVerifyKey {
			t.Errorf("Expected ErrInvalidVerifyKey, got: %v", err)
		}
	})

	t.Run("NilMessage", func(t *testing.T) {
		err := Verify(verifyKey, nil, signature)
		if err == nil {
			t.Error("Verifying with nil message should fail")
		}
	})

	t.Run("NilSignature", func(t *testing.T) {
		err := Verify(verifyKey, message, nil)
		if err != ErrInvalidSignature && !bytes.Contains([]byte(err.Error()), []byte("signature verification failed")) {
			t.Errorf("Expected signature verification error, got: %v", err)
		}
	})
}

// TestVerifyKeyBytes tests public key serialization
func TestVerifyKeyBytes(t *testing.T) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()

	bytes1, err := verifyKey.Bytes()
	if err != nil {
		t.Fatalf("Failed to serialize public key: %v", err)
	}

	if len(bytes1) == 0 {
		t.Error("Serialized public key is empty")
	}

	// Verify deterministic serialization
	bytes2, err := verifyKey.Bytes()
	if err != nil {
		t.Fatalf("Failed to serialize public key second time: %v", err)
	}

	if !bytes.Equal(bytes1, bytes2) {
		t.Error("Public key serialization is not deterministic")
	}
}

// TestVerifyKeyBytesNil tests serialization of nil public key
func TestVerifyKeyBytesNil(t *testing.T) {
	var nilKey *HybridVerifyKey
	_, err := nilKey.Bytes()
	if err != ErrInvalidVerifyKey {
		t.Errorf("Expected ErrInvalidVerifyKey for nil key, got: %v", err)
	}
}

// TestMultipleSignatures tests that same key can sign multiple messages
func TestMultipleSignatures(t *testing.T) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()

	messages := [][]byte{
		[]byte("First message"),
		[]byte("Second message"),
		[]byte("Third message with different content"),
	}

	signatures := make([][]byte, len(messages))

	// Sign all messages
	for i, msg := range messages {
		sig, err := Sign(signingKey, msg)
		if err != nil {
			t.Fatalf("Failed to sign message %d: %v", i, err)
		}
		signatures[i] = sig
	}

	// Verify all signatures
	for i, msg := range messages {
		err := Verify(verifyKey, msg, signatures[i])
		if err != nil {
			t.Errorf("Failed to verify signature %d: %v", i, err)
		}
	}

	// Verify cross-verification fails (message i with signature j)
	for i := range messages {
		for j := range signatures {
			if i != j {
				err := Verify(verifyKey, messages[i], signatures[j])
				if err == nil {
					t.Errorf("Verification should fail for message %d with signature %d", i, j)
				}
			}
		}
	}
}

// TestEmptyMessage tests signing and verifying empty messages
func TestEmptyMessage(t *testing.T) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()

	// Empty message (but not nil)
	emptyMessage := []byte{}

	signature, err := Sign(signingKey, emptyMessage)
	if err != nil {
		t.Fatalf("Failed to sign empty message: %v", err)
	}

	err = Verify(verifyKey, emptyMessage, signature)
	if err != nil {
		t.Fatalf("Failed to verify signature of empty message: %v", err)
	}
}

// TestLargeMessage tests signing and verifying large messages
func TestLargeMessage(t *testing.T) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()

	// Create a large message (1 MB)
	largeMessage := make([]byte, 1024*1024)
	for i := range largeMessage {
		largeMessage[i] = byte(i % 256)
	}

	signature, err := Sign(signingKey, largeMessage)
	if err != nil {
		t.Fatalf("Failed to sign large message: %v", err)
	}

	err = Verify(verifyKey, largeMessage, signature)
	if err != nil {
		t.Fatalf("Failed to verify signature of large message: %v", err)
	}
}
