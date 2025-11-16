package nat

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// NATType represents the detected NAT type
type NATType int

const (
	// NAT type classifications based on RFC 3489/5389
	NATTypeUnknown            NATType = iota
	NATTypeNoNAT                      // Public IP, no translation
	NATTypeFullCone                   // One-to-one mapping, any external host can send
	NATTypeRestrictedCone             // External host must have received packet first
	NATTypePortRestrictedCone         // External host must match both IP and port
	NATTypeSymmetric                  // Different mapping for each destination
)

// String returns the human-readable name of the NAT type
func (n NATType) String() string {
	switch n {
	case NATTypeNoNAT:
		return "NoNAT"
	case NATTypeFullCone:
		return "FullCone"
	case NATTypeRestrictedCone:
		return "RestrictedCone"
	case NATTypePortRestrictedCone:
		return "PortRestrictedCone"
	case NATTypeSymmetric:
		return "Symmetric"
	default:
		return "Unknown"
	}
}

// ParseNATType converts a string to NATType
func ParseNATType(s string) (NATType, error) {
	switch s {
	case "NoNAT":
		return NATTypeNoNAT, nil
	case "FullCone":
		return NATTypeFullCone, nil
	case "RestrictedCone":
		return NATTypeRestrictedCone, nil
	case "PortRestrictedCone":
		return NATTypePortRestrictedCone, nil
	case "Symmetric":
		return NATTypeSymmetric, nil
	case "Unknown":
		return NATTypeUnknown, nil
	default:
		return NATTypeUnknown, fmt.Errorf("invalid NAT type: %s", s)
	}
}

// DetectionResult contains NAT detection results
type DetectionResult struct {
	NATType       NATType
	PublicIP      net.IP
	PublicPort    int
	DetectedAt    time.Time
	DetectionTime time.Duration // How long detection took
}

// NATDetector performs NAT type detection using STUN-like protocol
type NATDetector struct {
	stunClient *STUNClient

	// Result caching
	cachedResult         *DetectionResult
	cacheExpiration      time.Time
	cacheMutex           sync.RWMutex
	defaultCacheDuration time.Duration

	// Configuration
	manualOverride *NATType // Manual NAT type override
	timeout        time.Duration
}

// NewNATDetector creates a new NAT type detector
func NewNATDetector() *NATDetector {
	return &NATDetector{
		stunClient:           NewSTUNClient(),
		defaultCacheDuration: 24 * time.Hour,  // Cache for 24 hours
		timeout:              2 * time.Second, // Detection timeout <2s (AC #5)
	}
}

// SetManualOverride sets a manual NAT type override for debugging
func (nd *NATDetector) SetManualOverride(natType NATType) {
	nd.cacheMutex.Lock()
	defer nd.cacheMutex.Unlock()
	nd.manualOverride = &natType
}

// ClearManualOverride clears the manual NAT type override
func (nd *NATDetector) ClearManualOverride() {
	nd.cacheMutex.Lock()
	defer nd.cacheMutex.Unlock()
	nd.manualOverride = nil
}

// GetCachedResult returns the cached detection result if still valid
func (nd *NATDetector) GetCachedResult() (*DetectionResult, bool) {
	nd.cacheMutex.RLock()
	defer nd.cacheMutex.RUnlock()

	// Check manual override first
	if nd.manualOverride != nil {
		return &DetectionResult{
			NATType:    *nd.manualOverride,
			DetectedAt: time.Now(),
		}, true
	}

	// Check cache
	if nd.cachedResult != nil && time.Now().Before(nd.cacheExpiration) {
		return nd.cachedResult, true
	}

	return nil, false
}

// CacheResult caches a detection result with specified duration
func (nd *NATDetector) CacheResult(result *DetectionResult, duration time.Duration) {
	nd.cacheMutex.Lock()
	defer nd.cacheMutex.Unlock()

	nd.cachedResult = result
	nd.cacheExpiration = time.Now().Add(duration)
}

// InvalidateCache invalidates the cached detection result
func (nd *NATDetector) InvalidateCache() {
	nd.cacheMutex.Lock()
	defer nd.cacheMutex.Unlock()

	nd.cachedResult = nil
	nd.cacheExpiration = time.Time{}
}

// DetectNATType performs NAT type detection using STUN-like protocol
// Returns cached result if available and still valid
func (nd *NATDetector) DetectNATType(ctx context.Context) (*DetectionResult, error) {
	startTime := time.Now()

	// Check cache first
	if cached, ok := nd.GetCachedResult(); ok {
		return cached, nil
	}

	// Create context with timeout
	detectionCtx, cancel := context.WithTimeout(ctx, nd.timeout)
	defer cancel()

	// Perform detection
	result, err := nd.detectNATTypeInternal(detectionCtx)
	if err != nil {
		return nil, fmt.Errorf("NAT detection failed: %w", err)
	}

	// Record detection time
	result.DetectionTime = time.Since(startTime)

	// Cache result
	nd.CacheResult(result, nd.defaultCacheDuration)

	return result, nil
}

// detectNATTypeInternal performs the actual NAT type detection
// Based on RFC 3489/5389 STUN protocol
func (nd *NATDetector) detectNATTypeInternal(ctx context.Context) (*DetectionResult, error) {
	// Test 1: Get public address from STUN server
	localPort := 0 // Use random port
	publicAddr1, err := nd.stunClient.DiscoverPublicAddress(localPort)
	if err != nil {
		return nil, fmt.Errorf("STUN Test 1 failed: %w", err)
	}

	// Get local address
	localAddrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get local addresses: %w", err)
	}

	var localIP net.IP
	for _, addr := range localAddrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipv4 := ipnet.IP.To4(); ipv4 != nil {
				localIP = ipv4
				break
			}
		}
	}

	if localIP == nil {
		return nil, fmt.Errorf("no local IPv4 address found")
	}

	// Check if public IP == local IP (No NAT)
	if publicAddr1.IP.Equal(localIP) {
		return &DetectionResult{
			NATType:    NATTypeNoNAT,
			PublicIP:   publicAddr1.IP,
			PublicPort: publicAddr1.Port,
			DetectedAt: time.Now(),
		}, nil
	}

	// Test 2: Get public address again from different port
	// If port changes, it's Symmetric NAT
	publicAddr2, err := nd.stunClient.DiscoverPublicAddress(localPort)
	if err != nil {
		// If second test fails, assume Restricted Cone (conservative)
		return &DetectionResult{
			NATType:    NATTypeRestrictedCone,
			PublicIP:   publicAddr1.IP,
			PublicPort: publicAddr1.Port,
			DetectedAt: time.Now(),
		}, nil
	}

	// If public port differs between tests, it's Symmetric NAT
	if publicAddr1.Port != publicAddr2.Port {
		return &DetectionResult{
			NATType:    NATTypeSymmetric,
			PublicIP:   publicAddr1.IP,
			PublicPort: publicAddr1.Port,
			DetectedAt: time.Now(),
		}, nil
	}

	// For MVP, classify remaining as Restricted Cone
	// Full classification would require:
	// - Test 3: Change request (different IP, same port) for Full Cone vs Restricted
	// - Test 4: Change request (different IP, different port) for Restricted vs Port-Restricted
	// This requires STUN server with CHANGE-REQUEST support
	//
	// Conservative approach: assume Port-Restricted Cone
	// This works for most NAT configurations and is safer than Full Cone
	return &DetectionResult{
		NATType:    NATTypePortRestrictedCone,
		PublicIP:   publicAddr1.IP,
		PublicPort: publicAddr1.Port,
		DetectedAt: time.Now(),
	}, nil
}

// GetNATTypeString returns the human-readable NAT type
func (nd *NATDetector) GetNATTypeString() string {
	result, ok := nd.GetCachedResult()
	if !ok {
		return "Unknown (not detected yet)"
	}
	return result.NATType.String()
}

// IsP2PFeasible returns whether direct P2P is feasible with this NAT type
func (nd *NATDetector) IsP2PFeasible() bool {
	result, ok := nd.GetCachedResult()
	if !ok {
		return false // Unknown NAT type, assume not feasible
	}

	switch result.NATType {
	case NATTypeNoNAT:
		return true // Always works (public IP)
	case NATTypeFullCone:
		return true // >80% success rate expected
	case NATTypeRestrictedCone:
		return true // >60% success rate expected
	case NATTypePortRestrictedCone:
		return true // Moderate success rate
	case NATTypeSymmetric:
		return false // Hole punching typically fails
	default:
		return false
	}
}
