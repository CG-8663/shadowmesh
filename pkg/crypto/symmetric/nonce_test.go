package symmetric

import (
	"sync"
	"testing"
)

// TestNewNonceGenerator tests nonce generator initialization
func TestNewNonceGenerator(t *testing.T) {
	ng, err := NewNonceGenerator()
	if err != nil {
		t.Fatalf("NewNonceGenerator failed: %v", err)
	}

	if ng == nil {
		t.Fatal("NewNonceGenerator returned nil")
	}

	// Verify counter starts at 0
	if ng.GetCounter() != 0 {
		t.Errorf("Initial counter should be 0, got %d", ng.GetCounter())
	}

	// Verify salt is non-zero (random)
	salt := ng.GetSalt()
	allZero := true
	for _, b := range salt {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("Salt should not be all zeros")
	}
}

// TestGenerateNonce tests basic nonce generation
func TestGenerateNonce(t *testing.T) {
	ng, err := NewNonceGenerator()
	if err != nil {
		t.Fatalf("NewNonceGenerator failed: %v", err)
	}

	nonce, err := ng.GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce failed: %v", err)
	}

	// Verify nonce length
	if len(nonce) != NonceSize {
		t.Errorf("Nonce size should be %d bytes, got %d", NonceSize, len(nonce))
	}

	// Verify counter incremented
	if ng.GetCounter() != 1 {
		t.Errorf("Counter should be 1 after first nonce, got %d", ng.GetCounter())
	}
}

// TestNonceUniqueness tests that nonces are unique
func TestNonceUniqueness(t *testing.T) {
	ng, err := NewNonceGenerator()
	if err != nil {
		t.Fatalf("NewNonceGenerator failed: %v", err)
	}

	const numNonces = 10000
	nonces := make(map[[NonceSize]byte]bool, numNonces)

	for i := 0; i < numNonces; i++ {
		nonce, err := ng.GenerateNonce()
		if err != nil {
			t.Fatalf("GenerateNonce failed at iteration %d: %v", i, err)
		}

		// Check for duplicates
		if nonces[nonce] {
			t.Fatalf("Duplicate nonce found at iteration %d: %x", i, nonce)
		}
		nonces[nonce] = true
	}

	// Verify counter matches number of nonces generated
	if ng.GetCounter() != numNonces {
		t.Errorf("Counter should be %d, got %d", numNonces, ng.GetCounter())
	}
}

// TestConcurrentNonceGeneration tests thread-safety
func TestConcurrentNonceGeneration(t *testing.T) {
	ng, err := NewNonceGenerator()
	if err != nil {
		t.Fatalf("NewNonceGenerator failed: %v", err)
	}

	const numGoroutines = 100
	const noncesPerGoroutine = 1000
	const totalNonces = numGoroutines * noncesPerGoroutine

	var wg sync.WaitGroup
	nonceChan := make(chan [NonceSize]byte, totalNonces)

	// Spawn goroutines to generate nonces concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < noncesPerGoroutine; j++ {
				nonce, err := ng.GenerateNonce()
				if err != nil {
					t.Errorf("GenerateNonce failed: %v", err)
					return
				}
				nonceChan <- nonce
			}
		}()
	}

	wg.Wait()
	close(nonceChan)

	// Collect and verify uniqueness
	nonces := make(map[[NonceSize]byte]bool, totalNonces)
	for nonce := range nonceChan {
		if nonces[nonce] {
			t.Fatalf("Duplicate nonce found in concurrent test: %x", nonce)
		}
		nonces[nonce] = true
	}

	// Verify we got all nonces
	if len(nonces) != totalNonces {
		t.Errorf("Expected %d unique nonces, got %d", totalNonces, len(nonces))
	}

	// Verify counter matches
	if ng.GetCounter() != totalNonces {
		t.Errorf("Counter should be %d, got %d", totalNonces, ng.GetCounter())
	}
}

// TestNonceFormat tests nonce structure (counter || salt)
func TestNonceFormat(t *testing.T) {
	ng, err := NewNonceGenerator()
	if err != nil {
		t.Fatalf("NewNonceGenerator failed: %v", err)
	}

	salt := ng.GetSalt()

	// Generate two nonces
	nonce1, _ := ng.GenerateNonce()
	nonce2, _ := ng.GenerateNonce()

	// Verify salt portion is the same (last 6 bytes)
	for i := 0; i < SaltSize; i++ {
		if nonce1[CounterSize+i] != salt[i] {
			t.Errorf("Nonce1 salt mismatch at byte %d: got %x, want %x", i, nonce1[CounterSize+i], salt[i])
		}
		if nonce2[CounterSize+i] != salt[i] {
			t.Errorf("Nonce2 salt mismatch at byte %d: got %x, want %x", i, nonce2[CounterSize+i], salt[i])
		}
		if nonce1[CounterSize+i] != nonce2[CounterSize+i] {
			t.Errorf("Salt mismatch between nonce1 and nonce2 at byte %d", i)
		}
	}

	// Verify counter portion is different (first 6 bytes)
	counterSame := true
	for i := 0; i < CounterSize; i++ {
		if nonce1[i] != nonce2[i] {
			counterSame = false
			break
		}
	}
	if counterSame {
		t.Error("Counter portion should be different between sequential nonces")
	}
}

// TestCounterIncrement tests that counter increments correctly
func TestCounterIncrement(t *testing.T) {
	ng, err := NewNonceGenerator()
	if err != nil {
		t.Fatalf("NewNonceGenerator failed: %v", err)
	}

	for i := 1; i <= 100; i++ {
		_, err := ng.GenerateNonce()
		if err != nil {
			t.Fatalf("GenerateNonce failed at iteration %d: %v", i, err)
		}

		if ng.GetCounter() != uint64(i) {
			t.Errorf("Counter should be %d after %d nonces, got %d", i, i, ng.GetCounter())
		}
	}
}

// TestReset tests nonce generator reset
func TestReset(t *testing.T) {
	ng, err := NewNonceGenerator()
	if err != nil {
		t.Fatalf("NewNonceGenerator failed: %v", err)
	}

	// Generate some nonces
	for i := 0; i < 100; i++ {
		_, err := ng.GenerateNonce()
		if err != nil {
			t.Fatalf("GenerateNonce failed: %v", err)
		}
	}

	initialSalt := ng.GetSalt()

	// Reset
	err = ng.Reset()
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Verify counter is reset
	if ng.GetCounter() != 0 {
		t.Errorf("Counter should be 0 after reset, got %d", ng.GetCounter())
	}

	// Verify salt changed
	newSalt := ng.GetSalt()
	saltChanged := false
	for i := 0; i < SaltSize; i++ {
		if initialSalt[i] != newSalt[i] {
			saltChanged = true
			break
		}
	}
	if !saltChanged {
		t.Error("Salt should change after reset")
	}

	// Generate a nonce and verify counter increments from 0
	_, err = ng.GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce after reset failed: %v", err)
	}
	if ng.GetCounter() != 1 {
		t.Errorf("Counter should be 1 after reset and one nonce, got %d", ng.GetCounter())
	}
}

// TestDifferentGeneratorsDifferentSalts tests that different generators have different salts
func TestDifferentGeneratorsDifferentSalts(t *testing.T) {
	ng1, err := NewNonceGenerator()
	if err != nil {
		t.Fatalf("NewNonceGenerator 1 failed: %v", err)
	}

	ng2, err := NewNonceGenerator()
	if err != nil {
		t.Fatalf("NewNonceGenerator 2 failed: %v", err)
	}

	salt1 := ng1.GetSalt()
	salt2 := ng2.GetSalt()

	// Salts should be different (extremely unlikely to be the same with crypto/rand)
	if salt1 == salt2 {
		t.Error("Different generators should have different salts (extremely unlikely collision)")
	}
}

// TestNoncesFromDifferentGenerators tests that nonces from different generators are unique
func TestNoncesFromDifferentGenerators(t *testing.T) {
	ng1, _ := NewNonceGenerator()
	ng2, _ := NewNonceGenerator()

	nonce1, _ := ng1.GenerateNonce()
	nonce2, _ := ng2.GenerateNonce()

	// Nonces should be different (due to different salts)
	if nonce1 == nonce2 {
		t.Error("Nonces from different generators should be different")
	}
}

// TestCounterOverflow tests counter overflow handling
// Note: This test would take too long to reach 2^48, so we test the logic manually
func TestCounterOverflowSimulation(t *testing.T) {
	ng, err := NewNonceGenerator()
	if err != nil {
		t.Fatalf("NewNonceGenerator failed: %v", err)
	}

	// Manually set counter near overflow
	ng.counter = MaxCounter - 5

	initialSalt := ng.GetSalt()

	// Generate 10 nonces (should trigger overflow and salt regeneration)
	for i := 0; i < 10; i++ {
		_, err := ng.GenerateNonce()
		if err != nil {
			t.Fatalf("GenerateNonce failed at iteration %d: %v", i, err)
		}
	}

	// Counter should have reset
	if ng.GetCounter() > 10 {
		t.Logf("Counter after overflow: %d", ng.GetCounter())
	}

	// Salt should have changed after overflow
	newSalt := ng.GetSalt()
	saltChanged := false
	for i := 0; i < SaltSize; i++ {
		if initialSalt[i] != newSalt[i] {
			saltChanged = true
			break
		}
	}
	if !saltChanged {
		t.Error("Salt should change after counter overflow")
	}
}

// BenchmarkGenerateNonce benchmarks nonce generation
func BenchmarkGenerateNonce(b *testing.B) {
	ng, err := NewNonceGenerator()
	if err != nil {
		b.Fatalf("NewNonceGenerator failed: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ng.GenerateNonce()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateNonceParallel benchmarks concurrent nonce generation
func BenchmarkGenerateNonceParallel(b *testing.B) {
	ng, err := NewNonceGenerator()
	if err != nil {
		b.Fatalf("NewNonceGenerator failed: %v", err)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := ng.GenerateNonce()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
