package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/shadowmesh/shadowmesh/pkg/client"
	"github.com/shadowmesh/shadowmesh/pkg/layer2"
	"github.com/shadowmesh/shadowmesh/pkg/nat"
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
	tapDevice := flag.String("tap", "", "TAP device name (leave empty to auto-create)")
	tapIP := flag.String("tap-ip", "", "TAP device IP address (e.g., 10.10.10.3)")
	tapNetmask := flag.String("tap-netmask", "24", "TAP device netmask (default: 24)")
	ipAddress := flag.String("ip", "", "External IP address for P2P (required)")
	flag.Parse()

	log.Println("═══════════════════════════════════════════════════════════")
	log.Println("  ShadowMesh Light Node - Layer 2")
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

	log.Printf("✓ P2P listening on port %d", *listenPort)
	log.Println()

	// Create or attach to TAP device
	var tapInterface *layer2.TAPInterface
	if *tapIP != "" {
		// Create new TAP device with IP configuration
		log.Printf("Creating TAP device: %s (IP: %s/%s)", *tapDevice, *tapIP, *tapNetmask)
		tapInterface, err = layer2.NewTAPInterface(*tapDevice, *tapIP, *tapNetmask)
		if err != nil {
			log.Fatalf("Failed to create TAP device: %v", err)
		}
		log.Printf("✓ Created TAP device: %s (IP: %s/%s)", tapInterface.GetName(), *tapIP, *tapNetmask)
	} else if *tapDevice != "" {
		// Attach to existing TAP device
		log.Printf("Attaching to existing TAP device: %s", *tapDevice)
		tapInterface, err = layer2.AttachToExisting(*tapDevice)
		if err != nil {
			log.Fatalf("Failed to attach to TAP device: %v", err)
		}
		log.Printf("✓ Attached to TAP device: %s", tapInterface.GetName())
	} else {
		log.Fatalf("Error: Either -tap-ip or -tap must be specified")
	}
	log.Println()

	// Create NAT traversal candidate exchange
	natExchange := nat.NewCandidateExchange(*backboneURL, keyPair.PeerID, authClient.GetSessionToken())

	// Gather and publish local candidates
	log.Println("Gathering NAT traversal candidates...")
	localCandidates, err := natExchange.GatherLocalCandidates(*listenPort)
	if err != nil {
		log.Printf("Warning: failed to gather local candidates: %v", err)
	} else {
		log.Printf("Gathered %d local candidates", len(localCandidates))

		// Publish candidates to Kademlia-based backbone
		if err := natExchange.PublishCandidates(localCandidates); err != nil {
			log.Printf("Warning: failed to publish candidates: %v", err)
		} else {
			log.Println("✓ Published candidates to backbone")
		}
	}

	// Set up connection handler to start bidirectional forwarding for each peer
	peerManager.SetMessageHandler(func(conn *p2p.Connection, msg *p2p.Message) {
		// Start TAP→P2P forwarding immediately on first message (any type)
		tapForwardingMu.Lock()
		if !tapForwardingStarted[conn.GetPeerID()] {
			tapForwardingStarted[conn.GetPeerID()] = true
			tapForwardingMu.Unlock()
			go forwardTAPFrames(tapInterface, conn)
		} else {
			tapForwardingMu.Unlock()
		}

		// Handle P2P → TAP direction
		if msg.Type == "layer2_frame" {
			startTotal := time.Now()

			// Unmarshal payload
			startUnmarshal := time.Now()
			var payload struct {
				Frame []byte `json:"frame"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				log.Printf("Failed to unmarshal frame: %v", err)
				return
			}
			unmarshalDuration := time.Since(startUnmarshal)

			// Write frame to TAP device
			startTAPWrite := time.Now()
			err := tapInterface.WriteFrame(payload.Frame)
			if err != nil {
				log.Printf("Failed to write to TAP: %v", err)
				return
			}
			tapWriteDuration := time.Since(startTAPWrite)
			totalDuration := time.Since(startTotal)

			// Log timing every 100th frame
			frameCountMu.Lock()
			count := frameRecvCount[conn.GetPeerID()]
			if count%100 == 0 {
				log.Printf("[PROFILE-TAP-RECV-%s] Total=%v UnmarshalPayload=%v TAPWrite=%v FrameSize=%d",
					conn.GetPeerID(), totalDuration, unmarshalDuration, tapWriteDuration, len(payload.Frame))
			}
			frameRecvCount[conn.GetPeerID()]++
			frameCountMu.Unlock()
		}
	})

	// If connecting to a peer, establish P2P connection and create tunnel
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

		// Try direct connection first
		log.Println("Attempting direct connection...")
		err = peerManager.ConnectToPeer(peer.PeerID, peer.IPAddress, peer.Port, keyPair.PeerID)
		if err != nil {
			log.Printf("Direct connection failed: %v", err)
			log.Println("Attempting NAT traversal...")

			// Get remote peer's candidates
			remoteCandidates, err := natExchange.GetCandidates(peer.PeerID)
			if err != nil {
				log.Fatalf("Failed to get remote candidates: %v", err)
			}

			log.Printf("Retrieved %d candidates from peer", len(remoteCandidates))

			// Build candidate address list
			candidateAddrs := make([]string, 0, len(remoteCandidates))
			for _, cand := range remoteCandidates {
				candidateAddrs = append(candidateAddrs, fmt.Sprintf("%s:%d", cand.IP, cand.Port))
			}

			// Try NAT traversal connection
			if err := peerManager.ConnectToPeerWithNAT(peer.PeerID, candidateAddrs, keyPair.PeerID); err != nil {
				log.Fatalf("Failed to connect via NAT traversal: %v", err)
			}

			log.Println("✓ Connected via NAT traversal")
		} else {
			log.Println("✓ Connected directly")
		}

		// Start TAP → P2P forwarding immediately (P2P → TAP handled by message handler)
		log.Println("Starting Layer 2 tunnel...")
		conn := peerManager.GetConnection(peer.PeerID)
		if conn != nil {
			tapForwardingMu.Lock()
			tapForwardingStarted[peer.PeerID] = true
			tapForwardingMu.Unlock()
			go forwardTAPFrames(tapInterface, conn)
		}
		log.Println("✓ Layer 2 tunnel active")
	} else {
		// Listening mode
		log.Println("Waiting for incoming connections...")
	}

	log.Println()
	log.Println("Layer 2 node running. Press Ctrl+C to exit.")
	log.Printf("Peer ID: %s", keyPair.PeerID)
	log.Printf("TAP Device: %s", *tapDevice)
	log.Printf("Listening on: 0.0.0.0:%d", *listenPort)
	log.Println()

	// Wait for shutdown
	waitForShutdown(peerManager, tapInterface)
}

var (
	tapForwardingStarted = make(map[string]bool)
	tapForwardingMu      sync.Mutex
	frameSendCount       = make(map[string]uint64)
	frameRecvCount       = make(map[string]uint64)
	frameCountMu         sync.Mutex
)

// forwardTAPFrames forwards TAP frames to a specific peer connection (TAP → P2P direction)
func forwardTAPFrames(tap *layer2.TAPInterface, conn *p2p.Connection) {
	for {
		if !tap.IsActive() || !conn.IsActive() {
			break
		}

		startTotal := time.Now()

		// TAP read
		startTAPRead := time.Now()
		frame, err := tap.ReadFrame()
		if err != nil {
			if tap.IsActive() {
				log.Printf("TAP read error: %v", err)
			}
			continue
		}
		tapReadDuration := time.Since(startTAPRead)

		// P2P send (includes JSON marshal + TCP write)
		startP2PSend := time.Now()
		err = conn.SendMessage("layer2_frame", map[string]interface{}{
			"frame": frame,
		})
		if err != nil {
			log.Printf("Failed to send frame: %v", err)
			continue
		}
		p2pSendDuration := time.Since(startP2PSend)
		totalDuration := time.Since(startTotal)

		// Log timing every 100th frame
		frameCountMu.Lock()
		count := frameSendCount[conn.GetPeerID()]
		if count%100 == 0 {
			log.Printf("[PROFILE-TAP-SEND-%s] Total=%v TAPRead=%v P2PSend=%v FrameSize=%d",
				conn.GetPeerID(), totalDuration, tapReadDuration, p2pSendDuration, len(frame))
		}
		frameSendCount[conn.GetPeerID()]++
		frameCountMu.Unlock()
	}
}

// waitForShutdown waits for shutdown signal
func waitForShutdown(peerManager *p2p.PeerManager, tap *layer2.TAPInterface) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	fmt.Println()
	log.Printf("Received signal: %v. Shutting down...", sig)

	if err := peerManager.Stop(); err != nil {
		log.Printf("Error stopping peer manager: %v", err)
	}

	if err := tap.Close(); err != nil {
		log.Printf("Error closing TAP device: %v", err)
	}

	log.Println("Shutdown complete")
	os.Exit(0)
}
