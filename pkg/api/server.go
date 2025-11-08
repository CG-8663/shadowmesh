package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shadowmesh/shadowmesh/pkg/authentication"
	"github.com/shadowmesh/shadowmesh/pkg/discovery"
)

// NATCandidate represents a NAT traversal connection candidate
type NATCandidate struct {
	Type string `json:"type"` // "host", "public"
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

// CandidateStore holds NAT traversal candidates for peers
type CandidateStore struct {
	candidates map[string][]NATCandidate // peer_id -> candidates
	mu         sync.RWMutex
	ttl        time.Duration
}

// NewCandidateStore creates a new candidate store
func NewCandidateStore(ttl time.Duration) *CandidateStore {
	return &CandidateStore{
		candidates: make(map[string][]NATCandidate),
		ttl:        ttl,
	}
}

// Set stores candidates for a peer
func (cs *CandidateStore) Set(peerID string, candidates []NATCandidate) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.candidates[peerID] = candidates
}

// Get retrieves candidates for a peer
func (cs *CandidateStore) Get(peerID string) ([]NATCandidate, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	candidates, ok := cs.candidates[peerID]
	return candidates, ok
}

// APIServer handles HTTP API requests for peer discovery
type APIServer struct {
	authServer     *authentication.AuthServer
	kademliaTable  *discovery.KademliaTable
	candidateStore *CandidateStore
	httpServer     *http.Server
	port           int
	mu             sync.RWMutex
}

// NewAPIServer creates a new API server
func NewAPIServer(port int, authServer *authentication.AuthServer, kademliaTable *discovery.KademliaTable) *APIServer {
	server := &APIServer{
		authServer:     authServer,
		kademliaTable:  kademliaTable,
		candidateStore: NewCandidateStore(10 * time.Minute), // 10 minute TTL
		port:           port,
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Authentication endpoints
	mux.HandleFunc("/api/auth/challenge", server.handleGetChallenge)
	mux.HandleFunc("/api/auth/verify", server.handleVerifyAuth)
	mux.HandleFunc("/api/auth/validate", server.handleValidateSession)

	// Peer lookup endpoints
	mux.HandleFunc("/api/peers/lookup", server.requireAuth(server.handlePeerLookup))
	mux.HandleFunc("/api/peers/", server.requireAuth(server.handlePeerEndpoint))
	mux.HandleFunc("/api/peers/register", server.requireAuth(server.handleRegisterPeer))

	// NAT traversal endpoints (Kademlia-based)
	mux.HandleFunc("/api/nat/candidates/publish", server.requireAuth(server.handlePublishCandidates))
	mux.HandleFunc("/api/nat/candidates/", server.requireAuth(server.handleGetCandidates))

	// Health and stats
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/stats", server.handleStats)

	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server
}

// Start starts the HTTP server
func (s *APIServer) Start() error {
	log.Printf("Starting API server on port %d", s.port)
	return s.httpServer.ListenAndServe()
}

// Stop stops the HTTP server gracefully
func (s *APIServer) Stop() error {
	log.Println("Stopping API server")
	return s.httpServer.Close()
}

// requireAuth is middleware that validates session tokens
func (s *APIServer) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		// Extract bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			s.writeError(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		token := parts[1]

		// Validate session
		session, err := s.authServer.ValidateSession(token)
		if err != nil {
			s.writeError(w, http.StatusUnauthorized, err.Error())
			return
		}

		// Add peer ID to request context (for logging)
		log.Printf("Authenticated request from peer: %s", session.PeerID)

		// Call next handler
		next(w, r)
	}
}

// handleGetChallenge generates a new authentication challenge
// GET /api/auth/challenge
func (s *APIServer) handleGetChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	challenge, err := s.authServer.GenerateChallenge()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to generate challenge")
		return
	}

	s.writeJSON(w, http.StatusOK, challenge)
}

// handleVerifyAuth verifies authentication request
// POST /api/auth/verify
// Body: {"peer_id": "...", "challenge": "...", "signature": "...", "public_key": "..."}
func (s *APIServer) handleVerifyAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req authentication.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := s.authServer.VerifyAuthentication(&req)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleValidateSession validates an existing session token
// GET /api/auth/validate?token=<session-token>
func (s *APIServer) handleValidateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		s.writeError(w, http.StatusBadRequest, "missing token parameter")
		return
	}

	session, err := s.authServer.ValidateSession(token)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":      true,
		"peer_id":    session.PeerID,
		"expires_at": session.ExpiresAt.Unix(),
	})
}

// handlePeerLookup finds closest peers to a target peer ID
// GET /api/peers/lookup?peer_id=<peer-id>&count=<count>
func (s *APIServer) handlePeerLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	peerID := r.URL.Query().Get("peer_id")
	if peerID == "" {
		s.writeError(w, http.StatusBadRequest, "missing peer_id parameter")
		return
	}

	count := discovery.K // Default to K
	if countStr := r.URL.Query().Get("count"); countStr != "" {
		parsedCount, err := strconv.Atoi(countStr)
		if err != nil || parsedCount <= 0 {
			s.writeError(w, http.StatusBadRequest, "invalid count parameter")
			return
		}
		count = parsedCount
	}

	peers, err := s.kademliaTable.FindClosest(peerID, count)
	if err != nil {
		s.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"target_peer_id": peerID,
		"count":          len(peers),
		"peers":          peers,
	})
}

// handlePeerEndpoint handles GET/DELETE for specific peer
// GET /api/peers/<peer-id>
// DELETE /api/peers/<peer-id>
func (s *APIServer) handlePeerEndpoint(w http.ResponseWriter, r *http.Request) {
	// Extract peer ID from URL path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		s.writeError(w, http.StatusBadRequest, "invalid URL path")
		return
	}
	peerID := parts[3]

	switch r.Method {
	case http.MethodGet:
		peer, err := s.kademliaTable.GetPeer(peerID)
		if err != nil {
			s.writeError(w, http.StatusNotFound, "peer not found")
			return
		}
		s.writeJSON(w, http.StatusOK, peer)

	case http.MethodDelete:
		removed := s.kademliaTable.RemovePeer(peerID)
		if !removed {
			s.writeError(w, http.StatusNotFound, "peer not found")
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "peer removed",
		})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleRegisterPeer registers a new peer in the DHT
// POST /api/peers/register
// Body: {"peer_id": "...", "public_key": "...", "ip_address": "...", "port": 8443, "is_public": true}
func (s *APIServer) handleRegisterPeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var peer discovery.PeerInfo
	if err := json.NewDecoder(r.Body).Decode(&peer); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Set metadata
	peer.LastSeen = time.Now()
	peer.Verified = true

	// Add to routing table
	if err := s.kademliaTable.AddPeer(&peer); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "peer registered",
		"peer":    peer,
	})
}

// handleHealth returns health check status
// GET /health
func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"uptime":      time.Since(time.Now()).String(), // TODO: track actual uptime
		"total_peers": s.kademliaTable.Size(),
	})
}

// handleStats returns DHT statistics
// GET /stats
func (s *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	stats := s.kademliaTable.GetStats()
	s.writeJSON(w, http.StatusOK, stats)
}

// writeJSON writes JSON response
func (s *APIServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes JSON error response
func (s *APIServer) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]interface{}{
		"error": message,
	})
}

// handlePublishCandidates publishes NAT traversal candidates for a peer
// POST /api/nat/candidates/publish
// Body: {"peer_id": "...", "candidates": [{"type": "host", "ip": "...", "port": 9443}]}
func (s *APIServer) handlePublishCandidates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		PeerID     string         `json:"peer_id"`
		Candidates []NATCandidate `json:"candidates"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.PeerID == "" || len(req.Candidates) == 0 {
		s.writeError(w, http.StatusBadRequest, "peer_id and candidates required")
		return
	}

	// Store candidates in memory (Kademlia-based routing)
	s.candidateStore.Set(req.PeerID, req.Candidates)

	log.Printf("Published %d candidates for peer %s", len(req.Candidates), req.PeerID)

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("published %d candidates", len(req.Candidates)),
	})
}

// handleGetCandidates retrieves NAT traversal candidates for a peer
// GET /api/nat/candidates/<peer-id>
func (s *APIServer) handleGetCandidates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract peer ID from URL path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		s.writeError(w, http.StatusBadRequest, "invalid URL path")
		return
	}
	peerID := parts[4]

	if peerID == "" {
		s.writeError(w, http.StatusBadRequest, "peer_id required")
		return
	}

	// Retrieve candidates from store
	candidates, ok := s.candidateStore.Get(peerID)
	if !ok {
		s.writeError(w, http.StatusNotFound, "no candidates found for peer")
		return
	}

	log.Printf("Retrieved %d candidates for peer %s", len(candidates), peerID)

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"peer_id":    peerID,
		"candidates": candidates,
	})
}
