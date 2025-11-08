# ShadowMesh Quick Start Guide

This guide will help you get started with ShadowMesh development in under 5 minutes.

## Prerequisites

- Go 1.21 or later
- Git
- Linux OS (for TAP device support in client)
- Make (for build automation)

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/shadowmesh/shadowmesh.git
cd shadowmesh
```

### 2. Install Dependencies

```bash
make deps
```

This will download all Go module dependencies and tidy the go.mod file.

### 3. Build All Components

```bash
make build
```

This creates three binaries in the `build/` directory:
- `shadowmesh-daemon` - Client background service
- `shadowmesh` - CLI tool for managing the daemon
- `shadowmesh-relay` - Relay node server

## Verify Installation

### Check Binary Versions

```bash
./build/shadowmesh version
```

Should output: `Version: 0.1.0-alpha`

### View CLI Help

```bash
./build/shadowmesh help
```

## Development Workflow

### Run Tests

```bash
make test
```

### Format Code

```bash
make fmt
```

### Run Linter

```bash
make vet
```

### Clean Build Artifacts

```bash
make clean
```

### Full Clean Build

```bash
make clean build
```

## Running Components

### Start Client Daemon (requires root for TAP device)

```bash
sudo ./build/shadowmesh-daemon
```

Press Ctrl+C to stop the daemon.

### Use CLI Tool

```bash
# Check status
./build/shadowmesh status

# Connect to network
./build/shadowmesh connect

# View connected peers
./build/shadowmesh peers

# Stop the daemon
./build/shadowmesh stop
```

### Start Relay Server

```bash
./build/shadowmesh-relay
```

Press Ctrl+C to stop the relay server.

## Next Steps

1. **Read the Architecture**: See `docs/ARCHITECTURE.md` for system design
2. **Review the PRD**: See `docs/PRD.md` for complete requirements
3. **Implement Features**: Start with Epic 1, Story 1.2 (Crypto Library Setup)
4. **Run Tests**: Add tests as you implement new features
5. **Submit PRs**: Follow the contribution guidelines in `docs/CONTRIBUTING.md`

## Common Issues

### Build Failures

If you encounter build errors:

```bash
# Clean and rebuild
make clean
go mod tidy
make build
```

### Permission Denied (TAP Device)

The client daemon requires root privileges to create TAP devices:

```bash
sudo ./build/shadowmesh-daemon
```

### Missing Dependencies

If Go dependencies are missing:

```bash
go mod download
go mod tidy
```

## Development Tips

1. **Use Make**: The Makefile provides convenient shortcuts for common tasks
2. **Format Code**: Run `make fmt` before committing
3. **Run Tests**: Add tests for new features and run `make test`
4. **Check Coverage**: Use `make test-coverage` to see test coverage
5. **Read Logs**: The daemon outputs detailed logs for debugging

## Project Structure Overview

```
shadowmesh/
├── client/          # Client daemon and CLI
│   ├── daemon/      # Background service
│   ├── cli/         # Command-line interface
│   └── dashboard/   # Monitoring dashboards
├── relay/           # Relay node server
├── contracts/       # Smart contracts
├── shared/          # Shared libraries
│   ├── crypto/      # Cryptography
│   ├── networking/  # Network utilities
│   └── blockchain/  # Blockchain interaction
├── monitoring/      # Prometheus/Grafana
├── tools/           # Build and deployment
├── scripts/         # Installation scripts
└── docs/            # Documentation
```

## Getting Help

- **Documentation**: Read the docs in `docs/`
- **Issues**: Report bugs at https://github.com/shadowmesh/shadowmesh/issues
- **Discord**: Join our community at https://discord.gg/shadowmesh
- **Email**: Contact support@shadowmesh.network

## What to Build Next

According to the PRD (Epic 1), the next priority is:

**Story 1.2: Crypto Library Setup**
- Implement CRYSTALS-Kyber wrapper
- Implement CRYSTALS-Dilithium wrapper
- Implement AES-256-GCM encryption
- Add performance benchmarks

Start by creating files in `shared/crypto/`:
- `kyber.go` - Kyber key exchange
- `dilithium.go` - Dilithium signatures
- `aes.go` - AES-GCM encryption
- `crypto_test.go` - Test suite
- `benchmark_test.go` - Performance benchmarks

Happy coding!
