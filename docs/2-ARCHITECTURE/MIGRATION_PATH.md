# ShadowMesh Migration Path

**From**: v11 (UDP+PQC) and v19 (QUIC) - Centralized Discovery
**To**: v20+ (Kademlia DHT + PQC + QUIC) - Standalone Decentralized

**Timeline**: 18 weeks (Sprint 0-16)
**Status**: Sprint 0 starting

---

## Migration Strategy

### Phase 1: Foundation (Sprint 0-2) - Weeks 1-6

**Goal**: Implement Kademlia DHT for decentralized peer discovery

#### Sprint 0: Architecture POC (Weeks 1-2)

**Research & Design**:
- [ ] Study Kademlia paper and existing implementations (libp2p, BitTorrent)
- [ ] Design PeerID generation from ML-DSA-87 public keys
- [ ] Define routing table structure (160 k-buckets, k=20)
- [ ] Plan bootstrap node strategy

**POC Deliverables**:
- [ ] Local 3-node DHT network (localhost testing)
- [ ] FIND_NODE operation working
- [ ] Routing table population successful
- [ ] Peer liveness probing implemented

**Success Criteria**:
- 3 nodes can discover each other via DHT
- FIND_NODE lookup completes in <500ms
- Routing table converges in <60 seconds

#### Sprint 1-2: Kademlia DHT Core (Weeks 3-6)

**DHT Operations**:
- [ ] FIND_NODE iterative lookup (α=3 parallel requests)
- [ ] STORE operation with TTL (24-hour default)
- [ ] FIND_VALUE with caching (10-minute cache)
- [ ] Routing table maintenance (LRU eviction, hourly liveness check)

**NAT Traversal Planning**:
- [ ] Research QUIC NAT traversal techniques
- [ ] Design relay fallback mechanism
- [ ] Plan UPnP integration (optional)

**Testing**:
- [ ] 5-10 node test network (multiple machines)
- [ ] DHT lookup success rate >95%
- [ ] Peer discovery latency <100ms
- [ ] Network partition recovery <5 minutes

**Deliverables**:
- `pkg/discovery/kademlia.go` - Complete DHT implementation
- `pkg/discovery/routing_table.go` - k-bucket routing table
- `pkg/discovery/peer_id.go` - PeerID generation and verification
- Integration tests for all DHT operations

---

### Phase 2: Integration (Sprint 3-4) - Weeks 7-10

**Goal**: Merge v11 (PQC) + v19 (QUIC) into unified v20 architecture

#### Sprint 3: QUIC + PQC Handshake (Weeks 7-8)

**PQC over QUIC**:
- [ ] Port ML-KEM-1024 key exchange to QUIC handshake
- [ ] Integrate ML-DSA-87 signatures for peer authentication
- [ ] Generate QUIC TLS certificates signed with ML-DSA-87
- [ ] Implement certificate pinning for peer verification

**Code Refactoring**:
- [ ] Extract PQC crypto into `pkg/crypto/pqc/`
- [ ] Create unified transport interface (`pkg/transport/`)
- [ ] Migrate TUN device logic to `pkg/layer3/`

**Testing**:
- [ ] Full PQC handshake over QUIC successful
- [ ] Benchmark handshake latency (<100ms target)
- [ ] Test with 2-node direct connection

**Deliverables**:
- `pkg/crypto/pqc/` - ML-KEM-1024, ML-DSA-87, hybrid mode
- `pkg/transport/quic.go` - QUIC transport with PQC
- `cmd/lightnode-l3-v20/` - Unified v20 implementation

#### Sprint 4: DHT + QUIC Integration (Weeks 9-10)

**Peer Discovery Flow**:
```
1. Node starts → Generate PeerID from ML-DSA-87 key
2. Connect to bootstrap nodes (DHT seed nodes)
3. Query DHT for peers close to own PeerID
4. Establish QUIC connections to discovered peers
5. Begin routing traffic through mesh
```

**Implementation**:
- [ ] Connect DHT operations to QUIC transport
- [ ] Implement peer exchange via DHT STORE/FIND_VALUE
- [ ] Peer metadata storage (IP, port, public keys, capabilities)
- [ ] Automatic peer connection based on routing table

**Testing**:
- [ ] End-to-end test: Node starts, discovers peers via DHT, connects via QUIC
- [ ] 5-node mesh network with zero configuration
- [ ] Peer discovery completes in <30 seconds
- [ ] Traffic routing successful through multiple hops

**Deliverables**:
- Fully integrated v20 client (DHT + PQC + QUIC + TUN)
- Zero-config operation demonstrated
- Performance baseline established

---

### Phase 3: Optimization (Sprint 5-8) - Weeks 11-14

**Goal**: Achieve performance targets (6-7 Gbps, <2ms latency)

#### Sprint 5-6: Performance Optimization (Weeks 11-12)

**Throughput Improvements**:
- [ ] Zero-copy packet handling (reduce memcpy overhead)
- [ ] SIMD acceleration for ChaCha20-Poly1305 (AVX2, NEON)
- [ ] Parallel QUIC streams (utilize multiple CPU cores)
- [ ] Buffer pool optimization (per-stream allocation)

**Latency Improvements**:
- [ ] Async I/O for TUN device (io_uring on Linux)
- [ ] DHT route caching (avoid repeated lookups)
- [ ] Pre-establish QUIC connections (connection pool)
- [ ] Reduce lock contention (lock-free data structures)

**Benchmarking**:
- [ ] iperf3 throughput tests (target: 1+ Gbps first, then 6-7 Gbps)
- [ ] ping latency tests (target: <5ms first, then <2ms)
- [ ] Packet loss monitoring (<1% target)

#### Sprint 7-8: Scalability & Reliability (Weeks 13-14)

**Multi-Peer Testing**:
- [ ] 10-node mesh network (concurrent connections)
- [ ] 100-node simulation (stress testing)
- [ ] Network partition recovery (split-brain scenarios)
- [ ] Peer churn handling (nodes joining/leaving frequently)

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
