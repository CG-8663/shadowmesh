package transport

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/shadowmesh/shadowmesh/pkg/crypto"
)

// QUICTransport manages QUIC listener and multiple peer connections
type QUICTransport struct {
	listener    *quic.Listener
	connections map[string]*QUICConnection
	connMux     sync.RWMutex
	tlsConfig   *tls.Config
	quicConfig  *quic.Config
}

// QUICConnection represents a single peer connection over QUIC
type QUICConnection struct {
	conn      *quic.Conn
	stream    *quic.Stream // Bidirectional stream for data transfer
	peerID    string
	cipher    *crypto.ChaCha20Cipher
	sendChan  chan []byte
	closeChan chan struct{}
	closed    bool
	closeMux  sync.Mutex
}

// NewQUICTransport creates a QUIC listener on the specified address
func NewQUICTransport(addr string, tlsConfig *tls.Config) (*QUICTransport, error) {
	// Create UDP listener
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP listener: %w", err)
	}

	// Configure QUIC parameters
	quicConfig := &quic.Config{
		MaxIncomingStreams:    1, // One bidirectional stream per connection
		MaxIncomingUniStreams: 0, // No unidirectional streams
		KeepAlivePeriod:       10 * time.Second,
		MaxIdleTimeout:        30 * time.Second,
	}

	// Create QUIC listener
	listener, err := quic.Listen(udpConn, tlsConfig, quicConfig)
	if err != nil {
		udpConn.Close()
		return nil, fmt.Errorf("failed to create QUIC listener: %w", err)
	}

	log.Printf("[QUIC-TRANSPORT] Listening on %s", addr)

	return &QUICTransport{
		listener:    listener,
		connections: make(map[string]*QUICConnection),
		tlsConfig:   tlsConfig,
		quicConfig:  quicConfig,
	}, nil
}

// AcceptConnection waits for and accepts an incoming QUIC connection
func (t *QUICTransport) AcceptConnection(ctx context.Context) (*QUICConnection, error) {
	conn, err := t.listener.Accept(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to accept QUIC connection: %w", err)
	}

	// Accept bidirectional stream
	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		conn.CloseWithError(1, "failed to accept stream")
		return nil, fmt.Errorf("failed to accept stream: %w", err)
	}

	qConn := &QUICConnection{
		conn:      conn,
		stream:    stream,
		sendChan:  make(chan []byte, 100),
		closeChan: make(chan struct{}),
	}

	log.Printf("[QUIC-TRANSPORT] Accepted connection from %s", conn.RemoteAddr())

	return qConn, nil
}

// DialConnection establishes an outbound QUIC connection to a peer
func (t *QUICTransport) DialConnection(ctx context.Context, addr string, peerID string) (*QUICConnection, error) {
	// Dial QUIC connection
	conn, err := quic.DialAddr(ctx, addr, t.tlsConfig, t.quicConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial QUIC connection: %w", err)
	}

	// Open bidirectional stream
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		conn.CloseWithError(1, "failed to open stream")
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}

	qConn := &QUICConnection{
		conn:      conn,
		stream:    stream,
		peerID:    peerID,
		sendChan:  make(chan []byte, 100),
		closeChan: make(chan struct{}),
	}

	// Store connection
	t.connMux.Lock()
	t.connections[peerID] = qConn
	t.connMux.Unlock()

	log.Printf("[QUIC-TRANSPORT] Connected to peer %s at %s", peerID, addr)

	return qConn, nil
}

// SetCipher installs ChaCha20-Poly1305 cipher for this connection
func (c *QUICConnection) SetCipher(cipher *crypto.ChaCha20Cipher) {
	c.cipher = cipher
	log.Printf("[QUIC-CRYPTO] Encryption enabled for peer %s", c.peerID)
}

// SetPeerID sets the peer ID (used for incoming connections after handshake)
func (c *QUICConnection) SetPeerID(peerID string) {
	c.peerID = peerID
}

// SendFrame sends an encrypted frame to the peer
// Frame format: [4 bytes length][encrypted data]
func (c *QUICConnection) SendFrame(frame []byte) error {
	c.closeMux.Lock()
	if c.closed {
		c.closeMux.Unlock()
		return fmt.Errorf("connection closed")
	}
	c.closeMux.Unlock()

	// Encrypt frame if cipher is available
	var dataToSend []byte
	if c.cipher != nil {
		encrypted, err := c.cipher.Encrypt(frame)
		if err != nil {
			return fmt.Errorf("encryption failed: %w", err)
		}
		dataToSend = encrypted
	} else {
		dataToSend = frame
	}

	// Length-prefix framing: [4 bytes length][data]
	lengthPrefix := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthPrefix, uint32(len(dataToSend)))

	// Write length prefix
	if _, err := c.stream.Write(lengthPrefix); err != nil {
		return fmt.Errorf("failed to write length prefix: %w", err)
	}

	// Write encrypted data
	if _, err := c.stream.Write(dataToSend); err != nil {
		return fmt.Errorf("failed to write frame: %w", err)
	}

	return nil
}

// ReadFrame reads and decrypts a frame from the peer
func (c *QUICConnection) ReadFrame() ([]byte, error) {
	// Read length prefix (4 bytes)
	lengthPrefix := make([]byte, 4)
	if _, err := io.ReadFull(c.stream, lengthPrefix); err != nil {
		return nil, fmt.Errorf("failed to read length prefix: %w", err)
	}

	frameLen := binary.BigEndian.Uint32(lengthPrefix)
	if frameLen == 0 || frameLen > 65535 {
		return nil, fmt.Errorf("invalid frame length: %d", frameLen)
	}

	// Read encrypted frame data
	encryptedData := make([]byte, frameLen)
	if _, err := io.ReadFull(c.stream, encryptedData); err != nil {
		return nil, fmt.Errorf("failed to read frame data: %w", err)
	}

	// Decrypt if cipher is available
	if c.cipher != nil {
		decrypted, err := c.cipher.Decrypt(encryptedData)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
		return decrypted, nil
	}

	return encryptedData, nil
}

// Close gracefully closes the QUIC connection
func (c *QUICConnection) Close() error {
	c.closeMux.Lock()
	defer c.closeMux.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.closeChan)

	// Close stream
	if c.stream != nil {
		c.stream.Close()
	}

	// Close QUIC connection with error code 0 (graceful shutdown)
	if c.conn != nil {
		c.conn.CloseWithError(0, "connection closed")
	}

	log.Printf("[QUIC-TRANSPORT] Closed connection to peer %s", c.peerID)
	return nil
}

// RemoveConnection removes a connection from the transport's map
func (t *QUICTransport) RemoveConnection(peerID string) {
	t.connMux.Lock()
	delete(t.connections, peerID)
	t.connMux.Unlock()
	log.Printf("[QUIC-TRANSPORT] Removed peer %s from connection map", peerID)
}

// GetConnection retrieves a connection by peer ID
func (t *QUICTransport) GetConnection(peerID string) (*QUICConnection, bool) {
	t.connMux.RLock()
	defer t.connMux.RUnlock()
	conn, exists := t.connections[peerID]
	return conn, exists
}

// Close shuts down the QUIC transport and all connections
func (t *QUICTransport) Close() error {
	// Close all connections
	t.connMux.Lock()
	for peerID, conn := range t.connections {
		conn.Close()
		delete(t.connections, peerID)
	}
	t.connMux.Unlock()

	// Close listener
	if t.listener != nil {
		return t.listener.Close()
	}

	return nil
}
