# ShadowMesh Kademlia DHT Architecture Specification

**Version**: 1.0
**Date**: November 10, 2025
**Author**: Winston (Architect)
**Status**: Specification for v0.2.0-alpha (First Standalone Release)

---

## Executive Summary

This specification defines the Kademlia Distributed Hash Table (DHT) architecture for ShadowMesh v0.2.0-alpha, enabling **fully decentralized peer discovery** and eliminating the centralized discovery server dependency that currently blocks standalone operation.

**Key Outcomes**:
- **Zero central dependencies** - nodes operate autonomously from first boot
- **Cryptographically verifiable peer identity** using ML-DSA-87 public keys
- **Sub-second peer discovery** via DHT iterative lookup
- **Fault-tolerant routing** with automatic network partition recovery
- **Foundation for v1.0.0** production release

---

## Technology Stack & Versions

**Verified**: November 11, 2025

| Technology | Version | Purpose | Source |
|------------|---------|---------|--------|
| **Go** | 1.25.4 | Primary language | https://go.dev/dl/ |
| **cloudflare/circl** | v1.6.1 | Post-quantum cryptography (ML-KEM, ML-DSA) | https://github.com/cloudflare/circl |
| **golang.org/x/crypto** | v0.41.0 | ChaCha20-Poly1305, additional crypto primitives | https://pkg.go.dev/golang.org/x/crypto |
| **gorilla/websocket** | v1.5.3 | WebSocket Secure (WSS) transport | https://github.com/gorilla/websocket |
| **Protocol** | Kademlia DHT + chronara.eth | Hybrid peer/relay discovery | DHT: Standard, Smart Contract: Custom |
| **Transport** | WebSocket Secure (WSS) | Traffic obfuscation, censorship resistance | RFC 6455 + TLS 1.3 |
| **Transport Fallback** | UDP (Direct P2P) | Low-latency for direct connections | Standard |
| **Network Layer** | TAP Device | Virtual network interface (Layer 2, Ethernet frames) | Standard (Linux/macOS) |
| **Blockchain** | Ethereum + ENS | Relay node registry via chronara.eth | go-ethereum v1.14+ |
| **Database** | PostgreSQL | User/device management, connection history | PostgreSQL 14+ |
| **Monitoring** | Prometheus + Grafana | Metrics collection and visualization | Prometheus 2.45+, Grafana 10+ |

**Version Selection Rationale**:
- **Go 1.25.4**: Latest stable release (November 2025), includes security fixes and performance improvements
- **circl v1.6.1**: Latest stable release with NIST-standardized PQC algorithms (ML-KEM-1024, ML-DSA-87)
- **x/crypto v0.41.0**: Latest stable release with ChaCha20-Poly1305 optimizations

**Compatibility Notes**:
- Go 1.25+ required for improved generics support and crypto performance
- circl v1.6.1 provides NIST FIPS 203 (ML-KEM) and FIPS 204 (ML-DSA)
- All dependencies verified for arm64 and amd64 compatibility

---

## Current Problem

**v0.1.0-alpha (v11)** demonstrated excellent performance (28.3 Mbps, video streaming, 45% faster than Tailscale) but suffers from a critical architectural flaw:

```
❌ CENTRALIZED DISCOVERY SERVER
   • HTTP API: 209.151.148.121:8080
   • POST /register - Register peer
   • GET /peers - Retrieve peer list
   • NOW SHUT DOWN → Network inoperable
```

**Consequences**:
- Cannot operate standalone
- Single point of failure
- Contradicts DPN (Decentralized Private Network) vision
- Operational costs for infrastructure
- No censorship resistance

---

## Solution: Hybrid Architecture

ShadowMesh combines **multiple peer discovery mechanisms** for optimal reliability and decentralization:

### 1. Kademlia DHT (Primary Peer Discovery)

**Kademlia** is a proven distributed hash table protocol used by:
- **BitTorrent** - 30+ million concurrent nodes
- **IPFS** (libp2p) - Decentralized storage
- **Ethereum** - Node discovery layer

**Why Kademlia?**:
- ✅ Logarithmic lookup complexity: O(log N) hops for N nodes
- ✅ Symmetric architecture: Every node is equal (no super-nodes)
- ✅ Automatic routing table convergence via XOR distance metric
- ✅ Fault tolerance: Multiple redundant paths to any peer
- ✅ Built-in peer liveness monitoring

### 2. Ethereum Smart Contract Registry (Relay Node Discovery)

**chronara.eth** smart contract provides cryptographically verifiable relay node discovery:

**Why Smart Contract Registry?**:
- ✅ Public transparency: All relay nodes registered on-chain
- ✅ Stake-based trust: Operators stake 0.1 ETH (default) to register
- ✅ Automatic slashing: Malicious/offline nodes lose stake (Beta phase)
- ✅ Immutable audit trail: All registrations/deregistrations logged
- ✅ ENS human-readable: chronara.eth vs `0x...` address

**Relay Node Responsibilities**:
- Assist with CGNAT/NAT traversal for peers behind restrictive firewalls
- Route traffic through multi-hop paths (3-5 hops) for censorship resistance
- Provide fallback connectivity when direct P2P fails
- Maintain 99.9% uptime via 24-hour heartbeat transactions

**Discovery Priority**:
1. **Direct P2P** (fastest, <500ms timeout)
2. **DHT Lookup** (decentralized, <500ms)
3. **Smart Contract Relay** (fallback for CGNAT scenarios)

---

## Architecture Decision Summary

| Category | Decision | Version/Spec | Rationale |
|----------|----------|--------------|-----------|
| **Language** | Go | 1.25.4 | Performance, concurrency primitives, excellent crypto libraries, cross-platform |
| **DHT Protocol** | Kademlia | Standard | Proven at massive scale (BitTorrent 30M+ nodes, IPFS), O(log N) lookups |
| **PQC Key Exchange** | ML-KEM-1024 (Kyber) | NIST FIPS 203 | Quantum-safe key encapsulation, IND-CCA2 secure |
| **PQC Signatures** | ML-DSA-87 (Dilithium) | NIST FIPS 204 | Quantum-safe authentication, EUF-CMA secure |
| **Symmetric Crypto** | ChaCha20-Poly1305 | RFC 8439 | High-performance AEAD, 2.5x faster than AES-GCM on non-AES-NI CPUs |
| **PeerID Derivation** | SHA256(ML-DSA-87 pubkey) | SHA256 | Collision-resistant (2^128), verifiable identity, Sybil-resistant |
| **Routing Table** | 256 k-buckets, k=20 | Standard | Balances memory (5KB per bucket) vs performance |
| **Lookup Parallelism** | α=3 | Standard | Optimal latency/overhead trade-off for P2P networks |
| **Peer TTL** | 24 hours | Standard | Balances freshness vs network overhead |
| **Liveness Check** | PING every 15 min | Standard | Detects stale peers, 3 failures → eviction |
| **Transport** | WebSocket Secure (WSS) | RFC 6455 + TLS 1.3 | Traffic obfuscation (appears as HTTPS), censorship resistance, firewall traversal |
| **Transport Fallback** | UDP (direct P2P) | Standard | Low latency for direct connections, fallback when WSS unavailable |
| **Future Transport** | QUIC | RFC 9000 | Better NAT traversal via connection migration, stream multiplexing (post-MVP) |
| **Network Layer** | TAP Device (Layer 2) | Standard | Ethernet frame encryption, hides IP headers, prevents traffic analysis |
| **Bootstrap Nodes** | 3 nodes (US, EU, Asia) | Custom | Geographic redundancy, 99.9% uptime target |
| **PQC Library** | cloudflare/circl | v1.6.1 | Production-ready, NIST-standardized, actively maintained by Cloudflare |
| **Crypto Library** | golang.org/x/crypto | v0.41.0 | Official Go supplementary crypto, ChaCha20 optimizations |

**Key Design Principles**:
1. **Boring Technology** - Kademlia proven in production at scale (BitTorrent, IPFS)
2. **Quantum-Safe First** - PQC integrated from day one (ML-KEM + ML-DSA)
3. **Decentralization** - Zero central dependencies (DHT eliminates discovery server)
4. **Incremental Migration** - UDP first (v0.2.0), QUIC later (v0.3.0+) to reduce complexity
5. **Security by Design** - All messages signed (ML-DSA-87), PeerID verified, rate limiting

---

## Architecture Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                      ShadowMesh v0.2.0-alpha                        │
│                  (Kademlia DHT + PQC + UDP/QUIC)                    │
└────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────┐
│  Layer 1: Application                                               │
│  • CLI commands: connect, disconnect, status                        │
│  • Configuration: bootstrap nodes, network settings                 │
└────────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────────┐
│  Layer 2: Kademlia DHT (Peer Discovery)                            │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  PeerID Generation                                            │  │
│  │  • SHA256(ML-DSA-87 Public Key) → 256-bit PeerID            │  │
│  │  • Cryptographically verifiable identity                     │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Routing Table (160 k-buckets, k=20)                         │  │
│  │  • XOR distance metric for peer organization                 │  │
│  │  • LRU eviction policy per k-bucket                          │  │
│  │  • Automatic table refresh every 1 hour                      │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  DHT Operations                                               │  │
│  │  • FIND_NODE(target) → Iterative lookup (α=3 parallel)      │  │
│  │  • STORE(key, value, TTL=24h) → Store peer metadata         │  │
│  │  • FIND_VALUE(key) → Retrieve peer metadata                 │  │
│  │  • PING → Peer liveness check (every 15 min)                │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────────┐
│  Layer 3: Post-Quantum Cryptography                                │
│  • ML-KEM-1024 (Kyber) - Key Exchange                             │
│  • ML-DSA-87 (Dilithium) - Peer Authentication & Signatures       │
│  • ChaCha20-Poly1305 - Symmetric Encryption                       │
│  • Hybrid Mode: Classical (X25519, Ed25519) + PQC                 │
└────────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────────┐
│  Layer 4: Transport (WebSocket Secure + UDP Fallback)             │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Primary: WebSocket Secure (WSS/TLS 1.3)                     │  │
│  │  • Appears as HTTPS traffic (port 443)                       │  │
│  │  • Defeats deep packet inspection (DPI)                      │  │
│  │  • Firewall traversal (works on restricted networks)         │  │
│  │  • Packet size/timing randomization                          │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Fallback: UDP (Direct P2P)                                  │  │
│  │  • Low latency for direct peer connections                   │  │
│  │  • Used when both peers not behind restrictive NAT           │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Future: QUIC (Post-MVP)                                     │  │
│  │  • Stream multiplexing, connection migration                 │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────────┐
│  Layer 5: Networking (Layer 2 - TAP Device)                        │
│  • Virtual network interface (TAP - captures Ethernet frames)      │
│  • Raw Ethernet frame encryption (hides IP headers)                │
│  • Prevents traffic analysis via header inspection                 │
│  • Addresses: 10.10.x.x/16 (assigned via DHT peer discovery)       │
└────────────────────────────────────────────────────────────────────┘
```

---

## Ethereum Smart Contract: chronara.eth Relay Registry

### Overview

The **chronara.eth** smart contract deployed on Ethereum mainnet maintains a verifiable registry of all relay nodes with stake-based trust and automatic slashing for misbehavior.

**Contract Address**: Deployed via ENS (chronara.eth resolves to contract address)

### Smart Contract Architecture

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract RelayNodeRegistry {
    // Relay node metadata
    struct RelayNode {
        address operator;           // Ethereum address of operator
        bytes32 publicKeyHash;      // SHA256(ML-DSA-87 public key)
        string ipAddress;           // IPv4/IPv6 address (obfuscated to city-level)
        uint16 port;                // Port number
        uint256 stakeAmount;        // ETH staked (wei)
        uint256 registrationTime;   // Block timestamp
        uint256 lastHeartbeat;      // Last heartbeat timestamp
        string geolocation;         // City, Country (e.g., "New York, USA")
        bool isActive;              // Active status
        uint256 uptimePercentage;   // Uptime percentage (0-10000 = 0%-100.00%)
    }

    // Minimum stake required to register
    uint256 public constant MIN_STAKE = 0.1 ether;

    // Maximum time between heartbeats (24 hours)
    uint256 public constant HEARTBEAT_INTERVAL = 24 hours;

    // Slashing amount for missed heartbeats (Beta feature)
    uint256 public constant SLASH_AMOUNT = 0.01 ether;

    // Mapping: publicKeyHash => RelayNode
    mapping(bytes32 => RelayNode) public relayNodes;

    // Array of active node hashes for enumeration
    bytes32[] public activeNodeHashes;

    // Events
    event NodeRegistered(bytes32 indexed publicKeyHash, address indexed operator, string geolocation, uint256 stakeAmount);
    event NodeDeregistered(bytes32 indexed publicKeyHash, address indexed operator);
    event HeartbeatReceived(bytes32 indexed publicKeyHash, uint256 timestamp);
    event NodeSlashed(bytes32 indexed publicKeyHash, uint256 slashAmount);

    // Register new relay node
    function registerNode(
        bytes32 publicKeyHash,
        string memory ipAddress,
        uint16 port,
        string memory geolocation
    ) external payable {
        require(msg.value >= MIN_STAKE, "Insufficient stake");
        require(!relayNodes[publicKeyHash].isActive, "Node already registered");

        relayNodes[publicKeyHash] = RelayNode({
            operator: msg.sender,
            publicKeyHash: publicKeyHash,
            ipAddress: ipAddress,
            port: port,
            stakeAmount: msg.value,
            registrationTime: block.timestamp,
            lastHeartbeat: block.timestamp,
            geolocation: geolocation,
            isActive: true,
            uptimePercentage: 10000  // Start at 100%
        });

        activeNodeHashes.push(publicKeyHash);

        emit NodeRegistered(publicKeyHash, msg.sender, geolocation, msg.value);
    }

    // Submit heartbeat to maintain active status
    function submitHeartbeat(bytes32 publicKeyHash) external {
        RelayNode storage node = relayNodes[publicKeyHash];
        require(node.isActive, "Node not active");
        require(msg.sender == node.operator, "Not authorized");

        node.lastHeartbeat = block.timestamp;

        emit HeartbeatReceived(publicKeyHash, block.timestamp);
    }

    // Deregister node and return stake
    function deregisterNode(bytes32 publicKeyHash) external {
        RelayNode storage node = relayNodes[publicKeyHash];
        require(node.isActive, "Node not active");
        require(msg.sender == node.operator, "Not authorized");

        node.isActive = false;
        uint256 refundAmount = node.stakeAmount;
        node.stakeAmount = 0;

        // Remove from active array
        _removeFromActiveNodes(publicKeyHash);

        // Refund stake
        payable(msg.sender).transfer(refundAmount);

        emit NodeDeregistered(publicKeyHash, msg.sender);
    }

    // Query all active relay nodes (for client discovery)
    function getActiveNodes() external view returns (RelayNode[] memory) {
        uint256 count = 0;
        for (uint256 i = 0; i < activeNodeHashes.length; i++) {
            if (relayNodes[activeNodeHashes[i]].isActive) {
                count++;
            }
        }

        RelayNode[] memory nodes = new RelayNode[](count);
        uint256 index = 0;
        for (uint256 i = 0; i < activeNodeHashes.length; i++) {
            bytes32 hash = activeNodeHashes[i];
            if (relayNodes[hash].isActive) {
                nodes[index] = relayNodes[hash];
                index++;
            }
        }

        return nodes;
    }

    // Check node health (missed heartbeat)
    function checkNodeHealth(bytes32 publicKeyHash) external view returns (bool) {
        RelayNode storage node = relayNodes[publicKeyHash];
        if (!node.isActive) return false;

        return (block.timestamp - node.lastHeartbeat) <= HEARTBEAT_INTERVAL;
    }

    // Slash node for missed heartbeat (Beta feature - callable by anyone)
    function slashNode(bytes32 publicKeyHash) external {
        RelayNode storage node = relayNodes[publicKeyHash];
        require(node.isActive, "Node not active");
        require((block.timestamp - node.lastHeartbeat) > HEARTBEAT_INTERVAL, "Heartbeat not missed");
        require(node.stakeAmount >= SLASH_AMOUNT, "Insufficient stake to slash");

        node.stakeAmount -= SLASH_AMOUNT;

        // Transfer slashed amount to caller as incentive
        payable(msg.sender).transfer(SLASH_AMOUNT);

        emit NodeSlashed(publicKeyHash, SLASH_AMOUNT);
    }

    // Internal: Remove node from active array
    function _removeFromActiveNodes(bytes32 publicKeyHash) private {
        for (uint256 i = 0; i < activeNodeHashes.length; i++) {
            if (activeNodeHashes[i] == publicKeyHash) {
                activeNodeHashes[i] = activeNodeHashes[activeNodeHashes.length - 1];
                activeNodeHashes.pop();
                break;
            }
        }
    }
}
```

### Contract Integration (Go Client)

```go
// pkg/blockchain/contract.go
import (
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/ethereum/go-ethereum/accounts/abi/bind"
    "github.com/ethereum/go-ethereum/common"
    "github.com/wealdtech/go-ens/v3"
)

type RelayRegistry struct {
    client       *ethclient.Client
    contract     *RelayNodeRegistry  // Generated bindings
    ensResolver  *ens.Registry
}

func NewRelayRegistry(rpcURL string) (*RelayRegistry, error) {
    // Connect to Ethereum via Infura/Alchemy
    client, err := ethclient.Dial(rpcURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to Ethereum: %w", err)
    }

    // Resolve chronara.eth ENS name to contract address
    ensRegistry, err := ens.NewRegistry(client)
    if err != nil {
        return nil, fmt.Errorf("failed to create ENS registry: %w", err)
    }

    contractAddress, err := ensRegistry.ResolveAddress("chronara.eth")
    if err != nil {
        return nil, fmt.Errorf("failed to resolve chronara.eth: %w", err)
    }

    // Load contract
    contract, err := NewRelayNodeRegistry(contractAddress, client)
    if err != nil {
        return nil, fmt.Errorf("failed to load contract: %w", err)
    }

    return &RelayRegistry{
        client:      client,
        contract:    contract,
        ensResolver: ensRegistry,
    }, nil
}

// Query active relay nodes
func (r *RelayRegistry) GetActiveRelayNodes(ctx context.Context) ([]RelayNodeInfo, error) {
    // Call smart contract
    nodes, err := r.contract.GetActiveNodes(&bind.CallOpts{Context: ctx})
    if err != nil {
        return nil, fmt.Errorf("contract query failed: %w", err)
    }

    // Convert to internal format
    relayNodes := make([]RelayNodeInfo, 0, len(nodes))
    for _, node := range nodes {
        // Verify heartbeat is recent
        if time.Since(time.Unix(node.LastHeartbeat.Int64(), 0)) > 24*time.Hour {
            continue  // Skip stale nodes
        }

        relayNodes = append(relayNodes, RelayNodeInfo{
            PublicKeyHash: node.PublicKeyHash,
            Address:       fmt.Sprintf("%s:%d", node.IpAddress, node.Port),
            Geolocation:   node.Geolocation,
            StakeAmount:   node.StakeAmount,
            UptimePercent: float64(node.UptimePercentage.Uint64()) / 100.0,
        })
    }

    return relayNodes, nil
}

// Register relay node (operator only)
func (r *RelayRegistry) RegisterNode(
    privateKey *ecdsa.PrivateKey,
    publicKeyHash [32]byte,
    ipAddress string,
    port uint16,
    geolocation string,
    stakeETH float64,
) error {
    auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1)) // Mainnet
    if err != nil {
        return fmt.Errorf("failed to create transactor: %w", err)
    }

    // Set stake amount
    stakeWei := new(big.Int)
    stakeWei.SetString(fmt.Sprintf("%.0f", stakeETH*1e18), 10)
    auth.Value = stakeWei

    // Submit transaction
    tx, err := r.contract.RegisterNode(auth, publicKeyHash, ipAddress, port, geolocation)
    if err != nil {
        return fmt.Errorf("registration transaction failed: %w", err)
    }

    log.Printf("[BLOCKCHAIN] Node registered: tx=%s", tx.Hash().Hex())
    return nil
}

// Submit heartbeat (operator only)
func (r *RelayRegistry) SubmitHeartbeat(privateKey *ecdsa.PrivateKey, publicKeyHash [32]byte) error {
    auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1))
    if err != nil {
        return fmt.Errorf("failed to create transactor: %w", err)
    }

    tx, err := r.contract.SubmitHeartbeat(auth, publicKeyHash)
    if err != nil {
        return fmt.Errorf("heartbeat transaction failed: %w", err)
    }

    log.Printf("[BLOCKCHAIN] Heartbeat submitted: tx=%s", tx.Hash().Hex())
    return nil
}
```

### Gas Cost Analysis

**Registration Transaction**:
- Contract call gas: ~150,000 gas
- Storage writes: ~40,000 gas (new struct)
- Event emission: ~5,000 gas
- **Total**: ~195,000 gas

**Cost at $3,000 ETH, 25 gwei**:
- 195,000 × 25 gwei = 4,875,000 gwei = 0.004875 ETH
- 0.004875 ETH × $3,000 = **$14.63 USD**

**Optimization**: Target <150,000 gas (under $10 USD)

**Heartbeat Transaction**:
- Simple storage update: ~25,000 gas
- Cost: ~$1.88 USD per 24 hours

### Caching Strategy

To avoid excessive RPC calls and gas costs:

```go
// Client-side caching
type RelayNodeCache struct {
    nodes      []RelayNodeInfo
    lastUpdate time.Time
    ttl        time.Duration
    mutex      sync.RWMutex
}

func (c *RelayNodeCache) GetNodes(ctx context.Context, registry *RelayRegistry) ([]RelayNodeInfo, error) {
    c.mutex.RLock()
    if time.Since(c.lastUpdate) < c.ttl {
        defer c.mutex.RUnlock()
        return c.nodes, nil  // Return cached nodes
    }
    c.mutex.RUnlock()

    // Fetch fresh data
    c.mutex.Lock()
    defer c.mutex.Unlock()

    nodes, err := registry.GetActiveRelayNodes(ctx)
    if err != nil {
        // Return stale cache on error
        if c.nodes != nil {
            log.Printf("[BLOCKCHAIN] Using stale cache due to error: %v", err)
            return c.nodes, nil
        }
        return nil, err
    }

    c.nodes = nodes
    c.lastUpdate = time.Now()
    return nodes, nil
}
```

**Cache Configuration**:
- TTL: 10 minutes (balance freshness vs RPC load)
- Fallback: Use stale cache if RPC fails
- Background refresh: Update cache every 5 minutes in goroutine

---

## WebSocket Secure (WSS) Transport Layer

### Overview

ShadowMesh uses **WebSocket Secure (WSS) over TLS 1.3** as the primary transport protocol to defeat deep packet inspection (DPI) and enable operation in censored networks.

**Key Benefits**:
- Appears identical to normal HTTPS web traffic (port 443)
- Bypasses firewall restrictions (web traffic rarely blocked)
- Defeats China's Great Firewall DPI systems
- Works through corporate proxies and restrictive NAT
- TLS 1.3 provides forward secrecy and modern cryptography

### Traffic Obfuscation Techniques

**1. HTTPS Mimicry**:
```
Normal Web Traffic (HTTPS):
- Port 443
- TLS 1.3 handshake
- HTTP/1.1 Upgrade: websocket
- Binary WebSocket frames

ShadowMesh Traffic (WSS):
- Port 443 ✓
- TLS 1.3 handshake ✓
- HTTP/1.1 Upgrade: websocket ✓
- Binary WebSocket frames ✓

→ Indistinguishable from legitimate web traffic
```

**2. Packet Size Randomization**:
```go
// pkg/transport/websocket/obfuscation.go
type Obfuscator struct {
    minPaddingBytes int  // Default: 0
    maxPaddingBytes int  // Default: 256
    rng             *rand.Rand
}

func (o *Obfuscator) ObfuscatePacket(payload []byte) []byte {
    // Add random padding to prevent statistical fingerprinting
    paddingSize := o.rng.Intn(o.maxPaddingBytes - o.minPaddingBytes + 1) + o.minPaddingBytes

    packet := make([]byte, len(payload)+paddingSize+2)

    // Format: [original_length:2][payload][random_padding]
    binary.BigEndian.PutUint16(packet[0:2], uint16(len(payload)))
    copy(packet[2:], payload)
    o.rng.Read(packet[2+len(payload):])  // Random padding

    return packet
}

func (o *Obfuscator) DeobfuscatePacket(packet []byte) ([]byte, error) {
    if len(packet) < 2 {
        return nil, errors.New("packet too short")
    }

    originalLength := binary.BigEndian.Uint16(packet[0:2])
    if int(originalLength)+2 > len(packet) {
        return nil, errors.New("invalid packet length")
    }

    return packet[2 : 2+originalLength], nil
}
```

**3. Timing Randomization (Jitter)**:
```go
// Add minimal jitter to disrupt timing analysis
func (c *WSSConnection) SendWithJitter(data []byte) error {
    // Add 0-5ms random delay
    jitter := time.Duration(rand.Intn(5)) * time.Millisecond
    time.Sleep(jitter)

    return c.conn.WriteMessage(websocket.BinaryMessage, data)
}
```

**Trade-off**: Jitter adds <5ms latency (acceptable for NFR2: <5ms target) but disrupts timing-based fingerprinting.

### WebSocket Server Architecture

**Server Implementation (Relay Nodes & Bootstrap Nodes)**:
```go
// pkg/transport/websocket/server.go
import (
    "github.com/gorilla/websocket"
    "crypto/tls"
)

type WSSServer struct {
    addr        string
    tlsCert     tls.Certificate
    upgrader    websocket.Upgrader
    connections map[PeerID]*WSSConnection
    mutex       sync.RWMutex
}

func NewWSSServer(addr string, certFile, keyFile string) (*WSSServer, error) {
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
    }

    return &WSSServer{
        addr:     addr,
        tlsCert:  cert,
        upgrader: websocket.Upgrader{
            ReadBufferSize:  32768,  // 32 KB
            WriteBufferSize: 32768,
            CheckOrigin: func(r *http.Request) bool {
                return true  // Allow all origins (P2P network)
            },
        },
        connections: make(map[PeerID]*WSSConnection),
    }, nil
}

func (s *WSSServer) Start() error {
    // Configure TLS 1.3
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{s.tlsCert},
        MinVersion:   tls.VersionTLS13,  // Force TLS 1.3
        CipherSuites: []uint16{
            tls.TLS_AES_128_GCM_SHA256,      // Fast on modern CPUs
            tls.TLS_CHACHA20_POLY1305_SHA256, // Fast on non-AES-NI CPUs
        },
    }

    // HTTP server with TLS
    mux := http.NewServeMux()
    mux.HandleFunc("/ws", s.handleWebSocket)

    server := &http.Server{
        Addr:      s.addr,
        Handler:   mux,
        TLSConfig: tlsConfig,
        ReadHeaderTimeout: 10 * time.Second,
    }

    log.Printf("[WSS] Server starting on %s", s.addr)
    return server.ListenAndServeTLS("", "")  // Cert already in TLSConfig
}

func (s *WSSServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Upgrade HTTP connection to WebSocket
    conn, err := s.upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("[WSS] Upgrade failed: %v", err)
        return
    }

    // Wrap in WSSConnection
    wssConn := NewWSSConnection(conn)

    // Perform PQC handshake
    peerID, err := s.performHandshake(wssConn)
    if err != nil {
        log.Printf("[WSS] Handshake failed: %v", err)
        conn.Close()
        return
    }

    // Register connection
    s.mutex.Lock()
    s.connections[peerID] = wssConn
    s.mutex.Unlock()

    log.Printf("[WSS] Peer connected: %s from %s", peerID.Short(), r.RemoteAddr)

    // Handle messages
    s.handleConnection(peerID, wssConn)
}
```

### WebSocket Client (Direct Peer Connection)

**Client Implementation**:
```go
// pkg/transport/websocket/client.go
type WSSClient struct {
    obfuscator *Obfuscator
}

func (c *WSSClient) ConnectToRelay(relayAddr string) (*WSSConnection, error) {
    // Dial WSS connection
    dialer := websocket.Dialer{
        TLSClientConfig: &tls.Config{
            MinVersion: tls.VersionTLS13,
        },
        HandshakeTimeout: 10 * time.Second,
    }

    url := fmt.Sprintf("wss://%s/ws", relayAddr)
    conn, _, err := dialer.Dial(url, nil)
    if err != nil {
        return nil, fmt.Errorf("WSS dial failed: %w", err)
    }

    wssConn := NewWSSConnection(conn)
    wssConn.obfuscator = c.obfuscator

    log.Printf("[WSS] Connected to relay: %s", relayAddr)
    return wssConn, nil
}
```

### Performance Considerations

**Throughput**:
- WebSocket frame overhead: ~2-14 bytes per message
- TLS 1.3 overhead: ~40 bytes per record
- **Target**: Maintain 1+ Gbps (NFR1) despite overhead

**Latency**:
- TLS 1.3 handshake: ~50-100ms (1-RTT)
- WebSocket upgrade: ~50ms
- Per-message overhead: <1ms
- Jitter: 0-5ms
- **Target**: <5ms added latency (NFR2) for established connections

**Optimization**:
- Use binary WebSocket frames (not text)
- Disable compression (incompressible encrypted data)
- Connection pooling for relay nodes
- Keep-alive pings every 30 seconds

### NAT Traversal via WSS

**Challenge**: CGNAT and symmetric NAT block direct UDP connections

**Solution**: WSS works through most NAT types:
1. Client → NAT → Relay (port 443, looks like HTTPS)
2. NAT typically allows outbound HTTPS
3. Relay maintains bidirectional WSS connection
4. No NAT hole-punching required

**Multi-Hop Routing** (3-5 hops for anonymity):
```
Client → Relay1 (WSS) → Relay2 (WSS) → Relay3 (WSS) → Target
```

Each hop sees only encrypted WebSocket frames, cannot determine:
- Original source IP
- Final destination IP
- Intermediate path

---

## Core Component: Kademlia DHT

### 1. PeerID Generation

**Design Decision**: Derive PeerID from post-quantum signature public key

```go
// PeerID generation from ML-DSA-87 public key
func GeneratePeerID(mldsaPublicKey []byte) PeerID {
    hash := sha256.Sum256(mldsaPublicKey)
    return PeerID(hash[:]) // 256-bit PeerID
}

// PeerID structure
type PeerID [32]byte // 256 bits (32 bytes)

// Verification: Peer proves ownership by signing challenge
func VerifyPeerOwnership(peerID PeerID, publicKey []byte, signature []byte, challenge []byte) bool {
    // 1. Verify PeerID matches public key
    derivedID := GeneratePeerID(publicKey)
    if derivedID != peerID {
        return false
    }

    // 2. Verify signature on challenge
    return VerifyMLDSA87Signature(publicKey, challenge, signature)
}
```

**Properties**:
- ✅ **Cryptographically verifiable**: Peer must prove ownership via signature
- ✅ **Collision-resistant**: SHA256 provides 2^128 security against birthday attacks
- ✅ **Quantum-safe**: Derived from post-quantum public key
- ✅ **Sybil-resistant**: Creating fake peers requires valid ML-DSA-87 keypairs

**Rationale**: Using the PQ signature key (not encryption key) allows peers to prove identity through signed challenges, preventing identity spoofing.

---

### 2. Routing Table Structure

**Kademlia k-bucket design** with XOR distance metric:

```
Routing Table:
┌─────────────────────────────────────────────────────────────┐
│ k-bucket[0]:   Distance [2^255, 2^256)  - Furthest peers   │
│ k-bucket[1]:   Distance [2^254, 2^255)                      │
│ k-bucket[2]:   Distance [2^253, 2^254)                      │
│ ...                                                          │
│ k-bucket[254]: Distance [2^1, 2^2)                          │
│ k-bucket[255]: Distance [2^0, 2^1)      - Closest peers    │
└─────────────────────────────────────────────────────────────┘

Each k-bucket stores up to k=20 peers (configurable)
```

**Implementation**:

```go
type RoutingTable struct {
    localPeerID  PeerID
    kBuckets     [256]*KBucket  // 256 buckets for 256-bit PeerID space
    k            int            // Max peers per bucket (default: 20)
    mutex        sync.RWMutex
}

type KBucket struct {
    peers        []PeerInfo     // Up to k peers
    lastUpdated  time.Time
    mutex        sync.RWMutex
}

type PeerInfo struct {
    PeerID       PeerID
    Address      string         // "IP:port"
    PublicKey    []byte         // ML-DSA-87 public key
    LastSeen     time.Time
    Capabilities []string       // ["relay", "exit-node", "bootstrap"]
}

// XOR distance metric
func XORDistance(id1, id2 PeerID) *big.Int {
    distance := new(big.Int)
    distance.SetBytes(id1[:])
    distance.Xor(distance, new(big.Int).SetBytes(id2[:]))
    return distance
}

// Find k-bucket index for peer
func (rt *RoutingTable) BucketIndex(peerID PeerID) int {
    distance := XORDistance(rt.localPeerID, peerID)
    // Count leading zeros to determine bucket
    leadingZeros := distance.BitLen()
    return 256 - leadingZeros
}
```

**Routing Table Maintenance**:
- **LRU Eviction**: When bucket full, replace least-recently-seen peer
- **Liveness Checks**: PING all peers every 15 minutes
- **Table Refresh**: Re-query all buckets every 1 hour
- **Stale Peer Removal**: Evict peers that fail 3 consecutive PINGs

---

### 3. DHT Operations

#### 3.1. FIND_NODE - Iterative Lookup

**Goal**: Find k closest peers to target PeerID

**Algorithm** (α=3 parallel requests, k=20 results):

```
1. Start with k closest known peers to target
2. Send FIND_NODE(target) to α peers in parallel
3. Each peer responds with k closest peers from their routing table
4. Add new peers to candidate list
5. Mark queried peers as "queried"
6. Select α closest unqueried peers from candidate list
7. Repeat until:
   - No closer peers found, OR
   - Top k peers all queried, OR
   - Timeout (5 seconds)
8. Return k closest peers found
```

**Implementation**:

```go
func (dht *DHT) FindNode(target PeerID, timeout time.Duration) []PeerInfo {
    // Iterative lookup with α=3 concurrent requests
    α := 3
    k := 20

    // Start with closest known peers
    candidates := dht.routingTable.FindClosest(target, k)
    queried := make(map[PeerID]bool)
    results := make([]PeerInfo, 0, k)

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    for {
        // Select α closest unqueried peers
        toQuery := selectClosestUnqueried(candidates, queried, α, target)
        if len(toQuery) == 0 {
            break // No more peers to query
        }

        // Query in parallel
        responses := queryPeersInParallel(ctx, toQuery, target)

        // Merge responses into candidate list
        for _, response := range responses {
            candidates = mergePeers(candidates, response.Peers)
            queried[response.FromPeer] = true
        }

        // Check termination conditions
        if hasConverged(results, candidates, k) {
            break
        }
    }

    // Return k closest peers
    return selectClosest(candidates, target, k)
}
```

**Performance**:
- **Latency**: O(log N) hops for N nodes
- **Target**: <500ms for 100,000-node network
- **Parallelism**: α=3 concurrent requests reduces latency

#### 3.2. STORE - Store Peer Metadata

**Goal**: Store peer metadata (IP, port, public key) in DHT

**Strategy**: Store at k closest nodes to peer's own PeerID

```go
func (dht *DHT) StoreSelf() error {
    // Find k closest nodes to own PeerID
    closestPeers := dht.FindNode(dht.localPeerID, 5*time.Second)

    // Build metadata record
    metadata := PeerMetadata{
        PeerID:       dht.localPeerID,
        Address:      dht.localAddress,
        PublicKey:    dht.mldsaPublicKey,
        Capabilities: []string{"peer"},
        TTL:          24 * time.Hour,
        Timestamp:    time.Now(),
        Signature:    dht.SignMetadata(), // Prove ownership
    }

    // Send STORE to k closest peers
    successCount := 0
    for _, peer := range closestPeers {
        if dht.sendStore(peer, metadata) {
            successCount++
        }
    }

    // Success if majority stored
    return successCount >= len(closestPeers)/2
}
```

**Data Structure**:

```go
type PeerMetadata struct {
    PeerID       PeerID
    Address      string         // "IP:port"
    PublicKey    []byte         // ML-DSA-87 public key
    Capabilities []string       // ["peer", "relay", "exit-node", "bootstrap"]
    TTL          time.Duration  // 24 hours default
    Timestamp    time.Time
    Signature    []byte         // ML-DSA-87 signature over above fields
}
```

**Validation**:
- Verify PeerID matches public key: `GeneratePeerID(PublicKey) == PeerID`
- Verify signature over metadata fields
- Reject if timestamp too old (>5 minutes drift)
- Reject if TTL exceeds maximum (48 hours)

#### 3.3. FIND_VALUE - Retrieve Peer Metadata

**Goal**: Retrieve metadata for target PeerID

**Algorithm**: Similar to FIND_NODE but returns metadata if found

```go
func (dht *DHT) FindValue(target PeerID) (*PeerMetadata, error) {
    // Check local cache first
    if cached := dht.cache.Get(target); cached != nil {
        return cached, nil
    }

    // Iterative lookup with α=3
    candidates := dht.routingTable.FindClosest(target, 20)

    for _, peer := range candidates {
        response := dht.sendFindValue(peer, target)
        if response.Found {
            // Validate and cache metadata
            if validateMetadata(response.Metadata) {
                dht.cache.Set(target, response.Metadata, 10*time.Minute)
                return response.Metadata, nil
            }
        }
    }

    return nil, ErrPeerNotFound
}
```

**Caching Strategy**:
- Cache successful lookups for 10 minutes
- LRU eviction (max 10,000 entries)
- Invalidate on failed connection attempts

#### 3.4. PING - Liveness Check

**Goal**: Verify peer is still online

```go
func (dht *DHT) Ping(peer PeerInfo) bool {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    // Send PING message
    response, err := dht.sendPing(ctx, peer)
    if err != nil {
        return false
    }

    // Update last seen time
    dht.routingTable.UpdateLastSeen(peer.PeerID, time.Now())
    return response.Success
}

// Background liveness checker
func (dht *DHT) startLivenessChecker() {
    ticker := time.NewTicker(15 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        peers := dht.routingTable.AllPeers()
        for _, peer := range peers {
            go func(p PeerInfo) {
                if !dht.Ping(p) {
                    dht.handleFailedPing(p)
                }
            }(peer)
        }
    }
}

func (dht *DHT) handleFailedPing(peer PeerInfo) {
    failCount := dht.incrementFailCount(peer.PeerID)
    if failCount >= 3 {
        dht.routingTable.RemovePeer(peer.PeerID)
        log.Printf("Removed dead peer: %s", peer.PeerID)
    }
}
```

---

### 4. Bootstrap Process

**Challenge**: New node joining network needs initial peers

**Solution**: Hardcoded bootstrap nodes + iterative expansion

```
┌────────────────────────────────────────────────────────────┐
│  Bootstrap Flow                                             │
└────────────────────────────────────────────────────────────┘

1. Node starts with empty routing table
   ↓
2. Connect to 3-5 hardcoded bootstrap nodes
   Bootstrap nodes: Known long-running peers
   ↓
3. FIND_NODE(self) to bootstrap nodes
   ↓
4. Receive k peers close to own PeerID
   ↓
5. Add peers to routing table
   ↓
6. FIND_NODE(random IDs) to populate all buckets
   ↓
7. STORE(self) at k closest peers
   ↓
8. Routing table converges (target: <60 seconds)
   ↓
9. Node fully integrated into DHT
```

**Bootstrap Node Configuration**:

```yaml
# config.yaml
bootstrap_nodes:
  - peer_id: "2f8a3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b"
    address: "bootstrap1.shadowmesh.net:9443"
    public_key: "base64_encoded_mldsa87_pubkey"

  - peer_id: "3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b"
    address: "bootstrap2.shadowmesh.net:9443"
    public_key: "base64_encoded_mldsa87_pubkey"

  - peer_id: "4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5"
    address: "bootstrap3.shadowmesh.net:9443"
    public_key: "base64_encoded_mldsa87_pubkey"
```

**Bootstrap Node Requirements**:
- 99.9% uptime (3 nodes provide redundancy)
- Static IP addresses with DNS names
- Pre-registered ML-DSA-87 keypairs
- Deployed across geographic regions (US, EU, Asia)

**Graceful Degradation**:
- If 0 bootstrap nodes reachable → Error: "Cannot connect to network"
- If 1-2 bootstrap nodes reachable → Warning: "Degraded connectivity"
- If 3+ bootstrap nodes reachable → Normal operation

**Future**: Peer exchange protocol allows nodes to bootstrap from any known peer, reducing bootstrap node dependency.

---

### 5. DHT Message Protocol

**Wire Protocol**: Binary messages over UDP (v0.2.0) or QUIC (v0.3.0+)

```go
// Message types
const (
    MSG_PING        = 0x01
    MSG_PONG        = 0x02
    MSG_FIND_NODE   = 0x03
    MSG_FOUND_NODES = 0x04
    MSG_STORE       = 0x05
    MSG_STORE_ACK   = 0x06
    MSG_FIND_VALUE  = 0x07
    MSG_FOUND_VALUE = 0x08
)

// Message structure (simplified)
type DHTMessage struct {
    Type      uint8      // Message type
    RequestID [16]byte   // Random nonce for request/response matching
    SenderID  PeerID     // Sender's PeerID
    Payload   []byte     // Type-specific payload
    Signature []byte     // ML-DSA-87 signature over above fields
}

// FIND_NODE payload
type FindNodePayload struct {
    TargetID PeerID
}

// FOUND_NODES payload
type FoundNodesPayload struct {
    Peers []PeerInfo // Up to k=20 peers
}

// STORE payload
type StorePayload struct {
    Metadata PeerMetadata
}
```

**Security**:
- All messages signed with ML-DSA-87
- Verify signature before processing
- Rate limiting: Max 100 messages/second per peer
- Reject messages with invalid PeerID → PublicKey mapping

---

## Integration with Existing Components

### 1. PQC Handshake Flow

**After DHT peer discovery, establish encrypted tunnel:**

```
┌────────────────────────────────────────────────────────────┐
│  Connection Establishment Flow                              │
└────────────────────────────────────────────────────────────┘

1. DHT Lookup: FIND_VALUE(target_peer_id)
   ↓
2. Retrieve: IP address, port, ML-DSA-87 public key
   ↓
3. Initiate UDP connection to peer
   ↓
4. PQC Handshake:
   a. Generate ephemeral ML-KEM-1024 keypair
   b. Send key encapsulation request
   c. Peer encapsulates shared secret
   d. Both derive symmetric key via HKDF
   ↓
5. Authenticate with ML-DSA-87 signatures
   ↓
6. Begin encrypted traffic with ChaCha20-Poly1305
```

**Code Integration**:

```go
// Combine DHT + PQC
func (node *Node) ConnectToPeer(targetPeerID PeerID) error {
    // 1. DHT lookup
    metadata, err := node.dht.FindValue(targetPeerID)
    if err != nil {
        return fmt.Errorf("peer not found in DHT: %w", err)
    }

    // 2. Verify peer identity
    if !node.crypto.VerifyPeerID(metadata.PeerID, metadata.PublicKey) {
        return errors.New("PeerID mismatch")
    }

    // 3. Establish UDP connection
    conn, err := net.Dial("udp", metadata.Address)
    if err != nil {
        return fmt.Errorf("connection failed: %w", err)
    }

    // 4. PQC handshake
    sharedSecret, err := node.crypto.PerformMLKEMHandshake(conn, metadata.PublicKey)
    if err != nil {
        return fmt.Errorf("handshake failed: %w", err)
    }

    // 5. Derive symmetric key
    sessionKey := node.crypto.DeriveSessionKey(sharedSecret)

    // 6. Create encrypted tunnel
    tunnel := NewEncryptedTunnel(conn, sessionKey)
    node.addTunnel(targetPeerID, tunnel)

    log.Printf("Connected to peer: %s", targetPeerID)
    return nil
}
```

### 2. TAP Device Integration (Layer 2 Networking)

**Why TAP vs TUN**:
- **TAP (Layer 2)**: Captures raw Ethernet frames, hides IP headers in transit
- **TUN (Layer 3)**: Captures IP packets, exposes IP headers
- **Security**: TAP prevents traffic analysis via IP header inspection

**Traffic routing flow:**

```
Application → TAP Device → ShadowMesh Daemon → DHT Lookup → Encrypted Ethernet Frame → WSS/UDP → Remote Peer
```

**TAP Device Benefits**:
- IP headers encrypted inside Ethernet frame
- Prevents DPI from analyzing destination IPs
- Supports ARP, IPv6 NDP, and other Layer 2 protocols
- More complex but significantly more secure

**Implementation**:

```go
// pkg/tap/device.go
import "github.com/songgao/water"

type TAPDevice struct {
    iface      *water.Interface
    mtu        int
    macAddress net.HardwareAddr
}

func NewTAPDevice(name string) (*TAPDevice, error) {
    config := water.Config{
        DeviceType: water.TAP,  // Layer 2 (Ethernet frames)
    }
    config.Name = name

    iface, err := water.New(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create TAP device: %w", err)
    }

    return &TAPDevice{
        iface: iface,
        mtu:   1500,
        macAddress: generateRandomMAC(),
    }, nil
}

func (d *TAPDevice) ConfigureInterface(ipAddr string, subnet string) error {
    // Configure IP address (Layer 3 over Layer 2)
    cmd := exec.Command("ip", "addr", "add", fmt.Sprintf("%s/%s", ipAddr, subnet), "dev", d.iface.Name())
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to set IP: %w", err)
    }

    // Bring interface up
    cmd = exec.Command("ip", "link", "set", "dev", d.iface.Name(), "up")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to bring interface up: %w", err)
    }

    log.Printf("[TAP] Interface %s configured: %s/%s", d.iface.Name(), ipAddr, subnet)
    return nil
}

func (d *TAPDevice) ReadFrame() ([]byte, error) {
    frame := make([]byte, d.mtu+14)  // MTU + Ethernet header (14 bytes)
    n, err := d.iface.Read(frame)
    if err != nil {
        return nil, err
    }
    return frame[:n], nil
}

func (d *TAPDevice) WriteFrame(frame []byte) error {
    _, err := d.iface.Write(frame)
    return err
}

// Ethernet frame structure (14-byte header + payload)
type EthernetFrame struct {
    DestMAC   net.HardwareAddr  // 6 bytes
    SourceMAC net.HardwareAddr  // 6 bytes
    EtherType uint16            // 2 bytes (0x0800 = IPv4, 0x86DD = IPv6)
    Payload   []byte            // IP packet or other L3 protocol
}

func ParseEthernetFrame(raw []byte) (*EthernetFrame, error) {
    if len(raw) < 14 {
        return nil, errors.New("frame too short")
    }

    return &EthernetFrame{
        DestMAC:   net.HardwareAddr(raw[0:6]),
        SourceMAC: net.HardwareAddr(raw[6:12]),
        EtherType: binary.BigEndian.Uint16(raw[12:14]),
        Payload:   raw[14:],
    }, nil
}

// Traffic handler (replaces TUN handler)
func (node *Node) handleTAPFrame(frame []byte) {
    // 1. Parse Ethernet frame
    ethFrame, err := ParseEthernetFrame(frame)
    if err != nil {
        log.Printf("[TAP] Invalid frame: %v", err)
        return
    }

    // 2. Extract destination IP from payload (if IPv4/IPv6)
    var destIP net.IP
    switch ethFrame.EtherType {
    case 0x0800:  // IPv4
        if len(ethFrame.Payload) < 20 {
            return
        }
        destIP = net.IP(ethFrame.Payload[16:20])

    case 0x86DD:  // IPv6
        if len(ethFrame.Payload) < 40 {
            return
        }
        destIP = net.IP(ethFrame.Payload[24:40])

    case 0x0806:  // ARP
        // Handle ARP locally
        node.handleARP(ethFrame)
        return

    default:
        log.Printf("[TAP] Unsupported EtherType: 0x%04x", ethFrame.EtherType)
        return
    }

    // 3. Map IP to PeerID
    peerID, found := node.ipToPeerID[destIP.String()]
    if !found {
        log.Printf("[TAP] Unknown destination: %s", destIP)
        return
    }

    // 4. Check for existing tunnel
    tunnel := node.getTunnel(peerID)
    if tunnel == nil {
        // Establish tunnel via DHT
        if err := node.ConnectToPeer(peerID); err != nil {
            log.Printf("[TAP] Failed to connect: %v", err)
            return
        }
        tunnel = node.getTunnel(peerID)
    }

    // 5. Encrypt entire Ethernet frame and send
    //    IMPORTANT: Entire frame encrypted, IP header hidden
    encryptedFrame := node.crypto.EncryptFrame(frame)
    tunnel.SendFrame(encryptedFrame)

    log.Printf("[TAP] Frame sent: %d bytes to %s (IP: %s)",
        len(frame), peerID.Short(), destIP)
}

// ARP handling for Layer 2
func (node *Node) handleARP(frame *EthernetFrame) {
    // Parse ARP packet
    // Respond with our MAC address for mesh network IPs
    // Required for Layer 2 operation
}
```

**Security Benefits of Layer 2**:
```
TUN (Layer 3):
┌─────────────────────────────────┐
│ Encrypted IP Packet             │
│ [Src IP] [Dst IP] [Payload]     │  ← IP headers visible to DPI
└─────────────────────────────────┘

TAP (Layer 2):
┌────────────────────────────────────────┐
│ Encrypted Ethernet Frame               │
│ [MAC] [MAC] [IP hidden] [Payload]      │  ← IP headers encrypted
└────────────────────────────────────────┘

DPI sees: Encrypted blob (no destination IP visible)
```

**Trade-offs**:
- **Complexity**: TAP requires ARP/NDP handling
- **Performance**: +14 bytes Ethernet header overhead
- **Privilege**: Requires CAP_NET_ADMIN or root
- **Security**: ✅ IP headers hidden, prevents traffic analysis

---

## Monitoring & Observability: Grafana + Prometheus

### Overview

ShadowMesh includes a comprehensive monitoring stack using **Prometheus** for metrics collection and **Grafana** for visualization, deployed via Docker Compose alongside the client daemon.

**Architecture Goals** (from PRD):
- Real-time connection health monitoring
- Performance metrics (throughput, latency, packet loss)
- Security metrics (PQC handshakes, key rotations)
- Geographic peer/relay visualization
- Zero-configuration setup (automatic provisioning)

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│  ShadowMesh Monitoring Stack (Docker Compose)               │
└─────────────────────────────────────────────────────────────┘

┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│ ShadowMesh       │────→│ Prometheus       │────→│ Grafana          │
│ Daemon           │     │ :9091            │     │ :8080            │
│                  │     │                  │     │                  │
│ Exposes metrics  │     │ Scrapes every    │     │ Queries via      │
│ on :9090/metrics │     │ 15 seconds       │     │ PromQL           │
│                  │     │                  │     │                  │
│ (Host network)   │     │ 7-day retention  │     │ Pre-configured   │
│ (Privileged)     │     │ ~150 MB memory   │     │ dashboards       │
└──────────────────┘     └──────────────────┘     └──────────────────┘
         │                        │                        │
         │                        │                        │
         v                        v                        v
    TAP Device              Time-series DB          User's Browser
    eth frames             (metrics storage)      http://localhost:8080
```

### Docker Compose Configuration

```yaml
# monitoring/docker-compose.yml
version: '3.8'

services:
  shadowmesh-daemon:
    image: shadowmesh/daemon:latest
    container_name: shadowmesh-daemon
    network_mode: host  # Required for TAP device access
    privileged: true    # Required for CAP_NET_ADMIN
    restart: unless-stopped
    volumes:
      - /etc/shadowmesh:/etc/shadowmesh:ro  # Config files
      - /var/log/shadowmesh:/var/log/shadowmesh  # Logs
    environment:
      - SHADOWMESH_CONFIG=/etc/shadowmesh/config.yaml
      - PROMETHEUS_PORT=9090
    ports:
      - "9090:9090"  # Prometheus metrics endpoint

  prometheus:
    image: prom/prometheus:v2.45.0
    container_name: shadowmesh-prometheus
    restart: unless-stopped
    ports:
      - "9091:9090"  # Expose on 9091 to avoid conflicts
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=7d'  # 7-day retention
      - '--web.console.libraries=/usr/share/prometheus/console_libraries'
      - '--web.console.templates=/usr/share/prometheus/consoles'
    depends_on:
      - shadowmesh-daemon

  grafana:
    image: grafana/grafana:10.0.0
    container_name: shadowmesh-grafana
    restart: unless-stopped
    ports:
      - "8080:3000"  # Expose on port 8080 as per PRD
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
      - ./grafana/dashboards:/var/lib/grafana/dashboards:ro
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=shadowmesh
      - GF_AUTH_ANONYMOUS_ENABLED=true  # Localhost-only, no auth
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Viewer
      - GF_SERVER_ROOT_URL=http://localhost:8080
      - GF_INSTALL_PLUGINS=grafana-worldmap-panel  # World map for peers
    depends_on:
      - prometheus

volumes:
  prometheus-data:
  grafana-data:
```

### Prometheus Configuration

```yaml
# monitoring/prometheus/prometheus.yml
global:
  scrape_interval: 15s  # Scrape metrics every 15 seconds
  evaluation_interval: 15s
  external_labels:
    monitor: 'shadowmesh-client'

scrape_configs:
  - job_name: 'shadowmesh-daemon'
    static_configs:
      - targets: ['host.docker.internal:9090']  # Daemon on host network
    metrics_path: '/metrics'
    scrape_interval: 15s
    scrape_timeout: 10s
```

### Metrics Exposed by Daemon

**Implementation** (`pkg/metrics/prometheus.go`):
```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Connection metrics
var (
    ConnectionStatus = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shadowmesh_connection_status",
        Help: "Connection status (0=disconnected, 1=relay-routed, 2=direct P2P)",
    })

    ConnectionLatency = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shadowmesh_connection_latency_ms",
        Help: "Connection latency in milliseconds",
    })

    NATTraversalType = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shadowmesh_nat_traversal_type",
        Help: "NAT traversal type (0=direct, 1=relay, 2=multi-hop)",
    })
)

// Network metrics
var (
    ThroughputBytes = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "shadowmesh_throughput_bytes_total",
        Help: "Total bytes transmitted/received",
    }, []string{"direction"})  // direction: tx, rx

    PacketLossRatio = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shadowmesh_packet_loss_ratio",
        Help: "Packet loss ratio (0.0-1.0)",
    })

    FrameEncryptionRate = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shadowmesh_frame_encryption_rate_per_sec",
        Help: "Ethernet frames encrypted per second",
    })
)

// Cryptography metrics
var (
    PQCHandshakes = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "shadowmesh_pqc_handshakes_total",
        Help: "Total PQC handshakes (ML-KEM + ML-DSA)",
    }, []string{"status"})  // status: success, failure

    KeyRotations = promauto.NewCounter(prometheus.CounterOpts{
        Name: "shadowmesh_key_rotation_total",
        Help: "Total key rotations performed",
    })

    CryptoCPUUsage = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shadowmesh_crypto_cpu_usage_percent",
        Help: "CPU usage for cryptographic operations (%)",
    })
)

// Relay metrics
var (
    RelayNodesAvailable = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shadowmesh_relay_nodes_available",
        Help: "Number of available relay nodes from chronara.eth",
    })

    RelayNodeLatency = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "shadowmesh_relay_node_latency_ms",
        Help: "Latency to specific relay nodes (ms)",
    }, []string{"relay_id"})

    RelayHopsCount = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shadowmesh_relay_hops_count",
        Help: "Number of relay hops in current connection",
    })
)

// System metrics
var (
    CPUUsage = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shadowmesh_cpu_usage_percent",
        Help: "Total CPU usage (%)",
    })

    MemoryUsage = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shadowmesh_memory_usage_bytes",
        Help: "Total memory usage (bytes)",
    })

    ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "shadowmesh_active_connections_count",
        Help: "Number of active peer connections",
    })
)

// Expose metrics via HTTP
func StartMetricsServer(port int) {
    http.Handle("/metrics", promhttp.Handler())
    addr := fmt.Sprintf(":%d", port)
    log.Printf("[METRICS] Starting Prometheus metrics server on %s", addr)
    log.Fatal(http.ListenAndServe(addr, nil))
}
```

### Grafana Dashboard Provisioning

**Datasource Configuration** (`monitoring/grafana/provisioning/datasources/prometheus.yml`):
```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: false
```

**Dashboard Configuration** (`monitoring/grafana/provisioning/dashboards/dashboard.yml`):
```yaml
apiVersion: 1

providers:
  - name: 'ShadowMesh'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards
```

### Grafana Dashboard Layout (Main User Dashboard)

**Dashboard JSON** (simplified, actual dashboard is ~1000 lines):
```json
{
  "dashboard": {
    "title": "ShadowMesh - Main User Dashboard",
    "panels": [
      {
        "title": "Connection Status",
        "targets": [{"expr": "shadowmesh_connection_status"}],
        "type": "stat",
        "fieldConfig": {
          "mappings": [
            {"value": 0, "text": "Disconnected", "color": "red"},
            {"value": 1, "text": "Relay-Routed", "color": "yellow"},
            {"value": 2, "text": "Direct P2P", "color": "green"}
          ]
        },
        "gridPos": {"x": 0, "y": 0, "w": 6, "h": 4}
      },
      {
        "title": "Connection Latency",
        "targets": [{"expr": "shadowmesh_connection_latency_ms"}],
        "type": "gauge",
        "fieldConfig": {
          "min": 0,
          "max": 200,
          "thresholds": [
            {"value": 0, "color": "green"},
            {"value": 50, "color": "yellow"},
            {"value": 100, "color": "red"}
          ]
        },
        "gridPos": {"x": 6, "y": 0, "w": 6, "h": 4}
      },
      {
        "title": "Throughput (Mbps)",
        "targets": [
          {"expr": "rate(shadowmesh_throughput_bytes_total{direction=\"tx\"}[1m]) * 8 / 1000000", "legendFormat": "Upload"},
          {"expr": "rate(shadowmesh_throughput_bytes_total{direction=\"rx\"}[1m]) * 8 / 1000000", "legendFormat": "Download"}
        ],
        "type": "graph",
        "gridPos": {"x": 0, "y": 4, "w": 12, "h": 6}
      },
      {
        "title": "Peer Map (Geographic Distribution)",
        "type": "grafana-worldmap-panel",
        "targets": [{"expr": "shadowmesh_active_connections_count"}],
        "gridPos": {"x": 0, "y": 10, "w": 12, "h": 6}
      }
    ]
  }
}
```

**Complete Dashboard Specifications** (as per PRD FR29):

**Row 1 - Connection Health:**
- Connection status (stat panel with color mapping)
- NAT traversal type (stat panel)
- Active peer count (gauge)
- Relay node count (gauge from chronara.eth)

**Row 2 - Network Performance:**
- Throughput graph (Mbps tx/rx, last 1h/6h/24h selectable)
- Latency graph (ms average, time series)
- Packet loss gauge (% with thresholds: <1% green, 1-5% yellow, >5% red)

**Row 3 - Security Metrics:**
- PQC handshake counter (success/failure, counter stat)
- Key rotation timeline (time series showing rotation events)
- Crypto CPU usage (gauge % with threshold at 10%)

**Row 4 - Peer Map:**
- World map showing geographic distribution of connected peers
- Relay node list table with columns: Location, Latency (ms), Uptime (%)

### Resource Usage

**Expected Memory Consumption** (as per PRD NFR5):
- ShadowMesh daemon: 100-150 MB
- Prometheus: ~150 MB (7-day retention, 15s scrape interval)
- Grafana: ~200 MB
- System overhead: ~450 MB
- **Total**: ~900 MB (within 1 GB NFR5 limit)

**Disk Usage**:
- Prometheus data (7 days): 500 MB - 1 GB
- Grafana database: ~50 MB
- **Total**: ~1 GB

### Deployment & User Experience

**Installation** (as per PRD NFR19):
1. User installs ShadowMesh client package (`.deb`, `.rpm`)
2. Runs: `shadowmesh install-monitoring`
3. Script:
   - Installs Docker + Docker Compose if needed
   - Downloads pre-built images (no build step)
   - Provisions Grafana dashboards automatically
   - Starts stack via `docker-compose up -d`
4. User accesses http://localhost:8080 (Grafana)
   - No authentication required (localhost-only, anonymous viewer)
   - Default dashboard loads automatically
5. **Total time**: <5 minutes (within NFR19: 10-minute target)

**User Workflow**:
- **95% of time**: Passive monitoring (glance at dashboard to confirm "green")
- **5% of time**: Active troubleshooting (drill down into latency spikes, check relay node health)

---

## PostgreSQL Database Design

### Overview

ShadowMesh uses **PostgreSQL 14+** for persistent storage of user profiles, device registrations, peer relationships, and connection history with comprehensive audit logging (as per PRD FR23).

**Why PostgreSQL**:
- ACID compliance for critical user data
- Excellent Go support via `pgx` library
- JSON columns for flexible metadata storage
- Full-text search for audit log queries
- Mature replication and backup tools

### Database Schema

```sql
-- users table: Core user profiles
CREATE TABLE users (
    user_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT UNIQUE,  -- Optional email (for account recovery)
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active       BOOLEAN DEFAULT TRUE
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created_at ON users(created_at);

-- devices table: User devices (laptop, desktop, server)
CREATE TABLE devices (
    device_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    device_name       TEXT NOT NULL,  -- User-friendly name (e.g., "Work Laptop")
    peer_id           BYTEA NOT NULL UNIQUE,  -- PeerID (32 bytes, SHA256 of ML-DSA-87 pubkey)
    public_key_mldsa  BYTEA NOT NULL,  -- ML-DSA-87 public key (4627 bytes)
    public_key_mlkem  BYTEA NOT NULL,  -- ML-KEM-1024 public key (1568 bytes)
    registered_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_connected_at TIMESTAMP WITH TIME ZONE,
    is_active         BOOLEAN DEFAULT TRUE,
    device_type       TEXT,  -- "laptop", "desktop", "server", "mobile"
    os_info           JSONB  -- {"os": "Ubuntu 22.04", "arch": "amd64"}
);

CREATE INDEX idx_devices_user_id ON devices(user_id);
CREATE INDEX idx_devices_peer_id ON devices(peer_id);
CREATE INDEX idx_devices_last_connected ON devices(last_connected_at);

-- peer_relationships table: Friend lists and device groups
CREATE TABLE peer_relationships (
    relationship_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    peer_device_id  UUID NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    relationship_type TEXT NOT NULL,  -- "friend", "device_group", "team"
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata        JSONB,  -- {"group_name": "Dev Team", "permissions": ["read", "write"]}
    UNIQUE(user_id, peer_device_id, relationship_type)
);

CREATE INDEX idx_peer_relationships_user_id ON peer_relationships(user_id);
CREATE INDEX idx_peer_relationships_peer_device_id ON peer_relationships(peer_device_id);

-- connection_history table: Historical connection logs
CREATE TABLE connection_history (
    connection_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_device_id UUID NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    target_device_id UUID NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    connected_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    disconnected_at  TIMESTAMP WITH TIME ZONE,
    connection_type  TEXT NOT NULL,  -- "direct_p2p", "relay_single", "relay_multi_hop"
    relay_nodes      TEXT[],  -- Array of relay node addresses used
    throughput_bytes_tx BIGINT DEFAULT 0,
    throughput_bytes_rx BIGINT DEFAULT 0,
    avg_latency_ms   FLOAT,
    packet_loss_percent FLOAT,
    disconnect_reason TEXT  -- "user_initiated", "network_failure", "timeout"
);

CREATE INDEX idx_connection_history_source ON connection_history(source_device_id, connected_at DESC);
CREATE INDEX idx_connection_history_target ON connection_history(target_device_id, connected_at DESC);
CREATE INDEX idx_connection_history_connected_at ON connection_history(connected_at DESC);

-- access_logs table: Comprehensive audit logging
CREATE TABLE access_logs (
    log_id        BIGSERIAL PRIMARY KEY,
    timestamp     TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    user_id       UUID REFERENCES users(user_id) ON DELETE SET NULL,
    device_id     UUID REFERENCES devices(device_id) ON DELETE SET NULL,
    action        TEXT NOT NULL,  -- "device_registered", "connection_established", "key_rotated", "auth_failed"
    resource      TEXT,  -- Resource affected (e.g., "peer:2f8a3c4d")
    result        TEXT NOT NULL,  -- "success", "failure", "denied"
    ip_address    INET,  -- Source IP (if applicable)
    metadata      JSONB,  -- Additional structured data
    error_message TEXT  -- Error details for failed actions
);

CREATE INDEX idx_access_logs_timestamp ON access_logs(timestamp DESC);
CREATE INDEX idx_access_logs_user_id ON access_logs(user_id, timestamp DESC);
CREATE INDEX idx_access_logs_action ON access_logs(action);
CREATE INDEX idx_access_logs_result ON access_logs(result);

-- Add GIN index for JSONB metadata queries
CREATE INDEX idx_access_logs_metadata_gin ON access_logs USING GIN(metadata);

-- device_groups table: User-defined device groups for access control
CREATE TABLE device_groups (
    group_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    group_name  TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, group_name)
);

CREATE INDEX idx_device_groups_user_id ON device_groups(user_id);

-- device_group_members table: Many-to-many relationship for groups
CREATE TABLE device_group_members (
    group_id   UUID NOT NULL REFERENCES device_groups(group_id) ON DELETE CASCADE,
    device_id  UUID NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    added_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (group_id, device_id)
);

CREATE INDEX idx_device_group_members_device_id ON device_group_members(device_id);
```

### Database Configuration

```yaml
# config.yaml
database:
  host: localhost
  port: 5432
  name: shadowmesh
  user: shadowmesh
  password: ${SHADOWMESH_DB_PASSWORD}  # Load from env var
  ssl_mode: require  # Production: require, Dev: disable
  max_connections: 25
  connection_timeout: 10s
  idle_timeout: 5m
```

### Go Integration (`pkg/database/postgres.go`)

```go
import (
    "github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
    pool *pgxpool.Pool
}

func NewDatabase(connString string) (*Database, error) {
    config, err := pgxpool.ParseConfig(connString)
    if err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    // Connection pool settings
    config.MaxConns = 25
    config.MinConns = 5
    config.MaxConnLifetime = time.Hour
    config.MaxConnIdleTime = 5 * time.Minute

    pool, err := pgxpool.NewWithConfig(context.Background(), config)
    if err != nil {
        return nil, fmt.Errorf("failed to create pool: %w", err)
    }

    return &Database{pool: pool}, nil
}

// RegisterDevice - Insert new device
func (db *Database) RegisterDevice(ctx context.Context, device *Device) error {
    query := `
        INSERT INTO devices (user_id, device_name, peer_id, public_key_mldsa, public_key_mlkem, device_type, os_info)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING device_id, registered_at
    `

    err := db.pool.QueryRow(ctx, query,
        device.UserID,
        device.DeviceName,
        device.PeerID,
        device.PublicKeyMLDSA,
        device.PublicKeyMLKEM,
        device.DeviceType,
        device.OSInfo,
    ).Scan(&device.DeviceID, &device.RegisteredAt)

    if err != nil {
        return fmt.Errorf("failed to register device: %w", err)
    }

    // Audit log
    db.LogAccess(ctx, AccessLog{
        UserID:   device.UserID,
        DeviceID: device.DeviceID,
        Action:   "device_registered",
        Resource: fmt.Sprintf("device:%s", device.DeviceName),
        Result:   "success",
    })

    return nil
}

// GetDeviceByPeerID - Retrieve device by PeerID
func (db *Database) GetDeviceByPeerID(ctx context.Context, peerID []byte) (*Device, error) {
    query := `
        SELECT device_id, user_id, device_name, peer_id, public_key_mldsa, public_key_mlkem,
               registered_at, last_connected_at, is_active, device_type, os_info
        FROM devices
        WHERE peer_id = $1 AND is_active = true
    `

    var device Device
    err := db.pool.QueryRow(ctx, query, peerID).Scan(
        &device.DeviceID,
        &device.UserID,
        &device.DeviceName,
        &device.PeerID,
        &device.PublicKeyMLDSA,
        &device.PublicKeyMLKEM,
        &device.RegisteredAt,
        &device.LastConnectedAt,
        &device.IsActive,
        &device.DeviceType,
        &device.OSInfo,
    )

    if err != nil {
        return nil, fmt.Errorf("device not found: %w", err)
    }

    return &device, nil
}

// LogConnection - Record connection history
func (db *Database) LogConnection(ctx context.Context, conn *ConnectionHistory) error {
    query := `
        INSERT INTO connection_history (source_device_id, target_device_id, connection_type, relay_nodes)
        VALUES ($1, $2, $3, $4)
        RETURNING connection_id, connected_at
    `

    err := db.pool.QueryRow(ctx, query,
        conn.SourceDeviceID,
        conn.TargetDeviceID,
        conn.ConnectionType,
        conn.RelayNodes,
    ).Scan(&conn.ConnectionID, &conn.ConnectedAt)

    return err
}

// UpdateDisconnection - Update connection with disconnect info
func (db *Database) UpdateDisconnection(ctx context.Context, connID uuid.UUID, stats *ConnectionStats) error {
    query := `
        UPDATE connection_history
        SET disconnected_at = NOW(),
            throughput_bytes_tx = $2,
            throughput_bytes_rx = $3,
            avg_latency_ms = $4,
            packet_loss_percent = $5,
            disconnect_reason = $6
        WHERE connection_id = $1
    `

    _, err := db.pool.Exec(ctx, query,
        connID,
        stats.ThroughputBytesTx,
        stats.ThroughputBytesRx,
        stats.AvgLatencyMs,
        stats.PacketLossPercent,
        stats.DisconnectReason,
    )

    return err
}

// LogAccess - Audit logging
func (db *Database) LogAccess(ctx context.Context, log AccessLog) error {
    query := `
        INSERT INTO access_logs (user_id, device_id, action, resource, result, ip_address, metadata, error_message)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

    _, err := db.pool.Exec(ctx, query,
        log.UserID,
        log.DeviceID,
        log.Action,
        log.Resource,
        log.Result,
        log.IPAddress,
        log.Metadata,
        log.ErrorMessage,
    )

    return err
}

// GetUserDevices - Retrieve all devices for a user
func (db *Database) GetUserDevices(ctx context.Context, userID uuid.UUID) ([]Device, error) {
    query := `
        SELECT device_id, device_name, peer_id, registered_at, last_connected_at, is_active, device_type
        FROM devices
        WHERE user_id = $1
        ORDER BY last_connected_at DESC NULLS LAST
    `

    rows, err := db.pool.Query(ctx, query, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var devices []Device
    for rows.Next() {
        var device Device
        err := rows.Scan(
            &device.DeviceID,
            &device.DeviceName,
            &device.PeerID,
            &device.RegisteredAt,
            &device.LastConnectedAt,
            &device.IsActive,
            &device.DeviceType,
        )
        if err != nil {
            return nil, err
        }
        devices = append(devices, device)
    }

    return devices, rows.Err()
}
```

### Data Types

```go
type Device struct {
    DeviceID        uuid.UUID
    UserID          uuid.UUID
    DeviceName      string
    PeerID          []byte  // 32 bytes
    PublicKeyMLDSA  []byte  // 4627 bytes
    PublicKeyMLKEM  []byte  // 1568 bytes
    RegisteredAt    time.Time
    LastConnectedAt *time.Time
    IsActive        bool
    DeviceType      string
    OSInfo          map[string]interface{}  // JSON
}

type ConnectionHistory struct {
    ConnectionID      uuid.UUID
    SourceDeviceID    uuid.UUID
    TargetDeviceID    uuid.UUID
    ConnectedAt       time.Time
    DisconnectedAt    *time.Time
    ConnectionType    string
    RelayNodes        []string
    ThroughputBytesTx int64
    ThroughputBytesRx int64
    AvgLatencyMs      float64
    PacketLossPercent float64
    DisconnectReason  string
}

type AccessLog struct {
    LogID        int64
    Timestamp    time.Time
    UserID       *uuid.UUID
    DeviceID     *uuid.UUID
    Action       string
    Resource     string
    Result       string
    IPAddress    net.IP
    Metadata     map[string]interface{}  // JSON
    ErrorMessage string
}
```

### Backup & Maintenance

```bash
# Automated daily backups
pg_dump -U shadowmesh -d shadowmesh -F c -f shadowmesh_backup_$(date +%Y%m%d).dump

# Restore from backup
pg_restore -U shadowmesh -d shadowmesh shadowmesh_backup_20251111.dump

# Vacuum and analyze (weekly)
psql -U shadowmesh -d shadowmesh -c "VACUUM ANALYZE;"

# Partition access_logs table (for large deployments)
# Partition by month for efficient retention management
```

### Performance Considerations

**Expected Load**:
- Device registrations: ~10/minute (burst)
- Connection logs: ~100/minute
- Access logs: ~1,000/minute
- Queries: ~500/minute (device lookups, relationship checks)

**Optimizations**:
- Connection pooling (25 max connections)
- Prepared statements for frequent queries
- Indexes on all foreign keys and timestamp columns
- GIN index on JSONB columns for metadata queries
- Partitioning for `access_logs` and `connection_history` (production)

**Scaling Strategy** (post-MVP):
- Read replicas for query load
- Partition large tables by time (monthly partitions)
- Archive old data to S3/cold storage after 90 days

---

## Public Network Map (https://map.shadowmesh.network)

### Overview

The **public network map** is a separate web service that visualizes all registered relay nodes from the chronara.eth smart contract, providing transparency and trust for prospective users (as per PRD FR35-39).

**Design Goals**:
- Public transparency of relay node coverage
- Privacy-preserving (city/country level, no IP addresses)
- Real-time updates (<60s after blockchain events)
- Zero private user data exposure
- Lightweight frontend (React + Leaflet.js)

### Architecture Diagram

```
┌────────────────────────────────────────────────────────────────┐
│  Public Network Map Architecture                                │
└────────────────────────────────────────────────────────────────┘

┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│ Ethereum         │────→│ Indexer Service  │────→│ PostgreSQL       │
│ chronara.eth     │     │ (Go)             │     │ (Cache DB)       │
│                  │     │                  │     │                  │
│ Smart contract   │     │ Monitors events: │     │ Stores:          │
│ events emitted   │     │ NodeRegistered   │     │ - Relay nodes    │
│ every block      │     │ NodeDeregistered │     │ - Geolocations   │
│                  │     │ HeartbeatReceived│     │ - Uptime stats   │
│                  │     │                  │     │                  │
│                  │     │ Updates cache    │     │ Aggregates:      │
│                  │     │ every 30 seconds │     │ - Total nodes    │
│                  │     │                  │     │ - Coverage stats │
└──────────────────┘     └──────────────────┘     └──────────────────┘
                                   │                        │
                                   │                        │
                                   v                        v
                         ┌──────────────────┐     ┌──────────────────┐
                         │ REST API (Go)    │←────│ React Frontend   │
                         │ :8080            │     │ (Static Site)    │
                         │                  │     │                  │
                         │ GET /api/nodes   │     │ Leaflet.js map   │
                         │ GET /api/stats   │     │ Node markers     │
                         │                  │     │ Aggregate stats  │
                         └──────────────────┘     └──────────────────┘
                                                            │
                                                            v
                                                   User's Browser
                                            https://map.shadowmesh.network
```

### Backend Service (Go + Indexer)

**Blockchain Event Indexer** (`services/map-indexer/main.go`):
```go
import (
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type MapIndexer struct {
    client   *ethclient.Client
    contract *RelayNodeRegistry
    db       *Database
}

func (m *MapIndexer) Start() {
    // Subscribe to contract events
    eventChan := make(chan *RelayNodeRegistryNodeRegistered)

    sub, err := m.contract.WatchNodeRegistered(&bind.WatchOpts{}, eventChan, nil, nil)
    if err != nil {
        log.Fatalf("Failed to subscribe: %v", err)
    }

    go func() {
        for {
            select {
            case event := <-eventChan:
                log.Printf("[INDEXER] New node registered: %s", hex.EncodeToString(event.PublicKeyHash[:8]))

                // Fetch full node details from contract
                node, err := m.contract.RelayNodes(&bind.CallOpts{}, event.PublicKeyHash)
                if err != nil {
                    log.Printf("[INDEXER] Failed to fetch node: %v", err)
                    continue
                }

                // Obfuscate IP to city/country level
                geoLocation := m.obfuscateIPToCity(node.IpAddress)

                // Store in cache database
                m.db.UpsertRelayNode(RelayNodeCache{
                    PublicKeyHash:     event.PublicKeyHash,
                    Geolocation:       geoLocation,  // "New York, USA" (no IP)
                    StakeAmount:       node.StakeAmount,
                    RegistrationTime:  time.Unix(node.RegistrationTime.Int64(), 0),
                    LastHeartbeat:     time.Unix(node.LastHeartbeat.Int64(), 0),
                    UptimePercentage:  float64(node.UptimePercentage.Uint64()) / 100.0,
                    IsActive:          node.IsActive,
                })

                // Update aggregate stats
                m.updateAggregateStats()

            case err := <-sub.Err():
                log.Printf("[INDEXER] Subscription error: %v", err)
            }
        }
    }()

    // Periodic sync (every 5 minutes) to catch missed events
    ticker := time.NewTicker(5 * time.Minute)
    go func() {
        for range ticker.C {
            m.fullSync()
        }
    }()
}

func (m *MapIndexer) obfuscateIPToCity(ipAddr string) string {
    // Use GeoIP database (MaxMind GeoLite2) to map IP → City, Country
    // NEVER store actual IP addresses
    city, country, err := m.geoIP.Lookup(ipAddr)
    if err != nil {
        return "Unknown Location"
    }
    return fmt.Sprintf("%s, %s", city, country)
}

func (m *MapIndexer) updateAggregateStats() error {
    // Calculate aggregate statistics
    stats := AggregateStats{
        TotalRelayNodes:   m.db.CountActiveNodes(),
        GeographicCoverage: m.db.CountUniqueCountries(),
        AverageUptime:     m.db.CalculateAverageUptime(),
        NetworkHealthStatus: m.calculateNetworkHealth(),
        LastUpdated:       time.Now(),
    }

    return m.db.UpdateAggregateStats(stats)
}
```

**REST API Server** (`services/map-api/main.go`):
```go
type MapAPI struct {
    db *Database
}

func (api *MapAPI) Start() {
    r := chi.NewRouter()
    r.Use(middleware.CORS)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)

    r.Get("/api/nodes", api.handleGetNodes)
    r.Get("/api/stats", api.handleGetStats)

    log.Printf("[API] Starting on :8080")
    http.ListenAndServe(":8080", r)
}

func (api *MapAPI) handleGetNodes(w http.ResponseWriter, r *http.Request) {
    nodes, err := api.db.GetActiveRelayNodes()
    if err != nil {
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }

    // Filter sensitive data
    publicNodes := make([]PublicRelayNode, 0, len(nodes))
    for _, node := range nodes {
        publicNodes = append(publicNodes, PublicRelayNode{
            NodeID:           hex.EncodeToString(node.PublicKeyHash[:8]),  // First 8 bytes only
            Geolocation:      node.Geolocation,  // "New York, USA"
            Lat:              node.ApproxLat,   // City center coordinates
            Lon:              node.ApproxLon,
            StakeETH:         weiToETH(node.StakeAmount),
            UptimePercentage: node.UptimePercentage,
            LastSeen:         node.LastHeartbeat.Format(time.RFC3339),
        })
    }

    json.NewEncoder(w).Encode(publicNodes)
}

func (api *MapAPI) handleGetStats(w http.ResponseWriter, r *http.Request) {
    stats, err := api.db.GetAggregateStats()
    if err != nil {
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(stats)
}
```

### Frontend (React + Leaflet.js)

**`frontend/src/App.tsx`**:
```typescript
import React, { useEffect, useState } from 'react';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import L from 'leaflet';

interface RelayNode {
  nodeID: string;
  geolocation: string;
  lat: number;
  lon: number;
  stakeETH: number;
  uptimePercentage: number;
  lastSeen: string;
}

interface AggregateStats {
  totalRelayNodes: number;
  geographicCoverage: number;
  averageUptime: number;
  networkHealthStatus: string;
}

export default function App() {
  const [nodes, setNodes] = useState<RelayNode[]>([]);
  const [stats, setStats] = useState<AggregateStats | null>(null);

  useEffect(() => {
    // Fetch nodes every 60 seconds
    const fetchNodes = async () => {
      const response = await fetch('/api/nodes');
      const data = await response.json();
      setNodes(data);
    };

    const fetchStats = async () => {
      const response = await fetch('/api/stats');
      const data = await response.json();
      setStats(data);
    };

    fetchNodes();
    fetchStats();

    const interval = setInterval(() => {
      fetchNodes();
      fetchStats();
    }, 60000);  // Update every 60 seconds

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="App">
      <header>
        <h1>ShadowMesh Network Map</h1>
        <p>Powered by <a href="https://chronara.eth">chronara.eth</a></p>
      </header>

      {stats && (
        <div className="stats">
          <div className="stat-card">
            <h3>{stats.totalRelayNodes}</h3>
            <p>Relay Nodes</p>
          </div>
          <div className="stat-card">
            <h3>{stats.geographicCoverage}</h3>
            <p>Countries</p>
          </div>
          <div className="stat-card">
            <h3>{stats.averageUptime.toFixed(2)}%</h3>
            <p>Average Uptime</p>
          </div>
          <div className="stat-card">
            <span className={`health-badge ${stats.networkHealthStatus.toLowerCase()}`}>
              {stats.networkHealthStatus}
            </span>
            <p>Network Health</p>
          </div>
        </div>
      )}

      <MapContainer
        center={[20, 0]}
        zoom={2}
        style={{ height: '600px', width: '100%' }}
      >
        <TileLayer
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          attribution='&copy; OpenStreetMap contributors'
        />

        {nodes.map((node) => (
          <Marker
            key={node.nodeID}
            position={[node.lat, node.lon]}
            icon={L.icon({
              iconUrl: '/relay-icon.png',
              iconSize: [25, 41],
            })}
          >
            <Popup>
              <div>
                <h4>{node.geolocation}</h4>
                <p>Node ID: {node.nodeID}</p>
                <p>Stake: {node.stakeETH} ETH</p>
                <p>Uptime: {node.uptimePercentage.toFixed(2)}%</p>
                <p>Last Seen: {new Date(node.lastSeen).toLocaleString()}</p>
              </div>
            </Popup>
          </Marker>
        ))}
      </MapContainer>

      <footer>
        <p>All data sourced from Ethereum blockchain (chronara.eth smart contract)</p>
        <p>No user data, connection graphs, or traffic patterns are collected or displayed</p>
      </footer>
    </div>
  );
}
```

### Privacy Guarantees

**What is PUBLIC**:
- Total number of relay nodes
- Geographic distribution (city/country level)
- Stake amounts (public blockchain data)
- Uptime percentages (aggregated)
- Node registration timestamps

**What is NEVER DISPLAYED** (as per PRD FR39):
- Private user data
- Connection graphs
- Traffic patterns
- Client identifiers
- Precise IP addresses or hostnames
- Destination IPs or routing paths

### Deployment

```yaml
# docker-compose.yml for map service
version: '3.8'

services:
  map-indexer:
    image: shadowmesh/map-indexer:latest
    restart: unless-stopped
    environment:
      - ETHEREUM_RPC_URL=${ETHEREUM_RPC_URL}
      - DATABASE_URL=postgresql://mapuser:${DB_PASSWORD}@db:5432/shadowmesh_map
    depends_on:
      - db

  map-api:
    image: shadowmesh/map-api:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgresql://mapuser:${DB_PASSWORD}@db:5432/shadowmesh_map
    depends_on:
      - db

  map-frontend:
    image: shadowmesh/map-frontend:latest
    restart: unless-stopped
    ports:
      - "80:80"
    depends_on:
      - map-api

  db:
    image: postgres:14
    restart: unless-stopped
    volumes:
      - map-db-data:/var/lib/postgresql/data
    environment:
      - POSTGRES_DB=shadowmesh_map
      - POSTGRES_USER=mapuser
      - POSTGRES_PASSWORD=${DB_PASSWORD}

volumes:
  map-db-data:
```

### Update Latency

**Target**: <60 seconds from blockchain event to map display (PRD FR38)

**Measured Latency**:
- Ethereum block time: ~12 seconds
- Event indexer processing: ~5 seconds
- Cache update: ~2 seconds
- Frontend poll interval: ~60 seconds (worst case)
- **Total**: 19-79 seconds (meets requirement)

**Optimization** (future):
- WebSocket for real-time updates (<2s latency)
- Server-sent events (SSE) for push notifications

---

## Project Structure

```
shadowmesh/
├── cmd/
│   ├── shadowmesh/              # Main CLI binary (client daemon)
│   │   └── main.go              # Entry point, CLI flags, daemon orchestration
│   ├── shadowmesh-relay/        # Relay node binary
│   │   └── main.go              # Relay-specific logic with chronara.eth integration
│   ├── shadowmesh-bootstrap/    # Bootstrap node binary
│   │   └── main.go              # Bootstrap-specific logic, long-running service
│   └── map-services/            # Public network map services
│       ├── indexer/             # Blockchain event indexer
│       │   └── main.go
│       └── api/                 # REST API for map frontend
│           └── main.go
│
├── pkg/                         # Public APIs (importable by other projects)
│   ├── dht/                     # Kademlia DHT implementation
│   │   ├── dht.go               # DHT main struct and public API
│   │   ├── routing_table.go     # Routing table with 256 k-buckets
│   │   ├── peer_id.go           # PeerID generation and verification
│   │   ├── operations.go        # FIND_NODE, STORE, FIND_VALUE, PING
│   │   ├── protocol.go          # DHT message protocol (serialization/deserialization)
│   │   ├── bootstrap.go         # Bootstrap process logic
│   │   └── liveness.go          # Liveness checker background goroutine
│   │
│   ├── crypto/                  # Post-quantum crypto wrappers
│   │   ├── mldsa.go             # ML-DSA-87 signature operations
│   │   ├── mlkem.go             # ML-KEM-1024 key exchange operations
│   │   ├── chacha20.go          # ChaCha20-Poly1305 symmetric crypto
│   │   ├── peer_id.go           # Cryptographic PeerID derivation
│   │   └── handshake.go         # PQC handshake flow
│   │
│   ├── transport/               # Network transport layer
│   │   ├── websocket/           # WebSocket Secure (WSS) transport
│   │   │   ├── server.go        # WSS server (relay nodes)
│   │   │   ├── client.go        # WSS client (peer connections)
│   │   │   ├── connection.go    # WSS connection wrapper
│   │   │   └── obfuscation.go   # Packet size/timing randomization
│   │   ├── udp.go               # UDP transport (fallback for direct P2P)
│   │   ├── quic.go              # QUIC transport (future, post-MVP)
│   │   └── tunnel.go            # Encrypted tunnel abstraction
│   │
│   ├── tap/                     # TAP device management (Layer 2)
│   │   ├── device.go            # TAP device creation and management
│   │   ├── ethernet.go          # Ethernet frame parsing
│   │   ├── arp.go               # ARP handling for Layer 2
│   │   └── frame_handler.go    # Frame capture and injection
│   │
│   ├── blockchain/              # Ethereum smart contract integration
│   │   ├── contract.go          # chronara.eth relay registry client
│   │   ├── relay_registry.go    # Generated contract bindings
│   │   ├── ens.go               # ENS resolution
│   │   └── cache.go             # Relay node caching strategy
│   │
│   ├── database/                # PostgreSQL integration
│   │   ├── postgres.go          # Database connection and queries
│   │   ├── migrations/          # Database migration files
│   │   │   ├── 001_initial_schema.sql
│   │   │   └── 002_add_device_groups.sql
│   │   └── models.go            # Data models (Device, User, AccessLog)
│   │
│   ├── metrics/                 # Prometheus metrics
│   │   ├── prometheus.go        # Metrics definitions and server
│   │   └── collectors.go        # Custom metric collectors
│   │
│   └── types/                   # Shared types and interfaces
│       ├── peer.go              # PeerInfo, PeerMetadata structs
│       ├── relay.go             # RelayNode, ConnectionType structs
│       ├── errors.go            # Custom error types
│       └── config.go            # Configuration structures
│
├── internal/                    # Private implementation (not importable)
│   ├── config/                  # Configuration management
│   │   ├── loader.go            # YAML config file parsing
│   │   ├── validator.go         # Config validation
│   │   └── defaults.go          # Default values
│   │
│   └── logging/                 # Logging utilities
│       ├── logger.go            # Structured logger wrapper
│       └── formats.go           # Log formatting
│
├── contracts/                   # Solidity smart contracts
│   ├── RelayNodeRegistry.sol    # chronara.eth relay registry contract
│   ├── test/                    # Hardhat tests
│   │   └── RelayNodeRegistry.test.js
│   ├── scripts/                 # Deployment scripts
│   │   └── deploy.js
│   ├── hardhat.config.js
│   └── package.json
│
├── monitoring/                  # Docker Compose monitoring stack
│   ├── docker-compose.yml       # Daemon + Prometheus + Grafana
│   ├── prometheus/
│   │   └── prometheus.yml       # Prometheus configuration
│   └── grafana/
│       ├── provisioning/        # Auto-provisioned datasources
│       │   ├── datasources/
│       │   │   └── prometheus.yml
│       │   └── dashboards/
│       │       └── dashboard.yml
│       └── dashboards/          # Pre-configured dashboards
│           ├── main-user-dashboard.json
│           ├── relay-operator-dashboard.json
│           └── developer-debug-dashboard.json
│
├── web/                         # Public network map frontend
│   ├── frontend/                # React + TypeScript
│   │   ├── src/
│   │   │   ├── App.tsx          # Main component with Leaflet.js map
│   │   │   ├── components/
│   │   │   └── styles/
│   │   ├── package.json
│   │   └── tsconfig.json
│   └── docker-compose.yml       # Map services deployment
│
├── test/
│   ├── integration/             # Integration tests
│   │   ├── dht_test.go          # 3-node DHT integration test
│   │   ├── websocket_test.go    # WSS transport test
│   │   ├── blockchain_test.go   # Smart contract integration test
│   │   ├── tap_device_test.go   # TAP device test (requires root)
│   │   └── database_test.go     # PostgreSQL integration test
│   │
│   ├── e2e/                     # End-to-end tests
│   │   ├── full_mesh_test.go    # 5-node mesh network test
│   │   ├── relay_routing_test.go # Multi-hop relay test
│   │   └── performance_test.go  # Throughput and latency benchmarks
│   │
│   └── testdata/                # Test fixtures
│       ├── config.yaml          # Test configuration
│       ├── contracts/           # Test contract addresses
│       └── keys/                # Test keypairs
│
├── scripts/
│   ├── build.sh                 # Build all binaries
│   ├── test.sh                  # Run all tests
│   ├── test-local-network.sh   # Spin up 3-node local test network
│   ├── generate-keys.sh         # Generate ML-DSA-87 keypairs
│   ├── deploy-contracts.sh      # Deploy smart contracts to testnet/mainnet
│   ├── install-monitoring.sh   # Install Docker Compose monitoring stack
│   └── setup-database.sh        # Initialize PostgreSQL database
│
├── configs/
│   ├── shadowmesh.example.yaml  # Example client configuration
│   ├── relay.example.yaml       # Example relay node configuration
│   └── bootstrap.example.yaml   # Example bootstrap node config
│
├── systemd/
│   ├── shadowmesh.service       # Client daemon systemd service
│   ├── shadowmesh-relay.service # Relay node systemd service
│   └── shadowmesh-bootstrap.service
│
├── docs/
│   ├── 1-PLANNING/              # PRD, project brief
│   ├── 2-ARCHITECTURE/          # Architecture documentation (this file)
│   ├── 3-IMPLEMENTATION/        # Implementation guides
│   └── 4-OPERATIONS/            # Deployment and operations
│
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
├── Makefile                     # Build automation
├── README.md                    # Project overview
└── LICENSE                      # License file (MIT or Apache 2.0)
```

**Package Organization Principles**:
- `pkg/` contains **public APIs** - importable by other projects
- `internal/` contains **private implementation** - not importable outside shadowmesh
- `cmd/` contains **executable entry points** - thin wrappers around pkg/ logic
- `test/` contains **tests beyond unit tests** - integration, e2e, performance

**File Naming Conventions**:
- Go source files: `snake_case.go` (e.g., `routing_table.go`, `peer_id.go`)
- Test files: `*_test.go` in same package as code under test
- One primary type per file (e.g., `RoutingTable` in `routing_table.go`)

**Package Dependencies**:
```
cmd/shadowmesh → pkg/dht, pkg/crypto, pkg/transport, pkg/tun → internal/config, internal/logging
                                                               ↘ external dependencies (circl, x/crypto)
```

---

## Implementation Patterns

### Naming Conventions

**Files**:
- Go source: `snake_case.go` (e.g., `routing_table.go`, `peer_id.go`, `operations.go`)
- Test files: `*_test.go` (e.g., `routing_table_test.go`)
- Build scripts: `kebab-case.sh` (e.g., `test-local-network.sh`)

**Packages**:
- Lowercase, single word preferred: `dht`, `crypto`, `transport`, `tun`
- Multi-word only if necessary: `config`, `logging`, `types`

**Types**:
- Exported (public): `PascalCase` (e.g., `PeerID`, `RoutingTable`, `DHTMessage`)
- Unexported (private): `camelCase` (e.g., `kBucket`, `peerCache`)

**Functions**:
- Exported (public): `PascalCase` (e.g., `FindNode`, `GeneratePeerID`)
- Unexported (private): `camelCase` (e.g., `selectClosestUnqueried`, `validateMetadata`)

**Constants**:
- Exported: `PascalCase` or `UPPER_SNAKE_CASE` for globals (e.g., `DefaultK`, `MAX_PEERS_PER_BUCKET`)
- Unexported: `camelCase`

**Variables**:
- Standard Go conventions: `camelCase` for both exported and unexported
- Abbreviations: Keep short and consistent (e.g., `ID` not `Id`, `DHT` not `Dht`)

### File Organization

**One Primary Type Per File**:
```go
// pkg/dht/routing_table.go - Contains RoutingTable type and its methods
type RoutingTable struct { ... }
func NewRoutingTable() *RoutingTable { ... }
func (rt *RoutingTable) AddPeer() { ... }
func (rt *RoutingTable) FindClosest() { ... }

// pkg/dht/peer_id.go - Contains PeerID type and related functions
type PeerID [32]byte
func GeneratePeerID(pubkey []byte) PeerID { ... }
func (p PeerID) String() string { ... }
```

**Test Files**:
- Place `*_test.go` in **same package** as code under test
- Use `_test` package suffix only for black-box testing of exported APIs
```go
// pkg/dht/routing_table_test.go
package dht  // White-box testing (can access unexported functions)

// pkg/dht/integration_test.go
package dht_test  // Black-box testing (only exported APIs)
```

**Interface Definitions**:
- Define interfaces in **consumer package**, not implementation package
- Example: `pkg/transport/tunnel.go` defines `Transport interface`, implemented by `pkg/transport/udp.go`

**Internal Helpers**:
- Shared unexported utilities go in `internal/` packages
- Keep `internal/` focused (config, logging, metrics only)

### Error Handling

**Always Check Errors**:
```go
// ✅ CORRECT - Check and wrap errors with context
peer, err := dht.FindValue(targetID)
if err != nil {
    return nil, fmt.Errorf("failed to find peer %s: %w", targetID, err)
}

// ❌ INCORRECT - Don't ignore errors
peer, _ := dht.FindValue(targetID)  // NEVER DO THIS
```

**Error Wrapping**:
```go
// Use fmt.Errorf with %w to wrap errors (preserves error chain)
if err := conn.Write(data); err != nil {
    return fmt.Errorf("write to peer %s failed: %w", peerID, err)
}
```

**Custom Error Types**:
```go
// pkg/types/errors.go - Define custom errors for domain logic
type ErrPeerNotFound struct {
    PeerID PeerID
}

func (e *ErrPeerNotFound) Error() string {
    return fmt.Sprintf("peer not found: %s", e.PeerID)
}

// Usage - Check for specific error types
peer, err := dht.FindValue(targetID)
if errors.Is(err, &ErrPeerNotFound{}) {
    // Handle peer not found specifically
}
```

**Error Messages**:
- Start with lowercase (Go convention): `"failed to connect"` not `"Failed to connect"`
- Include context: peer ID, operation, timestamp
- User-facing errors: Separate from internal errors

### Logging

**Structured Logging**:
```go
// Use structured format: [COMPONENT] Action: details
log.Printf("[DHT] FindNode: target=%s, found=%d peers, latency=%dms",
    targetID.Short(), len(peers), latency.Milliseconds())

log.Printf("[TRANSPORT] Connection established: peer=%s, addr=%s",
    peerID.Short(), addr)

log.Printf("[CRYPTO] Handshake failed: peer=%s, error=%v",
    peerID.Short(), err)
```

**Log Levels** (use standard log package + level prefixes for now):
- `[ERROR]` - Failures that prevent operation (connection lost, handshake failed)
- `[WARN]` - Degradation or recoverable issues (peer unreachable, fallback activated)
- `[INFO]` - Lifecycle events (node started, peer connected, bootstrap complete)
- `[DEBUG]` - Verbose details (packet sent, cache hit, routing table update)

**Sensitive Data**:
- **Never log private keys or session keys**
- **Truncate PeerIDs** to first 8 hex chars: `peerID.Short()` → `"2f8a3c4d"`
- **Redact IP addresses in public logs** (log only in debug mode or internal monitoring)

**PeerID Display Format**:
```go
// pkg/dht/peer_id.go
func (p PeerID) String() string {
    return hex.EncodeToString(p[:])  // Full 64-char hex
}

func (p PeerID) Short() string {
    return hex.EncodeToString(p[:4])  // First 8 hex chars
}

// Usage
log.Printf("Peer connected: %s", peerID.Short())  // "2f8a3c4d"
```

### Testing Patterns

**Table-Driven Tests**:
```go
// pkg/dht/peer_id_test.go
func TestGeneratePeerID(t *testing.T) {
    tests := []struct {
        name      string
        pubkey    []byte
        expectErr bool
    }{
        {"valid ML-DSA-87 pubkey", validPubkey, false},
        {"invalid pubkey length", shortPubkey, true},
        {"nil pubkey", nil, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            peerID, err := GeneratePeerID(tt.pubkey)
            if tt.expectErr && err == nil {
                t.Errorf("expected error, got nil")
            }
            if !tt.expectErr && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
        })
    }
}
```

**Test Helpers**:
```go
// test/testdata/helpers.go - Shared test utilities
func NewTestPeer(t *testing.T) (*Peer, *mldsa.PrivateKey) {
    t.Helper()  // Mark as helper for better error reporting
    // Generate test keys, create peer
}

func StartTestNetwork(t *testing.T, nodeCount int) []*DHT {
    t.Helper()
    // Spin up n-node local test network
}
```

**Integration Tests**:
```go
// test/integration/dht_test.go
func TestDHTThreeNodeNetwork(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    // Create 3-node network, verify FindNode works
}
```

**Mocking**:
- Use interfaces for external dependencies (network, crypto, time)
- Generate mocks with `mockgen` (github.com/golang/mock) or manual mocks
```go
// pkg/transport/transport.go - Define interface
type Transport interface {
    Send(data []byte, addr string) error
    Receive() ([]byte, string, error)
}

// test/integration/dht_test.go - Use mock transport for testing
type mockTransport struct { ... }
```

### Context Usage

**All Network Operations MUST Accept context.Context**:
```go
// ✅ CORRECT - Accept context for cancellation
func (dht *DHT) FindNode(ctx context.Context, target PeerID) ([]PeerInfo, error) {
    // Check context before expensive operations
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Pass context to nested calls
    peers, err := dht.sendFindNodeRequest(ctx, target)
    ...
}

// ❌ INCORRECT - No context means no cancellation
func (dht *DHT) FindNode(target PeerID) ([]PeerInfo, error) {
    // Can't cancel this operation
}
```

**Context Timeouts**:
```go
// Set timeouts for bounded operations
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

peers, err := dht.FindNode(ctx, targetID)
if errors.Is(err, context.DeadlineExceeded) {
    log.Printf("[DHT] FindNode timeout: target=%s", targetID.Short())
}
```

**Context Propagation**:
```go
// Pass context through entire call chain
func (node *Node) ConnectToPeer(ctx context.Context, targetID PeerID) error {
    metadata, err := node.dht.FindValue(ctx, targetID)  // Pass context
    if err != nil {
        return err
    }

    conn, err := node.transport.Connect(ctx, metadata.Address)  // Pass context
    ...
}
```

### Concurrency Patterns

**Goroutines**:
- Always have clear exit conditions
- Use `context.Context` for cancellation
- Avoid naked `go func()` - wrap in functions for testing

```go
// Background liveness checker
func (dht *DHT) StartLivenessChecker(ctx context.Context) {
    ticker := time.NewTicker(15 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            log.Printf("[DHT] Liveness checker stopped")
            return
        case <-ticker.C:
            dht.checkAllPeers(ctx)
        }
    }
}
```

**Synchronization**:
- Use `sync.RWMutex` for read-heavy data structures (routing table)
- Use channels for coordination between goroutines
- Avoid naked mutexes - wrap in methods

```go
type RoutingTable struct {
    kBuckets [256]*KBucket
    mutex    sync.RWMutex  // Protect concurrent access
}

func (rt *RoutingTable) AddPeer(peer PeerInfo) {
    rt.mutex.Lock()
    defer rt.mutex.Unlock()
    // Safe concurrent access
}

func (rt *RoutingTable) FindClosest(target PeerID, k int) []PeerInfo {
    rt.mutex.RLock()  // Read lock (multiple readers allowed)
    defer rt.mutex.RUnlock()
    // Safe concurrent reads
}
```

### Configuration Patterns

**YAML Configuration**:
```yaml
# configs/shadowmesh.example.yaml
node:
  peer_id_file: /etc/shadowmesh/peer_id
  private_key_file: /etc/shadowmesh/private_key.pem

network:
  listen_address: "0.0.0.0:9443"
  tun_device: "shadowmesh0"
  tun_address: "10.10.0.1/16"

dht:
  k: 20  # Peers per k-bucket
  alpha: 3  # Parallel lookups
  bootstrap_nodes:
    - peer_id: "2f8a3c4d5e6f7a8b..."
      address: "bootstrap1.shadowmesh.net:9443"
      public_key: "base64_encoded_key"

logging:
  level: "info"  # error, warn, info, debug
  file: "/var/log/shadowmesh/shadowmesh.log"
```

**Configuration Loading**:
```go
// internal/config/loader.go
type Config struct {
    Node    NodeConfig    `yaml:"node"`
    Network NetworkConfig `yaml:"network"`
    DHT     DHTConfig     `yaml:"dht"`
    Logging LoggingConfig `yaml:"logging"`
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    return &cfg, nil
}
```

### API Response Patterns

**DHT Message Responses**:
```go
// All DHT messages include:
// - RequestID: [16]byte nonce for request/response matching
// - SenderID: PeerID of sender
// - Signature: ML-DSA-87 signature over message

type DHTResponse struct {
    RequestID [16]byte
    SenderID  PeerID
    Payload   interface{}  // Type-specific payload
    Signature []byte
}
```

**Error Responses**:
```go
// Return structured errors with context
type DHTError struct {
    Code    int    // Error code
    Message string // Human-readable message
    PeerID  PeerID // Peer that caused error
}
```

### Consistency Patterns

**Timestamp Format**:
- Internal: `time.Time` (Go standard library)
- Serialization: RFC3339 or Unix timestamp (seconds since epoch)
- Display: RFC3339 for logs and user-facing output

**Address Format**:
- Internal: `string` in format `"IP:port"` (e.g., `"192.168.1.1:9443"`)
- IPv6: Use brackets `"[2001:db8::1]:9443"`

**Duration Format**:
- Internal: `time.Duration`
- Configuration: String with unit (e.g., `"15m"`, `"24h"`, `"5s"`)

---

## Project Initialization

### Prerequisites

**System Requirements**:
- **Operating System**: Linux (Ubuntu 20.04+, Debian 11+, Arch) or macOS (12+)
- **Go**: Version 1.25.4 or later
- **TUN/TAP Support**: Kernel module enabled (`modprobe tun` on Linux)
- **Root/Sudo Access**: Required for TUN device creation and network configuration
- **Disk Space**: 500 MB for dependencies and binaries
- **Memory**: 512 MB minimum, 2 GB recommended

**Development Tools**:
```bash
# Linux (Ubuntu/Debian)
sudo apt-get update
sudo apt-get install -y build-essential git make

# macOS
xcode-select --install  # Install Command Line Tools
```

### Go Installation

**Install Go 1.25.4** (if not already installed):

```bash
# Linux (amd64)
wget https://go.dev/dl/go1.25.4.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# Linux (arm64)
wget https://go.dev/dl/go1.25.4.linux-arm64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.4.linux-arm64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# macOS (arm64)
wget https://go.dev/dl/go1.25.4.darwin-arm64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.4.darwin-arm64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Verify installation
go version  # Should show: go version go1.25.4 ...
```

### Project Setup

**1. Clone Repository**:
```bash
git clone https://github.com/yourusername/shadowmesh.git
cd shadowmesh
```

**2. Install Dependencies**:
```bash
# Initialize go.mod (if not already present)
go mod init github.com/yourusername/shadowmesh

# Add required dependencies
go get github.com/cloudflare/circl@v1.6.1
go get golang.org/x/crypto@v0.41.0
go get gopkg.in/yaml.v3@latest  # For YAML configuration parsing

# Download all dependencies
go mod download

# Verify dependencies
go mod verify

# Tidy up go.mod and go.sum
go mod tidy
```

**Expected `go.mod`**:
```go
module github.com/yourusername/shadowmesh

go 1.25

require (
    github.com/cloudflare/circl v1.6.1
    golang.org/x/crypto v0.41.0
    gopkg.in/yaml.v3 v3.0.1
)
```

**3. Verify TUN/TAP Support**:
```bash
# Linux - Check if TUN module is loaded
lsmod | grep tun

# If not loaded, load it
sudo modprobe tun

# Verify TUN device can be created (requires root)
sudo ip tuntap add mode tun dev test0
sudo ip tuntap del mode tun dev test0  # Clean up
```

### Build Project

**Build All Binaries**:
```bash
# Build main CLI
go build -o bin/shadowmesh cmd/shadowmesh/main.go

# Build bootstrap node
go build -o bin/shadowmesh-bootstrap cmd/shadowmesh-bootstrap/main.go

# Or use Makefile
make build  # Builds all binaries

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o bin/shadowmesh-linux-amd64 cmd/shadowmesh/main.go
GOOS=linux GOARCH=arm64 go build -o bin/shadowmesh-linux-arm64 cmd/shadowmesh/main.go
GOOS=darwin GOARCH=arm64 go build -o bin/shadowmesh-darwin-arm64 cmd/shadowmesh/main.go
```

**Build with Optimization** (production):
```bash
# Strip debug symbols, reduce binary size
go build -ldflags="-s -w" -o bin/shadowmesh cmd/shadowmesh/main.go

# With version info embedded
VERSION=$(git describe --tags --always --dirty)
go build -ldflags="-s -w -X main.version=$VERSION" -o bin/shadowmesh cmd/shadowmesh/main.go
```

### Run Tests

**Unit Tests**:
```bash
# Run all unit tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run tests with race detector
go test -race ./...

# Run tests in verbose mode
go test -v ./...

# Run specific package tests
go test ./pkg/dht/
```

**Integration Tests**:
```bash
# Run integration tests (skipped by default with -short)
go test ./test/integration/

# Run with full output
go test -v ./test/integration/
```

**End-to-End Tests**:
```bash
# Run E2E tests (requires root for TUN device)
sudo go test ./test/e2e/

# Run specific E2E test
sudo go test -run TestDHTMeshNetwork ./test/e2e/
```

**Benchmark Tests**:
```bash
# Run all benchmarks
go test -bench=. ./pkg/dht/

# Run specific benchmark
go test -bench=BenchmarkFindNode ./pkg/dht/

# Benchmark with memory profiling
go test -bench=. -benchmem ./pkg/dht/
```

### Local Development Network

**Start 3-Node Test Network**:
```bash
# Use helper script to spin up local test network
./scripts/test-local-network.sh

# Or manually:
# Terminal 1 - Node 1
sudo ./bin/shadowmesh --config configs/node1.yaml

# Terminal 2 - Node 2
sudo ./bin/shadowmesh --config configs/node2.yaml

# Terminal 3 - Node 3
sudo ./bin/shadowmesh --config configs/node3.yaml
```

**Generate Test Keypairs**:
```bash
# Generate ML-DSA-87 keypairs for testing
./scripts/generate-keys.sh node1
./scripts/generate-keys.sh node2
./scripts/generate-keys.sh node3

# Keys stored in test/testdata/keys/
```

### Configuration

**Create Configuration File**:
```bash
# Copy example configuration
cp configs/shadowmesh.example.yaml ~/.shadowmesh/config.yaml

# Edit configuration
nano ~/.shadowmesh/config.yaml
```

**Minimal Configuration** (for development):
```yaml
node:
  peer_id_file: ~/.shadowmesh/peer_id
  private_key_file: ~/.shadowmesh/private_key.pem

network:
  listen_address: "0.0.0.0:9443"
  tun_device: "shadowmesh0"
  tun_address: "10.10.0.1/16"

dht:
  k: 20
  alpha: 3
  bootstrap_nodes:
    - peer_id: "2f8a3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b"
      address: "bootstrap1.shadowmesh.net:9443"
      public_key: "BASE64_ENCODED_PUBLIC_KEY"

logging:
  level: "debug"  # For development
  file: "/tmp/shadowmesh.log"
```

### Common Development Commands

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Update dependencies
go get -u ./...
go mod tidy

# Clean build artifacts
make clean
rm -rf bin/ coverage.out

# View documentation
godoc -http=:6060  # Browse to http://localhost:6060/pkg/github.com/yourusername/shadowmesh/

# Build Docker image (for containerized testing)
docker build -t shadowmesh:dev .

# Run in Docker
docker run --rm --cap-add=NET_ADMIN shadowmesh:dev
```

### Troubleshooting

**TUN Device Permission Denied**:
```bash
# Solution: Run with sudo
sudo ./bin/shadowmesh

# Or set capabilities (Linux only)
sudo setcap cap_net_admin=eip ./bin/shadowmesh
./bin/shadowmesh  # Can now run without sudo
```

**Module Not Found Errors**:
```bash
# Clean module cache and re-download
go clean -modcache
go mod download
go mod verify
```

**Import Cycle Errors**:
```bash
# Check for circular dependencies
go list -f '{{join .DepsErrors "\n"}}' ./...
```

**Build Errors with circl**:
```bash
# Ensure Go 1.25+ is installed
go version

# Clear build cache
go clean -cache

# Rebuild from scratch
go build -a -o bin/shadowmesh cmd/shadowmesh/main.go
```

---

## Migration Path: v11 → v0.2.0-alpha

### Phase 1: DHT Implementation (Weeks 1-4)

**Week 1-2: Core DHT**
```
[ ] Implement PeerID generation from ML-DSA-87
[ ] Implement routing table with 256 k-buckets
[ ] Implement XOR distance metric
[ ] Implement FIND_NODE iterative lookup
[ ] Local 3-node test network
```

**Week 3-4: DHT Operations**
```
[ ] Implement STORE/FIND_VALUE operations
[ ] Implement PING liveness checks
[ ] Implement bootstrap process
[ ] 5-10 node test network
[ ] Peer discovery latency <100ms
```

### Phase 2: Integration (Weeks 5-6)

**Week 5: v11 Integration**
```
[ ] Replace centralized discovery with DHT
[ ] Integrate DHT lookup before PQC handshake
[ ] Update peer connection flow
[ ] End-to-end test: DHT → PQC → TUN
```

**Week 6: Testing & Validation**
```
[ ] Multi-node mesh testing (5 nodes)
[ ] Peer discovery success rate >95%
[ ] Performance regression tests (maintain 28+ Mbps)
[ ] Network partition recovery tests
```

### Phase 3: Standalone Release (Week 7-8)

**Week 7: Release Preparation**
```
[ ] Deploy bootstrap nodes (3 locations)
[ ] Create installation packages (Linux: deb, rpm, arch)
[ ] Write user documentation
[ ] Create quick start guide
```

**Week 8: Alpha Release**
```
[ ] v0.2.0-alpha release with DHT
[ ] Standalone operation validated
[ ] Community testing phase
[ ] Gather feedback for v0.3.0
```

---

## Standalone Release Criteria (v0.2.0-alpha)

### Functional Requirements

✅ **Zero Central Dependencies**
- [ ] Node starts without centralized discovery server
- [ ] Connects to 3+ bootstrap nodes
- [ ] Routing table converges in <60 seconds
- [ ] Peer discovery via DHT successful

✅ **Performance Targets**
- [ ] Throughput: ≥25 Mbps (maintain v11 performance)
- [ ] Latency: <50ms added overhead
- [ ] Packet loss: <5%
- [ ] DHT lookup: <500ms

✅ **Reliability**
- [ ] Peer discovery success rate: >95%
- [ ] Network partition recovery: <5 minutes
- [ ] Uptime: 24+ hour stress test without crashes

✅ **Security**
- [ ] PeerID verification working
- [ ] ML-DSA-87 signature validation
- [ ] No unauthenticated peer connections
- [ ] DHT message rate limiting

### Testing Checklist

**Unit Tests**
- [ ] PeerID generation and verification
- [ ] XOR distance calculations
- [ ] Routing table operations (add, remove, find)
- [ ] DHT message serialization/deserialization

**Integration Tests**
- [ ] 3-node local test network
- [ ] 5-node distributed test network
- [ ] Bootstrap process from cold start
- [ ] Peer discovery and connection establishment
- [ ] Traffic routing through mesh

**Performance Tests**
- [ ] iperf3 throughput tests (≥25 Mbps target)
- [ ] ping latency tests (<50ms target)
- [ ] DHT lookup latency (<500ms target)
- [ ] Memory usage (<500 MB per node)

**Stress Tests**
- [ ] 24-hour uptime test
- [ ] Peer churn test (nodes joining/leaving)
- [ ] Network partition recovery
- [ ] 10+ concurrent connections per node

### Release Artifacts

**Binaries**
- [ ] Linux: shadowmesh-v0.2.0-alpha-linux-amd64
- [ ] Linux: shadowmesh-v0.2.0-alpha-linux-arm64
- [ ] macOS: shadowmesh-v0.2.0-alpha-darwin-arm64

**Packages**
- [ ] Debian/Ubuntu: shadowmesh_0.2.0-alpha_amd64.deb
- [ ] RHEL/Fedora: shadowmesh-0.2.0-alpha.x86_64.rpm
- [ ] Arch: shadowmesh-0.2.0-alpha-x86_64.pkg.tar.zst

**Documentation**
- [ ] README.md (updated with DHT architecture)
- [ ] INSTALL.md (installation instructions)
- [ ] QUICKSTART.md (5-minute setup guide)
- [ ] ARCHITECTURE.md (this document)

**Infrastructure**
- [ ] 3 bootstrap nodes deployed and operational
- [ ] DNS records: bootstrap{1,2,3}.shadowmesh.net
- [ ] Monitoring dashboards for bootstrap nodes

---

## Future Work (v0.3.0+)

### QUIC Migration
- Replace UDP transport with QUIC
- Better NAT traversal via QUIC connection migration
- Stream multiplexing for multiple tunnels

### Advanced DHT Features
- DHT replication factor k=3 (store at 3 nodes for redundancy)
- Republish peer metadata every 12 hours (maintain availability)
- Peer exchange protocol (reduce bootstrap dependency)

### Performance Optimization
- Zero-copy packet handling
- SIMD acceleration for ChaCha20-Poly1305
- Parallel DHT lookups (multiple targets simultaneously)

### Security Enhancements
- Eclipse attack mitigation (diverse routing table)
- Sybil attack defense (stake-based admission)
- Traffic analysis resistance (dummy traffic padding)

---

## Future Enhancements (Post v0.2.0-alpha)

### Built-in Web Services

**Requirement**: Enable nodes to run web services (HTTP/HTTPS servers) accessible only within the ShadowMesh network, with automatic TLS and service discovery.

**Architecture Design** (v0.3.0):

```
┌────────────────────────────────────────────────────────────┐
│  Built-in Web Service Layer                                │
└────────────────────────────────────────────────────────────┘

Node A (10.10.0.1)                    Node B (10.10.0.2)
├── HTTP Server (port 8080)           ├── Access http://10.10.0.1:8080
│   └── shadowmesh-web-proxy          │   via ShadowMesh tunnel
│       ├── Auto TLS (self-signed)    │
│       ├── Access control (PeerID)   └── Encrypted tunnel
│       └── Service registry
```

**Implementation**:
```go
// pkg/webservice/server.go
type WebService struct {
    ListenAddr string         // Local port (e.g., ":8080")
    ServiceName string         // Service identifier (e.g., "myapp-api")
    AllowedPeers []PeerID      // Access control list
    TLSCert  tls.Certificate   // Auto-generated TLS certificate
}

// Register service in DHT
func (ws *WebService) RegisterInDHT(dht *DHT) error {
    // Store service metadata at hash(ServiceName)
    serviceKey := sha256.Sum256([]byte(ws.ServiceName))
    metadata := ServiceMetadata{
        Name: ws.ServiceName,
        PeerID: dht.localPeerID,
        Address: dht.tunAddress,  // TUN IP (10.10.x.x)
        Port: ws.ListenAddr,
        TLSFingerprint: ws.TLSCert.Fingerprint(),
    }
    return dht.Store(serviceKey, metadata)
}

// Discover service by name
func (node *Node) DiscoverService(name string) (*ServiceMetadata, error) {
    serviceKey := sha256.Sum256([]byte(name))
    return node.dht.FindValue(serviceKey)
}
```

**Service Discovery CLI**:
```bash
# Node A - Start web service
shadowmesh service start --name myapp-api --port 8080 --allow-peer <PeerID>

# Node B - Discover and connect
shadowmesh service discover myapp-api
# Output: Service found at http://10.10.0.1:8080

curl http://10.10.0.1:8080/api/status
# Request routed through ShadowMesh tunnel
```

**Security**:
- All HTTP traffic encrypted via TUN tunnel (ChaCha20-Poly1305)
- Optional TLS on top for defense-in-depth
- ACLs enforce PeerID-based access control
- Services only accessible within ShadowMesh network

**Status**: Deferred to v0.3.0 (requires DHT to be stable first)

---

### Cloudflare Integration (Proxy + DNS)

**Requirement**: Integrate Cloudflare Tunnel to hide origin server IP addresses and provide global CDN for services exposed to the public internet.

**Architecture Design** (v0.4.0):

```
┌────────────────────────────────────────────────────────────────────┐
│  Cloudflare Integration Architecture                                │
└────────────────────────────────────────────────────────────────────┘

Public Internet                  Cloudflare Network         ShadowMesh Network
     │                                   │                          │
     │ HTTPS                             │                          │
     │ example.com                       │                          │
     ↓                                   ↓                          ↓
┌──────────┐                    ┌──────────────┐          ┌─────────────┐
│ Browser  │ ─────────────────→ │ Cloudflare   │          │ Node A      │
└──────────┘   DNS: example.com │ Edge (CDN)   │          │ (10.10.0.1) │
               resolves to CF IP │              │          │             │
                                 │ - WAF        │ Encrypted │ - HTTP:8080 │
                                 │ - DDoS       │ Tunnel    │ - Cloudflare│
                                 │ - Cache      │ (QUIC)    │   Tunnel    │
                                 │              │ ────────→ │   Daemon    │
                                 └──────────────┘           └─────────────┘
                                        │                          │
                                        │                          ↓
                                        │                   ShadowMesh Tunnel
                                        │                          │
                                        │                          ↓
                                        │                   ┌─────────────┐
                                        │                   │ Node B      │
                                        └──────────────────→│ (10.10.0.2) │
                                          (optional backup  │ - Backup    │
                                           tunnel to Node B)│   Origin    │
                                                            └─────────────┘
```

**Implementation**:

**1. Cloudflare Tunnel Setup**:
```bash
# Install cloudflared on ShadowMesh node
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 \
  -o /usr/local/bin/cloudflared
chmod +x /usr/local/bin/cloudflared

# Authenticate with Cloudflare
cloudflared tunnel login

# Create tunnel
cloudflared tunnel create shadowmesh-node-a

# Configure tunnel to route to local service
cat > /etc/cloudflared/config.yml <<EOF
tunnel: shadowmesh-node-a
credentials-file: /root/.cloudflared/<TUNNEL-ID>.json

ingress:
  - hostname: example.com
    service: http://localhost:8080  # Local service running on ShadowMesh node
  - service: http_status:404
EOF

# Start tunnel
cloudflared tunnel run shadowmesh-node-a
```

**2. DNS Configuration**:
```bash
# Point DNS to Cloudflare Tunnel
cloudflared tunnel route dns shadowmesh-node-a example.com

# Result: example.com → Cloudflare CDN → Tunnel → ShadowMesh Node → Local Service
```

**3. ShadowMesh Integration**:
```go
// pkg/cloudflare/tunnel.go
type CloudflareTunnel struct {
    TunnelID   string
    Hostname   string
    LocalPort  int              // Port on ShadowMesh node
    ConfigPath string           // /etc/cloudflared/config.yml
    Process    *exec.Cmd        // cloudflared process
}

func (node *Node) StartCloudflareTunnel(cfg CloudflareTunnel) error {
    // 1. Verify local service is running
    if !node.isServiceRunning(cfg.LocalPort) {
        return fmt.Errorf("local service not running on port %d", cfg.LocalPort)
    }

    // 2. Generate cloudflared config
    if err := generateCloudflaredConfig(cfg); err != nil {
        return fmt.Errorf("config generation failed: %w", err)
    }

    // 3. Start cloudflared tunnel
    cmd := exec.Command("cloudflared", "tunnel", "run", cfg.TunnelID)
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("tunnel start failed: %w", err)
    }

    cfg.Process = cmd
    log.Printf("[CLOUDFLARE] Tunnel started: %s → http://localhost:%d", cfg.Hostname, cfg.LocalPort)
    return nil
}
```

**4. CLI Integration**:
```bash
# Start ShadowMesh node with Cloudflare tunnel
shadowmesh start --cloudflare-tunnel shadowmesh-node-a \
  --cloudflare-hostname example.com \
  --local-service-port 8080

# Result:
# - ShadowMesh node starts with DHT peer discovery
# - Local HTTP service starts on port 8080
# - Cloudflare tunnel connects and routes example.com → localhost:8080
# - Origin IP hidden, traffic proxied through Cloudflare
```

**Benefits**:
- **Origin IP Hidden**: Cloudflare CDN shields ShadowMesh node IP from public internet
- **DDoS Protection**: Cloudflare absorbs attacks before reaching origin
- **Global CDN**: Content cached at 200+ edge locations worldwide
- **WAF**: Web Application Firewall filters malicious requests
- **TLS Termination**: Cloudflare handles TLS, simplifies certificate management
- **Zero Trust**: Cloudflare Access can add authentication layer

**Use Cases**:
- **Personal Websites**: Host on ShadowMesh node, expose via Cloudflare without revealing home IP
- **API Services**: Private API accessible globally via Cloudflare, origin hidden
- **Gaming Servers**: Host on ShadowMesh, Cloudflare DDoS protection, no IP leakage
- **IoT Dashboards**: Remotely access home devices via Cloudflare Tunnel → ShadowMesh

**Security Considerations**:
- Cloudflare Tunnel uses authenticated QUIC connections (no public ports exposed)
- Origin validation via Cloudflare-signed JWTs
- Rate limiting and firewall rules at Cloudflare edge
- ShadowMesh provides encrypted peer-to-peer backup routes if Cloudflare fails

**Status**: Planned for v0.4.0 (after DHT and built-in web services are stable)

**Dependency**: Requires `cloudflared` binary and Cloudflare account with configured tunnel

---

## Conclusion

This Kademlia DHT architecture provides a solid foundation for ShadowMesh's transition to **fully decentralized operation**. By eliminating the centralized discovery server, v0.2.0-alpha will achieve:

✅ **True DPN** - No single point of failure
✅ **Standalone** - Operates from first boot
✅ **Scalable** - O(log N) lookup complexity
✅ **Secure** - Cryptographically verifiable peer identity
✅ **Production-ready** - Battle-tested Kademlia protocol

**Next Steps**:
1. Review this specification with team
2. Begin Phase 1 implementation (DHT core)
3. Target v0.2.0-alpha release in 8 weeks
4. Path to v1.0.0 production with QUIC migration

---

**Document Control**
- Version: 1.0
- Author: Winston (Architect)
- Reviewers: [To be assigned]
- Status: Approved for Implementation
- Next Review: After v0.2.0-alpha release
