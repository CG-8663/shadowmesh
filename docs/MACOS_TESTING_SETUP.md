# macOS Testing Setup for ShadowMesh

**Status**: Epic 2 UDP P2P Testing on macOS
**Platform**: macOS (Darwin)
**Challenge**: TAP devices require kernel extension

---

## The Challenge

ShadowMesh uses TAP (Layer 2) devices for frame-level encryption. On macOS, TAP devices require a kernel extension that is not installed by default.

**Current Status**:
- ✅ Relay server running at `94.237.121.21:9545`
- ✅ Binaries built successfully
- ❌ No TAP devices available on macOS
- 🚧 TAP device creation required for daemon startup

---

## Testing Options

### Option 1: Install TunTap for macOS (Full Testing)

**What**: Install kernel extension for TAP/TUN devices
**Pros**: Full local testing with TAP devices
**Cons**: Requires system extension installation

**Steps**:
```bash
# Install via Homebrew
brew install --cask tuntap

# Reboot required after installation
sudo reboot

# Verify installation
ls -la /dev/tap*
```

**After Installation**:
```bash
# Run full test script
sudo ./scripts/test-udp-p2p.sh
```

---

### Option 2: Remote Linux Testing (Recommended)

**What**: Deploy to Linux VMs/servers for full UDP P2P testing
**Pros**: Native TAP support, real-world NAT environment
**Cons**: Requires two remote machines

**Deployment Targets**:
- **Option A**: Existing production endpoints (Belgium RPi + UK VPS)
- **Option B**: Local VMs (VirtualBox, VMware, Parallels with Linux)
- **Option C**: Cloud VMs (AWS, DigitalOcean, UpCloud)

**Quick Deploy to Remote Linux**:
```bash
# Build Linux binary
GOOS=linux GOARCH=amd64 go build -o bin/shadowmesh-daemon-linux cmd/shadowmesh-daemon/main.go

# Deploy to machine 1
scp bin/shadowmesh-daemon-linux user@machine1:/tmp/shadowmesh-daemon
scp configs/endpoint1-udp-test.yaml user@machine1:/tmp/config.yaml

# Deploy to machine 2
scp bin/shadowmesh-daemon-linux user@machine2:/tmp/shadowmesh-daemon
scp configs/endpoint2-udp-test.yaml user@machine2:/tmp/config.yaml

# SSH and run
ssh user@machine1
sudo /tmp/shadowmesh-daemon -config /tmp/config.yaml
```

---

### Option 3: Relay-Only Testing (Connection Logic)

**What**: Test relay fallback without TAP devices
**Pros**: Quick validation of connection flow
**Cons**: Can't test actual data transfer

**Steps**:

1. **Modify daemon to make TAP optional** (temporary for testing):

```go
// pkg/daemonmgr/manager.go line 123
// Phase 1: Initialize TAP device (make optional for testing)
if err := dm.initTAPDevice(); err != nil {
    log.Printf("⚠️  TAP device initialization failed (continuing in test mode): %v", err)
    // Don't return error, continue for connection testing
}
```

2. **Run connection test**:
```bash
go build -o bin/shadowmesh-daemon cmd/shadowmesh-daemon/main.go
./scripts/test-connection-only.sh
```

3. **Observe logs** for:
   - NAT detection attempt
   - UDP hole punching attempt
   - Relay fallback triggered
   - Connection established

---

## Recommended Approach

**For Epic 2 Testing**: Use existing production infrastructure

You already have:
- ✅ Relay server deployed at `94.237.121.21:9545`
- ✅ Belgium Raspberry Pi endpoint
- ✅ UK VPS endpoint

**Test Plan**:

1. **Update configurations on both endpoints**:
   ```bash
   # On Belgium RPi
   scp configs/endpoint1-udp-test.yaml rpi@belgium:/etc/shadowmesh/config.yaml

   # On UK VPS
   scp configs/endpoint2-udp-test.yaml vps@uk:/etc/shadowmesh/config.yaml
   ```

2. **Deploy new binaries with Epic 2 code**:
   ```bash
   # Build for ARM (Raspberry Pi)
   GOOS=linux GOARCH=arm64 go build -o bin/shadowmesh-daemon-arm64 cmd/shadowmesh-daemon/main.go
   scp bin/shadowmesh-daemon-arm64 rpi@belgium:/usr/local/bin/shadowmesh-daemon

   # Build for amd64 (VPS)
   GOOS=linux GOARCH=amd64 go build -o bin/shadowmesh-daemon-amd64 cmd/shadowmesh-daemon/main.go
   scp bin/shadowmesh-daemon-amd64 vps@uk:/usr/local/bin/shadowmesh-daemon
   ```

3. **Restart daemons and test**:
   ```bash
   # Belgium RPi
   ssh rpi@belgium
   sudo systemctl restart shadowmesh

   # UK VPS
   ssh vps@uk
   sudo systemctl restart shadowmesh
   ```

4. **Observe connection logs**:
   ```bash
   # On Belgium RPi
   ssh rpi@belgium
   sudo journalctl -u shadowmesh -f

   # Look for:
   # - "Attempting direct UDP P2P connection..."
   # - "UDP hole punching failed" or "Connection established"
   # - "Falling back to relay mode..." or "Direct UDP P2P connection established"
   ```

---

## Quick Reference

### Current Production Infrastructure

| Component | Location | Address | Status |
|-----------|----------|---------|--------|
| Relay Server | UpCloud DC | `94.237.121.21:9545` | ✅ Running |
| Endpoint 1 | Belgium RPi | TBD | 🚧 Available |
| Endpoint 2 | UK VPS | TBD | 🚧 Available |

### Test Scenarios by Platform

| Scenario | macOS | Linux | Cloud VMs |
|----------|-------|-------|-----------|
| Relay Fallback | ⚠️ Needs TunTap | ✅ Native | ✅ Native |
| Direct UDP P2P | ❌ Not possible (localhost) | ✅ Ideal | ✅ Ideal |
| Connection Logic | ⚠️ With code mod | ✅ Full test | ✅ Full test |

---

## Next Steps

**Immediate** (Choose one):

1. Install TunTap for macOS → Run local test
2. Deploy to production endpoints → Real-world test
3. Spin up Linux VMs → Controlled environment test

**After Testing**:

1. Document results in `docs/EPIC2_UDP_P2P_TESTING.md`
2. Update performance benchmarks
3. Commit test results
4. Move to Epic 3 (Smart Contract Registry)

---

## Support

**TAP Device Issues**:
- macOS: `brew install --cask tuntap` + reboot
- Linux: Built-in support, use `ip tuntap` commands
- Cloud VMs: Already supported on all major platforms

**Connection Testing**:
- Use `curl http://localhost:9090/status` to check daemon
- Check logs for "UDP hole punching" or "relay fallback"
- Verify with `ping` between tunnel IPs

**Remote Deployment**:
- See `docs/EPIC2_UDP_P2P_TESTING.md` for full deployment guide
- Use existing production scripts in `scripts/deploy/`

---

**Recommendation**: Deploy to existing Belgium RPi + UK VPS for real-world UDP P2P testing
