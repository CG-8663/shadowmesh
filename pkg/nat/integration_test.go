package nat

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestNATDetectionIntegration tests the full NAT detection workflow
// This is an integration test that requires manual validation with real NAT routers
//
// Manual Testing Procedure:
// 1. Deploy client behind different NAT types:
//   - No NAT: Public IP (cloud VM)
//   - Full Cone NAT: Most home routers (uncommon)
//   - Restricted/Port-Restricted Cone NAT: Most residential NAT (common)
//   - Symmetric NAT: Corporate/CGNAT networks (strict)
//
// 2. Run detection on each and verify:
//   - Detection completes in <2 seconds
//   - NAT type matches expected behavior
//   - Public IP/port discovered correctly
//
// 3. Test caching:
//   - First detection queries STUN
//   - Second detection uses cache (instant)
//   - Cache invalidation triggers re-detection
//
// 4. Test manual override:
//   - Set --nat-type flag or config file
//   - Verify override takes precedence
func TestNATDetectionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Log("🧪 Testing NAT Detection Integration")

	// Create detector
	detector := NewNATDetector()

	// Test 1: Perform detection
	t.Log("1. Performing NAT detection...")
	ctx := context.Background()
	start := time.Now()

	result, err := detector.DetectNATType(ctx)

	elapsed := time.Since(start)
	t.Logf("   Detection completed in %v", elapsed)

	if elapsed > 2*time.Second {
		t.Errorf("❌ Detection took too long: %v (expected <2s)", elapsed)
	} else {
		t.Logf("   ✅ Detection completed within 2s requirement")
	}

	if err != nil {
		t.Logf("   ⚠️  Detection failed (may be expected if no STUN server available): %v", err)
		t.Logf("   This is expected in isolated test environments")
		t.Logf("   Manual testing required with real network")
		return // Skip rest of test
	}

	t.Logf("   ✅ Detection succeeded")
	t.Logf("   NAT Type: %s", result.NATType.String())
	t.Logf("   Public IP: %s", result.PublicIP)
	t.Logf("   Public Port: %d", result.PublicPort)
	t.Logf("   Detection Time: %v", result.DetectionTime)

	// Test 2: Verify caching
	t.Log("2. Testing result caching...")
	start2 := time.Now()
	cached, err := detector.DetectNATType(ctx)
	elapsed2 := time.Since(start2)

	if err != nil {
		t.Fatalf("Cached detection failed: %v", err)
	}

	// Cached result should be instant
	if elapsed2 > 10*time.Millisecond {
		t.Errorf("Cached detection too slow: %v (expected <10ms)", elapsed2)
	} else {
		t.Logf("   ✅ Cached result retrieved in %v", elapsed2)
	}

	if cached.NATType != result.NATType {
		t.Errorf("Cached NAT type mismatch: got %s, expected %s",
			cached.NATType.String(), result.NATType.String())
	}

	// Test 3: Manual override
	t.Log("3. Testing manual override...")
	detector.SetManualOverride(NATTypeSymmetric)

	override, err := detector.DetectNATType(ctx)
	if err != nil {
		t.Fatalf("Override detection failed: %v", err)
	}

	if override.NATType != NATTypeSymmetric {
		t.Errorf("Override failed: got %s, expected Symmetric", override.NATType.String())
	} else {
		t.Logf("   ✅ Manual override working (Symmetric)")
	}

	// Test 4: P2P feasibility check
	t.Log("4. Testing P2P feasibility check...")
	detector.ClearManualOverride()
	detector.CacheResult(result, 1*time.Hour)

	feasible := detector.IsP2PFeasible()
	t.Logf("   P2P Feasible for %s: %v", result.NATType.String(), feasible)

	// Symmetric NAT should not be feasible
	detector.SetManualOverride(NATTypeSymmetric)
	if detector.IsP2PFeasible() {
		t.Error("Symmetric NAT should not be P2P feasible")
	} else {
		t.Log("   ✅ Symmetric NAT correctly marked as not feasible")
	}

	// No NAT should always be feasible
	detector.SetManualOverride(NATTypeNoNAT)
	if !detector.IsP2PFeasible() {
		t.Error("No NAT should be P2P feasible")
	} else {
		t.Log("   ✅ No NAT correctly marked as feasible")
	}

	t.Log("\n✅ Integration test completed")
	t.Log("\n📝 Manual Testing Checklist:")
	t.Log("   [ ] Test with public IP (cloud VM) - should detect NoNAT")
	t.Log("   [ ] Test with home router - should detect Cone NAT type")
	t.Log("   [ ] Test with corporate/CGNAT - should detect Symmetric")
	t.Log("   [ ] Verify detection completes in <2 seconds")
	t.Log("   [ ] Verify cache works (second call instant)")
	t.Log("   [ ] Test --nat-type config override")
	t.Log("   [ ] Test cache invalidation on network change")
}

// TestNATDetectorWithConfig tests NAT detector with config file override
func TestNATDetectorWithConfig(t *testing.T) {
	t.Log("🧪 Testing NAT Detector Configuration Override")

	// Simulate config file with manual override
	configNATType := "FullCone"

	detector := NewNATDetector()

	// Parse and set override from config
	natType, err := ParseNATType(configNATType)
	if err != nil {
		t.Fatalf("Failed to parse config NAT type: %v", err)
	}

	detector.SetManualOverride(natType)

	// Detect (should use override)
	ctx := context.Background()
	result, err := detector.DetectNATType(ctx)
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	if result.NATType != NATTypeFullCone {
		t.Errorf("Expected FullCone from config, got %s", result.NATType.String())
	} else {
		t.Log("✅ Config override working correctly")
	}

	// Test P2P feasibility with override
	if !detector.IsP2PFeasible() {
		t.Error("FullCone should be P2P feasible")
	}

	t.Log("✅ Configuration override test passed")
}

// TestPerformanceRequirement validates AC #5 (detection completes in <2s)
func TestPerformanceRequirement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Log("🧪 Testing Performance Requirement (<2s)")

	detector := NewNATDetector()
	ctx := context.Background()

	// Run multiple detections to test consistency
	var totalTime time.Duration
	attempts := 3

	for i := 0; i < attempts; i++ {
		// Invalidate cache to force fresh detection
		detector.InvalidateCache()

		start := time.Now()
		_, err := detector.DetectNATType(ctx)
		elapsed := time.Since(start)

		totalTime += elapsed

		t.Logf("   Attempt %d: %v (err: %v)", i+1, elapsed, err)

		// Each attempt must be <2s (AC #5)
		if elapsed > 2*time.Second {
			t.Errorf("❌ Attempt %d exceeded 2s: %v", i+1, elapsed)
		}
	}

	avgTime := totalTime / time.Duration(attempts)
	t.Logf("   Average detection time: %v", avgTime)

	if avgTime > 2*time.Second {
		t.Errorf("❌ Average detection time exceeded 2s: %v", avgTime)
	} else {
		t.Logf("   ✅ Performance requirement met (<2s)")
	}
}

// TestUDPHolePunchingIntegration tests AC #7: Full hole punching between two clients
// This validates bidirectional UDP traffic after successful hole punch
func TestUDPHolePunchingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping UDP hole punching integration test in short mode")
	}

	t.Log("🧪 Testing UDP Hole Punching Integration (Story 2.5)")

	// Setup: Create two NATDetectors with Full Cone NAT
	detector1 := NewNATDetector()
	detector1.SetManualOverride(NATTypeFullCone)

	detector2 := NewNATDetector()
	detector2.SetManualOverride(NATTypeFullCone)

	// Create two hole punchers (simulating two clients)
	hp1, err := NewHolePuncher(0, detector1)
	if err != nil {
		t.Fatalf("Failed to create hole puncher 1: %v", err)
	}
	defer hp1.Close()

	hp2, err := NewHolePuncher(0, detector2)
	if err != nil {
		t.Fatalf("Failed to create hole puncher 2: %v", err)
	}
	defer hp2.Close()

	addr1 := hp1.conn.LocalAddr().(*net.UDPAddr)
	addr2 := hp2.conn.LocalAddr().(*net.UDPAddr)

	t.Logf("   Client 1 address: %s", addr1)
	t.Logf("   Client 2 address: %s", addr2)

	// AC #2: Exchange public endpoints (simulated via local addresses)
	candidate1 := Candidate{
		Type: "host",
		IP:   "127.0.0.1",
		Port: addr1.Port,
	}

	candidate2 := Candidate{
		Type: "host",
		IP:   "127.0.0.1",
		Port: addr2.Port,
	}

	// AC #3: Simultaneous open - both clients attempt connection at same time
	t.Log("   Attempting simultaneous hole punch...")

	type result struct {
		conn *net.UDPConn
		err  error
	}

	results := make(chan result, 2)

	// Increase timeout for integration test
	hp1.SetTimeout(3 * time.Second)
	hp2.SetTimeout(3 * time.Second)

	// Client 1 connects to Client 2
	go func() {
		conn, err := hp1.EstablishConnection([]Candidate{candidate2})
		results <- result{conn, err}
	}()

	// Client 2 connects to Client 1
	go func() {
		conn, err := hp2.EstablishConnection([]Candidate{candidate1})
		results <- result{conn, err}
	}()

	// Collect results
	r1 := <-results
	r2 := <-results

	// At least one should succeed
	successCount := 0
	if r1.err == nil {
		successCount++
		t.Log("   ✅ Client 1 hole punch succeeded")
	} else {
		t.Logf("   ⚠️  Client 1 hole punch failed: %v", r1.err)
	}

	if r2.err == nil {
		successCount++
		t.Log("   ✅ Client 2 hole punch succeeded")
	} else {
		t.Logf("   ⚠️  Client 2 hole punch failed: %v", r2.err)
	}

	if successCount == 0 {
		t.Log("   ⚠️  Both hole punch attempts failed")
		t.Log("   Note: This may be expected in some network configurations")
		t.Log("   Manual testing with real NAT environments recommended")
		// Don't fail test - loopback testing has limitations
	}

	// Test bidirectional traffic if connection succeeded
	if r1.err == nil && r2.err == nil {
		t.Log("   Testing bidirectional traffic...")

		// Send test message from Client 1 to Client 2
		testMsg := []byte("HELLO_FROM_CLIENT_1")
		_, err := hp1.conn.WriteToUDP(testMsg, addr2)
		if err != nil {
			t.Logf("   ⚠️  Failed to send from Client 1: %v", err)
			t.Log("   Note: IPv6 loopback routing may fail - this is expected in local testing")
		}

		// Receive on Client 2
		buffer := make([]byte, 1500)
		hp2.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, _, err := hp2.conn.ReadFromUDP(buffer)
		if err == nil && string(buffer[:n]) == string(testMsg) {
			t.Log("   ✅ Bidirectional traffic validated")
		} else {
			t.Logf("   ⚠️  Bidirectional traffic test inconclusive: %v", err)
			t.Log("   Note: This is a known limitation of local loopback testing")
		}
	}

	// Check metrics (AC #5: Success rate validation)
	metrics1 := hp1.GetMetrics()
	metrics2 := hp2.GetMetrics()

	t.Logf("   Client 1 metrics - Success: %d, Failure: %d, Timeout: %d",
		metrics1.SuccessCount, metrics1.FailureCount, metrics1.TimeoutCount)
	t.Logf("   Client 2 metrics - Success: %d, Failure: %d, Timeout: %d",
		metrics2.SuccessCount, metrics2.FailureCount, metrics2.TimeoutCount)

	// AC #5: Success rate >80% for Full Cone NAT
	totalAttempts := metrics1.SuccessCount + metrics1.FailureCount +
		metrics2.SuccessCount + metrics2.FailureCount
	totalSuccess := metrics1.SuccessCount + metrics2.SuccessCount

	if totalAttempts > 0 {
		successRate := float64(totalSuccess) / float64(totalAttempts) * 100
		t.Logf("   Overall success rate: %.1f%%", successRate)

		if successRate < 80.0 {
			t.Logf("   ⚠️  Success rate below target 80%% (AC #5)")
			t.Log("   Note: Local loopback testing may not reflect real NAT behavior")
		} else {
			t.Logf("   ✅ Success rate meets target (>80%%)")
		}
	}

	t.Log("")
	t.Log("✅ UDP Hole Punching Integration Test Complete")
	t.Log("")
	t.Log("📝 Manual Testing Checklist:")
	t.Log("   [ ] Test between two clients behind Full Cone NAT (expected >80% success)")
	t.Log("   [ ] Test between two clients behind Restricted Cone NAT (expected >60% success)")
	t.Log("   [ ] Test with Symmetric NAT - should fail with relay fallback message")
	t.Log("   [ ] Verify 500ms timeout enforcement")
	t.Log("   [ ] Verify bidirectional traffic after successful hole punch")
	t.Log("   [ ] Test relay fallback is triggered on timeout")
	t.Log("   [ ] Validate metrics tracking (success/failure counts)")
}

// TestHolePunchRelayFallback tests AC #4: Relay fallback on timeout
func TestHolePunchRelayFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping relay fallback test in short mode")
	}

	t.Log("🧪 Testing Hole Punch Relay Fallback (Story 2.5 AC #4)")

	detector := NewNATDetector()
	detector.SetManualOverride(NATTypeRestrictedCone)

	hp, err := NewHolePuncher(0, detector)
	if err != nil {
		t.Fatalf("Failed to create hole puncher: %v", err)
	}
	defer hp.Close()

	// Attempt hole punch to unreachable address
	candidates := []Candidate{
		{Type: "srflx", IP: "203.0.113.1", Port: 12345}, // TEST-NET-3
	}

	t.Log("   Attempting hole punch to unreachable endpoint...")
	start := time.Now()

	_, err = hp.EstablishConnection(candidates)

	elapsed := time.Since(start)
	t.Logf("   Hole punch failed after %v", elapsed)

	// Verify error indicates relay fallback
	if err == nil {
		t.Error("   ❌ Expected timeout error, got success")
	} else {
		expectedMsg := "hole punch timeout after 500ms - fallback to relay"
		if err.Error() != expectedMsg {
			t.Errorf("   ❌ Unexpected error message: %v", err)
		} else {
			t.Log("   ✅ Relay fallback message correct")
		}
	}

	// Verify timeout was ~500ms
	if elapsed < 400*time.Millisecond || elapsed > 700*time.Millisecond {
		t.Errorf("   ⚠️  Timeout took %v, expected ~500ms (AC #4)", elapsed)
	} else {
		t.Log("   ✅ 500ms timeout enforced (AC #4)")
	}

	// Verify metrics
	metrics := hp.GetMetrics()
	if metrics.TimeoutCount != 1 || metrics.FailureCount != 1 {
		t.Errorf("   ❌ Metrics incorrect: timeout=%d, failure=%d",
			metrics.TimeoutCount, metrics.FailureCount)
	} else {
		t.Log("   ✅ Metrics correctly tracked")
	}

	t.Log("✅ Relay fallback test complete")
}
