# ShadowMesh Kademlia DHT Architecture Specification

**Version**: 1.0
**Date**: November 10, 2025
**Author**: Winston (Architect)
**Status**: Specification for v0.2.0-alpha (First Standalone Release)

---

## Executive Summary

This specification defines the Kademlia Distributed Hash Table (DHT) architecture for ShadowMesh v0.2.0-alpha, enabling **fully decentralized peer discovery** and eliminating the centralized discovery server dependency that currently blocks standalone operation.

**Key Outcomes**:
- **Zero central dependencies** - nodes operate autonomously from first boot
- **Cryptographically verifiable peer identity** using ML-DSA-87 public keys
- **Sub-second peer discovery** via DHT iterative lookup
- **Fault-tolerant routing** with automatic network partition recovery
- **Foundation for v1.0.0** production release

---

## Current Problem

**v0.1.0-alpha (v11)** demonstrated excellent performance (28.3 Mbps, video streaming, 45% faster than Tailscale) but suffers from a critical architectural flaw:

```
❌ CENTRALIZED DISCOVERY SERVER
   • HTTP API: 209.151.148.121:8080
   • POST /register - Register peer
   • GET /peers - Retrieve peer list
   • NOW SHUT DOWN → Network inoperable
```

**Consequences**:
- Cannot operate standalone
- Single point of failure
- Contradicts DPN (Decentralized Private Network) vision
- Operational costs for infrastructure
- No censorship resistance

---

## Solution: Kademlia DHT

**Kademlia** is a proven distributed hash table protocol used by:
- **BitTorrent** - 30+ million concurrent nodes
- **IPFS** (libp2p) - Decentralized storage
- **Ethereum** - Node discovery layer

**Why Kademlia?**:
- ✅ Logarithmic lookup complexity: O(log N) hops for N nodes
- ✅ Symmetric architecture: Every node is equal (no super-nodes)
- ✅ Automatic routing table convergence via XOR distance metric
- ✅ Fault tolerance: Multiple redundant paths to any peer
- ✅ Built-in peer liveness monitoring

---

## Architecture Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                      ShadowMesh v0.2.0-alpha                        │
│                  (Kademlia DHT + PQC + UDP/QUIC)                    │
└────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────┐
│  Layer 1: Application                                               │
│  • CLI commands: connect, disconnect, status                        │
│  • Configuration: bootstrap nodes, network settings                 │
└────────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────────┐
│  Layer 2: Kademlia DHT (Peer Discovery)                            │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  PeerID Generation                                            │  │
│  │  • SHA256(ML-DSA-87 Public Key) → 256-bit PeerID            │  │
│  │  • Cryptographically verifiable identity                     │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Routing Table (160 k-buckets, k=20)                         │  │
│  │  • XOR distance metric for peer organization                 │  │
│  │  • LRU eviction policy per k-bucket                          │  │
│  │  • Automatic table refresh every 1 hour                      │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  DHT Operations                                               │  │
│  │  • FIND_NODE(target) → Iterative lookup (α=3 parallel)      │  │
│  │  • STORE(key, value, TTL=24h) → Store peer metadata         │  │
│  │  • FIND_VALUE(key) → Retrieve peer metadata                 │  │
│  │  • PING → Peer liveness check (every 15 min)                │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────────┐
│  Layer 3: Post-Quantum Cryptography                                │
│  • ML-KEM-1024 (Kyber) - Key Exchange                             │
│  • ML-DSA-87 (Dilithium) - Peer Authentication & Signatures       │
│  • ChaCha20-Poly1305 - Symmetric Encryption                       │
│  • Hybrid Mode: Classical (X25519, Ed25519) + PQC                 │
└────────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────────┐
│  Layer 4: Transport (UDP → QUIC migration path)                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  v0.2.0-alpha: UDP Transport (proven in v11)                 │  │
│  │  • Frame-based protocol                                      │  │
│  │  • Direct peer-to-peer                                       │  │
│  │  • Encrypted payload with ChaCha20-Poly1305                  │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  v0.3.0+: QUIC Transport (future migration)                  │  │
│  │  • Stream-based protocol                                     │  │
│  │  • Better NAT traversal                                      │  │
│  │  • Built-in congestion control                               │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────────┐
│  Layer 5: Networking (Layer 3 - TUN Device)                        │
│  • Virtual network interface (10.10.x.x addresses)                 │
│  • IP packet capture/injection                                     │
│  • Full network stack support (TCP, UDP, ICMP)                     │
└────────────────────────────────────────────────────────────────────┘
```

---

## Core Component: Kademlia DHT

### 1. PeerID Generation

**Design Decision**: Derive PeerID from post-quantum signature public key

```go
// PeerID generation from ML-DSA-87 public key
func GeneratePeerID(mldsaPublicKey []byte) PeerID {
    hash := sha256.Sum256(mldsaPublicKey)
    return PeerID(hash[:]) // 256-bit PeerID
}

// PeerID structure
type PeerID [32]byte // 256 bits (32 bytes)

// Verification: Peer proves ownership by signing challenge
func VerifyPeerOwnership(peerID PeerID, publicKey []byte, signature []byte, challenge []byte) bool {
    // 1. Verify PeerID matches public key
    derivedID := GeneratePeerID(publicKey)
    if derivedID != peerID {
        return false
    }

    // 2. Verify signature on challenge
    return VerifyMLDSA87Signature(publicKey, challenge, signature)
}
```

**Properties**:
- ✅ **Cryptographically verifiable**: Peer must prove ownership via signature
- ✅ **Collision-resistant**: SHA256 provides 2^128 security against birthday attacks
- ✅ **Quantum-safe**: Derived from post-quantum public key
- ✅ **Sybil-resistant**: Creating fake peers requires valid ML-DSA-87 keypairs

**Rationale**: Using the PQ signature key (not encryption key) allows peers to prove identity through signed challenges, preventing identity spoofing.

---

### 2. Routing Table Structure

**Kademlia k-bucket design** with XOR distance metric:

```
Routing Table:
┌─────────────────────────────────────────────────────────────┐
│ k-bucket[0]:   Distance [2^255, 2^256)  - Furthest peers   │
│ k-bucket[1]:   Distance [2^254, 2^255)                      │
│ k-bucket[2]:   Distance [2^253, 2^254)                      │
│ ...                                                          │
│ k-bucket[254]: Distance [2^1, 2^2)                          │
│ k-bucket[255]: Distance [2^0, 2^1)      - Closest peers    │
└─────────────────────────────────────────────────────────────┘

Each k-bucket stores up to k=20 peers (configurable)
```

**Implementation**:

```go
type RoutingTable struct {
    localPeerID  PeerID
    kBuckets     [256]*KBucket  // 256 buckets for 256-bit PeerID space
    k            int            // Max peers per bucket (default: 20)
    mutex        sync.RWMutex
}

type KBucket struct {
    peers        []PeerInfo     // Up to k peers
    lastUpdated  time.Time
    mutex        sync.RWMutex
}

type PeerInfo struct {
    PeerID       PeerID
    Address      string         // "IP:port"
    PublicKey    []byte         // ML-DSA-87 public key
    LastSeen     time.Time
    Capabilities []string       // ["relay", "exit-node", "bootstrap"]
}

// XOR distance metric
func XORDistance(id1, id2 PeerID) *big.Int {
    distance := new(big.Int)
    distance.SetBytes(id1[:])
    distance.Xor(distance, new(big.Int).SetBytes(id2[:]))
    return distance
}

// Find k-bucket index for peer
func (rt *RoutingTable) BucketIndex(peerID PeerID) int {
    distance := XORDistance(rt.localPeerID, peerID)
    // Count leading zeros to determine bucket
    leadingZeros := distance.BitLen()
    return 256 - leadingZeros
}
```

**Routing Table Maintenance**:
- **LRU Eviction**: When bucket full, replace least-recently-seen peer
- **Liveness Checks**: PING all peers every 15 minutes
- **Table Refresh**: Re-query all buckets every 1 hour
- **Stale Peer Removal**: Evict peers that fail 3 consecutive PINGs

---

### 3. DHT Operations

#### 3.1. FIND_NODE - Iterative Lookup

**Goal**: Find k closest peers to target PeerID

**Algorithm** (α=3 parallel requests, k=20 results):

```
1. Start with k closest known peers to target
2. Send FIND_NODE(target) to α peers in parallel
3. Each peer responds with k closest peers from their routing table
4. Add new peers to candidate list
5. Mark queried peers as "queried"
6. Select α closest unqueried peers from candidate list
7. Repeat until:
   - No closer peers found, OR
   - Top k peers all queried, OR
   - Timeout (5 seconds)
8. Return k closest peers found
```

**Implementation**:

```go
func (dht *DHT) FindNode(target PeerID, timeout time.Duration) []PeerInfo {
    // Iterative lookup with α=3 concurrent requests
    α := 3
    k := 20

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
            break // No more peers to query
        }

        // Query in parallel
        responses := queryPeersInParallel(ctx, toQuery, target)

        // Merge responses into candidate list
        for _, response := range responses {
            candidates = mergePeers(candidates, response.Peers)
            queried[response.FromPeer] = true
        }

        // Check termination conditions
        if hasConverged(results, candidates, k) {
            break
        }
    }

    // Return k closest peers
    return selectClosest(candidates, target, k)
}
```

**Performance**:
- **Latency**: O(log N) hops for N nodes
- **Target**: <500ms for 100,000-node network
- **Parallelism**: α=3 concurrent requests reduces latency

#### 3.2. STORE - Store Peer Metadata

**Goal**: Store peer metadata (IP, port, public key) in DHT

**Strategy**: Store at k closest nodes to peer's own PeerID

```go
func (dht *DHT) StoreSelf() error {
    // Find k closest nodes to own PeerID
    closestPeers := dht.FindNode(dht.localPeerID, 5*time.Second)

    // Build metadata record
    metadata := PeerMetadata{
        PeerID:       dht.localPeerID,
        Address:      dht.localAddress,
        PublicKey:    dht.mldsaPublicKey,
        Capabilities: []string{"peer"},
        TTL:          24 * time.Hour,
        Timestamp:    time.Now(),
        Signature:    dht.SignMetadata(), // Prove ownership
    }

    // Send STORE to k closest peers
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

**Data Structure**:

```go
type PeerMetadata struct {
    PeerID       PeerID
    Address      string         // "IP:port"
    PublicKey    []byte         // ML-DSA-87 public key
    Capabilities []string       // ["peer", "relay", "exit-node", "bootstrap"]
    TTL          time.Duration  // 24 hours default
    Timestamp    time.Time
    Signature    []byte         // ML-DSA-87 signature over above fields
}
```

**Validation**:
- Verify PeerID matches public key: `GeneratePeerID(PublicKey) == PeerID`
- Verify signature over metadata fields
- Reject if timestamp too old (>5 minutes drift)
- Reject if TTL exceeds maximum (48 hours)

#### 3.3. FIND_VALUE - Retrieve Peer Metadata

**Goal**: Retrieve metadata for target PeerID

**Algorithm**: Similar to FIND_NODE but returns metadata if found

```go
func (dht *DHT) FindValue(target PeerID) (*PeerMetadata, error) {
    // Check local cache first
    if cached := dht.cache.Get(target); cached != nil {
        return cached, nil
    }

    // Iterative lookup with α=3
    candidates := dht.routingTable.FindClosest(target, 20)

    for _, peer := range candidates {
        response := dht.sendFindValue(peer, target)
        if response.Found {
            // Validate and cache metadata
            if validateMetadata(response.Metadata) {
                dht.cache.Set(target, response.Metadata, 10*time.Minute)
                return response.Metadata, nil
            }
        }
    }

    return nil, ErrPeerNotFound
}
```

**Caching Strategy**:
- Cache successful lookups for 10 minutes
- LRU eviction (max 10,000 entries)
- Invalidate on failed connection attempts

#### 3.4. PING - Liveness Check

**Goal**: Verify peer is still online

```go
func (dht *DHT) Ping(peer PeerInfo) bool {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    // Send PING message
    response, err := dht.sendPing(ctx, peer)
    if err != nil {
        return false
    }

    // Update last seen time
    dht.routingTable.UpdateLastSeen(peer.PeerID, time.Now())
    return response.Success
}

// Background liveness checker
func (dht *DHT) startLivenessChecker() {
    ticker := time.NewTicker(15 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        peers := dht.routingTable.AllPeers()
        for _, peer := range peers {
            go func(p PeerInfo) {
                if !dht.Ping(p) {
                    dht.handleFailedPing(p)
                }
            }(peer)
        }
    }
}

func (dht *DHT) handleFailedPing(peer PeerInfo) {
    failCount := dht.incrementFailCount(peer.PeerID)
    if failCount >= 3 {
        dht.routingTable.RemovePeer(peer.PeerID)
        log.Printf("Removed dead peer: %s", peer.PeerID)
    }
}
```

---

### 4. Bootstrap Process

**Challenge**: New node joining network needs initial peers

**Solution**: Hardcoded bootstrap nodes + iterative expansion

```
┌────────────────────────────────────────────────────────────┐
│  Bootstrap Flow                                             │
└────────────────────────────────────────────────────────────┘

1. Node starts with empty routing table
   ↓
2. Connect to 3-5 hardcoded bootstrap nodes
   Bootstrap nodes: Known long-running peers
   ↓
3. FIND_NODE(self) to bootstrap nodes
   ↓
4. Receive k peers close to own PeerID
   ↓
5. Add peers to routing table
   ↓
6. FIND_NODE(random IDs) to populate all buckets
   ↓
7. STORE(self) at k closest peers
   ↓
8. Routing table converges (target: <60 seconds)
   ↓
9. Node fully integrated into DHT
```

**Bootstrap Node Configuration**:

```yaml
# config.yaml
bootstrap_nodes:
  - peer_id: "2f8a3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b"
    address: "bootstrap1.shadowmesh.net:9443"
    public_key: "base64_encoded_mldsa87_pubkey"

  - peer_id: "3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b"
    address: "bootstrap2.shadowmesh.net:9443"
    public_key: "base64_encoded_mldsa87_pubkey"

  - peer_id: "4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5"
    address: "bootstrap3.shadowmesh.net:9443"
    public_key: "base64_encoded_mldsa87_pubkey"
```

**Bootstrap Node Requirements**:
- 99.9% uptime (3 nodes provide redundancy)
- Static IP addresses with DNS names
- Pre-registered ML-DSA-87 keypairs
- Deployed across geographic regions (US, EU, Asia)

**Graceful Degradation**:
- If 0 bootstrap nodes reachable → Error: "Cannot connect to network"
- If 1-2 bootstrap nodes reachable → Warning: "Degraded connectivity"
- If 3+ bootstrap nodes reachable → Normal operation

**Future**: Peer exchange protocol allows nodes to bootstrap from any known peer, reducing bootstrap node dependency.

---

### 5. DHT Message Protocol

**Wire Protocol**: Binary messages over UDP (v0.2.0) or QUIC (v0.3.0+)

```go
// Message types
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

// Message structure (simplified)
type DHTMessage struct {
    Type      uint8      // Message type
    RequestID [16]byte   // Random nonce for request/response matching
    SenderID  PeerID     // Sender's PeerID
    Payload   []byte     // Type-specific payload
    Signature []byte     // ML-DSA-87 signature over above fields
}

// FIND_NODE payload
type FindNodePayload struct {
    TargetID PeerID
}

// FOUND_NODES payload
type FoundNodesPayload struct {
    Peers []PeerInfo // Up to k=20 peers
}

// STORE payload
type StorePayload struct {
    Metadata PeerMetadata
}
```

**Security**:
- All messages signed with ML-DSA-87
- Verify signature before processing
- Rate limiting: Max 100 messages/second per peer
- Reject messages with invalid PeerID → PublicKey mapping

---

## Integration with Existing Components

### 1. PQC Handshake Flow

**After DHT peer discovery, establish encrypted tunnel:**

```
┌────────────────────────────────────────────────────────────┐
│  Connection Establishment Flow                              │
└────────────────────────────────────────────────────────────┘

1. DHT Lookup: FIND_VALUE(target_peer_id)
   ↓
2. Retrieve: IP address, port, ML-DSA-87 public key
   ↓
3. Initiate UDP connection to peer
   ↓
4. PQC Handshake:
   a. Generate ephemeral ML-KEM-1024 keypair
   b. Send key encapsulation request
   c. Peer encapsulates shared secret
   d. Both derive symmetric key via HKDF
   ↓
5. Authenticate with ML-DSA-87 signatures
   ↓
6. Begin encrypted traffic with ChaCha20-Poly1305
```

**Code Integration**:

```go
// Combine DHT + PQC
func (node *Node) ConnectToPeer(targetPeerID PeerID) error {
    // 1. DHT lookup
    metadata, err := node.dht.FindValue(targetPeerID)
    if err != nil {
        return fmt.Errorf("peer not found in DHT: %w", err)
    }

    // 2. Verify peer identity
    if !node.crypto.VerifyPeerID(metadata.PeerID, metadata.PublicKey) {
        return errors.New("PeerID mismatch")
    }

    // 3. Establish UDP connection
    conn, err := net.Dial("udp", metadata.Address)
    if err != nil {
        return fmt.Errorf("connection failed: %w", err)
    }

    // 4. PQC handshake
    sharedSecret, err := node.crypto.PerformMLKEMHandshake(conn, metadata.PublicKey)
    if err != nil {
        return fmt.Errorf("handshake failed: %w", err)
    }

    // 5. Derive symmetric key
    sessionKey := node.crypto.DeriveSessionKey(sharedSecret)

    // 6. Create encrypted tunnel
    tunnel := NewEncryptedTunnel(conn, sessionKey)
    node.addTunnel(targetPeerID, tunnel)

    log.Printf("Connected to peer: %s", targetPeerID)
    return nil
}
```

### 2. TUN Device Integration

**Traffic routing flow:**

```
Application → TUN Device → ShadowMesh Daemon → DHT Lookup → Encrypted Tunnel → Remote Peer
```

**Implementation**:

```go
func (node *Node) handleTUNPacket(packet []byte) {
    // 1. Parse destination IP from packet header
    destIP := parseDestinationIP(packet)

    // 2. Map IP to PeerID (via local routing table)
    peerID, found := node.ipToPeerID[destIP]
    if !found {
        log.Printf("Unknown destination: %s", destIP)
        return
    }

    // 3. Check for existing tunnel
    tunnel := node.getTunnel(peerID)
    if tunnel == nil {
        // 4. Establish tunnel via DHT
        if err := node.ConnectToPeer(peerID); err != nil {
            log.Printf("Failed to connect: %v", err)
            return
        }
        tunnel = node.getTunnel(peerID)
    }

    // 5. Encrypt and send packet
    tunnel.SendPacket(packet)
}
```

---

## Migration Path: v11 → v0.2.0-alpha

### Phase 1: DHT Implementation (Weeks 1-4)

**Week 1-2: Core DHT**
```
[ ] Implement PeerID generation from ML-DSA-87
[ ] Implement routing table with 256 k-buckets
[ ] Implement XOR distance metric
[ ] Implement FIND_NODE iterative lookup
[ ] Local 3-node test network
```

**Week 3-4: DHT Operations**
```
[ ] Implement STORE/FIND_VALUE operations
[ ] Implement PING liveness checks
[ ] Implement bootstrap process
[ ] 5-10 node test network
[ ] Peer discovery latency <100ms
```

### Phase 2: Integration (Weeks 5-6)

**Week 5: v11 Integration**
```
[ ] Replace centralized discovery with DHT
[ ] Integrate DHT lookup before PQC handshake
[ ] Update peer connection flow
[ ] End-to-end test: DHT → PQC → TUN
```

**Week 6: Testing & Validation**
```
[ ] Multi-node mesh testing (5 nodes)
[ ] Peer discovery success rate >95%
[ ] Performance regression tests (maintain 28+ Mbps)
[ ] Network partition recovery tests
```

### Phase 3: Standalone Release (Week 7-8)

**Week 7: Release Preparation**
```
[ ] Deploy bootstrap nodes (3 locations)
[ ] Create installation packages (Linux: deb, rpm, arch)
[ ] Write user documentation
[ ] Create quick start guide
```

**Week 8: Alpha Release**
```
[ ] v0.2.0-alpha release with DHT
[ ] Standalone operation validated
[ ] Community testing phase
[ ] Gather feedback for v0.3.0
```

---

## Standalone Release Criteria (v0.2.0-alpha)

### Functional Requirements

✅ **Zero Central Dependencies**
- [ ] Node starts without centralized discovery server
- [ ] Connects to 3+ bootstrap nodes
- [ ] Routing table converges in <60 seconds
- [ ] Peer discovery via DHT successful

✅ **Performance Targets**
- [ ] Throughput: ≥25 Mbps (maintain v11 performance)
- [ ] Latency: <50ms added overhead
- [ ] Packet loss: <5%
- [ ] DHT lookup: <500ms

✅ **Reliability**
- [ ] Peer discovery success rate: >95%
- [ ] Network partition recovery: <5 minutes
- [ ] Uptime: 24+ hour stress test without crashes

✅ **Security**
- [ ] PeerID verification working
- [ ] ML-DSA-87 signature validation
- [ ] No unauthenticated peer connections
- [ ] DHT message rate limiting

### Testing Checklist

**Unit Tests**
- [ ] PeerID generation and verification
- [ ] XOR distance calculations
- [ ] Routing table operations (add, remove, find)
- [ ] DHT message serialization/deserialization

**Integration Tests**
- [ ] 3-node local test network
- [ ] 5-node distributed test network
- [ ] Bootstrap process from cold start
- [ ] Peer discovery and connection establishment
- [ ] Traffic routing through mesh

**Performance Tests**
- [ ] iperf3 throughput tests (≥25 Mbps target)
- [ ] ping latency tests (<50ms target)
- [ ] DHT lookup latency (<500ms target)
- [ ] Memory usage (<500 MB per node)

**Stress Tests**
- [ ] 24-hour uptime test
- [ ] Peer churn test (nodes joining/leaving)
- [ ] Network partition recovery
- [ ] 10+ concurrent connections per node

### Release Artifacts

**Binaries**
- [ ] Linux: shadowmesh-v0.2.0-alpha-linux-amd64
- [ ] Linux: shadowmesh-v0.2.0-alpha-linux-arm64
- [ ] macOS: shadowmesh-v0.2.0-alpha-darwin-arm64

**Packages**
- [ ] Debian/Ubuntu: shadowmesh_0.2.0-alpha_amd64.deb
- [ ] RHEL/Fedora: shadowmesh-0.2.0-alpha.x86_64.rpm
- [ ] Arch: shadowmesh-0.2.0-alpha-x86_64.pkg.tar.zst

**Documentation**
- [ ] README.md (updated with DHT architecture)
- [ ] INSTALL.md (installation instructions)
- [ ] QUICKSTART.md (5-minute setup guide)
- [ ] ARCHITECTURE.md (this document)

**Infrastructure**
- [ ] 3 bootstrap nodes deployed and operational
- [ ] DNS records: bootstrap{1,2,3}.shadowmesh.net
- [ ] Monitoring dashboards for bootstrap nodes

---

## Future Work (v0.3.0+)

### QUIC Migration
- Replace UDP transport with QUIC
- Better NAT traversal via QUIC connection migration
- Stream multiplexing for multiple tunnels

### Advanced DHT Features
- DHT replication factor k=3 (store at 3 nodes for redundancy)
- Republish peer metadata every 12 hours (maintain availability)
- Peer exchange protocol (reduce bootstrap dependency)

### Performance Optimization
- Zero-copy packet handling
- SIMD acceleration for ChaCha20-Poly1305
- Parallel DHT lookups (multiple targets simultaneously)

### Security Enhancements
- Eclipse attack mitigation (diverse routing table)
- Sybil attack defense (stake-based admission)
- Traffic analysis resistance (dummy traffic padding)

---

## Conclusion

This Kademlia DHT architecture provides a solid foundation for ShadowMesh's transition to **fully decentralized operation**. By eliminating the centralized discovery server, v0.2.0-alpha will achieve:

✅ **True DPN** - No single point of failure
✅ **Standalone** - Operates from first boot
✅ **Scalable** - O(log N) lookup complexity
✅ **Secure** - Cryptographically verifiable peer identity
✅ **Production-ready** - Battle-tested Kademlia protocol

**Next Steps**:
1. Review this specification with team
2. Begin Phase 1 implementation (DHT core)
3. Target v0.2.0-alpha release in 8 weeks
4. Path to v1.0.0 production with QUIC migration

---

**Document Control**
- Version: 1.0
- Author: Winston (Architect)
- Reviewers: [To be assigned]
- Status: Approved for Implementation
- Next Review: After v0.2.0-alpha release
