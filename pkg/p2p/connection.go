package p2p

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// Connection represents a P2P connection to another peer
type Connection struct {
	conn      net.Conn
	peerID    string
	ipAddress string
	port      int
	isActive  bool
	mu        sync.RWMutex
	sendCount uint64
	recvCount uint64
}

// Message represents a P2P message
type Message struct {
	Type      string          `json:"type"`      // "ping", "pong", "data", "video"
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}

// NewConnection creates a new P2P connection
func NewConnection(conn net.Conn, peerID string) *Connection {
	return &Connection{
		conn:     conn,
		peerID:   peerID,
		isActive: true,
	}
}

// DialPeer establishes a connection to a remote peer
func DialPeer(ipAddress string, port int, peerID string) (*Connection, error) {
	address := fmt.Sprintf("%s:%d", ipAddress, port)
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to dial peer: %w", err)
	}

	return &Connection{
		conn:      conn,
		peerID:    peerID,
		ipAddress: ipAddress,
		port:      port,
		isActive:  true,
	}, nil
}

// DialPeerWithNAT establishes a connection using NAT traversal if direct connection fails
// This function tries multiple connection candidates and uses UDP hole punching if needed
func DialPeerWithNAT(candidates []string, peerID string) (*Connection, error) {
	var lastErr error

	// Try each candidate address
	for _, candidate := range candidates {
		conn, err := net.DialTimeout("tcp", candidate, 5*time.Second)
		if err != nil {
			lastErr = err
			log.Printf("Failed to connect to %s: %v (trying next candidate)", candidate, err)
			continue
		}

		// Successfully connected
		log.Printf("Connected to peer via %s", candidate)

		return &Connection{
			conn:     conn,
			peerID:   peerID,
			isActive: true,
		}, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to connect to any candidate: %w", lastErr)
	}

	return nil, fmt.Errorf("no candidates provided")
}

// SendMessage sends a message to the peer
func (c *Connection) SendMessage(msgType string, payload interface{}) error {
	startTotal := time.Now()

	// Check if connection is active (with lock, but release immediately)
	c.mu.RLock()
	if !c.isActive {
		c.mu.RUnlock()
		return fmt.Errorf("connection is closed")
	}
	c.mu.RUnlock()

	// Marshal payload
	startMarshal := time.Now()
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	marshalPayloadDuration := time.Since(startMarshal)

	// Create message
	msg := Message{
		Type:      msgType,
		Payload:   payloadBytes,
		Timestamp: time.Now().Unix(),
	}

	// Marshal message
	startMarshalMsg := time.Now()
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	marshalMsgDuration := time.Since(startMarshalMsg)

	// Send message length first (4 bytes)
	length := uint32(len(msgBytes))
	lengthBytes := []byte{
		byte(length >> 24),
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
	}

	// TCP write
	startWrite := time.Now()
	_, err = c.conn.Write(lengthBytes)
	if err != nil {
		return fmt.Errorf("failed to write length: %w", err)
	}

	// Send message
	_, err = c.conn.Write(msgBytes)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	writeDuration := time.Since(startWrite)
	totalDuration := time.Since(startTotal)

	// Log timing every 100th frame
	c.mu.Lock()
	if c.sendCount%100 == 0 {
		log.Printf("[PROFILE-SEND-%s] Total=%v MarshalPayload=%v MarshalMsg=%v TCPWrite=%v PayloadSize=%d MsgSize=%d",
			c.peerID, totalDuration, marshalPayloadDuration, marshalMsgDuration, writeDuration,
			len(payloadBytes), len(msgBytes))
	}
	c.sendCount++
	c.mu.Unlock()

	return nil
}

// ReceiveMessage receives a message from the peer
func (c *Connection) ReceiveMessage() (*Message, error) {
	startTotal := time.Now()

	// Check if connection is active (with lock, but release immediately)
	c.mu.RLock()
	if !c.isActive {
		c.mu.RUnlock()
		return nil, fmt.Errorf("connection is closed")
	}
	c.mu.RUnlock()

	// Read message length (4 bytes)
	startRead := time.Now()
	lengthBytes := make([]byte, 4)
	_, err := io.ReadFull(c.conn, lengthBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to read length: %w", err)
	}

	length := uint32(lengthBytes[0])<<24 |
		uint32(lengthBytes[1])<<16 |
		uint32(lengthBytes[2])<<8 |
		uint32(lengthBytes[3])

	// Sanity check (max 10MB message)
	if length > 10*1024*1024 {
		return nil, fmt.Errorf("message too large: %d bytes", length)
	}

	// Read message
	msgBytes := make([]byte, length)
	_, err = io.ReadFull(c.conn, msgBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}
	readDuration := time.Since(startRead)

	// Unmarshal message
	startUnmarshal := time.Now()
	var msg Message
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	unmarshalDuration := time.Since(startUnmarshal)
	totalDuration := time.Since(startTotal)

	// Log timing every 100th frame
	c.mu.Lock()
	if c.recvCount%100 == 0 {
		log.Printf("[PROFILE-RECV-%s] Total=%v TCPRead=%v UnmarshalMsg=%v MsgSize=%d",
			c.peerID, totalDuration, readDuration, unmarshalDuration, len(msgBytes))
	}
	c.recvCount++
	c.mu.Unlock()

	return &msg, nil
}

// Close closes the connection
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isActive = false
	return c.conn.Close()
}

// GetPeerID returns the peer ID
func (c *Connection) GetPeerID() string {
	return c.peerID
}

// IsActive returns whether the connection is active
func (c *Connection) IsActive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isActive
}

// PeerManager manages P2P connections (TCP control + UDP data)
type PeerManager struct {
	connections    map[string]*Connection    // TCP control connections
	udpConnections map[string]*UDPConnection // UDP data connections
	listener       net.Listener
	mu             sync.RWMutex
	onMessage      func(*Connection, *Message)
}

// NewPeerManager creates a new peer manager
func NewPeerManager() *PeerManager {
	return &PeerManager{
		connections:    make(map[string]*Connection),
		udpConnections: make(map[string]*UDPConnection),
	}
}

// StartListening starts listening for incoming peer connections
func (pm *PeerManager) StartListening(port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	pm.listener = listener
	log.Printf("P2P listening on port %d", port)

	go pm.acceptConnections()

	return nil
}

// acceptConnections accepts incoming connections
func (pm *PeerManager) acceptConnections() {
	for {
		conn, err := pm.listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go pm.handleConnection(conn)
	}
}

// handleConnection handles an incoming connection
func (pm *PeerManager) handleConnection(conn net.Conn) {
	// First message should be handshake with peer ID
	p2pConn := NewConnection(conn, "unknown")

	msg, err := p2pConn.ReceiveMessage()
	if err != nil {
		log.Printf("Failed to receive handshake: %v", err)
		conn.Close()
		return
	}

	if msg.Type != "handshake" {
		log.Printf("Expected handshake, got %s", msg.Type)
		conn.Close()
		return
	}

	var handshake struct {
		PeerID string `json:"peer_id"`
	}
	if err := json.Unmarshal(msg.Payload, &handshake); err != nil {
		log.Printf("Failed to parse handshake: %v", err)
		conn.Close()
		return
	}

	p2pConn.peerID = handshake.PeerID

	pm.mu.Lock()
	pm.connections[handshake.PeerID] = p2pConn
	pm.mu.Unlock()

	log.Printf("Accepted connection from peer: %s", handshake.PeerID)

	// Send handshake response
	p2pConn.SendMessage("handshake_ack", map[string]string{"status": "ok"})

	// Handle messages
	pm.handleMessages(p2pConn)
}

// ConnectToPeer connects to a remote peer
func (pm *PeerManager) ConnectToPeer(peerID, ipAddress string, port int, localPeerID string) error {
	conn, err := DialPeer(ipAddress, port, peerID)
	if err != nil {
		return err
	}

	// Send handshake with our peer ID
	if err := conn.SendMessage("handshake", map[string]string{"peer_id": localPeerID}); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	// Wait for handshake ack
	msg, err := conn.ReceiveMessage()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to receive handshake ack: %w", err)
	}

	if msg.Type != "handshake_ack" {
		conn.Close()
		return fmt.Errorf("expected handshake_ack, got %s", msg.Type)
	}

	pm.mu.Lock()
	pm.connections[peerID] = conn
	pm.mu.Unlock()

	log.Printf("Connected to peer: %s at %s:%d", peerID, ipAddress, port)

	// Handle messages
	go pm.handleMessages(conn)

	return nil
}

// ConnectToPeerWithNAT connects to a remote peer using NAT traversal
// Tries multiple candidates from Kademlia-based discovery
func (pm *PeerManager) ConnectToPeerWithNAT(peerID string, candidates []string, localPeerID string) error {
	conn, err := DialPeerWithNAT(candidates, peerID)
	if err != nil {
		return err
	}

	// Send handshake with our peer ID
	if err := conn.SendMessage("handshake", map[string]string{"peer_id": localPeerID}); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	// Wait for handshake ack
	msg, err := conn.ReceiveMessage()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to receive handshake ack: %w", err)
	}

	if msg.Type != "handshake_ack" {
		conn.Close()
		return fmt.Errorf("expected handshake_ack, got %s", msg.Type)
	}

	pm.mu.Lock()
	pm.connections[peerID] = conn
	pm.mu.Unlock()

	log.Printf("Connected to peer via NAT traversal: %s", peerID)

	// Handle messages
	go pm.handleMessages(conn)

	return nil
}

// handleMessages handles incoming messages from a connection
func (pm *PeerManager) handleMessages(conn *Connection) {
	log.Printf("Starting handleMessages goroutine for peer %s", conn.peerID)
	defer func() {
		log.Printf("Stopping handleMessages goroutine for peer %s", conn.peerID)
		pm.mu.Lock()
		delete(pm.connections, conn.peerID)
		pm.mu.Unlock()
		conn.Close()
	}()

	for conn.IsActive() {
		msg, err := conn.ReceiveMessage()
		if err != nil {
			log.Printf("Error receiving message from %s: %v", conn.peerID, err)
			return
		}

		if pm.onMessage != nil {
			pm.onMessage(conn, msg)
		}
	}
}

// SetMessageHandler sets the message handler callback
func (pm *PeerManager) SetMessageHandler(handler func(*Connection, *Message)) {
	pm.onMessage = handler
}

// SendToPeer sends a message to a specific peer
func (pm *PeerManager) SendToPeer(peerID string, msgType string, payload interface{}) error {
	pm.mu.RLock()
	conn, exists := pm.connections[peerID]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("not connected to peer: %s", peerID)
	}

	return conn.SendMessage(msgType, payload)
}

// GetConnections returns list of connected peer IDs
func (pm *PeerManager) GetConnections() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]string, 0, len(pm.connections))
	for peerID := range pm.connections {
		peers = append(peers, peerID)
	}
	return peers
}

// GetConnection returns the connection for a specific peer ID
func (pm *PeerManager) GetConnection(peerID string) *Connection {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.connections[peerID]
}

// Stop stops the peer manager
func (pm *PeerManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Close all TCP connections
	for _, conn := range pm.connections {
		conn.Close()
	}

	// Close all UDP connections
	for _, udpConn := range pm.udpConnections {
		udpConn.Close()
	}

	// Close listener
	if pm.listener != nil {
		return pm.listener.Close()
	}

	return nil
}

// GetUDPConnection returns the UDP connection for a specific peer ID
func (pm *PeerManager) GetUDPConnection(peerID string) *UDPConnection {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.udpConnections[peerID]
}

// SetupUDPConnection creates and configures a UDP connection for a peer
func (pm *PeerManager) SetupUDPConnection(peerID, localIP string, localUDPPort int) (*UDPConnection, error) {
	udpConn, err := NewUDPConnection(localUDPPort, peerID)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %w", err)
	}

	pm.mu.Lock()
	pm.udpConnections[peerID] = udpConn
	pm.mu.Unlock()

	log.Printf("Created UDP connection for peer %s on port %d", peerID, localUDPPort)
	return udpConn, nil
}

// ExchangeUDPEndpoint exchanges UDP endpoint information over TCP control connection
func (pm *PeerManager) ExchangeUDPEndpoint(peerID, localIP string, localUDPPort int, isInitiator bool) error {
	pm.mu.RLock()
	tcpConn, exists := pm.connections[peerID]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no TCP connection found for peer: %s", peerID)
	}

	if isInitiator {
		// Send our UDP endpoint
		err := tcpConn.SendMessage("udp_endpoint", map[string]interface{}{
			"ip":   localIP,
			"port": localUDPPort,
		})
		if err != nil {
			return fmt.Errorf("failed to send UDP endpoint: %w", err)
		}

		// Wait for remote UDP endpoint
		msg, err := tcpConn.ReceiveMessage()
		if err != nil {
			return fmt.Errorf("failed to receive UDP endpoint: %w", err)
		}

		if msg.Type != "udp_endpoint" {
			return fmt.Errorf("expected udp_endpoint, got %s", msg.Type)
		}

		var remoteEndpoint struct {
			IP   string `json:"ip"`
			Port int    `json:"port"`
		}
		if err := json.Unmarshal(msg.Payload, &remoteEndpoint); err != nil {
			return fmt.Errorf("failed to parse UDP endpoint: %w", err)
		}

		// Set remote address on UDP connection
		pm.mu.RLock()
		udpConn, exists := pm.udpConnections[peerID]
		pm.mu.RUnlock()

		if !exists {
			return fmt.Errorf("no UDP connection found for peer: %s", peerID)
		}

		err = udpConn.SetRemoteAddr(remoteEndpoint.IP, remoteEndpoint.Port)
		if err != nil {
			return fmt.Errorf("failed to set remote UDP address: %w", err)
		}

		log.Printf("UDP endpoint exchange complete with %s (%s:%d)", peerID, remoteEndpoint.IP, remoteEndpoint.Port)
	} else {
		// Wait for remote UDP endpoint
		msg, err := tcpConn.ReceiveMessage()
		if err != nil {
			return fmt.Errorf("failed to receive UDP endpoint: %w", err)
		}

		if msg.Type != "udp_endpoint" {
			return fmt.Errorf("expected udp_endpoint, got %s", msg.Type)
		}

		var remoteEndpoint struct {
			IP   string `json:"ip"`
			Port int    `json:"port"`
		}
		if err := json.Unmarshal(msg.Payload, &remoteEndpoint); err != nil {
			return fmt.Errorf("failed to parse UDP endpoint: %w", err)
		}

		// Set remote address on UDP connection
		pm.mu.RLock()
		udpConn, exists := pm.udpConnections[peerID]
		pm.mu.RUnlock()

		if !exists {
			return fmt.Errorf("no UDP connection found for peer: %s", peerID)
		}

		err = udpConn.SetRemoteAddr(remoteEndpoint.IP, remoteEndpoint.Port)
		if err != nil {
			return fmt.Errorf("failed to set remote UDP address: %w", err)
		}

		// Send our UDP endpoint
		err = tcpConn.SendMessage("udp_endpoint", map[string]interface{}{
			"ip":   localIP,
			"port": localUDPPort,
		})
		if err != nil {
			return fmt.Errorf("failed to send UDP endpoint: %w", err)
		}

		log.Printf("UDP endpoint exchange complete with %s (%s:%d)", peerID, remoteEndpoint.IP, remoteEndpoint.Port)
	}

	return nil
}
