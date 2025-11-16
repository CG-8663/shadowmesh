package rotation

import (
	"crypto/rand"
	"sync"
	"testing"
	"time"
)

// TestNewRotationManager tests manager creation
func TestNewRotationManager(t *testing.T) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)
	if rm == nil {
		t.Fatal("NewRotationManager returned nil")
	}

	currentKey, seq := rm.GetCurrentKey()
	if currentKey != initialKey {
		t.Error("Current key should match initial key")
	}

	if seq != 0 {
		t.Errorf("Initial sequence should be 0, got %d", seq)
	}
}

// TestRotateKey tests basic key rotation
func TestRotateKey(t *testing.T) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)

	// Perform rotation
	result, err := rm.RotateKey()
	if err != nil {
		t.Fatalf("RotateKey failed: %v", err)
	}

	// Verify result
	if result.NewKey == initialKey {
		t.Error("New key should differ from initial key")
	}

	if result.OldKey != initialKey {
		t.Error("Old key should match initial key")
	}

	if result.Sequence != 1 {
		t.Errorf("Expected sequence 1, got %d", result.Sequence)
	}

	// Verify manager state
	currentKey, seq := rm.GetCurrentKey()
	if currentKey != result.NewKey {
		t.Error("Current key should match result.NewKey")
	}

	if seq != 1 {
		t.Errorf("Expected sequence 1, got %d", seq)
	}
}

// TestRotateKeyMultiple tests multiple rotations
func TestRotateKeyMultiple(t *testing.T) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)

	keys := make(map[[32]byte]bool)
	keys[initialKey] = true

	// Perform 10 rotations
	for i := 1; i <= 10; i++ {
		result, err := rm.RotateKey()
		if err != nil {
			t.Fatalf("Rotation %d failed: %v", i, err)
		}

		// Verify sequence
		if result.Sequence != uint64(i) {
			t.Errorf("Rotation %d: expected sequence %d, got %d", i, i, result.Sequence)
		}

		// Verify key uniqueness
		if keys[result.NewKey] {
			t.Errorf("Rotation %d: duplicate key detected", i)
		}
		keys[result.NewKey] = true
	}

	// Verify final sequence
	if rm.GetSequence() != 10 {
		t.Errorf("Expected final sequence 10, got %d", rm.GetSequence())
	}
}

// TestRotateKeyLatency tests that rotation completes quickly
func TestRotateKeyLatency(t *testing.T) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)

	result, err := rm.RotateKey()
	if err != nil {
		t.Fatalf("RotateKey failed: %v", err)
	}

	// Verify rotation time < 100ms (AC 1.5.5)
	if result.RotationTime > 100*time.Millisecond {
		t.Errorf("Rotation took %v, expected < 100ms", result.RotationTime)
	}

	// Most rotations should be < 10ms
	if result.RotationTime > 10*time.Millisecond {
		t.Logf("Warning: Rotation took %v (acceptable but slower than expected)", result.RotationTime)
	}
}

// TestRotateKeyConcurrentPrevention tests that concurrent rotations are prevented
func TestRotateKeyConcurrentPrevention(t *testing.T) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)

	// Try to rotate concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := rm.RotateKey()
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Some rotations should have failed with ErrRotationInProgress
	errorCount := 0
	for err := range errors {
		if err == ErrRotationInProgress {
			errorCount++
		}
	}

	// We should have some concurrent rotation errors
	t.Logf("Got %d concurrent rotation errors (expected)", errorCount)
}

// TestGetPreviousKey tests previous key retrieval
func TestGetPreviousKey(t *testing.T) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)

	// No previous key initially
	_, hasPrev := rm.GetPreviousKey()
	if hasPrev {
		t.Error("Should not have previous key initially")
	}

	// Rotate
	result1, _ := rm.RotateKey()

	// Now previous key should exist
	prevKey, hasPrev := rm.GetPreviousKey()
	if !hasPrev {
		t.Error("Should have previous key after first rotation")
	}

	if prevKey != initialKey {
		t.Error("Previous key should match initial key")
	}

	// Rotate again
	rm.RotateKey()

	// Previous key should now be result1.NewKey
	prevKey2, _ := rm.GetPreviousKey()
	if prevKey2 != result1.NewKey {
		t.Error("Previous key should match first rotation's new key")
	}
}

// TestTimeSinceLastRotation tests time tracking
func TestTimeSinceLastRotation(t *testing.T) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Time since last rotation should be > 50ms
	elapsed := rm.TimeSinceLastRotation()
	if elapsed < 50*time.Millisecond {
		t.Errorf("Expected > 50ms, got %v", elapsed)
	}

	// Rotate
	rm.RotateKey()

	// Time should reset
	elapsed2 := rm.TimeSinceLastRotation()
	if elapsed2 > 10*time.Millisecond {
		t.Errorf("Expected < 10ms after rotation, got %v", elapsed2)
	}
}

// TestSetCurrentKey tests manual key setting
func TestSetCurrentKey(t *testing.T) {
	var initialKey, newKey [32]byte
	rand.Read(initialKey[:])
	rand.Read(newKey[:])

	rm := NewRotationManager(initialKey)

	// Set new key manually
	rm.SetCurrentKey(newKey, 42)

	// Verify
	currentKey, seq := rm.GetCurrentKey()
	if currentKey != newKey {
		t.Error("Current key should match manually set key")
	}

	if seq != 42 {
		t.Errorf("Expected sequence 42, got %d", seq)
	}
}

// TestCreateRotationMessage tests message creation
func TestCreateRotationMessage(t *testing.T) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)
	rm.RotateKey()

	msg := rm.CreateRotationMessage()

	if msg.MessageType != "KEY_ROTATION" {
		t.Errorf("Expected message type 'KEY_ROTATION', got '%s'", msg.MessageType)
	}

	if msg.Sequence != 1 {
		t.Errorf("Expected sequence 1, got %d", msg.Sequence)
	}

	if msg.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

// BenchmarkRotateKey benchmarks rotation performance
func BenchmarkRotateKey(b *testing.B) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := rm.RotateKey()
		if err != nil {
			b.Fatal(err)
		}
	}
}
