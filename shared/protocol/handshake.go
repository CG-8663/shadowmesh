package protocol

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/shadowmesh/shadowmesh/shared/crypto"

	"golang.org/x/crypto/hkdf"
)

const (
	// HKDF salts and info strings
	hkdfSaltMasterSecret = "ShadowMesh-v1-KDF"
	hkdfInfoHandshake    = "handshake-master-secret"
	hkdfSaltTX           = "ShadowMesh-v1-TX"
	hkdfSaltRX           = "ShadowMesh-v1-RX"
	hkdfInfoSession      = "session-keys"
)

// HandshakeState tracks the state of a handshake in progress
type HandshakeState struct {
	// Identity
	IsClient       bool
	LocalID        [32]byte  // Local node ID (Ed25519 pubkey hash)
	RemoteID       [32]byte  // Remote node ID (Ed25519 pubkey hash)
	SessionID      [16]byte  // Session identifier

	// Crypto keys
	LocalKEMKeys   *crypto.HybridKeyPair
	LocalECDHKeys  *crypto.HybridKeyPair
	LocalSigKeys   *crypto.HybridSigningKey

	RemoteKEMPubKey  *crypto.HybridPublicKey
	RemoteECDHPubKey *crypto.HybridPublicKey

	// Shared secrets
	KEMSharedSecret   []byte
	ECDHSharedSecret  []byte
	MasterSecret      []byte

	// Session keys (derived after handshake)
	TXKey []byte // Transmission key (for sending)
	RXKey []byte // Reception key (for receiving)

	// Handshake metadata
	Nonce         [NonceSize]byte
	Timestamp     time.Time
	Capabilities  uint32
}

// NewClientHandshakeState initializes a new client-side handshake
func NewClientHandshakeState(localID [32]byte, sigKeys *crypto.HybridSigningKey) (*HandshakeState, error) {
	// Generate ephemeral key exchange keys
	kemKeys, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate KEM keys: %w", err)
	}

	ecdhKeys, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ECDH keys: %w", err)
	}

	return &HandshakeState{
		IsClient:     true,
		LocalID:      localID,
		LocalKEMKeys: kemKeys,
		LocalECDHKeys: ecdhKeys,
		LocalSigKeys: sigKeys,
		Timestamp:    time.Now(),
		Capabilities: CapMultiHop | CapIPv6, // Default client capabilities
	}, nil
}

// NewRelayHandshakeState initializes a new relay-side handshake
func NewRelayHandshakeState(localID [32]byte, sigKeys *crypto.HybridSigningKey) (*HandshakeState, error) {
	// Generate ephemeral ECDH keys (KEM will be done during challenge)
	ecdhKeys, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ECDH keys: %w", err)
	}

	// Generate random nonce for proof
	var nonce [NonceSize]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Generate random session ID
	var sessionID [SessionIDSize]byte
	if _, err := rand.Read(sessionID[:]); err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	return &HandshakeState{
		IsClient:     false,
		LocalID:      localID,
		SessionID:    sessionID,
		LocalECDHKeys: ecdhKeys,
		LocalSigKeys: sigKeys,
		Nonce:        nonce,
		Timestamp:    time.Now(),
		Capabilities: CapMultiHop | CapIPv6 | CapObfuscation, // Relay capabilities
	}, nil
}

// CreateHelloMessage creates a HELLO message from the handshake state
func (hs *HandshakeState) CreateHelloMessage() (*Message, error) {
	if !hs.IsClient {
		return nil, fmt.Errorf("only clients can send HELLO messages")
	}

	// Get public key bytes (just the individual keys, not combined)
	kemPubKey, err := hs.LocalKEMKeys.KEMPublicKey.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal KEM public key: %w", err)
	}
	ecdhPubBytes := hs.LocalECDHKeys.ECDHPublicKey.Bytes()

	// Prepare fixed-size arrays
	var kemPubKeyArray [KEMPublicKeySize]byte
	var ecdhPubKeyArray [ECDHPublicKeySize]byte

	// Copy KEM public key (1568 bytes)
	copy(kemPubKeyArray[:], kemPubKey)
	// Copy ECDH public key (32 bytes)
	copy(ecdhPubKeyArray[:], ecdhPubBytes)

	// Create signature data: ClientID || KEM_PK || ECDH_PK || Timestamp
	timestamp := hs.Timestamp.UnixNano()
	sigData := make([]byte, 0, ClientIDSize+KEMPublicKeySize+ECDHPublicKeySize+8)
	sigData = append(sigData, hs.LocalID[:]...)
	sigData = append(sigData, kemPubKeyArray[:]...)
	sigData = append(sigData, ecdhPubKeyArray[:]...)
	sigData = append(sigData, uint64ToBytes(uint64(timestamp))...)

	// Sign with hybrid signature
	signature, err := crypto.Sign(hs.LocalSigKeys, sigData)
	if err != nil {
		return nil, fmt.Errorf("failed to sign HELLO: %w", err)
	}

	// Parse signature (format: PQ signature || classical signature)
	var sig [SignatureSize]byte
	var classicalSig [Ed25519SignatureSize]byte
	copy(sig[:], signature[:SignatureSize])
	copy(classicalSig[:], signature[SignatureSize:SignatureSize+Ed25519SignatureSize])

	return NewHelloMessage(hs.LocalID, kemPubKeyArray, ecdhPubKeyArray, sig, classicalSig, timestamp), nil
}

// ProcessHelloMessage processes a received HELLO message (relay side)
func (hs *HandshakeState) ProcessHelloMessage(msg *HelloMessage) error {
	if hs.IsClient {
		return fmt.Errorf("clients cannot process HELLO messages")
	}

	// Parse KEM public key
	kemPub, err := crypto.ParseKEMPublicKey(msg.KEMPublicKey[:])
	if err != nil {
		return fmt.Errorf("failed to parse KEM public key: %w", err)
	}

	// Parse ECDH public key
	ecdhPub, err := crypto.ParseECDHPublicKey(msg.ECDHPublicKey[:])
	if err != nil {
		return fmt.Errorf("failed to parse ECDH public key: %w", err)
	}

	// Create hybrid public key from separate components
	hs.RemoteKEMPubKey = crypto.NewHybridPublicKey(kemPub, ecdhPub)
	hs.RemoteECDHPubKey = hs.RemoteKEMPubKey // Same hybrid key for both (contains both KEM and ECDH)

	copy(hs.RemoteID[:], msg.ClientID[:])

	// Verify timestamp (prevent replay attacks)
	msgTime := time.Unix(0, msg.Timestamp)
	if time.Since(msgTime) > HandshakeTimeout {
		return fmt.Errorf("HELLO message expired")
	}

	// TODO: Verify signature against known client public key
	// For now, we accept any valid signature
	// In production: Implement client key registry/verification

	return nil
}

// CreateChallengeMessage creates a CHALLENGE message (relay side)
func (hs *HandshakeState) CreateChallengeMessage() (*Message, error) {
	if hs.IsClient {
		return nil, fmt.Errorf("only relays can send CHALLENGE messages")
	}

	// Encapsulate shared secret using client's KEM public key (KEM only, not hybrid)
	kemSharedSecret, kemCiphertext, err := crypto.EncapsulateKEM(hs.RemoteKEMPubKey.KEMPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encapsulate KEM secret: %w", err)
	}
	hs.KEMSharedSecret = kemSharedSecret

	// Perform ECDH with client's ECDH public key
	ecdhSharedSecret, err := crypto.PerformECDH(hs.LocalECDHKeys.ECDHPrivateKey, hs.RemoteKEMPubKey.ECDHPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to perform ECDH: %w", err)
	}
	hs.ECDHSharedSecret = ecdhSharedSecret

	// Get our ECDH public key (just the 32-byte ECDH part)
	ecdhPubBytes := hs.LocalECDHKeys.ECDHPublicKey.Bytes()

	// Prepare message fields
	var kemCT [KEMCiphertextSize]byte
	var ecdhPubKey [ECDHPublicKeySize]byte

	copy(kemCT[:], kemCiphertext)
	copy(ecdhPubKey[:], ecdhPubBytes)

	// Create signature
	timestamp := time.Now().UnixNano()
	sigData := make([]byte, 0, RelayIDSize+SessionIDSize+KEMCiphertextSize+ECDHPublicKeySize+NonceSize+8)
	sigData = append(sigData, hs.LocalID[:]...)
	sigData = append(sigData, hs.SessionID[:]...)
	sigData = append(sigData, kemCT[:]...)
	sigData = append(sigData, ecdhPubKey[:]...)
	sigData = append(sigData, hs.Nonce[:]...)
	sigData = append(sigData, uint64ToBytes(uint64(timestamp))...)

	signature, err := crypto.Sign(hs.LocalSigKeys, sigData)
	if err != nil {
		return nil, fmt.Errorf("failed to sign CHALLENGE: %w", err)
	}

	var sig [SignatureSize]byte
	var classicalSig [Ed25519SignatureSize]byte
	copy(sig[:], signature[:SignatureSize])
	copy(classicalSig[:], signature[SignatureSize:SignatureSize+Ed25519SignatureSize])

	// Derive master secret from both KEM and ECDH secrets using HKDF
	combinedSecret := append(hs.KEMSharedSecret, hs.ECDHSharedSecret...)
	masterSecret, err := deriveKey(combinedSecret, nil, []byte("ShadowMesh-v1-MasterSecret"), 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive master secret: %w", err)
	}
	hs.MasterSecret = masterSecret

	return NewChallengeMessage(hs.LocalID, hs.SessionID, kemCT, ecdhPubKey,
		hs.Nonce, sig, classicalSig, timestamp), nil
}

// ProcessChallengeMessage processes a received CHALLENGE message (client side)
func (hs *HandshakeState) ProcessChallengeMessage(msg *ChallengeMessage) error {
	if !hs.IsClient {
		return fmt.Errorf("only clients can process CHALLENGE messages")
	}

	// Store relay info
	copy(hs.RemoteID[:], msg.RelayID[:])
	copy(hs.SessionID[:], msg.SessionID[:])
	copy(hs.Nonce[:], msg.Nonce[:])

	// Parse relay's ECDH public key (only ECDH, no KEM in CHALLENGE)
	ecdhPub, err := crypto.ParseECDHPublicKey(msg.ECDHPublicKey[:])
	if err != nil {
		return fmt.Errorf("failed to parse ECDH public key: %w", err)
	}
	// Create a hybrid public key with only ECDH (KEM was already encapsulated in ciphertext)
	// We'll use a nil KEM public key since we don't need it for ECDH
	hs.RemoteECDHPubKey = crypto.NewHybridPublicKey(nil, ecdhPub)

	// Verify timestamp
	msgTime := time.Unix(0, msg.Timestamp)
	if time.Since(msgTime) > HandshakeTimeout {
		return fmt.Errorf("CHALLENGE message expired")
	}

	// Decapsulate KEM shared secret (KEM only, not hybrid)
	kemSharedSecret, err := crypto.DecapsulateKEM(hs.LocalKEMKeys.KEMPrivateKey, msg.KEMCiphertext[:])
	if err != nil {
		return fmt.Errorf("failed to decapsulate KEM secret: %w", err)
	}
	hs.KEMSharedSecret = kemSharedSecret

	// Perform ECDH with relay's ECDH public key
	ecdhSharedSecret, err := crypto.PerformECDH(hs.LocalECDHKeys.ECDHPrivateKey, hs.RemoteECDHPubKey.ECDHPublicKey)
	if err != nil {
		return fmt.Errorf("failed to perform ECDH: %w", err)
	}
	hs.ECDHSharedSecret = ecdhSharedSecret

	// Derive master secret from both KEM and ECDH secrets using HKDF
	combinedSecret := append(hs.KEMSharedSecret, hs.ECDHSharedSecret...)
	masterSecret, err := deriveKey(combinedSecret, nil, []byte("ShadowMesh-v1-MasterSecret"), 32)
	if err != nil {
		return fmt.Errorf("failed to derive master secret: %w", err)
	}
	hs.MasterSecret = masterSecret

	return nil
}

// CreateResponseMessage creates a RESPONSE message proving possession of shared secret
func (hs *HandshakeState) CreateResponseMessage() (*Message, error) {
	if !hs.IsClient {
		return nil, fmt.Errorf("only clients can send RESPONSE messages")
	}

	// Compute proof: HMAC-SHA256(master_secret, nonce)
	h := hmac.New(sha256.New, hs.MasterSecret)
	h.Write(hs.Nonce[:])
	proofBytes := h.Sum(nil)

	var proof [ProofSize]byte
	copy(proof[:], proofBytes)

	return NewResponseMessage(hs.SessionID, proof, hs.Capabilities), nil
}

// VerifyResponseMessage verifies a RESPONSE message (relay side)
func (hs *HandshakeState) VerifyResponseMessage(msg *ResponseMessage) error {
	if hs.IsClient {
		return fmt.Errorf("only relays can verify RESPONSE messages")
	}

	// Verify session ID
	if msg.SessionID != hs.SessionID {
		return fmt.Errorf("session ID mismatch")
	}

	// Compute expected proof
	h := hmac.New(sha256.New, hs.MasterSecret)
	h.Write(hs.Nonce[:])
	expectedProof := h.Sum(nil)

	// Constant-time comparison
	if !hmac.Equal(expectedProof, msg.Proof[:]) {
		return fmt.Errorf("proof verification failed")
	}

	// Store client capabilities
	hs.Capabilities = msg.Capabilities

	return nil
}

// DeriveSessionKeys derives TX and RX keys from the master secret
func (hs *HandshakeState) DeriveSessionKeys() error {
	if len(hs.MasterSecret) == 0 {
		return fmt.Errorf("master secret not established")
	}

	// Derive TX key
	txInfo := append([]byte(hkdfInfoSession), hs.SessionID[:]...)
	txInfo = append(txInfo, hs.LocalID[:]...)
	txInfo = append(txInfo, hs.RemoteID[:]...)

	txKey, err := deriveKey(hs.MasterSecret, []byte(hkdfSaltTX), txInfo, 32)
	if err != nil {
		return fmt.Errorf("failed to derive TX key: %w", err)
	}
	hs.TXKey = txKey

	// Derive RX key
	rxInfo := append([]byte(hkdfInfoSession), hs.SessionID[:]...)
	rxInfo = append(rxInfo, hs.RemoteID[:]...)
	rxInfo = append(rxInfo, hs.LocalID[:]...)

	rxKey, err := deriveKey(hs.MasterSecret, []byte(hkdfSaltRX), rxInfo, 32)
	if err != nil {
		return fmt.Errorf("failed to derive RX key: %w", err)
	}
	hs.RXKey = rxKey

	return nil
}

// deriveKey uses HKDF to derive a key from input key material
func deriveKey(ikm, salt, info []byte, length int) ([]byte, error) {
	kdf := hkdf.New(sha256.New, ikm, salt, info)
	key := make([]byte, length)
	if _, err := kdf.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// uint64ToBytes converts uint64 to big-endian byte slice
func uint64ToBytes(n uint64) []byte {
	b := make([]byte, 8)
	b[0] = byte(n >> 56)
	b[1] = byte(n >> 48)
	b[2] = byte(n >> 40)
	b[3] = byte(n >> 32)
	b[4] = byte(n >> 24)
	b[5] = byte(n >> 16)
	b[6] = byte(n >> 8)
	b[7] = byte(n)
	return b
}
