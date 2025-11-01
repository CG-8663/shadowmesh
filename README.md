# ShadowMesh

**Post-Quantum Encrypted Private Network**

ShadowMesh is a decentralized, quantum-resistant private network that provides secure communication using post-quantum cryptography (PQC) and blockchain-based relay node verification. The system is designed for Linux clients and features automatic failover, multi-path routing, and comprehensive monitoring.

## Overview

ShadowMesh combines cutting-edge post-quantum cryptography with blockchain technology to create a future-proof private network:

- **Post-Quantum Security**: Uses CRYSTALS-Kyber for key exchange and CRYSTALS-Dilithium for digital signatures
- **Decentralized Relay Network**: Blockchain-verified relay nodes with stake-based incentives
- **Client VPN**: Linux CLI client with daemon for seamless private networking
- **Multi-Path Routing**: Automatic failover and load balancing across relay nodes
- **Comprehensive Monitoring**: Built-in Prometheus metrics and Grafana dashboards

## Repository Structure

```
shadowmesh/
├── client/              # Linux CLI client + daemon
│   ├── daemon/          # Background service (Go)
│   ├── cli/             # Command-line interface (Go)
│   └── dashboard/       # Grafana dashboard configs
├── relay/               # Relay node software (Go)
│   ├── server/          # Main relay server
│   └── config/          # Configuration templates
├── contracts/           # Solidity smart contracts
│   ├── src/             # Contract source files
│   ├── test/            # Contract tests
│   └── migrations/      # Deployment scripts
├── shared/              # Shared Go libraries
│   ├── crypto/          # PQC and classical crypto wrappers
│   ├── networking/      # WebSocket, TAP devices, routing
│   └── blockchain/      # Smart contract interaction
├── monitoring/          # Prometheus/Grafana configurations
│   ├── prometheus/      # Prometheus config and rules
│   └── grafana/         # Grafana dashboards
│       └── dashboards/
├── tools/               # Build and deployment tools
│   ├── build/           # Build scripts
│   └── deployment/      # Deployment automation
├── scripts/             # Installation and deployment scripts
│   ├── install/         # Installation scripts
│   └── deploy/          # Deployment scripts
└── docs/                # Documentation
    └── research/        # Research papers and references
```

## Quick Start

### Prerequisites

- Go 1.21 or later
- Linux OS (for client daemon with TAP device support)
- Docker and Docker Compose (for relay nodes)
- Node.js 18+ (for smart contract development)

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/shadowmesh/shadowmesh.git
   cd shadowmesh
   ```

2. **Install Go dependencies**
   ```bash
   go mod download
   go mod tidy
   ```

3. **Verify installation**
   ```bash
   go test ./...
   ```

### Build Commands

#### Build Client Daemon
```bash
cd client/daemon
go build -o shadowmesh-daemon .
```

#### Build CLI Tool
```bash
cd client/cli
go build -o shadowmesh .
```

#### Build Relay Server
```bash
cd relay/server
go build -o shadowmesh-relay .
```

#### Build All Components
```bash
# From repository root
make build
```

### Running Components

#### Start Client Daemon
```bash
sudo ./client/daemon/shadowmesh-daemon
```

#### Use CLI Tool
```bash
./client/cli/shadowmesh status
./client/cli/shadowmesh connect
./client/cli/shadowmesh peers
```

#### Start Relay Node
```bash
./relay/server/shadowmesh-relay
```

## Development Workflow

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./shared/crypto/...

# Run with verbose output
go test -v ./...
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Vet code
go vet ./...
```

### Building for Production

```bash
# Build with optimizations
go build -ldflags="-s -w" -o build/shadowmesh-daemon client/daemon/main.go
go build -ldflags="-s -w" -o build/shadowmesh client/cli/main.go
go build -ldflags="-s -w" -o build/shadowmesh-relay relay/server/main.go
```

## Architecture

### Client Architecture

The client consists of two main components:

1. **Daemon** (`client/daemon/`): Background service that:
   - Manages TAP network device
   - Handles post-quantum key exchange
   - Routes traffic through relay network
   - Maintains persistent connections

2. **CLI** (`client/cli/`): Command-line interface for:
   - Starting/stopping daemon
   - Connecting to network
   - Viewing status and peers
   - Configuration management

### Relay Node Architecture

Relay nodes (`relay/server/`) provide:
- WebSocket server for client connections
- Traffic routing with multi-path support
- Blockchain registration and verification
- Performance metrics collection
- Automatic failover handling

### Shared Libraries

The `shared/` directory contains common libraries:

- **crypto/**: Post-quantum and classical cryptography
  - CRYSTALS-Kyber (key exchange)
  - CRYSTALS-Dilithium (signatures)
  - AES-256-GCM (symmetric encryption)

- **networking/**: Network functionality
  - WebSocket client/server
  - TAP device management
  - Packet routing

- **blockchain/**: Smart contract interaction
  - Relay node registration
  - Stake management
  - Reputation tracking

## Smart Contracts

The Solidity contracts (`contracts/`) handle:
- Relay node registration and verification
- Stake deposits and slashing
- Reputation and performance tracking
- Payment distribution

See `contracts/README.md` for detailed contract documentation.

## Monitoring

### Prometheus Metrics

The system exposes metrics for:
- Connection counts
- Bandwidth usage
- Latency measurements
- Packet loss rates
- Crypto operation performance

### Grafana Dashboards

Pre-configured dashboards for:
- Network overview
- Client connections
- Relay node performance
- System resource usage

Access dashboards at: `http://localhost:3000` (when running locally)

## Configuration

### Client Configuration

Default location: `/etc/shadowmesh/client.yaml`

```yaml
relay_nodes:
  - address: relay1.shadowmesh.network:8443
  - address: relay2.shadowmesh.network:8443

crypto:
  algorithm: kyber1024

network:
  interface: tun0
  mtu: 1500
```

### Relay Node Configuration

Default location: `/etc/shadowmesh/relay.yaml`

```yaml
server:
  listen_address: 0.0.0.0:8443
  max_clients: 1000

blockchain:
  rpc_url: https://mainnet.infura.io/v3/YOUR_KEY
  contract_address: 0x...

monitoring:
  metrics_port: 9090
```

## Security

### Post-Quantum Cryptography

ShadowMesh uses NIST-standardized post-quantum algorithms:
- **CRYSTALS-Kyber**: Key encapsulation mechanism
- **CRYSTALS-Dilithium**: Digital signatures

### Classical Cryptography

For symmetric encryption and performance:
- **AES-256-GCM**: Authenticated encryption
- **ChaCha20-Poly1305**: Alternative stream cipher

### Threat Model

See `docs/SECURITY.md` for:
- Threat model analysis
- Security assumptions
- Audit reports
- Vulnerability disclosure policy

## Contributing

We welcome contributions! Please see `docs/CONTRIBUTING.md` for:
- Development guidelines
- Code style requirements
- Pull request process
- Testing requirements

## Documentation

Comprehensive documentation is available in the `docs/` directory:

- `docs/PRD.md` - Product Requirements Document
- `docs/ARCHITECTURE.md` - System architecture details
- `docs/API.md` - API documentation
- `docs/DEPLOYMENT.md` - Deployment guide
- `docs/SECURITY.md` - Security documentation

## Roadmap

### Phase 1: Foundation (Current)
- [x] Monorepo structure
- [ ] Core crypto libraries
- [ ] Basic client daemon
- [ ] Smart contracts

### Phase 2: Networking
- [ ] WebSocket relay server
- [ ] TAP device integration
- [ ] Multi-path routing
- [ ] Client-relay protocol

### Phase 3: Blockchain
- [ ] Smart contract deployment
- [ ] Relay registration
- [ ] Stake management
- [ ] Reputation system

### Phase 4: Production
- [ ] Monitoring dashboards
- [ ] Performance optimization
- [ ] Security audit
- [ ] Mainnet launch

## License

MIT License - see `LICENSE` file for details

## Support

- Documentation: https://docs.shadowmesh.network
- GitHub Issues: https://github.com/shadowmesh/shadowmesh/issues
- Discord: https://discord.gg/shadowmesh
- Email: support@shadowmesh.network

## Acknowledgments

ShadowMesh builds upon research and implementations from:
- NIST Post-Quantum Cryptography Standardization
- Open Quantum Safe project
- WireGuard protocol design
- Ethereum smart contract ecosystem

---

**Status**: Alpha - Under Active Development

**Warning**: This software is in early development and should not be used in production environments.
