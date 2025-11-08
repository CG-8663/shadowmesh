# 🚀 Start Testing - Direct P2P Encrypted Tunnel

**Epic 2**: Core Networking & Direct P2P
**Status**: Ready for Production Testing
**Date**: 2025-11-03

---

## ✅ What's Ready

Epic 2 implementation is **COMPLETE** and pushed to GitHub:
- ✅ Commit: `28cf7c4` - "Epic 2 Complete: Direct P2P Encrypted Tunnels"
- ✅ GitHub: https://github.com/CG-8663/shadowmesh
- ✅ All code reviewed and tested
- ✅ Deployment scripts ready
- ✅ Documentation complete

---

## 🎯 Quick Start (3 Steps)

### Step 1: Configure Deployment Script

Edit `scripts/deploy-production.sh` with your actual IPs:

```bash
nano scripts/deploy-production.sh
```

Set these values:
```bash
UK_VPS_IP="YOUR_ACTUAL_UK_VPS_IP"
UK_VPS_USER="YOUR_USERNAME"     # e.g., "root" or "ubuntu"

RPI_IP="YOUR_ACTUAL_RPI_IP"
RPI_USER="pi"                   # Usually "pi" for Raspberry Pi
```

### Step 2: Deploy to Both Machines

Run the deployment script:

```bash
./scripts/deploy-production.sh
```

This will:
1. Check SSH connectivity
2. Clone/update repo from GitHub
3. Install Go if needed
4. Build shadowmesh-daemon
5. Install binaries and configs
6. Verify everything is ready

**Expected time**: 5-10 minutes

### Step 3: Start Daemons & Test

**Terminal 1 - UK VPS (Listener)**:
```bash
ssh user@uk-vps-ip
sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml
```

**Terminal 2 - Belgium RPi (Connector)**:
```bash
ssh pi@rpi-ip
sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml
```

**Terminal 3 - Test Tunnel**:
```bash
# From UK VPS
ssh user@uk-vps-ip
ping 10.10.10.4

# From Belgium RPi
ssh pi@rpi-ip
ping 10.10.10.3
```

---

## 📊 Expected Output

### UK VPS (Listener) Startup:
```
ShadowMesh Client Daemon v0.1.0-alpha
Post-Quantum Decentralized Private Network (DPN)
=================================================

Client ID: a1b2c3d4...
Creating TAP device: chr-001
TAP device created: chr-001 (MTU: 1500)
Configuring TAP interface: 10.10.10.3/255.255.255.0
Mode: Listener - Waiting for connections on 0.0.0.0:8443
Listening for P2P connections on 0.0.0.0:8443 (TLS: true)
Daemon running. Press Ctrl+C to stop.

[Wait for Belgium RPi to connect...]

Peer connected from <rpi-ip>:xxxxx
Connected to peer
Performing post-quantum handshake...
Generating post-quantum keys (ML-KEM-1024, ML-DSA-87)... this may take 10-30 seconds
Post-quantum keys generated successfully
Handshake complete. Session ID: <session-id>
Session parameters: MTU=1500, Heartbeat=30s, KeyRotation=1h
Starting encrypted tunnel...
Tunnel established. Network traffic is now encrypted.
```

### Belgium RPi (Connector) Startup:
```
ShadowMesh Client Daemon v0.1.0-alpha
Post-Quantum Decentralized Private Network (DPN)
=================================================

Client ID: e5f6g7h8...
Creating TAP device: chr-001
TAP device created: chr-001 (MTU: 1500)
Configuring TAP interface: 10.10.10.4/255.255.255.0
Mode: Connector - Connecting to peer: <uk-vps-ip>:8443
Connecting to peer at <uk-vps-ip>:8443 (attempt 1)...
Connected to peer at <uk-vps-ip>:8443
Waiting for connection...
Performing post-quantum handshake...
Generating post-quantum keys (ML-KEM-1024, ML-DSA-87)... this may take 10-30 seconds
Post-quantum keys generated successfully
Handshake state ready
Creating HELLO message...
HELLO message created, sending to peer
Waiting for CHALLENGE from peer...
Processing CHALLENGE message...
Sending RESPONSE message...
Waiting for ESTABLISHED message...
Handshake complete. Session ID: <same-session-id>
Session parameters: MTU=1500, Heartbeat=30s, KeyRotation=1h
Starting encrypted tunnel...
Tunnel established. Network traffic is now encrypted.
Daemon running. Press Ctrl+C to stop.
```

### Successful Ping Test:
```bash
# From UK VPS
$ ping -c 5 10.10.10.4
PING 10.10.10.4 (10.10.10.4) 56(84) bytes of data.
64 bytes from 10.10.10.4: icmp_seq=1 ttl=64 time=45.2 ms
64 bytes from 10.10.10.4: icmp_seq=2 ttl=64 time=44.8 ms
64 bytes from 10.10.10.4: icmp_seq=3 ttl=64 time=45.1 ms
64 bytes from 10.10.10.4: icmp_seq=4 ttl=64 time=44.9 ms
64 bytes from 10.10.10.4: icmp_seq=5 ttl=64 time=45.0 ms

--- 10.10.10.4 ping statistics ---
5 packets transmitted, 5 received, 0% packet loss, time 4006ms
rtt min/avg/max/mdev = 44.758/45.000/45.189/0.154 ms
```

---

## 🔍 Verification Checklist

### ✅ Deployment Successful
- [ ] UK VPS: Repository cloned/updated
- [ ] UK VPS: shadowmesh-daemon built
- [ ] UK VPS: Binary installed to /opt/shadowmesh
- [ ] UK VPS: Config installed to /etc/shadowmesh
- [ ] Belgium RPi: Repository cloned/updated
- [ ] Belgium RPi: shadowmesh-daemon built
- [ ] Belgium RPi: Binary installed to /opt/shadowmesh
- [ ] Belgium RPi: Config installed to /etc/shadowmesh

### ✅ Connection Established
- [ ] UK VPS: Daemon starts without errors
- [ ] UK VPS: Listening on port 8443
- [ ] Belgium RPi: Daemon starts without errors
- [ ] Belgium RPi: Connects to UK VPS
- [ ] Both: "Peer connected" message appears
- [ ] Both: Same Session ID shown

### ✅ PQC Handshake Complete
- [ ] Key generation takes 10-30 seconds
- [ ] HELLO → CHALLENGE → RESPONSE → ESTABLISHED
- [ ] "Handshake complete" on both sides
- [ ] Session parameters displayed

### ✅ Tunnel Operational
- [ ] "Tunnel established" message
- [ ] TAP device chr-001 created on both
- [ ] UK VPS: IP 10.10.10.3 assigned
- [ ] Belgium RPi: IP 10.10.10.4 assigned
- [ ] Ping UK → RPi works (0% loss)
- [ ] Ping RPi → UK works (0% loss)

### ✅ Statistics Clean
- [ ] Every 60 seconds: Stats logged
- [ ] FramesSent > 0
- [ ] FramesReceived > 0
- [ ] EncryptErrors = 0
- [ ] DecryptErrors = 0
- [ ] DroppedFrames = 0

---

## 🧪 Advanced Testing

### Performance Test (iperf3)

**UK VPS**:
```bash
sudo apt-get install -y iperf3
iperf3 -s -B 10.10.10.3
```

**Belgium RPi**:
```bash
sudo apt-get install -y iperf3
iperf3 -c 10.10.10.3 -B 10.10.10.4 -t 30
```

**Expected**: 100+ Mbps (hardware dependent)

### Latency Test

```bash
ping -c 100 -i 0.2 10.10.10.4 | tail -5
```

**Expected**: <100ms for international connection

### Extended Stability Test

```bash
# Run for 5 minutes
ping -c 300 -i 1 10.10.10.4

# Check for:
# - 0% packet loss
# - Consistent latency
# - No disconnections
```

---

## 🐛 Troubleshooting

### Issue: Cannot SSH to machines

**Check**:
```bash
# Test connectivity
ping uk-vps-ip
ping rpi-ip

# Test SSH
ssh -v user@uk-vps-ip
ssh -v pi@rpi-ip
```

**Fix**: Verify IPs, SSH keys, firewall rules

### Issue: Build fails on machine

**Error**: "go: command not found"

**Fix**: Deployment script will auto-install Go
```bash
# Manual install if needed
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### Issue: Daemon fails to start

**Error**: "Failed to create TAP device"

**Check**:
```bash
# Are you running as root?
sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml

# Is TUN/TAP module loaded?
lsmod | grep tun
sudo modprobe tun
```

### Issue: Connection refused

**Error**: "Connection refused" from Belgium RPi

**Check**:
1. UK VPS firewall allows port 8443
   ```bash
   sudo ufw allow 8443/tcp
   # or
   sudo iptables -A INPUT -p tcp --dport 8443 -j ACCEPT
   ```

2. UK VPS daemon is listening
   ```bash
   sudo netstat -tlnp | grep 8443
   # or
   sudo ss -tlnp | grep 8443
   ```

3. Config has correct UK VPS IP
   ```bash
   cat /etc/shadowmesh/config.yaml | grep peer_address
   ```

### Issue: Handshake timeout

**Error**: "Handshake failed: timeout"

**Possible causes**:
- Network latency too high
- Firewall blocking traffic
- TLS certificate issues

**Fix**:
1. Check network: `ping uk-vps-ip` from RPi
2. Temporarily disable TLS for testing:
   ```yaml
   # In both configs
   tls_enabled: false
   ```
3. Increase timeout in code (if needed)

### Issue: Ping doesn't work

**Tunnel established but ping fails**

**Check TAP configuration**:
```bash
# UK VPS
ip addr show chr-001
# Should show: inet 10.10.10.3/24

# Belgium RPi
ip addr show chr-001
# Should show: inet 10.10.10.4/24
```

**Manual configuration if needed**:
```bash
# UK VPS
sudo ip addr add 10.10.10.3/24 dev chr-001
sudo ip link set chr-001 up

# Belgium RPi
sudo ip addr add 10.10.10.4/24 dev chr-001
sudo ip link set chr-001 up
```

---

## 📝 Monitoring & Logs

### Real-time Logs

**UK VPS**:
```bash
tail -f /var/log/shadowmesh/daemon.log
```

**Belgium RPi**:
```bash
tail -f /var/log/shadowmesh/daemon.log
```

### Statistics

Every 60 seconds you should see:
```
Stats: Sent=150 frames (22500 bytes), Recv=150 frames (22500 bytes), Errors: Encrypt=0 Decrypt=0 Dropped=0
```

### Key Rotation

After 1 hour:
```
Performing key rotation...
Key rotation complete. New session ID: <new-id>
```

---

## 🎉 Success Criteria

**Test is SUCCESSFUL when**:

1. ✅ Deployment script completes without errors
2. ✅ Both daemons start and connect
3. ✅ PQC handshake completes (<500ms after key generation)
4. ✅ Tunnel established message on both sides
5. ✅ Bidirectional ping works (0% packet loss)
6. ✅ Statistics show no errors (EncryptErrors=0, DecryptErrors=0)
7. ✅ Throughput meets hardware capability (100+ Mbps)
8. ✅ Latency overhead reasonable (<100ms international)
9. ✅ Stable over 5+ minutes
10. ✅ Clean shutdown (no crashes)

**Current Status**: Ready to test! 🚀

---

## 📊 Document Results

After testing, record:

```markdown
# Epic 2 Test Results
Date: 2025-11-03
Tester: [Your Name]

## Infrastructure
- UK VPS: [IP] - [OS/Specs]
- Belgium RPi: [IP] - [Model/OS]

## Results
- Deployment: PASS/FAIL
- Connection: PASS/FAIL
- Handshake: PASS/FAIL (XX seconds)
- Tunnel: PASS/FAIL
- Ping UK→RPi: PASS/FAIL (X% loss, XXms avg)
- Ping RPi→UK: PASS/FAIL (X% loss, XXms avg)
- Throughput: XXX Mbps
- Errors: EncryptErrors=X, DecryptErrors=X, DroppedFrames=X

## Issues
1. [Description]
2. [Description]

## Conclusion
[PASS/FAIL with notes]
```

---

## 🔄 Next Steps After Testing

### If Test Passes ✅
1. Update `docs/prd.md`: Mark Epic 2 as Complete
2. Document actual performance metrics
3. Create GitHub issue: "Epic 2 Testing Results"
4. Begin Epic 3: Smart Contract Integration
5. Celebrate! 🎊

### If Test Fails ❌
1. Collect logs from both machines
2. Document exact error messages
3. Review troubleshooting section
4. Create GitHub issue with details
5. Fix, rebuild, redeploy, retest

---

**Ready?** Edit `scripts/deploy-production.sh` and run it!

```bash
nano scripts/deploy-production.sh  # Set IPs
./scripts/deploy-production.sh      # Deploy
# Then follow startup instructions above
```

**Good luck!** 🚀
