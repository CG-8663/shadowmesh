# ShadowMesh Terminology Guide

**Purpose**: Define ShadowMesh components and compare terminology with WireGuard, Tailscale, and ZeroTier

**Last Updated**: November 9, 2025

---

## The Nexus: ShadowMesh Naming Convention

The ShadowMesh network is officially named **The Nexus**. This designation reflects the system's foundational, self-organizing, and decentralized nature—a single point of origin from which a vast, secure network reality is built. The terminology below uses a Blake's 7 theme, reflecting the Nexus as a force for security and freedom operating against a chaotic, unorganized universe.

### The ShadowMesh Nexus Hierarchy

| ShadowMesh Component | Nexus/Blake's 7 Term | Conceptual Role |
|---------------------|---------------------|-----------------|
| **THE NETWORK ITSELF (ShadowMesh)** | **The Nexus** | The entire secure, decentralized network. The foundation of all reality in the system. |
| **All Peers (General)** | **The Free Territory** | All the devices and nodes collectively existing within the Nexus. |
| **Kademlia DHT Network** | **Orac's Web** | The distributed, self-organizing knowledge base for routing and discovery. The mind of the Nexus. |
| **Blockchain Smart Contracts** | **The Federation Registry** | The immutable rule-set (Registry, Staking/Slashing) that governs trust and incentivization. The Law of the Nexus. |
| **Bootstrap Node** | **A Federation Exile** | A long-lived, community-run node that helps new Peers join Orac's Web. Nodes that know the initial location of the universe. |
| **Standard Peer (End Device)** | **A Seven** | A critical, independent, and trusted end-user device. A sentient unit existing within the Nexus. |
| **Relay Node (Staked)** | **An Avenger** | A trusted, incentivized Peer that stakes tokens to provide circuit relay services. An active defender and facilitator of the Nexus. |
| **Subnet Router / Gateway** | **A Transport** | A Peer configured as a secure access point to external LANs or as an Exit Node. A vessel for movement across the boundaries of the Nexus. |
| **P2P Connection Status** | **The Liberator's Flight** | A successful, direct, relay-free connection. The fastest, freest path within the Nexus. |

**Narrative Context**: In the Blake's 7 universe, the Liberator was the protagonists' advanced alien spacecraft, representing freedom and resistance against the authoritarian Federation. Similarly, The Nexus represents a decentralized force for security and privacy operating against centralized surveillance and control.

---

## ShadowMesh Architecture Components

### Network Layer

#### **PeerID**
**Definition**: Unique identifier for each node in the mesh network, derived from ML-DSA-87 public key via SHA256 hash.

**Comparison**:
- **WireGuard**: Public key (base64-encoded Curve25519 key)
- **Tailscale**: Node ID (derived from node's public key)
- **ZeroTier**: ZeroTier Address (10-digit hex identifier)

**ShadowMesh Advantage**: Quantum-safe PeerID derived from post-quantum signature key, cryptographically bound to node identity.

---

#### **Kademlia DHT (Distributed Hash Table)**
**Definition**: Decentralized peer discovery mechanism using XOR distance metric and k-bucket routing tables. Zero central servers.

**Comparison**:
- **WireGuard**: No peer discovery (manual endpoint configuration)
- **Tailscale**: Centralized coordination server (control plane)
- **ZeroTier**: Centralized root servers + planet/moon infrastructure

**ShadowMesh Advantage**: Fully decentralized, no single point of failure, token-based operational model (gas + storage fees via Chronara tokens).

---

#### **Bootstrap Nodes**
**Definition**: Initial hardcoded peers (3-5 nodes) used to enter the DHT network on first startup. After joining, peers discover others via DHT queries.

**Comparison**:
- **WireGuard**: No concept (manual peer configuration)
- **Tailscale**: Coordination server (login.tailscale.com)
- **ZeroTier**: Root servers (hardcoded planet file)

**ShadowMesh Advantage**: Bootstrap nodes are temporary - once in DHT, node operates independently. No ongoing dependency.

---

#### **k-Buckets**
**Definition**: Routing table structure with 160 buckets, each holding up to k=20 peers. Organized by XOR distance from own PeerID.

**Comparison**:
- **WireGuard**: Flat peer list (no routing table)
- **Tailscale**: Centralized routing managed by control server
- **ZeroTier**: Centralized routing via controller

**ShadowMesh Advantage**: Self-organizing routing table, automatic peer discovery, scales to millions of nodes.

---

### Cryptography Layer

#### **ML-KEM-1024 (Kyber)**
**Definition**: Post-quantum key encapsulation mechanism (NIST FIPS 203) for establishing shared secrets resistant to quantum attacks.

**Comparison**:
- **WireGuard**: X25519 ECDH (vulnerable to quantum computers)
- **Tailscale**: WireGuard crypto (quantum vulnerable)
- **ZeroTier**: ECDH P-384 (quantum vulnerable)

**ShadowMesh Advantage**: Quantum-safe key exchange, 5+ year security lead.

---

#### **ML-DSA-87 (Dilithium)**
**Definition**: Post-quantum digital signature algorithm (NIST FIPS 204) for peer authentication and PeerID generation.

**Comparison**:
- **WireGuard**: No signatures (relies on PSK for authentication)
- **Tailscale**: Ed25519 signatures (quantum vulnerable)
- **ZeroTier**: ECDSA P-384 (quantum vulnerable)

**ShadowMesh Advantage**: Quantum-resistant peer authentication, prevents impersonation attacks.

---

#### **Hybrid Mode**
**Definition**: Simultaneous use of classical (X25519, Ed25519) and post-quantum (ML-KEM-1024, ML-DSA-87) algorithms. Provides defense-in-depth.

**Comparison**:
- **WireGuard**: Classical only
- **Tailscale**: Classical only
- **ZeroTier**: Classical only

**ShadowMesh Advantage**: Protected against both current and future (quantum) threats.

---

#### **ChaCha20-Poly1305**
**Definition**: Symmetric authenticated encryption for tunnel traffic (same as WireGuard).

**Comparison**:
- **WireGuard**: ChaCha20-Poly1305 ✅
- **Tailscale**: ChaCha20-Poly1305 (via WireGuard) ✅
- **ZeroTier**: Salsa20-Poly1305

**ShadowMesh**: Industry-standard AEAD cipher, 6-7 Gbps throughput target.

---

### Transport Layer

#### **QUIC Transport**
**Definition**: UDP-based protocol with built-in reliability, congestion control, and 0-RTT reconnection. Multiplexed streams over single connection.

**Comparison**:
- **WireGuard**: Raw UDP with custom reliability
- **Tailscale**: WireGuard UDP + DERP fallback (TCP)
- **ZeroTier**: UDP with TCP fallback

**ShadowMesh Advantage**: QUIC provides better NAT traversal, connection migration (IP changes), and multiplexing without head-of-line blocking.

---

#### **TUN Device**
**Definition**: Layer 3 virtual network interface for IP packet tunneling. Operates at network layer (IP datagrams).

**Comparison**:
- **WireGuard**: TUN device ✅
- **Tailscale**: TUN device (via WireGuard) ✅
- **ZeroTier**: TAP device (Layer 2 Ethernet frames)

**ShadowMesh**: Layer 3 TUN for simplicity and performance (no Ethernet overhead).

---

### Peer Discovery

#### **FIND_NODE**
**Definition**: DHT operation to locate k closest peers to a target PeerID. Returns peer contact information (IP, port, public keys).

**Comparison**:
- **WireGuard**: No peer discovery
- **Tailscale**: Centralized DERP map from control server
- **ZeroTier**: Centralized peer list from controller

**ShadowMesh**: Iterative DHT lookup with α=3 parallel requests, <100ms target latency.

---

#### **STORE / FIND_VALUE**
**Definition**: DHT operations to publish and retrieve peer metadata (IP addresses, capabilities, public keys) with TTL (24-hour default).

**Comparison**:
- **WireGuard**: No metadata storage
- **Tailscale**: Centralized database
- **ZeroTier**: Centralized controller database

**ShadowMesh**: Distributed key-value storage, no central database, automatic expiration.

---

### NAT Traversal

#### **Hole Punching**
**Definition**: Technique to establish direct connections through NATs by coordinating simultaneous packet sends from both peers.

**Comparison**:
- **WireGuard**: No built-in hole punching (requires manual keepalive)
- **Tailscale**: Automatic via coordination server
- **ZeroTier**: Automatic via root servers

**ShadowMesh**: QUIC-based hole punching coordinated via DHT peer exchange (no central server).

---

#### **Relay Fallback**
**Definition**: Optional relay servers for symmetric NAT scenarios where direct connection impossible. Community-run, not required for operation.

**Comparison**:
- **WireGuard**: No relay support
- **Tailscale**: DERP relay servers (Tailscale-operated)
- **ZeroTier**: Relay via planet servers (ZeroTier-operated)

**ShadowMesh**: Optional community relays, not required for most deployments, zero dependency on project operators.

---

## Network Topology Comparison

### WireGuard: Manual Peer-to-Peer

```
┌─────────┐     Manual Config      ┌─────────┐
│ Peer A  │ ←──────────────────→  │ Peer B  │
└─────────┘                        └─────────┘
     ↑                                  ↑
     └──────── Manual Config ───────────┘
                  ┌─────────┐
                  │ Peer C  │
                  └─────────┘
```

**Characteristics**:
- Manual endpoint configuration required
- No peer discovery
- Static configuration files
- No automatic routing

---

### Tailscale: Star Topology (Centralized Control)

```
              ┌──────────────────┐
              │ Control Server   │
              │ (Coordination)   │
              └────────┬─────────┘
                       │
         ┌─────────────┼─────────────┐
         ↓             ↓             ↓
    ┌────────┐    ┌────────┐    ┌────────┐
    │ Peer A │───→│ Peer B │←───│ Peer C │
    └────────┘    └────────┘    └────────┘

    Direct P2P after coordination
```

**Characteristics**:
- Centralized control plane (coordination server)
- Automatic peer discovery via control server
- Direct P2P after initial coordination
- DERP relay fallback for difficult NATs

---

### ZeroTier: Centralized Controller + Root Servers

```
         ┌──────────────┐
         │ Root Servers │ (planet file)
         └──────┬───────┘
                │
         ┌──────┴───────┐
         │  Controller  │ (network config)
         └──────┬───────┘
                │
    ┌───────────┼───────────┐
    ↓           ↓           ↓
┌────────┐  ┌────────┐  ┌────────┐
│ Peer A │  │ Peer B │  │ Peer C │
└────────┘  └────────┘  └────────┘
```

**Characteristics**:
- Centralized root servers for discovery
- Centralized controller for network configuration
- P2P after discovery (via root servers)
- Layer 2 bridging support

---

### ShadowMesh: Fully Decentralized Mesh (Kademlia DHT)

```
    ┌────────────────────────────────┐
    │     Kademlia DHT Network       │
    │   (Distributed Peer Discovery) │
    └────────┬───────────────────────┘
             │
    ┌────────┼────────┐
    ↓        ↓        ↓
┌────────┐ ┌────────┐ ┌────────┐
│ Peer A │→│ Peer B │←│ Peer C │
└───┬────┘ └────────┘ └───┬────┘
    │                      │
    └──────────┬───────────┘
               ↓
          ┌────────┐
          │ Peer D │
          └────────┘
```

**Characteristics**:
- **Zero central servers** (fully decentralized)
- DHT-based peer discovery (no single point of failure)
- Self-organizing routing table (k-buckets)
- Bootstrap nodes only for initial entry (3-5 hardcoded peers)
- Operates independently after joining DHT

---

## Terminology Cross-Reference

| ShadowMesh Term | WireGuard Equivalent | Tailscale Equivalent | ZeroTier Equivalent |
|----------------|---------------------|---------------------|---------------------|
| **PeerID** | Public Key | Node ID | ZeroTier Address |
| **Kademlia DHT** | N/A (manual) | Coordination Server | Root Servers |
| **Bootstrap Nodes** | N/A | login.tailscale.com | Planet file |
| **k-Buckets** | N/A | N/A | N/A |
| **QUIC Transport** | UDP + WireGuard | WireGuard UDP | UDP + ZeroTier |
| **TUN Device** | TUN ✅ | TUN (via WireGuard) ✅ | TAP (Layer 2) |
| **ML-KEM-1024** | X25519 (classical) | X25519 (classical) | ECDH P-384 |
| **ML-DSA-87** | N/A (PSK) | Ed25519 | ECDSA P-384 |
| **Hybrid Mode** | N/A | N/A | N/A |
| **Relay Fallback** | N/A | DERP Servers | Planet/Moon Relays |
| **FIND_NODE** | N/A | Control Server API | Root Server Query |
| **STORE/FIND_VALUE** | N/A | Control Server DB | Controller DB |

---

## Key Differentiators

### 1. **Centralization vs Decentralization**

| Solution | Architecture | Single Point of Failure? | Operational Cost |
|----------|-------------|--------------------------|------------------|
| **WireGuard** | Manual P2P | No (but no discovery) | $0 |
| **Tailscale** | Centralized control | **Yes** (control server) | Tailscale pays |
| **ZeroTier** | Centralized roots | **Yes** (root servers) | ZeroTier pays |
| **ShadowMesh** | Decentralized DHT | **No** | **$0** |

---

### 2. **Post-Quantum Cryptography**

| Solution | Key Exchange | Signatures | Quantum-Safe? |
|----------|-------------|-----------|---------------|
| **WireGuard** | X25519 | N/A (PSK) | ❌ No |
| **Tailscale** | X25519 | Ed25519 | ❌ No |
| **ZeroTier** | ECDH P-384 | ECDSA P-384 | ❌ No |
| **ShadowMesh** | ML-KEM-1024 + X25519 | ML-DSA-87 + Ed25519 | ✅ **Yes** |

**ShadowMesh**: 5+ year security advantage with NIST-standardized PQC.

---

### 3. **NAT Traversal**

| Solution | Automatic Hole Punching? | Relay Fallback? | Relay Dependency? |
|----------|--------------------------|----------------|-------------------|
| **WireGuard** | Manual keepalive | No | N/A |
| **Tailscale** | ✅ Yes (via control) | ✅ DERP | Required |
| **ZeroTier** | ✅ Yes (via roots) | ✅ Planet/Moon | Required |
| **ShadowMesh** | ✅ Yes (via DHT) | ⏳ Optional | **Not required** |

---

### 4. **Performance Targets**

| Solution | Throughput | Latency Overhead | Connection Setup |
|----------|-----------|------------------|------------------|
| **WireGuard** | 1+ Gbps | <1ms | Fast (1-RTT) |
| **Tailscale** | 1+ Gbps | <1ms (WireGuard) | Medium (via control) |
| **ZeroTier** | ~500 Mbps | ~5ms | Slow (via roots) |
| **ShadowMesh** | **6-7 Gbps** | **<2ms** | Fast (QUIC 0-RTT) |

**ShadowMesh Target**: 6-7x throughput improvement over competitors.

---

## Configuration Comparison

### WireGuard Configuration

```ini
[Interface]
PrivateKey = <base64-private-key>
Address = 10.0.0.1/24
ListenPort = 51820

[Peer]
PublicKey = <base64-public-key>
Endpoint = 192.168.1.100:51820
AllowedIPs = 10.0.0.2/32
PersistentKeepalive = 25
```

**Manual**: Requires manual endpoint configuration, no peer discovery.

---

### Tailscale Configuration

```bash
# Login (uses centralized control server)
tailscale login

# No config file needed - control server manages everything
# Automatic peer discovery via login.tailscale.com
```

**Automatic**: Zero configuration, but requires Tailscale account and control server.

---

### ZeroTier Configuration

```bash
# Join network (requires network ID from controller)
zerotier-cli join <network-id>

# Controller manages routing and authorization
```

**Hybrid**: Minimal config, but requires ZeroTier Central controller.

---

### ShadowMesh Configuration (Target v1.0.0)

```yaml
# shadowmesh.yaml
network:
  peer_id: auto-generated  # From ML-DSA-87 key

bootstrap_nodes:
  - peer1.shadowmesh.io:9000
  - peer2.shadowmesh.io:9000
  - peer3.shadowmesh.io:9000

crypto:
  mode: hybrid  # ML-KEM-1024 + X25519

quic:
  listen_port: 9000
  max_streams: 100
```

**Minimal Config**: Bootstrap nodes only, fully automatic peer discovery via DHT afterward.

---

## Operational Model Comparison

### WireGuard: DIY Infrastructure

**User Responsibility**:
- Manual server deployment
- Manual peer configuration
- Manual firewall rules
- Manual endpoint updates

**Complexity**: High (requires networking knowledge)

---

### Tailscale: Managed Service

**Tailscale Provides**:
- Coordination servers (global fleet)
- DERP relay servers (automatic fallback)
- Web-based admin panel
- Automatic updates

**User Cost**: $0 (personal), $6-18/user/month (business)
**Dependency**: Tailscale infrastructure must remain operational

---

### ZeroTier: Managed Service

**ZeroTier Provides**:
- Root servers (planet file)
- Network controllers (ZeroTier Central)
- Web-based admin panel
- Relay infrastructure

**User Cost**: $0 (personal), $50-500/month (business)
**Dependency**: ZeroTier infrastructure must remain operational

---

### ShadowMesh: Decentralized Token-Based P2P

**ShadowMesh Provides**:
- 3-5 bootstrap nodes (temporary, for initial DHT entry)
- Open-source client software
- DHT-based peer discovery (decentralized)
- Blockchain-based relay verification (The Federation Registry)

**User Cost**:
- **Personal**: $10/month subscription + Chronara tokens for gas/storage fees
- **Business**: $30/user/month + Chronara tokens
- **Enterprise**: $100/user/month + Chronara tokens
- **Custom Implementations**: $10/client/month

**Token Requirements**:
- **Gas Fees**: 10-50 CHRONARA/month for blockchain operations
- **Database Fees**: 0.1 CHRONARA per GB/month for DHT storage
- **Interconnect Fees**: 0.01 CHRONARA per GB transferred via relay fallback
- **Relay Staking**: 100-1000 CHRONARA (one-time) to operate an Avenger relay node

**Dependency**: Bootstrap nodes for initial DHT entry, Chronara tokens for network operations

**Operational Cost**: Token-based (gas + storage + bandwidth costs)

---

## Use Case Terminology

### **Mesh Network**
**ShadowMesh**: Every peer discovers and connects to multiple other peers via DHT, forming a self-organizing mesh.

**WireGuard**: Manual mesh (requires full peer configuration matrix).

**Tailscale**: Star topology mesh (via control server coordination).

**ZeroTier**: Controller-managed mesh (via centralized routing).

---

### **Exit Node**
**Definition**: Peer that routes internet traffic for other peers (VPN exit point).

**ShadowMesh**: Future feature (v1.3.0+) with TPM/SGX attestation for zero-trust verification.

**WireGuard**: Manual configuration via AllowedIPs = 0.0.0.0/0.

**Tailscale**: Automatic exit node feature (Exit Nodes menu).

**ZeroTier**: No native exit node support.

---

### **Site-to-Site VPN**
**Definition**: Connect two networks (e.g., office to datacenter) via VPN.

**ShadowMesh**: Supported via TUN device routing (v1.0.0+).

**WireGuard**: Supported (manual routing configuration).

**Tailscale**: Supported via Subnet Routes feature.

**ZeroTier**: Supported via Layer 2 bridging.

---

## Future Terminology (Post v1.0.0)

### **Multi-Hop Routing** (v1.2.0)
**Definition**: Route traffic through 3-5 intermediate peers for enhanced privacy (onion routing).

**ShadowMesh**: Planned feature with configurable hop count.

**Competitors**: Not supported by WireGuard/Tailscale/ZeroTier.

---

### **Traffic Obfuscation** (v1.1.0)
**Definition**: Disguise VPN traffic as HTTPS/WebSocket to bypass deep packet inspection (DPI).

**ShadowMesh**: Planned feature with QUIC mimicry and cover traffic.

**Competitors**: Not supported (easily detected and blocked).

---

### **Blockchain Governance** (v2.1.0)
**Definition**: Smart contract-based relay node verification and reputation system.

**ShadowMesh**: Future enhancement for zero-trust relay infrastructure.

**Competitors**: Centralized governance only.

---

## Glossary

**AEAD**: Authenticated Encryption with Associated Data (e.g., ChaCha20-Poly1305)

**DPI**: Deep Packet Inspection (network traffic analysis to detect VPN usage)

**DHT**: Distributed Hash Table (decentralized key-value store)

**ECDH**: Elliptic Curve Diffie-Hellman (classical key exchange)

**KEM**: Key Encapsulation Mechanism (quantum-safe key exchange)

**NAT**: Network Address Translation (firewall that blocks unsolicited inbound connections)

**PFS**: Perfect Forward Secrecy (past sessions remain secure if keys compromised)

**PQC**: Post-Quantum Cryptography (algorithms resistant to quantum computer attacks)

**PSK**: Pre-Shared Key (symmetric secret shared before connection)

**QUIC**: Quick UDP Internet Connections (modern transport protocol)

**RTT**: Round-Trip Time (time for packet to reach peer and return)

**TAP**: Layer 2 virtual network device (Ethernet frames)

**TUN**: Layer 3 virtual network device (IP packets)

**XOR Distance**: Kademlia metric for measuring "closeness" between PeerIDs

---

## Summary: Why ShadowMesh?

| Feature | WireGuard | Tailscale | ZeroTier | ShadowMesh |
|---------|-----------|-----------|----------|------------|
| **Quantum-Safe** | ❌ | ❌ | ❌ | ✅ |
| **Decentralized** | ⚠️ Manual | ❌ | ❌ | ✅ DHT |
| **Pricing (Personal)** | Free | Free* | Free* | **$10/mo** + tokens |
| **Pricing (Business)** | N/A | $6-18/user | $50-500/mo | **$30/user** + tokens |
| **Pricing (Enterprise)** | N/A | Custom | Custom | **$100/user** + tokens |
| **Auto Peer Discovery** | ❌ | ✅ | ✅ | ✅ |
| **No Central Dependency** | ✅ | ❌ | ❌ | ⚠️ Bootstrap |
| **Throughput Target** | 1 Gbps | 1 Gbps | 500 Mbps | **6-7 Gbps** |
| **Token Economy** | ❌ | ❌ | ❌ | ✅ Chronara |
| **Open Source** | ✅ | ⚠️ Client | ⚠️ Client | ✅ Full |

*Free tiers have limitations

**ShadowMesh**: The only fully decentralized, quantum-safe VPN with blockchain-verified relay nodes and token-based economics.

---

**Next Steps**: Begin Kademlia DHT implementation (Sprint 0) to build foundation for zero-server peer discovery.

**Reference**: See [MIGRATION_PATH.md](MIGRATION_PATH.md) for 18-week roadmap to v1.0.0.
