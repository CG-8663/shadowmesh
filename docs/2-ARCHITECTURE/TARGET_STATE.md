# ShadowMesh Target State Architecture

**Date**: November 8, 2025
**Version**: v20+ (Kademlia DHT + PQC + QUIC)
**Status**: Target architecture for standalone decentralized operation

---

## Vision

**ShadowMesh v1.0.0** will be a **fully decentralized, quantum-safe, high-performance VPN** with:

- **Zero central dependencies** (no discovery servers, no control plane)
- **Kademlia DHT** for peer discovery and routing
- **QUIC transport** for reliable, low-latency connections
- **Post-quantum cryptography** (ML-KEM-1024, ML-DSA-87)
- **6-7 Gbps throughput** with <2ms latency overhead
- **Standalone operation** from first boot

---

## Target Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                      Client Application                           │
│  • Configuration management                                       │
│  • PeerID generation (from ML-DSA-87 keys)                       │
│  • Bootstrap node list (hardcoded or configured)                 │
└──────────────────────────────────────────────────────────────────┘
                              ↓
┌──────────────────────────────────────────────────────────────────┐
│                     Kademlia DHT Layer                            │
│  • Decentralized peer discovery (zero central servers)           │
│  • Routing table (k-buckets, k=20)                               │
│  • FIND_NODE iterative lookup                                    │
│  • STORE/FIND_VALUE operations (peer metadata)                   │
│  • Peer exchange via gossip protocol                             │
│  • Bootstrap from seed nodes (3-5 hardcoded peers)               │
└──────────────────────────────────────────────────────────────────┘
                              ↓
┌──────────────────────────────────────────────────────────────────┐
│              Post-Quantum Cryptography Layer                      │
│  • ML-KEM-1024 (Kyber) - Key Exchange over QUIC                  │
│  • ML-DSA-87 (Dilithium) - Peer Authentication                   │
│  • ChaCha20-Poly1305 - Symmetric Encryption                      │
│  • Hybrid Mode: Classical (X25519, Ed25519) + PQC                │
│  • PeerID derived from ML-DSA-87 public key (SHA256 hash)        │
└──────────────────────────────────────────────────────────────────┘
                              ↓
┌──────────────────────────────────────────────────────────────────┐
│                       QUIC Transport Layer                        │
│  • quic-go library (stream-based protocol)                       │
│  • TLS 1.3+ handshake with PQC certificates                      │
│  • 0-RTT reconnection (after initial handshake)                  │
│  • Built-in congestion control (BBR, Cubic)                      │
│  • Connection migration (IP address changes)                     │
│  • NAT traversal (better than raw UDP)                           │
└──────────────────────────────────────────────────────────────────┘
                              ↓
┌──────────────────────────────────────────────────────────────────┐
│                   Layer 3 (TUN Device)                            │
│  • Virtual network interface (10.x.x.x IP addresses)             │
│  • IP packet capture/injection                                   │
│  • Full network stack support (TCP, UDP, ICMP)                   │
└──────────────────────────────────────────────────────────────────┘
                              ↓
┌──────────────────────────────────────────────────────────────────┐
│                     Physical Network                              │
│  • Internet connectivity (any ISP)                               │
│  • NAT traversal (automatic)                                     │
│  • Multi-path routing (optional)                                 │
└──────────────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. Kademlia DHT

**Purpose**: Decentralized peer discovery and routing

**Key Features**:
- **Routing Table**: k-buckets (k=20) for 160-bit PeerID space
- **FIND_NODE**: Iterative lookup to find peers closest to target PeerID
- **STORE**: Store peer metadata (IP, port, public keys, capabilities)
- **FIND_VALUE**: Retrieve peer metadata by PeerID
- **Peer Exchange**: Gossip protocol to discover new peers

**PeerID Generation**:
```
PeerID = SHA256(ML-DSA-87 Public Key)
```
- 256-bit identifier derived from post-quantum signature key
- Cryptographically verifiable (peer proves ownership via signature)
- Collision-resistant (SHA256)

**Bootstrap Process**:
1. Start with hardcoded seed nodes (3-5 well-known peers)
2. Query seed nodes for peers close to own PeerID
3. Populate routing table with discovered peers
4. Begin accepting connections from other peers
5. Participate in DHT routing and peer exchange

**Routing Table Structure**:
- 160 k-buckets (one per bit distance)
- Each bucket holds up to k=20 peers
- LRU eviction (least recently seen peers removed first)
- Active probing (ping peers to verify liveness)

**DHT Operations**:
- **FIND_NODE(target_id)**: Find k closest peers to target
- **STORE(key, value, ttl)**: Store peer metadata with expiration
- **FIND_VALUE(key)**: Retrieve stored value by key
- **PING/PONG**: Liveness checks for routing table maintenance

### 2. Post-Quantum Cryptography

**Purpose**: Quantum-safe encryption and authentication

**ML-KEM-1024 (Kyber) - Key Exchange**:
- NIST FIPS 203 standardized algorithm
- 1024-bit security level (IND-CCA2 secure)
- Key encapsulation mechanism (KEM)
- Generates shared secret for ChaCha20-Poly1305

**ML-DSA-87 (Dilithium) - Digital Signatures**:
- NIST FIPS 204 standardized algorithm
- 87-bit security level (EUF-CMA secure)
- Used for peer authentication and PeerID derivation
- Signs QUIC TLS certificates for peer verification

**ChaCha20-Poly1305 - Symmetric Encryption**:
- IETF RFC 8439 standardized cipher
- 256-bit key, 96-bit nonce
- Authenticated encryption (AEAD)
- High performance (software implementation)

**Hybrid Mode** (Classical + PQC):
- X25519 (ECDH) + ML-KEM-1024 → Combine shared secrets
- Ed25519 + ML-DSA-87 → Dual signatures
- Defense in depth: Protected even if one algorithm broken

**PQC Handshake over QUIC**:
```
Client                                  Server
  |                                       |
  |-------- QUIC ClientHello ----------->|
  |  (TLS 1.3 + ML-DSA-87 cert)          |
  |                                       |
  |<------- QUIC ServerHello ------------|
  |  (TLS 1.3 + ML-DSA-87 cert)          |
  |                                       |
  |-------- ML-KEM-1024 Encapsulation -->|
  |  (KEM ciphertext)                    |
  |                                       |
  |<------- Session Keys Established ----|
  |  (Shared secret → ChaCha20 key)      |
  |                                       |
  |<======= Encrypted QUIC Stream ======>|
  |  (IP packets encrypted with PQC)     |
```

### 3. QUIC Transport

**Purpose**: Reliable, low-latency peer-to-peer connections

**QUIC Protocol Benefits**:
- **UDP-based**: Firewall and NAT friendly
- **Stream Multiplexing**: Multiple streams per connection (no head-of-line blocking)
- **0-RTT Reconnection**: Instant reconnection after first handshake
- **Congestion Control**: BBR, Cubic, or custom algorithms
- **Connection Migration**: Survives IP address changes
- **Built-in Encryption**: TLS 1.3 integrated into protocol

**Frame-Based Protocol**:
```
QUIC Stream Frame Format:
┌─────────────────────────────────────────────────────┐
│ Frame Header (16 bytes)                             │
│  • Magic (4 bytes): 0x534D5348 ("SMSH")            │
│  • Version (2 bytes): 0x0001                       │
│  • Type (1 byte): DATA, PING, PONG, etc.          │
│  • Sequence (8 bytes): Monotonic counter           │
│  • Length (2 bytes): Payload size                  │
│  • Reserved (1 byte): Flags for future use         │
├─────────────────────────────────────────────────────┤
│ Encrypted Payload (variable length)                │
│  • ChaCha20-Poly1305 encrypted IP packet           │
│  • AEAD tag (16 bytes) for authentication          │
└─────────────────────────────────────────────────────┘
```

**Stream Types**:
- **Data Stream (0)**: Encrypted IP packets
- **Control Stream (1)**: DHT operations, peer exchange
- **Heartbeat Stream (2)**: Keepalive, latency measurement

**NAT Traversal**:
- QUIC's UDP-based design better for NAT than TCP
- Connection migration handles NAT rebinding
- Peer exchange via DHT provides fallback routes
- Relay nodes (optional) for symmetric NAT scenarios

### 4. Layer 3 Networking

**Purpose**: Virtual network interface for IP-level routing

**TUN Device**:
- Creates virtual network interface (e.g., `tun0`)
- Assigns private IP address (e.g., `10.100.1.5/16`)
- Captures outgoing IP packets from applications
- Injects incoming IP packets to applications

**IP Address Assignment**:
```
IP Address = 10.100.X.Y
where:
  X = (PeerID[0:8]) % 256
  Y = (PeerID[8:16]) % 256
```
- Deterministic assignment from PeerID
- Collision-resistant (SHA256-derived PeerID)
- /16 subnet supports 65,536 peers

**Routing**:
- All traffic through TUN device routed to ShadowMesh
- Split tunneling (optional): Route only specific subnets
- Full tunneling (default): Route all traffic through mesh

**Packet Flow**:
```
Application → TUN device → ShadowMesh client
  → Encrypt with ChaCha20-Poly1305
  → Send over QUIC stream to peer
  → Peer decrypts and injects to TUN device
  → Peer application receives packet
```

---

## Performance Targets

### Throughput

**Target**: 6-7 Gbps (single connection)
- Current (v11): 100+ Mbps
- Gap: 60-70x improvement needed

**Optimization Strategies**:
- Zero-copy packet handling (avoid memcpy)
- SIMD acceleration for ChaCha20-Poly1305
- Kernel bypass (io_uring, XDP) for TUN device
- Parallel QUIC streams (multiple cores)
- Hardware offload (AES-NI, AVX2)

### Latency

**Target**: <2ms overhead
- Current (v11): <50ms overhead
- Gap: 25x improvement needed

**Optimization Strategies**:
- Reduce buffer pool contention (per-stream buffers)
- Optimize DHT lookup (cache peer routes)
- Pre-establish QUIC connections (anticipate traffic)
- Async I/O (non-blocking TUN reads/writes)

### Packet Loss

**Target**: <1%
- Current (v11): <5%
- Gap: 5x improvement needed

**Optimization Strategies**:
- QUIC's built-in retransmission (automatic)
- Congestion control tuning (BBR vs Cubic)
- MTU discovery (avoid fragmentation)

### Scalability

**Target**: 1000+ concurrent peer connections
- Current: Not tested
- Expected: 100-200 connections per node

**Resource Requirements**:
- CPU: 2 cores minimum, 4+ cores recommended
- RAM: 512 MB minimum, 1 GB recommended
- Network: 1 Gbps link minimum

---

## Security Goals

### Quantum Resistance

✅ **ML-KEM-1024**: Quantum-safe key exchange (NIST FIPS 203)
✅ **ML-DSA-87**: Quantum-safe signatures (NIST FIPS 204)
✅ **Hybrid Mode**: Classical + PQC for defense in depth
✅ **256-bit Security**: Exceeds recommended 128-bit minimum

### Privacy

✅ **Zero Knowledge Discovery**: DHT reveals only PeerID (not real identity)
✅ **Encrypted Metadata**: Peer exchange over encrypted QUIC streams
✅ **No Central Logging**: No servers to log peer IPs or connections
✅ **Traffic Obfuscation**: QUIC traffic looks like HTTPS (future: mimicry)

### Authentication

✅ **PeerID Verification**: Peers prove ownership via ML-DSA-87 signature
✅ **Certificate Pinning**: QUIC TLS certificates signed with PQC keys
✅ **Replay Protection**: Monotonic sequence numbers in frame headers
✅ **Mutual TLS**: Both client and server authenticate each other

### Future Security Enhancements

⏳ **Traffic Obfuscation**: QUIC mimicry to look like HTTPS traffic
⏳ **Cover Traffic**: Random padding to hide traffic patterns
⏳ **Multi-Hop Routing**: 3-5 hop onion routing (like Tor)
⏳ **Zero-Trust Exit Nodes**: TPM/SGX attestation for exit nodes
⏳ **Blockchain Governance**: Smart contract-based relay verification

---

## Zero-Dependency Operation

### Bootstrap Process

**First Boot** (no configuration):
1. Generate ML-DSA-87 keypair
2. Derive PeerID from public key (SHA256)
3. Assign IP address (10.100.X.Y from PeerID)
4. Create TUN device (tun0 with assigned IP)
5. Connect to hardcoded bootstrap nodes (3-5 peers)
6. Query DHT for peers close to own PeerID
7. Populate routing table with discovered peers
8. Begin routing traffic through mesh

**Hardcoded Bootstrap Nodes**:
```yaml
bootstrap_nodes:
  - peer_id: "abcd1234..."
    address: "bootstrap1.shadowmesh.io:4433"
  - peer_id: "ef567890..."
    address: "bootstrap2.shadowmesh.io:4433"
  - peer_id: "1234abcd..."
    address: "bootstrap3.shadowmesh.io:4433"
```
- Bootstrap nodes are long-running stable peers
- Operated by community (not centralized company)
- Only needed for initial DHT entry (not ongoing dependency)

**After Bootstrap**:
- Peer exchange via DHT (gossip protocol)
- Routing table self-maintains (LRU eviction, liveness probes)
- New peers discovered automatically (no bootstrap needed)

### No Infrastructure Costs

**Zero Operational Expenses**:
- No discovery servers ($0 vs $180-360/year for v11)
- No control plane servers
- No monitoring infrastructure (optional self-hosted)
- No databases (in-memory DHT only)

**Community-Run Bootstrap Nodes**:
- Volunteers run bootstrap nodes (like Bitcoin/Ethereum)
- Multiple independent operators (no single entity control)
- Optional incentives (future: token rewards for operators)

---

## Comparison: Current vs Target

| Feature | Current (v11/v19) | Target (v20+) | Improvement |
|---------|-------------------|---------------|-------------|
| **Peer Discovery** | Centralized server | Kademlia DHT | ✅ Decentralized |
| **PQC** | ✅ v11 only | ✅ Integrated | ✅ Unified |
| **Transport** | UDP (v11) / QUIC (v19) | QUIC + PQC | ✅ Best of both |
| **Throughput** | 100 Mbps | 6-7 Gbps | ✅ 60-70x faster |
| **Latency** | <50ms | <2ms | ✅ 25x faster |
| **Infrastructure Cost** | $180-360/year | $0 | ✅ 100% savings |
| **Standalone** | ❌ No (discovery) | ✅ Yes (DHT) | ✅ Zero dependencies |
| **Bootstrap** | Central server | 3-5 seed nodes | ✅ Minimal dependency |
| **NAT Traversal** | Limited (v11) | Built-in (QUIC) | ✅ Improved |
| **Security Audit** | ❌ None | ⏳ Planned (v1.0) | ⏳ Future |

---

## Development Roadmap

### Sprint 0: Architecture POC (Weeks 1-2)

**DHT Research**:
- [ ] Study Kademlia paper (Maymounkov & Mazières, 2002)
- [ ] Analyze libp2p's Kademlia implementation
- [ ] Design PeerID generation from ML-DSA-87 keys
- [ ] Define routing table structure (k-buckets, k=20)

**POC Goals**:
- [ ] Local 3-node DHT network (same machine)
- [ ] FIND_NODE operation working
- [ ] Routing table population successful
- [ ] Basic peer exchange demonstrated

### Sprint 1-2: Kademlia DHT Core (Weeks 3-6)

**DHT Operations**:
- [ ] Implement FIND_NODE iterative lookup
- [ ] Implement STORE operation with TTL
- [ ] Implement FIND_VALUE with caching
- [ ] Routing table management (LRU eviction, liveness probes)

**Testing**:
- [ ] 5-node local test network
- [ ] Peer discovery latency <100ms
- [ ] Routing table convergence <30 seconds
- [ ] DHT lookup success rate >95%

### Sprint 3-4: QUIC + PQC Integration (Weeks 7-10)

**Merge v11 + v19**:
- [ ] Port ML-KEM-1024 handshake to QUIC
- [ ] Port ML-DSA-87 authentication to QUIC
- [ ] Integrate ChaCha20-Poly1305 over QUIC streams
- [ ] TUN device with QUIC transport

**Testing**:
- [ ] Full PQC handshake over QUIC successful
- [ ] End-to-end encrypted traffic working
- [ ] Performance baseline (throughput, latency)

### Sprint 5+: Standalone Operation (Weeks 11-18)

**Zero Dependencies**:
- [ ] Remove all discovery backbone references
- [ ] Implement bootstrap node connection
- [ ] Peer exchange via DHT gossip
- [ ] Connection migration handling

**Optimization**:
- [ ] Performance tuning (target 6-7 Gbps)
- [ ] Memory optimization (reduce allocations)
- [ ] CPU profiling and bottleneck removal

**Security Audit**:
- [ ] Third-party security audit (external firm)
- [ ] Penetration testing
- [ ] Cryptography review
- [ ] Protocol analysis

**Beta Release** (v1.0.0-beta.1):
- [ ] All features complete
- [ ] Security audit passed
- [ ] Performance targets met
- [ ] Documentation complete
- [ ] 100+ beta testers

---

## Success Criteria

### Technical Goals

✅ **Kademlia DHT**: Fully decentralized peer discovery
✅ **Zero Dependencies**: No central servers required
✅ **PQC + QUIC**: Quantum-safe transport with QUIC reliability
✅ **6-7 Gbps Throughput**: High-performance networking
✅ **<2ms Latency**: Low overhead for real-time applications
✅ **<1% Packet Loss**: Reliable data transmission

### Operational Goals

✅ **$0 Infrastructure Cost**: No ongoing operational expenses
✅ **Standalone Operation**: Works from first boot without config
✅ **Community Bootstrap**: Volunteer-run bootstrap nodes
✅ **Security Audit**: Third-party audit passed

### Adoption Goals

✅ **1000+ GitHub Stars**: Community interest
✅ **100+ Beta Testers**: Real-world validation
✅ **10+ Contributors**: Community development
✅ **99.9% Uptime**: Reliable network operation

---

## Risks & Mitigation

### Technical Risks

**Risk**: DHT lookup latency too high (>1 second)
- **Mitigation**: Cache peer routes, pre-populate routing table

**Risk**: QUIC performance not meeting 6-7 Gbps target
- **Mitigation**: Kernel bypass (io_uring), hardware offload, parallel streams

**Risk**: NAT traversal failures in symmetric NAT
- **Mitigation**: Optional relay nodes, UPnP, manual port forwarding

**Risk**: PQC key sizes too large (slow handshake)
- **Mitigation**: Pre-compute keys, cache handshake results, hybrid mode

### Operational Risks

**Risk**: Bootstrap nodes go offline (network partitioned)
- **Mitigation**: Multiple bootstrap operators, peer exchange, local cache

**Risk**: DHT poisoning attacks (malicious peers)
- **Mitigation**: PeerID verification via ML-DSA-87, reputation system

**Risk**: Sybil attacks (one attacker controls many PeerIDs)
- **Mitigation**: Proof-of-work for PeerID generation (future), rate limiting

### Adoption Risks

**Risk**: Complexity too high for average users
- **Mitigation**: Zero-config defaults, GUI client, clear documentation

**Risk**: Performance issues deter users
- **Mitigation**: Benchmarking, optimization, realistic targets

**Risk**: Security vulnerabilities discovered
- **Mitigation**: Third-party audit, bug bounty program, rapid patching

---

## Competitive Advantages

### vs Tailscale

✅ **Post-Quantum Cryptography**: Tailscale uses classical crypto only
✅ **Decentralized**: No central control server dependency
✅ **Zero Cost**: No infrastructure expenses
✅ **Open Source**: Fully transparent vs Tailscale's proprietary server

### vs WireGuard

✅ **Post-Quantum**: WireGuard uses X25519 (quantum vulnerable)
✅ **Automatic Discovery**: WireGuard requires manual peer configuration
✅ **DHT Routing**: Dynamic peer discovery vs static configs
✅ **QUIC Transport**: Better NAT traversal than WireGuard's UDP

### vs ZeroTier

✅ **Post-Quantum**: ZeroTier uses classical crypto
✅ **True P2P**: ZeroTier has central "moon" servers
✅ **Zero Dependencies**: No root servers required
✅ **QUIC**: Modern protocol vs ZeroTier's custom protocol

---

## Conclusion

### Target State Vision

**ShadowMesh v1.0.0** will be the **world's first fully decentralized, quantum-safe, high-performance VPN** with:

1. **Kademlia DHT** for zero-dependency peer discovery
2. **QUIC + PQC** for quantum-resistant, reliable transport
3. **6-7 Gbps throughput** with <2ms latency overhead
4. **Standalone operation** from first boot (no servers needed)
5. **$0 infrastructure cost** (community-run bootstrap nodes)

### Why This Matters

**Quantum Threat**: "Harvest now, decrypt later" attacks are real
- Nation-states collecting encrypted traffic today
- Quantum computers will break classical crypto (10-15 years)
- ShadowMesh protects against future decryption

**Decentralization**: No single point of failure or control
- No company can shut down the network
- No government can censor peer discovery
- No surveillance of peer connections

**Performance**: Competitive with commercial VPNs
- 6-7 Gbps throughput matches WireGuard
- <2ms latency suitable for gaming, VoIP
- QUIC provides reliability without TCP overhead

**Open Source**: Community-driven, transparent, auditable
- No backdoors, no telemetry, no hidden agendas
- Security through transparency
- Community contributions welcomed

---

**Document Status**: ✅ COMPLETE
**Last Updated**: November 8, 2025
**Next Review**: After Sprint 2 (DHT core complete)
