# ShadowMesh Bootstrap Node Deployment Plan

**Version**: 1.0
**Target Release**: v0.2.0-alpha
**Last Updated**: November 10, 2025
**Status**: Ready for Deployment

---

## Executive Summary

Bootstrap nodes are **critical infrastructure** for ShadowMesh's decentralized network. They provide initial peer discovery for new nodes joining the network, enabling the DHT to function without centralized coordination.

**Deployment Strategy**:
- **3 bootstrap nodes** across geographic regions
- **99.9% uptime** SLA (43.8 minutes downtime per month)
- **Static IP addresses** with DNS names
- **Automated monitoring** with alerting
- **Estimated cost**: $30-45/month total

---

## Bootstrap Node Requirements

### Hardware Specifications

**Minimum Requirements** (small VPS):
```
CPU:      2 vCPUs
RAM:      2 GB
Storage:  20 GB SSD
Network:  1 Gbps, unmetered bandwidth
OS:       Ubuntu 22.04 LTS (or Debian 12)
```

**Recommended for Production**:
```
CPU:      4 vCPUs
RAM:      4 GB
Storage:  40 GB SSD
Network:  10 Gbps, unmetered
OS:       Ubuntu 22.04 LTS
```

**Rationale**: Bootstrap nodes handle high connection volume but minimal data transfer (just DHT operations, not tunnel traffic).

### Network Requirements

- **Static public IP address** (no dynamic IP)
- **Open ports**:
  - `8443/udp` - DHT protocol messages
  - `9443/udp` - ShadowMesh tunnel connections (for testing)
  - `22/tcp` - SSH management (restrict to admin IPs)
  - `9090/tcp` - Prometheus metrics (localhost only or VPN)
- **IPv4 and IPv6** support (optional but recommended)
- **DDoS protection** (Cloudflare, AWS Shield, or provider-level)

### Geographic Distribution

Deploy nodes across 3 regions to ensure global coverage and redundancy:

**Region 1: United States (East Coast)**
- Location: New York, Virginia, or Toronto
- Provider: AWS, DigitalOcean, or Linode
- Serves: North America, South America

**Region 2: Europe**
- Location: London, Frankfurt, or Amsterdam
- Provider: AWS, Hetzner, or OVH
- Serves: Europe, Middle East, Africa

**Region 3: Asia-Pacific**
- Location: Singapore, Tokyo, or Sydney
- Provider: AWS, DigitalOcean, or Vultr
- Serves: Asia, Australia, Oceania

**Reasoning**: Users connect to geographically closest bootstrap node first, reducing latency for initial DHT lookups.

---

## Deployment Architecture

### Node Configuration

```yaml
# bootstrap-node-config.yaml
node:
  name: "bootstrap1.shadowmesh.net"
  region: "us-east"
  port: 8443

keypair:
  # Pre-generated ML-DSA-87 keypair
  private_key_file: "/etc/shadowmesh/bootstrap-private.key"
  public_key_file: "/etc/shadowmesh/bootstrap-public.key"
  peer_id: "2f8a3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b"

bootstrap:
  # Other bootstrap nodes (for redundancy)
  peers:
    - peer_id: "3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b"
      address: "bootstrap2.shadowmesh.net:8443"
    - peer_id: "4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5"
      address: "bootstrap3.shadowmesh.net:8443"

dht:
  k: 20                          # Max peers per k-bucket
  alpha: 3                       # Parallel lookup requests
  routing_table_refresh: 1h      # Refresh routing table every hour
  liveness_check_interval: 15m   # Check peer liveness every 15 minutes

logging:
  level: "info"
  file: "/var/log/shadowmesh/bootstrap.log"
  max_size_mb: 100
  max_backups: 10

monitoring:
  prometheus_port: 9090
  health_check_port: 9091
```

### DNS Configuration

Create DNS A records for each bootstrap node:

```
bootstrap1.shadowmesh.net  →  [US-EAST-IP]
bootstrap2.shadowmesh.net  →  [EU-WEST-IP]
bootstrap3.shadowmesh.net  →  [APAC-SOUTHEAST-IP]

# Optional: Round-robin DNS for load balancing
bootstrap.shadowmesh.net   →  [US-EAST-IP, EU-WEST-IP, APAC-SOUTHEAST-IP]
```

**TTL**: 300 seconds (5 minutes) for quick failover

**DNSSEC**: Recommended for preventing DNS spoofing attacks

---

## Deployment Process

### Step 1: Pre-Deployment Preparation

#### Generate ML-DSA-87 Keypairs

```bash
# Generate keypairs for each bootstrap node
shadowmesh-keygen --output bootstrap1-keypair.json
shadowmesh-keygen --output bootstrap2-keypair.json
shadowmesh-keygen --output bootstrap3-keypair.json

# Extract PeerIDs
shadowmesh-keygen --peer-id bootstrap1-keypair.json
# Output: 2f8a3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b

# Store keypairs securely (use secrets manager in production)
```

**Security**: Private keys should be encrypted at rest and never committed to git.

#### Provision VPS Instances

**Using Terraform** (recommended):

```hcl
# bootstrap-infra.tf
provider "aws" {
  region = "us-east-1"
}

resource "aws_instance" "bootstrap1" {
  ami           = "ami-0c55b159cbfafe1f0"  # Ubuntu 22.04
  instance_type = "t3.small"

  tags = {
    Name = "shadowmesh-bootstrap1"
    Role = "bootstrap-node"
  }

  security_group_ids = [aws_security_group.bootstrap.id]

  user_data = file("bootstrap-init.sh")
}

resource "aws_security_group" "bootstrap" {
  name = "shadowmesh-bootstrap"

  ingress {
    from_port   = 8443
    to_port     = 8443
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["YOUR_ADMIN_IP/32"]  # Restrict SSH
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
```

**Manual Provisioning** (alternative):
- DigitalOcean: Create Droplet, Ubuntu 22.04, 2 vCPU / 2 GB RAM
- Linode: Create Linode, Ubuntu 22.04, Linode 2GB plan
- AWS: EC2 t3.small instance

### Step 2: Server Setup

#### Install Dependencies

```bash
# SSH into server
ssh root@bootstrap1-ip

# Update system
apt update && apt upgrade -y

# Install required packages
apt install -y curl wget git build-essential ufw fail2ban

# Configure firewall
ufw allow 22/tcp
ufw allow 8443/udp
ufw allow 9443/udp
ufw enable

# Install Docker (optional, for containerized deployment)
curl -fsSL https://get.docker.com | sh
systemctl enable docker
systemctl start docker
```

#### Install ShadowMesh Binary

```bash
# Download latest v0.2.0-alpha binary
wget https://github.com/shadowmesh/shadowmesh/releases/download/v0.2.0-alpha/shadowmesh-v0.2.0-alpha-linux-amd64

# Verify checksum
sha256sum shadowmesh-v0.2.0-alpha-linux-amd64
# Compare with published checksum

# Install binary
chmod +x shadowmesh-v0.2.0-alpha-linux-amd64
mv shadowmesh-v0.2.0-alpha-linux-amd64 /usr/local/bin/shadowmesh

# Verify installation
shadowmesh --version
# Output: ShadowMesh v0.2.0-alpha (DHT + PQC)
```

#### Deploy Configuration

```bash
# Create directory structure
mkdir -p /etc/shadowmesh
mkdir -p /var/log/shadowmesh

# Copy configuration
cat > /etc/shadowmesh/config.yaml <<EOF
node:
  name: "bootstrap1.shadowmesh.net"
  region: "us-east"
  port: 8443

keypair:
  private_key_file: "/etc/shadowmesh/bootstrap-private.key"
  public_key_file: "/etc/shadowmesh/bootstrap-public.key"

bootstrap:
  peers:
    - peer_id: "3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b"
      address: "bootstrap2.shadowmesh.net:8443"
    - peer_id: "4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5"
      address: "bootstrap3.shadowmesh.net:8443"

dht:
  k: 20
  alpha: 3

logging:
  level: "info"
  file: "/var/log/shadowmesh/bootstrap.log"
EOF

# Copy ML-DSA-87 keypair (secure transfer via SCP)
scp bootstrap1-keypair.json root@bootstrap1-ip:/tmp/
ssh root@bootstrap1-ip "cat /tmp/bootstrap1-keypair.json | jq -r '.private_key' > /etc/shadowmesh/bootstrap-private.key"
ssh root@bootstrap1-ip "cat /tmp/bootstrap1-keypair.json | jq -r '.public_key' > /etc/shadowmesh/bootstrap-public.key"

# Secure permissions
chmod 600 /etc/shadowmesh/bootstrap-private.key
chmod 644 /etc/shadowmesh/bootstrap-public.key
```

### Step 3: Create Systemd Service

```bash
# Create systemd service file
cat > /etc/systemd/system/shadowmesh-bootstrap.service <<EOF
[Unit]
Description=ShadowMesh Bootstrap Node
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/shadowmesh bootstrap --config /etc/shadowmesh/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=shadowmesh-bootstrap

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd
systemctl daemon-reload

# Enable and start service
systemctl enable shadowmesh-bootstrap
systemctl start shadowmesh-bootstrap

# Check status
systemctl status shadowmesh-bootstrap

# View logs
journalctl -u shadowmesh-bootstrap -f
```

### Step 4: Verify Deployment

```bash
# Check if process is running
ps aux | grep shadowmesh

# Check UDP port is listening
netstat -ulnp | grep 8443

# Test DHT connectivity
shadowmesh-cli ping bootstrap1.shadowmesh.net:8443

# Check routing table population
shadowmesh-cli routing-table --node bootstrap1.shadowmesh.net:8443
# Should show peers from bootstrap2 and bootstrap3
```

---

## Monitoring & Alerting

### Prometheus Metrics Exporter

Bootstrap nodes expose Prometheus metrics on port 9090 (localhost only):

```yaml
# Exposed metrics
shadowmesh_dht_peers_total             # Total peers in routing table
shadowmesh_dht_requests_total          # Total DHT requests handled (by type)
shadowmesh_dht_lookup_latency_seconds  # DHT lookup latency histogram
shadowmesh_uptime_seconds              # Node uptime
shadowmesh_memory_usage_bytes          # Memory usage
shadowmesh_cpu_usage_percent           # CPU usage
```

### Grafana Dashboard

```bash
# Install Grafana (optional, can use hosted Grafana Cloud)
apt install -y grafana
systemctl enable grafana-server
systemctl start grafana-server

# Access Grafana: http://bootstrap1-ip:3000
# Default credentials: admin/admin

# Import ShadowMesh bootstrap dashboard
# Dashboard ID: TBD (will be published with v0.2.0-alpha)
```

### Uptime Monitoring

**Option 1: UptimeRobot** (Free tier: 50 monitors)
```
Monitor Type: Port
Host: bootstrap1.shadowmesh.net
Port: 8443
Protocol: UDP
Check Interval: 5 minutes
Alerts: Email, Slack, PagerDuty
```

**Option 2: Pingdom / StatusCake**
```
Similar configuration
More detailed monitoring and reporting
```

**Option 3: Custom Health Check Script**
```bash
#!/bin/bash
# /usr/local/bin/bootstrap-health-check.sh

BOOTSTRAP_URL="bootstrap1.shadowmesh.net:8443"

# Attempt DHT PING
shadowmesh-cli ping $BOOTSTRAP_URL --timeout 5s

if [ $? -eq 0 ]; then
  echo "OK: Bootstrap node is responsive"
  exit 0
else
  echo "CRITICAL: Bootstrap node is down"
  # Send alert (email, Slack webhook, PagerDuty API)
  curl -X POST "https://hooks.slack.com/services/YOUR/WEBHOOK/URL" \
    -H 'Content-Type: application/json' \
    -d '{"text":"🚨 Bootstrap1 is down!"}'
  exit 2
fi
```

Schedule with cron:
```bash
# Run health check every 5 minutes
*/5 * * * * /usr/local/bin/bootstrap-health-check.sh >> /var/log/shadowmesh/health-check.log 2>&1
```

---

## Security Hardening

### SSH Security

```bash
# Disable root login
sed -i 's/PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config

# Disable password authentication (use SSH keys only)
sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config

# Change SSH port (optional obfuscation)
sed -i 's/#Port 22/Port 2222/' /etc/ssh/sshd_config

# Restart SSH
systemctl restart sshd
```

### Fail2Ban Configuration

```bash
# Install fail2ban
apt install -y fail2ban

# Configure jail for SSH
cat > /etc/fail2ban/jail.d/shadowmesh.conf <<EOF
[sshd]
enabled = true
port = 22
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
bantime = 3600
findtime = 600
EOF

# Start fail2ban
systemctl enable fail2ban
systemctl start fail2ban
```

### Rate Limiting (at Application Layer)

DHT message handler already implements rate limiting (100 msg/sec per peer), but add iptables rules for DDoS protection:

```bash
# Limit UDP connections to DHT port
iptables -A INPUT -p udp --dport 8443 -m state --state NEW -m limit --limit 100/second --limit-burst 200 -j ACCEPT
iptables -A INPUT -p udp --dport 8443 -j DROP

# Save rules
iptables-save > /etc/iptables/rules.v4
```

### Automatic Security Updates

```bash
# Enable unattended upgrades
apt install -y unattended-upgrades

# Configure
cat > /etc/apt/apt.conf.d/50unattended-upgrades <<EOF
Unattended-Upgrade::Allowed-Origins {
    "\${distro_id}:\${distro_codename}-security";
};
Unattended-Upgrade::Automatic-Reboot "true";
Unattended-Upgrade::Automatic-Reboot-Time "03:00";
EOF

# Enable automatic updates
systemctl enable unattended-upgrades
systemctl start unattended-upgrades
```

---

## Backup & Disaster Recovery

### Configuration Backup

```bash
# Backup script
#!/bin/bash
# /usr/local/bin/backup-bootstrap-config.sh

BACKUP_DIR="/root/backups"
DATE=$(date +%Y%m%d-%H%M%S)

mkdir -p $BACKUP_DIR

# Backup configuration
tar -czf $BACKUP_DIR/shadowmesh-config-$DATE.tar.gz \
  /etc/shadowmesh/ \
  /var/log/shadowmesh/

# Keep only last 30 days of backups
find $BACKUP_DIR -name "shadowmesh-config-*.tar.gz" -mtime +30 -delete

echo "Backup completed: $BACKUP_DIR/shadowmesh-config-$DATE.tar.gz"
```

Schedule daily backups:
```bash
# Run backup daily at 2 AM
0 2 * * * /usr/local/bin/backup-bootstrap-config.sh >> /var/log/shadowmesh/backup.log 2>&1
```

### Disaster Recovery Plan

**Scenario 1: Bootstrap node becomes unresponsive**
1. Other 2 bootstrap nodes continue serving (redundancy)
2. Alert triggers via monitoring
3. Investigate via SSH or console access
4. Restart service: `systemctl restart shadowmesh-bootstrap`
5. If hardware failure, deploy new node from backup

**Scenario 2: Complete node loss (server destroyed)**
1. Provision new VPS in same region
2. Restore configuration from backup
3. Update DNS A record to new IP
4. Verify connectivity from other bootstrap nodes
5. Resume operation (5-15 minutes total downtime)

**Scenario 3: DDoS attack**
1. Enable DDoS protection at provider level
2. Temporarily restrict traffic to known good peers
3. Increase rate limiting thresholds if false positives
4. Consider moving to DDoS-resistant provider (Cloudflare, AWS Shield)

---

## Maintenance Procedures

### Regular Maintenance Schedule

**Daily**:
- Automated health checks (via monitoring)
- Log rotation (logrotate handles this automatically)

**Weekly**:
- Review logs for errors: `journalctl -u shadowmesh-bootstrap --since "1 week ago" | grep ERROR`
- Check disk space: `df -h`
- Review routing table size: `shadowmesh-cli routing-table --count`

**Monthly**:
- Security updates: `apt update && apt upgrade`
- Review Prometheus metrics for anomalies
- Test disaster recovery procedure (restore from backup)

**Quarterly**:
- Rotate SSH keys
- Review and update firewall rules
- Performance audit (CPU, RAM, network usage trends)

### Updating ShadowMesh Binary

```bash
# Download new version
wget https://github.com/shadowmesh/shadowmesh/releases/download/v0.2.1-alpha/shadowmesh-v0.2.1-alpha-linux-amd64

# Verify checksum
sha256sum shadowmesh-v0.2.1-alpha-linux-amd64

# Stop service
systemctl stop shadowmesh-bootstrap

# Backup old binary
cp /usr/local/bin/shadowmesh /usr/local/bin/shadowmesh.backup

# Replace binary
mv shadowmesh-v0.2.1-alpha-linux-amd64 /usr/local/bin/shadowmesh
chmod +x /usr/local/bin/shadowmesh

# Start service
systemctl start shadowmesh-bootstrap

# Verify
systemctl status shadowmesh-bootstrap
shadowmesh --version
```

**Rolling Update Strategy** (zero downtime):
1. Update bootstrap3 first
2. Wait 10 minutes, monitor
3. Update bootstrap2
4. Wait 10 minutes, monitor
5. Update bootstrap1
6. Verify all nodes healthy

---

## Cost Analysis

### VPS Provider Comparison

**DigitalOcean** (Droplet):
```
Plan:      Basic - 2 vCPU / 2 GB RAM / 60 GB SSD
Cost:      $12/month per node
Total:     $36/month for 3 nodes
Bandwidth: 3 TB included per node
```

**Linode**:
```
Plan:      Linode 2GB - 1 vCPU / 2 GB RAM / 50 GB SSD
Cost:      $10/month per node
Total:     $30/month for 3 nodes
Bandwidth: 2 TB included per node
```

**AWS EC2** (with Reserved Instance):
```
Plan:      t3.small - 2 vCPU / 2 GB RAM / 20 GB EBS
Cost:      ~$15/month per node (1-year reserved)
Total:     $45/month for 3 nodes
Bandwidth: $0.09/GB egress (first 10 TB/month)
```

**Hetzner** (Europe only):
```
Plan:      CX11 - 1 vCPU / 2 GB RAM / 20 GB SSD
Cost:      €4.15/month (~$4.50 USD) per node
Total:     $13.50/month for 3 nodes
Bandwidth: 20 TB included per node
```

### Recommended Configuration

**For v0.2.0-alpha (Testing)**:
- **Linode**: $30/month total (best value)
- Deploy 3 nodes across US, EU, Asia
- Sufficient for 1,000-10,000 users

**For v1.0.0 (Production)**:
- **AWS EC2** or **DigitalOcean**: $45-50/month
- Upgrade to 4 vCPU / 4 GB RAM if needed
- Add DDoS protection ($10-20/month)
- Total: ~$60-70/month

---

## Deployment Checklist

### Pre-Deployment
- [ ] Generate ML-DSA-87 keypairs for all 3 bootstrap nodes
- [ ] Provision 3 VPS instances across geographic regions
- [ ] Configure DNS A records for each node
- [ ] Set up secrets management for private keys

### Node Setup
- [ ] Install Ubuntu 22.04 on all nodes
- [ ] Configure firewall rules (UFW)
- [ ] Install ShadowMesh v0.2.0-alpha binary
- [ ] Deploy configuration files
- [ ] Create systemd service
- [ ] Start and verify bootstrap service

### Monitoring
- [ ] Set up Prometheus metrics collection
- [ ] Configure Grafana dashboards
- [ ] Set up uptime monitoring (UptimeRobot or equivalent)
- [ ] Configure alerting (email, Slack, PagerDuty)
- [ ] Test alert delivery

### Security
- [ ] Harden SSH configuration
- [ ] Install and configure fail2ban
- [ ] Set up automatic security updates
- [ ] Configure iptables rate limiting
- [ ] Review and lock down unnecessary ports

### Testing
- [ ] Verify inter-bootstrap connectivity (all 3 nodes see each other)
- [ ] Test DHT PING between all node pairs
- [ ] Simulate client connection from test machine
- [ ] Verify routing table population
- [ ] Load test with 100 concurrent client connections

### Documentation
- [ ] Document IP addresses and DNS names
- [ ] Store keypair backups securely
- [ ] Create runbook for common issues
- [ ] Document disaster recovery procedures
- [ ] Share bootstrap node list with development team

---

## Troubleshooting Guide

### Issue: Bootstrap node not responding to DHT requests

**Symptoms**:
- Clients cannot connect to bootstrap node
- Other bootstrap nodes cannot PING this node

**Diagnosis**:
```bash
# Check if service is running
systemctl status shadowmesh-bootstrap

# Check if port is listening
netstat -ulnp | grep 8443

# Check logs for errors
journalctl -u shadowmesh-bootstrap -n 100
```

**Resolution**:
1. Restart service: `systemctl restart shadowmesh-bootstrap`
2. Check firewall rules: `ufw status`
3. Verify configuration: `cat /etc/shadowmesh/config.yaml`
4. Test UDP connectivity: `nc -u bootstrap1.shadowmesh.net 8443`

---

### Issue: High CPU usage on bootstrap node

**Symptoms**:
- CPU usage consistently >80%
- Slow response to DHT queries

**Diagnosis**:
```bash
# Check CPU usage
top -bn1 | grep shadowmesh

# Check number of connections
netstat -an | grep 8443 | wc -l

# Review Prometheus metrics
curl http://localhost:9090/metrics | grep cpu_usage
```

**Resolution**:
1. Increase instance size (upgrade to 4 vCPU)
2. Review routing table size (may need to limit k-bucket size)
3. Check for attack (excessive requests from single IP)
4. Enable more aggressive rate limiting

---

### Issue: DNS resolution failures

**Symptoms**:
- Clients report "cannot resolve bootstrap1.shadowmesh.net"
- Intermittent connectivity

**Diagnosis**:
```bash
# Test DNS resolution
dig bootstrap1.shadowmesh.net
nslookup bootstrap1.shadowmesh.net

# Check DNS propagation
dig bootstrap1.shadowmesh.net @8.8.8.8
```

**Resolution**:
1. Verify DNS A record is correct
2. Wait for DNS propagation (up to 48 hours, typically <1 hour)
3. Lower TTL to 300 seconds for faster updates
4. Use IP address temporarily as fallback

---

## Conclusion

This deployment plan provides a complete blueprint for deploying 3 production-ready bootstrap nodes for ShadowMesh v0.2.0-alpha. Key highlights:

✅ **99.9% uptime** through geographic redundancy
✅ **$30-45/month** total cost (affordable for alpha)
✅ **Automated monitoring** with alerting
✅ **Security hardened** with fail2ban, iptables, automatic updates
✅ **Disaster recovery** procedures documented

**Next Steps**:
1. Provision VPS instances (Linode recommended for alpha)
2. Generate ML-DSA-87 keypairs
3. Follow deployment process step-by-step
4. Verify all 3 nodes are healthy and connected
5. Publish bootstrap node list for v0.2.0-alpha release

---

**Document Control**
- Version: 1.0
- Created: November 10, 2025
- Author: Winston (Architect)
- Status: Ready for Deployment
- Next Review: After v0.2.0-alpha deployment
