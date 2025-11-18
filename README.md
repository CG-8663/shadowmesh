# ShadowMesh - Post-Quantum Decentralized Private Network (DPN)

[![Version](https://img.shields.io/badge/version-0.2.0--alpha-blue.svg)](https://github.com/CG-8663/shadowmesh/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

---

ShadowMesh is a peer-to-peer encrypted private network with relay fallback for Symmetric NAT traversal. Built for quantum-safe networking with Layer 2 encryption and WebSocket transport.

**Current Status**: Story 2-8 Complete - Relay Mode with Production Deployment

---

## Roadmap

### Completed ✅

**Epic 1: Core Relay Infrastructure**
- [x] Story 2-1: Daemon Manager Core
- [x] Story 2-2: Frame Encryption
- [x] Story 2-3: TAP Device Management
- [x] Story 2-4: Frame Router
- [x] Story 2-5: WebSocket P2P Connection
- [x] Story 2-6: Daemon API Server
- [x] Story 2-7: CLI Tool
- [x] Story 2-8: Relay Server Mode (Production Deployment)

### In Progress 🚧

- [x] Epic 2: Direct P2P with UDP hole punching (Implementation complete, testing pending)

### Planned 📋

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

**Next Milestone**: Epic 2 - Direct P2P with UDP hole punching for sub-5ms latency
