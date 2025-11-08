# ShadowMesh Peer Discovery and Connection Sequence Diagram

**Recommended Architecture**: Hybrid (Relay-Assisted Discovery + Direct P2P)

---

## Complete Connection Flow

```
┌──────────────┐           ┌──────────────┐           ┌──────────────┐
│  Client A    │           │    Relay     │           │  Client B    │
│  (UK VPS)    │           │  (Frankfurt) │           │ (Belgium RPi)│
│ 94.109.190   │           │ 83.136.252   │           │ 80.229.0.71  │
└──────┬───────┘           └──────┬───────┘           └──────┬───────┘
       │                          │                          │
       │                          │                          │

╔══════════════════════════════════════════════════════════════════════╗
║ PHASE 1: RELAY DISCOVERY (Epic 3)                                   ║
╚══════════════════════════════════════════════════════════════════════╝
       │                          │                          │
       │ 1. Query chronara.eth    │                          │
       │    for relay nodes       │                          │
       ├─────────────────────────►│                          │
       │                          │                          │
       │ 2. Return relay list     │                          │
       │    [{frankfurt}, ...]    │                          │
       │◄─────────────────────────┤                          │
       │                          │                          │
       │                          │  3. Query chronara.eth   │
       │                          │     for relay nodes      │
       │                          │◄─────────────────────────┤
       │                          │                          │
       │                          │  4. Return relay list    │
       │                          │     [{frankfurt}, ...]   │
       │                          ├─────────────────────────►│
       │                          │                          │

╔══════════════════════════════════════════════════════════════════════╗
║ PHASE 2: CONNECT TO RELAY (Epic 2 - Initial Connection)             ║
╚══════════════════════════════════════════════════════════════════════╝
       │                          │                          │
       │ 5. WebSocket Connect     │                          │
       │    WSS://relay:443       │                          │
       ├─────────────────────────►│                          │
       │                          │                          │
       │ 6. WS Upgrade (200 OK)   │                          │
       │    Sec-WebSocket-Accept  │                          │
       │◄─────────────────────────┤                          │
       │                          │                          │
       │    [Connected]           │                          │
       │                          │                          │
       │                          │  7. WebSocket Connect    │
       │                          │     WSS://relay:443      │
       │                          │◄─────────────────────────┤
       │                          │                          │
       │                          │  8. WS Upgrade (200 OK)  │
       │                          │    Sec-WebSocket-Accept  │
       │                          ├─────────────────────────►│
       │                          │                          │
       │                          │       [Connected]        │
       │                          │                          │

╔══════════════════════════════════════════════════════════════════════╗
║ PHASE 3: PQC HANDSHAKE via RELAY (Epic 1)                           ║
╚══════════════════════════════════════════════════════════════════════╝
       │                          │                          │
       │ 9. HELLO Message         │                          │
       │    - Peer ID             │                          │
       │    - ML-DSA-87 signature │                          │
       │    - Timestamp           │                          │
       ├─────────────────────────►│                          │
       │                          │                          │
       │                          │ 10. Forward HELLO        │
       │                          │     (relay routes msg)   │
       │                          ├─────────────────────────►│
       │                          │                          │
       │                          │                          │
       │                          │ 11. CHALLENGE Message    │
       │                          │     - ML-KEM-1024 pubkey │
       │                          │     - Nonce              │
       │                          │◄─────────────────────────┤
       │                          │                          │
       │ 12. Forward CHALLENGE    │                          │
       │     (relay routes msg)   │                          │
       │◄─────────────────────────┤                          │
       │                          │                          │
       │                          │                          │
       │ 13. RESPONSE Message     │                          │
       │     - ML-KEM ciphertext  │                          │
       │     - Encrypted payload  │                          │
       ├─────────────────────────►│                          │
       │                          │                          │
       │                          │ 14. Forward RESPONSE     │
       │                          │     (relay routes msg)   │
       │                          ├─────────────────────────►│
       │                          │                          │
       │                          │                          │
       │                          │ 15. ESTABLISHED Message  │
       │                          │     - Session ID         │
       │                          │     - Session keys OK    │
       │                          │     - MTU: 1500          │
       │                          │     - Heartbeat: 30s     │
       │                          │     + peer_addr_A: ...   │  <-- NEW
       │                          │◄─────────────────────────┤
       │                          │                          │
       │ 16. Forward ESTABLISHED  │                          │
       │     + peer_addr_B:       │                          │
       │       80.229.0.71:56789  │  <-- NEW                 │
       │◄─────────────────────────┤                          │
       │                          │                          │
       │                          │ 17. Forward ESTABLISHED  │
       │                          │     + peer_addr_A:       │
       │                          │       94.109.190:45678   │  <-- NEW
       │                          ├─────────────────────────►│
       │                          │                          │
       │                          │                          │

╔══════════════════════════════════════════════════════════════════════╗
║ [Session Established via Relay - Keys Derived]                      ║
║ Both clients now know:                                               ║
║ - Session keys (ChaCha20-Poly1305)                                   ║
║ - Peer's public IP:port                                              ║
║ - Peer supports direct P2P                                           ║
╚══════════════════════════════════════════════════════════════════════╝
       │                          │                          │

╔══════════════════════════════════════════════════════════════════════╗
║ PHASE 4: TRANSITION TO DIRECT P2P (NEW - Epic 2 Enhancement)        ║
╚══════════════════════════════════════════════════════════════════════╝
       │                          │                          │
       │ 18. Start local WS       │                          │
       │     listener :45678      │                          │
       │     (random port)        │                          │
       │                          │                          │
       │                          │  19. Start local WS      │
       │                          │      listener :56789     │
       │                          │      (random port)       │
       │                          │                          │
       │                          │                          │
       │ 20. Direct WSS Connect   │                          │
       │     to 80.229.0.71:56789 │                          │
       ├──────────────────────────┼─────────────────────────►│
       │                          │                          │
       │                          │                          │
       │                          │  21. Direct WSS Connect  │
       │                          │      to 94.109.190:45678 │
       │◄─────────────────────────┼──────────────────────────┤
       │                          │                          │
       │                          │                          │
       │ 22. WS Upgrade (200 OK)  │                          │
       │     [Simultaneous Open]  │                          │
       ├──────────────────────────┼─────────────────────────►│
       │                          │                          │
       │                          │  23. WS Upgrade (200 OK) │
       │                          │      [Simultaneous Open] │
       │◄─────────────────────────┼──────────────────────────┤
       │                          │                          │

╔══════════════════════════════════════════════════════════════════════╗
║ [Direct P2P Connection Established]                                  ║
╚══════════════════════════════════════════════════════════════════════╝
       │                          │                          │
       │ 24. SESSION_RESUME       │                          │
       │     (re-auth with        │                          │
       │      existing session    │                          │
       │      keys)               │                          │
       ├──────────────────────────┼─────────────────────────►│
       │                          │                          │
       │                          │  25. SESSION_RESUME      │
       │                          │      (confirm session)   │
       │◄─────────────────────────┼──────────────────────────┤
       │                          │                          │

╔══════════════════════════════════════════════════════════════════════╗
║ PHASE 5: MIGRATE TUNNEL TO DIRECT CONNECTION                        ║
╚══════════════════════════════════════════════════════════════════════╝
       │                          │                          │
       │ 26. Pause outbound       │                          │
       │     traffic on relay     │                          │
       │                          │                          │
       │                          │  27. Pause outbound      │
       │                          │      traffic on relay    │
       │                          │                          │
       │ 28. Wait for in-flight   │                          │
       │     frames to complete   │                          │
       │                          │                          │
       │                          │  29. Wait for in-flight  │
       │                          │      frames to complete  │
       │                          │                          │
       │ 30. Switch tunnel to     │                          │
       │     direct connection    │                          │
       │                          │                          │
       │                          │  31. Switch tunnel to    │
       │                          │      direct connection   │
       │                          │                          │
       │ 32. Close relay WS       │                          │
       ├─────────────────────────►│                          │
       │                          │                          │
       │                          │  33. Close relay WS      │
       │                          │◄─────────────────────────┤
       │                          │                          │
       │ 34. Relay WS Close       │                          │
       │     (graceful)           │                          │
       │◄─────────────────────────┤                          │
       │                          │                          │
       │                          │  35. Relay WS Close      │
       │                          │      (graceful)          │
       │                          ├─────────────────────────►│
       │                          │                          │

╔══════════════════════════════════════════════════════════════════════╗
║ [Relay Disconnected - Direct P2P Active]                             ║
╚══════════════════════════════════════════════════════════════════════╝
       │                                                     │
       │                                                     │

╔══════════════════════════════════════════════════════════════════════╗
║ PHASE 6: DIRECT P2P TRAFFIC (Epic 2 Goal Achieved)                  ║
╚══════════════════════════════════════════════════════════════════════╝
       │                                                     │
       │ 36. Ethernet Frame (TAP chr-001)                   │
       │     Src: 10.10.10.3 → Dst: 10.10.10.4             │
       │     [ICMP Echo Request]                            │
       │                                                     │
       │ 37. Encrypt with ChaCha20-Poly1305                 │
       │     (session keys from Epic 1)                     │
       │                                                     │
       │ 38. Protocol Encode (EncryptedFrame message)       │
       │     [Binary WebSocket Frame]                       │
       ├─────────────────────────────────────────────────────►
       │                    DIRECT CONNECTION                │
       │              (no relay in the middle!)              │
       │                                                     │
       │                                                     │
       │                 39. Receive WS Frame                │
       │                     (binary)                        │
       │                                                     │
       │                 40. Protocol Decode                 │
       │                     (EncryptedFrame)                │
       │                                                     │
       │                 41. Decrypt ChaCha20-Poly1305       │
       │                     (validate auth tag)             │
       │                                                     │
       │                 42. TAP Write (inject frame)        │
       │                     chr-001 receives frame          │
       │                     [ICMP Echo Request arrives]     │
       │                                                     │
       │                                                     │
       │                 43. Process ICMP                    │
       │                 44. Generate Echo Reply             │
       │                                                     │
       │ 45. Return Traffic (Encrypted Frame)                │
       │◄─────────────────────────────────────────────────────┤
       │                    DIRECT CONNECTION                │
       │                                                     │
       │                                                     │

╔══════════════════════════════════════════════════════════════════════╗
║ [Continuous Direct P2P Traffic]                                      ║
║ - Latency: ~30ms (UK → Belgium direct)                               ║
║ - Throughput: 1+ Gbps (1500 byte MTU)                                ║
║ - No relay overhead                                                  ║
║ - Layer 2 tunnel active                                              ║
╚══════════════════════════════════════════════════════════════════════╝
       │                                                     │
       ▼                                                     ▼


═══════════════════════════════════════════════════════════════════════

ALTERNATIVE FLOW: RELAY FALLBACK (Epic 4)

If direct P2P connection fails (timeout after 5 seconds):

╔══════════════════════════════════════════════════════════════════════╗
║ FALLBACK: CONTINUE USING RELAY (Epic 4)                             ║
╚══════════════════════════════════════════════════════════════════════╝

       │                          │                          │
       │ Direct P2P timeout       │                          │
       │ (5 seconds)              │                          │
       │                          │                          │
       │ Continue using           │                          │
       │ relay connection         │                          │
       │                          │                          │
       │ Encrypted Frame          │                          │
       ├─────────────────────────►│                          │
       │                          │                          │
       │                          │ Forward Frame            │
       │                          ├─────────────────────────►│
       │                          │                          │
       │                          │ Encrypted Frame          │
       │                          │◄─────────────────────────┤
       │                          │                          │
       │ Forward Frame            │                          │
       │◄─────────────────────────┤                          │
       │                          │                          │

╔══════════════════════════════════════════════════════════════════════╗
║ [Relay Routing Active - Fallback Mode]                               ║
║ - Latency: ~80ms (UK → Frankfurt → Belgium)                          ║
║ - Reliable but slower                                                ║
║ - Retry direct P2P every 60 seconds                                  ║
╚══════════════════════════════════════════════════════════════════════╝
       │                          │                          │
       ▼                          ▼                          ▼
```

---

## Timeline Analysis

### Phase 1: Relay Discovery (Epic 3)
- **Time**: 2-5 seconds (blockchain query)
- **Purpose**: Find available relay nodes
- **Output**: List of relay IPs/ports

### Phase 2: Connect to Relay (Epic 2)
- **Time**: 100-200ms per client (WebSocket handshake)
- **Purpose**: Establish connection to discovery coordinator
- **Output**: Both clients connected to relay

### Phase 3: PQC Handshake via Relay (Epic 1)
- **Time**: 300-500ms (4 round trips through relay)
- **Purpose**: Authenticate peers, derive session keys
- **Output**: Encrypted session established, peer addresses exchanged

### Phase 4: Transition to Direct P2P (NEW)
- **Time**: 500ms - 5 seconds (simultaneous open)
- **Purpose**: Establish direct connection
- **Success Case**: Direct P2P established → proceed to Phase 5
- **Failure Case**: Timeout → fallback to relay routing

### Phase 5: Migrate Tunnel (NEW)
- **Time**: 100-200ms (zero packet loss)
- **Purpose**: Seamlessly switch from relay to direct
- **Output**: Relay disconnected, direct P2P active

### Phase 6: Direct P2P Traffic (Epic 2 Goal)
- **Latency**: ~30ms (direct UK → Belgium)
- **Throughput**: 1+ Gbps
- **Relay**: Disconnected (not in the path)

---

## Performance Comparison

### Before (Relay-Only)
```
UK VPS → Frankfurt Relay → Belgium RPi
  20ms        40ms           20ms
         Total: 80ms latency

Relay bandwidth: 2x traffic (in + out)
```

### After (Direct P2P)
```
UK VPS ←───────────────────► Belgium RPi
            30ms
         Total: 30ms latency

Relay bandwidth: Only during handshake (~10KB)
```

**Improvement**: 62% latency reduction, 99% relay bandwidth savings

---

## Failure Scenarios

### Scenario 1: Direct P2P Fails (Firewall)

```
Client A                     Client B
   │                            │
   ├─── Direct WS (timeout) ───►│
   │         ✗ BLOCKED          │
   │                            │
   └─── Continue via relay ─────┘
        ✓ FALLBACK WORKS
```

**Result**: Relay routing (Epic 4 fallback)

---

### Scenario 2: Symmetric NAT

```
Client A (Symmetric NAT)     Client B
   │                            │
   ├─── UDP hole punch fails ──►│
   │    (port changes)          │
   │                            │
   └─── Continue via relay ─────┘
        ✓ FALLBACK WORKS
```

**Result**: Relay routing (Epic 4 fallback)

---

### Scenario 3: Both Behind CGNAT

```
Client A (CGNAT)             Client B (CGNAT)
   │                            │
   ├─── Direct WS (timeout) ───►│
   │    (no public IP)          │
   │                            │
   └─── Multi-hop relay ────────┘
        ✓ 3-HOP RELAY (Epic 4)
```

**Result**: Multi-hop relay routing (Epic 4)

---

## Message Definitions

### New Message: SESSION_RESUME

```go
type SessionResumeMessage struct {
    SessionID      [16]byte
    Nonce          [24]byte
    Timestamp      uint64
    Signature      []byte   // ML-DSA-87 signature
}
```

**Purpose**: Re-authenticate on direct connection using existing session keys

---

### Modified Message: ESTABLISHED

```go
type EstablishedMessage struct {
    SessionID            [16]byte
    MTU                  uint16
    HeartbeatInterval    uint16
    KeyRotationInterval  uint32

    // NEW FIELDS for direct P2P
    PeerPublicIP         string   // "94.109.190.138"
    PeerPublicPort       uint16   // 45678
    SupportsDirectP2P    bool     // true if peer can accept direct
}
```

**Purpose**: Include peer network addresses for direct P2P transition

---

## Implementation Files

### Files to Create
- `client/daemon/direct_p2p.go` - Direct P2P transition manager
- `client/daemon/connection_migration.go` - Tunnel migration logic

### Files to Modify
- `shared/protocol/messages.go` - Add peer address fields to ESTABLISHED
- `relay/server/connection.go` - Detect and include peer IPs
- `client/daemon/daemon.go` - Orchestrate transition

---

**End of Sequence Diagram**
