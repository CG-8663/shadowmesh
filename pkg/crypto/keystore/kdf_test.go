package keystore

import (
	"crypto/rand"
	"testing"
	"time"
)

// TestValidatePassphrase tests passphrase validation logic
func TestValidatePassphrase(t *testing.T) {
	tests := []struct {
		name       string
		passphrase string
		wantErr    bool
		errType    error
	}{
		{
			name:       "valid minimum length",
			passphrase: "validpass123",
			wantErr:    false,
		},
		{
			name:       "valid long passphrase",
			passphrase: "this is a very long and secure passphrase with multiple words",
			wantErr:    false,
		},
		{
			name:       "valid UTF-8 passphrase",
			passphrase: "パスワード1234567",
			wantErr:    false,
		},
		{
			name:       "too short",
			passphrase: "short",
			wantErr:    true,
			errType:    ErrPassphraseTooShort,
		},
		{
			name:       "exactly 11 characters",
			passphrase: "12345678901",
			wantErr:    true,
			errType:    ErrPassphraseTooShort,
		},
		{
			name:       "empty passphrase",
			passphrase: "",
			wantErr:    true,
			errType:    ErrEmptyPassphrase,
		},
		{
			name:       "whitespace only",
			passphrase: "            ",
			wantErr:    true,
		},
		{
			name:       "common weak passphrase",
			passphrase: "password1234",
			wantErr:    true,
		},
		{
			name:       "weak passphrase uppercase",
			passphrase: "PASSWORD1234",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassphrase(tt.passphrase)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassphrase() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errType != nil && err != nil {
				// Check if error is of expected type
				if err.Error()[:len(tt.errType.Error())] != tt.errType.Error() {
					t.Errorf("ValidatePassphrase() error = %v, want error type %v", err, tt.errType)
				}
			}
		})
	}
}

// TestDeriveKey tests basic key derivation
func TestDeriveKey(t *testing.T) {
	passphrase := "my-secure-passphrase-123"
	salt := make([]byte, SaltSize)
	rand.Read(salt)
	iterations := DefaultIterations

	key, err := DeriveKey(passphrase, salt, iterations)
	if err != nil {
		t.Fatalf("DeriveKey() failed: %v", err)
	}

	// Verify key is not all zeros
	allZero := true
	for _, b := range key {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("DeriveKey() produced all-zero key")
	}

	// Verify key is 32 bytes
	if len(key) != KeySize {
		t.Errorf("DeriveKey() key size = %d, want %d", len(key), KeySize)
	}
}

// TestDeriveKeyDeterministic tests that same inputs produce same output
func TestDeriveKeyDeterministic(t *testing.T) {
	passphrase := "deterministic-test-passphrase"
	salt := make([]byte, SaltSize)
	rand.Read(salt)
	iterations := 10000

	// Derive key twice with same inputs
	key1, err1 := DeriveKey(passphrase, salt, iterations)
	key2, err2 := DeriveKey(passphrase, salt, iterations)

	if err1 != nil {
		t.Fatalf("First DeriveKey() failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Second DeriveKey() failed: %v", err2)
	}

	// Keys should be identical
	if key1 != key2 {
		t.Error("DeriveKey() is not deterministic: same inputs produced different keys")
	}
}

// TestDeriveKeyDifferentPassphrases tests that different passphrases produce different keys
func TestDeriveKeyDifferentPassphrases(t *testing.T) {
	salt := make([]byte, SaltSize)
	rand.Read(salt)
	iterations := 10000

	key1, err1 := DeriveKey("passphrase-one-123", salt, iterations)
	key2, err2 := DeriveKey("passphrase-two-456", salt, iterations)

	if err1 != nil || err2 != nil {
		t.Fatalf("DeriveKey() failed: %v, %v", err1, err2)
	}

	// Keys should be different
	if key1 == key2 {
		t.Error("DeriveKey() produced same key for different passphrases")
	}
}

// TestDeriveKeyDifferentSalts tests that different salts produce different keys
func TestDeriveKeyDifferentSalts(t *testing.T) {
	passphrase := "same-passphrase-123"
	iterations := 10000

	salt1 := make([]byte, SaltSize)
	salt2 := make([]byte, SaltSize)
	rand.Read(salt1)
	rand.Read(salt2)

	key1, err1 := DeriveKey(passphrase, salt1, iterations)
	key2, err2 := DeriveKey(passphrase, salt2, iterations)

	if err1 != nil || err2 != nil {
		t.Fatalf("DeriveKey() failed: %v, %v", err1, err2)
	}

	// Keys should be different
	if key1 == key2 {
		t.Error("DeriveKey() produced same key for different salts")
	}
}

// TestDeriveKeyInvalidPassphrase tests error handling for invalid passphrases
func TestDeriveKeyInvalidPassphrase(t *testing.T) {
	salt := make([]byte, SaltSize)
	rand.Read(salt)

	tests := []struct {
		name       string
		passphrase string
		iterations int
		wantErr    bool
	}{
		{
			name:       "too short",
			passphrase: "short",
			iterations: 100000,
			wantErr:    true,
		},
		{
			name:       "empty",
			passphrase: "",
			iterations: 100000,
			wantErr:    true,
		},
		{
			name:       "valid passphrase",
			passphrase: "valid-passphrase-123",
			iterations: 100000,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeriveKey(tt.passphrase, salt, tt.iterations)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeriveKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDeriveKeyInvalidSalt tests error handling for invalid salt
func TestDeriveKeyInvalidSalt(t *testing.T) {
	passphrase := "valid-passphrase-123"
	iterations := 100000

	tests := []struct {
		name     string
		saltSize int
		wantErr  bool
	}{
		{
			name:     "valid salt size",
			saltSize: SaltSize,
			wantErr:  false,
		},
		{
			name:     "salt too short",
			saltSize: 16,
			wantErr:  true,
		},
		{
			name:     "salt too long",
			saltSize: 64,
			wantErr:  true,
		},
		{
			name:     "empty salt",
			saltSize: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			salt := make([]byte, tt.saltSize)
			if tt.saltSize > 0 {
				rand.Read(salt)
			}
			_, err := DeriveKey(passphrase, salt, iterations)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeriveKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDeriveKeyInvalidIterations tests error handling for invalid iteration counts
func TestDeriveKeyInvalidIterations(t *testing.T) {
	passphrase := "valid-passphrase-123"
	salt := make([]byte, SaltSize)
	rand.Read(salt)

	tests := []struct {
		name       string
		iterations int
		wantErr    bool
	}{
		{
			name:       "valid iterations",
			iterations: 100000,
			wantErr:    false,
		},
		{
			name:       "minimum iterations",
			iterations: 10000,
			wantErr:    false,
		},
		{
			name:       "too few iterations",
			iterations: 5000,
			wantErr:    true,
		},
		{
			name:       "zero iterations",
			iterations: 0,
			wantErr:    true,
		},
		{
			name:       "negative iterations",
			iterations: -1,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeriveKey(passphrase, salt, tt.iterations)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeriveKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDeriveKeyIterationVariation tests that different iteration counts produce different keys
func TestDeriveKeyIterationVariation(t *testing.T) {
	passphrase := "same-passphrase-123"
	salt := make([]byte, SaltSize)
	rand.Read(salt)

	key1, err1 := DeriveKey(passphrase, salt, 10000)
	key2, err2 := DeriveKey(passphrase, salt, 20000)

	if err1 != nil || err2 != nil {
		t.Fatalf("DeriveKey() failed: %v, %v", err1, err2)
	}

	// Keys should be different
	if key1 == key2 {
		t.Error("DeriveKey() produced same key for different iteration counts")
	}
}

// TestDeriveKeyPerformance tests that PBKDF2 meets performance requirements
// AC 1.6.7: PBKDF2 should be intentionally slow (target: 50-100ms for 100k iterations)
func TestDeriveKeyPerformance(t *testing.T) {
	passphrase := "performance-test-passphrase"
	salt := make([]byte, SaltSize)
	rand.Read(salt)
	iterations := DefaultIterations // 100000

	start := time.Now()
	_, err := DeriveKey(passphrase, salt, iterations)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("DeriveKey() failed: %v", err)
	}

	t.Logf("PBKDF2 derivation time (100k iterations): %v", elapsed)

	// PBKDF2 should be slow (intentional security feature)
	// Minimum: 1ms (too fast indicates misconfiguration)
	// Maximum: 1000ms (too slow for user experience)
	if elapsed < 1*time.Millisecond {
		t.Errorf("PBKDF2 is suspiciously fast (%v), expected >1ms", elapsed)
	}
	if elapsed > 1000*time.Millisecond {
		t.Errorf("PBKDF2 is too slow (%v), expected <1000ms", elapsed)
	}
}

// BenchmarkDeriveKey benchmarks PBKDF2 key derivation
func BenchmarkDeriveKey(b *testing.B) {
	passphrase := "benchmark-passphrase-123"
	salt := make([]byte, SaltSize)
	rand.Read(salt)
	iterations := DefaultIterations

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := DeriveKey(passphrase, salt, iterations)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDeriveKeyFast benchmarks PBKDF2 with fewer iterations (for comparison)
func BenchmarkDeriveKeyFast(b *testing.B) {
	passphrase := "benchmark-passphrase-123"
	salt := make([]byte, SaltSize)
	rand.Read(salt)
	iterations := 10000 // 10x fewer iterations

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := DeriveKey(passphrase, salt, iterations)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidatePassphrase benchmarks passphrase validation
func BenchmarkValidatePassphrase(b *testing.B) {
	passphrase := "valid-benchmark-passphrase"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ValidatePassphrase(passphrase)
	}
}
