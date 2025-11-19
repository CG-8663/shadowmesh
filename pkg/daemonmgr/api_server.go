package daemonmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// DaemonAPI provides HTTP API for CLI communication
type DaemonAPI struct {
	manager *DaemonManager
	server  *http.Server
	mu      sync.RWMutex
}

// NewDaemonAPI creates a new daemon API server
func NewDaemonAPI(addr string, manager *DaemonManager) (*DaemonAPI, error) {
	api := &DaemonAPI{
		manager: manager,
		server: &http.Server{
			Addr:         addr,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/connect", api.handleConnect)
	mux.HandleFunc("/disconnect", api.handleDisconnect)
	mux.HandleFunc("/status", api.handleStatus)
	mux.HandleFunc("/health", api.handleHealth)

	api.server.Handler = mux

	return api, nil
}

// Start starts the HTTP API server
func (api *DaemonAPI) Start() error {
	log.Printf("Starting daemon API server on %s", api.server.Addr)
	if err := api.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("API server error: %w", err)
	}
	return nil
}

// Stop gracefully stops the API server
func (api *DaemonAPI) Stop() error {
	log.Println("Stopping daemon API server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return api.server.Shutdown(ctx)
}

// ConnectRequest represents a connect request from CLI
type ConnectRequest struct {
	PeerAddress string `json:"peer_address"` // e.g., "192.168.1.100:9001"
	UseRelay    bool   `json:"use_relay"`    // true to use relay server
	RelayServer string `json:"relay_server"` // e.g., "94.237.121.21:9545" (optional, uses default if empty)
	PeerID      string `json:"peer_id"`      // peer ID for relay mode (optional, auto-generated if empty)
}

// ConnectResponse represents the response to a connect request
type ConnectResponse struct {
	Status  string `json:"status"`  // "success" or "error"
	Message string `json:"message"` // Human-readable message
}

// DisconnectResponse represents the response to a disconnect request
type DisconnectResponse struct {
	Status  string `json:"status"`  // "success" or "error"
	Message string `json:"message"` // Human-readable message
}

// StatusResponse represents the daemon status
type StatusResponse struct {
	Status      string                 `json:"status"`  // "success" or "error"
	DaemonState string                 `json:"state"`   // "connected", "disconnected", etc.
	Details     map[string]interface{} `json:"details"` // Detailed status from manager
}

// HealthResponse represents the daemon health check
type HealthResponse struct {
	Status  string `json:"status"`  // "healthy" or "unhealthy"
	Message string `json:"message"` // Health message
}

// handleConnect handles /connect endpoint
func (api *DaemonAPI) handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendJSON(w, http.StatusBadRequest, ConnectResponse{
			Status:  "error",
			Message: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Check if relay mode is requested (via API or config)
	useRelay := req.UseRelay || api.manager.config.Relay.Enabled

	if useRelay {
		// Relay mode
		relayServer := req.RelayServer
		if relayServer == "" {
			relayServer = api.manager.config.Relay.Server
		}
		if relayServer == "" {
			api.sendJSON(w, http.StatusBadRequest, ConnectResponse{
				Status:  "error",
				Message: "relay_server is required when use_relay is true",
			})
			return
		}

		peerID := req.PeerID
		if peerID == "" {
			peerID = api.manager.config.Peer.ID
		}
		if peerID == "" {
			// Generate random peer ID
			peerID = fmt.Sprintf("peer-%d", time.Now().UnixNano())
		}

		log.Printf("Connecting via relay server: %s (peer ID: %s)", relayServer, peerID)

		// Temporarily set config for relay connection
		api.manager.config.Relay.Enabled = true
		api.manager.config.Relay.Server = relayServer
		api.manager.config.Peer.ID = peerID

		// Connect to relay server
		if err := api.manager.Connect(""); err != nil {
			api.sendJSON(w, http.StatusInternalServerError, ConnectResponse{
				Status:  "error",
				Message: fmt.Sprintf("Relay connection failed: %v", err),
			})
			return
		}

		api.sendJSON(w, http.StatusOK, ConnectResponse{
			Status:  "success",
			Message: fmt.Sprintf("Connected to relay server %s as peer %s", relayServer, peerID),
		})
		return
	}

	// Direct P2P mode requires peer_address
	if req.PeerAddress == "" {
		api.sendJSON(w, http.StatusBadRequest, ConnectResponse{
			Status:  "error",
			Message: "peer_address is required for direct P2P mode",
		})
		return
	}

	// Connect to peer directly
	if err := api.manager.Connect(req.PeerAddress); err != nil {
		api.sendJSON(w, http.StatusInternalServerError, ConnectResponse{
			Status:  "error",
			Message: fmt.Sprintf("Connection failed: %v", err),
		})
		return
	}

	api.sendJSON(w, http.StatusOK, ConnectResponse{
		Status:  "success",
		Message: fmt.Sprintf("Connected to peer at %s", req.PeerAddress),
	})
}

// handleDisconnect handles /disconnect endpoint
func (api *DaemonAPI) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Disconnect from peer
	if err := api.manager.Disconnect(); err != nil {
		api.sendJSON(w, http.StatusInternalServerError, DisconnectResponse{
			Status:  "error",
			Message: fmt.Sprintf("Disconnect failed: %v", err),
		})
		return
	}

	api.sendJSON(w, http.StatusOK, DisconnectResponse{
		Status:  "success",
		Message: "Disconnected successfully",
	})
}

// handleStatus handles /status endpoint
func (api *DaemonAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := api.manager.GetStatus()

	api.sendJSON(w, http.StatusOK, StatusResponse{
		Status:      "success",
		DaemonState: status["state"].(string),
		Details:     status,
	})
}

// handleHealth handles /health endpoint
func (api *DaemonAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	api.sendJSON(w, http.StatusOK, HealthResponse{
		Status:  "healthy",
		Message: "Daemon is running",
	})
}

// sendJSON sends a JSON response
func (api *DaemonAPI) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("⚠️  Error encoding JSON response: %v", err)
	}
}
