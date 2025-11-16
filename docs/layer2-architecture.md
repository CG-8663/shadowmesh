# Layer 2 Architecture - TAP Device Management

## Overview

ShadowMesh uses a **Layer 2 (Ethernet) architecture** via TAP devices instead of the traditional Layer 3 (IP) approach used by most VPNs. This provides enhanced security, better performance, and complete protocol transparency.

## TAP vs TUN Comparison

| Feature | TAP (Layer 2) | TUN (Layer 3) |
|---------|---------------|---------------|
| **Protocol Layer** | Ethernet (Layer 2) | IP (Layer 3) |
| **Frame Type** | Ethernet frames | IP packets |
| **Supported Protocols** | All (IP, ARP, IPv6, custom) | IP only |
| **Overhead** | 14-byte Ethernet header | No Ethernet header |
| **Use Case** | Bridging, full network isolation | Point-to-point tunnels |
| **ShadowMesh Choice** | ✅ TAP (Layer 2) | ❌ Not used |

## Why Layer 2?

1. **Protocol Transparency**: Supports all network protocols (IP, ARP, IPv6, custom protocols)
2. **Security**: Complete isolation from host network stack until exit node
3. **Performance**: Direct frame encryption without IP stack overhead
4. **Future-Proof**: Supports emerging protocols without code changes

## TAP Device Implementation

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    User Application                      │
│              (Browser, SSH, etc.)                        │
└─────────────────────┬───────────────────────────────────┘
                      │
                      │ IP packets
                      ▼
┌─────────────────────────────────────────────────────────┐
│                  Host OS Network Stack                   │
│              (Routing, TCP/IP, etc.)                     │
└─────────────────────┬───────────────────────────────────┘
                      │
                      │ Ethernet frames
                      ▼
┌─────────────────────────────────────────────────────────┐
│               TAP Device (shadowmesh0)                   │
│          IP: 10.99.X.Y/16 (assigned by relay)           │
└─────────────────────┬───────────────────────────────────┘
                      │
                      │ Raw Ethernet frames
                      ▼
┌─────────────────────────────────────────────────────────┐
│              ShadowMesh Client Daemon                    │
│         (Frame Encryption + Tunnel Manager)              │
└─────────────────────┬───────────────────────────────────┘
                      │
                      │ Encrypted frames
                      ▼
┌─────────────────────────────────────────────────────────┐
│          WebSocket Transport (WSS + Obfuscation)         │
│              → Relay or Direct P2P Peer                  │
└─────────────────────────────────────────────────────────┘
```

### TAP Device Configuration

**Device Name**: Platform-specific

- **Linux**: `shadowmesh0` (configurable in config.yaml)
- **macOS**: System-assigned `utunX` or `tapX` (e.g., `utun0`, `tap1`)
  - Custom names not supported on macOS
  - `shadowmesh0` used as internal identifier only
  - Daemon logs actual OS-assigned name on startup

**IP Subnet**: `10.99.0.0/16`
- Reserved for ShadowMesh internal network
- Relay assigns unique IP to each client (10.99.X.Y)
- Supports 65,534 simultaneous clients per relay

**MTU**: `1500 bytes` (Ethernet standard)
- Optimized for maximum compatibility
- Configurable for high-throughput scenarios

### Platform-Specific Naming

**Linux TAP Device Naming**:
- Supports arbitrary custom names (e.g., `shadowmesh0`, `vpn0`, `mesh1`)
- If no name specified, kernel assigns `tap0`, `tap1`, etc. sequentially
- Name configured via `water.Config.Name` parameter

**macOS TAP Device Naming**:
- **utunX**: User-space tunnel interfaces (macOS Sierra+)
  - System automatically assigns number (`utun0`, `utun1`, `utun2`, etc.)
  - Most common on modern macOS
- **tapX**: Requires third-party kernel extension (kext)
  - Requires Tunnelblick or legacy tuntap kext
  - System assigns number (`tap0`, `tap1`, etc.)
  - Custom names not allowed (e.g., cannot use `shadowmesh0`)
- **Internal Naming**: Application uses `shadowmesh0` as internal identifier
  - Config file specifies `shadowmesh0` for consistency
  - Daemon logs actual assigned name: `TAP device created: utun3`
  - Application code references device via `tap.Name()` method

### Privilege Requirements

TAP device creation and configuration requires **CAP_NET_ADMIN** capability or root privileges:

```bash
# Option 1: Run as root (not recommended)
sudo shadowmesh-daemon

# Option 2: Grant CAP_NET_ADMIN capability (recommended)
sudo setcap cap_net_admin+eip /usr/local/bin/shadowmesh-daemon

# Option 3: systemd service with capabilities (production)
# See systemd/shadowmesh.service
AmbientCapabilities=CAP_NET_ADMIN
CapabilityBoundingSet=CAP_NET_ADMIN
```

## Frame Processing Pipeline

### Ethernet Frame Structure

All frames captured from and written to the TAP device follow the standard Ethernet II frame format:

```
┌──────────────────┬──────────────────┬──────────────┬────────────────────┐
│  Destination MAC │   Source MAC     │  EtherType   │      Payload       │
│    (6 bytes)     │    (6 bytes)     │  (2 bytes)   │   (46-1500 bytes)  │
├──────────────────┼──────────────────┼──────────────┼────────────────────┤
│   Bytes 0-5      │   Bytes 6-11     │  Bytes 12-13 │   Bytes 14-end     │
└──────────────────┴──────────────────┴──────────────┴────────────────────┘

Total Frame Size: 14-1514 bytes (14-byte header + 0-1500 byte payload)
```

**Supported EtherTypes** (network byte order, big-endian):
- `0x0800` - IPv4 (Internet Protocol version 4)
- `0x0806` - ARP (Address Resolution Protocol)
- `0x86DD` - IPv6 (Internet Protocol version 6)
- All other values - Pass-through (custom protocols, VLAN tags, etc.)

**Frame Size Constraints**:
- **Minimum**: 14 bytes (header only, no payload)
- **Maximum**: 1514 bytes (1500 MTU + 14 header)
- Frames outside this range are rejected with error

### Outbound Frames (Client → Network)

1. **Application** sends IP packet
2. **Host OS** wraps in Ethernet frame, routes to TAP device
3. **TAP Device** captures raw Ethernet frame (14-1514 bytes)
4. **Frame Parser** (`layer2.ParseFrame()`) validates and parses frame:
   - Validates size (14-1514 bytes)
   - Extracts destination MAC, source MAC, EtherType
   - Extracts payload
   - Rejects malformed frames (logged and dropped)
5. **Serializer** (`frame.Serialize()`) converts parsed frame back to bytes
6. **Encryption Pipeline** encrypts frame bytes with ChaCha20-Poly1305
7. **Tunnel Manager** sends encrypted frame via WebSocket
8. **Relay/Peer** receives and forwards

### Inbound Frames (Network → Client)

1. **Relay/Peer** sends encrypted frame via WebSocket
2. **Tunnel Manager** receives frame
3. **Decryption Pipeline** decrypts frame to raw bytes
4. **Frame Validator** checks size (14-1514 bytes)
5. **TAP Device** injects validated frame into OS network stack
6. **Host OS** processes Ethernet frame
7. **Application** receives IP packet

### Frame Parsing Performance

**Benchmark Results** (Apple M1 Max, Go 1.21):
```
BenchmarkParseFrame-10           34,517,388 ops   35.44 ns/op    72 B/op   2 allocs/op
BenchmarkParseFrameMaxSize-10     5,341,740 ops  233.1  ns/op  1584 B/op   2 allocs/op
```

**Throughput Capacity**:
- Regular frames: ~28 million frames/sec (1 / 35.44ns)
- Max-size frames: ~4.3 million frames/sec (1 / 233.1ns)
- Real-world target: >10,000 frames/sec (easily achieved)

**Latency**:
- Frame parsing: 0.000035 ms (35.44 ns)
- Frame serialization: ~35 ns (same order of magnitude)
- Total parsing overhead: <0.0001 ms (negligible)

## Configuration

### config.yaml

```yaml
tap:
  name: shadowmesh0           # TAP device name
  mtu: 1500                   # Maximum Transmission Unit
  ip_addr: 10.99.1.1         # Assigned by relay (placeholder)
  netmask: 16                # /16 subnet (10.99.0.0/16)
```

### Daemon Integration

The daemon automatically:
1. Creates TAP device on startup
2. Configures IP address and brings interface up
3. Starts read/write goroutines for frame processing
4. Cleans up TAP device on shutdown

## Debugging

### Check TAP Device Status

**Linux**:
```bash
# List all network interfaces
ip link show

# Show TAP device details
ip link show shadowmesh0

# Show IP configuration
ip addr show shadowmesh0

# Check interface statistics
ip -s link show shadowmesh0
```

**macOS**:
```bash
# List all network interfaces (look for utunX or tapX)
ifconfig

# Show specific interface (replace utun0 with actual assigned name)
ifconfig utun0

# List all hardware ports
networksetup -listallhardwareports

# Show routing table
netstat -rn
```

### Monitor Frame Traffic

**Linux**:
```bash
# Capture frames on TAP device (requires root)
sudo tcpdump -i shadowmesh0 -v

# Monitor specific protocols
sudo tcpdump -i shadowmesh0 icmp           # ICMP (ping)
sudo tcpdump -i shadowmesh0 tcp port 443   # HTTPS traffic
sudo tcpdump -i shadowmesh0 arp            # ARP requests
```

**macOS**:
```bash
# Capture frames on TAP device (use actual assigned name)
sudo tcpdump -i utun0 -v

# Monitor specific protocols
sudo tcpdump -i utun0 icmp
sudo tcpdump -i utun0 tcp port 443
```

### Troubleshooting

**Problem**: TAP device not created (Linux)

```bash
# Check if running as root or with CAP_NET_ADMIN
id
getcap /usr/local/bin/shadowmesh-daemon

# Check kernel module loaded
lsmod | grep tun

# Load TUN/TAP module if missing
sudo modprobe tun
```

**Problem**: TAP device not created (macOS)

```bash
# Check if running as root
id

# Check if TUN/TAP kext is loaded
kextstat | grep tap

# Install TUN/TAP kext if missing
# Option 1: Install Tunnelblick (includes tuntap kext)
# Download from: https://tunnelblick.net/

# Option 2: Install legacy tuntap package
# Download from: http://tuntaposx.sourceforge.net/

# After installation, verify kext loaded
kextstat | grep net.sf.tuntaposx
```

**Problem**: Interface not coming up

```bash
# Manually bring up interface
sudo ip link set shadowmesh0 up

# Check for errors in daemon logs
journalctl -u shadowmesh -f
```

**Problem**: No connectivity through tunnel

```bash
# Verify IP configuration
ip addr show shadowmesh0

# Check routing table
ip route show

# Ping relay gateway (if configured)
ping -c 3 10.99.0.1

# Test frame capture
sudo tcpdump -i shadowmesh0 -c 10
```

## Performance Considerations

### Throughput

- **Target**: >10,000 frames/second per client
- **Bottleneck**: Encryption pipeline (ChaCha20-Poly1305)
- **Optimization**: Batch frame processing, CPU affinity

### Latency

- **TAP Overhead**: <0.1ms (frame capture/injection)
- **Total Overhead**: <2ms (TAP + encryption + WebSocket)
- **Comparison**: WireGuard adds ~0.5-1ms overhead

### Buffer Sizing

```go
readChan:  make(chan *layer2.EthernetFrame, 100)  // 100 parsed frames buffered
writeChan: make(chan []byte, 100)                 // 100 raw frames buffered
```

- **readChan**: Parsed `EthernetFrame` structs (validated at capture)
- **writeChan**: Raw byte slices (validated before injection)
- Adjust buffer sizes for high-throughput scenarios
- Monitor dropped frames via error channel

## Security Implications

1. **No IP Stack Exposure**: Frames are encrypted before IP processing at relay
2. **Protocol Obfuscation**: Exit node performs IP processing, hiding client's protocol usage
3. **ARP Isolation**: ARP requests stay within TAP device, not broadcast to physical network
4. **Privilege Separation**: Only TAP creation requires CAP_NET_ADMIN, not entire daemon

## Testing

### Unit Tests

```bash
# Run tests (requires root or CAP_NET_ADMIN)
cd client/daemon
sudo go test -v -run TestTAPDevice

# Run without root (tests will skip)
go test -v -run TestTAPDevice
```

### Integration Tests

```bash
# Test full tunnel setup
cd test/integration
sudo go test -v -run TestTunnelSetup

# Benchmark frame throughput
sudo go test -bench=BenchmarkTAPDeviceFrameProcessing
```

## References

- TAP/TUN Linux Documentation: https://www.kernel.org/doc/Documentation/networking/tuntap.txt
- `github.com/songgao/water` library: https://github.com/songgao/water
- Linux Capabilities: `man capabilities`
- systemd Security: `man systemd.exec`

## Future Enhancements

1. **Multi-TAP Support**: Multiple TAP devices for network isolation
2. **Dynamic MTU**: Adjust MTU based on path MTU discovery
3. **Hardware Offload**: Leverage NIC features for frame processing
4. **DPDK Integration**: Bypass kernel for ultra-low latency
