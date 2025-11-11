package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/shadowmesh/shadowmesh/shared/crypto"
)

var (
	version = "0.1.0-alpha"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", getDefaultConfigPath(), "Path to configuration file")
	genKeys := flag.Bool("gen-keys", false, "Generate relay keys and exit")
	showConfig := flag.Bool("show-config", false, "Show configuration and exit")
	showVersion := flag.Bool("version", false, "Show version and exit")

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("ShadowMesh Relay Server v%s\n", version)
		os.Exit(0)
	}

	// Load or create configuration
	config, err := LoadOrCreateConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Handle gen-keys flag
	if *genKeys {
		if err := generateKeys(config); err != nil {
			log.Fatalf("Failed to generate keys: %v", err)
		}
		os.Exit(0)
	}

	// Handle show-config flag
	if *showConfig {
		if err := showConfigInfo(config); err != nil {
			log.Fatalf("Failed to show config: %v", err)
		}
		os.Exit(0)
	}

	// Print banner
	printBanner()

	// Load or generate relay identity
	relayID, sigKeys, err := loadOrGenerateIdentity(config)
	if err != nil {
		log.Fatalf("Failed to load identity: %v", err)
	}

	log.Printf("Relay ID: %x", relayID[:])

	// Create connection manager
	connMgr := NewConnectionManager(config)

	// Create router
	router := NewRouter(connMgr, RoutingModeBroadcast, config.Limits.MaxFrameSize)
	connMgr.SetRouter(router)

	// Create TLS certificate manager for Epic 2 Direct P2P
	tlsCertManager := NewTLSCertificateManager(sigKeys)

	// Extract relay's public IP from bind address for certificate SAN
	// Use bind address directly (e.g., "83.136.252.52:8080" -> "83.136.252.52")
	relayIP := config.Server.ListenAddr
	if colonIdx := len(relayIP) - 1; colonIdx > 0 {
		for i := len(relayIP) - 1; i >= 0; i-- {
			if relayIP[i] == ':' {
				relayIP = relayIP[:i]
				break
			}
		}
	}

	// Generate ephemeral TLS certificate at startup
	if err := tlsCertManager.GenerateEphemeralCertificate(relayIP); err != nil {
		log.Fatalf("Failed to generate TLS certificate: %v", err)
	}

	certFingerprint := tlsCertManager.GetCertificateFingerprint()
	log.Printf("Generated TLS certificate for Direct P2P (fingerprint: %x)", certFingerprint[:8])

	// Create handshake handler with TLS certificate manager
	handshakeHandler := NewRelayHandshakeHandler(relayID, sigKeys, tlsCertManager)
	connMgr.SetHandshakeHandler(handshakeHandler)

	// Start statistics reporter
	stopStats := startStatsReporter(connMgr, router)
	defer stopStats()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := connMgr.Start(); err != nil {
			errChan <- err
		}
	}()

	log.Println("ShadowMesh Relay Server started successfully")

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	case err := <-errChan:
		log.Printf("Server error: %v", err)
	}

	// Graceful shutdown
	log.Println("Shutting down...")
	if err := connMgr.Stop(); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("Goodbye!")
}

// loadOrGenerateIdentity loads or generates relay identity (ID + signing keys)
func loadOrGenerateIdentity(config *Config) ([32]byte, *crypto.HybridSigningKey, error) {
	var relayID [32]byte

	// Ensure keys directory exists
	keysDir, err := config.GetKeysDir()
	if err != nil {
		return relayID, nil, err
	}

	sigKeyPath := config.GetSigningKeyPath()
	relayIDPath := filepath.Join(keysDir, "relay_id.txt")

	// Check if signing key exists
	if _, err := os.Stat(sigKeyPath); os.IsNotExist(err) {
		if !config.Identity.AutoGenerate {
			return relayID, nil, fmt.Errorf("signing key not found and auto_generate is disabled")
		}

		log.Println("Generating new relay identity...")

		// Generate signing key
		sigKeys, err := crypto.GenerateSigningKey()
		if err != nil {
			return relayID, nil, fmt.Errorf("failed to generate signing key: %w", err)
		}

		// Generate relay ID (random 32 bytes)
		if _, err := rand.Read(relayID[:]); err != nil {
			return relayID, nil, fmt.Errorf("failed to generate relay ID: %w", err)
		}

		// Save signing key
		if err := saveSigningKey(sigKeyPath, sigKeys); err != nil {
			return relayID, nil, fmt.Errorf("failed to save signing key: %w", err)
		}

		// Save relay ID
		if err := os.WriteFile(relayIDPath, []byte(hex.EncodeToString(relayID[:])), 0600); err != nil {
			return relayID, nil, fmt.Errorf("failed to save relay ID: %w", err)
		}

		log.Printf("Generated new relay ID: %x", relayID[:8])
		log.Printf("Signing key saved to: %s", sigKeyPath)
		log.Printf("Relay ID saved to: %s", relayIDPath)

		return relayID, sigKeys, nil
	}

	// Load existing identity
	log.Println("Loading existing relay identity...")

	// Load signing key
	sigKeys, err := loadSigningKey(sigKeyPath)
	if err != nil {
		return relayID, nil, fmt.Errorf("failed to load signing key: %w", err)
	}

	// Load relay ID
	idData, err := os.ReadFile(relayIDPath)
	if err != nil {
		return relayID, nil, fmt.Errorf("failed to load relay ID: %w", err)
	}

	idBytes, err := hex.DecodeString(string(idData))
	if err != nil {
		return relayID, nil, fmt.Errorf("failed to decode relay ID: %w", err)
	}

	if len(idBytes) != 32 {
		return relayID, nil, fmt.Errorf("invalid relay ID length: %d", len(idBytes))
	}

	copy(relayID[:], idBytes)

	log.Printf("Loaded relay ID: %x", relayID[:8])

	return relayID, sigKeys, nil
}

// generateKeys generates and saves relay keys (for --gen-keys flag)
func generateKeys(config *Config) error {
	log.Println("Generating relay keys...")

	// Ensure keys directory exists
	keysDir, err := config.GetKeysDir()
	if err != nil {
		return err
	}

	sigKeyPath := config.GetSigningKeyPath()
	relayIDPath := filepath.Join(keysDir, "relay_id.txt")

	// Generate signing key
	sigKeys, err := crypto.GenerateSigningKey()
	if err != nil {
		return fmt.Errorf("failed to generate signing key: %w", err)
	}

	// Generate relay ID
	var relayID [32]byte
	if _, err := rand.Read(relayID[:]); err != nil {
		return fmt.Errorf("failed to generate relay ID: %w", err)
	}

	// Save signing key
	if err := saveSigningKey(sigKeyPath, sigKeys); err != nil {
		return fmt.Errorf("failed to save signing key: %w", err)
	}

	// Save relay ID
	if err := os.WriteFile(relayIDPath, []byte(hex.EncodeToString(relayID[:])), 0600); err != nil {
		return fmt.Errorf("failed to save relay ID: %w", err)
	}

	log.Println("Successfully generated relay keys:")
	log.Printf("  Relay ID: %x", relayID[:])
	log.Printf("  Signing key: %s", sigKeyPath)
	log.Printf("  Relay ID file: %s", relayIDPath)

	return nil
}

// showConfigInfo displays configuration information
func showConfigInfo(config *Config) error {
	log.Println("=== Configuration ===")
	log.Printf("Server Address: %s", config.Server.ListenAddr)
	log.Printf("TLS Enabled: %v", config.Server.TLS.Enabled)
	if config.Server.TLS.Enabled {
		log.Printf("  Cert File: %s", config.Server.TLS.CertFile)
		log.Printf("  Key File: %s", config.Server.TLS.KeyFile)
	}
	log.Printf("\nLimits:")
	log.Printf("  Max Clients: %d", config.Limits.MaxClients)
	log.Printf("  Handshake Timeout: %ds", config.Limits.HandshakeTimeout)
	log.Printf("  Heartbeat Interval: %ds", config.Limits.HeartbeatInterval)
	log.Printf("  Heartbeat Timeout: %ds", config.Limits.HeartbeatTimeout)
	log.Printf("  Max Frame Size: %d bytes", config.Limits.MaxFrameSize)
	log.Printf("\nIdentity:")
	log.Printf("  Keys Dir: %s", config.Identity.KeysDir)
	log.Printf("  Signing Key: %s", config.Identity.SigningKey)
	log.Printf("  Auto Generate: %v", config.Identity.AutoGenerate)
	log.Printf("\nLogging:")
	log.Printf("  Level: %s", config.Logging.Level)
	log.Printf("  Format: %s", config.Logging.Format)
	if config.Logging.OutputFile != "" {
		log.Printf("  Output: %s", config.Logging.OutputFile)
	} else {
		log.Printf("  Output: stdout")
	}

	return nil
}

// saveSigningKey saves a signing key to file (placeholder)
// TODO: Implement proper key serialization with ML-DSA and Ed25519 key formats
func saveSigningKey(path string, sigKeys *crypto.HybridSigningKey) error {
	// For now, create a placeholder file
	// In production, this would serialize the actual key material
	return os.WriteFile(path, []byte("signing_key_placeholder"), 0600)
}

// loadSigningKey loads a signing key from file (placeholder)
// TODO: Implement proper key deserialization
func loadSigningKey(path string) (*crypto.HybridSigningKey, error) {
	// For now, generate a new key as placeholder
	// In production, this would deserialize the actual key material
	return crypto.GenerateSigningKey()
}

// getDefaultConfigPath returns the default config file path
func getDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/etc/shadowmesh-relay/config.yaml"
	}
	return filepath.Join(homeDir, ".shadowmesh-relay", "config.yaml")
}

// printBanner prints the startup banner
func printBanner() {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║         ShadowMesh Relay Server v" + version + "             ║")
	fmt.Println("║     Post-Quantum Encrypted Private Network Relay          ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// startStatsReporter starts a goroutine that periodically reports statistics
func startStatsReporter(connMgr *ConnectionManager, router *Router) func() {
	stopChan := make(chan struct{})

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				routerStats := router.GetStats()
				log.Printf("Stats: active_clients=%d, total_connections=%d, frames_routed=%d, bytes_routed=%d",
					connMgr.activeConnections.Load(),
					connMgr.totalConnections.Load(),
					routerStats.FramesRouted,
					routerStats.BytesRouted)
			}
		}
	}()

	return func() {
		close(stopChan)
	}
}
