# ShadowMesh - Post-Quantum Decentralized Private Network (DPN)

[![Version](https://img.shields.io/badge/version-0.2.0--alpha-blue.svg)](https://github.com/CG-8663/shadowmesh/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

---

ShadowMesh is a peer-to-peer encrypted private network with relay fallback for Symmetric NAT traversal. Built for quantum-safe networking with Layer 2 encryption and WebSocket transport.

**Current Status**: Story 2-8 Complete - Relay Mode with Production Deployment

---

## Features

### Implemented ✅

- **Layer 2 Networking**: TAP devices with Ethernet frame encryption
- **ChaCha20-Poly1305**: Symmetric encryption (minimal CPU overhead)
- **WebSocket Transport**: HTTP-compatible traffic for censorship resistance
- **Relay Server**: Production relay at 94.237.121.21:9545 for Symmetric NAT traversal
- **TCP BBR**: Optimized congestion control for high-latency paths
- **Daemon Manager**: HTTP API for connection management
- **CLI Tool**: Command-line interface for daemon control
- **NAT Traversal**: STUN-based NAT type detection

### Performance

**Production Test Results** (via relay server):
- **Throughput**: 36.6 Mbps (63% faster than Tailscale's 22.4 Mbps)
- **Latency**: ~55ms (relay adds 5ms overhead)
- **Bandwidth Utilization**: 81-87% (near-optimal)
- **CPU Overhead**: 1.3% on Raspberry Pi 4, 4.3% on Intel Xeon
- **Retransmissions**: 0 (perfect stability)
- **Scalability**: >2 Gbps potential on Raspberry Pi, >800 Mbps on Intel Xeon

**Bottleneck**: Internet upload bandwidth, not ShadowMesh architecture.

### In Development 🚧

- **Post-Quantum Cryptography**: ML-KEM-1024 (key exchange) + ML-DSA-87 (signatures)
- **Direct P2P**: UDP hole punching for low-latency direct connections
- **Smart Contract**: Ethereum relay registry with staking/slashing
- **Monitoring**: Prometheus + Grafana dashboards

---

## Project Structure

```
shadowmesh/
├── cmd/                       # Binary entry points
│   ├── shadowmesh-daemon/     # Main daemon (client/server)
│   ├── relay-server/          # Relay server (deployed to production)
│   └── shadowmesh/            # CLI tool for daemon management
├── pkg/                       # Shared libraries
│   ├── crypto/                # Cryptography modules
│   │   ├── classical/         # X25519 + Ed25519 (classical crypto)
│   │   ├── hybrid/            # Hybrid PQC orchestration (in dev)
│   │   ├── mlkem/             # ML-KEM-1024 (in dev)
│   │   ├── mldsa/             # ML-DSA-87 (in dev)
│   │   ├── symmetric/         # ChaCha20-Poly1305 (production)
│   │   └── frameencryption/   # Frame-level encryption (production)
│   ├── daemonmgr/             # Daemon manager
│   │   ├── api_server.go      # HTTP API server
│   │   ├── p2p.go             # WebSocket P2P connection
│   │   ├── frame_router.go    # Frame routing logic
│   │   └── manager.go         # Main daemon orchestration
│   ├── layer2/                # TAP device management
│   │   ├── tap_device.go      # TAP device (2000 frame buffers)
│   │   └── ethernet.go        # Ethernet frame parsing
│   └── nat/                   # NAT traversal
│       └── stun.go            # STUN client for NAT detection
├── relay/                     # Relay server implementation
│   └── server/                # WebSocket relay server
├── scripts/                   # Deployment and testing scripts
│   ├── deploy/                # Deployment scripts
│   │   └── upcloud-relay.sh   # UpCloud relay deployment
│   └── optimize-tcp-performance.sh  # TCP BBR optimization
├── docs/                      # Documentation
│   └── QUICK_START_TESTING.md # Performance testing guide
├── bin/                       # Compiled binaries
├── go.mod                     # Go module definition
└── .github/                   # CI/CD workflows
```

---

## Quick Start

### Prerequisites

- **Go**: 1.21+ ([install](https://golang.org/dl/))
- **Root/sudo access**: Required for TAP device creation
- **Linux**: Ubuntu 20.04+, Debian 11+, or Raspberry Pi OS (macOS experimental)

### Build

```bash
# Clone repository
git clone https://github.com/CG-8663/shadowmesh.git
cd shadowmesh

# Install dependencies
go mod download

# Build daemon
go build -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon

# Build CLI tool
go build -o bin/shadowmesh ./cmd/shadowmesh

# Build relay server (optional)
go build -o bin/relay-server ./cmd/relay-server
```

### Usage

#### Start Daemon (Relay Mode)

```bash
# Start daemon with default config
sudo bin/shadowmesh-daemon

# Or use systemd service
sudo systemctl start shadowmesh
sudo systemctl enable shadowmesh
```

#### Connect to Peer via Relay

```bash
# Using CLI tool
bin/shadowmesh connect --relay --relay-server 94.237.121.21:9545 --peer-id peer-001

# Or using API directly
curl -X POST http://localhost:9090/connect \
  -H "Content-Type: application/json" \
  -d '{
    "use_relay": true,
    "relay_server": "94.237.121.21:9545",
    "peer_id": "peer-001"
  }'
```

#### Check Status

```bash
# Using CLI
bin/shadowmesh status

# Or using API
curl http://localhost:9090/status
```

#### Performance Testing

See [docs/QUICK_START_TESTING.md](docs/QUICK_START_TESTING.md) for complete performance testing guide.

```bash
# Test throughput (requires iperf3 on both endpoints)
iperf3 -s -B 10.10.10.3  # On endpoint 1
iperf3 -c 10.10.10.3 -t 30 -P 4  # On endpoint 2

# Optimize TCP for high-latency relay connections
sudo scripts/optimize-tcp-performance.sh
```

---

## Architecture

### Current Implementation (Story 2-8)

**Transport Layer**:
- **WebSocket**: Primary transport (relay mode)
- **TCP**: Control plane
- **HTTP API**: Daemon management (port 9090)

**Network Layer**:
- **TAP Devices** (Layer 2): Ethernet frame encryption
- **IP Tunnel**: 10.10.10.0/24 default network
- **Frame Routing**: Direct TAP ↔ WebSocket frame forwarding

**Security**:
- **ChaCha20-Poly1305**: Symmetric encryption (1-4% CPU overhead)
- **Frame-level Encryption**: All Ethernet frames encrypted before relay transit
- **No IP Leakage**: IP headers encrypted within Ethernet frames

**Relay Server**:
- **Location**: UpCloud datacenter (94.237.121.21:9545)
- **Protocol**: WebSocket with 2MB buffers
- **Capacity**: 1000+ concurrent connections
- **Latency**: ~50-60ms relay hop

### Network Flow

```
Endpoint A (TAP)  →  WebSocket  →  Relay Server  →  WebSocket  →  Endpoint B (TAP)
   10.10.10.3          (WSS)      94.237.121.21       (WSS)         10.10.10.4
     ↓                                                                   ↓
  Ethernet Frame  →  Encrypted  →  Relayed  →  Encrypted  →  Ethernet Frame
```

### Performance Optimizations

**TAP Buffers**: 2000 frames (increased from 100)
- Prevents "TAP write channel full" warnings
- Handles burst traffic (iperf3 stress test validated)

**WebSocket Buffers**: 2MB (increased from 4KB)
- Prevents "send buffer full" errors
- Sustains 35+ Mbps relay throughput

**TCP BBR**: Bottleneck Bandwidth and RTT congestion control
- 5% throughput improvement on high-bandwidth endpoints
- Zero retransmissions in testing

---

## Development Workflow

### Building

```bash
# Build all binaries
go build -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon
go build -o bin/relay-server ./cmd/relay-server
go build -o bin/shadowmesh ./cmd/shadowmesh

# Run tests
go test ./...

# Run with race detection
go test ./... -race

# Format code
go fmt ./...

# Vet code
go vet ./...
```

### Testing

```bash
# Unit tests
go test ./pkg/layer2/... -v
go test ./pkg/crypto/... -v
go test ./pkg/daemonmgr/... -v

# Integration test (requires 2 machines)
# See docs/QUICK_START_TESTING.md

# Performance benchmarks
go test ./pkg/crypto/... -bench=. -benchmem
```

### Deployment

```bash
# Deploy relay server to UpCloud
./scripts/deploy/upcloud-relay.sh

# Deploy to Raspberry Pi endpoint
ssh shadowmesh-002
git clone https://github.com/CG-8663/shadowmesh.git
cd shadowmesh
go build -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon
sudo bin/shadowmesh-daemon
```

---

## Configuration

### Daemon Config (`/etc/shadowmesh/client.yaml`)

```yaml
daemon:
  listen_addr: "0.0.0.0:9090"  # API server address
  p2p_listen_port: 9545        # WebSocket P2P port

tap:
  device_name: "shadowmesh0"
  ip_address: "10.10.10.3"     # Tunnel IP
  netmask: "255.255.255.0"
  mtu: 1500

crypto:
  key_file: "/etc/shadowmesh/keys/session.key"

relay:
  enabled: false               # Set true for relay mode
  server: "94.237.121.21:9545" # Production relay
  peer_id: ""                  # Auto-generated if empty

logging:
  level: "info"                # debug, info, warn, error
```

### Relay Server Config

```yaml
relay:
  listen_addr: "0.0.0.0:9545"
  max_peers: 1000
  buffer_size: 2097152  # 2MB

logging:
  level: "info"
```

---

## Performance Testing Results

### Test Setup

- **Endpoint 1**: Intel Xeon D-2166NT VM (14.44 Mbps upload)
- **Endpoint 2**: Raspberry Pi 4 ARM (48.07 Mbps upload)
- **Relay**: UpCloud datacenter (94.237.121.21:9545)
- **Test**: iperf3 -P 4 -t 30 (4 parallel streams, 30 seconds)

### Results

| Metric | ShadowMesh | Tailscale | Advantage |
|--------|------------|-----------|-----------|
| Throughput (002→001) | 36.6 Mbps | 22.4 Mbps | **+63%** |
| Throughput (001→002) | 12.4 Mbps | N/A | Limited by upload |
| Retransmissions | 0 | N/A | Perfect |
| CPU (Raspberry Pi) | 1.3% | N/A | Minimal |
| CPU (Intel Xeon) | 4.3% | N/A | Minimal |
| Latency | ~55ms | ~50ms | +5ms |
| Bandwidth Utilization | 81-87% | N/A | Excellent |

**Conclusion**: Performance limited by internet upload bandwidth, not ShadowMesh architecture.

### Scalability Projections

Based on CPU profiling:
- **Raspberry Pi 4**: Could handle >2 Gbps (1.3% CPU at 35 Mbps)
- **Intel Xeon VM**: Could handle >800 Mbps (4.3% CPU at 35 Mbps)

---

## Roadmap

### Completed ✅

- [x] Story 2-1: Daemon Manager Core
- [x] Story 2-2: Frame Encryption
- [x] Story 2-3: TAP Device Management
- [x] Story 2-4: Frame Router
- [x] Story 2-5: WebSocket P2P Connection
- [x] Story 2-6: Daemon API Server
- [x] Story 2-7: CLI Tool
- [x] Story 2-8: Relay Server Mode (Production Deployment)

### In Progress 🚧

- [ ] Epic 2: Direct P2P with UDP hole punching
- [ ] Epic 3: Smart contract relay registry
- [ ] Epic 4: Post-quantum cryptography (ML-KEM + ML-DSA)
- [ ] Epic 5: Monitoring and metrics

### Future 📋

- [ ] Mobile clients (iOS, Android)
- [ ] Multi-hop routing (3-5 hops)
- [ ] Traffic obfuscation (packet size/timing randomization)
- [ ] TPM/SGX relay attestation

---

## Contributing

We welcome contributions! Please see:

- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) - Community standards
- [SECURITY.md](SECURITY.md) - Security policy

### Pre-commit Hooks

Pre-commit hooks run automatically on `git commit`:
- `go fmt` - Go formatting
- `go vet` - Go static analysis

---

## Documentation

- **[docs/QUICK_START_TESTING.md](docs/QUICK_START_TESTING.md)** - Complete performance testing guide with optimization results
- **[CHANGELOG.md](CHANGELOG.md)** - Version history and release notes

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Support

- **Issues**: [GitHub Issues](https://github.com/CG-8663/shadowmesh/issues)
- **Discussions**: [GitHub Discussions](https://github.com/CG-8663/shadowmesh/discussions)
- **Security**: See [SECURITY.md](SECURITY.md) for vulnerability reporting

---

**Current Status**: Story 2-8 Complete - Relay mode tested and validated in production

**Performance**: 36.6 Mbps (63% faster than Tailscale), zero retransmissions, minimal CPU overhead

**Next Milestone**: Epic 2 - Direct P2P with UDP hole punching for sub-5ms latency
