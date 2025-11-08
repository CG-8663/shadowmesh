package client

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AuthClient handles authentication with discovery backbone
type AuthClient struct {
	backboneURL string
	httpClient  *http.Client
	sessionToken string
	expiresAt    time.Time
	keyPair     *KeyPair
}

// Challenge represents an authentication challenge from backbone
type Challenge struct {
	Challenge string `json:"challenge"`
	Timestamp int64  `json:"timestamp"`
	ExpiresAt int64  `json:"expires_at"`
}

// AuthRequest represents authentication request
type AuthRequest struct {
	PeerID    string `json:"peer_id"`
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
	PublicKey string `json:"public_key"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	SessionToken  string         `json:"session_token"`
	ExpiresAt     int64          `json:"expires_at"`
	BackboneNodes []BackboneNode `json:"backbone_nodes"`
}

// BackboneNode represents a regional backbone node
type BackboneNode struct {
	Region string `json:"region"`
	IP     string `json:"ip"`
	Port   int    `json:"port"`
}

// NewAuthClient creates a new authentication client
func NewAuthClient(backboneURL string, keyPair *KeyPair) *AuthClient {
	return &AuthClient{
		backboneURL: backboneURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		keyPair: keyPair,
	}
}

// Authenticate performs full authentication flow with discovery backbone
func (ac *AuthClient) Authenticate() error {
	// Step 1: Get challenge
	challenge, err := ac.getChallenge()
	if err != nil {
		return fmt.Errorf("failed to get challenge: %w", err)
	}

	// Step 2: Sign challenge
	challengeBytes, err := hex.DecodeString(challenge.Challenge)
	if err != nil {
		return fmt.Errorf("failed to decode challenge: %w", err)
	}

	signature, err := ac.keyPair.Sign(challengeBytes)
	if err != nil {
		return fmt.Errorf("failed to sign challenge: %w", err)
	}

	// Step 3: Verify authentication
	authReq := AuthRequest{
		PeerID:    ac.keyPair.PeerID,
		Challenge: challenge.Challenge,
		Signature: hex.EncodeToString(signature),
		PublicKey: ac.keyPair.PublicKeyHex(),
	}

	response, err := ac.verifyAuth(authReq)
	if err != nil {
		return fmt.Errorf("failed to verify authentication: %w", err)
	}

	// Store session token
	ac.sessionToken = response.SessionToken
	ac.expiresAt = time.Unix(response.ExpiresAt, 0)

	return nil
}

// getChallenge requests a challenge from the backbone
func (ac *AuthClient) getChallenge() (*Challenge, error) {
	url := fmt.Sprintf("%s/api/auth/challenge", ac.backboneURL)

	resp, err := ac.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var challenge Challenge
	if err := json.NewDecoder(resp.Body).Decode(&challenge); err != nil {
		return nil, err
	}

	return &challenge, nil
}

// verifyAuth sends signed challenge to backbone for verification
func (ac *AuthClient) verifyAuth(req AuthRequest) (*AuthResponse, error) {
	url := fmt.Sprintf("%s/api/auth/verify", ac.backboneURL)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := ac.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	return &authResp, nil
}

// RegisterPeer registers this peer with the discovery backbone
func (ac *AuthClient) RegisterPeer(ipAddress string, port int, isPublic bool) error {
	if ac.sessionToken == "" {
		return fmt.Errorf("not authenticated, call Authenticate() first")
	}

	url := fmt.Sprintf("%s/api/peers/register", ac.backboneURL)

	peerData := map[string]interface{}{
		"peer_id":    ac.keyPair.PeerID,
		"public_key": ac.keyPair.PublicKeyHex(),
		"ip_address": ipAddress,
		"port":       port,
		"is_public":  isPublic,
	}

	jsonData, err := json.Marshal(peerData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ac.sessionToken))

	resp, err := ac.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to register peer: %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// FindPeers finds closest peers to a target peer ID
func (ac *AuthClient) FindPeers(targetPeerID string, count int) ([]Peer, error) {
	if ac.sessionToken == "" {
		return nil, fmt.Errorf("not authenticated, call Authenticate() first")
	}

	url := fmt.Sprintf("%s/api/peers/lookup?peer_id=%s&count=%d", ac.backboneURL, targetPeerID, count)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ac.sessionToken))

	resp, err := ac.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to find peers: %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		TargetPeerID string `json:"target_peer_id"`
		Count        int    `json:"count"`
		Peers        []Peer `json:"peers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Peers, nil
}

// Peer represents a peer from the DHT
type Peer struct {
	PeerID    string    `json:"peer_id"`
	PublicKey string    `json:"public_key"`
	IPAddress string    `json:"ip_address"`
	Port      int       `json:"port"`
	IsPublic  bool      `json:"is_public"`
	LastSeen  time.Time `json:"last_seen"`
	Verified  bool      `json:"verified"`
}

// GetSessionToken returns the current session token
func (ac *AuthClient) GetSessionToken() string {
	return ac.sessionToken
}

// IsAuthenticated checks if client is authenticated and session is valid
func (ac *AuthClient) IsAuthenticated() bool {
	return ac.sessionToken != "" && time.Now().Before(ac.expiresAt)
}
