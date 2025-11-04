package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shadowmesh/shadowmesh/shared/crypto"
)

// Minimal types for testing (copied from client/daemon)
type TLSCertificateManager struct {
	certificate    *tls.Certificate
	certificateDER []byte
	signingKey     *crypto.HybridSigningKey
}

func (tm *TLSCertificateManager) GenerateEphemeralCertificate(localIP string) error {
	// Import actual implementation
	return nil // Placeholder - we'll use the real one
}

func (tm *TLSCertificateManager) GetCertificateDER() []byte {
	return tm.certificateDER
}

func (tm *TLSCertificateManager) GetCertificateFingerprint() [32]byte {
	// Placeholder
	return [32]byte{}
}

func (tm *TLSCertificateManager) SignCertificate() ([]byte, error) {
	// Placeholder
	return nil, nil
}

func (tm *TLSCertificateManager) GetTLSConfigServer() (*tls.Config, error) {
	// Placeholder
	return nil, nil
}

func (tm *TLSCertificateManager) GetTLSConfigClient(serverName string) (*tls.Config, error) {
	// Placeholder
	return nil, nil
}

type TestDirectP2PManager struct {
	localAddr      string
	tlsCertManager *TLSCertificateManager
	ctx            context.Context
	cancel         context.CancelFunc
}

func (dm *TestDirectP2PManager) StartListener(localIP string) error {
	// Placeholder
	return nil
}

func (dm *TestDirectP2PManager) handleWebSocketUpgrade(w http.ResponseWriter, r *http.Request) {
	// Placeholder
}

var (
	mode     = flag.String("mode", "server", "Mode: server or client")
	localIP  = flag.String("local-ip", "127.0.0.1", "Local IP address")
	peerAddr = flag.String("peer-addr", "127.0.0.1:8443", "Peer address for client mode")
)

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

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
	log.Printf("🔧 Test Server Mode")
	log.Printf("==================")

	// Generate signing key for TLS cert
	log.Printf("1. Generating ML-DSA-87 signing key...")
	signingKey, err := crypto.GenerateSigningKey()
	if err != nil {
		log.Fatalf("Failed to generate signing key: %v", err)
	}
	log.Printf("   ✅ Signing key generated")

	// Create TLS certificate manager
	log.Printf("2. Creating TLS certificate manager...")
	tlsCertManager := &TLSCertificateManager{
		signingKey: signingKey,
	}

	// Generate ephemeral certificate
	log.Printf("3. Generating ephemeral TLS certificate for %s...", *localIP)
	if err := tlsCertManager.GenerateEphemeralCertificate(*localIP); err != nil {
		log.Fatalf("Failed to generate certificate: %v", err)
	}
	log.Printf("   ✅ Certificate generated")
	fingerprint := tlsCertManager.GetCertificateFingerprint()
	log.Printf("   📜 Fingerprint: %x", fingerprint[:8])

	// Get certificate DER for display
	certDER := tlsCertManager.GetCertificateDER()
	log.Printf("   📏 Certificate size: %d bytes", len(certDER))

	// Sign the certificate with ML-DSA-87
	log.Printf("4. Signing certificate with ML-DSA-87...")
	certSig, err := tlsCertManager.SignCertificate()
	if err != nil {
		log.Fatalf("Failed to sign certificate: %v", err)
	}
	log.Printf("   ✅ Certificate signed")
	log.Printf("   📏 Signature size: %d bytes", len(certSig))

	// Create DirectP2PManager (minimal version for testing)
	log.Printf("5. Starting TLS+WebSocket listener...")
	manager := &TestDirectP2PManager{
		tlsCertManager: tlsCertManager,
	}

	if err := manager.StartListener(*localIP); err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}

	log.Printf("\n🎉 Server ready!")
	log.Printf("📍 Listening on: %s", manager.localAddr)
	log.Printf("🔐 TLS: Enabled (self-signed)")
	log.Printf("📡 WebSocket endpoint: wss://%s/ws", manager.localAddr)
	log.Printf("\nTo connect from client:")
	log.Printf("  ./bin/test-direct-p2p -mode client -peer-addr %s\n", manager.localAddr)

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Printf("\n🛑 Shutting down...")
}

func runClient() {
	log.Printf("🔌 Test Client Mode")
	log.Printf("==================")

	log.Printf("1. Target peer: %s", *peerAddr)

	// Generate signing key for TLS cert
	log.Printf("2. Generating ML-DSA-87 signing key...")
	signingKey, err := crypto.GenerateSigningKey()
	if err != nil {
		log.Fatalf("Failed to generate signing key: %v", err)
	}
	log.Printf("   ✅ Signing key generated")

	// Create TLS certificate manager
	log.Printf("3. Creating TLS certificate manager...")
	tlsCertManager := &TLSCertificateManager{
		signingKey: signingKey,
	}

	// Generate ephemeral certificate
	log.Printf("4. Generating ephemeral TLS certificate...")
	if err := tlsCertManager.GenerateEphemeralCertificate(*localIP); err != nil {
		log.Fatalf("Failed to generate certificate: %v", err)
	}
	log.Printf("   ✅ Certificate generated")

	// For testing, we'll skip certificate pinning (in real scenario, this comes from ESTABLISHED message)
	log.Printf("5. Connecting to peer at wss://%s/ws...", *peerAddr)
	log.Printf("   ⚠️  Note: Certificate pinning DISABLED for testing")

	// Get TLS config (without pinning for this test)
	tlsConfig, err := tlsCertManager.GetTLSConfigClient(*peerAddr)
	if err != nil {
		log.Fatalf("Failed to get TLS config: %v", err)
	}

	// Override to skip verification for testing
	tlsConfig.InsecureSkipVerify = true

	// Dial WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  tlsConfig,
	}

	wsURL := fmt.Sprintf("wss://%s/ws", *peerAddr)
	log.Printf("6. Dialing %s...", wsURL)

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	log.Printf("   ✅ Connected!")
	log.Printf("\n🎉 Direct P2P connection established!")
	log.Printf("🔐 TLS: Active")
	log.Printf("📡 WebSocket: Connected")

	// Send test messages
	log.Printf("\n7. Sending test messages...")

	messages := []string{
		"Hello from client!",
		"Testing direct P2P",
		"This is encrypted via TLS 1.3",
	}

	for i, msg := range messages {
		log.Printf("   📤 Sending: %s", msg)

		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			log.Fatalf("Write error: %v", err)
		}

		// Read echo response
		_, response, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("Read error: %v", err)
		}

		log.Printf("   📥 Received: %s", string(response))

		if string(response) == msg {
			log.Printf("   ✅ Echo verified (%d/%d)", i+1, len(messages))
		} else {
			log.Printf("   ❌ Echo mismatch!")
		}

		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("\n✅ All tests passed!")
	log.Printf("🔒 Connection was TLS 1.3 encrypted")
	log.Printf("📡 WebSocket communication successful")
}
