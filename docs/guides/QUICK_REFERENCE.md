# ShadowMesh Quick Reference Guide

<img src="https://pbs.twimg.com/profile_images/1969957304679473152/QW21M-FO_400x400.jpg" alt="Chronara Group Logo" width="80" align="right"/>

**Chronara Group ShadowMesh - Quick Reference**

## Quick Commands

### Development Setup

```bash
# Initialize Go module
go mod init github.com/yourusername/shadowmesh
go mod tidy

# Run tests
go test ./... -v -cover

# Run with race detector
go test ./... -race

# Run benchmarks
go test -bench=. -benchmem ./...

# Security scan
gosec ./...

# Lint code
golangci-lint run
```

### Docker Commands

```bash
# Build image
docker build -t shadowmesh:latest .

# Run container
docker run -d -p 8080:8080 shadowmesh:latest

# View logs
docker logs -f <container_id>

# Docker Compose
docker-compose up -d
docker-compose down
docker-compose logs -f
```

### Terraform Commands

```bash
# Initialize
terraform init

# Format
terraform fmt -recursive

# Validate
terraform validate

# Plan
terraform plan -out=tfplan

# Apply
terraform apply tfplan

# Destroy
terraform destroy

# View outputs
terraform output

# Import existing resource
terraform import aws_s3_bucket.main my-bucket-name
```

### AWS CLI

```bash
# Configure credentials
aws configure

# Test credentials
aws sts get-caller-identity

# List S3 buckets
aws s3 ls

# Copy to S3
aws s3 cp file.txt s3://my-bucket/

# Get VPN status
aws ec2 describe-vpn-connections

# KMS key info
aws kms describe-key --key-id alias/my-key
```

---

## 📝 Essential Configurations

### Go Project Structure

```
shadowmesh/
├── cmd/
│   ├── client/          # Client application
│   ├── relay/           # Relay node
│   └── cli/             # CLI tool
├── pkg/
│   ├── protocol/        # Frame protocol
│   ├── crypto/          # Ed25519, ChaCha20
│   ├── websocket/       # WS client/server
│   ├── blockchain/      # Smart contract integration
│   └── cloud/           # Cloud provider APIs
├── internal/
│   └── config/          # Configuration
├── test/
│   ├── integration/     # Integration tests
│   └── e2e/             # End-to-end tests
├── terraform/           # Infrastructure as code
├── contracts/           # Solidity contracts
├── go.mod
├── go.sum
├── Dockerfile
└── docker-compose.yml
```

### Environment Variables

```bash
# .env file
SHADOWMESH_ENV=development
LOG_LEVEL=debug
LISTEN_ADDR=0.0.0.0:8080

# Blockchain
ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_KEY
PRIVATE_KEY_PATH=/path/to/keystore.json

# AWS
AWS_REGION=us-east-1
AWS_PROFILE=shadowmesh

# Encryption
KMS_KEY_ID=alias/shadowmesh-encryption

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=shadowmesh
DB_USER=app
DB_PASSWORD=changeme  # Use Secrets Manager in production
```

### strongSwan Quick Config

```
# /etc/ipsec.conf (minimal)
conn aws
    auto=start
    type=tunnel
    keyexchange=ikev2
    left=%defaultroute
    leftsubnet=192.168.0.0/16
    right=AWS_VPN_IP
    rightsubnet=10.0.0.0/16
    ike=aes256gcm128-sha384-ecp384!
    esp=aes256gcm128-ecp384!
```

### Terraform Variables

```hcl
# terraform.tfvars
aws_region  = "us-east-1"
environment = "prod"
bucket_name = "shadowmesh-data-prod"

authorized_role_arns = [
  "arn:aws:iam::123456789012:role/AppRole"
]

tags = {
  Project = "ShadowMesh"
  Owner   = "Platform Team"
}
```

---

## 🔑 Cryptography Quick Reference

### Ed25519 Key Generation (Go)

```go
import "crypto/ed25519"

// Generate keypair
publicKey, privateKey, err := ed25519.GenerateKey(nil)

// Sign message
signature := ed25519.Sign(privateKey, message)

// Verify signature
valid := ed25519.Verify(publicKey, message, signature)
```

### ChaCha20-Poly1305 Encryption (Go)

```go
import "golang.org/x/crypto/chacha20poly1305"

// Create cipher
aead, err := chacha20poly1305.NewX(key[:])

// Generate nonce
nonce := make([]byte, chacha20poly1305.NonceSizeX)
rand.Read(nonce)

// Encrypt
ciphertext := aead.Seal(nil, nonce, plaintext, nil)

// Decrypt
plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
```

---

## 🌐 Network Configuration

### Firewall Rules (iptables)

```bash
# Allow VPN traffic
iptables -A INPUT -p udp --dport 500 -j ACCEPT   # IKE
iptables -A INPUT -p udp --dport 4500 -j ACCEPT  # NAT-T
iptables -A INPUT -p esp -j ACCEPT               # ESP

# Allow WebSocket
iptables -A INPUT -p tcp --dport 443 -j ACCEPT   # WSS

# NAT for outbound
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
```

### Testing Connectivity

```bash
# Ping test
ping -c 5 10.0.1.10

# TCP port test
telnet 10.0.1.10 443
nc -zv 10.0.1.10 443

# Bandwidth test
iperf3 -s                  # Server
iperf3 -c 10.0.1.10        # Client

# Traceroute
traceroute 10.0.1.10

# MTU test
ping -M do -s 1400 10.0.1.10
```

---

## 🔍 Debugging

### strongSwan Logs

```bash
# View logs
journalctl -u strongswan-starter -f

# Status
ipsec status
ipsec statusall

# Traffic counters
ipsec statusall | grep bytes
```

### Go Debugging

```go
import "log"

// Simple logging
log.Printf("Connection from %s", addr)

// With stack trace
log.Println(string(debug.Stack()))

// Conditional logging
if os.Getenv("DEBUG") == "true" {
    log.Printf("Debug: %+v", data)
}
```

### Network Debugging

```bash
# Capture packets
tcpdump -i eth0 -w capture.pcap
tcpdump -i eth0 esp

# Analyze in Wireshark
wireshark capture.pcap

# Check routes
ip route show
ip route get 10.0.1.10

# Check connections
ss -tuln
netstat -tuln
```

---

## 📊 Monitoring

### Prometheus Metrics (Go)

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    connectionsTotal = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "shadowmesh_connections_total",
            Help: "Total number of connections",
        },
    )
)

func init() {
    prometheus.MustRegister(connectionsTotal)
}

// Increment
connectionsTotal.Inc()
```

### AWS CloudWatch

```bash
# Get metrics
aws cloudwatch get-metric-statistics \
    --namespace AWS/VPN \
    --metric-name TunnelState \
    --dimensions Name=VpnId,Value=vpn-xxx \
    --start-time 2024-01-01T00:00:00Z \
    --end-time 2024-01-02T00:00:00Z \
    --period 3600 \
    --statistics Average

# Put custom metric
aws cloudwatch put-metric-data \
    --namespace ShadowMesh \
    --metric-name ConnectionCount \
    --value 42
```

---

## 🔐 Security Checks

### Quick Security Audit

```bash
# Check for hardcoded secrets
grep -r "password\|secret\|key" . | grep -v ".git\|node_modules"

# Go security scan
gosec ./...

# Dependency check
go list -json -m all | nancy sleuth

# Container scanning
docker scan shadowmesh:latest

# Terraform security
tfsec .
checkov -d .
```

### Key Rotation Script

```bash
#!/bin/bash
# rotate-keys.sh

OLD_KEY_ID=$(aws kms describe-key --key-id alias/shadowmesh | jq -r .KeyMetadata.KeyId)

# Create new key
NEW_KEY_ID=$(aws kms create-key --description "Rotated key" | jq -r .KeyMetadata.KeyId)

# Update alias
aws kms update-alias --alias-name alias/shadowmesh --target-key-id $NEW_KEY_ID

# Schedule old key deletion (30 days)
aws kms schedule-key-deletion --key-id $OLD_KEY_ID --pending-window-in-days 30
```

---

## 💾 Backup and Recovery

### Backup strongSwan Config

```bash
#!/bin/bash
BACKUP_DIR="/backup/vpn/$(date +%Y%m%d)"
mkdir -p $BACKUP_DIR

cp /etc/ipsec.conf $BACKUP_DIR/
cp /etc/ipsec.secrets $BACKUP_DIR/
chmod 600 $BACKUP_DIR/ipsec.secrets

# Upload to S3
aws s3 cp $BACKUP_DIR/ s3://my-backups/vpn/ --recursive
```

### Export Terraform State

```bash
# Pull state
terraform state pull > terraform.tfstate.backup

# Upload to S3
aws s3 cp terraform.tfstate.backup s3://my-backups/terraform/
```

---

## 🎯 Performance Tuning

### System Parameters

```bash
# /etc/sysctl.conf
net.ipv4.ip_forward = 1
net.ipv4.tcp_window_scaling = 1
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.ipv4.tcp_rmem = 4096 87380 67108864
net.ipv4.tcp_wmem = 4096 65536 67108864

# Apply
sysctl -p
```

### Go Performance

```go
// Use buffer pools
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}

buf := bufferPool.Get().([]byte)
defer bufferPool.Put(buf)

// Enable pprof
import _ "net/http/pprof"
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

---

## 📱 Common Scenarios

### Scenario 1: Deploy New Relay Node

```bash
# 1. Update Terraform
cd terraform/aws
terraform apply -target=aws_instance.relay[3]

# 2. Configure node
ssh ubuntu@NEW_IP
curl -sSL https://get.docker.com | sh
docker pull shadowmesh/relay:latest
docker run -d shadowmesh/relay:latest

# 3. Update DNS
aws route53 change-resource-record-sets \
    --hosted-zone-id Z123 \
    --change-batch file://dns-change.json
```

### Scenario 2: Investigate Connection Issues

```bash
# 1. Check tunnel status
ipsec status

# 2. Check AWS side
aws ec2 describe-vpn-connections --vpn-connection-ids vpn-xxx

# 3. Check logs
journalctl -u strongswan-starter -n 100

# 4. Test connectivity
ping 10.0.1.10

# 5. Check routing
ip route show table all
```

### Scenario 3: Rotate Blockchain Keys

```bash
# 1. Generate new keypair
shadowmesh keygen --output new-key.json

# 2. Register new device
shadowmesh register --key new-key.json

# 3. Revoke old device
shadowmesh revoke --key old-key.json

# 4. Update configuration
shadowmesh config set --key new-key.json
```

---

## 🆘 Emergency Procedures

### VPN Down

```bash
# 1. Check service
systemctl status strongswan-starter

# 2. Restart service
systemctl restart strongswan-starter

# 3. Check AWS VPN
aws ec2 describe-vpn-connections --vpn-connection-ids vpn-xxx

# 4. If still down, create new VPN connection
terraform apply -replace=aws_vpn_connection.main
```

### Security Incident

```bash
# 1. Isolate affected resources
aws ec2 modify-instance-attribute --instance-id i-xxx \
    --groups sg-emergency-isolation

# 2. Capture forensics
aws ssm send-command --instance-ids i-xxx \
    --document-name "AWS-RunShellScript" \
    --parameters 'commands=["ps aux > /tmp/processes.txt"]'

# 3. Review logs
aws logs tail /aws/ec2/security --since 1h --follow

# 4. Notify team
aws sns publish --topic-arn arn:aws:sns:us-east-1:xxx:security-alerts \
    --message "Security incident detected"
```

---

## 📚 Useful Links

- **Go Docs:** https://pkg.go.dev/
- **AWS Docs:** https://docs.aws.amazon.com/
- **Terraform Registry:** https://registry.terraform.io/
- **strongSwan Wiki:** https://wiki.strongswan.org/
- **Ethereum Docs:** https://ethereum.org/en/developers/

---

**Last Updated:** 2024  
**Version:** 1.0
