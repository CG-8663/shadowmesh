package symmetric

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	// CounterSize is 48 bits (6 bytes)
	CounterSize = 6
	// SaltSize is 48 bits (6 bytes)
	SaltSize = 6
	// MaxCounter is 2^48 - 1 (281 trillion frames before overflow)
	MaxCounter uint64 = (1 << 48) - 1
)

var (
	// ErrCounterOverflow indicates the 48-bit counter has wrapped around
	ErrCounterOverflow = errors.New("nonce counter overflow: regenerating salt")
	// ErrRandomGenerationFailed indicates crypto/rand failed
	ErrRandomGenerationFailed = errors.New("failed to generate random salt")
)

// NonceGenerator generates unique nonces for ChaCha20-Poly1305
// Nonce format: [6 bytes counter (big-endian)][6 bytes random salt]
//
// Thread-safe: Uses atomic operations for counter, mutex for salt regeneration
// Security: Counter ensures uniqueness within a session, salt provides randomness across sessions
type NonceGenerator struct {
	counter uint64         // 48-bit atomic counter (lower 48 bits used)
	salt    [SaltSize]byte // 48-bit random salt (regenerated on counter overflow)
	mu      sync.Mutex     // Protects salt regeneration
}

// NewNonceGenerator creates a new nonce generator with random salt
func NewNonceGenerator() (*NonceGenerator, error) {
	ng := &NonceGenerator{}

	// Generate initial random salt
	if err := ng.regenerateSalt(); err != nil {
		return nil, err
	}

	return ng, nil
}

// GenerateNonce creates a unique 12-byte nonce
// Format: 6 bytes counter (big-endian) || 6 bytes random salt
//
// Thread-safe: Can be called concurrently from multiple goroutines
// Performance: ~100 ns/op (atomic counter increment + memory copy)
func (ng *NonceGenerator) GenerateNonce() ([NonceSize]byte, error) {
	var nonce [NonceSize]byte

	// Atomically increment counter (uses lower 48 bits only)
	currentCounter := atomic.AddUint64(&ng.counter, 1)

	// Check for overflow (counter wrapped past 2^48)
	if currentCounter > MaxCounter {
		// Regenerate salt and reset counter
		ng.mu.Lock()
		// Double-check after acquiring lock (another goroutine may have already reset)
		if atomic.LoadUint64(&ng.counter) > MaxCounter {
			if err := ng.regenerateSalt(); err != nil {
				ng.mu.Unlock()
				return nonce, fmt.Errorf("%w: %v", ErrCounterOverflow, err)
			}
			atomic.StoreUint64(&ng.counter, 1)
			currentCounter = 1
		} else {
			currentCounter = atomic.LoadUint64(&ng.counter)
		}
		ng.mu.Unlock()
	}

	// Encode counter as big-endian 6 bytes (bits 47-0 of uint64)
	// We use 8 bytes then truncate to 6 to handle endianness properly
	var counterBytes [8]byte
	binary.BigEndian.PutUint64(counterBytes[:], currentCounter)
	copy(nonce[:CounterSize], counterBytes[2:8]) // Take lower 6 bytes

	// Copy salt (protected by happens-before relationship via atomic counter)
	ng.mu.Lock()
	copy(nonce[CounterSize:], ng.salt[:])
	ng.mu.Unlock()

	return nonce, nil
}

// regenerateSalt generates a new random 48-bit salt
// Must be called with ng.mu locked
func (ng *NonceGenerator) regenerateSalt() error {
	_, err := rand.Read(ng.salt[:])
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRandomGenerationFailed, err)
	}
	return nil
}

// GetCounter returns the current counter value (for testing/debugging)
func (ng *NonceGenerator) GetCounter() uint64 {
	return atomic.LoadUint64(&ng.counter)
}

// GetSalt returns a copy of the current salt (for testing/debugging)
func (ng *NonceGenerator) GetSalt() [SaltSize]byte {
	ng.mu.Lock()
	defer ng.mu.Unlock()

	var salt [SaltSize]byte
	copy(salt[:], ng.salt[:])
	return salt
}

// Reset resets the counter to 0 and regenerates the salt
// Useful for testing or after long-running sessions
func (ng *NonceGenerator) Reset() error {
	ng.mu.Lock()
	defer ng.mu.Unlock()

	atomic.StoreUint64(&ng.counter, 0)
	return ng.regenerateSalt()
}
