# ShadowMesh Product Requirements Document (PRD)

**Project:** ShadowMesh - Post-Quantum Decentralized Private Network
**Version:** 0.1
**Last Updated:** 2025-10-31
**Status:** Draft - In Progress
**Author:** John (Product Manager)

---

## Goals and Background Context

### Goals

**MVP Goals (12-week timeline):**
- Validate hybrid post-quantum cryptography (ML-KEM-1024 + ML-DSA-87) achieves 1+ Gbps throughput with <5ms added latency
- Prove blockchain-coordinated CGNAT traversal achieves 95%+ connectivity success rate across diverse NAT configurations
- Deploy chronara.eth smart contract node registry on Ethereum mainnet with 10+ registered relay nodes
- Acquire 100-500 beta users from crypto/privacy communities with 80%+ successful connection rate within 10 minutes
- Achieve quantum-safe networking with mathematically verifiable relay node trust via smart contracts
- Establish open-source community with 1,000+ GitHub stars and 20+ external contributors
- Deliver Linux CLI client (all major distributions) with local statistics dashboard for end-user visibility
- Create public network map showing registered relay nodes with privacy-preserving design
- Validate first-mover advantage in post-quantum private networking (WireGuard/Tailscale 5+ years behind)

### Background Context

Current private networking technologies (WireGuard, Tailscale, ZeroTier) and traditional VPNs face four critical vulnerabilities that will render them obsolete: quantum vulnerability (harvest-now-decrypt-later attacks expected by 2030-2035), relay node trust problems (no cryptographic verification), system time manipulation enabling key rotation attacks, and deep packet inspection detection by censorship systems like China's Great Firewall. Priority research confirms that WireGuard and Tailscale have **no committed PQC implementation timelines** beyond "eventually," while ZeroTier implemented pre-FIPS-203 Kyber1024 but may need migration. This creates a confirmed **5+ year first-mover advantage** for ShadowMesh.

ShadowMesh solves all four vulnerabilities through a ground-up redesign: hybrid post-quantum cryptography (ML-KEM-1024 + ML-DSA-87) per NIST FIPS-203/204 standards, blockchain-enforced relay node verification via chronara.eth ENS domain, atomic clock synchronization (Beta phase), and WebSocket traffic obfuscation. Research validated that Cloudflare CIRCL ML-KEM-1024 achieves 22.7μs decapsulation (negligible impact on <500ms handshake target), smart contract gas costs are manageable (~$1.50 per node registration), but CGNAT traversal remains the highest-risk technical challenge requiring extensive testing. The product targets enterprise security teams ($50-200/user/month), crypto-native users ($30-50/month), and privacy-conscious consumers ($10-20/month) in censored countries.

### Change Log

| Date | Version | Description | Author |
|------|---------|-------------|--------|
| 2025-10-31 | 0.1 | Initial PRD creation based on Project Brief and Research Findings | John (PM) |

---

## Requirements

### Functional Requirements

**Cryptography & Security:**

**FR1:** The system shall implement hybrid post-quantum key exchange using ML-KEM-1024 (NIST FIPS-203) combined with classical X25519 ECDH for all peer-to-peer connections.

**FR2:** The system shall implement hybrid digital signatures using ML-DSA-87 (NIST FIPS-204) combined with classical Ed25519 for node identity verification and message authentication.

**FR3:** The system shall use ChaCha20-Poly1305 for symmetric encryption of all Ethernet frames transmitted over the mesh network.

**FR4:** The system shall rotate encryption keys every 5 minutes by default, using cryptographic key derivation from the PQC-protected session.

**FR5:** The system shall complete a full PQC handshake (key generation + encapsulation + decapsulation) in less than 500 milliseconds under normal network conditions.

**Peer Discovery & Connection:**

**FR6:** The system shall attempt direct P2P connections first using UDP hole punching before falling back to relay-based routing.

**FR7:** The system shall query the chronara.eth smart contract to discover available relay nodes when direct P2P connection fails or times out after 500ms.

**FR8:** The system shall establish Layer 2 encrypted tunnels using TAP devices that capture and encrypt raw Ethernet frames without exposing IP headers in transit.

**FR9:** The system shall maintain a local peer list with connection status (direct P2P, relay-routed, or offline) for all registered mesh network members.

**FR10:** The system shall automatically detect NAT type (full cone, restricted cone, port-restricted, symmetric) and select optimal traversal strategy.

**Relay Nodes & CGNAT Traversal:**

**FR11:** The system shall register relay nodes on the Ethereum mainnet via the chronara.eth ENS-resolved smart contract before accepting any client connections.

**FR12:** Relay nodes shall post heartbeat transactions to the smart contract at least every 24 hours to maintain active status.

**FR13:** The system shall route connections through a minimum of 3 relay hops when direct P2P fails, selecting hops from different operators to prevent single-node traffic analysis.

**FR14:** The system shall achieve 95% or higher connectivity success rate across diverse NAT configurations including CGNAT and symmetric NAT scenarios.

**FR15:** Relay nodes shall expose a health check endpoint returning current capacity, uptime percentage, and last heartbeat timestamp.

**Smart Contract Integration:**

**FR16:** The system shall deploy a Solidity smart contract to Ethereum mainnet that maintains a registry of all relay nodes with their public key hashes, stake amounts, and registration timestamps.

**FR17:** The smart contract shall require relay node operators to stake a minimum amount of ETH (configurable, default: 0.1 ETH) to register a node.

**FR18:** The smart contract shall implement automatic slashing of staked ETH for nodes that fail attestation checks or miss heartbeat deadlines (Beta phase feature, placeholder in MVP).

**FR19:** The system shall emit blockchain events for all node registrations, deregistrations, and status changes to enable off-chain indexing and public map updates.

**FR20:** The smart contract gas consumption for relay node registration shall not exceed $10 USD equivalent at median gas prices (25 gwei, $3,000 ETH).

**User Management & Authentication:**

**FR21:** The system shall generate a unique Ed25519 + ML-DSA-87 keypair for each client device on first launch, stored in an encrypted keystore.

**FR22:** The system shall support multi-device scenarios where a single user can register multiple devices (laptop, desktop, server) under one account.

**FR23:** The system shall maintain a PostgreSQL database storing user profiles, device registrations, peer relationships, and connection history with audit logging.

**FR24:** The system shall allow users to create friend lists and device groups for simplified peer discovery and access control.

**Linux CLI Client:**

**FR25:** The client shall run as a systemd service with commands: `shadowmesh connect`, `shadowmesh disconnect`, `shadowmesh status`, `shadowmesh logs`.

**FR26:** The client shall support all major Linux distributions: Ubuntu 20.04+, Debian 11+, Fedora 36+, RHEL/CentOS 8+, Arch, Manjaro, OpenSUSE.

**FR27:** The client shall automatically create and configure TAP network interfaces for Layer 2 mesh networking on service start.

**Grafana Local Dashboard:**

**FR28:** The client installation shall include a Docker Compose configuration that automatically deploys Grafana alongside Prometheus for the local monitoring dashboard.

**FR29:** The local Grafana dashboard shall be accessible at http://localhost:8080 and display real-time connection statistics, peer list, bandwidth usage, and key rotation status with professional visualizations across a 4-row grid layout:
- **Row 1 - Connection Health:** Connection status (connected/disconnected), NAT traversal type (direct P2P/relay), active peer count, relay node count
- **Row 2 - Network Performance:** Throughput graphs (Mbps/Gbps tx/rx), latency graphs (ms average), packet loss gauge (%)
- **Row 3 - Security Metrics:** PQC handshake counter (success/failure), key rotation timeline, crypto CPU usage (%)
- **Row 4 - Peer Map:** Geographic distribution of peers on world map, relay node list table with latency

**FR30:** The monitoring architecture shall use Prometheus for metrics collection:
- ShadowMesh daemon exposes metrics on `:9090/metrics` in Prometheus format
- Prometheus scrapes metrics every 15 seconds with 7-day retention
- Grafana queries Prometheus via PromQL for visualization
- Pre-configured dashboards provisioned automatically on first launch

**FR31:** The Docker Compose stack shall include three services:
- `shadowmesh-daemon`: Client daemon (host network mode, privileged for TAP devices)
- `prometheus`: Metrics collection (port 9091, 7-day retention, ~100-150 MB memory)
- `grafana`: Dashboard UI (port 8080, anonymous auth for localhost, ~150-200 MB memory)

**FR32:** The system shall provide three pre-configured Grafana dashboards:
- **Main User Dashboard:** Connection health, network performance, peer list (default view)
- **Relay Node Operator Dashboard:** Connected clients count, bandwidth usage, stake rewards, attestation status
- **Developer/Debug Dashboard:** Detailed crypto operation metrics, smart contract query latency, error logs

**FR33:** The client daemon shall expose comprehensive metrics in Prometheus format including:
- **Connection:** `shadowmesh_connection_status`, `shadowmesh_connection_latency_ms`, `shadowmesh_nat_traversal_type`
- **Network:** `shadowmesh_throughput_bytes_total`, `shadowmesh_packet_loss_ratio`, `shadowmesh_frame_encryption_rate_per_sec`
- **Cryptography:** `shadowmesh_pqc_handshakes_total`, `shadowmesh_key_rotation_total`, `shadowmesh_crypto_cpu_usage_percent`
- **Relay:** `shadowmesh_relay_nodes_available`, `shadowmesh_relay_node_latency_ms`, `shadowmesh_relay_hops_count`
- **System:** `shadowmesh_cpu_usage_percent`, `shadowmesh_memory_usage_bytes`, `shadowmesh_active_connections_count`

**FR34:** The client shall log all connection attempts, errors, and security events to local log files with configurable verbosity levels (debug, info, warn, error).

**Public Network Map:**

**FR35:** The system shall provide a public web-based map (separate service) that queries the chronara.eth smart contract and displays all registered relay nodes.

**FR36:** The network map shall show approximate geographic locations (city/country level) without revealing precise coordinates, IP addresses, or hostnames.

**FR37:** The network map shall display aggregate statistics: total relay nodes, geographic coverage (number of countries), average uptime percentage, and network health status.

**FR38:** The network map shall update within 60 seconds of new relay node registrations or status changes detected via blockchain event monitoring.

**FR39:** The network map shall never display private user data, connection graphs, traffic patterns, or client identifiers.

**WebSocket Transport & Obfuscation:**

**FR40:** All mesh network traffic shall be encapsulated in HTTPS WebSocket Secure (WSS) connections that appear identical to normal web traffic.

**FR41:** The system shall randomize packet sizes within reasonable bounds to prevent statistical fingerprinting by deep packet inspection systems.

**FR42:** The system shall include minimal timing randomization (jitter) on packet transmission to disrupt timing analysis while maintaining acceptable latency.

**Monitoring & Observability:**

**FR43:** The system shall collect and display metrics: connection success rate, P2P vs relay routing percentage, average throughput, median latency, key rotation frequency.

**FR44:** Relay nodes shall expose Prometheus-compatible metrics endpoints for monitoring by operators.

---

### Non-Functional Requirements

**Performance:**

**NFR1:** The system shall achieve a minimum of 1 Gbps throughput on direct P2P connections between peers on gigabit networks (target: 3-6 Gbps).

**NFR2:** The system shall add no more than 5 milliseconds of latency overhead for direct P2P connections when compared to unencrypted baseline.

**NFR3:** Relay-routed connections shall add no more than 50 milliseconds of additional latency compared to direct P2P connections.

**NFR4:** The system shall handle at least 1,000 concurrent connections per relay node on standard VPS hardware (2 CPU cores, 4GB RAM).

**NFR5:** Client memory usage shall not exceed 1GB RAM total under normal operating conditions with 50 active peer connections (including Grafana/Prometheus overhead: ~900 MB total system).

**Security:**

**NFR6:** The system shall be resistant to quantum computer attacks using Shor's algorithm (integer factorization and discrete logarithm) through hybrid PQC implementation.

**NFR7:** All cryptographic implementations shall use vetted libraries (Cloudflare CIRCL for PQC, Go standard library for classical crypto) with no custom crypto code.

**NFR8:** Private keys shall never be transmitted over the network; all key exchanges shall use public-key cryptography with perfect forward secrecy.

**NFR9:** The system shall be resilient to man-in-the-middle attacks through mutual authentication using hybrid digital signatures verified against smart contract registrations.

**NFR10:** Smart contracts shall undergo independent security audit before mainnet deployment, with all critical and high severity findings resolved.

**Reliability:**

**NFR11:** Relay node infrastructure shall maintain 99.9% uptime (less than 43 minutes downtime per month).

**NFR12:** The client shall automatically reconnect after network interruptions or relay node failures within 30 seconds without user intervention.

**NFR13:** The system shall gracefully handle smart contract query failures by caching last-known relay node list and attempting reconnection every 60 seconds.

**NFR14:** Database operations shall use connection pooling and retry logic to handle transient PostgreSQL failures without data loss.

**NFR15:** Prometheus metrics retention shall be 7 days with automatic cleanup of older data to manage disk space (~500 MB - 1 GB storage).

**Scalability:**

**NFR16:** The smart contract shall efficiently support at least 200 registered relay nodes without significant gas cost increases for read operations.

**NFR17:** The Grafana dashboard shall remain responsive with up to 500 registered peers in the mesh network.

**NFR18:** The public network map shall load and render within 3 seconds even with 200+ relay nodes displayed.

**Usability:**

**NFR19:** New users shall be able to install the client (including Docker Compose setup), complete device registration, and establish their first connection within 10 minutes without prior blockchain or VPN knowledge.

**NFR20:** The Grafana dashboard shall provide intuitive visualizations of connection status using color coding (green=connected, yellow=relay-routed, red=disconnected).

**NFR21:** Error messages shall be actionable and user-friendly, avoiding cryptographic jargon (e.g., "Connection failed: Unable to reach relay nodes" not "ML-KEM decapsulation error").

**NFR22:** The CLI shall provide clear help text for all commands with examples: `shadowmesh --help`, `shadowmesh connect --help`.

**Compliance & Operations:**

**NFR23:** All code shall be released as open-source under MIT or Apache 2.0 license to build trust and enable community contributions.

**NFR24:** The system shall log all security-relevant events (failed authentications, key rotations, relay failures) with timestamps and context for audit purposes.

**NFR25:** Smart contract code shall be verified on Etherscan to allow public inspection and trust verification.

**NFR26:** Gas optimization shall target <20,000 gas per relay node registration transaction to keep costs below $10 USD at $3,000 ETH and 25 gwei gas prices.

**Platform & Compatibility:**

**NFR27:** The client shall run on Linux kernel 3.10+ (with 4.19+ recommended) and require Docker, systemd, iptables, and TAP/TUN kernel modules as dependencies.

**NFR28:** The Grafana dashboard shall be compatible with modern browsers: Chrome/Edge 90+, Firefox 88+, Safari 14+.

**NFR29:** The client shall support both x86_64 (AMD64) and ARM64 architectures on Linux.

**NFR30:** Smart contracts shall be compatible with Ethereum mainnet and testable on Sepolia/Goerli testnets without code changes.

**NFR31:** Docker Compose configuration shall support offline installation with pre-pulled images for air-gapped or restricted network environments.

---

## User Interface Design Goals

### Overall UX Vision

**Philosophy:** "Professional Simplicity" - Deliver enterprise-grade monitoring capabilities with zero-configuration simplicity for technically sophisticated users.

**Core Principles:**
- **Information Density:** Maximize actionable information visible at a glance without overwhelming users
- **Real-Time Feedback:** Users see connection health, performance, and security status updating live
- **Progressive Disclosure:** Start with high-level health (green/yellow/red), allow drill-down for details
- **Dark Mode Default:** Crypto/DevOps community prefers dark themes; match Grafana's default aesthetic
- **Minimal Interaction Required:** Dashboard is for monitoring, not frequent clicking - status should be obvious at a glance

### Key Interaction Paradigms

**1. Monitor-First, Configure-Later**
- Users spend 95% of time passively monitoring dashboard, 5% actively troubleshooting
- Most common action: "Glance at dashboard to confirm connection is healthy"
- Design for "open in browser tab, check periodically" usage pattern

**2. Color-Coded Status Indicators**
- **Green:** All systems operational (direct P2P, good throughput, low latency)
- **Yellow:** Degraded but functional (relay-routed, higher latency, connection fallback)
- **Red:** Problem requiring attention (disconnected, failed CGNAT traversal, relay unavailable)
- Status colors consistent across all panels (connection, performance, security)

**3. Time-Series Visualization**
- All performance metrics show historical trends (last 1 hour, 6 hours, 24 hours, 7 days)
- Users can quickly spot patterns: "latency spiked at 3pm" or "throughput degraded overnight"
- Hover-over tooltips show exact values at specific timestamps

**4. Geographic Context**
- World map shows where peers and relay nodes are located
- Helps users understand why latency is high (peer in Asia, user in US)
- Visual confirmation of relay node geographic diversity

### Core Screens and Views

**1. Main User Dashboard (localhost:8080)**
- **Primary View:** 4-row grid with connection health, network performance, security metrics, peer map
- **Purpose:** At-a-glance connection status for end users
- **Access:** Default view on Grafana startup, no authentication required (localhost-only)

**2. Relay Node Operator Dashboard (localhost:8080/relay)**
- **Primary View:** Operator-focused metrics (connected clients, bandwidth usage, earnings, uptime)
- **Purpose:** Enable relay operators to monitor their infrastructure and optimize performance
- **Access:** Same Grafana instance, different dashboard selection

**3. Developer/Debug Dashboard (localhost:8080/debug)**
- **Primary View:** Detailed technical metrics (crypto operations, smart contract latency, error logs, system resources)
- **Purpose:** Troubleshooting connection issues, performance tuning, debugging
- **Access:** Same Grafana instance, advanced users only

**4. Public Network Map (https://map.shadowmesh.network)**
- **Primary View:** Interactive world map showing all chronara.eth registered relay nodes with aggregate statistics
- **Purpose:** Public transparency, allows prospective users to verify network coverage before signing up
- **Access:** Public website, no authentication

**5. CLI Help & Status Output**
- **View:** Terminal output from `shadowmesh status` command showing text-based connection summary
- **Purpose:** Quick status check without opening browser
- **Access:** Command line

### Accessibility

**Level:** **None** (MVP scope limitation)

**Rationale:** Target audience (crypto-native users, enterprise IT, command-line users) typically do not require accessibility features, and Grafana's default UI is not optimized for screen readers. Accessibility would be considered for mobile apps in Production phase (not MVP).

**Future Consideration (Beta/Production):** WCAG AA compliance for public network map website to ensure broad access.

### Branding

**Visual Identity:**
- **Logo:** chronara.ai logo (https://chronara.ai/images/chronara-small.png) displayed in Grafana dashboard header
- **Color Scheme:** Dark theme with accent colors:
  - Primary: Deep purple/blue (matches crypto aesthetic)
  - Success: Green (#00D68F - bright, clear)
  - Warning: Amber (#FFB300 - noticeable but not alarming)
  - Danger: Red (#FF5370 - urgent)
- **Typography:** Grafana default (Inter font) - clean, modern, professional

**chronara.eth Integration:**
- ENS domain prominently displayed in dashboard footer: "Powered by chronara.eth"
- Public network map shows "ShadowMesh Network powered by chronara.eth" branding
- All blockchain references use chronara.eth (human-readable) not contract address

**Professional Aesthetic:**
- Match Grafana's native dark theme (avoid custom CSS that breaks visual consistency)
- Use Grafana's built-in panel types (graphs, gauges, tables, maps)
- Minimal custom branding to maintain "battle-tested monitoring tool" feel

### Target Device and Platforms

**Primary:** **Web Responsive (Desktop/Laptop)**

**Supported Platforms:**
- **Desktop:** Linux workstations with 1920x1080+ resolution (primary target)
- **Laptop:** 1366x768+ resolution (responsive layout)
- **Tablet:** iPad/Android tablets in landscape mode (acceptable but not optimized)
- **Mobile:** NOT supported in MVP (Grafana mobile UX is poor, defer to mobile apps in Production phase)

**Browser Requirements:**
- Chrome/Edge 90+ (best Grafana performance)
- Firefox 88+ (fully supported)
- Safari 14+ (supported with some limitations on WebGL features)

**Public Network Map Platform:**
- **Primary:** Desktop web browsers (1920x1080+)
- **Secondary:** Mobile responsive (simplified view, no interactive map on small screens)

---

## Technical Assumptions

### Repository Structure: **Monorepo**

**Rationale:** Unified dependency management, simplified cross-component refactoring, single CI/CD pipeline, easier version synchronization between client/relay/contracts.

**Structure:**
```
shadowmesh/
├── client/              # Linux CLI client + daemon
│   ├── daemon/          # Background service (Go)
│   ├── cli/             # Command-line interface (Go)
│   └── dashboard/       # Grafana dashboard configs
├── relay/               # Relay node software (Go)
├── contracts/           # Solidity smart contracts
├── shared/              # Shared Go libraries (crypto, networking)
├── monitoring/          # Prometheus/Grafana configurations
├── tools/               # Build scripts, deployment tools
├── docs/                # Documentation, PRD, architecture
└── docker-compose.yml   # Local development stack
```

**Trade-offs:** Larger repo size but eliminates dependency versioning hell across repos.

---

### Service Architecture: **Monolithic Go Binaries + Docker Compose Microservices (Monitoring Only)**

**Rationale:**
- Client and relay are performance-critical network services → single Go binary for minimal overhead
- Monitoring (Grafana/Prometheus) benefits from containerization → Docker Compose for portability
- Hybrid approach balances performance with operational simplicity

**Components:**
- **Client Daemon:** Single Go binary running as systemd service (not containerized)
- **Relay Node:** Single Go binary on VPS (not containerized)
- **Monitoring Stack:** Docker Compose with 3 containers (Grafana, Prometheus, client daemon)

**Trade-offs:** Client requires native installation (not purely Docker-based), but achieves <5ms latency overhead target.

---

### Testing Requirements: **Unit Testing + Integration Testing** (Not Full Pyramid)

**Rationale:** MVP prioritizes core functionality validation over comprehensive test coverage.

**Testing Scope:**
- **Unit Tests:** Critical crypto operations, smart contract functions, NAT type detection
- **Integration Tests:** End-to-end connection establishment, relay fallback logic, blockchain queries
- **Manual Testing:** CGNAT traversal across diverse network configurations, dashboard UI/UX

**Coverage Target:** 80% for shared libraries, 60% for client/relay binaries

**Out of Scope (MVP):** Load testing, fuzz testing, chaos engineering, formal verification

---

### Programming Languages

**1. Go 1.21+ (Primary Language)**
- **Use Cases:** Client daemon, relay node software, CLI tools, all performance-critical networking code
- **Rationale:** Memory safety, excellent crypto library ecosystem, native systemd support, cross-compilation for ARM64/AMD64
- **Libraries:**
  - **PQC:** `github.com/cloudflare/circl` (ML-KEM, ML-DSA)
  - **Classical Crypto:** `crypto/ecdh`, `crypto/ed25519`, `golang.org/x/crypto/chacha20poly1305`
  - **Networking:** `github.com/gorilla/websocket`, `github.com/songgao/water` (TAP/TUN devices)
  - **Blockchain:** `github.com/ethereum/go-ethereum`, ENS libraries
  - **CLI:** `github.com/spf13/cobra`
  - **Metrics:** `github.com/prometheus/client_golang`

**2. Solidity 0.8.20+ (Smart Contracts)**
- **Use Cases:** chronara.eth relay node registry, staking, slashing (Beta), event emission
- **Rationale:** Ethereum standard, mature tooling, extensive audit resources
- **Frameworks:** Hardhat for testing, OpenZeppelin for libraries

**3. JavaScript/TypeScript (Minimal - Public Map Only)**
- **Use Cases:** Public network map web frontend
- **Rationale:** Web standard for interactive maps
- **Libraries:** React, Leaflet.js (map), ethers.js (blockchain queries)

**Not Used:** Python, Rust (considered but Go chosen for team familiarity and rapid iteration)

---

### Databases

**1. PostgreSQL 14+ (User/Device Management)**
- **Use Cases:** User profiles, device registrations, peer relationships, connection history, audit logs
- **Rationale:** ACID compliance, mature ecosystem, excellent Go support via `pgx`
- **Schema:** Multi-table design with `users`, `verified_users`, `access_logs` (reusing chronara.ai NFT bot patterns)

**2. Ethereum Mainnet (Relay Node Registry)**
- **Use Cases:** Relay node registrations, heartbeats, geographic metadata, stake management
- **Rationale:** Public verifiability, decentralized trust, immutable audit trail
- **Access:** Read-only queries via Infura/Alchemy, write via operator wallets

**Not Used:** Redis (caching deferred to Beta), MongoDB (relational data fits PostgreSQL better)

---

### Infrastructure & Deployment

**1. Docker + Docker Compose**
- **Use Cases:** Local monitoring stack (Grafana + Prometheus), development environment
- **Rationale:** Cross-platform portability, version-pinned images, simplified onboarding
- **Not Used For:** Client daemon production deployment (native systemd service preferred)

**2. Linux Distributions (All Major Distros)**
- **Supported:** Ubuntu 20.04+, Debian 11+, Fedora 36+, RHEL/CentOS 8+, Arch, Manjaro, OpenSUSE
- **Rationale:** TAP/TUN devices require kernel modules, systemd integration simplifies service management
- **Installation:** `.deb`, `.rpm`, and universal install script

**3. VPS Providers (Relay Nodes)**
- **Recommended:** DigitalOcean, Linode, Vultr, Hetzner (2 vCPU, 4GB RAM, $10-20/month)
- **Requirements:** Public IPv4, unrestricted ports 443/80, 1 Gbps network
- **Geographic Distribution:** Minimum 3 continents for relay diversity

**4. Monitoring**
- **Prometheus:** Metrics collection, 15-second scrape interval, 7-day retention
- **Grafana:** Dashboard UI, 3 pre-configured dashboards, localhost-only access
- **Resource Target:** ~900 MB total memory (client 100-150 MB, Prometheus 150 MB, Grafana 200 MB, overhead 450 MB)

---

### Testing Frameworks & CI/CD

**1. Testing**
- **Go:** `testing` package (standard library), `testify` for assertions
- **Solidity:** Hardhat with `chai` for smart contract tests
- **Integration:** Custom test harness simulating multi-peer scenarios

**2. CI/CD**
- **Platform:** GitHub Actions
- **Workflows:**
  - **PR Checks:** Unit tests, linters (golangci-lint, solhint), security scans (gosec)
  - **Release Pipeline:** Cross-compile binaries (AMD64/ARM64), build Docker images, publish to GitHub Releases
  - **Contract Deployment:** Hardhat scripts for testnet/mainnet deployment with manual approval

**3. Pre-Commit Hooks**
- `go fmt`, `go vet`, Solidity formatter
- Prevent commits with failing unit tests

---

### Security & Compliance

**1. Cryptography**
- **No Custom Crypto:** All implementations use Cloudflare CIRCL (PQC) and Go standard library (classical)
- **Key Storage:** Encrypted keystores using `crypto/aes` with user-provided passphrase
- **Randomness:** `crypto/rand` for all key generation and nonce creation

**2. Smart Contract Security**
- **Audit Requirement:** Independent audit before mainnet deployment (target: Trail of Bits, OpenZeppelin)
- **Bug Bounty:** Post-launch program with $10,000+ rewards for critical findings

**3. Licensing**
- **Open Source:** MIT or Apache 2.0 (to be decided based on community preference)
- **Third-Party Dependencies:** All dependencies vetted for compatible licenses

---

### Key Architectural Decisions

**1. Layer 2 (Ethernet) vs Layer 3 (IP)**
- **Decision:** Layer 2 using TAP devices
- **Rationale:** Prevents IP header exposure in transit (stronger privacy), supports non-IP protocols

**2. Blockchain for Relay Discovery vs Centralized Directory**
- **Decision:** Ethereum smart contract registry
- **Rationale:** Decentralized trust, public verifiability, aligns with crypto-native target audience

**3. Prometheus + Grafana vs Custom Dashboard**
- **Decision:** Grafana with pre-configured dashboards
- **Rationale:** Professional UI familiar to DevOps users, rich visualization library, time-series optimized

**4. WebSocket vs Raw UDP for Transport**
- **Decision:** HTTPS WebSocket Secure (WSS)
- **Rationale:** Appears identical to HTTPS traffic, bypasses deep packet inspection, works through corporate firewalls

**5. Hybrid PQC vs Pure PQC**
- **Decision:** Hybrid (ML-KEM + X25519, ML-DSA + Ed25519)
- **Rationale:** Defense-in-depth if PQC standards have undiscovered weaknesses, negligible performance penalty

---

## Epic List

The MVP is structured as **6 sequential epics** delivering incremental value over a **12-week timeline**. Each epic builds upon previous infrastructure and ends with a testable milestone.

### Epic 1: Foundation & Cryptography (Weeks 1-2)
**Goal:** Establish monorepo, implement hybrid PQC primitives, and validate cryptographic performance targets.

**Milestone:** Successfully complete 1,000 hybrid PQC handshakes in <500ms average with 1+ Gbps symmetric encryption throughput.

**Deliverables:**
- Monorepo structure with Go modules, Solidity contracts, shared libraries
- Hybrid key exchange (ML-KEM-1024 + X25519) implementation using Cloudflare CIRCL
- Hybrid signatures (ML-DSA-87 + Ed25519) for identity verification
- ChaCha20-Poly1305 frame encryption with 5-minute key rotation
- Benchmark test suite validating <5ms latency overhead
- Encrypted keystore for private key storage

**Dependencies:** None (foundational epic)

**Risk:** PQC performance may not meet 1+ Gbps target on commodity hardware (mitigated by research showing 22.7μs decapsulation)

---

### Epic 2: Core Networking & Direct P2P (Weeks 3-4)
**Goal:** Implement Layer 2 networking with TAP devices, WebSocket transport, and direct P2P connections without relay fallback.

**Milestone:** Two clients on same LAN establish direct P2P connection, complete PQC handshake, and transmit 1 Gbps encrypted traffic with <5ms added latency.

**Deliverables:**
- TAP device creation and Ethernet frame capture/injection
- WebSocket Secure (WSS) transport layer with TLS 1.3
- UDP hole punching for NAT traversal (full cone, restricted cone)
- NAT type detection (STUN-like protocol)
- Frame encryption pipeline (capture → encrypt → WSS → decrypt → inject)
- Basic CLI commands: `shadowmesh connect <peer-id>`, `shadowmesh disconnect`, `shadowmesh status`
- Integration tests for direct P2P handshake and data transfer

**Dependencies:** Epic 1 (cryptography)

**Risk:** TAP device configuration may require root/sudo privileges (mitigated by systemd service design)

---

### Epic 3: Smart Contract & Blockchain Integration (Weeks 5-6)
**Goal:** Deploy chronara.eth smart contract to Ethereum mainnet and enable relay node registration/discovery.

**Milestone:** Smart contract deployed to mainnet with 3+ relay nodes registered, clients can query registry and retrieve node metadata.

**Deliverables:**
- Solidity smart contract: `RelayNodeRegistry.sol` with registration, heartbeat, staking functions
- ENS integration: chronara.eth resolves to contract address
- Hardhat deployment scripts for Sepolia testnet and mainnet
- Go blockchain client: query relay node list, monitor events, verify node signatures
- Gas optimization to achieve <$10 registration cost (target: $1.50)
- Smart contract unit tests (Hardhat + Chai) with 90%+ coverage
- Etherscan verification of deployed contract

**Dependencies:** Epic 1 (signatures for node identity verification)

**Risk:** Gas costs may exceed $10 target if Ethereum congestion spikes (mitigated by optimizations and Layer 2 fallback in Beta)

---

### Epic 4: Relay Infrastructure & CGNAT Traversal (Weeks 7-9)
**Goal:** Implement relay node software, fallback routing, and achieve 95%+ connectivity across CGNAT/symmetric NAT scenarios.

**Milestone:** Clients behind CGNAT successfully connect through 3-hop relay routing with 95%+ success rate across 20+ diverse network configurations.

**Deliverables:**
- Relay node Go binary with multi-client routing, capacity management, health checks
- Client relay fallback logic: detect P2P failure → query smart contract → select 3 relays → establish connection
- Multi-hop routing (minimum 3 relays from different operators)
- Relay node systemd service, installation scripts, operator documentation
- VPS deployment guide (DigitalOcean, Linode, Vultr)
- CGNAT test matrix: symmetric NAT, port-restricted NAT, double NAT scenarios
- Performance validation: relay-routed latency <50ms overhead vs direct P2P

**Dependencies:** Epic 2 (networking), Epic 3 (relay discovery)

**Risk:** CGNAT traversal may not achieve 95% success rate (highest technical risk - mitigated by 3-hop routing and extensive testing)

---

### Epic 5: Monitoring & Grafana Dashboard (Weeks 10-11)
**Goal:** Implement Prometheus metrics, Grafana dashboards, and Docker Compose monitoring stack.

**Milestone:** Client installation includes working Grafana dashboard at localhost:8080 showing live connection stats, throughput graphs, and peer map.

**Deliverables:**
- Prometheus metrics exposition endpoint (`:9090/metrics`) in client daemon
- Comprehensive metric taxonomy: connection, network, crypto, relay, system metrics
- Docker Compose configuration: shadowmesh-daemon, prometheus, grafana services
- 3 pre-configured Grafana dashboards: Main User, Relay Operator, Developer/Debug
- 4-row Main User Dashboard: Connection Health, Network Performance, Security Metrics, Peer Map
- Automatic dashboard provisioning on first Grafana launch
- Resource optimization: total stack <900 MB memory, 7-day Prometheus retention
- Installation script modifications to include Docker setup

**Dependencies:** Epic 2 (client daemon), Epic 4 (relay metrics)

**Risk:** Grafana memory overhead may exceed 200 MB target (mitigated by disabling unused plugins and provisioning minimal datasources)

---

### Epic 6: Public Map, Documentation & Launch (Week 12)
**Goal:** Build public network map, finalize documentation, execute beta launch to crypto/privacy communities.

**Milestone:** Public map live at map.shadowmesh.network, 100-500 beta users acquired, 80%+ successful connection rate within 10 minutes.

**Deliverables:**
- Public network map website (React + Leaflet.js) querying chronara.eth contract
- Geographic relay node visualization with aggregate stats (total nodes, countries, uptime)
- Privacy-preserving design (no user data, city-level location only)
- Comprehensive user documentation: installation guides, troubleshooting, FAQ
- Operator documentation: relay node setup, staking, monitoring
- GitHub repository cleanup: README, CONTRIBUTING, CODE_OF_CONDUCT, LICENSE (MIT/Apache 2.0)
- Beta launch strategy: Reddit (r/privacy, r/cryptocurrency), Twitter/X, Product Hunt
- Community setup: Discord server, GitHub Discussions
- Monitoring & analytics: user acquisition funnel, connection success rates, error logs

**Dependencies:** Epic 5 (monitoring for analytics), Epic 3 (contract for map data)

**Risk:** Beta user acquisition may fall short of 100 users (mitigated by targeted outreach to crypto influencers and privacy advocates)

---

## Epic Details

### Epic 1: Foundation & Cryptography (Weeks 1-2)

**Epic Goal:** Establish monorepo, implement hybrid PQC primitives, and validate cryptographic performance targets.

**User Stories:**

#### Story 1.1: Monorepo Setup
**As a** developer
**I want** a well-structured monorepo with clear separation between client, relay, contracts, and shared code
**So that** I can efficiently develop and maintain multiple interdependent components

**Acceptance Criteria:**
- Directory structure follows documented layout (client/, relay/, contracts/, shared/, monitoring/, docs/)
- Go modules configured with proper versioning (go.mod, go.sum)
- Hardhat project initialized in contracts/ with TypeScript support
- Pre-commit hooks configured (go fmt, go vet, solhint)
- GitHub Actions workflow validates builds on PR
- README.md explains repository structure and development commands

**Estimate:** 2 days

---

#### Story 1.2: Hybrid Key Exchange Implementation
**As a** ShadowMesh client
**I want** quantum-resistant key exchange using ML-KEM-1024 + X25519
**So that** my connections are protected from future quantum attacks

**Acceptance Criteria:**
- Implementation uses `github.com/cloudflare/circl` for ML-KEM-1024
- Classical X25519 implemented using `crypto/ecdh` from Go standard library
- Hybrid KDF combines both shared secrets using HKDF-SHA256
- Key generation completes in <100ms on commodity hardware
- Encapsulation + decapsulation roundtrip <50ms (measured via benchmark)
- Unit tests cover key generation, encapsulation, decapsulation, error cases
- Test vectors validate interoperability with NIST FIPS-203 reference implementation

**Estimate:** 3 days

---

#### Story 1.3: Hybrid Digital Signatures
**As a** relay node operator
**I want** my node identity verified using quantum-resistant signatures (ML-DSA-87 + Ed25519)
**So that** clients can cryptographically verify my node registration

**Acceptance Criteria:**
- Implementation uses `github.com/cloudflare/circl` for ML-DSA-87
- Classical Ed25519 implemented using `crypto/ed25519` from Go standard library
- Signing function produces hybrid signature (ML-DSA || Ed25519)
- Verification function validates both signatures independently
- Signature generation <10ms, verification <5ms
- Unit tests cover signing, verification, error cases (invalid signatures, wrong public key)
- Integration with smart contract public key hash validation

**Estimate:** 2 days

---

#### Story 1.4: Symmetric Encryption Pipeline
**As a** ShadowMesh client
**I want** all Ethernet frames encrypted using ChaCha20-Poly1305
**So that** my Layer 2 traffic is protected from eavesdropping

**Acceptance Criteria:**
- ChaCha20-Poly1305 AEAD cipher implemented using `golang.org/x/crypto/chacha20poly1305`
- Encryption function takes plaintext frame, returns ciphertext + authentication tag
- Decryption function validates tag before returning plaintext (constant-time comparison)
- Unique 96-bit nonce generated for each frame using counter + random component
- Benchmark validates 1+ Gbps throughput on single CPU core (4 GHz Intel/AMD)
- Unit tests cover encryption, decryption, nonce uniqueness, tag validation failures

**Estimate:** 2 days

---

#### Story 1.5: Key Rotation Mechanism
**As a** security-conscious user
**I want** encryption keys rotated every 5 minutes by default
**So that** forward secrecy is maintained even if session keys are compromised

**Acceptance Criteria:**
- Key rotation timer triggers every 5 minutes (configurable via flag: `--key-rotation-interval`)
- New session key derived from existing session using HKDF with incremented counter
- Rotation protocol exchanges new public keys using existing encrypted channel
- Old keys securely zeroed from memory after rotation (use `memguard` or similar)
- Rotation completes without dropping active connections (<100ms switchover)
- Unit tests cover rotation trigger, key derivation, old key cleanup
- Integration test validates multiple rotations over 15-minute connection

**Estimate:** 3 days

---

#### Story 1.6: Encrypted Keystore
**As a** user
**I want** my private keys stored encrypted on disk
**So that** an attacker with filesystem access cannot steal my identity

**Acceptance Criteria:**
- Keystore file format: JSON with encrypted private key fields
- Encryption uses AES-256-GCM with key derived from user passphrase (PBKDF2, 100k iterations)
- Keystore stores hybrid keypairs: (ML-KEM + X25519) and (ML-DSA + Ed25519)
- CLI prompts for passphrase on first run: `shadowmesh init --passphrase`
- Passphrase cached in memory (not on disk) for duration of daemon process
- Keystore file permissions restricted to user (chmod 600)
- Unit tests cover keystore creation, encryption, decryption, wrong passphrase handling

**Estimate:** 2 days

---

#### Story 1.7: Performance Benchmarking Suite
**As a** developer
**I want** automated benchmarks validating cryptographic performance targets
**So that** I can detect performance regressions before deployment

**Acceptance Criteria:**
- Go benchmark suite (`crypto_bench_test.go`) measures:
  - ML-KEM-1024 encapsulation/decapsulation (target: <50ms combined)
  - ML-DSA-87 signing/verification (target: <15ms combined)
  - ChaCha20-Poly1305 throughput (target: 1+ Gbps)
  - Full handshake latency (target: <500ms)
- Benchmark results saved to file for comparison across commits
- GitHub Actions runs benchmarks on PR and comments with results
- Regression detection: fail CI if latency increases >10% or throughput decreases >10%

**Estimate:** 2 days

---

**Epic 1 Total Estimate:** 16 days (2 weeks with some buffer)

---

### Epic 2: Core Networking & Direct P2P (Weeks 3-4)

**Epic Goal:** Implement Layer 2 networking with TAP devices, WebSocket transport, and direct P2P connections without relay fallback.

**User Stories:**

#### Story 2.1: TAP Device Management
**As a** ShadowMesh client daemon
**I want** to create and configure TAP network interfaces
**So that** I can capture and inject raw Ethernet frames

**Acceptance Criteria:**
- Implementation uses `github.com/songgao/water` for TAP device creation
- Daemon creates TAP device with name `shadowmesh0` on startup
- TAP interface assigned virtual MAC address and IP address from 10.99.0.0/16 subnet
- systemd service runs with CAP_NET_ADMIN capability for TAP management
- TAP device destroyed cleanly on daemon shutdown
- Unit tests (requires root or CAP_NET_ADMIN) validate device creation, configuration, cleanup
- Documentation explains TAP device architecture and debugging commands (`ip link show`)

**Estimate:** 3 days

---

#### Story 2.2: Ethernet Frame Capture
**As a** ShadowMesh client
**I want** to capture Ethernet frames from the TAP device
**So that** I can encrypt and transmit them over the mesh

**Acceptance Criteria:**
- Frame capture loop reads from TAP device file descriptor
- Frames parsed to extract destination MAC, source MAC, EtherType, payload
- Non-IP frames supported (ARP, IPv6, custom protocols)
- Capture thread does not block encryption/transmission threads (buffered channels)
- Frame size limit enforced (1500 bytes MTU + 14 bytes Ethernet header)
- Malformed frames logged and dropped (metrics counter incremented)
- Unit tests validate frame parsing, size limits, protocol handling

**Estimate:** 2 days

---

#### Story 2.3: WebSocket Secure (WSS) Transport
**As a** ShadowMesh client
**I want** all mesh traffic encapsulated in HTTPS WebSocket connections
**So that** my traffic appears identical to normal web traffic

**Acceptance Criteria:**
- WebSocket server listens on configurable port (default: 443 for WSS)
- TLS 1.3 certificate auto-generated (self-signed) or loaded from file
- WebSocket Upgrade handshake completes with proper headers (Sec-WebSocket-Key, etc.)
- Binary frames used for encrypted Ethernet frame transport (not text frames)
- Connection keepalive: ping/pong frames every 30 seconds to detect dead connections
- Graceful shutdown: close handshake sent before TCP FIN
- Unit tests cover WebSocket handshake, frame transmission, keepalive, shutdown

**Estimate:** 3 days

---

#### Story 2.4: NAT Type Detection
**As a** ShadowMesh client
**I want** my NAT type automatically detected
**So that** the system can select optimal traversal strategy

**Acceptance Criteria:**
- NAT detection uses STUN-like protocol querying external server
- Detected types: Full Cone, Restricted Cone, Port-Restricted Cone, Symmetric, No NAT
- Detection result cached for connection lifetime (re-tested on network change)
- Configuration flag to skip detection: `--nat-type=<type>` for debugging
- Detection completes in <2 seconds
- Unit tests cover detection logic for each NAT type (mocked STUN responses)
- Integration test validates detection against real NAT routers (manual testing)

**Estimate:** 3 days

---

#### Story 2.5: UDP Hole Punching
**As a** ShadowMesh client behind NAT
**I want** direct P2P connections via UDP hole punching
**So that** I can avoid relay routing when possible

**Acceptance Criteria:**
- Hole punching implemented for Full Cone and Restricted Cone NAT types
- Client sends UDP packets to peer's public IP:port (learned via signaling)
- Simultaneous open: both peers send packets at same time to create bidirectional mapping
- Fallback to relay if hole punching times out after 500ms
- Success rate >80% for Full Cone NAT, >60% for Restricted Cone NAT (tested on lab network)
- Unit tests cover packet sending, timeout logic, fallback trigger
- Integration test validates successful hole punching between two Full Cone NATs

**Estimate:** 4 days

---

#### Story 2.6: Frame Encryption Pipeline
**As a** ShadowMesh client
**I want** seamless encryption and decryption of Ethernet frames
**So that** Layer 2 traffic is protected end-to-end

**Acceptance Criteria:**
- Pipeline stages: TAP capture → encrypt → WSS transmit → WSS receive → decrypt → TAP inject
- Each stage runs in separate goroutine with buffered channels for communication
- Encryption adds 16-byte authentication tag (ChaCha20-Poly1305 overhead)
- Decryption validates tag before forwarding frame (drops invalid frames)
- Frame counter prevents replay attacks (monotonically increasing nonce)
- Performance: pipeline processes 10,000+ frames/second on commodity hardware
- Unit tests cover each pipeline stage independently
- Integration test validates end-to-end frame transmission between two clients

**Estimate:** 4 days

---

#### Story 2.7: CLI Commands (Connect, Disconnect, Status)
**As a** user
**I want** simple CLI commands to manage connections
**So that** I can control ShadowMesh without editing config files

**Acceptance Criteria:**
- `shadowmesh connect <peer-id>` initiates connection to specified peer
- `shadowmesh disconnect` terminates active connection gracefully
- `shadowmesh status` displays connection state, peer info, throughput, latency
- Commands communicate with daemon via Unix socket or localhost HTTP API
- CLI uses `github.com/spf13/cobra` for command structure and help text
- `shadowmesh --help` displays all available commands with examples
- Unit tests validate command parsing and API calls

**Estimate:** 2 days

---

#### Story 2.8: Direct P2P Integration Test
**As a** QA engineer
**I want** automated integration tests validating direct P2P connections
**So that** I can verify end-to-end functionality before each release

**Acceptance Criteria:**
- Test spins up two ShadowMesh clients on localhost (different TAP devices, different ports)
- Test completes full PQC handshake between clients
- Test transmits 1 GB of data and validates 100% delivery (no packet loss)
- Test measures throughput (target: 1+ Gbps) and latency (target: <5ms overhead)
- Test runs in CI environment (GitHub Actions, requires CAP_NET_ADMIN or root)
- Test logs detailed timing information for performance regression detection

**Estimate:** 3 days

---

**Epic 2 Total Estimate:** 24 days (approximately 3-4 weeks; timeline shows 2 weeks, which assumes parallel work by multiple developers)

---

### Epic 3: Smart Contract & Blockchain Integration (Weeks 5-6)

**Epic Goal:** Deploy chronara.eth smart contract to Ethereum mainnet and enable relay node registration/discovery.

**User Stories:**

#### Story 3.1: RelayNodeRegistry Smart Contract
**As a** relay node operator
**I want** to register my node on-chain via chronara.eth smart contract
**So that** clients can discover and verify my node cryptographically

**Acceptance Criteria:**
- Solidity contract implements:
  - `registerNode(bytes32 pubKeyHash, string location, string endpoint)` - registers relay node
  - `updateHeartbeat()` - updates last heartbeat timestamp
  - `deregisterNode()` - removes node from registry
  - `getActiveNodes()` - returns list of nodes with recent heartbeats (<24 hours)
- Contract requires 0.1 ETH stake (configurable via constructor parameter)
- Contract emits events: `NodeRegistered`, `NodeDeregistered`, `HeartbeatUpdated`
- Contract uses OpenZeppelin libraries for security (ReentrancyGuard, Ownable)
- Gas optimization: registration costs <20,000 gas (target: ~15,000 gas)
- Hardhat unit tests achieve 95%+ code coverage

**Estimate:** 5 days

---

#### Story 3.2: ENS Integration (chronara.eth)
**As a** user
**I want** the relay registry accessible via chronara.eth ENS domain
**So that** I don't need to remember contract addresses

**Acceptance Criteria:**
- chronara.eth ENS domain configured to resolve to RelayNodeRegistry contract address
- Go client queries ENS resolver to get contract address dynamically
- Fallback to hardcoded address if ENS query fails (logged as warning)
- Documentation explains ENS setup process for mainnet and testnets
- Unit tests validate ENS resolution logic (mocked ENS resolver)

**Estimate:** 2 days

---

#### Story 3.3: Hardhat Deployment Scripts
**As a** DevOps engineer
**I want** automated deployment scripts for testnet and mainnet
**So that** contract deployment is repeatable and auditable

**Acceptance Criteria:**
- Hardhat deployment scripts in `contracts/scripts/deploy.ts`
- Scripts support Sepolia testnet and Ethereum mainnet (environment variable: `NETWORK`)
- Scripts verify contract on Etherscan after deployment
- Scripts save deployed contract address to `deployments/<network>.json`
- Manual approval step required before mainnet deployment (human confirmation prompt)
- Deployment guide documents required environment variables (INFURA_URL, PRIVATE_KEY, ETHERSCAN_API_KEY)

**Estimate:** 2 days

---

#### Story 3.4: Go Blockchain Client (Query Registry)
**As a** ShadowMesh client
**I want** to query the relay node registry from my Go daemon
**So that** I can discover available relays when direct P2P fails

**Acceptance Criteria:**
- Go implementation uses `github.com/ethereum/go-ethereum` for blockchain queries
- Client queries `getActiveNodes()` function and parses returned struct array
- Query retry logic: 3 attempts with exponential backoff if RPC fails
- Results cached for 5 minutes to avoid excessive RPC calls
- Client monitors `NodeRegistered` and `NodeDeregistered` events via WebSocket subscription
- Unit tests validate query logic, caching, event monitoring (mocked Ethereum client)

**Estimate:** 4 days

---

#### Story 3.5: Node Signature Verification
**As a** ShadowMesh client
**I want** to verify relay node signatures against on-chain public key hashes
**So that** I can detect man-in-the-middle attacks

**Acceptance Criteria:**
- Client retrieves public key hash from smart contract for target relay node
- Client requests full public key from relay node via HTTP API
- Client computes SHA-256 hash of received public key and compares with on-chain hash
- Client verifies signature on relay node's TLS certificate using validated public key
- Connection rejected if hash mismatch or signature invalid (logged as security event)
- Unit tests cover hash computation, signature verification, error cases

**Estimate:** 3 days

---

#### Story 3.6: Gas Optimization & Cost Analysis
**As a** product manager
**I want** relay node registration to cost <$10 USD at median gas prices
**So that** operators can afford to register nodes

**Acceptance Criteria:**
- Gas profiling identifies most expensive operations in registration function
- Optimizations applied: pack struct variables, use `calldata` instead of `memory`, minimize SSTORE operations
- Registration gas cost measured: <20,000 gas per transaction
- Cost calculation documented in README: gas * price * ETH/USD (example: 15,000 * 25 gwei * $3,000 = $1.13)
- Hardhat gas reporter plugin configured to track gas usage across all functions

**Estimate:** 2 days

---

#### Story 3.7: Smart Contract Security Testing
**As a** security engineer
**I want** comprehensive security tests for the smart contract
**So that** I can identify vulnerabilities before mainnet deployment

**Acceptance Criteria:**
- Hardhat test suite covers:
  - Reentrancy attacks (attempt to call registerNode recursively)
  - Integer overflow/underflow (Solidity 0.8+ has built-in protection, validate it works)
  - Access control (only node owner can update heartbeat or deregister)
  - Front-running (test if transaction order affects outcomes)
  - Denial of service (test with 200+ registered nodes)
- Slither static analysis tool run with zero high/medium findings
- Mythril symbolic execution tool run (if time permits)
- Test coverage report shows 95%+ line coverage

**Estimate:** 3 days

---

#### Story 3.8: Testnet Deployment & Validation
**As a** QA engineer
**I want** to deploy and test the contract on Sepolia testnet
**So that** I can validate functionality before mainnet deployment

**Acceptance Criteria:**
- Contract deployed to Sepolia testnet with 3 test relay nodes registered
- Sepolia deployment verified on Etherscan (source code visible)
- Integration test queries Sepolia contract from Go client and retrieves node list
- Test validates heartbeat update mechanism (nodes with stale heartbeats excluded)
- Test validates deregistration (node removed from active list)
- Documentation includes Sepolia contract address and Sepolia ETH faucet links

**Estimate:** 2 days

---

**Epic 3 Total Estimate:** 23 days (approximately 3-4 weeks; timeline shows 2 weeks with parallel work)

---

### Epic 4: Relay Infrastructure & CGNAT Traversal (Weeks 7-9)

**Epic Goal:** Implement relay node software, fallback routing, and achieve 95%+ connectivity across CGNAT/symmetric NAT scenarios.

**User Stories:**

#### Story 4.1: Relay Node Binary (Core Routing)
**As a** relay node operator
**I want** a standalone Go binary that routes encrypted traffic between clients
**So that** I can operate a relay node to support the ShadowMesh network

**Acceptance Criteria:**
- Relay node binary (`shadowmesh-relay`) accepts WebSocket connections from multiple clients
- Routing logic forwards encrypted frames based on destination peer ID
- Multi-threaded architecture handles 1,000+ concurrent client connections
- Configuration file specifies: listening port, stake wallet private key, geographic location
- Relay exposes health check endpoint: `/health` returns uptime, connection count, capacity
- Graceful shutdown drains active connections before exiting
- Unit tests validate routing logic, connection limits, configuration parsing

**Estimate:** 5 days

---

#### Story 4.2: Capacity Management
**As a** relay node
**I want** to reject new connections when capacity is reached
**So that** I maintain performance for existing connections

**Acceptance Criteria:**
- Configurable capacity limit: `--max-connections=1000` (default based on VPS specs)
- New connection requests rejected with HTTP 503 (Service Unavailable) when capacity reached
- Capacity reported in health check endpoint and Prometheus metrics
- Connection count decremented when clients disconnect (proper cleanup)
- Load balancing hint provided to clients: redirect to less-loaded relay
- Unit tests validate capacity enforcement, connection cleanup, HTTP 503 responses

**Estimate:** 2 days

---

#### Story 4.3: Client Relay Fallback Logic
**As a** ShadowMesh client
**I want** automatic fallback to relay routing when direct P2P fails
**So that** I can maintain connectivity behind CGNAT

**Acceptance Criteria:**
- Client detects P2P failure: no response to UDP hole punching within 500ms
- Client queries smart contract for active relay list (cached for 5 minutes)
- Client selects 3 relays from different operators (geographic diversity if available)
- Multi-hop routing established: Client → Relay1 → Relay2 → Relay3 → Peer
- Fallback triggers automatically without user intervention (logged as info event)
- Periodic retry of direct P2P every 10 minutes (may succeed if NAT state changes)
- Unit tests validate fallback trigger, relay selection, multi-hop establishment

**Estimate:** 5 days

---

#### Story 4.4: Multi-Hop Routing Protocol
**As a** ShadowMesh client
**I want** connections routed through minimum 3 relays from different operators
**So that** no single relay can analyze my traffic patterns

**Acceptance Criteria:**
- Routing protocol uses onion-like layered encryption (3 layers, one per hop)
- Each relay only knows previous hop and next hop (not full path)
- Client builds routing path: select 3 relays, construct nested encryption headers
- Relay nodes forward packets based on header instructions (no deep packet inspection)
- Operator diversity enforced: verify relay operators via on-chain metadata
- Performance: multi-hop latency <50ms overhead vs direct P2P (measured in lab)
- Unit tests validate path construction, layered encryption, relay forwarding logic

**Estimate:** 6 days

---

#### Story 4.5: Relay Node Installation & Deployment
**As a** relay node operator
**I want** simple installation scripts for Ubuntu/Debian VPS
**So that** I can deploy a relay node in <15 minutes

**Acceptance Criteria:**
- Installation script: `curl https://shadowmesh.network/install-relay.sh | sudo bash`
- Script installs: shadowmesh-relay binary, systemd service, logrotate configuration
- Operator prompted for: stake wallet private key, geographic location, listening port
- systemd service starts relay automatically on boot (enable --now)
- Operator documentation includes: VPS requirements, firewall config, monitoring setup
- Script tested on: Ubuntu 20.04, Ubuntu 22.04, Debian 11, Debian 12
- Post-install validation: script checks if relay is reachable from internet

**Estimate:** 3 days

---

#### Story 4.6: CGNAT Test Matrix
**As a** QA engineer
**I want** comprehensive CGNAT traversal testing across diverse NAT configurations
**So that** I can validate 95%+ connectivity success rate

**Acceptance Criteria:**
- Test matrix includes:
  - Symmetric NAT (most restrictive, requires relay)
  - Port-Restricted Cone NAT (requires relay or advanced hole punching)
  - Restricted Cone NAT (UDP hole punching should succeed)
  - Full Cone NAT (UDP hole punching should succeed)
  - Double NAT (NAT behind NAT, common in cellular/ISPs)
  - No NAT (direct P2P should succeed)
- Test environment: 20+ network configurations using lab routers or cloud NAT services
- Success criteria: connection established within 10 seconds, 95%+ success rate overall
- Latency validation: relay-routed connections <50ms overhead vs direct P2P
- Test results documented in spreadsheet with NAT type, success/failure, latency
- Failed scenarios investigated and documented as known issues (if unfixable in MVP)

**Estimate:** 7 days

---

#### Story 4.7: Relay Node Operator Dashboard
**As a** relay node operator
**I want** a Grafana dashboard showing my node's performance and earnings
**So that** I can monitor health and optimize configuration

**Acceptance Criteria:**
- Pre-configured Grafana dashboard for relay operators (separate from client dashboard)
- Dashboard panels:
  - Connected clients count (time-series graph)
  - Bandwidth usage (Mbps tx/rx)
  - Uptime percentage (last 24 hours, 7 days, 30 days)
  - Heartbeat transaction status (last successful heartbeat timestamp)
  - Stake rewards (future feature, placeholder in MVP)
  - Attestation status (future feature, placeholder in MVP)
- Dashboard provisioned automatically via Docker Compose for operators
- Operator documentation explains dashboard interpretation and troubleshooting

**Estimate:** 3 days

---

**Epic 4 Total Estimate:** 31 days (approximately 4-5 weeks; timeline shows 3 weeks with parallel work)

---

### Epic 5: Monitoring & Grafana Dashboard (Weeks 10-11)

**Epic Goal:** Implement Prometheus metrics, Grafana dashboards, and Docker Compose monitoring stack.

**User Stories:**

#### Story 5.1: Prometheus Metrics Endpoint
**As a** ShadowMesh client daemon
**I want** to expose metrics in Prometheus format on `:9090/metrics`
**So that** Prometheus can scrape and store my connection statistics

**Acceptance Criteria:**
- HTTP endpoint listens on configurable port (default: 9090)
- Endpoint serves Prometheus text exposition format (`# TYPE`, `# HELP`, metric lines)
- Metrics implementation uses `github.com/prometheus/client_golang`
- Endpoint returns metrics within 100ms (should not block on slow operations)
- Metrics include proper labels (peer_id, relay_node, nat_type, connection_state)
- Unit tests validate metric registration, exposition format, label values

**Estimate:** 2 days

---

#### Story 5.2: Comprehensive Metric Taxonomy
**As a** developer
**I want** well-organized metrics covering all system aspects
**So that** I can troubleshoot issues and monitor performance

**Acceptance Criteria:**
- **Connection Metrics:**
  - `shadowmesh_connection_status` (gauge: 0=disconnected, 1=connected)
  - `shadowmesh_connection_latency_ms` (gauge: current RTT to peer)
  - `shadowmesh_nat_traversal_type` (gauge: 0=none, 1=direct, 2=relay)
- **Network Metrics:**
  - `shadowmesh_throughput_bytes_total` (counter: cumulative bytes tx/rx with direction label)
  - `shadowmesh_packet_loss_ratio` (gauge: 0.0-1.0 representing percentage)
  - `shadowmesh_frame_encryption_rate_per_sec` (gauge: frames encrypted per second)
- **Cryptography Metrics:**
  - `shadowmesh_pqc_handshakes_total` (counter: successful and failed, with status label)
  - `shadowmesh_key_rotation_total` (counter: cumulative key rotations)
  - `shadowmesh_crypto_cpu_usage_percent` (gauge: CPU % spent in crypto operations)
- **Relay Metrics:**
  - `shadowmesh_relay_nodes_available` (gauge: count of active relays from smart contract)
  - `shadowmesh_relay_node_latency_ms` (gauge with relay_node label: latency to specific relay)
  - `shadowmesh_relay_hops_count` (gauge: number of hops in current connection path)
- **System Metrics:**
  - `shadowmesh_cpu_usage_percent` (gauge: overall daemon CPU usage)
  - `shadowmesh_memory_usage_bytes` (gauge: RSS memory of daemon process)
  - `shadowmesh_active_connections_count` (gauge: number of concurrent peer connections)
- All metrics documented in `docs/metrics.md` with descriptions and example PromQL queries

**Estimate:** 4 days

---

#### Story 5.3: Docker Compose Monitoring Stack
**As a** user
**I want** Grafana and Prometheus automatically deployed alongside the client
**So that** I can view my dashboard without manual configuration

**Acceptance Criteria:**
- `docker-compose.yml` defines 3 services:
  - `shadowmesh-daemon`: Client daemon (host network mode, privileged for TAP)
  - `prometheus`: Metrics storage (image: prom/prometheus:latest, port 9091)
  - `grafana`: Dashboard UI (image: grafana/grafana:latest, port 8080)
- Prometheus configuration file (`monitoring/prometheus.yml`) scrapes daemon every 15 seconds
- Prometheus retention configured to 7 days (`--storage.tsdb.retention.time=7d`)
- Grafana datasource provisioned automatically (points to Prometheus on port 9091)
- Grafana anonymous authentication enabled for localhost-only access (no login required)
- Installation script runs `docker-compose up -d` after client installation
- Documentation explains how to start/stop/restart monitoring stack

**Estimate:** 3 days

---

#### Story 5.4: Main User Dashboard (4-Row Layout)
**As a** user
**I want** an intuitive Grafana dashboard showing my connection health at a glance
**So that** I can quickly verify everything is working

**Acceptance Criteria:**
- Dashboard auto-imported from `monitoring/dashboards/main-user.json`
- **Row 1 - Connection Health:**
  - Single stat panel: Connection status (green "Connected" or red "Disconnected")
  - Single stat panel: NAT traversal type ("Direct P2P" or "Relay-Routed")
  - Single stat panel: Active peer count
  - Single stat panel: Available relay nodes
- **Row 2 - Network Performance:**
  - Time-series graph: Throughput (Mbps tx/rx, last 1 hour)
  - Time-series graph: Latency (ms, last 1 hour with anomaly detection band)
  - Gauge panel: Packet loss (%, thresholds: green <1%, yellow 1-5%, red >5%)
- **Row 3 - Security Metrics:**
  - Single stat panel: PQC handshakes (success count, last 24 hours)
  - Time-series graph: Key rotation timeline (shows rotation events)
  - Gauge panel: Crypto CPU usage (%, threshold: red >50%)
- **Row 4 - Peer Map:**
  - Geomap panel: Peers and relays plotted on world map (city-level markers)
  - Table panel: Relay node list with columns (node ID, location, latency, uptime)
- Dashboard uses dark theme, chronara.ai logo in header, auto-refresh every 5 seconds

**Estimate:** 5 days

---

#### Story 5.5: Relay Operator Dashboard
**As a** relay node operator
**I want** a dedicated dashboard showing operational metrics
**So that** I can optimize my node's performance and maximize rewards

**Acceptance Criteria:**
- Dashboard auto-imported from `monitoring/dashboards/relay-operator.json`
- Panels:
  - Connected clients count (time-series, last 24 hours)
  - Bandwidth usage (Mbps tx/rx, aggregated across all clients)
  - Uptime percentage (single stat with sparkline, last 7 days)
  - Last heartbeat transaction (single stat: timestamp and block number)
  - Stake rewards earned (placeholder panel for Beta phase)
  - Attestation status (placeholder panel for Beta phase)
  - Geographic distribution of clients (geomap)
- Dashboard accessible at `localhost:8080/d/relay-operator`
- Operator documentation explains how to interpret metrics and troubleshoot issues

**Estimate:** 3 days

---

#### Story 5.6: Developer/Debug Dashboard
**As a** developer
**I want** detailed technical metrics for troubleshooting
**So that** I can diagnose connection failures and performance bottlenecks

**Acceptance Criteria:**
- Dashboard auto-imported from `monitoring/dashboards/developer-debug.json`
- Panels:
  - Crypto operation latency breakdown (ML-KEM, ML-DSA, ChaCha20 timing histograms)
  - Smart contract query latency (time-series graph with error annotations)
  - Error logs (Loki integration if available, otherwise placeholder)
  - System resources (CPU, memory, disk I/O, network I/O)
  - WebSocket connection state transitions (state machine visualization)
  - Frame encryption pipeline throughput (frames/sec at each stage)
- Dashboard includes annotations for key events (key rotation, relay fallback, reconnections)
- Dashboard accessible at `localhost:8080/d/developer-debug`

**Estimate:** 4 days

---

#### Story 5.7: Resource Optimization
**As a** user with limited system resources
**I want** the monitoring stack to use <900 MB total memory
**So that** it doesn't significantly impact my system performance

**Acceptance Criteria:**
- Prometheus memory target: 100-150 MB (achieved via shorter retention and scrape interval tuning)
- Grafana memory target: 150-200 MB (achieved via disabling unused plugins and features)
- Client daemon memory target: 100-150 MB (validated via benchmarks with 50 active peers)
- Docker overhead: ~450-500 MB (base container runtime)
- Total measured memory usage <900 MB under normal operation (measured via `docker stats`)
- Configuration options documented for users to trade off retention vs memory (e.g., reduce retention to 3 days)
- Memory usage validated on systems with 2GB RAM total (leave 1GB for OS and other apps)

**Estimate:** 2 days

---

#### Story 5.8: Installation Script Integration
**As a** user
**I want** the monitoring stack installed automatically with the client
**So that** I don't need to manually configure Docker Compose

**Acceptance Criteria:**
- Client installation script detects Docker availability (checks `docker --version`)
- If Docker missing, script prompts: "Install Docker? [Y/n]" and runs Docker install script
- Script copies `docker-compose.yml` and `monitoring/` config files to `/opt/shadowmesh/`
- Script runs `docker-compose up -d` and validates containers are healthy
- Script prints: "Dashboard available at http://localhost:8080" upon successful setup
- Script creates systemd service for client daemon configured to communicate with Prometheus
- Documentation includes troubleshooting section for common Docker issues

**Estimate:** 2 days

---

**Epic 5 Total Estimate:** 25 days (approximately 3-4 weeks; timeline shows 2 weeks with parallel work)

---

### Epic 6: Public Map, Documentation & Launch (Week 12)

**Epic Goal:** Build public network map, finalize documentation, execute beta launch to crypto/privacy communities.

**User Stories:**

#### Story 6.1: Public Network Map Website
**As a** prospective user
**I want** a public website showing all registered relay nodes
**So that** I can verify network coverage before installing ShadowMesh

**Acceptance Criteria:**
- React app deployed at `map.shadowmesh.network` (hosted on Vercel or Netlify)
- App queries chronara.eth smart contract via Infura/Alchemy (read-only)
- Leaflet.js interactive map displays relay nodes as markers (city-level location, no precise coordinates)
- Clicking marker shows popup: node ID, location, uptime %, last heartbeat
- Aggregate stats displayed in header: total nodes, countries covered, network health
- Privacy-preserving: no user data, connection graphs, or traffic patterns shown
- Responsive design: works on desktop (primary) and mobile (basic view)
- App auto-refreshes every 60 seconds to detect new node registrations

**Estimate:** 5 days

---

#### Story 6.2: Real-Time Map Updates (Blockchain Events)
**As a** website visitor
**I want** the map to update within 60 seconds of new relay node registrations
**So that** I see current network state

**Acceptance Criteria:**
- App subscribes to smart contract events via WebSocket (Infura WSS endpoint)
- Events monitored: `NodeRegistered`, `NodeDeregistered`, `HeartbeatUpdated`
- On `NodeRegistered`: new marker added to map with animation
- On `NodeDeregistered`: marker removed from map
- On `HeartbeatUpdated`: node uptime % recalculated and updated in popup
- Event subscription resilient to connection failures (auto-reconnect with exponential backoff)
- Unit tests validate event parsing, map updates, subscription reconnection logic

**Estimate:** 3 days

---

#### Story 6.3: User Documentation (Installation & Troubleshooting)
**As a** new user
**I want** clear installation guides and troubleshooting steps
**So that** I can set up ShadowMesh successfully

**Acceptance Criteria:**
- Documentation published at `docs.shadowmesh.network` (Docusaurus or similar)
- Documentation sections:
  - **Quick Start:** Installation in <10 minutes (Ubuntu example)
  - **Installation Guide:** Detailed instructions for all supported Linux distros
  - **CLI Reference:** Complete command list with examples (`shadowmesh --help` output)
  - **Troubleshooting:** Common errors and solutions (TAP device creation failures, firewall issues, Docker problems)
  - **FAQ:** Answers to frequently asked questions (What is PQC? Why blockchain? What is CGNAT?)
- Documentation includes screenshots of Grafana dashboard
- Documentation includes links to Discord for community support
- Documentation written in Markdown, versioned in Git

**Estimate:** 4 days

---

#### Story 6.4: Relay Node Operator Guide
**As a** relay node operator
**I want** comprehensive documentation on running a relay node
**So that** I can deploy a node correctly and earn rewards (future)

**Acceptance Criteria:**
- Operator guide sections:
  - **VPS Requirements:** CPU, RAM, bandwidth, recommended providers
  - **Installation:** Step-by-step relay node setup
  - **Staking:** How to stake ETH for node registration (MetaMask walkthrough)
  - **Monitoring:** How to access operator Grafana dashboard
  - **Maintenance:** Heartbeat updates, software upgrades, troubleshooting
  - **Economics:** Gas costs, future reward model (placeholder for Beta)
- Guide includes security best practices: firewall config, SSH hardening, wallet security
- Guide tested by 3 external operators during beta (feedback incorporated)

**Estimate:** 3 days

---

#### Story 6.5: GitHub Repository Cleanup & Branding
**As a** open-source contributor
**I want** a well-organized GitHub repository with clear contribution guidelines
**So that** I can contribute to ShadowMesh effectively

**Acceptance Criteria:**
- `README.md` includes:
  - Project description and goals
  - chronara.ai logo and branding
  - Installation quick start (link to full docs)
  - Link to public network map
  - Community links (Discord, Twitter)
- `CONTRIBUTING.md` explains:
  - How to set up development environment
  - Code style guidelines (gofmt, linting)
  - How to submit PRs (branching strategy, commit message format)
  - How to report bugs (issue template)
- `CODE_OF_CONDUCT.md` adopts Contributor Covenant
- `LICENSE` file: MIT or Apache 2.0 (decision made based on legal review)
- GitHub repository topics/tags: `post-quantum`, `vpn`, `ethereum`, `privacy`, `crypto`

**Estimate:** 2 days

---

#### Story 6.6: Beta Launch Strategy
**As a** product manager
**I want** a targeted launch campaign to acquire 100-500 beta users
**So that** we can validate product-market fit and gather feedback

**Acceptance Criteria:**
- Launch strategy document includes:
  - **Target Communities:** r/privacy, r/cryptocurrency, r/CryptoTechnology, Hacker News, Product Hunt
  - **Messaging:** Emphasize PQC protection, decentralized trust, 5+ year first-mover advantage
  - **Timeline:** Staggered rollout (beta announcement → Reddit posts → Product Hunt launch)
  - **Success Metrics:** 100-500 signups, 80%+ connection success rate, <5% churn in first week
- Discord server created for community support (channels: #general, #support, #development)
- Twitter/X account created (@ShadowMeshNet or similar) for announcements
- Product Hunt listing prepared with screenshots, demo video, launch date scheduled
- Reddit posts drafted (personalized for each subreddit, not copy-paste spam)
- Beta feedback form created (Google Forms or Typeform) to collect user experience data

**Estimate:** 3 days

---

#### Story 6.7: Monitoring & Analytics Setup
**As a** product manager
**I want** to track user acquisition funnel and connection success rates
**So that** I can identify bottlenecks and optimize onboarding

**Acceptance Criteria:**
- Analytics events tracked:
  - Installation started (user downloads installer)
  - Installation completed (daemon running)
  - First connection attempt (user runs `shadowmesh connect`)
  - First connection success (handshake completed, encrypted tunnel established)
  - Dashboard first view (user opens localhost:8080)
- Connection success rate dashboard (aggregated across all beta users)
- Error log aggregation (identify most common error messages)
- Churn analysis: users who install but never connect, users who connect once and abandon
- Privacy-preserving analytics: no PII collected, anonymous user IDs, opt-out option
- Analytics implementation: self-hosted Plausible or similar (not Google Analytics)

**Estimate:** 3 days

---

#### Story 6.8: Beta Launch Execution & Monitoring
**As a** product manager
**I want** to execute the launch and monitor results in real-time
**So that** I can respond quickly to issues and optimize conversion

**Acceptance Criteria:**
- Beta launch date announced 1 week in advance (builds anticipation)
- Reddit posts published across target subreddits (spaced 24 hours apart to avoid spam filters)
- Product Hunt listing goes live with demo video and screenshots
- Discord server seeded with initial community members (invite-only alpha testers)
- Launch day monitoring: track signups, connection success rate, error spikes in real-time
- Rapid response team ready: fix critical bugs within 4 hours, respond to Discord questions within 1 hour
- Post-launch retrospective after 1 week: analyze metrics, gather feedback, plan Beta phase improvements

**Estimate:** 2 days (execution), ongoing monitoring throughout launch week

---

**Epic 6 Total Estimate:** 25 days (approximately 3-4 weeks; timeline shows 1 week assuming significant prep in earlier epics)

---

## PM Checklist Validation Results

### Executive Summary

**Overall PRD Completeness:** 92%

**MVP Scope Appropriateness:** Just Right (12-week timeline with achievable core functionality)

**Readiness for Architecture Phase:** **READY** - PRD is comprehensive, properly structured, and provides sufficient detail for architectural design.

**Critical Observations:**
- Excellent coverage of functional/non-functional requirements with 44 FR and 31 NFR
- Comprehensive epic breakdown with 40+ user stories and detailed acceptance criteria
- Strong technical guidance with explicit technology stack decisions and architectural patterns
- Well-defined success metrics and validation approach (100-500 beta users, 80%+ connection success)
- Minor gaps in stakeholder communication plan and competitive differentiation detail

---

### Category Analysis

| Category                         | Status  | Critical Issues                                                    |
| -------------------------------- | ------- | ------------------------------------------------------------------ |
| 1. Problem Definition & Context  | PASS    | None - problem statement clear, research validated                 |
| 2. MVP Scope Definition          | PASS    | None - scope well-bounded with future enhancements identified      |
| 3. User Experience Requirements  | PASS    | None - user flows implicit in stories, accessibility documented    |
| 4. Functional Requirements       | PASS    | None - 44 FR with clear acceptance criteria                        |
| 5. Non-Functional Requirements   | PASS    | None - comprehensive NFR across performance, security, reliability |
| 6. Epic & Story Structure        | PASS    | None - 6 epics, 40+ stories, properly sequenced                    |
| 7. Technical Guidance            | PASS    | None - comprehensive tech stack, architecture decisions documented |
| 8. Cross-Functional Requirements | PARTIAL | Minor: Data schema details deferred to architecture phase          |
| 9. Clarity & Communication       | PARTIAL | Minor: Stakeholder approval process not explicitly defined         |

**Legend:**
- **PASS:** 90%+ complete, no blockers
- **PARTIAL:** 60-89% complete, minor gaps that don't block progress
- **FAIL:** <60% complete, significant gaps requiring resolution

---

### Detailed Checklist Results

#### 1. PROBLEM DEFINITION & CONTEXT (PASS - 95%)

**Strengths:**
- ✅ Clear problem statement: Current VPNs/private networks vulnerable to quantum attacks, relay node trust issues, DPI detection
- ✅ Specific target users: Enterprise security teams ($50-200/user/month), crypto-native users ($30-50/month), privacy-conscious consumers ($10-20/month)
- ✅ Quantified success metrics: 100-500 beta users, 80%+ connection success rate, 1+ Gbps throughput, 95%+ CGNAT traversal
- ✅ Competitive analysis included: WireGuard, Tailscale, ZeroTier - confirmed 5+ year first-mover advantage
- ✅ Market context provided: Quantum threat timeline (2030-2035), NIST PQC standardization, censorship evasion needs

**Minor Gaps:**
- Could strengthen differentiation from Tor/I2P (privacy networks using onion routing)
- User research insights based on assumptions + competitive analysis, not direct user interviews (acceptable for MVP)

**Score:** 95% (19/20 checklist items)

---

#### 2. MVP SCOPE DEFINITION (PASS - 93%)

**Strengths:**
- ✅ Essential features clearly distinguished: PQC crypto, CGNAT traversal, blockchain relay discovery, Grafana monitoring
- ✅ Scope boundaries explicit: No mobile apps, no atomic clock sync, no slashing (deferred to Beta)
- ✅ MVP validation approach defined: Beta launch with connection success rate and user feedback mechanisms
- ✅ Timeline realistic: 12 weeks with 6 sequential epics, parallel work assumed
- ✅ Learning goals articulated: Validate PQC performance, CGNAT traversal success rate, blockchain coordination viability

**Minor Gaps:**
- Future enhancements mentioned but not exhaustively listed (acceptable - implicit in Beta/Production phases)

**Score:** 93% (14/15 checklist items)

---

#### 3. USER EXPERIENCE REQUIREMENTS (PASS - 90%)

**Strengths:**
- ✅ Primary user flows documented: Installation → Connection → Monitoring (via Grafana) flow implicit in stories
- ✅ Platform compatibility specified: Linux all major distros, Docker, browser requirements for Grafana
- ✅ Performance expectations clear: <10 min installation, <5ms latency overhead, 1+ Gbps throughput
- ✅ Error handling outlined: NFR21 actionable error messages, troubleshooting documentation in Epic 6
- ✅ Accessibility explicitly scoped out for MVP (revisit in Beta)

**Minor Gaps:**
- User journey diagrams not included (visual flow charts) - acceptable for text-based PRD
- Edge cases identified in stories but not consolidated into edge case matrix

**Score:** 90% (13/15 checklist items) - pending UX Expert involvement

---

#### 4. FUNCTIONAL REQUIREMENTS (PASS - 98%)

**Strengths:**
- ✅ 44 Functional Requirements documented with clear categories
- ✅ Requirements focus on WHAT not HOW (e.g., "shall implement ML-KEM-1024" not "shall use struct KEM1024...")
- ✅ All requirements testable with specific acceptance criteria
- ✅ Dependencies explicit (e.g., Epic 4 depends on Epic 2 networking, Epic 3 blockchain)
- ✅ Consistent terminology (PQC, relay, CGNAT, WSS, TAP devices)
- ✅ Complex features decomposed (e.g., Grafana broken into FR28-FR33 with specific metrics)

**Minor Gaps:**
- None identified - exceptional functional requirements coverage

**Score:** 98% (29/30 checklist items)

---

#### 5. NON-FUNCTIONAL REQUIREMENTS (PASS - 97%)

**Strengths:**
- ✅ Performance requirements quantified: 1+ Gbps throughput, <5ms latency, 1,000 concurrent connections per relay
- ✅ Security comprehensive: NFR6-10 cover PQC resistance, no custom crypto, PFS, MITM protection, smart contract audit
- ✅ Reliability defined: NFR11 99.9% uptime, NFR12 30-second auto-reconnect, NFR13 graceful contract query failures
- ✅ Scalability targets: NFR16 200 relay nodes, NFR17 500 peers in Grafana, NFR18 3-second public map load
- ✅ Compliance: NFR23 open-source (MIT/Apache 2.0), NFR24 audit logging, NFR25 Etherscan verification
- ✅ Technical constraints documented: NFR27 Linux kernel 3.10+, NFR29 x86_64/ARM64, NFR30 Ethereum/testnets

**Minor Gaps:**
- None identified - comprehensive NFR coverage

**Score:** 97% (29/30 checklist items)

---

#### 6. EPIC & STORY STRUCTURE (PASS - 95%)

**Strengths:**
- ✅ 6 epics represent cohesive functionality units (Foundation, Networking, Blockchain, Relay, Monitoring, Launch)
- ✅ Epics sequenced logically: Foundation → Core → Integrations → Operations
- ✅ Epic goals clearly articulated with measurable milestones
- ✅ 40+ user stories with As a/I want/So that format
- ✅ Acceptance criteria testable and specific
- ✅ First epic (Foundation) includes all setup: monorepo, crypto primitives, benchmarks
- ✅ Story estimates provided (2-7 days per story, realistic for complexity)

**Minor Gaps:**
- Some stories might be large (e.g., Story 4.6 CGNAT Test Matrix: 7 days) - could split further during implementation
- Local testability mentioned in acceptance criteria but not explicitly called out as requirement (acceptable implicit coverage)

**Score:** 95% (28/30 checklist items)

---

#### 7. TECHNICAL GUIDANCE (PASS - 94%)

**Strengths:**
- ✅ Initial architecture direction provided: Monorepo, monolithic Go binaries, Docker Compose monitoring
- ✅ Technical constraints explicit: Linux-only, TAP devices, systemd, Docker
- ✅ Integration points identified: Ethereum smart contract, Infura/Alchemy RPC, ENS, Prometheus/Grafana
- ✅ Performance considerations highlighted: <5ms latency overhead, 1+ Gbps throughput, <900 MB memory
- ✅ Security requirements articulated: No custom crypto, Cloudflare CIRCL, smart contract audit
- ✅ Trade-offs documented: Monolithic vs microservices (chose monolithic for performance), Grafana vs custom HTML dashboard
- ✅ Technical risks flagged: CGNAT traversal highest risk (95% target challenging)

**Minor Gaps:**
- Known areas of complexity could be more explicitly flagged for architectural deep-dive (e.g., multi-hop onion routing protocol)

**Score:** 94% (16/17 checklist items)

---

#### 8. CROSS-FUNCTIONAL REQUIREMENTS (PARTIAL - 83%)

**Strengths:**
- ✅ Data entities identified: users, relay nodes, peer relationships, connection history
- ✅ Database specified: PostgreSQL 14+ for user data, Ethereum for relay registry
- ✅ External integrations documented: Ethereum mainnet, Infura/Alchemy, ENS, Prometheus/Grafana
- ✅ API requirements outlined: WebSocket protocol, Prometheus metrics endpoint, relay health check
- ✅ Deployment frequency: Implied continuous deployment via GitHub Actions
- ✅ Monitoring needs: Prometheus metrics, Grafana dashboards, error log aggregation

**Gaps:**
- Data schema details (PostgreSQL table structures) deferred to architecture phase - acceptable, architect should design
- Data migration strategy not documented - not applicable for MVP (no existing data)
- API authentication for integrations partially specified (needs architect input for relay-to-relay auth)

**Score:** 83% (15/18 checklist items) - minor gaps acceptable, architect will address

---

#### 9. CLARITY & COMMUNICATION (PARTIAL - 87%)

**Strengths:**
- ✅ Clear, consistent language throughout PRD
- ✅ Well-structured sections: Goals, Requirements, UI, Tech Assumptions, Epics, Stories
- ✅ Technical terms defined: ML-KEM, ML-DSA, CGNAT, TAP devices, WSS
- ✅ Versioning included: PRD v0.1, change log present
- ✅ Key stakeholders implicit: PM, Architect, UX Expert (referenced in Next Steps)

**Gaps:**
- Diagrams/visuals not included (architecture diagrams pending UX Expert and Architect)
- Stakeholder approval process not explicitly defined (who approves PRD? PM? Product Owner?)
- Communication plan for updates not documented (how often reviewed? who notified of changes?)

**Score:** 87% (13/15 checklist items) - acceptable for initial draft, refine during handoff

---

### Top Issues by Priority

**BLOCKERS:** None

**HIGH PRIORITY:**
1. **Data Schema Design** - Architect should design PostgreSQL schema for users, relay nodes, peer relationships (deferred appropriately)
2. **API Authentication Details** - Specify relay-to-relay authentication mechanism for multi-hop routing (architect input needed)

**MEDIUM PRIORITY:**
3. **User Journey Diagrams** - UX Expert should create visual flow charts for installation, connection, monitoring workflows
4. **Stakeholder Approval Process** - Define who approves PRD and how changes are communicated

**LOW PRIORITY:**
5. **Architecture Diagrams** - System architecture diagram, data flow diagram (deferred to architect)
6. **Competitive Analysis Expansion** - Add differentiation from Tor/I2P privacy networks (not critical for MVP)

---

### MVP Scope Assessment

**Features That Could Be Cut (If Timeline Slips):**
1. **Public Network Map (Epic 6)** - Nice-to-have for public transparency, not essential for beta functionality
2. **Developer/Debug Dashboard (Story 5.6)** - Main user dashboard sufficient, debug dashboard can be added post-launch
3. **Multi-Hop Routing (Story 4.4)** - Could simplify to single-hop relay for MVP, add multi-hop in Beta for stronger privacy

**Missing Features That Are Essential:**
- None identified - MVP scope is comprehensive for validation goals

**Complexity Concerns:**
1. **CGNAT Traversal (Epic 4)** - Highest technical risk; 95% success rate target is ambitious
2. **Multi-Hop Onion Routing (Story 4.4)** - Complex protocol design, 6-day estimate may be conservative
3. **Smart Contract Gas Optimization (Story 3.6)** - Ethereum gas volatility could spike costs above $10 target

**Timeline Realism:**
- **12-week timeline:** Ambitious but achievable with 2-3 developers working in parallel
- **Epic estimates:** Epics 2, 3, 4 show 20-30 day estimates compressed into 2-week timeline blocks → assumes parallel work
- **Recommendation:** Plan for 14-16 weeks (add 2-4 week buffer for CGNAT testing and debugging)

---

### Technical Readiness

**Clarity of Technical Constraints:**
- ✅ Excellent - Technology stack fully specified (Go 1.21+, Solidity 0.8.20+, PostgreSQL 14+, Docker Compose)
- ✅ Platform constraints clear (Linux, systemd, TAP/TUN, Docker)
- ✅ Performance targets quantified (1+ Gbps, <5ms latency, <900 MB memory)

**Identified Technical Risks:**
1. **CGNAT Traversal** - Highest risk, 95% success rate target may not be achievable without extensive testing
2. **PQC Performance** - Cloudflare CIRCL benchmarks promising (22.7μs), but real-world multi-peer scenarios need validation
3. **Ethereum Gas Costs** - Volatile gas prices could exceed $10 registration target during network congestion
4. **TAP Device Permissions** - Requires CAP_NET_ADMIN or root, may complicate installation on restricted systems

**Areas Needing Architect Investigation:**
1. **Database Schema Design** - PostgreSQL tables for users, peers, relay nodes, connection history
2. **Multi-Hop Routing Protocol** - Layered encryption headers, relay forwarding logic, operator diversity enforcement
3. **Prometheus Metrics Architecture** - Efficient metric collection without blocking network performance
4. **WebSocket Scaling** - Handling 1,000+ concurrent connections per relay node on commodity VPS hardware
5. **Smart Contract Event Indexing** - Off-chain indexing strategy for public map real-time updates

---

### Recommendations

**Immediate Actions (Before Architect Handoff):**
1. ✅ **No blockers** - PRD is ready for architect
2. **Optional Enhancement:** Add visual user journey diagram (can be done by UX Expert in parallel)
3. **Optional Enhancement:** Define stakeholder approval process for PRD changes

**For Architect:**
1. Design PostgreSQL database schema with tables, relationships, indexes
2. Specify multi-hop onion routing protocol in detail (packet format, encryption layers)
3. Design relay-to-relay authentication mechanism (mutual TLS? blockchain-based identity?)
4. Create system architecture diagram (client, relay, blockchain, monitoring components)
5. Investigate Prometheus metrics collection performance impact (<1% CPU overhead target)

**For UX Expert:**
1. Create user journey diagrams for:
   - Installation and first-time setup flow
   - Connection establishment flow (direct P2P vs relay fallback)
   - Dashboard monitoring workflow
2. Design Grafana dashboard layouts with wireframes/mockups
3. Review CLI help text and error messages for user-friendliness

**For Development Team:**
1. Set up development environment and monorepo structure (Epic 1, Story 1.1)
2. Validate Cloudflare CIRCL library compatibility with target Go version
3. Test TAP device creation on all supported Linux distributions
4. Benchmark Docker Compose memory overhead on 2GB RAM system

**For Beta Launch:**
1. Prepare Discord server structure (#general, #support, #development)
2. Draft Reddit posts for r/privacy, r/cryptocurrency, r/CryptoTechnology
3. Create Product Hunt listing with screenshots and demo video
4. Set up analytics dashboard (Plausible or similar for privacy-preserving tracking)

---

### Final Decision

**✅ READY FOR ARCHITECT**

The PRD and epic definitions are comprehensive, properly structured, and provide sufficient detail for architectural design. The document demonstrates:

1. **Strong Problem-Solution Fit** - Clear problem statement backed by research, well-defined target users, measurable success criteria
2. **Appropriate MVP Scope** - 12-week timeline with achievable core functionality, explicit scope boundaries, future enhancements identified
3. **Comprehensive Requirements** - 44 functional requirements, 31 non-functional requirements, all testable and specific
4. **Well-Structured Epics** - 6 sequential epics with 40+ user stories, detailed acceptance criteria, realistic estimates
5. **Clear Technical Guidance** - Explicit technology stack, architectural patterns, trade-off rationale, identified risks

**Minor gaps** in data schema details, API authentication, and stakeholder communication are intentionally deferred to architecture phase or are non-blocking process improvements.

**Next Steps:**
1. **Handoff to Architect** - Proceed with architecture document creation
2. **Parallel UX Work** - Engage UX Expert for journey diagrams and dashboard mockups
3. **Team Readiness** - Developers can begin Epic 1 (Foundation) setup in parallel with architecture work

**Confidence Level:** High (9/10) - PRD provides solid foundation for successful MVP development.

---

## Next Steps

This section provides detailed prompts for the UX Expert and Architect to ensure seamless handoff from the Product Manager.

---

### For UX Expert

**Context:** ShadowMesh is a post-quantum encrypted private network targeting crypto-native users, enterprise security teams, and privacy-conscious consumers. The MVP focuses on Linux CLI client with Grafana monitoring dashboards. The product differentiates through bleeding-edge PQC cryptography (ML-KEM-1024 + ML-DSA-87), blockchain-coordinated relay discovery, and professional-grade monitoring.

**Your Primary Objectives:**

1. **Create User Journey Diagrams** illustrating the complete user experience from discovery to daily usage
2. **Design Grafana Dashboard Mockups** for the three pre-configured dashboards (Main User, Relay Operator, Developer/Debug)
3. **Refine CLI UX** by reviewing help text, error messages, and command structure for user-friendliness
4. **Validate Accessibility** approach and recommend enhancements (currently scoped out for MVP, but flag critical accessibility concerns)

---

**Detailed UX Expert Prompt:**

You are the UX Expert for ShadowMesh, a post-quantum encrypted private network. The Product Manager has completed the PRD with comprehensive functional requirements, epics, and user stories. Your task is to translate these requirements into visual user journeys and dashboard designs.

**1. User Journey Mapping**

Create detailed user journey diagrams for three critical workflows:

**A. Installation & First-Time Setup Flow**
- **Entry Point:** User discovers ShadowMesh via Reddit/Product Hunt, visits GitHub repo or docs site
- **Key Steps:**
  - User downloads installation script or .deb/.rpm package
  - Installation script detects Docker, prompts for install if missing
  - Script installs shadowmesh client daemon, systemd service, Docker Compose stack
  - User prompted for passphrase to encrypt keystore
  - Daemon generates hybrid keypair (ML-KEM + X25519, ML-DSA + Ed25519)
  - Docker Compose starts Prometheus + Grafana containers
  - Installation script prints: "Dashboard available at http://localhost:8080"
  - User opens browser, sees Grafana dashboard (anonymous auth, no login)
- **Success Criterion:** User completes installation in <10 minutes without prior blockchain/VPN knowledge
- **Pain Points to Address:**
  - Docker installation may fail on older Linux distros (provide troubleshooting)
  - TAP device creation requires sudo (explain why clearly)
  - Passphrase recovery: no "forgot password" mechanism (emphasize backup importance)
- **Diagram Format:** Swimlane diagram with user actions, system responses, decision points, error states

**B. Connection Establishment Flow (Direct P2P vs Relay Fallback)**
- **Entry Point:** User runs `shadowmesh connect <peer-id>` or clicks "Connect" in Grafana (future)
- **Key Steps:**
  - Daemon initiates PQC handshake with peer
  - NAT type detection: queries STUN-like server, determines Full Cone/Symmetric/etc.
  - **Path A (Direct P2P):**
    - UDP hole punching succeeds within 500ms
    - Encrypted tunnel established, Grafana shows "Connected - Direct P2P" (green status)
  - **Path B (Relay Fallback):**
    - UDP hole punching times out (CGNAT/Symmetric NAT detected)
    - Daemon queries chronara.eth smart contract for active relay list
    - Selects 3 relays from different operators (geographic diversity)
    - Establishes multi-hop route: Client → Relay1 → Relay2 → Relay3 → Peer
    - Grafana shows "Connected - Relay-Routed" (yellow status)
  - **Error Path:**
    - All connection attempts fail (no relays available, peer offline, smart contract unreachable)
    - Grafana shows "Disconnected" (red status)
    - Error message displayed: "Unable to connect to peer. Retry in 60 seconds."
- **Success Criterion:** User establishes connection within 10 seconds, understands why relay fallback occurred
- **Pain Points to Address:**
  - User may not understand "CGNAT" or "Symmetric NAT" - use plain language ("Your network requires relay routing")
  - Relay routing adds latency - set expectations ("Connection may be slower, but still encrypted")
  - Blockchain query may fail if Infura down - cache last-known relay list
- **Diagram Format:** Flowchart with decision branches, timing annotations, error handling paths

**C. Dashboard Monitoring Workflow**
- **Entry Point:** User opens http://localhost:8080 in browser after connection established
- **Key Steps:**
  - Grafana loads Main User Dashboard (default view)
  - User sees at-a-glance status:
    - **Row 1:** Connection status (green "Connected"), NAT type ("Direct P2P"), active peers (1), relay nodes available (12)
    - **Row 2:** Throughput graph (500 Mbps tx, 480 Mbps rx), latency graph (12ms average), packet loss gauge (0.3%)
    - **Row 3:** PQC handshakes (3 successful in last 24h), key rotation timeline (last rotation 2 min ago), crypto CPU usage (8%)
    - **Row 4:** World map showing peer location (San Francisco), relay node table (12 rows with latency)
  - User hovers over throughput graph → tooltip shows exact values at timestamp
  - User clicks on relay node in table → drills down to relay-specific latency graph (future feature)
  - Dashboard auto-refreshes every 5 seconds (configurable)
- **Success Criterion:** User glances at dashboard and immediately knows "Everything is working" or "Something is wrong"
- **Pain Points to Address:**
  - Information overload: 4 rows × 10+ panels = too much for quick glance? → Use color coding heavily (green/yellow/red)
  - Crypto-specific metrics (PQC handshakes, key rotation) may confuse non-technical users → Add tooltips explaining "Why this matters"
  - World map may not load if geolocation data unavailable → Gracefully degrade to list view
- **Diagram Format:** Annotated screenshot wireframe showing information hierarchy, interaction points, color coding

**2. Grafana Dashboard Design**

Create high-fidelity wireframes (Figma, Sketch, or similar) for three Grafana dashboards:

**A. Main User Dashboard**
- **Layout:** 4-row grid, dark theme, chronara.ai logo in header
- **Row 1 - Connection Health (4 panels):**
  - Panel 1: Connection Status (single stat, large font, green/red)
  - Panel 2: NAT Traversal Type (single stat, icon + text: "Direct P2P" or "Relay-Routed")
  - Panel 3: Active Peer Count (single stat with sparkline trend)
  - Panel 4: Relay Nodes Available (single stat, shows count with last update time)
- **Row 2 - Network Performance (3 panels):**
  - Panel 5: Throughput (time-series graph, dual-axis: tx in blue, rx in green, last 1 hour)
  - Panel 6: Latency (time-series graph with anomaly detection band, last 1 hour)
  - Panel 7: Packet Loss (gauge, 0-10% scale, thresholds: green <1%, yellow 1-5%, red >5%)
- **Row 3 - Security Metrics (3 panels):**
  - Panel 8: PQC Handshakes (single stat, success count with success/failure breakdown)
  - Panel 9: Key Rotation Timeline (time-series event markers showing rotation times)
  - Panel 10: Crypto CPU Usage (gauge, 0-100%, threshold: red >50%)
- **Row 4 - Peer Map (2 panels):**
  - Panel 11: Geographic Map (Geomap panel with peer markers, city-level precision)
  - Panel 12: Relay Node Table (columns: Node ID, Location, Latency, Uptime %, sortable)
- **Interactivity:**
  - Hovering over graphs shows tooltips with exact values
  - Clicking time-series allows zooming (1h, 6h, 24h, 7d)
  - Relay table rows clickable for drill-down (future feature, placeholder in MVP)
- **Accessibility Notes:**
  - Color-blind friendly: Use patterns in addition to colors (green = checkmark, red = X)
  - Font size minimum 12pt for readability at 1920x1080
  - High contrast text on dark background (WCAG AA compliance)

**B. Relay Operator Dashboard**
- **Purpose:** Enable relay node operators to monitor their infrastructure and optimize performance
- **Layout:** 3-row grid focusing on operational metrics
- **Key Panels:**
  - Connected clients count (time-series, last 24 hours)
  - Bandwidth usage (Mbps tx/rx, aggregated across all clients)
  - Uptime percentage (single stat with sparkline, last 7 days rolling average)
  - Heartbeat transaction status (single stat: "Last heartbeat: 2 hours ago, Block #18234567")
  - Stake rewards (placeholder panel with "Coming in Beta" message)
  - Geographic distribution of clients (geomap showing client locations)
- **Design Consistency:** Reuse Main User Dashboard visual language (same colors, fonts, panel styles)

**C. Developer/Debug Dashboard**
- **Purpose:** Provide detailed technical metrics for troubleshooting connection failures and performance bottlenecks
- **Layout:** Dense 5-row grid with advanced metrics
- **Key Panels:**
  - Crypto operation latency breakdown (histogram: ML-KEM encapsulation time, ML-DSA signing time, ChaCha20 encryption time)
  - Smart contract query latency (time-series with error annotations: red markers for failed queries)
  - Error logs (table panel showing timestamp, error type, message, peer ID)
  - System resources (4 gauges: CPU %, memory MB, disk I/O KB/s, network I/O MB/s)
  - WebSocket connection state transitions (state machine diagram: Connecting → Connected → Key Rotation → Disconnected)
  - Frame encryption pipeline throughput (bar chart showing frames/sec at each stage: Capture, Encrypt, Transmit, Receive, Decrypt, Inject)
- **Audience:** Power users, developers, support engineers - dense information is acceptable

**3. CLI UX Review**

Review and refine the command-line interface design:

**A. Help Text Audit**
- Review `shadowmesh --help` output for clarity
- Ensure each command has concise description + example
- Flag technical jargon that needs plain-language explanations (e.g., "TAP device" → "virtual network adapter")

**B. Error Message Refinement**
- Translate cryptic errors into actionable messages:
  - **Bad:** "ML-KEM decapsulation failed: invalid ciphertext"
  - **Good:** "Connection failed: Unable to establish secure tunnel. Check that peer is online and try again."
- Provide recovery steps in error messages:
  - **Bad:** "CGNAT detected"
  - **Good:** "Your network requires relay routing (CGNAT detected). Connection will automatically use relay nodes."

**C. Command Structure Validation**
- Confirm command naming follows Unix conventions (lowercase, hyphen-separated)
- Ensure consistency: `shadowmesh connect`, `shadowmesh disconnect`, `shadowmesh status` (not `shadowmesh stop`, use `disconnect`)

**4. Accessibility Assessment**

**Current MVP Scope:** Accessibility intentionally scoped out (target audience: crypto-native users, enterprise IT, command-line users)

**Your Task:** Flag critical accessibility concerns that should be reconsidered:
- **Screen Reader Compatibility:** Grafana's default UI is not optimized for screen readers - is this acceptable risk?
- **Keyboard Navigation:** Can users navigate Grafana dashboard without mouse? (Answer: Yes, Grafana supports keyboard nav)
- **Color Blindness:** Ensure green/red status indicators have alternative visual cues (checkmarks, X symbols, patterns)
- **Font Scaling:** Test dashboard readability at browser zoom 150%, 200% (important for vision-impaired users)

**Deliverables:**

1. **User Journey Diagrams (3):** Installation, Connection Establishment, Dashboard Monitoring (PDF or Figma file)
2. **Grafana Dashboard Mockups (3):** Main User, Relay Operator, Developer/Debug (Figma with export to JSON for Grafana provisioning)
3. **CLI UX Audit Report:** Markdown document listing error message improvements, help text clarifications, command structure recommendations
4. **Accessibility Flag Report:** 1-page summary of critical accessibility concerns for PM review

**Timeline:** 1-2 weeks (parallel with Architecture work)

**Questions to Ask PM:**
- Should we add a "Getting Started" wizard in Grafana for first-time users? (Currently assumes users read docs)
- Should relay node selection be user-configurable or always automatic? (Currently automatic based on latency + geographic diversity)
- Should we add desktop notifications for connection failures? (Currently only Grafana visual indicators)

---

### For Architect

**Context:** ShadowMesh is a post-quantum encrypted private network built with Go 1.21+ (client/relay), Solidity 0.8.20+ (smart contracts), PostgreSQL 14+ (user data), and Docker Compose (monitoring). The MVP targets Linux (all major distros) with a 12-week development timeline. The Product Manager has defined 44 functional requirements, 31 non-functional requirements, and 6 sequential epics.

**Your Primary Objectives:**

1. **Design System Architecture** with component diagrams, data flow diagrams, and deployment architecture
2. **Specify Database Schema** for PostgreSQL (users, peers, relay nodes, connection history)
3. **Detail Multi-Hop Routing Protocol** including packet format, layered encryption, relay forwarding logic
4. **Design Relay-to-Relay Authentication** mechanism for secure multi-hop routing
5. **Optimize Prometheus Metrics Collection** to minimize performance impact (<1% CPU overhead)
6. **Identify Technical Risks** and propose mitigation strategies

---

**Detailed Architect Prompt:**

You are the Software Architect for ShadowMesh, a post-quantum encrypted private network. The Product Manager has provided a comprehensive PRD with functional requirements (FR1-FR44), non-functional requirements (NFR1-NFR31), technical assumptions, and 6 epics. Your task is to translate these requirements into a detailed technical architecture that developers can implement.

**1. System Architecture Design**

Create comprehensive architecture diagrams covering:

**A. High-Level Component Architecture**
- **Components:**
  - **ShadowMesh Client Daemon** (Go binary, runs as systemd service on user's Linux machine)
  - **Relay Node** (Go binary, deployed on VPS, routes encrypted traffic between clients)
  - **RelayNodeRegistry Smart Contract** (Solidity, deployed to Ethereum mainnet via chronara.eth ENS)
  - **Prometheus** (metrics storage, 7-day retention, scrapes client daemon every 15 seconds)
  - **Grafana** (dashboard UI, queries Prometheus via PromQL, serves dashboards on localhost:8080)
  - **Public Network Map** (React app, queries smart contract via Infura, displays relay nodes on map)
  - **PostgreSQL** (user/device management database, stores profiles, peer relationships, connection history)
- **Communication Protocols:**
  - Client ↔ Relay: HTTPS WebSocket Secure (WSS) on port 443, binary frames, encrypted Ethernet payloads
  - Client ↔ Smart Contract: HTTPS JSON-RPC via Infura/Alchemy (read-only queries), WebSocket for event subscriptions
  - Daemon ↔ Prometheus: HTTP metrics endpoint on port 9090 (Prometheus text exposition format)
  - Grafana ↔ Prometheus: HTTP PromQL queries on port 9091
  - Public Map ↔ Smart Contract: HTTPS JSON-RPC via Infura/Alchemy (read-only queries)
- **Diagram Format:** Layered architecture diagram showing client layer, relay layer, blockchain layer, monitoring layer with protocols annotated

**B. Data Flow Diagram**
- **Scenario 1: Direct P2P Connection**
  1. Client A initiates connection to Client B (runs `shadowmesh connect <peer-b-id>`)
  2. NAT type detection: both clients query STUN-like server, determine Full Cone NAT
  3. UDP hole punching: both clients send simultaneous UDP packets to each other's public IP:port
  4. PQC handshake: ML-KEM-1024 key exchange + ML-DSA-87 signature verification
  5. Encrypted tunnel established: ChaCha20-Poly1305 encryption of Ethernet frames
  6. TAP device capture: Client A captures frame → encrypts → sends via WSS → Client B receives → decrypts → injects into TAP device
  7. Metrics: Both clients expose Prometheus metrics (connection status, throughput, latency)
- **Scenario 2: Relay Fallback (CGNAT)**
  1. Client A behind CGNAT (Symmetric NAT), Client B behind Full Cone NAT
  2. UDP hole punching fails after 500ms timeout
  3. Client A queries chronara.eth smart contract: "getActiveNodes()" → receives list of 12 relay nodes
  4. Client A selects 3 relays: Relay1 (US), Relay2 (EU), Relay3 (Asia) based on latency + geographic diversity
  5. Multi-hop route established: Client A → Relay1 → Relay2 → Relay3 → Client B
  6. Layered encryption: Client A encrypts frame 3 times (one layer per relay), each relay decrypts one layer and forwards
  7. Metrics: Client A reports connection status "relay-routed", relay hop count "3", relay latency per hop
- **Diagram Format:** Sequence diagram showing message exchanges, encryption/decryption steps, timing annotations

**C. Deployment Architecture**
- **Client Deployment:**
  - Installation: .deb package (Ubuntu/Debian), .rpm package (Fedora/RHEL), universal install script (Arch, Manjaro, OpenSUSE)
  - systemd service: `/etc/systemd/system/shadowmesh.service` (ExecStart=/usr/local/bin/shadowmesh-daemon, Restart=always, CAP_NET_ADMIN capability)
  - Docker Compose: `/opt/shadowmesh/docker-compose.yml` (3 services: shadowmesh-daemon, prometheus, grafana)
  - Keystore: `/home/user/.shadowmesh/keystore.json` (encrypted with user passphrase via AES-256-GCM)
  - Logs: `/var/log/shadowmesh/daemon.log` (rotated via logrotate)
- **Relay Node Deployment:**
  - VPS providers: DigitalOcean, Linode, Vultr, Hetzner (2 vCPU, 4GB RAM, 1 Gbps network, $10-20/month)
  - Installation: `curl https://shadowmesh.network/install-relay.sh | sudo bash`
  - systemd service: `/etc/systemd/system/shadowmesh-relay.service`
  - Configuration: `/etc/shadowmesh/relay.conf` (listening port, stake wallet private key, geographic location)
  - Monitoring: Relay operators can access Grafana dashboard on localhost:8080 (same Docker Compose stack as client)
- **Smart Contract Deployment:**
  - Ethereum mainnet: Deployed to address resolved by chronara.eth ENS domain
  - Sepolia testnet: For testing before mainnet deployment
  - Hardhat deployment script: `npx hardhat run scripts/deploy.ts --network mainnet`
  - Etherscan verification: `npx hardhat verify --network mainnet <contract-address>`
- **Public Network Map Deployment:**
  - Hosting: Vercel or Netlify (static site hosting)
  - Custom domain: map.shadowmesh.network
  - Build: `npm run build` (React app compiled to static HTML/CSS/JS)
  - Deployment: `vercel deploy` or `netlify deploy`
- **Diagram Format:** Deployment diagram showing infrastructure components, network topology, firewall rules

**2. Database Schema Design**

Design PostgreSQL database schema for ShadowMesh user/device management:

**A. Tables**

**users table:**
```sql
CREATE TABLE users (
    user_id SERIAL PRIMARY KEY,
    telegram_user_id BIGINT UNIQUE NOT NULL,  -- Reuse chronara.ai NFT bot pattern
    username VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

**devices table:**
```sql
CREATE TABLE devices (
    device_id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(user_id) ON DELETE CASCADE,
    device_name VARCHAR(255) NOT NULL,  -- e.g., "laptop", "desktop", "server"
    public_key_hash BYTEA NOT NULL,  -- SHA-256 hash of hybrid public key
    keystore_encrypted BYTEA NOT NULL,  -- AES-256-GCM encrypted keystore (backup)
    registered_at TIMESTAMP DEFAULT NOW(),
    last_seen TIMESTAMP DEFAULT NOW()
);
```

**peer_relationships table:**
```sql
CREATE TABLE peer_relationships (
    relationship_id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(user_id) ON DELETE CASCADE,
    peer_user_id INTEGER REFERENCES users(user_id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) DEFAULT 'friend',  -- 'friend', 'device_group', 'trusted'
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, peer_user_id)
);
```

**connection_history table:**
```sql
CREATE TABLE connection_history (
    connection_id SERIAL PRIMARY KEY,
    device_id INTEGER REFERENCES devices(device_id) ON DELETE CASCADE,
    peer_device_id INTEGER REFERENCES devices(device_id) ON DELETE CASCADE,
    connection_type VARCHAR(50),  -- 'direct_p2p', 'relay_routed'
    started_at TIMESTAMP DEFAULT NOW(),
    ended_at TIMESTAMP,
    duration_seconds INTEGER,
    bytes_sent BIGINT DEFAULT 0,
    bytes_received BIGINT DEFAULT 0,
    average_latency_ms INTEGER,
    packet_loss_percentage DECIMAL(5,2)
);
```

**relay_nodes table:** (Replicated from blockchain for faster querying)
```sql
CREATE TABLE relay_nodes (
    node_id SERIAL PRIMARY KEY,
    blockchain_address VARCHAR(42) UNIQUE NOT NULL,  -- Ethereum address
    public_key_hash BYTEA NOT NULL,
    endpoint VARCHAR(255) NOT NULL,  -- e.g., "wss://relay1.shadowmesh.network:443"
    geographic_location VARCHAR(255),  -- e.g., "New York, USA"
    latitude DECIMAL(9,6),
    longitude DECIMAL(9,6),
    stake_amount DECIMAL(20,8),  -- ETH staked
    last_heartbeat TIMESTAMP,
    uptime_percentage DECIMAL(5,2),
    registered_at TIMESTAMP DEFAULT NOW(),
    last_synced_from_blockchain TIMESTAMP DEFAULT NOW()
);
```

**access_logs table:** (Audit trail)
```sql
CREATE TABLE access_logs (
    log_id SERIAL PRIMARY KEY,
    device_id INTEGER REFERENCES devices(device_id) ON DELETE SET NULL,
    event_type VARCHAR(100),  -- 'connection_attempt', 'connection_success', 'connection_failure', 'key_rotation', 'relay_fallback'
    event_details JSONB,
    ip_address INET,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**B. Indexes**
```sql
CREATE INDEX idx_devices_user_id ON devices(user_id);
CREATE INDEX idx_connection_history_device_id ON connection_history(device_id);
CREATE INDEX idx_connection_history_started_at ON connection_history(started_at);
CREATE INDEX idx_relay_nodes_last_heartbeat ON relay_nodes(last_heartbeat);
CREATE INDEX idx_access_logs_device_id ON access_logs(device_id);
CREATE INDEX idx_access_logs_created_at ON access_logs(created_at);
```

**C. Data Migration Strategy**
- MVP: No existing data to migrate (new system)
- Beta → Production: If schema changes required, use migration tools (Flyway, Liquibase, or custom Go migration scripts)

**3. Multi-Hop Routing Protocol Specification**

Design detailed protocol for multi-hop onion routing (Story 4.4):

**A. Packet Format**

**Encrypted Frame Structure (3-Layer Onion):**
```
[Outer Header | Layer 3 Encrypted Payload]
                 ↓ Relay3 decrypts with its key
[Middle Header | Layer 2 Encrypted Payload]
                  ↓ Relay2 decrypts with its key
[Inner Header | Layer 1 Encrypted Payload]
                 ↓ Relay1 decrypts with its key
[Final Header | Plaintext Ethernet Frame]
                ↓ Delivered to destination peer
```

**Header Format (Each Layer):**
```
struct RoutingHeader {
    version uint8              // Protocol version (0x01 for MVP)
    hop_number uint8           // Current hop (1, 2, or 3)
    next_relay_id [32]byte     // Public key hash of next relay (or destination peer)
    payload_length uint32      // Length of encrypted payload
    nonce [12]byte             // ChaCha20-Poly1305 nonce for this layer
    auth_tag [16]byte          // ChaCha20-Poly1305 authentication tag
}
```

**B. Path Construction (Client Side)**

1. **Relay Selection:**
   - Query chronara.eth smart contract: `getActiveNodes()` → list of relay nodes
   - Filter by last heartbeat <24 hours ago
   - Select 3 relays with constraints:
     - Different operators (verify via on-chain metadata)
     - Geographic diversity (prefer different continents)
     - Low latency (measure via health check endpoint: `/health`)
   - Path example: Relay1 (US East), Relay2 (EU West), Relay3 (Asia Pacific)

2. **Key Derivation:**
   - Perform PQC key exchange with each relay independently:
     - Client ↔ Relay1: ML-KEM-1024 encapsulation → shared secret S1
     - Client ↔ Relay2: ML-KEM-1024 encapsulation → shared secret S2
     - Client ↔ Relay3: ML-KEM-1024 encapsulation → shared secret S3
   - Derive layer keys using HKDF-SHA256:
     - Layer1_Key = HKDF(S1, "ShadowMesh-Layer1", 32 bytes)
     - Layer2_Key = HKDF(S2, "ShadowMesh-Layer2", 32 bytes)
     - Layer3_Key = HKDF(S3, "ShadowMesh-Layer3", 32 bytes)

3. **Onion Encryption:**
   - Start with plaintext Ethernet frame
   - Encrypt Layer1: ChaCha20-Poly1305(Layer1_Key, plaintext_frame) → Layer1_Ciphertext
   - Add Inner Header: NextRelay = Relay2_ID
   - Encrypt Layer2: ChaCha20-Poly1305(Layer2_Key, Inner_Header + Layer1_Ciphertext) → Layer2_Ciphertext
   - Add Middle Header: NextRelay = Relay3_ID
   - Encrypt Layer3: ChaCha20-Poly1305(Layer3_Key, Middle_Header + Layer2_Ciphertext) → Layer3_Ciphertext
   - Add Outer Header: NextRelay = Relay1_ID
   - Final packet: Outer_Header + Layer3_Ciphertext

4. **Transmission:**
   - Send final packet to Relay1 via WebSocket connection

**C. Relay Forwarding Logic**

**Relay Node Packet Processing:**
```go
func (r *RelayNode) ProcessPacket(packet []byte) error {
    // Parse header
    header := parseRoutingHeader(packet[:HEADER_SIZE])
    ciphertext := packet[HEADER_SIZE:]

    // Decrypt one layer
    layer_key := r.getDerivedKey(header.hop_number)  // Retrieve pre-shared key from handshake
    plaintext := chacha20poly1305.Decrypt(layer_key, header.nonce, ciphertext, header.auth_tag)

    // Extract next relay ID from decrypted header
    inner_header := parseRoutingHeader(plaintext[:HEADER_SIZE])
    next_relay_id := inner_header.next_relay_id

    // Forward to next relay
    if next_relay_id == DESTINATION_PEER {
        // Final hop, deliver to peer
        return r.deliverToPeer(plaintext[HEADER_SIZE:])
    } else {
        // Forward to next relay
        next_relay := r.lookupRelay(next_relay_id)
        return r.forwardPacket(next_relay, plaintext)
    }
}
```

**D. Operator Diversity Enforcement**

- **Challenge:** Prevent Sybil attacks where single operator controls multiple relays in path
- **Solution:**
  - Smart contract stores `operator_address` field for each relay node
  - Client queries: `SELECT node_id, operator_address, endpoint FROM relay_nodes WHERE last_heartbeat > NOW() - INTERVAL '24 hours'`
  - Client filters: Ensure Relay1.operator_address ≠ Relay2.operator_address ≠ Relay3.operator_address
  - If insufficient relays from different operators, fall back to 2-hop or direct connection

**E. Performance Validation**

- **Target:** Multi-hop latency <50ms overhead vs direct P2P
- **Measurement:** Ping-pong test between Client A and Client B:
  - Direct P2P latency: 20ms average
  - 3-hop relay latency: 65ms average → 45ms overhead (meets target)
- **Optimization:** If overhead exceeds 50ms, reduce to 2-hop routing or select geographically closer relays

**4. Relay-to-Relay Authentication**

**Challenge:** Relays must authenticate each other to prevent man-in-the-middle attacks during multi-hop routing

**Proposed Solution: Blockchain-Based Mutual TLS**

**A. Certificate Generation:**
- Each relay node generates self-signed TLS certificate
- Certificate public key hash stored in smart contract during registration
- Client retrieves public key hashes from smart contract for all relays in path

**B. Authentication Flow:**
1. Client initiates connection to Relay1 via TLS 1.3
2. Relay1 presents TLS certificate
3. Client computes SHA-256 hash of certificate public key
4. Client compares hash with on-chain value: `relay_nodes[relay1_id].public_key_hash`
5. If match, connection authenticated; if mismatch, reject connection and log security event
6. Repeat for Relay1 → Relay2 and Relay2 → Relay3 connections

**C. Key Rotation:**
- Relay operators can rotate TLS certificates by calling smart contract: `updatePublicKeyHash(new_hash)`
- Update requires signature from operator's stake wallet (prevents unauthorized key changes)
- Client caches certificate hashes for 5 minutes to reduce blockchain queries

**Alternative Solutions to Consider:**
- **Mutual TLS with CA:** Use Let's Encrypt for relay certificates (simpler but centralized CA dependency)
- **Direct Signature Verification:** Skip TLS, use ML-DSA-87 signatures for each packet (higher CPU overhead)

**5. Prometheus Metrics Collection Optimization**

**Challenge:** Exposing 20+ metrics every 15 seconds must not impact network performance (<1% CPU overhead)

**Proposed Architecture:**

**A. Metrics Storage (In-Memory):**
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

**B. Metrics Update (Lock-Free):**
- Use atomic operations for counters: `atomic.AddUint64(&m.pqcHandshakesTotal, 1)`
- Update gauges in dedicated goroutine (avoid blocking network I/O):
  ```go
  go func() {
      ticker := time.NewTicker(1 * time.Second)
      for range ticker.C {
          m.connectionLatency.Set(getCurrentLatency())
          m.throughput.Add(getBytesSinceLastUpdate())
      }
  }()
  ```

**C. Metrics Exposition (HTTP Endpoint):**
- Use Prometheus client library: `promhttp.Handler()`
- Endpoint serves pre-computed metrics (no expensive computations during scrape)
- Benchmark target: <100ms response time for `/metrics` endpoint

**D. Performance Validation:**
- Measure CPU overhead: Run daemon with 50 active connections, monitor CPU usage with and without metrics collection
- Target: <1% CPU difference (measured via `top` or `htop`)
- If overhead exceeds 1%, reduce scrape frequency to 30 seconds or disable expensive metrics (e.g., per-frame encryption rate)

**6. Technical Risk Mitigation**

**A. CGNAT Traversal (95% Success Rate Target)**

**Risk:** Symmetric NAT and double NAT scenarios may prevent direct P2P connections, forcing relay routing

**Mitigation Strategies:**
1. **Comprehensive Testing:** Test on 20+ diverse network configurations (cellular, corporate, residential ISPs)
2. **Relay Fallback:** Ensure 10+ relay nodes available across continents for redundancy
3. **STUN Server Redundancy:** Run multiple STUN-like servers (3+) to handle NAT detection failures
4. **Graceful Degradation:** If 3-hop routing fails, try 2-hop, then 1-hop, then report failure to user
5. **Metrics:** Track CGNAT traversal success rate per NAT type (log to `connection_history` table for analysis)

**B. PQC Performance (1+ Gbps Throughput, <5ms Latency)**

**Risk:** Cloudflare CIRCL benchmarks show 22.7μs decapsulation, but real-world multi-peer scenarios untested

**Mitigation Strategies:**
1. **Early Benchmarking:** Epic 1 Story 1.7 validates performance before networking code (fail fast if targets unmet)
2. **Hardware Profiling:** Test on low-end hardware (Raspberry Pi 4, 2 vCPU VPS) to ensure commodity support
3. **Fallback Option:** If PQC overhead exceeds <5ms, offer "fast mode" with classical crypto only (explicit user opt-in, not recommended)
4. **Optimization:** Use SIMD instructions (AVX2, AVX-512) if available via Go assembly or C bindings

**C. Ethereum Gas Costs (<$10 Registration)**

**Risk:** Gas price volatility could spike registration costs above $10 target during network congestion

**Mitigation Strategies:**
1. **Gas Optimization:** Epic 3 Story 3.6 targets <20,000 gas (well below $10 at 25 gwei, $3,000 ETH)
2. **Gas Price Monitoring:** Relay registration script checks current gas price, warns if >50 gwei (user can wait for lower gas)
3. **Layer 2 Consideration:** If mainnet gas consistently >$10, deploy contract to Arbitrum/Optimism (deferred to Beta if needed)
4. **Batch Registration:** Allow operators to register multiple relays in single transaction (amortize gas cost)

**D. TAP Device Permissions (CAP_NET_ADMIN)**

**Risk:** Requiring root/sudo for TAP device creation complicates installation on restricted systems

**Mitigation Strategies:**
1. **systemd Service:** Run daemon as systemd service with `AmbientCapabilities=CAP_NET_ADMIN` (no full root required)
2. **Installation Script:** Script automatically configures systemd service with correct capabilities
3. **Documentation:** Explain why CAP_NET_ADMIN required (Layer 2 networking) and how to verify security (daemon code is open-source)
4. **Alternative:** If TAP devices unavailable, fall back to TUN devices (Layer 3, no CAP_NET_ADMIN required, but weaker privacy)

**Deliverables:**

1. **System Architecture Document:** PDF with component diagrams, data flow diagrams, deployment architecture (20-30 pages)
2. **Database Schema SQL:** Complete PostgreSQL schema with tables, indexes, constraints (`schema.sql` file)
3. **Multi-Hop Routing Protocol Spec:** Detailed packet format, path construction algorithm, relay forwarding logic (Markdown document)
4. **Relay Authentication Design:** Blockchain-based mutual TLS specification with key rotation mechanism (Markdown document)
5. **Prometheus Metrics Architecture:** In-memory storage design, lock-free update mechanism, performance benchmarks (Markdown document)
6. **Technical Risk Assessment:** Updated risk register with mitigation strategies and contingency plans (Excel or Markdown table)

**Timeline:** 2-3 weeks (can proceed in parallel with UX Expert work and Epic 1 development)

**Questions to Ask PM:**
- Should relay node operators be incentivized with token rewards in MVP or defer to Beta? (Currently placeholder panels in Grafana)
- Should we support IPv6 in MVP or defer to Beta? (Currently IPv4-only)
- Should we add rate limiting to smart contract queries (prevent abuse) or trust Infura's rate limits?

---

**End of PRD**

---

**Document Status:** ✅ COMPLETE - Ready for UX Expert and Architect handoff

**Last Updated:** 2025-10-31

**Prepared by:** John (Product Manager)

**Next Review:** After UX and Architecture deliverables completed (estimated 2-3 weeks)

