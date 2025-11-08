# ShadowMesh Layer 2 Testing - Current Status

**Date**: November 4, 2025
**Status**: ⚠️ Partial deployment - Awaiting Belgium server recovery

---

## What Happened

### Issue Discovered: chr001 Device Conflict

When attempting to run ShadowMesh Layer 2 with `-tap chr001`, encountered error:
```
Failed to attach to TAP device chr001: device or resource busy
```

**Root Cause**: Tailscale daemon owns the chr001 TAP device with exclusive access. Cannot attach to it while Tailscale is running.

### Solution Implemented

Modified `cmd/lightnode-l2/main.go` to support **creating NEW TAP devices** instead of attaching to existing ones:

**New Flags Added**:
- `-tap-ip <IP>` - IP address for TAP device (e.g., 10.10.10.3)
- `-tap-netmask <mask>` - Netmask for TAP device (default: 24)

**Behavior**:
- If `-tap-ip` provided: Creates NEW TAP device (tap0, tap1, etc.) with specified IP
- If `-tap <name>` only: Attempts to attach to existing device (original behavior)
- This allows Tailscale (chr001) and ShadowMesh (tap0) to coexist

### Deployment Status

| Server | Binary | Status |
|--------|--------|--------|
| Belgium (100.90.48.10) | ARM64 | ⚠️ **Unreachable** - Tailscale stopped, lost connectivity |
| UK (100.115.193.115) | x86_64 | ✅ **Deployed** - Updated binary with TAP IP support |

---

## Current Architecture

### New Approach: Parallel TAP Devices

```
Belgium Node (shadowmesh-002)          UK Node (shadowmesh-001)
┌─────────────────────────┐           ┌─────────────────────────┐
│ Tailscale (Management)  │           │ Tailscale (Management)  │
│   chr001: 10.10.10.4    │           │   chr001: 10.10.10.3    │
│   (owned by tailscaled) │           │   (owned by tailscaled) │
└─────────────────────────┘           └─────────────────────────┘
            │                                     │
┌─────────────────────────┐           ┌─────────────────────────┐
│ ShadowMesh Layer 2      │           │ ShadowMesh Layer 2      │
│   tap0: 10.10.10.4      │◄─────────►│   tap0: 10.10.10.3      │
│   (new device)          │  P2P Mesh  │   (new device)          │
└─────────────────────────┘           └─────────────────────────┘
```

**Key Points**:
- Tailscale keeps chr001 for SSH/management (100.x.x.x IPs)
- ShadowMesh creates tap0/tap1 for Layer 2 VPN (10.10.10.x IPs)
- Both systems coexist independently
- Proves ShadowMesh works without relying on Tailscale for data plane

---

## Next Steps

### 1. Restore Belgium Server Access

**Option A**: Reboot server via console/IPMI
```bash
# Access server console and run:
sudo systemctl start tailscaled
```

**Option B**: Access via public IP (if available)
```bash
# If Belgium server has public IP, use that instead
ssh pxcghost@<public_ip>
sudo systemctl start tailscaled
```

### 2. Deploy Updated Binary to Belgium

Once Belgium is accessible:
```bash
# From local machine:
scp build/shadowmesh-l2-linux-arm64 pxcghost@100.90.48.10:~/shadowmesh/shadowmesh-l2

# On Belgium server:
ssh pxcghost@100.90.48.10
chmod +x ~/shadowmesh/shadowmesh-l2
```

### 3. Run Layer 2 Test

**Belgium (Receiver)**:
```bash
cd ~/shadowmesh
sudo ./shadowmesh-l2 \
  -keydir ./keys \
  -backbone http://209.151.148.121:8080 \
  -ip 100.90.48.10 \
  -port 9443 \
  -tap-ip 10.10.10.4 \
  -tap-netmask 24 \
  -public
```

**UK (Sender)**:
```bash
cd ~/shadowmesh
sudo ./shadowmesh-l2 \
  -keydir ./keys \
  -backbone http://209.151.148.121:8080 \
  -ip 100.115.193.115 \
  -port 9443 \
  -tap-ip 10.10.10.3 \
  -tap-netmask 24 \
  -connect fb9f1ad65f7b67bfc35cd285ab28898e0d486afe
```

### 4. Test Connectivity

```bash
# From UK server:
ping -c 10 10.10.10.4
```

**Expected Result**:
- ✅ ICMP echo request/reply through ShadowMesh tunnel
- ✅ Frame logs: `TAP→P2P: EthernetFrame[...]` and `P2P→TAP: EthernetFrame[...]`
- ✅ 0% packet loss
- ✅ <10ms latency

### 5. Video Streaming Test

Once ping works:
```bash
# Belgium:
iperf3 -s -B 10.10.10.4

# UK:
iperf3 -c 10.10.10.4 -t 30
```

**Expected**: 50-100+ Mbps throughput

---

## Technical Details

### Code Changes

**File**: `cmd/lightnode-l2/main.go`

**Lines 25-28** - New flags:
```go
tapDevice := flag.String("tap", "", "TAP device name (leave empty to auto-create)")
tapIP := flag.String("tap-ip", "", "TAP device IP address (e.g., 10.10.10.3)")
tapNetmask := flag.String("tap-netmask", "24", "TAP device netmask (default: 24)")
ipAddress := flag.String("ip", "", "External IP address for P2P (required)")
```

**Lines 99-120** - TAP device creation logic:
```go
var tapInterface *layer2.TAPInterface
if *tapIP != "" {
	// Create new TAP device with IP configuration
	tapInterface, err = layer2.NewTAPInterface(*tapDevice, *tapIP, *tapNetmask)
} else if *tapDevice != "" {
	// Attach to existing TAP device
	tapInterface, err = layer2.AttachToExisting(*tapDevice)
} else {
	log.Fatalf("Error: Either -tap-ip or -tap must be specified")
}
```

### Why This Approach Works

1. **Independence**: ShadowMesh creates its own TAP devices, doesn't interfere with Tailscale
2. **Flexibility**: Supports both new device creation and existing device attachment
3. **Simplicity**: User just specifies IP address, device is auto-configured
4. **Coexistence**: Tailscale for management, ShadowMesh for VPN data plane

---

## Success Criteria

- [ ] Belgium server accessible via Tailscale
- [ ] Updated binary deployed to Belgium
- [ ] Both nodes create tap0 devices successfully
- [ ] Ping works: `ping 10.10.10.4` from UK succeeds
- [ ] Frame logs visible: `TAP→P2P` and `P2P→TAP`
- [ ] 0% packet loss
- [ ] Video/iperf3 test passes
- [ ] Proves Layer 2 VPN works independently of Tailscale

---

## Files Modified

- `cmd/lightnode-l2/main.go` - Added TAP IP configuration flags and logic
- `build/shadowmesh-l2-linux-amd64` - UK binary (8.9MB)
- `build/shadowmesh-l2-linux-arm64` - Belgium binary (8.3MB)
- `QUICK_TEST.md` - Updated with new command syntax
- `LAYER2_STATUS.md` - This file

---

## References

- `QUICK_TEST.md` - Quick start guide with new commands
- `LAYER2_TEST_GUIDE.md` - Detailed testing procedures
- `pkg/layer2/tap.go` - TAP device implementation
- `pkg/p2p/connection.go` - P2P connection management
