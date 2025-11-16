package nat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ConnectionCandidate represents a connection attempt candidate
type ConnectionCandidate struct {
	LocalAddr  *net.UDPAddr
	RemoteAddr *net.UDPAddr
	Conn       *net.UDPConn
}

// HolePunchMetrics tracks hole punching performance
type HolePunchMetrics struct {
	SuccessCount uint64 // Successful hole punch attempts
	FailureCount uint64 // Failed hole punch attempts
	TimeoutCount uint64 // Attempts that timed out
}

// HolePuncher handles UDP hole punching for NAT traversal
type HolePuncher struct {
	localPort int
	conn      *net.UDPConn
	mu        sync.Mutex
	detector  *NATDetector // NAT type detector for feasibility check
	metrics   HolePunchMetrics
	timeout   time.Duration // Configurable timeout (default 500ms per AC #4)
}

// NewHolePuncher creates a new hole puncher with NAT detection
func NewHolePuncher(localPort int, detector *NATDetector) (*HolePuncher, error) {
	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", localPort))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, err
	}

	return &HolePuncher{
		localPort: localPort,
		conn:      conn,
		detector:  detector,
		timeout:   500 * time.Millisecond, // AC #4: 500ms timeout
	}, nil
}

// SetTimeout allows configuring the hole punch timeout
func (h *HolePuncher) SetTimeout(timeout time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.timeout = timeout
}

// GetMetrics returns current hole punching metrics
func (h *HolePuncher) GetMetrics() HolePunchMetrics {
	return HolePunchMetrics{
		SuccessCount: atomic.LoadUint64(&h.metrics.SuccessCount),
		FailureCount: atomic.LoadUint64(&h.metrics.FailureCount),
		TimeoutCount: atomic.LoadUint64(&h.metrics.TimeoutCount),
	}
}

// EstablishConnection attempts to establish UDP connection through NAT
// AC #1: Only attempts hole punching for Full Cone and Restricted Cone NAT types
// AC #4: Uses 500ms timeout with relay fallback on failure
func (h *HolePuncher) EstablishConnection(remoteCandidates []Candidate) (*net.UDPConn, error) {
	// AC #1: Check NAT type feasibility before attempting hole punch
	if h.detector != nil && !h.detector.IsP2PFeasible() {
		atomic.AddUint64(&h.metrics.FailureCount, 1)
		return nil, fmt.Errorf("NAT type not compatible with hole punching (Symmetric NAT detected)")
	}

	results := make(chan *ConnectionCandidate, len(remoteCandidates))

	// AC #4: Use 500ms timeout (configurable via SetTimeout)
	h.mu.Lock()
	timeout := h.timeout
	h.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// AC #3: Try all candidates in parallel for simultaneous open
	for _, candidate := range remoteCandidates {
		go h.tryCandidate(ctx, candidate, results)
	}

	// Wait for first successful connection
	select {
	case result := <-results:
		atomic.AddUint64(&h.metrics.SuccessCount, 1)
		log.Printf("HolePunch: Connection established to %s", result.RemoteAddr)
		return result.Conn, nil
	case <-ctx.Done():
		atomic.AddUint64(&h.metrics.TimeoutCount, 1)
		atomic.AddUint64(&h.metrics.FailureCount, 1)
		return nil, fmt.Errorf("hole punch timeout after %v - fallback to relay", timeout)
	}
}

// tryCandidate attempts connection to a specific candidate
func (h *HolePuncher) tryCandidate(ctx context.Context, candidate Candidate, results chan<- *ConnectionCandidate) {
	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", candidate.IP, candidate.Port))
	if err != nil {
		return
	}

	// Send punch packets
	h.sendPunchPackets(remoteAddr, 5)

	// Wait for response
	h.conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	buffer := make([]byte, 1500)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, addr, err := h.conn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			// Verify this is from expected remote
			if addr.IP.Equal(remoteAddr.IP) && addr.Port == remoteAddr.Port {
				// Connection established
				results <- &ConnectionCandidate{
					LocalAddr:  h.conn.LocalAddr().(*net.UDPAddr),
					RemoteAddr: addr,
					Conn:       h.conn,
				}
				return
			}

			log.Printf("Received packet from unexpected address: %v (expected %v)", addr, remoteAddr)
		}
	}
}

// sendPunchPackets sends UDP packets to punch through NAT
func (h *HolePuncher) sendPunchPackets(remoteAddr *net.UDPAddr, count int) {
	// Send "PUNCH" packets
	punchMsg := []byte("SHADOWMESH_PUNCH")

	for i := 0; i < count; i++ {
		h.conn.WriteToUDP(punchMsg, remoteAddr)
		time.Sleep(100 * time.Millisecond)
	}
}

// Close closes the UDP connection
func (h *HolePuncher) Close() error {
	return h.conn.Close()
}

// CandidateExchange handles candidate exchange through Kademlia-based backbone
type CandidateExchange struct {
	backboneURL  string
	peerID       string
	sessionToken string
}

// NewCandidateExchange creates a new candidate exchange handler
func NewCandidateExchange(backboneURL, peerID, sessionToken string) *CandidateExchange {
	return &CandidateExchange{
		backboneURL:  backboneURL,
		peerID:       peerID,
		sessionToken: sessionToken,
	}
}

// GatherLocalCandidates gathers local network interface candidates
func (c *CandidateExchange) GatherLocalCandidates(localPort int) ([]Candidate, error) {
	candidates := []Candidate{}

	// Get local addresses from network interfaces
	localAddrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get local addresses: %w", err)
	}

	for _, addr := range localAddrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				candidates = append(candidates, Candidate{
					Type: "host",
					IP:   ipnet.IP.String(),
					Port: localPort,
				})
			}
		}
	}

	return candidates, nil
}

// PublishCandidates sends local candidates to Kademlia-based backbone
func (c *CandidateExchange) PublishCandidates(candidates []Candidate) error {
	// POST /api/nat/candidates/publish
	data := map[string]interface{}{
		"peer_id":    c.peerID,
		"candidates": candidates,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.backboneURL+"/api/nat/candidates/publish", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.sessionToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to publish candidates: %d", resp.StatusCode)
	}

	return nil
}

// GetCandidates retrieves remote peer's candidates from Kademlia-based backbone
func (c *CandidateExchange) GetCandidates(remotePeerID string) ([]Candidate, error) {
	// GET /api/nat/candidates/{peer_id}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/nat/candidates/%s", c.backboneURL, remotePeerID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.sessionToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get candidates: %d", resp.StatusCode)
	}

	var result struct {
		Candidates []Candidate `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Candidates, nil
}
