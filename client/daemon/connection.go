package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shadowmesh/shadowmesh/shared/protocol"
)

// ConnectionState represents the current state of the WebSocket connection
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateHandshaking
	StateEstablished
	StateReconnecting
	StateClosed
)

func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateHandshaking:
		return "Handshaking"
	case StateEstablished:
		return "Established"
	case StateReconnecting:
		return "Reconnecting"
	case StateClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// ConnectionManager manages a WebSocket connection to a relay server
type ConnectionManager struct {
	// Configuration
	relayURL          string
	tlsSkipVerify     bool
	reconnectInterval time.Duration
	maxReconnects     int

	// Connection state
	conn  *websocket.Conn
	state ConnectionState
	mu    sync.RWMutex

	// Handshake state
	handshakeState *protocol.HandshakeState

	// Channels
	sendChan    chan *protocol.Message
	receiveChan chan *protocol.Message
	errorChan   chan error
	closeChan   chan struct{}

	// Context for lifecycle management
	ctx    context.Context
	cancel context.CancelFunc

	// Callbacks
	onConnected    func()
	onDisconnected func(error)
	onMessage      func(*protocol.Message)
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(relayURL string, tlsSkipVerify bool) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConnectionManager{
		relayURL:          relayURL,
		tlsSkipVerify:     tlsSkipVerify,
		reconnectInterval: 5 * time.Second,
		maxReconnects:     10,
		state:             StateDisconnected,
		sendChan:          make(chan *protocol.Message, 100),
		receiveChan:       make(chan *protocol.Message, 100),
		errorChan:         make(chan error, 10),
		closeChan:         make(chan struct{}),
		ctx:               ctx,
		cancel:            cancel,
	}
}

// SetHandshakeState sets the handshake state for the connection
func (cm *ConnectionManager) SetHandshakeState(hs *protocol.HandshakeState) {
	cm.handshakeState = hs
}

// SetCallbacks sets the callback functions
func (cm *ConnectionManager) SetCallbacks(onConnected func(), onDisconnected func(error), onMessage func(*protocol.Message)) {
	cm.onConnected = onConnected
	cm.onDisconnected = onDisconnected
	cm.onMessage = onMessage
}

// Start initiates the connection and starts the message handling loops
func (cm *ConnectionManager) Start() error {
	if err := cm.connect(); err != nil {
		return fmt.Errorf("initial connection failed: %w", err)
	}

	// Start goroutines for message handling
	go cm.readLoop()
	go cm.writeLoop()
	go cm.heartbeatLoop()

	return nil
}

// Stop gracefully stops the connection
func (cm *ConnectionManager) Stop() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.state == StateClosed {
		return nil
	}

	// Cancel context to stop all goroutines
	cm.cancel()

	// Send CLOSE message
	if cm.conn != nil && cm.state == StateEstablished {
		closeMsg := protocol.NewCloseMessage(protocol.CloseNormalShutdown, "Client shutdown")
		if data, err := protocol.EncodeMessage(closeMsg); err == nil {
			cm.conn.WriteMessage(websocket.BinaryMessage, data)
		}
	}

	// Close WebSocket connection
	if cm.conn != nil {
		cm.conn.Close()
	}

	cm.state = StateClosed
	close(cm.closeChan)

	return nil
}

// connect establishes a WebSocket connection to the relay
func (cm *ConnectionManager) connect() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.state = StateConnecting

	// Parse and validate URL
	u, err := url.Parse(cm.relayURL)
	if err != nil {
		return fmt.Errorf("invalid relay URL: %w", err)
	}

	// Ensure WebSocket scheme
	if u.Scheme == "http" {
		u.Scheme = "ws"
	} else if u.Scheme == "https" {
		u.Scheme = "wss"
	}

	// Dial WebSocket connection
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cm.tlsSkipVerify,
		},
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		cm.state = StateDisconnected
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	cm.conn = conn
	cm.state = StateHandshaking

	return nil
}

// reconnect attempts to reconnect to the relay
func (cm *ConnectionManager) reconnect() error {
	attempts := 0

	for attempts < cm.maxReconnects {
		select {
		case <-cm.ctx.Done():
			return fmt.Errorf("reconnection cancelled")
		case <-time.After(cm.reconnectInterval):
			attempts++

			if err := cm.connect(); err != nil {
				if cm.onDisconnected != nil {
					cm.onDisconnected(err)
				}
				continue
			}

			// Restart read/write loops
			go cm.readLoop()
			go cm.writeLoop()

			return nil
		}
	}

	return fmt.Errorf("max reconnection attempts reached")
}

// SendMessage queues a message for sending
func (cm *ConnectionManager) SendMessage(msg *protocol.Message) error {
	select {
	case cm.sendChan <- msg:
		return nil
	case <-cm.ctx.Done():
		return fmt.Errorf("connection closed")
	default:
		return fmt.Errorf("send queue full")
	}
}

// readLoop continuously reads messages from the WebSocket
func (cm *ConnectionManager) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			cm.errorChan <- fmt.Errorf("read loop panic: %v", r)
		}
	}()

	for {
		select {
		case <-cm.ctx.Done():
			return
		default:
			// Read message from WebSocket
			messageType, data, err := cm.conn.ReadMessage()
			if err != nil {
				cm.errorChan <- fmt.Errorf("read error: %w", err)
				// Attempt reconnection
				if err := cm.reconnect(); err != nil {
					cm.errorChan <- fmt.Errorf("reconnection failed: %w", err)
					return
				}
				continue
			}

			if messageType != websocket.BinaryMessage {
				cm.errorChan <- fmt.Errorf("unexpected message type: %d", messageType)
				continue
			}

			// Decode protocol message
			msg, err := protocol.DecodeMessage(data)
			if err != nil {
				cm.errorChan <- fmt.Errorf("message decode error: %w", err)
				continue
			}

			// Handle the message
			if cm.onMessage != nil {
				cm.onMessage(msg)
			}

			// Also send to receive channel for direct consumption
			select {
			case cm.receiveChan <- msg:
			default:
				cm.errorChan <- fmt.Errorf("receive queue full, dropping message")
			}
		}
	}
}

// writeLoop continuously sends messages to the WebSocket
func (cm *ConnectionManager) writeLoop() {
	defer func() {
		if r := recover(); r != nil {
			cm.errorChan <- fmt.Errorf("write loop panic: %v", r)
		}
	}()

	for {
		select {
		case <-cm.ctx.Done():
			return

		case msg := <-cm.sendChan:
			// Encode message
			data, err := protocol.EncodeMessage(msg)
			if err != nil {
				cm.errorChan <- fmt.Errorf("message encode error: %w", err)
				continue
			}

			// Send to WebSocket
			if err := cm.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				cm.errorChan <- fmt.Errorf("write error: %w", err)
				// Attempt reconnection
				if err := cm.reconnect(); err != nil {
					cm.errorChan <- fmt.Errorf("reconnection failed: %w", err)
					return
				}
			}
		}
	}
}

// heartbeatLoop sends periodic heartbeat messages
func (cm *ConnectionManager) heartbeatLoop() {
	ticker := time.NewTicker(protocol.DefaultHeartbeatInterval)
	defer ticker.Stop()

	missedHeartbeats := 0

	for {
		select {
		case <-cm.ctx.Done():
			return

		case <-ticker.C:
			cm.mu.RLock()
			state := cm.state
			cm.mu.RUnlock()

			if state != StateEstablished {
				continue
			}

			// Send heartbeat
			heartbeat := protocol.NewHeartbeatMessage()
			if err := cm.SendMessage(heartbeat); err != nil {
				missedHeartbeats++
				if missedHeartbeats >= protocol.MaxMissedHeartbeats {
					cm.errorChan <- fmt.Errorf("too many missed heartbeats")
					cm.reconnect()
					missedHeartbeats = 0
				}
			} else {
				missedHeartbeats = 0
			}
		}
	}
}

// GetState returns the current connection state
func (cm *ConnectionManager) GetState() ConnectionState {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.state
}

// SetState updates the connection state
func (cm *ConnectionManager) SetState(state ConnectionState) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.state = state
}

// ReceiveChannel returns the channel for receiving messages
func (cm *ConnectionManager) ReceiveChannel() <-chan *protocol.Message {
	return cm.receiveChan
}

// ErrorChannel returns the channel for receiving errors
func (cm *ConnectionManager) ErrorChannel() <-chan error {
	return cm.errorChan
}

// WaitForClose blocks until the connection is closed
func (cm *ConnectionManager) WaitForClose() {
	<-cm.closeChan
}

// IsConnected returns whether the connection is established
// Returns true if WebSocket is connected and ready for handshake or already established
func (cm *ConnectionManager) IsConnected() bool {
	state := cm.GetState()
	return state == StateHandshaking || state == StateEstablished
}
