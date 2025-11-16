package daemonmgr

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/shadowmesh/shadowmesh/pkg/crypto/frameencryption"
	"github.com/shadowmesh/shadowmesh/pkg/crypto/symmetric"
	"github.com/shadowmesh/shadowmesh/pkg/layer2"
	"github.com/shadowmesh/shadowmesh/pkg/nat"
)

// DaemonConfig contains complete daemon configuration
type DaemonConfig struct {
	Daemon struct {
		ListenAddress string `yaml:"listen_address"` // HTTP API address
		LogLevel      string `yaml:"log_level"`
	} `yaml:"daemon"`

	Network struct {
		TAPDevice string `yaml:"tap_device"`
		LocalIP   string `yaml:"local_ip"` // IP with CIDR (e.g., "10.0.0.1/24")
	} `yaml:"network"`

	Encryption struct {
		Key string `yaml:"key"` // Hex-encoded 32-byte key
	} `yaml:"encryption"`

	Peer struct {
		Address string `yaml:"address"` // Peer address (set dynamically via CLI)
	} `yaml:"peer"`

	NAT struct {
		Enabled    bool   `yaml:"enabled"`
		STUNServer string `yaml:"stun_server"`
	} `yaml:"nat"`
}

// ConnectionState represents daemon connection state
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateError
)

func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// DaemonManager manages the complete ShadowMesh daemon lifecycle
type DaemonManager struct {
	config *DaemonConfig

	// Epic 2 Components
	tapDevice          *layer2.TAPDevice
	encryptionPipeline *frameencryption.EncryptionPipeline
	p2pConnection      *P2PConnection
	natDetector        *nat.NATDetector
	holePuncher        *nat.HolePuncher
	daemonAPI          *DaemonAPI

	// State management
	state     ConnectionState
	stateMu   sync.RWMutex
	lastError error

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Frame routing
	frameRouterStop chan struct{}
}

// NewDaemonManager creates a new daemon manager
func NewDaemonManager(config *DaemonConfig) (*DaemonManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	dm := &DaemonManager{
		config:          config,
		state:           StateDisconnected,
		ctx:             ctx,
		cancel:          cancel,
		frameRouterStop: make(chan struct{}),
	}

	return dm, nil
}

// Start initializes and starts the daemon
func (dm *DaemonManager) Start(ctx context.Context) error {
	log.Printf("Initializing ShadowMesh daemon components...")

	// Phase 1: Initialize TAP device
	if err := dm.initTAPDevice(); err != nil {
		return fmt.Errorf("TAP device initialization failed: %w", err)
	}

	// Phase 2: Initialize encryption pipeline
	if err := dm.initEncryptionPipeline(); err != nil {
		return fmt.Errorf("encryption pipeline initialization failed: %w", err)
	}

	// Phase 3: Initialize NAT components (optional)
	if dm.config.NAT.Enabled {
		if err := dm.initNATComponents(); err != nil {
			log.Printf("⚠️  NAT initialization failed (continuing anyway): %v", err)
		}
	}

	// Phase 4: Initialize HTTP API
	if err := dm.initAPI(); err != nil {
		return fmt.Errorf("API initialization failed: %w", err)
	}

	log.Printf("✅ All daemon components initialized successfully")

	return nil
}

// Stop performs graceful shutdown
func (dm *DaemonManager) Stop() error {
	log.Printf("Stopping daemon components...")

	// Stop frame router
	if dm.frameRouterStop != nil {
		close(dm.frameRouterStop)
	}

	// Disconnect if connected
	if dm.state == StateConnected {
		if err := dm.Disconnect(); err != nil {
			log.Printf("⚠️  Error disconnecting: %v", err)
		}
	}

	// Stop encryption pipeline
	if dm.encryptionPipeline != nil {
		dm.encryptionPipeline.Stop()
		log.Printf("✅ Encryption pipeline stopped")
	}

	// Close TAP device
	if dm.tapDevice != nil {
		if err := dm.tapDevice.Stop(); err != nil {
			log.Printf("⚠️  Error stopping TAP device: %v", err)
		} else {
			log.Printf("✅ TAP device stopped")
		}
	}

	// Cancel context and wait for goroutines
	dm.cancel()
	dm.wg.Wait()

	return nil
}

// Connect establishes P2P connection to peer
func (dm *DaemonManager) Connect(peerAddr string) error {
	dm.stateMu.Lock()
	if dm.state == StateConnected {
		dm.stateMu.Unlock()
		return fmt.Errorf("already connected")
	}
	dm.state = StateConnecting
	dm.stateMu.Unlock()

	log.Printf("Connecting to peer: %s", peerAddr)

	// Initialize P2P connection if not already done
	if dm.p2pConnection == nil {
		dm.p2pConnection = NewP2PConnection()
	}

	// Establish WebSocket connection
	if err := dm.p2pConnection.Connect(peerAddr); err != nil {
		dm.setState(StateError, err)
		return fmt.Errorf("connection failed: %w", err)
	}

	// Update config with peer address
	dm.config.Peer.Address = peerAddr

	// Start frame router
	dm.startFrameRouter()

	dm.setState(StateConnected, nil)
	log.Printf("✅ Connected to peer successfully")

	return nil
}

// Disconnect closes the P2P connection
func (dm *DaemonManager) Disconnect() error {
	dm.stateMu.Lock()
	if dm.state != StateConnected {
		dm.stateMu.Unlock()
		return fmt.Errorf("not connected")
	}
	dm.stateMu.Unlock()

	log.Printf("Disconnecting from peer...")

	// Stop frame router
	if dm.frameRouterStop != nil {
		close(dm.frameRouterStop)
		dm.frameRouterStop = make(chan struct{}) // Reset for next connection
	}

	// Close P2P connection
	if dm.p2pConnection != nil {
		if err := dm.p2pConnection.Close(); err != nil {
			log.Printf("⚠️  Error during disconnect: %v", err)
		}
	}

	dm.setState(StateDisconnected, nil)
	log.Printf("✅ Disconnected successfully")

	return nil
}

// GetStatus returns current daemon status
func (dm *DaemonManager) GetStatus() map[string]interface{} {
	dm.stateMu.RLock()
	state := dm.state
	lastError := dm.lastError
	dm.stateMu.RUnlock()

	status := map[string]interface{}{
		"state":      state.String(),
		"tap_device": dm.config.Network.TAPDevice,
		"local_ip":   dm.config.Network.LocalIP,
	}

	if lastError != nil {
		status["last_error"] = lastError.Error()
	}

	if dm.p2pConnection != nil && state == StateConnected {
		status["peer_address"] = dm.config.Peer.Address
		status["connected"] = true
	} else {
		status["connected"] = false
	}

	return status
}

// initTAPDevice initializes the TAP device
func (dm *DaemonManager) initTAPDevice() error {
	log.Printf("Creating TAP device: %s", dm.config.Network.TAPDevice)

	// Create TAP device with config
	tapConfig := layer2.TAPConfig{
		Name: dm.config.Network.TAPDevice,
		MTU:  1500,
	}

	tap, err := layer2.NewTAPDevice(tapConfig)
	if err != nil {
		return fmt.Errorf("failed to create TAP device: %w", err)
	}
	dm.tapDevice = tap

	// Parse IP address and netmask from CIDR
	ip, ipNet, err := net.ParseCIDR(dm.config.Network.LocalIP)
	if err != nil {
		return fmt.Errorf("invalid local IP address: %w", err)
	}

	// Calculate netmask as string (e.g., "24" for /24)
	ones, _ := ipNet.Mask.Size()
	netmask := fmt.Sprintf("%d", ones)

	// Configure IP address using ip command
	if err := tap.ConfigureInterface(ip.String(), netmask); err != nil {
		return fmt.Errorf("failed to configure interface: %w", err)
	}

	log.Printf("✅ TAP device %s created with IP %s", tap.Name(), dm.config.Network.LocalIP)

	// Start reading/writing frames
	tap.Start()

	log.Printf("✅ TAP device reading/writing started")

	return nil
}

// initEncryptionPipeline initializes the frame encryption pipeline
func (dm *DaemonManager) initEncryptionPipeline() error {
	log.Printf("Initializing encryption pipeline...")

	// Decode hex key
	keyBytes, err := hex.DecodeString(dm.config.Encryption.Key)
	if err != nil {
		return fmt.Errorf("invalid encryption key (must be hex): %w", err)
	}

	if len(keyBytes) != symmetric.KeySize {
		return fmt.Errorf("invalid key length: got %d bytes, expected %d", len(keyBytes), symmetric.KeySize)
	}

	var encKey [symmetric.KeySize]byte
	copy(encKey[:], keyBytes)

	// Create pipeline config
	pipelineConfig := &frameencryption.PipelineConfig{
		Key:        encKey,
		BufferSize: 100,
	}

	// Create pipeline
	pipeline, err := frameencryption.NewEncryptionPipeline(pipelineConfig)
	if err != nil {
		return fmt.Errorf("failed to create encryption pipeline: %w", err)
	}
	dm.encryptionPipeline = pipeline

	// Start pipeline goroutines
	pipeline.Start()

	log.Printf("✅ Encryption pipeline started (ChaCha20-Poly1305)")

	return nil
}

// initNATComponents initializes NAT detection and hole punching
func (dm *DaemonManager) initNATComponents() error {
	log.Printf("Initializing NAT components...")

	// Create NAT detector (no parameters - uses default STUN client)
	detector := nat.NewNATDetector()
	dm.natDetector = detector

	// Detect NAT type
	ctx, cancel := context.WithTimeout(dm.ctx, 5*time.Second)
	defer cancel()

	result, err := detector.DetectNATType(ctx)
	if err != nil {
		return fmt.Errorf("NAT detection failed: %w", err)
	}

	log.Printf("✅ NAT Type: %s (detected in %v)", result.NATType, result.DetectionTime)
	log.Printf("   Public IP: %s", result.PublicIP)

	// Check if P2P is feasible
	feasible := detector.IsP2PFeasible()
	log.Printf("   P2P Feasible: %v", feasible)

	// Create hole puncher if P2P is feasible
	if feasible {
		// Use port 0 for automatic port assignment
		holePuncher, err := nat.NewHolePuncher(0, detector)
		if err != nil {
			return fmt.Errorf("failed to create hole puncher: %w", err)
		}
		dm.holePuncher = holePuncher
		log.Printf("✅ UDP hole puncher initialized")
	}

	return nil
}


// initAPI initializes the HTTP API server
func (dm *DaemonManager) initAPI() error {
	log.Printf("Starting HTTP API on %s", dm.config.Daemon.ListenAddress)

	api, err := NewDaemonAPI(dm.config.Daemon.ListenAddress, dm)
	if err != nil {
		return fmt.Errorf("failed to create API: %w", err)
	}
	dm.daemonAPI = api

	// Start API server in background
	dm.wg.Add(1)
	go func() {
		defer dm.wg.Done()
		if err := api.Start(); err != nil {
			log.Printf("⚠️  API server error: %v", err)
		}
	}()

	log.Printf("✅ HTTP API started successfully")

	return nil
}

// startFrameRouter starts the frame routing goroutines
func (dm *DaemonManager) startFrameRouter() {
	log.Printf("Starting frame router...")

	// Outbound: TAP → Encrypt → WebSocket
	dm.wg.Add(1)
	go func() {
		defer dm.wg.Done()
		dm.frameRouterOutbound()
	}()

	// Inbound: WebSocket → Decrypt → TAP
	dm.wg.Add(1)
	go func() {
		defer dm.wg.Done()
		dm.frameRouterInbound()
	}()

	log.Printf("✅ Frame router started")
}

// frameRouterOutbound routes frames from TAP → Encrypt → WebSocket
func (dm *DaemonManager) frameRouterOutbound() {
	for {
		select {
		case <-dm.frameRouterStop:
			log.Printf("Frame router outbound stopped")
			return
		case <-dm.ctx.Done():
			return
		case frame := <-dm.tapDevice.ReadChannel():
			// Send to encryption pipeline (non-blocking)
			if !dm.encryptionPipeline.SendFrame(frame) {
				log.Printf("⚠️  Encryption pipeline full, dropping frame")
				continue
			}

			// Receive encrypted frame (blocking with context)
			ctx, cancel := context.WithTimeout(dm.ctx, 100*time.Millisecond)
			encryptedFrame, err := dm.encryptionPipeline.ReceiveEncryptedFrame(ctx)
			cancel()

			if err != nil {
				if err != context.DeadlineExceeded {
					log.Printf("⚠️  Failed to receive encrypted frame: %v", err)
				}
				continue
			}

			// Serialize encrypted frame to bytes for WebSocket transmission
			// Format: [12-byte nonce][ciphertext with tag]
			frameBytes := make([]byte, len(encryptedFrame.Frame.Nonce)+len(encryptedFrame.Frame.Ciphertext))
			copy(frameBytes[:len(encryptedFrame.Frame.Nonce)], encryptedFrame.Frame.Nonce[:])
			copy(frameBytes[len(encryptedFrame.Frame.Nonce):], encryptedFrame.Frame.Ciphertext)

			// Send over WebSocket
			if err := dm.p2pConnection.SendFrame(frameBytes); err != nil {
				log.Printf("⚠️  Failed to send frame over WebSocket: %v", err)
			}
		}
	}
}

// frameRouterInbound routes frames from WebSocket → Decrypt → TAP
func (dm *DaemonManager) frameRouterInbound() {
	for {
		select {
		case <-dm.frameRouterStop:
			log.Printf("Frame router inbound stopped")
			return
		case <-dm.ctx.Done():
			return
		case encryptedBytes := <-dm.p2pConnection.RecvChannel():
			// Parse encrypted frame from bytes
			// Format: [12-byte nonce][ciphertext with tag]
			if len(encryptedBytes) < symmetric.NonceSize {
				log.Printf("⚠️  Invalid encrypted frame: too short")
				continue
			}

			var nonce [symmetric.NonceSize]byte
			copy(nonce[:], encryptedBytes[:symmetric.NonceSize])
			ciphertext := encryptedBytes[symmetric.NonceSize:]

			encryptedFrame := &frameencryption.EncryptedEthernetFrame{
				Frame: &symmetric.EncryptedFrame{
					Nonce:      nonce,
					Ciphertext: ciphertext,
				},
				Timestamp: time.Now(),
			}

			// Send to decryption pipeline (non-blocking)
			if !dm.encryptionPipeline.SendEncryptedFrame(encryptedFrame) {
				log.Printf("⚠️  Decryption pipeline full, dropping frame")
				continue
			}

			// Receive decrypted frame (blocking with context)
			ctx, cancel := context.WithTimeout(dm.ctx, 100*time.Millisecond)
			decryptedBytes, err := dm.encryptionPipeline.ReceiveDecryptedFrame(ctx)
			cancel()

			if err != nil {
				if err != context.DeadlineExceeded {
					log.Printf("⚠️  Failed to receive decrypted frame: %v", err)
				}
				continue
			}

			// Write to TAP device
			select {
			case dm.tapDevice.WriteChannel() <- decryptedBytes:
			case <-dm.frameRouterStop:
				return
			default:
				log.Printf("⚠️  TAP write channel full, dropping frame")
			}
		}
	}
}

// setState updates the connection state
func (dm *DaemonManager) setState(state ConnectionState, err error) {
	dm.stateMu.Lock()
	dm.state = state
	dm.lastError = err
	dm.stateMu.Unlock()
}
