# 🚀 Run Direct P2P Testing

## Simple: Just Pull & Run Locally

### On UK VPS (Listener)

```bash
# Clone if first time
git clone https://github.com/CG-8663/shadowmesh.git
cd shadowmesh

# Or update if already cloned
cd shadowmesh
git pull origin main

# Run as listener
./scripts/update-and-run.sh listener
```

### On Belgium RPi (Connector)

```bash
# Clone if first time
git clone https://github.com/CG-8663/shadowmesh.git
cd shadowmesh

# Or update if already cloned
cd shadowmesh
git pull origin main

# Run as connector (replace with your VPS IP)
./scripts/update-and-run.sh connector YOUR_VPS_IP
```

**Example**:
```bash
./scripts/update-and-run.sh connector 123.45.67.89
```

---

## What This Does

The script automatically:
1. ✅ Pulls latest from GitHub
2. ✅ Builds shadowmesh-daemon
3. ✅ Configures with correct IPs
4. ✅ Starts the daemon

---

## Test the Tunnel

Once both are running:

**From UK VPS**:
```bash
ping 10.10.10.4
```

**From Belgium RPi**:
```bash
ping 10.10.10.3
```

**Expected**: 0% packet loss ✅

---

## Expected Output

### UK VPS (Listener):
```
╔════════════════════════════════════════════════╗
║  ShadowMesh - Update & Run                     ║
╚════════════════════════════════════════════════╝

Configuration:
  Mode:      listener
  TAP IP:    10.10.10.3

Step 1: Updating from GitHub...
✓ Repository updated

Step 2: Building...
✓ Build successful

Step 3: Preparing configuration...
✓ Configuration ready

Step 4: Ready to start daemon

About to run:
  sudo ./build/shadowmesh-daemon --config configs/vps-uk-listener.yaml

This machine will:
  - Listen on port 8443
  - Accept incoming P2P connections
  - Create TAP device chr-001 with IP 10.10.10.3

Step 5: Starting daemon...
Press Ctrl+C to stop

ShadowMesh Client Daemon v0.1.0-alpha
Post-Quantum Decentralized Private Network (DPN)
=================================================

Client ID: <id>
Creating TAP device: chr-001
TAP device created: chr-001 (MTU: 1500)
Mode: Listener - Waiting for connections on 0.0.0.0:8443
Listening for P2P connections on 0.0.0.0:8443 (TLS: true)
Daemon running. Press Ctrl+C to stop.

[Waiting for peer...]

Peer connected from <rpi-ip>
Performing post-quantum handshake...
Handshake complete. Session ID: <id>
Tunnel established. Network traffic is now encrypted.
```

### Belgium RPi (Connector):
```
╔════════════════════════════════════════════════╗
║  ShadowMesh - Update & Run                     ║
╚════════════════════════════════════════════════╝

Configuration:
  Mode:      connector
  TAP IP:    10.10.10.4
  Peer IP:   123.45.67.89

Step 1: Updating from GitHub...
✓ Repository updated

Step 2: Building...
✓ Build successful

Step 3: Preparing configuration...
✓ Configuration ready

Step 4: Ready to start daemon

About to run:
  sudo ./build/shadowmesh-daemon --config /tmp/shadowmesh-config.yaml

This machine will:
  - Connect to 123.45.67.89:8443
  - Establish P2P encrypted tunnel
  - Create TAP device chr-001 with IP 10.10.10.4

Step 5: Starting daemon...
Press Ctrl+C to stop

ShadowMesh Client Daemon v0.1.0-alpha
Post-Quantum Decentralized Private Network (DPN)
=================================================

Client ID: <id>
Creating TAP device: chr-001
TAP device created: chr-001 (MTU: 1500)
Mode: Connector - Connecting to peer: 123.45.67.89:8443
Connecting to peer at 123.45.67.89:8443 (attempt 1)...
Connected to peer
Performing post-quantum handshake...
Generating post-quantum keys (ML-KEM-1024, ML-DSA-87)... this may take 10-30 seconds
Handshake complete. Session ID: <id>
Tunnel established. Network traffic is now encrypted.
Daemon running. Press Ctrl+C to stop.
```

---

## Troubleshooting

### "Must run from shadowmesh directory"

```bash
cd ~/shadowmesh
./scripts/update-and-run.sh listener
```

### "Failed to create TAP device"

Already using sudo - check if TUN/TAP module loaded:
```bash
lsmod | grep tun
sudo modprobe tun
```

### Connection refused (from connector)

Check firewall on UK VPS:
```bash
sudo ufw allow 8443/tcp
```

### Ping doesn't work

Check TAP devices:
```bash
ip addr show chr-001
```

Manual setup if needed:
```bash
# UK VPS
sudo ip addr add 10.10.10.3/24 dev chr-001
sudo ip link set chr-001 up

# Belgium RPi
sudo ip addr add 10.10.10.4/24 dev chr-001
sudo ip link set chr-001 up
```

---

## Prerequisites

### First Time Setup

**Install Go** (if not already installed):
```bash
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
```

**Install Git** (if not already installed):
```bash
sudo apt-get update
sudo apt-get install -y git
```

---

## Architecture

This creates a **direct peer-to-peer encrypted tunnel**:

```
UK VPS                     Belgium RPi
10.10.10.3                 10.10.10.4
chr-001                    chr-001
    │                          │
    └──── Direct P2P WSS ──────┘
         Port 8443
    ML-KEM-1024 + ChaCha20-Poly1305
```

**No relay server** - pure P2P encrypted connection!

The relay server you tested before becomes a **backup** for cases where direct P2P can't work.

---

## Full Workflow

```bash
# UK VPS - Terminal 1
ssh user@vps-ip
cd shadowmesh
git pull origin main
./scripts/update-and-run.sh listener

# Belgium RPi - Terminal 2
ssh pi@rpi-ip
cd shadowmesh
git pull origin main
./scripts/update-and-run.sh connector VPS_IP

# Test - Terminal 3
ssh user@vps-ip
ping 10.10.10.4
# Should see: 0% packet loss

ssh pi@rpi-ip
ping 10.10.10.3
# Should see: 0% packet loss
```

---

## That's It!

Two commands, direct P2P encrypted tunnel with post-quantum cryptography! 🚀
