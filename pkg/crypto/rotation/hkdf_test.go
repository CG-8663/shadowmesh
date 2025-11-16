package rotation

import (
	"crypto/rand"
	"testing"
)

// TestDeriveRotationKey tests basic key derivation
func TestDeriveRotationKey(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	newKey, err := DeriveRotationKey(key, 1)
	if err != nil {
		t.Fatalf("DeriveRotationKey failed: %v", err)
	}

	// Verify new key is 32 bytes
	if len(newKey) != KeySize {
		t.Errorf("Expected %d bytes, got %d", KeySize, len(newKey))
	}

	// Verify new key is different from original
	if newKey == key {
		t.Error("Derived key should be different from original key")
	}
}

// TestDeriveRotationKeyDeterministic tests that derivation is deterministic
func TestDeriveRotationKeyDeterministic(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	// Derive same key twice
	key1, err := DeriveRotationKey(key, 1)
	if err != nil {
		t.Fatalf("First derivation failed: %v", err)
	}

	key2, err := DeriveRotationKey(key, 1)
	if err != nil {
		t.Fatalf("Second derivation failed: %v", err)
	}

	// Keys should be identical
	if key1 != key2 {
		t.Error("Derived keys should be identical for same input")
	}
}

// TestDeriveRotationKeyDifferentSequences tests that different sequences produce different keys
func TestDeriveRotationKeyDifferentSequences(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	key1, _ := DeriveRotationKey(key, 1)
	key2, _ := DeriveRotationKey(key, 2)
	key3, _ := DeriveRotationKey(key, 100)

	// All keys should be different
	if key1 == key2 {
		t.Error("Keys for sequence 1 and 2 should be different")
	}
	if key1 == key3 {
		t.Error("Keys for sequence 1 and 100 should be different")
	}
	if key2 == key3 {
		t.Error("Keys for sequence 2 and 100 should be different")
	}
}

// TestDeriveRotationKeySequenceZero tests sequence number 0
func TestDeriveRotationKeySequenceZero(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	key0, err := DeriveRotationKey(key, 0)
	if err != nil {
		t.Fatalf("Derivation with sequence 0 failed: %v", err)
	}

	key1, _ := DeriveRotationKey(key, 1)

	// Sequence 0 and 1 should produce different keys
	if key0 == key1 {
		t.Error("Sequence 0 and 1 should produce different keys")
	}
}

// TestDeriveRotationKeyHighSequence tests large sequence numbers
func TestDeriveRotationKeyHighSequence(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	// Test with large sequence numbers
	sequences := []uint64{
		1000000,
		18446744073709551615, // MaxUint64
	}

	for _, seq := range sequences {
		derivedKey, err := DeriveRotationKey(key, seq)
		if err != nil {
			t.Errorf("Derivation failed for sequence %d: %v", seq, err)
		}

		if derivedKey == key {
			t.Errorf("Derived key should differ from original for sequence %d", seq)
		}
	}
}

// TestDeriveMultipleKeys tests deriving a chain of keys
func TestDeriveMultipleKeys(t *testing.T) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	keys, err := DeriveMultipleKeys(initialKey, 1, 5)
	if err != nil {
		t.Fatalf("DeriveMultipleKeys failed: %v", err)
	}

	// Verify we got 5 keys
	if len(keys) != 5 {
		t.Fatalf("Expected 5 keys, got %d", len(keys))
	}

	// Verify all keys are unique
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] == keys[j] {
				t.Errorf("Keys %d and %d are identical", i, j)
			}
		}
	}
}

// TestDeriveMultipleKeysInvalidCount tests error handling for invalid count
func TestDeriveMultipleKeysInvalidCount(t *testing.T) {
	var key [32]byte

	testCases := []int{0, -1, -100}

	for _, count := range testCases {
		_, err := DeriveMultipleKeys(key, 0, count)
		if err == nil {
			t.Errorf("Expected error for count %d, got nil", count)
		}
	}
}

// TestVerifyKeyDerivation tests the verification function
func TestVerifyKeyDerivation(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	// Verify derivation is deterministic
	ok, err := VerifyKeyDerivation(key, 1)
	if err != nil {
		t.Fatalf("VerifyKeyDerivation failed: %v", err)
	}

	if !ok {
		t.Error("Key derivation should be deterministic")
	}
}

// TestDeriveRotationKeyAllZeroKey tests derivation with all-zero key
func TestDeriveRotationKeyAllZeroKey(t *testing.T) {
	var zeroKey [32]byte // All zeros

	derivedKey, err := DeriveRotationKey(zeroKey, 1)
	if err != nil {
		t.Fatalf("Derivation with zero key failed: %v", err)
	}

	// Derived key should not be all zeros
	allZero := true
	for _, b := range derivedKey {
		if b != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		t.Error("Derived key should not be all zeros even with zero input key")
	}
}

// TestDeriveRotationKeyMaxKey tests derivation with all 0xFF key
func TestDeriveRotationKeyMaxKey(t *testing.T) {
	var maxKey [32]byte
	for i := range maxKey {
		maxKey[i] = 0xFF
	}

	derivedKey, err := DeriveRotationKey(maxKey, 1)
	if err != nil {
		t.Fatalf("Derivation with max key failed: %v", err)
	}

	// Derived key should be different from input
	if derivedKey == maxKey {
		t.Error("Derived key should differ from input")
	}
}

// BenchmarkDeriveRotationKey benchmarks key derivation performance
func BenchmarkDeriveRotationKey(b *testing.B) {
	var key [32]byte
	rand.Read(key[:])

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := DeriveRotationKey(key, uint64(i))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDeriveMultipleKeys benchmarks chain derivation
func BenchmarkDeriveMultipleKeys(b *testing.B) {
	var key [32]byte
	rand.Read(key[:])

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := DeriveMultipleKeys(key, 0, 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVerifyKeyDerivation benchmarks verification
func BenchmarkVerifyKeyDerivation(b *testing.B) {
	var key [32]byte
	rand.Read(key[:])

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := VerifyKeyDerivation(key, uint64(i))
		if err != nil {
			b.Fatal(err)
		}
	}
}
