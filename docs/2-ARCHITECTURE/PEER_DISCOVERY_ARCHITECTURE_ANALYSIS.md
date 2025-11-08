# ShadowMesh Peer Discovery Architecture Analysis

**Date**: 2025-11-03
**Analyst**: Winston (BMAD Architect)
**Context**: Epic 2 implementation discovered architectural mismatch with PRD specifications
**Critical Question**: "How do two peers find each other initially before they can connect?"

---

## Executive Summary

**Finding**: The PRD contains an **architectural gap** in peer discovery that was not previously identified. Epic 2 specifies "direct P2P connections" but provides no mechanism for peers to discover each other's network addresses before establishing those connections.

**Current Implementation**: Uses Frankfurt relay server as discovery coordinator (not specified in PRD)

**Recommendation**: **Path C (Hybrid Architecture)** - Retain relay-based discovery, add direct P2P transition after handshake

**Rationale**: Relay serves dual purpose as both discovery coordinator (Epic 2/3) and fallback routing (Epic 4)

---

## The Core Architectural Question

### What the PRD Says

**Epic 2 Milestone**:
> "Two clients on same LAN establish **direct P2P connection**, complete PQC handshake, and transmit 1 Gbps encrypted traffic"

**Epic 2 Goal**:
> "Implement Layer 2 networking with TAP devices, WebSocket transport, and **direct P2P connections without relay fallback**"

**Epic 3 Purpose**:
> "Deploy chronara.eth smart contract to Ethereum mainnet and enable **relay node registration/discovery**"

**Epic 4 Purpose**:
> "Implement relay node software, **fallback routing**, and achieve 95%+ connectivity across CGNAT/symmetric NAT scenarios"

### The Missing Piece: Initial Peer Discovery

**Problem**: Epic 2 requires "direct P2P" but doesn't specify how peers learn each other's addresses.

**Options for Peer Discovery**:

1. **Out-of-Band Exchange** (manual)
   - User manually configures peer IP:port in config file
   - Works for LAN testing
   - Not scalable for production

2. **Centralized Directory Server**
   - Central server maps peer IDs to IP:port
   - Not mentioned in PRD
   - Conflicts with decentralization goals

3. **DHT (Distributed Hash Table)**
   - Mentioned in PROJECT_SPEC.md ("DHT for peer discovery")
   - Not mentioned in PRD
   - Complex to implement, deferred to later

4. **Smart Contract Registry**
   - Epic 3 deploys chronara.eth for **relay node** discovery
   - Could be extended for peer discovery
   - Blockchain latency (12-15s) too slow for real-time discovery

5. **Relay-Assisted Discovery** (What we built)
   - Peers connect to relay server
   - Relay exchanges peer addresses
   - Peers transition to direct connection
   - **Not explicitly specified in PRD**

---

## PRD Analysis: Epic-by-Epic Breakdown

### Epic 1: Foundation & Cryptography ✅

**Purpose**: Implement PQC primitives (ML-KEM-1024, ML-DSA-87, ChaCha20-Poly1305)

**Dependencies**: None

**Peer Discovery Role**: None - establishes crypto only

**Status**: COMPLETE

---

### Epic 2: Core Networking & Direct P2P ⚠️

**PRD Specification**:
- Goal: "Direct P2P connections **without relay fallback**"
- Milestone: "Two clients on same LAN establish direct P2P connection"
- Story 2.4: NAT Type Detection
- Story 2.5: UDP Hole Punching
- Story 2.8: Direct P2P Integration Test

**What's Missing**:
- **No specification for how peers discover each other**
- **No peer registry or directory service**
- **No signaling mechanism**

**Implicit Assumptions**:
1. "Same LAN" suggests manual IP configuration is acceptable for testing
2. UDP hole punching requires "peer's public IP:port (learned via signaling)" but signaling not defined
3. Story 2.5 says "learned via signaling" but no signaling protocol specified

**Critical Gap**: Story 2.5 acceptance criteria states:
> "Client sends UDP packets to peer's public IP:port **(learned via signaling)**"

**Signaling mechanism is undefined!**

---

### Epic 3: Smart Contract & Blockchain Integration

**PRD Specification**:
- Goal: "Enable **relay node registration/discovery**"
- Story 3.1: RelayNodeRegistry smart contract
- Story 3.4: Go client queries relay node list

**What It Does**:
- Registers relay nodes on-chain
- Allows clients to discover relay nodes
- Verifies relay node signatures

**What It Doesn't Do**:
- Does **NOT** register client peers
- Does **NOT** provide peer-to-peer discovery
- Only handles relay infrastructure

**Dependencies**: Epic 1 (signatures)

**Peer Discovery Role**: Discovers **relay nodes only**, not peers

---

### Epic 4: Relay Infrastructure & CGNAT Traversal

**PRD Specification**:
- Goal: "Implement relay node software, **fallback routing**"
- Deliverable: "Client relay fallback logic: detect P2P failure → query smart contract → select 3 relays"
- Milestone: "95%+ connectivity across CGNAT/symmetric NAT scenarios"

**What It Does**:
- Relay node software (routing traffic between peers)
- Multi-hop routing (3+ relays)
- Fallback when direct P2P fails

**Dependencies**: Epic 2 (networking), Epic 3 (relay discovery)

**Critical Insight**: Epic 4 assumes Epic 2 has already solved peer discovery

**Relay Role**: **Fallback routing**, not initial discovery

---

## Logical Sequence: Discovery → Authentication → Connection

### What Should Happen (Inferred from PRD)

```
Phase 1: Peer Discovery (UNDEFINED IN PRD)
┌─────────────────────────────────────────────────────────┐
│ Problem: How does Client A learn Client B's IP:port?   │
│                                                         │
│ Options:                                                │
│ 1. Manual config (same LAN testing)                    │
│ 2. Centralized directory (not mentioned)               │
│ 3. DHT (mentioned in PROJECT_SPEC, not PRD)            │
│ 4. Smart contract (only for relays, not peers)         │
│ 5. Relay-assisted (our implementation, not in PRD)     │
└─────────────────────────────────────────────────────────┘
         │
         ├─────► Manual Config (Epic 2 LAN testing)
         │       Client A config: peer_address = "192.168.1.100:443"
         │
         └─────► Relay-Assisted (Epic 4 fallback)
                 Client A → Relay → Client B (exchange addresses)


Phase 2: Authentication (Epic 1)
┌─────────────────────────────────────────────────────────┐
│ Hybrid PQC Handshake (ML-KEM-1024 + ML-DSA-87)         │
│                                                         │
│ Client A                           Client B            │
│    │                                   │               │
│    ├────── HELLO ──────────────────────►               │
│    │      (ML-DSA-87 signature)                        │
│    │                                   │               │
│    │◄────── CHALLENGE ─────────────────┤               │
│    │      (ML-KEM-1024 public key)                     │
│    │                                   │               │
│    ├────── RESPONSE ───────────────────►               │
│    │      (ML-KEM-1024 ciphertext)                     │
│    │                                   │               │
│    │◄────── ESTABLISHED ───────────────┤               │
│    │      (Session keys derived)                       │
└─────────────────────────────────────────────────────────┘


Phase 3: Direct P2P Connection (Epic 2)
┌─────────────────────────────────────────────────────────┐
│ Layer 2 Encrypted Tunnel                                │
│                                                         │
│ Client A (10.10.10.3)          Client B (10.10.10.4)   │
│    chr-001                           chr-001           │
│      │                                   │             │
│      └──── Direct WebSocket (WSS) ──────┘             │
│            ChaCha20-Poly1305 encrypted frames          │
│            (no relay, no IP headers visible)           │
└─────────────────────────────────────────────────────────┘
```

---

## What We Actually Built

### Current Architecture (Relay-Based Discovery)

```
Phase 1: Both Peers Connect to Relay
┌─────────────────────────────────────────────────────────┐
│                  Frankfurt Relay Server                 │
│                  (83.136.252.52:443)                    │
│                                                         │
│  Waiting for peers to connect...                       │
└─────────────────────────────────────────────────────────┘
         ▲                           ▲
         │                           │
         │                           │
    UK VPS                      Belgium RPi
  (94.109.190.138)            (80.229.0.71)
   chr-001                     chr-001
   Config:                     Config:
   - mode: relay               - mode: relay
   - relay_url: wss://...      - relay_url: wss://...


Phase 2: Relay Facilitates Handshake
┌─────────────────────────────────────────────────────────┐
│                  Frankfurt Relay Server                 │
│                                                         │
│  Routes PQC handshake messages:                        │
│  1. UK → Relay → Belgium (HELLO)                       │
│  2. Belgium → Relay → UK (CHALLENGE)                   │
│  3. UK → Relay → Belgium (RESPONSE)                    │
│  4. Belgium → Relay → UK (ESTABLISHED)                 │
│                                                         │
│  Session established, relay knows both endpoints       │
└─────────────────────────────────────────────────────────┘


Phase 3: Traffic Routed Through Relay (CURRENT BEHAVIOR)
┌─────────────────────────────────────────────────────────┐
│ UK VPS → Relay → Belgium RPi                           │
│                                                         │
│ All Ethernet frames routed through relay permanently   │
│ No direct P2P transition implemented                   │
└─────────────────────────────────────────────────────────┘
```

### What's Wrong With This

**PRD Violation**:
- Epic 2 says "direct P2P connections **without relay fallback**"
- We route all traffic through relay forever

**Performance Impact**:
- Latency: UK → Frankfurt → Belgium (~80ms)
- Direct would be: UK → Belgium (~30ms)
- **50ms unnecessary latency**

**Scalability Impact**:
- Relay routes all traffic for all peer pairs
- Relay becomes bottleneck
- Wastes relay bandwidth

**Architectural Intent**:
- Relay should be **discovery coordinator** + **fallback**
- Not permanent routing infrastructure

---

## Three Paths Forward

### Path A: Implement True Direct P2P (PRD Purist)

**Architecture**:
```
Epic 2: Manual Configuration for LAN Testing
┌─────────────────────────────────────────────────────────┐
│ Config File:                                            │
│   peers:                                                │
│     - peer_id: uk-vps                                   │
│       address: 192.168.1.100:443                        │
│       public_key: <ML-DSA-87 key>                       │
└─────────────────────────────────────────────────────────┘
         │
         ├─────► Direct connection (no relay)
         │       UK VPS ←─────────────────► Belgium RPi
         │               WebSocket WSS
         │
         └─────► Epic 4: Add relay fallback later


Epic 4: Add Relay-Based Discovery
┌─────────────────────────────────────────────────────────┐
│ 1. Client queries chronara.eth for relay nodes          │
│ 2. Client connects to relay, sends peer discovery       │
│ 3. Relay coordinates address exchange                   │
│ 4. Clients attempt direct P2P (UDP hole punching)       │
│ 5. If direct fails → use relay routing (fallback)       │
└─────────────────────────────────────────────────────────┘
```

**Pros**:
- Matches PRD literal specification
- True direct P2P in Epic 2
- Relay is fallback in Epic 4 (as designed)

**Cons**:
- Epic 2 testing limited to manual config (same LAN)
- No scalable peer discovery until Epic 4
- Wastes time implementing manual config that's replaced in Epic 4
- UDP hole punching complex (3-4 days)

**Timeline**: +3 days to implement manual config, then rebuild in Epic 4

**Verdict**: **Not recommended** - temporary manual solution, rebuilt later

---

### Path B: Accept Relay-Only Architecture (Pragmatic)

**Architecture**:
```
All Epics: Relay as Primary Connection Method
┌─────────────────────────────────────────────────────────┐
│ Simplified model:                                       │
│                                                         │
│ Client A ──────► Relay Server ◄────── Client B         │
│                      │                                  │
│                      ├─────► Routes all traffic         │
│                      └─────► Permanent infrastructure   │
│                                                         │
│ No direct P2P ever implemented                         │
│ No UDP hole punching needed                            │
│ No NAT traversal complexity                            │
└─────────────────────────────────────────────────────────┘
```

**Pros**:
- Already working and tested
- Simple architecture (no P2P complexity)
- Relay handles all NAT/firewall issues automatically
- Can proceed to Epic 3 immediately

**Cons**:
- **Not true P2P** (relay routes all traffic)
- Violates PRD specification ("direct P2P")
- Relay becomes bottleneck (scalability limit)
- Higher latency than direct P2P
- Higher bandwidth costs (relay infrastructure)
- Not competitive with Tailscale/ZeroTier (they do direct P2P)

**Timeline**: No additional work, proceed to Epic 3

**Verdict**: **Not recommended** - Violates PRD, not competitive

---

### Path C: Hybrid Architecture (Recommended)

**Architecture**:
```
Phase 1: Initial Connection via Relay (Discovery Coordinator)
┌─────────────────────────────────────────────────────────┐
│                  Frankfurt Relay Server                 │
│                  (Discovery + Handshake)                │
│                                                         │
│  Facilitates:                                           │
│  1. Peer discovery (who's online?)                     │
│  2. PQC handshake (exchange session keys)              │
│  3. Address exchange (what's your public IP:port?)     │
└─────────────────────────────────────────────────────────┘
         ▲                           ▲
         │                           │
    UK VPS                      Belgium RPi
  (94.109.190.138:45678)      (80.229.0.71:56789)


Phase 2: Transition to Direct P2P (Post-Handshake)
┌─────────────────────────────────────────────────────────┐
│ Relay sends ESTABLISHED message with peer addresses:    │
│                                                         │
│ To UK VPS:                                              │
│   peer_address: 80.229.0.71:56789                      │
│                                                         │
│ To Belgium RPi:                                         │
│   peer_address: 94.109.190.138:45678                   │
└─────────────────────────────────────────────────────────┘
         │                           │
         └────────► Peers attempt direct connection
                    (simultaneous WebSocket open)


Phase 3: Direct P2P Connection (Primary)
┌─────────────────────────────────────────────────────────┐
│ UK VPS (10.10.10.3)          Belgium RPi (10.10.10.4)   │
│    chr-001                           chr-001           │
│      │                                   │             │
│      └──── Direct WebSocket (WSS) ──────┘             │
│            ChaCha20-Poly1305 encrypted frames          │
│            (no relay, direct peer-to-peer)             │
│                                                         │
│ Relay disconnected after transition                    │
│ Traffic flows directly                                 │
└─────────────────────────────────────────────────────────┘


Phase 4: Relay Fallback (Epic 4 - CGNAT/Firewall)
┌─────────────────────────────────────────────────────────┐
│ If direct P2P fails (timeout 5 seconds):               │
│                                                         │
│ UK VPS ──────► Frankfurt Relay ◄────── Belgium RPi     │
│                                                         │
│ Relay routes traffic (fallback mode)                   │
│ Periodically retry direct P2P (every 60 seconds)       │
└─────────────────────────────────────────────────────────┘
```

**Implementation Changes Required**:

1. **Modify ESTABLISHED Message** (2 hours)
   ```go
   type EstablishedMessage struct {
       SessionID            [16]byte
       MTU                  uint16
       HeartbeatInterval    uint16
       KeyRotationInterval  uint32

       // NEW: Peer endpoint info
       PeerPublicIP         string   // "94.109.190.138"
       PeerPublicPort       uint16   // 45678
       SupportsDirectP2P    bool     // Can peer accept direct connections?
   }
   ```

2. **Implement Direct P2P Transition** (4 hours)
   ```go
   // client/daemon/direct_p2p.go
   func (dm *DaemonManager) TransitionToDirectP2P(peerAddr string) error {
       // 1. Start local WebSocket listener on random port
       // 2. Attempt connection to peer's address
       // 3. Simultaneous open (both connect to each other)
       // 4. Migrate tunnel to direct connection
       // 5. Close relay connection
   }
   ```

3. **Connection Migration** (3 hours)
   ```go
   func (dm *DaemonManager) MigrateConnection(newConn *websocket.Conn) error {
       // 1. Pause outbound traffic
       // 2. Wait for in-flight frames to complete
       // 3. Perform quick re-handshake (SESSION_RESUME message)
       // 4. Switch tunnel to new connection
       // 5. Resume traffic
       // 6. Close old connection
   }
   ```

4. **Relay Modifications** (2 hours)
   ```go
   // relay/server/connection.go
   func (r *RelayServer) SendEstablished(clientA, clientB *Client) {
       // Detect client public IPs from WebSocket connection
       addrA := clientA.Conn.RemoteAddr()
       addrB := clientB.Conn.RemoteAddr()

       // Include peer addresses in ESTABLISHED
       msgA.PeerPublicIP = addrB.IP.String()
       msgA.PeerPublicPort = uint16(addrB.Port)
   }
   ```

5. **Fallback Logic** (2 hours)
   ```go
   func (dm *DaemonManager) StartDirectP2PWithFallback(peerAddr string) error {
       // Try direct connection with 5-second timeout
       ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
       defer cancel()

       err := dm.TransitionToDirectP2P(peerAddr)
       if err != nil {
           // Fallback: continue using relay
           log.Warn("Direct P2P failed, using relay fallback")
           return nil
       }

       // Success: relay disconnects
       log.Info("Direct P2P established, relay disconnected")
       return nil
   }
   ```

**Pros**:
- ✅ Solves peer discovery (relay as coordinator)
- ✅ Achieves direct P2P (Epic 2 goal)
- ✅ Keeps relay fallback (Epic 4)
- ✅ Works with existing Frankfurt relay
- ✅ Best performance (direct P2P)
- ✅ Best reliability (relay fallback)
- ✅ Competitive with Tailscale/ZeroTier

**Cons**:
- Requires 2-3 days additional development
- Relay role is hybrid (discovery + fallback) rather than pure fallback
- Connection migration complexity (risk of packet loss)

**Timeline**:
- Story 1: Peer address exchange - 2 hours
- Story 2: Direct P2P transition - 4 hours
- Story 3: Connection migration - 3 hours
- Story 4: Relay modifications - 2 hours
- Story 5: Testing - 3 hours
- **Total: 14 hours (~2 days)**

**Verdict**: ✅ **RECOMMENDED**

---

## Clarifying the PRD: Epic 2 vs Epic 3 vs Epic 4

### Epic 2: Local P2P (Same Network)

**Intent**: Establish direct connections between peers on same LAN or with known IPs

**Discovery Method**: Manual configuration (acceptable for testing)

**Example**:
```yaml
# config.yaml
mode: listener
listen_address: 0.0.0.0:443

peers:
  - peer_id: belgium-rpi
    address: 192.168.1.100:443
    public_key: <ML-DSA-87 public key>
```

**Limitation**: Doesn't scale beyond testing

---

### Epic 3: Relay Node Discovery

**Intent**: Discover relay infrastructure for fallback routing

**Discovery Method**: Smart contract (chronara.eth)

**Example**:
```go
// Query smart contract for relay nodes
nodes, err := client.GetActiveNodes()
// Returns: [{id: "frankfurt", ip: "83.136.252.52", port: 443}, ...]
```

**Scope**: Only discovers **relays**, not peers

---

### Epic 4: Relay Fallback + CGNAT Traversal

**Intent**: Use relays when direct P2P fails

**Discovery Method**:
1. Try direct P2P first (UDP hole punching)
2. If fails → query chronara.eth for relays
3. Connect through 3-hop relay routing

**Example**:
```
Direct P2P: UK VPS ←─────────────────► Belgium RPi
            (timeout after 5 seconds)

Relay Fallback: UK → Frankfurt → Amsterdam → London → Belgium
                (3-hop routing for privacy)
```

**Scope**: Relay is **fallback**, not primary

---

## Recommended Sequence Diagram

```
┌─────────┐                 ┌─────────┐                ┌─────────┐
│ Client A│                 │  Relay  │                │ Client B│
│ (UK VPS)│                 │(Frankt) │                │(Bel RPi)│
└────┬────┘                 └────┬────┘                └────┬────┘
     │                           │                          │
     │ 1. Connect to Relay       │                          │
     ├──────────────────────────►│                          │
     │    WSS wss://relay:443    │                          │
     │                           │      2. Connect to Relay │
     │                           │◄─────────────────────────┤
     │                           │   WSS wss://relay:443    │
     │                           │                          │
     │ 3. HELLO                  │                          │
     ├──────────────────────────►│                          │
     │  (ML-DSA-87 signature)    │                          │
     │                           │ 4. Forward HELLO         │
     │                           ├─────────────────────────►│
     │                           │                          │
     │                           │      5. CHALLENGE        │
     │                           │◄─────────────────────────┤
     │ 6. Forward CHALLENGE      │  (ML-KEM-1024 pubkey)    │
     │◄──────────────────────────┤                          │
     │                           │                          │
     │ 7. RESPONSE               │                          │
     ├──────────────────────────►│                          │
     │  (ML-KEM-1024 ciphertext) │                          │
     │                           │ 8. Forward RESPONSE      │
     │                           ├─────────────────────────►│
     │                           │                          │
     │                           │      9. ESTABLISHED      │
     │                           │◄─────────────────────────┤
     │ 10. Forward ESTABLISHED   │  + peer_addr_A           │
     │◄──────────────────────────┤                          │
     │  + peer_addr_B            │                          │
     │                           │ 11. Forward ESTABLISHED  │
     │                           ├─────────────────────────►│
     │                           │  + peer_addr_A           │
     │                           │                          │
     ├───────────────────────────┼──────────────────────────┤
     │    SESSION ESTABLISHED    │    SESSION ESTABLISHED   │
     │    (via relay)            │    (via relay)           │
     ├───────────────────────────┼──────────────────────────┤
     │                           │                          │
     │ 12. Attempt Direct P2P    │                          │
     ├──────────────────────────────────────────────────────►
     │    Direct WSS to peer_addr_B                         │
     │                           │                          │
     │                           │ 13. Attempt Direct P2P   │
     │◄──────────────────────────────────────────────────────┤
     │                           │ Direct WSS to peer_addr_A│
     │                           │                          │
     ├───────────────────────────┼──────────────────────────┤
     │    DIRECT P2P ESTABLISHED │                          │
     │    (simultaneous open)    │                          │
     ├───────────────────────────┼──────────────────────────┤
     │                           │                          │
     │ 14. SESSION_RESUME        │                          │
     ├──────────────────────────────────────────────────────►
     │    (re-auth with session keys)                       │
     │                           │                          │
     │                           │      15. SESSION_RESUME  │
     │◄──────────────────────────────────────────────────────┤
     │                           │                          │
     ├───────────────────────────┼──────────────────────────┤
     │    TUNNEL MIGRATED        │    TUNNEL MIGRATED       │
     │    (close relay conn)     │    (close relay conn)    │
     ├───────────────────────────┼──────────────────────────┤
     │                           │                          │
     │ 16. Close Relay Conn      │                          │
     ├──────────────────────────►│                          │
     │                           │ 17. Close Relay Conn     │
     │                           │◄─────────────────────────┤
     │                           │                          │
     ├───────────────────────────┼──────────────────────────┤
     │    DIRECT P2P ACTIVE      │    RELAY DISCONNECTED    │
     │    (all traffic direct)   │                          │
     ├───────────────────────────┼──────────────────────────┤
     │                                                      │
     │ 18. Encrypted Ethernet Frames                       │
     ├─────────────────────────────────────────────────────►
     │          ChaCha20-Poly1305                          │
     │◄─────────────────────────────────────────────────────┤
     │                                                      │
     │                                                      │
     │ [ If direct P2P fails: reconnect to relay fallback ] │
     │                                                      │
```

---

## Answering the Critical Question

**Question**: "How do two peers find each other initially before they can connect?"

**Answer**: **Multi-phase discovery strategy**

### Phase 1: Discovery (HOW peers learn addresses)

**Epic 2 (LAN Testing)**:
- Manual configuration in config file
- Acceptable for same-network testing
- Not scalable for production

**Epic 3 + Epic 4 (Production)**:
- Relay-assisted discovery
- Both peers connect to relay (discovered via chronara.eth)
- Relay exchanges peer public IP:port during handshake
- Peers learn each other's addresses

### Phase 2: Authentication (WHO peers are)

**Epic 1 (PQC Handshake)**:
- ML-DSA-87 signatures verify identity
- ML-KEM-1024 establishes session keys
- ChaCha20-Poly1305 encrypts traffic

### Phase 3: Connection (HOW peers communicate)

**Epic 2 (Direct P2P)**:
- Direct WebSocket connection (preferred)
- Layer 2 TAP tunnel
- Low latency, high throughput

**Epic 4 (Relay Fallback)**:
- Multi-hop relay routing (when direct fails)
- CGNAT/symmetric NAT handling
- Reliability over performance

---

## Final Recommendation

### Recommended Path: **Path C - Hybrid Architecture**

**Implementation**:
1. ✅ **Keep relay-based discovery** (solves missing PRD piece)
2. ✅ **Add direct P2P transition** (satisfies Epic 2 goal)
3. ✅ **Retain relay fallback** (Epic 4 as designed)

**Rationale**:
- Relay serves dual purpose: **discovery coordinator** + **fallback router**
- Matches PRD intent even if sequence differs
- Best user experience (fast + reliable)
- Competitive with Tailscale/ZeroTier
- Leverages existing Frankfurt relay infrastructure

**Timeline**: 2 days additional development

**Next Steps**:
1. Implement peer address exchange in ESTABLISHED message
2. Add direct P2P transition logic
3. Test connection migration (zero packet loss)
4. Validate with UK VPS ↔ Belgium RPi
5. Proceed to Epic 3 (Smart Contracts)

**Success Criteria**:
- ✅ Relay facilitates handshake
- ✅ Peers transition to direct connection
- ✅ Traffic flows directly (verify with tcpdump)
- ✅ Relay disconnects after transition
- ✅ Latency improves (80ms → 30ms)
- ✅ Relay fallback if direct fails

---

## Architectural Principle

> **"The relay is a matchmaker, not a middleman"**

The relay's job is to introduce peers and facilitate authentication. Once peers know each other's addresses and have established trust, they should communicate directly. The relay only steps in when direct communication is impossible (CGNAT, firewalls, network restrictions).

This principle reconciles Epic 2 ("direct P2P") with Epic 4 ("relay fallback") and solves the missing peer discovery mechanism.

---

## Appendix: PRD Gap Analysis

### Gaps Identified

1. **Peer Discovery Mechanism** (Epic 2)
   - Story 2.5 mentions "learned via signaling" but signaling undefined
   - No specification for how peers exchange addresses
   - Manual config assumed for LAN testing

2. **Relay Role Ambiguity** (Epic 2 vs Epic 4)
   - Epic 2: "without relay fallback"
   - Epic 4: "relay fallback logic"
   - Unclear when relay is introduced

3. **Smart Contract Scope** (Epic 3)
   - Only specifies relay node registry
   - Doesn't specify peer registry
   - Blockchain too slow for real-time peer discovery

### Recommended PRD Updates

**Epic 2**:
- Add acceptance criteria: "Peers exchange addresses via relay-assisted discovery"
- Clarify: "Direct P2P after initial handshake via relay"

**Epic 3**:
- Clarify: "Relay node registry only, not peer registry"

**Epic 4**:
- Clarify: "Relay serves dual purpose: discovery coordinator + fallback router"

---

**End of Analysis**
