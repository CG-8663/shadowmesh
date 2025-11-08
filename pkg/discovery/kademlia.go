package discovery

import (
	"encoding/hex"
	"errors"
	"math/big"
	"sync"
	"time"
)

const (
	// Kademlia parameters
	K               = 20  // K-bucket size (number of peers per bucket)
	Alpha           = 3   // Parallelism for lookups
	BucketCount     = 160 // Number of k-buckets (160-bit address space)
	RefreshInterval = 3600 // Bucket refresh interval in seconds (1 hour)
)

// PeerInfo represents a peer in the DHT
type PeerInfo struct {
	PeerID       string    `json:"peer_id"`
	PublicKey    string    `json:"public_key"`
	IPAddress    string    `json:"ip_address"`
	Port         int       `json:"port"`
	IsPublic     bool      `json:"is_public"`     // PUBLIC or PRIVATE service mode
	LastSeen     time.Time `json:"last_seen"`
	Verified     bool      `json:"verified"`      // Has valid session
	SessionToken string    `json:"-"`             // Not exported
}

// KBucket represents a single k-bucket in the routing table
type KBucket struct {
	peers       []*PeerInfo
	lastUpdated time.Time
	mu          sync.RWMutex
}

// NewKBucket creates a new k-bucket
func NewKBucket() *KBucket {
	return &KBucket{
		peers:       make([]*PeerInfo, 0, K),
		lastUpdated: time.Now(),
	}
}

// Add adds a peer to the k-bucket (LRU eviction policy)
func (kb *KBucket) Add(peer *PeerInfo) bool {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	// Check if peer already exists
	for i, p := range kb.peers {
		if p.PeerID == peer.PeerID {
			// Move to end (most recently seen)
			kb.peers = append(kb.peers[:i], kb.peers[i+1:]...)
			kb.peers = append(kb.peers, peer)
			kb.lastUpdated = time.Now()
			return true
		}
	}

	// Add new peer
	if len(kb.peers) < K {
		kb.peers = append(kb.peers, peer)
		kb.lastUpdated = time.Now()
		return true
	}

	// Bucket full, check if first peer is stale (not seen in 24h)
	if time.Since(kb.peers[0].LastSeen) > 24*time.Hour {
		// Evict stale peer, add new one
		kb.peers = append(kb.peers[1:], peer)
		kb.lastUpdated = time.Now()
		return true
	}

	return false // Bucket full, peer rejected
}

// Remove removes a peer from the k-bucket
func (kb *KBucket) Remove(peerID string) bool {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	for i, p := range kb.peers {
		if p.PeerID == peerID {
			kb.peers = append(kb.peers[:i], kb.peers[i+1:]...)
			return true
		}
	}
	return false
}

// GetAll returns all peers in the bucket
func (kb *KBucket) GetAll() []*PeerInfo {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	result := make([]*PeerInfo, len(kb.peers))
	copy(result, kb.peers)
	return result
}

// Size returns the number of peers in the bucket
func (kb *KBucket) Size() int {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	return len(kb.peers)
}

// KademliaTable represents the full Kademlia routing table
type KademliaTable struct {
	buckets    [BucketCount]*KBucket
	localID    string // This node's peer ID
	mu         sync.RWMutex
	peerCount  int
}

// NewKademliaTable creates a new Kademlia routing table
func NewKademliaTable(localID string) *KademliaTable {
	kt := &KademliaTable{
		localID:   localID,
		peerCount: 0,
	}
	for i := 0; i < BucketCount; i++ {
		kt.buckets[i] = NewKBucket()
	}
	return kt
}

// AddPeer adds a peer to the routing table
func (kt *KademliaTable) AddPeer(peer *PeerInfo) error {
	if peer.PeerID == kt.localID {
		return errors.New("cannot add self to routing table")
	}

	// Calculate bucket index using XOR distance
	bucketIndex := kt.bucketIndex(kt.localID, peer.PeerID)
	if bucketIndex < 0 || bucketIndex >= BucketCount {
		return errors.New("invalid bucket index")
	}

	// Add to bucket
	added := kt.buckets[bucketIndex].Add(peer)
	if added {
		kt.mu.Lock()
		kt.peerCount++
		kt.mu.Unlock()
	}

	return nil
}

// RemovePeer removes a peer from the routing table
func (kt *KademliaTable) RemovePeer(peerID string) bool {
	bucketIndex := kt.bucketIndex(kt.localID, peerID)
	if bucketIndex < 0 || bucketIndex >= BucketCount {
		return false
	}

	removed := kt.buckets[bucketIndex].Remove(peerID)
	if removed {
		kt.mu.Lock()
		kt.peerCount--
		kt.mu.Unlock()
	}
	return removed
}

// FindClosest finds the K closest peers to a target peer ID
func (kt *KademliaTable) FindClosest(targetID string, count int) ([]*PeerInfo, error) {
	if count <= 0 {
		count = K
	}

	kt.mu.RLock()
	defer kt.mu.RUnlock()

	// Collect all peers from all buckets
	allPeers := make([]*PeerInfo, 0, kt.peerCount)
	for i := 0; i < BucketCount; i++ {
		peers := kt.buckets[i].GetAll()
		allPeers = append(allPeers, peers...)
	}

	if len(allPeers) == 0 {
		return nil, errors.New("no peers in routing table")
	}

	// Calculate distances and sort
	type peerDistance struct {
		peer     *PeerInfo
		distance *big.Int
	}

	distances := make([]peerDistance, len(allPeers))
	for i, peer := range allPeers {
		dist := xorDistance(targetID, peer.PeerID)
		distances[i] = peerDistance{peer: peer, distance: dist}
	}

	// Simple bubble sort for K peers (small K, so acceptable)
	for i := 0; i < len(distances)-1; i++ {
		for j := 0; j < len(distances)-i-1; j++ {
			if distances[j].distance.Cmp(distances[j+1].distance) > 0 {
				distances[j], distances[j+1] = distances[j+1], distances[j]
			}
		}
	}

	// Return top K peers
	resultCount := count
	if len(distances) < resultCount {
		resultCount = len(distances)
	}

	result := make([]*PeerInfo, resultCount)
	for i := 0; i < resultCount; i++ {
		result[i] = distances[i].peer
	}

	return result, nil
}

// GetPeer retrieves a specific peer by ID
func (kt *KademliaTable) GetPeer(peerID string) (*PeerInfo, error) {
	bucketIndex := kt.bucketIndex(kt.localID, peerID)
	if bucketIndex < 0 || bucketIndex >= BucketCount {
		return nil, errors.New("invalid bucket index")
	}

	peers := kt.buckets[bucketIndex].GetAll()
	for _, peer := range peers {
		if peer.PeerID == peerID {
			return peer, nil
		}
	}

	return nil, errors.New("peer not found")
}

// GetAllPeers returns all peers in the routing table
func (kt *KademliaTable) GetAllPeers() []*PeerInfo {
	kt.mu.RLock()
	defer kt.mu.RUnlock()

	allPeers := make([]*PeerInfo, 0, kt.peerCount)
	for i := 0; i < BucketCount; i++ {
		peers := kt.buckets[i].GetAll()
		allPeers = append(allPeers, peers...)
	}

	return allPeers
}

// GetPublicPeers returns only peers with IsPublic=true
func (kt *KademliaTable) GetPublicPeers() []*PeerInfo {
	allPeers := kt.GetAllPeers()
	publicPeers := make([]*PeerInfo, 0)
	for _, peer := range allPeers {
		if peer.IsPublic {
			publicPeers = append(publicPeers, peer)
		}
	}
	return publicPeers
}

// Size returns the total number of peers in the routing table
func (kt *KademliaTable) Size() int {
	kt.mu.RLock()
	defer kt.mu.RUnlock()
	return kt.peerCount
}

// bucketIndex calculates the bucket index for a peer based on XOR distance
// Returns the index of the most significant bit in the XOR result
func (kt *KademliaTable) bucketIndex(localID, peerID string) int {
	dist := xorDistance(localID, peerID)
	if dist.Sign() == 0 {
		return -1 // Same peer ID
	}

	// Find the most significant bit position
	bitLength := dist.BitLen()
	return BucketCount - bitLength
}

// xorDistance calculates the XOR distance between two peer IDs
func xorDistance(id1, id2 string) *big.Int {
	// Decode hex peer IDs
	bytes1, err := hex.DecodeString(id1)
	if err != nil {
		return big.NewInt(0)
	}

	bytes2, err := hex.DecodeString(id2)
	if err != nil {
		return big.NewInt(0)
	}

	// Pad to 20 bytes if needed
	if len(bytes1) < 20 {
		padded := make([]byte, 20)
		copy(padded[20-len(bytes1):], bytes1)
		bytes1 = padded
	}
	if len(bytes2) < 20 {
		padded := make([]byte, 20)
		copy(padded[20-len(bytes2):], bytes2)
		bytes2 = padded
	}

	// XOR byte by byte
	result := make([]byte, 20)
	for i := 0; i < 20; i++ {
		result[i] = bytes1[i] ^ bytes2[i]
	}

	// Convert to big.Int
	return new(big.Int).SetBytes(result)
}

// CleanupStale removes peers not seen in 24 hours
func (kt *KademliaTable) CleanupStale() int {
	removed := 0
	for i := 0; i < BucketCount; i++ {
		peers := kt.buckets[i].GetAll()
		for _, peer := range peers {
			if time.Since(peer.LastSeen) > 24*time.Hour {
				if kt.buckets[i].Remove(peer.PeerID) {
					removed++
				}
			}
		}
	}

	kt.mu.Lock()
	kt.peerCount -= removed
	kt.mu.Unlock()

	return removed
}

// GetStats returns routing table statistics
func (kt *KademliaTable) GetStats() map[string]interface{} {
	kt.mu.RLock()
	defer kt.mu.RUnlock()

	bucketSizes := make([]int, BucketCount)
	for i := 0; i < BucketCount; i++ {
		bucketSizes[i] = kt.buckets[i].Size()
	}

	publicCount := 0
	verifiedCount := 0
	for _, peer := range kt.GetAllPeers() {
		if peer.IsPublic {
			publicCount++
		}
		if peer.Verified {
			verifiedCount++
		}
	}

	return map[string]interface{}{
		"total_peers":    kt.peerCount,
		"public_peers":   publicCount,
		"verified_peers": verifiedCount,
		"bucket_sizes":   bucketSizes,
	}
}
