# Quick Start: Testing Story 2-8

**Goal:** Validate ShadowMesh P2P tunnel between two Raspberry Pis

**Time Required:** 30-60 minutes

**Test Environment:** Two Raspberry Pis connected via internet/tethered connections

---

## Prerequisites Checklist

- [ ] Two Raspberry Pis (or any Linux machines)
- [ ] Both have sudo/root access
- [ ] Internet connectivity on both
- [ ] Git installed (`sudo apt install git`)
- [ ] Go 1.21+ installed (see Step 0 if not installed)

---

## Step 0: Install Prerequisites (One-time setup on BOTH Raspberry Pis)

### Install Go (if not already installed)

```bash
# Check if Go is installed
go version

# If not installed, install Go 1.21+
wget https://go.dev/dl/go1.21.5.linux-arm64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.5.linux-arm64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

### Install Git (if not already installed)

```bash
sudo apt update
sudo apt install -y git
```

---

## Step 1: Install ShadowMesh (on BOTH Raspberry Pis)

### Option A: Automated Installation (Recommended)

**On BOTH Raspberry Pis**, run the installation script:

```bash
cd ~
git clone https://github.com/yourusername/shadowmesh.git
cd shadowmesh
sudo ./scripts/install-raspberry-pi.sh
```

**Note:** Replace `yourusername` with the actual GitHub username/organization.

The script will:
- ✅ Install Go if needed
- ✅ Install Git if needed
- ✅ Clone/update repository
- ✅ Build daemon binary
- ✅ Install to /usr/local/bin
- ✅ Generate encryption key
- ✅ Create configuration file

**IMPORTANT:** The script will generate an encryption key. **Save this key** and use it on BOTH Pis!

### Option B: Manual Installation

If you prefer manual installation:

```bash
# Clone repository
cd ~
git clone https://github.com/yourusername/shadowmesh.git
cd shadowmesh

# Build daemon
go build -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon/

# Install system-wide
sudo cp bin/shadowmesh-daemon /usr/local/bin/
sudo chmod +x /usr/local/bin/shadowmesh-daemon

# Verify installation
which shadowmesh-daemon
```

Expected output:
```
-rwxr-xr-x 1 pi pi 9.0M Nov 16 19:33 bin/shadowmesh-daemon
/usr/local/bin/shadowmesh-daemon
```

---

## Step 2: Generate Shared Encryption Key

On **either machine**, generate a key:

```bash
openssl rand -hex 32
```

Example output:
```
683619f144d2c3354f47c51c7470042c26ff9f1d44d17140235d50b708cdc059
```

**IMPORTANT:** Copy this key - you'll need it on BOTH machines!

---

## Step 3: Configure Raspberry Pi A (Initiator)

**On Pi A**, create configuration:

```bash
sudo mkdir -p /etc/shadowmesh
sudo nano /etc/shadowmesh/daemon.yaml
```

Paste this configuration (**replace `YOUR_GENERATED_KEY_HERE` with key from Step 2**):

```yaml
daemon:
  listen_address: "127.0.0.1:9090"
  log_level: "info"

network:
  tap_device: "tap0"
  local_ip: "10.0.0.1/24"  # Pi A uses .1

encryption:
  key: "YOUR_GENERATED_KEY_HERE"  # Paste key from Step 2

peer:
  address: ""

nat:
  enabled: true
  stun_server: "stun.l.google.com:19302"
```

Save and exit: `Ctrl+X`, then `Y`, then `Enter`

---

## Step 4: Configure Raspberry Pi B (Responder)

**On Pi B**, create configuration:

```bash
sudo mkdir -p /etc/shadowmesh
sudo nano /etc/shadowmesh/daemon.yaml
```

Paste this configuration (**SAME key as Pi A!**):

```yaml
daemon:
  listen_address: "127.0.0.1:9090"
  log_level: "info"

network:
  tap_device: "tap0"
  local_ip: "10.0.0.2/24"  # Pi B uses .2

encryption:
  key: "YOUR_GENERATED_KEY_HERE"  # SAME key as Pi A!

peer:
  address: ""

nat:
  enabled: true
  stun_server: "stun.l.google.com:19302"
```

Save and exit: `Ctrl+X`, then `Y`, then `Enter`

---

## Step 5: Start Daemon on Raspberry Pi B (Responder)

**On Pi B**, navigate to shadowmesh directory and start daemon:

```bash
cd ~/shadowmesh
sudo ./bin/shadowmesh-daemon /etc/shadowmesh/daemon.yaml
```

**OR** if you installed system-wide:

```bash
sudo shadowmesh-daemon /etc/shadowmesh/daemon.yaml
```

Expected output:
```
ShadowMesh Daemon v0.1.0-epic2
Loading configuration from: /etc/shadowmesh/daemon.yaml
Configuration loaded successfully
  Listen Address: 127.0.0.1:9090
  TAP Device: tap0
  Local IP: 10.0.0.2/24

Starting ShadowMesh daemon...
Creating TAP device: tap0
✅ TAP device tap0 created with IP 10.0.0.2/24
✅ TAP device reading/writing started
Initializing encryption pipeline...
✅ Encryption pipeline started (ChaCha20-Poly1305)
Starting HTTP API on 127.0.0.1:9090
✅ HTTP API started successfully
✅ ShadowMesh daemon started successfully
   API listening on: 127.0.0.1:9090
   TAP device: tap0 (10.0.0.2/24)

Use 'shadowmesh connect <peer-address>' to establish P2P connection
Press Ctrl+C to shutdown gracefully
```

---

## Step 6: Start Daemon on Raspberry Pi A (Initiator)

**On Pi A**, navigate to shadowmesh directory and start daemon:

```bash
cd ~/shadowmesh
sudo ./bin/shadowmesh-daemon /etc/shadowmesh/daemon.yaml
```

**OR** if you installed system-wide:

```bash
sudo shadowmesh-daemon /etc/shadowmesh/daemon.yaml
```

Same expected output as Pi B (but with IP 10.0.0.1).

---

## Step 7: Find Pi B's Public IP Address

**On Pi B**, find its public IP address:

```bash
# Option 1: Use curl
curl ifconfig.me

# Option 2: Use ip command for local IP (if on same LAN)
ip addr show | grep "inet " | grep -v 127.0.0.1
```

Example output: `203.0.113.42` or `192.168.1.100` (if on same network)

**Write down this IP address** - you'll need it for the next step!

---

## Step 8: Establish Connection from Pi A to Pi B

**On Pi A**, connect to Pi B using the IP address from Step 7:

```bash
# Replace PI_B_IP_ADDRESS with actual IP from Step 7
PI_B_IP="203.0.113.42"  # Example: replace with your Pi B's IP

curl -X POST http://127.0.0.1:9090/connect \
  -H "Content-Type: application/json" \
  -d "{\"peer_address\": \"${PI_B_IP}:9001\"}"
```

Expected response:
```json
{
  "status": "success",
  "message": "Connected to peer at 192.168.1.100:9001"
}
```

---

## Step 9: Verify Connection Status

On **both Raspberry Pis**, check status:

```bash
curl http://127.0.0.1:9090/status
```

Expected response:
```json
{
  "status": "success",
  "state": "Connected",
  "details": {
    "state": "Connected",
    "tap_device": "tap0",
    "local_ip": "10.0.0.1/24",
    "peer_address": "192.168.1.100:9001",
    "connected": true
  }
}
```

---

## Step 10: Test Encrypted Tunnel

### From Pi A → Pi B:

```bash
ping 10.0.0.2
```

Expected output:
```
PING 10.0.0.2 (10.0.0.2): 56 data bytes
64 bytes from 10.0.0.2: icmp_seq=0 ttl=64 time=2.345 ms
64 bytes from 10.0.0.2: icmp_seq=1 ttl=64 time=1.823 ms
```

### From Pi B → Pi A:

```bash
ping 10.0.0.1
```

Expected output:
```
PING 10.0.0.1 (10.0.0.1): 56 data bytes
64 bytes from 10.0.0.1: icmp_seq=0 ttl=64 time=2.123 ms
64 bytes from 10.0.0.1: icmp_seq=1 ttl=64 time=1.956 ms
```

---

## Step 11: Verify Traffic is Encrypted

On **Pi A**, capture network traffic:

```bash
sudo tcpdump -i eth0 port 9001 -X | head -50
```

Expected: You should see **encrypted binary data**, NOT plaintext ICMP packets.

Example encrypted traffic:
```
19:45:23.123456 IP 192.168.1.50.54321 > 192.168.1.100.9001: Flags [P.], seq 1:145, ack 1, win 502, length 144
0x0000:  4500 00b4 1234 4000 4006 abcd c0a8 0132  E....4@.@......2
0x0010:  c0a8 0164 d431 2329 5a8f 3c12 9f6e 4a83  ...d.1#)Z.<..nJ.
0x0020:  5018 01f6 8c3d 0000 a7b9 f2e4 c8d1 5f3a  P....=........_:
[... encrypted binary data ...]
```

---

## Step 12: Disconnect and Shutdown

### Disconnect:

```bash
curl -X POST http://127.0.0.1:9090/disconnect
```

### Shutdown Daemon:

Press `Ctrl+C` in the terminal running the daemon.

Expected output:
```
🛑 Shutdown signal received, stopping daemon...
Stopping daemon components...
Frame router outbound stopped
Frame router inbound stopped
✅ Encryption pipeline stopped
✅ TAP device stopped
✅ Daemon stopped successfully
```

---

## Success Criteria ✅

- [ ] Both daemons started without errors
- [ ] Connection established (status shows "Connected")
- [ ] Ping works from Machine A → Machine B
- [ ] Ping works from Machine B → Machine A
- [ ] Network capture shows encrypted traffic
- [ ] Graceful shutdown works

---

## Troubleshooting

### "Permission denied" when starting daemon
**Solution:** Run with `sudo`

### "Connection refused" when connecting
**Solutions:**
- Verify Machine B's IP address is correct
- Check firewall allows port 9001
- Ensure Machine B daemon is running

### Ping fails through tunnel
**Solutions:**
- Check both TAP devices created: `ip addr show tap0`
- Verify IPs are different (.1 and .2)
- Check daemon logs for errors
- Verify same encryption key on both machines

### "NAT detection failed"
**Solution:** Disable NAT if not needed:
```yaml
nat:
  enabled: false
```

---

## Next Steps After Successful Test

1. Document any issues encountered
2. Measure performance (ping latency, throughput)
3. Test failure scenarios (disconnect/reconnect)
4. Get project lead (james) sign-off
5. Resume Epic 2 retrospective with findings

---

## Quick Reference Commands

```bash
# Start daemon
sudo ./bin/shadowmesh-daemon /etc/shadowmesh/daemon.yaml

# Connect to peer
curl -X POST http://127.0.0.1:9090/connect \
  -H "Content-Type: application/json" \
  -d '{"peer_address": "PEER_IP:9001"}'

# Check status
curl http://127.0.0.1:9090/status

# Test tunnel
ping 10.0.0.X

# Disconnect
curl -X POST http://127.0.0.1:9090/disconnect

# Shutdown
# Press Ctrl+C
```

---

**Ready to test!** Follow these steps and the P2P tunnel should work. 🚀
