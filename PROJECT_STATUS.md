# ShadowMesh Project Status & Roadmap

**Last Updated**: November 10, 2025
**Current Version**: v0.1.0-alpha (Released)
**Next Version**: v0.2.0-alpha (In Development)
**Project Phase**: DHT Implementation for Standalone Operation

---

## Executive Summary

ShadowMesh is a post-quantum decentralized private network (DPN) that surpasses WireGuard, Tailscale, and ZeroTier. After successfully releasing v0.1.0-alpha with excellent performance (28.3 Mbps, 45% faster than Tailscale), we are now transitioning to **v0.2.0-alpha** with Kademlia DHT for fully decentralized, standalone operation.

**Critical Milestone**: Removing centralized discovery dependency to enable standalone install and testing.

---

## Current Status (November 10, 2025)

### ✅ v0.1.0-alpha Released

**Achievement**: Functional post-quantum VPN with strong performance

**Key Metrics**:
- **Throughput**: 28.3 Mbps sender, 27.0 Mbps receiver
- **Comparison**: 45% faster than Tailscale (35.5 vs 24.5 Mbps)
- **Video Streaming**: 640x480 @ 547 kb/s successful
- **Test Duration**: 3-hour stability test, zero packet loss
- **Architecture**: UDP transport + ML-KEM-1024 + ML-DSA-87

**Blocker Identified**: Centralized discovery server at 209.151.148.121:8080 (now shut down)
- Nodes cannot discover peers without central infrastructure
- Blocks standalone deployment and testing
- Creates single point of failure

**Documentation**:
- `docs/benchmarks/v0.1.0-alpha-performance.md`
- `docs/benchmarks/v0.1.0-alpha-video-streaming.md`
- `docs/2-ARCHITECTURE/CURRENT_STATE.md`

---

## Active Development: v0.2.0-alpha DHT Migration

### 🔄 Phase 1: Kademlia DHT Implementation (4 Weeks)

**Objective**: Replace centralized discovery with Kademlia DHT for peer-to-peer discovery

**Timeline**: November 10 - December 8, 2025 (4 weeks)

**Architecture Decision**: Conservative migration path
- Keep UDP transport from v11 (proven performance)
- Add Kademlia DHT for decentralized discovery
- Deploy 3-5 bootstrap nodes for network entry
- Defer QUIC migration to v0.3.0+ (reduce complexity)

**Implementation Plan**:

| Sprint | Duration | Focus | Tickets |
|--------|----------|-------|---------|
| **Sprint 0** | Weeks 1-2 | DHT Foundation | TICKET-001 to TICKET-007 |
| **Sprint 1** | Weeks 3-4 | DHT Operations | TICKET-008 to TICKET-015 |

**Sprint 0 (Weeks 1-2): Foundation**
- TICKET-001: PeerID generation from ML-DSA-87 public keys
- TICKET-002: XOR distance calculations
- TICKET-003: PeerInfo data structures
- TICKET-004: k-bucket implementation (LRU eviction)
- TICKET-005: 256-bucket routing table
- TICKET-006: UDP packet protocol (PING, PONG, FIND_NODE, STORE)
- TICKET-007: PING/PONG handlers for peer liveness

**Sprint 1 (Weeks 3-4): Operations**
- TICKET-008: FIND_NODE iterative lookup (α=3 parallelism)
- TICKET-009: Bootstrap node integration (3-5 seed nodes)
- TICKET-010: Integration with ML-DSA-87 keypairs
- TICKET-011: STORE/FIND_VALUE operations
- TICKET-012: NAT detection and peer reachability
- TICKET-013: Routing table persistence
- TICKET-014: Health monitoring and Prometheus metrics
- TICKET-015: Integration tests (3-node, 10-node networks)

**Estimated Effort**: 36 developer-days (~4 weeks with parallelization)

**Critical Path**:
```
TICKET-001 → TICKET-005 → TICKET-007 → TICKET-008 → TICKET-011
(PeerID)     (Routing)    (Handlers)   (Lookup)     (Store/Find)
```

**Documentation**:
- Architecture, implementation, operations, and testing documentation (internal - available at release)

---

### 🎯 Success Criteria for v0.2.0-alpha

**Functional Requirements**:
- [ ] Standalone operation (no discovery server dependency)
- [ ] 3-node local test network converges in <60 seconds
- [ ] FIND_NODE lookup completes in <500ms
- [ ] 95%+ peer discovery success rate
- [ ] Integration with existing v11 UDP transport

**Performance Requirements** (maintain v0.1.0-alpha baseline):
- [ ] ≥25 Mbps throughput (maintain 28.3 Mbps from v11)
- [ ] <50ms latency overhead
- [ ] <5% packet loss
- [ ] 3-hour stability test passes

**Quality Requirements**:
- [ ] 85%+ unit test coverage
- [ ] 50+ integration tests passing
- [ ] Automated CI/CD pipeline
- [ ] Zero critical security vulnerabilities

---

## Infrastructure Plan

### Bootstrap Node Deployment

**Objective**: Deploy 3 bootstrap nodes for DHT network entry

**Geographic Distribution**:
- **US East** (Virginia/New York) - Primary
- **EU West** (London/Frankfurt) - Secondary
- **Asia Pacific** (Singapore/Tokyo) - Tertiary

**Hardware Specs** (per node):
- CPU: 2 vCPU
- RAM: 2 GB
- Storage: 20 GB SSD
- Network: 1 Gbps unmetered
- OS: Ubuntu 22.04 LTS

**Provider**: Linode or DigitalOcean (recommended for alpha)

**Cost**: $30-45/month total (3 nodes × $10-15/month)

**Security**:
- fail2ban for brute-force protection
- iptables rate limiting (100 conn/min per IP)
- Automatic security updates
- Prometheus monitoring on port 9090
- UptimeRobot health checks

**SLA Target**: 99.9% uptime (43 minutes downtime/month)

**Documentation**: `docs/4-OPERATIONS/BOOTSTRAP_NODE_DEPLOYMENT.md`

---

## Long-Term Vision

### v0.3.0-alpha: QUIC Migration (Weeks 5-10)

**Objective**: Replace UDP with QUIC for better NAT traversal and reliability

**Key Features**:
- QUIC transport with quic-go library
- 0-RTT reconnection for faster handshakes
- Built-in congestion control (BBR/Cubic)
- Connection migration for IP address changes
- Improved NAT traversal over raw UDP

**Estimated Timeline**: 6 weeks after v0.2.0-alpha release

---

### v1.0.0: Production Release (6-12 Months)

**Objective**: Production-ready quantum-safe DPN with enterprise features

**Core Features**:
- ✅ Kademlia DHT (from v0.2.0-alpha)
- ✅ QUIC transport (from v0.3.0-alpha)
- ✅ Post-quantum cryptography (ML-KEM-1024, ML-DSA-87)
- Advanced NAT traversal (STUN/TURN fallback)
- Multi-hop routing (3-5 hops configurable)
- Traffic obfuscation (WebSocket mimicry)
- Atomic clock synchronization (Rubidium/Cesium)
- TPM/SGX attestation for exit nodes

**Performance Targets**:
- 6-7 Gbps single-connection throughput
- <2ms latency overhead
- 99.9% uptime
- 1000+ concurrent connections per relay

**Platform Support**:
- Linux (amd64, arm64) - CLI and GUI
- macOS (arm64, x86_64) - Native app
- Windows (x86_64) - Native app
- iOS - Mobile app
- Android - Mobile app

**Certifications**:
- SOC 2 Type II
- HIPAA compliance
- PCI DSS ready

---

## Project Timeline (8-Week Focus)

```
Week 1-2: Sprint 0 - DHT Foundation
├─ TICKET-001: PeerID generation
├─ TICKET-002: XOR distance
├─ TICKET-003: PeerInfo structures
├─ TICKET-004: k-bucket LRU
├─ TICKET-005: Routing table
├─ TICKET-006: UDP packet protocol
└─ TICKET-007: PING/PONG handlers

Week 3-4: Sprint 1 - DHT Operations
├─ TICKET-008: FIND_NODE iterative lookup
├─ TICKET-009: Bootstrap integration
├─ TICKET-010: ML-DSA-87 integration
├─ TICKET-011: STORE/FIND_VALUE
├─ TICKET-012: NAT detection
├─ TICKET-013: Routing persistence
├─ TICKET-014: Monitoring
└─ TICKET-015: Integration tests

Week 5-6: Testing & Validation
├─ 3-node local network testing
├─ 10-node distributed network testing
├─ Performance regression tests
├─ Bootstrap node deployment (3 regions)
└─ End-to-end standalone operation

Week 7-8: Documentation & Release
├─ README.md with quick start
├─ INSTALLATION.md for all platforms
├─ TROUBLESHOOTING.md
├─ Performance benchmarks
├─ Release binaries (Linux amd64/arm64, macOS arm64)
└─ v0.2.0-alpha release to GitHub
```

---

## Key Metrics

### Development Velocity
- **Current Sprint**: Sprint 0 (DHT Foundation)
- **Sprint Duration**: 2 weeks
- **Total Tickets**: 15 (7 in Sprint 0, 8 in Sprint 1)
- **Estimated Effort**: 36 developer-days

### Performance Baselines (from v0.1.0-alpha)
- **Throughput**: 28.3 Mbps (target: maintain ≥25 Mbps)
- **Latency**: <50ms overhead
- **Packet Loss**: <5%
- **Stability**: 3-hour test, zero packet loss

### Quality Metrics (Target for v0.2.0-alpha)
- **Unit Test Coverage**: 85%+
- **Integration Tests**: 50+ tests
- **Security Vulnerabilities**: 0 critical
- **Documentation Pages**: 10+ markdown files

---

## Dependencies & Risks

### Technical Dependencies
- Go 1.21+ (programming language)
- cloudflare/circl (post-quantum cryptography)
- Existing v11 UDP transport (proven stable)
- 3 bootstrap nodes (infrastructure dependency)

### Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| DHT convergence failure | Medium | High | Extensive 10K peer simulation + testnet validation |
| NAT traversal <85% success | High | Medium | UDP punch-through + relay fallback ensures 100% connectivity |
| Bootstrap nodes offline | Low | High | 3-5 nodes across regions, 99.9% uptime SLA |
| Performance regression | Low | High | Automated regression tests maintain v11 baseline (28.3 Mbps) |

### Blockers (None Currently)
No active blockers. v0.1.0-alpha achieved strong performance, architecture proven.

---

## Resources

### Code Repository
- **Language**: Go 1.21+
- **Architecture**: Layer 3 (TUN device) + UDP + PQC
- **Lines of Code**: ~8,000 (estimated for v0.2.0-alpha)

### Documentation
- **Architecture**: 4 documents (CURRENT_STATE, TARGET_STATE, KADEMLIA_DHT_ARCHITECTURE, MIGRATION_PATH)
- **Implementation**: 3 documents (TICKETS, DEPLOYMENT, TESTING)
- **Benchmarks**: 2 documents (performance, video streaming)

### Infrastructure
- **Bootstrap Nodes**: 3 nodes (US, EU, Asia) - $30-45/month
- **Test Nodes**: 3-4 machines for distributed testing
- **CI/CD**: GitHub Actions (free tier)

---

## Next Actions

### This Week (Week 1)
1. Begin TICKET-001: PeerID generation from ML-DSA-87 keys
2. Implement TICKET-002: XOR distance calculations
3. Design TICKET-003: PeerInfo data structures
4. Start TICKET-004: k-bucket LRU eviction logic

### Next Week (Week 2)
1. Complete TICKET-005: 256-bucket routing table
2. Define TICKET-006: UDP packet protocol format
3. Implement TICKET-007: PING/PONG handlers
4. Begin integration testing (3-node local network)

---

## Status Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Complete |
| 🔄 | In Progress |
| 📋 | Planned |
| ⏳ | Waiting/Blocked |
| ❌ | Blocker |

---

## Document Control

**Owner**: Technical Lead + Product Manager
**Review Frequency**: Weekly (every Monday)
**Change Log**:
- 2025-11-10: Initial consolidated roadmap created
- Next review: 2025-11-17 (after Sprint 0 Week 1)

---

**ShadowMesh** - Building the future of quantum-safe decentralized networking, one sprint at a time.

**Current Focus**: Kademlia DHT implementation for standalone operation (v0.2.0-alpha)
