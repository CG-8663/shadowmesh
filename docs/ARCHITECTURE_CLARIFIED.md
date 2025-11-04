# ShadowMesh Architecture Clarification - November 4, 2025

## Executive Summary

**Critical Insight**: The backbone is for **authentication + DHT lookups**, NOT traffic relay.

Traffic flows **peer-to-peer** after authentication. This dramatically changes the architecture.

---

## Corrected Architecture

### Backbone Purpose (AWS - Pending Credits)

```
┌─────────────────────────────────────────────────────────────┐
│          AWS Regional Authentication Backbone               │
│  (High-availability, production-grade infrastructure)       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Authentication: Verify ML-DSA-87 signatures            │
│  2. Kademlia DHT: Full peer registry (complete database)   │
│  3. Peer Discovery: Lookup peer IPs by PeerID              │
│  4. Regional Presence: 5+ regions globally                  │
│                                                             │
│  Traffic: ZERO (only handles authentication + lookups)     │
│  Bandwidth: Low (<1 Mbps per node)                         │
│  Scalability: Unlimited (no traffic bottleneck)            │
└─────────────────────────────────────────────────────────────┘
```

**NOT a relay server** - backbone does NOT route traffic!

---

### Light Node Purpose (User Devices)

```
┌─────────────────────────────────────────────────────────────┐
│                User Light Nodes                             │
│  (Laptops, Raspberry Pis, Home Labs - CGNAT-friendly)      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Local Kademlia DHT Cache: Small subset of peer DB      │
│  2. Regional Queries: Ask backbone when peer not in cache  │
│  3. Direct P2P Traffic: Connect peer-to-peer for data      │
│  4. CGNAT-Aware: Use relay chains when direct P2P fails    │
│                                                             │
│  Traffic: ALL user traffic (peer-to-peer or relay chains)  │
│  Bandwidth: High (100 Mbps - 1 Gbps per node)              │
│  Scalability: Horizontal (more users = more relay nodes)   │
└─────────────────────────────────────────────────────────────┘
```

---

## Traffic Flow Example

### Scenario: UK user wants to reach Belgium user

```
Step 1: Authentication (happens once at startup)
┌────────────┐                    ┌──────────────────┐
│ UK Light   │  ───AUTHENTICATE──>│ EU AWS Backbone  │
│ Node       │  <──SESSION_TOKEN──│ (London)         │
└────────────┘                    └──────────────────┘

Step 2: Peer Discovery (local cache miss)
┌────────────┐                    ┌──────────────────┐
│ UK Light   │  ──LOOKUP_PEER───> │ EU AWS Backbone  │
│ Node       │      (Belgium      │ (Kademlia DHT)   │
│            │       PeerID)      │                  │
│            │  <──PEER_INFO──────│                  │
│            │     (IP, Port)     └──────────────────┘
└────────────┘

Step 3: Direct P2P Traffic (backbone NOT involved)
┌────────────┐                    ┌──────────────────┐
│ UK Light   │ ═══════════════════│ Belgium Light    │
│ Node       │    Direct P2P      │ Node             │
│ 10.10.10.3 │    (encrypted)     │ 10.10.10.4       │
└────────────┘                    └──────────────────┘

Backbone involvement: ZERO after peer discovery
Traffic through backbone: ZERO
All user data: Peer-to-peer or through relay chains
```

---

## Why This Design Is Brilliant

### 1. Backbone Scalability
```
Traditional VPN Relay:
  - 1000 users @ 100 Mbps = 100 Gbps backbone traffic
  - Cost: $10K/month bandwidth
  - Bottleneck: Relay server capacity

ShadowMesh Backbone:
  - 1000 users @ 100 Mbps = 0 Gbps backbone traffic
  - Authentication: <1 KB per session
  - DHT lookup: <1 KB per query
  - Cost: $100/month (authentication only)
  - No bottleneck: Traffic is peer-to-peer
```

**Cost Savings**: 99% cheaper than traditional relay infrastructure

### 2. CGNAT Compatibility
```
If direct P2P fails (CGNAT, symmetric NAT):
  - Light nodes form relay chains through PUBLIC nodes
  - PUBLIC nodes are OTHER USERS sharing bandwidth
  - Backbone NEVER routes traffic
  - No backbone bandwidth cost
```

### 3. Local Kademlia Cache
```
Light Node Memory:
  - Full DHT: 10,000 peers × 100 bytes = 1 MB
  - Local cache: 100 peers × 100 bytes = 10 KB

Query Pattern:
  - Cache hit (90%): Instant lookup, zero network
  - Cache miss (10%): Query backbone, 10-50ms

Result: Fast peer discovery with minimal memory
```

---

## Architecture Components

### AWS Authentication Backbone (Pending Credits)

**Deployment**: 5-7 regions globally

| Region | AWS Zone | Purpose | Instance Type |
|--------|----------|---------|---------------|
| North America | us-east-1 | Auth + DHT | t3.medium |
| Latin America | sa-east-1 | Auth + DHT | t3.medium |
| Europe | eu-west-1 | Auth + DHT | t3.medium |
| Asia-Pacific | ap-southeast-1 | Auth + DHT | t3.medium |
| Australia | ap-southeast-2 | Auth + DHT | t3.medium |

**Cost**: ~$20-30/month per region = $100-150/month total (AWS credits)

**Services Per Node**:
1. **Authentication Server**: Verify ML-DSA-87 signatures
2. **Kademlia DHT Server**: Full peer registry
3. **NodeNexus Query API**: HTTP/JSON peer lookups
4. **Health Check API**: Monitoring and failover

**Tech Stack**:
- Language: Go (same as light nodes)
- Database: Redis (in-memory DHT cache)
- Storage: PostgreSQL (persistent peer registry)
- API: REST + WebSocket (for live updates)

---

### Light Node Software (User Devices)

**Deployment**: User laptops, Raspberry Pis, home servers

```go
// Light node components
type LightNode struct {
    // Authentication
    SessionToken    string       // From AWS backbone
    PeerID          [20]byte     // ML-DSA-87 public key hash
    SigningKey      *mldsaKey    // For authentication

    // Local Kademlia cache (small)
    LocalDHT        *KademliaDHT // ~100 peers cached
    BackboneClient  *BackboneClient // Query AWS when cache miss

    // P2P networking
    DirectP2P       *DirectP2PManager
    RelayChains     *RelayChainManager

    // Service configuration
    ServiceType     ServiceType  // PRIVATE or PUBLIC
    Bandwidth       int          // If PUBLIC: how much to share
}

// Peer lookup with cache
func (ln *LightNode) LookupPeer(peerID [20]byte) (*PeerInfo, error) {
    // Step 1: Check local cache
    if peer := ln.LocalDHT.Get(peerID); peer != nil {
        return peer, nil
    }

    // Step 2: Cache miss - query backbone
    peer, err := ln.BackboneClient.QueryPeer(peerID)
    if err != nil {
        return nil, err
    }

    // Step 3: Update local cache
    ln.LocalDHT.Put(peerID, peer)

    return peer, nil
}
```

---

### NodeNexus Smart Contract (Polygon Mumbai)

**Purpose**: Registry of ALL nodes (backbone + light nodes)

```solidity
contract NodeNexus {
    enum NodeType {
        BACKBONE_AUTH,    // AWS authentication + DHT servers
        LIGHT_PRIVATE,    // User node, private use only
        LIGHT_PUBLIC      // User node, sharing bandwidth
    }

    struct Node {
        bytes20 peerID;           // ML-DSA-87 public key hash
        NodeType nodeType;
        string region;            // "north_america", "europe", etc.
        address operator;         // Ethereum address
        uint256 stake;            // SHM tokens staked (0 for PRIVATE)
        uint256 bandwidthMbps;    // Bandwidth offered
        uint256 lastSeen;         // Timestamp
        bool active;
    }

    // Query backbone nodes by region
    function getBackboneNodes(string region)
        external view returns (bytes20[] memory);

    // Query public light nodes for relay chains
    function getPublicLightNodes(string region, uint256 count)
        external view returns (bytes20[] memory);

    // Register new node
    function registerNode(
        bytes20 peerID,
        NodeType nodeType,
        string region
    ) external payable;
}
```

---

## Current Status vs. Target

### What We Have (UpCloud, Limited Credits)
```
✅ London relay server (83.136.252.52) - operational
✅ Singapore server (94.237.2.168) - ready for testing
✅ Sydney server (95.111.218.30) - ready for testing
✅ €95.73 credits remaining (~1.7 months at 5-region burn)
```

### What We're Waiting For (AWS Credits)
```
⏳ Production authentication backbone (5-7 regions)
⏳ High-availability Kademlia DHT servers
⏳ Redis + PostgreSQL infrastructure
⏳ Load balancers and auto-scaling
```

---

## Development Plan (While Waiting for AWS)

### Phase 1: Software Development (No AWS Required) ✅

**Can Do Now**:
1. ✅ Build light node daemon locally
2. ✅ Implement local Kademlia DHT cache
3. ✅ Create backbone query protocol (spec)
4. ✅ Build authentication protocol
5. ✅ Develop web GUI for PRIVATE/PUBLIC toggle
6. ✅ Deploy NodeNexus contract to Polygon Mumbai testnet
7. ✅ Test light node software on UpCloud Singapore/Sydney

**Timeline**: 4-6 weeks
**Cost**: €0 (use existing UpCloud nodes for testing)

---

### Phase 2: AWS Backbone Deployment (Once Credits Available) ⏳

**Requires AWS Credits**:
1. Provision 5-7 AWS EC2 instances (t3.medium)
2. Deploy authentication servers
3. Deploy Kademlia DHT servers
4. Configure Redis + PostgreSQL
5. Set up load balancers
6. Configure Route53 DNS
7. SSL certificates (Let's Encrypt)

**Timeline**: 1-2 weeks after AWS credits
**Cost**: $100-150/month (covered by AWS credits)

---

### Phase 3: Integration Testing (AWS + Light Nodes) ⏳

**After Both Are Ready**:
1. Connect light nodes to AWS backbone
2. Test authentication flow
3. Test peer discovery (cache hit/miss)
4. Test direct P2P traffic
5. Test relay chains for CGNAT users
6. Load testing (1000+ light nodes)

**Timeline**: 2-3 weeks
**Cost**: Minimal (testing only)

---

### Phase 4: Beta Launch ⏳

**Public Beta**:
1. Recruit 100 beta users
2. Deploy light node software (Windows, macOS, Linux)
3. Monitor backbone load
4. Measure peer discovery latency
5. Gather feedback
6. Iterate

**Timeline**: 4-6 weeks
**Cost**: $150-200/month (AWS + UpCloud)

---

## Immediate Actions (This Week)

### 1. Software Development (Local, No Cloud Cost)

**Build Light Node Daemon**:
```bash
cd /Volumes/BACKUPDISK/webcode/shadowmesh
mkdir -p cmd/lightnode pkg/lightnode pkg/kademlia pkg/backbone-client

# Implement:
# - cmd/lightnode/main.go (entry point)
# - pkg/lightnode/lightnode.go (core logic)
# - pkg/kademlia/dht.go (local cache)
# - pkg/backbone-client/client.go (query AWS backbone)
```

**Timeline**: 16-20 hours (~2-3 weeks part-time)

---

### 2. Authentication Protocol Spec

**Design Document**:
```markdown
# ShadowMesh Authentication Protocol

## Step 1: Generate Session Token
Light Node → AWS Backbone:
  - Challenge: Random 32 bytes
  - PeerID: ML-DSA-87 public key hash
  - Signature: Sign(challenge, ML-DSA-87 private key)

AWS Backbone → Light Node:
  - SessionToken: JWT with 24-hour expiry
  - BackboneNodes: List of regional backbone IPs
```

**Timeline**: 2-4 hours

---

### 3. Deploy NodeNexus Contract (Polygon Mumbai)

**Smart Contract Deployment**:
```bash
cd /Volumes/BACKUPDISK/webcode/shadowmesh/contracts
npx hardhat run scripts/deploy-nexus.ts --network mumbai
```

**Timeline**: 4-6 hours
**Cost**: $0 (testnet)

---

### 4. Test on Existing UpCloud Nodes

**Use Singapore + Sydney for Testing**:
```bash
# Deploy light node to Singapore
ssh root@94.237.2.168
# Deploy light node to Sydney
ssh root@95.111.218.30

# Test peer discovery (mock backbone responses)
# Test direct P2P (Singapore ↔ Sydney)
# Test relay chains (if CGNAT detected)
```

**Timeline**: 1 week
**Cost**: €0 (existing nodes)

---

## Budget Management (Limited UpCloud Credits)

### Current UpCloud Spend
```
London relay: €11/month (keep running for prod mesh)
Credits: €95.73
Runway: 8-9 months

Recommendation: Don't provision new UpCloud nodes yet
Wait for AWS credits for production backbone
```

### Future AWS Spend (Once Credits Available)
```
5-7 regions × $20-30/month = $100-150/month
Covered by AWS credits
No impact on UpCloud budget
```

---

## Success Criteria

### Software Development Phase (No AWS)
- [ ] Light node daemon builds successfully
- [ ] Local Kademlia DHT cache working (100 peers)
- [ ] Backbone query protocol implemented
- [ ] Authentication protocol designed
- [ ] Web GUI functional
- [ ] NodeNexus contract deployed to Mumbai testnet

### AWS Integration Phase (After Credits)
- [ ] 5-7 AWS backbone nodes deployed
- [ ] Authentication working (light node ↔ backbone)
- [ ] Peer discovery working (local cache + backbone queries)
- [ ] Direct P2P traffic flowing (backbone not involved)
- [ ] Relay chains working for CGNAT users

### Beta Launch Phase
- [ ] 100 beta users onboarded
- [ ] Backbone handling 1000+ authentications/day
- [ ] <50ms peer discovery latency
- [ ] >95% direct P2P success rate
- [ ] Relay chains for remaining 5% CGNAT users

---

## Summary

**Corrected Architecture**:
- ✅ Backbone = Authentication + DHT lookups (AWS, pending credits)
- ✅ Light Nodes = Local cache + P2P traffic (user devices)
- ✅ Traffic = Peer-to-peer (NOT through backbone)
- ✅ Scalability = Unlimited (no backbone bottleneck)

**Current State**:
- UpCloud: Limited credits (€95.73), conserve for testing
- AWS: Waiting for credits, then deploy production backbone
- Development: Can build software NOW without cloud costs

**Next Action**: Build light node daemon + authentication protocol while waiting for AWS credits

---

**Document Created**: November 4, 2025, 11:30 UTC
**Status**: Architecture Clarified
**Review Date**: After AWS credits available
