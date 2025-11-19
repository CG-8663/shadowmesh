# Epic 2 - UDP P2P Testing Guide

**Status**: Implementation Complete, Testing in Progress
**Date**: 2025-11-19
**Epic**: Direct UDP P2P with Relay Fallback

---

## Overview

Epic 2 implements direct UDP peer-to-peer connections with automatic fallback to relay server when UDP hole punching fails. This document describes testing procedures and expected results.

---

## Implementation Summary

### Connection Strategy

```
1. Check NAT compatibility (IsP2PFeasible)
2. Attempt UDP hole punching (500ms timeout)
3. If successful → Direct UDP P2P connection
4. If failed → Fallback to relay WebSocket
5. If no relay → Fallback to direct WebSocket
```

### Transport Modes

- **UDP P2P**: Low latency, direct peer-to-peer (target: <5ms overhead)
- **WebSocket Relay**: High reliability, works with all NAT types
- **WebSocket Direct**: Fallback for testing without relay

---

## Test Scenarios

### Scenario 1: Relay Fallback (Localhost Testing) ✅

**What**: Test that relay fallback works when UDP P2P is not possible
**Why**: UDP hole punching won't work on localhost (loopback limitation)
**Expected**: System should fallback to relay WebSocket automatically

**Test Procedure**:

1. Start two daemons on localhost with NAT enabled
2. Connect endpoint 1 to endpoint 2
3. Observe logs showing:
   - NAT detection attempt
   - UDP hole punching attempt
   - Timeout after 500ms
   - Automatic fallback to relay
   - Successful relay connection

**Commands**:
```bash
# Terminal 1: Start endpoint 1
sudo ./bin/shadowmesh-daemon -config configs/endpoint1-udp-test.yaml

# Terminal 2: Start endpoint 2
sudo ./bin/shadowmesh-daemon -config configs/endpoint2-udp-test.yaml

# Terminal 3: Connect endpoint 1
curl -X POST http://localhost:9090/connect

# Terminal 4: Test ping
ping -c 3 10.10.10.4
```

**Success Criteria**:
- ✅ NAT detection completes
- ✅ UDP hole punching attempted
- ✅ 500ms timeout triggered
- ✅ Relay connection established
- ✅ Ping works between endpoints

---

### Scenario 2: Direct UDP P2P (Requires Two Machines) 🚧

**What**: Test UDP hole punching between two real endpoints
**Why**: Validate sub-5ms latency goal with direct UDP
**Expected**: UDP connection succeeds without relay

**Requirements**:
- Two separate machines (VMs or physical)
- Compatible NAT types (Full Cone or Restricted Cone)
- Public IP addresses or NAT gateway

**Test Procedure**:

1. Deploy endpoint 1 on Machine A
2. Deploy endpoint 2 on Machine B
3. Detect NAT types on both machines
4. Attempt connection
5. Measure latency with ping/iperf3

**Commands** (Machine A):
```bash
# Build binary
go build -o shadowmesh-daemon cmd/shadowmesh-daemon/main.go

# Deploy to remote machine
scp shadowmesh-daemon user@machine-a:/tmp/
scp configs/endpoint1-udp-test.yaml user@machine-a:/tmp/

# SSH to machine A
ssh user@machine-a
sudo /tmp/shadowmesh-daemon -config /tmp/endpoint1-udp-test.yaml
```

**Commands** (Machine B):
```bash
# Similar deployment to machine B
scp shadowmesh-daemon user@machine-b:/tmp/
scp configs/endpoint2-udp-test.yaml user@machine-b:/tmp/

ssh user@machine-b
sudo /tmp/shadowmesh-daemon -config /tmp/endpoint2-udp-test.yaml
```

**Success Criteria**:
- ✅ NAT detection shows compatible type
- ✅ UDP hole punching succeeds within 500ms
- ✅ Direct UDP connection established
- ✅ Ping latency <5ms overhead vs direct connection
- ✅ iperf3 throughput comparable to relay mode (30+ Mbps)

---

### Scenario 3: Symmetric NAT (Force Relay) ✅

**What**: Test that Symmetric NAT correctly skips UDP and uses relay
**Why**: Symmetric NAT cannot do hole punching
**Expected**: System should immediately use relay, skipping UDP attempt

**Test Procedure**:

1. Modify NAT detector to simulate Symmetric NAT
2. Start two daemons
3. Connect and observe relay is used immediately

**Code Modification** (temporary, for testing):
```go
// pkg/nat/detector.go
func (d *NATDetector) IsP2PFeasible() bool {
    return false  // Force Symmetric NAT behavior
}
```

**Success Criteria**:
- ✅ NAT detection shows Symmetric type
- ✅ UDP hole punching skipped
- ✅ Relay connection used immediately
- ✅ Ping works through relay

---

## Testing Configuration

### Endpoint 1 Configuration

**File**: `configs/endpoint1-udp-test.yaml`

```yaml
daemon:
  listen_address: "127.0.0.1:9090"
  log_level: "info"

network:
  tap_device: "tap0"
  local_ip: "10.10.10.3/24"

encryption:
  key: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

peer:
  address: ""
  id: "endpoint1"

nat:
  enabled: true
  stun_server: "stun.l.google.com:19302"

relay:
  enabled: true
  server: "ws://94.237.121.21:9545"
```

### Endpoint 2 Configuration

**File**: `configs/endpoint2-udp-test.yaml`

```yaml
daemon:
  listen_address: "127.0.0.1:9091"  # Different port
  log_level: "info"

network:
  tap_device: "tap1"  # Different TAP device
  local_ip: "10.10.10.4/24"  # Different IP

encryption:
  key: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"  # SAME key

peer:
  address: ""
  id: "endpoint2"  # Different peer ID

nat:
  enabled: true
  stun_server: "stun.l.google.com:19302"

relay:
  enabled: true
  server: "ws://94.237.121.21:9545"
```

---

## Expected Log Output

### Successful UDP P2P Connection

```
[INFO] Attempting direct UDP P2P connection to 192.168.1.100:9545...
[INFO] NAT type is compatible with direct P2P, attempting UDP hole punching...
[INFO] HolePunch: Connection established to 192.168.1.100:9545
[INFO] ✅ Direct UDP P2P connection established to 192.168.1.100:9545
[INFO] ✅ All daemon components initialized successfully
```

### Relay Fallback (UDP Failed)

```
[INFO] Attempting direct UDP P2P connection to 127.0.0.1:9545...
[INFO] NAT type is compatible with direct P2P, attempting UDP hole punching...
[WARN] ⚠️  UDP hole punching failed: hole punch timeout after 500ms - fallback to relay
[INFO] Falling back to relay mode...
[INFO] Connecting via relay server: ws://94.237.121.21:9545/relay (peer ID: endpoint1)
[INFO] ✅ Connected to relay server as peer endpoint1
[INFO] ✅ All daemon components initialized successfully
```

### Symmetric NAT (Immediate Relay)

```
[INFO] Attempting direct UDP P2P connection to 192.168.1.100:9545...
[INFO] NAT type not compatible with direct P2P (Symmetric NAT detected)
[INFO] Falling back to relay mode...
[INFO] Connecting via relay server: ws://94.237.121.21:9545/relay (peer ID: endpoint1)
[INFO] ✅ Connected to relay server successfully
```

---

## Performance Benchmarks

### Target Metrics

| Metric | UDP P2P | Relay | Notes |
|--------|---------|-------|-------|
| **Latency Overhead** | <5ms | ~55ms | UDP is 10x lower |
| **Throughput** | 30+ Mbps | 30+ Mbps | Both should match |
| **Connection Time** | <500ms | <2s | UDP attempts first |
| **NAT Success Rate** | 80% Full Cone | 100% | Relay is fallback |

### Measurement Commands

```bash
# Latency test
ping -c 100 10.10.10.4

# Throughput test (endpoint 2 as server)
iperf3 -s -B 10.10.10.4

# Throughput test (endpoint 1 as client)
iperf3 -c 10.10.10.4 -B 10.10.10.3 -t 30 -P 4
```

---

## Known Limitations

### Localhost Testing

❌ **UDP hole punching will NOT work on localhost**

**Reason**: Loopback interface (127.0.0.1) doesn't use actual NAT traversal
**Workaround**: Test relay fallback instead
**Real Testing**: Use two separate machines/VMs

### NAT Type Detection

⚠️ **STUN may fail behind restrictive firewalls**

**Symptoms**: NAT detection timeout
**Workaround**: Use relay mode only (disable NAT in config)
**Fix**: Configure firewall to allow UDP to STUN server

---

## Test Checklist

### Localhost Testing (Relay Fallback)

- [ ] TAP devices created (tap0, tap1)
- [ ] IP addresses assigned (10.10.10.3, 10.10.10.4)
- [ ] Endpoint 1 daemon started successfully
- [ ] Endpoint 2 daemon started successfully
- [ ] NAT detection attempted
- [ ] UDP hole punching attempted
- [ ] Relay fallback triggered after 500ms timeout
- [ ] Ping works between endpoints
- [ ] iperf3 shows 30+ Mbps throughput

### Two-Machine Testing (Real UDP P2P)

- [ ] Two machines deployed (different networks)
- [ ] NAT types detected (both compatible)
- [ ] UDP hole punching succeeded
- [ ] Direct UDP connection established
- [ ] Ping latency <5ms overhead
- [ ] iperf3 throughput 30+ Mbps
- [ ] No packet loss
- [ ] Connection stable for 5+ minutes

### Symmetric NAT Testing

- [ ] NAT detector modified to return Symmetric
- [ ] UDP hole punching skipped
- [ ] Relay used immediately
- [ ] Connection successful
- [ ] Performance same as relay mode

---

## Troubleshooting

### Issue: "tap0: Operation not permitted"

**Cause**: Not running as root
**Fix**: Use `sudo` to run daemon

### Issue: "UDP hole punching failed immediately"

**Cause**: STUN server unreachable or NAT incompatible
**Fix**: Check NAT type with `stun_server` configuration

### Issue: "Relay connection failed"

**Cause**: Relay server down or unreachable
**Fix**: Check relay server at http://94.237.121.21:9545/health

### Issue: "Ping works but iperf3 fails"

**Cause**: Frame encryption or buffer issues
**Fix**: Check daemon logs for errors, verify encryption key matches

---

## Next Steps

1. ✅ Complete localhost relay fallback testing
2. 🚧 Deploy to two remote machines for real UDP P2P test
3. 📊 Measure performance: UDP vs Relay latency/throughput
4. 📝 Document results in this file
5. 🎯 Validate <5ms latency goal for UDP P2P

---

## Test Results

### Test Run 1: Production Deployment - Relay Fallback

**Date**: 2025-11-19
**Environment**: Production endpoints (UK VPS + Belgium RPi5)
**Result**: ✅ SUCCESS

**Configuration**:
- Endpoint 1: shadowmesh-001 (UK VPS, Intel AMD64) - 100.115.193.115
- Endpoint 2: shadowmesh-002 (Belgium RPi5, ARM64) - 100.90.48.10
- Relay Server: ws://94.237.121.21:9545
- TAP Devices: tap0 (10.10.10.3/24), tap1 (10.10.10.4/24)

**NAT Detection Results**:
- Both endpoints detected as **Symmetric NAT**
- P2P Feasibility: false (as expected)
- System correctly skipped UDP hole punching
- Automatic fallback to relay mode triggered

**Connection Results**:
- Both daemons connected to relay server successfully
- Connection established: ✅
- Connection mode: WebSocket relay (UDP P2P not feasible)

**Ping Latency Test**:
```
Endpoint 1 → Endpoint 2 (UK → Belgium):
- Average: 47.1ms
- Range: 44.5ms - 51.3ms
- Packet loss: 0%

Endpoint 2 → Endpoint 1 (Belgium → UK):
- Average: 59.7ms
- Range: 47.7ms - 102ms
- Packet loss: 0%
```

**iperf3 Throughput Test** (30 seconds, 4 parallel streams):
```
Aggregate Throughput:
- Sender: 14.2 Mbps
- Receiver: 13.1 Mbps
- Retransmissions: 0 (no packet loss)

Per-Stream Performance:
- Stream 1: 4.05 Mbps
- Stream 2: 3.88 Mbps
- Stream 3: 3.36 Mbps
- Stream 4: 2.94 Mbps
```

**Analysis**:
- ✅ NAT detection working correctly (Symmetric NAT identified)
- ✅ Relay fallback triggered as designed
- ✅ Connection establishment successful
- ✅ Stable connectivity (0% packet loss)
- ⚠️ Throughput (14 Mbps) below 30 Mbps target
  - Likely bottleneck: RPi5 CPU on ChaCha20-Poly1305 encryption
  - Future optimization: Hardware acceleration, buffer tuning

**Conclusion**:
Epic 2 relay fallback functionality validated successfully on production infrastructure. System correctly detects incompatible NAT types and seamlessly falls back to relay mode with stable, encrypted connectivity.

---

### Test Run 2: Real UDP P2P (Two Machines)

**Date**: Not yet tested
**Environment**: Requires endpoints with compatible NAT types (Full Cone or Restricted Cone)
**Result**: PENDING

**Notes**: Current production endpoints both have Symmetric NAT, which prevents UDP hole punching. To test direct UDP P2P, would need endpoints with compatible NAT configurations or cloud VMs with Full Cone NAT.

---

**Epic 2 Status**: Implementation Complete, Relay Fallback Validated ✅
**Next Milestone**: Epic 3 - Smart Contract Relay Registry
