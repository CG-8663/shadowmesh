# ShadowMesh Beta Release

**Purpose**: Internal beta testing and validation before public release

**Audience**: Beta testers, internal team, early adopters

**Status**: BETA - Not for production use

---

## 📁 Directory Structure

```
beta-release/
├── bin/                    # Compiled binaries for testing
├── config/                 # Beta configuration templates
├── docs/                   # Beta-specific documentation
├── scripts/                # Deployment and testing scripts
└── tests/                  # Beta test cases and results
```

---

## 🎯 Beta Testing Objectives

### Phase 1: Core Functionality (Weeks 1-2)
- [ ] QUIC transport layer stability
- [ ] PQC key exchange (ML-KEM-1024)
- [ ] PQC signatures (ML-DSA-87)
- [ ] Basic peer connectivity
- [ ] ChaCha20-Poly1305 encryption

### Phase 2: DHT Integration (Weeks 3-4)
- [ ] Kademlia DHT peer discovery
- [ ] FIND_NODE operations
- [ ] STORE/FIND_VALUE functionality
- [ ] Routing table maintenance
- [ ] NAT traversal

### Phase 3: Performance & Stability (Weeks 5-6)
- [ ] Throughput benchmarks (target: 1+ Gbps)
- [ ] Latency testing (target: <5ms overhead)
- [ ] Connection stability (24+ hour tests)
- [ ] Multi-peer mesh networking
- [ ] Failure recovery scenarios

---

## 🔧 Beta Build Process

### Building Beta Binaries

```bash
# Build for macOS (ARM64)
GOOS=darwin GOARCH=arm64 go build -o beta-release/bin/shadowmesh-client-darwin-arm64 cmd/lightnode-l3-v20/main.go

# Build for Linux (AMD64) - production servers
GOOS=linux GOARCH=amd64 go build -o beta-release/bin/shadowmesh-client-linux-amd64 cmd/lightnode-l3-v20/main.go

# Build for Windows (AMD64)
GOOS=windows GOARCH=amd64 go build -o beta-release/bin/shadowmesh-client-windows-amd64.exe cmd/lightnode-l3-v20/main.go
```

### Versioning
- Beta versions: `v0.X.0-beta.N` (e.g., `v0.3.0-beta.1`)
- Tag format: `git tag v0.3.0-beta.1`

---

## 📋 Beta Test Checklist

### Pre-Deployment
- [ ] All unit tests passing
- [ ] Integration tests passing
- [ ] Security audit completed
- [ ] Performance benchmarks meet targets
- [ ] Documentation updated
- [ ] Known issues documented

### Deployment
- [ ] Beta binaries compiled for all platforms
- [ ] Configuration templates prepared
- [ ] Test environment provisioned
- [ ] Monitoring and logging enabled
- [ ] Beta tester access credentials distributed

### Testing
- [ ] Functional testing (all features)
- [ ] Performance testing (throughput, latency)
- [ ] Stability testing (long-running connections)
- [ ] Security testing (penetration tests)
- [ ] Usability testing (user feedback)
- [ ] Edge case testing (network failures, etc.)

### Post-Testing
- [ ] Test results documented
- [ ] Issues logged and prioritized
- [ ] User feedback collected
- [ ] Performance metrics analyzed
- [ ] Go/no-go decision for first release

---

## 🐛 Known Issues

### Current Beta Limitations
- DHT implementation incomplete (15% complete)
- QUIC + PQC integration not yet merged
- Centralized backbone dependency still exists
- Limited NAT traversal testing

### Planned Fixes
See `.internal/planning/sprint-XX-plan.md` for detailed roadmap.

---

## 📊 Beta Metrics

### Performance Targets
- **Throughput**: 1+ Gbps (target: 6-7 Gbps)
- **Latency**: <5ms overhead (target: <2ms)
- **Uptime**: 99%+ (target: 99.9%)
- **Connection Success**: 95%+ (target: 99%)

### Test Coverage
- **Unit Tests**: 80%+ (target: 90%+)
- **Integration Tests**: 70%+ (target: 80%+)
- **E2E Tests**: 60%+ (target: 70%+)

---

## 🔐 Security Notice

**Beta Access**: Restricted to approved beta testers only

**Data Handling**: Beta environment may contain test data only - no production secrets

**Reporting**: Security issues must be reported to security@shadowmesh.io (internal team)

---

## 📞 Beta Support

**Internal Team**: See `.internal/team/` for contact information

**Issue Tracking**: GitHub Issues (private repository during beta)

**Documentation**: See `docs/` for full technical documentation

---

## 🚀 Beta to First Release Transition

### Graduation Criteria
1. All critical bugs resolved
2. Performance targets met
3. Security audit passed
4. 100+ hours of stable operation
5. Positive feedback from 10+ beta testers
6. Documentation complete and reviewed

### Transition Process
1. Freeze beta features
2. Final testing and validation
3. Documentation finalization
4. Security audit
5. Version bump to `v1.0.0-rc.1`
6. Move to `first-release/` directory
7. Prepare for public GitHub launch

---

**Last Updated**: November 8, 2025
**Current Version**: v0.3.0-beta.1 (planned)
**Status**: Beta testing in progress
