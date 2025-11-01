package main

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/shadowmesh/shadowmesh/shared/protocol"
)

// RoutingMode defines how frames are routed
type RoutingMode int

const (
	// RoutingModeBroadcast broadcasts frames to all other clients
	RoutingModeBroadcast RoutingMode = iota

	// RoutingModeDirect routes frames based on destination address (future)
	RoutingModeDirect
)

// Router handles frame routing between clients
type Router struct {
	// Configuration
	mode           RoutingMode
	maxFrameSize   int

	// Connection manager reference
	connMgr        *ConnectionManager

	// Statistics
	framesRouted   atomic.Uint64
	framesFailed   atomic.Uint64
	bytesRouted    atomic.Uint64
	broadcastCount atomic.Uint64

	// Routing table (for future direct routing)
	routingTable   map[[6]byte][32]byte // MAC address -> ClientID
	routingMutex   sync.RWMutex
}

// NewRouter creates a new frame router
func NewRouter(connMgr *ConnectionManager, mode RoutingMode, maxFrameSize int) *Router {
	return &Router{
		mode:         mode,
		maxFrameSize: maxFrameSize,
		connMgr:      connMgr,
		routingTable: make(map[[6]byte][32]byte),
	}
}

// RouteFrame routes a data frame from source client to destination(s)
func (r *Router) RouteFrame(source *ClientConnection, msg *protocol.Message) {
	// Validate message type
	if msg.Header.Type != protocol.MsgTypeDataFrame {
		log.Printf("Router received non-data frame (type %d) from client %x",
			msg.Header.Type,
			source.clientID[:8])
		r.framesFailed.Add(1)
		return
	}

	// Validate frame size
	if msg.Header.Length > uint32(r.maxFrameSize) {
		log.Printf("Oversized frame (%d bytes) from client %x, dropping",
			msg.Header.Length,
			source.clientID[:8])
		r.framesFailed.Add(1)
		return
	}

	// Extract payload
	dataPayload, ok := msg.Payload.(*protocol.DataFrame)
	if !ok {
		log.Printf("Invalid data frame payload from client %x", source.clientID[:8])
		r.framesFailed.Add(1)
		return
	}

	// Route based on mode
	switch r.mode {
	case RoutingModeBroadcast:
		r.routeBroadcast(source, msg, dataPayload)

	case RoutingModeDirect:
		r.routeDirect(source, msg, dataPayload)

	default:
		log.Printf("Unknown routing mode: %d", r.mode)
		r.framesFailed.Add(1)
	}
}

// routeBroadcast broadcasts a frame to all other clients
func (r *Router) routeBroadcast(source *ClientConnection, msg *protocol.Message, data *protocol.DataFrame) {
	// STEP 1: Decrypt the frame using relay's RX encryptor for source client
	// Use the persistent encryptor to maintain nonce consistency
	if source.rxEncryptor == nil {
		log.Printf("RX encryptor not initialized for source client %x", source.clientID[:8])
		r.framesFailed.Add(1)
		return
	}

	plaintext, err := source.rxEncryptor.Decrypt(data.EncryptedData)
	if err != nil {
		log.Printf("Failed to decrypt frame from client %x: %v", source.clientID[:8], err)
		r.framesFailed.Add(1)
		return
	}

	// STEP 2: Get all destination clients except source
	r.connMgr.clientsMutex.RLock()
	destinations := make([]*ClientConnection, 0, len(r.connMgr.clients)-1)
	for clientID, client := range r.connMgr.clients {
		// Skip source client
		if clientID == source.clientID {
			continue
		}

		// Only send to established clients
		if client.getState() == ClientStateEstablished {
			destinations = append(destinations, client)
		}
	}
	r.connMgr.clientsMutex.RUnlock()

	// STEP 3: Re-encrypt and send to each destination
	successCount := 0
	for _, dest := range destinations {
		// Use the persistent TX encryptor for destination client
		// This maintains nonce consistency for all encrypted frames to this client
		if dest.txEncryptor == nil {
			log.Printf("TX encryptor not initialized for dest client %x", dest.clientID[:8])
			r.framesFailed.Add(1)
			continue
		}

		// Re-encrypt plaintext for destination
		reEncrypted, err := dest.txEncryptor.Encrypt(plaintext)
		if err != nil {
			log.Printf("Failed to re-encrypt frame for client %x: %v", dest.clientID[:8], err)
			r.framesFailed.Add(1)
			continue
		}

		// Create new message with re-encrypted data
		reEncryptedMsg := protocol.NewDataFrameMessage(data.Counter, reEncrypted)

		// Send re-encrypted frame to destination
		if err := dest.SendMessage(reEncryptedMsg); err != nil {
			log.Printf("Failed to route frame from %x to %x: %v",
				source.clientID[:8],
				dest.clientID[:8],
				err)
			r.framesFailed.Add(1)
		} else {
			successCount++
		}
	}

	// Update statistics
	if successCount > 0 {
		r.framesRouted.Add(1)
		r.bytesRouted.Add(uint64(len(plaintext)))
		r.broadcastCount.Add(uint64(successCount))
	}
}

// routeDirect routes a frame to a specific destination based on MAC address
//
// This is a placeholder for future implementation. In a real Layer 2 network,
// the relay would maintain a routing table mapping MAC addresses to client IDs.
func (r *Router) routeDirect(source *ClientConnection, msg *protocol.Message, data *protocol.DataFrame) {
	// Extract destination MAC from Ethernet frame
	// Ethernet frame format: [6 bytes dest MAC][6 bytes source MAC][2 bytes ethertype][payload]
	if len(data.EncryptedData) < 14 {
		log.Printf("Frame too short for Ethernet header from client %x", source.clientID[:8])
		r.framesFailed.Add(1)
		return
	}

	// Note: Frame is encrypted, so we can't actually read the MAC address
	// In production, we would need either:
	// 1. A control plane protocol for route distribution
	// 2. Authenticated but unencrypted headers
	// 3. Initial broadcast with learning

	// For now, fall back to broadcast
	log.Printf("Direct routing not yet implemented, falling back to broadcast")
	r.routeBroadcast(source, msg, data)
}

// LearnRoute learns a route mapping (for future direct routing)
//
// In a learning bridge/switch, when a frame arrives from a client,
// we learn that the source MAC address is reachable via that client.
func (r *Router) LearnRoute(macAddr [6]byte, clientID [32]byte) {
	r.routingMutex.Lock()
	defer r.routingMutex.Unlock()

	// Check if route already exists
	if existingClientID, exists := r.routingTable[macAddr]; exists {
		if existingClientID != clientID {
			log.Printf("MAC %x moved from client %x to %x",
				macAddr[:],
				existingClientID[:8],
				clientID[:8])
		}
	}

	r.routingTable[macAddr] = clientID
}

// LookupRoute looks up a route for a MAC address
func (r *Router) LookupRoute(macAddr [6]byte) ([32]byte, bool) {
	r.routingMutex.RLock()
	defer r.routingMutex.RUnlock()

	clientID, exists := r.routingTable[macAddr]
	return clientID, exists
}

// RemoveClientRoutes removes all routes for a disconnected client
func (r *Router) RemoveClientRoutes(clientID [32]byte) {
	r.routingMutex.Lock()
	defer r.routingMutex.Unlock()

	// Find and remove all routes pointing to this client
	for macAddr, routeClientID := range r.routingTable {
		if routeClientID == clientID {
			delete(r.routingTable, macAddr)
			log.Printf("Removed route for MAC %x (client %x disconnected)",
				macAddr[:],
				clientID[:8])
		}
	}
}

// GetStats returns routing statistics
func (r *Router) GetStats() RouterStats {
	return RouterStats{
		FramesRouted:   r.framesRouted.Load(),
		FramesFailed:   r.framesFailed.Load(),
		BytesRouted:    r.bytesRouted.Load(),
		BroadcastCount: r.broadcastCount.Load(),
		RoutingTableSize: func() int {
			r.routingMutex.RLock()
			defer r.routingMutex.RUnlock()
			return len(r.routingTable)
		}(),
	}
}

// RouterStats holds routing statistics
type RouterStats struct {
	FramesRouted     uint64 `json:"frames_routed"`
	FramesFailed     uint64 `json:"frames_failed"`
	BytesRouted      uint64 `json:"bytes_routed"`
	BroadcastCount   uint64 `json:"broadcast_count"`
	RoutingTableSize int    `json:"routing_table_size"`
}

// SetRoutingMode changes the routing mode
func (r *Router) SetRoutingMode(mode RoutingMode) {
	r.mode = mode
	log.Printf("Routing mode changed to %d", mode)
}

// GetRoutingMode returns the current routing mode
func (r *Router) GetRoutingMode() RoutingMode {
	return r.mode
}

// ClearRoutingTable clears all learned routes
func (r *Router) ClearRoutingTable() {
	r.routingMutex.Lock()
	defer r.routingMutex.Unlock()

	r.routingTable = make(map[[6]byte][32]byte)
	log.Println("Routing table cleared")
}

// GetRoutingTableSnapshot returns a copy of the routing table
func (r *Router) GetRoutingTableSnapshot() map[[6]byte][32]byte {
	r.routingMutex.RLock()
	defer r.routingMutex.RUnlock()

	// Create a copy
	snapshot := make(map[[6]byte][32]byte, len(r.routingTable))
	for mac, clientID := range r.routingTable {
		snapshot[mac] = clientID
	}

	return snapshot
}
