# ShadowMesh Distributed Testing Guide

## Overview

This guide walks through setting up a production-like test environment with:
- **UpCloud VM** - Running the relay server (public cloud)
- **Proxmox VM** - Running the client (local network)

This setup validates real-world connectivity, NAT traversal, and cloud deployment.

## Architecture

```
┌─────────────────────────┐
│    Proxmox VM           │
│    (Client)             │
│                         │
│  10.42.0.2 (tap0)       │
│    │                    │
│    │ WebSocket/TLS      │
│    │ (PQC Handshake)    │
│    └─────────┐          │
└──────────────┼──────────┘
               │
               │ Internet
               │
        ┌──────▼─────────────┐
        │  UpCloud VM        │
        │  (Relay Server)    │
        │                    │
        │  Public IP:        │
        │  xxx.xxx.xxx.xxx   │
        │  Port: 8443        │
        └────────────────────┘
```

## Prerequisites

### UpCloud VM Requirements
- Ubuntu 22.04 LTS or Debian 11+
- 1 GB RAM minimum (2 GB recommended)
- 1 vCPU (2+ recommended for production)
- 10 GB disk space
- Public IPv4 address
- Port 8443 open in firewall

### Proxmox VM Requirements
- Ubuntu 22.04 LTS or similar
- 2 GB RAM
- 2 vCPU
- 20 GB disk space
- Internet connectivity
- Root/sudo access

### Local Development Machine
- Git installed
- Go 1.21+ installed (for building client)

## Part 1: Deploy Relay Server on UpCloud

### Step 1.1: Create UpCloud VM

1. Log into UpCloud console: https://hub.upcloud.com/
2. Click "Deploy a new server"
3. Select configuration:
   - **Location**: Choose closest to you (e.g., Frankfurt, Amsterdam, London)
   - **Plan**: Simple, 1 GB RAM, 1 vCPU, 25 GB SSD
   - **OS**: Ubuntu 22.04 LTS
   - **Hostname**: `shadowmesh-relay`
   - **SSH Keys**: Add your SSH public key
4. Click "Deploy"
5. Wait for VM to start (~60 seconds)
6. Note the public IP address (e.g., `94.237.85.123`)

### Step 1.2: SSH to UpCloud VM

```bash
# SSH to your UpCloud VM
ssh root@YOUR_UPCLOUD_IP

# Example:
ssh root@94.237.85.123
```

### Step 1.3: Install Relay Server

Run the one-line installer:

```bash
curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/scripts/install-relay.sh | sudo bash
```

**What this does:**
- Installs Go 1.21.5
- Clones ShadowMesh repository
- Builds relay server binary
- Creates system user `shadowmesh`
- Generates relay identity (post-quantum keys)
- Creates default configuration
- Generates self-signed TLS certificate
- Creates systemd service
- Configures firewall (if ufw present)

**Expected output:**
```
╔═══════════════════════════════════════════════════════════╗
║       ShadowMesh Relay Server Installer                  ║
║       Post-Quantum VPN Network                            ║
╚═══════════════════════════════════════════════════════════╝

Detected: Linux x86_64
✅ Go already installed: go1.21.5
Installing to /opt/shadowmesh...
Building relay server...
✅ Configuration created: /etc/shadowmesh/config.yaml
✅ Self-signed certificate generated
✅ Firewall rule added: allow 8443/tcp

╔═══════════════════════════════════════════════════════════╗
║       Installation Complete!                              ║
╚═══════════════════════════════════════════════════════════╝

📋 Relay Information:
   Relay ID: 3a5f8e2d4c1b9f7a...
   Server IP: 94.237.85.123
   Listen Port: 8443
   WebSocket URL: wss://94.237.85.123:8443/ws
```

**Save this information** - you'll need the WebSocket URL for client configuration.

### Step 1.4: Start Relay Service

```bash
# Enable service to start on boot
sudo systemctl enable shadowmesh-relay

# Start the service
sudo systemctl start shadowmesh-relay

# Check status
sudo systemctl status shadowmesh-relay
```

**Expected output:**
```
● shadowmesh-relay.service - ShadowMesh Post-Quantum VPN Relay Server
     Loaded: loaded (/etc/systemd/system/shadowmesh-relay.service; enabled)
     Active: active (running) since Fri 2025-11-01 18:00:00 UTC; 5s ago
   Main PID: 12345 (shadowmesh-rela)
      Tasks: 10 (limit: 1131)
     Memory: 15.2M
        CPU: 250ms
     CGroup: /system.slice/shadowmesh-relay.service
             └─12345 /usr/local/bin/shadowmesh-relay --config /etc/shadowmesh/config.yaml

Nov 01 18:00:00 shadowmesh-relay systemd[1]: Started ShadowMesh Post-Quantum VPN Relay Server.
Nov 01 18:00:00 shadowmesh-relay shadowmesh-relay[12345]: Relay ID: 3a5f8e2d4c1b9f7a...
Nov 01 18:00:00 shadowmesh-relay shadowmesh-relay[12345]: Starting relay server on 0.0.0.0:8443
Nov 01 18:00:00 shadowmesh-relay shadowmesh-relay[12345]: ShadowMesh Relay Server started successfully
```

### Step 1.5: Verify Relay is Running

```bash
# Test health endpoint
curl -k https://localhost:8443/health

# Expected: {"status":"ok","active_clients":0}

# Check stats
curl -k https://localhost:8443/stats

# Expected: {"total_connections":0,"active_connections":0,"registered_clients":0}
```

### Step 1.6: View Relay Logs

```bash
# Follow logs in real-time
sudo journalctl -u shadowmesh-relay -f

# View last 50 lines
sudo journalctl -u shadowmesh-relay -n 50
```

**Keep this terminal open** to watch relay logs during client connection.

### Step 1.7: Get TLS Certificate Fingerprint (Optional)

For added security, you can verify the certificate fingerprint on the client:

```bash
openssl x509 -in /etc/shadowmesh/relay-cert.pem -noout -fingerprint -sha256

# Output: SHA256 Fingerprint=3A:5F:8E:2D:4C:1B:9F:7A:...
```

## Part 2: Build Client on Local Machine

### Step 2.1: Navigate to Project

```bash
cd ~/Webcode/shadowmesh
```

### Step 2.2: Build Client Binary

```bash
# Build client
make build-client

# Verify binary
ls -lh bin/shadowmesh-client

# Expected: -rwxr-xr-x  1 user  staff   7.2M Nov  1 18:00 bin/shadowmesh-client
```

### Step 2.3: Transfer Client to Proxmox VM

```bash
# SCP to Proxmox VM
scp bin/shadowmesh-client root@YOUR_PROXMOX_VM_IP:/usr/local/bin/

# Example:
scp bin/shadowmesh-client root@192.168.1.100:/usr/local/bin/
```

## Part 3: Configure Client on Proxmox VM

### Step 3.1: SSH to Proxmox VM

```bash
ssh root@YOUR_PROXMOX_VM_IP

# Example:
ssh root@192.168.1.100
```

### Step 3.2: Create Client Configuration

Replace `YOUR_UPCLOUD_IP` with your actual UpCloud public IP:

```bash
# Create config directory
mkdir -p ~/.shadowmesh/keys

# Create configuration
cat > ~/.shadowmesh/config.yaml <<EOF
relay:
  url: "wss://YOUR_UPCLOUD_IP:8443/ws"
  reconnect_interval: 5s
  max_reconnect_attempts: 20
  heartbeat_interval: 30s
  insecure_skip_verify: true  # For self-signed certs

tap:
  name: "tap0"
  mtu: 1500
  ip_addr: "10.42.0.2"
  netmask: "255.255.255.0"

crypto:
  enable_key_rotation: false
  key_rotation_interval: 1h

identity:
  keys_dir: "$HOME/.shadowmesh/keys"
  private_key_file: "$HOME/.shadowmesh/keys/signing_key.json"
  client_id_file: "$HOME/.shadowmesh/keys/client_id.txt"

logging:
  level: "info"
  format: "text"
EOF

echo "✅ Client configuration created"
```

**Example with real IP:**
```yaml
relay:
  url: "wss://94.237.85.123:8443/ws"
  # ... rest of config
```

### Step 3.3: Install TUN/TAP Support (if needed)

```bash
# Install TUN/TAP kernel module
sudo apt-get update
sudo apt-get install -y kmod

# Load TUN module
sudo modprobe tun

# Verify
lsmod | grep tun
# Expected: tun                    49152  0
```

## Part 4: Run Test

### Step 4.1: Start Client

On Proxmox VM:

```bash
# Run client (generates keys on first run)
sudo /usr/local/bin/shadowmesh-client

# Expected output:
# ╔═══════════════════════════════════════════════════════════╗
# ║         ShadowMesh Client v0.1.0-alpha                   ║
# ║     Post-Quantum Encrypted Private Network               ║
# ╚═══════════════════════════════════════════════════════════╝
#
# 2025/11/01 18:05:00 Loading configuration...
# 2025/11/01 18:05:00 Generating new client identity...
# 2025/11/01 18:05:01 Client ID: 7b2e9a4f3c8d...
# 2025/11/01 18:05:01 Creating TAP device: tap0
# 2025/11/01 18:05:01 TAP device created: tap0 (10.42.0.2/24)
# 2025/11/01 18:05:01 Connecting to relay: wss://94.237.85.123:8443/ws
# 2025/11/01 18:05:02 Connection established
# 2025/11/01 18:05:02 Starting handshake...
# 2025/11/01 18:05:02 Sent HELLO message
# 2025/11/01 18:05:02 Received CHALLENGE message (session: 9f3c2a1d...)
# 2025/11/01 18:05:03 Sent RESPONSE message
# 2025/11/01 18:05:03 Received ESTABLISHED message
# 2025/11/01 18:05:03 Handshake complete! Session established
# 2025/11/01 18:05:03 Starting tunnel...
```

### Step 4.2: Check Relay Logs

Back on UpCloud VM terminal:

```bash
sudo journalctl -u shadowmesh-relay -f
```

**You should see:**
```
Nov 01 18:05:02 shadowmesh-relay shadowmesh-relay[12345]: New connection from xxx.xxx.xxx.xxx:xxxxx (total: 1, active: 1)
Nov 01 18:05:02 shadowmesh-relay shadowmesh-relay[12345]: Starting handshake with client from xxx.xxx.xxx.xxx
Nov 01 18:05:02 shadowmesh-relay shadowmesh-relay[12345]: Received HELLO from client 7b2e9a4f
Nov 01 18:05:02 shadowmesh-relay shadowmesh-relay[12345]: Sent CHALLENGE to client 7b2e9a4f (session: 9f3c2a1d)
Nov 01 18:05:03 shadowmesh-relay shadowmesh-relay[12345]: Received RESPONSE from client 7b2e9a4f
Nov 01 18:05:03 shadowmesh-relay shadowmesh-relay[12345]: Sent ESTABLISHED to client 7b2e9a4f
Nov 01 18:05:03 shadowmesh-relay shadowmesh-relay[12345]: Handshake complete with client 7b2e9a4f (session: 9f3c2a1d)
Nov 01 18:05:03 shadowmesh-relay shadowmesh-relay[12345]: Client 7b2e9a4f established (session: 9f3c2a1d)
Nov 01 18:05:03 shadowmesh-relay shadowmesh-relay[12345]: Registered client 7b2e9a4f (total clients: 1)
```

### Step 4.3: Verify TAP Device

On Proxmox VM:

```bash
# Check TAP interface
ip addr show tap0

# Expected:
# tap0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc pfifo_fast state UNKNOWN group default qlen 1000
#     inet 10.42.0.2/24 scope global tap0

# Check routing
ip route show dev tap0

# Expected:
# 10.42.0.0/24 dev tap0 proto kernel scope link src 10.42.0.2
```

### Step 4.4: Test Connectivity to Relay

From Proxmox VM:

```bash
# Ping tap0 interface (should work)
ping -c 3 -I tap0 10.42.0.2

# Expected:
# PING 10.42.0.2 (10.42.0.2) from 10.42.0.2 tap0: 56(84) bytes of data.
# 64 bytes from 10.42.0.2: icmp_seq=1 ttl=64 time=0.045 ms
```

## Part 5: Add Second Client (Optional)

To test frame routing, add another client (can be another Proxmox VM or your local machine):

### Step 5.1: Configure Second Client

On second machine:

```bash
mkdir -p ~/.shadowmesh-client2/keys

cat > ~/.shadowmesh-client2/config.yaml <<EOF
relay:
  url: "wss://YOUR_UPCLOUD_IP:8443/ws"
  reconnect_interval: 5s
  max_reconnect_attempts: 20
  heartbeat_interval: 30s
  insecure_skip_verify: true

tap:
  name: "tap1"
  mtu: 1500
  ip_addr: "10.42.0.3"  # Different IP!
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
```

### Step 5.2: Start Second Client

```bash
sudo shadowmesh-client --config ~/.shadowmesh-client2/config.yaml
```

### Step 5.3: Test Ping Between Clients

From first client (10.42.0.2):

```bash
ping -c 5 10.42.0.3

# Expected:
# PING 10.42.0.3 (10.42.0.3) 56(84) bytes of data.
# 64 bytes from 10.42.0.3: icmp_seq=1 ttl=64 time=45.2 ms  # Internet round-trip!
# 64 bytes from 10.42.0.3: icmp_seq=2 ttl=64 time=42.8 ms
```

**Watch relay logs** - you should see frame routing:

```
Nov 01 18:10:00 shadowmesh-relay shadowmesh-relay[12345]: Routing frame from client 7b2e9a4f (98 bytes)
Nov 01 18:10:00 shadowmesh-relay shadowmesh-relay[12345]: Broadcasting to 1 clients
```

## Part 6: Monitor and Validate

### Check Relay Statistics

```bash
# From UpCloud VM or any machine:
curl -k https://YOUR_UPCLOUD_IP:8443/stats

# Expected:
# {"total_connections":2,"active_connections":2,"registered_clients":2}
```

### Monitor Client Statistics

Client logs show periodic statistics:

```
2025/11/01 18:15:00 Stats: tx_frames=234, rx_frames=189, session_age=10m0s
```

### Verify Heartbeats

Every 30 seconds, you should see:

**Client logs:**
```
2025/11/01 18:15:30 Sending heartbeat
2025/11/01 18:15:30 Received heartbeat ACK
```

**Relay logs:**
```
Nov 01 18:15:30 shadowmesh-relay shadowmesh-relay[12345]: Heartbeat from client 7b2e9a4f
```

## Validation Checklist

- [ ] Relay server installed on UpCloud VM
- [ ] Relay service running and accessible on port 8443
- [ ] Health endpoint returns 200 OK
- [ ] Client built and transferred to Proxmox VM
- [ ] Client connects to relay successfully
- [ ] Post-quantum handshake completes (4 messages)
- [ ] TAP device created with correct IP
- [ ] Client can ping its own TAP interface
- [ ] Second client connects successfully
- [ ] Clients can ping each other through relay
- [ ] Frame routing visible in relay logs
- [ ] Heartbeats exchanged every 30 seconds
- [ ] Statistics endpoints return correct data

## Troubleshooting

### Issue: Client can't connect to relay

**Check:**
```bash
# Test relay connectivity from client machine
curl -k https://YOUR_UPCLOUD_IP:8443/health

# If fails:
# 1. Check UpCloud firewall allows port 8443
# 2. Check relay service is running: systemctl status shadowmesh-relay
# 3. Check relay logs: journalctl -u shadowmesh-relay -n 50
```

### Issue: Handshake timeout

**Check client logs for specific error:**
```bash
# On Proxmox VM
sudo shadowmesh-client 2>&1 | tee client.log
```

**Common causes:**
- Certificate verification failing (should use `insecure_skip_verify: true` for self-signed)
- Wrong relay URL in client config
- Firewall blocking outbound connection on client side

### Issue: "Permission denied" creating TAP device

```bash
# Run with sudo
sudo shadowmesh-client

# Or grant capability:
sudo setcap cap_net_admin=eip /usr/local/bin/shadowmesh-client
```

### Issue: TAP device already exists

```bash
# Delete existing device
sudo ip link delete tap0

# Restart client
sudo shadowmesh-client
```

## Performance Testing

### Latency Test

```bash
# From client 1 to client 2
ping -c 100 10.42.0.3 | tail -1

# Expected: avg ~40-80ms (depends on internet connection)
```

### Throughput Test (iperf3)

On client 2:
```bash
iperf3 -s -B 10.42.0.3
```

On client 1:
```bash
iperf3 -c 10.42.0.3 -t 30

# Expected: 10-50 Mbps (depends on bottleneck)
```

## Production Considerations

### For Production Deployment:

1. **Use Let's Encrypt for TLS:**
   ```bash
   sudo apt-get install certbot
   sudo certbot certonly --standalone -d your-domain.com
   ```

2. **Update relay config:**
   ```yaml
   tls:
     enabled: true
     cert_file: "/etc/letsencrypt/live/your-domain.com/fullchain.pem"
     key_file: "/etc/letsencrypt/live/your-domain.com/privkey.pem"
   ```

3. **Enable automatic certificate renewal:**
   ```bash
   sudo systemctl enable certbot-renew.timer
   ```

4. **Monitor logs with log aggregation** (ELK, Loki, CloudWatch)

5. **Set up monitoring alerts** for:
   - Relay service down
   - High CPU/memory usage
   - Certificate expiration
   - No client connections

6. **Backup relay identity:**
   ```bash
   sudo tar -czf shadowmesh-relay-backup.tar.gz /var/lib/shadowmesh/keys/
   ```

## Clean Up

### Stop Services

On UpCloud VM:
```bash
sudo systemctl stop shadowmesh-relay
sudo systemctl disable shadowmesh-relay
```

On Proxmox VM:
```bash
# Stop client (Ctrl+C)
sudo ip link delete tap0
```

### Remove Installation (if needed)

On UpCloud VM:
```bash
sudo systemctl stop shadowmesh-relay
sudo systemctl disable shadowmesh-relay
sudo rm /etc/systemd/system/shadowmesh-relay.service
sudo rm -rf /opt/shadowmesh /etc/shadowmesh /var/lib/shadowmesh
sudo rm /usr/local/bin/shadowmesh-relay
sudo userdel shadowmesh
```

## Next Steps

After successful distributed testing:

1. **Load Testing** - Test with 10+ concurrent clients
2. **Multi-Region** - Deploy multiple relay servers in different regions
3. **Monitoring** - Set up Prometheus + Grafana
4. **Production TLS** - Use Let's Encrypt certificates
5. **Automation** - Use Terraform/Ansible for deployment
6. **CI/CD** - Automate builds and deployments

## Summary

You've successfully deployed a production-like ShadowMesh network:

✅ **Cloud Relay Server** - Running on UpCloud with public IP
✅ **Client Connection** - From Proxmox VM over internet
✅ **Post-Quantum Handshake** - ML-KEM-1024 + ML-DSA-87
✅ **Encrypted Tunnel** - ChaCha20-Poly1305 frame encryption
✅ **Frame Routing** - Broadcast mesh networking
✅ **Production Ready** - Systemd service, logging, monitoring

**Your quantum-safe VPN is working! 🎉**
