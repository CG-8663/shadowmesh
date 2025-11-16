package rotation

import (
	"context"
	"crypto/rand"
	"sync/atomic"
	"testing"
	"time"
)

// TestIntegrationKeyRotation tests a complete 15-minute rotation scenario
// Uses 1 second = 1 minute time scale for fast testing
// Simulates 3 rotations over 15 minutes (scaled to 15 seconds)
func TestIntegrationKeyRotation(t *testing.T) {
	// Time scale: 1 second = 1 minute
	// 5 minutes rotation interval = 5 seconds
	rotationInterval := 5 * time.Second

	// Initialize
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)

	rotationCount := uint32(0)
	var lastRotationTime time.Time

	// Create timer with callback that rotates keys
	timer := NewRotationTimer(rotationInterval, func() {
		result, err := rm.RotateKey()
		if err != nil {
			t.Errorf("Rotation failed: %v", err)
			return
		}

		count := atomic.AddUint32(&rotationCount, 1)
		t.Logf("Rotation %d completed in %v at sequence %d",
			count, result.RotationTime, result.Sequence)

		// Verify rotation latency < 100ms (AC 1.5.5)
		if result.RotationTime > 100*time.Millisecond {
			t.Errorf("Rotation %d took %v, expected < 100ms",
				count, result.RotationTime)
		}

		// Zero old key after grace period (simulated)
		defer SecureZero(&result.OldKey)

		lastRotationTime = result.Timestamp
	})

	// Start timer
	ctx, cancel := context.WithTimeout(context.Background(), 16*time.Second)
	defer cancel()

	timer.Start(ctx)

	// Wait for rotations to complete (15 seconds = 15 simulated minutes)
	time.Sleep(16 * time.Second)
	timer.Stop()

	// Verify 3 rotations occurred (AC 1.5.7)
	finalCount := atomic.LoadUint32(&rotationCount)
	if finalCount != 3 {
		t.Errorf("Expected 3 rotations, got %d", finalCount)
	}

	// Verify final sequence
	if rm.GetSequence() != 3 {
		t.Errorf("Expected sequence 3, got %d", rm.GetSequence())
	}

	// Verify last rotation was recent
	timeSinceLast := time.Since(lastRotationTime)
	if timeSinceLast > 10*time.Second {
		t.Errorf("Last rotation was %v ago, expected < 10s", timeSinceLast)
	}

	// Calculate total rotation overhead
	// Each rotation should be < 100ms, total < 300ms for 3 rotations
	t.Logf("Integration test completed: %d rotations over 15 simulated minutes", finalCount)
}

// TestIntegrationManualRotations tests manual rotation sequences
func TestIntegrationManualRotations(t *testing.T) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)

	// Perform 5 manual rotations
	keys := make([][32]byte, 5)
	totalLatency := time.Duration(0)

	for i := 0; i < 5; i++ {
		result, err := rm.RotateKey()
		if err != nil {
			t.Fatalf("Rotation %d failed: %v", i+1, err)
		}

		keys[i] = result.NewKey
		totalLatency += result.RotationTime

		// Verify old key is different from new key
		if result.OldKey == result.NewKey {
			t.Errorf("Rotation %d: old and new keys should differ", i+1)
		}

		// Zero old key
		SecureZero(&result.OldKey)

		// Short delay between rotations
		time.Sleep(10 * time.Millisecond)
	}

	// Verify all keys are unique
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] == keys[j] {
				t.Errorf("Keys %d and %d are identical", i, j)
			}
		}
	}

	// Verify average rotation latency
	avgLatency := totalLatency / 5
	t.Logf("Average rotation latency: %v", avgLatency)

	if avgLatency > 10*time.Millisecond {
		t.Logf("Warning: Average latency %v is higher than expected (<10ms)", avgLatency)
	}
}

// TestIntegrationKeyZeroing tests that old keys are properly zeroed
func TestIntegrationKeyZeroing(t *testing.T) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)

	// Keep references to old keys for verification
	oldKeys := make([]*[32]byte, 3)

	// Perform 3 rotations
	for i := 0; i < 3; i++ {
		result, err := rm.RotateKey()
		if err != nil {
			t.Fatalf("Rotation %d failed: %v", i+1, err)
		}

		// Save reference to old key
		oldKey := result.OldKey
		oldKeys[i] = &oldKey

		// Zero it
		SecureZero(oldKeys[i])
	}

	// Verify all old keys are zeroed
	for i, key := range oldKeys {
		if !VerifyZeroed(key) {
			t.Errorf("Old key %d was not properly zeroed", i)
		}
	}
}

// TestIntegrationTimerAndManager tests timer + manager integration
func TestIntegrationTimerAndManager(t *testing.T) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)
	rotationCount := uint32(0)

	// Create timer that rotates every 100ms
	timer := NewRotationTimer(100*time.Millisecond, func() {
		_, err := rm.RotateKey()
		if err != nil {
			t.Errorf("Timer-triggered rotation failed: %v", err)
		}
		atomic.AddUint32(&rotationCount, 1)
	})

	// Run for 550ms (should get 5 rotations)
	ctx, cancel := context.WithTimeout(context.Background(), 550*time.Millisecond)
	defer cancel()

	timer.Start(ctx)
	time.Sleep(600 * time.Millisecond)
	timer.Stop()

	// Verify rotations occurred
	finalCount := atomic.LoadUint32(&rotationCount)
	if finalCount < 4 || finalCount > 6 {
		t.Errorf("Expected 4-6 rotations, got %d", finalCount)
	}

	t.Logf("Timer-triggered %d rotations in 550ms", finalCount)
}

// BenchmarkIntegrationFullRotation benchmarks complete rotation with zeroing
func BenchmarkIntegrationFullRotation(b *testing.B) {
	var initialKey [32]byte
	rand.Read(initialKey[:])

	rm := NewRotationManager(initialKey)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, err := rm.RotateKey()
		if err != nil {
			b.Fatal(err)
		}

		// Zero old key (as would happen in production)
		SecureZero(&result.OldKey)
	}
}
