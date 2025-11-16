// Package main implements the ShadowMesh production daemon
// This service wires together all Epic 2 components into a working P2P VPN daemon.
//
// Story 2.8: Direct P2P Integration Test
// Architecture: docs/DAEMON_ARCHITECTURE.md
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/shadowmesh/shadowmesh/pkg/daemonmgr"
	"gopkg.in/yaml.v3"
)

const (
	version = "0.1.0-epic2"
)

func main() {
	// Parse command-line arguments
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	configPath := os.Args[1]

	// Load configuration
	log.Printf("ShadowMesh Daemon v%s", version)
	log.Printf("Loading configuration from: %s", configPath)

	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	setupLogging(config.Daemon.LogLevel)

	// Validate configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("  Listen Address: %s", config.Daemon.ListenAddress)
	log.Printf("  TAP Device: %s", config.Network.TAPDevice)
	log.Printf("  Local IP: %s", config.Network.LocalIP)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create and start daemon
	dm, err := daemonmgr.NewDaemonManager(config)
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	// Start daemon
	log.Printf("Starting ShadowMesh daemon...")
	if err := dm.Start(ctx); err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	log.Printf("✅ ShadowMesh daemon started successfully")
	log.Printf("   API listening on: %s", config.Daemon.ListenAddress)
	log.Printf("   TAP device: %s (%s)", config.Network.TAPDevice, config.Network.LocalIP)
	log.Printf("")
	log.Printf("Use 'shadowmesh connect <peer-address>' to establish P2P connection")
	log.Printf("Press Ctrl+C to shutdown gracefully")

	// Wait for shutdown signal
	<-sigChan
	log.Printf("\n🛑 Shutdown signal received, stopping daemon...")

	// Graceful shutdown
	if err := dm.Stop(); err != nil {
		log.Printf("⚠️  Error during shutdown: %v", err)
	} else {
		log.Printf("✅ Daemon stopped successfully")
	}
}

// loadConfig loads daemon configuration from YAML file
func loadConfig(path string) (*daemonmgr.DaemonConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config daemonmgr.DaemonConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	// Set defaults
	if config.Daemon.ListenAddress == "" {
		config.Daemon.ListenAddress = "127.0.0.1:9090"
	}
	if config.Daemon.LogLevel == "" {
		config.Daemon.LogLevel = "info"
	}
	if config.Network.TAPDevice == "" {
		config.Network.TAPDevice = "tap0"
	}
	if config.NAT.STUNServer == "" {
		config.NAT.STUNServer = "stun.l.google.com:19302"
	}

	return &config, nil
}

// validateConfig validates daemon configuration
func validateConfig(config *daemonmgr.DaemonConfig) error {
	if config.Network.LocalIP == "" {
		return fmt.Errorf("network.local_ip is required")
	}

	if len(config.Encryption.Key) != 64 {
		return fmt.Errorf("encryption.key must be 64 hex characters (32 bytes)")
	}

	return nil
}

// setupLogging configures logging based on log level
func setupLogging(level string) {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	switch level {
	case "debug":
		log.SetPrefix("[DEBUG] ")
	case "info":
		log.SetPrefix("[INFO] ")
	case "warn":
		log.SetPrefix("[WARN] ")
	case "error":
		log.SetPrefix("[ERROR] ")
	default:
		log.SetPrefix("[INFO] ")
	}
}

// printUsage prints usage information
func printUsage() {
	fmt.Fprintf(os.Stderr, `ShadowMesh Daemon v%s

Usage:
  shadowmesh-daemon <config-file>

Arguments:
  config-file    Path to YAML configuration file

Example:
  shadowmesh-daemon /etc/shadowmesh/daemon.yaml

Configuration file format (YAML):
  daemon:
    listen_address: "127.0.0.1:9090"
    log_level: "info"

  network:
    tap_device: "tap0"
    local_ip: "10.0.0.1/24"

  encryption:
    key: "0123456789abcdef..."  # 64 hex chars (32 bytes)

  peer:
    address: ""  # Set via CLI 'connect' command

  nat:
    enabled: true
    stun_server: "stun.l.google.com:19302"

`, version)
}
