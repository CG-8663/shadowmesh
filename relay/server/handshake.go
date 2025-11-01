package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shadowmesh/shadowmesh/shared/crypto"
	"github.com/shadowmesh/shadowmesh/shared/protocol"
)

// RelayHandshakeHandler handles the relay side of the handshake protocol
type RelayHandshakeHandler struct {
	relayID  [32]byte
	sigKeys  *crypto.HybridSigningKey
}

// NewRelayHandshakeHandler creates a new relay handshake handler
func NewRelayHandshakeHandler(relayID [32]byte, sigKeys *crypto.HybridSigningKey) *RelayHandshakeHandler {
	return &RelayHandshakeHandler{
		relayID: relayID,
		sigKeys: sigKeys,
	}
}

// HandleHandshake performs the relay side of the handshake protocol
//
// Handshake flow (4 messages):
// 1. Client → Relay: HELLO (client identity + KEM public key)
// 2. Relay → Client: CHALLENGE (relay identity + KEM ciphertext)
// 3. Client → Relay: RESPONSE (session proof)
// 4. Relay → Client: ESTABLISHED (confirmation)
//
// After successful handshake, both parties derive symmetric session keys.
func (rh *RelayHandshakeHandler) HandleHandshake(ctx context.Context, client *ClientConnection) error {
	log.Printf("Starting handshake with client from %s", client.conn.RemoteAddr())

	// Create handshake state
	handshakeState, err := protocol.NewRelayHandshakeState(rh.relayID, rh.sigKeys)
	if err != nil {
		return fmt.Errorf("failed to create handshake state: %w", err)
	}

	// Step 1: Receive HELLO message from client
	helloMsg, err := rh.receiveMessage(ctx, client, protocol.MsgTypeHello)
	if err != nil {
		return fmt.Errorf("failed to receive HELLO: %w", err)
	}

	helloPayload, ok := helloMsg.Payload.(*protocol.HelloMessage)
	if !ok {
		return fmt.Errorf("invalid HELLO payload type")
	}

	log.Printf("Received HELLO from client %x", helloPayload.ClientID[:8])

	// Process HELLO message
	if err := handshakeState.ProcessHelloMessage(helloPayload); err != nil {
		return fmt.Errorf("failed to process HELLO: %w", err)
	}

	// Store client ID
	client.clientID = helloPayload.ClientID

	// Step 2: Create and send CHALLENGE message
	challengeMsg, err := handshakeState.CreateChallengeMessage()
	if err != nil {
		return fmt.Errorf("failed to create CHALLENGE: %w", err)
	}

	if err := rh.sendMessage(ctx, client, challengeMsg); err != nil {
		return fmt.Errorf("failed to send CHALLENGE: %w", err)
	}

	challengePayload := challengeMsg.Payload.(*protocol.ChallengeMessage)
	log.Printf("Sent CHALLENGE to client %x (session: %x)",
		client.clientID[:8],
		challengePayload.SessionID[:8])

	// Store session ID
	client.sessionID = challengePayload.SessionID

	// Step 3: Receive RESPONSE message from client
	responseMsg, err := rh.receiveMessage(ctx, client, protocol.MsgTypeResponse)
	if err != nil {
		return fmt.Errorf("failed to receive RESPONSE: %w", err)
	}

	responsePayload, ok := responseMsg.Payload.(*protocol.ResponseMessage)
	if !ok {
		return fmt.Errorf("invalid RESPONSE payload type")
	}

	log.Printf("Received RESPONSE from client %x", client.clientID[:8])

	// Verify RESPONSE
	if err := handshakeState.VerifyResponseMessage(responsePayload); err != nil {
		return fmt.Errorf("failed to verify RESPONSE: %w", err)
	}

	// Derive session keys first
	if err := handshakeState.DeriveSessionKeys(); err != nil {
		return fmt.Errorf("failed to derive session keys: %w", err)
	}

	// Step 4: Create and send ESTABLISHED message
	establishedMsg := protocol.NewEstablishedMessage(
		client.sessionID,
		0,      // Server capabilities (future use)
		30,     // Heartbeat interval (seconds)
		1500,   // MTU
		3600,   // Key rotation interval (seconds)
	)

	if err := rh.sendMessage(ctx, client, establishedMsg); err != nil {
		return fmt.Errorf("failed to send ESTABLISHED: %w", err)
	}

	log.Printf("Sent ESTABLISHED to client %x", client.clientID[:8])

	// Store session keys in client connection (copy to fixed-size arrays)
	if len(handshakeState.TXKey) != 32 || len(handshakeState.RXKey) != 32 {
		return fmt.Errorf("invalid session key length")
	}

	client.sessionKeys = &SessionKeys{}
	copy(client.sessionKeys.TXKey[:], handshakeState.TXKey)  // Relay TX = Client RX
	copy(client.sessionKeys.RXKey[:], handshakeState.RXKey)  // Relay RX = Client TX

	// Create persistent frame encryptors for this client
	// This ensures nonce consistency across all encrypted/decrypted frames
	txEncryptor, err := crypto.NewFrameEncryptor(client.sessionKeys.TXKey)
	if err != nil {
		return fmt.Errorf("failed to create TX encryptor: %w", err)
	}
	client.txEncryptor = txEncryptor

	rxEncryptor, err := crypto.NewFrameEncryptor(client.sessionKeys.RXKey)
	if err != nil {
		return fmt.Errorf("failed to create RX encryptor: %w", err)
	}
	client.rxEncryptor = rxEncryptor

	log.Printf("Handshake complete with client %x (session: %x)",
		client.clientID[:8],
		client.sessionID[:8])

	return nil
}

// receiveMessage receives a message of the expected type with timeout
func (rh *RelayHandshakeHandler) receiveMessage(
	ctx context.Context,
	client *ClientConnection,
	expectedType byte,
) (*protocol.Message, error) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Wait for message
	select {
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("timeout waiting for message type %d", expectedType)
	case msg, ok := <-client.receiveChan:
		if !ok {
			return nil, fmt.Errorf("connection closed while waiting for message")
		}

		// Verify message type
		if msg.Header.Type != expectedType {
			return nil, fmt.Errorf("expected message type %d, got %d", expectedType, msg.Header.Type)
		}

		return msg, nil
	}
}

// sendMessage sends a message to the client with timeout
func (rh *RelayHandshakeHandler) sendMessage(
	ctx context.Context,
	client *ClientConnection,
	msg *protocol.Message,
) error {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Send message
	select {
	case <-timeoutCtx.Done():
		return fmt.Errorf("timeout sending message type %d", msg.Header.Type)
	case client.sendChan <- msg:
		return nil
	}
}

// HandleKeyRotation handles a key rotation request from a client
// NOTE: Key rotation not yet fully implemented in protocol
func (rh *RelayHandshakeHandler) HandleKeyRotation(
	ctx context.Context,
	client *ClientConnection,
) error {
	log.Printf("Key rotation requested by client %x (not yet implemented)", client.clientID[:8])
	return fmt.Errorf("key rotation not yet implemented")
}
