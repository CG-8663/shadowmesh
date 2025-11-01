package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shadowmesh/shadowmesh/shared/crypto"
	"github.com/shadowmesh/shadowmesh/shared/protocol"
)

// ClientState represents the state of a client connection
type ClientState int

const (
	ClientStateConnecting ClientState = iota
	ClientStateHandshaking
	ClientStateEstablished
	ClientStateDisconnecting
	ClientStateDisconnected
)

func (cs ClientState) String() string {
	switch cs {
	case ClientStateConnecting:
		return "CONNECTING"
	case ClientStateHandshaking:
		return "HANDSHAKING"
	case ClientStateEstablished:
		return "ESTABLISHED"
	case ClientStateDisconnecting:
		return "DISCONNECTING"
	case ClientStateDisconnected:
		return "DISCONNECTED"
	default:
		return "UNKNOWN"
	}
}

// ClientConnection represents a connected client
type ClientConnection struct {
	// Connection info
	conn         *websocket.Conn
	clientID     [32]byte
	state        ClientState
	stateMutex   sync.RWMutex

	// Session info
	sessionID    [16]byte
	sessionKeys  *SessionKeys

	// Frame encryption (persistent encryptors)
	txEncryptor  *crypto.FrameEncryptor
	rxEncryptor  *crypto.FrameEncryptor

	// Communication channels
	sendChan     chan *protocol.Message
	receiveChan  chan *protocol.Message

	// Lifecycle
	ctx          context.Context
	cancel       context.CancelFunc
	closeChan    chan struct{}
	closeOnce    sync.Once

	// Statistics
	connectedAt  time.Time
	lastHeartbeat time.Time
	framesSent   atomic.Uint64
	framesRecv   atomic.Uint64
	bytesSent    atomic.Uint64
	bytesRecv    atomic.Uint64
}

// ConnectionManager manages all client connections
type ConnectionManager struct {
	// Server config
	config       *Config
	listenAddr   string

	// HTTP server
	httpServer   *http.Server
	upgrader     websocket.Upgrader

	// Client management
	clients      map[[32]byte]*ClientConnection // clientID -> connection
	clientsMutex sync.RWMutex

	// Handshake handler (injected)
	handshakeHandler HandshakeHandler

	// Router (injected)
	router       *Router

	// Lifecycle
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup

	// Statistics
	totalConnections atomic.Uint64
	activeConnections atomic.Int64
}

// HandshakeHandler interface for processing handshakes
type HandshakeHandler interface {
	HandleHandshake(ctx context.Context, client *ClientConnection) error
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(config *Config) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())

	cm := &ConnectionManager{
		config:     config,
		listenAddr: config.Server.ListenAddr,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.Limits.ReadBufferSize,
			WriteBufferSize: config.Limits.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins (adjust for production)
			},
		},
		clients: make(map[[32]byte]*ClientConnection),
		ctx:     ctx,
		cancel:  cancel,
	}

	return cm
}

// SetHandshakeHandler sets the handshake handler
func (cm *ConnectionManager) SetHandshakeHandler(handler HandshakeHandler) {
	cm.handshakeHandler = handler
}

// SetRouter sets the frame router
func (cm *ConnectionManager) SetRouter(router *Router) {
	cm.router = router
}

// Start starts the WebSocket server
func (cm *ConnectionManager) Start() error {
	log.Printf("Starting relay server on %s", cm.listenAddr)

	// Create HTTP handler
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", cm.handleWebSocket)
	mux.HandleFunc("/health", cm.handleHealth)
	mux.HandleFunc("/stats", cm.handleStats)

	// Create HTTP server
	cm.httpServer = &http.Server{
		Addr:         cm.listenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Configure TLS if enabled
	if cm.config.Server.TLS.Enabled {
		certFile, keyFile, err := cm.config.GetTLSFiles()
		if err != nil {
			return fmt.Errorf("TLS configuration error: %w", err)
		}

		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS13,
			CurvePreferences: []tls.CurveID{
				tls.X25519,
				tls.CurveP256,
			},
		}
		cm.httpServer.TLSConfig = tlsConfig

		// Start heartbeat monitor
		cm.wg.Add(1)
		go cm.heartbeatMonitor()

		log.Println("Starting HTTPS server with TLS 1.3")
		return cm.httpServer.ListenAndServeTLS(certFile, keyFile)
	}

	// Start heartbeat monitor
	cm.wg.Add(1)
	go cm.heartbeatMonitor()

	log.Println("WARNING: Starting HTTP server without TLS (not recommended for production)")
	return cm.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (cm *ConnectionManager) Stop() error {
	log.Println("Stopping relay server...")

	// Cancel context
	cm.cancel()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := cm.httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Close all client connections
	cm.clientsMutex.Lock()
	for _, client := range cm.clients {
		client.Close()
	}
	cm.clientsMutex.Unlock()

	// Wait for goroutines
	cm.wg.Wait()

	log.Println("Relay server stopped")
	return nil
}

// handleWebSocket handles new WebSocket connections
func (cm *ConnectionManager) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check if we're at capacity
	if int(cm.activeConnections.Load()) >= cm.config.Limits.MaxClients {
		http.Error(w, "Server at capacity", http.StatusServiceUnavailable)
		log.Println("Rejected connection: server at capacity")
		return
	}

	// Upgrade connection
	conn, err := cm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create client connection
	client := cm.newClientConnection(conn)

	// Update statistics
	cm.totalConnections.Add(1)
	cm.activeConnections.Add(1)

	log.Printf("New connection from %s (total: %d, active: %d)",
		r.RemoteAddr,
		cm.totalConnections.Load(),
		cm.activeConnections.Load())

	// Handle client in separate goroutine
	cm.wg.Add(1)
	go cm.handleClient(client)
}

// newClientConnection creates a new client connection
func (cm *ConnectionManager) newClientConnection(conn *websocket.Conn) *ClientConnection {
	ctx, cancel := context.WithCancel(cm.ctx)

	return &ClientConnection{
		conn:          conn,
		state:         ClientStateConnecting,
		sendChan:      make(chan *protocol.Message, 100),
		receiveChan:   make(chan *protocol.Message, 100),
		ctx:           ctx,
		cancel:        cancel,
		closeChan:     make(chan struct{}),
		connectedAt:   time.Now(),
		lastHeartbeat: time.Now(),
	}
}

// handleClient handles a client connection lifecycle
func (cm *ConnectionManager) handleClient(client *ClientConnection) {
	defer cm.wg.Done()
	defer cm.activeConnections.Add(-1)
	defer client.Close()

	// Start read/write loops
	go client.readLoop(cm)
	go client.writeLoop()

	// Set handshake timeout
	handshakeCtx, handshakeCancel := context.WithTimeout(
		client.ctx,
		time.Duration(cm.config.Limits.HandshakeTimeout)*time.Second,
	)
	defer handshakeCancel()

	// Update state
	client.setState(ClientStateHandshaking)

	// Perform handshake
	if cm.handshakeHandler != nil {
		if err := cm.handshakeHandler.HandleHandshake(handshakeCtx, client); err != nil {
			log.Printf("Handshake failed: %v", err)
			return
		}
	} else {
		log.Println("WARNING: No handshake handler configured")
		return
	}

	// Update state to established
	client.setState(ClientStateEstablished)

	// Register client
	cm.registerClient(client)
	defer cm.unregisterClient(client)

	log.Printf("Client %x established (session: %x)", client.clientID[:8], client.sessionID[:8])

	// Process messages until disconnection
	for {
		select {
		case <-client.ctx.Done():
			return
		case msg, ok := <-client.receiveChan:
			if !ok {
				return
			}
			cm.handleClientMessage(client, msg)
		}
	}
}

// handleClientMessage handles a message from a client
func (cm *ConnectionManager) handleClientMessage(client *ClientConnection, msg *protocol.Message) {
	switch msg.Header.Type {
	case protocol.MsgTypeDataFrame:
		// Route data frame to destination
		if cm.router != nil {
			cm.router.RouteFrame(client, msg)
		}
		client.framesRecv.Add(1)

	case protocol.MsgTypeHeartbeat:
		// Update last heartbeat time
		client.lastHeartbeat = time.Now()

		// Send heartbeat response
		response := protocol.NewHeartbeatMessage()
		select {
		case client.sendChan <- response:
		default:
			log.Printf("Failed to send heartbeat response to client %x", client.clientID[:8])
		}

	// Future: Add key rotation handling
	// case protocol.MsgTypeKeyRotation:
	// 	log.Printf("Key rotation request from client %x", client.clientID[:8])

	default:
		log.Printf("Unexpected message type %d from client %x", msg.Header.Type, client.clientID[:8])
	}
}

// registerClient registers a client in the active clients map
func (cm *ConnectionManager) registerClient(client *ClientConnection) {
	cm.clientsMutex.Lock()
	defer cm.clientsMutex.Unlock()

	cm.clients[client.clientID] = client
	log.Printf("Registered client %x (total clients: %d)", client.clientID[:8], len(cm.clients))
}

// unregisterClient removes a client from the active clients map
func (cm *ConnectionManager) unregisterClient(client *ClientConnection) {
	cm.clientsMutex.Lock()
	defer cm.clientsMutex.Unlock()

	delete(cm.clients, client.clientID)
	log.Printf("Unregistered client %x (remaining clients: %d)", client.clientID[:8], len(cm.clients))
}

// heartbeatMonitor monitors client heartbeats and disconnects stale clients
func (cm *ConnectionManager) heartbeatMonitor() {
	defer cm.wg.Done()

	ticker := time.NewTicker(time.Duration(cm.config.Limits.HeartbeatInterval) * time.Second)
	defer ticker.Stop()

	timeoutDuration := time.Duration(cm.config.Limits.HeartbeatTimeout) * time.Second

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.clientsMutex.RLock()
			now := time.Now()
			for clientID, client := range cm.clients {
				if client.getState() == ClientStateEstablished {
					if now.Sub(client.lastHeartbeat) > timeoutDuration {
						log.Printf("Client %x heartbeat timeout, disconnecting", clientID[:8])
						client.Close()
					}
				}
			}
			cm.clientsMutex.RUnlock()
		}
	}
}

// handleHealth handles health check endpoint
func (cm *ConnectionManager) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","active_clients":%d}`, cm.activeConnections.Load())
}

// handleStats handles statistics endpoint
func (cm *ConnectionManager) handleStats(w http.ResponseWriter, r *http.Request) {
	cm.clientsMutex.RLock()
	defer cm.clientsMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"total_connections":%d,"active_connections":%d,"registered_clients":%d}`,
		cm.totalConnections.Load(),
		cm.activeConnections.Load(),
		len(cm.clients))
}

// GetClient retrieves a client by ID
func (cm *ConnectionManager) GetClient(clientID [32]byte) (*ClientConnection, bool) {
	cm.clientsMutex.RLock()
	defer cm.clientsMutex.RUnlock()

	client, ok := cm.clients[clientID]
	return client, ok
}

// readLoop reads messages from the WebSocket connection
func (cc *ClientConnection) readLoop(cm *ConnectionManager) {
	defer close(cc.receiveChan)

	for {
		select {
		case <-cc.ctx.Done():
			return
		default:
		}

		// Read message from WebSocket
		messageType, data, err := cc.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("Read error: %v", err)
			}
			return
		}

		if messageType != websocket.BinaryMessage {
			log.Printf("Unexpected message type: %d", messageType)
			continue
		}

		// Decode protocol message
		msg, err := protocol.DecodeMessage(data)
		if err != nil {
			log.Printf("Message decode error: %v", err)
			continue
		}

		// Update statistics
		cc.bytesRecv.Add(uint64(msg.Header.Length))

		// Send to receive channel
		select {
		case cc.receiveChan <- msg:
		case <-cc.ctx.Done():
			return
		}
	}
}

// writeLoop writes messages to the WebSocket connection
func (cc *ClientConnection) writeLoop() {
	for {
		select {
		case <-cc.ctx.Done():
			return
		case msg, ok := <-cc.sendChan:
			if !ok {
				return
			}

			// Encode message
			data, err := protocol.EncodeMessage(msg)
			if err != nil {
				log.Printf("Message encode error: %v", err)
				continue
			}

			// Write message to WebSocket
			if err := cc.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				log.Printf("Write error: %v", err)
				return
			}

			// Update statistics
			cc.framesSent.Add(1)
			cc.bytesSent.Add(uint64(msg.Header.Length))
		}
	}
}

// SendMessage sends a message to the client
func (cc *ClientConnection) SendMessage(msg *protocol.Message) error {
	select {
	case cc.sendChan <- msg:
		return nil
	case <-cc.ctx.Done():
		return fmt.Errorf("client disconnected")
	default:
		return fmt.Errorf("send buffer full")
	}
}

// Close closes the client connection
func (cc *ClientConnection) Close() {
	cc.closeOnce.Do(func() {
		cc.setState(ClientStateDisconnecting)
		cc.cancel()
		cc.conn.Close()
		close(cc.closeChan)
		cc.setState(ClientStateDisconnected)
	})
}

// setState safely updates the client state
func (cc *ClientConnection) setState(state ClientState) {
	cc.stateMutex.Lock()
	defer cc.stateMutex.Unlock()
	cc.state = state
}

// getState safely retrieves the client state
func (cc *ClientConnection) getState() ClientState {
	cc.stateMutex.RLock()
	defer cc.stateMutex.RUnlock()
	return cc.state
}

// SessionKeys holds the encryption keys for a session
type SessionKeys struct {
	TXKey [32]byte
	RXKey [32]byte
}
