# ShadowMesh Regional Light Node Architecture

**Document Type**: Technical Architecture Specification
**Date**: November 4, 2025
**Status**: Approved Design
**Key Insight**: CGNAT-aware design with regional discovery backbone

---

## Executive Summary

ShadowMesh deploys a **two-tier architecture** optimized for the reality that most user nodes operate behind **CGNAT (Carrier-Grade NAT)**:

1. **Regional Backbone Discovery Nodes**: Company-operated nodes providing DHT bootstrap and regional peer discovery
2. **User Light Nodes**: Bandwidth-sharing nodes behind CGNAT, discovered via regional backbone

This design **eliminates the need for complex STUN/TURN** by:
- Accepting that CGNAT prevents direct P2P (no fighting physics)
- Using light nodes as **relays** rather than direct P2P endpoints
- Routing traffic through relay chains selected by smart contract nexus
- Allowing users to choose **public bandwidth sharing** vs **private personal use** via GUI

**Result**: Massively scalable network without complex NAT traversal engineering.

---

## 1. Regional Backbone Discovery Nodes

### Purpose

**NOT full relays** - they are **discovery coordinators** that:
1. Bootstrap DHT for new nodes joining the network
2. Maintain regional peer registries
3. Coordinate relay chain construction
4. Monitor network health and partition detection

### Regional Deployment

```yaml
# Regional backbone structure (Phase 1-2)
backbone_regions:
  north_america:
    discovery_nodes:
      - us-east-1: AWS c6g.xlarge (Virginia)
      - us-west-1: AWS c6g.xlarge (California)
    coverage: USA, Canada, Mexico
    light_nodes_expected: 500-1000

  europe:
    discovery_nodes:
      - eu-west-2: UpCloud (London, UK) - shadowmesh-001
      - eu-central-1: UpCloud (Frankfurt, Germany)
    coverage: UK, Germany, France, Netherlands, Belgium
    light_nodes_expected: 500-1000

  asia_pacific:
    discovery_nodes:
      - ap-southeast-1: AWS c6g.xlarge (Singapore)
      - ap-northeast-1: AWS c6g.xlarge (Tokyo)
    coverage: Singapore, Japan, Australia
    light_nodes_expected: 200-500
```

### Discovery Node Responsibilities

```go
// Regional discovery node implementation
type RegionalDiscoveryNode struct {
    Region           string  // "north_america", "europe", "asia_pacific"
    DHT              *KademliaNode
    SmartContract    *EthereumClient  // Read nexus registry
    LightNodeRegistry map[PeerID]*LightNodeInfo
    Stats            *RegionalStats
}

type LightNodeInfo struct {
    PeerID           PeerID
    Region           string
    PublicIP         net.IP
    NATType          NATType  // "CGNAT", "Symmetric", "Full Cone", "Restricted"
    BandwidthOffered int      // Mbps user is sharing
    ServiceType      ServiceType  // "PUBLIC" or "PRIVATE"
    Reputation       float64  // 0.0-1.0
    LastSeen         time.Time
}

type ServiceType int
const (
    PRIVATE ServiceType = iota  // Personal use only (not sharing bandwidth)
    PUBLIC                      // Sharing bandwidth with network
)

// Discovery node operations
func (dn *RegionalDiscoveryNode) RegisterLightNode(node *LightNodeInfo) error {
    // Step 1: Verify node is in correct region (latency check)
    if !dn.verifyRegion(node.PublicIP) {
        return errors.New("node not in this region")
    }

    // Step 2: Record in smart contract nexus
    tx := dn.SmartContract.RegisterNode(
        node.PeerID,
        node.Region,
        node.ServiceType,
        node.BandwidthOffered,
    )

    // Step 3: Add to local DHT routing table
    dn.DHT.AddPeer(node.PeerID, node.PublicIP)

    // Step 4: Add to regional registry
    dn.LightNodeRegistry[node.PeerID] = node

    log.Printf("Registered light node %s in region %s (type: %s, bandwidth: %d Mbps)",
        node.PeerID, dn.Region, node.ServiceType, node.BandwidthOffered)

    return nil
}

// Provide list of nearby light nodes for relay chain construction
func (dn *RegionalDiscoveryNode) FindNearbyLightNodes(
    clientIP net.IP,
    count int,
    serviceType ServiceType,
) []*LightNodeInfo {
    // Step 1: Filter by service type (PUBLIC only if sharing bandwidth)
    eligible := dn.filterByServiceType(serviceType)

    // Step 2: Sort by geographic proximity (GeoIP lookup)
    sorted := dn.sortByProximity(clientIP, eligible)

    // Step 3: Filter by reputation (>0.7 required)
    filtered := dn.filterByReputation(sorted, 0.7)

    // Step 4: Return top N nodes
    if len(filtered) > count {
        return filtered[:count]
    }
    return filtered
}
```

**Cost per Regional Discovery Node**: $500/month (c6g.xlarge + bandwidth)
**Total Cost (6 discovery nodes)**: $3K/month

---

## 2. User Light Nodes (CGNAT-Aware Design)

### Why CGNAT Matters

**Carrier-Grade NAT (CGNAT)** is deployed by most ISPs:
- Mobile networks: 95%+ behind CGNAT
- Home ISPs: 60-80% behind CGNAT (especially fiber/cable)
- Enterprise networks: 90%+ behind corporate NAT

**Result**: Direct P2P is **impossible** for most users. Accept this reality and design around it.

### Light Node Architecture

```go
// User light node running on consumer hardware
type LightNode struct {
    PeerID           PeerID
    Region           string  // Detected from nearest discovery node
    ServiceConfig    ServiceConfig
    NATTraversal     *NATTraversalManager
    RelayChain       *RelayChainManager
    GUI              *WebGUI
}

type ServiceConfig struct {
    // User-configurable via GUI
    ServiceType      ServiceType  // PUBLIC (share bandwidth) or PRIVATE (personal only)
    MaxBandwidth     int          // Mbps to share (if PUBLIC)
    MaxStorage       int          // GB for DHT storage (if PUBLIC)
    AllowedRegions   []string     // Which regions can use this node as relay
    PricingPerGB     float64      // SHM tokens per GB (if PUBLIC)

    // Private services (user's own traffic)
    EnableVPN        bool
    EnableMessaging  bool
    EnableFileshare  bool
}

// Light node startup sequence
func (ln *LightNode) Start() error {
    // Step 1: Detect region by pinging discovery nodes
    ln.Region = ln.detectRegion()

    // Step 2: Detect NAT type (CGNAT, Symmetric, Full Cone, etc.)
    natType, externalIP := ln.NATTraversal.DetectNATType()
    log.Printf("NAT type: %s, external IP: %s", natType, externalIP)

    // Step 3: Register with regional discovery node
    discoveryNode := ln.findDiscoveryNode(ln.Region)
    err := discoveryNode.RegisterLightNode(&LightNodeInfo{
        PeerID:           ln.PeerID,
        Region:           ln.Region,
        PublicIP:         externalIP,
        NATType:          natType,
        BandwidthOffered: ln.ServiceConfig.MaxBandwidth,
        ServiceType:      ln.ServiceConfig.ServiceType,
    })

    if err != nil {
        return fmt.Errorf("failed to register with discovery node: %w", err)
    }

    // Step 4: Start relay chain listener (if PUBLIC)
    if ln.ServiceConfig.ServiceType == PUBLIC {
        ln.RelayChain.StartListener()
        log.Printf("Light node is PUBLIC - accepting relay traffic")
    } else {
        log.Printf("Light node is PRIVATE - personal use only")
    }

    // Step 5: Start web GUI for configuration
    ln.GUI.Start(":8080")

    return nil
}
```

### CGNAT Traversal Strategy

**Key Insight**: Don't fight CGNAT - embrace it.

```go
// NAT traversal for CGNAT environments
type NATTraversalManager struct {
    discoveryNode *RegionalDiscoveryNode
    natType       NATType
    externalAddr  net.Addr
}

func (ntm *NATTraversalManager) DetectNATType() (NATType, net.IP) {
    // Step 1: Send UDP packets to discovery node from multiple local ports
    responses := ntm.sendFromMultiplePorts()

    // Step 2: Discovery node reports external IP:port mappings
    // - Full Cone NAT: Same external port for all destinations
    // - Restricted NAT: Same external port, but destination-restricted
    // - Port Restricted: Same as Restricted, port-restricted too
    // - Symmetric NAT: Different external port for each destination
    // - CGNAT: External IP is shared across multiple users

    if ntm.isSharedExternalIP(responses) {
        return CGNAT, responses[0].ExternalIP
    }

    if ntm.isDifferentPortPerDestination(responses) {
        return SymmetricNAT, responses[0].ExternalIP
    }

    return FullConeNAT, responses[0].ExternalIP
}

// For CGNAT nodes: Use relay chains instead of direct P2P
func (ntm *NATTraversalManager) BuildRelayChain(
    destination PeerID,
    hopCount int,
) (*RelayChain, error) {
    // Step 1: Query smart contract nexus for available relays
    availableRelays := ntm.discoveryNode.FindNearbyLightNodes(
        ntm.externalAddr.(*net.IPAddr).IP,
        hopCount * 3,  // Get 3x candidates per hop
        PUBLIC,        // Only PUBLIC nodes can be relays
    )

    // Step 2: Select relays based on:
    // - Geographic diversity (not all in same datacenter)
    // - Reputation score (>0.8 preferred)
    // - Latency (prefer <50ms per hop)
    selectedRelays := ntm.selectRelays(availableRelays, hopCount)

    // Step 3: Construct relay chain
    chain := &RelayChain{
        Hops:    selectedRelays,
        Session: generateSessionID(),
    }

    // Step 4: Establish encrypted tunnels through each hop
    for i, relay := range selectedRelays {
        err := chain.EstablishHop(i, relay)
        if err != nil {
            return nil, fmt.Errorf("failed to establish hop %d: %w", i, err)
        }
    }

    log.Printf("Built relay chain with %d hops: %v", hopCount, chain.HopIDs())
    return chain, nil
}
```

**Performance**:
- 3-hop relay chain: +15-30ms latency (acceptable for VPN)
- Throughput: Limited by slowest relay (typically 50-100 Mbps)
- Reliability: 99.5% (if any relay fails, rebuild chain)

---

## 3. Smart Contract Nexus (Registry and Orchestration)

### Nexus Architecture

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";

/**
 * @title NodeNexus
 * @notice Central registry for all ShadowMesh nodes (backbone + light nodes)
 * @dev Controls discovery, authentication, and service routing
 */
contract NodeNexus is UUPSUpgradeable {
    // Node types
    enum NodeType {
        BACKBONE_DISCOVERY,  // Company-operated regional discovery
        LIGHT_PUBLIC,        // User node sharing bandwidth
        LIGHT_PRIVATE        // User node for personal use only
    }

    enum ServiceVisibility {
        PRIVATE,  // Personal use only (not discoverable)
        PUBLIC    // Discoverable and available for relay chains
    }

    // Node registry entry
    struct Node {
        bytes20 peerID;          // Hash of ML-DSA-87 public key
        NodeType nodeType;
        string region;           // "north_america", "europe", "asia_pacific"
        address operator;        // Ethereum address of node operator
        uint256 stake;           // SHM tokens staked (0 for PRIVATE nodes)
        uint256 bandwidthMbps;   // Bandwidth offered (0 for PRIVATE)
        uint256 pricePerGB;      // Price in SHM tokens (18 decimals)
        ServiceVisibility visibility;
        uint256 reputation;      // Reputation score (0-1000, scaled 3 decimals)
        uint256 lastAttestation; // Timestamp of last TPM attestation
        bool active;             // Is node currently online?
    }

    // Regional discovery node mapping
    mapping(string => bytes20[]) public regionalDiscovery;

    // Node registry
    mapping(bytes20 => Node) public nodes;

    // User service selection
    mapping(address => ServiceVisibility) public userServiceType;

    // Events
    event NodeRegistered(bytes20 indexed peerID, NodeType nodeType, string region);
    event NodeUpdated(bytes20 indexed peerID, ServiceVisibility visibility);
    event ServiceTypeChanged(address indexed user, ServiceVisibility oldType, ServiceVisibility newType);

    /**
     * @notice Register a new node in the nexus
     * @param peerID Hash of ML-DSA-87 public key
     * @param nodeType Type of node (BACKBONE, LIGHT_PUBLIC, LIGHT_PRIVATE)
     * @param region Geographic region
     * @param bandwidthMbps Bandwidth offered (0 for PRIVATE)
     */
    function registerNode(
        bytes20 peerID,
        NodeType nodeType,
        string memory region,
        uint256 bandwidthMbps
    ) external payable {
        require(nodes[peerID].operator == address(0), "Node already registered");

        // Determine service visibility
        ServiceVisibility visibility = (nodeType == NodeType.LIGHT_PRIVATE)
            ? ServiceVisibility.PRIVATE
            : ServiceVisibility.PUBLIC;

        // Stake requirements
        uint256 requiredStake = 0;
        if (nodeType == NodeType.LIGHT_PUBLIC) {
            requiredStake = 100 * 10**18;  // 100 SHM for PUBLIC nodes
        }
        require(msg.value >= requiredStake, "Insufficient stake");

        // Create node entry
        nodes[peerID] = Node({
            peerID: peerID,
            nodeType: nodeType,
            region: region,
            operator: msg.sender,
            stake: msg.value,
            bandwidthMbps: bandwidthMbps,
            pricePerGB: 0.01 * 10**18,  // Default 0.01 SHM per GB
            visibility: visibility,
            reputation: 500,  // Start at 50% reputation
            lastAttestation: block.timestamp,
            active: true
        });

        // Add to regional discovery if PUBLIC
        if (visibility == ServiceVisibility.PUBLIC) {
            regionalDiscovery[region].push(peerID);
        }

        emit NodeRegistered(peerID, nodeType, region);
    }

    /**
     * @notice User changes service type between PRIVATE and PUBLIC
     * @param newType New service visibility
     */
    function changeServiceType(ServiceVisibility newType) external {
        ServiceVisibility oldType = userServiceType[msg.sender];
        require(oldType != newType, "Already set to this type");

        userServiceType[msg.sender] = newType;

        // Update all nodes operated by this user
        // Note: Would need to track operator->nodes mapping in production
        emit ServiceTypeChanged(msg.sender, oldType, newType);
    }

    /**
     * @notice Get all PUBLIC nodes in a region for relay chain construction
     * @param region Geographic region
     * @return Array of peer IDs
     */
    function getRegionalNodes(string memory region)
        external
        view
        returns (bytes20[] memory)
    {
        return regionalDiscovery[region];
    }

    /**
     * @notice Get node details
     * @param peerID Node peer ID
     * @return Node struct
     */
    function getNode(bytes20 peerID) external view returns (Node memory) {
        return nodes[peerID];
    }

    /**
     * @notice Update node reputation (called by reputation oracle)
     * @param peerID Node to update
     * @param newReputation New reputation score (0-1000)
     */
    function updateReputation(bytes20 peerID, uint256 newReputation)
        external
        onlyRole(REPUTATION_ORACLE_ROLE)
    {
        require(newReputation <= 1000, "Invalid reputation");
        nodes[peerID].reputation = newReputation;
    }
}
```

### GUI-Driven Service Selection

```typescript
// Web GUI for user service configuration
interface ServiceSettings {
    // Primary toggle: PUBLIC or PRIVATE
    serviceType: 'PRIVATE' | 'PUBLIC';

    // If PUBLIC: Bandwidth sharing settings
    publicSettings?: {
        maxBandwidthMbps: number;      // Slider: 10, 50, 100, 500
        pricePerGB: number;            // SHM tokens (default: 0.01)
        acceptedRegions: string[];     // Which regions can use this relay
        maxConcurrentUsers: number;    // How many users can relay through this node
    };

    // If PRIVATE: Personal services
    privateSettings?: {
        enableVPN: boolean;            // Use for personal VPN
        enableMessaging: boolean;      // E2E encrypted messaging
        enableFileSharing: boolean;    // Private file sync
    };

    // Common settings
    region: string;                    // Auto-detected, can override
    stake: number;                     // SHM tokens staked (0 for PRIVATE)
}

// React component for service selection
export function ServiceSelectionPanel() {
    const [serviceType, setServiceType] = useState<'PRIVATE' | 'PUBLIC'>('PRIVATE');
    const [settings, setSettings] = useState<ServiceSettings>({
        serviceType: 'PRIVATE',
        region: 'north_america',
        stake: 0,
    });

    const handleServiceTypeChange = async (newType: 'PRIVATE' | 'PUBLIC') => {
        // Step 1: Update local state
        setServiceType(newType);

        // Step 2: Call smart contract to update nexus
        const contract = new ethers.Contract(NEXUS_ADDRESS, NEXUS_ABI, signer);
        const tx = await contract.changeServiceType(
            newType === 'PUBLIC'
                ? ServiceVisibility.PUBLIC
                : ServiceVisibility.PRIVATE
        );
        await tx.wait();

        // Step 3: Notify light node daemon to reconfigure
        await fetch('http://localhost:8080/api/configure', {
            method: 'POST',
            body: JSON.stringify(settings),
        });

        toast.success(`Service type changed to ${newType}`);
    };

    return (
        <div className="service-selection">
            <h2>Choose Your Service Type</h2>

            {/* Toggle between PRIVATE and PUBLIC */}
            <div className="toggle-group">
                <button
                    className={serviceType === 'PRIVATE' ? 'active' : ''}
                    onClick={() => handleServiceTypeChange('PRIVATE')}
                >
                    🔒 Private Use
                    <p>Use ShadowMesh for your own VPN, messaging, file sharing</p>
                    <p className="cost">Free (no stake required)</p>
                </button>

                <button
                    className={serviceType === 'PUBLIC' ? 'active' : ''}
                    onClick={() => handleServiceTypeChange('PUBLIC')}
                >
                    🌐 Public Sharing
                    <p>Share your bandwidth and earn SHM tokens</p>
                    <p className="earnings">Earn $50-200/month</p>
                    <p className="cost">Requires 100 SHM stake ($1,000)</p>
                </button>
            </div>

            {/* Conditional settings based on service type */}
            {serviceType === 'PUBLIC' && (
                <PublicServiceSettings settings={settings} onChange={setSettings} />
            )}

            {serviceType === 'PRIVATE' && (
                <PrivateServiceSettings settings={settings} onChange={setSettings} />
            )}

            {/* Estimated earnings calculator (if PUBLIC) */}
            {serviceType === 'PUBLIC' && (
                <EarningsCalculator bandwidthMbps={settings.publicSettings?.maxBandwidthMbps || 100} />
            )}
        </div>
    );
}

// Earnings calculator component
function EarningsCalculator({ bandwidthMbps }: { bandwidthMbps: number }) {
    const utilizationRate = 0.4; // Assume 40% average utilization
    const pricePerGB = 0.01; // SHM
    const shmPrice = 10; // USD

    const hoursPerMonth = 720;
    const gbPerHour = (bandwidthMbps * 0.125 * 3600) / 1024;
    const gbPerMonth = gbPerHour * hoursPerMonth * utilizationRate;
    const shmEarnings = gbPerMonth * pricePerGB;
    const usdEarnings = shmEarnings * shmPrice;

    return (
        <div className="earnings-card">
            <h3>Estimated Monthly Earnings</h3>
            <div className="earnings-breakdown">
                <div>Bandwidth: {bandwidthMbps} Mbps</div>
                <div>Utilization: {utilizationRate * 100}% (average)</div>
                <div>Data Served: {gbPerMonth.toFixed(0)} GB/month</div>
                <div className="primary">
                    Earnings: {shmEarnings.toFixed(1)} SHM (${usdEarnings.toFixed(2)}/month)
                </div>
            </div>
        </div>
    );
}
```

---

## 4. Traffic Routing Architecture

### Relay Chain Construction

```go
// Client builds relay chain through PUBLIC light nodes
type RelayChainBuilder struct {
    smartContract *NodeNexusContract
    region        string
    hopCount      int  // Default: 3 hops
}

func (rcb *RelayChainBuilder) BuildChain(destination PeerID) (*RelayChain, error) {
    // Step 1: Query smart contract for available PUBLIC nodes in region
    publicNodes, err := rcb.smartContract.GetRegionalNodes(rcb.region)
    if err != nil {
        return nil, fmt.Errorf("failed to get regional nodes: %w", err)
    }

    // Step 2: Filter by reputation and availability
    eligibleNodes := rcb.filterEligible(publicNodes)
    if len(eligibleNodes) < rcb.hopCount {
        return nil, errors.New("insufficient relay nodes available")
    }

    // Step 3: Select geographically diverse relays
    selectedRelays := rcb.selectDiverseRelays(eligibleNodes, rcb.hopCount)

    // Step 4: Construct encrypted relay chain
    chain := &RelayChain{
        ClientID:    rcb.clientID,
        Destination: destination,
        Hops:        make([]*RelayHop, rcb.hopCount),
        TotalCost:   0,
    }

    for i, relay := range selectedRelays {
        // Establish tunnel to this hop
        tunnel, err := rcb.establishTunnel(relay)
        if err != nil {
            return nil, fmt.Errorf("failed to establish hop %d: %w", i, err)
        }

        chain.Hops[i] = &RelayHop{
            Relay:  relay,
            Tunnel: tunnel,
        }

        // Calculate payment for this relay (pay per GB used)
        chain.TotalCost += relay.PricePerGB
    }

    log.Printf("Built relay chain: %s", chain.String())
    return chain, nil
}

// Example relay chain for CGNAT user
/*
Client (CGNAT) → Relay 1 (PUBLIC, UK) → Relay 2 (PUBLIC, France) → Relay 3 (PUBLIC, Germany) → Destination

Each hop encrypted with post-quantum crypto (ML-KEM-1024)
Total latency: ~45ms (15ms per hop)
Total cost: 0.03 SHM per GB (0.01 per relay × 3 hops)
*/
```

### Service Routing Logic

```go
// Route traffic based on user's service selection
func (ln *LightNode) RouteTraffic(packet *Packet) error {
    // Check service type from smart contract
    serviceType := ln.ServiceConfig.ServiceType

    switch serviceType {
    case PRIVATE:
        // PRIVATE mode: Route through backbone discovery nodes only
        // User is NOT sharing bandwidth, so cannot use other users' nodes
        return ln.routeThroughBackbone(packet)

    case PUBLIC:
        // PUBLIC mode: Route through other PUBLIC light nodes
        // User is sharing bandwidth, so can use peer network
        return ln.routeThroughPeerNetwork(packet)
    }
}

func (ln *LightNode) routeThroughBackbone(packet *Packet) error {
    // Step 1: Find nearest backbone discovery node
    discoveryNode := ln.findNearestDiscoveryNode()

    // Step 2: Send packet through backbone
    // (Backbone has direct internet connectivity, not behind CGNAT)
    return discoveryNode.Forward(packet)
}

func (ln *LightNode) routeThroughPeerNetwork(packet *Packet) error {
    // Step 1: Query smart contract for PUBLIC relay nodes
    relays := ln.queryPublicRelays(ln.Region, 3)

    // Step 2: Build relay chain
    chain := ln.buildRelayChain(relays)

    // Step 3: Route packet through chain
    return chain.Forward(packet)
}
```

---

## 5. Scalability Analysis

### Network Growth Projections

| Metric | Phase 1 (Week 4) | Phase 3 (Week 24) | Phase 4 (Week 39) | Year 2 | Year 3 |
|--------|------------------|-------------------|-------------------|--------|--------|
| **Backbone Discovery Nodes** | 3 | 6 | 8 | 12 | 20 |
| **PRIVATE Light Nodes** | 80 | 300 | 1200 | 5000 | 15000 |
| **PUBLIC Light Nodes** | 20 | 200 | 800 | 3000 | 10000 |
| **Total Active Nodes** | 103 | 506 | 2008 | 8012 | 25020 |
| **Aggregate Bandwidth (Gbps)** | 2 | 20 | 80 | 300 | 1000 |

### Cost Efficiency vs. Traditional Relays

**Traditional Relay Infrastructure** (centralized):
- 1000 users @ 100 Mbps average = 100 Gbps aggregate
- Relay servers: 10× AWS c6g.4xlarge @ $500/month = $5K/month
- Bandwidth costs: 100 TB/month @ $0.05/GB = $5K/month
- **Total: $10K/month for 1000 users = $10/user/month**

**ShadowMesh Light Node Network** (decentralized):
- 1000 users @ 100 Mbps average = 100 Gbps aggregate
- Backbone discovery: 6 nodes @ $500/month = $3K/month
- Light node bandwidth: Provided by users (FREE to network)
- Payment to light node operators: 100 TB × 0.01 SHM/GB × $10/SHM = $100/month
- **Total: $3.1K/month for 1000 users = $3.1/user/month**

**Cost Savings**: 68% cheaper than traditional relays

### CGNAT Impact on Adoption

| User Segment | CGNAT Prevalence | Can Be PUBLIC Relay? | Network Contribution |
|-------------|------------------|----------------------|---------------------|
| **Mobile Users** | 95% | ❌ No (CGNAT blocks inbound) | PRIVATE only (consumer) |
| **Home Fiber/Cable** | 60% | ⚠️ Some (depends on ISP) | 40% PUBLIC, 60% PRIVATE |
| **Home DSL** | 40% | ✅ Yes (direct public IP) | 60% PUBLIC, 40% PRIVATE |
| **Business/Datacenter** | 10% | ✅ Yes (static public IP) | 90% PUBLIC, 10% PRIVATE |

**Estimated PUBLIC vs. PRIVATE Ratio**: 30% PUBLIC, 70% PRIVATE
- 30% PUBLIC nodes provide relay infrastructure
- 70% PRIVATE nodes consume bandwidth
- Ratio is sustainable as long as PUBLIC nodes are well-compensated

---

## 6. Phase 1 Implementation Plan

### Week 1-2: Deploy Regional Backbone

1. **Provision 3 regional discovery nodes**:
   - [ ] US East (AWS us-east-1): Already tested in Epic 2
   - [ ] EU West (UpCloud London): shadowmesh-001 repurposed
   - [ ] EU Central (UpCloud Frankfurt): New deployment

2. **Configure discovery protocol**:
   - [ ] Implement regional DHT bootstrap
   - [ ] Set up peer registry with smart contract sync
   - [ ] Deploy NAT type detection service

3. **Smart contract deployment**:
   - [ ] Deploy NodeNexus.sol to Polygon Mumbai testnet
   - [ ] Configure regional discovery node addresses
   - [ ] Set initial stake requirements (0 for PRIVATE, 100 SHM for PUBLIC)

### Week 3-4: Light Node Client Development

1. **Build light node daemon**:
   - [ ] Auto-detect region (ping discovery nodes)
   - [ ] NAT type detection (CGNAT, Symmetric, Full Cone)
   - [ ] Register with smart contract nexus
   - [ ] Implement relay chain construction

2. **Build web GUI**:
   - [ ] Service type toggle (PRIVATE / PUBLIC)
   - [ ] Bandwidth sharing configuration
   - [ ] Earnings calculator
   - [ ] Real-time stats dashboard

3. **Beta testing**:
   - [ ] Recruit 100 users (80 PRIVATE, 20 PUBLIC)
   - [ ] Test relay chain construction under various NAT types
   - [ ] Measure latency and throughput

---

## 7. Success Metrics

### Technical Metrics

| Metric | Target | Measurement Method |
|--------|--------|--------------------|
| **Discovery latency** | <500ms | Time to find nearest discovery node |
| **Relay chain construction time** | <2 seconds | Time to build 3-hop chain |
| **Throughput (3-hop chain)** | >50 Mbps | iperf3 through relay chain |
| **CGNAT traversal success** | 100% | All CGNAT users can connect via relays |
| **Light node uptime** | >95% | Percentage of time node is active |

### Economic Metrics

| Metric | Phase 1 | Phase 3 | Phase 4 |
|--------|---------|---------|---------|
| **PUBLIC node earnings** | $50/month | $100/month | $200/month |
| **Network cost per user** | $5/user | $3/user | $2/user |
| **Payment to light nodes** | $100/month | $10K/month | $50K/month |
| **Net margin** | 50% | 60% | 70% |

---

## Conclusion

The **regional backbone + light node architecture** with **CGNAT-aware relay chains** provides:

1. **Scalability**: Light nodes scale horizontally without infrastructure investment
2. **Cost efficiency**: 68% cheaper than traditional relay infrastructure
3. **CGNAT compatibility**: 100% connectivity even for mobile/CGNAT users
4. **User choice**: PRIVATE (personal use) vs. PUBLIC (earn tokens) via GUI
5. **Smart contract orchestration**: Nexus controls discovery, routing, payments

This design **eliminates complex NAT traversal** by accepting CGNAT reality and routing through relay chains instead of fighting for direct P2P. Result: **massively scalable quantum-safe network** with minimal operational complexity.

**Next Steps**: Proceed with Phase 1 deployment (Weeks 1-4) per roadmap.

---

**Document Status**: ✅ Architecture Approved
**Implementation**: Starting Week 1
**Owner**: Protocol Team + Blockchain Team
