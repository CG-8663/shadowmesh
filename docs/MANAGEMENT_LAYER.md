# ShadowMesh Management Layer Architecture

## Version: v0.3.0 (Planned)

This document describes the architecture for the ShadowMesh Management Layer, which enables user-controlled private networks, local controllers, P2P direct connections, and centralized network management.

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture Components](#architecture-components)
3. [Network Topology Modes](#network-topology-modes)
4. [Local Controller Design](#local-controller-design)
5. [P2P Direct Connection Logic](#p2p-direct-connection-logic)
6. [Web Management UI](#web-management-ui)
7. [Authentication & Authorization](#authentication--authorization)
8. [Network Isolation Model](#network-isolation-model)
9. [API Specifications](#api-specifications)
10. [Database Schema](#database-schema)
11. [Deployment Models](#deployment-models)
12. [Migration from v0.2.0](#migration-from-v020)

---

## Overview

### Vision

**The Management Layer transforms ShadowMesh from a relay-based mesh into a fully decentralized private network platform where users can:**

1. **Create Private Networks**: Isolated virtual networks with custom addressing
2. **Control Access**: Invite-only networks with user authentication
3. **Self-Host Controllers**: Run local controllers on-premise (no cloud dependency)
4. **Enable P2P**: Automatic direct connections when NAT traversal succeeds
5. **Manage via UI**: Web dashboard for network administration

### Architecture Philosophy

```
┌─────────────────────────────────────────────────────────────────┐
│                      ShadowMesh Ecosystem                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────┐         ┌──────────────────┐             │
│  │  Cloud Relay     │         │  Local Controller│             │
│  │  (Public)        │◄────────┤  (Private)       │             │
│  └──────────────────┘         └──────────────────┘             │
│         ▲                              ▲                         │
│         │                              │                         │
│    Fallback only              Primary management                │
│         │                              │                         │
│  ┌──────┴──────────────────────────────┴──────┐                │
│  │                                              │                │
│  ▼                                              ▼                │
│  Client 1 ◄────── P2P Direct ──────►  Client 2                 │
│  (Auto P2P when possible, relay fallback)                       │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

**Key Principles**:
- **Controller-Managed**: All networks governed by a controller (cloud or local)
- **P2P First**: Direct connections preferred, relay as fallback
- **Zero Trust**: All connections authenticated and encrypted
- **Network Isolation**: Complete traffic segregation between networks
- **User Privacy**: Self-hosted controllers see no user data

---

## Architecture Components

### 1. Controller Service

**Purpose**: Central management plane for private networks

**Responsibilities**:
- User authentication and authorization
- Network configuration management
- Client registration and peer discovery
- P2P capability exchange (STUN results, ports, NAT type)
- Relay fallback coordination
- Access control enforcement
- Metrics aggregation

**Deployment Options**:
- **Cloud Controller** (ShadowMesh-hosted): `https://controller.shadowmesh.io`
- **Local Controller** (Self-hosted): User's infrastructure (Docker/VM/bare metal)

**Technology Stack**:
- Go HTTP server with TLS
- PostgreSQL database
- Redis for session management
- HTTPS REST API + WebSocket for real-time updates

### 2. Web Management UI

**Purpose**: User-facing dashboard for network management

**Features**:
- Network creation/deletion
- User invitation and access control
- Peer status monitoring (online/offline, latency, throughput)
- Connection topology visualization (P2P vs relay)
- Logs and audit trail
- Configuration templates
- Billing (cloud-hosted only)

**Technology Stack**:
- React + TypeScript frontend
- Hosted alongside controller service
- Real-time updates via WebSocket

### 3. Enhanced Client Daemon

**Purpose**: Existing daemon with management layer integration

**New Features**:
- Controller registration and authentication
- P2P capability detection (STUN, TURN)
- Automatic P2P connection attempts
- Relay fallback when P2P fails
- Real-time status reporting to controller
- Multi-network support (join multiple isolated networks)

**Backward Compatibility**: v0.2.0 relay mode still supported

### 4. Public Relay Infrastructure

**Purpose**: Fallback routing when P2P fails

**New Behavior**:
- Relays report to controller for load balancing
- Per-network isolation (relays can't see which network client belongs to)
- Usage metering for billing

**Unchanged**: Zero-knowledge routing, ChaCha20-Poly1305 encryption

---

## Network Topology Modes

### Mode 1: Controller + P2P Direct (Preferred)

```
┌───────────────────────────────────────────────────────────────┐
│                    Network: "acme-prod"                        │
├───────────────────────────────────────────────────────────────┤
│                                                                 │
│                  ┌──────────────────┐                          │
│                  │   Controller     │                          │
│                  │  (Management)    │                          │
│                  └──────────────────┘                          │
│                       │        │                                │
│              Register │        │ Register                      │
│                       ▼        ▼                                │
│                  Client A    Client B                          │
│                  10.100.1.1  10.100.1.2                        │
│                       │        │                                │
│                       └────────┘                                │
│                      P2P Direct                                 │
│                  (UDP hole-punched)                             │
│                                                                 │
└───────────────────────────────────────────────────────────────┘
```

**When Used**: Both clients have public IPs or successful NAT traversal

**Benefits**:
- Lowest latency (no relay hop)
- Highest throughput (direct path)
- No relay infrastructure cost
- No bandwidth limits

**Traffic Flow**:
1. Clients register with controller
2. Controller exchanges P2P capability info (public IP, STUN results, ports)
3. Clients attempt UDP hole-punching
4. If successful, establish direct WireGuard-style tunnel
5. Controller only used for management (no data plane)

### Mode 2: Controller + Relay Fallback

```
┌───────────────────────────────────────────────────────────────┐
│                    Network: "acme-prod"                        │
├───────────────────────────────────────────────────────────────┤
│                                                                 │
│                  ┌──────────────────┐                          │
│                  │   Controller     │                          │
│                  │  (Management)    │                          │
│                  └──────────────────┘                          │
│                       │        │                                │
│              Register │        │ Register                      │
│                       ▼        ▼                                │
│                  Client A    Client B                          │
│                  10.100.1.1  10.100.1.2                        │
│                       │        │                                │
│                       │        │                                │
│                       └───┬────┘                                │
│                           │                                     │
│                      ┌────▼────┐                                │
│                      │  Relay  │                                │
│                      │ Server  │                                │
│                      └─────────┘                                │
│                                                                 │
└───────────────────────────────────────────────────────────────┘
```

**When Used**: P2P fails (symmetric NAT, firewall, etc.)

**Benefits**:
- Works in restrictive networks
- Reliable connectivity
- Same security (end-to-end encryption)

**Traffic Flow**:
1. Clients register with controller
2. P2P attempt fails
3. Controller assigns relay server
4. All traffic routes through relay (zero-knowledge)

### Mode 3: Hybrid (P2P + Relay)

```
┌───────────────────────────────────────────────────────────────┐
│                    Network: "acme-prod"                        │
├───────────────────────────────────────────────────────────────┤
│                                                                 │
│  Client A ◄──P2P──► Client B ◄──Relay──► Client C            │
│  10.100.1.1         10.100.1.2           10.100.1.3            │
│                                                                 │
│  (A and B use P2P because both have public IPs)               │
│  (C is behind restrictive NAT, uses relay)                     │
│                                                                 │
└───────────────────────────────────────────────────────────────┘
```

**When Used**: Mixed network conditions (some clients can P2P, others can't)

**Benefits**:
- Best of both worlds
- Optimizes each connection independently
- Graceful degradation

---

## Local Controller Design

### Architecture

**Local Controller = Self-Hosted Management Plane**

```
┌─────────────────────────────────────────────────────────────┐
│                  Enterprise Datacenter                       │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────────────────────────────┐               │
│  │       Local Controller (On-Premise)      │               │
│  │  - PostgreSQL database                   │               │
│  │  - Web UI (self-hosted)                  │               │
│  │  - API server                             │               │
│  │  - No internet dependency                 │               │
│  └──────────────────────────────────────────┘               │
│         ▲                                                     │
│         │                                                     │
│   ┌─────┴─────┬──────────┬──────────┐                       │
│   ▼           ▼          ▼          ▼                       │
│  Client 1  Client 2  Client 3  Client 4                     │
│  (All within private network, no cloud relay needed)        │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### Installation

**Docker Compose (Recommended)**:
```bash
# Download compose file
wget https://github.com/shadowmesh/shadowmesh/releases/download/v0.3.0/shadowmesh-controller-compose.yml

# Configure
nano .env
# CONTROLLER_DOMAIN=controller.company.local
# ADMIN_EMAIL=admin@company.com
# ADMIN_PASSWORD=changeme
# DATABASE_PASSWORD=random_secret

# Deploy
docker-compose -f shadowmesh-controller-compose.yml up -d

# Access UI
https://controller.company.local
```

**Kubernetes (Enterprise)**:
```bash
# Apply Helm chart
helm repo add shadowmesh https://charts.shadowmesh.io
helm install shadowmesh-controller shadowmesh/controller \
  --set domain=controller.company.local \
  --set database.password=random_secret
```

**Bare Metal (Advanced)**:
```bash
# Install binaries
sudo dpkg -i shadowmesh-controller_0.3.0_amd64.deb

# Configure
sudo nano /etc/shadowmesh/controller-config.yaml

# Start
sudo systemctl enable shadowmesh-controller
sudo systemctl start shadowmesh-controller
```

### Configuration

`controller-config.yaml`:
```yaml
controller:
  mode: "local"  # or "cloud"
  listen_address: "0.0.0.0:8443"
  tls:
    cert_file: "/etc/shadowmesh/certs/tls.crt"
    key_file: "/etc/shadowmesh/certs/tls.key"

database:
  host: "localhost"
  port: 5432
  name: "shadowmesh_controller"
  user: "shadowmesh"
  password: "FROM_ENV_OR_SECRET"

authentication:
  method: "local"  # or "oauth2", "ldap", "saml"
  session_timeout: "24h"
  jwt_secret: "FROM_ENV_OR_SECRET"

networks:
  max_per_user: 10
  default_subnet: "10.100.0.0/16"

relay:
  enabled: false  # Local controller doesn't need cloud relay
  fallback_relay: "ws://relay.shadowmesh.io:9545"  # Optional public fallback

p2p:
  enabled: true
  stun_servers:
    - "stun:stun.l.google.com:19302"
    - "stun:stun1.l.google.com:19302"

logging:
  level: "info"
  retention_days: 90
```

### Features

**Air-Gapped Support**:
- No internet dependency (all traffic stays on-premise)
- Self-contained database
- Optional relay fallback for remote workers

**LDAP/Active Directory Integration**:
- Use existing corporate credentials
- Group-based access control
- SSO support (SAML, OAuth2)

**Compliance**:
- All data stays on-premise
- Audit logs for compliance
- No ShadowMesh cloud dependency

**High Availability**:
- PostgreSQL replication
- Redis cluster for sessions
- Load-balanced controller instances

---

## P2P Direct Connection Logic

### NAT Traversal Strategy

**Phase 1: Capability Detection**

When client daemon starts:
```go
// Detect NAT type via STUN
stunResult := detectNATType(stunServers)

// Report to controller
controllerClient.ReportCapability(Capability{
    PublicIP:    stunResult.PublicIP,
    PublicPort:  stunResult.PublicPort,
    NATType:     stunResult.NATType,  // Full Cone, Restricted, Port-Restricted, Symmetric
    LocalPort:   config.P2P.ListenerPort,
    Reachable:   stunResult.Success,
})
```

**Phase 2: P2P Attempt**

When two clients need to communicate:
```go
// Controller provides peer info
peerInfo := controller.GetPeerInfo(peerID)

// Attempt UDP hole-punching
if canAttemptP2P(localNATType, peerInfo.NATType) {
    // Send simultaneous UDP packets to punch holes
    conn, err := punchUDPHole(peerInfo.PublicIP, peerInfo.PublicPort)
    if err == nil {
        // Success! Establish encrypted tunnel
        return establishP2PTunnel(conn, peerInfo.PublicKey)
    }
}

// Fallback to relay
return connectViaRelay(relayServer, peerID)
```

**Phase 3: Connection Upgrade**

While connected via relay, continue P2P attempts in background:
```go
// Background P2P prober
go func() {
    for {
        time.Sleep(60 * time.Second)
        if !currentConnection.IsP2P() {
            if conn := attemptP2P(peerInfo); conn != nil {
                // Upgrade to P2P
                currentConnection.UpgradeToP2P(conn)
                log.Printf("Upgraded to P2P connection with %s", peerID)
            }
        }
    }
}()
```

### NAT Traversal Success Matrix

| Client A NAT | Client B NAT | P2P Success | Notes |
|--------------|--------------|-------------|-------|
| Full Cone | Any | ✅ 100% | Easiest case |
| Restricted | Full Cone | ✅ 100% | Symmetric UDP works |
| Port-Restricted | Port-Restricted | ✅ 95% | UDP hole-punching |
| Symmetric | Full Cone | ✅ 80% | One-way punch |
| Symmetric | Symmetric | ❌ 0% | Impossible, use relay |

### TURN Relay Integration (Future)

For Symmetric-Symmetric NAT cases:
```yaml
p2p:
  enabled: true
  stun_servers:
    - "stun:stun.l.google.com:19302"
  turn_servers:
    - "turn:turn.shadowmesh.io:3478"
    - username: "user"
      password: "pass"
```

TURN provides relay as P2P fallback (before falling back to ShadowMesh relay).

---

## Web Management UI

### Dashboard (Home)

```
┌─────────────────────────────────────────────────────────────┐
│  ShadowMesh Controller                          [admin▼]     │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  My Networks                                     [+ Create]   │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ acme-prod                              [Settings]    │    │
│  │ 4 clients online    10.100.1.0/24                    │    │
│  │ Status: ✅ Healthy                                    │    │
│  │ ────────────────────────────────────────────────     │    │
│  │ • server-1 (10.100.1.1)    P2P     2ms    ↑1.2Gbps  │    │
│  │ • server-2 (10.100.1.2)    P2P     3ms    ↑800Mbps  │    │
│  │ • laptop-3 (10.100.1.3)    Relay   12ms   ↑50Mbps   │    │
│  │ • vpn-gw   (10.100.1.4)    P2P     1ms    ↑2.5Gbps  │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ acme-dev                               [Settings]    │    │
│  │ 2 clients online    10.100.2.0/24                    │    │
│  │ Status: ✅ Healthy                                    │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### Network Settings

```
┌─────────────────────────────────────────────────────────────┐
│  Network: acme-prod                              [Delete]    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  General                                                      │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Name:        [acme-prod___________________________]  │    │
│  │ Subnet:      [10.100.1.0/24_____________________]    │    │
│  │ Description: [Production environment____________]    │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                               │
│  Access Control                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Invitation Code:  [regen45-prod-k3fj92]  [Regenerate]│    │
│  │ Auto-Approve:     [✓] Require admin approval         │    │
│  │ Max Clients:      [50_____]                           │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                               │
│  P2P Settings                                                 │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Enable P2P:       [✓] Enabled                        │    │
│  │ Relay Fallback:   [✓] Enabled                        │    │
│  │ Relay Server:     [us-east-relay.shadowmesh.io___]   │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                               │
│  [Save Changes]                                               │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### Client Registration

```
┌─────────────────────────────────────────────────────────────┐
│  Join Network: acme-prod                                     │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  Step 1: Install ShadowMesh Client                           │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ $ curl -fsSL https://get.shadowmesh.io | sh          │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                               │
│  Step 2: Connect to Controller                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ $ shadowmesh-daemon register \                       │    │
│  │     --controller https://controller.shadowmesh.io \  │    │
│  │     --invite regen45-prod-k3fj92                     │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                               │
│  Step 3: Verify Connection                                   │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ $ shadowmesh-daemon status                           │    │
│  │ Network: acme-prod                                   │    │
│  │ IP: 10.100.1.5                                       │    │
│  │ Status: Connected (P2P)                              │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                               │
│  [Download Config] [View Docs]                               │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### Topology Visualization

```
┌─────────────────────────────────────────────────────────────┐
│  Network Topology: acme-prod                                 │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│                    [Controller]                               │
│                         │                                     │
│         ┌───────────────┼───────────────┐                    │
│         │               │               │                     │
│    [server-1]      [server-2]      [laptop-3]                │
│      P2P ◄──────────► P2P            │                       │
│                                    [Relay]                    │
│                                       │                       │
│                                   [vpn-gw]                    │
│                                                               │
│  Legend:                                                      │
│  ─── P2P Direct Connection                                   │
│  ╌╌╌ Relay Connection                                        │
│  [□] Client Node                                             │
│                                                               │
│  [Refresh] [Export PNG]                                      │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

---

## Authentication & Authorization

### User Authentication

**Cloud Controller**:
- Email/password (bcrypt)
- OAuth 2.0 (Google, GitHub, Microsoft)
- Magic links (passwordless)
- MFA (TOTP, WebAuthn)

**Local Controller**:
- LDAP/Active Directory integration
- SAML SSO
- OAuth 2.0 (Okta, Auth0, etc.)
- Local user database (fallback)

### API Authentication

**Client Daemon → Controller**:
```yaml
authentication:
  method: "api_key"  # or "jwt", "mtls"
  api_key: "sk_prod_abc123..."

# Or JWT
authentication:
  method: "jwt"
  jwt_token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  refresh_url: "https://controller.shadowmesh.io/api/v1/auth/refresh"
```

**Controller → Relay**:
- mTLS with client certificates
- Relay API keys
- Network-specific tokens

### Authorization Model

**Role-Based Access Control (RBAC)**:

```
User Roles:
- Admin:    Create/delete networks, manage users, view all logs
- Owner:    Manage specific network, invite users, configure settings
- Member:   Join network, connect clients, view own logs
- Guest:    Read-only access (for monitoring dashboards)

Network Permissions:
- network.create
- network.delete
- network.configure
- network.invite
- network.view
- client.register
- client.remove
- logs.view
```

**Example Policy**:
```json
{
  "user_id": "user123",
  "networks": {
    "acme-prod": {
      "role": "owner",
      "permissions": [
        "network.configure",
        "network.invite",
        "network.view",
        "client.register",
        "logs.view"
      ]
    },
    "acme-dev": {
      "role": "member",
      "permissions": [
        "network.view",
        "client.register"
      ]
    }
  }
}
```

---

## Network Isolation Model

### Complete Traffic Segregation

**Problem**: How to prevent clients in `network-A` from accessing clients in `network-B`?

**Solution**: Network ID embedded in encrypted payload

```go
// Frame format with network ID
type Frame struct {
    NetworkID   [32]byte   // SHA256 hash of network secret
    SourceIP    [4]byte    // Client IP within network
    DestIP      [4]byte    // Target IP within network
    Payload     []byte     // Encrypted application data
    MAC         [16]byte   // ChaCha20-Poly1305 MAC
}
```

**Isolation Enforcement**:

1. **Controller Level**:
   - Each network has unique encryption key
   - Clients only receive keys for authorized networks
   - Network ID derived from key: `NetworkID = SHA256(network_key)`

2. **Relay Level**:
   - Relay routes by NetworkID (can't decrypt payload)
   - Frames with mismatched NetworkID dropped
   - No cross-network routing possible

3. **Client Level**:
   - Daemon enforces NetworkID in every frame
   - Invalid NetworkID frames rejected
   - Multi-network clients maintain separate TUN devices

**Multi-Network Client**:
```
┌─────────────────────────────────────────────────────────┐
│  Client Daemon                                           │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  shadowmesh0 (10.100.1.5)  →  network-A (prod)          │
│  shadowmesh1 (10.200.1.3)  →  network-B (staging)       │
│  shadowmesh2 (10.50.0.10)  →  network-C (personal)      │
│                                                           │
│  Each TUN device = separate routing table                │
│  Each network = separate encryption key                  │
│                                                           │
└─────────────────────────────────────────────────────────┘
```

---

## API Specifications

### Controller REST API

**Authentication**: Bearer token in `Authorization` header

#### Network Management

```http
# Create network
POST /api/v1/networks
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
  "name": "acme-prod",
  "subnet": "10.100.1.0/24",
  "description": "Production environment",
  "p2p_enabled": true,
  "relay_fallback": true
}

Response 201:
{
  "network_id": "net_abc123",
  "invitation_code": "regen45-prod-k3fj92",
  "encryption_key": "0123456789abcdef..."
}
```

```http
# List networks
GET /api/v1/networks
Authorization: Bearer <jwt_token>

Response 200:
{
  "networks": [
    {
      "network_id": "net_abc123",
      "name": "acme-prod",
      "subnet": "10.100.1.0/24",
      "clients_online": 4,
      "created_at": "2025-11-01T12:00:00Z"
    }
  ]
}
```

```http
# Delete network
DELETE /api/v1/networks/:network_id
Authorization: Bearer <jwt_token>

Response 204: No Content
```

#### Client Registration

```http
# Register client
POST /api/v1/networks/:network_id/clients
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
  "client_name": "server-1",
  "invitation_code": "regen45-prod-k3fj92",
  "public_key": "base64_encoded_key",
  "nat_capability": {
    "public_ip": "203.0.113.45",
    "public_port": 51820,
    "nat_type": "port_restricted"
  }
}

Response 201:
{
  "client_id": "cli_xyz789",
  "assigned_ip": "10.100.1.5",
  "network_key": "0123456789abcdef...",
  "relay_server": "ws://relay-us-east.shadowmesh.io:9545"
}
```

```http
# List clients in network
GET /api/v1/networks/:network_id/clients
Authorization: Bearer <jwt_token>

Response 200:
{
  "clients": [
    {
      "client_id": "cli_xyz789",
      "name": "server-1",
      "ip": "10.100.1.5",
      "status": "online",
      "connection_type": "p2p",
      "latency_ms": 2,
      "last_seen": "2025-11-27T10:30:00Z"
    }
  ]
}
```

#### Peer Discovery

```http
# Get peer info for P2P
GET /api/v1/networks/:network_id/peers/:peer_id
Authorization: Bearer <jwt_token>

Response 200:
{
  "peer_id": "cli_abc456",
  "public_key": "base64_encoded_key",
  "endpoints": [
    {
      "type": "direct",
      "address": "203.0.113.45:51820",
      "nat_type": "port_restricted"
    },
    {
      "type": "relay",
      "address": "ws://relay-us-east.shadowmesh.io:9545"
    }
  ]
}
```

### WebSocket API (Real-Time Updates)

```javascript
// Connect to controller WebSocket
const ws = new WebSocket('wss://controller.shadowmesh.io/api/v1/ws');
ws.headers = { 'Authorization': 'Bearer ' + jwt_token };

// Subscribe to network events
ws.send(JSON.stringify({
  type: 'subscribe',
  network_id: 'net_abc123'
}));

// Receive events
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);

  switch (data.type) {
    case 'client_online':
      console.log(`Client ${data.client_id} came online`);
      break;
    case 'client_offline':
      console.log(`Client ${data.client_id} went offline`);
      break;
    case 'connection_upgrade':
      console.log(`Connection upgraded to P2P: ${data.client_id}`);
      break;
  }
};
```

---

## Database Schema

### PostgreSQL Tables

```sql
-- Users
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),  -- NULL for OAuth users
    name VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    last_login TIMESTAMP,
    mfa_enabled BOOLEAN DEFAULT FALSE,
    mfa_secret VARCHAR(32)
);

-- Networks
CREATE TABLE networks (
    network_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    subnet CIDR NOT NULL,
    description TEXT,
    encryption_key BYTEA NOT NULL,  -- 32 bytes ChaCha20-Poly1305 key
    invitation_code VARCHAR(50) UNIQUE,
    p2p_enabled BOOLEAN DEFAULT TRUE,
    relay_fallback BOOLEAN DEFAULT TRUE,
    relay_server VARCHAR(255),
    max_clients INT DEFAULT 100,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Network Membership
CREATE TABLE network_members (
    membership_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    network_id UUID REFERENCES networks(network_id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,  -- 'owner', 'admin', 'member', 'guest'
    joined_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(network_id, user_id)
);

-- Clients
CREATE TABLE clients (
    client_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    network_id UUID REFERENCES networks(network_id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    client_name VARCHAR(100) NOT NULL,
    assigned_ip INET NOT NULL,
    public_key TEXT NOT NULL,  -- Base64 encoded
    nat_type VARCHAR(20),  -- 'full_cone', 'restricted', 'port_restricted', 'symmetric'
    public_ip INET,
    public_port INT,
    status VARCHAR(20) DEFAULT 'offline',  -- 'online', 'offline', 'connecting'
    connection_type VARCHAR(20),  -- 'p2p', 'relay', 'hybrid'
    last_seen TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(network_id, assigned_ip)
);

-- Connection Logs
CREATE TABLE connection_logs (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID REFERENCES clients(client_id) ON DELETE CASCADE,
    event_type VARCHAR(20) NOT NULL,  -- 'connect', 'disconnect', 'upgrade_p2p'
    connection_type VARCHAR(20),
    latency_ms INT,
    remote_peer_id UUID,
    timestamp TIMESTAMP DEFAULT NOW()
);

-- Relay Servers
CREATE TABLE relay_servers (
    relay_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region VARCHAR(50) NOT NULL,
    server_address VARCHAR(255) NOT NULL,
    capacity INT NOT NULL,
    current_load INT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active',  -- 'active', 'maintenance', 'offline'
    last_health_check TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_clients_network ON clients(network_id);
CREATE INDEX idx_clients_status ON clients(status);
CREATE INDEX idx_connection_logs_client ON connection_logs(client_id);
CREATE INDEX idx_connection_logs_timestamp ON connection_logs(timestamp);
```

---

## Deployment Models

### Model 1: Cloud Controller + Public Relays

**Target Audience**: Small teams, individuals, startups

**Infrastructure**:
- Controller: ShadowMesh-hosted at `controller.shadowmesh.io`
- Relays: Multi-region public relays
- Clients: User machines (laptops, servers, phones)

**Pros**:
- Zero ops overhead
- Automatic updates
- Global relay coverage

**Cons**:
- Dependency on ShadowMesh cloud
- Data residency concerns (metadata only)

**Pricing**:
- Free tier: 3 clients, 10GB/month
- Pro: $10/month, 10 clients, 100GB/month
- Team: $50/month, 50 clients, unlimited bandwidth

### Model 2: Local Controller + Public Relays

**Target Audience**: Enterprises with on-premise requirements

**Infrastructure**:
- Controller: Self-hosted (Docker/K8s)
- Relays: Public relays (optional fallback)
- Clients: On-premise + remote workers

**Pros**:
- Control over user data
- LDAP/AD integration
- Custom compliance policies

**Cons**:
- Ops overhead (controller maintenance)
- Public relay dependency for remote workers

**Pricing**:
- Controller license: $500/year per controller
- Public relay fallback: $5/client/month

### Model 3: Local Controller + Private Relays

**Target Audience**: Defense, finance, healthcare (strict compliance)

**Infrastructure**:
- Controller: Self-hosted
- Relays: Self-hosted in multiple regions
- Clients: All on-premise or private cloud

**Pros**:
- Complete air-gapped deployment
- Zero ShadowMesh dependency
- Full compliance control

**Cons**:
- Highest ops overhead
- Relay infrastructure cost

**Pricing**:
- Enterprise license: Custom (includes support)

---

## Migration from v0.2.0

### Step-by-Step Migration

**Phase 1: Deploy Controller** (Week 1)

```bash
# Option A: Use ShadowMesh cloud (no setup)
# Register at https://controller.shadowmesh.io

# Option B: Self-host controller
docker-compose -f shadowmesh-controller-compose.yml up -d
```

**Phase 2: Create Networks** (Week 1)

```bash
# Via UI: controller.shadowmesh.io → Create Network
# Or API:
curl -X POST https://controller.shadowmesh.io/api/v1/networks \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "name": "my-network",
    "subnet": "10.100.1.0/24",
    "p2p_enabled": true
  }'
```

**Phase 3: Upgrade Clients** (Week 2-3)

```bash
# Download v0.3.0 daemon
wget https://github.com/shadowmesh/shadowmesh/releases/download/v0.3.0/shadowmesh-daemon-linux-amd64

# Update config
nano /etc/shadowmesh/client-config.yaml

# Add controller section:
controller:
  enabled: true
  url: "https://controller.shadowmesh.io"
  api_key: "sk_prod_abc123..."
  network_id: "net_xyz789"

# Restart daemon
sudo systemctl restart shadowmesh
```

**Phase 4: Verify P2P** (Week 3)

```bash
# Check connection type
shadowmesh-daemon status

# Should show:
# Connection: P2P (direct)
# or
# Connection: Relay (fallback)
```

**Phase 5: Decommission Old Relays** (Week 4)

```bash
# Once all clients on v0.3.0 and using controller
# Shut down v0.2.0 standalone relays
sudo systemctl stop shadowmesh-relay
```

### Backward Compatibility

v0.3.0 clients support v0.2.0 relay mode:
```yaml
# Legacy relay mode (no controller)
relay:
  enabled: true
  server: "ws://94.237.121.21:9545"

controller:
  enabled: false
```

But v0.2.0 clients cannot connect to v0.3.0 managed networks.

---

## Next Steps

### Immediate (v0.3.0 Development)

1. **Controller Service**:
   - Implement REST API (Go HTTP server)
   - PostgreSQL schema and migrations
   - Authentication (JWT, OAuth2)
   - WebSocket for real-time updates

2. **Web UI**:
   - React dashboard
   - Network creation/management
   - Client registration flow
   - Topology visualization

3. **Enhanced Client**:
   - Controller registration logic
   - P2P capability detection (STUN)
   - Automatic P2P attempts
   - Multi-network support

4. **Testing**:
   - End-to-end P2P scenarios
   - Relay fallback testing
   - Network isolation verification

### Future (v0.4.0+)

- Post-quantum cryptography (ML-KEM, ML-DSA)
- Mobile apps (iOS, Android)
- Exit node support
- Multi-hop routing
- Advanced obfuscation

---

## Conclusion

The Management Layer transforms ShadowMesh from a basic relay network into a **full-featured decentralized private network platform** with:

- ✅ **User Control**: Create and manage private networks
- ✅ **Self-Hosting**: Local controllers, no cloud dependency
- ✅ **P2P First**: Direct connections, relay as fallback
- ✅ **Enterprise Ready**: LDAP, SSO, compliance
- ✅ **Scalable**: Supports 1000s of clients per network

**Target Release**: v0.3.0 (Q1 2026)

---

## References

- **Tailscale Architecture**: https://tailscale.com/blog/how-tailscale-works/
- **WireGuard Protocol**: https://www.wireguard.com/protocol/
- **STUN/TURN NAT Traversal**: https://datatracker.ietf.org/doc/html/rfc5389
- **ZeroTier Architecture**: https://docs.zerotier.com/zerotier/manual/
