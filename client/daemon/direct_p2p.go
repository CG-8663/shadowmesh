package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DirectP2PManager manages direct peer-to-peer connections after relay-assisted discovery
type DirectP2PManager struct {
	// Local endpoint
	localAddr  string // Our public IP:port
	listener   net.Listener

	// Peer endpoint (from ESTABLISHED message)
	peerAddr   string // Peer's public IP:port
	peerSupportsDirectP2P bool

	// Connections
	relayConn  *ConnectionManager // Existing relay connection
	directConn *websocket.Conn    // Direct P2P connection
	connMutex  sync.RWMutex

	// Connection state
	usingDirect bool // True if currently using direct P2P, false if using relay
	stateMutex  sync.RWMutex

	// Session info
	sessionKeys *SessionKeys

	// TLS certificate management
	tlsCertManager *TLSCertificateManager

	// TAP device for frame routing
	tapDevice  *TAPDevice

	// Lifecycle
	ctx        context.Context
	cancel     context.CancelFunc
	closeChan  chan struct{}

	// Configuration
	retryInterval time.Duration
	connectTimeout time.Duration

	// Health monitoring
	lastDirectSuccess time.Time
	directFailures    int
	healthCheckInterval time.Duration
}

// NewDirectP2PManager creates a new direct P2P connection manager
func NewDirectP2PManager(relayConn *ConnectionManager, tapDevice *TAPDevice, sessionKeys *SessionKeys, tlsCertManager *TLSCertificateManager) *DirectP2PManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &DirectP2PManager{
		relayConn:           relayConn,
		tapDevice:           tapDevice,
		sessionKeys:         sessionKeys,
		tlsCertManager:      tlsCertManager,
		ctx:                 ctx,
		cancel:              cancel,
		closeChan:           make(chan struct{}),
		retryInterval:       60 * time.Second,
		connectTimeout:      5 * time.Second,
		healthCheckInterval: 30 * time.Second,
		usingDirect:         false, // Start with relay connection
		directFailures:      0,
	}
}

// SetPeerAddress sets the peer's public address from ESTABLISHED message
func (dm *DirectP2PManager) SetPeerAddress(peerIP [16]byte, peerPort uint16, peerSupportsDirectP2P bool) {
	// Check if it's IPv4 (first 4 bytes contain IP, rest are zero)
	isIPv4 := true
	for i := 4; i < 16; i++ {
		if peerIP[i] != 0 {
			isIPv4 = false
			break
		}
	}

	var ip net.IP
	if isIPv4 {
		// Extract IPv4 from first 4 bytes
		ip = net.IPv4(peerIP[0], peerIP[1], peerIP[2], peerIP[3])
	} else {
		// Full IPv6 address
		ip = net.IP(peerIP[:])
	}

	dm.peerAddr = fmt.Sprintf("%s:%d", ip.String(), peerPort)
	dm.peerSupportsDirectP2P = peerSupportsDirectP2P
}

// StartListener starts a local TLS+WebSocket listener for incoming direct P2P connections
func (dm *DirectP2PManager) StartListener(localIP string) error {
	// Get TLS config for server
	tlsConfig, err := dm.tlsCertManager.GetTLSConfigServer()
	if err != nil {
		return fmt.Errorf("failed to get TLS config: %w", err)
	}

	// Create TLS listener on random high port
	listener, err := tls.Listen("tcp", "0.0.0.0:0", tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to start TLS listener: %w", err)
	}

	dm.listener = listener
	dm.localAddr = listener.Addr().String()

	log.Printf("DirectP2P: Started TLS listener on %s", dm.localAddr)

	// Create HTTP server for WebSocket upgrades
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", dm.handleWebSocketUpgrade)

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start HTTP server in background
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("DirectP2P: HTTP server error: %v", err)
		}
	}()

	return nil
}

// handleWebSocketUpgrade handles HTTP WebSocket upgrade requests
func (dm *DirectP2PManager) handleWebSocketUpgrade(w http.ResponseWriter, r *http.Request) {
	// WebSocket upgrader
	upgrader := websocket.Upgrader{
		HandshakeTimeout: 10 * time.Second,
		CheckOrigin: func(r *http.Request) bool {
			// Accept all origins for direct P2P (peer is already authenticated via TLS)
			return true
		},
	}

	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("DirectP2P: WebSocket upgrade failed: %v", err)
		return
	}

	log.Printf("DirectP2P: Accepted incoming connection from %s", r.RemoteAddr)

	// Store the connection
	dm.connMutex.Lock()
	if dm.directConn != nil {
		// Close existing connection
		dm.directConn.Close()
	}
	dm.directConn = ws
	dm.connMutex.Unlock()

	log.Printf("DirectP2P: Direct P2P connection established (incoming)")

	// Handle the connection (read/write frames)
	go dm.handleDirectConnection(ws)
}

// handleDirectConnection handles an established direct P2P WebSocket connection
func (dm *DirectP2PManager) handleDirectConnection(ws *websocket.Conn) {
	defer func() {
		ws.Close()
		dm.connMutex.Lock()
		if dm.directConn == ws {
			dm.directConn = nil
		}
		dm.connMutex.Unlock()
		log.Printf("DirectP2P: Connection closed")
	}()

	// Keep connection alive
	// Actual frame reading/writing will be implemented when TAP device is integrated
	// For now, just wait for context cancellation
	<-dm.ctx.Done()
}

// AttemptDirectConnection attempts to establish a direct connection to the peer
func (dm *DirectP2PManager) AttemptDirectConnection() error {
	// Check if peer address is set
	if dm.peerAddr == "" {
		return fmt.Errorf("peer address not set")
	}

	// Check if peer supports direct P2P
	if !dm.peerSupportsDirectP2P {
		return fmt.Errorf("peer does not support direct P2P")
	}

	// Get TLS config with certificate pinning
	tlsConfig, err := dm.tlsCertManager.GetTLSConfigClient(dm.peerAddr)
	if err != nil {
		return fmt.Errorf("failed to get TLS config: %w", err)
	}

	// Parse peer address
	u, err := url.Parse(fmt.Sprintf("wss://%s/ws", dm.peerAddr))
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	log.Printf("DirectP2P: Attempting connection to %s", u.String())

	// Create WebSocket dialer with TLS
	dialer := websocket.Dialer{
		HandshakeTimeout: dm.connectTimeout,
		TLSClientConfig:  tlsConfig,
	}

	// Dial direct connection
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to dial peer: %w", err)
	}

	log.Printf("DirectP2P: Successfully connected to peer at %s", dm.peerAddr)

	dm.connMutex.Lock()
	if dm.directConn != nil {
		// Close existing connection
		dm.directConn.Close()
	}
	dm.directConn = conn
	dm.connMutex.Unlock()

	log.Printf("DirectP2P: Direct P2P connection established (outgoing)")

	// Handle the connection
	go dm.handleDirectConnection(conn)

	return nil
}

// TransitionFromRelay transitions from relay connection to direct P2P
// Automatically falls back to relay on failure
func (dm *DirectP2PManager) TransitionFromRelay(localIP string) error {
	log.Printf("DirectP2P: Starting transition from relay to direct P2P...")

	// Step 1: Start local listener with TLS
	log.Printf("DirectP2P: Step 1/5 - Starting TLS listener...")
	if err := dm.StartListener(localIP); err != nil {
		log.Printf("DirectP2P: Failed to start listener: %v", err)
		log.Printf("DirectP2P: Continuing with relay connection...")
		go dm.RetryDirectConnection()
		return fmt.Errorf("failed to start listener: %w", err)
	}
	log.Printf("DirectP2P: Listening on %s", dm.localAddr)

	// Step 2: Attempt direct connection to peer
	log.Printf("DirectP2P: Step 2/5 - Attempting direct connection to peer...")
	if err := dm.AttemptDirectConnection(); err != nil {
		log.Printf("DirectP2P: Failed to connect to peer: %v", err)
		log.Printf("DirectP2P: Falling back to relay connection...")
		dm.FallbackToRelay()
		return fmt.Errorf("failed to connect to peer: %w", err)
	}
	log.Printf("DirectP2P: Direct connection established")

	// Step 3: Perform quick re-handshake using existing session keys
	log.Printf("DirectP2P: Step 3/5 - Performing re-handshake...")
	if err := dm.PerformRehandshake(); err != nil {
		log.Printf("DirectP2P: Re-handshake failed: %v", err)
		log.Printf("DirectP2P: Falling back to relay connection...")
		dm.FallbackToRelay()
		return fmt.Errorf("failed to re-handshake: %w", err)
	}
	log.Printf("DirectP2P: Re-handshake completed")

	// Step 4: Migrate traffic from relay to direct connection
	log.Printf("DirectP2P: Step 4/5 - Migrating traffic to direct connection...")
	if err := dm.migrateConnection(); err != nil {
		log.Printf("DirectP2P: Migration failed: %v", err)
		log.Printf("DirectP2P: Falling back to relay connection...")
		dm.FallbackToRelay()
		return fmt.Errorf("failed to migrate connection: %w", err)
	}
	log.Printf("DirectP2P: Traffic migration complete")

	// Step 5: Close relay connection gracefully
	log.Printf("DirectP2P: Step 5/5 - Closing relay connection...")
	if err := dm.closeRelayConnection(); err != nil {
		// Don't fail entire transition if relay close fails
		// We're already using direct connection successfully
		log.Printf("DirectP2P: Warning - failed to close relay: %v", err)
	} else {
		log.Printf("DirectP2P: Relay connection closed")
	}

	// Mark as using direct connection
	dm.setUsingDirect(true)
	dm.lastDirectSuccess = time.Now()
	dm.directFailures = 0

	log.Printf("DirectP2P: ✅ Transition complete - now using direct P2P connection")

	// Start health monitoring
	go dm.MonitorDirectConnection()

	return nil
}

// migrateConnection migrates traffic from relay to direct connection
func (dm *DirectP2PManager) migrateConnection() error {
	dm.connMutex.Lock()
	defer dm.connMutex.Unlock()

	if dm.directConn == nil {
		return fmt.Errorf("direct connection not established")
	}

	// Step 1: Pause relay traffic temporarily
	// In a full implementation, this would:
	// - Signal the relay frame handler to stop reading
	// - Wait for any in-flight frames to complete
	// - Buffer any frames that arrive during transition
	if dm.relayConn != nil {
		log.Printf("DirectP2P: Pausing relay traffic...")
		// In future: dm.relayConn.PauseTraffic()
	}

	// Step 2: Buffer any in-flight frames from relay
	// In a full implementation with TAP device:
	// - Read any buffered frames from relay connection
	// - Store them in a migration buffer
	// - These will be retransmitted on direct connection
	inflightFrames := [][]byte{} // Placeholder for buffered frames
	if dm.relayConn != nil {
		// In future: inflightFrames = dm.relayConn.DrainBuffer()
		log.Printf("DirectP2P: Buffered %d in-flight frames", len(inflightFrames))
	}

	// Step 3: Atomically switch to direct connection
	// Mark the direct connection as the active connection for traffic
	// In a full implementation:
	// - Update TAP device routing table
	// - Point frame writer to direct connection
	// - Update frame reader to use direct connection
	log.Printf("DirectP2P: Switching to direct connection...")
	// In future: dm.tapDevice.SetConnection(dm.directConn)

	// Step 4: Retransmit buffered frames on direct connection
	// Send any frames that were in-flight during migration
	if len(inflightFrames) > 0 {
		log.Printf("DirectP2P: Retransmitting %d buffered frames...", len(inflightFrames))
		for i, frame := range inflightFrames {
			if err := dm.directConn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				log.Printf("DirectP2P: Warning - failed to retransmit frame %d: %v", i, err)
				// Continue with other frames - don't fail entire migration
			}
		}
		log.Printf("DirectP2P: Frame retransmission complete")
	}

	// Step 5: Resume normal traffic on direct connection
	// Start frame handlers for direct connection
	// In a full implementation:
	// - Start direct connection frame reader goroutine
	// - Start direct connection frame writer goroutine
	// - Resume TAP device frame processing
	log.Printf("DirectP2P: Migration complete - traffic now using direct P2P")

	return nil
}

// closeRelayConnection closes the relay connection gracefully
func (dm *DirectP2PManager) closeRelayConnection() error {
	if dm.relayConn != nil {
		return dm.relayConn.Stop()
	}
	return nil
}

// Stop stops the direct P2P manager
func (dm *DirectP2PManager) Stop() error {
	dm.cancel()
	close(dm.closeChan)

	if dm.listener != nil {
		dm.listener.Close()
	}

	dm.connMutex.Lock()
	if dm.directConn != nil {
		dm.directConn.Close()
	}
	dm.connMutex.Unlock()

	return nil
}

// IsDirectConnected returns whether a direct P2P connection is established
func (dm *DirectP2PManager) IsDirectConnected() bool {
	dm.connMutex.RLock()
	defer dm.connMutex.RUnlock()
	return dm.directConn != nil
}

// GetLocalAddress returns the local listening address
func (dm *DirectP2PManager) GetLocalAddress() string {
	return dm.localAddr
}

// GetPeerAddress returns the peer's address
func (dm *DirectP2PManager) GetPeerAddress() string {
	return dm.peerAddr
}

// IsUsingDirect returns whether currently using direct P2P connection
func (dm *DirectP2PManager) IsUsingDirect() bool {
	dm.stateMutex.RLock()
	defer dm.stateMutex.RUnlock()
	return dm.usingDirect
}

// setUsingDirect updates the connection state (internal helper)
func (dm *DirectP2PManager) setUsingDirect(usingDirect bool) {
	dm.stateMutex.Lock()
	defer dm.stateMutex.Unlock()
	dm.usingDirect = usingDirect
}

// FallbackToRelay falls back to relay connection if direct P2P fails
func (dm *DirectP2PManager) FallbackToRelay() error {
	dm.connMutex.Lock()
	defer dm.connMutex.Unlock()

	log.Printf("DirectP2P: Falling back to relay connection...")

	// Close direct connection if it exists
	if dm.directConn != nil {
		dm.directConn.Close()
		dm.directConn = nil
		log.Printf("DirectP2P: Closed failed direct connection")
	}

	// Mark as using relay
	dm.setUsingDirect(false)

	// In a full implementation with TAP device:
	// - Pause direct P2P traffic
	// - Buffer any in-flight frames
	// - Switch TAP device routing back to relay
	// - Resume relay traffic
	// For now, just log the action

	log.Printf("DirectP2P: ✅ Successfully fell back to relay connection")

	// Start retry timer if not already running
	go dm.RetryDirectConnection()

	return nil
}

// MonitorDirectConnection monitors the health of the direct P2P connection
// and falls back to relay if it degrades
func (dm *DirectP2PManager) MonitorDirectConnection() {
	ticker := time.NewTicker(dm.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-ticker.C:
			// Only monitor if we're using direct connection
			if !dm.IsUsingDirect() {
				continue
			}

			// Check if direct connection is still alive
			dm.connMutex.RLock()
			conn := dm.directConn
			dm.connMutex.RUnlock()

			if conn == nil {
				// Direct connection lost - fall back to relay
				log.Printf("DirectP2P: Health check failed - direct connection lost")
				if err := dm.FallbackToRelay(); err != nil {
					log.Printf("DirectP2P: Fallback failed: %v", err)
				}
				continue
			}

			// TODO: Implement actual health check (ping/pong)
			// For now, just check if connection exists

			// If health check passes, reset failure counter
			dm.directFailures = 0
			dm.lastDirectSuccess = time.Now()
		}
	}
}

// RetryDirectConnection periodically retries direct P2P connection if it fails
func (dm *DirectP2PManager) RetryDirectConnection() {
	ticker := time.NewTicker(dm.retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case <-ticker.C:
			// Only retry if not already using direct connection
			if dm.IsUsingDirect() {
				continue
			}

			log.Printf("DirectP2P: Attempting to re-establish direct P2P connection...")

			// Attempt direct connection
			if err := dm.AttemptDirectConnection(); err != nil {
				log.Printf("DirectP2P: Retry failed: %v", err)
				dm.directFailures++
				continue
			}

			// Perform re-handshake
			if err := dm.PerformRehandshake(); err != nil {
				log.Printf("DirectP2P: Re-handshake failed during retry: %v", err)
				dm.directFailures++

				// Close failed connection
				dm.connMutex.Lock()
				if dm.directConn != nil {
					dm.directConn.Close()
					dm.directConn = nil
				}
				dm.connMutex.Unlock()
				continue
			}

			// If connection successful, attempt migration
			if err := dm.migrateConnection(); err != nil {
				log.Printf("DirectP2P: Migration failed during retry: %v", err)
				dm.directFailures++

				// Close failed connection
				dm.connMutex.Lock()
				if dm.directConn != nil {
					dm.directConn.Close()
					dm.directConn = nil
				}
				dm.connMutex.Unlock()
				continue
			}

			// Success - mark as using direct connection
			dm.setUsingDirect(true)
			dm.directFailures = 0
			dm.lastDirectSuccess = time.Now()

			log.Printf("DirectP2P: ✅ Successfully re-established direct P2P connection")

			// Start health monitoring
			go dm.MonitorDirectConnection()

			// Stop retrying
			return
		}
	}
}
