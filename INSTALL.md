# ShadowMesh Client Installation Guide

## Quick Install (Linux/macOS)

```bash
curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/scripts/install-client.sh | sudo bash
```

## Manual Installation

### Prerequisites

- Go 1.21+ installed
- Root/sudo access (required for TAP device)
- Linux, macOS, or Windows WSL2

### Step 1: Clone Repository

```bash
git clone https://github.com/CG-8663/shadowmesh.git
cd shadowmesh
```

### Step 2: Build Client

```bash
make build-client
```

Or manually:

```bash
go build -o bin/shadowmesh-client ./client/daemon
```

### Step 3: Install Binary

```bash
sudo cp bin/shadowmesh-client /usr/local/bin/
sudo chmod +x /usr/local/bin/shadowmesh-client
```

### Step 4: Generate Keys

```bash
shadowmesh-client --gen-keys
```

This creates:
- `~/.shadowmesh/keys/signing_key.json` - Your post-quantum signing key
- `~/.shadowmesh/keys/client_id.txt` - Your client identifier
- `~/.shadowmesh/config.yaml` - Default configuration

### Step 5: Configure

Edit `~/.shadowmesh/config.yaml`:

```yaml
relay:
  url: "wss://relay.shadowmesh.network:443"  # Your relay server URL
  
tap:
  name: "tap0"
  mtu: 1500
  ip_addr: "10.42.0.2"
  netmask: "255.255.255.0"
  
crypto:
  enable_key_rotation: true
  key_rotation_interval: 1h
```

### Step 6: Run Client

```bash
sudo shadowmesh-client
```

**Note:** Requires root for TAP device creation.

## System Service (Linux)

Create `/etc/systemd/system/shadowmesh.service`:

```ini
[Unit]
Description=ShadowMesh Post-Quantum VPN Client
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/shadowmesh-client
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable shadowmesh
sudo systemctl start shadowmesh
```

## Docker Installation

```bash
docker run -d \
  --name shadowmesh-client \
  --cap-add=NET_ADMIN \
  --device=/dev/net/tun \
  -v ~/.shadowmesh:/root/.shadowmesh \
  ghcr.io/cg-8663/shadowmesh-client:latest
```

## Platform-Specific Notes

### macOS

Install TUN/TAP driver:

```bash
brew install tuntaposx
```

### Windows (WSL2)

Run from WSL2 with admin privileges:

```bash
wsl --install
wsl
# Then follow Linux instructions
```

## Verify Installation

```bash
shadowmesh-client --version
shadowmesh-client --show-config
```

## Troubleshooting

### Permission Denied

```bash
sudo shadowmesh-client
```

### TAP Device Creation Failed

Ensure TUN/TAP kernel module is loaded:

```bash
# Linux
sudo modprobe tun

# macOS
# Reinstall tuntaposx
```

### Connection Failed

Check relay URL in config:

```bash
shadowmesh-client --show-config
```

## Security

- **Keys** are stored in `~/.shadowmesh/keys/` with 0600 permissions
- **Traffic** is encrypted with ChaCha20-Poly1305
- **Handshake** uses ML-KEM-1024 (Kyber) + ML-DSA-87 (Dilithium)
- **Key rotation** happens every hour by default

## Support

- GitHub Issues: https://github.com/CG-8663/shadowmesh/issues
- Documentation: https://github.com/CG-8663/shadowmesh/tree/main/docs
