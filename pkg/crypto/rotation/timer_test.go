package rotation

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewRotationTimer tests timer creation
func TestNewRotationTimer(t *testing.T) {
	callCount := uint32(0)
	callback := func() {
		atomic.AddUint32(&callCount, 1)
	}

	timer := NewRotationTimer(100*time.Millisecond, callback)
	if timer == nil {
		t.Fatal("NewRotationTimer returned nil")
	}

	if timer.GetInterval() != 100*time.Millisecond {
		t.Errorf("Expected interval 100ms, got %v", timer.GetInterval())
	}
}

// TestRotationTimerStartStop tests starting and stopping
func TestRotationTimerStartStop(t *testing.T) {
	callCount := uint32(0)
	callback := func() {
		atomic.AddUint32(&callCount, 1)
	}

	timer := NewRotationTimer(50*time.Millisecond, callback)
	ctx := context.Background()

	// Start timer
	timer.Start(ctx)
	if !timer.IsRunning() {
		t.Error("Timer should be running after Start()")
	}

	// Wait for a few ticks
	time.Sleep(150 * time.Millisecond)

	// Stop timer
	timer.Stop()
	if timer.IsRunning() {
		t.Error("Timer should not be running after Stop()")
	}

	// Verify callback was called at least twice
	count := atomic.LoadUint32(&callCount)
	if count < 2 {
		t.Errorf("Expected at least 2 callbacks, got %d", count)
	}
}

// TestRotationTimerContextCancellation tests context cancellation
func TestRotationTimerContextCancellation(t *testing.T) {
	callCount := uint32(0)
	callback := func() {
		atomic.AddUint32(&callCount, 1)
	}

	timer := NewRotationTimer(50*time.Millisecond, callback)
	ctx, cancel := context.WithCancel(context.Background())

	timer.Start(ctx)
	time.Sleep(150 * time.Millisecond)

	// Cancel context
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Timer should have stopped
	if timer.IsRunning() {
		t.Error("Timer should stop when context is cancelled")
	}
}

// TestRotationTimerReset tests resetting the timer
func TestRotationTimerReset(t *testing.T) {
	callCount := uint32(0)
	callback := func() {
		atomic.AddUint32(&callCount, 1)
	}

	timer := NewRotationTimer(100*time.Millisecond, callback)
	ctx := context.Background()

	timer.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	// Reset with shorter interval
	timer.Reset(30*time.Millisecond, ctx)

	if timer.GetInterval() != 30*time.Millisecond {
		t.Errorf("Expected interval 30ms after reset, got %v", timer.GetInterval())
	}

	time.Sleep(100 * time.Millisecond)
	timer.Stop()

	// Should have been called with new interval
	count := atomic.LoadUint32(&callCount)
	if count < 2 {
		t.Errorf("Expected at least 2 calls after reset, got %d", count)
	}
}

// TestRotationTimerMultipleStartStop tests multiple start/stop cycles
func TestRotationTimerMultipleStartStop(t *testing.T) {
	callback := func() {}
	timer := NewRotationTimer(50*time.Millisecond, callback)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		timer.Start(ctx)
		if !timer.IsRunning() {
			t.Errorf("Timer should be running after Start() cycle %d", i)
		}

		time.Sleep(30 * time.Millisecond)

		timer.Stop()
		if timer.IsRunning() {
			t.Errorf("Timer should not be running after Stop() cycle %d", i)
		}
	}
}

// TestRotationTimerDoubleStart tests starting an already running timer
func TestRotationTimerDoubleStart(t *testing.T) {
	callback := func() {}
	timer := NewRotationTimer(50*time.Millisecond, callback)
	ctx := context.Background()

	timer.Start(ctx)
	timer.Start(ctx) // Should be no-op

	if !timer.IsRunning() {
		t.Error("Timer should still be running")
	}

	timer.Stop()
}

// TestRotationTimerDoubleStop tests stopping an already stopped timer
func TestRotationTimerDoubleStop(t *testing.T) {
	callback := func() {}
	timer := NewRotationTimer(50*time.Millisecond, callback)

	timer.Stop() // Should be no-op
	timer.Stop() // Should be no-op

	if timer.IsRunning() {
		t.Error("Timer should not be running")
	}
}
