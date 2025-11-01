package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shadowmesh/shadowmesh/shared/crypto"
)

const version = "0.1.0-alpha"

var (
	configPath = flag.String("config", GetConfigPath(), "Path to configuration file")
	genKeys    = flag.Bool("gen-keys", false, "Generate new signing keys and exit")
	showConfig = flag.Bool("show-config", false, "Show configuration and exit")
)

func main() {
	flag.Parse()

	fmt.Printf("ShadowMesh Client Daemon v%s\n", version)
	fmt.Println("Post-Quantum Decentralized Private Network (DPN)")
	fmt.Println("=================================================")
	fmt.Println()

	// Load or create configuration
	config, err := LoadOrCreateConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Handle --show-config flag
	if *showConfig {
		fmt.Printf("Configuration loaded from: %s\n\n", *configPath)
		fmt.Printf("Relay URL: %s\n", config.Relay.URL)
		fmt.Printf("TAP Device: %s (MTU: %d)\n", config.TAP.Name, config.TAP.MTU)
		fmt.Printf("TAP IP: %s/%s\n", config.TAP.IPAddr, config.TAP.Netmask)
		fmt.Printf("Key Rotation: %v (interval: %v)\n", config.Crypto.EnableKeyRotation, config.Crypto.KeyRotationInterval)
		fmt.Printf("Keys Directory: %s\n", config.Identity.KeysDir)
		fmt.Printf("Log Level: %s\n", config.Logging.Level)
		return
	}

	// Ensure directories exist
	if err := config.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to create directories: %v", err)
	}

	// Handle --gen-keys flag
	if *genKeys {
		if err := generateKeys(config); err != nil {
			log.Fatalf("Failed to generate keys: %v", err)
		}
		return
	}

	// Load or generate client identity
	clientID, sigKeys, err := loadOrGenerateIdentity(config)
	if err != nil {
		log.Fatalf("Failed to load identity: %v", err)
	}

	log.Printf("Client ID: %s", hex.EncodeToString(clientID[:]))

	// Create TAP device (requires root privileges)
	log.Printf("Creating TAP device: %s", config.TAP.Name)
	tapConfig := TAPConfig{
		Name: config.TAP.Name,
		MTU:  config.TAP.MTU,
	}
	tap, err := NewTAPDevice(tapConfig)
	if err != nil {
		log.Fatalf("Failed to create TAP device: %v (are you running as root?)", err)
	}
	defer tap.Stop()

	log.Printf("TAP device created: %s (MTU: %d)", tap.Name(), tap.MTU())

	// Start TAP device
	tap.Start()

	// Create connection manager
	log.Printf("Connecting to relay: %s", config.Relay.URL)
	connMgr := NewConnectionManager(config.Relay.URL, config.Relay.TLSSkipVerify)
	connMgr.SetCallbacks(
		func() {
			log.Println("Connected to relay")
		},
		func(err error) {
			log.Printf("Disconnected from relay: %v", err)
		},
		nil, // Message callback handled by tunnel manager
	)

	// Start connection
	if err := connMgr.Start(); err != nil {
		log.Fatalf("Failed to start connection: %v", err)
	}
	defer connMgr.Stop()

	// Perform handshake
	log.Println("Performing post-quantum handshake...")
	handshakeOrch := NewHandshakeOrchestrator(connMgr, clientID, sigKeys)
	sessionKeys, err := handshakeOrch.PerformHandshake()
	if err != nil {
		log.Fatalf("Handshake failed: %v", err)
	}

	log.Printf("Handshake complete. Session ID: %s", hex.EncodeToString(sessionKeys.SessionID[:]))
	log.Printf("Session parameters: MTU=%d, Heartbeat=%v, KeyRotation=%v",
		sessionKeys.MTU, sessionKeys.HeartbeatInterval, sessionKeys.KeyRotationInterval)

	// Create tunnel manager
	log.Println("Starting encrypted tunnel...")
	tunnelMgr, err := NewTunnelManager(tap, connMgr, sessionKeys)
	if err != nil {
		log.Fatalf("Failed to create tunnel: %v", err)
	}
	defer tunnelMgr.Stop()

	// Start tunnel
	tunnelMgr.Start()
	log.Println("Tunnel established. Network traffic is now encrypted.")

	// Start key rotation goroutine if enabled
	if config.Crypto.EnableKeyRotation {
		go keyRotationLoop(handshakeOrch, tunnelMgr, config.Crypto.KeyRotationInterval)
	}

	// Start statistics reporter
	go statsReporter(tunnelMgr)

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Daemon running. Press Ctrl+C to stop.")

	<-sigChan
	log.Println("Received shutdown signal. Cleaning up...")

	// Graceful shutdown
	log.Println("Stopping tunnel...")
	tunnelMgr.Stop()

	log.Println("Closing connection...")
	connMgr.Stop()

	log.Println("Stopping TAP device...")
	tap.Stop()

	log.Println("Daemon stopped.")
}

// loadOrGenerateIdentity loads existing keys or generates new ones
func loadOrGenerateIdentity(config *Config) ([32]byte, *crypto.HybridSigningKey, error) {
	var clientID [32]byte

	// Try to load existing keys
	if _, err := os.Stat(config.Identity.PrivateKeyFile); err == nil {
		log.Println("Loading existing signing keys...")
		sigKeys, err := loadSigningKey(config.Identity.PrivateKeyFile)
		if err != nil {
			return clientID, nil, fmt.Errorf("failed to load signing key: %w", err)
		}

		// Load client ID
		if data, err := os.ReadFile(config.Identity.ClientIDFile); err == nil {
			decoded, err := hex.DecodeString(string(data))
			if err == nil && len(decoded) == 32 {
				copy(clientID[:], decoded)
				return clientID, sigKeys, nil
			}
		}

		// If client ID doesn't exist, derive it from public key
		pubKey := sigKeys.PublicKey()
		clientID = crypto.PublicKeyHash(pubKey)
		saveClientID(config.Identity.ClientIDFile, clientID)

		return clientID, sigKeys, nil
	}

	// Generate new keys
	log.Println("Generating new signing keys...")
	sigKeys, err := crypto.GenerateSigningKey()
	if err != nil {
		return clientID, nil, fmt.Errorf("failed to generate signing key: %w", err)
	}

	// Derive client ID from public key hash
	pubKey := sigKeys.PublicKey()
	clientID = crypto.PublicKeyHash(pubKey)

	// Save keys
	if err := saveSigningKey(config.Identity.PrivateKeyFile, sigKeys); err != nil {
		return clientID, nil, fmt.Errorf("failed to save signing key: %w", err)
	}

	if err := saveClientID(config.Identity.ClientIDFile, clientID); err != nil {
		return clientID, nil, fmt.Errorf("failed to save client ID: %w", err)
	}

	log.Printf("Keys generated and saved to: %s", config.Identity.KeysDir)

	return clientID, sigKeys, nil
}

// generateKeys generates and saves new signing keys
func generateKeys(config *Config) error {
	log.Println("Generating new signing keys...")

	sigKeys, err := crypto.GenerateSigningKey()
	if err != nil {
		return fmt.Errorf("failed to generate signing key: %w", err)
	}

	// Derive client ID
	pubKey := sigKeys.PublicKey()
	clientID := crypto.PublicKeyHash(pubKey)

	// Save keys
	if err := saveSigningKey(config.Identity.PrivateKeyFile, sigKeys); err != nil {
		return fmt.Errorf("failed to save signing key: %w", err)
	}

	if err := saveClientID(config.Identity.ClientIDFile, clientID); err != nil {
		return fmt.Errorf("failed to save client ID: %w", err)
	}

	log.Printf("Keys generated successfully!")
	log.Printf("Client ID: %s", hex.EncodeToString(clientID[:]))
	log.Printf("Keys saved to: %s", config.Identity.KeysDir)

	return nil
}

// saveSigningKey saves a signing key to file (placeholder)
func saveSigningKey(path string, key *crypto.HybridSigningKey) error {
	// TODO: Implement proper key serialization
	// For now, create a placeholder file
	return os.WriteFile(path, []byte("signing_key_placeholder"), 0600)
}

// loadSigningKey loads a signing key from file (placeholder)
func loadSigningKey(path string) (*crypto.HybridSigningKey, error) {
	// TODO: Implement proper key deserialization
	// For now, generate a new key as placeholder
	return crypto.GenerateSigningKey()
}

// saveClientID saves client ID to file
func saveClientID(path string, clientID [32]byte) error {
	hexID := hex.EncodeToString(clientID[:])
	return os.WriteFile(path, []byte(hexID), 0600)
}

// keyRotationLoop periodically rotates session keys
func keyRotationLoop(handshake *HandshakeOrchestrator, tunnel *TunnelManager, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Performing key rotation...")

		newSessionKeys, err := handshake.PerformKeyRotation()
		if err != nil {
			log.Printf("Key rotation failed: %v", err)
			continue
		}

		if err := tunnel.RotateKeys(newSessionKeys); err != nil {
			log.Printf("Failed to apply rotated keys: %v", err)
			continue
		}

		log.Printf("Key rotation complete. New session ID: %s",
			hex.EncodeToString(newSessionKeys.SessionID[:]))
	}
}

// statsReporter periodically reports tunnel statistics
func statsReporter(tunnel *TunnelManager) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := tunnel.GetStats()
		log.Printf("Stats: Sent=%d frames (%d bytes), Recv=%d frames (%d bytes), Errors: Encrypt=%d Decrypt=%d Dropped=%d",
			stats.FramesSent.Load(), stats.BytesSent.Load(),
			stats.FramesReceived.Load(), stats.BytesReceived.Load(),
			stats.EncryptErrors.Load(), stats.DecryptErrors.Load(),
			stats.DroppedFrames.Load())
	}
}

// generateRandomBytes generates random bytes
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}
