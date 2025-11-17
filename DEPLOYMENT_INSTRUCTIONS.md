# ShadowMesh Deployment Instructions

## Quick Reference: WebSocket Buffer Fix & Performance Optimization

### Problem Identified
- WebSocket send buffers were only 4KB
- Caused "send buffer full" errors during burst traffic
- Limited throughput to 13.4 Mbps (33% of 41 Mbps available)
- 1,797 retransmissions in 30-second iperf3 test

### Solution Applied
- Increased WebSocket buffers: 4KB → 2MB
- Applied TCP optimizations: BBR, 16MB kernel buffers
- Automated deployment and testing scripts

---

## Deployment Steps

### Option 1: Fully Automated (Recommended)

Run this on **one endpoint** (e.g., shadowmesh-001):

```bash
cd ~/shadowmesh
git pull origin main
chmod +x scripts/*.sh

# This handles everything
./scripts/full-deploy-and-test.sh
```

When prompted:
- Deploy relay server? **Yes** → Enter `root@94.237.121.21`
- Apply TCP optimizations? **Yes**
- Reconnect to peer? **Yes** → Enter peer address
- Run performance tests? **Yes** → Enter peer tunnel IP

### Option 2: Manual Deployment

**On Both Endpoints (shadowmesh-001 and shadowmesh-002):**

```bash
cd ~/shadowmesh
git pull origin main

# Build and deploy daemon
go build -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon/
sudo cp bin/shadowmesh-daemon /usr/local/bin/
sudo systemctl restart shadowmesh

# Apply TCP optimizations
sudo ./scripts/optimize-tcp-performance.sh
```

**On Relay Server (94.237.121.21):**

```bash
cd ~/shadowmesh
git pull origin main

# Build Linux binary
GOOS=linux GOARCH=amd64 go build -o bin/relay-server-linux ./cmd/relay-server/

# Deploy
sudo pkill relay-server
sudo mv bin/relay-server-linux /usr/local/bin/relay-server
sudo chmod +x /usr/local/bin/relay-server
nohup sudo /usr/local/bin/relay-server -port 9545 > /var/log/relay-server.log 2>&1 &

# Verify
curl http://localhost:9545/health
```

**Restart Connections (on endpoint):**

```bash
# Disconnect
curl -X POST http://127.0.0.1:9090/disconnect

# Reconnect with relay
curl -X POST http://127.0.0.1:9090/connect \
  -H "Content-Type: application/json" \
  -d '{"peer_address": "PEER_IP:9001", "use_relay": true}'
```

---

## Running Performance Tests

**On Server Endpoint (e.g., 10.0.0.2):**

```bash
./scripts/automated-perf-test.sh --server
```

**On Client Endpoint (e.g., 10.0.0.1):**

```bash
./scripts/automated-perf-test.sh --client 10.0.0.2 --duration 30 --parallel 4
```

Results saved to: `perf-results/`

---

## Expected Results

### Before Optimization
```
Throughput:      13.4 Mbps (receiver)
Retransmissions: 1,797
Bandwidth Use:   33% of 41 Mbps
Errors:          WebSocket "send buffer full"
```

### After Optimization
```
Throughput:      30-40 Mbps (receiver)
Retransmissions: <500
Bandwidth Use:   80-95% of 41 Mbps
Errors:          None (2MB buffers)
```

---

## Verification Commands

### Check WebSocket Buffer Sizes
```bash
# Daemon code (should show 2MB)
grep -A2 "ReadBufferSize" pkg/daemonmgr/p2p.go

# Relay code (should show 2MB)
grep -A2 "ReadBufferSize" relay/server/config.go
```

### Check TCP Settings
```bash
# BBR enabled?
sysctl net.ipv4.tcp_congestion_control
# Should output: bbr

# Buffer sizes?
sysctl net.ipv4.tcp_rmem net.ipv4.tcp_wmem
# Should output: 4096 131072 16777216
```

### Check for Buffer Errors
```bash
# No "buffer full" messages
journalctl -u shadowmesh --since "5 minutes ago" | grep -i "buffer full"
# Should return nothing
```

### Check Connection Status
```bash
curl http://127.0.0.1:9090/status | python3 -m json.tool
```

---

## Troubleshooting

### Daemon Won't Start
```bash
# Check logs
journalctl -u shadowmesh -f

# Verify binary
ls -lh /usr/local/bin/shadowmesh-daemon

# Test manually
sudo /usr/local/bin/shadowmesh-daemon /etc/shadowmesh/daemon.yaml
```

### Relay Server Down
```bash
# SSH to relay
ssh root@94.237.121.21

# Check process
pgrep relay-server

# Check logs
tail -f /var/log/relay-server.log

# Restart
sudo pkill relay-server
nohup sudo /usr/local/bin/relay-server -port 9545 > /var/log/relay-server.log 2>&1 &
```

### Still Seeing Buffer Errors
```bash
# Verify you're running new binary
strings /usr/local/bin/shadowmesh-daemon | grep -i "2048 KiB"

# Rebuild if needed
cd ~/shadowmesh
git pull origin main
go build -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon/
sudo cp bin/shadowmesh-daemon /usr/local/bin/
sudo systemctl restart shadowmesh
```

### Low Throughput
```bash
# Check TCP settings applied
sysctl net.ipv4.tcp_congestion_control
sysctl net.ipv4.tcp_rmem

# Re-apply if needed
sudo ./scripts/optimize-tcp-performance.sh

# Check CPU usage
top -bn1 | grep shadowmesh
```

---

## File Locations

### Binaries
```
/usr/local/bin/shadowmesh-daemon
/usr/local/bin/relay-server
```

### Configuration
```
/etc/shadowmesh/daemon.yaml
/etc/sysctl.d/99-shadowmesh-tcp.conf
```

### Logs
```
journalctl -u shadowmesh -f
/var/log/relay-server.log
```

### Test Results
```
./perf-results/
```

---

## Quick Commands Cheat Sheet

```bash
# Pull latest code
git pull origin main

# Full automation
./scripts/full-deploy-and-test.sh

# Manual rebuild daemon
go build -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon/
sudo cp bin/shadowmesh-daemon /usr/local/bin/
sudo systemctl restart shadowmesh

# TCP optimization
sudo ./scripts/optimize-tcp-performance.sh

# Run tests
./scripts/automated-perf-test.sh --client 10.0.0.2

# Check status
curl http://127.0.0.1:9090/status

# View logs
journalctl -u shadowmesh -f
```

---

## Support

### Documentation
- Full testing guide: `docs/QUICK_START_TESTING.md`
- Project specs: `PROJECT_SPEC.md`
- Security details: `ENHANCED_SECURITY_SPECS.md`

### GitHub Repository
https://github.com/CG-8663/shadowmesh

### Recent Commits
- `5673f22` - Automated deployment documentation
- `2392e59` - Full automation scripts
- `0f504fa` - Critical WebSocket buffer fix (4KB → 2MB)
- `e0b660c` - TCP optimization script
