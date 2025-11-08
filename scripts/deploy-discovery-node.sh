#!/bin/bash
set -e

# ShadowMesh Discovery Node Deployment Script
# Deploys discovery node software to regional backbone servers

echo "═══════════════════════════════════════════════════════════"
echo "  ShadowMesh Discovery Node Deployment"
echo "═══════════════════════════════════════════════════════════"
echo

# Check if server IP provided
if [ -z "$1" ]; then
    echo "Usage: $0 <server-ip> <region>"
    echo ""
    echo "Example:"
    echo "  $0 209.151.148.121 north_america"
    echo "  $0 213.163.206.44 asia_pacific"
    echo "  $0 95.111.223.37 australia"
    echo "  $0 83.136.252.52 europe"
    echo ""
    exit 1
fi

SERVER_IP=$1
REGION=${2:-north_america}
SSH_KEY=~/.ssh/shadowmesh_relay_ed25519
SSH_USER=root
BINARY_PATH=build/shadowmesh-discovery

echo "Deployment Details:"
echo "  Server IP: $SERVER_IP"
echo "  Region: $REGION"
echo "  SSH Key: $SSH_KEY"
echo ""

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Binary not found at $BINARY_PATH"
    echo "Please build first: GOOS=linux GOARCH=amd64 go build -o $BINARY_PATH ./cmd/discovery/"
    exit 1
fi

echo "[1/6] Uploading binary..."
scp -i "$SSH_KEY" "$BINARY_PATH" "$SSH_USER@$SERVER_IP:/usr/local/bin/shadowmesh-discovery"

echo "[2/6] Uploading systemd service file..."
scp -i "$SSH_KEY" systemd/shadowmesh-discovery.service "$SSH_USER@$SERVER_IP:/tmp/"

echo "[3/6] Uploading configuration template..."
scp -i "$SSH_KEY" config/discovery.example.yaml "$SSH_USER@$SERVER_IP:/tmp/"

echo "[4/6] Setting up directories and permissions..."
ssh -i "$SSH_KEY" "$SSH_USER@$SERVER_IP" << 'EOF'
# Create directories
mkdir -p /etc/shadowmesh
mkdir -p /var/lib/shadowmesh/keys
mkdir -p /var/log/shadowmesh

# Set permissions
chown -R shadowmesh:shadowmesh /var/lib/shadowmesh /var/log/shadowmesh
chmod +x /usr/local/bin/shadowmesh-discovery

# Install systemd service
mv /tmp/shadowmesh-discovery.service /etc/systemd/system/
systemctl daemon-reload
EOF

echo "[5/6] Generating configuration file..."
ssh -i "$SSH_KEY" "$SSH_USER@$SERVER_IP" "bash -s" -- "$REGION" << 'EOF'
REGION=$1

# Use template to create config
cp /tmp/discovery.example.yaml /etc/shadowmesh/discovery.yaml

# Update region in config
sed -i "s/region: \"north_america\"/region: \"$REGION\"/" /etc/shadowmesh/discovery.yaml

# Generate random peer ID (first 40 hex chars of SHA256 of hostname + timestamp)
PEER_ID=$(echo "$(hostname)-$(date +%s)" | sha256sum | cut -c1-40)
sed -i "s/peer_id: \"generate-random-peer-id\"/peer_id: \"$PEER_ID\"/" /etc/shadowmesh/discovery.yaml

# Generate random passwords
DB_PASSWORD=$(openssl rand -hex 16)
REDIS_PASSWORD=$(openssl rand -hex 16)

# Update PostgreSQL password
sudo -u postgres psql -c "ALTER USER shadowmesh WITH ENCRYPTED PASSWORD '$DB_PASSWORD';"

# Update Redis password
if grep -q "^requirepass" /etc/redis/redis.conf; then
    sed -i "s/^requirepass.*/requirepass $REDIS_PASSWORD/" /etc/redis/redis.conf
else
    echo "requirepass $REDIS_PASSWORD" >> /etc/redis/redis.conf
fi
systemctl restart redis-server

# Update config file with passwords
sed -i "s/password: \"changeme\"/password: \"$DB_PASSWORD\"/" /etc/shadowmesh/discovery.yaml
sed -i "s/password: \"\"/password: \"$REDIS_PASSWORD\"/" /etc/shadowmesh/discovery.yaml

echo "Generated peer ID: $PEER_ID"
echo "Config file: /etc/shadowmesh/discovery.yaml"
echo "Passwords updated in database and config"
EOF

echo "[6/6] Starting discovery node service..."
ssh -i "$SSH_KEY" "$SSH_USER@$SERVER_IP" << 'EOF'
# Enable and start service
systemctl enable shadowmesh-discovery
systemctl start shadowmesh-discovery

# Wait a few seconds for startup
sleep 3

# Check status
systemctl status shadowmesh-discovery --no-pager
EOF

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "  Deployment Complete!"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "Service management:"
echo "  ssh -i $SSH_KEY $SSH_USER@$SERVER_IP"
echo "  systemctl status shadowmesh-discovery"
echo "  systemctl restart shadowmesh-discovery"
echo "  journalctl -u shadowmesh-discovery -f"
echo ""
echo "Test endpoints:"
echo "  curl http://$SERVER_IP:8080/health"
echo "  curl http://$SERVER_IP:8080/stats"
echo ""
echo "Configuration file: /etc/shadowmesh/discovery.yaml"
echo "Logs: /var/log/shadowmesh/discovery.log"
echo ""
