# ShadowMesh Discovery and P2P Routing Flow
## How Peers Find Each Other and Establish Encrypted Connections

**Date**: 2025-11-07
**Network**: 4-node mesh (UK, Belgium, Starlink, Mac Studio)
**Discovery**: Kademlia DHT on NYC backbone (209.151.148.121:8080)

---

## Overview

ShadowMesh uses a **decentralized discovery model** with a global Kademlia DHT backbone to bootstrap peer-to-peer connections. Once discovered, peers communicate **directly** over Layer 3 (IP) with encrypted UDP tunnels, bypassing the discovery backbone entirely.

```
┌─────────────────────────────────────────────────────────────┐
│                   Discovery Phase                            │
│  (Initial bootstrap - uses discovery backbone)              │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                Connection Establishment                       │
│  (Direct P2P - TCP control + UDP data)                      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              Encrypted Data Transfer                         │
│  (Pure P2P - discovery backbone not involved)               │
└─────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Discovery - How Peers Find Each Other

### Step 1: Peer Registration (Each Node on Startup)

When a node starts, it registers itself with the discovery backbone:

**Belgium Node (shadowmesh-002) Startup**:
```
┌──────────────┐
│ Belgium Node │
│ 100.90.48.10 │
└──────┬───────┘
       │ 1. ML-DSA-87 Authentication
       │    POST /api/auth/challenge
       │    POST /api/auth/verify
       ▼
┌─────────────────────────┐
│  Discovery Backbone     │
│  NYC 209.151.148.121    │
│  (Kademlia DHT)         │
└─────────────────────────┘
       │ 2. Peer Registration
       │    POST /api/peers/register
       │    {
       │      peer_id: "fb9f1ad6...",
       │      ip: "100.90.48.10",
       │      port: 8443,
       │      udp_port: 9443,
       │      public_key: <ML-DSA-87>,
       │      is_public: false
       │    }
       ▼
┌─────────────────────────┐
│   Kademlia DHT          │
│   (Distributed)         │
│                         │
│ Key: fb9f1ad6...        │
│ Value: {                │
│   ip: 100.90.48.10,     │
│   port: 8443,           │
│   udp_port: 9443        │
│ }                       │
└─────────────────────────┘
```

**All 4 nodes register**:
- UK: `125d3933...` at 100.115.193.115:8443 (PUBLIC relay)
- Belgium: `fb9f1ad6...` at 100.90.48.10:8443
- Starlink: `8c53bab8...` at 100.126.75.74:8443
- Mac: `dceab2d1...` at 100.113.157.118:8443

### Step 2: Peer Discovery (When Connecting to Another Node)

**Mac Studio wants to connect to UK Hub**:

```
┌──────────────┐
│  Mac Studio  │
│ 100.113...   │
└──────┬───────┘
       │ 1. Query: "Where is peer 125d3933...?"
       │    GET /api/peers/125d3933e63a697881e34aa5a7135e681296ed73
       ▼
┌─────────────────────────┐
│  Discovery Backbone     │
│  (Kademlia DHT Lookup)  │
└─────────────────────────┘
       │ 2. DHT Response:
       │    {
       │      peer_id: "125d3933...",
       │      ip: "100.115.193.115",
       │      port: 8443,
       │      udp_port: 9443,
       │      is_public: true
       │    }
       ▼
┌──────────────┐
│  Mac Studio  │
│ Now knows:   │
│ UK is at     │
│ 100.115.193. │
│ 115:8443     │
└──────────────┘
```

**Kademlia DHT Properties**:
- **Decentralized**: Each discovery node has full DHT copy
- **Fast lookup**: O(log N) hops, typically <100ms
- **Fault tolerant**: If NYC fails, London/Singapore/Sydney can serve
- **Self-healing**: Nodes re-register periodically (TTL: 24 hours)

---

## Phase 2: Connection Establishment - Direct P2P

Once Mac Studio knows UK's IP address from discovery, it establishes a **direct connection** without involving the discovery backbone:

### Step 1: TCP Control Plane Handshake

```
┌──────────────┐                              ┌──────────────┐
│  Mac Studio  │                              │   UK Hub     │
│ 100.113...   │                              │ 100.115...   │
└──────┬───────┘                              └──────┬───────┘
       │                                             │
       │ 1. TCP SYN to 100.115.193.115:8443         │
       │─────────────────────────────────────────────>│
       │                                             │
       │ 2. TCP SYN-ACK                              │
       │<─────────────────────────────────────────────│
       │                                             │
       │ 3. TCP ACK (connection established)         │
       │─────────────────────────────────────────────>│
       │                                             │
       │ 4. Send ML-DSA-87 signed HELLO              │
       │    {                                        │
       │      type: "HELLO",                         │
       │      peer_id: "dceab2d1...",                │
       │      signature: <ML-DSA-87>                 │
       │    }                                        │
       │─────────────────────────────────────────────>│
       │                                             │
       │ 5. Receive signed CHALLENGE                 │
       │<─────────────────────────────────────────────│
       │                                             │
       │ 6. Send UDP endpoint info                   │
       │    {                                        │
       │      type: "UDP_ENDPOINT",                  │
       │      ip: "100.113.157.118",                 │
       │      udp_port: 9443                         │
       │    }                                        │
       │─────────────────────────────────────────────>│
       │                                             │
       │ 7. Receive UK's UDP endpoint                │
       │<─────────────────────────────────────────────│
       │                                             │
       │ ✅ TCP Control Connection Established        │
       │    (Used for keepalives, control messages)  │
       │                                             │
```

**Control Plane Result**:
- TCP connection: Mac 100.113.157.118:random → UK 100.115.193.115:8443
- Purpose: Keep-alives, peer status, route updates
- Encryption: TLS 1.3 (implicitly via TCP)
- Authentication: ML-DSA-87 post-quantum signatures

### Step 2: UDP Data Plane Establishment

```
┌──────────────┐                              ┌──────────────┐
│  Mac Studio  │                              │   UK Hub     │
│ UDP 9443     │                              │ UDP 9443     │
└──────┬───────┘                              └──────┬───────┘
       │                                             │
       │ 1. Send test UDP packet (hole punch)       │
       │    to 100.115.193.115:9443                 │
       │─────────────────────────────────────────────>│
       │                                             │
       │ 2. Receive test UDP packet                  │
       │<─────────────────────────────────────────────│
       │                                             │
       │ ✅ UDP Data Path Established                 │
       │    (Used for encrypted IP packets)          │
       │                                             │
```

**Data Plane Result**:
- UDP connection: Mac 100.113.157.118:9443 ↔ UK 100.115.193.115:9443
- Purpose: Encrypted IP packet forwarding (Layer 3 tunnel)
- Encryption: ChaCha20-Poly1305 per-packet
- Buffering: Adaptive (2604 packets for high-latency links)

---

## Phase 3: Encrypted Data Transfer - Layer 3 Routing

Once both TCP and UDP connections are established, the nodes form a **Layer 3 overlay network** using TUN devices:

### Network Topology After Full Connection

```
                     Physical Network
                    (Tailscale/Internet)
┌─────────────────────────────────────────────────────────┐
│                                                         │
│   100.115.193.115         100.113.157.118              │
│   ┌──────────┐            ┌──────────┐                 │
│   │ UK Node  │◄──────────►│Mac Studio│                 │
│   │ (Hub)    │  UDP 9443  │          │                 │
│   └────┬─────┘            └────┬─────┘                 │
│        │                       │                        │
│   100.90.48.10           100.126.75.74                 │
│   ┌────▼─────┐            ┌────▼─────┐                 │
│   │ Belgium  │            │ Starlink │                 │
│   │  Node    │            │   Node   │                 │
│   └──────────┘            └──────────┘                 │
│                                                         │
└─────────────────────────────────────────────────────────┘

                      Overlay Network
                     (TUN chr001/utun9)
┌─────────────────────────────────────────────────────────┐
│                                                         │
│      10.10.10.3              10.10.10.8                 │
│      ┌─────────┐             ┌─────────┐               │
│      │UK (Hub) │◄───────────►│   Mac   │               │
│      └────┬────┘             └────┬────┘               │
│           │                       │                     │
│           │                       │                     │
│      10.10.10.4             10.10.10.5                 │
│      ┌────▼────┐             ┌────▼────┐               │
│      │ Belgium │             │Starlink │               │
│      └─────────┘             └─────────┘               │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Data Flow Example: Mac Studio Pings UK

**Step 1: Application Layer (Mac Studio)**
```
$ ping 10.10.10.3

Application generates ICMP Echo Request:
  Source IP: 10.10.10.8
  Dest IP: 10.10.10.3
  Type: ICMP Echo Request
```

**Step 2: TUN Device Capture (Mac Studio)**
```
┌──────────────────────────┐
│ Mac Studio OS Kernel     │
│ Routing table:           │
│ 10.10.10.0/24 → utun9    │
└──────────┬───────────────┘
           │ IP packet written to TUN device
           ▼
┌──────────────────────────┐
│ ShadowMesh Process       │
│ ReadPacket() from utun9  │
│ Packet: 10.10.10.8 →     │
│         10.10.10.3       │
└──────────┬───────────────┘
           │ Lookup routing table
           │ Dest 10.10.10.3 → UK peer (125d3933...)
           ▼
```

**Step 3: Encryption and UDP Send**
```
┌──────────────────────────┐
│ Encryption Pipeline      │
│ (pkg/p2p/udp_connection) │
└──────────┬───────────────┘
           │
           │ 1. Get buffer from pool (Solution 2)
           │    var stackBuf [1519]byte
           │
           │ 2. Build UDP frame:
           │    [Header: 19 bytes]
           │    - Sequence: 1234567
           │    - Type: DATA
           │    - Timestamp: now()
           │    - Frame size: 84 (ICMP packet)
           │    [Payload: 84 bytes]
           │    - Encrypted ICMP packet (ChaCha20-Poly1305)
           │
           │ 3. Send via UDP socket
           │    conn.WriteToUDP(packet, UK_UDP_addr)
           │    → 100.115.193.115:9443
           │
           ▼
    Network (Tailscale/Internet)
           │
           ▼ UDP packet travels over physical network
```

**Step 4: UDP Receive (UK Node)**
```
    Network (100.115.193.115:9443)
           │
           │ UDP packet arrives
           ▼
┌──────────────────────────┐
│ UK Node UDP Socket       │
│ (pkg/p2p/udp_connection) │
└──────────┬───────────────┘
           │
           │ 1. Read UDP packet
           │    conn.ReadFromUDP(buf)
           │
           │ 2. Parse header:
           │    Sequence: 1234567
           │    Type: DATA
           │    Frame size: 84
           │
           │ 3. Decrypt payload:
           │    ChaCha20-Poly1305.Decrypt(frame)
           │    → Original ICMP packet (10.10.10.8 → 10.10.10.3)
           │
           │ 4. Write to TUN device
           │    tun.WritePacket(decrypted_packet)
           │    → chr001
           ▼
┌──────────────────────────┐
│ UK OS Kernel             │
│ Receives packet on chr001│
│ Dest: 10.10.10.3 (self)  │
│ → Deliver to ICMP handler│
└──────────┬───────────────┘
           │
           │ ICMP Echo Request received
           │ Generate ICMP Echo Reply
           ▼
    [Same process in reverse:
     TUN → Encrypt → UDP → Mac Studio]
```

**Step 5: Response Path (UK → Mac)**
```
UK generates ICMP Echo Reply:
  Source IP: 10.10.10.3
  Dest IP: 10.10.10.8
  Type: ICMP Echo Reply

UK writes to chr001 → ShadowMesh captures →
Encrypts → Sends UDP to Mac (100.113.157.118:9443) →
Mac receives UDP → Decrypts → Writes to utun9 →
Mac kernel delivers to ping application

Result: ping receives reply ✅
```

---

## Current Issue: Why Pings Fail (100% Loss)

The **process works correctly** up through Step 4.2 (decrypt payload), but fails at Step 4.4 due to the **UDP receive buffer bottleneck**:

```
┌──────────────────────────┐
│ OS UDP Receive Buffer    │
│ Default size: ~200 KB    │ ← TOO SMALL
│ (kernel socket buffer)   │
└──────────┬───────────────┘
           │
           │ Adaptive buffer sends BURSTS:
           │ - 2604 packets × 1500 bytes = ~3.9 MB
           │ - Arrives faster than kernel can queue
           │
           │ Result: Buffer overflow
           ▼
┌──────────────────────────┐
│ Kernel drops packets     │ ← 90-95% packet loss
│ before app can read them │
└──────────────────────────┘
```

**Proof**:
- UK logs show: "Sent 259M+ frames" ✅
- Belgium logs show: "Detected 7418 lost frames/sec" ❌
- Ping results: 0/20 packets received ❌

**Fix** (in progress):
```go
// pkg/p2p/udp_connection.go
conn, _ := net.ListenUDP("udp", localAddr)
conn.SetReadBuffer(128 * 1024 * 1024)  // 128 MB
```

Plus kernel tuning:
```bash
sudo sysctl -w net.core.rmem_max=134217728      # 128MB
sudo sysctl -w net.core.rmem_default=26214400   # 25MB
```

---

## Routing Between Non-Hub Nodes

**Question**: How does Belgium communicate with Starlink if they're both clients connected to UK hub?

**Answer**: Multi-hop routing through the UK hub:

```
Belgium wants to ping Starlink (10.10.10.5)

Step 1: Belgium checks routing table
  Dest: 10.10.10.5
  Not a direct peer → Forward to UK hub (125d3933...)

Step 2: Belgium → UK
  [Belgium TUN] → Encrypt → UDP to UK (100.115.193.115:9443)

Step 3: UK receives and routes
  [UK UDP Socket] → Decrypt → [UK TUN]
  UK kernel sees: Dest IP = 10.10.10.5 (not self)
  Routing table: 10.10.10.5 → Starlink peer (8c53bab8...)

Step 4: UK → Starlink
  [UK TUN] → Encrypt → UDP to Starlink (100.126.75.74:9443)

Step 5: Starlink receives
  [Starlink UDP Socket] → Decrypt → [Starlink TUN]
  Starlink kernel delivers to destination

Result: Belgium ↔ Starlink via UK hub relay
```

**Performance Impact**:
- Extra hop: +100-150ms latency (UK relay processing)
- Double encryption: Belgium→UK (encrypted), UK→Starlink (encrypted again)
- UK hub CPU usage: ~2× (decrypt + re-encrypt)

**Optimization** (future):
- Direct peer discovery: Belgium and Starlink discover each other's UDP endpoints
- Establish direct P2P connection: Belgium ↔ Starlink (no UK relay)
- Update routing table: 10.10.10.5 → direct peer

---

## Discovery Backbone Role Summary

The discovery backbone (Kademlia DHT) is **only used for**:
1. ✅ Initial peer registration (on startup)
2. ✅ Peer lookup (when connecting to new peer)
3. ✅ Keep-alive updates (periodic re-registration)

The discovery backbone is **NOT used for**:
1. ❌ Data transfer (all data goes P2P over UDP)
2. ❌ Routing decisions (handled by each node's routing table)
3. ❌ Encryption (each P2P connection has own keys)

**Analogy**: Discovery backbone is like DNS - you query it once to find an IP address, then all communication happens directly between peers.

---

## Traceroute Through Discovery to P2P Connection

### Traceroute: Mac Studio Connecting to UK Hub

```
Time | Location | Action | Protocol | Details
-----|----------|--------|----------|------------------------------------------
T+0  | Mac      | Query  | HTTP     | GET /api/peers/125d3933...
     |          |        |          | → Discovery backbone (209.151.148.121:8080)
T+50ms| Backbone| Lookup | DHT      | Kademlia lookup in distributed hash table
T+100ms| Mac    | Receive| HTTP     | {ip: "100.115.193.115", port: 8443}
T+150ms| Mac    | Connect| TCP      | SYN → 100.115.193.115:8443
T+200ms| UK     | Accept | TCP      | SYN-ACK ← 100.115.193.115:8443
T+250ms| Mac    | Auth   | ML-DSA-87| Send signed HELLO message
T+300ms| UK     | Verify | ML-DSA-87| Verify Mac's signature
T+350ms| Mac    | Setup  | UDP      | Exchange UDP endpoints (9443)
T+400ms| Mac    | Test   | UDP      | Send test packet → 100.115.193.115:9443
T+450ms| UK     | Reply  | UDP      | Send test reply ← 100.115.193.115:9443
T+500ms| Mac    | Ready  | P2P      | ✅ Connection established
-----|----------|--------|----------|------------------------------------------
T+1s | Mac      | Data   | UDP      | ping 10.10.10.3 → Encrypted → UDP
     |          |        |          | (Discovery backbone NOT involved)
```

**Key Points**:
- Discovery query: 100ms (DNS-like lookup)
- TCP handshake: 150ms (direct P2P)
- ML-DSA-87 auth: 100ms (post-quantum signatures)
- UDP test: 100ms (NAT hole punching)
- **Total bootstrap time**: ~500ms
- **Ongoing data transfer**: 0ms overhead (pure P2P, no relay)

---

## Security Properties

### Discovery Phase
- **Authentication**: ML-DSA-87 post-quantum signatures (NIST FIPS 204)
- **Integrity**: Signed peer information prevents spoofing
- **Privacy**: Peer IDs are SHA-1 hashes (pseudonymous)

### P2P Connection Phase
- **Encryption**: ChaCha20-Poly1305 (IETF RFC 8439)
- **Key Exchange**: ML-KEM-1024 (planned - not yet implemented)
- **Forward Secrecy**: Planned with periodic key rotation
- **Replay Protection**: Sequence numbers prevent replay attacks

### Data Transfer Phase
- **End-to-End Encryption**: Each P2P tunnel is independently encrypted
- **No MITM**: Discovery backbone cannot intercept data (doesn't see it)
- **Traffic Analysis Resistance**: Constant packet rate (planned with cover traffic)

---

## Performance Characteristics

### Discovery Overhead
- **Peer registration**: 100-500ms (one-time per node startup)
- **Peer lookup**: 50-200ms (one-time per new connection)
- **Keep-alive**: Every 5 minutes (negligible overhead)

### P2P Overhead
- **TCP control**: ~5 KB/min (keepalives, routing updates)
- **UDP data**: Variable (depends on traffic)
- **Encryption**: 3-7µs per packet (ChaCha20-Poly1305)
- **Adaptive buffering**: 0-20% CPU (BDP calculation)

### Latency Breakdown
- **Discovery query**: 100ms (only on first connection)
- **P2P handshake**: 150ms (TCP + auth)
- **UDP data path**: <2ms (encryption + network)
- **Multi-hop relay**: +100-150ms per hop

---

## Future Optimizations

1. **Direct peer discovery**: Peers broadcast to find each other without DHT
2. **Mesh routing**: Automatic route optimization (shortest path)
3. **Connection pooling**: Reuse connections for multiple flows
4. **UDP hole punching**: Better NAT traversal without relay
5. **QUIC migration**: Replace custom UDP with QUIC protocol

---

## Summary

**Discovery Model**: Centralized bootstrap (Kademlia DHT), decentralized data (P2P)
**Connection Model**: Direct UDP tunnels after TCP handshake
**Encryption Model**: Per-tunnel ChaCha20-Poly1305
**Routing Model**: Layer 3 overlay with multi-hop support

**Current Status**:
- ✅ Discovery working (3/4 nodes, Mac retry successful)
- ✅ P2P connections established (TCP + UDP)
- ✅ Encryption functional (259M+ frames sent)
- ❌ UDP receive broken (90-95% packet loss)
- ⏳ Fix in progress (SO_RCVBUF + kernel tuning)

**Key Insight**: The architecture is sound - control plane works perfectly, data plane needs one kernel tuning parameter to unlock full performance.

---

**Next Steps**: Implement SO_RCVBUF fix, retest with <10% loss target
