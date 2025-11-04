package main

import (
	"fmt"
	"log"
	"time"

	"github.com/shadowmesh/shadowmesh/shared/crypto"
	"github.com/shadowmesh/shadowmesh/shared/protocol"
)

// HandshakeOrchestrator manages the client-side handshake process
type HandshakeOrchestrator struct {
	conn           ConnectionInterface
	handshakeState *protocol.HandshakeState
	clientID       [32]byte
	sigKeys        *crypto.HybridSigningKey
	timeout        time.Duration
}

// NewHandshakeOrchestrator creates a new handshake orchestrator
func NewHandshakeOrchestrator(conn ConnectionInterface, clientID [32]byte, sigKeys *crypto.HybridSigningKey) *HandshakeOrchestrator {
	return &HandshakeOrchestrator{
		conn:     conn,
		clientID: clientID,
		sigKeys:  sigKeys,
		timeout:  protocol.HandshakeTimeout,
	}
}

// PerformHandshake executes the complete 4-message handshake sequence
// Returns the session keys (TX/RX) and session ID upon success
func (ho *HandshakeOrchestrator) PerformHandshake() (*SessionKeys, error) {
	// Step 1: Initialize handshake state
	log.Println("Generating post-quantum keys (ML-KEM-1024, ML-DSA-87)... this may take 10-30 seconds")
	hs, err := protocol.NewClientHandshakeState(ho.clientID, ho.sigKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize handshake state: %w", err)
	}
	log.Println("Post-quantum keys generated successfully")
	ho.handshakeState = hs
	log.Println("Handshake state ready")

	// Step 2: Send HELLO message
	log.Println("About to call sendHello()...")
	if err := ho.sendHello(); err != nil {
		return nil, fmt.Errorf("failed to send HELLO: %w", err)
	}
	log.Println("sendHello() completed successfully")

	// Step 3: Wait for CHALLENGE message
	challengeMsg, err := ho.waitForMessage(protocol.MsgTypeChallenge, ho.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to receive CHALLENGE: %w", err)
	}

	// Step 4: Process CHALLENGE
	if err := ho.processChallenge(challengeMsg); err != nil {
		return nil, fmt.Errorf("failed to process CHALLENGE: %w", err)
	}

	// Step 5: Send RESPONSE
	if err := ho.sendResponse(); err != nil {
		return nil, fmt.Errorf("failed to send RESPONSE: %w", err)
	}

	// Step 6: Wait for ESTABLISHED message
	establishedMsg, err := ho.waitForMessage(protocol.MsgTypeEstablished, ho.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to receive ESTABLISHED: %w", err)
	}

	// Step 7: Process ESTABLISHED
	sessionKeys, err := ho.processEstablished(establishedMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to process ESTABLISHED: %w", err)
	}

	// Step 8: Handshake complete
	log.Println("Handshake sequence complete")

	return sessionKeys, nil
}

// sendHello creates and sends a HELLO message
func (ho *HandshakeOrchestrator) sendHello() error {
	log.Println("Creating HELLO message (generating ephemeral Kyber keys)...")
	helloMsg, err := ho.handshakeState.CreateHelloMessage()
	if err != nil {
		log.Printf("ERROR: Failed to create HELLO message: %v", err)
		return fmt.Errorf("failed to create HELLO message: %w", err)
	}
	log.Println("HELLO message created successfully")

	log.Println("Sending HELLO message to relay...")
	if err := ho.conn.SendMessage(helloMsg); err != nil {
		return fmt.Errorf("failed to send HELLO message: %w", err)
	}
	log.Println("HELLO message queued, waiting for CHALLENGE...")

	return nil
}

// processChallenge processes a received CHALLENGE message
func (ho *HandshakeOrchestrator) processChallenge(msg *protocol.Message) error {
	challengePayload, ok := msg.Payload.(*protocol.ChallengeMessage)
	if !ok {
		return fmt.Errorf("invalid CHALLENGE payload type")
	}

	if err := ho.handshakeState.ProcessChallengeMessage(challengePayload); err != nil {
		return fmt.Errorf("handshake state failed to process CHALLENGE: %w", err)
	}

	return nil
}

// sendResponse creates and sends a RESPONSE message
func (ho *HandshakeOrchestrator) sendResponse() error {
	responseMsg, err := ho.handshakeState.CreateResponseMessage()
	if err != nil {
		return fmt.Errorf("failed to create RESPONSE message: %w", err)
	}

	if err := ho.conn.SendMessage(responseMsg); err != nil {
		return fmt.Errorf("failed to send RESPONSE message: %w", err)
	}

	return nil
}

// processEstablished processes an ESTABLISHED message and derives session keys
func (ho *HandshakeOrchestrator) processEstablished(msg *protocol.Message) (*SessionKeys, error) {
	establishedPayload, ok := msg.Payload.(*protocol.EstablishedMessage)
	if !ok {
		return nil, fmt.Errorf("invalid ESTABLISHED payload type")
	}

	// Verify session ID matches
	if establishedPayload.SessionID != ho.handshakeState.SessionID {
		return nil, fmt.Errorf("session ID mismatch")
	}

	// Derive session keys
	if err := ho.handshakeState.DeriveSessionKeys(); err != nil {
		return nil, fmt.Errorf("failed to derive session keys: %w", err)
	}

	// Create session keys structure
	var txKey, rxKey [32]byte
	copy(txKey[:], ho.handshakeState.TXKey)
	copy(rxKey[:], ho.handshakeState.RXKey)

	sessionKeys := &SessionKeys{
		// Epic 1: Original fields
		SessionID:           establishedPayload.SessionID,
		TXKey:               txKey,
		RXKey:               rxKey,
		HeartbeatInterval:   time.Duration(establishedPayload.HeartbeatInterval) * time.Second,
		MTU:                 establishedPayload.MTU,
		KeyRotationInterval: time.Duration(establishedPayload.KeyRotationInterval) * time.Second,
		ServerCapabilities:  establishedPayload.ServerCapabilities,

		// Epic 2: Direct P2P fields
		PeerPublicIP:          establishedPayload.PeerPublicIP,
		PeerPublicPort:        establishedPayload.PeerPublicPort,
		PeerSupportsDirectP2P: establishedPayload.PeerSupportsDirectP2P,

		// Epic 2: TLS certificate fields
		PeerTLSCert:    establishedPayload.PeerTLSCertificate,
		PeerTLSCertSig: establishedPayload.PeerTLSCertSignature,
	}

	return sessionKeys, nil
}

// waitForMessage waits for a specific message type with timeout
func (ho *HandshakeOrchestrator) waitForMessage(expectedType byte, timeout time.Duration) (*protocol.Message, error) {
	log.Printf("Waiting for %s message (timeout: %v)...", protocol.MessageTypeName(expectedType), timeout)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			log.Printf("Timeout waiting for %s message after %v", protocol.MessageTypeName(expectedType), timeout)
			return nil, fmt.Errorf("timeout waiting for %s message", protocol.MessageTypeName(expectedType))

		case msg := <-ho.conn.ReceiveChannel():
			log.Printf("Received message type: %s", protocol.MessageTypeName(msg.Header.Type))
			// Handle ERROR messages
			if msg.Header.Type == protocol.MsgTypeError {
				errorPayload, ok := msg.Payload.(*protocol.ErrorMessage)
				if ok {
					return nil, fmt.Errorf("received ERROR from relay: %s (%s)",
						errorPayload.ErrorMessage, protocol.ErrorCodeName(errorPayload.ErrorCode))
				}
				return nil, fmt.Errorf("received malformed ERROR message")
			}

			// Check if this is the expected message type
			if msg.Header.Type == expectedType {
				return msg, nil
			}

			// Unexpected message type
			return nil, fmt.Errorf("unexpected message type: got %s, expected %s",
				protocol.MessageTypeName(msg.Header.Type), protocol.MessageTypeName(expectedType))

		case err := <-ho.conn.ErrorChannel():
			log.Printf("Connection error during handshake: %v", err)
			return nil, fmt.Errorf("connection error during handshake: %w", err)
		}
	}
}

// PerformKeyRotation performs a key rotation (re-handshake)
func (ho *HandshakeOrchestrator) PerformKeyRotation() (*SessionKeys, error) {
	// Generate new ephemeral keys
	newHS, err := protocol.NewClientHandshakeState(ho.clientID, ho.sigKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize key rotation state: %w", err)
	}
	ho.handshakeState = newHS

	// Create HELLO with rotation flag
	helloMsg, err := newHS.CreateHelloMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to create rotation HELLO: %w", err)
	}

	// Set rotation flag
	helloMsg.Header.Flags = protocol.FlagKeyRotation

	// Send and follow normal handshake flow
	if err := ho.conn.SendMessage(helloMsg); err != nil {
		return nil, fmt.Errorf("failed to send rotation HELLO: %w", err)
	}

	// Wait for CHALLENGE
	challengeMsg, err := ho.waitForMessage(protocol.MsgTypeChallenge, ho.timeout)
	if err != nil {
		return nil, fmt.Errorf("rotation: failed to receive CHALLENGE: %w", err)
	}

	if err := ho.processChallenge(challengeMsg); err != nil {
		return nil, fmt.Errorf("rotation: failed to process CHALLENGE: %w", err)
	}

	if err := ho.sendResponse(); err != nil {
		return nil, fmt.Errorf("rotation: failed to send RESPONSE: %w", err)
	}

	establishedMsg, err := ho.waitForMessage(protocol.MsgTypeEstablished, ho.timeout)
	if err != nil {
		return nil, fmt.Errorf("rotation: failed to receive ESTABLISHED: %w", err)
	}

	sessionKeys, err := ho.processEstablished(establishedMsg)
	if err != nil {
		return nil, fmt.Errorf("rotation: failed to process ESTABLISHED: %w", err)
	}

	return sessionKeys, nil
}
