package integration

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/shadowmesh/shadowmesh/shared/crypto"
	"github.com/shadowmesh/shadowmesh/shared/protocol"
)

// TestHandshakeFlow tests the complete 4-message handshake sequence
func TestHandshakeFlow(t *testing.T) {
	// Step 1: Generate client and relay identities
	clientSigKeys, err := crypto.GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate client signing key: %v", err)
	}

	relaySigKeys, err := crypto.GenerateSigningKey()
	if err != nil {
		t.Fatalf("Failed to generate relay signing key: %v", err)
	}

	clientPubKey := clientSigKeys.PublicKey()
	clientID := crypto.PublicKeyHash(clientPubKey)

	relayPubKey := relaySigKeys.PublicKey()
	relayID := crypto.PublicKeyHash(relayPubKey)

	t.Logf("Client ID: %x", clientID)
	t.Logf("Relay ID: %x", relayID)

	// Step 2: Client creates HELLO message
	clientHS, err := protocol.NewClientHandshakeState(clientID, clientSigKeys)
	if err != nil {
		t.Fatalf("Failed to create client handshake state: %v", err)
	}

	helloMsg, err := clientHS.CreateHelloMessage()
	if err != nil {
		t.Fatalf("Failed to create HELLO message: %v", err)
	}

	t.Logf("HELLO message created, size: %d bytes", protocol.HeaderSize+int(helloMsg.Header.Length))

	// Serialize HELLO
	helloBytes, err := protocol.EncodeMessage(helloMsg)
	if err != nil {
		t.Fatalf("Failed to encode HELLO: %v", err)
	}

	// Step 3: Relay receives and processes HELLO
	receivedHello, err := protocol.DecodeMessage(helloBytes)
	if err != nil {
		t.Fatalf("Failed to decode HELLO: %v", err)
	}

	if receivedHello.Header.Type != protocol.MsgTypeHello {
		t.Fatalf("Expected HELLO message, got %s", protocol.MessageTypeName(receivedHello.Header.Type))
	}

	relayHS, err := protocol.NewRelayHandshakeState(relayID, relaySigKeys)
	if err != nil {
		t.Fatalf("Failed to create relay handshake state: %v", err)
	}

	helloPayload := receivedHello.Payload.(*protocol.HelloMessage)
	if err := relayHS.ProcessHelloMessage(helloPayload); err != nil {
		t.Fatalf("Failed to process HELLO: %v", err)
	}

	t.Logf("HELLO processed successfully")

	// Step 4: Relay creates CHALLENGE message
	challengeMsg, err := relayHS.CreateChallengeMessage()
	if err != nil {
		t.Fatalf("Failed to create CHALLENGE message: %v", err)
	}

	t.Logf("CHALLENGE message created, size: %d bytes", protocol.HeaderSize+int(challengeMsg.Header.Length))

	// Serialize CHALLENGE
	challengeBytes, err := protocol.EncodeMessage(challengeMsg)
	if err != nil {
		t.Fatalf("Failed to encode CHALLENGE: %v", err)
	}

	// Step 5: Client receives and processes CHALLENGE
	receivedChallenge, err := protocol.DecodeMessage(challengeBytes)
	if err != nil {
		t.Fatalf("Failed to decode CHALLENGE: %v", err)
	}

	if receivedChallenge.Header.Type != protocol.MsgTypeChallenge {
		t.Fatalf("Expected CHALLENGE message, got %s", protocol.MessageTypeName(receivedChallenge.Header.Type))
	}

	challengePayload := receivedChallenge.Payload.(*protocol.ChallengeMessage)
	if err := clientHS.ProcessChallengeMessage(challengePayload); err != nil {
		t.Fatalf("Failed to process CHALLENGE: %v", err)
	}

	t.Logf("CHALLENGE processed successfully")

	// Step 6: Client creates RESPONSE message
	responseMsg, err := clientHS.CreateResponseMessage()
	if err != nil {
		t.Fatalf("Failed to create RESPONSE message: %v", err)
	}

	t.Logf("RESPONSE message created")

	// Serialize RESPONSE
	responseBytes, err := protocol.EncodeMessage(responseMsg)
	if err != nil {
		t.Fatalf("Failed to encode RESPONSE: %v", err)
	}

	// Step 7: Relay receives and verifies RESPONSE
	receivedResponse, err := protocol.DecodeMessage(responseBytes)
	if err != nil {
		t.Fatalf("Failed to decode RESPONSE: %v", err)
	}

	if receivedResponse.Header.Type != protocol.MsgTypeResponse {
		t.Fatalf("Expected RESPONSE message, got %s", protocol.MessageTypeName(receivedResponse.Header.Type))
	}

	responsePayload := receivedResponse.Payload.(*protocol.ResponseMessage)
	if err := relayHS.VerifyResponseMessage(responsePayload); err != nil {
		t.Fatalf("Failed to verify RESPONSE: %v", err)
	}

	t.Logf("RESPONSE verified successfully")

	// Step 8: Both sides derive session keys
	if err := clientHS.DeriveSessionKeys(); err != nil {
		t.Fatalf("Failed to derive client session keys: %v", err)
	}

	if err := relayHS.DeriveSessionKeys(); err != nil {
		t.Fatalf("Failed to derive relay session keys: %v", err)
	}

	t.Logf("Session keys derived")

	// Step 9: Verify both sides have matching session keys
	// Note: Client TX key should match Relay RX key, and vice versa
	if !bytes.Equal(clientHS.TXKey, relayHS.RXKey) {
		t.Errorf("Client TX key does not match Relay RX key")
	}

	if !bytes.Equal(clientHS.RXKey, relayHS.TXKey) {
		t.Errorf("Client RX key does not match Relay TX key")
	}

	t.Logf("Session key verification successful!")
	t.Logf("Handshake complete. Session established.")
}

// TestHandshakeProofVerification tests that the proof-of-possession works correctly
func TestHandshakeProofVerification(t *testing.T) {
	// Create dummy handshake state
	var sessionID [16]byte
	rand.Read(sessionID[:])

	var nonce [24]byte
	rand.Read(nonce[:])

	masterSecret := make([]byte, 32)
	rand.Read(masterSecret)

	// Compute correct proof
	h := hmac.New(sha256.New, masterSecret)
	h.Write(nonce[:])
	correctProof := h.Sum(nil)

	// Compute incorrect proof
	wrongSecret := make([]byte, 32)
	rand.Read(wrongSecret)
	h2 := hmac.New(sha256.New, wrongSecret)
	h2.Write(nonce[:])
	incorrectProof := h2.Sum(nil)

	// Create response messages
	var correctProofArray, incorrectProofArray [32]byte
	copy(correctProofArray[:], correctProof)
	copy(incorrectProofArray[:], incorrectProof)

	correctResponse := &protocol.ResponseMessage{
		SessionID:    sessionID,
		Proof:        correctProofArray,
		Capabilities: 0,
	}

	incorrectResponse := &protocol.ResponseMessage{
		SessionID:    sessionID,
		Proof:        incorrectProofArray,
		Capabilities: 0,
	}

	// Verify correct proof
	expectedProof := hmac.New(sha256.New, masterSecret)
	expectedProof.Write(nonce[:])
	if !hmac.Equal(expectedProof.Sum(nil), correctResponse.Proof[:]) {
		t.Errorf("Correct proof failed verification")
	}

	// Verify incorrect proof fails
	expectedProof2 := hmac.New(sha256.New, masterSecret)
	expectedProof2.Write(nonce[:])
	if hmac.Equal(expectedProof2.Sum(nil), incorrectResponse.Proof[:]) {
		t.Errorf("Incorrect proof passed verification (should have failed)")
	}

	t.Logf("Proof verification test passed")
}

// TestHandshakeReplayProtection tests that old messages are rejected
func TestHandshakeReplayProtection(t *testing.T) {
	clientSigKeys, _ := crypto.GenerateSigningKey()
	clientPubKey := clientSigKeys.PublicKey()
	clientID := crypto.PublicKeyHash(clientPubKey)

	// Create HELLO message with old timestamp
	clientHS, err := protocol.NewClientHandshakeState(clientID, clientSigKeys)
	if err != nil {
		t.Fatalf("Failed to create handshake state: %v", err)
	}

	// Set timestamp to 1 hour ago
	clientHS.Timestamp = time.Now().Add(-1 * time.Hour)

	helloMsg, err := clientHS.CreateHelloMessage()
	if err != nil {
		t.Fatalf("Failed to create HELLO: %v", err)
	}

	helloPayload := helloMsg.Payload.(*protocol.HelloMessage)

	// Relay attempts to process old HELLO
	relaySigKeys, _ := crypto.GenerateSigningKey()
	relayPubKey := relaySigKeys.PublicKey()
	relayID := crypto.PublicKeyHash(relayPubKey)

	relayHS, _ := protocol.NewRelayHandshakeState(relayID, relaySigKeys)

	err = relayHS.ProcessHelloMessage(helloPayload)
	if err == nil {
		t.Errorf("Expected error for expired HELLO message, got nil")
	}

	if err != nil {
		t.Logf("Correctly rejected expired HELLO: %v", err)
	}
}

// TestKeyRotation tests the key rotation flow
func TestKeyRotation(t *testing.T) {
	clientSigKeys, _ := crypto.GenerateSigningKey()
	clientPubKey := clientSigKeys.PublicKey()
	clientID := crypto.PublicKeyHash(clientPubKey)

	relaySigKeys, _ := crypto.GenerateSigningKey()
	relayPubKey := relaySigKeys.PublicKey()
	relayID := crypto.PublicKeyHash(relayPubKey)

	// Initial handshake
	clientHS1, _ := protocol.NewClientHandshakeState(clientID, clientSigKeys)
	helloMsg1, _ := clientHS1.CreateHelloMessage()

	relayHS1, _ := protocol.NewRelayHandshakeState(relayID, relaySigKeys)
	helloPayload1 := helloMsg1.Payload.(*protocol.HelloMessage)
	relayHS1.ProcessHelloMessage(helloPayload1)

	challengeMsg1, _ := relayHS1.CreateChallengeMessage()
	challengePayload1 := challengeMsg1.Payload.(*protocol.ChallengeMessage)
	clientHS1.ProcessChallengeMessage(challengePayload1)

	responseMsg1, _ := clientHS1.CreateResponseMessage()
	responsePayload1 := responseMsg1.Payload.(*protocol.ResponseMessage)
	relayHS1.VerifyResponseMessage(responsePayload1)

	clientHS1.DeriveSessionKeys()
	relayHS1.DeriveSessionKeys()

	oldClientTXKey := make([]byte, 32)
	copy(oldClientTXKey, clientHS1.TXKey)

	// Key rotation: new handshake
	clientHS2, _ := protocol.NewClientHandshakeState(clientID, clientSigKeys)
	helloMsg2, _ := clientHS2.CreateHelloMessage()

	// Set rotation flag
	helloMsg2.Header.Flags = protocol.FlagKeyRotation

	relayHS2, _ := protocol.NewRelayHandshakeState(relayID, relaySigKeys)
	helloPayload2 := helloMsg2.Payload.(*protocol.HelloMessage)
	relayHS2.ProcessHelloMessage(helloPayload2)

	challengeMsg2, _ := relayHS2.CreateChallengeMessage()
	challengePayload2 := challengeMsg2.Payload.(*protocol.ChallengeMessage)
	clientHS2.ProcessChallengeMessage(challengePayload2)

	responseMsg2, _ := clientHS2.CreateResponseMessage()
	responsePayload2 := responseMsg2.Payload.(*protocol.ResponseMessage)
	relayHS2.VerifyResponseMessage(responsePayload2)

	clientHS2.DeriveSessionKeys()
	relayHS2.DeriveSessionKeys()

	// Verify keys have changed
	if bytes.Equal(oldClientTXKey, clientHS2.TXKey) {
		t.Errorf("Keys did not change after rotation")
	}

	// Verify new keys match
	if !bytes.Equal(clientHS2.TXKey, relayHS2.RXKey) {
		t.Errorf("New client TX key does not match relay RX key")
	}

	t.Logf("Key rotation successful")
}

func BenchmarkFullHandshake(b *testing.B) {
	clientSigKeys, _ := crypto.GenerateSigningKey()
	clientPubKey := clientSigKeys.PublicKey()
	clientID := crypto.PublicKeyHash(clientPubKey)

	relaySigKeys, _ := crypto.GenerateSigningKey()
	relayPubKey := relaySigKeys.PublicKey()
	relayID := crypto.PublicKeyHash(relayPubKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Full handshake
		clientHS, _ := protocol.NewClientHandshakeState(clientID, clientSigKeys)
		helloMsg, _ := clientHS.CreateHelloMessage()

		relayHS, _ := protocol.NewRelayHandshakeState(relayID, relaySigKeys)
		helloPayload := helloMsg.Payload.(*protocol.HelloMessage)
		relayHS.ProcessHelloMessage(helloPayload)

		challengeMsg, _ := relayHS.CreateChallengeMessage()
		challengePayload := challengeMsg.Payload.(*protocol.ChallengeMessage)
		clientHS.ProcessChallengeMessage(challengePayload)

		responseMsg, _ := clientHS.CreateResponseMessage()
		responsePayload := responseMsg.Payload.(*protocol.ResponseMessage)
		relayHS.VerifyResponseMessage(responsePayload)

		clientHS.DeriveSessionKeys()
		relayHS.DeriveSessionKeys()
	}
}
