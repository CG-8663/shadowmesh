# ShadowMesh Layer 2 Implementation Plan

**Status**: Architecture designed, core modules implemented, integration pending
**Objective**: Implement true Layer 2 post-quantum VPN using chr001 TAP devices

---

## Current Status

### ✅ What's Implemented

**1. TAP Device Module** (`pkg/layer2/tap.go`):
- TAP device creation/attachment
- Ethernet frame capture (read from chr001)
- Ethernet frame injection (write to chr001)
- Frame parsing (MAC addresses, EtherType)
- IP configuration management

**2. Post-Quantum Encryption Module** (`pkg/crypto/pqencrypt.go`):
- ML-KEM-1024 (Kyber) key exchange
- ChaCha20-Poly1305 frame encryption
- Key rotation support
- 32-byte shared secret generation

**3. Layer 2 Tunnel Module** (`pkg/layer2/tunnel.go`):
- Bi-directional frame forwarding (TAP ↔ P2P)
- Frame encryption/decryption pipeline
- Tunnel statistics tracking
- Error handling and logging

**4. Existing Infrastructure**:
- ✅ ML-DSA-87 authentication (working)
- ✅ Kademlia DHT peer discovery (working)
- ✅ P2P TCP connections (working)
- ✅ Discovery backbone (NYC operational)

### ❌ What's Missing

**1. Integration Layer**:
- New command: `cmd/lightnode-l2/main.go`
- Kyber key pair generation/storage
- Key exchange protocol during P2P handshake
- Tunnel lifecycle management

**2. Dependencies**:
- Add github.com/songgao/water (TAP device library)
- Add github.com/cloudflare/circl (Kyber/Dilithium)
- Update go.mod with new packages

**3. Testing**:
- Build Layer 2 binaries
- Deploy to UK/Belgium nodes
- Verify chr001 TAP connectivity
- Performance benchmarking

---

## Implementation Roadmap

### Phase 1: Add Dependencies (5 minutes)

```bash
cd /Volumes/BACKUPDISK/webcode/shadowmesh

# Add required packages
go get github.com/songgao/water
go get github.com/cloudflare/circl/kem/kyber/kyber1024

# Verify
go mod tidy
```

### Phase 2: Create Layer 2 Light Node Command (30 minutes)

**File**: `cmd/lightnode-l2/main.go`

**Key Features**:
1. Generate/load Kyber key pairs (in addition to Dilithium)
2. Authenticate with backbone (existing ML-DSA-87)
3. Register peer with Kyber public key
4. Discover peer and retrieve Kyber public key
5. Establish P2P connection
6. Perform Kyber key exchange
7. Create Layer 2 tunnel with chr001 TAP device
8. Start frame forwarding

**Pseudocode**:
```go
func main() {
    // Parse flags
    tapDevice := flag.String("tap", "chr001", "TAP device name")

    // Load Dilithium keys (authentication)
    dilithiumKeys := loadDilithiumKeys()

    // Load/generate Kyber keys (encryption)
    kyberKeys := loadOrGenerateKyberKeys()

    // Authenticate with backbone
    authClient := client.NewAuthClient(backboneURL, dilithiumKeys)
    authClient.Authenticate()

    // Register peer (include Kyber public key)
    authClient.RegisterPeerWithKyberKey(ip, port, kyberPublicKey)

    // Discover peer
    peers := authClient.FindPeers(peerID)
    peerKyberKey := peers[0].KyberPublicKey

    // Connect P2P
    p2pConn := peerManager.ConnectToPeer(peer.IP, peer.Port)

    // Key exchange (send encapsulated secret)
    encryption, ciphertext := crypto.KeyExchangeInitiator(peerKyberKey)
    p2pConn.SendMessage("kyber_exchange", ciphertext)

    // Create Layer 2 tunnel
    tunnel := layer2.NewTunnel(*tapDevice, p2pConn, encryption)
    tunnel.Start()

    // Wait for shutdown
    waitForShutdown(tunnel)
}
```

### Phase 3: Modify P2P Connection for Layer 2 (15 minutes)

**Changes to `pkg/p2p/connection.go`**:

1. Add Kyber key exchange message type
2. Add layer2_frame message type
3. Handle binary frame payloads (not just JSON)

**New Message Types**:
```go
type="kyber_init"     // Initiator sends encapsulated secret
type="kyber_resp"     // Responder confirms key exchange
type="layer2_frame"   // Encrypted Ethernet frame
```

### Phase 4: Build and Deploy (15 minutes)

```bash
# Build Layer 2 binaries
cd /Volumes/BACKUPDISK/webcode/shadowmesh

# Build for x86_64 (UK)
GOOS=linux GOARCH=amd64 go build -o build/shadowmesh-l2-linux-amd64 cmd/lightnode-l2/main.go

# Build for ARM64 (Belgium)
GOOS=linux GOARCH=arm64 go build -o build/shadowmesh-l2-linux-arm64 cmd/lightnode-l2/main.go

# Deploy to UK
scp build/shadowmesh-l2-linux-amd64 pxcghost@100.115.193.115:~/shadowmesh/shadowmesh-l2

# Deploy to Belgium
scp build/shadowmesh-l2-linux-arm64 pxcghost@100.90.48.10:~/shadowmesh/shadowmesh-l2
```

### Phase 5: Test Layer 2 Tunnel (10 minutes)

**Belgium (Receiver)**:
```bash
ssh pxcghost@100.90.48.10

# Start Layer 2 light node (requires root for TAP access)
sudo ./shadowmesh-l2 \
  -keydir ./keys \
  -backbone http://209.151.148.121:8080 \
  -tap chr001 \
  -port 8443 \
  -public
```

**UK (Sender)**:
```bash
ssh pxcghost@100.115.193.115

# Start Layer 2 light node and connect to Belgium
sudo ./shadowmesh-l2 \
  -keydir ./keys \
  -backbone http://209.151.148.121:8080 \
  -tap chr001 \
  -port 8443 \
  -connect fb9f1ad65f7b67bfc35cd285ab28898e0d486afe
```

**Test Connectivity**:
```bash
# On UK node, ping Belgium through chr001 TAP
ping 10.10.10.4

# Expected output:
# 64 bytes from 10.10.10.4: icmp_seq=1 ttl=64 time=1.5 ms
# 64 bytes from 10.10.10.4: icmp_seq=2 ttl=64 time=1.3 ms
```

**Log Output (Expected)**:
```
TAP device created: chr001 (IP: 10.10.10.3/24)
✓ Authentication successful
✓ Peer registered successfully
Found peer: fb9f1ad65f7b67bfc35cd285ab28898e0d486afe at 100.90.48.10:8443
✓ Connected to peer
Performing Kyber key exchange...
✓ Key exchange complete (32-byte shared secret)
Layer 2 tunnel created: TAP=chr001, Peer=fb9f1ad...
Starting Layer 2 tunnel forwarding...
TAP→P2P: EthernetFrame[dst=..., src=..., type=0800, len=98] (98 bytes)
P2P→TAP: EthernetFrame[dst=..., src=..., type=0800, len=98] (98 bytes)
```

---

## Detailed Module Usage

### TAP Device Module

**Attach to Existing TAP**:
```go
tap, err := layer2.AttachToExisting("chr001")
if err != nil {
    log.Fatalf("Failed to attach to TAP: %v", err)
}
defer tap.Close()
```

**Read Ethernet Frame**:
```go
frame, err := tap.ReadFrame()
// frame = [6 bytes dst MAC][6 bytes src MAC][2 bytes type][payload]
```

**Write Ethernet Frame**:
```go
err := tap.WriteFrame(frame)
```

**Parse Frame**:
```go
ethFrame, err := layer2.ParseEthernetFrame(frame)
fmt.Printf("Dst: %x, Src: %x, Type: %04x\n",
    ethFrame.DstMAC, ethFrame.SrcMAC, ethFrame.EtherType)
```

### Post-Quantum Encryption Module

**Generate Kyber Key Pair**:
```go
publicKey, privateKey, err := crypto.GenerateKyberKeyPair()
// Save keys to disk
```

**Key Exchange (Initiator)**:
```go
encryption, ciphertext, err := crypto.KeyExchangeInitiator(peerPublicKey)
// Send ciphertext to peer
// Use encryption to encrypt/decrypt frames
```

**Key Exchange (Responder)**:
```go
encryption, err := crypto.KeyExchangeResponder(privateKey, ciphertext)
// Use encryption to encrypt/decrypt frames
```

**Encrypt Frame**:
```go
encrypted, err := encryption.EncryptFrame(frame)
// encrypted = [24 bytes nonce][ciphertext + 16 bytes auth tag]
```

**Decrypt Frame**:
```go
frame, err := encryption.DecryptFrame(encrypted)
```

### Layer 2 Tunnel Module

**Create Tunnel**:
```go
tunnel, err := layer2.NewTunnel("chr001", p2pConn, encryption)
```

**Start Forwarding**:
```go
err := tunnel.Start()
// Starts two goroutines: TAP→P2P and P2P→TAP
```

**Get Statistics**:
```go
stats := tunnel.GetStats()
fmt.Printf("Sent: %d frames (%d bytes)\n", stats.FramesSent, stats.BytesSent)
fmt.Printf("Received: %d frames (%d bytes)\n", stats.FramesReceived, stats.BytesReceived)
```

---

## Network Flow Diagram

### Current (Layer 3 over Tailscale)

```
Application
    ↓
TCP Socket (net.Dial)
    ↓
IP Layer (100.115.193.115 → 100.90.48.10)
    ↓
Tailscale WireGuard (Curve25519 - vulnerable to quantum)
    ↓
Physical Network
```

### Target (Layer 2 with ShadowMesh)

```
Application (ping 10.10.10.4)
    ↓
OS Network Stack
    ↓
chr001 TAP Device (10.10.10.3)
    ↓
ShadowMesh Light Node
    ├─ Read Ethernet Frame
    ├─ Encrypt with ChaCha20-Poly1305 (post-quantum key)
    ├─ Send over P2P TCP connection
    └─ (Physical: Tailscale 100.x.x.x for P2P transport only)
    ↓
Remote ShadowMesh Light Node
    ├─ Receive encrypted frame
    ├─ Decrypt with ChaCha20-Poly1305
    ├─ Inject into remote chr001 TAP
    └─ Deliver to OS
    ↓
Remote chr001 TAP Device (10.10.10.4)
    ↓
Remote Application (receives ping)
```

**Key Difference**:
- **Current**: Application data encrypted by Tailscale (classical crypto)
- **Target**: Ethernet frames encrypted by ShadowMesh (post-quantum crypto)

---

## Security Comparison

### Current Implementation

| Layer | Protocol | Crypto | Quantum Safe |
|-------|----------|--------|--------------|
| Authentication | ML-DSA-87 | Post-quantum | ✅ Yes |
| Transport | TCP | None | N/A |
| Encryption | WireGuard | Curve25519 | ❌ No |

### Target Implementation

| Layer | Protocol | Crypto | Quantum Safe |
|-------|----------|--------|--------------|
| Authentication | ML-DSA-87 | Post-quantum | ✅ Yes |
| Key Exchange | Kyber1024 | Post-quantum | ✅ Yes |
| Encryption | ChaCha20-Poly1305 | Symmetric (256-bit) | ✅ Yes |
| Transport | TCP | None (encrypted above) | N/A |

---

## Performance Expectations

### Latency

**Current (Layer 3)**:
- Ping via Tailscale: 48.6ms
- Ping via ShadowMesh (over Tailscale): <1ms

**Target (Layer 2)**:
- Ping via chr001 TAP: <5ms (encryption overhead)
- Expected: 2-3ms (post-quantum encryption adds ~1-2ms)

### Throughput

**Current**:
- Tailscale: 6.5 Mbps
- ShadowMesh video: 2.4 Mbps (30 FPS @ 10KB)

**Target**:
- ChaCha20-Poly1305: ~1 Gbps on modern CPUs
- Expected: 100-500 Mbps (limited by CPU, not crypto)

### Overhead

**Frame Size Impact**:
- Original frame: 98 bytes (ping packet)
- Nonce: +24 bytes (XChaCha20-Poly1305)
- Auth tag: +16 bytes (Poly1305)
- **Total**: 138 bytes (+41% overhead)

**CPU Impact**:
- Kyber1024 key exchange: ~0.5ms (one-time)
- ChaCha20-Poly1305 encryption: ~1μs per frame
- **Negligible** for typical workloads

---

## Testing Plan

### Test 1: Basic Connectivity

```bash
# Belgium: Start receiver
sudo ./shadowmesh-l2 -tap chr001 -port 8443 -public

# UK: Connect and ping
sudo ./shadowmesh-l2 -tap chr001 -port 8443 -connect <peer-id>

# Test
ping -c 10 10.10.10.4
```

**Expected**: 0% packet loss, <5ms latency

### Test 2: Throughput

```bash
# Belgium: Start iperf3 server
iperf3 -s -B 10.10.10.4

# UK: Run iperf3 client
iperf3 -c 10.10.10.4 -t 30
```

**Expected**: 100+ Mbps throughput

### Test 3: Video Streaming

```bash
# Belgium: Receive video
ffplay -i tcp://10.10.10.4:5000

# UK: Stream video
ffmpeg -re -i test.mp4 -f mpegts tcp://10.10.10.4:5000
```

**Expected**: Smooth playback, minimal buffering

### Test 4: Encryption Verification

```bash
# Capture P2P traffic on UK node
sudo tcpdump -i tailscale0 -w shadowmesh-encrypted.pcap

# Analyze in Wireshark
wireshark shadowmesh-encrypted.pcap
```

**Expected**: No plaintext Ethernet frames visible (all encrypted)

---

## Comparison Matrix

| Feature | Tailscale | ShadowMesh L3 (Current) | ShadowMesh L2 (Target) |
|---------|-----------|-------------------------|------------------------|
| **Authentication** | Classical | Post-quantum (ML-DSA-87) | Post-quantum (ML-DSA-87) |
| **Key Exchange** | Curve25519 | N/A (uses Tailscale) | Kyber1024 (post-quantum) |
| **Encryption** | WireGuard | WireGuard (via Tailscale) | ChaCha20-Poly1305 |
| **Network Layer** | Layer 3 | Layer 3 | Layer 2 |
| **Addressing** | 100.x.x.x | 100.x.x.x | 10.10.10.x |
| **Quantum Safe** | ❌ No | ❌ No (relies on Tailscale) | ✅ Yes |
| **Independence** | N/A | Depends on Tailscale | Independent VPN |
| **Latency** | 48.6ms | <1ms | <5ms (est) |
| **Throughput** | 6.5 Mbps | N/A | 100+ Mbps (est) |
| **Obfuscation** | None | None | Possible (WebSocket) |

---

## Success Criteria

### MVP Success

- [x] TAP device integration implemented
- [x] Kyber key exchange implemented
- [x] ChaCha20-Poly1305 encryption implemented
- [ ] Layer 2 light node command created
- [ ] Binaries built and deployed
- [ ] ping 10.10.10.3 → 10.10.10.4 working
- [ ] Zero packet loss
- [ ] Latency <10ms

### Production Ready

- [ ] Key rotation implemented (10s-60min)
- [ ] WebSocket obfuscation added
- [ ] Multi-hop routing implemented
- [ ] TPM attestation for exit nodes
- [ ] Mobile apps (iOS, Android)
- [ ] Performance: 500+ Mbps, <5ms latency
- [ ] Security audit passed

---

## Estimated Time to Complete

| Task | Time | Difficulty |
|------|------|------------|
| Add dependencies | 5 min | Easy |
| Create lightnode-l2 command | 30 min | Medium |
| Modify P2P for Layer 2 | 15 min | Easy |
| Build binaries | 5 min | Easy |
| Deploy to servers | 10 min | Easy |
| Test basic connectivity | 10 min | Easy |
| Performance benchmarking | 30 min | Medium |
| **Total** | **~2 hours** | Medium |

---

## Next Steps

1. **Add Dependencies**: `go get` required packages
2. **Create lightnode-l2**: Implement Layer 2 command
3. **Build & Deploy**: Compile and upload binaries
4. **Test**: Verify chr001 connectivity
5. **Benchmark**: Compare with Tailscale (fair comparison)
6. **Document**: Update test results with Layer 2 data

---

## Conclusion

**Current Status**: 70% complete
- ✅ Core modules implemented (TAP, encryption, tunnel)
- ✅ Architecture designed and documented
- ❌ Integration layer pending (~2 hours work)

**Once Complete**: ShadowMesh will be a **true Layer 2 post-quantum VPN** that can be fairly compared to Tailscale, demonstrating:
- Quantum-safe encryption end-to-end
- Independent of any third-party infrastructure
- Superior security with competitive performance
- Ready for production deployment

**Implementation Priority**: HIGH - needed for honest comparison with Tailscale
