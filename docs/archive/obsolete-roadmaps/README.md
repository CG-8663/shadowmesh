# Archived Roadmap Documents

**Date Archived**: November 10, 2025
**Reason**: Consolidation to single source of truth

---

## Why These Documents Were Archived

These documents contained conflicting timelines, version numbers, and strategic approaches. They have been replaced by a single master roadmap.

### Documents in This Archive

1. **ROADMAP.md** (Nov 4, 2025)
   - Focus: Phase 2 (Direct P2P) complete, Phase 3 (Exit Nodes) next
   - Conflict: Exit Nodes not the current focus (DHT implementation is)
   - Replaced by: `PROJECT_STATUS.md`

2. **ROADMAP_MULTI_YEAR_STRATEGY.md** (Nov 4, 2025)
   - Focus: Multi-year business strategy, Epic 2 complete
   - Conflict: Complex multi-phase plan, AWS backbone strategy
   - Replaced by: `PROJECT_STATUS.md` (v1.0.0 section)

3. **DEVELOPMENT_TIMELINE.md**
   - Focus: v10 → v11 progression, Phase 3 UDP testing
   - Conflict: Version numbering confusion (v10, v11 vs v0.1.0-alpha)
   - Replaced by: `PROJECT_STATUS.md` (Current Status section)

4. **IMMEDIATE_ACTION_PLAN.md** (Nov 4, 2025)
   - Focus: Light node daemon, waiting for AWS credits
   - Conflict: Outdated approach (3 bootstrap nodes strategy adopted instead)
   - Replaced by: `docs/3-IMPLEMENTATION/DHT_IMPLEMENTATION_TICKETS.md`

---

## Current Source of Truth

**Master Roadmap**: `/PROJECT_STATUS.md`

This single document contains:
- Current status (v0.1.0-alpha achievements)
- Active work (v0.2.0-alpha DHT implementation)
- 8-week timeline (Sprint 0-1, Testing, Release)
- Long-term vision (v0.3.0-alpha QUIC, v1.0.0 production)

**Supporting Documents**:
- `docs/2-ARCHITECTURE/KADEMLIA_DHT_ARCHITECTURE.md` - DHT design
- `docs/2-ARCHITECTURE/MIGRATION_PATH.md` - 8-week migration plan
- `docs/3-IMPLEMENTATION/DHT_IMPLEMENTATION_TICKETS.md` - 15 implementation tickets
- `docs/3-IMPLEMENTATION/TESTING_STRATEGY_DHT.md` - Testing strategy
- `docs/4-OPERATIONS/BOOTSTRAP_NODE_DEPLOYMENT.md` - Infrastructure plan

---

## Key Changes in New Approach

1. **Version Naming**: v11, v19, v20+ → v0.1.0-alpha, v0.2.0-alpha, v1.0.0
2. **Timeline**: 18-week Sprint 0-16 → 8-week focused DHT implementation
3. **Infrastructure**: AWS backbone → 3 bootstrap nodes ($30-45/month)
4. **Scope**: Exit Nodes + Multi-Hop → DHT implementation (one thing at a time)
5. **Transport**: QUIC immediately → Keep UDP (proven), QUIC later (v0.3.0+)

---

## Reference Only

These documents are preserved for historical reference but should not be used for planning or development decisions.

**Always refer to**: `/PROJECT_STATUS.md` or `/STATUS.md` for current project state.
