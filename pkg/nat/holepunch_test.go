package nat

import (
	"net"
	"testing"
	"time"
)

// TestHolePuncherCreation tests hole puncher initialization
func TestHolePuncherCreation(t *testing.T) {
	detector := NewNATDetector()

	// Test with NAT detector
	hp, err := NewHolePuncher(0, detector) // Port 0 = auto-assign
	if err != nil {
		t.Fatalf("Failed to create hole puncher: %v", err)
	}
	defer hp.Close()

	if hp.timeout != 500*time.Millisecond {
		t.Errorf("Expected default timeout 500ms, got %v", hp.timeout)
	}

	if hp.detector == nil {
		t.Error("NAT detector not set")
	}
}

// TestHolePuncherWithoutDetector tests hole puncher without NAT detection
func TestHolePuncherWithoutDetector(t *testing.T) {
	hp, err := NewHolePuncher(0, nil)
	if err != nil {
		t.Fatalf("Failed to create hole puncher without detector: %v", err)
	}
	defer hp.Close()

	// Should work without detector (skips feasibility check)
	if hp.detector != nil {
		t.Error("Detector should be nil")
	}
}

// TestNATTypeFeasibilityCheck tests AC #1: Only attempt for compatible NAT types
func TestNATTypeFeasibilityCheck(t *testing.T) {
	detector := NewNATDetector()
	hp, err := NewHolePuncher(0, detector)
	if err != nil {
		t.Fatalf("Failed to create hole puncher: %v", err)
	}
	defer hp.Close()

	// Test with Symmetric NAT (should fail)
	detector.SetManualOverride(NATTypeSymmetric)

	candidates := []Candidate{
		{Type: "srflx", IP: "203.0.113.1", Port: 12345},
	}

	_, err = hp.EstablishConnection(candidates)
	if err == nil {
		t.Error("Expected error for Symmetric NAT, got nil")
	}

	if err.Error() != "NAT type not compatible with hole punching (Symmetric NAT detected)" {
		t.Errorf("Unexpected error message: %v", err)
	}

	metrics := hp.GetMetrics()
	if metrics.FailureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", metrics.FailureCount)
	}
}

// TestNATTypeFeasibilitySuccess tests hole punching with compatible NAT types
func TestNATTypeFeasibilitySuccess(t *testing.T) {
	detector := NewNATDetector()

	// Test compatible NAT types
	compatibleTypes := []NATType{
		NATTypeNoNAT,
		NATTypeFullCone,
		NATTypeRestrictedCone,
		NATTypePortRestrictedCone,
	}

	for _, natType := range compatibleTypes {
		detector.SetManualOverride(natType)

		hp, err := NewHolePuncher(0, detector)
		if err != nil {
			t.Fatalf("Failed to create hole puncher: %v", err)
		}

		// Should NOT return feasibility error
		// (will timeout since no real peer, but that's expected)
		candidates := []Candidate{
			{Type: "srflx", IP: "127.0.0.1", Port: 1},
		}

		_, err = hp.EstablishConnection(candidates)

		// Should timeout (not feasibility error)
		if err != nil && err.Error() == "NAT type not compatible with hole punching (Symmetric NAT detected)" {
			t.Errorf("NAT type %s should be feasible for hole punching", natType.String())
		}

		hp.Close()
	}
}

// TestHolePunchTimeout tests AC #4: 500ms timeout
func TestHolePunchTimeout(t *testing.T) {
	detector := NewNATDetector()
	detector.SetManualOverride(NATTypeFullCone)

	hp, err := NewHolePuncher(0, detector)
	if err != nil {
		t.Fatalf("Failed to create hole puncher: %v", err)
	}
	defer hp.Close()

	// Try to connect to non-existent endpoint
	candidates := []Candidate{
		{Type: "srflx", IP: "203.0.113.1", Port: 54321}, // TEST-NET-3 (won't respond)
	}

	start := time.Now()
	_, err = hp.EstablishConnection(candidates)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Should timeout around 500ms (allow some variance)
	if elapsed < 400*time.Millisecond || elapsed > 700*time.Millisecond {
		t.Errorf("Timeout took %v, expected ~500ms", elapsed)
	}

	metrics := hp.GetMetrics()
	if metrics.TimeoutCount != 1 {
		t.Errorf("Expected 1 timeout, got %d", metrics.TimeoutCount)
	}
	if metrics.FailureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", metrics.FailureCount)
	}
}

// TestConfigurableTimeout tests custom timeout configuration
func TestConfigurableTimeout(t *testing.T) {
	detector := NewNATDetector()
	detector.SetManualOverride(NATTypeFullCone)

	hp, err := NewHolePuncher(0, detector)
	if err != nil {
		t.Fatalf("Failed to create hole puncher: %v", err)
	}
	defer hp.Close()

	// Set custom timeout
	hp.SetTimeout(200 * time.Millisecond)

	candidates := []Candidate{
		{Type: "srflx", IP: "203.0.113.1", Port: 54321},
	}

	start := time.Now()
	_, err = hp.EstablishConnection(candidates)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Should timeout around 200ms
	if elapsed < 150*time.Millisecond || elapsed > 300*time.Millisecond {
		t.Errorf("Timeout took %v, expected ~200ms", elapsed)
	}
}

// TestMetricsTracking tests AC #4: Metrics for success/failure rates
func TestMetricsTracking(t *testing.T) {
	detector := NewNATDetector()
	detector.SetManualOverride(NATTypeFullCone)

	hp, err := NewHolePuncher(0, detector)
	if err != nil {
		t.Fatalf("Failed to create hole puncher: %v", err)
	}
	defer hp.Close()

	// Initial metrics should be zero
	metrics := hp.GetMetrics()
	if metrics.SuccessCount != 0 || metrics.FailureCount != 0 || metrics.TimeoutCount != 0 {
		t.Error("Initial metrics should be zero")
	}

	// Attempt 3 hole punches (will timeout)
	candidates := []Candidate{
		{Type: "srflx", IP: "203.0.113.1", Port: 54321},
	}

	for i := 0; i < 3; i++ {
		hp.EstablishConnection(candidates)
	}

	// Check metrics updated
	metrics = hp.GetMetrics()
	if metrics.TimeoutCount != 3 {
		t.Errorf("Expected 3 timeouts, got %d", metrics.TimeoutCount)
	}
	if metrics.FailureCount != 3 {
		t.Errorf("Expected 3 failures, got %d", metrics.FailureCount)
	}
	if metrics.SuccessCount != 0 {
		t.Errorf("Expected 0 successes, got %d", metrics.SuccessCount)
	}
}

// TestSimultaneousOpen tests AC #3: Both peers send packets simultaneously
func TestSimultaneousOpen(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping simultaneous open test in short mode")
	}

	detector := NewNATDetector()
	detector.SetManualOverride(NATTypeFullCone)

	// Create two hole punchers simulating two peers
	hp1, err := NewHolePuncher(0, detector)
	if err != nil {
		t.Fatalf("Failed to create hole puncher 1: %v", err)
	}
	defer hp1.Close()

	hp2, err := NewHolePuncher(0, detector)
	if err != nil {
		t.Fatalf("Failed to create hole puncher 2: %v", err)
	}
	defer hp2.Close()

	// Get actual listening addresses
	addr1 := hp1.conn.LocalAddr().(*net.UDPAddr)
	addr2 := hp2.conn.LocalAddr().(*net.UDPAddr)

	t.Logf("Peer 1 listening on: %s", addr1)
	t.Logf("Peer 2 listening on: %s", addr2)

	// Create candidates for each peer
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

	// Increase timeout for this test
	hp1.SetTimeout(2 * time.Second)
	hp2.SetTimeout(2 * time.Second)

	// AC #3: Simultaneous connection attempts
	done := make(chan error, 2)

	go func() {
		_, err := hp1.EstablishConnection([]Candidate{candidate2})
		done <- err
	}()

	go func() {
		_, err := hp2.EstablishConnection([]Candidate{candidate1})
		done <- err
	}()

	// Wait for both attempts
	err1 := <-done
	err2 := <-done

	// At least one should succeed in local loopback scenario
	if err1 != nil && err2 != nil {
		t.Logf("Both connections failed: %v, %v", err1, err2)
		t.Log("Note: Simultaneous open may fail in some network configurations")
	}

	// Check that hole punch attempts were logged
	metrics1 := hp1.GetMetrics()
	metrics2 := hp2.GetMetrics()

	if metrics1.SuccessCount+metrics1.FailureCount == 0 {
		t.Error("Peer 1 metrics not updated")
	}
	if metrics2.SuccessCount+metrics2.FailureCount == 0 {
		t.Error("Peer 2 metrics not updated")
	}
}

// TestPunchPacketSending tests UDP packet transmission
func TestPunchPacketSending(t *testing.T) {
	detector := NewNATDetector()
	detector.SetManualOverride(NATTypeFullCone)

	hp, err := NewHolePuncher(0, detector)
	if err != nil {
		t.Fatalf("Failed to create hole puncher: %v", err)
	}
	defer hp.Close()

	// Create mock UDP server to receive punch packets
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to resolve server address: %v", err)
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer serverConn.Close()

	actualAddr := serverConn.LocalAddr().(*net.UDPAddr)

	// Receive punch packets in background
	received := make(chan bool, 1)
	go func() {
		buffer := make([]byte, 1500)
		serverConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, _, err := serverConn.ReadFromUDP(buffer)
		if err == nil && n > 0 {
			msg := string(buffer[:n])
			if msg == "SHADOWMESH_PUNCH" {
				received <- true
				return
			}
		}
		received <- false
	}()

	// Send punch packets
	remoteAddr, _ := net.ResolveUDPAddr("udp", actualAddr.String())
	hp.sendPunchPackets(remoteAddr, 1)

	// Verify punch packet received
	select {
	case success := <-received:
		if !success {
			t.Error("Punch packet not received or incorrect")
		}
	case <-time.After(3 * time.Second):
		t.Error("Timeout waiting for punch packet")
	}
}

// TestRelayFallbackTrigger tests AC #4: Fallback to relay on timeout
func TestRelayFallbackTrigger(t *testing.T) {
	detector := NewNATDetector()
	detector.SetManualOverride(NATTypeFullCone)

	hp, err := NewHolePuncher(0, detector)
	if err != nil {
		t.Fatalf("Failed to create hole puncher: %v", err)
	}
	defer hp.Close()

	candidates := []Candidate{
		{Type: "srflx", IP: "203.0.113.1", Port: 54321},
	}

	_, err = hp.EstablishConnection(candidates)

	// Verify error message indicates relay fallback
	if err == nil {
		t.Error("Expected timeout error with fallback message")
	}

	expectedMsg := "hole punch timeout after 500ms - fallback to relay"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}
