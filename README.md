<h1>
  <img src="https://pbs.twimg.com/profile_images/1969957304679473152/QW21M-FO_400x400.jpg" alt="Chronara Group Logo" height="60" style="vertical-align: middle; margin-right: 15px;"/>
  Chronara Group ShadowMesh
</h1>

**Post-Quantum Decentralized Private Network (DPN)**

[![Version](https://img.shields.io/badge/version-0.1.0--alpha-blue.svg)](https://github.com/CG-8663/shadowmesh/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Security](https://img.shields.io/badge/Security-Policy-red)](SECURITY.md)
[![Code of Conduct](https://img.shields.io/badge/Code%20of-Conduct-blue)](CODE_OF_CONDUCT.md)

---

> **📋 MASTER ROADMAP (Single Source of Truth):**
> **[PROJECT_STATUS.md](PROJECT_STATUS.md)** - Complete project status, roadmap, timeline, and metrics
> **[STATUS.md](STATUS.md)** - Quick 30-second status overview

---

## Project Status

**Current Version**: v0.1.0-alpha (Released)
**Next Version**: v0.2.0-alpha (In Development - DHT Migration)
**Status**: Active Development - Sprint 0 (Week 1 of 4)
**Last Updated**: November 10, 2025

### Recent Achievements (v0.1.0-alpha)
- ✅ 28.3 Mbps throughput (45% faster than Tailscale)
- ✅ Video streaming successful (640x480 @ 547 kb/s)
- ✅ 3-hour stability test, zero packet loss
- ✅ Post-quantum cryptography (ML-KEM-1024, ML-DSA-87)

### Current Focus (v0.2.0-alpha)
- 🔄 Kademlia DHT implementation (4-week sprint)
- 🔄 Standalone operation (no discovery server dependency)
- 🔄 3 bootstrap nodes for network entry

**Tested Platforms**:
- Linux (amd64, arm64)
- macOS (arm64)
- Raspberry Pi (ARM64)

**Known Limitations**:
- Alpha development phase
- No formal security audit
- DHT implementation in progress (standalone operation coming soon)

---

ShadowMesh is a fully decentralized, quantum-safe DPN (Decentralized Private Network) implementing NIST-standardized post-quantum cryptography. Unlike proxy-based VPN services with centralized servers, ShadowMesh uses Kademlia DHT for peer discovery, eliminating all central dependencies and proxy infrastructure.

**v0.1.0-alpha** achieved strong performance with UDP transport and post-quantum crypto. **v0.2.0-alpha** (in development) adds Kademlia DHT for standalone, fully decentralized operation.

## Implementation Features

**Current (v0.1.0-alpha)**:
- **Post-Quantum Cryptography**: ML-KEM-1024 (NIST FIPS 203), ML-DSA-87 (NIST FIPS 204)
- **UDP Transport**: Proven 28.3 Mbps throughput with low latency
- **Layer 3 Networking**: TUN device for IP-level routing
- **Symmetric Encryption**: ChaCha20-Poly1305 (IETF RFC 8439)
- **Hybrid Mode**: Classical (X25519, Ed25519) + PQC for defense in depth
- **Forward Secrecy**: Ephemeral session keys with automatic rotation

**In Development (v0.2.0-alpha)**:
- **Kademlia DHT**: Decentralized peer discovery (zero central servers)
- **PeerID Generation**: Derived from ML-DSA-87 public keys (SHA256)
- **Standalone Operation**: No infrastructure dependencies, 3 bootstrap nodes only

**Planned (v0.3.0-alpha+)**:
- **QUIC Transport**: Reliable, low-latency stream protocol with better NAT traversal
- **Multi-hop Routing**: 3-5 configurable hops for enhanced privacy
- **Traffic Obfuscation**: WebSocket mimicry and cover traffic

## 🚀 Current Development Focus

**Active Development (v0.2.0-alpha)**: Kademlia DHT Implementation for Standalone Operation

ShadowMesh is transitioning from centralized discovery to fully decentralized peer-to-peer architecture:

**Current Sprint (Sprint 0, Week 1 of 4)**:
- **Kademlia DHT**: Decentralized peer discovery (replacing centralized discovery server)
- **PeerID Generation**: Derived from ML-DSA-87 public keys for cryptographic verification
- **Bootstrap Nodes**: 3-5 seed nodes for network entry (US, EU, Asia)
- **UDP Transport**: Keeping proven v0.1.0-alpha transport (28.3 Mbps baseline)

**Future Work**:
- **v0.3.0-alpha** (6 weeks): QUIC transport migration for better NAT traversal
- **v1.0.0** (6-12 months): Production features (multi-hop routing, traffic obfuscation, TPM attestation)

---

### 📋 Full Roadmap & Documentation

**👉 [PROJECT_STATUS.md](PROJECT_STATUS.md) - MASTER ROADMAP (Single Source of Truth)**

This document contains:
- 8-week v0.2.0-alpha timeline with 15 implementation tickets
- Current sprint progress and weekly updates
- Performance targets and success criteria
- Long-term vision (v0.3.0-alpha, v1.0.0)

**Supporting Documentation**:
- [STATUS.md](STATUS.md) - Quick 30-second status overview
- [Kademlia DHT Architecture](docs/2-ARCHITECTURE/KADEMLIA_DHT_ARCHITECTURE.md) - Complete DHT design
- [Bootstrap Node Deployment](docs/4-OPERATIONS/BOOTSTRAP_NODE_DEPLOYMENT.md) - Infrastructure plan
- [Migration Path](docs/2-ARCHITECTURE/MIGRATION_PATH.md) - 8-week migration timeline
- Implementation tickets and testing strategy (internal documentation - available at release)

---

**No Public Releases Yet**: Pre-built binaries will be available when v0.2.0-alpha is released (target: December 8, 2025).

## 🏗️ Architecture

### Client Daemon

The client daemon provides:
- **TAP Device Management**: Layer 2 Ethernet frame capture/injection
- **PQC Handshake**: 4-message protocol (HELLO → CHALLENGE → RESPONSE → ESTABLISHED)
- **WebSocket Connection**: Auto-reconnect with exponential backoff
- **Frame Encryption Pipeline**: ChaCha20-Poly1305 with counter-based nonces
- **Key Rotation**: Automatic re-handshake at configurable intervals
- **Statistics Reporting**: Real-time metrics on frames sent/received

### Protocol Stack

```
┌─────────────────────────────────────────┐
│         Application Layer               │
│   (Configuration, Key Management)       │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│      Handshake Layer (PQC)              │
│  ML-KEM-1024 + ML-DSA-87 + X25519       │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│      Session Layer                      │
│  HKDF Key Derivation (TX/RX Keys)       │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│    Encryption Layer                     │
│  ChaCha20-Poly1305 (Frame Encryption)   │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│      Transport Layer                    │
│   WebSocket over TLS 1.3                │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│      Data Link Layer                    │
│   TAP Device (Ethernet Frames)          │
└─────────────────────────────────────────┘
```

## 📁 Repository Structure

```
shadowmesh/
├── client/
│   ├── daemon/              # Client daemon (COMPLETE)
│   │   ├── main.go          # Main entry point with signal handling
│   │   ├── config.go        # YAML configuration management
│   │   ├── connection.go    # WebSocket connection manager
│   │   ├── handshake.go     # PQC handshake orchestrator
│   │   ├── tap.go           # TAP device integration
│   │   └── tunnel.go        # Frame encryption/decryption pipeline
│   └── cli/                 # CLI tool (stub)
├── relay/
│   └── server/              # Relay server (IN PROGRESS)
├── shared/
│   ├── crypto/              # Cryptography library (COMPLETE)
│   │   ├── keyexchange.go   # ML-KEM-1024 + X25519 hybrid KEM
│   │   ├── signature.go     # ML-DSA-87 + Ed25519 hybrid signatures
│   │   └── symmetric.go     # ChaCha20-Poly1305 frame encryption
│   └── protocol/            # Wire protocol (COMPLETE)
│       ├── types.go         # Message type definitions
│       ├── header.go        # Header encoding/decoding
│       ├── messages.go      # Message serialization (13 types)
│       └── handshake.go     # Handshake state machine
├── contracts/               # Smart contracts (Solidity)
├── test/
│   └── integration/         # Integration tests (COMPLETE)
└── docs/                    # Documentation
```

## 🔐 Security

### Post-Quantum Cryptography

- **ML-KEM-1024 (Kyber)**: NIST Security Level 5 - Key encapsulation
- **ML-DSA-87 (Dilithium)**: NIST Security Level 5 - Digital signatures
- **Hybrid Mode**: Classical algorithms (X25519, Ed25519) run in parallel

### Performance Targets

- **Latency overhead**: <2ms for encryption/decryption
- **Throughput**: 1+ Gbps on single CPU core
- **Memory**: <100 MB per connection
- **CPU**: <5% for 100 Mbps sustained traffic

### Security Audit Status

- ⏳ Pending third-party security audit
- ⏳ Pending formal verification of protocol
- ✅ Using NIST-standardized PQC algorithms
- ✅ Comprehensive unit tests and integration tests

## 📊 Development Roadmap

**Current Focus**: Sprint 0-2 - Kademlia DHT + PQC + QUIC Integration

ShadowMesh is transitioning to a fully decentralized architecture with standalone peer discovery:

### Sprint 0: Architecture POC (Weeks 1-2)
- [ ] Kademlia DHT research and design
- [ ] Bootstrap node strategy definition
- [ ] PeerID generation from ML-DSA-87 keys
- [ ] Local DHT testing (3-5 nodes)

### Sprint 1-2: Kademlia DHT Core (Weeks 3-6)
- [ ] FIND_NODE iterative lookup
- [ ] STORE operation with TTL
- [ ] FIND_VALUE with caching
- [ ] Routing table management (k-buckets)
- [ ] NAT traversal integration

### Sprint 3-4: QUIC + PQC Integration (Weeks 7-10)
- [ ] Merge v19 (QUIC) + v11 (PQC)
- [ ] ML-KEM-1024 key exchange over QUIC
- [ ] ML-DSA-87 signatures for peer authentication
- [ ] ChaCha20-Poly1305 over QUIC streams
- [ ] Layer 3 TUN device with QUIC

### Sprint 5+: Standalone Operation (Weeks 11-18)
- [ ] Zero central dependencies
- [ ] Bootstrap from hardcoded peer list
- [ ] Peer exchange via DHT gossip
- [ ] Performance optimization (6-7 Gbps target)
- [ ] Security audit preparation
- [ ] Beta release (v1.0.0-beta.1)

**Full Roadmap**: See **[PROJECT_STATUS.md](PROJECT_STATUS.md)** for 8-week v0.2.0-alpha timeline and long-term vision.

**Historical Achievements**: See [docs/benchmarks/](docs/benchmarks/) for v0.1.0-alpha performance results and [docs/archive/](docs/archive/) for development history.

## 🧪 Testing

### Unit and Integration Tests

```bash
# Run all tests
go test ./...

# Run crypto tests with benchmarks
go test -bench=. ./shared/crypto/

# Run protocol tests
go test -v ./shared/protocol/

# Run integration tests
go test -v ./test/integration/

# Generate coverage report
go test -cover -coverprofile=coverage.txt ./...
go tool cover -html=coverage.txt
```

### Performance Testing (Production Network)

**Quick Performance Test** (5 minutes):
```bash
./scripts/quick-perf-test.sh
```

**ShadowMesh vs Tailscale Comparison** (15 minutes):
```bash
./scripts/compare-tailscale-shadowmesh.sh
```

**What Gets Tested**:
- ✅ Latency measurements (min/avg/max/jitter)
- ✅ Packet loss rates
- ✅ TCP throughput (single and parallel streams)
- ✅ Large packet handling (MTU testing)
- ✅ Side-by-side comparison with Tailscale

**See**: [docs/performance/PERFORMANCE_TESTING.md](docs/performance/PERFORMANCE_TESTING.md) for comprehensive testing guide

**Results**: See [docs/performance/PERFORMANCE_RESULTS.md](docs/performance/PERFORMANCE_RESULTS.md) and [docs/performance/PRODUCTION_VALIDATION_REPORT.md](docs/performance/PRODUCTION_VALIDATION_REPORT.md) for proven benchmarks showing **ShadowMesh outperforms Tailscale** by 30% on latency!

## 🛠️ Build Commands

**Note**: No public builds until v1.0.0 release. Development builds for contributors only.

```bash
# Run tests (development)
make test
go test ./...

# Format code
make fmt
go fmt ./...

# Run linter
make lint
golangci-lint run

# View all commands
make help
```

**For Contributors**: See [docs/3-IMPLEMENTATION/DEVELOPMENT_GUIDELINES.md](docs/3-IMPLEMENTATION/DEVELOPMENT_GUIDELINES.md) for development setup.

## 📖 Documentation

### 🏗️ Architecture
- **[shared/protocol/PROTOCOL_SPEC.md](shared/protocol/PROTOCOL_SPEC.md)** - Wire protocol specification
- **[docs/architecture/PROJECT_SPEC.md](docs/architecture/PROJECT_SPEC.md)** - Technical specifications
- **[docs/architecture/ENHANCED_SECURITY_SPECS.md](docs/architecture/ENHANCED_SECURITY_SPECS.md)** - Advanced security features
- **[docs/architecture/ZERO_TRUST_ARCHITECTURE.md](docs/architecture/ZERO_TRUST_ARCHITECTURE.md)** - Zero-trust design
- **[docs/architecture/SITE_TO_SITE_VPN_CONFIG.md](docs/architecture/SITE_TO_SITE_VPN_CONFIG.md)** - Site-to-site DPN configuration
- **[docs/architecture/AWS_S3_KMS_TERRAFORM.md](docs/architecture/AWS_S3_KMS_TERRAFORM.md)** - Cloud infrastructure templates

## Potential Applications

This experimental implementation may be suitable for:

- Research and evaluation of post-quantum DPN protocols
- Investigation of decentralized private network architectures
- Testing quantum-resistant cryptography in network contexts
- Educational purposes and cryptography study
- Proof-of-concept deployments (non-production)

**Not Recommended For** (at this stage):
- Production enterprise deployments
- Mission-critical communications
- Environments requiring security certifications
- Use cases requiring guaranteed uptime or support

**DPN vs VPN**: ShadowMesh is a **Decentralized Private Network (DPN)**, not a traditional VPN service. Key differences:
- ❌ No proxy servers routing your traffic
- ❌ No centralized infrastructure to compromise
- ❌ No trust required in third-party VPN providers
- ✅ Peer-to-peer encrypted tunnels only
- ✅ Full decentralization with Kademlia DHT
- ✅ Post-quantum cryptographic security

## 🤝 Contributing

We welcome contributions from the community! ShadowMesh is open source and benefits from diverse perspectives.

### How to Contribute

1. **Read** [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines
2. **Check** existing issues or create a new one
3. **Fork** the repository and create your feature branch
4. **Write** tests for new functionality
5. **Submit** a pull request

### What We're Looking For

- 🐛 Bug fixes and improvements
- 📚 Documentation enhancements
- 🧪 Test coverage improvements
- 🚀 Client performance optimizations
- 🔧 Platform support (Windows, macOS, ARM)

### Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Security Considerations

**Alpha Status**: This software is experimental DPN (Decentralized Private Network) technology and has not undergone independent security audit. Use at your own risk.

- **Reporting vulnerabilities**: See [SECURITY.md](SECURITY.md) for responsible disclosure
- **Do NOT** open public issues for security vulnerabilities
- **Contact**: projectsupernode@chronara.io for security-related matters

**Known Security Limitations**:
- No formal security audit completed
- Limited peer review of implementation
- Alpha software may contain vulnerabilities
- Not recommended for protecting sensitive data at this time

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

**Note**: This license applies to the client code in this repository. The relay server implementation is proprietary.

## 🙏 Acknowledgments

ShadowMesh builds upon:
- **NIST Post-Quantum Cryptography Standardization**
- **Cloudflare's CIRCL library** (PQC implementations)
- **WireGuard protocol design** (inspiration)
- **Go standard library crypto** (classical algorithms)

## 📞 Support

- **Documentation**: See `/docs` directory and wiki
- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and ideas
- **Security**: projectsupernode@chronara.io

---

**ShadowMesh** - An experimental post-quantum Decentralized Private Network (DPN). Contributions and feedback welcome.
