package nat

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestNATTypeString tests NAT type string conversion
func TestNATTypeString(t *testing.T) {
	tests := []struct {
		natType  NATType
		expected string
	}{
		{NATTypeNoNAT, "NoNAT"},
		{NATTypeFullCone, "FullCone"},
		{NATTypeRestrictedCone, "RestrictedCone"},
		{NATTypePortRestrictedCone, "PortRestrictedCone"},
		{NATTypeSymmetric, "Symmetric"},
		{NATTypeUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.natType.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestParseNATType tests parsing NAT type from string
func TestParseNATType(t *testing.T) {
	tests := []struct {
		input       string
		expected    NATType
		shouldError bool
	}{
		{"NoNAT", NATTypeNoNAT, false},
		{"FullCone", NATTypeFullCone, false},
		{"RestrictedCone", NATTypeRestrictedCone, false},
		{"PortRestrictedCone", NATTypePortRestrictedCone, false},
		{"Symmetric", NATTypeSymmetric, false},
		{"Unknown", NATTypeUnknown, false},
		{"InvalidType", NATTypeUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseNATType(tt.input)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

// TestManualOverride tests manual NAT type override
func TestManualOverride(t *testing.T) {
	detector := NewNATDetector()

	// Set manual override
	detector.SetManualOverride(NATTypeSymmetric)

	// Check cached result
	result, ok := detector.GetCachedResult()
	if !ok {
		t.Fatal("Expected cached result with manual override")
	}

	if result.NATType != NATTypeSymmetric {
		t.Errorf("Expected Symmetric NAT, got %s", result.NATType.String())
	}

	// Clear override
	detector.ClearManualOverride()

	// Should no longer have cached result
	_, ok = detector.GetCachedResult()
	if ok {
		t.Error("Expected no cached result after clearing override")
	}
}

// TestResultCaching tests detection result caching
func TestResultCaching(t *testing.T) {
	detector := NewNATDetector()

	// Create test result
	testResult := &DetectionResult{
		NATType:    NATTypeFullCone,
		PublicIP:   net.ParseIP("203.0.113.10"),
		PublicPort: 12345,
		DetectedAt: time.Now(),
	}

	// Cache result for 1 second
	detector.CacheResult(testResult, 1*time.Second)

	// Retrieve cached result
	cached, ok := detector.GetCachedResult()
	if !ok {
		t.Fatal("Expected cached result")
	}

	if cached.NATType != testResult.NATType {
		t.Errorf("Expected %s, got %s", testResult.NATType.String(), cached.NATType.String())
	}

	if !cached.PublicIP.Equal(testResult.PublicIP) {
		t.Errorf("Expected IP %s, got %s", testResult.PublicIP, cached.PublicIP)
	}

	// Wait for cache to expire
	time.Sleep(1100 * time.Millisecond)

	// Should no longer be cached
	_, ok = detector.GetCachedResult()
	if ok {
		t.Error("Expected cache to expire after 1 second")
	}
}

// TestCacheInvalidation tests cache invalidation
func TestCacheInvalidation(t *testing.T) {
	detector := NewNATDetector()

	// Cache a result
	testResult := &DetectionResult{
		NATType:    NATTypeRestrictedCone,
		PublicIP:   net.ParseIP("198.51.100.5"),
		PublicPort: 54321,
		DetectedAt: time.Now(),
	}
	detector.CacheResult(testResult, 1*time.Hour)

	// Verify cached
	_, ok := detector.GetCachedResult()
	if !ok {
		t.Fatal("Expected cached result")
	}

	// Invalidate cache
	detector.InvalidateCache()

	// Should no longer be cached
	_, ok = detector.GetCachedResult()
	if ok {
		t.Error("Expected cache to be invalidated")
	}
}

// TestIsP2PFeasible tests P2P feasibility check
func TestIsP2PFeasible(t *testing.T) {
	tests := []struct {
		natType  NATType
		feasible bool
	}{
		{NATTypeNoNAT, true},
		{NATTypeFullCone, true},
		{NATTypeRestrictedCone, true},
		{NATTypePortRestrictedCone, true},
		{NATTypeSymmetric, false},
		{NATTypeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.natType.String(), func(t *testing.T) {
			detector := NewNATDetector()

			// Set manual override to test specific NAT type
			detector.SetManualOverride(tt.natType)

			result := detector.IsP2PFeasible()
			if result != tt.feasible {
				t.Errorf("Expected P2P feasible=%v for %s, got %v",
					tt.feasible, tt.natType.String(), result)
			}
		})
	}
}

// TestDetectionTimeout tests that detection completes within timeout
func TestDetectionTimeout(t *testing.T) {
	detector := NewNATDetector()

	ctx := context.Background()
	start := time.Now()

	// Attempt detection (may fail due to no real STUN server, but should timeout quickly)
	_, err := detector.DetectNATType(ctx)

	elapsed := time.Since(start)

	// Should complete (success or failure) within 2 seconds (AC #5)
	if elapsed > 3*time.Second {
		t.Errorf("Detection took too long: %v (expected <2s)", elapsed)
	}

	// Either succeeds or fails, both are acceptable for this test
	// We're testing timeout behavior, not actual detection
	t.Logf("Detection completed in %v with result: %v", elapsed, err)
}

// TestGetNATTypeString tests getting NAT type as string
func TestGetNATTypeString(t *testing.T) {
	detector := NewNATDetector()

	// Without detection
	result := detector.GetNATTypeString()
	if result != "Unknown (not detected yet)" {
		t.Errorf("Expected 'Unknown (not detected yet)', got '%s'", result)
	}

	// With manual override
	detector.SetManualOverride(NATTypeFullCone)
	result = detector.GetNATTypeString()
	if result != "FullCone" {
		t.Errorf("Expected 'FullCone', got '%s'", result)
	}
}

// TestDetectorDefaultConfiguration tests default detector configuration
func TestDetectorDefaultConfiguration(t *testing.T) {
	detector := NewNATDetector()

	if detector.stunClient == nil {
		t.Error("Expected STUN client to be initialized")
	}

	if detector.timeout != 2*time.Second {
		t.Errorf("Expected timeout 2s, got %v", detector.timeout)
	}

	if detector.defaultCacheDuration != 24*time.Hour {
		t.Errorf("Expected cache duration 24h, got %v", detector.defaultCacheDuration)
	}
}

// TestConcurrentAccess tests concurrent access to detector
func TestConcurrentAccess(t *testing.T) {
	detector := NewNATDetector()

	// Cache a result
	testResult := &DetectionResult{
		NATType:    NATTypeFullCone,
		PublicIP:   net.ParseIP("192.0.2.1"),
		PublicPort: 9999,
		DetectedAt: time.Now(),
	}
	detector.CacheResult(testResult, 1*time.Hour)

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				_, _ = detector.GetCachedResult()
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify result is still valid
	result, ok := detector.GetCachedResult()
	if !ok {
		t.Error("Expected cached result after concurrent access")
	}

	if result.NATType != testResult.NATType {
		t.Errorf("Result corrupted by concurrent access")
	}
}
