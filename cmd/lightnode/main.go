package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shadowmesh/shadowmesh/pkg/client"
	"github.com/shadowmesh/shadowmesh/pkg/p2p"
)

func main() {
	// Parse command-line flags
	keyDir := flag.String("keydir", "./keys", "Directory to store keys")
	generateKeys := flag.Bool("generate-keys", false, "Generate new key pair")
	backboneURL := flag.String("backbone", "http://209.151.148.121:8080", "Discovery backbone URL")
	listenPort := flag.Int("port", 8443, "Port to listen for P2P connections")
	isPublic := flag.Bool("public", false, "Register as PUBLIC relay (default: PRIVATE)")
	connectTo := flag.String("connect", "", "Peer ID to connect to")
	testVideo := flag.Bool("test-video", false, "Test video streaming")
	ipAddress := flag.String("ip", "", "External IP address (required)")
	flag.Parse()

	log.Println("═══════════════════════════════════════════════════════════")
	log.Println("  ShadowMesh Light Node")
	log.Println("═══════════════════════════════════════════════════════════")
	log.Println()

	// Generate keys if requested
	if *generateKeys {
		log.Println("Generating new ML-DSA-87 key pair...")
		keyPair, err := client.GenerateKeyPair()
		if err != nil {
			log.Fatalf("Failed to generate keys: %v", err)
		}

		if err := keyPair.Save(*keyDir); err != nil {
			log.Fatalf("Failed to save keys: %v", err)
		}

		log.Printf("Keys generated and saved to: %s", *keyDir)
		log.Printf("Peer ID: %s", keyPair.PeerID)
		return
	}

	// Validate IP address is provided
	if *ipAddress == "" {
		log.Fatalf("Error: -ip flag is required (e.g., -ip 100.90.48.10)")
	}

	// Load keys
	log.Printf("Loading keys from: %s", *keyDir)
	keyPair, err := client.LoadKeyPair(*keyDir)
	if err != nil {
		log.Fatalf("Failed to load keys: %v (run with -generate-keys to create new keys)", err)
	}

	log.Printf("Peer ID: %s", keyPair.PeerID)
	log.Println()

	// Create authentication client
	log.Printf("Connecting to discovery backbone: %s", *backboneURL)
	authClient := client.NewAuthClient(*backboneURL, keyPair)

	// Authenticate with backbone
	log.Println("Authenticating with backbone...")
	if err := authClient.Authenticate(); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	log.Println("✓ Authentication successful")

	// Register as peer
	log.Printf("Using IP address: %s", *ipAddress)
	log.Printf("Registering as peer (PUBLIC=%v, port=%d)...", *isPublic, *listenPort)
	if err := authClient.RegisterPeer(*ipAddress, *listenPort, *isPublic); err != nil {
		log.Fatalf("Failed to register peer: %v", err)
	}
	log.Println("✓ Peer registered successfully")
	log.Println()

	// Start P2P manager
	log.Println("Starting P2P manager...")
	peerManager := p2p.NewPeerManager()

	if err := peerManager.StartListening(*listenPort); err != nil {
		log.Fatalf("Failed to start P2P listener: %v", err)
	}

	// Set message handler
	peerManager.SetMessageHandler(func(conn *p2p.Connection, msg *p2p.Message) {
		handleMessage(conn, msg, *testVideo)
	})

	log.Printf("✓ P2P listening on port %d", *listenPort)
	log.Println()

	// Connect to specific peer if requested
	if *connectTo != "" {
		log.Printf("Looking up peer: %s", *connectTo)
		peers, err := authClient.FindPeers(*connectTo, 1)
		if err != nil {
			log.Fatalf("Failed to find peer: %v", err)
		}

		if len(peers) == 0 {
			log.Fatalf("Peer not found: %s", *connectTo)
		}

		peer := peers[0]
		log.Printf("Found peer: %s at %s:%d", peer.PeerID, peer.IPAddress, peer.Port)

		log.Println("Connecting to peer...")
		if err := peerManager.ConnectToPeer(peer.PeerID, peer.IPAddress, peer.Port, keyPair.PeerID); err != nil {
			log.Fatalf("Failed to connect to peer: %v", err)
		}

		log.Println("✓ Connected to peer")

		// Send test message
		if err := peerManager.SendToPeer(peer.PeerID, "ping", map[string]string{"message": "hello"}); err != nil {
			log.Printf("Failed to send ping: %v", err)
		} else {
			log.Println("✓ Sent ping to peer")
		}

		// If test video mode, start sending video data
		if *testVideo {
			go sendTestVideo(peerManager, peer.PeerID)
		}
	}

	log.Println("Light node running. Press Ctrl+C to exit.")
	log.Printf("Peer ID: %s", keyPair.PeerID)
	log.Printf("Listening on: 0.0.0.0:%d", *listenPort)
	log.Println()

	// Wait for shutdown
	waitForShutdown(peerManager)
}

// handleMessage handles incoming P2P messages
func handleMessage(conn *p2p.Connection, msg *p2p.Message, testVideo bool) {
	switch msg.Type {
	case "ping":
		var payload map[string]string
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Failed to parse ping: %v", err)
			return
		}

		log.Printf("Received ping from %s: %s", conn.GetPeerID(), payload["message"])

		// Send pong
		conn.SendMessage("pong", map[string]string{"message": "pong"})

	case "pong":
		var payload map[string]string
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Failed to parse pong: %v", err)
			return
		}

		log.Printf("Received pong from %s: %s", conn.GetPeerID(), payload["message"])

	case "video":
		var payload struct {
			FrameNumber int    `json:"frame_number"`
			DataSize    int    `json:"data_size"`
			Timestamp   int64  `json:"timestamp"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Failed to parse video: %v", err)
			return
		}

		latency := time.Now().Unix() - payload.Timestamp
		log.Printf("Received video frame #%d from %s (size=%d bytes, latency=%dms)",
			payload.FrameNumber, conn.GetPeerID(), payload.DataSize, latency*1000)

	default:
		log.Printf("Received unknown message type: %s from %s", msg.Type, conn.GetPeerID())
	}
}

// sendTestVideo sends test video frames
func sendTestVideo(pm *p2p.PeerManager, peerID string) {
	frameNumber := 0
	ticker := time.NewTicker(33 * time.Millisecond) // ~30 FPS
	defer ticker.Stop()

	log.Println("Starting test video stream (30 FPS)...")

	for range ticker.C {
		frameNumber++

		// Simulate video frame (10KB each)
		payload := map[string]interface{}{
			"frame_number": frameNumber,
			"data_size":    10240,
			"timestamp":    time.Now().Unix(),
		}

		if err := pm.SendToPeer(peerID, "video", payload); err != nil {
			log.Printf("Failed to send video frame: %v", err)
			return
		}

		if frameNumber%30 == 0 {
			log.Printf("Sent %d video frames", frameNumber)
		}
	}
}

// waitForShutdown waits for shutdown signal
func waitForShutdown(peerManager *p2p.PeerManager) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	fmt.Println()
	log.Printf("Received signal: %v. Shutting down...", sig)

	if err := peerManager.Stop(); err != nil {
		log.Printf("Error stopping peer manager: %v", err)
	}

	log.Println("Shutdown complete")
	os.Exit(0)
}
