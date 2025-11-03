package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shadowmesh/shadowmesh/shared/protocol"
)

// P2PConnectionManager handles direct peer-to-peer WebSocket connections
type P2PConnectionManager struct {
	config    P2PConfig
	mode      string // "listener" or "connector"
	conn      *websocket.Conn
	sendChan  chan *protocol.Message
	recvChan  chan *protocol.Message
	errorChan chan error
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	// Callbacks
	onConnect    func()
	onDisconnect func(error)
	onMessage    func(*protocol.Message)

	mu         sync.RWMutex
	isRunning  bool
	listener   net.Listener
}

// NewP2PConnectionManager creates a new P2P connection manager
func NewP2PConnectionManager(config P2PConfig, mode string) *P2PConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &P2PConnectionManager{
		config:    config,
		mode:      mode,
		sendChan:  make(chan *protocol.Message, 100),
		recvChan:  make(chan *protocol.Message, 100),
		errorChan: make(chan error, 10),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// SetCallbacks sets the connection callbacks
func (p *P2PConnectionManager) SetCallbacks(onConnect func(), onDisconnect func(error), onMessage func(*protocol.Message)) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.onConnect = onConnect
	p.onDisconnect = onDisconnect
	p.onMessage = onMessage
}

// Start starts the connection manager
func (p *P2PConnectionManager) Start() error {
	p.mu.Lock()
	if p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("already running")
	}
	p.isRunning = true
	p.mu.Unlock()

	switch p.mode {
	case "listener":
		return p.startListener()
	case "connector":
		return p.startConnector()
	default:
		return fmt.Errorf("invalid mode: %s", p.mode)
	}
}

// startListener starts in listener mode (waits for incoming connections)
func (p *P2PConnectionManager) startListener() error {
	// Create TLS config if enabled
	var tlsConfig *tls.Config
	if p.config.TLSEnabled {
		cert, err := tls.LoadX509KeyPair(p.config.TLSCertFile, p.config.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificate: %w", err)
		}

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS13,
		}
	}

	// Create HTTP server for WebSocket upgrade
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleWebSocketUpgrade)

	server := &http.Server{
		Addr:      p.config.ListenAddress,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	// Start listening
	var listener net.Listener
	var err error

	if p.config.TLSEnabled {
		listener, err = tls.Listen("tcp", p.config.ListenAddress, tlsConfig)
	} else {
		listener, err = net.Listen("tcp", p.config.ListenAddress)
	}

	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	p.listener = listener
	log.Printf("Listening for P2P connections on %s (TLS: %v)", p.config.ListenAddress, p.config.TLSEnabled)

	// Serve in background
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			select {
			case p.errorChan <- fmt.Errorf("server error: %w", err):
			default:
			}
		}
	}()

	return nil
}

// handleWebSocketUpgrade handles WebSocket upgrade requests
func (p *P2PConnectionManager) handleWebSocketUpgrade(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Accept all origins for P2P
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	log.Printf("Peer connected from %s", r.RemoteAddr)

	// Set the connection
	p.mu.Lock()
	p.conn = conn
	p.mu.Unlock()

	// Trigger callback
	if p.onConnect != nil {
		p.onConnect()
	}

	// Start message handlers
	p.startMessageHandlers()
}

// startConnector starts in connector mode (connects to a peer)
func (p *P2PConnectionManager) startConnector() error {
	p.wg.Add(1)
	go p.connectLoop()
	return nil
}

// connectLoop attempts to connect to the peer with reconnection logic
func (p *P2PConnectionManager) connectLoop() {
	defer p.wg.Done()

	attempt := 0
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		attempt++
		log.Printf("Connecting to peer at %s (attempt %d)...", p.config.PeerAddress, attempt)

		// Build WebSocket URL
		scheme := "ws"
		if p.config.TLSEnabled {
			scheme = "wss"
		}
		url := fmt.Sprintf("%s://%s/", scheme, p.config.PeerAddress)

		// Configure dialer
		dialer := websocket.Dialer{
			HandshakeTimeout: p.config.HandshakeTimeout,
		}

		if p.config.TLSSkipVerify {
			dialer.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}

		// Attempt connection
		conn, _, err := dialer.Dial(url, nil)
		if err != nil {
			log.Printf("Connection failed: %v", err)

			// Check if we should retry
			if p.config.MaxReconnects > 0 && attempt >= p.config.MaxReconnects {
				select {
				case p.errorChan <- fmt.Errorf("max reconnection attempts reached"):
				default:
				}
				return
			}

			// Wait before retry
			select {
			case <-p.ctx.Done():
				return
			case <-time.After(p.config.ReconnectInterval):
				continue
			}
		}

		log.Printf("Connected to peer at %s", p.config.PeerAddress)

		// Set connection
		p.mu.Lock()
		p.conn = conn
		p.mu.Unlock()

		// Trigger callback
		if p.onConnect != nil {
			p.onConnect()
		}

		// Start message handlers (this blocks until disconnection)
		p.startMessageHandlers()

		// If we got here, we disconnected
		log.Println("Disconnected from peer")

		// Trigger callback
		if p.onDisconnect != nil {
			p.onDisconnect(fmt.Errorf("connection lost"))
		}

		// Reset for reconnection
		attempt = 0

		// Wait before reconnecting
		select {
		case <-p.ctx.Done():
			return
		case <-time.After(p.config.ReconnectInterval):
		}
	}
}

// startMessageHandlers starts the send and receive loops
func (p *P2PConnectionManager) startMessageHandlers() {
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		p.receiveLoop()
	}()
	go func() {
		defer wg.Done()
		p.sendLoop()
	}()

	wg.Wait()
}

// sendLoop sends messages from the channel to the WebSocket
func (p *P2PConnectionManager) sendLoop() {
	for {
		select {
		case <-p.ctx.Done():
			return

		case msg := <-p.sendChan:
			p.mu.RLock()
			conn := p.conn
			p.mu.RUnlock()

			if conn == nil {
				continue
			}

			// Serialize message
			data, err := protocol.EncodeMessage(msg)
			if err != nil {
				select {
				case p.errorChan <- fmt.Errorf("failed to encode message: %w", err):
				default:
				}
				continue
			}

			// Send binary message
			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				select {
				case p.errorChan <- fmt.Errorf("failed to send message: %w", err):
				default:
				}
				return
			}
		}
	}
}

// receiveLoop receives messages from the WebSocket
func (p *P2PConnectionManager) receiveLoop() {
	p.mu.RLock()
	conn := p.conn
	p.mu.RUnlock()

	if conn == nil {
		return
	}

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		// Read message
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			select {
			case p.errorChan <- fmt.Errorf("read error: %w", err):
			default:
			}
			return
		}

		if messageType != websocket.BinaryMessage {
			continue
		}

		// Deserialize message
		msg, err := protocol.DecodeMessage(data)
		if err != nil {
			select {
			case p.errorChan <- fmt.Errorf("failed to decode message: %w", err):
			default:
			}
			continue
		}

		// Deliver message
		if p.onMessage != nil {
			p.onMessage(msg)
		} else {
			select {
			case p.recvChan <- msg:
			default:
				// Channel full, drop message
			}
		}
	}
}

// SendMessage sends a message to the peer
func (p *P2PConnectionManager) SendMessage(msg *protocol.Message) error {
	select {
	case p.sendChan <- msg:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("connection closed")
	default:
		return fmt.Errorf("send channel full")
	}
}

// ReceiveChannel returns the receive channel
func (p *P2PConnectionManager) ReceiveChannel() <-chan *protocol.Message {
	return p.recvChan
}

// ErrorChannel returns the error channel
func (p *P2PConnectionManager) ErrorChannel() <-chan error {
	return p.errorChan
}

// Stop gracefully stops the connection manager
func (p *P2PConnectionManager) Stop() error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return nil
	}
	p.isRunning = false
	p.mu.Unlock()

	// Cancel context
	p.cancel()

	// Close connection
	p.mu.Lock()
	if p.conn != nil {
		p.conn.Close()
	}
	if p.listener != nil {
		p.listener.Close()
	}
	p.mu.Unlock()

	// Wait for goroutines
	p.wg.Wait()

	// Close channels
	close(p.sendChan)
	close(p.recvChan)
	close(p.errorChan)

	return nil
}

// IsConnected returns whether the manager is connected
func (p *P2PConnectionManager) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.conn != nil
}
