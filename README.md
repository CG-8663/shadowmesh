# ShadowMesh - Post-Quantum Decentralized Private Network (DPN)

[![Version](https://img.shields.io/badge/version-0.2.0--alpha%20MVP-blue.svg)](https://github.com/CG-8663/shadowmesh/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://golang.org)
[![Solidity](https://img.shields.io/badge/Solidity-0.8.20+-363636?logo=solidity)](https://soliditylang.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

---

ShadowMesh is a fully decentralized, quantum-safe DPN (Decentralized Private Network) implementing NIST-standardized post-quantum cryptography. Unlike proxy-based VPN services with centralized servers, ShadowMesh uses **Kademlia DHT + Ethereum smart contracts** for peer/relay discovery, eliminating all central dependencies.

**Current Status**: MVP Development (Epic 1: Foundation & Cryptography)

---

## Features

### Core Technology
- ✅ **Post-Quantum Cryptography**: ML-KEM-1024 (key exchange) + ML-DSA-87 (signatures)
- ✅ **Hybrid Mode**: Classical (X25519, Ed25519) + PQC for defense-in-depth
- ✅ **Layer 2 Networking**: TAP devices with Ethernet frame encryption
- ✅ **WebSocket Secure (WSS)**: Traffic obfuscation for censorship resistance
- ✅ **Smart Contract**: Ethereum relay registry (chronara.eth) with staking/slashing
- ✅ **Monitoring**: Grafana + Prometheus with 3 pre-configured dashboards
- ✅ **Database**: PostgreSQL with user/device management and audit logs
- ✅ **Public Network Map**: React + Leaflet.js visualization

### Security
- **NIST-Standardized PQC**: First DPN with FIPS 203 (ML-KEM) and FIPS 204 (ML-DSA)
- **Key Rotation**: 5-minute default (configurable: 60s - 60min)
- **Zero-Trust Relays**: TPM/SGX attestation + blockchain verification
- **Traffic Obfuscation**: Packet size/timing randomization

### Performance Targets
- **Throughput**: 1+ Gbps (goal: 6-7 Gbps)
- **Latency**: <2ms overhead
- **CGNAT Traversal**: 95%+ success rate
- **Relay Capacity**: 1000+ concurrent connections

---

## Project Structure

```
shadowmesh/
├── cmd/                     # Binary entry points
│   ├── shadowmesh/          # Client daemon
│   ├── shadowmesh-relay/    # Relay node (Epic 4)
│   └── shadowmesh-bootstrap/# Bootstrap node
├── pkg/                     # Shared libraries
│   ├── crypto/              # Cryptography modules (Epic 1)
│   │   ├── mlkem/           # ML-KEM-1024 key exchange
│   │   ├── mldsa/           # ML-DSA-87 signatures
│   │   ├── classical/       # X25519 + Ed25519
│   │   ├── hybrid/          # Hybrid PQC orchestration
│   │   ├── symmetric/       # ChaCha20-Poly1305
│   │   ├── keystore/        # Encrypted keystore
│   │   └── rotation/        # Key rotation scheduler
│   ├── transport/           # Transport layer (Epic 2)
│   │   └── websocket/       # WebSocket Secure (WSS)
│   ├── tap/                 # TAP device management (Epic 2)
│   ├── blockchain/          # Smart contract integration (Epic 3)
│   └── metrics/             # Prometheus metrics (Epic 5)
├── contracts/               # Ethereum smart contracts (Epic 3)
│   ├── contracts/           # Solidity source files
│   │   └── RelayNodeRegistry.sol
│   ├── scripts/             # Deployment scripts
│   ├── test/                # Contract tests
│   └── hardhat.config.ts    # Hardhat configuration
├── monitoring/              # Monitoring stack (Epic 5)
│   ├── docker-compose.yml   # Prometheus + Grafana
│   ├── prometheus.yml       # Metrics configuration
│   └── dashboards/          # Grafana dashboards
├── test/                    # Test suites
│   ├── integration/         # Integration tests
│   └── e2e/                 # End-to-end tests
├── docs/                    # Documentation
│   ├── prd.md               # Product Requirements Document
│   └── 2-ARCHITECTURE/      # Architecture specifications
├── .github/                 # GitHub workflows
│   └── workflows/
│       ├── ci.yml           # CI/CD pipeline
│       └── benchmarks.yml   # Performance benchmarks
├── go.mod                   # Go module definition
└── Makefile                 # Build automation
```

---

## Quick Start

### Prerequisites

- **Go**: 1.25+ ([install](https://golang.org/dl/))
- **Node.js**: 18+ ([install](https://nodejs.org/))
- **Docker**: 20.10+ ([install](https://docs.docker.com/get-docker/))
- **Make**: Build automation tool (usually pre-installed)

### Clone and Build

```bash
# Clone repository
git clone https://github.com/CG-8663/shadowmesh.git
cd shadowmesh

# Install dependencies
go mod download

# Build all binaries
make build

# Run tests
make test

# Run linting
make lint
```

### Development Commands

```bash
# Build client
make build-client

# Build relay node
make build-relay

# Run all tests with coverage
go test ./... -race -cover

# Run specific package tests
go test ./pkg/crypto/... -v

# Format code
go fmt ./...

# Lint code
golangci-lint run

# Compile smart contracts
cd contracts && npx hardhat compile

# Run smart contract tests
cd contracts && npx hardhat test

# Start monitoring stack
cd monitoring && docker-compose up -d
```

---

## Architecture

### Hybrid Peer Discovery

**Kademlia DHT** (P2P peer discovery):
- O(log N) lookup complexity
- 256 k-buckets with k=20 peers per bucket
- 24-hour peer TTL
- PING/PONG liveness checks every 15 minutes

**Ethereum Smart Contract** (relay node registry):
- chronara.eth ENS name
- 0.1 ETH stake requirement
- 24-hour heartbeat requirement
- Automatic slashing for offline nodes

### Transport Layer

**Primary**: WebSocket Secure (WSS)
- TLS 1.3 encryption
- Appears as HTTPS traffic (port 443)
- Defeats deep packet inspection (DPI)
- Packet size/timing randomization

**Fallback**: UDP (direct P2P)
- Low latency for direct connections
- NAT hole punching support

### Network Layer

**TAP Devices** (Layer 2):
- Ethernet frame encryption
- IP headers hidden from transit
- Prevents traffic analysis

### Security

**Post-Quantum Cryptography**:
- ML-KEM-1024: Key encapsulation (<50ms)
- ML-DSA-87: Digital signatures (<15ms)
- ChaCha20-Poly1305: Symmetric encryption (1+ Gbps)

**Key Rotation**:
- Default: Every 5 minutes
- Enterprise: Every 60 seconds
- Ultra-secure: Every 10 seconds

**Keystore**:
- AES-256-GCM encryption
- PBKDF2 passphrase derivation (100k iterations)
- chmod 600 permissions

---

## Development Workflow

### Epic Structure

The project is organized into 6 epics:

1. **Epic 1**: Foundation & Cryptography (Weeks 1-2) ← **Current**
2. **Epic 2**: Core Networking & Direct P2P (Weeks 3-4)
3. **Epic 3**: Smart Contract & Blockchain Integration (Weeks 5-6)
4. **Epic 4**: Relay Infrastructure & CGNAT Traversal (Weeks 7-9)
5. **Epic 5**: Monitoring & Grafana Dashboard (Weeks 10-11)
6. **Epic 6**: Public Map, Documentation & Launch (Week 12)

### Story Development

Stories are tracked in `.bmad-ephemeral/sprint-status.yaml`. Development workflow:

1. **Epic Context**: Generate technical specification for epic
2. **Create Story**: Draft user story with acceptance criteria and tasks
3. **Develop Story**: Implement tasks, write tests, validate
4. **Code Review**: Review and address findings
5. **Mark Done**: Update sprint status and move to next story

### Testing Strategy

**Unit Tests** (target: 85%+ coverage):
```bash
go test ./pkg/crypto/... -cover
```

**Integration Tests**:
```bash
go test ./test/integration/... -v
```

**Benchmarks**:
```bash
go test ./pkg/crypto/... -bench=. -benchmem
```

---

## CI/CD

GitHub Actions runs on every push and PR:

- **Build**: Linux (amd64, arm64)
- **Test**: All packages with race detection
- **Lint**: golangci-lint
- **Coverage**: Uploaded to Codecov
- **Contracts**: Hardhat compilation and tests

See `.github/workflows/ci.yml` for full configuration.

---

## Contributing

We welcome contributions! Please see:

- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) - Community standards
- [SECURITY.md](SECURITY.md) - Security policy and vulnerability reporting

### Pre-commit Hooks

Pre-commit hooks run automatically on `git commit`:
- `go fmt` - Go formatting
- `go vet` - Go static analysis
- `golangci-lint` - Comprehensive linting (if installed)
- `solhint` - Solidity linting (if contracts changed)

Install golangci-lint for full pre-commit checks:
```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

---

## Documentation

- **[PROJECT_STATUS.md](PROJECT_STATUS.md)** - Master roadmap with timeline, metrics, and progress
- **[docs/prd.md](docs/prd.md)** - Product Requirements Document (44 FR, 31 NFR)
- **[docs/2-ARCHITECTURE/](docs/2-ARCHITECTURE/)** - Complete architecture specifications (4,053 lines)
- **[CHANGELOG.md](CHANGELOG.md)** - Version history and release notes

---

## Performance

### v0.1.0-alpha Baseline
- **Throughput**: 28.3 Mbps (45% faster than Tailscale)
- **Video Streaming**: 640x480 @ 547 kb/s
- **Stability**: 3-hour test, zero packet loss

### MVP Targets (v0.2.0-alpha)
- **Throughput**: 1+ Gbps (single connection)
- **Latency**: <2ms overhead
- **CGNAT Traversal**: 95%+ success rate
- **Key Exchange**: <100ms (ML-KEM + X25519)
- **Signatures**: <15ms (ML-DSA + Ed25519)

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Support

- **Issues**: [GitHub Issues](https://github.com/CG-8663/shadowmesh/issues)
- **Discussions**: [GitHub Discussions](https://github.com/CG-8663/shadowmesh/discussions)
- **Security**: See [SECURITY.md](SECURITY.md) for vulnerability reporting

---

**Status**: MVP Development - Epic 1 (Foundation & Cryptography) in progress

**Current Focus**: Implementing hybrid post-quantum cryptography (ML-KEM-1024 + ML-DSA-87) with performance benchmarking

**Next Milestone**: Complete Epic 1 → Begin Epic 2 (Core Networking & Direct P2P)
