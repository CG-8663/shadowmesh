# ShadowMesh Pre-Release Deployment Guide

## Overview

This guide walks through deploying a stable 5-node ShadowMesh mesh network for pre-release testing.

## Network Topology

```
┌────────────────────────────────────────────────────────────┐
│ ShadowMesh Mesh Network v0.1.0-epic2                      │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  Node                Location        Mesh IP      Mgmt IP  │
│  ────────────────────────────────────────────────────────  │
│  shadowmesh-001     UK (1Gbps)      10.10.10.3   100.115  │
│  shadowmesh-002     Belgium (Pi)    10.10.10.4   100.90   │
│  shadowmesh-003     Starlink        10.10.10.7   100.126  │
│  shadowmesh-004     Philippines     10.10.10.6   100.87   │
│  VM 111             Chronara API    10.10.10.5   Local    │
│                                                            │
│  Relay Server       UpCloud         94.237.121.21:9545    │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

## Installation Command (All Nodes)

```bash
SHADOWMESH_AUTO_INSTALL=1 curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

## Step-by-Step Deployment

### 1. Deploy Relay Server (Already Done)
✅ Relay running at `94.237.121.21:9545`

### 2. Deploy Client Nodes

#### UK (shadowmesh-001)
```bash
ssh pxcghost@100.115.193.115
SHADOWMESH_AUTO_INSTALL=1 curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

#### Belgium (shadowmesh-002)
```bash
ssh pxcghost@100.90.48.10
SHADOWMESH_AUTO_INSTALL=1 curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

#### Starlink (shadowmesh-003)
```bash
ssh pxcghost@100.126.75.74
SHADOWMESH_AUTO_INSTALL=1 curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

#### Philippines (shadowmesh-004)
```bash
ssh pxcghost@100.87.142.44
SHADOWMESH_AUTO_INSTALL=1 curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

#### VM 111 (Chronara API) - **NEW**
```bash
# Run directly on VM 111 (Proxmox)
SHADOWMESH_AUTO_INSTALL=1 curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

### 3. Verification

#### Check Relay Status
```bash
curl http://94.237.121.21:9545/status | jq .
```

Expected output (5 peers):
```json
{
  "status": "ok",
  "connected_peers": 5,
  "peers": [
    {"id": "peer-10.10.10.3"},  // UK
    {"id": "peer-10.10.10.4"},  // Belgium
    {"id": "peer-10.10.10.5"},  // Chronara API
    {"id": "peer-10.10.10.6"},  // Philippines
    {"id": "peer-10.10.10.7"}   // Starlink
  ]
}
```

#### Check Individual Node Status

```bash
# On any node
sudo systemctl status shadowmesh-daemon
ip addr show chr001
sudo journalctl -u shadowmesh-daemon -n 20
```

#### Test Connectivity (From UK Node)

```bash
ssh pxcghost@100.115.193.115

# Test all nodes
ping -c 2 10.10.10.4  # Belgium
ping -c 2 10.10.10.5  # Chronara API
ping -c 2 10.10.10.6  # Philippines
ping -c 2 10.10.10.7  # Starlink
```

## Pre-Release Stability Checklist

### Infrastructure
- [ ] All 5 nodes deployed successfully
- [ ] All TAP interfaces (chr001) created
- [ ] All nodes connected to relay server
- [ ] Full mesh connectivity (all nodes can ping each other)

### Services
- [ ] shadowmesh-daemon running on all nodes
- [ ] shadowmesh-daemon enabled (auto-start on boot)
- [ ] No errors in logs (`journalctl -u shadowmesh-daemon`)
- [ ] Relay server running and stable

### Performance
- [ ] Latency measurements documented
- [ ] No packet loss under normal conditions
- [ ] Automatic reconnection after network disruption tested

### Security
- [ ] ChaCha20-Poly1305 encryption active
- [ ] Shared encryption key deployed (temporary - OK for pre-release)
- [ ] Relay server cannot decrypt traffic (verified)

### Chronara API Integration
- [ ] Chronara API accessible via mesh (http://10.10.10.5:8080)
- [ ] Cloudflare tunnel still operational (external access)
- [ ] API responds to mesh requests
- [ ] No public exposure of mesh network

### Documentation
- [ ] Network topology documented
- [ ] Installation commands documented
- [ ] Troubleshooting guide created
- [ ] Management commands documented

## Troubleshooting

### Node Not Connecting to Relay

```bash
# Check daemon logs
sudo journalctl -u shadowmesh-daemon -f

# Look for:
# "Auto-connecting to configured peer/relay"
# "Connected to relay server successfully"

# Restart if needed
sudo systemctl restart shadowmesh-daemon
```

### TAP Interface Not Created

```bash
# Check if daemon has CAP_NET_ADMIN
sudo systemctl status shadowmesh-daemon

# Verify service runs as root
grep User /etc/systemd/system/shadowmesh-daemon.service
# Should show: User=root

# Check logs for TAP creation errors
sudo journalctl -u shadowmesh-daemon | grep TAP
```

### Cannot Ping Other Nodes

```bash
# Verify TAP interface has correct IP
ip addr show chr001

# Check routing
ip route | grep 10.10.10

# Check if daemon is connected
curl http://localhost:9090/status

# Verify relay has this peer
curl http://94.237.121.21:9545/status | jq '.peers[] | select(.id == "peer-10.10.10.X")'
```

### High Latency or Packet Loss

```bash
# Check for P2P vs Relay routing
sudo journalctl -u shadowmesh-daemon | grep -E "P2P|relay|UDP"

# Expected: All traffic through relay (Symmetric NAT)

# Monitor real-time logs
sudo journalctl -u shadowmesh-daemon -f
```

## Management Commands

### Service Control
```bash
# Start
sudo systemctl start shadowmesh-daemon

# Stop
sudo systemctl stop shadowmesh-daemon

# Restart
sudo systemctl restart shadowmesh-daemon

# Status
sudo systemctl status shadowmesh-daemon

# Enable auto-start
sudo systemctl enable shadowmesh-daemon

# Disable auto-start
sudo systemctl disable shadowmesh-daemon
```

### Logs
```bash
# Last 50 lines
sudo journalctl -u shadowmesh-daemon -n 50

# Follow (real-time)
sudo journalctl -u shadowmesh-daemon -f

# Since timestamp
sudo journalctl -u shadowmesh-daemon --since "5 minutes ago"

# Full log
sudo journalctl -u shadowmesh-daemon --no-pager
```

### Network
```bash
# Show TAP interface
ip addr show chr001

# Show routes
ip route | grep 10.10.10

# Test connectivity
ping 10.10.10.X

# Check listening ports
sudo ss -tlnp | grep shadowmesh
```

### Debugging
```bash
# View configuration
cat /etc/shadowmesh/daemon.yaml

# Test relay connection
curl http://94.237.121.21:9545/health
curl http://94.237.121.21:9545/status

# Check daemon API
curl http://localhost:9090/status
```

## Next Steps After Pre-Release

1. **Performance Testing**
   - iperf3 throughput tests
   - Latency measurements under load
   - Encryption overhead analysis

2. **Security Hardening**
   - Per-peer key exchange (Phase 3)
   - Web3 authentication for Chronara API
   - TPM attestation for exit nodes

3. **Monitoring & Alerting**
   - Prometheus metrics
   - Grafana dashboards
   - Alert on disconnection

4. **Production Deployment**
   - Multi-region relay servers
   - Load balancing
   - High availability
   - SOC 2 compliance

## Support

- **GitHub**: https://github.com/CG-8663/shadowmesh
- **Installation**: See `INSTALL.md`
- **Deployment**: See `DEPLOYMENT_GUIDE.md`
- **Issues**: https://github.com/CG-8663/shadowmesh/issues
