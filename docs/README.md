# 📚 ShadowMesh Documentation

Welcome to the ShadowMesh documentation. This guide will help you navigate our organized documentation structure.

## 🗂️ Documentation Structure

### [1-PRODUCT/](1-PRODUCT/) - Product Management
Strategic planning, roadmaps, and market analysis.

**Key Documents:**
- [ROADMAP.md](1-PRODUCT/ROADMAP.md) - 18-week development roadmap
- [PRD.md](prd.md) - Product Requirements Document
- [COMPETITIVE_ANALYSIS.md](1-PRODUCT/COMPETITIVE_ANALYSIS.md) - vs Tailscale/WireGuard

### [2-ARCHITECTURE/](2-ARCHITECTURE/) - Technical Design
Architecture specifications and design decisions.

**Key Documents:**
- [DECENTRALIZED_P2P.md](2-ARCHITECTURE/DECENTRALIZED_P2P.md) - Kademlia DHT design
- [SECURITY_SPECS.md](architecture/ENHANCED_SECURITY_SPECS.md) - Post-quantum cryptography
- [Architecture Decisions](2-ARCHITECTURE/decisions/) - ADRs and evaluations

### [3-IMPLEMENTATION/](3-IMPLEMENTATION/) - Developer Guides
Code structure, development guidelines, and testing.

**Key Documents:**
- [DEVELOPMENT_GUIDELINES.md](3-IMPLEMENTATION/DEVELOPMENT_GUIDELINES.md) - Coding standards
- [TESTING_GUIDE.md](3-IMPLEMENTATION/testing/TESTING_GUIDE.md) - Testing strategy
- [DEVELOPMENT_TIMELINE.md](3-IMPLEMENTATION/DEVELOPMENT_TIMELINE.md) - Project history

### [4-OPERATIONS/](4-OPERATIONS/) - DevOps & Support
Monitoring, troubleshooting, and operational procedures.

**Key Documents:**
- Monitoring guides
- Troubleshooting procedures
- Runbooks

### [archive/](archive/) - Historical Documentation
Completed epics, version-specific docs, and obsolete files.

**Subdirectories:**
- [epic2/](archive/epic2/) - Epic 2 completion reports
- [v10/](archive/v10/) - Version 10 documentation
- [v11/](archive/v11/) - Version 11 documentation
- [obsolete/](archive/obsolete/) - Deprecated documentation

---

## 🔐 Confidential Documentation

**`.project-private/`** contains sensitive infrastructure, testing, and deployment documentation with production IPs, credentials, and server topology. This directory is excluded from git.

**Access**: Authorized team members only. See `.project-private/README.md` for details.

---

## 📖 Quick Navigation

### For New Developers
1. Start with [README.md](../README.md) (project overview)
2. Read [DEVELOPMENT_GUIDELINES.md](3-IMPLEMENTATION/DEVELOPMENT_GUIDELINES.md)
3. Follow [Quick Start Guide](../README.md#quick-start)

### For Product Managers
1. Review [ROADMAP.md](1-PRODUCT/ROADMAP.md)
2. Read [PRD.md](prd.md) for requirements
3. Check [COMPETITIVE_ANALYSIS.md](1-PRODUCT/COMPETITIVE_ANALYSIS.md)

### For Architects
1. Read [DECENTRALIZED_P2P.md](2-ARCHITECTURE/DECENTRALIZED_P2P.md)
2. Review [Architecture Decisions](2-ARCHITECTURE/decisions/)
3. Study [SECURITY_SPECS.md](architecture/ENHANCED_SECURITY_SPECS.md)

### For QA Engineers
1. Start with [TESTING_GUIDE.md](3-IMPLEMENTATION/testing/TESTING_GUIDE.md)
2. Read [TEST_METHODOLOGY.md](3-IMPLEMENTATION/testing/TEST_METHODOLOGY.md)
3. Check [Sprint Roadmap](1-PRODUCT/ROADMAP.md) for testing requirements

---

## 🏗️ Project Status

**Current Version**: v11-phase3 (UDP + PQC) / v19 (QUIC prototype)
**Target**: Standalone Kademlia DHT + PQC + QUIC
**Timeline**: 18 weeks (Sprint 0-16)
**Next Milestone**: Sprint 0 - Architecture POC (4 weeks)

See [ROADMAP.md](1-PRODUCT/ROADMAP.md) for detailed timeline.

---

## 📝 Contributing to Documentation

1. Follow the 4-tier structure (PRODUCT → ARCHITECTURE → IMPLEMENTATION → OPERATIONS)
2. Never commit `.project-private/` contents to git
3. Use generic examples (no production IPs/credentials) in public docs
4. Update relevant README.md files when adding new documents
5. Archive obsolete docs to `archive/` instead of deleting

---

## 🔗 External Resources

- **GitHub**: [shadowmesh/shadowmesh](https://github.com/shadowmesh/shadowmesh) *(planned)*
- **Discord**: Community support server *(planned)*
- **Website**: https://shadowmesh.io *(planned)*

---

**Last Updated**: November 8, 2025
**Maintained by**: ShadowMesh Core Team
