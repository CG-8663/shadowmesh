package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/shadowmesh/shadowmesh/pkg/relay"
)

const version = "v0.2.0-relay"

func main() {
	log.Printf("🚀 ShadowMesh Relay Server %s", version)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Parse command line flags
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	log.Printf("📋 Loading configuration from: %s", *configFile)
	config, err := relay.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	log.Printf("✅ Configuration loaded successfully")
	log.Printf("   Region: %s", config.Region)
	log.Printf("   Relay Port: %d", config.RelayPort)
	log.Printf("   Max Connections: %d", config.MaxConnections)
	log.Printf("   Health Port: %d", config.HealthPort)
	log.Printf("   Metrics Port: %d", config.MetricsPort)

	// Create relay server
	log.Println("🔧 Initializing relay server...")
	server, err := relay.NewServer(config)
	if err != nil {
		log.Fatalf("❌ Failed to create relay server: %v", err)
	}

	// Start health check endpoint
	log.Printf("💚 Starting health check endpoint on :%d", config.HealthPort)
	go server.StartHealthCheck()

	// Start Prometheus metrics endpoint
	log.Printf("📊 Starting metrics endpoint on :%d", config.MetricsPort)
	go server.StartMetrics()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n⚠️  Shutdown signal received, stopping relay server...")
		server.Stop()
		os.Exit(0)
	}()

	// Start relay server
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("✅ ShadowMesh Relay Server listening on :%d", config.RelayPort)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println()
	log.Println("🔐 Zero-Knowledge Relay: Encrypted frames only, no plaintext access")
	log.Printf("🌍 Region: %s", config.Region)
	log.Printf("👥 Max Connections: %d", config.MaxConnections)
	log.Println()
	log.Println("Press Ctrl+C to shutdown gracefully")
	log.Println()

	// Start serving
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ Relay server error: %v", err)
	}
}
