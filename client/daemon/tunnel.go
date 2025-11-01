package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/shadowmesh/shadowmesh/shared/crypto"
	"github.com/shadowmesh/shadowmesh/shared/protocol"
)

// TunnelManager manages the encrypted tunnel between TAP device and relay
type TunnelManager struct {
	tap        *TAPDevice
	conn       *ConnectionManager
	sessionKeys *SessionKeys

	// Frame encryption/decryption
	txEncryptor *crypto.FrameEncryptor
	rxEncryptor *crypto.FrameEncryptor

	// Frame counters for replay protection
	txCounter uint64
	rxCounter uint64

	// Context for lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Statistics
	stats TunnelStats
}

// TunnelStats tracks tunnel statistics
type TunnelStats struct {
	FramesSent     atomic.Uint64
	FramesReceived atomic.Uint64
	BytesSent      atomic.Uint64
	BytesReceived  atomic.Uint64
	EncryptErrors  atomic.Uint64
	DecryptErrors  atomic.Uint64
	DroppedFrames  atomic.Uint64
}

// NewTunnelManager creates a new tunnel manager
func NewTunnelManager(tap *TAPDevice, conn *ConnectionManager, sessionKeys *SessionKeys) (*TunnelManager, error) {
	// Create frame encryptors for TX and RX
	txEncryptor, err := crypto.NewFrameEncryptor(sessionKeys.TXKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create TX encryptor: %w", err)
	}

	rxEncryptor, err := crypto.NewFrameEncryptor(sessionKeys.RXKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create RX encryptor: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TunnelManager{
		tap:         tap,
		conn:        conn,
		sessionKeys: sessionKeys,
		txEncryptor: txEncryptor,
		rxEncryptor: rxEncryptor,
		txCounter:   0,
		rxCounter:   0,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// Start begins the tunnel data flow
func (tm *TunnelManager) Start() {
	tm.wg.Add(2)
	go tm.tapToRelayLoop()
	go tm.relayToTapLoop()
}

// Stop gracefully stops the tunnel
func (tm *TunnelManager) Stop() {
	tm.cancel()
	tm.wg.Wait()
}

// tapToRelayLoop reads frames from TAP, encrypts them, and sends to relay
func (tm *TunnelManager) tapToRelayLoop() {
	defer tm.wg.Done()

	for {
		select {
		case <-tm.ctx.Done():
			return

		case frame := <-tm.tap.ReadChannel():
			// Encrypt the frame
			encryptedFrame, err := tm.txEncryptor.Encrypt(frame)
			if err != nil {
				tm.stats.EncryptErrors.Add(1)
				continue
			}

			// Increment counter
			counter := atomic.AddUint64(&tm.txCounter, 1)

			// Create DATA_FRAME message
			dataMsg := protocol.NewDataFrameMessage(counter, encryptedFrame)

			// Send to relay
			if err := tm.conn.SendMessage(dataMsg); err != nil {
				tm.stats.DroppedFrames.Add(1)
				continue
			}

			// Update statistics
			tm.stats.FramesSent.Add(1)
			tm.stats.BytesSent.Add(uint64(len(frame)))
		}
	}
}

// relayToTapLoop receives frames from relay, decrypts them, and writes to TAP
func (tm *TunnelManager) relayToTapLoop() {
	defer tm.wg.Done()

	for {
		select {
		case <-tm.ctx.Done():
			return

		case msg := <-tm.conn.ReceiveChannel():
			// Only process DATA_FRAME messages
			if msg.Header.Type != protocol.MsgTypeDataFrame {
				// Ignore non-data messages (they're handled by connection manager)
				continue
			}

			dataPayload, ok := msg.Payload.(*protocol.DataFrame)
			if !ok {
				tm.stats.DroppedFrames.Add(1)
				continue
			}

			// Check for replay attacks
			if dataPayload.Counter <= tm.rxCounter {
				tm.stats.DroppedFrames.Add(1)
				continue
			}

			// Decrypt the frame
			decryptedFrame, err := tm.rxEncryptor.Decrypt(dataPayload.EncryptedData)
			if err != nil {
				tm.stats.DecryptErrors.Add(1)
				continue
			}

			// Update counter
			atomic.StoreUint64(&tm.rxCounter, dataPayload.Counter)

			// Write decrypted frame to TAP device
			select {
			case tm.tap.WriteChannel() <- decryptedFrame:
				tm.stats.FramesReceived.Add(1)
				tm.stats.BytesReceived.Add(uint64(len(decryptedFrame)))
			case <-tm.ctx.Done():
				return
			default:
				tm.stats.DroppedFrames.Add(1)
			}
		}
	}
}

// RotateKeys updates the encryption keys after key rotation
func (tm *TunnelManager) RotateKeys(newSessionKeys *SessionKeys) error {
	// Create new encryptors
	newTXEncryptor, err := crypto.NewFrameEncryptor(newSessionKeys.TXKey)
	if err != nil {
		return fmt.Errorf("failed to create new TX encryptor: %w", err)
	}

	newRXEncryptor, err := crypto.NewFrameEncryptor(newSessionKeys.RXKey)
	if err != nil {
		return fmt.Errorf("failed to create new RX encryptor: %w", err)
	}

	// Atomic swap of encryptors
	tm.txEncryptor = newTXEncryptor
	tm.rxEncryptor = newRXEncryptor
	tm.sessionKeys = newSessionKeys

	// Reset counters for new session
	atomic.StoreUint64(&tm.txCounter, 0)
	atomic.StoreUint64(&tm.rxCounter, 0)

	return nil
}

// GetStats returns a pointer to the tunnel statistics
func (tm *TunnelManager) GetStats() *TunnelStats {
	return &tm.stats
}

// GetTXCounter returns the current transmission counter
func (tm *TunnelManager) GetTXCounter() uint64 {
	return atomic.LoadUint64(&tm.txCounter)
}

// GetRXCounter returns the current reception counter
func (tm *TunnelManager) GetRXCounter() uint64 {
	return atomic.LoadUint64(&tm.rxCounter)
}
