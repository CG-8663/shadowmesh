package rotation

import (
	"context"
	"sync"
	"time"
)

// RotationTimer triggers periodic key rotation at configured intervals
//
// Thread-safe: Can be started/stopped from multiple goroutines
// Graceful shutdown: Supports context cancellation
//
// Usage:
//
//	timer := NewRotationTimer(5*time.Minute, func() {
//	    log.Println("Rotating keys...")
//	    // Perform key rotation
//	})
//	timer.Start(context.Background())
//	defer timer.Stop()
type RotationTimer struct {
	interval time.Duration
	callback func()
	ticker   *time.Ticker
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// NewRotationTimer creates a new rotation timer
//
// Parameters:
// - interval: Time between rotations (e.g., 5*time.Minute)
// - callback: Function to call on each rotation trigger
//
// Returns:
// - *RotationTimer: New timer instance (not started)
func NewRotationTimer(interval time.Duration, callback func()) *RotationTimer {
	return &RotationTimer{
		interval: interval,
		callback: callback,
		stopChan: make(chan struct{}),
	}
}

// Start begins the rotation timer
//
// Parameters:
// - ctx: Context for cancellation (timer stops when context is cancelled)
//
// Behavior:
// - Starts a goroutine that ticks at the configured interval
// - Calls callback function on each tick
// - Returns immediately (non-blocking)
// - If already running, this is a no-op
//
// Usage:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	timer.Start(ctx)
//	// ... later ...
//	cancel() // Stops the timer
func (rt *RotationTimer) Start(ctx context.Context) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if rt.running {
		return // Already running
	}

	rt.ticker = time.NewTicker(rt.interval)
	rt.running = true

	rt.wg.Add(1)
	go rt.run(ctx)
}

// run is the main timer loop (internal goroutine)
func (rt *RotationTimer) run(ctx context.Context) {
	defer rt.wg.Done()
	defer func() {
		// Mark as not running when goroutine exits
		rt.mu.Lock()
		rt.running = false
		rt.mu.Unlock()
	}()

	for {
		select {
		case <-rt.ticker.C:
			// Timer tick - call callback
			if rt.callback != nil {
				rt.callback()
			}

		case <-rt.stopChan:
			// Explicit stop requested
			return

		case <-ctx.Done():
			// Context cancelled
			return
		}
	}
}

// Stop stops the rotation timer
//
// Behavior:
// - Stops the ticker
// - Waits for goroutine to finish
// - Safe to call multiple times
// - Blocks until timer fully stopped
func (rt *RotationTimer) Stop() {
	rt.mu.Lock()

	if !rt.running {
		rt.mu.Unlock()
		return // Not running
	}

	// Stop ticker
	if rt.ticker != nil {
		rt.ticker.Stop()
	}

	// Signal stop
	close(rt.stopChan)
	rt.running = false

	rt.mu.Unlock()

	// Wait for goroutine to finish
	rt.wg.Wait()

	// Recreate stop channel for future Start() calls
	rt.mu.Lock()
	rt.stopChan = make(chan struct{})
	rt.mu.Unlock()
}

// Reset stops and restarts the timer with a new interval
//
// Parameters:
// - newInterval: New time between rotations
// - ctx: Context for the restarted timer
//
// Behavior:
// - Stops current timer if running
// - Updates interval
// - Starts timer with new interval
func (rt *RotationTimer) Reset(newInterval time.Duration, ctx context.Context) {
	rt.Stop()

	rt.mu.Lock()
	rt.interval = newInterval
	rt.mu.Unlock()

	rt.Start(ctx)
}

// GetInterval returns the current rotation interval
func (rt *RotationTimer) GetInterval() time.Duration {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.interval
}

// IsRunning returns true if the timer is currently running
func (rt *RotationTimer) IsRunning() bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.running
}
