package rotation

import (
	"crypto/rand"
	"fmt"
	"testing"
)

// TestSecureZero tests basic key zeroing
func TestSecureZero(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	// Verify key is not zero initially
	if VerifyZeroed(&key) {
		t.Fatal("Key should not be zeroed initially")
	}

	// Zero the key
	SecureZero(&key)

	// Verify key is now zeroed
	if !VerifyZeroed(&key) {
		t.Error("Key should be zeroed after SecureZero()")
	}
}

// TestSecureZeroNil tests nil pointer handling
func TestSecureZeroNil(t *testing.T) {
	// Should not panic
	SecureZero(nil)
}

// TestSecureZeroMultipleTimes tests that zeroing multiple times is safe
func TestSecureZeroMultipleTimes(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	SecureZero(&key)
	SecureZero(&key)
	SecureZero(&key)

	if !VerifyZeroed(&key) {
		t.Error("Key should remain zeroed after multiple SecureZero() calls")
	}
}

// TestZeroSlice tests slice zeroing
func TestZeroSlice(t *testing.T) {
	sizes := []int{0, 1, 32, 100, 1024, 10000}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			data := make([]byte, size)
			if size > 0 {
				rand.Read(data)
			}

			ZeroSlice(data)

			if !VerifySliceZeroed(data) {
				t.Errorf("Slice of size %d should be zeroed", size)
			}
		})
	}
}

// TestZeroSliceNil tests nil slice handling
func TestZeroSliceNil(t *testing.T) {
	// Should not panic
	ZeroSlice(nil)
}

// TestZeroSliceEmpty tests empty slice handling
func TestZeroSliceEmpty(t *testing.T) {
	empty := []byte{}
	ZeroSlice(empty)
	// Should not panic and should still be empty
	if len(empty) != 0 {
		t.Error("Empty slice should remain empty")
	}
}

// TestVerifyZeroedAllPatterns tests verification with various patterns
func TestVerifyZeroedAllPatterns(t *testing.T) {
	testCases := []struct {
		name     string
		key      [32]byte
		expected bool
	}{
		{"all zeros", [32]byte{}, true},
		{"one non-zero at start", [32]byte{1, 0, 0}, false},
		{"one non-zero at end", [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, false},
		{"one non-zero in middle", [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := VerifyZeroed(&tc.key)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestVerifyZeroedNil tests nil verification
func TestVerifyZeroedNil(t *testing.T) {
	if VerifyZeroed(nil) {
		t.Error("Nil pointer should return false")
	}
}

// TestSecureZeroMultiple tests zeroing multiple keys at once
func TestSecureZeroMultiple(t *testing.T) {
	var key1, key2, key3 [32]byte
	rand.Read(key1[:])
	rand.Read(key2[:])
	rand.Read(key3[:])

	SecureZeroMultiple(&key1, &key2, &key3)

	if !VerifyZeroed(&key1) || !VerifyZeroed(&key2) || !VerifyZeroed(&key3) {
		t.Error("All keys should be zeroed after SecureZeroMultiple()")
	}
}

// TestSecureZeroMultipleWithNil tests multiple zeroing with nil keys
func TestSecureZeroMultipleWithNil(t *testing.T) {
	var key1, key2 [32]byte
	rand.Read(key1[:])
	rand.Read(key2[:])

	// Should not panic even with nil in the middle
	SecureZeroMultiple(&key1, nil, &key2)

	if !VerifyZeroed(&key1) || !VerifyZeroed(&key2) {
		t.Error("Non-nil keys should be zeroed")
	}
}

// TestZeroingDoesNotAffectOtherData tests isolation
func TestZeroingDoesNotAffectOtherData(t *testing.T) {
	var key1, key2 [32]byte
	rand.Read(key1[:])
	rand.Read(key2[:])

	// Save key2 for comparison
	key2Original := key2

	// Zero key1
	SecureZero(&key1)

	// key2 should be unchanged
	if key2 != key2Original {
		t.Error("Zeroing key1 should not affect key2")
	}

	// key1 should be zeroed
	if !VerifyZeroed(&key1) {
		t.Error("key1 should be zeroed")
	}
}

// BenchmarkSecureZero benchmarks key zeroing performance
func BenchmarkSecureZero(b *testing.B) {
	var key [32]byte
	rand.Read(key[:])

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		SecureZero(&key)
	}
}

// BenchmarkZeroSlice1KB benchmarks slice zeroing (1 KB)
func BenchmarkZeroSlice1KB(b *testing.B) {
	data := make([]byte, 1024)
	rand.Read(data)

	b.ResetTimer()
	b.SetBytes(1024)

	for i := 0; i < b.N; i++ {
		ZeroSlice(data)
	}
}

// BenchmarkZeroSlice1MB benchmarks slice zeroing (1 MB)
func BenchmarkZeroSlice1MB(b *testing.B) {
	data := make([]byte, 1024*1024)
	rand.Read(data)

	b.ResetTimer()
	b.SetBytes(1024 * 1024)

	for i := 0; i < b.N; i++ {
		ZeroSlice(data)
	}
}

// BenchmarkSecureZeroMultiple benchmarks multiple key zeroing
func BenchmarkSecureZeroMultiple(b *testing.B) {
	var key1, key2, key3, key4, key5 [32]byte

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		SecureZeroMultiple(&key1, &key2, &key3, &key4, &key5)
	}
}
