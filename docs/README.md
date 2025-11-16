# ShadowMesh Documentation

**Version**: MVP (Post-Quantum DPN with Full Feature Set)
**Last Updated**: November 12, 2025
**Status**: Architecture Complete - Ready for Implementation

---

## Documentation Structure

```
docs/
├── prd.md                                    # Product Requirements Document (PRD)
├── bmm-workflow-status.yaml                  # BMAD workflow tracking
├── 2-ARCHITECTURE/
│   ├── KADEMLIA_DHT_ARCHITECTURE.md         # COMPLETE ARCHITECTURE (4,053 lines)
│   └── README.md                            # Architecture overview
├── README.md                                # This file
└── archive/                                 # Historical documentation (for reference only)
    ├── epic2/                               # Previous epic completion reports
    ├── obsolete-roadmaps/                   # Old roadmaps and timelines
    ├── obsolete/                            # Obsolete documentation
    └── v11/                                 # v0.1.0-alpha (v11) completion reports
```

---

## Quick Start

### For Developers

**Start Here**: [`2-ARCHITECTURE/KADEMLIA_DHT_ARCHITECTURE.md`](2-ARCHITECTURE/KADEMLIA_DHT_ARCHITECTURE.md)

This is the **complete, production-ready architecture** covering:

1. **Hybrid Peer Discovery**
   - Kademlia DHT for P2P peer discovery
   - Ethereum smart contract (chronara.eth) for relay node registry

2. **Transport Layer**
   - WebSocket Secure (WSS) with TLS 1.3 for censorship resistance
   - Traffic obfuscation (packet size/timing randomization)
   - UDP fallback for direct P2P connections

3. **Network Layer**
   - TAP devices (Layer 2) for Ethereum frame encryption
   - IP headers hidden to prevent traffic analysis

4. **Security**
   - Post-Quantum Cryptography (ML-KEM-1024 + ML-DSA-87)
   - Hybrid mode with classical crypto (X25519 + Ed25519)
   - 5-minute key rotation (default)

5. **Smart Contract Integration**
   - Solidity contract with staking, heartbeats, and slashing
   - ENS resolution (chronara.eth)
   - Caching strategy (10-minute TTL)

6. **Monitoring Stack**
   - Grafana + Prometheus via Docker Compose
   - 15+ metrics exposed
   - 3 pre-configured dashboards

7. **Database**
   - PostgreSQL 14+ with complete schema
   - User/device management
   - Connection history and audit logs

8. **Public Network Map**
   - React + Leaflet.js frontend
   - Blockchain event indexer
   - Privacy-preserving (city/country only)

### For Product Managers

**Start Here**: [`prd.md`](prd.md)

The PRD defines all functional and non-functional requirements for the MVP.

**Implementation Status**: Solutioning gate check completed November 11, 2025. Architecture expanded to 4,053 lines covering all PRD requirements (Ethereum smart contract, WebSocket Secure, TAP devices, monitoring, database, public map). All 6 critical PRD-Architecture conflicts resolved.

---

## Architecture Highlights

### Technology Stack

| Component | Technology | Version | Purpose |
|-----------|------------|---------|---------|
| **Language** | Go | 1.25.4 | Primary language |
| **PQC** | cloudflare/circl | v1.6.1 | ML-KEM-1024 + ML-DSA-87 |
| **Transport** | gorilla/websocket | v1.5.3 | WebSocket Secure (WSS) |
| **Smart Contract** | Solidity | 0.8.20+ | chronara.eth relay registry |
| **Database** | PostgreSQL | 14+ | User/device/audit data |
| **Monitoring** | Prometheus + Grafana | 2.45+ / 10+ | Metrics and visualization |
| **Frontend** | React + TypeScript | Latest | Public network map |

### Key Features

✅ **Post-Quantum Secure**: First DPN with NIST-standardized PQC (5+ year head start)
✅ **Censorship Resistant**: WebSocket traffic mimics HTTPS, defeats DPI
✅ **True Decentralization**: DHT + blockchain, no central infrastructure
✅ **Privacy-Preserving**: Layer 2 encryption hides IP headers
✅ **Zero-Trust Relay Nodes**: Stake-based trust with on-chain verification
✅ **Comprehensive Monitoring**: Real-time metrics with Grafana dashboards
✅ **Enterprise-Ready**: PostgreSQL audit logs, SOC 2/HIPAA/PCI DSS ready

---

## Development Workflow

### Current Phase: **Solutioning Gate Check Complete**

**Workflow Status**: See [`bmm-workflow-status.yaml`](bmm-workflow-status.yaml)

```yaml
workflow_status:
  # Phase 1: Planning
  prd: docs/prd.md  # ✅ Complete
  validate-prd: optional
  create-design: conditional

  # Phase 2: Solutioning
  create-architecture: docs/2-ARCHITECTURE/KADEMLIA_DHT_ARCHITECTURE.md  # ✅ Complete
  validate-architecture: completed  # ✅ Complete
  solutioning-gate-check: completed  # ✅ Complete (Nov 11, 2025)

  # Phase 3: Implementation
  sprint-planning: required  # ⏭️ NEXT STEP
```

### Next Steps

1. **Run Sprint Planning**: `/bmad:bmm:workflows:sprint-planning`
   - Generate sprint status tracking file
   - Extract all epics and stories from PRD
   - Track development lifecycle

2. **Create Stories**: `/bmad:bmm:workflows:create-story`
   - Generate user stories from architecture
   - Use standard template
   - Save to stories/ folder

3. **Development**: `/bmad:bmm:workflows:dev-story`
   - Implement tasks/subtasks
   - Write tests
   - Validate against acceptance criteria

---

## Archive

The `archive/` folder contains historical documentation for reference:

- **epic2/**: Completion reports for previous development epic
- **obsolete-roadmaps/**: Old roadmaps superseded by current PRD
- **obsolete/**: Outdated documentation (deployment guides, old summaries)
- **v11/**: v0.1.0-alpha (v11) performance reports and completion status

**Note**: Archive documents reference old architecture approaches (DHT-only, centralized discovery) and are **NOT** relevant to current development. Kept for historical reference only.

---

## Support

- **Architecture Questions**: See `2-ARCHITECTURE/KADEMLIA_DHT_ARCHITECTURE.md` (comprehensive, 4,053 lines)
- **Requirements Questions**: See `prd.md` (complete PRD with 44 functional requirements)
- **Implementation Status**: Solutioning complete, ready for sprint planning (see `bmm-workflow-status.yaml`)

---

## Version History

| Version | Date | Description |
|---------|------|-------------|
| **MVP** | 2025-11-12 | Architecture expanded to full PRD alignment (Option B) |
| v0.2.0-alpha | 2025-11-11 | DHT-only architecture (superseded by MVP) |
| v0.1.0-alpha (v11) | 2025-11-10 | Initial release with centralized discovery (archived) |

---

**Ready for Development**: All planning and solutioning phases complete. Architecture validated. Proceed to sprint planning and story creation.
