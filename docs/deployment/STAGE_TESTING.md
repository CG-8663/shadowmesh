# ShadowMesh Stage Testing Guide

<img src="https://pbs.twimg.com/profile_images/1969957304679473152/QW21M-FO_400x400.jpg" alt="Chronara Group Logo" width="80" align="right"/>

**Chronara Group ShadowMesh - Stage Testing**

## Overview

This guide walks through setting up a local test environment to verify the complete client-relay handshake and frame routing functionality.

**What you'll test:**
- Post-quantum cryptographic handshake (ML-KEM-1024 + ML-DSA-87)
- WebSocket connection over TLS 1.3
- Frame encryption/decryption pipeline
- Broadcast routing between clients
- Heartbeat monitoring
- TAP device integration

## Prerequisites

- Go 1.21+ installed
- OpenSSL for certificate generation
- Root/sudo access (required for TAP devices)
- 3 terminal windows
- Linux or macOS

## Architecture

```
┌─────────────┐                  ┌─────────────┐
│   Client 1  │                  │   Client 2  │
│  (10.42.0.2)│                  │ (10.42.0.3) │
│             │                  │             │
│   tap0      │                  │   tap1      │
└──────┬──────┘                  └──────┬──────┘
       │                                │
       │  WebSocket/TLS 1.3            │
       │  (PQC Handshake)              │
       │                                │
       └────────────┬───────────────────┘
                    │
            ┌───────▼────────┐
            │  Relay Server  │
            │  (localhost:8443)│
            │                │
            │  Broadcasts    │
            │  frames to all │
            │  connected     │
            │  clients       │
            └────────────────┘
```

## Step 1: Build All Components

```bash
cd ~/Webcode/shadowmesh

# Build client and relay
make build

# Verify binaries
ls -lh build/
# Should see:
#   shadowmesh-client (7.2M)
#   shadowmesh-relay (7.1M)
```

## Step 2: Generate Test Certificates

```bash
# Generate self-signed certificates for TLS
./scripts/generate-test-certs.sh test-certs

# Output:
#   test-certs/ca-cert.pem      (CA certificate)
#   test-certs/ca-key.pem       (CA private key)
#   test-certs/relay-cert.pem   (Relay certificate)
#   test-certs/relay-key.pem    (Relay private key)
```

## Step 3: Configure Relay Server

Create relay configuration:

```bash
mkdir -p ~/.shadowmesh-relay
cat > ~/.shadowmesh-relay/config.yaml <<EOF
server:
  listen_addr: "0.0.0.0:8443"
  tls:
    enabled: true
    cert_file: "$PWD/test-certs/relay-cert.pem"
    key_file: "$PWD/test-certs/relay-key.pem"

limits:
  max_clients: 100
  handshake_timeout: 30
  heartbeat_interval: 30
  heartbeat_timeout: 90
  max_frame_size: 65536

identity:
  keys_dir: "$HOME/.shadowmesh-relay/keys"
  signing_key: "$HOME/.shadowmesh-relay/keys/signing_key.json"
  auto_generate: true

logging:
  level: "info"
  format: "text"
  output_file: ""
EOF

echo "✅ Relay configuration created"
```

Generate relay identity:

```bash
./build/shadowmesh-relay --gen-keys

# Output:
#   Generated new relay ID: 3a5f8e2d...
#   Signing key saved to: ~/.shadowmesh-relay/keys/signing_key.json
#   Relay ID file: ~/.shadowmesh-relay/keys/relay_id.txt
```

## Step 4: Configure Client 1

Create first client configuration:

```bash
mkdir -p ~/.shadowmesh-client1/keys
cat > ~/.shadowmesh-client1/config.yaml <<EOF
relay:
  url: "wss://localhost:8443/ws"
  reconnect_interval: 5s
  max_reconnect_attempts: 10
  heartbeat_interval: 30s
  insecure_skip_verify: true  # For testing with self-signed certs

tap:
  name: "tap0"
  mtu: 1500
  ip_addr: "10.42.0.2"
  netmask: "255.255.255.0"

crypto:
  enable_key_rotation: false
  key_rotation_interval: 1h

identity:
  keys_dir: "$HOME/.shadowmesh-client1/keys"
  private_key_file: "$HOME/.shadowmesh-client1/keys/signing_key.json"
  client_id_file: "$HOME/.shadowmesh-client1/keys/client_id.txt"

logging:
  level: "info"
  format: "text"
EOF

echo "✅ Client 1 configuration created"
```

Generate client 1 identity:

```bash
# We'll generate keys on first run, or manually:
# (The client will auto-generate on first run)
```

## Step 5: Configure Client 2

Create second client configuration:

```bash
mkdir -p ~/.shadowmesh-client2/keys
cat > ~/.shadowmesh-client2/config.yaml <<EOF
relay:
  url: "wss://localhost:8443/ws"
  reconnect_interval: 5s
  max_reconnect_attempts: 10
  heartbeat_interval: 30s
  insecure_skip_verify: true

tap:
  name: "tap1"
  mtu: 1500
  ip_addr: "10.42.0.3"
  netmask: "255.255.255.0"

crypto:
  enable_key_rotation: false
  key_rotation_interval: 1h

identity:
  keys_dir: "$HOME/.shadowmesh-client2/keys"
  private_key_file: "$HOME/.shadowmesh-client2/keys/signing_key.json"
  client_id_file: "$HOME/.shadowmesh-client2/keys/client_id.txt"

logging:
  level: "info"
  format: "text"
EOF

echo "✅ Client 2 configuration created"
```

## Step 6: Start Relay Server

**Terminal 1 - Relay Server:**

```bash
cd ~/Webcode/shadowmesh

# Start relay server (requires sudo for port <1024, optional for 8443)
sudo ./build/shadowmesh-relay

# Expected output:
# ╔═══════════════════════════════════════════════════════════╗
# ║         ShadowMesh Relay Server v0.1.0-alpha             ║
# ║     Post-Quantum Encrypted Private Network Relay          ║
# ╚═══════════════════════════════════════════════════════════╝
#
# 2025/11/01 17:30:00 Relay ID: 3a5f8e2d4c1b...
# 2025/11/01 17:30:00 Starting relay server on 0.0.0.0:8443
# 2025/11/01 17:30:00 Starting HTTPS server with TLS 1.3
# 2025/11/01 17:30:00 ShadowMesh Relay Server started successfully
```

Verify relay is listening:

```bash
# In another terminal:
curl -k https://localhost:8443/health

# Expected: {"status":"ok","active_clients":0}
```

## Step 7: Start Client 1

**Terminal 2 - Client 1:**

```bash
cd ~/Webcode/shadowmesh

# Start client 1 (requires sudo for TAP device)
sudo ./build/shadowmesh-client --config ~/.shadowmesh-client1/config.yaml

# Expected output:
# ╔═══════════════════════════════════════════════════════════╗
# ║         ShadowMesh Client v0.1.0-alpha                   ║
# ║     Post-Quantum Encrypted Private Network               ║
# ╚═══════════════════════════════════════════════════════════╝
#
# 2025/11/01 17:31:00 Loading configuration...
# 2025/11/01 17:31:00 Generating new client identity...
# 2025/11/01 17:31:00 Client ID: 7b2e9a4f...
# 2025/11/01 17:31:00 Creating TAP device: tap0
# 2025/11/01 17:31:00 TAP device created: tap0 (10.42.0.2/24)
# 2025/11/01 17:31:00 Connecting to relay: wss://localhost:8443/ws
# 2025/11/01 17:31:00 Connection established
# 2025/11/01 17:31:00 Starting handshake...
# 2025/11/01 17:31:00 Sent HELLO message
# 2025/11/01 17:31:00 Received CHALLENGE message (session: 9f3c2a1d...)
# 2025/11/01 17:31:01 Sent RESPONSE message
# 2025/11/01 17:31:01 Received ESTABLISHED message
# 2025/11/01 17:31:01 Handshake complete! Session established
# 2025/11/01 17:31:01 Starting tunnel...
```

**Back in Terminal 1 (Relay), you should see:**

```
2025/11/01 17:31:00 New connection from 127.0.0.1:xxxxx (total: 1, active: 1)
2025/11/01 17:31:00 Starting handshake with client from 127.0.0.1:xxxxx
2025/11/01 17:31:00 Received HELLO from client 7b2e9a4f...
2025/11/01 17:31:00 Sent CHALLENGE to client 7b2e9a4f (session: 9f3c2a1d...)
2025/11/01 17:31:01 Received RESPONSE from client 7b2e9a4f
2025/11/01 17:31:01 Sent ESTABLISHED to client 7b2e9a4f
2025/11/01 17:31:01 Handshake complete with client 7b2e9a4f (session: 9f3c2a1d...)
2025/11/01 17:31:01 Client 7b2e9a4f established (session: 9f3c2a1d...)
2025/11/01 17:31:01 Registered client 7b2e9a4f (total clients: 1)
```

## Step 8: Start Client 2

**Terminal 3 - Client 2:**

```bash
cd ~/Webcode/shadowmesh

# Start client 2
sudo ./build/shadowmesh-client --config ~/.shadowmesh-client2/config.yaml

# Expected output similar to Client 1
```

**Relay should now show 2 connected clients:**

```
2025/11/01 17:32:00 New connection from 127.0.0.1:xxxxx (total: 2, active: 2)
...
2025/11/01 17:32:01 Registered client 4d8c1f2a (total clients: 2)
```

## Step 9: Test Connectivity

### Test 1: Ping Between Clients

**From a new terminal:**

```bash
# Ping from Client 1 to Client 2
ping -I tap0 10.42.0.3

# Expected output:
# PING 10.42.0.3 (10.42.0.3) from 10.42.0.2 tap0: 56(84) bytes of data.
# 64 bytes from 10.42.0.3: icmp_seq=1 ttl=64 time=2.35 ms
# 64 bytes from 10.42.0.3: icmp_seq=2 ttl=64 time=1.98 ms
# 64 bytes from 10.42.0.3: icmp_seq=3 ttl=64 time=2.12 ms
```

**Watch relay logs** - you should see frame routing:

```
2025/11/01 17:33:00 Routing frame from client 7b2e9a4f (98 bytes)
2025/11/01 17:33:00 Broadcasting to 1 clients
2025/11/01 17:33:01 Routing frame from client 4d8c1f2a (98 bytes)
2025/11/01 17:33:01 Broadcasting to 1 clients
```

### Test 2: Check Network Interfaces

```bash
# On Client 1 machine:
ip addr show tap0

# Expected:
# tap0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500
#     inet 10.42.0.2/24 scope global tap0

# On Client 2 machine:
ip addr show tap1

# Expected:
# tap1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500
#     inet 10.42.0.3/24 scope global tap1
```

### Test 3: Monitor Statistics

**Check relay statistics:**

```bash
curl -k https://localhost:8443/stats

# Expected:
# {
#   "total_connections": 2,
#   "active_connections": 2,
#   "registered_clients": 2
# }
```

**Watch client statistics** (in client terminal output):

```
2025/11/01 17:35:00 Stats: tx_frames=127, rx_frames=95, session_age=4m0s
```

### Test 4: Heartbeat Monitoring

Wait 30 seconds and watch for heartbeat exchanges in all terminals:

**Client logs:**
```
2025/11/01 17:35:30 Sending heartbeat
2025/11/01 17:35:30 Received heartbeat ACK
```

**Relay logs:**
```
2025/11/01 17:35:30 Heartbeat from client 7b2e9a4f
```

## Step 10: Test Graceful Shutdown

### Test Client Reconnection

1. In Terminal 2 (Client 1), press `Ctrl+C`
2. Client should shut down gracefully
3. Relay should log disconnection:
   ```
   2025/11/01 17:36:00 Unregistered client 7b2e9a4f (remaining clients: 1)
   ```

4. Restart Client 1:
   ```bash
   sudo ./build/shadowmesh-client --config ~/.shadowmesh-client1/config.yaml
   ```

5. Client should reconnect and perform handshake again
6. Test ping again - should work!

### Test Relay Shutdown

1. In Terminal 1 (Relay), press `Ctrl+C`
2. Relay should shut down gracefully:
   ```
   ^C2025/11/01 17:37:00 Received signal: interrupt
   2025/11/01 17:37:00 Shutting down...
   2025/11/01 17:37:00 Relay server stopped
   2025/11/01 17:37:00 Goodbye!
   ```

3. Clients should log connection errors and attempt reconnection:
   ```
   2025/11/01 17:37:00 Connection error: websocket: close 1006
   2025/11/01 17:37:00 Attempting reconnection (1/10)...
   ```

4. Restart relay - clients should reconnect automatically!

## Expected Results

### ✅ Success Criteria

- [ ] Relay server starts without errors
- [ ] Both clients connect successfully
- [ ] Post-quantum handshake completes (4 messages exchanged)
- [ ] TAP devices created with correct IP addresses
- [ ] Ping works between Client 1 ↔ Client 2
- [ ] Frames are encrypted and routed through relay
- [ ] Heartbeats are exchanged every 30 seconds
- [ ] Statistics endpoints return data
- [ ] Graceful shutdown works for all components
- [ ] Automatic reconnection works after temporary failure

### ✅ What to Look For in Logs

**Successful Handshake:**
- Client: `HELLO → CHALLENGE → RESPONSE → ESTABLISHED`
- Relay: Processes all 4 messages, derives session keys

**Frame Routing:**
- Client: `TAP → Encrypt → WebSocket`
- Relay: `WebSocket → Broadcast → WebSocket`
- Client: `WebSocket → Decrypt → TAP`

**Heartbeat:**
- Every 30 seconds, both sides exchange heartbeat messages
- Client timeout: 90 seconds (3 missed heartbeats)

## Troubleshooting

### Issue: "Permission denied" creating TAP device

**Solution:**
```bash
# Run client with sudo
sudo ./build/shadowmesh-client --config ~/.shadowmesh-client1/config.yaml

# Or add CAP_NET_ADMIN capability:
sudo setcap cap_net_admin=eip ./build/shadowmesh-client
```

### Issue: "Connection refused" to relay

**Check:**
1. Relay is running: `curl -k https://localhost:8443/health`
2. Firewall allows port 8443: `sudo ufw allow 8443` (if using ufw)
3. Correct URL in client config: `wss://localhost:8443/ws`

### Issue: Handshake timeout

**Check:**
1. TLS certificates are valid (check relay logs)
2. Client config has `insecure_skip_verify: true` for self-signed certs
3. Increase `handshake_timeout` in relay config

### Issue: Ping doesn't work

**Check:**
1. Both clients connected: `curl -k https://localhost:8443/stats`
2. TAP devices have correct IPs: `ip addr show tap0` and `ip addr show tap1`
3. Routes are correct: `ip route show dev tap0`
4. Encryption keys derived: Check "Handshake complete" in logs

### Issue: "TAP device already exists"

**Solution:**
```bash
# Delete existing TAP devices
sudo ip link delete tap0
sudo ip link delete tap1

# Restart clients
```

## Performance Metrics

### Expected Latency

- Handshake: <500ms (includes PQC operations)
- Frame round-trip: <5ms (localhost)
- Heartbeat: 30s interval

### Expected Throughput

- Single frame: Up to MTU (1500 bytes)
- Burst rate: Limited by CPU encryption speed (~100-500 Mbps on single core)
- Relay capacity: 1000+ concurrent clients (configurable)

## Next Steps

After successful stage testing:

1. **Multi-hop Testing** - Add more relay nodes
2. **Load Testing** - Simulate 10+ clients
3. **Network Conditions** - Test with packet loss, latency
4. **Key Rotation** - Implement and test key rotation
5. **Direct Routing** - Implement MAC learning for efficiency
6. **Benchmarking** - Measure throughput and latency under load

## Clean Up

```bash
# Stop all processes (Ctrl+C in each terminal)

# Remove TAP devices
sudo ip link delete tap0 2>/dev/null || true
sudo ip link delete tap1 2>/dev/null || true

# Remove test certificates
rm -rf test-certs/

# Remove test configurations (optional)
rm -rf ~/.shadowmesh-client1/ ~/.shadowmesh-client2/ ~/.shadowmesh-relay/
```

## Summary

This stage test validates:
- ✅ Complete PQC handshake protocol
- ✅ TLS 1.3 encrypted transport
- ✅ Frame encryption with ChaCha20-Poly1305
- ✅ Broadcast routing between clients
- ✅ TAP device integration (Layer 2)
- ✅ Heartbeat and connection management
- ✅ Graceful shutdown and reconnection

**You now have a working ShadowMesh mesh network!** 🎉
