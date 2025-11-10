# ShadowMesh DHT Implementation Tickets - Phase 1

**Timeline**: Weeks 1-4 (Sprint 0-1)
**Target**: v0.2.0-alpha standalone release preparation
**Last Updated**: November 10, 2025

---

## Sprint 0: DHT Foundation (Weeks 1-2)

### Epic 1: PeerID Generation & Verification

#### TICKET-001: Implement PeerID Generation from ML-DSA-87
**Priority**: P0 (Blocker)
**Estimate**: 2 days
**Assignee**: [TBD]

**Description**:
Implement cryptographically verifiable PeerID generation from ML-DSA-87 post-quantum signature public keys.

**Acceptance Criteria**:
- [ ] `GeneratePeerID(mldsaPublicKey []byte) PeerID` function implemented
- [ ] PeerID is SHA256 hash of ML-DSA-87 public key (32 bytes)
- [ ] PeerID structure defined with proper serialization
- [ ] Unit tests: Generate 1000 PeerIDs, verify no collisions
- [ ] Unit tests: Same public key generates same PeerID (deterministic)
- [ ] Benchmark: PeerID generation <1ms per operation

**Implementation Details**:
```go
// File: pkg/discovery/peer_id.go

package discovery

import "crypto/sha256"

type PeerID [32]byte // 256-bit identifier

func GeneratePeerID(mldsaPublicKey []byte) PeerID {
    hash := sha256.Sum256(mldsaPublicKey)
    return PeerID(hash)
}

func (p PeerID) String() string {
    return hex.EncodeToString(p[:])
}

func ParsePeerID(s string) (PeerID, error) {
    // Parse hex string to PeerID
}
```

**Testing**:
```bash
go test ./pkg/discovery -run TestGeneratePeerID
go test ./pkg/discovery -bench BenchmarkGeneratePeerID
```

**Dependencies**: None

---

#### TICKET-002: Implement PeerID Ownership Verification
**Priority**: P0 (Blocker)
**Estimate**: 2 days
**Assignee**: [TBD]

**Description**:
Implement challenge-response protocol for peers to prove ownership of their PeerID through ML-DSA-87 signatures.

**Acceptance Criteria**:
- [ ] `VerifyPeerOwnership(peerID, publicKey, signature, challenge) bool` implemented
- [ ] Generate random 32-byte challenge
- [ ] Verify PeerID matches public key: `GeneratePeerID(publicKey) == peerID`
- [ ] Verify ML-DSA-87 signature over challenge
- [ ] Unit tests: Valid signatures pass verification
- [ ] Unit tests: Invalid signatures fail verification
- [ ] Unit tests: Mismatched PeerID/public key fails

**Implementation Details**:
```go
// File: pkg/discovery/peer_verification.go

func VerifyPeerOwnership(peerID PeerID, publicKey []byte, signature []byte, challenge []byte) error {
    // 1. Verify PeerID derivation
    derivedID := GeneratePeerID(publicKey)
    if derivedID != peerID {
        return ErrPeerIDMismatch
    }

    // 2. Verify ML-DSA-87 signature
    if !crypto.VerifyMLDSA87(publicKey, challenge, signature) {
        return ErrInvalidSignature
    }

    return nil
}

func GenerateChallenge() [32]byte {
    var challenge [32]byte
    rand.Read(challenge[:])
    return challenge
}
```

**Testing**:
```bash
go test ./pkg/discovery -run TestVerifyPeerOwnership
```

**Dependencies**: TICKET-001, Existing ML-DSA-87 crypto implementation

---

### Epic 2: Routing Table Implementation

#### TICKET-003: Implement XOR Distance Metric
**Priority**: P0 (Blocker)
**Estimate**: 1 day
**Assignee**: [TBD]

**Description**:
Implement XOR distance calculation between two PeerIDs for Kademlia routing.

**Acceptance Criteria**:
- [ ] `XORDistance(id1, id2 PeerID) *big.Int` implemented
- [ ] Commutative property: `XOR(A, B) == XOR(B, A)`
- [ ] Identity property: `XOR(A, A) == 0`
- [ ] Triangle inequality: `XOR(A, C) <= XOR(A, B) + XOR(B, C)`
- [ ] Unit tests: All XOR properties validated
- [ ] Benchmark: XOR calculation <10μs

**Implementation Details**:
```go
// File: pkg/discovery/distance.go

import "math/big"

func XORDistance(id1, id2 PeerID) *big.Int {
    distance := new(big.Int).SetBytes(id1[:])
    xor := new(big.Int).SetBytes(id2[:])
    distance.Xor(distance, xor)
    return distance
}

func CompareDistance(target, id1, id2 PeerID) int {
    d1 := XORDistance(target, id1)
    d2 := XORDistance(target, id2)
    return d1.Cmp(d2) // -1 if d1 < d2, 0 if equal, 1 if d1 > d2
}
```

**Testing**:
```bash
go test ./pkg/discovery -run TestXORDistance
go test ./pkg/discovery -bench BenchmarkXORDistance
```

**Dependencies**: TICKET-001

---

#### TICKET-004: Implement k-bucket Structure
**Priority**: P0 (Blocker)
**Estimate**: 3 days
**Assignee**: [TBD]

**Description**:
Implement individual k-bucket data structure with LRU eviction policy.

**Acceptance Criteria**:
- [ ] `KBucket` struct stores up to k=20 peers
- [ ] LRU eviction when bucket full
- [ ] Thread-safe operations (sync.RWMutex)
- [ ] `AddPeer()`, `RemovePeer()`, `GetPeers()` methods
- [ ] `UpdateLastSeen()` moves peer to front (most recently used)
- [ ] Unit tests: Add 20 peers, verify all stored
- [ ] Unit tests: Add 21st peer, verify oldest evicted
- [ ] Unit tests: Concurrent access (100 goroutines)

**Implementation Details**:
```go
// File: pkg/discovery/kbucket.go

type KBucket struct {
    peers       []PeerInfo    // Up to k peers
    k           int           // Max peers (default: 20)
    lastUpdated time.Time
    mutex       sync.RWMutex
}

type PeerInfo struct {
    PeerID       PeerID
    Address      string        // "IP:port"
    PublicKey    []byte        // ML-DSA-87 public key
    LastSeen     time.Time
    Capabilities []string      // ["peer", "relay", "exit-node"]
}

func NewKBucket(k int) *KBucket {
    return &KBucket{
        peers:       make([]PeerInfo, 0, k),
        k:           k,
        lastUpdated: time.Now(),
    }
}

func (kb *KBucket) AddPeer(peer PeerInfo) {
    kb.mutex.Lock()
    defer kb.mutex.Unlock()

    // Check if peer exists (update last seen)
    for i, p := range kb.peers {
        if p.PeerID == peer.PeerID {
            kb.peers[i].LastSeen = time.Now()
            // Move to front (LRU)
            kb.peers = append([]PeerInfo{kb.peers[i]}, append(kb.peers[:i], kb.peers[i+1:]...)...)
            return
        }
    }

    // Add new peer
    if len(kb.peers) < kb.k {
        kb.peers = append([]PeerInfo{peer}, kb.peers...)
    } else {
        // Replace least recently used (last in list)
        kb.peers = append([]PeerInfo{peer}, kb.peers[:kb.k-1]...)
    }

    kb.lastUpdated = time.Now()
}

func (kb *KBucket) GetPeers() []PeerInfo {
    kb.mutex.RLock()
    defer kb.mutex.RUnlock()

    peers := make([]PeerInfo, len(kb.peers))
    copy(peers, kb.peers)
    return peers
}
```

**Testing**:
```bash
go test ./pkg/discovery -run TestKBucket
go test ./pkg/discovery -run TestKBucketConcurrency -race
```

**Dependencies**: TICKET-001

---

#### TICKET-005: Implement Routing Table
**Priority**: P0 (Blocker)
**Estimate**: 4 days
**Assignee**: [TBD]

**Description**:
Implement complete routing table with 256 k-buckets for Kademlia DHT.

**Acceptance Criteria**:
- [ ] `RoutingTable` struct with 256 k-buckets
- [ ] `BucketIndex(peerID) int` calculates correct bucket (0-255)
- [ ] `AddPeer(peer)` adds to correct bucket
- [ ] `FindClosest(target, k) []PeerInfo` returns k closest peers
- [ ] `RemovePeer(peerID)` removes from routing table
- [ ] `AllPeers() []PeerInfo` returns all peers across all buckets
- [ ] Thread-safe operations
- [ ] Unit tests: Add 1000 peers, verify distribution across buckets
- [ ] Unit tests: FindClosest returns peers in distance order
- [ ] Unit tests: Bucket index calculation correct for all PeerIDs

**Implementation Details**:
```go
// File: pkg/discovery/routing_table.go

type RoutingTable struct {
    localPeerID PeerID
    kBuckets    [256]*KBucket
    k           int           // Max peers per bucket
    mutex       sync.RWMutex
}

func NewRoutingTable(localPeerID PeerID, k int) *RoutingTable {
    rt := &RoutingTable{
        localPeerID: localPeerID,
        k:           k,
    }
    for i := 0; i < 256; i++ {
        rt.kBuckets[i] = NewKBucket(k)
    }
    return rt
}

func (rt *RoutingTable) BucketIndex(peerID PeerID) int {
    distance := XORDistance(rt.localPeerID, peerID)
    // Count leading zeros
    bitLen := distance.BitLen()
    if bitLen == 0 {
        return 0 // Same peer (distance = 0)
    }
    return 256 - bitLen
}

func (rt *RoutingTable) FindClosest(target PeerID, k int) []PeerInfo {
    rt.mutex.RLock()
    defer rt.mutex.RUnlock()

    // Collect all peers
    allPeers := make([]PeerInfo, 0, 256*rt.k)
    for _, bucket := range rt.kBuckets {
        allPeers = append(allPeers, bucket.GetPeers()...)
    }

    // Sort by XOR distance to target
    sort.Slice(allPeers, func(i, j int) bool {
        return CompareDistance(target, allPeers[i].PeerID, allPeers[j].PeerID) < 0
    })

    // Return top k
    if len(allPeers) < k {
        return allPeers
    }
    return allPeers[:k]
}
```

**Testing**:
```bash
go test ./pkg/discovery -run TestRoutingTable
go test ./pkg/discovery -run TestFindClosest
```

**Dependencies**: TICKET-003, TICKET-004

---

### Epic 3: DHT Protocol Messages

#### TICKET-006: Define DHT Message Wire Protocol
**Priority**: P0 (Blocker)
**Estimate**: 2 days
**Assignee**: [TBD]

**Description**:
Define binary wire protocol for DHT messages (PING, FIND_NODE, STORE, etc.)

**Acceptance Criteria**:
- [ ] `DHTMessage` struct with type, request ID, sender ID, payload, signature
- [ ] Message types defined: PING, PONG, FIND_NODE, FOUND_NODES, STORE, STORE_ACK, FIND_VALUE, FOUND_VALUE
- [ ] Binary serialization/deserialization (gob, protobuf, or custom)
- [ ] ML-DSA-87 signature over message fields
- [ ] Unit tests: Serialize/deserialize all message types
- [ ] Unit tests: Signature verification on messages
- [ ] Benchmark: Serialization <100μs per message

**Implementation Details**:
```go
// File: pkg/discovery/protocol.go

const (
    MSG_PING        = 0x01
    MSG_PONG        = 0x02
    MSG_FIND_NODE   = 0x03
    MSG_FOUND_NODES = 0x04
    MSG_STORE       = 0x05
    MSG_STORE_ACK   = 0x06
    MSG_FIND_VALUE  = 0x07
    MSG_FOUND_VALUE = 0x08
)

type DHTMessage struct {
    Type      uint8      // Message type
    RequestID [16]byte   // Nonce for request/response matching
    SenderID  PeerID     // Sender's PeerID
    Payload   []byte     // Type-specific payload
    Signature []byte     // ML-DSA-87 signature
}

type FindNodePayload struct {
    TargetID PeerID
}

type FoundNodesPayload struct {
    Peers []PeerInfo // Up to k=20 peers
}

type StorePayload struct {
    Key       PeerID
    Value     []byte
    TTL       time.Duration
    Timestamp time.Time
}

func (msg *DHTMessage) Serialize() ([]byte, error) {
    // Binary serialization
}

func DeserializeDHTMessage(data []byte) (*DHTMessage, error) {
    // Binary deserialization
}

func (msg *DHTMessage) Sign(privateKey []byte) error {
    // Sign message with ML-DSA-87
}

func (msg *DHTMessage) Verify(publicKey []byte) error {
    // Verify ML-DSA-87 signature
}
```

**Testing**:
```bash
go test ./pkg/discovery -run TestDHTMessageProtocol
go test ./pkg/discovery -bench BenchmarkMessageSerialization
```

**Dependencies**: TICKET-001, TICKET-002

---

#### TICKET-007: Implement DHT Message Handler
**Priority**: P0 (Blocker)
**Estimate**: 3 days
**Assignee**: [TBD]

**Description**:
Implement message handler for processing incoming DHT messages.

**Acceptance Criteria**:
- [ ] UDP listener for DHT messages (default port: 8443)
- [ ] Dispatch messages to handlers by type
- [ ] Rate limiting: Max 100 messages/second per peer
- [ ] Request/response matching via RequestID
- [ ] Signature verification before processing
- [ ] Reject invalid/malformed messages
- [ ] Unit tests: Send/receive all message types
- [ ] Unit tests: Rate limiting blocks excessive requests
- [ ] Unit tests: Invalid signatures rejected

**Implementation Details**:
```go
// File: pkg/discovery/message_handler.go

type MessageHandler struct {
    routingTable *RoutingTable
    privateKey   []byte        // ML-DSA-87 private key
    publicKey    []byte        // ML-DSA-87 public key
    peerID       PeerID
    conn         *net.UDPConn
    rateLimiter  *RateLimiter
    pendingReqs  sync.Map      // RequestID -> chan Response
}

func (h *MessageHandler) Start(port int) error {
    addr := &net.UDPAddr{Port: port}
    conn, err := net.ListenUDP("udp", addr)
    if err != nil {
        return err
    }
    h.conn = conn

    go h.receiveLoop()
    return nil
}

func (h *MessageHandler) receiveLoop() {
    buf := make([]byte, 4096)
    for {
        n, remoteAddr, err := h.conn.ReadFromUDP(buf)
        if err != nil {
            continue
        }

        msg, err := DeserializeDHTMessage(buf[:n])
        if err != nil {
            continue
        }

        // Rate limiting
        if !h.rateLimiter.Allow(msg.SenderID) {
            continue
        }

        // Verify signature
        // (Need to look up public key from routing table or message)

        // Dispatch message
        go h.handleMessage(msg, remoteAddr)
    }
}

func (h *MessageHandler) handleMessage(msg *DHTMessage, addr *net.UDPAddr) {
    switch msg.Type {
    case MSG_PING:
        h.handlePing(msg, addr)
    case MSG_FIND_NODE:
        h.handleFindNode(msg, addr)
    case MSG_STORE:
        h.handleStore(msg, addr)
    // ... other handlers
    }
}
```

**Testing**:
```bash
go test ./pkg/discovery -run TestMessageHandler
```

**Dependencies**: TICKET-005, TICKET-006

---

## Sprint 1: DHT Operations (Weeks 3-4)

### Epic 4: Core DHT Operations

#### TICKET-008: Implement PING Operation
**Priority**: P0 (Blocker)
**Estimate**: 2 days
**Assignee**: [TBD]

**Description**:
Implement PING/PONG for peer liveness checking.

**Acceptance Criteria**:
- [ ] `Ping(peer PeerInfo) (bool, error)` sends PING, waits for PONG
- [ ] 2-second timeout for PONG response
- [ ] Update routing table last seen time on successful PONG
- [ ] Handle failed PINGs (increment fail counter)
- [ ] Unit tests: Successful PING/PONG exchange
- [ ] Unit tests: Timeout on no response
- [ ] Integration test: 3-node network, all nodes PING each other

**Implementation Details**:
```go
// File: pkg/discovery/ping.go

func (dht *DHT) Ping(peer PeerInfo) (bool, error) {
    // Generate request ID
    requestID := generateRequestID()

    // Create PING message
    msg := &DHTMessage{
        Type:      MSG_PING,
        RequestID: requestID,
        SenderID:  dht.peerID,
        Payload:   nil,
    }
    msg.Sign(dht.privateKey)

    // Send PING
    responseChan := make(chan *DHTMessage, 1)
    dht.pendingRequests[requestID] = responseChan

    if err := dht.sendMessage(msg, peer.Address); err != nil {
        return false, err
    }

    // Wait for PONG (2-second timeout)
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    select {
    case <-responseChan:
        // Update routing table
        dht.routingTable.UpdateLastSeen(peer.PeerID, time.Now())
        return true, nil
    case <-ctx.Done():
        return false, ErrTimeout
    }
}

func (h *MessageHandler) handlePing(msg *DHTMessage, addr *net.UDPAddr) {
    // Send PONG response
    pong := &DHTMessage{
        Type:      MSG_PONG,
        RequestID: msg.RequestID,
        SenderID:  h.peerID,
        Payload:   nil,
    }
    pong.Sign(h.privateKey)

    h.sendMessage(pong, addr.String())
}
```

**Testing**:
```bash
go test ./pkg/discovery -run TestPing
```

**Dependencies**: TICKET-005, TICKET-007

---

#### TICKET-009: Implement FIND_NODE Iterative Lookup
**Priority**: P0 (Blocker)
**Estimate**: 5 days
**Assignee**: [TBD]

**Description**:
Implement iterative FIND_NODE operation (core DHT lookup algorithm).

**Acceptance Criteria**:
- [ ] `FindNode(target PeerID, timeout) []PeerInfo` returns k closest peers
- [ ] Iterative algorithm with α=3 parallel requests
- [ ] Converges when no closer peers found
- [ ] 5-second overall timeout
- [ ] Unit tests: Find node in 3-node network
- [ ] Integration test: Find node in 10-node network (<500ms)
- [ ] Integration test: Lookup converges in O(log N) hops

**Implementation Details**:
```go
// File: pkg/discovery/find_node.go

func (dht *DHT) FindNode(target PeerID, timeout time.Duration) []PeerInfo {
    α := 3  // Parallelism factor
    k := 20 // Result count

    // Start with closest known peers
    candidates := dht.routingTable.FindClosest(target, k)
    queried := make(map[PeerID]bool)
    results := make([]PeerInfo, 0, k)

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    for {
        // Select α closest unqueried peers
        toQuery := selectClosestUnqueried(candidates, queried, α, target)
        if len(toQuery) == 0 {
            break
        }

        // Query in parallel
        var wg sync.WaitGroup
        responsesChan := make(chan []PeerInfo, α)

        for _, peer := range toQuery {
            wg.Add(1)
            go func(p PeerInfo) {
                defer wg.Done()
                peers, err := dht.sendFindNode(ctx, p, target)
                if err == nil {
                    responsesChan <- peers
                }
                queried[p.PeerID] = true
            }(peer)
        }

        go func() {
            wg.Wait()
            close(responsesChan)
        }()

        // Collect responses
        for peers := range responsesChan {
            candidates = mergePeers(candidates, peers)
        }

        // Check convergence
        if hasConverged(results, candidates, k) {
            break
        }

        // Update results
        results = selectClosest(candidates, target, k)
    }

    return results
}
```

**Testing**:
```bash
go test ./pkg/discovery -run TestFindNode
go test ./pkg/discovery -run TestFindNodeConvergence
```

**Dependencies**: TICKET-005, TICKET-007, TICKET-008

---

#### TICKET-010: Implement STORE Operation
**Priority**: P1 (High)
**Estimate**: 3 days
**Assignee**: [TBD]

**Description**:
Implement STORE operation to store peer metadata in DHT.

**Acceptance Criteria**:
- [ ] `StoreSelf() error` stores own metadata at k closest peers
- [ ] `Store(key, value, TTL)` generic store operation
- [ ] Metadata includes: PeerID, address, public key, capabilities, TTL
- [ ] Signature over metadata for verification
- [ ] Unit tests: Store at 3 peers, verify success
- [ ] Integration test: Store and retrieve across 10-node network

**Implementation Details**:
```go
// File: pkg/discovery/store.go

type PeerMetadata struct {
    PeerID       PeerID
    Address      string
    PublicKey    []byte
    Capabilities []string
    TTL          time.Duration
    Timestamp    time.Time
    Signature    []byte
}

func (dht *DHT) StoreSelf() error {
    // Find k closest nodes to own PeerID
    closestPeers := dht.FindNode(dht.peerID, 5*time.Second)

    // Build metadata
    metadata := &PeerMetadata{
        PeerID:       dht.peerID,
        Address:      dht.localAddress,
        PublicKey:    dht.publicKey,
        Capabilities: []string{"peer"},
        TTL:          24 * time.Hour,
        Timestamp:    time.Now(),
    }
    metadata.Sign(dht.privateKey)

    // Store at each peer
    successCount := 0
    for _, peer := range closestPeers {
        if dht.sendStore(peer, metadata) {
            successCount++
        }
    }

    // Success if majority stored
    return successCount >= len(closestPeers)/2
}
```

**Testing**:
```bash
go test ./pkg/discovery -run TestStore
```

**Dependencies**: TICKET-009

---

#### TICKET-011: Implement FIND_VALUE Operation
**Priority**: P1 (High)
**Estimate**: 3 days
**Assignee**: [TBD]

**Description**:
Implement FIND_VALUE operation to retrieve peer metadata from DHT.

**Acceptance Criteria**:
- [ ] `FindValue(target PeerID) (*PeerMetadata, error)` retrieves metadata
- [ ] Check local cache first (10-minute TTL)
- [ ] Falls back to iterative DHT lookup if not cached
- [ ] Validates metadata signature
- [ ] Caches successful lookups
- [ ] Unit tests: Find stored value in 3-node network
- [ ] Integration test: Find value across 10-node network (<500ms)

**Implementation Details**:
```go
// File: pkg/discovery/find_value.go

func (dht *DHT) FindValue(target PeerID) (*PeerMetadata, error) {
    // Check cache first
    if cached := dht.cache.Get(target); cached != nil {
        return cached, nil
    }

    // Find k closest peers
    candidates := dht.FindNode(target, 5*time.Second)

    // Query each peer for value
    for _, peer := range candidates {
        metadata, err := dht.sendFindValue(peer, target)
        if err == nil && validateMetadata(metadata) {
            // Cache and return
            dht.cache.Set(target, metadata, 10*time.Minute)
            return metadata, nil
        }
    }

    return nil, ErrPeerNotFound
}
```

**Testing**:
```bash
go test ./pkg/discovery -run TestFindValue
```

**Dependencies**: TICKET-010

---

### Epic 5: Local Test Network

#### TICKET-012: Create 3-Node Local Test Network
**Priority**: P0 (Blocker)
**Estimate**: 3 days
**Assignee**: [TBD]

**Description**:
Build local 3-node test network on localhost for DHT validation.

**Acceptance Criteria**:
- [ ] 3 nodes running on localhost with different ports (8443, 8444, 8445)
- [ ] Each node has unique ML-DSA-87 keypair and PeerID
- [ ] Nodes bootstrap from each other
- [ ] All nodes discover each other via DHT
- [ ] Routing tables converge in <60 seconds
- [ ] PING between all node pairs successful
- [ ] Docker Compose configuration for reproducible setup
- [ ] Integration test script: Start network, validate convergence, shutdown

**Implementation Details**:
```yaml
# docker-compose-local-test.yml
version: '3.8'
services:
  node1:
    build: .
    command: ["--port", "8443", "--bootstrap", "node2:8444,node3:8445"]
    ports:
      - "8443:8443"
    networks:
      - shadowmesh-test

  node2:
    build: .
    command: ["--port", "8444", "--bootstrap", "node1:8443,node3:8445"]
    ports:
      - "8444:8444"
    networks:
      - shadowmesh-test

  node3:
    build: .
    command: ["--port", "8445", "--bootstrap", "node1:8443,node2:8444"]
    ports:
      - "8445:8445"
    networks:
      - shadowmesh-test

networks:
  shadowmesh-test:
    driver: bridge
```

**Testing**:
```bash
# Start test network
docker-compose -f docker-compose-local-test.yml up -d

# Run integration tests
go test ./test/integration -run TestLocalNetwork

# Shutdown
docker-compose -f docker-compose-local-test.yml down
```

**Dependencies**: All Sprint 0-1 tickets

---

## Summary

### Sprint 0 (Weeks 1-2)
- **11 tickets** covering PeerID, routing table, and message protocol
- **Estimated effort**: ~20 developer-days
- **Key deliverables**: Core DHT data structures, message protocol

### Sprint 1 (Weeks 3-4)
- **4 tickets** covering DHT operations and local testing
- **Estimated effort**: ~16 developer-days
- **Key deliverables**: Working FIND_NODE, STORE, FIND_VALUE, 3-node test network

### Total Phase 1
- **15 tickets**
- **~36 developer-days** (assuming 1 developer full-time)
- **Timeline**: 4 weeks (with some parallelization possible)

### Critical Path
```
TICKET-001 → TICKET-002 → TICKET-003 → TICKET-004 → TICKET-005
     ↓
TICKET-006 → TICKET-007 → TICKET-008 → TICKET-009
     ↓
TICKET-010 → TICKET-011 → TICKET-012
```

### Parallel Work Opportunities
- TICKET-001 and TICKET-006 can be done in parallel (different developers)
- TICKET-008, TICKET-010, TICKET-011 can be parallelized after TICKET-009

---

## Next Steps

1. **Assign tickets** to development team
2. **Set up project board** (GitHub Projects, Jira, etc.)
3. **Schedule daily standups** during Sprint 0-1
4. **Begin TICKET-001** immediately (PeerID generation)
5. **Weekly demos** to stakeholders

---

**Document Control**
- Version: 1.0
- Created: November 10, 2025
- Author: Winston (Architect)
- Status: Ready for Assignment
