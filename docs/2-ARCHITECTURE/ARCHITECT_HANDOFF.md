# ShadowMesh Architecture Handoff Document

**From:** John (Product Manager)
**To:** Architect
**Date:** 2025-10-31
**PRD Version:** 0.1
**Status:** Ready for Architecture Phase

---

## Executive Summary

ShadowMesh is a **post-quantum encrypted private network** targeting a 12-week MVP delivery. The PRD is complete with 44 functional requirements, 31 non-functional requirements, and 6 detailed epics covering 40+ user stories.

**Your mission:** Design the technical architecture that enables developers to build a quantum-safe, blockchain-coordinated private network with professional-grade monitoring.

**Readiness Score:** 92% (READY FOR ARCHITECT)

---

## Critical Context

### What Makes This Project Unique

1. **Post-Quantum Cryptography (PQC)**
   - Hybrid ML-KEM-1024 + X25519 for key exchange
   - Hybrid ML-DSA-87 + Ed25519 for signatures
   - Must achieve 1+ Gbps throughput with <5ms latency overhead
   - Research shows Cloudflare CIRCL achieves 22.7μs decapsulation (promising)

2. **Blockchain-Coordinated Relay Discovery**
   - chronara.eth ENS domain resolves to Ethereum smart contract
   - Relay nodes register on-chain with stake (0.1 ETH default)
   - Clients query contract for cryptographically verified relay list
   - Novel approach - competitors use centralized directories

3. **CGNAT Traversal Challenge**
   - Target: 95%+ connectivity success rate
   - UDP hole punching for direct P2P
   - 3-hop relay fallback with onion routing
   - **Highest technical risk** in project

4. **Professional Monitoring**
   - Grafana + Prometheus stack (user-requested enhancement)
   - 4-row dashboard: Connection Health, Network Performance, Security Metrics, Peer Map
   - Must stay under 900 MB total memory footprint

### Target Users

- **Enterprise Security Teams:** $50-200/user/month, need quantum-safe networking
- **Crypto-Native Users:** $30-50/month, comfortable with blockchain concepts
- **Privacy-Conscious Consumers:** $10-20/month in censored countries

### Success Metrics (MVP)

- 100-500 beta users acquired
- 80%+ successful connection rate within 10 minutes of installation
- 95%+ CGNAT traversal success rate
- 1+ Gbps throughput, <5ms latency overhead
- <$10 USD relay node registration gas cost

---

## Technology Stack (Decided)

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Client/Relay** | Go 1.21+ | Memory safety, crypto libraries, systemd support |
| **Smart Contracts** | Solidity 0.8.20+ | Ethereum standard, mature tooling |
| **Database** | PostgreSQL 14+ | ACID compliance, excellent Go support |
| **Monitoring** | Prometheus + Grafana | Professional UI, time-series optimized |
| **Transport** | WebSocket Secure (WSS) | Appears identical to HTTPS, bypasses DPI |
| **Networking** | TAP devices (Layer 2) | Stronger privacy (no IP header exposure) |
| **Deployment** | Docker Compose | Portability, version-pinned images |
| **CI/CD** | GitHub Actions | Integrated with GitHub repo |

**Libraries:**
- PQC: `github.com/cloudflare/circl`
- Classical Crypto: `crypto/ecdh`, `crypto/ed25519`, `golang.org/x/crypto/chacha20poly1305`
- Networking: `github.com/gorilla/websocket`, `github.com/songgao/water`
- Blockchain: `github.com/ethereum/go-ethereum`
- Metrics: `github.com/prometheus/client_golang`

---

## Your Architecture Deliverables

### 1. System Architecture Document (20-30 pages)

**Required Diagrams:**

**A. High-Level Component Architecture**
- Show: Client Daemon, Relay Node, Smart Contract, Prometheus, Grafana, Public Map, PostgreSQL
- Annotate: Communication protocols (WSS, JSON-RPC, HTTP)
- Format: Layered architecture (client layer, relay layer, blockchain layer, monitoring layer)

**B. Data Flow Diagrams**

**Scenario 1: Direct P2P Connection**
```
Client A → NAT Detection → UDP Hole Punching → PQC Handshake →
Encrypted Tunnel (ChaCha20-Poly1305) → TAP Device Injection → Client B
```

**Scenario 2: Relay Fallback (CGNAT)**
```
Client A → CGNAT Detected → Query chronara.eth Smart Contract →
Select 3 Relays (different operators) → Multi-Hop Onion Routing →
Relay1 → Relay2 → Relay3 → Client B
```

**C. Deployment Architecture**
- Client: systemd service, Docker Compose stack, .deb/.rpm packages
- Relay: VPS deployment (DigitalOcean/Linode), systemd service
- Smart Contract: Ethereum mainnet + Sepolia testnet
- Public Map: Vercel/Netlify static hosting

### 2. Database Schema (`schema.sql`)

**Required Tables:**

```sql
-- User/Device Management
users (user_id, telegram_user_id, username, created_at, updated_at)
devices (device_id, user_id, device_name, public_key_hash, keystore_encrypted, registered_at, last_seen)
peer_relationships (relationship_id, user_id, peer_user_id, relationship_type, created_at)

-- Connection Tracking
connection_history (connection_id, device_id, peer_device_id, connection_type, started_at, ended_at,
                    duration_seconds, bytes_sent, bytes_received, average_latency_ms, packet_loss_percentage)

-- Relay Node Registry (replicated from blockchain)
relay_nodes (node_id, blockchain_address, public_key_hash, endpoint, geographic_location,
             latitude, longitude, stake_amount, last_heartbeat, uptime_percentage, registered_at, last_synced_from_blockchain)

-- Audit Trail
access_logs (log_id, device_id, event_type, event_details JSONB, ip_address, created_at)
```

**Required Indexes:**
- Foreign key indexes: `devices.user_id`, `connection_history.device_id`
- Time-series indexes: `connection_history.started_at`, `access_logs.created_at`
- Query optimization: `relay_nodes.last_heartbeat` (for active node queries)

**Notes:**
- Reuse chronara.ai NFT bot pattern (`telegram_user_id` as user identifier)
- `relay_nodes` table is blockchain replica for faster querying (synced every 5 minutes)
- No data migration needed for MVP (new system)

### 3. Multi-Hop Routing Protocol Specification

**Critical Requirements:**

**Packet Format:**
```
[Outer Header | Layer 3 Encrypted Payload]
                 ↓ Relay3 decrypts
[Middle Header | Layer 2 Encrypted Payload]
                  ↓ Relay2 decrypts
[Inner Header | Layer 1 Encrypted Payload]
                 ↓ Relay1 decrypts
[Final Header | Plaintext Ethernet Frame] → Destination Peer
```

**Header Structure (per layer):**
- Protocol version (uint8)
- Hop number (uint8: 1, 2, or 3)
- Next relay ID (32 bytes: public key hash)
- Payload length (uint32)
- ChaCha20-Poly1305 nonce (12 bytes)
- Authentication tag (16 bytes)

**Path Construction Algorithm:**
1. Query chronara.eth contract: `getActiveNodes()` → list of relays
2. Filter: last_heartbeat <24 hours ago
3. Select 3 relays with constraints:
   - Different operators (verify `operator_address` field on-chain)
   - Geographic diversity (prefer different continents)
   - Low latency (measure via `/health` endpoint)
4. Perform PQC key exchange with each relay independently
5. Build onion: Encrypt Layer1 → Add Header → Encrypt Layer2 → Add Header → Encrypt Layer3
6. Transmit to Relay1 via WSS

**Relay Forwarding Logic:**
- Each relay decrypts one layer using pre-shared key
- Extracts `next_relay_id` from decrypted header
- If `next_relay_id == DESTINATION_PEER`, deliver to peer
- Else, forward packet to next relay

**Performance Target:**
- Multi-hop latency overhead: <50ms vs direct P2P
- If exceeded, fall back to 2-hop or 1-hop routing

**Operator Diversity Enforcement:**
- Prevent Sybil attacks (single operator controlling multiple relays in path)
- Client validates: `Relay1.operator_address ≠ Relay2.operator_address ≠ Relay3.operator_address`

### 4. Relay-to-Relay Authentication Design

**Challenge:** Prevent MITM attacks during multi-hop routing

**Recommended Approach: Blockchain-Based Mutual TLS**

**Flow:**
1. Each relay generates self-signed TLS certificate
2. Certificate public key hash stored in smart contract during registration
3. Client retrieves hashes from contract for all relays in path
4. Relay presents TLS certificate during connection
5. Client computes SHA-256 hash of certificate public key
6. Client compares with on-chain value: `relay_nodes[relay_id].public_key_hash`
7. If match → authenticated; if mismatch → reject and log security event

**Key Rotation:**
- Relay calls smart contract: `updatePublicKeyHash(new_hash)`
- Requires signature from operator's stake wallet (prevents unauthorized changes)
- Client caches hashes for 5 minutes to reduce blockchain queries

**Alternative to Consider:**
- Let's Encrypt certificates (simpler but centralized CA dependency)
- Direct ML-DSA-87 signatures per packet (higher CPU overhead)

**Your Decision:** Choose approach and document rationale

### 5. Prometheus Metrics Collection Optimization

**Challenge:** 20+ metrics exposed every 15 seconds must not impact network performance

**Target:** <1% CPU overhead

**Recommended Architecture:**

**In-Memory Storage:**
```go
type MetricsCollector struct {
    connectionStatus prometheus.Gauge
    connectionLatency prometheus.Gauge
    throughputBytes prometheus.Counter
    pqcHandshakes prometheus.Counter
    // ... (20+ metrics)

    mu sync.RWMutex  // Protect concurrent reads/writes
}
```

**Lock-Free Updates:**
- Use atomic operations for counters: `atomic.AddUint64(&m.pqcHandshakesTotal, 1)`
- Update gauges in dedicated goroutine (avoid blocking network I/O)
- Pre-compute metrics, serve from cache during Prometheus scrape

**HTTP Endpoint:**
- Use Prometheus client library: `promhttp.Handler()`
- Endpoint serves metrics in <100ms (no expensive computations during scrape)

**Validation:**
- Benchmark: Run daemon with 50 active connections, measure CPU with/without metrics
- If overhead >1%, reduce scrape frequency to 30s or disable expensive metrics

### 6. Technical Risk Assessment

**Your Task:** Update risk register with mitigation strategies

**Known Risks (from PRD):**

| Risk | Severity | Mitigation Strategy |
|------|----------|---------------------|
| **CGNAT Traversal <95%** | HIGH | Test 20+ NAT configs, ensure 10+ relays available, graceful degradation (3-hop → 2-hop → 1-hop) |
| **PQC Performance <1 Gbps** | MEDIUM | Early benchmarking (Epic 1), hardware profiling, SIMD optimizations |
| **Gas Costs >$10** | MEDIUM | Optimize to <20k gas, monitor gas prices, Layer 2 fallback if needed |
| **TAP Device Permissions** | LOW | Use CAP_NET_ADMIN via systemd, document security, TUN fallback |

**Additional Risks to Identify:**
- WebSocket scaling (1,000+ concurrent connections per relay)
- Smart contract event indexing for public map real-time updates
- PostgreSQL schema evolution (if requirements change mid-MVP)
- Prometheus retention disk space management (7 days × 15s intervals)

---

## Critical Constraints

**Performance (Non-Negotiable):**
- 1+ Gbps throughput on direct P2P connections
- <5ms latency overhead vs unencrypted baseline
- <50ms latency overhead for relay routing
- 1,000+ concurrent connections per relay node (2 vCPU, 4GB RAM VPS)
- <900 MB total memory (client daemon + Prometheus + Grafana)

**Security (Non-Negotiable):**
- No custom crypto code (use Cloudflare CIRCL + Go stdlib only)
- Smart contract audit before mainnet deployment
- Private keys never transmitted (keystore encrypted with AES-256-GCM)
- Relay node verification via blockchain (no trust, verify)

**Compatibility:**
- Linux kernel 3.10+ (4.19+ recommended)
- All major distros: Ubuntu 20.04+, Debian 11+, Fedora 36+, RHEL 8+, Arch, Manjaro, OpenSUSE
- x86_64 and ARM64 architectures
- Docker + Docker Compose for monitoring stack

**Blockchain:**
- Ethereum mainnet deployment
- chronara.eth ENS domain integration
- Gas cost target: <$10 USD at 25 gwei, $3,000 ETH
- Testable on Sepolia testnet without code changes

---

## Open Questions for You

Please document your decisions on these architectural choices:

1. **Relay Authentication:** Blockchain-based mutual TLS vs Let's Encrypt vs Direct ML-DSA-87 signatures?
2. **IPv6 Support:** Include in MVP or defer to Beta? (Currently IPv4-only)
3. **Smart Contract Rate Limiting:** Add rate limiting or trust Infura's limits?
4. **Token Rewards:** Relay operator incentives in MVP or Beta? (Currently placeholder Grafana panels)
5. **Database Connection Pooling:** Max pool size? Idle timeout? (High-volume connection logging)
6. **Prometheus Metric Cardinality:** Limit relay_node labels to prevent cardinality explosion?

---

## Timeline & Coordination

**Your Timeline:** 2-3 weeks for architecture deliverables

**Parallel Work:**
- UX Expert: Creating user journey diagrams and Grafana dashboard mockups (1-2 weeks)
- Epic 1 Development: Can start monorepo setup and crypto benchmarking in parallel

**Dependencies:**
- UX Expert needs your system architecture diagram for journey mapping
- Developers need your database schema before starting Epic 2 (Core Networking)
- Smart contract development blocked on your authentication mechanism design

**Handoff Meeting (Recommended):**
- Schedule 1-hour kickoff with PM, UX Expert, Lead Developer
- Walk through this document
- Answer open questions
- Align on timeline and dependencies

---

## PRD Reference

**Full PRD Location:** `/Users/jamestervit/Webcode/shadowmesh/docs/prd.md`

**Key Sections for You:**
- **Requirements (Lines 40-250):** All FR and NFR with acceptance criteria
- **Technical Assumptions (Lines 366-549):** Technology stack decisions and rationale
- **Epic Details (Lines 677-1623):** User stories with implementation context
- **Next Steps for Architect (Lines 2136-2561):** Detailed prompts and deliverable specs

**PM Availability:**
- Questions: Slack DM or email
- Quick decisions: <4 hour response time
- Architecture review: Schedule 30-min slot when draft ready

---

## Success Criteria for Your Deliverables

**System Architecture Document:**
- ✅ All components clearly defined with responsibilities
- ✅ Data flows documented for primary scenarios
- ✅ Deployment architecture covers VPS, client, blockchain, monitoring
- ✅ Diagrams use consistent notation (C4 model or UML)
- ✅ Performance bottlenecks identified and addressed

**Database Schema:**
- ✅ All tables support MVP functional requirements
- ✅ Indexes optimize common queries (relay discovery, connection history)
- ✅ Foreign key constraints enforce referential integrity
- ✅ Schema supports future enhancements without major refactoring

**Multi-Hop Routing Protocol:**
- ✅ Packet format specified with exact byte layout
- ✅ Encryption/decryption steps clearly documented
- ✅ Relay forwarding algorithm implementable by developers
- ✅ Performance validated via calculation (not just estimate)
- ✅ Operator diversity enforced programmatically

**Relay Authentication Design:**
- ✅ Mechanism prevents MITM attacks
- ✅ Integrates with blockchain verification
- ✅ Key rotation process documented
- ✅ Performance impact analyzed (<10ms overhead target)

**Prometheus Metrics Architecture:**
- ✅ <1% CPU overhead validated
- ✅ Lock-free update mechanism specified
- ✅ Metric cardinality bounded (prevent Prometheus memory explosion)
- ✅ Scrape interval and retention justified

**Technical Risk Assessment:**
- ✅ All known risks from PRD addressed
- ✅ New risks identified during architecture work documented
- ✅ Mitigation strategies feasible within MVP timeline
- ✅ Fallback options defined for high-severity risks

---

## Resources

**Code Examples:**
- chronara.ai NFT bot: `/Users/jamestervit/Webcode/nft-telegram-gate/` (PostgreSQL patterns)
- Cloudflare CIRCL: https://github.com/cloudflare/circl (PQC examples)

**Research Documents:**
- Priority Research Findings: `docs/research/priority-research-findings.md`
- Project Brief: `docs/brief.md`

**Standards:**
- NIST FIPS-203 (ML-KEM): https://csrc.nist.gov/pubs/fips/203/final
- NIST FIPS-204 (ML-DSA): https://csrc.nist.gov/pubs/fips/204/final
- Ethereum Smart Contracts: OpenZeppelin library patterns

---

## Next Steps

1. **Review this handoff document** - Flag any missing context
2. **Read full PRD** - Especially Requirements and Technical Assumptions sections
3. **Schedule kickoff meeting** - Align with PM and UX Expert
4. **Begin architecture work** - Start with high-level component diagram
5. **Weekly check-ins** - 30-min status updates with PM

**When you're ready to deliver:**
- Create architecture document in: `docs/architecture.md`
- Create database schema in: `database/schema.sql`
- Create protocol specs in: `docs/protocols/multi-hop-routing.md`
- Create auth design in: `docs/protocols/relay-authentication.md`
- Update this handoff with decisions on open questions

---

**Questions?** Contact John (PM) via Slack or email.

**Good luck!** You're building the future of quantum-safe private networking. 🚀

---

**Document Version:** 1.0
**Last Updated:** 2025-10-31
**Status:** Ready for Architect Review
