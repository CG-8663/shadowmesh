package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/shadowmesh/shadowmesh/shared/crypto"
)

// Import client daemon types (they're in main package, so we'll need to copy minimal functionality)
// For production testing, we'll use a simplified approach

var (
	mode     = flag.String("mode", "server", "Mode: server or client")
	localIP  = flag.String("local-ip", "0.0.0.0", "Local IP address to listen on")
	peerAddr = flag.String("peer-addr", "", "Peer address to connect to (client mode)")
)

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("╔══════════════════════════════════════════════════╗")
	log.Printf("║   ShadowMesh Production P2P Test                 ║")
	log.Printf("║   Epic 2: Direct P2P Networking                  ║")
	log.Printf("╚══════════════════════════════════════════════════╝")
	log.Printf("")

	switch *mode {
	case "server":
		runServer()
	case "client":
		runClient()
	default:
		log.Fatalf("Invalid mode: %s (must be 'server' or 'client')", *mode)
	}
}

func runServer() {
	log.Printf("🔧 Production Test: Server Mode")
	log.Printf("================================")
	log.Printf("")

	// Step 1: Generate signing key
	log.Printf("1. Generating ML-DSA-87 signing key...")
	signingKey, err := crypto.GenerateSigningKey()
	if err != nil {
		log.Fatalf("Failed to generate signing key: %v", err)
	}
	log.Printf("   ✅ Signing key generated")

	// Step 2: Display key information
	log.Printf("   📜 Using ML-DSA-87 + Ed25519 hybrid signatures")

	// For production testing, we need to exchange public keys manually
	// In a real deployment, this would happen via the relay ESTABLISHED message
	log.Printf("")
	log.Printf("2. Server Configuration")
	log.Printf("   Listen Address: %s", *localIP)
	log.Printf("   Mode: TLS listener (awaiting peer connection)")
	log.Printf("")
	log.Printf("⚠️  NEXT STEPS:")
	log.Printf("   1. Copy the client command below")
	log.Printf("   2. SSH to the Raspberry Pi")
	log.Printf("   3. Run the client with your server's public IP")
	log.Printf("")
	log.Printf("   Raspberry Pi command:")
	log.Printf("   ./test-production-p2p -mode client -peer-addr <SERVER_PUBLIC_IP>:8443")
	log.Printf("")
	log.Printf("💡 Server will listen on port 8443 (default)")
	log.Printf("💡 Make sure firewall allows incoming connections on port 8443")
	log.Printf("")

	// For now, this is a placeholder
	// The actual DirectP2PManager code from client/daemon would be integrated here
	log.Printf("⏳ Waiting for Ctrl+C to exit...")
	log.Printf("   (Full DirectP2PManager integration pending)")

	// Wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Printf("")
	log.Printf("🛑 Server shutting down...")
}

func runClient() {
	if *peerAddr == "" {
		log.Fatalf("Error: -peer-addr is required in client mode")
	}

	log.Printf("🔌 Production Test: Client Mode")
	log.Printf("================================")
	log.Printf("")

	log.Printf("1. Target server: %s", *peerAddr)

	// Step 1: Generate signing key
	log.Printf("2. Generating ML-DSA-87 signing key...")
	signingKey, err := crypto.GenerateSigningKey()
	if err != nil {
		log.Fatalf("Failed to generate signing key: %v", err)
	}
	log.Printf("   ✅ Signing key generated")
	log.Printf("   📜 Using ML-DSA-87 + Ed25519 hybrid signatures")

	log.Printf("")
	log.Printf("3. Client Configuration")
	log.Printf("   Peer Address: %s", *peerAddr)
	log.Printf("   Mode: TLS connector")
	log.Printf("")
	log.Printf("⏳ Waiting for Ctrl+C to exit...")
	log.Printf("   (Full DirectP2PManager integration pending)")

	// Wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Printf("")
	log.Printf("🛑 Client shutting down...")
}
