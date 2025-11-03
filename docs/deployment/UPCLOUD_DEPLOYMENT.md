# UpCloud Automated Deployment

<img src="https://pbs.twimg.com/profile_images/1969957304679473152/QW21M-FO_400x400.jpg" alt="Chronara Group Logo" width="80" align="right"/>

**Chronara Group ShadowMesh - UpCloud Deployment**

## Quick Start

### 1. Commit and Push Changes to GitHub

```bash
cd ~/Webcode/shadowmesh

# Stage all changes
git add README.md DISTRIBUTED_TESTING.md UPCLOUD_DEPLOYMENT.md
git add scripts/install-relay.sh scripts/deploy-upcloud.sh

# Commit
git commit -m "Add UpCloud automated deployment and distributed testing

- scripts/deploy-upcloud.sh: Automated deployment using upctl CLI
  * Creates VM with cloud-init
  * Auto-installs relay server
  * Configures firewall
  * Returns WebSocket URL for clients

- scripts/install-relay.sh: Production relay installer
- DISTRIBUTED_TESTING.md: Complete cloud testing guide
- UPCLOUD_DEPLOYMENT.md: Quick reference for upctl deployment

🤖 Generated with Development Team (https://claude.com/claude-code)

Co-Authored-By: Development Team <noreply@shadowmesh.dev>"

# Push to GitHub
git push
```

**⚠️ Important:** Wait for push to complete before proceeding - the deployment script downloads the installer from GitHub.

### 2. Configure UpCloud CLI

```bash
# Set your API token (replace with your actual token)
upctl config set --key username=YOUR_USERNAME token=ucat_01K8ZDVJTT5CPFY22Z06ZQKXHT

# Test configuration
upctl account show

# Expected output:
# Username: your-username
# Credits: xxx.xx
```

### 3. Deploy Relay Server

```bash
cd ~/Webcode/shadowmesh

# Deploy to Frankfurt (default)
./scripts/deploy-upcloud.sh shadowmesh-relay de-fra1

# Or choose different location:
# Amsterdam: ./scripts/deploy-upcloud.sh shadowmesh-relay nl-ams1
# London: ./scripts/deploy-upcloud.sh shadowmesh-relay uk-lon1
# Helsinki: ./scripts/deploy-upcloud.sh shadowmesh-relay fi-hel1
# Singapore: ./scripts/deploy-upcloud.sh shadowmesh-relay sg-sin1
```

**The script will:**
1. ✅ Verify upctl is configured
2. ✅ Generate SSH key for server access
3. ✅ Upload SSH key to UpCloud
4. ✅ Create VM with cloud-init
5. ✅ Auto-install relay server
6. ✅ Configure firewall (ports 22, 8443)
7. ✅ Start relay service
8. ✅ Return server IP and WebSocket URL

**Expected output:**
```
╔═══════════════════════════════════════════════════════════╗
║       UpCloud VM Deployed Successfully!                   ║
╚═══════════════════════════════════════════════════════════╝

📋 Server Information:
   UUID: 00e1c8d0-xxxx-xxxx-xxxx-xxxxxxxxxxxx
   Hostname: shadowmesh-relay
   Public IP: <YOUR_SERVER_IP>
   Zone: de-fra1
   Plan: 1xCPU-2GB

🔐 SSH Access:
   ssh root@<YOUR_SERVER_IP> -i ~/.ssh/shadowmesh_relay_ed25519

⏳ Installation Progress:
   The relay server is being installed via cloud-init (2-3 minutes)

🎯 After Installation Completes:
   WebSocket URL: wss://<YOUR_SERVER_IP>:8443/ws
```

### 4. Wait for Installation (2-3 minutes)

Monitor installation progress:

```bash
# SSH to server
ssh root@YOUR_SERVER_IP -i ~/.ssh/shadowmesh_relay_ed25519

# Watch installation log
tail -f /var/log/shadowmesh-install.log

# Expected final message:
# "ShadowMesh relay installation completed at [timestamp]"
```

### 5. Verify Relay is Running

```bash
# Test health endpoint (from anywhere)
curl -k https://YOUR_SERVER_IP:8443/health

# Expected: {"status":"ok","active_clients":0}

# Check status (from server)
ssh root@YOUR_SERVER_IP -i ~/.ssh/shadowmesh_relay_ed25519 'systemctl status shadowmesh-relay'

# Expected: Active: active (running)
```

### 6. View Relay Logs

```bash
# Real-time logs
ssh root@YOUR_SERVER_IP -i ~/.ssh/shadowmesh_relay_ed25519 'journalctl -u shadowmesh-relay -f'

# Expected:
# Starting relay server on 0.0.0.0:8443
# ShadowMesh Relay Server started successfully
# Relay ID: 3a5f8e2d4c1b9f7a...
```

### 7. Get Relay Information

```bash
# SSH to server
ssh root@YOUR_SERVER_IP -i ~/.ssh/shadowmesh_relay_ed25519

# View configuration
cat /etc/shadowmesh/config.yaml

# Get Relay ID
cat /var/lib/shadowmesh/keys/relay_id.txt

# Check certificate
openssl x509 -in /etc/shadowmesh/relay-cert.pem -noout -text | head -20
```

## Test Client Connection

### On Proxmox VM:

```bash
# Build client locally first
cd ~/Webcode/shadowmesh
make build-client

# Transfer to Proxmox VM
scp bin/shadowmesh-client root@YOUR_PROXMOX_IP:/usr/local/bin/
```

### Configure client on Proxmox VM:

```bash
mkdir -p ~/.shadowmesh/keys

cat > ~/.shadowmesh/config.yaml <<EOF
relay:
  url: "wss://YOUR_UPCLOUD_IP:8443/ws"
  reconnect_interval: 5s
  max_reconnect_attempts: 20
  heartbeat_interval: 30s
  insecure_skip_verify: true

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
```

### Start client:

```bash
sudo /usr/local/bin/shadowmesh-client
```

### Expected output:

```
╔═══════════════════════════════════════════════════════════╗
║         ShadowMesh Client v0.1.0-alpha                   ║
║     Post-Quantum Encrypted Private Network               ║
╚═══════════════════════════════════════════════════════════╝

Generating new client identity...
Client ID: 7b2e9a4f3c8d1e5a...
Creating TAP device: tap0
TAP device created: tap0 (10.42.0.2/24)
Connecting to relay: wss://<YOUR_SERVER_IP>:8443/ws
Connection established
Starting handshake...
Sent HELLO message
Received CHALLENGE message (session: 9f3c2a1d...)
Sent RESPONSE message
Received ESTABLISHED message
Handshake complete! Session established
Starting tunnel...
```

### On relay server, you'll see:

```bash
ssh root@YOUR_SERVER_IP -i ~/.ssh/shadowmesh_relay_ed25519 'journalctl -u shadowmesh-relay -f'

# Expected:
# New connection from xxx.xxx.xxx.xxx:xxxxx (total: 1, active: 1)
# Starting handshake with client from xxx.xxx.xxx.xxx
# Received HELLO from client 7b2e9a4f
# Sent CHALLENGE to client 7b2e9a4f (session: 9f3c2a1d)
# Received RESPONSE from client 7b2e9a4f
# Sent ESTABLISHED to client 7b2e9a4f
# Handshake complete with client 7b2e9a4f (session: 9f3c2a1d)
# Client 7b2e9a4f established (session: 9f3c2a1d)
# Registered client 7b2e9a4f (total clients: 1)
```

## Success Criteria

- [ ] UpCloud VM created via upctl CLI
- [ ] Relay server auto-installed via cloud-init
- [ ] Relay service running and listening on port 8443
- [ ] Health endpoint returns 200 OK
- [ ] SSH access works with generated key
- [ ] Client connects from Proxmox VM
- [ ] Post-quantum handshake completes (4 messages)
- [ ] TAP device created successfully
- [ ] Tunnel established

## Troubleshooting

### upctl not configured

```bash
# Configure with your API credentials
upctl config set --key username=YOUR_USERNAME token=YOUR_TOKEN

# Test
upctl account show
```

### SSH key already exists

The script reuses existing SSH key at `~/.ssh/shadowmesh_relay_ed25519` if found.

To generate new key:
```bash
rm ~/.ssh/shadowmesh_relay_ed25519*
./scripts/deploy-upcloud.sh
```

### Cloud-init installation taking too long

```bash
# SSH to server and check cloud-init status
ssh root@YOUR_SERVER_IP -i ~/.ssh/shadowmesh_relay_ed25519 'cloud-init status'

# Expected: status: done

# If still running, wait for completion
ssh root@YOUR_SERVER_IP -i ~/.ssh/shadowmesh_relay_ed25519 'cloud-init status --wait'
```

### Relay service not starting

```bash
# SSH to server
ssh root@YOUR_SERVER_IP -i ~/.ssh/shadowmesh_relay_ed25519

# Check installation log
cat /var/log/shadowmesh-install.log

# Check service status
systemctl status shadowmesh-relay

# View detailed logs
journalctl -u shadowmesh-relay -n 100
```

## Clean Up

### Delete UpCloud VM

```bash
# List your servers
upctl server list

# Delete by UUID
upctl server delete UUID_FROM_LIST

# Or delete by hostname
upctl server delete --hostname shadowmesh-relay
```

### Remove SSH key

```bash
# List SSH keys
upctl sshkey list

# Delete SSH key
upctl sshkey delete --title shadowmesh-relay-key

# Remove local key
rm ~/.ssh/shadowmesh_relay_ed25519*
```

## Available UpCloud Zones

- **Europe:**
  - `de-fra1` - Frankfurt, Germany
  - `nl-ams1` - Amsterdam, Netherlands
  - `uk-lon1` - London, United Kingdom
  - `fi-hel1` - Helsinki, Finland
  - `fi-hel2` - Helsinki, Finland
  - `es-mad1` - Madrid, Spain
  - `pl-waw1` - Warsaw, Poland
  - `se-sto1` - Stockholm, Sweden

- **North America:**
  - `us-chi1` - Chicago, USA
  - `us-nyc1` - New York, USA
  - `us-sjo1` - San Jose, USA

- **Asia-Pacific:**
  - `sg-sin1` - Singapore
  - `au-syd1` - Sydney, Australia

Choose the zone closest to your Proxmox VM for best latency!

## Cost Estimate

**1xCPU-2GB Plan:**
- Monthly: ~$7-10 USD
- Hourly: ~$0.012 USD
- Storage: 50 GB SSD included

**For testing:** Deploy for a few hours, then delete = pennies!

## Next Steps

After successful deployment:

1. **Add more clients** - Test with multiple Proxmox VMs
2. **Performance test** - Run iperf3 between clients
3. **Monitoring** - Set up Prometheus + Grafana
4. **Production TLS** - Use Let's Encrypt certificates
5. **Multi-region** - Deploy relay servers in multiple zones
6. **Load balancing** - Distribute clients across relay servers

See [DISTRIBUTED_TESTING.md](DISTRIBUTED_TESTING.md) for complete testing procedures.
