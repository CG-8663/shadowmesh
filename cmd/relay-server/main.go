package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// PeerConnection represents a connected peer
type PeerConnection struct {
	ID         string
	Conn       *websocket.Conn
	SendChan   chan []byte
	LastActive time.Time
	mu         sync.Mutex
}

// RelayServer manages peer connections and frame forwarding
type RelayServer struct {
	peers      map[string]*PeerConnection
	peersMutex sync.RWMutex
	upgrader   websocket.Upgrader
	port       int
}

// NewRelayServer creates a new relay server
func NewRelayServer(port int) *RelayServer {
	return &RelayServer{
		peers: make(map[string]*PeerConnection),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true // Accept all origins for now
			},
		},
		port: port,
	}
}

// handleWebSocket handles incoming WebSocket connections
func (rs *RelayServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract peer ID from URL query parameter
	peerID := r.URL.Query().Get("peer_id")
	if peerID == "" {
		http.Error(w, "peer_id required", http.StatusBadRequest)
		return
	}

	// Upgrade to WebSocket
	conn, err := rs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}

	log.Printf("✅ Peer connected: %s from %s", peerID, r.RemoteAddr)

	// Create peer connection
	peer := &PeerConnection{
		ID:         peerID,
		Conn:       conn,
		SendChan:   make(chan []byte, 1000),
		LastActive: time.Now(),
	}

	// Register peer
	rs.peersMutex.Lock()
	if existingPeer, exists := rs.peers[peerID]; exists {
		// Close old connection
		existingPeer.Conn.Close()
		log.Printf("⚠️  Replacing existing connection for peer %s", peerID)
	}
	rs.peers[peerID] = peer
	rs.peersMutex.Unlock()

	// Cleanup on disconnect
	defer func() {
		rs.peersMutex.Lock()
		delete(rs.peers, peerID)
		rs.peersMutex.Unlock()
		conn.Close()
		log.Printf("🔌 Peer disconnected: %s", peerID)
	}()

	// Start send/receive loops
	var wg sync.WaitGroup
	wg.Add(2)

	// Send loop
	go func() {
		defer wg.Done()
		for frame := range peer.SendChan {
			if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				log.Printf("⚠️  Failed to send frame to %s: %v", peerID, err)
				return
			}
		}
	}()

	// Receive loop
	go func() {
		defer wg.Done()
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				log.Printf("⚠️  Read error from %s: %v", peerID, err)
				return
			}

			if msgType != websocket.BinaryMessage {
				log.Printf("⚠️  Unexpected message type from %s: %d", peerID, msgType)
				continue
			}

			// Update last active time
			peer.mu.Lock()
			peer.LastActive = time.Now()
			peer.mu.Unlock()

			// Forward frame to all other peers (broadcast mode for now)
			rs.forwardFrame(peerID, data)
		}
	}()

	wg.Wait()
}

// forwardFrame forwards a frame from one peer to all others
func (rs *RelayServer) forwardFrame(senderID string, frame []byte) {
	rs.peersMutex.RLock()
	defer rs.peersMutex.RUnlock()

	forwarded := 0
	for id, peer := range rs.peers {
		if id == senderID {
			continue // Don't forward to sender
		}

		select {
		case peer.SendChan <- frame:
			forwarded++
		default:
			log.Printf("⚠️  Send buffer full for peer %s, dropping frame", id)
		}
	}

	if forwarded > 0 {
		log.Printf("📤 Forwarded frame from %s to %d peer(s)", senderID, forwarded)
	}
}

// handleStatus provides server status endpoint
func (rs *RelayServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	rs.peersMutex.RLock()
	defer rs.peersMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","connected_peers":%d,"peers":[`, len(rs.peers))

	first := true
	for id, peer := range rs.peers {
		if !first {
			fmt.Fprint(w, ",")
		}
		peer.mu.Lock()
		lastActive := peer.LastActive
		peer.mu.Unlock()
		fmt.Fprintf(w, `{"id":"%s","last_active":"%s"}`, id, lastActive.Format(time.RFC3339))
		first = false
	}

	fmt.Fprint(w, `]}`)
}

// cleanupStaleConnections removes inactive peers
func (rs *RelayServer) cleanupStaleConnections(ctx context.Context, timeout time.Duration) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rs.peersMutex.Lock()
			now := time.Now()
			for id, peer := range rs.peers {
				peer.mu.Lock()
				lastActive := peer.LastActive
				peer.mu.Unlock()

				if now.Sub(lastActive) > timeout {
					log.Printf("🧹 Removing stale peer: %s (inactive for %v)", id, now.Sub(lastActive))
					peer.Conn.Close()
					delete(rs.peers, id)
				}
			}
			rs.peersMutex.Unlock()
		}
	}
}

// Start starts the relay server
func (rs *RelayServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/relay", rs.handleWebSocket)
	mux.HandleFunc("/status", rs.handleStatus)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", rs.port),
		Handler: mux,
	}

	// Start cleanup goroutine
	go rs.cleanupStaleConnections(ctx, 5*time.Minute)

	// Start server
	go func() {
		log.Printf("🚀 ShadowMesh Relay Server starting on port %d", rs.port)
		log.Printf("   WebSocket endpoint: ws://0.0.0.0:%d/relay?peer_id=<id>", rs.port)
		log.Printf("   Status endpoint: http://0.0.0.0:%d/status", rs.port)
		log.Printf("   Health endpoint: http://0.0.0.0:%d/health", rs.port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Wait for shutdown
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("🛑 Shutting down relay server...")
	return server.Shutdown(shutdownCtx)
}

func main() {
	port := flag.Int("port", 9545, "Port to listen on")
	flag.Parse()

	// Create relay server
	relay := NewRelayServer(*port)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n⚠️  Received shutdown signal")
		cancel()
	}()

	// Start relay server
	if err := relay.Start(ctx); err != nil {
		log.Fatalf("❌ Relay server error: %v", err)
	}

	log.Println("✅ Relay server shutdown complete")
}
