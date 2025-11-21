# ShadowMesh Client Deployment Guide

## One-Line Installation

### Current (GitHub)
```bash
curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

### Future (Custom Domain)
```bash
curl -sSL https://get.shadowmesh.io | sudo bash
```

## Deployment Instructions

### 1. UK Client (shadowmesh-001)
SSH to UK server and run:
```bash
ssh user@shadowmesh-001
curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

Auto-detects as: UK Client (`10.10.10.3/24`)

### 2. Belgium Client (shadowmesh-002)
SSH to Belgium server and run:
```bash
ssh pxcghost@100.90.48.10
curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

Auto-detects as: Belgium Client (`10.10.10.4/24`)

### 3. Chronara API (VM 111)
SSH to Chronara API server and run:
```bash
ssh user@vm111
curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

Auto-detects as: Chronara API Client (`10.10.10.5/24`)

## Post-Installation Verification

### Check Service Status
```bash
sudo systemctl status shadowmesh-daemon
```

### Check Network Interface
```bash
ip addr show chr001
```

Should show:
```
chr001: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500
    inet 10.10.10.X/24
```

### Test Connectivity
From any node, ping the others:
```bash
ping 10.10.10.3  # UK
ping 10.10.10.4  # Belgium
ping 10.10.10.5  # Chronara API
```

### View Logs
```bash
sudo journalctl -u shadowmesh-daemon -f
```

Look for:
- `P2P connection established` (direct connection working)
- `Connected to relay` (fallback available)
- `TAP interface created` (chr001 up)

## Network Architecture

```
┌─────────────────────────────────────────────────────────┐
│ ShadowMesh P2P Mesh Network (10.10.10.0/24)            │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  UK Client ──────────────► Belgium Client              │
│  10.10.10.3        P2P      10.10.10.4                 │
│       │                          │                      │
│       │                          │                      │
│       └────► Chronara API ◄──────┘                     │
│              10.10.10.5                                 │
│                   ▲                                     │
│                   │                                     │
│              P2P direct paths (primary)                 │
│                                                         │
├─────────────────────────────────────────────────────────┤
│                     │                                   │
│              Relay Fallback                             │
│                     ▼                                   │
│          94.237.121.21:9545                            │
│         (UpCloud Relay Server)                          │
│                                                         │
│  Used for:                                              │
│  • Initial peer discovery                               │
│  • NAT traversal (STUN)                                │
│  • Fallback when P2P fails                             │
│  • Network topology changes                             │
│  • Key rotation                                         │
└─────────────────────────────────────────────────────────┘
```

## Relay Server Configuration

Current relay: `94.237.121.21:9545`

### Check Relay Health
```bash
curl http://94.237.121.21:9545/health
# Should return: OK

curl http://94.237.121.21:9545/status
# Should return: {"status":"ok","connected_peers":N,"peers":[...]}
```

### Deploy New Relay (if needed)
From local machine:
```bash
cd /Volumes/BACKUPDISK/webcode/shadowmesh
./scripts/deploy/upcloud-relay.sh
```

Update `RELAY_IP` in `install.sh` with new relay IP.

## Updating Clients

### Manual Update
```bash
ssh user@node
cd /opt/shadowmesh
sudo git pull origin main
sudo go build -ldflags="-s -w" -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon/
sudo cp bin/shadowmesh-daemon /usr/local/bin/
sudo systemctl restart shadowmesh-daemon
```

### Reinstall (Clean)
```bash
# Uninstall first
curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/uninstall.sh | sudo bash

# Reinstall
curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

## Configuration Files

### Daemon Config
Location: `/etc/shadowmesh/daemon.yaml`

Default settings:
```yaml
daemon:
  listen_address: "127.0.0.1:9090"
  log_level: "info"

network:
  tap_device: "chr001"
  local_ip: "10.10.10.X/24"

peer:
  address: "94.237.121.21:9545"  # Relay server

nat:
  enabled: true
  stun_server: "stun.l.google.com:19302"

max_throughput:
  enabled: true
  # Performance optimizations...
```

### Systemd Service
Location: `/etc/systemd/system/shadowmesh-daemon.service`

```ini
[Unit]
Description=ShadowMesh Daemon
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/shadowmesh-daemon /etc/shadowmesh/daemon.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

## Troubleshooting

### Installation fails
```bash
# Check prerequisites
which git go curl

# Install manually
sudo apt-get update
sudo apt-get install -y git curl wget

# Re-run installer
curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

### Daemon won't start
```bash
# Check logs
sudo journalctl -u shadowmesh-daemon -n 50 --no-pager

# Common issues:
# 1. TAP device creation failed (need root)
# 2. Port already in use
# 3. Binary not found
# 4. Invalid config

# Verify binary exists
ls -lh /usr/local/bin/shadowmesh-daemon

# Verify config exists
cat /etc/shadowmesh/daemon.yaml

# Test manually
sudo /usr/local/bin/shadowmesh-daemon /etc/shadowmesh/daemon.yaml
```

### No P2P connectivity
```bash
# Check TAP interface
ip addr show chr001

# Check routing
ip route | grep 10.10.10

# Check firewall
sudo iptables -L -n

# Ping relay server
ping 94.237.121.21

# Test relay connection
curl http://94.237.121.21:9545/health
```

### High CPU/Memory usage
```bash
# Check resource usage
top -p $(pgrep shadowmesh-daemon)

# Check connection count
sudo journalctl -u shadowmesh-daemon | grep "P2P connection"

# Restart daemon
sudo systemctl restart shadowmesh-daemon
```

## Security Considerations

1. **Encryption**: All traffic encrypted with ChaCha20-Poly1305
2. **Shared Key**: Currently using shared mesh key (suitable for trusted network)
3. **Relay Trust Model**: Relay cannot decrypt traffic, only forwards encrypted packets
4. **Future**: Per-peer key exchange with automatic rotation

## Performance Tuning

### Increase buffer sizes (high throughput)
Edit `/etc/shadowmesh/daemon.yaml`:
```yaml
max_throughput:
  socket:
    send_buffer_kb: 512
    recv_buffer_kb: 1024
  batching:
    max_batch_size: 20
    max_batch_bytes: 16000
```

Restart: `sudo systemctl restart shadowmesh-daemon`

### Decrease buffer sizes (low latency)
```yaml
max_throughput:
  socket:
    send_buffer_kb: 128
    recv_buffer_kb: 256
  batching:
    max_batch_size: 5
    max_batch_bytes: 4000
    coalesce_timeout_ms: 1
```

## Next Steps

1. **Deploy to all nodes** using the one-line installer
2. **Verify connectivity** between all nodes
3. **Monitor logs** for P2P establishment
4. **Test performance** with iperf3 between nodes
5. **Add Web3 authentication** to Chronara API (Phase 3)
6. **Configure custom domain** (get.shadowmesh.io)

## Support

- Repository: https://github.com/CG-8663/shadowmesh
- Issues: https://github.com/CG-8663/shadowmesh/issues
- Installation Guide: INSTALL.md
