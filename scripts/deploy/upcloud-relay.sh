#!/bin/bash
# Deploy ShadowMesh Relay Server to UpCloud
#
# Prerequisites:
#   - upcloud-cli installed
#   - upcloud-cli configured with credentials
#   - Go 1.23+ installed on local machine for building

set -e

echo "════════════════════════════════════════════════════════"
echo "ShadowMesh Relay Server - UpCloud Deployment"
echo "════════════════════════════════════════════════════════"
echo ""

# Configuration
SERVER_NAME="shadowmesh-relay"
ZONE="uk-lon1"  # London datacenter (change as needed)
PLAN="1xCPU-1GB"  # Smallest plan is sufficient
OS_TEMPLATE="Debian GNU/Linux 13 (Trixie)"
PORT=9545

echo "Configuration:"
echo "  Server name: $SERVER_NAME"
echo "  Zone: $ZONE"
echo "  Plan: $PLAN"
echo "  OS: $OS_TEMPLATE"
echo "  Port: $PORT"
echo ""

# Step 1: Check if upcloud CLI is installed
if ! command -v upctl &> /dev/null; then
    echo "❌ Error: upcloud-cli (upctl) is not installed"
    echo ""
    echo "Install with:"
    echo "  macOS: brew install upcloud/tap/upcloud-cli"
    echo "  Linux: curl -Lo upcloud https://github.com/UpCloudLtd/upcloud-cli/releases/latest/download/upcloud_linux_amd64.tar.gz && tar -xzf upcloud_linux_amd64.tar.gz && sudo mv upcloud /usr/local/bin/"
    echo ""
    exit 1
fi

echo "✅ upcloud-cli found: $(upctl version)"
echo ""

# Step 2: Build relay server binary
echo "Step 1: Building relay server binary..."
cd "$(dirname "$0")/../.."
go build -o bin/relay-server ./cmd/relay-server/
echo "✅ Binary built: bin/relay-server"
echo ""

# Step 3: Create server
echo "Step 2: Creating UpCloud server..."
echo "This will take 2-3 minutes..."
echo ""

# Create server and capture output
SERVER_JSON=$(upctl server create \
    --hostname "$SERVER_NAME" \
    --zone "$ZONE" \
    --plan "$PLAN" \
    --os "$OS_TEMPLATE" \
    --ssh-keys @$HOME/.ssh/id_rsa.pub \
    --format json)

# Extract server UUID and IP
SERVER_UUID=$(echo "$SERVER_JSON" | jq -r '.uuid')
SERVER_IP=$(echo "$SERVER_JSON" | jq -r '.networking.interfaces[0].ip_addresses[0].address')

if [ -z "$SERVER_IP" ] || [ "$SERVER_IP" == "null" ]; then
    echo "❌ Failed to create server or get IP address"
    echo "$SERVER_JSON"
    exit 1
fi

echo "✅ Server created"
echo "   UUID: $SERVER_UUID"
echo "   IP: $SERVER_IP"
echo ""

# Step 4: Wait for server to be ready
echo "Step 3: Waiting for server to be ready..."
sleep 30

MAX_RETRIES=20
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 root@$SERVER_IP "echo 'Server ready'" &>/dev/null; then
        echo "✅ Server is ready"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo "   Waiting... ($RETRY_COUNT/$MAX_RETRIES)"
    sleep 10
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo "❌ Server failed to become ready"
    exit 1
fi
echo ""

# Step 5: Open firewall port
echo "Step 4: Configuring firewall..."
upctl server firewall-rule create $SERVER_UUID \
    --direction in \
    --family IPv4 \
    --protocol tcp \
    --source-address-start 0.0.0.0 \
    --source-address-end 255.255.255.255 \
    --destination-port-start $PORT \
    --destination-port-end $PORT \
    --action accept \
    --comment "ShadowMesh Relay WebSocket" || true

# Allow SSH
upctl server firewall-rule create $SERVER_UUID \
    --direction in \
    --family IPv4 \
    --protocol tcp \
    --source-address-start 0.0.0.0 \
    --source-address-end 255.255.255.255 \
    --destination-port-start 22 \
    --destination-port-end 22 \
    --action accept \
    --comment "SSH" || true

echo "✅ Firewall configured"
echo ""

# Step 6: Install dependencies
echo "Step 5: Installing dependencies on server..."
ssh -o StrictHostKeyChecking=no root@$SERVER_IP << 'EOF'
    apt-get update -qq
    apt-get install -y wget tar
EOF
echo "✅ Dependencies installed"
echo ""

# Step 7: Install Go on server
echo "Step 6: Installing Go on server..."
ssh -o StrictHostKeyChecking=no root@$SERVER_IP << 'EOF'
    GO_VERSION="1.23.3"
    wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    rm go${GO_VERSION}.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
EOF
echo "✅ Go installed"
echo ""

# Step 8: Upload binary
echo "Step 7: Uploading relay server binary..."
scp -o StrictHostKeyChecking=no bin/relay-server root@$SERVER_IP:/usr/local/bin/
ssh -o StrictHostKeyChecking=no root@$SERVER_IP "chmod +x /usr/local/bin/relay-server"
echo "✅ Binary uploaded"
echo ""

# Step 9: Create systemd service
echo "Step 8: Creating systemd service..."
ssh -o StrictHostKeyChecking=no root@$SERVER_IP << EOF
cat > /etc/systemd/system/shadowmesh-relay.service << 'UNIT'
[Unit]
Description=ShadowMesh Relay Server
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/relay-server -port $PORT
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable shadowmesh-relay
systemctl start shadowmesh-relay
EOF

echo "✅ Service created and started"
echo ""

# Step 10: Verify service
echo "Step 9: Verifying service..."
sleep 3
ssh -o StrictHostKeyChecking=no root@$SERVER_IP "systemctl status shadowmesh-relay --no-pager" || true
echo ""

# Step 11: Test health endpoint
echo "Step 10: Testing health endpoint..."
sleep 2
if curl -s --connect-timeout 5 http://$SERVER_IP:$PORT/health | grep -q "OK"; then
    echo "✅ Health check passed"
else
    echo "⚠️  Health check failed (service may still be starting)"
fi
echo ""

# Step 12: Save connection info
echo "════════════════════════════════════════════════════════"
echo "Deployment Complete!"
echo "════════════════════════════════════════════════════════"
echo ""
echo "Relay Server Details:"
echo "  Public IP: $SERVER_IP"
echo "  WebSocket: ws://$SERVER_IP:$PORT/relay?peer_id=<your-peer-id>"
echo "  Status: http://$SERVER_IP:$PORT/status"
echo "  Health: http://$SERVER_IP:$PORT/health"
echo ""
echo "Connection Example:"
echo "  shadowmesh-001: ws://$SERVER_IP:$PORT/relay?peer_id=peer-001"
echo "  shadowmesh-002: ws://$SERVER_IP:$PORT/relay?peer_id=peer-002"
echo ""
echo "SSH Access:"
echo "  ssh root@$SERVER_IP"
echo ""
echo "Service Management:"
echo "  sudo systemctl status shadowmesh-relay"
echo "  sudo systemctl restart shadowmesh-relay"
echo "  sudo systemctl stop shadowmesh-relay"
echo "  sudo journalctl -u shadowmesh-relay -f"
echo ""
echo "Costs:"
echo "  ~\$5/month (1xCPU-1GB plan)"
echo ""
echo "To delete server:"
echo "  upctl server delete $SERVER_UUID"
echo ""
echo "════════════════════════════════════════════════════════"

# Save to file
cat > .relay-server-info << INFO
RELAY_SERVER_IP=$SERVER_IP
RELAY_SERVER_UUID=$SERVER_UUID
RELAY_SERVER_PORT=$PORT
RELAY_SERVER_URL=ws://$SERVER_IP:$PORT/relay
INFO

echo "Server info saved to: .relay-server-info"
