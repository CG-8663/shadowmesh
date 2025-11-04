# ShadowMesh Immediate Action Plan - November 4, 2025

## Constraints

**UpCloud Credits**: €95.73 (limited, conserve for testing)
**AWS Credits**: Pending (waiting for approval)
**Timeline**: 4-6 weeks until AWS backbone ready

---

## What We Can Do NOW (Zero Cloud Cost)

### Week 1-2: Core Software Development

#### Task 1: Build Light Node Daemon (16-20 hours)

**Create directory structure**:
```bash
cd /Volumes/BACKUPDISK/webcode/shadowmesh
mkdir -p cmd/lightnode
mkdir -p pkg/lightnode
mkdir -p pkg/kademlia
mkdir -p pkg/backbone-client
mkdir -p pkg/authentication
```

**Components to build**:
1. `cmd/lightnode/main.go` - Entry point, CLI flags, config loading
2. `pkg/lightnode/lightnode.go` - Core light node logic
3. `pkg/kademlia/dht.go` - Local DHT cache (100-peer subset)
4. `pkg/backbone-client/client.go` - HTTP client for backbone queries
5. `pkg/authentication/auth.go` - ML-DSA-87 authentication protocol

**Target**: Functional light node that can:
- Generate ML-DSA-87 keys
- Authenticate (mock backbone for now)
- Maintain local Kademlia cache
- Query backbone for peer discovery (mock API)
- Attempt direct P2P connections
- Fall back to relay chains if P2P fails

---

#### Task 2: Local Kademlia DHT Implementation (8-12 hours)

**Kademlia DHT Basics**:
```go
// pkg/kademlia/dht.go

type KademliaDHT struct {
    localPeerID  [20]byte           // This node's peer ID
    buckets      [160]*KBucket      // 160 buckets (1 per bit of PeerID)
    maxPeers     int                // Max peers to cache (100 for light nodes)
}

type KBucket struct {
    peers        []*PeerInfo        // Max 20 peers per bucket
    lastUpdated  time.Time
}

type PeerInfo struct {
    PeerID       [20]byte
    Region       string
    PublicIP     net.IP
    PublicPort   uint16
    NATType      NATType
    LastSeen     time.Time
}

// XOR distance between two peer IDs
func distance(a, b [20]byte) *big.Int {
    result := new(big.Int)
    for i := 0; i < 20; i++ {
        result.SetBytes(append(result.Bytes(), a[i]^b[i]))
    }
    return result
}

// Find K closest peers to target
func (dht *KademliaDHT) FindClosest(target [20]byte, k int) []*PeerInfo {
    // Calculate distance to all cached peers
    // Return K closest
}

// Add peer to appropriate bucket
func (dht *KademliaDHT) AddPeer(peer *PeerInfo) {
    dist := distance(dht.localPeerID, peer.PeerID)
    bucketIndex := dist.BitLen() - 1
    if bucketIndex < 0 || bucketIndex >= 160 {
        return
    }

    bucket := dht.buckets[bucketIndex]
    bucket.AddPeer(peer)
}
```

**Why 100 peers limit?**
- Memory: 100 peers × 100 bytes = 10 KB (tiny)
- Coverage: 100 peers distributed across 160 buckets = good coverage
- Query rate: Cache miss ~10% → 1 backbone query per 10 lookups

---

#### Task 3: Authentication Protocol (6-8 hours)

**Design Document**: `docs/AUTHENTICATION_PROTOCOL.md`

```markdown
# ShadowMesh Authentication Protocol

## Overview
Light nodes authenticate with AWS backbone using ML-DSA-87 signatures.

## Authentication Flow

### Step 1: Request Challenge
Light Node → Backbone:
GET /api/auth/challenge

Backbone → Light Node:
{
  "challenge": "<32-byte-random-hex>",
  "timestamp": 1730718600,
  "expires_at": 1730718630
}

### Step 2: Sign Challenge
Light Node:
- signature = Sign(challenge, ML-DSA-87 private key)
- peerID = Hash(ML-DSA-87 public key)

### Step 3: Submit Authentication
Light Node → Backbone:
POST /api/auth/verify
{
  "peer_id": "<20-byte-peer-id-hex>",
  "challenge": "<32-byte-challenge-hex>",
  "signature": "<ml-dsa-87-signature-base64>",
  "public_key": "<ml-dsa-87-public-key-base64>"
}

Backbone:
- Verify signature using public key
- Verify peer_id = Hash(public_key)
- Generate session token (JWT)

Backbone → Light Node:
{
  "session_token": "<jwt-token>",
  "expires_at": 1730804200,
  "backbone_nodes": [
    {"region": "north_america", "ip": "3.80.123.45", "port": 8443},
    {"region": "europe", "ip": "18.200.45.67", "port": 8443}
  ]
}

### Step 4: Use Session Token
All subsequent queries include session token:
GET /api/peers/lookup?peer_id=<target-peer-id>
Authorization: Bearer <session-token>
```

**Implementation**: `pkg/authentication/auth.go`

---

#### Task 4: Backbone Query Protocol (4-6 hours)

**Mock Backbone Client** (for testing without AWS):
```go
// pkg/backbone-client/client.go

type BackboneClient struct {
    backboneURL  string
    sessionToken string
    httpClient   *http.Client
}

// Authenticate with backbone
func (bc *BackboneClient) Authenticate(peerID [20]byte, signingKey *mldsaKey) error {
    // Step 1: Request challenge
    challenge, err := bc.requestChallenge()
    if err != nil {
        return err
    }

    // Step 2: Sign challenge
    signature := signingKey.Sign(challenge)

    // Step 3: Submit authentication
    token, err := bc.submitAuthentication(peerID, challenge, signature, signingKey.PublicKey())
    if err != nil {
        return err
    }

    bc.sessionToken = token
    return nil
}

// Lookup peer by PeerID
func (bc *BackboneClient) LookupPeer(peerID [20]byte) (*PeerInfo, error) {
    req, _ := http.NewRequest("GET",
        fmt.Sprintf("%s/api/peers/lookup?peer_id=%x", bc.backboneURL, peerID),
        nil)
    req.Header.Set("Authorization", "Bearer "+bc.sessionToken)

    resp, err := bc.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var peer PeerInfo
    if err := json.NewDecoder(resp.Body).Decode(&peer); err != nil {
        return nil, err
    }

    return &peer, nil
}
```

**Mock Server** (for local testing):
```go
// cmd/mock-backbone/main.go

func main() {
    http.HandleFunc("/api/auth/challenge", handleChallenge)
    http.HandleFunc("/api/auth/verify", handleVerify)
    http.HandleFunc("/api/peers/lookup", handleLookup)

    log.Println("Mock backbone server starting on :8080")
    http.ListenAndServe(":8080", nil)
}

func handleLookup(w http.ResponseWriter, r *http.Request) {
    peerIDHex := r.URL.Query().Get("peer_id")
    // Return mock peer info
    json.NewEncoder(w).Encode(PeerInfo{
        PeerID:     parsePeerID(peerIDHex),
        Region:     "europe",
        PublicIP:   net.ParseIP("83.136.252.52"),
        PublicPort: 8443,
    })
}
```

---

### Week 3-4: Smart Contract & GUI Development

#### Task 5: Deploy NodeNexus Contract (4-6 hours)

**Enhance existing RelayNodeRegistry.sol → NodeNexus.sol**:
```bash
cd /Volumes/BACKUPDISK/webcode/shadowmesh/contracts
```

**Add to contract**:
```solidity
// contracts/contracts/NodeNexus.sol

contract NodeNexus is Ownable, UUPSUpgradeable {
    enum NodeType {
        BACKBONE_AUTH,    // AWS authentication servers
        LIGHT_PRIVATE,    // User node, private use
        LIGHT_PUBLIC      // User node, sharing bandwidth
    }

    struct Node {
        bytes20 peerID;
        NodeType nodeType;
        string region;
        address operator;
        uint256 bandwidthMbps;
        uint256 stake;
        uint256 lastSeen;
        bool active;
    }

    mapping(bytes20 => Node) public nodes;
    mapping(string => bytes20[]) public regionalNodes;

    function registerNode(
        bytes20 peerID,
        NodeType nodeType,
        string memory region,
        uint256 bandwidthMbps
    ) external payable {
        // Require stake for PUBLIC nodes
        if (nodeType == NodeType.LIGHT_PUBLIC) {
            require(msg.value >= 100 ether, "Stake 100 SHM required");
        }

        nodes[peerID] = Node({
            peerID: peerID,
            nodeType: nodeType,
            region: region,
            operator: msg.sender,
            bandwidthMbps: bandwidthMbps,
            stake: msg.value,
            lastSeen: block.timestamp,
            active: true
        });

        regionalNodes[region].push(peerID);
        emit NodeRegistered(peerID, nodeType, region);
    }

    function getBackboneNodes(string memory region)
        external view returns (bytes20[] memory) {
        // Filter by BACKBONE_AUTH type
    }

    function getPublicLightNodes(string memory region, uint256 count)
        external view returns (bytes20[] memory) {
        // Filter by LIGHT_PUBLIC type, return top N by reputation
    }
}
```

**Deploy to Polygon Mumbai**:
```bash
npx hardhat run scripts/deploy-nexus.ts --network mumbai
npx hardhat verify --network mumbai <deployed-address>
```

---

#### Task 6: Web GUI for Service Selection (8-12 hours)

**Create Next.js app**:
```bash
cd /Volumes/BACKUPDISK/webcode/shadowmesh
mkdir -p gui
cd gui
npx create-next-app@latest . --typescript --tailwind --app
```

**Key Components**:
1. Service type toggle (PRIVATE/PUBLIC)
2. Bandwidth slider (if PUBLIC)
3. Earnings calculator
4. MetaMask integration
5. Smart contract interaction

**See**: `docs/PHASE1_WEEK1_TASKS.md` (lines 336-498) for full GUI code

---

### Week 5-6: Testing (Using Existing UpCloud Nodes)

#### Task 7: Test Light Node Software (1 week)

**Use Singapore + Sydney servers** (already paid for):
```bash
# Deploy to Singapore
scp build/shadowmesh-lightnode root@94.237.2.168:/usr/local/bin/
ssh root@94.237.2.168

# Configure
cat > /etc/shadowmesh/lightnode.yaml <<EOF
mode: lightnode
peer_id_file: /var/lib/shadowmesh/peer_id.txt
backbone_url: http://mock-backbone.local:8080  # Mock for now
region: asia_pacific
service_type: private
EOF

# Start
systemctl start shadowmesh-lightnode
```

**Test scenarios**:
1. Local Kademlia cache (100 peers)
2. Backbone queries (mock API)
3. Direct P2P (Singapore ↔ Sydney)
4. Relay chains (if NAT detected)

**Cost**: €0 (using existing servers)

---

## What We WAIT For (AWS Credits)

### AWS Backbone Deployment (1-2 weeks after credits)

**Provision 5-7 regions**:
```bash
# Example: Deploy to us-east-1
aws ec2 run-instances \
  --image-id ami-0c55b159cbfafe1f0 \
  --instance-type t3.medium \
  --key-name shadowmesh-prod \
  --security-group-ids sg-xxxxxxxxx \
  --user-data file://backbone-cloud-init.sh \
  --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=shadowmesh-backbone-us-east-1}]'
```

**Deploy backbone software**:
```bash
# On each AWS instance
# Install:
# - Authentication server
# - Kademlia DHT server (full registry)
# - Redis (in-memory cache)
# - PostgreSQL (persistent storage)
# - Nginx (load balancer)
```

**Timeline**: 1-2 weeks after AWS credits available
**Cost**: $100-150/month (covered by credits)

---

## Timeline Summary

| Week | Tasks | Cloud Cost | Status |
|------|-------|------------|--------|
| 1-2 | Light node daemon, Kademlia DHT, Authentication | €0 | ✅ Can do now |
| 3-4 | Smart contract, GUI development | €0 | ✅ Can do now |
| 5-6 | Test on UpCloud Singapore/Sydney | €0 | ✅ Can do now |
| 7-8 | AWS backbone deployment | $100-150 | ⏳ Waiting for credits |
| 9-10 | Integration testing (backbone + light nodes) | $150 | ⏳ After AWS ready |
| 11-12 | Beta launch (100 users) | $200 | ⏳ After integration |

---

## Success Metrics

### Phase 1 (Weeks 1-6, No AWS)
- [ ] Light node daemon builds successfully
- [ ] Local Kademlia DHT caches 100 peers
- [ ] Mock backbone client working
- [ ] Authentication protocol implemented
- [ ] Smart contract deployed to Mumbai testnet
- [ ] Web GUI functional
- [ ] Tested on Singapore + Sydney servers

### Phase 2 (Weeks 7-8, AWS Credits)
- [ ] 5-7 AWS backbone nodes deployed
- [ ] Authentication server operational
- [ ] Full Kademlia DHT server operational
- [ ] Light nodes connecting to real backbone
- [ ] Peer discovery working (<50ms latency)

### Phase 3 (Weeks 9-12, Integration)
- [ ] 100 beta users onboarded
- [ ] Direct P2P working (>95% success rate)
- [ ] Relay chains working for CGNAT users
- [ ] Backbone handling 1000+ auth/day
- [ ] Zero backbone traffic bottlenecks

---

## Budget Management

### UpCloud (Current)
```
Spend: €11/month (London relay only)
Credits: €95.73
Runway: 8-9 months
Action: Conserve credits, minimal testing
```

### AWS (Future)
```
Spend: $100-150/month (5-7 backbone nodes)
Credits: Pending approval
Action: Deploy backbone once credits available
```

---

## Next Steps (This Week)

1. **Create directory structure** for light node, Kademlia, backbone client
2. **Implement Kademlia DHT** with 100-peer local cache
3. **Build authentication protocol** (challenge-response with ML-DSA-87)
4. **Create mock backbone server** for local testing
5. **Design backbone query API** (REST endpoints)

**I'm ready to start building. Which component should I tackle first?**
- A) Kademlia DHT implementation
- B) Authentication protocol
- C) Light node daemon skeleton
- D) Mock backbone server for testing

Let me know and I'll begin implementation immediately.

---

**Document Created**: November 4, 2025, 11:40 UTC
**Status**: Ready to Execute
**Next Action**: Your decision on which component to build first
