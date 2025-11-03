# ShadowMesh Wire Protocol Specification v1.0

<img src="https://pbs.twimg.com/profile_images/1969957304679473152/QW21M-FO_400x400.jpg" alt="Chronara Group Logo" width="80" align="right"/>

**Chronara Group ShadowMesh - Protocol Specification v1.0**

## Overview

The Chronara Group ShadowMesh protocol is a post-quantum secure Decentralized Private Network (DPN) protocol operating at Layer 2 (Ethernet frames). It uses hybrid post-quantum cryptography for key exchange and signatures, with ChaCha20-Poly1305 for symmetric encryption.

## Transport Layer

**Primary Transport**: WebSocket over TLS 1.3
- Mimics HTTPS traffic for DPI evasion
- Binary frames (not text)
- Client initiates connection to relay server
- Persistent connection with automatic reconnection

**Fallback Transport**: Raw TCP with obfuscation (future)

## Message Structure

All messages use a common header format:

```
+----------------+----------------+----------------+----------------+
|  Version (1B)  |  Type (1B)     |  Flags (2B)    |  Length (4B)   |
+----------------+----------------+----------------+----------------+
|                         Payload (variable)                        |
+-------------------------------------------------------------------+
```

- **Version**: Protocol version (currently 0x01)
- **Type**: Message type (see Message Types below)
- **Flags**: Reserved for future use (set to 0x0000)
- **Length**: Payload length in bytes (big-endian uint32)
- **Payload**: Message-specific data

**Maximum Message Size**: 65,535 bytes (64 KB)

## Message Types

### Control Messages (0x00-0x0F)

| Type | Name | Direction | Description |
|------|------|-----------|-------------|
| 0x01 | HELLO | Client → Relay | Initiate handshake with PQC public keys |
| 0x02 | CHALLENGE | Relay → Client | Send encapsulated shared secret |
| 0x03 | RESPONSE | Client → Relay | Prove possession of shared secret |
| 0x04 | ESTABLISHED | Relay → Client | Handshake complete, session ready |
| 0x05 | HEARTBEAT | Bidirectional | Keep-alive message |
| 0x06 | HEARTBEAT_ACK | Bidirectional | Acknowledge heartbeat |
| 0x0E | ERROR | Bidirectional | Report error condition |
| 0x0F | CLOSE | Bidirectional | Graceful connection termination |

### Data Messages (0x10-0x1F)

| Type | Name | Direction | Description |
|------|------|-----------|-------------|
| 0x10 | DATA_FRAME | Bidirectional | Encrypted Ethernet frame |
| 0x11 | MULTI_HOP | Client → Relay | Multi-hop routing header |

### Management Messages (0x20-0x2F)

| Type | Name | Direction | Description |
|------|------|-----------|-------------|
| 0x20 | CONFIG_UPDATE | Relay → Client | Push configuration changes |
| 0x21 | STATS_REQUEST | Relay → Client | Request statistics |
| 0x22 | STATS_RESPONSE | Client → Relay | Report statistics |

## Handshake Protocol

The handshake establishes a shared secret using hybrid post-quantum key exchange (ML-KEM-1024 + X25519).

### Sequence Diagram

```
Client                                                  Relay
  |                                                       |
  |--- HELLO (KEM_PK_client, ECDH_PK_client, SIG) ------>|
  |                                                       |
  |<-- CHALLENGE (KEM_CT, ECDH_PK_relay, SIG) -----------|
  |                                                       |
  |--- RESPONSE (Proof = MAC(shared_secret, nonce)) ---->|
  |                                                       |
  |<-- ESTABLISHED (Config, session_id) -----------------|
  |                                                       |
```

### Message Details

#### 1. HELLO (Type 0x01)

**Payload**:
```
+-------------------------------------------------------------------+
| Client ID (32 bytes - Ed25519 public key hash)                    |
+-------------------------------------------------------------------+
| KEM Public Key (1,568 bytes - ML-KEM-1024)                        |
+-------------------------------------------------------------------+
| ECDH Public Key (32 bytes - X25519)                               |
+-------------------------------------------------------------------+
| Signature (4,627 bytes - ML-DSA-87)                               |
+-------------------------------------------------------------------+
| Classical Signature (64 bytes - Ed25519)                          |
+-------------------------------------------------------------------+
| Timestamp (8 bytes - Unix nanoseconds)                            |
+-------------------------------------------------------------------+
```

**Total Size**: ~6,331 bytes

**Signature covers**: ClientID || KEM_PK || ECDH_PK || Timestamp

#### 2. CHALLENGE (Type 0x02)

**Payload**:
```
+-------------------------------------------------------------------+
| Relay ID (32 bytes - Ed25519 public key hash)                     |
+-------------------------------------------------------------------+
| Session ID (16 bytes - random)                                    |
+-------------------------------------------------------------------+
| KEM Ciphertext (1,568 bytes - ML-KEM-1024 encapsulation)          |
+-------------------------------------------------------------------+
| ECDH Public Key (32 bytes - X25519)                               |
+-------------------------------------------------------------------+
| Nonce (24 bytes - for proof MAC)                                  |
+-------------------------------------------------------------------+
| Signature (4,627 bytes - ML-DSA-87)                               |
+-------------------------------------------------------------------+
| Classical Signature (64 bytes - Ed25519)                          |
+-------------------------------------------------------------------+
| Timestamp (8 bytes - Unix nanoseconds)                            |
+-------------------------------------------------------------------+
```

**Total Size**: ~6,371 bytes

**Signature covers**: RelayID || SessionID || KEM_CT || ECDH_PK || Nonce || Timestamp

**Shared Secret Derivation**:
```
kem_shared_secret = KEM.Decapsulate(KEM_CT, KEM_SK_client)
ecdh_shared_secret = X25519(ECDH_SK_client, ECDH_PK_relay)
shared_secret = HKDF-SHA256(kem_shared_secret || ecdh_shared_secret,
                            salt="ShadowMesh-v1-KDF",
                            info="handshake-master-secret",
                            length=32)
```

#### 3. RESPONSE (Type 0x03)

**Payload**:
```
+-------------------------------------------------------------------+
| Session ID (16 bytes - echoed from CHALLENGE)                     |
+-------------------------------------------------------------------+
| Proof (32 bytes - HMAC-SHA256(shared_secret, nonce))              |
+-------------------------------------------------------------------+
| Client Capabilities (4 bytes - bit flags)                         |
+-------------------------------------------------------------------+
```

**Total Size**: 52 bytes

**Capabilities Flags**:
- Bit 0: Multi-hop support
- Bit 1: Obfuscation support
- Bit 2: IPv6 support
- Bits 3-31: Reserved

#### 4. ESTABLISHED (Type 0x04)

**Payload**:
```
+-------------------------------------------------------------------+
| Session ID (16 bytes)                                             |
+-------------------------------------------------------------------+
| Server Capabilities (4 bytes - bit flags)                         |
+-------------------------------------------------------------------+
| Heartbeat Interval (4 bytes - seconds)                            |
+-------------------------------------------------------------------+
| MTU (2 bytes - Maximum Transmission Unit for DATA_FRAME)          |
+-------------------------------------------------------------------+
| Key Rotation Interval (4 bytes - seconds)                         |
+-------------------------------------------------------------------+
```

**Total Size**: 30 bytes

**After ESTABLISHED**: Both sides derive session keys:

```
tx_key = HKDF-SHA256(shared_secret,
                     salt="ShadowMesh-v1-TX",
                     info=session_id || client_id || relay_id,
                     length=32)

rx_key = HKDF-SHA256(shared_secret,
                     salt="ShadowMesh-v1-RX",
                     info=session_id || client_id || relay_id,
                     length=32)
```

## Data Transmission

### DATA_FRAME (Type 0x10)

Carries encrypted Ethernet frames from the TAP device.

**Payload** (encrypted with ChaCha20-Poly1305):
```
+-------------------------------------------------------------------+
| Counter (8 bytes - monotonic frame counter, big-endian)           |
+-------------------------------------------------------------------+
| Encrypted Frame (variable - Ethernet frame + Poly1305 tag)        |
+-------------------------------------------------------------------+
```

**Encryption**:
- **Key**: Session key (tx_key for sending, rx_key for receiving)
- **Nonce**: Derived from counter (12 bytes):
  - Bytes 0-7: Counter (big-endian)
  - Bytes 8-11: Session ID first 4 bytes
- **AAD**: Message header (8 bytes)
- **Tag**: Poly1305 (16 bytes, appended to ciphertext)

**Maximum Ethernet Frame Size**: 1,500 bytes (standard MTU)
**Maximum Payload Size**: 8 (counter) + 1,500 (frame) + 16 (tag) = 1,524 bytes

### Counter Management

- **Client**: Starts at 1, increments for each sent frame
- **Relay**: Starts at 1, increments for each sent frame
- **Separate counters** for each direction (full-duplex)
- **Replay protection**: Receiver tracks last seen counter, rejects lower values
- **Counter reset**: After key rotation (see below)

## Key Rotation

To maintain perfect forward secrecy, session keys rotate periodically.

**Default Interval**: 3600 seconds (1 hour)

**Rotation Protocol**:
1. At rotation time, client generates new KEM/ECDH keypair
2. Client sends new HELLO message (marked as rotation via flags)
3. Relay responds with new CHALLENGE
4. Client sends RESPONSE
5. Relay sends ESTABLISHED with new session parameters
6. **Seamless transition**: Old session remains active for 30s grace period
7. Both sides switch to new keys after receiving first DATA_FRAME with new key

**Flags for rotation** (in HELLO message):
- Bit 0 of Flags field: Set to 1 for rotation, 0 for initial handshake

## Heartbeat & Keepalive

**Purpose**: Detect dead connections, maintain NAT bindings

**Interval**: Configured by relay in ESTABLISHED message (default: 30s)

**Protocol**:
- Client sends HEARTBEAT every interval
- Relay responds with HEARTBEAT_ACK within 5 seconds
- If 3 consecutive heartbeats fail, connection is considered dead
- Automatic reconnection initiated by client

**Payload**: Empty (0 bytes)

## Error Handling

**ERROR Message Payload**:
```
+-------------------------------------------------------------------+
| Error Code (2 bytes)                                              |
+-------------------------------------------------------------------+
| Error Message (variable UTF-8 string)                             |
+-------------------------------------------------------------------+
```

**Error Codes**:
- 0x0001: Invalid protocol version
- 0x0002: Invalid message type
- 0x0003: Invalid signature
- 0x0004: Handshake timeout
- 0x0005: Decryption failure
- 0x0006: Replay attack detected
- 0x0007: Unsupported feature
- 0x0008: Rate limit exceeded
- 0x00FF: Internal server error

**Behavior**: After sending ERROR, connection is closed gracefully

## Connection Termination

**Graceful Shutdown**:
1. Either side sends CLOSE message
2. Flush any pending DATA_FRAME messages
3. Close WebSocket connection
4. Clear session keys from memory

**CLOSE Payload**:
```
+-------------------------------------------------------------------+
| Reason Code (2 bytes)                                             |
+-------------------------------------------------------------------+
| Reason String (variable UTF-8)                                    |
+-------------------------------------------------------------------+
```

**Reason Codes**:
- 0x0000: Normal shutdown
- 0x0001: Idle timeout
- 0x0002: Administrative shutdown
- 0x0003: Protocol violation

## Security Considerations

### Cryptographic Guarantees

- **Post-quantum security**: ML-KEM-1024 provides NIST Security Level 5
- **Hybrid security**: Falls back to X25519 if PQC is broken
- **Perfect forward secrecy**: Keys rotate, old sessions cannot be decrypted
- **Replay protection**: Monotonic counters prevent frame replay
- **Authentication**: Both client and relay authenticate via signatures

### Timing Attack Mitigation

- Constant-time crypto operations (provided by `circl` library)
- No early returns on verification failure
- Uniform error responses

### Denial of Service Protection

- Maximum message size enforced (64 KB)
- Rate limiting on HELLO messages (max 10/minute per IP)
- Handshake timeout (30 seconds)
- Memory limits on pending connections

### Obfuscation

- WebSocket over TLS appears as normal HTTPS traffic
- Random padding in messages (future)
- Traffic shaping to match common web patterns (future)

## Wire Format Examples

### HELLO Message (Hex)

```
01 01 00 00 00 00 18 B3
[Version=1, Type=1, Flags=0, Length=6,323]

[32 bytes: Client ID]
A1 B2 C3 ... (Ed25519 public key hash)

[1,568 bytes: ML-KEM-1024 public key]
04 05 06 ... (KEM public key)

[32 bytes: X25519 public key]
D1 E2 F3 ... (ECDH public key)

[4,627 bytes: ML-DSA-87 signature]
07 08 09 ... (Dilithium signature)

[64 bytes: Ed25519 signature]
0A 0B 0C ... (Ed25519 signature)

[8 bytes: Timestamp]
00 00 01 8C 3F A1 B2 C3 (Unix nanoseconds)
```

### DATA_FRAME Message (Hex)

```
01 10 00 00 00 00 05 F4
[Version=1, Type=16, Flags=0, Length=1,524]

[8 bytes: Counter]
00 00 00 00 00 00 00 01 (Counter = 1)

[1,516 bytes: Encrypted Ethernet frame + tag]
E1 F2 A3 ... (ChaCha20 ciphertext + Poly1305 tag)
```

## Implementation Notes

### Performance Targets

- **Latency overhead**: <2ms for encryption/decryption
- **Throughput**: 1+ Gbps on single CPU core
- **Memory**: <100 MB per connection
- **CPU**: <5% for 100 Mbps sustained traffic

### Recommended Libraries (Go)

- `github.com/cloudflare/circl` - Post-quantum crypto
- `golang.org/x/crypto` - Classical crypto
- `github.com/gorilla/websocket` - WebSocket transport
- `github.com/songgao/water` - TAP/TUN devices

### Testing Requirements

- Unit tests for all message serialization/deserialization
- Fuzz testing for message parsers
- Integration tests for full handshake
- Performance benchmarks for throughput/latency
- Security audit before production

## Version History

- **v1.0** (2025-11-01): Initial specification
  - Hybrid PQC handshake
  - ChaCha20-Poly1305 data encryption
  - Layer 2 Ethernet frame transport
  - Key rotation support
  - Heartbeat mechanism

## Future Extensions

- **v1.1**: Multi-hop routing protocol
- **v1.2**: Traffic obfuscation with cover traffic
- **v1.3**: UDP transport mode
- **v2.0**: Atomic clock synchronization protocol
