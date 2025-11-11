package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shadowmesh/shadowmesh/shared/crypto"
	"github.com/shadowmesh/shadowmesh/shared/protocol"
)

// TestRelayIPDetection is an end-to-end integration test that verifies
// the relay server correctly detects client IP addresses and includes them
// in the ESTABLISHED message
func TestRelayIPDetection(t *testing.T) {
	log.Printf("🧪 Testing Relay IP Detection")

	// Step 1: Generate signing keys for relay and client
	log.Printf("1. Generating ML-DSA-87 signing keys...")
	relaySigningKey, err := crypto.GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate relay signing key: %v", err)
	}

	clientSigningKey, err := crypto.GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate client signing key: %v", err)
	}
	log.Printf("   ✅ Signing keys generated")

	// Step 2: Start relay server
	log.Printf("2. Starting relay server...")

	// Create relay ID
	relayID := [32]byte{}
	copy(relayID[:], []byte("test-relay-server-001"))

	// Create relay handshake handler
	tlsCertManager := NewTLSCertificateManager(relaySigningKey)
	relayHandler := NewRelayHandshakeHandler(relayID, relaySigningKey, tlsCertManager)

	// Create HTTP server with WebSocket upgrade
	upgrader := websocket.Upgrader{
		HandshakeTimeout: 10 * time.Second,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/relay", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Failed to upgrade WebSocket: %v", err)
			return
		}
		defer ws.Close()

		// Create client connection
		client := &ClientConnection{
			conn:        ws,
			receiveChan: make(chan *protocol.Message, 10),
			sendChan:    make(chan *protocol.Message, 10),
		}

		// Start message handler goroutines
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Read goroutine
		go func() {
			for {
				_, data, err := ws.ReadMessage()
				if err != nil {
					return
				}

				msg, err := protocol.DecodeMessage(data)
				if err != nil {
					log.Printf("Failed to decode message: %v", err)
					return
				}

				client.receiveChan <- msg
			}
		}()

		// Write goroutine
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-client.sendChan:
					data, err := protocol.EncodeMessage(msg)
					if err != nil {
						log.Printf("Failed to encode message: %v", err)
						return
					}

					if err := ws.WriteMessage(websocket.BinaryMessage, data); err != nil {
						log.Printf("Failed to write message: %v", err)
						return
					}
				}
			}
		}()

		// Perform handshake
		if err := relayHandler.HandleHandshake(ctx, client); err != nil {
			t.Errorf("Handshake failed: %v", err)
			return
		}

		// Keep connection alive for a moment so client can read ESTABLISHED
		time.Sleep(500 * time.Millisecond)
	})

	// Start HTTP server on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	relayAddr := listener.Addr().String()
	log.Printf("   ✅ Relay server listening on %s", relayAddr)

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()
	defer server.Close()

	// Step 3: Connect client to relay
	log.Printf("3. Connecting client to relay...")

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	clientConn, _, err := dialer.Dial(fmt.Sprintf("ws://%s/relay", relayAddr), nil)
	if err != nil {
		t.Fatalf("Failed to dial relay: %v", err)
	}
	defer clientConn.Close()

	log.Printf("   ✅ Client connected from local address")

	// Step 4: Perform client-side handshake
	log.Printf("4. Performing client-side handshake...")

	// Create client handshake state
	clientID := [32]byte{}
	copy(clientID[:], []byte("test-client-001"))

	clientState, err := protocol.NewClientHandshakeState(clientID, clientSigningKey)
	if err != nil {
		t.Fatalf("Failed to create client handshake state: %v", err)
	}

	// Send HELLO
	helloMsg, err := clientState.CreateHelloMessage()
	if err != nil {
		t.Fatalf("Failed to create HELLO message: %v", err)
	}

	helloData, err := protocol.EncodeMessage(helloMsg)
	if err != nil {
		t.Fatalf("Failed to encode HELLO: %v", err)
	}

	if err := clientConn.WriteMessage(websocket.BinaryMessage, helloData); err != nil {
		t.Fatalf("Failed to send HELLO: %v", err)
	}

	// Receive CHALLENGE
	_, challengeData, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to receive CHALLENGE: %v", err)
	}

	challengeMsg, err := protocol.DecodeMessage(challengeData)
	if err != nil {
		t.Fatalf("Failed to decode CHALLENGE: %v", err)
	}

	challengePayload, ok := challengeMsg.Payload.(*protocol.ChallengeMessage)
	if !ok {
		t.Fatalf("Expected CHALLENGE message, got %T", challengeMsg.Payload)
	}

	// Process CHALLENGE
	if err := clientState.ProcessChallengeMessage(challengePayload); err != nil {
		t.Fatalf("Failed to process CHALLENGE: %v", err)
	}

	// Send RESPONSE
	responseMsg, err := clientState.CreateResponseMessage()
	if err != nil {
		t.Fatalf("Failed to create RESPONSE: %v", err)
	}

	responseData, err := protocol.EncodeMessage(responseMsg)
	if err != nil {
		t.Fatalf("Failed to encode RESPONSE: %v", err)
	}

	if err := clientConn.WriteMessage(websocket.BinaryMessage, responseData); err != nil {
		t.Fatalf("Failed to send RESPONSE: %v", err)
	}

	// Receive ESTABLISHED
	_, establishedData, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to receive ESTABLISHED: %v", err)
	}

	establishedMsg, err := protocol.DecodeMessage(establishedData)
	if err != nil {
		t.Fatalf("Failed to decode ESTABLISHED: %v", err)
	}

	establishedPayload, ok := establishedMsg.Payload.(*protocol.EstablishedMessage)
	if !ok {
		t.Fatalf("Expected ESTABLISHED message, got %T", establishedMsg.Payload)
	}

	log.Printf("   ✅ Handshake completed")

	// Step 5: Extract IP detection results from ESTABLISHED message
	detectedIP := establishedPayload.PeerPublicIP
	detectedPort := establishedPayload.PeerPublicPort
	detectedSupportsP2P := establishedPayload.PeerSupportsDirectP2P

	// Step 6: Verify IP detection
	log.Printf("5. Verifying IP detection...")

	// The relay should have detected the client's IP as 127.0.0.1
	expectedIP := [16]byte{127, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	log.Printf("   Detected IP: %v", formatIPFromArray(detectedIP))
	log.Printf("   Detected Port: %d", detectedPort)
	log.Printf("   Supports Direct P2P: %v", detectedSupportsP2P)

	// Verify IP
	if detectedIP != expectedIP {
		t.Errorf("Expected IP %v, got %v", formatIPFromArray(expectedIP), formatIPFromArray(detectedIP))
	} else {
		log.Printf("   ✅ IP correctly detected as 127.0.0.1")
	}

	// Verify port is non-zero
	if detectedPort == 0 {
		t.Error("Expected non-zero port")
	} else {
		log.Printf("   ✅ Port correctly detected as %d", detectedPort)
	}

	// Verify direct P2P support is enabled
	if !detectedSupportsP2P {
		t.Error("Expected PeerSupportsDirectP2P to be true")
	} else {
		log.Printf("   ✅ Direct P2P support correctly enabled")
	}

	log.Printf("\n🎉 All tests passed!")
	log.Printf("✅ Relay server correctly detects client IP addresses")
	log.Printf("✅ ESTABLISHED message includes real IP/port values")
	log.Printf("✅ Direct P2P support flag set correctly")
}
