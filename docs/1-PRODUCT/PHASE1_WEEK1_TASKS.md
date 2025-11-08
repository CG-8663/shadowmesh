# Phase 1 Week 1: Deployment Checklist

**Date**: November 4-11, 2025
**Objective**: Deploy 3 regional backbone discovery nodes and prepare light node infrastructure

---

## Day 1-2: Regional Backbone Deployment

### Task 1: Provision Infrastructure

**US East Coast** (New):
```bash
# AWS c6g.xlarge (ARM Graviton2, 4 vCPU, 8 GB RAM, 10 Gbps)
# Region: us-east-1 (Virginia)

# Provision via AWS CLI
aws ec2 run-instances \
  --image-id ami-0c55b159cbfafe1f0 \
  --instance-type c6g.xlarge \
  --key-name shadowmesh-prod \
  --security-group-ids sg-xxxxxxxxx \
  --subnet-id subnet-xxxxxxxxx \
  --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=shadowmesh-discovery-us-east-1}]'

# Assign Elastic IP
aws ec2 allocate-address --domain vpc
aws ec2 associate-address --instance-id i-xxxxxxxxx --allocation-id eipalloc-xxxxxxxxx

# Open ports: 8443 (WSS), 8080 (HTTP API)
aws ec2 authorize-security-group-ingress \
  --group-id sg-xxxxxxxxx \
  --protocol tcp \
  --port 8443 \
  --cidr 0.0.0.0/0
```

**EU West (UK)** (Existing - Repurpose shadowmesh-001):
```bash
# Already deployed: 80.229.0.71
# Convert from relay to discovery node

ssh pxcghost@80.229.0.71
cd /opt/shadowmesh

# Stop existing relay
sudo systemctl stop shadowmesh-relay

# Deploy discovery node binary
sudo ./build/shadowmesh-discovery \
  --region europe \
  --listen-addr 0.0.0.0:8443 \
  --smart-contract 0x... \
  --chain-id 80001  # Mumbai testnet
```

**EU Central (Germany)** (New):
```bash
# UpCloud Frankfurt
# Use upctl CLI or web interface

upctl server create \
  --hostname shadowmesh-discovery-eu-central \
  --zone de-fra1 \
  --plan 2xCPU-4GB \
  --os Ubuntu 22.04 \
  --ssh-keys shadowmesh-prod

# Public IP assigned automatically
# Configure firewall via UpCloud console
```

### Task 2: Deploy Discovery Node Software

```bash
# Build discovery node binary
cd /Users/jamestervit/Webcode/shadowmesh
GOOS=linux GOARCH=amd64 go build -o build/shadowmesh-discovery-amd64 ./cmd/discovery/
GOOS=linux GOARCH=arm64 go build -o build/shadowmesh-discovery-arm64 ./cmd/discovery/

# Create systemd service
cat > /tmp/shadowmesh-discovery.service <<EOF
[Unit]
Description=ShadowMesh Regional Discovery Node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/shadowmesh
ExecStart=/opt/shadowmesh/build/shadowmesh-discovery \\
  --region %REGION% \\
  --listen-addr 0.0.0.0:8443 \\
  --smart-contract %CONTRACT_ADDR% \\
  --chain-id 80001 \\
  --rpc-url https://polygon-mumbai.g.alchemy.com/v2/%API_KEY%
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Deploy to each node
for node in us-east-1 eu-west-2 eu-central-1; do
  scp build/shadowmesh-discovery-* root@$node:/opt/shadowmesh/build/
  scp /tmp/shadowmesh-discovery.service root@$node:/etc/systemd/system/
  ssh root@$node "systemctl daemon-reload && systemctl enable shadowmesh-discovery && systemctl start shadowmesh-discovery"
done
```

---

## Day 3-4: Smart Contract Deployment

### Task 3: Deploy NodeNexus Contract to Mumbai Testnet

```bash
# Install Hardhat if not already installed
cd /Users/jamestervit/Webcode/shadowmesh/contracts
npm install --save-dev hardhat @openzeppelin/contracts-upgradeable

# Initialize Hardhat project
npx hardhat init
# Choose: Create an advanced sample project

# Create deployment script
cat > scripts/deploy-nexus.ts <<EOF
import { ethers, upgrades } from "hardhat";

async function main() {
  const NodeNexus = await ethers.getContractFactory("NodeNexus");

  console.log("Deploying NodeNexus...");
  const nexus = await upgrades.deployProxy(
    NodeNexus,
    [],
    { initializer: 'initialize' }
  );

  await nexus.deployed();
  console.log("NodeNexus deployed to:", nexus.address);

  // Register backbone discovery nodes
  const backboneNodes = [
    {
      peerID: "0x...",  // Derived from node's ML-DSA-87 public key
      region: "north_america",
      endpoint: "wss://discovery-us-east-1.shadowmesh.io:8443"
    },
    {
      peerID: "0x...",
      region: "europe",
      endpoint: "wss://discovery-eu-west-2.shadowmesh.io:8443"
    },
    {
      peerID: "0x...",
      region: "europe",
      endpoint: "wss://discovery-eu-central-1.shadowmesh.io:8443"
    }
  ];

  for (const node of backboneNodes) {
    const tx = await nexus.registerNode(
      node.peerID,
      0,  // NodeType.BACKBONE_DISCOVERY
      node.region,
      0   // No bandwidth for discovery nodes
    );
    await tx.wait();
    console.log(\`Registered backbone node in \${node.region}\`);
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
EOF

# Deploy to Mumbai testnet
npx hardhat run scripts/deploy-nexus.ts --network mumbai
# Note: Save contract address from output

# Verify contract on Polygonscan
npx hardhat verify --network mumbai DEPLOYED_CONTRACT_ADDRESS
```

### Task 4: Test Contract Interaction

```bash
# Test contract via Hardhat console
npx hardhat console --network mumbai

# In console:
const NodeNexus = await ethers.getContractFactory("NodeNexus");
const nexus = await NodeNexus.attach("0x...");  // Contract address

// Get regional nodes
const nodes = await nexus.getRegionalNodes("europe");
console.log("Europe discovery nodes:", nodes);

// Check node details
const nodeDetails = await nexus.getNode(nodes[0]);
console.log("Node details:", nodeDetails);
```

---

## Day 5: Light Node Client Development

### Task 5: Implement Light Node Daemon Structure

```bash
cd /Users/jamestervit/Webcode/shadowmesh
mkdir -p cmd/lightnode pkg/lightnode

# Create main entry point
cat > cmd/lightnode/main.go <<'EOF'
package main

import (
    "flag"
    "log"

    "github.com/shadowmesh/shadowmesh/pkg/lightnode"
)

func main() {
    configPath := flag.String("config", "lightnode.yaml", "Config file path")
    flag.Parse()

    // Load configuration
    config, err := lightnode.LoadConfig(*configPath)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Create light node instance
    node := lightnode.NewLightNode(config)

    // Start node
    if err := node.Start(); err != nil {
        log.Fatalf("Failed to start light node: %v", err)
    }

    log.Println("Light node started successfully")

    // Wait for shutdown signal
    select {}
}
EOF

# Create light node package
cat > pkg/lightnode/lightnode.go <<'EOF'
package lightnode

import (
    "context"
    "log"

    "github.com/shadowmesh/shadowmesh/pkg/dht"
    "github.com/shadowmesh/shadowmesh/pkg/nattraversal"
)

type LightNode struct {
    config       *Config
    peerID       [20]byte
    region       string
    serviceType  ServiceType
    dht          *dht.KademliaNode
    natManager   *nattraversal.NATTraversalManager
    webGUI       *WebGUI
}

func NewLightNode(config *Config) *LightNode {
    return &LightNode{
        config: config,
    }
}

func (ln *LightNode) Start() error {
    log.Println("Starting light node...")

    // Step 1: Generate peer ID from ML-DSA-87 keys
    ln.peerID = ln.generatePeerID()
    log.Printf("Peer ID: %x", ln.peerID)

    // Step 2: Detect region
    ln.region = ln.detectRegion()
    log.Printf("Detected region: %s", ln.region)

    // Step 3: Detect NAT type
    natType, externalIP := ln.natManager.DetectNATType()
    log.Printf("NAT type: %s, external IP: %s", natType, externalIP)

    // Step 4: Register with smart contract
    if err := ln.registerWithNexus(); err != nil {
        return err
    }

    // Step 5: Start web GUI
    go ln.webGUI.Start(":8080")

    log.Println("Light node started")
    return nil
}

func (ln *LightNode) detectRegion() string {
    // TODO: Ping discovery nodes and select nearest
    return "north_america"
}

func (ln *LightNode) generatePeerID() [20]byte {
    // TODO: Hash ML-DSA-87 public key
    var peerID [20]byte
    return peerID
}

func (ln *LightNode) registerWithNexus() error {
    // TODO: Call smart contract registerNode()
    log.Println("Registered with NodeNexus")
    return nil
}
EOF

# Build light node binary
go build -o build/shadowmesh-lightnode ./cmd/lightnode/
```

---

## Day 6-7: Web GUI Development

### Task 6: Build Service Selection UI

```bash
# Create Next.js app for light node GUI
cd /Users/jamestervit/Webcode/shadowmesh
mkdir -p gui
cd gui

npx create-next-app@latest . --typescript --tailwind --app
# Accept defaults

# Create service selection component
mkdir -p app/components
cat > app/components/ServiceSelection.tsx <<'EOF'
'use client';

import { useState } from 'react';
import { ethers } from 'ethers';

enum ServiceType {
  PRIVATE,
  PUBLIC
}

export function ServiceSelection() {
  const [serviceType, setServiceType] = useState<ServiceType>(ServiceType.PRIVATE);
  const [bandwidth, setBandwidth] = useState(100);
  const [stake, setStake] = useState(0);

  const handleServiceChange = async (newType: ServiceType) => {
    setServiceType(newType);

    if (newType === ServiceType.PUBLIC) {
      setStake(100); // 100 SHM required
    } else {
      setStake(0);
    }

    // Call smart contract to update service type
    // TODO: Implement contract interaction
  };

  const calculateEarnings = () => {
    const utilizationRate = 0.4;
    const pricePerGB = 0.01;
    const shmPrice = 10;

    const hoursPerMonth = 720;
    const gbPerHour = (bandwidth * 0.125 * 3600) / 1024;
    const gbPerMonth = gbPerHour * hoursPerMonth * utilizationRate;

    return (gbPerMonth * pricePerGB * shmPrice).toFixed(2);
  };

  return (
    <div className="max-w-4xl mx-auto p-8">
      <h1 className="text-3xl font-bold mb-8">Configure Your ShadowMesh Node</h1>

      {/* Service Type Toggle */}
      <div className="grid grid-cols-2 gap-4 mb-8">
        <button
          className={\`p-6 rounded-lg border-2 \${
            serviceType === ServiceType.PRIVATE
              ? 'border-blue-500 bg-blue-50'
              : 'border-gray-300'
          }\`}
          onClick={() => handleServiceChange(ServiceType.PRIVATE)}
        >
          <div className="text-4xl mb-2">🔒</div>
          <h3 className="font-bold text-xl mb-2">Private Use</h3>
          <p className="text-gray-600">Use for your own VPN, messaging, file sharing</p>
          <p className="text-green-600 font-bold mt-2">Free - No stake required</p>
        </button>

        <button
          className={\`p-6 rounded-lg border-2 \${
            serviceType === ServiceType.PUBLIC
              ? 'border-blue-500 bg-blue-50'
              : 'border-gray-300'
          }\`}
          onClick={() => handleServiceChange(ServiceType.PUBLIC)}
        >
          <div className="text-4xl mb-2">🌐</div>
          <h3 className="font-bold text-xl mb-2">Public Sharing</h3>
          <p className="text-gray-600">Share bandwidth and earn SHM tokens</p>
          <p className="text-blue-600 font-bold mt-2">Earn ${calculateEarnings()}/month</p>
          <p className="text-gray-500 text-sm">Requires 100 SHM stake ($1,000)</p>
        </button>
      </div>

      {/* Public Configuration */}
      {serviceType === ServiceType.PUBLIC && (
        <div className="bg-white p-6 rounded-lg shadow mb-8">
          <h3 className="font-bold text-xl mb-4">Bandwidth Sharing Settings</h3>

          <label className="block mb-4">
            <span className="text-gray-700">Maximum Bandwidth (Mbps)</span>
            <input
              type="range"
              min="10"
              max="500"
              step="10"
              value={bandwidth}
              onChange={(e) => setBandwidth(Number(e.target.value))}
              className="w-full mt-2"
            />
            <span className="text-gray-600">{bandwidth} Mbps</span>
          </label>

          <div className="bg-gray-50 p-4 rounded">
            <h4 className="font-bold mb-2">Estimated Monthly Earnings</h4>
            <div className="text-3xl font-bold text-green-600">${calculateEarnings()}</div>
            <p className="text-sm text-gray-600 mt-1">
              Based on {bandwidth} Mbps @ 40% utilization
            </p>
          </div>
        </div>
      )}

      {/* Private Configuration */}
      {serviceType === ServiceType.PRIVATE && (
        <div className="bg-white p-6 rounded-lg shadow">
          <h3 className="font-bold text-xl mb-4">Personal Services</h3>

          <div className="space-y-3">
            <label className="flex items-center">
              <input type="checkbox" className="mr-2" defaultChecked />
              <span>Enable VPN</span>
            </label>
            <label className="flex items-center">
              <input type="checkbox" className="mr-2" defaultChecked />
              <span>Enable Private Messaging</span>
            </label>
            <label className="flex items-center">
              <input type="checkbox" className="mr-2" />
              <span>Enable File Sharing</span>
            </label>
          </div>
        </div>
      )}

      {/* Save Button */}
      <button className="w-full bg-blue-600 text-white py-3 rounded-lg font-bold mt-8 hover:bg-blue-700">
        Save Configuration
      </button>
    </div>
  );
}
EOF

# Update main page
cat > app/page.tsx <<'EOF'
import { ServiceSelection } from './components/ServiceSelection';

export default function Home() {
  return <ServiceSelection />;
}
EOF

# Start development server
npm run dev
# Access at http://localhost:3000
```

---

## Verification Checklist

### Infrastructure
- [ ] 3 regional discovery nodes deployed and running
- [ ] All nodes accessible via WSS on port 8443
- [ ] Systemd services configured for auto-restart
- [ ] Monitoring dashboards showing node health

### Smart Contract
- [ ] NodeNexus deployed to Mumbai testnet
- [ ] Contract verified on Polygonscan
- [ ] 3 backbone nodes registered in contract
- [ ] Can query regional nodes via contract

### Light Node Software
- [ ] Light node daemon builds successfully
- [ ] Can detect region by pinging discovery nodes
- [ ] NAT type detection working
- [ ] Registration with smart contract functional

### Web GUI
- [ ] GUI running on localhost:8080
- [ ] Service type toggle (PRIVATE/PUBLIC) working
- [ ] Bandwidth slider updates earnings calculation
- [ ] Can connect MetaMask wallet
- [ ] Can call smart contract to register node

---

## Success Criteria for Week 1

| Metric | Target | Status |
|--------|--------|--------|
| Regional discovery nodes deployed | 3 | ☐ |
| Smart contract deployed | 1 | ☐ |
| Light node daemon functional | Yes | ☐ |
| Web GUI accessible | Yes | ☐ |
| Beta testers recruited | 10-20 | ☐ |
| Contract registrations | 3 backbone + 10-20 light | ☐ |

---

## Next Steps (Week 2)

1. **Test relay chain construction** through PUBLIC light nodes
2. **Measure latency and throughput** for 3-hop chains
3. **Implement payment system** (state channels on testnet)
4. **Expand to 50 beta testers** (mix of PRIVATE/PUBLIC)
5. **Begin Phase 2 development** (custom DHT implementation)

---

**Document Status**: ✅ Ready for Execution
**Owner**: Protocol Team + Blockchain Team
**Timeline**: November 4-11, 2025
