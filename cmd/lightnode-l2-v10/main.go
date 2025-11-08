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
	"github.com/shadowmesh/shadowmesh/pkg/layer2"
	"github.com/shadowmesh/shadowmesh/pkg/nat"
	"github.com/shadowmesh/shadowmesh/pkg/p2p"
)

func main() {
	// Parse command-line flags
	keyDir := flag.String("keydir", "./keys", "Directory to store keys")
	generateKeys := flag.Bool("generate-keys", false, "Generate new key pair")
	backboneURL := flag.String("backbone", "http://209.151.148.121:8080", "Discovery backbone URL")
	listenPort := flag.Int("port", 8443, "Port to listen for P2P connections (TCP control)")
	udpPort := flag.Int("udp-port", 9443, "Port for UDP data path")
	isPublic := flag.Bool("public", false, "Register as PUBLIC relay (default: PRIVATE)")
	connectTo := flag.String("connect", "", "Peer ID to connect to")
	tapDevice := flag.String("tap", "", "TAP device name (leave empty to auto-create)")
	tapIP := flag.String("tap-ip", "", "TAP device IP address (e.g., 10.10.10.3)")
	tapNetmask := flag.String("tap-netmask", "24", "TAP device netmask (default: 24)")
	ipAddress := flag.String("ip", "", "External IP address for P2P (required)")
	flag.Parse()

	log.Println("═══════════════════════════════════════════════════════════")
	log.Println("  ShadowMesh Light Node - Layer 2 v10 (UDP Data Path)")
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
	log.Printf("Registering as peer (PUBLIC=%v, TCP port=%d, UDP port=%d)...", *isPublic, *listenPort, *udpPort)
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

	log.Printf("✓ P2P TCP control listening on port %d", *listenPort)
	log.Printf("✓ UDP data path will use port %d", *udpPort)
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

	// Helper function to setup UDP receive handler
	setupUDPReceiver := func(peerID string, udpConn *p2p.UDPConnection) {
		udpConn.SetFrameHandler(func(frame []byte) {
			err := tapInterface.WriteFrame(frame)
			if err != nil {
				log.Printf("Failed to write frame to TAP from %s: %v", peerID, err)
			}
		})
		udpConn.StartReceiving()
		go forwardTAPFramesUDP(tapInterface, udpConn)
		log.Printf("✓ UDP receive handler ready for %s", peerID)
	}

	// Set up TCP message handler for control messages and UDP endpoint exchange
	peerManager.SetMessageHandler(func(conn *p2p.Connection, msg *p2p.Message) {
		// TCP control path handles keepalives, signaling, and UDP endpoint exchange
		// UDP path handles layer2_frame messages
		if msg.Type == "keepalive" {
			log.Printf("Received keepalive from %s", conn.GetPeerID())
		} else if msg.Type == "udp_endpoint" {
			// Handle UDP endpoint exchange
			var remoteEndpoint struct {
				IP   string `json:"ip"`
				Port int    `json:"port"`
			}
			if err := json.Unmarshal(msg.Payload, &remoteEndpoint); err != nil {
				log.Printf("Failed to unmarshal UDP endpoint from %s: %v", conn.GetPeerID(), err)
				return
			}

			log.Printf("Received UDP endpoint from %s: %s:%d", conn.GetPeerID(), remoteEndpoint.IP, remoteEndpoint.Port)

			// Get or create UDP connection
			udpConn := peerManager.GetUDPConnection(conn.GetPeerID())
			if udpConn == nil {
				// Create UDP connection if not already exists
				var err error
				udpConn, err = peerManager.SetupUDPConnection(conn.GetPeerID(), *ipAddress, *udpPort)
				if err != nil {
					log.Printf("Failed to setup UDP connection for %s: %v", conn.GetPeerID(), err)
					return
				}
				// Setup receiver immediately for listener side
				setupUDPReceiver(conn.GetPeerID(), udpConn)
			}

			// Set remote address
			if err := udpConn.SetRemoteAddr(remoteEndpoint.IP, remoteEndpoint.Port); err != nil {
				log.Printf("Failed to set remote UDP address for %s: %v", conn.GetPeerID(), err)
				return
			}

			log.Printf("✓ UDP data path established with %s", conn.GetPeerID())
		} else if msg.Type != "layer2_frame" {
			log.Printf("Received control message type: %s from %s", msg.Type, conn.GetPeerID())
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

		// Try direct TCP connection first (control plane)
		log.Println("Attempting direct TCP connection for control plane...")
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

		// Setup UDP connection for data path
		log.Println("Setting up UDP data path...")
		udpConn, err := peerManager.SetupUDPConnection(peer.PeerID, *ipAddress, *udpPort)
		if err != nil {
			log.Fatalf("Failed to setup UDP connection: %v", err)
		}

		// Setup UDP receiver immediately for initiator side
		setupUDPReceiver(peer.PeerID, udpConn)

		// Send our UDP endpoint to peer (response will be handled asynchronously by message handler)
		tcpConn := peerManager.GetConnection(peer.PeerID)
		if tcpConn != nil {
			err = tcpConn.SendMessage("udp_endpoint", map[string]interface{}{
				"ip":   *ipAddress,
				"port": *udpPort,
			})
			if err != nil {
				log.Fatalf("Failed to send UDP endpoint: %v", err)
			}
			log.Printf("Sent UDP endpoint to %s: %s:%d", peer.PeerID, *ipAddress, *udpPort)
			log.Println("Waiting for peer's UDP endpoint response (handled by message handler)...")
		}
	} else {
		// Listening mode - setup UDP for incoming connections
		log.Println("Waiting for incoming connections...")
		log.Println("(UDP data path will be established after TCP handshake)")
	}

	log.Println()
	log.Println("Layer 2 node running. Press Ctrl+C to exit.")
	log.Printf("Peer ID: %s", keyPair.PeerID)
	log.Printf("TAP Device: %s", *tapDevice)
	log.Printf("Listening on: TCP 0.0.0.0:%d, UDP 0.0.0.0:%d", *listenPort, *udpPort)
	log.Println()

	// Wait for shutdown
	waitForShutdown(peerManager, tapInterface)
}

// forwardTAPFramesUDP forwards TAP frames to UDP connection (TAP → UDP direction)
func forwardTAPFramesUDP(tap *layer2.TAPInterface, udpConn *p2p.UDPConnection) {
	var frameCount uint64
	for {
		if !tap.IsActive() || !udpConn.IsActive() {
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

		// UDP send (includes frame header + UDP write)
		startUDPSend := time.Now()
		err = udpConn.SendFrame(frame)
		if err != nil {
			log.Printf("Failed to send frame via UDP: %v", err)
			continue
		}
		udpSendDuration := time.Since(startUDPSend)
		totalDuration := time.Since(startTotal)

		// Log timing every 100th frame
		frameCount++
		if frameCount%100 == 0 {
			log.Printf("[PROFILE-TAP-UDP-%s] Total=%v TAPRead=%v UDPSend=%v FrameSize=%d",
				udpConn.GetPeerID(), totalDuration, tapReadDuration, udpSendDuration, len(frame))
		}
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
