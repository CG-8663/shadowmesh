# Architecture Documentation

**Version**: MVP (Full PRD Alignment)
**Last Updated**: November 12, 2025
**Status**: Complete and Validated

---

## Current Architecture

### [KADEMLIA_DHT_ARCHITECTURE.md](KADEMLIA_DHT_ARCHITECTURE.md) - **COMPLETE ARCHITECTURE** (4,053 lines)

This is the **single source of truth** for ShadowMesh architecture. It covers all components required for the MVP:

**Core Components**:
1. ✅ **Hybrid Peer Discovery**: Kademlia DHT + Ethereum smart contract (chronara.eth)
2. ✅ **Transport Layer**: WebSocket Secure (WSS) with traffic obfuscation + UDP fallback
3. ✅ **Network Layer**: TAP devices (Layer 2) with Ethernet frame encryption
4. ✅ **Smart Contract**: Solidity relay node registry with staking and slashing
5. ✅ **Monitoring**: Grafana + Prometheus with Docker Compose
6. ✅ **Database**: PostgreSQL with complete schema
7. ✅ **Public Map**: React + Leaflet.js network visualization
8. ✅ **Security**: Post-Quantum Cryptography (ML-KEM-1024 + ML-DSA-87)

**Contents** (by section):
- Executive Summary
- Technology Stack & Versions
- Solution Architecture (Hybrid DHT + Smart Contract)
- Architecture Decision Summary
- Architecture Overview Diagram
- Ethereum Smart Contract (chronara.eth)
- WebSocket Secure (WSS) Transport Layer
- Core Component: Kademlia DHT
  - PeerID Generation
  - Routing Table Structure
  - DHT Operations (FIND_NODE, STORE, FIND_VALUE, PING)
  - Bootstrap Process
  - DHT Message Protocol
- Integration with Existing Components
  - PQC Handshake Flow
  - TAP Device Integration (Layer 2)
- Monitoring & Observability (Grafana + Prometheus)
  - Docker Compose Configuration
  - Prometheus Metrics
  - Grafana Dashboard Layout
- PostgreSQL Database Design
  - Complete Schema (7 tables)
  - Go Integration
  - Backup & Maintenance
- Public Network Map
  - Backend Service (Indexer + API)
  - Frontend (React + Leaflet.js)
  - Privacy Guarantees
- Project Structure
- Implementation Patterns
- Project Initialization
- Migration Path (v11 → v0.2.0-alpha)
- Standalone Release Criteria
- Future Work (v0.3.0+)

---

## Architecture Validation

The architecture was validated on November 11, 2025, addressing all critical gaps:
- ✅ Technology versions verified (Go 1.25.4, circl v1.6.1, websocket v1.5.3, etc.)
- ✅ Project structure added (8 major components with complete directory tree)
- ✅ Implementation patterns documented (naming, testing, error handling)
- ✅ Project initialization guide added
- ✅ Decision summary table added

**Solutioning Gate Check**: All 6 critical PRD-Architecture conflicts resolved (Ethereum smart contract, WebSocket Secure, TAP devices, monitoring, database, public map).

---

## Implementation Readiness

The architecture has been validated and is **ready for implementation**.

**Next Steps**:
1. Run sprint planning workflow: `/bmad:bmm:workflows:sprint-planning`
2. Create user stories from architecture
3. Begin development

**Workflow Status**: See [`../bmm-workflow-status.yaml`](../bmm-workflow-status.yaml)

---

## Archive

All previous architecture documents have been archived or removed to avoid confusion:

- **Removed**: 13 outdated architecture files (DHT-only, old approaches)
- **Removed**: `decisions/` folder (superseded by Architecture Decision Summary table)
- **Removed**: Old architecture specs in `../architecture/` folder

**Historical Reference**: See `../archive/` for previous architecture approaches (DHT-only, centralized discovery, etc.). These are **NOT** relevant to current development.

---

## Support

- **Questions**: All architecture questions should reference `KADEMLIA_DHT_ARCHITECTURE.md`
- **Updates**: This is the living architecture document - update as implementation progresses
- **Issues**: Report architecture issues to the team for discussion

---

**Status**: ✅ Complete | ✅ Validated | ✅ Ready for Development
