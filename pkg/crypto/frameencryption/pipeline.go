// Package frameencryption provides a high-performance pipeline for encrypting and decrypting Ethernet frames.
// This package implements the frame encryption pipeline for ShadowMesh (Story 2.6).
package frameencryption

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/shadowmesh/shadowmesh/pkg/crypto/symmetric"
	"github.com/shadowmesh/shadowmesh/pkg/layer2"
)

// Pipeline stages: TAP capture → encrypt → WSS transmit → WSS receive → decrypt → TAP inject

// EncryptionPipeline handles frame encryption/decryption with goroutine-based pipeline architecture
type EncryptionPipeline struct {
	// Encryption key (256-bit ChaCha20-Poly1305 key)
	key [symmetric.KeySize]byte

	// Nonce generator for replay protection
	nonceGen *symmetric.NonceGenerator

	// Pipeline channels
	inboundFrames  chan *layer2.EthernetFrame // TAP → Encrypt
	outboundFrames chan []byte                // Decrypt → TAP

	encryptedFrames chan *EncryptedEthernetFrame // Encrypt → WSS
	receivedFrames  chan *EncryptedEthernetFrame // WSS → Decrypt

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	encryptedCount uint64
	decryptedCount uint64
	droppedCount   uint64 // Invalid frames dropped
	startTime      time.Time

	// Configuration
	bufferSize int // Channel buffer size
}

// EncryptedEthernetFrame wraps a symmetric.EncryptedFrame with metadata
type EncryptedEthernetFrame struct {
	Frame     *symmetric.EncryptedFrame
	Timestamp time.Time
}

// PipelineConfig contains configuration for the encryption pipeline
type PipelineConfig struct {
	Key        [symmetric.KeySize]byte // Encryption key
	BufferSize int                     // Channel buffer size (default: 100)
}

// NewEncryptionPipeline creates a new frame encryption pipeline
func NewEncryptionPipeline(config *PipelineConfig) (*EncryptionPipeline, error) {
	// Create nonce generator for replay protection
	nonceGen, err := symmetric.NewNonceGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to create nonce generator: %w", err)
	}

	bufferSize := config.BufferSize
	if bufferSize == 0 {
		bufferSize = 100 // Default buffer size
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &EncryptionPipeline{
		key:      config.Key,
		nonceGen: nonceGen,

		// Buffered channels for pipeline stages
		inboundFrames:   make(chan *layer2.EthernetFrame, bufferSize),
		outboundFrames:  make(chan []byte, bufferSize),
		encryptedFrames: make(chan *EncryptedEthernetFrame, bufferSize),
		receivedFrames:  make(chan *EncryptedEthernetFrame, bufferSize),

		ctx:        ctx,
		cancel:     cancel,
		bufferSize: bufferSize,
		startTime:  time.Now(),
	}, nil
}

// Start starts all pipeline goroutines
func (p *EncryptionPipeline) Start() {
	// Encryption goroutine (TAP → Encrypt → WSS)
	p.wg.Add(1)
	go p.encryptionLoop()

	// Decryption goroutine (WSS → Decrypt → TAP)
	p.wg.Add(1)
	go p.decryptionLoop()
}

// Stop stops the pipeline gracefully
func (p *EncryptionPipeline) Stop() {
	p.cancel()
	p.wg.Wait()

	// Close channels
	close(p.inboundFrames)
	close(p.outboundFrames)
	close(p.encryptedFrames)
	close(p.receivedFrames)
}

// encryptionLoop handles frame encryption (runs in separate goroutine)
// Pipeline: TAP readChan → Serialize() → Encrypt() → encryptedFrames channel
func (p *EncryptionPipeline) encryptionLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case frame, ok := <-p.inboundFrames:
			if !ok {
				return
			}

			// Serialize frame to bytes
			plaintext := frame.Serialize()

			// Generate unique nonce for this frame
			nonce, err := p.nonceGen.GenerateNonce()
			if err != nil {
				log.Printf("FrameEncryption: Failed to generate nonce: %v", err)
				continue
			}

			// Encrypt frame with ChaCha20-Poly1305 AEAD
			encrypted, err := symmetric.Encrypt(plaintext, p.key, nonce)
			if err != nil {
				log.Printf("FrameEncryption: Encryption failed: %v", err)
				continue
			}

			// Send to encrypted frames channel (for WSS transmission)
			select {
			case p.encryptedFrames <- &EncryptedEthernetFrame{
				Frame:     encrypted,
				Timestamp: time.Now(),
			}:
				p.encryptedCount++
			case <-p.ctx.Done():
				return
			default:
				// Channel full - drop frame (backpressure)
				log.Printf("FrameEncryption: Encrypted channel full, dropping frame")
			}
		}
	}
}

// decryptionLoop handles frame decryption (runs in separate goroutine)
// Pipeline: WSS receivedFrames channel → Decrypt() → Validate tag → outboundFrames channel
func (p *EncryptionPipeline) decryptionLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case encFrame, ok := <-p.receivedFrames:
			if !ok {
				return
			}

			// Decrypt and validate authentication tag
			plaintext, err := symmetric.Decrypt(encFrame.Frame, p.key)
			if err != nil {
				// Invalid authentication tag - frame tampered or wrong key
				log.Printf("FrameEncryption: Decryption failed (invalid tag): %v", err)
				p.droppedCount++
				continue // Drop invalid frame
			}

			// Send decrypted frame to outbound channel (for TAP injection)
			select {
			case p.outboundFrames <- plaintext:
				p.decryptedCount++
			case <-p.ctx.Done():
				return
			default:
				// Channel full - drop frame (backpressure)
				log.Printf("FrameEncryption: Outbound channel full, dropping frame")
				p.droppedCount++
			}
		}
	}
}

// SendFrame sends a frame for encryption (called by TAP device)
// Non-blocking: returns immediately if channel is full
func (p *EncryptionPipeline) SendFrame(frame *layer2.EthernetFrame) bool {
	select {
	case p.inboundFrames <- frame:
		return true
	default:
		// Channel full - cannot accept frame
		return false
	}
}

// ReceiveEncryptedFrame receives an encrypted frame for transmission (called by WebSocket sender)
// Blocking: waits until frame is available or context is canceled
func (p *EncryptionPipeline) ReceiveEncryptedFrame(ctx context.Context) (*EncryptedEthernetFrame, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.ctx.Done():
		return nil, fmt.Errorf("pipeline stopped")
	case frame, ok := <-p.encryptedFrames:
		if !ok {
			return nil, fmt.Errorf("encrypted frames channel closed")
		}
		return frame, nil
	}
}

// SendEncryptedFrame sends an encrypted frame for decryption (called by WebSocket receiver)
// Non-blocking: returns immediately if channel is full
func (p *EncryptionPipeline) SendEncryptedFrame(frame *EncryptedEthernetFrame) bool {
	select {
	case p.receivedFrames <- frame:
		return true
	default:
		// Channel full - cannot accept frame
		return false
	}
}

// ReceiveDecryptedFrame receives a decrypted frame for TAP injection (called by TAP device)
// Blocking: waits until frame is available or context is canceled
func (p *EncryptionPipeline) ReceiveDecryptedFrame(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.ctx.Done():
		return nil, fmt.Errorf("pipeline stopped")
	case frame, ok := <-p.outboundFrames:
		if !ok {
			return nil, fmt.Errorf("outbound frames channel closed")
		}
		return frame, nil
	}
}

// GetMetrics returns pipeline performance metrics
func (p *EncryptionPipeline) GetMetrics() *PipelineMetrics {
	return &PipelineMetrics{
		EncryptedCount: p.encryptedCount,
		DecryptedCount: p.decryptedCount,
		DroppedCount:   p.droppedCount,
		Uptime:         time.Since(p.startTime),
		BufferSize:     p.bufferSize,
	}
}

// PipelineMetrics contains pipeline performance metrics
type PipelineMetrics struct {
	EncryptedCount uint64        // Total frames encrypted
	DecryptedCount uint64        // Total frames decrypted
	DroppedCount   uint64        // Total frames dropped (invalid tag or buffer full)
	Uptime         time.Duration // Pipeline uptime
	BufferSize     int           // Channel buffer size
}

// GetThroughput calculates throughput in frames per second
func (m *PipelineMetrics) GetThroughput() float64 {
	if m.Uptime.Seconds() == 0 {
		return 0
	}
	totalFrames := m.EncryptedCount + m.DecryptedCount
	return float64(totalFrames) / m.Uptime.Seconds()
}
