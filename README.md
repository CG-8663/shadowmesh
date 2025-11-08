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

## Project Status

**Current Version**: 0.1.0-alpha
**Status**: Alpha - Active Development
**Last Updated**: November 2025

ShadowMesh is currently in alpha testing. The client implementation is feature-complete for core functionality, but has not undergone independent security audit. We welcome community feedback and contributions.

**Tested Platforms**:
- Linux (Debian, Ubuntu)
- Raspberry Pi (ARM64)
- Cloud VMs (UpCloud, Proxmox)

**Known Limitations**:
- Alpha software - expect breaking changes
- Relay server not included (proprietary)
- No formal security audit completed
- Limited platform testing (Linux only)

---

ShadowMesh is an experimental Decentralized Private Network (DPN) client implementing NIST-standardized post-quantum cryptography. Unlike traditional VPNs with centralized trust models, ShadowMesh explores a decentralized architecture where relay nodes are verified via blockchain attestation. The project integrates quantum-resistant algorithms (ML-KEM-1024, ML-DSA-87) in a practical Layer 2 implementation using TAP devices for Ethernet-level encryption.

This project is designed to investigate post-quantum DPN architectures and gather community feedback on the decentralized trust model.

## Implementation Features

- **Post-Quantum Cryptography**: Implements ML-KEM-1024 (NIST FIPS 203) and ML-DSA-87 (NIST FIPS 204)
- **Hybrid Mode**: Combines post-quantum algorithms with classical X25519/Ed25519
- **Layer 2 DPN**: TAP device implementation for Ethernet frame encryption
- **Decentralized Trust**: Blockchain-based relay node verification (planned)
- **Symmetric Encryption**: ChaCha20-Poly1305 (IETF RFC 8439)
- **Key Rotation**: Configurable rotation intervals (10 seconds to 1 hour)
- **Transport**: WebSocket over TLS 1.3
- **Forward Secrecy**: Ephemeral session keys
- **Replay Protection**: Monotonic counter-based frame numbering

## 🚀 Current Development Focus

**Active Development**: Kademlia DHT + PQC + QUIC Integration

ShadowMesh is transitioning from centralized discovery to a fully decentralized peer-to-peer architecture:

- **Standalone DHT**: Kademlia implementation for peer discovery (zero central dependencies)
- **QUIC Transport**: Stream-based protocol replacing WebSocket
- **PQC Integration**: Merging ML-KEM-1024 + ML-DSA-87 with QUIC transport
- **Layer 3 Networking**: TUN device for IP-level routing

**No Public Releases Yet**: Pre-built binaries will be available when v1.0.0 is published to GitHub Releases.

For development roadmap and architecture details, see:
- [18-Week Roadmap](docs/1-PRODUCT/ROADMAP.md)
- [Architecture Overview](docs/2-ARCHITECTURE/README.md)
- [Development Guidelines](docs/3-IMPLEMENTATION/DEVELOPMENT_GUIDELINES.md)

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

**Full Roadmap**: See [docs/1-PRODUCT/ROADMAP.md](docs/1-PRODUCT/ROADMAP.md) for 18-week detailed plan.

**Historical Achievements**: See [docs/archive/](docs/archive/) for completed Epic 1, Epic 2, and v11 reports.

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

```bash
# Build client only
make build-client

# Build relay server
make build-relay

# Build all components (client + relay)
make build

# Install client to /usr/local/bin
sudo make install-client

# Run relay server (requires config and keys)
./build/shadowmesh-relay --gen-keys    # Generate relay keys
./build/shadowmesh-relay --show-config # View configuration
sudo ./build/shadowmesh-relay          # Start relay server

# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# View all commands
make help
```

## 📖 Documentation

### 🚀 Getting Started
- **[docs/guides/GETTING_STARTED.md](docs/guides/GETTING_STARTED.md)** - Complete getting started guide
- **[docs/guides/QUICK_REFERENCE.md](docs/guides/QUICK_REFERENCE.md)** - Common commands cheatsheet
- **[docs/guides/NEXT_STEPS.md](docs/guides/NEXT_STEPS.md)** - Roadmap and next actions

### 📦 Deployment
- **[docs/deployment/INSTALL.md](docs/deployment/INSTALL.md)** - Installation guide
- **[docs/deployment/STAGE_TESTING.md](docs/deployment/STAGE_TESTING.md)** - Local testing (localhost)
- **[docs/deployment/DISTRIBUTED_TESTING.md](docs/deployment/DISTRIBUTED_TESTING.md)** - Cloud testing (UpCloud + Proxmox)
- **[docs/deployment/UPCLOUD_DEPLOYMENT.md](docs/deployment/UPCLOUD_DEPLOYMENT.md)** - Automated UpCloud deployment
- **[docs/deployment/UPDATE_CLIENTS.md](docs/deployment/UPDATE_CLIENTS.md)** - Update client scripts

### 🏗️ Architecture
- **[shared/protocol/PROTOCOL_SPEC.md](shared/protocol/PROTOCOL_SPEC.md)** - Wire protocol specification
- **[docs/architecture/PROJECT_SPEC.md](docs/architecture/PROJECT_SPEC.md)** - Technical specifications
- **[docs/architecture/ENHANCED_SECURITY_SPECS.md](docs/architecture/ENHANCED_SECURITY_SPECS.md)** - Advanced security features
- **[docs/architecture/ZERO_TRUST_ARCHITECTURE.md](docs/architecture/ZERO_TRUST_ARCHITECTURE.md)** - Zero-trust design
- **[docs/architecture/SITE_TO_SITE_VPN_CONFIG.md](docs/architecture/SITE_TO_SITE_VPN_CONFIG.md)** - Site-to-site VPN setup
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

**Alpha Status**: This software is experimental DPN technology and has not undergone independent security audit. Use at your own risk.

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
