# ShadowMesh Quick Status

**Last Updated**: November 10, 2025

---

## Current Version

**v0.1.0-alpha** ✅ Released
- Performance: 28.3 Mbps (45% faster than Tailscale)
- Video streaming: Successful (640x480 @ 547 kb/s)
- Stability: 3-hour test, zero packet loss
- **Blocker**: Centralized discovery (shut down)

---

## Active Work

**v0.2.0-alpha** 🔄 In Development (Sprint 0, Week 1 of 4)
- **Goal**: Kademlia DHT for standalone operation
- **Timeline**: 4 weeks (November 10 - December 8, 2025)
- **Current Sprint**: Foundation (PeerID, routing table, k-buckets)
- **Progress**: 0/15 tickets complete

---

## This Week's Focus

1. TICKET-001: PeerID generation from ML-DSA-87 keys
2. TICKET-002: XOR distance calculations
3. TICKET-003: PeerInfo data structures
4. TICKET-004: k-bucket LRU eviction

---

## Next Milestone

**v0.2.0-alpha Release** (Week 8 - December 8, 2025)
- Standalone operation (no discovery server)
- 3 bootstrap nodes deployed (US, EU, Asia)
- Performance ≥25 Mbps (maintain v11 baseline)
- 85%+ test coverage

---

## Full Details

See `PROJECT_STATUS.md` for complete roadmap, architecture, and implementation plan.

---

## Quick Links

- **Roadmap**: `PROJECT_STATUS.md`
- **Architecture**: `docs/2-ARCHITECTURE/KADEMLIA_DHT_ARCHITECTURE.md`
- **Implementation**: `docs/3-IMPLEMENTATION/DHT_IMPLEMENTATION_TICKETS.md`
- **Testing**: `docs/3-IMPLEMENTATION/TESTING_STRATEGY_DHT.md`
- **Deployment**: `docs/4-OPERATIONS/BOOTSTRAP_NODE_DEPLOYMENT.md`
