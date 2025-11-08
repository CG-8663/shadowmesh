# Epic 2: Direct P2P Testing & Validation

**Epic**: Core Networking & Direct P2P
**Status**: Implementation Complete - Ready for Testing
**Date**: 2025-11-03

---

## Test Overview

This document outlines the comprehensive testing plan for **Epic 2: Core Networking & Direct P2P**, validating the direct encrypted tunnel between peers without relay infrastructure.

### Test Infrastructure

- **UK VPS**: Listener mode, TAP chr-001 at 10.10.10.3
- **Belgium RPi**: Connector mode, TAP chr-001 at 10.10.10.4
- **Connection**: Direct WebSocket Secure (WSS) on port 8443
- **Cryptography**: ML-KEM-1024 + ML-DSA-87 (Post-Quantum)
- **Encryption**: ChaCha20-Poly1305

---

## Test Phases

### Phase 1: Deployment Validation ✓

**Objective**: Verify binaries deploy correctly and configurations are valid.

**Tests**:
1. ✓ Build all binaries (`make build`)
2. ✓ Verify binary sizes and permissions
3. ✓ Deploy to UK VPS
4. ✓ Deploy to Belgium RPi
5. ✓ Validate configuration files parse correctly

**Success Criteria**:
- All binaries build without errors
- Deployment scripts execute without errors
- Config files pass validation (`--show-config`)

**Commands**:
```bash
# Local machine
make build
ls -lh build/

# Deploy
./scripts/quick-p2p-deploy.sh

# Verify configs
ssh user@uk-vps "sudo /opt/shadowmesh/shadowmesh-daemon --show-config --config /etc/shadowmesh/config.yaml"
ssh pi@rpi "sudo /opt/shadowmesh/shadowmesh-daemon --show-config --config /etc/shadowmesh/config.yaml"
```

---

### Phase 2: Connection Establishment

**Objective**: Validate P2P connection establishment and PQC handshake.

**Test 2.1: Listener Mode (UK VPS)**

Start the daemon in listener mode:
```bash
ssh user@uk-vps
sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml
```

**Expected Output**:
```
ShadowMesh Client Daemon v0.1.0-alpha
Post-Quantum Decentralized Private Network (DPN)
=================================================

Client ID: <32-byte-hex>
Creating TAP device: chr-001
TAP device created: chr-001 (MTU: 1500)
Configuring TAP interface: 10.10.10.3/255.255.255.0
Mode: Listener - Waiting for connections on 0.0.0.0:8443
Listening for P2P connections on 0.0.0.0:8443 (TLS: true)
Daemon running. Press Ctrl+C to stop.
```

**Validation**:
- [ ] Daemon starts without errors
- [ ] TAP device chr-001 created
- [ ] Listening on port 8443
- [ ] No error messages in first 30 seconds

**Test 2.2: Connector Mode (Belgium RPi)**

Start the daemon in connector mode:
```bash
ssh pi@rpi
sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml
```

**Expected Output**:
```
ShadowMesh Client Daemon v0.1.0-alpha
Post-Quantum Decentralized Private Network (DPN)
=================================================

Client ID: <32-byte-hex>
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
Handshake complete. Session ID: <session-id>
Session parameters: MTU=1500, Heartbeat=30s, KeyRotation=1h
Starting encrypted tunnel...
Tunnel established. Network traffic is now encrypted.
Daemon running. Press Ctrl+C to stop.
```

**Validation**:
- [ ] Connection to UK VPS succeeds
- [ ] PQC key generation completes (10-30 seconds)
- [ ] 4-way handshake completes (HELLO → CHALLENGE → RESPONSE → ESTABLISHED)
- [ ] Session ID generated
- [ ] Tunnel established

**Test 2.3: Listener Receives Connection**

On UK VPS terminal, verify you see:
```
Peer connected from <rpi-ip>:xxxxx
Connected to peer
Performing post-quantum handshake...
Generating post-quantum keys...
Post-quantum keys generated successfully
Handshake complete. Session ID: <same-session-id>
Tunnel established. Network traffic is now encrypted.
```

**Validation**:
- [ ] Peer connection accepted
- [ ] Both sides show same Session ID
- [ ] Both sides show "Tunnel established"

---

### Phase 3: TAP Device Verification

**Objective**: Verify TAP devices are created with correct configuration.

**Test 3.1: UK VPS TAP Configuration**

```bash
ssh user@uk-vps
ip addr show chr-001
ip route | grep chr-001
```

**Expected Output**:
```
<N>: chr-001: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 ...
    inet 10.10.10.3/24 scope global chr-001
```

**Validation**:
- [ ] chr-001 device exists
- [ ] IP address: 10.10.10.3/24
- [ ] MTU: 1500
- [ ] State: UP

**Test 3.2: Belgium RPi TAP Configuration**

```bash
ssh pi@rpi
ip addr show chr-001
ip route | grep chr-001
```

**Expected Output**:
```
<N>: chr-001: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 ...
    inet 10.10.10.4/24 scope global chr-001
```

**Validation**:
- [ ] chr-001 device exists
- [ ] IP address: 10.10.10.4/24
- [ ] MTU: 1500
- [ ] State: UP

---

### Phase 4: Encrypted Tunnel Testing

**Objective**: Validate end-to-end encrypted tunnel with ICMP traffic.

**Test 4.1: UK VPS → Belgium RPi (Ping)**

On UK VPS:
```bash
ping -c 10 10.10.10.4
```

**Expected Output**:
```
PING 10.10.10.4 (10.10.10.4) 56(84) bytes of data.
64 bytes from 10.10.10.4: icmp_seq=1 ttl=64 time=XX ms
64 bytes from 10.10.10.4: icmp_seq=2 ttl=64 time=XX ms
...
10 packets transmitted, 10 received, 0% packet loss
```

**Validation**:
- [ ] All 10 packets received (0% loss)
- [ ] Latency reasonable (<100ms for international)
- [ ] No errors in daemon logs

**Test 4.2: Belgium RPi → UK VPS (Ping)**

On Belgium RPi:
```bash
ping -c 10 10.10.10.3
```

**Expected Output**:
```
PING 10.10.10.3 (10.10.10.3) 56(84) bytes of data.
64 bytes from 10.10.10.3: icmp_seq=1 ttl=64 time=XX ms
64 bytes from 10.10.10.3: icmp_seq=2 ttl=64 time=XX ms
...
10 packets transmitted, 10 received, 0% packet loss
```

**Validation**:
- [ ] All 10 packets received (0% loss)
- [ ] Latency consistent with Test 4.1
- [ ] Bidirectional communication works

**Test 4.3: Extended Ping Test (Stability)**

```bash
# Run for 5 minutes
ping -c 300 -i 1 10.10.10.4
```

**Validation**:
- [ ] 0% packet loss over 5 minutes
- [ ] No connection drops
- [ ] Stable latency

---

### Phase 5: Encryption Validation

**Objective**: Verify traffic is encrypted and statistics are accurate.

**Test 5.1: Daemon Statistics**

After running ping tests, check daemon logs (both machines):

```bash
tail -f /var/log/shadowmesh/daemon.log
```

Look for stats output (every 60 seconds):
```
Stats: Sent=300 frames (45000 bytes), Recv=300 frames (45000 bytes), Errors: Encrypt=0 Decrypt=0 Dropped=0
```

**Validation**:
- [ ] FramesSent > 0
- [ ] FramesReceived > 0
- [ ] EncryptErrors = 0
- [ ] DecryptErrors = 0
- [ ] DroppedFrames = 0

**Test 5.2: Packet Capture (Encrypted Traffic)**

On UK VPS, capture WebSocket traffic:
```bash
sudo tcpdump -i eth0 'tcp port 8443' -w /tmp/encrypted.pcap -c 100
```

Then run: `ping -c 10 10.10.10.4` from UK VPS

Analyze capture:
```bash
tcpdump -r /tmp/encrypted.pcap -X | head -50
```

**Validation**:
- [ ] Traffic appears as TLS/WebSocket binary
- [ ] No plaintext ICMP visible
- [ ] All traffic encrypted

**Test 5.3: TAP Device Capture (Decrypted Traffic)**

On Belgium RPi, capture TAP device:
```bash
sudo tcpdump -i chr-001 -w /tmp/decrypted.pcap -c 100
```

Analyze:
```bash
tcpdump -r /tmp/decrypted.pcap -n
```

**Validation**:
- [ ] ICMP packets visible (decrypted)
- [ ] Source/dest: 10.10.10.3 ↔ 10.10.10.4
- [ ] Normal IP traffic on TAP

---

### Phase 6: Performance Testing

**Objective**: Measure throughput and latency overhead.

**Test 6.1: Throughput Test (iperf3)**

Install iperf3 on both machines:
```bash
# UK VPS
sudo apt-get install -y iperf3
iperf3 -s -B 10.10.10.3

# Belgium RPi
sudo apt-get install -y iperf3
iperf3 -c 10.10.10.3 -B 10.10.10.4 -t 30
```

**Validation**:
- [ ] Throughput measured
- [ ] No errors during test
- [ ] Record: _____ Mbps

**Target**: 100+ Mbps (depends on hardware)

**Test 6.2: Latency Measurement**

```bash
ping -c 100 -i 0.2 10.10.10.4 | tail -5
```

**Validation**:
- [ ] Average latency recorded: _____ ms
- [ ] Overhead <5ms vs direct connection
- [ ] Jitter minimal

---

### Phase 7: Resilience Testing

**Objective**: Test reconnection, error handling, and recovery.

**Test 7.1: Graceful Restart**

1. Stop Belgium RPi daemon (Ctrl+C)
2. Wait 10 seconds
3. Restart Belgium RPi daemon

**Validation**:
- [ ] Daemon stops cleanly
- [ ] TAP device removed
- [ ] Restart successful
- [ ] Reconnection automatic
- [ ] Ping resumes

**Test 7.2: Network Interruption**

1. On Belgium RPi, simulate network drop:
   ```bash
   sudo iptables -A OUTPUT -p tcp --dport 8443 -j DROP
   sleep 5
   sudo iptables -D OUTPUT -p tcp --dport 8443 -j DROP
   ```

**Validation**:
- [ ] Daemon detects disconnection
- [ ] Automatic reconnection attempts
- [ ] Connection re-established
- [ ] Tunnel resumes

---

### Phase 8: Security Validation

**Objective**: Verify cryptographic implementation and security properties.

**Test 8.1: Key Rotation**

Wait 1 hour (or modify config to shorter interval) and monitor logs:

```
Performing key rotation...
Key rotation complete. New session ID: <new-id>
```

**Validation**:
- [ ] Key rotation occurs at scheduled interval
- [ ] New Session ID generated
- [ ] Tunnel remains operational
- [ ] No dropped packets during rotation

**Test 8.2: Certificate Validation (if TLS enabled)**

Try connecting with invalid certificate:

**Validation**:
- [ ] Connection rejected if cert invalid
- [ ] TLS handshake fails appropriately
- [ ] Error logged clearly

---

## Test Results Summary

### Pass/Fail Criteria

**PASS** if all of the following are true:
1. ✓ Deployment succeeds on both machines
2. ✓ P2P connection establishes (listener + connector)
3. ✓ PQC handshake completes (4-way: HELLO/CHALLENGE/RESPONSE/ESTABLISHED)
4. ✓ TAP devices created with correct IPs
5. ✓ Bidirectional ping works (0% packet loss)
6. ✓ Encryption stats show no errors
7. ✓ Throughput meets hardware capability
8. ✓ Latency overhead <5ms
9. ✓ Automatic reconnection works
10. ✓ Key rotation works

**FAIL** if any of:
- Connection cannot be established
- Handshake fails or times out
- Packet loss >1%
- EncryptErrors or DecryptErrors >0
- Daemon crashes or hangs

---

## Test Execution Checklist

### Pre-Test
- [ ] Build fresh binaries: `make build`
- [ ] Review configs: `configs/vps-uk-listener.yaml`, `configs/rpi-belgium-connector.yaml`
- [ ] Update deployment script with actual IPs
- [ ] Verify SSH access to both machines

### Deployment
- [ ] Run `./scripts/quick-p2p-deploy.sh`
- [ ] Verify binaries uploaded
- [ ] Verify configs uploaded

### Execution
- [ ] Start UK VPS (listener)
- [ ] Start Belgium RPi (connector)
- [ ] Verify connection established
- [ ] Run Phase 3: TAP verification
- [ ] Run Phase 4: Ping tests
- [ ] Run Phase 5: Encryption validation
- [ ] Run Phase 6: Performance tests
- [ ] Run Phase 7: Resilience tests
- [ ] Run Phase 8: Security tests

### Post-Test
- [ ] Collect logs from both machines
- [ ] Document results
- [ ] Update roadmap/PRD
- [ ] Create GitHub issue with results (if using)

---

## Known Issues / Limitations

1. **TLS Certificates**: Currently using `tls_skip_verify: false` but not providing cert files - may need self-signed certs or disable TLS for initial testing
2. **TAP Configuration**: Network configuration (ip addr assignment) may require manual setup if `shared/networking/ifconfig.go` not fully integrated
3. **Root Privileges**: Daemon requires root for TAP device creation

---

## Next Steps After Testing

Based on test results:

1. **If all tests pass**:
   - Update roadmap: Mark Epic 2 as ✓ Complete
   - Begin Epic 3: Multi-Hop Routing
   - Document performance baseline

2. **If tests fail**:
   - Analyze logs for root cause
   - Fix issues and rebuild
   - Re-test failing phases

3. **Performance optimization** (if needed):
   - Profile encryption overhead
   - Optimize buffer sizes
   - Tune WebSocket parameters

---

## Test Log Template

```markdown
# Epic 2 Test Results
Date: YYYY-MM-DD
Tester: [Name]

## Infrastructure
- UK VPS: [IP] - [OS/Version]
- Belgium RPi: [IP] - [OS/Version]

## Test Results
- Phase 1 (Deployment): PASS/FAIL
- Phase 2 (Connection): PASS/FAIL
- Phase 3 (TAP Devices): PASS/FAIL
- Phase 4 (Tunnel): PASS/FAIL
- Phase 5 (Encryption): PASS/FAIL
- Phase 6 (Performance): PASS/FAIL
- Phase 7 (Resilience): PASS/FAIL
- Phase 8 (Security): PASS/FAIL

## Performance Metrics
- Ping latency: ___ ms
- Throughput: ___ Mbps
- Packet loss: ___%

## Issues Found
1. [Issue description]
2. ...

## Logs
[Attach logs or link to log files]
```

---

## Support & Troubleshooting

See `docs/P2P_TEST_GUIDE.md` for detailed troubleshooting steps.

Common issues:
- Firewall blocking port 8443
- TAP driver not loaded
- Incorrect IP configuration
- TLS certificate errors
