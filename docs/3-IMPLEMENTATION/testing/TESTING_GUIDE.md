# ShadowMesh Phase 3 - Quick Start Testing

## Files Ready

✅ **Binaries Built** (all 3 architectures):
- `shadowmesh-l3-v11-phase3-darwin-arm64` (8.7M) - macOS Apple Silicon
- `shadowmesh-l3-v11-phase3-amd64` (9.2M) - Linux x86_64
- `shadowmesh-l3-v11-phase3-arm64` (8.6M) - Linux ARM64

✅ **Documentation**:
- `PHASE3_TEST_GUIDE.md` - Complete testing procedures
- `V11_PHASE3_COMPLETION.md` - Implementation details
- `V11_UDP_PERFORMANCE_INVESTIGATION.md` - Performance analysis

---

## Quick Local Test (macOS)

Since sudo is configured without password, you can test locally:

```bash
cd /Volumes/BACKUPDISK/webcode/shadowmesh

# Generate test keys
sudo ./shadowmesh-l3-v11-phase3-darwin-arm64 \
  -generate-keys \
  -keydir ./keys-local-test

# Test TUN device creation (will run in background)
sudo ./shadowmesh-l3-v11-phase3-darwin-arm64 \
  -keydir ./keys-local-test \
  -backbone http://209.151.148.121:8080 \
  -ip $(ifconfig en0 | grep "inet " | awk '{print $2}') \
  -tun smtest0 \
  -tun-ip 10.100.0.1 \
  -tun-netmask 24 \
  -port 9443 \
  -udp-port 9444 &

# Wait for startup
sleep 5

# Check TUN device
ifconfig smtest0

# Check process
ps aux | grep shadowmesh-l3-v11-phase3

# Test local ping
ping -c 5 10.100.0.1

# Cleanup
sudo killall shadowmesh-l3-v11-phase3-darwin-arm64
```

---

## Production Deployment

### Step 1: Deploy Binaries

```bash
cd /Volumes/BACKUPDISK/webcode/shadowmesh

# Belgium server (80.229.0.71)
scp shadowmesh-l3-v11-phase3-amd64 pxcghost@80.229.0.71:~/shadowmesh/
ssh pxcghost@80.229.0.71 'chmod +x ~/shadowmesh/shadowmesh-l3-v11-phase3-amd64'

# UK server (195.178.84.80)
scp shadowmesh-l3-v11-phase3-amd64 pxcghost@195.178.84.80:~/
ssh pxcghost@195.178.84.80 'chmod +x ~/shadowmesh-l3-v11-phase3-amd64'
```

### Step 2: Configure Kernel (both servers)

```bash
# Belgium
ssh pxcghost@80.229.0.71 << 'KERNEL_EOF'
sudo sysctl -w net.ipv4.conf.chr001.rp_filter=0
sudo sysctl -w net.ipv4.conf.all.rp_filter=0
sudo sysctl -w net.ipv4.ip_forward=1
echo "Kernel configured"
KERNEL_EOF

# UK
ssh pxcghost@195.178.84.80 << 'KERNEL_EOF'
sudo sysctl -w net.ipv4.conf.chr001.rp_filter=0
sudo sysctl -w net.ipv4.conf.all.rp_filter=0
sudo sysctl -w net.ipv4.ip_forward=1
echo "Kernel configured"
KERNEL_EOF
```

### Step 3: Start UK Server (Listener)

```bash
ssh pxcghost@195.178.84.80

# Run Phase 3
./shadowmesh-l3-v11-phase3-amd64 \
  -keydir ./keys \
  -backbone http://209.151.148.121:8080 \
  -ip 195.178.84.80 \
  -tun chr001 \
  -tun-ip 10.10.10.3 \
  -tun-netmask 24 \
  -port 8443 \
  -udp-port 9443

# Copy the Peer ID from output
```

### Step 4: Start Belgium Server (Initiator)

```bash
ssh pxcghost@80.229.0.71

cd ~/shadowmesh/

# Set UK peer ID from previous step
UK_PEER_ID="<paste-peer-id-here>"

# Run Phase 3
./shadowmesh-l3-v11-phase3-amd64 \
  -keydir ./keys \
  -backbone http://209.151.148.121:8080 \
  -ip 80.229.0.71 \
  -tun chr001 \
  -tun-ip 10.10.10.4 \
  -tun-netmask 24 \
  -port 8443 \
  -udp-port 9443 \
  -connect "$UK_PEER_ID"
```

### Step 5: Test Performance

```bash
# From Belgium server
ping -c 100 10.10.10.3 | tee ~/ping-phase3-results.log

# Expected: <5% packet loss, <50ms latency
```

---

## Success Criteria

### Must Have (Blocking)
- [ ] Binaries deploy without errors
- [ ] P2P connection establishes
- [ ] ICMP packet loss <10%
- [ ] Average latency <100ms

### Should Have (Target)
- [ ] ICMP packet loss <5%
- [ ] Average latency <50ms
- [ ] No crashes during 100-packet test

### Nice to Have (Stretch)
- [ ] Packet loss <1%
- [ ] Average latency <20ms
- [ ] Throughput >100 Mbps (iperf3)

---

## Performance Comparison

| Metric | v11-chr001 (Before) | Phase 3 (Target) | Actual |
|--------|---------------------|------------------|--------|
| Packet Loss | 95% | <5% | ___ |
| Avg Latency | 3000ms | <50ms | ___ |
| Memory Alloc | High | -90% | ___ |

---

## Troubleshooting

### Binary won't execute
- Check permissions: `chmod +x shadowmesh-l3-v11-phase3-*`
- Verify architecture: `file shadowmesh-l3-v11-phase3-amd64`

### TUN device not created
- Check logs for "permission denied"
- Ensure running with sudo/root
- On macOS: May need TunTap driver

### Connection fails
- Verify both servers authenticated with backbone
- Check firewall rules (ports 8443, 9443)
- Ensure IPs are correct in -ip flags

### High packet loss
- Check adaptive buffer logs: `[ADAPTIVE-BUFFER]`
- Monitor system resources
- Consider rolling back to v11-chr001

---

## Next Steps

1. ✅ Local test (optional)
2. ⏳ Deploy to production
3. ⏳ Run ICMP tests
4. ⏳ Document actual results
5. ⏳ Compare to baseline

See `PHASE3_TEST_GUIDE.md` for detailed procedures.
