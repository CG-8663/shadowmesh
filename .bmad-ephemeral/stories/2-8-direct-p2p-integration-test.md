# Story 2.8: Direct P2P Integration Test

Status: drafted

## Story

As a ShadowMesh developer,
I want a working end-to-end integration test between two real machines,
so that I can validate the complete P2P tunnel functionality before releasing Epic 2.

## Acceptance Criteria

1. **Integration Binary** - Create executable that wires all Epic 2 components together into working P2P tunnel
2. **TAP Device Integration** - Binary creates TAP device, configures IP, captures/injects frames
3. **Encryption Pipeline Integration** - Frames from TAP → encryption pipeline → WebSocket transport
4. **WebSocket Transport** - TLS 1.3 secure WebSocket connection established between peers
5. **NAT Traversal** - Detect NAT type and attempt UDP hole punching for direct connection
6. **Decryption Pipeline** - Received frames → decryption → validation → TAP injection
7. **Two-Machine Test** - Test runs on two separate machines (not localhost simulation)
8. **End-to-End Validation** - Ping test demonstrates encrypted traffic flows through tunnel
9. **Documentation** - Setup instructions, expected output, troubleshooting guide
10. **User Sign-Off** - Project lead (james) validates working P2P tunnel between two locations

## Tasks / Subtasks

- [ ] **Task 1: Integration architecture design** (AC: #1, #2, #3, #4, #5, #6)
  - [ ] Design how all Epic 2 components wire together
  - [ ] Define configuration format for peer setup (peer A, peer B)
  - [ ] Plan connection establishment flow (discovery, handshake, tunnel)
  - [ ] Identify integration points between components
  - [ ] Create sequence diagram for P2P connection flow

- [ ] **Task 2: Integration binary implementation** (AC: #1)
  - [ ] Create `cmd/integration-test/main.go`
  - [ ] Implement peer configuration loading
  - [ ] Add command-line flags (--role=initiator|responder, --peer-addr, etc.)
  - [ ] Wire TAP device manager
  - [ ] Wire encryption pipeline
  - [ ] Wire WebSocket transport
  - [ ] Wire NAT detector and hole puncher
  - [ ] Implement connection state machine
  - [ ] Add comprehensive logging and diagnostics

- [ ] **Task 3: TAP device integration** (AC: #2)
  - [ ] Initialize TAP device on startup
  - [ ] Configure IP addresses (different for each peer)
  - [ ] Connect TAP read channel to encryption pipeline
  - [ ] Connect decryption pipeline to TAP write channel
  - [ ] Handle TAP device errors and cleanup

- [ ] **Task 4: Encryption/WebSocket pipeline** (AC: #3, #4, #6)
  - [ ] Initialize encryption pipeline with shared key
  - [ ] Connect encrypted frames to WebSocket sender
  - [ ] Connect WebSocket receiver to decryption pipeline
  - [ ] Implement WebSocket message framing
  - [ ] Handle pipeline backpressure and errors

- [ ] **Task 5: NAT traversal integration** (AC: #5)
  - [ ] Run NAT type detection on startup
  - [ ] Attempt UDP hole punching if NAT compatible
  - [ ] Fall back to direct TCP/WebSocket if hole punch fails
  - [ ] Log NAT traversal results

- [ ] **Task 6: Two-machine test scenario** (AC: #7, #8)
  - [ ] Create test procedure for two machines
  - [ ] Peer A: Initiator role (starts connection)
  - [ ] Peer B: Responder role (accepts connection)
  - [ ] Exchange IP addresses/endpoints
  - [ ] Establish tunnel
  - [ ] Run ping test (e.g., ping 10.0.0.2 from 10.0.0.1)
  - [ ] Verify encrypted traffic in WebSocket layer
  - [ ] Measure throughput and latency

- [ ] **Task 7: Documentation and validation** (AC: #9, #10)
  - [ ] Write setup instructions (prerequisites, configuration, execution)
  - [ ] Document expected output (logs, connection states)
  - [ ] Create troubleshooting guide (common failures, solutions)
  - [ ] Prepare test scenarios for user validation
  - [ ] Schedule validation session with james

## Dev Notes

### Epic 2 Component Integration Points

From the completed Epic 2 stories, the following components must be integrated:

**Story 2-1: TAP Device Management**
- `pkg/layer2/tap.go` - TAP device creation, configuration
- `TAPDevice` struct with `Read()` and `Write()` channels
- Requires root/elevated privileges

**Story 2-2: Ethernet Frame Capture**
- `pkg/layer2/frame.go` - EthernetFrame struct and serialization
- Frame validation and metrics tracking
- Integration: TAP device → Frame capture → Encryption pipeline

**Story 2-3: WebSocket Secure Transport**
- `client/daemon/direct_p2p.go` - DirectP2PManager with WebSocket client/server
- `client/daemon/tls.go` - TLS 1.3 certificate management
- Binary frame transmission over WSS
- Integration: Encrypted frames → WebSocket send/receive

**Story 2-4: NAT Type Detection**
- `pkg/nat/detector.go` - NATDetector with STUN protocol
- Detects: NoNAT, FullCone, RestrictedCone, PortRestrictedCone, Symmetric
- Caching and manual override support
- Integration: Run on startup to determine connection strategy

**Story 2-5: UDP Hole Punching**
- `pkg/nat/holepunch.go` - HolePuncher for NAT traversal
- 500ms timeout with relay fallback
- NAT feasibility check (Full/Restricted Cone only)
- Integration: Attempt before WebSocket fallback

**Story 2-6: Frame Encryption Pipeline**
- `pkg/crypto/frameencryption/pipeline.go` - EncryptionPipeline
- ChaCha20-Poly1305 AEAD encryption
- Goroutine-based with buffered channels
- Throughput: 345,720 fps (tested)
- Integration: TAP frames → Encrypt → WSS send, WSS receive → Decrypt → TAP inject

**Story 2-7: CLI Commands**
- `cmd/shadowmesh/main.go` - CLI commands (connect, disconnect, status)
- `client/daemon/api.go` - HTTP API for daemon communication
- Integration: Optional - can use API for status monitoring during test

### Architecture Constraints

**Security**:
- TLS 1.3 required for WebSocket connections
- ChaCha20-Poly1305 for frame encryption
- Certificate pinning for peer authentication
- Unique nonce per frame (replay protection)

**Performance Targets**:
- Frame encryption: >10,000 fps (achieved 345,720 fps)
- NAT detection: <2s (achieved 146ms)
- WebSocket latency: <10ms added overhead
- End-to-end tunnel latency: <50ms target

**Reliability**:
- Graceful shutdown on errors
- Context-based cancellation
- Resource cleanup (TAP device, goroutines, connections)
- Reconnection logic for dropped connections

### Testing Standards

**Integration Test Requirements**:
- Must run on two separate machines/VMs (not localhost)
- Real network between peers (not simulated)
- Actual NAT detection and traversal
- Genuine encrypted traffic over internet/network
- User (james) validates end-to-end functionality

**Test Scenarios**:
1. **Basic P2P Connection**:
   - Two machines on same LAN
   - Direct connection without NAT
   - Ping test validates tunnel

2. **NAT Traversal**:
   - One or both peers behind NAT
   - UDP hole punching attempt
   - Fallback to WebSocket if needed

3. **Throughput Test**:
   - Measure encrypted tunnel performance
   - Compare to performance benchmarks
   - Validate >10,000 fps under load

4. **Failure Scenarios**:
   - Network interruption and recovery
   - One peer crashes/restarts
   - Invalid encryption keys
   - TAP device failures

### Configuration Format

**Peer Configuration** (YAML or command-line flags):
```yaml
peer:
  role: initiator  # or responder
  local_ip: 10.0.0.1
  tap_device: tap0

remote_peer:
  address: 203.0.113.42:9001
  public_key: <ed25519 public key>

encryption:
  key: <32-byte ChaCha20-Poly1305 key>

nat:
  stun_server: stun.l.google.com:19302
  hole_punch_timeout: 500ms
```

### Integration Flow

```
[Peer A (Initiator)]                    [Peer B (Responder)]
        |                                        |
    1. Create TAP device (10.0.0.1)         Create TAP device (10.0.0.2)
        |                                        |
    2. Detect NAT type                      Detect NAT type
        |                                        |
    3. Attempt UDP hole punch ------>       Accept hole punch
        |                                        |
    4. Establish WebSocket (fallback)      Accept WebSocket
        |                                        |
    5. Start encryption pipeline            Start encryption pipeline
        |                                        |
    6. TAP frame → Encrypt → WSS --------> WSS → Decrypt → TAP
        |                                        |
    7. Ping 10.0.0.2 ------------------->  Respond to ping
        |                                        |
    8. WSS ← Encrypt ← TAP <-------------- TAP ← Decrypt ← WSS
        |                                        |
    9. Verify ping response (SUCCESS)
```

### Known Limitations from Retrospective

**Critical Finding**: Epic 2 marked "done" but Story 2-8 never implemented
- Zero manual testing of any Epic 2 components
- No integration validation performed
- Unit tests pass but real-world functionality unverified

**Risks**:
- TAP device creation may fail on real systems
- WebSocket connections may have integration issues
- NAT traversal may not work as expected
- Performance may differ from unit test benchmarks
- Components may not interoperate correctly

**Validation Required**:
- User (james) must test between two real locations
- Confirm encrypted traffic flows end-to-end
- Verify all Epic 2 components work together
- Sign-off required before Epic 2 completion

### References

- [Source: .bmad-ephemeral/stories/2-1-tap-device-management.md] - TAP device API
- [Source: .bmad-ephemeral/stories/2-2-ethernet-frame-capture.md] - Frame structures
- [Source: .bmad-ephemeral/stories/2-3-websocket-secure-wss-transport.md] - WebSocket transport
- [Source: .bmad-ephemeral/stories/2-4-nat-type-detection.md] - NAT detection API
- [Source: .bmad-ephemeral/stories/2-5-udp-hole-punching.md] - Hole punching API
- [Source: .bmad-ephemeral/stories/2-6-frame-encryption-pipeline.md] - Encryption pipeline API
- [Source: .bmad-ephemeral/stories/2-7-cli-commands-connect-disconnect-status.md] - CLI/API reference

## Dev Agent Record

### Context Reference

<!-- Will be added by story-context workflow -->

### Agent Model Used

claude-sonnet-4-5-20250929

### Debug Log References

### Completion Notes List

### File List
