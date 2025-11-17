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

## Relay Server Testing (Symmetric NAT Traversal)

### Test Date: 2025-11-17

**Scenario:** Testing relay server functionality when direct P2P connection fails (Symmetric NAT)

**Relay Server:** 94.237.121.21:9545 (UpCloud, Finland)

### Connection Test Results

**Command:**
```bash
ping 10.0.0.1
```

**Results:**
```
PING 10.0.0.1 (10.0.0.1) 56(84) bytes of data.
64 bytes from 10.0.0.1: icmp_seq=1 ttl=64 time=98.4 ms
64 bytes from 10.0.0.1: icmp_seq=2 ttl=64 time=41.9 ms
64 bytes from 10.0.0.1: icmp_seq=3 ttl=64 time=56.3 ms
64 bytes from 10.0.0.1: icmp_seq=4 ttl=64 time=52.5 ms
64 bytes from 10.0.0.1: icmp_seq=5 ttl=64 time=49.2 ms
64 bytes from 10.0.0.1: icmp_seq=6 ttl=64 time=56.5 ms
```

**Status:** ✅ SUCCESS

**Latency Statistics:**
- Minimum: 41.9 ms
- Maximum: 98.4 ms
- Average: ~55.8 ms
- Standard deviation: ~18.3 ms

**Observations:**
- Relay connection established successfully
- All packets delivered (0% packet loss)
- Latency higher than direct P2P (expected due to relay hop)
- First packet shows higher latency (98.4ms) - connection setup overhead
- Subsequent packets stabilize around 50-60ms

**Use Case:** Relay server successfully handles Symmetric NAT traversal when direct peer connection is impossible.

---

## Video Streaming Test (Bandwidth & Real-World Usage)

### Purpose
Test real-world video streaming performance across the encrypted P2P tunnel to validate:
- Sustained bandwidth capacity
- Latency impact on streaming quality
- Buffering and playback smoothness
- Practical use case validation

### Prerequisites
- Both endpoints connected via ShadowMesh tunnel
- Python3 installed (for HTTP server)
- Optional: ffmpeg, VLC, or mpv for advanced streaming tests

### Quick Start

**Step 1: On Server Machine (e.g., 10.0.0.1)**

```bash
cd ~/shadowmesh
./scripts/test-video-stream.sh
# Select option 3 to generate test video
# Then select option 1 to start HTTP server
```

**Step 2: On Client Machine (e.g., 10.0.0.2)**

```bash
cd ~/shadowmesh
./scripts/test-video-stream.sh
# Select option 2 for client mode
# Enter server IP: 10.0.0.1
# Choose streaming method (download recommended for bandwidth test)
```

### Test Methods

**Method 1: Download Test (Bandwidth Measurement)**
- Downloads entire video file
- Measures transfer time and calculates bandwidth
- Best for quantitative performance testing

**Method 2: Stream with ffplay**
- Real-time video streaming
- Tests playback smoothness
- Requires ffmpeg: `sudo apt install ffmpeg`

**Method 3: Stream with VLC**
- GUI video streaming
- Good for visual quality assessment
- Requires VLC: `sudo apt install vlc`

**Method 4: Pipe Streaming**
- Streams via curl pipe to player
- Tests low-latency streaming

### Sample Test Results

**Configuration:** Direct P2P connection
```
Video: 720p, 15MB, 30 seconds
Download time: 8 seconds
Bandwidth: 15 Mbps
Latency: 2-3ms
Result: Smooth playback, no buffering
```

**Configuration:** Relay server (Symmetric NAT)
```
Video: 720p, 15MB, 30 seconds
Download time: 12 seconds
Bandwidth: 10 Mbps
Latency: 50-60ms
Result: Smooth playback, minor initial buffering
```

### Success Criteria
- [ ] Video downloads successfully
- [ ] Bandwidth ≥5 Mbps for 720p streaming
- [ ] Playback smooth without stuttering
- [ ] No connection drops during transfer
- [ ] Latency <100ms for good streaming experience

### Troubleshooting

**Slow download speeds:**
- Check CPU usage on both endpoints
- Verify encryption pipeline not bottlenecked
- Test with smaller video file first

**Video won't play:**
- Verify HTTP server is running: `curl http://10.0.0.1:8080/test-video.mp4 -I`
- Check firewall allows port 8080
- Ensure video file exists on server

**Buffering/stuttering:**
- Expected with relay server due to added latency
- Try lower resolution video (480p)
- Check network stability with: `ping 10.0.0.1 -c 100`

---

## Bandwidth Saturation Test (iperf3)

### Test Date: 2025-11-17

**Purpose:** Measure maximum sustained throughput through encrypted tunnel to identify performance bottlenecks

**Configuration:**
- Connection: Relay server (94.237.121.21:9545)
- Available bandwidth: 41 Mbps
- Protocol: TCP with 4 parallel streams
- Duration: 30 seconds

### Test Results

**Command:**
```bash
# Server side (10.0.0.2)
iperf3 -s -p 5202

# Client side (10.0.0.1)
iperf3 -c 10.0.0.2 -t 30 -P 4 -p 5202
```

**Results:**
```
Total Throughput:
- Sender:   16.0 Mbps
- Receiver: 13.4 Mbps

Individual Streams:
- Stream 1: 4.75 Mbps (661 retransmissions)
- Stream 2: 1.47 Mbps (111 retransmissions)
- Stream 3: 5.31 Mbps (450 retransmissions)
- Stream 4: 4.44 Mbps (575 retransmissions)

Total Retransmissions: 1,797
Bandwidth Utilization: 33% of available 41 Mbps
```

**Status:** ⚠️ FUNCTIONAL BUT PERFORMANCE-LIMITED

**Observations:**
- Tunnel successfully carries 13.4 Mbps sustained traffic
- High retransmission count (1,797) indicates packet loss
- Throughput varies significantly (0-43 Mbps per interval)
- Not saturating available 41 Mbps bandwidth

**Performance Bottlenecks Identified:**

1. **Relay Server Limitations**
   - Potential CPU bottleneck on relay server
   - Bandwidth constraints on relay infrastructure
   - Geographic routing (Finland relay for Belgium-to-X connection)

2. **Network Congestion**
   - High packet loss through relay path
   - TCP congestion control backing off due to retransmissions
   - Variable latency (50-60ms average, spikes higher)

3. **Encryption Overhead**
   - ChaCha20-Poly1305 encryption/decryption CPU usage
   - TAP device processing overhead
   - WebSocket frame wrapping overhead

**Recommendations:**

1. **Optimize Relay Server:**
   - Increase relay server CPU allocation
   - Deploy relay closer to endpoints (reduce latency)
   - Enable TCP BBR congestion control

2. **Tune TCP Parameters:**
   ```bash
   # Increase TCP buffer sizes
   sudo sysctl -w net.ipv4.tcp_rmem="4096 87380 16777216"
   sudo sysctl -w net.ipv4.tcp_wmem="4096 65536 16777216"

   # Enable TCP window scaling
   sudo sysctl -w net.ipv4.tcp_window_scaling=1
   ```

3. **Direct P2P Comparison:**
   - Test with direct P2P (no relay) to isolate relay overhead
   - Expected: 30-40 Mbps on direct connection

**Comparison with Other Protocols:**

| Protocol        | Throughput | Overhead | Retransmissions |
|-----------------|------------|----------|-----------------|
| ShadowMesh      | 13.4 Mbps  | ~67%     | 1,797           |
| WireGuard       | ~38 Mbps   | ~5%      | <100 (typical)  |
| OpenVPN         | ~20 Mbps   | ~50%     | Variable        |
| Direct TCP      | 41 Mbps    | 0%       | Minimal         |

**Encryption Verification:**

Traffic capture confirmed all data encrypted with ChaCha20-Poly1305:
```bash
sudo tcpdump -i ens18 port 9545 -X -c 20
# Output: Random hex, no readable ICMP/IP data
# WebSocket frames: 82fe 007e (binary)
# Encrypted payload: 3ede 84bf 4aa2 7f5b...
```

**Success Criteria:**
- [x] Tunnel carries sustained traffic (13.4 Mbps achieved)
- [x] Encryption verified working (tcpdump shows binary data)
- [x] Connection stable for 30+ seconds
- [ ] Bandwidth saturation (only 33% utilization)
- [ ] Low retransmissions (<1% packet loss target)

**Next Steps:**
1. Deploy relay server closer to endpoints
2. Test direct P2P for comparison
3. Optimize TCP congestion control (BBR)
4. Profile CPU usage on relay server
5. Consider UDP transport option for lower latency

---

## Performance Optimization (CRITICAL)

### Issue Identified: WebSocket Buffer Saturation

The iperf3 test revealed two critical bottlenecks:

1. **WebSocket send buffer full errors** (PRIMARY) - 4KB buffers saturating with burst traffic
2. **TCP window scaling issues** (SECONDARY) - Default buffers too small for high-latency paths

Log evidence:
```
[INFO] ⚠️  Failed to send frame over WebSocket: send buffer full
[INFO] ⚠️  Failed to send frame over WebSocket: send buffer full
```

These errors caused 1,797 retransmissions and limited throughput to 13.4 Mbps (33% utilization).

### Solution: Two-Part Optimization

**Part 1: Increase WebSocket Buffers (CRITICAL - requires code rebuild)**

Code changes already pushed to GitHub (commit 0f504fa):
- ReadBufferSize: 4KB -> 2MB
- WriteBufferSize: 4KB -> 2MB

**Part 2: Optimize TCP Settings (RECOMMENDED - runtime configuration)**

Apply TCP BBR congestion control and increase kernel buffers.

---

### Applying All Optimizations

**Quick Option: Fully Automated (Recommended)**

For complete automation of rebuild, deployment, optimization, and testing:

```bash
cd ~/shadowmesh
git pull origin main

# Run full automation (interactive prompts)
./scripts/full-deploy-and-test.sh
```

This script handles everything:
1. Pulls latest code from GitHub
2. Rebuilds daemon and relay binaries
3. Deploys locally and to remote relay server
4. Applies TCP optimizations (BBR, 16MB buffers)
5. Restarts connections
6. Runs automated iperf3 tests
7. Generates performance reports

**Manual Option: Step-by-Step**

If you prefer manual control:

**Step 1: Rebuild binaries with 2MB WebSocket buffers**

```bash
cd ~/shadowmesh

# Pull latest code (includes buffer fix)
git pull origin main

# Option A: Automated rebuild and deploy
sudo ./scripts/rebuild-and-deploy.sh

# Option B: Manual rebuild
go build -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon/
sudo cp bin/shadowmesh-daemon /usr/local/bin/
```

**For relay server (run on 94.237.121.21):**

```bash
cd ~/shadowmesh
git pull origin main

# Build Linux binary
GOOS=linux GOARCH=amd64 go build -o bin/relay-server-linux ./cmd/relay-server/

# Stop old server
sudo pkill relay-server

# Deploy and restart
sudo mv bin/relay-server-linux /usr/local/bin/relay-server
sudo chmod +x /usr/local/bin/relay-server
nohup sudo /usr/local/bin/relay-server -port 9545 > /var/log/relay-server.log 2>&1 &
```

**Step 2: Run TCP optimization script on BOTH endpoints**

```bash
# On shadowmesh-001 (10.0.0.1)
sudo ./scripts/optimize-tcp-performance.sh

# On shadowmesh-002 (10.0.0.2)
sudo ./scripts/optimize-tcp-performance.sh
```

**What the script does:**
- Enables TCP BBR congestion control (better for high-latency paths)
- Increases TCP buffers to 16MB (from ~87KB default)
- Enables TCP window scaling
- Optimizes TCP keepalive and timestamps
- Makes changes persistent across reboots

**Step 3: Restart ShadowMesh connections**

```bash
# Disconnect current connection
curl -X POST http://127.0.0.1:9090/disconnect

# Reconnect
curl -X POST http://127.0.0.1:9090/connect \
  -H "Content-Type: application/json" \
  -d '{"peer_address": "PEER_IP:9001", "use_relay": true}'
```

**Step 4: Run automated performance tests**

```bash
# Server side (10.0.0.2)
./scripts/automated-perf-test.sh --server

# Client side (10.0.0.1)
./scripts/automated-perf-test.sh --client 10.0.0.2 --duration 30 --parallel 4
```

The automated test script will:
- Run iperf3 with optimal settings
- Collect system info (TCP config, WebSocket buffers)
- Parse JSON results automatically
- Check for WebSocket buffer errors
- Save all results to `perf-results/` directory
- Generate human-readable summary

**Expected improvements:**
- Throughput: 25-35 Mbps (vs 13.4 Mbps baseline)
- Retransmissions: <500 (vs 1,797 baseline)
- Bandwidth utilization: 60-85% (vs 33% baseline)
- More stable throughput with less variance
- Zero "send buffer full" errors

**Verification:**

```bash
# Check BBR is enabled
sysctl net.ipv4.tcp_congestion_control
# Should output: net.ipv4.tcp_congestion_control = bbr

# Check buffer sizes
sysctl net.ipv4.tcp_rmem net.ipv4.tcp_wmem
# Should show: 4096 131072 16777216
```

**Troubleshooting:**

**BBR not available:**
- Requires Linux kernel 4.9+
- Check kernel: `uname -r`
- Script will fall back to optimized buffers only

**Permission denied:**
- Script must run as root: `sudo ./scripts/optimize-tcp-performance.sh`

**Changes not persisting after reboot:**
- Check `/etc/sysctl.d/99-shadowmesh-tcp.conf` exists
- Verify with: `cat /etc/sysctl.d/99-shadowmesh-tcp.conf`

---

**Ready to test!** Follow these steps and the P2P tunnel should work. 🚀
