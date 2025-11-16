# ShadowMesh Daemon Architecture

**Status:** Design Document (Implementation Pending)
**Story:** 2-8 Direct P2P Integration Test
**Created:** 2025-11-16
**Purpose:** Architecture specification for production ShadowMesh daemon that integrates all Epic 2 components

---

## Executive Summary

The ShadowMesh daemon is the core client application that creates encrypted peer-to-peer tunnels between machines. It integrates all Epic 2 components (TAP devices, encryption pipeline, WebSocket transport, NAT traversal) into a unified service that users interact with via CLI commands.

**Key Requirements:**
- Run as background service (systemd or standalone)
- Manage complete P2P tunnel lifecycle
- Expose HTTP API for CLI control
- Handle connection state machine
- Provide monitoring and metrics
- Support two-machine validation testing

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    ShadowMesh Daemon Process                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐         ┌──────────────────┐            │
│  │   HTTP API       │         │  Config Manager  │            │
│  │  (Port 9090)     │◄────────┤  (YAML Config)   │            │
│  │                  │         │                  │            │
│  │ /connect         │         └────────┬─────────┘            │
│  │ /disconnect      │                  │                       │
│  │ /status          │                  │                       │
│  │ /health          │                  │                       │
│  └────────┬─────────┘                  │                       │
│           │                            │                       │
│           └───────────┬────────────────┘                       │
│                       │                                        │
│            ┌──────────▼──────────────┐                         │
│            │   Connection Manager    │                         │
│            │   (State Machine)       │                         │
│            │                         │                         │
│            │ States:                 │                         │
│            │  - Disconnected         │                         │
│            │  - Connecting           │                         │
│            │  - Connected            │                         │
│            │  - Error                │                         │
│            └──────────┬──────────────┘                         │
│                       │                                        │
│       ┌───────────────┼───────────────┐                        │
│       │               │               │                        │
│   ┌───▼────┐    ┌─────▼──────┐   ┌───▼──────┐                │
│   │  TAP   │    │ Encryption │   │   P2P    │                │
│   │ Device │    │  Pipeline  │   │ Manager  │                │
│   │Manager │    │            │   │(WebSocket│                │
│   │        │    │ChaCha20-   │   │   TLS)   │                │
│   │tap0    │    │Poly1305    │   │          │                │
│   └───┬────┘    └─────┬──────┘   └────┬─────┘                │
│       │               │               │                       │
│       │      ┌────────▼───────┐       │                       │
│       │      │  Frame Router  │       │                       │
│       │      │  (Data Flow)   │       │                       │
│       │      └────────┬───────┘       │                       │
│       │               │               │                       │
│       └───────────────┼───────────────┘                       │
│                       │                                       │
│                ┌──────▼───────┐                               │
│                │   Network    │                               │
│                │ (WebSocket)  │                               │
│                └──────────────┘                               │
└─────────────────────────────────────────────────────────────────┘
```

---

## Component Specifications

### 1. Main Entry Point (`cmd/shadowmesh-daemon/main.go`)

**Responsibilities:**
- Parse command-line flags and configuration file
- Initialize logging infrastructure
- Create and start all daemon components
- Handle OS signals (SIGTERM, SIGINT) for graceful shutdown
- Coordinate component lifecycle

**Configuration Sources (Priority Order):**
1. Command-line flags
2. Environment variables
3. Configuration file (`/etc/shadowmesh/daemon.yaml` or `~/.shadowmesh/config.yaml`)
4. Built-in defaults

**Startup Sequence:**
```go
1. Load configuration
2. Initialize logger
3. Create ConnectionManager
4. Create DaemonAPI (HTTP server)
5. Start API server (non-blocking)
6. Enter main event loop
7. Wait for shutdown signal
8. Graceful shutdown (reverse order)
```

---

### 2. Configuration Manager

**Configuration File Format (YAML):**

```yaml
# ShadowMesh Daemon Configuration

daemon:
  listen_address: "127.0.0.1:9090"  # HTTP API address
  log_level: "info"                  # debug, info, warn, error
  pid_file: "/var/run/shadowmesh.pid"

network:
  tap_device: "tap0"
  local_ip: "10.0.0.1/24"
  mtu: 1500

encryption:
  # Hex-encoded 32-byte ChaCha20-Poly1305 key
  # Generated on first run if not specified
  key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

  # Key rotation interval (future feature)
  # rotation_interval: "1h"

peer:
  # Static peer configuration (for testing)
  # In production, use bootstrap server or peer discovery
  address: ""           # e.g., "peer.example.com:9001"
  public_key: ""        # Ed25519 public key for peer authentication

nat:
  enabled: true
  stun_server: "stun.l.google.com:19302"
  detection_timeout: "5s"
  hole_punch_timeout: "500ms"

websocket:
  tls_cert: "/etc/shadowmesh/cert.pem"      # Auto-generated if missing
  tls_key: "/etc/shadowmesh/key.pem"
  keepalive_interval: "30s"
  read_timeout: "60s"
  write_timeout: "60s"

monitoring:
  metrics_enabled: true
  prometheus_port: 9091  # Future: Prometheus metrics endpoint
```

**Implementation:**
```go
type DaemonConfig struct {
    Daemon struct {
        ListenAddress string `yaml:"listen_address"`
        LogLevel      string `yaml:"log_level"`
        PIDFile       string `yaml:"pid_file"`
    } `yaml:"daemon"`

    Network struct {
        TAPDevice string `yaml:"tap_device"`
        LocalIP   string `yaml:"local_ip"`
        MTU       int    `yaml:"mtu"`
    } `yaml:"network"`

    Encryption struct {
        Key              string `yaml:"key"`
        RotationInterval string `yaml:"rotation_interval"`
    } `yaml:"encryption"`

    Peer struct {
        Address   string `yaml:"address"`
        PublicKey string `yaml:"public_key"`
    } `yaml:"peer"`

    NAT struct {
        Enabled           bool   `yaml:"enabled"`
        STUNServer        string `yaml:"stun_server"`
        DetectionTimeout  string `yaml:"detection_timeout"`
        HolePunchTimeout  string `yaml:"hole_punch_timeout"`
    } `yaml:"nat"`

    WebSocket struct {
        TLSCert           string `yaml:"tls_cert"`
        TLSKey            string `yaml:"tls_key"`
        KeepaliveInterval string `yaml:"keepalive_interval"`
        ReadTimeout       string `yaml:"read_timeout"`
        WriteTimeout      string `yaml:"write_timeout"`
    } `yaml:"websocket"`

    Monitoring struct {
        MetricsEnabled  bool `yaml:"metrics_enabled"`
        PrometheusPort  int  `yaml:"prometheus_port"`
    } `yaml:"monitoring"`
}
```

---

### 3. Connection Manager

**Purpose:** Central state machine coordinating all components for P2P connection lifecycle.

**State Machine:**

```
                    ┌──────────────┐
                    │ Disconnected │
                    └──────┬───────┘
                           │
                  connect(peer_id)
                           │
                    ┌──────▼──────┐
                    │ Connecting  │
                    └──────┬──────┘
                           │
                    ┌──────┴──────┐
                    │             │
            Success │             │ Failure
                    │             │
            ┌───────▼──┐      ┌───▼──────┐
            │Connected │      │  Error   │
            └───────┬──┘      └───┬──────┘
                    │             │
          disconnect()      retry/disconnect()
                    │             │
                    └──────┬──────┘
                           │
                    ┌──────▼───────┐
                    │ Disconnected │
                    └──────────────┘
```

**Implementation:**

```go
type ConnectionManager struct {
    // Configuration
    config *DaemonConfig

    // Components (managed lifecycle)
    tapDevice         *layer2.TAPDevice
    encryptionPipeline *frameencryption.EncryptionPipeline
    p2pManager        *daemon.DirectP2PManager
    natDetector       *nat.NATDetector
    holePuncher       *nat.HolePuncher
    daemonAPI         *daemon.DaemonAPI

    // State
    state          ConnectionState
    currentPeerID  string
    connectedAt    time.Time
    lastError      error

    // Data flow control
    frameRouter    *FrameRouter

    // Lifecycle
    ctx    context.Context
    cancel context.CancelFunc
    wg     sync.WaitGroup
    mu     sync.RWMutex
}

type ConnectionState string

const (
    StateDisconnected ConnectionState = "disconnected"
    StateConnecting   ConnectionState = "connecting"
    StateConnected    ConnectionState = "connected"
    StateError        ConnectionState = "error"
)

// API Methods
func (cm *ConnectionManager) Connect(peerID string) error
func (cm *ConnectionManager) Disconnect() error
func (cm *ConnectionManager) GetStatus() *ConnectionStatus
func (cm *ConnectionManager) Start() error
func (cm *ConnectionManager) Stop() error
```

**Connection Establishment Flow:**

```
1. Receive connect(peer_id) request
2. Validate not already connected
3. Transition to "connecting" state
4. Initialize TAP device
5. Start encryption pipeline
6. Perform NAT detection (optional)
7. Attempt UDP hole punching if NAT compatible
8. Establish WebSocket connection (direct or fallback)
9. Perform TLS handshake
10. Start frame router (data pipeline)
11. Transition to "connected" state
12. Notify API of success
```

**Disconnection Flow:**

```
1. Receive disconnect() request
2. Stop frame router
3. Close WebSocket connection
4. Stop encryption pipeline
5. Destroy TAP device
6. Transition to "disconnected" state
7. Clean up resources
8. Notify API of disconnection
```

---

### 4. Frame Router (Data Pipeline)

**Purpose:** Routes frames between TAP device, encryption pipeline, and network.

**Data Flow:**

```
Outbound (TAP → Network):
  TAP ReadChannel → EthernetFrame → Encryption Pipeline →
  Encrypted Frame → WebSocket Send

Inbound (Network → TAP):
  WebSocket Receive → Encrypted Frame → Decryption Pipeline →
  Plaintext Frame → TAP WriteChannel
```

**Implementation:**

```go
type FrameRouter struct {
    tapDevice         *layer2.TAPDevice
    encryptionPipeline *frameencryption.EncryptionPipeline
    p2pManager        *daemon.DirectP2PManager

    ctx    context.Context
    cancel context.CancelFunc
    wg     sync.WaitGroup

    // Metrics
    bytesIn  uint64
    bytesOut uint64
    framesIn uint64
    framesOut uint64
}

func (fr *FrameRouter) Start() error {
    // Start outbound goroutine (TAP → Network)
    fr.wg.Add(1)
    go fr.outboundLoop()

    // Start inbound goroutine (Network → TAP)
    fr.wg.Add(1)
    go fr.inboundLoop()

    return nil
}

func (fr *FrameRouter) outboundLoop() {
    defer fr.wg.Done()

    for {
        select {
        case <-fr.ctx.Done():
            return
        case frame := <-fr.tapDevice.ReadChannel():
            // Send frame to encryption pipeline
            if !fr.encryptionPipeline.SendFrame(frame) {
                // Pipeline full - drop frame
                continue
            }

            // Receive encrypted frame
            encFrame, err := fr.encryptionPipeline.ReceiveEncryptedFrame(fr.ctx)
            if err != nil {
                continue
            }

            // Send over WebSocket
            fr.p2pManager.SendFrame(encFrame)
            fr.framesOut++
        }
    }
}

func (fr *FrameRouter) inboundLoop() {
    defer fr.wg.Done()

    for {
        select {
        case <-fr.ctx.Done():
            return
        case encFrame := <-fr.p2pManager.ReceiveChannel():
            // Send to decryption pipeline
            if !fr.encryptionPipeline.SendEncryptedFrame(encFrame) {
                // Pipeline full - drop frame
                continue
            }

            // Receive decrypted frame
            plaintext, err := fr.encryptionPipeline.ReceiveDecryptedFrame(fr.ctx)
            if err != nil {
                // Invalid frame - drop
                continue
            }

            // Inject into TAP device
            fr.tapDevice.WriteChannel() <- plaintext
            fr.framesIn++
        }
    }
}
```

---

### 5. HTTP API Integration

The daemon integrates `DaemonAPI` from Story 2-7 (`client/daemon/api.go`).

**API Wiring:**

```go
func NewConnectionManager(config *DaemonConfig) *ConnectionManager {
    cm := &ConnectionManager{
        config: config,
        state: StateDisconnected,
    }

    // Create DaemonAPI
    api := daemon.NewDaemonAPI(config.Daemon.ListenAddress)

    // Wire API methods to ConnectionManager
    api.SetConnectionManager(cm)

    cm.daemonAPI = api

    return cm
}

// Implement API callback methods
func (cm *ConnectionManager) HandleConnect(peerID string) error {
    return cm.Connect(peerID)
}

func (cm *ConnectionManager) HandleDisconnect() error {
    return cm.Disconnect()
}

func (cm *ConnectionManager) HandleStatus() *daemon.StatusResponse {
    return cm.GetStatus()
}
```

---

## Component Integration Details

### Epic 2 Component Usage

**Story 2-1: TAP Device Management**
```go
import "github.com/shadowmesh/shadowmesh/pkg/layer2"

tap, err := layer2.NewTAPDevice(config.Network.TAPDevice)
tap.SetIPAddress(ipNet)
tap.Up()
tap.StartReading(ctx)
```

**Story 2-2: Ethernet Frame Capture**
```go
// Already integrated in TAP device
frameData := <-tap.ReadChannel()
frame, err := layer2.ParseEthernetFrame(frameData)
```

**Story 2-3: WebSocket Transport**
```go
import "github.com/shadowmesh/shadowmesh/client/daemon"

p2pMgr := daemon.NewDirectP2PManager(config, certMgr)
p2pMgr.ConnectToPeer(peerAddr)
```

**Story 2-4: NAT Detection**
```go
import "github.com/shadowmesh/shadowmesh/pkg/nat"

detector, _ := nat.NewNATDetector(config.NAT.STUNServer, timeout)
result, _ := detector.DetectNATType(ctx)
```

**Story 2-5: UDP Hole Punching**
```go
holePuncher := nat.NewHolePuncher(detector)
holePuncher.SetTimeout(config.NAT.HolePunchTimeout)
conn, err := holePuncher.EstablishConnection(candidates)
```

**Story 2-6: Frame Encryption Pipeline**
```go
import "github.com/shadowmesh/shadowmesh/pkg/crypto/frameencryption"

pipeline, _ := frameencryption.NewEncryptionPipeline(&frameencryption.PipelineConfig{
    Key:        encryptionKey,
    BufferSize: 100,
})
pipeline.Start()
```

**Story 2-7: CLI Commands**
```
# User executes (daemon must be running)
shadowmesh connect <peer-address>
shadowmesh status
shadowmesh disconnect
```

---

## Error Handling Strategy

**Principle:** Fail gracefully with clear error messages and automatic recovery where possible.

**Error Categories:**

1. **Configuration Errors** (Fatal - cannot start)
   - Invalid config file
   - Missing required parameters
   - Permission denied (e.g., TAP device creation)

2. **Connection Errors** (Recoverable - retry or fallback)
   - Network unreachable
   - Peer rejected connection
   - NAT traversal failed → fallback to relay (future)

3. **Runtime Errors** (Handled - log and continue)
   - Frame encryption/decryption failure → drop frame
   - WebSocket keepalive timeout → reconnect
   - TAP device read error → retry

**Error Handling Pattern:**

```go
func (cm *ConnectionManager) Connect(peerID string) error {
    // Validate state
    if cm.state != StateDisconnected {
        return ErrAlreadyConnected
    }

    // Attempt connection with error context
    cm.setState(StateConnecting)

    if err := cm.initializeComponents(); err != nil {
        cm.setState(StateError)
        cm.lastError = fmt.Errorf("component init failed: %w", err)
        return cm.lastError
    }

    if err := cm.establishP2PConnection(peerID); err != nil {
        cm.cleanup()
        cm.setState(StateError)
        cm.lastError = fmt.Errorf("P2P connection failed: %w", err)
        return cm.lastError
    }

    cm.setState(StateConnected)
    return nil
}
```

---

## Performance Targets

Based on Epic 2 component benchmarks:

**Throughput:**
- Frame encryption: 345,720 fps (measured in Story 2-6)
- Target tunnel throughput: >10,000 fps sustained
- Realistic expectation: 50,000-100,000 fps

**Latency:**
- TAP device: <1ms
- Encryption/Decryption: <0.1ms per frame
- WebSocket: <10ms network latency
- Total added overhead: <20ms

**Resource Usage:**
- Memory: <100 MB (daemon + components)
- CPU: <5% idle, <30% under load
- Network: Wire-speed encryption (limited by network, not crypto)

---

## Testing Strategy

### Unit Testing
- Each component already has >85% test coverage (from Epic 2 stories)
- Connection Manager state machine transitions
- Frame Router data flow logic

### Integration Testing
- Two-machine test (Story 2-8 acceptance criteria)
- Peer A (initiator) connects to Peer B (responder)
- Encrypted ping test validates end-to-end tunnel

### Failure Scenario Testing
- Network interruption and recovery
- Peer crash and reconnection
- Invalid encryption keys
- TAP device failures
- API request validation

---

## Systemd Service Integration

**Service File:** `/etc/systemd/system/shadowmesh.service`

```ini
[Unit]
Description=ShadowMesh Daemon - Post-Quantum P2P VPN
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/shadowmesh-daemon --config /etc/shadowmesh/daemon.yaml
Restart=on-failure
RestartSec=5s
User=root
Group=root

# Security hardening (future)
# PrivateTmp=true
# NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

**Usage:**
```bash
sudo systemctl start shadowmesh
sudo systemctl status shadowmesh
sudo systemctl enable shadowmesh  # Start on boot
sudo journalctl -u shadowmesh -f  # View logs
```

---

## Implementation Checklist

### Phase 1: Foundation (2 hours)
- [ ] Create `cmd/shadowmesh-daemon/main.go`
- [ ] Implement configuration loading (YAML parser)
- [ ] Set up logging infrastructure
- [ ] Add signal handling for graceful shutdown
- [ ] Create placeholder ConnectionManager

### Phase 2: Connection Manager (2 hours)
- [ ] Implement state machine (Disconnected → Connecting → Connected)
- [ ] Add Connect() method with component initialization
- [ ] Add Disconnect() method with cleanup
- [ ] Add GetStatus() for API integration
- [ ] Wire DaemonAPI callbacks

### Phase 3: Component Integration (3 hours)
- [ ] Initialize TAP device from config
- [ ] Initialize encryption pipeline with key
- [ ] Initialize P2P WebSocket manager
- [ ] Initialize NAT detector (optional)
- [ ] Wire components together
- [ ] Handle component lifecycle

### Phase 4: Frame Router (2 hours)
- [ ] Implement outbound loop (TAP → Encrypt → WSS)
- [ ] Implement inbound loop (WSS → Decrypt → TAP)
- [ ] Add metrics collection
- [ ] Handle backpressure and dropped frames
- [ ] Add error logging

### Phase 5: Testing & Documentation (1 hour)
- [ ] Build daemon binary
- [ ] Create example daemon.yaml config
- [ ] Write two-machine test procedure
- [ ] Test on localhost first
- [ ] Test between two real machines
- [ ] Document common issues and solutions

**Total Estimated Time: 10 hours (with breaks)**

---

## Known Limitations & Future Work

**Current Scope (Story 2-8):**
- Static peer configuration (no discovery)
- Single active connection (no multi-peer)
- No relay fallback (direct connections only)
- Basic metrics (no Prometheus export)
- Self-signed TLS certificates (no CA)

**Future Enhancements (Epic 3+):**
- Peer discovery via bootstrap server
- Multi-hop routing through relays
- Dynamic relay selection
- Prometheus metrics endpoint
- Certificate authority integration
- Automatic key rotation
- Connection pooling
- Traffic shaping

---

## References

- [Story 2-1: TAP Device Management](../.bmad-ephemeral/stories/2-1-tap-device-management.md)
- [Story 2-2: Ethernet Frame Capture](../.bmad-ephemeral/stories/2-2-ethernet-frame-capture.md)
- [Story 2-3: WebSocket Secure Transport](../.bmad-ephemeral/stories/2-3-websocket-secure-wss-transport.md)
- [Story 2-4: NAT Type Detection](../.bmad-ephemeral/stories/2-4-nat-type-detection.md)
- [Story 2-5: UDP Hole Punching](../.bmad-ephemeral/stories/2-5-udp-hole-punching.md)
- [Story 2-6: Frame Encryption Pipeline](../.bmad-ephemeral/stories/2-6-frame-encryption-pipeline.md)
- [Story 2-7: CLI Commands](../.bmad-ephemeral/stories/2-7-cli-commands-connect-disconnect-status.md)

---

**Next Steps:**
1. Review and approve architecture
2. Begin Phase 1 implementation
3. Test incrementally at each phase
4. Validate with two-machine test
5. User (james) sign-off on Epic 2
