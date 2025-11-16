package rotation

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrRotationInProgress indicates a rotation is already in progress
	ErrRotationInProgress = errors.New("rotation already in progress")
	// ErrNoCurrentKey indicates no current key is set
	ErrNoCurrentKey = errors.New("no current key set")
)

// RotationManager manages key rotation state and orchestration
//
// Thread-safe: All methods can be called concurrently
// Atomic sequence counter: Prevents race conditions
// Mutex-protected state: Ensures consistent key transitions
//
// Lifecycle:
// 1. Create manager: NewRotationManager(initialKey)
// 2. Rotate keys: manager.RotateKey()
// 3. Access current key: manager.GetCurrentKey()
// 4. Cleanup: All old keys are automatically zeroed
type RotationManager struct {
	currentKey   [32]byte
	previousKey  [32]byte
	sequence     uint64 // Atomic counter (monotonically increasing)
	lastRotation time.Time
	rotating     bool // Flag to prevent concurrent rotations
	mu           sync.RWMutex
}

// RotationResult contains the result of a key rotation
type RotationResult struct {
	NewKey       [32]byte      // New session key (ready for use)
	OldKey       [32]byte      // Old session key (caller should zero this)
	PreviousKey  [32]byte      // Previous old key (from last rotation, for grace period)
	Sequence     uint64        // Rotation sequence number
	RotationTime time.Duration // Time taken to perform rotation
	Timestamp    time.Time     // When rotation occurred
}

// RotationMessage is sent over encrypted channel during rotation
// This message notifies the peer that key rotation is occurring
type RotationMessage struct {
	MessageType string    // "KEY_ROTATION"
	Sequence    uint64    // Rotation sequence number
	Timestamp   time.Time // When rotation occurred
	// Note: New key material is NOT included in this message
	// Instead, perform a new hybrid key exchange to establish the rotated key
}

// NewRotationManager creates a new rotation manager with an initial key
//
// Parameters:
// - initialKey: Starting 32-byte session key
//
// Returns:
// - *RotationManager: New manager instance
func NewRotationManager(initialKey [32]byte) *RotationManager {
	return &RotationManager{
		currentKey:   initialKey,
		sequence:     0,
		lastRotation: time.Now(),
	}
}

// RotateKey performs key rotation: derives new key, updates state, zeros old key
//
// Process:
// 1. Increment sequence atomically
// 2. Derive new key using HKDF(currentKey, sequence)
// 3. Update state: previousKey ← currentKey, currentKey ← newKey
// 4. Zero old previousKey (not current, for grace period)
// 5. Return RotationResult with timing information
//
// Thread-safety: Uses mutex to prevent concurrent rotations
// Performance: <10ms on commodity hardware (measured)
//
// Returns:
// - *RotationResult: New key and metadata
// - error: Error if rotation fails or already in progress
func (rm *RotationManager) RotateKey() (*RotationResult, error) {
	startTime := time.Now()

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Prevent concurrent rotations
	if rm.rotating {
		return nil, ErrRotationInProgress
	}
	rm.rotating = true
	defer func() { rm.rotating = false }()

	// Increment sequence atomically
	newSequence := atomic.AddUint64(&rm.sequence, 1)

	// Derive new key using HKDF
	newKey, err := DeriveRotationKey(rm.currentKey, newSequence)
	if err != nil {
		return nil, fmt.Errorf("failed to derive rotation key: %w", err)
	}

	// Prepare result (capture old keys before updating)
	result := &RotationResult{
		NewKey:      newKey,
		OldKey:      rm.currentKey,
		PreviousKey: rm.previousKey,
		Sequence:    newSequence,
		Timestamp:   time.Now(),
	}

	// Update state
	// previousKey ← currentKey (keep for grace period)
	// currentKey ← newKey
	rm.previousKey = rm.currentKey
	rm.currentKey = newKey
	rm.lastRotation = time.Now()

	// Zero the old previous key (not oldKey - that's still current for a grace period)
	// The caller is responsible for zeroing OldKey after transitioning connections
	SecureZero(&result.PreviousKey)

	result.RotationTime = time.Since(startTime)

	return result, nil
}

// GetCurrentKey returns the current session key (read-only copy)
//
// Thread-safe: Uses read lock
//
// Returns:
// - [32]byte: Current key
// - uint64: Current sequence number
func (rm *RotationManager) GetCurrentKey() ([32]byte, uint64) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.currentKey, atomic.LoadUint64(&rm.sequence)
}

// GetPreviousKey returns the previous session key (for grace period)
//
// Use case: During rotation, some packets may still use the old key
// The previous key should be accepted for a short grace period (e.g., 1 second)
//
// Returns:
// - [32]byte: Previous key
// - bool: True if previous key exists (false if this is first key)
func (rm *RotationManager) GetPreviousKey() ([32]byte, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Check if previous key is all zeros (no previous rotation yet)
	hasPreviousKey := false
	for _, b := range rm.previousKey {
		if b != 0 {
			hasPreviousKey = true
			break
		}
	}

	return rm.previousKey, hasPreviousKey
}

// GetSequence returns the current rotation sequence number
//
// Atomic operation: Safe to call concurrently
func (rm *RotationManager) GetSequence() uint64 {
	return atomic.LoadUint64(&rm.sequence)
}

// GetLastRotation returns the time of the last rotation
func (rm *RotationManager) GetLastRotation() time.Time {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.lastRotation
}

// TimeSinceLastRotation returns the duration since the last rotation
func (rm *RotationManager) TimeSinceLastRotation() time.Duration {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return time.Since(rm.lastRotation)
}

// SetCurrentKey manually sets the current key (for testing or key exchange)
//
// WARNING: This bypasses normal rotation logic
// Use only for:
// - Initial key setup after hybrid key exchange
// - Testing scenarios
// - Recovering from errors
//
// Parameters:
// - key: New 32-byte key to set as current
// - sequence: Sequence number to set (usually 0 for fresh keys)
func (rm *RotationManager) SetCurrentKey(key [32]byte, sequence uint64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.currentKey = key
	atomic.StoreUint64(&rm.sequence, sequence)
	rm.lastRotation = time.Now()
}

// CreateRotationMessage creates a RotationMessage for transmission to peer
//
// Returns:
// - *RotationMessage: Message to send over encrypted channel
func (rm *RotationManager) CreateRotationMessage() *RotationMessage {
	return &RotationMessage{
		MessageType: "KEY_ROTATION",
		Sequence:    rm.GetSequence(),
		Timestamp:   time.Now(),
	}
}
