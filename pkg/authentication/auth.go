package authentication

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/cloudflare/circl/sign/dilithium/mode5"
)

// Challenge represents an authentication challenge
type Challenge struct {
	Challenge  string    `json:"challenge"`
	Timestamp  int64     `json:"timestamp"`
	ExpiresAt  int64     `json:"expires_at"`
	Used       bool      `json:"-"`
}

// AuthRequest represents an authentication request
type AuthRequest struct {
	PeerID    string `json:"peer_id"`
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
	PublicKey string `json:"public_key"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	SessionToken    string           `json:"session_token"`
	ExpiresAt       int64            `json:"expires_at"`
	BackboneNodes   []BackboneNode   `json:"backbone_nodes"`
}

// BackboneNode represents a regional backbone discovery node
type BackboneNode struct {
	Region string `json:"region"`
	IP     string `json:"ip"`
	Port   int    `json:"port"`
}

// AuthServer handles authentication for discovery nodes
type AuthServer struct {
	challenges map[string]*Challenge // challenge -> Challenge
	sessions   map[string]*Session   // sessionToken -> Session
}

// Session represents an authenticated session
type Session struct {
	PeerID     string
	Token      string
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

// NewAuthServer creates a new authentication server
func NewAuthServer() *AuthServer {
	return &AuthServer{
		challenges: make(map[string]*Challenge),
		sessions:   make(map[string]*Session),
	}
}

// GenerateChallenge creates a new authentication challenge
func (as *AuthServer) GenerateChallenge() (*Challenge, error) {
	// Generate 32 random bytes
	challengeBytes := make([]byte, 32)
	_, err := rand.Read(challengeBytes)
	if err != nil {
		return nil, err
	}

	challengeStr := hex.EncodeToString(challengeBytes)
	now := time.Now()
	expiresAt := now.Add(30 * time.Second) // 30 second expiry

	challenge := &Challenge{
		Challenge: challengeStr,
		Timestamp: now.Unix(),
		ExpiresAt: expiresAt.Unix(),
		Used:      false,
	}

	as.challenges[challengeStr] = challenge
	return challenge, nil
}

// VerifyAuthentication verifies an authentication request
func (as *AuthServer) VerifyAuthentication(req *AuthRequest) (*AuthResponse, error) {
	// Step 1: Check if challenge exists and is valid
	challenge, exists := as.challenges[req.Challenge]
	if !exists {
		return nil, errors.New("invalid challenge")
	}

	if challenge.Used {
		return nil, errors.New("challenge already used")
	}

	if time.Now().Unix() > challenge.ExpiresAt {
		return nil, errors.New("challenge expired")
	}

	// Step 2: Decode public key
	publicKeyBytes, err := hex.DecodeString(req.PublicKey)
	if err != nil {
		return nil, errors.New("invalid public key format")
	}

	// Step 3: Verify it's a valid ML-DSA-87 (Dilithium mode5) public key
	if len(publicKeyBytes) != mode5.PublicKeySize {
		return nil, errors.New("invalid public key size")
	}

	// Unpack public key
	var pkArray [mode5.PublicKeySize]byte
	copy(pkArray[:], publicKeyBytes)

	var publicKey mode5.PublicKey
	publicKey.Unpack(&pkArray)

	// Step 4: Decode signature
	signatureBytes, err := hex.DecodeString(req.Signature)
	if err != nil {
		return nil, errors.New("invalid signature format")
	}

	if len(signatureBytes) != mode5.SignatureSize {
		return nil, errors.New("invalid signature size")
	}

	// Step 5: Verify signature
	challengeBytes, _ := hex.DecodeString(req.Challenge)
	if !mode5.Verify(&publicKey, challengeBytes, signatureBytes) {
		return nil, errors.New("signature verification failed")
	}

	// Step 6: Verify peer ID matches public key hash
	expectedPeerID := ComputePeerID(publicKeyBytes)
	if req.PeerID != expectedPeerID {
		return nil, errors.New("peer ID does not match public key")
	}

	// Step 7: Mark challenge as used
	challenge.Used = true

	// Step 8: Generate session token
	sessionToken, err := generateSessionToken()
	if err != nil {
		return nil, err
	}

	// Step 9: Create session
	session := &Session{
		PeerID:    req.PeerID,
		Token:     sessionToken,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hour session
	}

	as.sessions[sessionToken] = session

	// Step 10: Return authentication response
	response := &AuthResponse{
		SessionToken: sessionToken,
		ExpiresAt:    session.ExpiresAt.Unix(),
		BackboneNodes: []BackboneNode{
			{Region: "north_america", IP: "209.151.148.121", Port: 8443},
			{Region: "europe", IP: "83.136.252.52", Port: 8443},
			{Region: "asia_pacific", IP: "213.163.206.44", Port: 8443},
			{Region: "australia", IP: "95.111.223.37", Port: 8443},
		},
	}

	return response, nil
}

// ValidateSession checks if a session token is valid
func (as *AuthServer) ValidateSession(token string) (*Session, error) {
	session, exists := as.sessions[token]
	if !exists {
		return nil, errors.New("invalid session token")
	}

	if time.Now().After(session.ExpiresAt) {
		delete(as.sessions, token)
		return nil, errors.New("session expired")
	}

	return session, nil
}

// CleanupExpired removes expired challenges and sessions
func (as *AuthServer) CleanupExpired() {
	now := time.Now().Unix()

	// Cleanup expired challenges
	for challengeStr, challenge := range as.challenges {
		if now > challenge.ExpiresAt {
			delete(as.challenges, challengeStr)
		}
	}

	// Cleanup expired sessions
	for token, session := range as.sessions {
		if time.Now().After(session.ExpiresAt) {
			delete(as.sessions, token)
		}
	}
}

// ComputePeerID computes the peer ID from a public key
// Peer ID is the first 20 bytes of SHA-256 hash of the public key
func ComputePeerID(publicKey []byte) string {
	// For now, we'll use a simple truncated hash
	// In production, use crypto/sha256
	// hash := sha256.Sum256(publicKey)
	// return hex.EncodeToString(hash[:20])

	// Placeholder: just take first 40 hex chars (20 bytes)
	if len(publicKey) >= 20 {
		return hex.EncodeToString(publicKey[:20])
	}
	return hex.EncodeToString(publicKey)
}

// generateSessionToken generates a random session token
func generateSessionToken() (string, error) {
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(tokenBytes), nil
}
