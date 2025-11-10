# ShadowMesh Migration Path

**From**: v0.1.0-alpha (UDP+PQC, Centralized Discovery)
**To**: v0.2.0-alpha (Kademlia DHT + UDP + PQC, Standalone)
**Future**: v0.3.0+ (QUIC Migration)

**Timeline**: 8 weeks (4 weeks implementation + 4 weeks testing/release)
**Status**: Sprint 0 starting (Week 1 of 4)

**NOTE**: This migration path has been updated to reflect current project status.
See `PROJECT_STATUS.md` for the master roadmap.

---

## Migration Strategy

**Architecture Decision**: Conservative phased approach
- **v0.2.0-alpha**: Keep UDP transport (proven), add Kademlia DHT (4 weeks)
- **v0.3.0-alpha**: Migrate to QUIC transport (6 weeks, future)
- **v1.0.0**: Production features (6-12 months)

### Phase 1: Kademlia DHT Implementation (Weeks 1-4)

**Goal**: Replace centralized discovery with Kademlia DHT for standalone operation

#### Sprint 0: DHT Foundation (Weeks 1-2)

**Tickets**: TICKET-001 to TICKET-007

**Implementation**:
- [ ] TICKET-001: PeerID generation from ML-DSA-87 public keys
- [ ] TICKET-002: XOR distance calculations
- [ ] TICKET-003: PeerInfo data structures
- [ ] TICKET-004: k-bucket LRU eviction logic
- [ ] TICKET-005: 256-bucket routing table
- [ ] TICKET-006: UDP packet protocol (PING, PONG, FIND_NODE, STORE)
- [ ] TICKET-007: PING/PONG handlers for peer liveness

**Success Criteria**:
- PeerID generation: <1ms per operation
- XOR distance: Correct for 1000+ test cases
- k-bucket eviction: LRU working correctly
- Routing table: 256 buckets initialized
- PING/PONG: Round-trip <10ms on localhost

#### Sprint 1: DHT Operations (Weeks 3-4)

**Tickets**: TICKET-008 to TICKET-015

**Implementation**:
- [ ] TICKET-008: FIND_NODE iterative lookup (α=3 parallel requests)
- [ ] TICKET-009: Bootstrap node integration (3-5 seed nodes)
- [ ] TICKET-010: Integration with ML-DSA-87 keypairs
- [ ] TICKET-011: STORE/FIND_VALUE operations
- [ ] TICKET-012: NAT detection and peer reachability
- [ ] TICKET-013: Routing table persistence (save/load from disk)
- [ ] TICKET-014: Health monitoring and Prometheus metrics
- [ ] TICKET-015: Integration tests (3-node, 10-node networks)

**Testing**:
- [ ] 3-node local network converges in <60 seconds
- [ ] 10-node distributed network, DHT lookup success rate >95%
- [ ] FIND_NODE latency <500ms
- [ ] Bootstrap node connection <5 seconds

**Deliverables**:
- `pkg/discovery/` - Complete DHT implementation
- `pkg/discovery/dht.go` - Kademlia DHT core
- `pkg/discovery/routing_table.go` - k-bucket routing table
- `pkg/discovery/peer_id.go` - PeerID generation and verification
- `pkg/discovery/bootstrap.go` - Bootstrap node client
- 50+ integration tests passing

---

### Phase 2: Testing & Validation (Weeks 5-6)

**Goal**: Validate DHT implementation and prepare for standalone release

**Testing Activities**:

**Unit Tests** (200+ tests):
- [ ] PeerID generation and validation
- [ ] XOR distance calculations
- [ ] k-bucket LRU eviction
- [ ] Routing table operations
- [ ] DHT message encoding/decoding
- [ ] FIND_NODE iterative lookup

**Integration Tests** (50+ tests):
- [ ] 3-node local network (all nodes discover each other)
- [ ] 10-node distributed network (cross-region)
- [ ] NAT traversal scenarios
- [ ] Bootstrap node failover
- [ ] Network partition recovery
- [ ] Peer churn handling (nodes joining/leaving)

**Performance Tests**:
- [ ] Throughput: ≥25 Mbps (maintain v0.1.0-alpha baseline)
- [ ] Latency: <50ms overhead
- [ ] Packet loss: <5%
- [ ] DHT lookup latency: <500ms
- [ ] Peer discovery: <60 seconds to convergence

**E2E Standalone Operation**:
- [ ] Fresh install, no configuration, successful peer discovery
- [ ] Ping test between discovered peers
- [ ] Video streaming test (640x480 @ 547 kb/s)
- [ ] 3-hour stability test (zero packet loss)

---

### Phase 3: Documentation & Release (Weeks 7-8)

**Goal**: Prepare v0.2.0-alpha for public release

**Documentation**:

- [ ] README.md with quick start guide
- [ ] INSTALLATION.md for Linux (amd64, arm64) and macOS (arm64)
- [ ] ARCHITECTURE.md with DHT design overview
- [ ] TROUBLESHOOTING.md with common issues
- [ ] Performance benchmarks vs v0.1.0-alpha
- [ ] Migration guide from v0.1.0-alpha

**Release Preparation**:
- [ ] Build release binaries (Linux amd64/arm64, macOS arm64)
- [ ] Deploy 3 bootstrap nodes (US, EU, Asia)
- [ ] Set up CI/CD pipeline (GitHub Actions)
- [ ] Create release notes
- [ ] Tag v0.2.0-alpha in Git

**Bootstrap Infrastructure**:
- [ ] Provision 3 VPS instances (Linode/DigitalOcean)
- [ ] Deploy bootstrap node software
- [ ] Configure monitoring (Prometheus + Grafana)
- [ ] Set up health checks (UptimeRobot)
- [ ] Document runbooks for operations

---

## Future Phases (Post v0.2.0-alpha)

### v0.3.0-alpha: QUIC Migration (6 Weeks)

**Goal**: Replace UDP with QUIC for better NAT traversal and reliability

**Key Features**:
- QUIC transport with quic-go library
- 0-RTT reconnection
- Built-in congestion control
- Connection migration
- Improved NAT traversal

**Deliverables**:
- `pkg/transport/quic.go` - QUIC transport layer
- ML-KEM-1024 over QUIC handshake
- Performance benchmarks (target: ≥100 Mbps)

### v1.0.0: Production Release (6-12 Months)

**Goal**: Enterprise-ready quantum-safe DPN

**Key Features**:
- Multi-hop routing (3-5 hops)
- Traffic obfuscation (WebSocket mimicry)
- Atomic clock synchronization
- TPM/SGX attestation
- Mobile apps (iOS, Android)
- SOC 2 certification

**Performance Targets**:
- 6-7 Gbps throughput
- <2ms latency overhead
- 99.9% uptime

**Reliability**:
- [ ] QUIC connection migration (handle IP changes)
- [ ] Automatic reconnection on failure
- [ ] DHT routing table consistency (split-brain recovery)
- [ ] Graceful degradation (relay fallback for NAT issues)

**Monitoring**:
- [ ] Prometheus metrics (throughput, latency, packet loss)
- [ ] Grafana dashboards (optional)
- [ ] Built-in diagnostics (`shadowmesh-client --stats`)

---

### Phase 4: Hardening (Sprint 9-12) - Weeks 15-18

**Goal**: Production-ready security and stability

#### Sprint 9-10: Security Hardening (Weeks 15-16)

**Third-Party Security Audit**:
- [ ] Engage external security firm (budget permitting)
- [ ] Cryptography review (PQC implementation correctness)
- [ ] Protocol analysis (handshake, DHT operations)
- [ ] Penetration testing (NAT traversal, DHT poisoning)

**Internal Security Review**:
- [ ] Code review for security best practices
- [ ] Dependency vulnerability scan (`go mod why`, Dependabot)
- [ ] OWASP Top 10 compliance check
- [ ] Fuzzing (crypto functions, protocol parsing)

**Address Audit Findings**:
- [ ] Fix all critical and high-severity issues
- [ ] Document medium/low issues with mitigation plans
- [ ] Re-test after fixes

#### Sprint 11-12: Beta Release Preparation (Weeks 17-18)

**Documentation**:
- [ ] Update README.md with installation instructions
- [ ] Quick start guide (`docs/3-IMPLEMENTATION/QUICK_START.md`)
- [ ] Architecture documentation complete
- [ ] API documentation (if applicable)

**Packaging**:
- [ ] Build binaries for all platforms (Linux, macOS, Windows)
- [ ] Code signing (macOS, Windows)
- [ ] Create release archives with checksums
- [ ] Docker images (optional)

**Beta Testing**:
- [ ] Recruit 100+ beta testers
- [ ] Deploy to `beta-release/` directory
- [ ] Monitor feedback and bug reports
- [ ] Iterate on critical issues

**Beta Release** (v1.0.0-beta.1):
- [ ] All features complete
- [ ] Security audit passed
- [ ] Performance targets met (6-7 Gbps, <2ms latency)
- [ ] 99.9% uptime in beta testing
- [ ] Zero critical bugs

---

### Phase 5: Public Launch (Sprint 13-16) - Post-Beta

**Goal**: v1.0.0 production release on GitHub

#### Sprint 13-14: Release Candidate (Weeks 19-20)

**RC Build**:
- [ ] Address all beta feedback
- [ ] Final performance tuning
- [ ] Final security review
- [ ] Documentation polish

**RC Testing** (1-2 weeks):
- [ ] Full regression testing
- [ ] No new features (bug fixes only)
- [ ] Final performance benchmarks

#### Sprint 15-16: Public Launch (Weeks 21-22)

**GitHub Release**:
- [ ] Tag v1.0.0
- [ ] Publish binaries to GitHub Releases
- [ ] Update CHANGELOG.md
- [ ] Announcement blog post

**Community Launch**:
- [ ] Hacker News submission
- [ ] Reddit (r/programming, r/privacy, r/netsec)
- [ ] Twitter/social media announcement
- [ ] Dev.to article

**Post-Launch Monitoring**:
- [ ] Monitor GitHub issues
- [ ] Respond to community questions
- [ ] Track adoption metrics (GitHub stars, downloads)

---

## Sprint-by-Sprint Comparison

| Sprint | Weeks | Goal | Current State | Target State | Gap |
|--------|-------|------|---------------|--------------|-----|
| **0** | 1-2 | DHT POC | Centralized discovery | Local 3-node DHT | Research & POC |
| **1-2** | 3-6 | DHT Core | No DHT | Full Kademlia | Implementation |
| **3** | 7-8 | PQC + QUIC | v11 (PQC) + v19 (QUIC) separate | Unified v20 | Integration |
| **4** | 9-10 | DHT Integration | Separate components | End-to-end mesh | Connection |
| **5-6** | 11-12 | Performance | 100 Mbps, 50ms | 1+ Gbps, <5ms | Optimization |
| **7-8** | 13-14 | Scalability | 3-5 nodes tested | 10-100 nodes | Stress testing |
| **9-10** | 15-16 | Security | No audit | Audit passed | Hardening |
| **11-12** | 17-18 | Beta | Alpha only | v1.0.0-beta.1 | Release prep |
| **13-14** | 19-20 | RC | Beta feedback | v1.0.0-rc.1 | Stabilization |
| **15-16** | 21-22 | Launch | Private repo | v1.0.0 public | GitHub release |

---

## Critical Path

### Must-Have for v1.0.0

✅ **Kademlia DHT**: Peer discovery without central servers
✅ **PQC + QUIC**: Quantum-safe, reliable transport
✅ **Standalone Operation**: Zero-config from first boot
✅ **Security Audit**: Third-party review passed
✅ **Performance**: 6-7 Gbps throughput, <2ms latency

### Nice-to-Have (Future Releases)

⏳ **Traffic Obfuscation**: QUIC mimicry, cover traffic (v1.1.0)
⏳ **Multi-Hop Routing**: 3-5 hop onion routing (v1.2.0)
⏳ **Zero-Trust Exit Nodes**: TPM/SGX attestation (v1.3.0)
⏳ **Mobile Clients**: iOS, Android apps (v2.0.0)
⏳ **Blockchain Governance**: Smart contract relay verification (v2.1.0)

---

## Risk Mitigation Timeline

### Sprint 0-2: Technical Risks

**Risk**: DHT implementation too complex
- **Mitigation**: Use libp2p Kademlia as reference, start with simple POC
- **Timeline**: Assess by end of Sprint 0 (Week 2)

### Sprint 3-4: Integration Risks

**Risk**: v11 + v19 merge fails (incompatible architectures)
- **Mitigation**: Refactor common code into shared packages first
- **Timeline**: Assess by end of Sprint 3 (Week 8)

### Sprint 5-8: Performance Risks

**Risk**: Performance targets not met (6-7 Gbps, <2ms latency)
- **Mitigation**: Profiling, SIMD optimization, kernel bypass (io_uring)
- **Fallback**: Reduce targets to 1 Gbps, <5ms for v1.0.0, optimize later
- **Timeline**: Assess by end of Sprint 6 (Week 12)

### Sprint 9-10: Security Risks

**Risk**: Security audit finds critical vulnerabilities
- **Mitigation**: Address all critical issues before beta release
- **Fallback**: Delay beta release by 1-2 sprints if needed
- **Timeline**: Audit complete by end of Sprint 10 (Week 16)

---

## Success Metrics

### Sprint 0 (Week 2)

- [ ] 3-node local DHT network operational
- [ ] FIND_NODE lookup successful
- [ ] Routing table population demonstrated

### Sprint 2 (Week 6)

- [ ] 5-10 node distributed DHT network operational
- [ ] DHT lookup success rate >95%
- [ ] Peer discovery latency <100ms

### Sprint 4 (Week 10)

- [ ] End-to-end encrypted mesh network (DHT + PQC + QUIC)
- [ ] 5 nodes can discover and connect without configuration
- [ ] Traffic routing successful

### Sprint 6 (Week 12)

- [ ] Throughput: 1+ Gbps (milestone toward 6-7 Gbps)
- [ ] Latency: <5ms overhead (milestone toward <2ms)
- [ ] Packet loss: <1%

### Sprint 10 (Week 16)

- [ ] Security audit passed (no critical/high issues)
- [ ] All DHT operations tested under adversarial conditions
- [ ] PQC implementation verified correct

### Sprint 12 (Week 18)

- [ ] Beta release (v1.0.0-beta.1) published
- [ ] 100+ beta testers recruited
- [ ] 99.9% uptime demonstrated (1000+ hours)

### Sprint 16 (Week 22)

- [ ] v1.0.0 production release on GitHub
- [ ] 1000+ GitHub stars
- [ ] 100+ production deployments
- [ ] 10+ community contributions

---

## Migration Checklist

### Pre-Sprint 0

- [x] Discovery nodes shut down (November 8, 2025)
- [x] Alpha builds archived (`.archive/alpha-builds/`)
- [x] Documentation reorganized (4-tier structure)
- [x] Roadmap published (`docs/1-PRODUCT/ROADMAP.md`)
- [ ] Team onboarded to BMAD methodology

### During Sprint 0-2

- [ ] DHT research and design complete
- [ ] Kademlia implementation 100% functional
- [ ] Bootstrap node strategy defined
- [ ] Testing framework established

### During Sprint 3-4

- [ ] v11 + v19 code merged into v20
- [ ] PQC handshake over QUIC working
- [ ] DHT + QUIC integration complete
- [ ] End-to-end testing successful

### During Sprint 5-8

- [ ] Performance optimization complete
- [ ] Scalability testing (10-100 nodes) passed
- [ ] Monitoring and diagnostics implemented
- [ ] Reliability testing (24+ hour runs) passed

### During Sprint 9-12

- [ ] Security audit complete and issues addressed
- [ ] Beta testing (100+ users) successful
- [ ] Documentation complete
- [ ] Packaging and release automation ready

### During Sprint 13-16

- [ ] Release candidate tested (no critical bugs)
- [ ] v1.0.0 published to GitHub
- [ ] Community launched (Hacker News, Reddit, etc.)
- [ ] Post-launch support established

---

## Conclusion

**Migration Path Summary**:
- **18 weeks** from centralized discovery to standalone DHT
- **5 phases**: Foundation → Integration → Optimization → Hardening → Launch
- **16 sprints** with clear milestones and success criteria

**Key Transitions**:
1. Centralized discovery → Kademlia DHT (Sprint 0-2)
2. Separate v11/v19 → Unified v20 (Sprint 3-4)
3. Alpha quality → Production quality (Sprint 5-12)
4. Private testing → Public release (Sprint 13-16)

**End State**: Fully decentralized, quantum-safe, high-performance VPN with zero infrastructure dependencies and $0 operational cost.

---

**Document Status**: ✅ COMPLETE
**Last Updated**: November 8, 2025
**Next Review**: After each sprint completion
