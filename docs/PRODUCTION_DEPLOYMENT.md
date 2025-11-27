# ShadowMesh Production Deployment Guide

## Version: v0.2.0-relay

This guide covers production deployment of ShadowMesh relay infrastructure and client daemons.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    ShadowMesh Network                        │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  Client 1 (10.10.10.1)                                       │
│      │                                                        │
│      │ WebSocket (ws://relay:9545)                          │
│      │ ChaCha20-Poly1305 encrypted                          │
│      └──────────┐                                            │
│                  │                                            │
│                  ▼                                            │
│         ┌───────────────┐                                    │
│         │ Relay Server  │ Zero-Knowledge Routing             │
│         │ (Port 9545)   │ 1000 concurrent connections        │
│         │ Prometheus    │ Metrics on :9090                   │
│         └───────────────┘                                    │
│                  │                                            │
│      ┌───────────┴───────────┐                              │
│      │                        │                               │
│      ▼                        ▼                               │
│  Client 2 (10.10.10.2)    Client 3 (10.10.10.3)             │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Requirements

### Relay Server
- **OS**: Linux (Ubuntu 22.04+ recommended)
- **CPU**: 2+ cores
- **RAM**: 4GB+ (8GB recommended for 1000+ clients)
- **Network**: Public IP, Port 9545 open
- **Binary**: `shadowmesh-relay-linux-amd64`

### Client Daemon
- **OS**: Linux, macOS, Windows
- **CPU**: 1+ core
- **RAM**: 512MB+
- **Network**: Outbound HTTPS/WebSocket (443/9545)
- **Privileges**: Root/Admin (for TUN device creation)
- **Binary**:
  - Linux: `shadowmesh-daemon-linux-amd64`
  - macOS (Apple Silicon): `shadowmesh-daemon-darwin-arm64`
  - macOS (Intel): `shadowmesh-daemon-darwin-amd64`
  - Windows: `shadowmesh-daemon-windows-amd64.exe`

---

## Relay Server Deployment

### 1. Server Preparation

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y curl wget

# Create shadowmesh user
sudo useradd -r -s /bin/false shadowmesh

# Create directories
sudo mkdir -p /opt/shadowmesh
sudo mkdir -p /etc/shadowmesh
sudo mkdir -p /var/log/shadowmesh
```

### 2. Binary Installation

```bash
# Download relay binary
wget https://github.com/shadowmesh/shadowmesh/releases/download/v0.2.0/shadowmesh-relay-linux-amd64

# Install
sudo mv shadowmesh-relay-linux-amd64 /opt/shadowmesh/shadowmesh-relay
sudo chmod +x /opt/shadowmesh/shadowmesh-relay
sudo chown shadowmesh:shadowmesh /opt/shadowmesh/shadowmesh-relay
```

### 3. Configuration

Create `/etc/shadowmesh/relay-config.yaml`:

```yaml
# ShadowMesh Relay Server Configuration
region: "us-east-1"  # Your region identifier
server_name: "relay-us-east-1"  # Unique server name

relay_port: 9545      # WebSocket relay port
health_port: 8080     # Health check endpoint
metrics_port: 9090    # Prometheus metrics

max_connections: 1000      # Maximum concurrent clients
connection_timeout: 300    # Idle timeout (seconds)

read_buffer_size: 4096
write_buffer_size: 4096

# Zero-knowledge routing (relay sees routing only, not payload)
zero_knowledge: true
frame_logging: false  # Disable for privacy

# Metrics
metrics_enabled: true

# Performance tuning
worker_threads: 4      # CPU cores
queue_size: 1000

# Security
allowed_origins:
  - "*"  # Restrict in production

# Logging
log_level: "info"
log_connections: true
log_routing: false  # Disable for privacy
```

### 4. Systemd Service

Create `/etc/systemd/system/shadowmesh-relay.service`:

```ini
[Unit]
Description=ShadowMesh Relay Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=shadowmesh
Group=shadowmesh

ExecStart=/opt/shadowmesh/shadowmesh-relay -config /etc/shadowmesh/relay-config.yaml

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=shadowmesh-relay

# Restart policy
Restart=always
RestartSec=5s

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/shadowmesh

# Resource limits
LimitNOFILE=65536
TasksMax=4096

[Install]
WantedBy=multi-user.target
```

### 5. Start Relay Server

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service
sudo systemctl enable shadowmesh-relay

# Start service
sudo systemctl start shadowmesh-relay

# Check status
sudo systemctl status shadowmesh-relay

# View logs
sudo journalctl -u shadowmesh-relay -f
```

### 6. Verify Relay Server

```bash
# Health check
curl http://localhost:8080/health

# Status
curl http://localhost:8080/status

# Metrics (Prometheus format)
curl http://localhost:9090/metrics
```

Expected output:
```json
{
  "status": "healthy",
  "uptime": "5m30s",
  "connections": 0,
  "max_connections": 1000,
  "utilization_pct": 0.0,
  "region": "us-east-1",
  "server_name": "relay-us-east-1"
}
```

### 7. Firewall Configuration

```bash
# Allow relay port
sudo ufw allow 9545/tcp comment 'ShadowMesh Relay'

# Allow health check (internal only)
sudo ufw allow from 10.0.0.0/8 to any port 8080 comment 'Health Check'

# Allow metrics (internal only)
sudo ufw allow from 10.0.0.0/8 to any port 9090 comment 'Metrics'

# Enable firewall
sudo ufw enable
```

---

## Client Daemon Deployment

### Linux Client

#### Installation

```bash
# Download binary
wget https://github.com/shadowmesh/shadowmesh/releases/download/v0.2.0/shadowmesh-daemon-linux-amd64

# Install
sudo mv shadowmesh-daemon-linux-amd64 /usr/local/bin/shadowmesh-daemon
sudo chmod +x /usr/local/bin/shadowmesh-daemon
```

#### Configuration

Create `/etc/shadowmesh/client-config.yaml`:

```yaml
daemon:
  listen_address: "127.0.0.1:9090"
  log_level: "info"

network:
  mode: "tun"
  device_name: "shadowmesh0"
  local_ip: "10.10.10.X/24"  # Unique IP for each client

encryption:
  key: "YOUR_64_CHAR_HEX_KEY"  # Generate with: openssl rand -hex 32

peer:
  address: ""
  id: "client-unique-id"  # Unique identifier

nat:
  enabled: false

relay:
  enabled: true
  server: "ws://YOUR_RELAY_IP:9545"  # Your relay server

p2p:
  listener_enabled: false
  listener_port: 0
```

#### Systemd Service

Create `/etc/systemd/system/shadowmesh.service`:

```ini
[Unit]
Description=ShadowMesh Daemon
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/shadowmesh-daemon /etc/shadowmesh/client-config.yaml

Restart=always
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable shadowmesh
sudo systemctl start shadowmesh
sudo systemctl status shadowmesh
```

### macOS Client

#### Installation

```bash
# Download binary (Apple Silicon)
curl -L -o shadowmesh-daemon https://github.com/shadowmesh/shadowmesh/releases/download/v0.2.0/shadowmesh-daemon-darwin-arm64

# Or Intel
curl -L -o shadowmesh-daemon https://github.com/shadowmesh/shadowmesh/releases/download/v0.2.0/shadowmesh-daemon-darwin-amd64

# Install
sudo mv shadowmesh-daemon /usr/local/bin/
sudo chmod +x /usr/local/bin/shadowmesh-daemon
```

#### Configuration

Create `/etc/shadowmesh/client-config.yaml` (same format as Linux)

#### Launch Daemon (as service)

Create `/Library/LaunchDaemons/com.shadowmesh.daemon.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.shadowmesh.daemon</string>

    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/shadowmesh-daemon</string>
        <string>/etc/shadowmesh/client-config.yaml</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>StandardOutPath</key>
    <string>/var/log/shadowmesh.log</string>

    <key>StandardErrorPath</key>
    <string>/var/log/shadowmesh-error.log</string>
</dict>
</plist>
```

Load the service:

```bash
sudo launchctl load /Library/LaunchDaemons/com.shadowmesh.daemon.plist
sudo launchctl start com.shadowmesh.daemon
```

### Windows Client

#### Installation

1. Download `shadowmesh-daemon-windows-amd64.exe` from releases
2. Create directory: `C:\Program Files\ShadowMesh\`
3. Copy binary to: `C:\Program Files\ShadowMesh\shadowmesh-daemon.exe`
4. Create config directory: `C:\ProgramData\ShadowMesh\`
5. Create `C:\ProgramData\ShadowMesh\client-config.yaml`

#### Configuration

Same YAML format as Linux/macOS.

#### Windows Service Installation

Using NSSM (Non-Sucking Service Manager):

```powershell
# Download NSSM
Invoke-WebRequest -Uri "https://nssm.cc/release/nssm-2.24.zip" -OutFile "nssm.zip"
Expand-Archive nssm.zip
cd nssm\win64

# Install service
.\nssm.exe install ShadowMesh "C:\Program Files\ShadowMesh\shadowmesh-daemon.exe" "C:\ProgramData\ShadowMesh\client-config.yaml"

# Start service
Start-Service ShadowMesh
```

---

## Multi-Region Relay Deployment

For global coverage, deploy relays in multiple regions:

### Recommended Regions

1. **US East** (Virginia) - Primary
2. **US West** (Oregon) - Secondary
3. **EU West** (London) - EMEA
4. **Asia Pacific** (Singapore) - APAC

### Configuration

Each relay gets unique `region` and `server_name` in config:

```yaml
# US East
region: "us-east-1"
server_name: "relay-us-east-1"

# EU West
region: "eu-west-2"
server_name: "relay-eu-west-2"
```

### Client Selection

Clients should connect to nearest relay for lowest latency. Future releases will support automatic region selection.

---

## Monitoring

### Prometheus Metrics

Available at `http://relay-ip:9090/metrics`:

```
# Connection metrics
shadowmesh_relay_connections{region="us-east-1"} 42
shadowmesh_relay_max_connections{region="us-east-1"} 1000
shadowmesh_relay_utilization_percent{region="us-east-1"} 4.2

# Frame metrics
shadowmesh_relay_frames_total{region="us-east-1"} 125634
shadowmesh_relay_frames_dropped{region="us-east-1"} 0
shadowmesh_relay_bytes_relayed{region="us-east-1"} 52428800

# Performance
shadowmesh_relay_latency_ms{region="us-east-1",quantile="0.5"} 1.2
shadowmesh_relay_latency_ms{region="us-east-1",quantile="0.99"} 5.8
```

### Grafana Dashboard

Import the included Grafana dashboard from `monitoring/grafana-dashboard.json`.

### Alerting

Recommended alerts:

```yaml
# High connection utilization
- alert: RelayHighUtilization
  expr: shadowmesh_relay_utilization_percent > 80
  for: 5m

# Frame drops
- alert: RelayFrameDrops
  expr: rate(shadowmesh_relay_frames_dropped[5m]) > 0.01
  for: 2m

# Relay down
- alert: RelayDown
  expr: up{job="shadowmesh-relay"} == 0
  for: 1m
```

---

## Security Hardening

### 1. TLS/HTTPS Support

For production, use TLS termination with nginx:

```nginx
upstream shadowmesh_relay {
    server 127.0.0.1:9545;
}

server {
    listen 443 ssl http2;
    server_name relay.shadowmesh.io;

    ssl_certificate /etc/letsencrypt/live/relay.shadowmesh.io/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/relay.shadowmesh.io/privkey.pem;

    location /relay {
        proxy_pass http://shadowmesh_relay;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

Update client config:

```yaml
relay:
  enabled: true
  server: "wss://relay.shadowmesh.io/relay"  # WSS for TLS
```

### 2. Rate Limiting

Protect against DoS:

```bash
# iptables rate limiting
sudo iptables -A INPUT -p tcp --dport 9545 -m state --state NEW -m recent --set
sudo iptables -A INPUT -p tcp --dport 9545 -m state --state NEW -m recent --update --seconds 60 --hitcount 20 -j DROP
```

### 3. Authentication (Future)

v0.3.0 will add API key authentication:

```yaml
relay:
  enabled: true
  server: "wss://relay.shadowmesh.io/relay"
  api_key: "YOUR_API_KEY"
```

---

## Scaling

### Vertical Scaling

Single relay server capacity:

| Spec | Connections | Throughput |
|------|-------------|------------|
| 2 vCPU, 4GB RAM | ~500 | 2 Gbps |
| 4 vCPU, 8GB RAM | ~1000 | 5 Gbps |
| 8 vCPU, 16GB RAM | ~2000 | 10 Gbps |

### Horizontal Scaling

Deploy multiple relay servers:

1. Use DNS round-robin or load balancer
2. Clients distributed across relays
3. No state sharing required (stateless relay)

Example DNS:

```
relay.shadowmesh.io  A  203.0.113.10  (US East)
relay.shadowmesh.io  A  198.51.100.20  (EU West)
relay.shadowmesh.io  A  192.0.2.30     (APAC)
```

---

## Troubleshooting

### Relay Server Issues

```bash
# Check if relay is listening
sudo netstat -tulpn | grep 9545

# Check logs
sudo journalctl -u shadowmesh-relay -n 100 --no-pager

# Test WebSocket connection
wscat -c ws://localhost:9545/relay?peer_id=test
```

### Client Issues

```bash
# Check TUN device
ip link show shadowmesh0  # Linux
ifconfig utun0            # macOS

# Check routing
ip route show dev shadowmesh0  # Linux
netstat -rn | grep utun0      # macOS

# Test connectivity
ping 10.10.10.1  # Ping another client
```

### Common Issues

**1. "Connection refused" on relay**
- Check firewall: `sudo ufw status`
- Verify relay is running: `systemctl status shadowmesh-relay`

**2. "Permission denied" creating TUN**
- Run daemon with sudo/admin privileges
- Check TUN kernel module: `lsmod | grep tun`

**3. "Frame drops" on relay**
- Increase queue_size in relay config
- Scale vertically or horizontally

---

## Next Steps

**Upcoming Features** (v0.3.0+):

1. **Management Layer**: Web UI for network management
2. **Private Networks**: User-controlled isolated networks
3. **Local Controller**: On-premise management server
4. **P2P Direct**: Automatic direct connections when possible
5. **Authentication**: API key and OAuth support
6. **Mobile Apps**: iOS and Android clients

See `MANAGEMENT_LAYER.md` for architecture details.

---

## Support

- **Issues**: https://github.com/shadowmesh/shadowmesh/issues
- **Docs**: https://docs.shadowmesh.io
- **Community**: https://discord.gg/shadowmesh
