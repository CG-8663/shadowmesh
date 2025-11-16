package classical

import (
	"bytes"
	"testing"
)

// TestX25519KeypairGeneration tests X25519 keypair generation
func TestX25519KeypairGeneration(t *testing.T) {
	kp, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() failed: %v", err)
	}

	// Verify public key size (32 bytes for X25519)
	if len(kp.PublicKey) != 32 {
		t.Errorf("Public key size mismatch: expected 32, got %d", len(kp.PublicKey))
	}

	// Verify private key size (32 bytes for X25519)
	if len(kp.PrivateKey) != 32 {
		t.Errorf("Private key size mismatch: expected 32, got %d", len(kp.PrivateKey))
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

// TestX25519Exchange tests ECDH key exchange between two parties
func TestX25519Exchange(t *testing.T) {
	// Generate keypairs for Alice and Bob
	alice, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() for Alice failed: %v", err)
	}

	bob, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() for Bob failed: %v", err)
	}

	// Alice computes shared secret with Bob's public key
	secretAlice, err := X25519Exchange(alice.PrivateKey, bob.PublicKey)
	if err != nil {
		t.Fatalf("X25519Exchange() for Alice failed: %v", err)
	}

	// Bob computes shared secret with Alice's public key
	secretBob, err := X25519Exchange(bob.PrivateKey, alice.PublicKey)
	if err != nil {
		t.Fatalf("X25519Exchange() for Bob failed: %v", err)
	}

	// Verify shared secrets match
	if !bytes.Equal(secretAlice, secretBob) {
		t.Error("Shared secrets do not match")
	}

	// Verify shared secret size (32 bytes)
	if len(secretAlice) != 32 {
		t.Errorf("Shared secret size mismatch: expected 32, got %d", len(secretAlice))
	}
}

// TestX25519MultipleExchanges tests multiple ECDH exchanges
func TestX25519MultipleExchanges(t *testing.T) {
	alice, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() for Alice failed: %v", err)
	}

	// Perform 10 exchanges with different Bob keypairs
	for i := 0; i < 10; i++ {
		bob, err := GenerateX25519Keypair()
		if err != nil {
			t.Fatalf("Exchange %d: GenerateX25519Keypair() for Bob failed: %v", i, err)
		}

		secretAlice, err := X25519Exchange(alice.PrivateKey, bob.PublicKey)
		if err != nil {
			t.Fatalf("Exchange %d: X25519Exchange() for Alice failed: %v", i, err)
		}

		secretBob, err := X25519Exchange(bob.PrivateKey, alice.PublicKey)
		if err != nil {
			t.Fatalf("Exchange %d: X25519Exchange() for Bob failed: %v", i, err)
		}

		if !bytes.Equal(secretAlice, secretBob) {
			t.Errorf("Exchange %d: Shared secrets do not match", i)
		}
	}
}

// TestX25519InvalidPrivateKey tests error handling for invalid private keys
func TestX25519InvalidPrivateKey(t *testing.T) {
	bob, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() for Bob failed: %v", err)
	}

	testCases := []struct {
		name       string
		privateKey []byte
		wantErr    error
	}{
		{
			name:       "nil private key",
			privateKey: nil,
			wantErr:    ErrECDHFailed,
		},
		{
			name:       "empty private key",
			privateKey: []byte{},
			wantErr:    ErrECDHFailed,
		},
		{
			name:       "too short private key",
			privateKey: make([]byte, 10),
			wantErr:    ErrECDHFailed,
		},
		{
			name:       "too long private key",
			privateKey: make([]byte, 64),
			wantErr:    ErrECDHFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := X25519Exchange(tc.privateKey, bob.PublicKey)
			if err == nil {
				t.Error("Expected error but got nil")
			}
			if !bytes.Contains([]byte(err.Error()), []byte(tc.wantErr.Error())) {
				t.Errorf("Expected error containing %q, got %q", tc.wantErr, err)
			}
		})
	}
}

// TestX25519InvalidPublicKey tests error handling for invalid public keys
func TestX25519InvalidPublicKey(t *testing.T) {
	alice, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() for Alice failed: %v", err)
	}

	testCases := []struct {
		name      string
		publicKey []byte
		wantErr   error
	}{
		{
			name:      "nil public key",
			publicKey: nil,
			wantErr:   ErrInvalidPublicKey,
		},
		{
			name:      "empty public key",
			publicKey: []byte{},
			wantErr:   ErrInvalidPublicKey,
		},
		{
			name:      "too short public key",
			publicKey: make([]byte, 10),
			wantErr:   ErrInvalidPublicKey,
		},
		{
			name:      "too long public key",
			publicKey: make([]byte, 64),
			wantErr:   ErrInvalidPublicKey,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := X25519Exchange(alice.PrivateKey, tc.publicKey)
			if err == nil {
				t.Error("Expected error but got nil")
			}
			if !bytes.Contains([]byte(err.Error()), []byte(tc.wantErr.Error())) {
				t.Errorf("Expected error containing %q, got %q", tc.wantErr, err)
			}
		})
	}
}

// TestX25519DifferentKeypairs tests that different keypairs produce different shared secrets
func TestX25519DifferentKeypairs(t *testing.T) {
	alice, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() for Alice failed: %v", err)
	}

	bob1, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() for Bob1 failed: %v", err)
	}

	bob2, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() for Bob2 failed: %v", err)
	}

	// Alice's shared secret with Bob1
	secret1, err := X25519Exchange(alice.PrivateKey, bob1.PublicKey)
	if err != nil {
		t.Fatalf("X25519Exchange() with Bob1 failed: %v", err)
	}

	// Alice's shared secret with Bob2
	secret2, err := X25519Exchange(alice.PrivateKey, bob2.PublicKey)
	if err != nil {
		t.Fatalf("X25519Exchange() with Bob2 failed: %v", err)
	}

	// Verify shared secrets are different
	if bytes.Equal(secret1, secret2) {
		t.Error("Different public keys produced same shared secret (security violation)")
	}
}

// BenchmarkX25519KeypairGeneration benchmarks X25519 keypair generation
func BenchmarkX25519KeypairGeneration(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := GenerateX25519Keypair()
		if err != nil {
			b.Fatalf("GenerateX25519Keypair() failed: %v", err)
		}
	}
}

// BenchmarkX25519Exchange benchmarks X25519 ECDH key exchange
func BenchmarkX25519Exchange(b *testing.B) {
	alice, err := GenerateX25519Keypair()
	if err != nil {
		b.Fatalf("GenerateX25519Keypair() for Alice failed: %v", err)
	}

	bob, err := GenerateX25519Keypair()
	if err != nil {
		b.Fatalf("GenerateX25519Keypair() for Bob failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := X25519Exchange(alice.PrivateKey, bob.PublicKey)
		if err != nil {
			b.Fatalf("X25519Exchange() failed: %v", err)
		}
	}
}

// TestX25519KeypairUniqueness tests that generated keypairs are unique
func TestX25519KeypairUniqueness(t *testing.T) {
	kp1, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() #1 failed: %v", err)
	}

	kp2, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() #2 failed: %v", err)
	}

	// Public keys should be different
	if bytes.Equal(kp1.PublicKey, kp2.PublicKey) {
		t.Error("Two keypairs have identical public keys (entropy failure)")
	}

	// Private keys should be different
	if bytes.Equal(kp1.PrivateKey, kp2.PrivateKey) {
		t.Error("Two keypairs have identical private keys (entropy failure)")
	}
}

// TestX25519CorruptedPublicKey tests exchange with corrupted public key
func TestX25519CorruptedPublicKey(t *testing.T) {
	alice, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() failed: %v", err)
	}

	bob, err := GenerateX25519Keypair()
	if err != nil {
		t.Fatalf("GenerateX25519Keypair() failed: %v", err)
	}

	// Original exchange
	secret1, err := X25519Exchange(alice.PrivateKey, bob.PublicKey)
	if err != nil {
		t.Fatalf("X25519Exchange() failed: %v", err)
	}

	// Corrupt public key
	corruptedPubKey := make([]byte, 32)
	copy(corruptedPubKey, bob.PublicKey)
	corruptedPubKey[0] ^= 0xFF

	// Exchange with corrupted key should produce different secret or error
	secret2, err := X25519Exchange(alice.PrivateKey, corruptedPubKey)
	if err == nil {
		// If no error, secrets must be different
		if bytes.Equal(secret1, secret2) {
			t.Error("Corrupted public key produced same shared secret")
		}
	}
}

// TestX25519SharedSecretSize tests that shared secret is always 32 bytes
func TestX25519SharedSecretSize(t *testing.T) {
	for i := 0; i < 100; i++ {
		alice, err := GenerateX25519Keypair()
		if err != nil {
			t.Fatalf("Iteration %d: GenerateX25519Keypair() for Alice failed: %v", i, err)
		}

		bob, err := GenerateX25519Keypair()
		if err != nil {
			t.Fatalf("Iteration %d: GenerateX25519Keypair() for Bob failed: %v", i, err)
		}

		secret, err := X25519Exchange(alice.PrivateKey, bob.PublicKey)
		if err != nil {
			t.Fatalf("Iteration %d: X25519Exchange() failed: %v", i, err)
		}

		if len(secret) != 32 {
			t.Errorf("Iteration %d: Shared secret size mismatch: expected 32, got %d", i, len(secret))
		}
	}
}
