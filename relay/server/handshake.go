package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/shadowmesh/shadowmesh/shared/crypto"
	"github.com/shadowmesh/shadowmesh/shared/protocol"
)

// RelayHandshakeHandler handles the relay side of the handshake protocol
type RelayHandshakeHandler struct {
	relayID        [32]byte
	sigKeys        *crypto.HybridSigningKey
	tlsCertManager *TLSCertificateManager
}

// NewRelayHandshakeHandler creates a new relay handshake handler
func NewRelayHandshakeHandler(relayID [32]byte, sigKeys *crypto.HybridSigningKey, tlsCertManager *TLSCertificateManager) *RelayHandshakeHandler {
	return &RelayHandshakeHandler{
		relayID:        relayID,
		sigKeys:        sigKeys,
		tlsCertManager: tlsCertManager,
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
	// Extract client's public IP and port from WebSocket connection
	peerIP, peerPort, err := extractClientAddress(client)
	if err != nil {
		// Log warning but continue with zero values
		// Client may still work in relay-only mode
		log.Printf("Warning: Failed to extract client address: %v", err)
		peerIP = [16]byte{}
		peerPort = 0
	}

	// Set direct P2P support based on whether we got a valid address
	peerSupportsDirectP2P := peerPort != 0

	log.Printf("Client %x public address: IP=%v, Port=%d, SupportsDirectP2P=%v",
		client.clientID[:8],
		formatIPFromArray(peerIP),
		peerPort,
		peerSupportsDirectP2P)

	// Get TLS certificate and signature for Epic 2 Direct P2P
	var peerTLSCert []byte
	var peerTLSCertSig []byte

	if rh.tlsCertManager != nil {
		// Get relay's TLS certificate (DER-encoded)
		peerTLSCert = rh.tlsCertManager.GetCertificateDER()

		// Sign the certificate with ML-DSA-87 to bind it to relay's PQC identity
		var err error
		peerTLSCertSig, err = rh.tlsCertManager.SignCertificate()
		if err != nil {
			log.Printf("Warning: Failed to sign TLS certificate: %v", err)
			// Continue with empty cert/sig - client will work in relay-only mode
			peerTLSCert = nil
			peerTLSCertSig = nil
		} else {
			log.Printf("Providing TLS certificate to client %x for Direct P2P (cert: %d bytes, sig: %d bytes)",
				client.clientID[:8],
				len(peerTLSCert),
				len(peerTLSCertSig))
		}
	} else {
		log.Printf("Warning: No TLS certificate manager configured, Direct P2P disabled")
	}

	establishedMsg := protocol.NewEstablishedMessage(
		client.sessionID,
		0,                     // Server capabilities (future use)
		30,                    // Heartbeat interval (seconds)
		1500,                  // MTU
		3600,                  // Key rotation interval (seconds)
		peerIP,                // Peer public IP (detected from connection)
		peerPort,              // Peer public port (detected from connection)
		peerSupportsDirectP2P, // Supports direct P2P (true if valid address)
		peerTLSCert,           // Peer TLS certificate (placeholder)
		peerTLSCertSig,        // Peer TLS certificate signature (placeholder)
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
	copy(client.sessionKeys.TXKey[:], handshakeState.TXKey) // Relay TX = Client RX
	copy(client.sessionKeys.RXKey[:], handshakeState.RXKey) // Relay RX = Client TX

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

// formatIPFromArray converts a [16]byte IP array to a readable string
func formatIPFromArray(ipArray [16]byte) string {
	// Check if it's IPv4 (first 4 bytes contain IP, rest are zero)
	isIPv4 := true
	for i := 4; i < 16; i++ {
		if ipArray[i] != 0 {
			isIPv4 = false
			break
		}
	}

	if isIPv4 {
		// Format as IPv4
		return fmt.Sprintf("%d.%d.%d.%d", ipArray[0], ipArray[1], ipArray[2], ipArray[3])
	} else {
		// Format as IPv6
		ip := net.IP(ipArray[:])
		return ip.String()
	}
}

// extractClientAddress extracts the client's public IP and port from WebSocket connection
// Returns IP as [16]byte array (IPv4 in first 4 bytes, rest zero; or full IPv6)
// and port as uint16
func extractClientAddress(client *ClientConnection) ([16]byte, uint16, error) {
	var ipArray [16]byte
	var port uint16

	// Get remote address from WebSocket connection
	remoteAddr := client.conn.RemoteAddr()
	if remoteAddr == nil {
		return ipArray, 0, fmt.Errorf("no remote address available")
	}

	// Parse address string (format: "ip:port" or "[ipv6]:port")
	addrStr := remoteAddr.String()

	// Split by last colon to separate IP and port
	// For IPv6, address is in brackets: [::1]:port
	// For IPv4: 192.168.1.1:port
	var ipStr string
	var portStr string

	if strings.HasPrefix(addrStr, "[") {
		// IPv6 format: [::1]:12345
		closeBracket := strings.Index(addrStr, "]")
		if closeBracket == -1 {
			return ipArray, 0, fmt.Errorf("invalid IPv6 address format: %s", addrStr)
		}
		ipStr = addrStr[1:closeBracket] // Remove brackets
		if closeBracket+2 < len(addrStr) {
			portStr = addrStr[closeBracket+2:] // Skip "]:"
		}
	} else {
		// IPv4 format: 192.168.1.1:12345
		lastColon := strings.LastIndex(addrStr, ":")
		if lastColon == -1 {
			return ipArray, 0, fmt.Errorf("invalid address format: %s", addrStr)
		}
		ipStr = addrStr[:lastColon]
		portStr = addrStr[lastColon+1:]
	}

	// Parse IP address
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ipArray, 0, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Convert IP to [16]byte array
	if ip4 := ip.To4(); ip4 != nil {
		// IPv4: store in first 4 bytes, rest are zero
		copy(ipArray[:4], ip4)
	} else {
		// IPv6: store full 16 bytes
		copy(ipArray[:], ip.To16())
	}

	// Parse port
	if portStr != "" {
		portNum, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			return ipArray, 0, fmt.Errorf("invalid port: %s", portStr)
		}
		port = uint16(portNum)
	}

	return ipArray, port, nil
}
