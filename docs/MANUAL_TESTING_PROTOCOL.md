# ShadowMesh Manual Testing Protocol

**Version:** 1.0
**Created:** 2025-11-16
**Purpose:** Comprehensive manual testing procedures for Epic 2 components and integrated system
**Owner:** Dana (QA Engineer)

---

## Overview

This protocol defines manual testing procedures to validate that ShadowMesh components actually work in real-world conditions, not just in unit tests.

**Key Principle:** Code that hasn't been executed manually is code that doesn't work.

**Scope:**
- Individual Epic 2 component testing
- Integrated system testing (daemon + CLI)
- Two-machine P2P tunnel validation
- User acceptance criteria

---

## Prerequisites

### Required Hardware
- **Two separate machines** (physical or VMs)
  - Option A: Two physical computers on different networks
  - Option B: Two VMs with bridged networking
  - Option C: Local machine + remote VPS

### Required Software
- Go 1.25+ installed
- Root/sudo access (for TAP device creation)
- Network connectivity between machines
- tcpdump or Wireshark (for traffic inspection)

### Build Requirements
```bash
# Clone repository
git clone https://github.com/yourusername/shadowmesh
cd shadowmesh

# Build binaries
go build -o bin/shadowmesh cmd/shadowmesh/main.go
go build -o bin/shadowmesh-daemon cmd/shadowmesh-daemon/main.go
go build -o bin/integration-test cmd/integration-test/main.go

# Verify builds
./bin/shadowmesh --version
./bin/shadowmesh-daemon --help
```

---

## Test Suite 1: Individual Component Testing

### Test 1.1: TAP Device Management

**Story:** 2-1 TAP Device Management
**Objective:** Verify TAP device can be created, configured, and destroyed

**Prerequisites:**
- Root/sudo access
- No existing `tap0` device

**Test Procedure:**

1. **Create TAP device:**
   ```bash
   sudo ./bin/integration-test --tap tap0 --local-ip 10.0.0.1 --role responder
   ```

2. **Verify device created:**
   ```bash
   ip addr show tap0
   # Expected: tap0 interface exists with IP 10.0.0.1/24
   ```

3. **Verify device is UP:**
   ```bash
   ip link show tap0
   # Expected: state UP
   ```

4. **Test frame capture (in another terminal):**
   ```bash
   sudo tcpdump -i tap0
   ```

5. **Send test traffic:**
   ```bash
   ping 10.0.0.1
   # Expected: Ping succeeds, tcpdump shows ICMP packets
   ```

6. **Cleanup (Ctrl+C integration-test):**
   ```bash
   ip addr show tap0
   # Expected: Device no longer exists
   ```

**Expected Outcomes:**
- ✅ TAP device created successfully
- ✅ IP address configured correctly
- ✅ Device status is UP
- ✅ Can capture packets
- ✅ Device removed cleanly on shutdown

**Common Failures:**
- ❌ Permission denied → Need root/sudo
- ❌ Device already exists → Run `sudo ip link delete tap0`
- ❌ IP conflict → Choose different subnet

---

### Test 1.2: Encryption Pipeline Performance

**Story:** 2-6 Frame Encryption Pipeline
**Objective:** Verify encryption pipeline achieves >10,000 fps throughput

**Test Procedure:**

1. **Run benchmark test:**
   ```bash
   cd pkg/crypto/frameencryption
   go test -bench=BenchmarkPipeline -benchtime=10s
   ```

2. **Verify throughput:**
   ```
   Expected output:
   BenchmarkPipeline-8    3457200    345720 fps
   ```

3. **Verify >10,000 fps:**
   ```
   ✅ Actual: 345,720 fps
   ✅ Target: 10,000 fps
   ✅ Performance: 34x faster than requirement
   ```

**Expected Outcomes:**
- ✅ Pipeline processes >10,000 frames/second
- ✅ No memory leaks during sustained operation
- ✅ Encryption/decryption round-trip succeeds

---

### Test 1.3: NAT Type Detection

**Story:** 2-4 NAT Type Detection
**Objective:** Verify NAT detection works against real STUN server

**Test Procedure:**

1. **Run NAT detection test:**
   ```bash
   cd pkg/nat
   go test -v -run TestNATDetection
   ```

2. **Check detection results:**
   ```
   Expected output:
   === RUN TestNATDetection
   NAT Type: [NoNAT|FullCone|RestrictedCone|PortRestrictedCone|Symmetric]
   Public IP: [actual public IP]
   P2P Feasible: [true|false]
   Detection Time: [<2s]
   --- PASS: TestNATDetection
   ```

3. **Verify detection speed:**
   ```
   ✅ Detection completes in <2 seconds
   ✅ Returns valid NAT type
   ✅ Public IP matches actual IP
   ```

**Expected Outcomes:**
- ✅ Detects NAT type correctly
- ✅ Completes in <2 seconds
- ✅ Returns P2P feasibility assessment

**Test Variation (Behind NAT):**
- Run same test from machine behind router/NAT
- Expected: Symmetric or Cone NAT type detected

---

### Test 1.4: CLI Commands

**Story:** 2-7 CLI Commands
**Objective:** Verify CLI commands communicate with daemon

**Prerequisites:**
- Daemon running: `sudo ./bin/shadowmesh-daemon`

**Test Procedure:**

1. **Test status command (disconnected):**
   ```bash
   ./bin/shadowmesh status
   ```
   Expected output:
   ```
   Connection: Disconnected
   ```

2. **Test connect command:**
   ```bash
   ./bin/shadowmesh connect peer.example.com:9001
   ```
   Expected output:
   ```
   Connecting to peer.example.com:9001...
   ```

3. **Test status command (connecting/connected):**
   ```bash
   ./bin/shadowmesh status
   ```
   Expected output:
   ```
   Connection: Connected (or Connecting)
   Peer ID: peer.example.com:9001
   Uptime: 0h 0m 15s
   ```

4. **Test disconnect command:**
   ```bash
   ./bin/shadowmesh disconnect
   ```
   Expected output:
   ```
   Disconnected from peer.example.com:9001
   ```

**Expected Outcomes:**
- ✅ CLI communicates with daemon (HTTP API works)
- ✅ Status reflects actual connection state
- ✅ Commands execute successfully

---

## Test Suite 2: Integrated System Testing

### Test 2.1: Daemon Startup and Initialization

**Objective:** Verify daemon starts and initializes all components

**Test Procedure:**

1. **Start daemon with verbose logging:**
   ```bash
   sudo ./bin/shadowmesh-daemon --config daemon.yaml --verbose
   ```

2. **Verify component initialization:**
   Expected log output:
   ```
   [INFO] Loading configuration from daemon.yaml
   [INFO] Starting ShadowMesh daemon...
   [INFO] Initializing TAP device: tap0
   [INFO] TAP device configured: 10.0.0.1/24
   [INFO] Encryption pipeline started
   [INFO] HTTP API server listening on 127.0.0.1:9090
   [INFO] Daemon ready
   ```

3. **Verify daemon health:**
   ```bash
   curl http://127.0.0.1:9090/health
   ```
   Expected response:
   ```json
   {"status":"healthy","time":"2025-11-16T..."}
   ```

4. **Check daemon status:**
   ```bash
   ./bin/shadowmesh status
   ```
   Expected:
   ```
   Connection: Disconnected
   Daemon: Running
   ```

**Expected Outcomes:**
- ✅ Daemon starts without errors
- ✅ All components initialize successfully
- ✅ HTTP API responds to requests
- ✅ TAP device created automatically

**Common Failures:**
- ❌ Port 9090 already in use → Change listen_address in config
- ❌ TAP device creation fails → Need root/sudo
- ❌ Config file not found → Verify path

---

### Test 2.2: Graceful Shutdown

**Objective:** Verify daemon cleans up resources on shutdown

**Test Procedure:**

1. **Start daemon:**
   ```bash
   sudo ./bin/shadowmesh-daemon --config daemon.yaml
   ```

2. **Verify TAP device exists:**
   ```bash
   ip addr show tap0
   # Expected: Device exists
   ```

3. **Send shutdown signal:**
   ```bash
   # In daemon terminal: Ctrl+C
   ```

4. **Verify cleanup:**
   Expected log output:
   ```
   [INFO] Shutdown signal received
   [INFO] Stopping encryption pipeline...
   [INFO] Encryption pipeline stopped
   [INFO] Closing TAP device...
   [INFO] TAP device closed
   [INFO] HTTP API server stopped
   [INFO] Shutdown complete
   ```

5. **Verify TAP device removed:**
   ```bash
   ip addr show tap0
   # Expected: Device does not exist (error message)
   ```

**Expected Outcomes:**
- ✅ Daemon shuts down gracefully
- ✅ TAP device removed
- ✅ No zombie processes
- ✅ Clean log messages

---

## Test Suite 3: Two-Machine P2P Tunnel Validation

### Test 3.1: Same-LAN P2P Connection (No NAT)

**Objective:** Validate P2P tunnel between two machines on same network

**Setup:**
- **Machine A (Initiator):** 192.168.1.100
- **Machine B (Responder):** 192.168.1.101
- Both on same LAN subnet

**Machine B (Responder) Setup:**

1. **Create config file (`daemon.yaml`):**
   ```yaml
   daemon:
     listen_address: "127.0.0.1:9090"
     log_level: "info"

   network:
     tap_device: "tap0"
     local_ip: "10.0.0.2/24"

   encryption:
     key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

   peer:
     address: ""  # Responder listens, doesn't connect

   websocket:
     listen: "0.0.0.0:9001"  # Listen for incoming connections
   ```

2. **Start daemon:**
   ```bash
   sudo ./bin/shadowmesh-daemon --config daemon.yaml
   ```

3. **Verify listening:**
   ```bash
   ss -tuln | grep 9001
   # Expected: LISTEN on 0.0.0.0:9001
   ```

**Machine A (Initiator) Setup:**

1. **Create config file (`daemon.yaml`):**
   ```yaml
   daemon:
     listen_address: "127.0.0.1:9090"
     log_level: "info"

   network:
     tap_device: "tap0"
     local_ip: "10.0.0.1/24"

   encryption:
     key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"  # SAME KEY

   peer:
     address: "192.168.1.101:9001"  # Machine B address
   ```

2. **Start daemon:**
   ```bash
   sudo ./bin/shadowmesh-daemon --config daemon.yaml
   ```

3. **Initiate connection:**
   ```bash
   ./bin/shadowmesh connect 192.168.1.101:9001
   ```

**Validation Steps:**

1. **Verify connection established:**
   ```bash
   # Machine A
   ./bin/shadowmesh status
   ```
   Expected:
   ```
   Connection: Connected
   Peer: 192.168.1.101:9001
   Uptime: 0h 0m 30s
   ```

   ```bash
   # Machine B
   ./bin/shadowmesh status
   ```
   Expected:
   ```
   Connection: Connected
   Peer: 192.168.1.100:[random port]
   Uptime: 0h 0m 30s
   ```

2. **Test encrypted tunnel with ping:**
   ```bash
   # Machine A
   ping 10.0.0.2
   ```
   Expected:
   ```
   PING 10.0.0.2: 56 data bytes
   64 bytes from 10.0.0.2: icmp_seq=0 ttl=64 time=15 ms
   64 bytes from 10.0.0.2: icmp_seq=1 ttl=64 time=12 ms
   ```

   ```bash
   # Machine B
   ping 10.0.0.1
   ```
   Expected:
   ```
   PING 10.0.0.1: 56 data bytes
   64 bytes from 10.0.0.1: icmp_seq=0 ttl=64 time=14 ms
   ```

3. **Verify traffic is encrypted (packet capture):**
   ```bash
   # Machine A (in another terminal)
   sudo tcpdump -i eth0 port 9001 -X
   ```
   Expected: Encrypted WebSocket frames (binary data, not plaintext ICMP)

4. **Verify traffic on TAP device (plaintext):**
   ```bash
   # Machine A
   sudo tcpdump -i tap0
   ```
   Expected: Plaintext ICMP packets

5. **Measure throughput:**
   ```bash
   # Machine B - Start iperf server
   iperf3 -s -B 10.0.0.2

   # Machine A - Run iperf client
   iperf3 -c 10.0.0.2 -t 30
   ```
   Expected: >10 Mbps throughput through encrypted tunnel

**Expected Outcomes:**
- ✅ Connection establishes successfully
- ✅ Ping works through tunnel (10.0.0.1 ↔ 10.0.0.2)
- ✅ Traffic encrypted on wire (eth0)
- ✅ Traffic plaintext on TAP device
- ✅ Throughput acceptable (>10 Mbps)
- ✅ Latency acceptable (<50ms added overhead)

---

### Test 3.2: Cross-Network P2P Connection (With NAT)

**Objective:** Validate P2P tunnel between machines on different networks

**Setup:**
- **Machine A:** Behind home NAT (192.168.1.100)
- **Machine B:** Public VPS (203.0.113.42)

**Follow same procedure as Test 3.1, but:**
- Machine B (VPS) uses public IP in config
- Machine A connects to VPS public IP
- NAT detection runs on Machine A (should detect NAT type)
- UDP hole punching may or may not succeed (depends on NAT type)
- WebSocket connection should establish (fallback)

**Additional Validation:**
- Check NAT detection results in daemon logs
- Verify hole punching attempt (success or fallback)
- Confirm connection works despite NAT

---

### Test 3.3: Connection Recovery

**Objective:** Verify tunnel recovers from network interruption

**Test Procedure:**

1. **Establish connection** (per Test 3.1)
2. **Start continuous ping:**
   ```bash
   ping 10.0.0.2
   ```

3. **Simulate network interruption:**
   ```bash
   # Machine A - Temporarily disable network
   sudo ifconfig eth0 down
   sleep 10
   sudo ifconfig eth0 up
   ```

4. **Verify reconnection:**
   - Ping should resume after network recovery
   - Daemon logs should show reconnection attempt
   - Connection status should return to "Connected"

**Expected Outcomes:**
- ✅ Tunnel detects disconnection
- ✅ Automatic reconnection attempt
- ✅ Tunnel resumes operation
- ✅ Minimal data loss

---

## Test Suite 4: Failure Scenario Testing

### Test 4.1: Invalid Encryption Key

**Objective:** Verify daemon rejects frames with wrong encryption key

**Test Procedure:**

1. **Machine A:** Use key `aaaa...`
2. **Machine B:** Use key `bbbb...` (different)
3. **Attempt connection**
4. **Verify connection fails or frames dropped**

**Expected Outcomes:**
- ✅ Daemon detects authentication tag mismatch
- ✅ Frames dropped with error log
- ✅ No plaintext traffic leaks

---

### Test 4.2: TAP Device Failure

**Objective:** Verify daemon handles TAP device errors gracefully

**Test Procedure:**

1. **Start daemon**
2. **Manually delete TAP device:**
   ```bash
   sudo ip link delete tap0
   ```
3. **Verify daemon response:**
   - Daemon logs error
   - Connection transitions to error state
   - Daemon attempts recovery or shuts down cleanly

**Expected Outcomes:**
- ✅ Daemon doesn't crash
- ✅ Error logged clearly
- ✅ Graceful degradation

---

## Test Suite 5: User Acceptance Testing

### Test 5.1: Project Lead (james) Validation

**Objective:** Project lead validates complete P2P tunnel between two real locations

**Procedure:**

1. **james selects two test machines:**
   - Example: Personal laptop + VPS
   - Example: Home computer + Office computer
   - Example: Two cloud VMs in different regions

2. **james follows Test 3.1 or 3.2 procedure**

3. **james validates:**
   - Can establish connection between machines
   - Ping works through encrypted tunnel
   - Traffic visibly encrypted on wire
   - Performance acceptable for use case
   - Setup procedure clear and documented

4. **james provides sign-off:**
   - ✅ Epic 2 validated and working
   - OR
   - ❌ Issues found (document for fixing)

**Acceptance Criteria:**
- james successfully establishes P2P tunnel
- james can ping between machines through tunnel
- james confirms traffic is encrypted
- james signs off on Epic 2 completion

---

## Test Results Documentation

### Test Execution Record

For each test executed, document:

```markdown
**Test:** [Test ID and Name]
**Date:** [YYYY-MM-DD]
**Tester:** [Name]
**Machines:** [Machine A info, Machine B info]
**Result:** [PASS / FAIL / BLOCKED]

**Observations:**
- [What worked]
- [What didn't work]
- [Performance metrics]
- [Issues discovered]

**Evidence:**
- [Log snippets]
- [Screenshots]
- [Packet captures]

**Follow-up Actions:**
- [Bugs filed]
- [Documentation updates]
```

### Test Summary Report

After completing all tests:

```markdown
## Epic 2 Manual Testing Summary

**Total Tests:** [count]
**Passed:** [count]
**Failed:** [count]
**Blocked:** [count]

**Critical Findings:**
1. [Finding 1]
2. [Finding 2]

**Recommendation:**
[Ready for release / Needs fixes / Major rework required]

**Sign-off:**
- QA Engineer: [Name] ✅
- Project Lead: [Name] ✅
```

---

## Troubleshooting Guide

### Issue: TAP device creation fails

**Symptom:** "Permission denied" or "Operation not permitted"

**Solution:**
```bash
# Ensure running with sudo/root
sudo ./bin/shadowmesh-daemon

# Verify TUN/TAP kernel module loaded
lsmod | grep tun
# If not loaded:
sudo modprobe tun
```

---

### Issue: Connection timeout

**Symptom:** "Connection to peer failed: timeout"

**Possible Causes:**
1. Firewall blocking port 9001
2. Wrong peer address
3. Peer daemon not running

**Solution:**
```bash
# Check firewall
sudo iptables -L | grep 9001
sudo ufw allow 9001

# Verify peer daemon running
ssh peer-machine
ps aux | grep shadowmesh-daemon

# Test connectivity
telnet peer-address 9001
```

---

### Issue: Ping works but no application traffic

**Symptom:** Ping succeeds through tunnel, but other apps don't work

**Possible Cause:** Routing or MTU issues

**Solution:**
```bash
# Check routing table
ip route

# Add route if needed
sudo ip route add 10.0.0.0/24 dev tap0

# Check/adjust MTU
sudo ip link set tap0 mtu 1400
```

---

## References

- [Story 2-8: Direct P2P Integration Test](../.bmad-ephemeral/stories/2-8-direct-p2p-integration-test.md)
- [Daemon Architecture](./DAEMON_ARCHITECTURE.md)
- [Epic 2 Retrospective](../.bmad-ephemeral/retrospectives/epic-2-retro-partial-2025-11-16.md)

---

**Protocol Version:** 1.0
**Last Updated:** 2025-11-16
**Next Review:** After Story 2-8 completion
