# ShadowMesh Direct P2P Test Guide

## Overview

This guide explains how to test the direct peer-to-peer (P2P) encrypted tunnel between:
- **UK VPS** (listener) - chr-001 TAP device at 10.100.0.1
- **Belgium Raspberry Pi** (connector) - chr-001 TAP device at 10.100.0.2

This test validates **Epic 2: Core Networking & Direct P2P** without using the relay server.

## Test Architecture

```
UK VPS (Listener)                    Belgium Raspberry Pi (Connector)
┌──────────────────────┐             ┌──────────────────────┐
│  chr-001: 10.100.0.1 │◄───────────►│  chr-001: 10.100.0.2 │
│                      │   WSS P2P   │                      │
│  shadowmesh-daemon   │   Direct    │  shadowmesh-daemon   │
│  (listener mode)     │   Tunnel    │  (connector mode)    │
│                      │             │                      │
│  Listens: 0.0.0.0:8443             │  Connects to: VPS:8443
│  ML-KEM-1024         │             │  ML-KEM-1024         │
│  ChaCha20-Poly1305   │             │  ChaCha20-Poly1305   │
└──────────────────────┘             └──────────────────────┘
```

**Data Flow**: Application → TAP → Encrypt → WSS → Decrypt → TAP → Application

## Prerequisites

### On Your Local Machine

1. **Build the binaries**:
   ```bash
   cd /Users/jamestervit/Webcode/shadowmesh
   make build
   ```

2. **Configure deployment script**:
   Edit `scripts/deploy-p2p-test.sh` and set:
   - `UK_VPS_IP` - Your UK VPS public IP address
   - `UK_VPS_USER` - SSH username for UK VPS
   - `RPI_IP` - Belgium Raspberry Pi IP address
   - `RPI_USER` - SSH username for RPi (default: pi)

3. **Configure UK VPS listener**:
   Edit `configs/vps-uk-listener.yaml` if needed:
   - Default listen address: `0.0.0.0:8443`
   - Default TAP IP: `10.100.0.1/24`
   - TAP device: `chr-001`

4. **Configure Belgium RPi connector**:
   Edit `configs/rpi-belgium-connector.yaml`:
   - Set `peer_address` to `<UK_VPS_IP>:8443`
   - Default TAP IP: `10.100.0.2/24`
   - TAP device: `chr-001`

### On Both Machines (UK VPS & Belgium RPi)

1. **Install TAP driver** (if not already installed):

   **Linux**:
   ```bash
   # Ubuntu/Debian
   sudo apt-get update
   sudo apt-get install -y uml-utilities

   # Check if TAP module is loaded
   lsmod | grep tun

   # Load if needed
   sudo modprobe tun
   ```

   **Raspberry Pi OS**:
   ```bash
   # Usually pre-installed, but verify
   sudo modprobe tun
   ```

2. **Open firewall** (UK VPS only - for incoming connections):
   ```bash
   # UFW
   sudo ufw allow 8443/tcp

   # iptables
   sudo iptables -A INPUT -p tcp --dport 8443 -j ACCEPT
   sudo iptables-save
   ```

## Deployment

### Automated Deployment

Run the deployment script from your local machine:

```bash
cd /Users/jamestervit/Webcode/shadowmesh
./scripts/deploy-p2p-test.sh
```

This will:
1. Build binaries if needed
2. Create directories on both machines
3. Upload `shadowmesh-daemon` binary
4. Upload configuration files
5. Set correct permissions

### Manual Deployment (Alternative)

If you prefer manual deployment:

**UK VPS**:
```bash
# On local machine
scp build/shadowmesh-daemon user@uk-vps:/opt/shadowmesh/
scp configs/vps-uk-listener.yaml user@uk-vps:/etc/shadowmesh/config.yaml

# On UK VPS
ssh user@uk-vps
sudo mkdir -p /opt/shadowmesh /etc/shadowmesh /var/lib/shadowmesh/keys /var/log/shadowmesh
sudo chown root:root /etc/shadowmesh/config.yaml
sudo chmod 600 /etc/shadowmesh/config.yaml
```

**Belgium RPi**:
```bash
# On local machine
scp build/shadowmesh-daemon pi@rpi:/opt/shadowmesh/
scp configs/rpi-belgium-connector.yaml pi@rpi:/etc/shadowmesh/config.yaml

# On RPi
ssh pi@rpi
sudo mkdir -p /opt/shadowmesh /etc/shadowmesh /var/lib/shadowmesh/keys /var/log/shadowmesh
sudo chown root:root /etc/shadowmesh/config.yaml
sudo chmod 600 /etc/shadowmesh/config.yaml
```

## Running the Test

### Step 1: Start UK VPS (Listener)

SSH to UK VPS and start the daemon:

```bash
ssh user@uk-vps-ip
sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml
```

**Expected output**:
```
ShadowMesh Client Daemon v0.1.0-alpha
Post-Quantum Decentralized Private Network (DPN)
=================================================

Configuration loaded from: /etc/shadowmesh/config.yaml
Client ID: <32-byte hex string>
Creating TAP device: chr-001
TAP device created: chr-001 (MTU: 1500)
Configuring TAP interface: 10.100.0.1/255.255.255.0
Mode: Listener - Waiting for connections on 0.0.0.0:8443
Listening for P2P connections on 0.0.0.0:8443 (TLS: true)
Daemon running. Press Ctrl+C to stop.
```

### Step 2: Start Belgium RPi (Connector)

In a new terminal, SSH to RPi and start the daemon:

```bash
ssh pi@rpi-ip
sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml
```

**Expected output**:
```
ShadowMesh Client Daemon v0.1.0-alpha
Post-Quantum Decentralized Private Network (DPN)
=================================================

Configuration loaded from: /etc/shadowmesh/config.yaml
Client ID: <32-byte hex string>
Creating TAP device: chr-001
TAP device created: chr-001 (MTU: 1500)
Configuring TAP interface: 10.100.0.2/255.255.255.0
Mode: Connector - Connecting to peer: <uk-vps-ip>:8443
Connecting to peer at <uk-vps-ip>:8443 (attempt 1)...
Connected to peer at <uk-vps-ip>:8443
Waiting for connection...
Performing post-quantum handshake...
Generating post-quantum keys (ML-KEM-1024, ML-DSA-87)... this may take 10-30 seconds
Post-quantum keys generated successfully
Handshake state ready
Creating HELLO message (generating ephemeral Kyber keys)...
HELLO message created, sending to peer
Waiting for CHALLENGE from peer...
Handshake complete. Session ID: <session-id>
Session parameters: MTU=1500, Heartbeat=30s, KeyRotation=1h
Starting encrypted tunnel...
Tunnel established. Network traffic is now encrypted.
Daemon running. Press Ctrl+C to stop.
```

On UK VPS, you should see:
```
Peer connected from <rpi-ip>:xxxxx
Connected to peer
Performing post-quantum handshake...
Handshake complete. Session ID: <session-id>
Tunnel established. Network traffic is now encrypted.
```

### Step 3: Verify TAP Devices

**On UK VPS**:
```bash
ip addr show chr-001
# Should show: inet 10.100.0.1/24
```

**On Belgium RPi**:
```bash
ip addr show chr-001
# Should show: inet 10.100.0.2/24
```

### Step 4: Test Encrypted Tunnel

**From UK VPS** (ping RPi):
```bash
ping -c 5 10.100.0.2
```

Expected result:
```
PING 10.100.0.2 (10.100.0.2) 56(84) bytes of data.
64 bytes from 10.100.0.2: icmp_seq=1 ttl=64 time=XX ms
64 bytes from 10.100.0.2: icmp_seq=2 ttl=64 time=XX ms
...
5 packets transmitted, 5 received, 0% packet loss
```

**From Belgium RPi** (ping UK VPS):
```bash
ping -c 5 10.100.0.1
```

Expected result:
```
PING 10.100.0.1 (10.100.0.1) 56(84) bytes of data.
64 bytes from 10.100.0.1: icmp_seq=1 ttl=64 time=XX ms
64 bytes from 10.100.0.1: icmp_seq=2 ttl=64 time=XX ms
...
5 packets transmitted, 5 received, 0% packet loss
```

### Step 5: Check Statistics

Every 60 seconds, the daemon logs statistics:

```
Stats: Sent=50 frames (7500 bytes), Recv=50 frames (7500 bytes), Errors: Encrypt=0 Decrypt=0 Dropped=0
```

## Advanced Testing

### Throughput Test

**From UK VPS**:
```bash
# Install iperf3 if needed
sudo apt-get install -y iperf3

# Start server
iperf3 -s -B 10.100.0.1
```

**From Belgium RPi**:
```bash
sudo apt-get install -y iperf3

# Test throughput over encrypted tunnel
iperf3 -c 10.100.0.1 -B 10.100.0.2 -t 30
```

**Target Performance**:
- Throughput: 1+ Gbps (depends on hardware)
- Latency overhead: <5ms

### Latency Test

```bash
# From either machine
ping -c 100 -i 0.2 10.100.0.X | tail -5
```

Look for `avg` latency in summary.

### Packet Capture

**On UK VPS** (capture encrypted traffic):
```bash
sudo tcpdump -i eth0 'tcp port 8443' -w /tmp/encrypted.pcap
```

**On chr-001** (capture decrypted traffic):
```bash
sudo tcpdump -i chr-001 -w /tmp/decrypted.pcap
```

Analyze with Wireshark:
- Encrypted traffic should appear as TLS/WebSocket binary data
- Decrypted traffic on chr-001 should show ICMP, IP packets

### Key Rotation Test

Wait 1 hour (or reduce `key_rotation_interval` in config) and check logs:

```
Performing key rotation...
Key rotation complete. New session ID: <new-session-id>
```

## Troubleshooting

### Connection Refuses

**Check firewall**:
```bash
# UK VPS
sudo ufw status
sudo iptables -L -n | grep 8443
```

**Check daemon is listening**:
```bash
sudo netstat -tlnp | grep 8443
# or
sudo ss -tlnp | grep 8443
```

### TAP Device Creation Fails

```bash
# Check if running as root
whoami  # Should be root

# Check TUN/TAP module
lsmod | grep tun
sudo modprobe tun

# Check permissions
ls -l /dev/net/tun
```

### Handshake Timeout

- Check network connectivity: `ping uk-vps-ip`
- Verify config file peer_address is correct
- Check TLS certificate paths (if using custom certs)
- Look for "failed to send HELLO" or "timeout waiting for CHALLENGE"

### Ping Doesn't Work

**Verify TAP configuration**:
```bash
ip addr show chr-001
ip route show
```

**Check daemon logs**:
```bash
tail -f /var/log/shadowmesh/daemon.log
```

Look for encryption/decryption errors.

**Verify encryption**:
```bash
# Should see stats increasing
# Watch for EncryptErrors or DecryptErrors counters
```

## Test Success Criteria

✅ **Test Passes If**:
1. Both daemons start without errors
2. P2P connection establishes (listener accepts, connector connects)
3. PQC handshake completes (ML-KEM-1024 + ML-DSA-87)
4. TAP devices created with correct IPs
5. Ping works bidirectionally (0% packet loss)
6. Encryption statistics show sent/received frames with no errors
7. Throughput meets target (1+ Gbps if hardware supports)
8. Latency overhead <5ms

## Cleanup

### Stop Daemons

Press `Ctrl+C` on both machines to gracefully stop the daemons.

Expected cleanup output:
```
Received shutdown signal. Cleaning up...
Stopping tunnel...
Closing connection...
Stopping TAP device...
Daemon stopped.
```

### Remove TAP Devices (if needed)

```bash
# TAP devices are automatically removed when daemon stops
# Verify:
ip addr show chr-001  # Should not exist
```

### Uninstall (if needed)

```bash
sudo rm -rf /opt/shadowmesh
sudo rm -rf /etc/shadowmesh
sudo rm -rf /var/lib/shadowmesh
sudo rm -rf /var/log/shadowmesh
```

## Next Steps

After successful P2P testing:

1. **Epic 3**: Multi-hop routing
2. **Epic 4**: Traffic obfuscation
3. **Epic 5**: Exit node verification (TPM/SGX)
4. **Epic 6**: Performance optimization

## Appendix: Configuration Files

### vps-uk-listener.yaml

```yaml
mode: listener

p2p:
  listen_address: "0.0.0.0:8443"
  tls_enabled: true
  tls_skip_verify: false

tap:
  name: "chr-001"
  mtu: 1500
  ip_addr: "10.100.0.1"
  netmask: "255.255.255.0"

crypto:
  key_rotation_interval: 1h
  enable_key_rotation: true

identity:
  keys_dir: "/root/.shadowmesh/keys"
  private_key_file: "/root/.shadowmesh/keys/signing_key.json"
  client_id_file: "/root/.shadowmesh/keys/client_id.txt"

logging:
  level: "debug"
  format: "text"
  file: "/var/log/shadowmesh/daemon.log"
```

### rpi-belgium-connector.yaml

```yaml
mode: connector

p2p:
  peer_address: "<UK_VPS_IP>:8443"
  tls_enabled: true
  tls_skip_verify: false

tap:
  name: "chr-001"
  mtu: 1500
  ip_addr: "10.100.0.2"
  netmask: "255.255.255.0"

crypto:
  key_rotation_interval: 1h
  enable_key_rotation: true

identity:
  keys_dir: "/home/pi/.shadowmesh/keys"
  private_key_file: "/home/pi/.shadowmesh/keys/signing_key.json"
  client_id_file: "/home/pi/.shadowmesh/keys/client_id.txt"

logging:
  level: "debug"
  format: "text"
  file: "/var/log/shadowmesh/daemon.log"
```

## Support

If you encounter issues, collect the following:
1. Daemon logs from both machines (`/var/log/shadowmesh/daemon.log`)
2. Network configuration (`ip addr`, `ip route`)
3. Firewall rules (`sudo iptables -L -n`)
4. Test output (ping results, error messages)

Report issues with full logs and system information.
