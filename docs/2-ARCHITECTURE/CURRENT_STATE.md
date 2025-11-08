# ShadowMesh Current State Analysis

**Date**: November 8, 2025
**Version**: v11 (UDP+PQC) and v19 (QUIC)
**Status**: Alpha development builds (centralized discovery dependency)

---

## Overview

ShadowMesh has two primary alpha implementations:
- **v11**: UDP transport with post-quantum cryptography (ML-KEM-1024, ML-DSA-87)
- **v19**: QUIC transport without PQC (security regression)

Both versions depend on a centralized discovery backbone (now shut down), blocking standalone operation.

---

## v11: UDP + PQC Implementation

**Source Code**: `cmd/lightnode-l3-v11/` (archived in `.archive/alpha-builds/l3/v11/`)

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Client Application                     │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│              Post-Quantum Cryptography                   │
│  • ML-KEM-1024 (Kyber) - Key Exchange                   │
│  • ML-DSA-87 (Dilithium) - Signatures                   │
│  • ChaCha20-Poly1305 - Symmetric Encryption             │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│                  Layer 3 (TUN Device)                    │
│  • IP packet capture/injection                          │
│  • Virtual network interface                            │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│                   UDP Transport                          │
│  • Direct peer-to-peer connections                      │
│  • Frame-based protocol                                 │
│  • Encrypted payload transmission                       │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│             Centralized Discovery Backbone               │
│  • HTTP API: 209.151.148.121:8080                       │
│  • POST /register - Register peer with IP/port          │
│  • GET /peers - Retrieve peer list                      │
└─────────────────────────────────────────────────────────┘
```

### Key Features

**Post-Quantum Cryptography**:
- ML-KEM-1024 (NIST FIPS 203) for key encapsulation
- ML-DSA-87 (NIST FIPS 204) for digital signatures
- Hybrid mode: Classical (X25519, Ed25519) + PQC
- ChaCha20-Poly1305 for symmetric encryption

**Performance** (v11 Phase 3):
- Throughput: 100+ Mbps
- Latency: <50ms overhead
- Packet loss: <5% (improved from 95% in early v11)
- Heap allocations: 90% reduction via buffer pools

**Layer 3 Networking**:
- TUN device for IP-level routing
- Captures IP packets from virtual interface
- Encrypts and transmits over UDP
- Injects decrypted packets back to TUN

**UDP Transport**:
- Direct peer-to-peer UDP connections
- Frame-based protocol (not stream-based)
- Encrypted frames with ChaCha20-Poly1305
- NAT traversal challenges (no STUN/hole punching)

### Strengths

✅ **Quantum-Safe**: First production implementation of NIST PQC algorithms
✅ **Performance**: Achieved 100+ Mbps throughput
✅ **Encryption**: Strong cryptography with perfect forward secrecy
✅ **Layer 3**: IP-level routing enables full network stack

### Weaknesses

❌ **Centralized Discovery**: Requires discovery backbone at 209.151.148.121:8080
❌ **No DHT**: Peer discovery depends on central server
❌ **NAT Issues**: Limited NAT traversal (no hole punching)
❌ **Single Point of Failure**: Discovery server down = network down
❌ **Operational Cost**: Infrastructure charges for centralized service

### Code Quality

**Metrics**:
- Lines of code: ~2,500 (main.go + packages)
- Platforms: Linux (amd64, arm64), macOS (darwin-arm64)
- Variants: 12+ builds (buffered, adaptive, rtt-fixed, phase3, udpfix)
- Testing: Manual testing only (no automated tests)

**Architecture Issues**:
- Hardcoded discovery URL in code
- No fallback discovery mechanism
- Monolithic main.go (needs refactoring)

---

## v19: QUIC Implementation

**Source Code**: `cmd/lightnode-l3-v19/` (archived in `.archive/alpha-builds/l3/v19/`)

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Client Application                     │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│                 QUIC Transport Layer                     │
│  • quic-go library (v0.56.0)                            │
│  • TLS 1.3+ handshake                                   │
│  • Stream multiplexing                                  │
│  • Built-in congestion control                          │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│              Frame-Based Protocol                        │
│  • ChaCha20-Poly1305 encryption over QUIC               │
│  • Stream-based frame transmission                      │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│                  Layer 3 (TUN Device)                    │
│  • IP packet capture/injection                          │
│  • Virtual network interface                            │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│             Centralized Discovery Backbone               │
│  • HTTP API: 209.151.148.121:8080                       │
│  • POST /register - Register peer                       │
│  • GET /peers - Retrieve peer list                      │
└─────────────────────────────────────────────────────────┘
```

### Key Features

**QUIC Protocol**:
- Modern transport protocol (HTTP/3 foundation)
- Built on UDP (like v11) but with reliability
- Stream multiplexing (multiple streams per connection)
- Built-in congestion control and flow control
- 0-RTT connection establishment (after first handshake)

**Transport Benefits**:
- Better NAT traversal than raw UDP
- Automatic retransmission of lost packets
- Head-of-line blocking eliminated (vs TCP)
- Connection migration (IP address changes)

**Security** (via QUIC TLS 1.3):
- Encrypted headers and payload
- Perfect forward secrecy
- Replay protection
- **BUT**: No post-quantum cryptography

### Strengths

✅ **Modern Protocol**: QUIC is the future of internet transport
✅ **Reliability**: Built-in packet loss recovery
✅ **Performance**: 0-RTT reconnection, congestion control
✅ **NAT Traversal**: Better than raw UDP
✅ **Stream Multiplexing**: Multiple concurrent streams

### Weaknesses

❌ **No PQC**: Security regression from v11 (quantum vulnerable)
❌ **Centralized Discovery**: Same dependency as v11
❌ **No DHT**: Peer discovery requires central server
❌ **Incomplete**: Missing ML-KEM-1024 and ML-DSA-87 integration
❌ **Not Production-Ready**: Alpha quality, testing incomplete

### Code Quality

**Metrics**:
- Lines of code: ~2,800 (main.go + QUIC integration)
- Platforms: Linux (amd64, arm64), macOS (darwin-arm64)
- Binary size: 11 MB (vs 9 MB for v11, due to quic-go library)
- Testing: Manual testing only

**Architecture Issues**:
- QUIC integration not fully optimized
- Discovery dependency hardcoded
- No PQC handshake layer

---

## Comparison: v11 vs v19

| Feature | v11 (UDP+PQC) | v19 (QUIC) | Winner |
|---------|---------------|------------|--------|
| **Post-Quantum Crypto** | ✅ ML-KEM-1024, ML-DSA-87 | ❌ None | 🏆 v11 |
| **Transport Protocol** | UDP (raw) | QUIC (reliable) | 🏆 v19 |
| **Reliability** | ❌ Manual retransmit | ✅ Built-in | 🏆 v19 |
| **NAT Traversal** | ⚠️ Limited | ✅ Better | 🏆 v19 |
| **Performance** | 100+ Mbps | Not tested | ❓ Unknown |
| **0-RTT Reconnect** | ❌ No | ✅ Yes | 🏆 v19 |
| **Binary Size** | 9 MB | 11 MB | 🏆 v11 |
| **Security** | 🏆 Quantum-safe | ⚠️ Classical only | 🏆 v11 |
| **Discovery** | ❌ Centralized | ❌ Centralized | 🟰 Tie (both bad) |

### Strategic Assessment

**Best of Both**:
- v11's **post-quantum cryptography** is critical for future-proofing
- v19's **QUIC transport** is superior for reliability and NAT traversal

**Target**: Merge v11 + v19 → v20 (QUIC + PQC + DHT)

---

## Centralized Discovery Dependency

### Discovery Backbone Architecture

**Endpoints**:
- `POST /register` - Peer registration with public IP/port
- `GET /peers` - Retrieve list of all registered peers
- `GET /health` - Health check

**Database**: In-memory (no persistence)
**TTL**: 5 minutes (peers must re-register)
**Geographic Distribution**:
- NYC (us-nyc1): 209.151.148.121
- Singapore (sg-sin1): [IP not recorded]
- Sydney (au-syd1): [IP not recorded]

### Problems with Centralized Discovery

**Single Point of Failure**:
- Discovery server down = entire network down
- Happened November 8, 2025 when servers shut down

**Operational Cost**:
- $15-30/month for 3 discovery nodes
- $180-360/year ongoing infrastructure cost
- Unsustainable for decentralized project

**Privacy Concerns**:
- Central server knows all peer IPs
- Can track peer connections
- Potential surveillance target

**Scalability Issues**:
- All peers query central server
- Bottleneck for large networks
- Regional servers add complexity

**Philosophical Contradiction**:
- Decentralized VPN with centralized discovery
- Defeats purpose of peer-to-peer architecture

---

## Infrastructure Shutdown (November 8, 2025)

**Discovery Nodes Stopped**:
- shadowmesh-discovery-nyc (us-nyc1)
- shadowmesh-discovery-sin (sg-sin1)
- shadowmesh-discovery-syd (au-syd1)

**Impact**:
- v11 alpha builds: **Non-operational**
- v19 alpha builds: **Non-operational**
- All peer discovery: **Failed**

**Cost Savings**: $180-360/year

**Rationale**: Transitioning to Kademlia DHT eliminates infrastructure dependency.

---

## Performance Analysis

### v11 Performance (Phase 3)

**Throughput**:
- Single connection: 100+ Mbps
- Multi-connection: Not tested
- Target: 6-7 Gbps (not yet achieved)

**Latency**:
- Overhead: <50ms (improved from 3000ms in early v11)
- Target: <2ms overhead (not yet achieved)

**Packet Loss**:
- Current: <5% (improved from 95% in early v11)
- Target: <1%

**Optimizations Applied**:
- Buffer pools (reduced heap allocations by 90%)
- Stack allocation for small buffers
- Adaptive channel sizing (BDP-based)
- RTT measurement fixes
- UDP frame handling optimizations

### v19 Performance

**Status**: Not benchmarked
- QUIC transport implemented but not performance tested
- Expected to be faster than v11 due to built-in optimizations
- Needs testing before production

---

## Code Organization

### Current Structure

```
shadowmesh/
├── cmd/
│   ├── lightnode-l2/           # Layer 2 (TAP) - deprecated
│   ├── lightnode-l2-v10/       # Layer 2 v10 - archived
│   ├── lightnode-l3-v11/       # Layer 3 + UDP + PQC
│   ├── lightnode-l3-v19/       # Layer 3 + QUIC (no PQC)
│   └── discovery/              # Discovery backbone server
├── pkg/
│   ├── crypto/                 # ChaCha20, TLS helpers
│   ├── p2p/                    # P2P connection logic (UDP)
│   ├── transport/              # QUIC transport (v19)
│   ├── layer3/                 # TUN device management
│   ├── discovery/              # Kademlia DHT (15% complete)
│   └── [other packages]
└── .archive/
    └── alpha-builds/
        ├── l2/                 # Layer 2 archived binaries
        └── l3/                 # Layer 3 archived binaries
```

### Issues with Current Organization

**Duplication**:
- v11 and v19 have separate implementations
- Crypto code in v11 not reused in v19
- TUN device logic duplicated

**Incomplete Packages**:
- `pkg/discovery/kademlia.go` only 15% complete
- `pkg/transport/quic.go` missing PQC integration
- `pkg/p2p/` only supports UDP (not QUIC)

**Monolithic main.go**:
- v11: 2,500+ lines in single file
- v19: 2,800+ lines in single file
- Needs refactoring into modular packages

---

## Testing Status

### v11 Testing

**Unit Tests**: None (manual testing only)
**Integration Tests**: None
**Performance Tests**: Manual (using quick-perf-test.sh)
**Security Tests**: None (no formal audit)

**Manual Test Results**:
- 3-node test: Successful peer connectivity
- 4-node test: NAT traversal issues
- Long-running stability: Not tested (>24 hours)

### v19 Testing

**Unit Tests**: None
**Integration Tests**: None
**Performance Tests**: Not conducted
**QUIC Protocol Tests**: Basic connectivity only

---

## Dependencies

### Go Modules (go.mod)

**PQC Cryptography**:
- `github.com/cloudflare/circl` - ML-KEM-1024, ML-DSA-87

**QUIC Transport**:
- `github.com/quic-go/quic-go v0.56.0` - QUIC implementation

**Networking**:
- `github.com/songgao/water` - TUN/TAP device management
- `github.com/google/gopacket` - Packet manipulation

**Other**:
- Standard library crypto (X25519, Ed25519, ChaCha20-Poly1305)

### External Services (Now Deprecated)

**Discovery Backbone**: 209.151.148.121:8080 (shut down November 8, 2025)

---

## Security Posture

### v11 Security

**Strengths**:
- ✅ Post-quantum key exchange (ML-KEM-1024)
- ✅ Post-quantum signatures (ML-DSA-87)
- ✅ Hybrid mode (classical + PQC)
- ✅ Perfect forward secrecy
- ✅ Replay protection (monotonic counters)

**Weaknesses**:
- ❌ No formal security audit
- ❌ No penetration testing
- ❌ Centralized discovery (privacy risk)
- ❌ No traffic obfuscation (DPI detectable)
- ❌ No multi-hop routing

### v19 Security

**Strengths**:
- ✅ TLS 1.3 via QUIC
- ✅ Encrypted headers and payload
- ✅ Perfect forward secrecy (TLS 1.3)

**Weaknesses**:
- ❌ **No post-quantum cryptography** (quantum vulnerable)
- ❌ No formal security audit
- ❌ Centralized discovery (same as v11)
- ❌ No traffic obfuscation
- ❌ No multi-hop routing

---

## Roadmap Alignment

### Current State Problems

1. **Centralized Discovery**: Blocks standalone operation
2. **Split Implementations**: v11 has PQC, v19 has QUIC (need merge)
3. **No DHT**: Peer discovery not decentralized
4. **Performance Gap**: 100 Mbps vs 6-7 Gbps target
5. **No Production Testing**: Alpha quality only

### Path to Target State

**Sprint 0-2**: Implement Kademlia DHT
**Sprint 3-4**: Merge v11 (PQC) + v19 (QUIC) → v20
**Sprint 5+**: Optimize performance, security audit, beta release

See [MIGRATION_PATH.md](MIGRATION_PATH.md) for detailed sprint plan.

---

## Conclusion

### Current State Summary

**v11 (UDP+PQC)**:
- ✅ Quantum-safe cryptography
- ✅ Working prototype (100+ Mbps)
- ❌ Centralized discovery dependency
- ❌ Limited NAT traversal

**v19 (QUIC)**:
- ✅ Modern QUIC protocol
- ✅ Better NAT traversal
- ❌ No post-quantum cryptography
- ❌ Centralized discovery dependency

**Infrastructure**:
- ❌ Discovery backbone shut down (November 8, 2025)
- ❌ Alpha builds non-operational
- ✅ Cost savings: $180-360/year

### Next Steps

1. **Implement Kademlia DHT** (Sprint 0-2)
2. **Merge v11 + v19** → v20 (Sprint 3-4)
3. **Eliminate discovery dependency** (Sprint 5+)
4. **Beta release** with standalone operation

**Target**: Fully decentralized, quantum-safe, high-performance VPN with zero infrastructure dependencies.

---

**Document Status**: ✅ COMPLETE
**Last Updated**: November 8, 2025
**Next Review**: After Sprint 0 (DHT POC complete)
