package networking

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shadowmesh/shadowmesh/shared/protocol"
)

// TransportConfig contains configuration for the WebSocket transport
type TransportConfig struct {
	URL              string        // WebSocket URL (ws:// or wss://)
	TLSConfig        *tls.Config   // TLS configuration for WSS
	HandshakeTimeout time.Duration // Timeout for WebSocket handshake
	ReadTimeout      time.Duration // Timeout for read operations
	WriteTimeout     time.Duration // Timeout for write operations
	PingInterval     time.Duration // Interval for sending ping frames
	MaxMessageSize   int64         // Maximum message size
}

// Transport manages WebSocket connections for ShadowMesh
type Transport struct {
	config TransportConfig
	conn   *websocket.Conn

	// Message channels
	recvChan chan *protocol.Message
	sendChan chan *protocol.Message
	errChan  chan error

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// Connection state
	connected bool
}

// DefaultTransportConfig returns default configuration
func DefaultTransportConfig() TransportConfig {
	return TransportConfig{
		HandshakeTimeout: 10 * time.Second,
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     10 * time.Second,
		PingInterval:     20 * time.Second,
		MaxMessageSize:   protocol.MaxMessageSize,
	}
}

// NewTransport creates a new WebSocket transport
func NewTransport(config TransportConfig) *Transport {
	ctx, cancel := context.WithCancel(context.Background())

	return &Transport{
		config:    config,
		recvChan:  make(chan *protocol.Message, 100),
		sendChan:  make(chan *protocol.Message, 100),
		errChan:   make(chan error, 10),
		ctx:       ctx,
		cancel:    cancel,
		connected: false,
	}
}

// Connect establishes WebSocket connection to remote peer
func (t *Transport) Connect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return fmt.Errorf("already connected")
	}

	// Parse WebSocket URL
	u, err := url.Parse(t.config.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Configure WebSocket dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: t.config.HandshakeTimeout,
		TLSClientConfig:  t.config.TLSConfig,
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := &net.Dialer{Timeout: t.config.HandshakeTimeout}
			return d.DialContext(ctx, network, addr)
		},
	}

	// Establish WebSocket connection
	conn, _, err := dialer.DialContext(t.ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Configure connection limits
	conn.SetReadLimit(t.config.MaxMessageSize)

	t.conn = conn
	t.connected = true

	// Start read/write loops
	t.wg.Add(3)
	go t.readLoop()
	go t.writeLoop()
	go t.pingLoop()

	return nil
}

// Close gracefully closes the transport
func (t *Transport) Close() error {
	t.mu.Lock()
	if !t.connected {
		t.mu.Unlock()
		return nil
	}
	t.mu.Unlock()

	// Cancel context to stop loops
	t.cancel()

	// Wait for goroutines to finish
	t.wg.Wait()

	// Close WebSocket connection
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn != nil {
		// Send close frame
		closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "closing")
		_ = t.conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(time.Second))

		err := t.conn.Close()
		t.conn = nil
		t.connected = false

		// Close channels
		close(t.recvChan)
		close(t.sendChan)
		close(t.errChan)

		return err
	}

	return nil
}

// Send queues a message for transmission
func (t *Transport) Send(msg *protocol.Message) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connected {
		return fmt.Errorf("not connected")
	}

	select {
	case t.sendChan <- msg:
		return nil
	case <-t.ctx.Done():
		return fmt.Errorf("transport closed")
	default:
		return fmt.Errorf("send channel full")
	}
}

// Receive returns the channel for incoming messages
func (t *Transport) Receive() <-chan *protocol.Message {
	return t.recvChan
}

// Errors returns the channel for transport errors
func (t *Transport) Errors() <-chan error {
	return t.errChan
}

// IsConnected returns connection status
func (t *Transport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected
}

// readLoop continuously reads messages from WebSocket
func (t *Transport) readLoop() {
	defer t.wg.Done()

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		// Set read deadline
		if t.config.ReadTimeout > 0 {
			_ = t.conn.SetReadDeadline(time.Now().Add(t.config.ReadTimeout))
		}

		// Read message from WebSocket
		_, data, err := t.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				select {
				case t.errChan <- fmt.Errorf("read error: %w", err):
				default:
				}
			}
			return
		}

		// Decode protocol message
		msg, err := protocol.DecodeMessage(data)
		if err != nil {
			select {
			case t.errChan <- fmt.Errorf("decode error: %w", err):
			default:
			}
			continue
		}

		// Send to receive channel
		select {
		case t.recvChan <- msg:
		case <-t.ctx.Done():
			return
		default:
			// Channel full, drop message
			select {
			case t.errChan <- fmt.Errorf("receive channel full, dropping message"):
			default:
			}
		}
	}
}

// writeLoop continuously writes messages to WebSocket
func (t *Transport) writeLoop() {
	defer t.wg.Done()

	for {
		select {
		case <-t.ctx.Done():
			return

		case msg := <-t.sendChan:
			// Encode message
			data, err := protocol.EncodeMessage(msg)
			if err != nil {
				select {
				case t.errChan <- fmt.Errorf("encode error: %w", err):
				default:
				}
				continue
			}

			// Set write deadline
			if t.config.WriteTimeout > 0 {
				_ = t.conn.SetWriteDeadline(time.Now().Add(t.config.WriteTimeout))
			}

			// Write to WebSocket
			if err := t.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				select {
				case t.errChan <- fmt.Errorf("write error: %w", err):
				default:
				}
				return
			}
		}
	}
}

// pingLoop sends periodic ping frames to keep connection alive
func (t *Transport) pingLoop() {
	defer t.wg.Done()

	ticker := time.NewTicker(t.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return

		case <-ticker.C:
			// Send ping frame
			if err := t.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second)); err != nil {
				select {
				case t.errChan <- fmt.Errorf("ping error: %w", err):
				default:
				}
				return
			}
		}
	}
}

// RemoteAddr returns the remote address of the connection
func (t *Transport) RemoteAddr() net.Addr {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.conn != nil {
		return t.conn.RemoteAddr()
	}
	return nil
}

// LocalAddr returns the local address of the connection
func (t *Transport) LocalAddr() net.Addr {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.conn != nil {
		return t.conn.LocalAddr()
	}
	return nil
}
