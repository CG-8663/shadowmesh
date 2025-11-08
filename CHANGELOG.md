# Changelog

All notable changes to ShadowMesh will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planning
- 18-week roadmap to standalone Kademlia DHT + PQC + QUIC operation
- Sprint 0: Architecture POC (4 weeks)
- Target: Fully decentralized peer-to-peer mesh network

## [0.2.0-alpha] - v19 QUIC Prototype - 2025-11-08

### Added
- QUIC transport layer (`pkg/transport/quic.go`)
- Layer 3 TUN device support
- Frame-based protocol over QUIC streams
- ChaCha20-Poly1305 encryption over QUIC

### Known Issues
- Post-quantum cryptography not integrated (security regression from v11)
- Centralized discovery backbone dependency
- No DHT peer discovery
- Limited testing coverage

## [0.1.3-alpha] - v11 Phase 3 - 2025-11-07

### Added
- Performance optimizations (buffer pools, stack allocation)
- Adaptive buffered channels (BDP-based sizing)
- UDP frame transmission optimizations

### Changed
- Reduced packet loss from 95% to <5%
- Reduced latency from 3000ms to <50ms
- 90% reduction in heap allocations

### Performance
- Throughput: 100+ Mbps achieved
- Latency: <10ms overhead
- Packet loss: <5%

## [0.1.2-alpha] - v11 PQC Implementation - 2025-11-05

### Added
- ML-KEM-1024 (Kyber) post-quantum key exchange
- ML-DSA-87 (Dilithium) post-quantum digital signatures
- Hybrid PQC mode (classical + post-quantum)
- ChaCha20-Poly1305 symmetric encryption
- Layer 3 UDP transport

### Security
- NIST FIPS 203 compliant (ML-KEM-1024)
- NIST FIPS 204 compliant (ML-DSA-87)
- Quantum-resistant cryptography
- Perfect forward secrecy

## [0.1.1-alpha] - Epic 2 Direct P2P - 2025-11-04

### Added
- Direct peer-to-peer connections
- TLS 1.3 + certificate pinning
- Re-handshake protocol (553μs)
- Seamless relay-to-direct migration (201ms, zero packet loss)
- Automatic relay fallback

### Changed
- Peer discovery via centralized backbone
- Connection flow: Relay → Direct P2P (when possible)

### Performance
- Re-handshake: 553μs (18x faster than 10ms target)
- Migration: 201ms total (within 250ms target)
- Packet loss: 0% during migration

## [0.1.0-alpha] - Epic 1 Foundation - 2024-11-03

### Added
- Core cryptography library (`shared/crypto/`)
- Protocol layer (16 message types)
- Client daemon with TAP device support
- Relay server infrastructure
- WebSocket transport
- Frame encryption/decryption pipeline
- Configuration system (YAML)
- Installation scripts

### Infrastructure
- Monorepo structure
- BMAD Method integration
- Build system (Makefile)
- Testing framework
- Comprehensive documentation

---

## Version Numbering

- **Major.Minor.Patch-stage**
- **Stage**: `alpha` → `beta` → `rc` → (none for stable)

### Current Status
- **Version**: 0.2.0-alpha (v19 QUIC prototype)
- **Stage**: Alpha (active development)
- **Target**: 1.0.0 (standalone decentralized mesh)

---

## Upcoming Releases

### [0.3.0-alpha] - Sprint 2 (Planned)
- Complete Kademlia DHT implementation
- FIND_NODE iterative lookup
- STORE/FIND_VALUE operations
- PeerID from ML-DSA-87 keys

### [0.4.0-alpha] - Sprint 4 (Planned)
- QUIC + PQC integration
- ML-KEM-1024 over QUIC handshake
- Hybrid transport (performance + security)

### [1.0.0-beta] - Sprint 14 (Planned)
- Standalone operation (zero central dependencies)
- Kademlia DHT peer discovery
- QUIC + PQC transport
- NodeNexus smart contract (Polygon)
- 100+ beta users

### [1.0.0] - Sprint 16 (Planned)
- Production-ready release
- Security audit complete
- Open source launch
- 1000+ GitHub stars

---

## Links

- **Roadmap**: [docs/1-PRODUCT/ROADMAP.md](docs/1-PRODUCT/ROADMAP.md)
- **Architecture**: [docs/2-ARCHITECTURE/](docs/2-ARCHITECTURE/)
- **Repository**: https://github.com/shadowmesh/shadowmesh (planned)
- **Website**: https://shadowmesh.io (planned)

---

[unreleased]: https://github.com/shadowmesh/shadowmesh/compare/v0.2.0-alpha...HEAD
[0.2.0-alpha]: https://github.com/shadowmesh/shadowmesh/releases/tag/v0.2.0-alpha
[0.1.3-alpha]: https://github.com/shadowmesh/shadowmesh/releases/tag/v0.1.3-alpha
[0.1.2-alpha]: https://github.com/shadowmesh/shadowmesh/releases/tag/v0.1.2-alpha
[0.1.1-alpha]: https://github.com/shadowmesh/shadowmesh/releases/tag/v0.1.1-alpha
[0.1.0-alpha]: https://github.com/shadowmesh/shadowmesh/releases/tag/v0.1.0-alpha
