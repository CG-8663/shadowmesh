# ShadowMesh Installation Guide

## Quick Install (One Command)

### Using GitHub (current)
```bash
curl -sSL https://raw.githubusercontent.com/cg-8663/shadowmesh/main/install.sh | sudo bash
```

### Using Custom Domain (coming soon)
```bash
curl -sSL https://get.shadowmesh.io | sudo bash
```

## How It Works

ShadowMesh is a **Decentralized Private Network (DPN)** that creates peer-to-peer encrypted tunnels between nodes:

1. **P2P Direct Connections** (Primary)
   - Clients establish direct encrypted tunnels to each other
   - NAT traversal using STUN for peer discovery
   - Best performance, lowest latency
   - No relay overhead

2. **Relay Fallback** (Secondary)
   - Used when P2P direct connection fails
   - Network topology changes (IP changes, NAT changes)
   - Key renewal and reconnection
   - Temporary network interference
   - Initial peer discovery

3. **Architecture**
   ```
   Client A (10.10.10.3) ←→ Client B (10.10.10.4)  [Direct P2P]
         ↓                           ↓
         └─────→ Relay Server ←──────┘             [Fallback only]
   ```

## Supported Nodes

The installer auto-detects and configures:

- **UK Client** (shadowmesh-001): `10.10.10.3/24`
- **Belgium Client** (shadowmesh-002): `10.10.10.4/24`
- **Chronara API** (VM 111): `10.10.10.5/24`
- **Custom**: Manual IP configuration

## Installation Steps

1. **Run Installer**
   ```bash
   curl -sSL https://raw.githubusercontent.com/cg-8663/shadowmesh/main/install.sh | sudo bash
   ```

2. **Auto-Detection**
   - Installer detects node based on hostname/IP
   - Confirms configuration before proceeding
   - Manual override available

3. **Installation Process**
   - Installs dependencies (Git, curl, Go)
   - Clones ShadowMesh repository
   - Builds daemon binary
   - Creates configuration
   - Sets up systemd service
   - Creates TAP interface (chr001)

4. **Verification**
   ```bash
   sudo systemctl status shadowmesh-daemon
   ip addr show chr001
   ```

## Post-Installation

### Test Connectivity
```bash
# From any node, ping other nodes
ping 10.10.10.3  # UK
ping 10.10.10.4  # Belgium
ping 10.10.10.5  # Chronara API
```

### Management Commands
```bash
# View logs
sudo journalctl -u shadowmesh-daemon -f

# Restart service
sudo systemctl restart shadowmesh-daemon

# Check status
sudo systemctl status shadowmesh-daemon

# View network interface
ip addr show chr001
```

## Network Configuration

### Default Settings
- **Mesh Network**: `10.10.10.0/24`
- **TAP Device**: `chr001`
- **Relay Server**: `94.237.121.21:9545`
- **Encryption**: ChaCha20-Poly1305
- **STUN Server**: `stun.l.google.com:19302`

### Configuration File
Location: `/etc/shadowmesh/daemon.yaml`

```yaml
daemon:
  listen_address: "127.0.0.1:9090"
  log_level: "info"

network:
  tap_device: "chr001"
  local_ip: "10.10.10.X/24"  # Auto-configured

encryption:
  key: "..."  # Shared mesh key

peer:
  address: "94.237.121.21:9545"  # Relay for discovery/fallback

nat:
  enabled: true
  stun_server: "stun.l.google.com:19302"
```

## Uninstall

```bash
curl -sSL https://raw.githubusercontent.com/cg-8663/shadowmesh/main/uninstall.sh | sudo bash
```

Or manually:
```bash
sudo systemctl stop shadowmesh-daemon
sudo systemctl disable shadowmesh-daemon
sudo rm /etc/systemd/system/shadowmesh-daemon.service
sudo rm /usr/local/bin/shadowmesh-daemon
sudo ip link delete chr001
sudo rm -rf /etc/shadowmesh  # Optional: remove config
sudo rm -rf /opt/shadowmesh  # Optional: remove repo
```

## Troubleshooting

### Daemon won't start
```bash
# Check logs
sudo journalctl -u shadowmesh-daemon -n 50

# Common issues:
# - TAP device creation failed (need root/CAP_NET_ADMIN)
# - Port already in use
# - Invalid configuration
```

### No connectivity to other nodes
```bash
# Check TAP interface exists
ip addr show chr001

# Check if daemon is running
sudo systemctl status shadowmesh-daemon

# Check firewall (if enabled)
sudo iptables -L -n | grep 9545

# Test relay connectivity
curl http://94.237.121.21:9545/health
```

### TAP interface not created
```bash
# Daemon needs CAP_NET_ADMIN capability
# Service runs as root by default

# Check if TUN/TAP module is loaded
lsmod | grep tun

# Load if needed
sudo modprobe tun
```

## Advanced Configuration

### Custom Node Setup
```bash
# Download installer
curl -sSL https://raw.githubusercontent.com/cg-8663/shadowmesh/main/install.sh > install.sh
chmod +x install.sh

# Edit variables at top of script
nano install.sh

# Run
sudo ./install.sh
```

### Multiple Mesh Networks
To join multiple meshes, create separate configs:
```bash
sudo cp /etc/shadowmesh/daemon.yaml /etc/shadowmesh/daemon-mesh2.yaml
# Edit mesh2 config with different:
# - local_ip (different subnet)
# - tap_device (chr002, chr003, etc.)
# - encryption key (different mesh key)
# - relay address (different relay)

# Create separate systemd service for mesh2
sudo cp /etc/systemd/system/shadowmesh-daemon.service /etc/systemd/system/shadowmesh-mesh2.service
# Edit to use daemon-mesh2.yaml
```

## Security Notes

1. **Encryption**: All traffic encrypted with ChaCha20-Poly1305
2. **Shared Key**: Current setup uses shared mesh key (OK for trusted network)
3. **Future**: Per-peer keys with automatic rotation
4. **Relay Trust**: Relay server cannot decrypt traffic, only forwards encrypted packets
5. **P2P First**: Direct connections bypass relay when possible

## Next Steps

After installation:
1. Test connectivity between all nodes
2. Monitor logs for P2P connection establishment
3. Verify direct paths are established (not all traffic through relay)
4. Check for errors or warnings in journalctl

## Support

- GitHub: https://github.com/cg-8663/shadowmesh
- Issues: https://github.com/cg-8663/shadowmesh/issues
- Docs: Coming soon
