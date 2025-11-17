package daemonmgr

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// P2PConnection manages a single WebSocket P2P connection
type P2PConnection struct {
	conn      *websocket.Conn
	connMutex sync.RWMutex

	peerAddr string

	// Channels for frame transmission
	sendChan chan []byte
	recvChan chan []byte

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// State
	connected   bool
	connectedMu sync.RWMutex

	// Connection callback - called when incoming connection is accepted
	onConnectionAccepted func()

	// Relay mode
	relayMode   bool
	relayServer string
	peerID      string
}

// NewP2PConnection creates a new P2P connection
func NewP2PConnection() *P2PConnection {
	ctx, cancel := context.WithCancel(context.Background())

	return &P2PConnection{
		sendChan: make(chan []byte, 1000), // Increased from 100 to handle bursts
		recvChan: make(chan []byte, 1000), // Increased from 100 to handle bursts
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Connect establishes WebSocket connection to peer
func (p *P2PConnection) Connect(peerAddr string) error {
	p.peerAddr = peerAddr

	// Parse peer address
	host, port, err := net.SplitHostPort(peerAddr)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	// Create WebSocket URL (use WS not WSS - frames are already encrypted)
	// Traffic is encrypted at frame level with ChaCha20-Poly1305, so TLS is redundant
	wsURL := fmt.Sprintf("ws://%s:%s/p2p", host, port)

	log.Printf("Connecting to peer WebSocket: %s", wsURL)

	dialer := &websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Establish WebSocket connection
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("WebSocket handshake failed (status %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("WebSocket connection failed: %w", err)
	}
	defer resp.Body.Close()

	p.connMutex.Lock()
	p.conn = conn
	p.connMutex.Unlock()

	p.setConnected(true)

	log.Printf("✅ WebSocket connection established to %s", peerAddr)

	// Start send/receive goroutines
	p.wg.Add(2)
	go p.sendLoop()
	go p.recvLoop()

	return nil
}

// Listen starts WebSocket server for incoming connections
func (p *P2PConnection) Listen(listenAddr string) error {
	// Create HTTP server for WebSocket upgrades
	// No TLS - frames are already encrypted with ChaCha20-Poly1305
	mux := http.NewServeMux()
	mux.HandleFunc("/p2p", p.handleWebSocket)

	// Create IPv4 TCP listener explicitly (not IPv6-only)
	listener, err := net.Listen("tcp4", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to create IPv4 listener: %w", err)
	}

	server := &http.Server{
		Handler: mux,
	}

	// Start server in background
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		log.Printf("WebSocket server listening on %s (IPv4, unencrypted transport, encrypted frames)", listenAddr)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("⚠️  WebSocket server error: %v", err)
		}
	}()

	return nil
}

// SendFrame sends an encrypted frame over WebSocket
func (p *P2PConnection) SendFrame(frame []byte) error {
	if !p.isConnected() {
		return fmt.Errorf("not connected")
	}

	select {
	case p.sendChan <- frame:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("connection closed")
	default:
		return fmt.Errorf("send buffer full")
	}
}

// RecvChannel returns the channel for receiving encrypted frames
func (p *P2PConnection) RecvChannel() <-chan []byte {
	return p.recvChan
}

// Close closes the P2P connection
func (p *P2PConnection) Close() error {
	p.cancel()

	p.connMutex.Lock()
	if p.conn != nil {
		p.conn.Close()
	}
	p.connMutex.Unlock()

	p.setConnected(false)

	p.wg.Wait()

	return nil
}

// sendLoop sends frames over WebSocket
func (p *P2PConnection) sendLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case frame := <-p.sendChan:
			p.connMutex.RLock()
			conn := p.conn
			p.connMutex.RUnlock()

			if conn == nil {
				continue
			}

			// Send binary frame
			if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				log.Printf("⚠️  Failed to send frame: %v", err)
				p.setConnected(false)
				return
			}
		}
	}
}

// recvLoop receives frames from WebSocket
func (p *P2PConnection) recvLoop() {
	defer p.wg.Done()

	for {
		p.connMutex.RLock()
		conn := p.conn
		p.connMutex.RUnlock()

		if conn == nil {
			return
		}

		// Read frame
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("⚠️  WebSocket read error: %v", err)
			p.setConnected(false)
			return
		}

		if msgType != websocket.BinaryMessage {
			log.Printf("⚠️  Unexpected message type: %d", msgType)
			continue
		}

		// Send to receive channel
		select {
		case p.recvChan <- data:
		case <-p.ctx.Done():
			return
		default:
			log.Printf("⚠️  Receive buffer full, dropping frame")
		}
	}
}

// handleWebSocket handles incoming WebSocket connections
func (p *P2PConnection) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  2 * 1024 * 1024, // 2MB (increased from 4KB for iperf3 burst traffic)
		WriteBufferSize: 2 * 1024 * 1024, // 2MB (prevents "send buffer full" errors)
		CheckOrigin: func(r *http.Request) bool {
			return true // Accept all origins (TODO: proper validation)
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("⚠️  WebSocket upgrade failed: %v", err)
		return
	}

	log.Printf("✅ Incoming WebSocket connection from %s", r.RemoteAddr)

	p.connMutex.Lock()
	p.conn = conn
	p.peerAddr = r.RemoteAddr
	p.connMutex.Unlock()

	p.setConnected(true)

	// Start send/receive goroutines
	p.wg.Add(2)
	go p.sendLoop()
	go p.recvLoop()

	// Trigger callback to notify DaemonManager to start frame router
	if p.onConnectionAccepted != nil {
		p.onConnectionAccepted()
	}
}

// isConnected returns connection status
func (p *P2PConnection) isConnected() bool {
	p.connectedMu.RLock()
	defer p.connectedMu.RUnlock()
	return p.connected
}

// setConnected sets connection status
func (p *P2PConnection) setConnected(connected bool) {
	p.connectedMu.Lock()
	p.connected = connected
	p.connectedMu.Unlock()
}

// SetOnConnectionAccepted sets the callback for when incoming connection is accepted
func (p *P2PConnection) SetOnConnectionAccepted(callback func()) {
	p.onConnectionAccepted = callback
}

// EnableRelayMode configures the connection to use a relay server
func (p *P2PConnection) EnableRelayMode(relayServer, peerID string) {
	p.relayMode = true
	p.relayServer = relayServer
	p.peerID = peerID
}

// ConnectViaRelay establishes WebSocket connection to relay server
func (p *P2PConnection) ConnectViaRelay() error {
	if !p.relayMode {
		return fmt.Errorf("relay mode not enabled")
	}

	// Build relay URL with peer ID
	relayURL := fmt.Sprintf("%s?peer_id=%s", p.relayServer, p.peerID)
	log.Printf("Connecting to relay server: %s", relayURL)

	dialer := &websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Establish WebSocket connection to relay
	conn, resp, err := dialer.Dial(relayURL, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("relay connection failed (status %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("relay connection failed: %w", err)
	}
	defer resp.Body.Close()

	p.connMutex.Lock()
	p.conn = conn
	p.peerAddr = relayURL
	p.connMutex.Unlock()

	p.setConnected(true)

	log.Printf("✅ Connected to relay server as peer %s", p.peerID)

	// Start send/receive goroutines
	p.wg.Add(2)
	go p.sendLoop()
	go p.recvLoop()

	return nil
}

// generateSelfSignedCert generates a self-signed TLS certificate
func generateSelfSignedCert() (tls.Certificate, error) {
	// Generate RSA 4096-bit key (satisfies strict crypto policies)
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // 1 year validity

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"ShadowMesh P2P"},
			CommonName:   "ShadowMesh Peer",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Self-sign the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Convert to tls.Certificate
	tlsCert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
	}

	return tlsCert, nil
}
