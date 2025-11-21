# VM 111 (Chronara API) Installation

## Quick Install

Run this command on VM 111:

```bash
SHADOWMESH_AUTO_INSTALL=1 curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/install.sh | sudo bash
```

## Expected Results

- **Auto-detected**: Chronara API Client
- **Mesh IP**: 10.10.10.5/24
- **TAP Device**: chr001
- **Relay**: 94.237.121.21:9545

## Verification

After installation:

```bash
# Check daemon status
sudo systemctl status shadowmesh-daemon

# Check TAP interface
ip addr show chr001

# Check relay connection
sudo journalctl -u shadowmesh-daemon -n 20 | grep relay

# Test connectivity to other nodes
ping 10.10.10.3  # UK
ping 10.10.10.4  # Belgium
ping 10.10.10.6  # Philippines
ping 10.10.10.7  # Starlink
```

## Chronara API Configuration

Once shadowmesh is running, the Chronara API will be accessible at:

### Internal Mesh Access (Secure)
```
http://10.10.10.5:8080/api/v1/attestation/verified-nodes
```

### External Access (via Cloudflare - to be secured with Web3 auth)
```
https://api.chronara.net/api/v1/attestation/verified-nodes
```

## Next: Configure Chronara API Binding

Update Chronara API to listen on mesh interface:

```go
// In cmd/chronara-mock/main.go or similar
// Change from:
// http.ListenAndServe(":8080", router)

// To dual-bind:
// Listen on both 0.0.0.0 (Cloudflare) and 10.10.10.5 (mesh)
// Or bind only to mesh: http.ListenAndServe("10.10.10.5:8080", router)
```

## Security Model

```
┌─────────────────────────────────────────────────────┐
│ External Access (Cloudflare)                        │
│   https://api.chronara.net                          │
│         ↓                                           │
│   Web3 Authentication (Future)                      │
│         ↓                                           │
│   Chronara API                                      │
├─────────────────────────────────────────────────────┤
│ Internal Mesh Access (Authenticated Clients)        │
│   10.10.10.5:8080                                   │
│         ↑                                           │
│   ShadowMesh Encrypted Tunnel                       │
│         ↑                                           │
│   Authorized mesh clients only                      │
└─────────────────────────────────────────────────────┘
```

## Pre-Release Stability Checklist

- [ ] VM 111 shadowmesh daemon installed
- [ ] TAP interface chr001 created at 10.10.10.5
- [ ] Connected to relay server
- [ ] All 5 nodes can ping each other
- [ ] Chronara API accessible from mesh network
- [ ] Cloudflare tunnel still operational (external access)
- [ ] All systemd services enabled and auto-start
- [ ] Logs clean (no errors in journalctl)
