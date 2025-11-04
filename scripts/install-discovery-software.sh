#!/bin/bash
set -e

# ShadowMesh Discovery Node Software Installation
# Run this on each discovery node after initial deployment

echo "=== ShadowMesh Discovery Node Software Installation ==="
echo

# Update system
echo "[1/8] Updating system packages..."
apt-get update
apt-get upgrade -y

# Install dependencies
echo "[2/8] Installing dependencies..."
apt-get install -y \
    curl \
    wget \
    git \
    build-essential \
    ufw \
    nginx \
    redis-server \
    postgresql \
    postgresql-contrib

# Install Go 1.21+
echo "[3/8] Installing Go 1.21.5..."
cd /tmp
wget -q https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
export PATH=$PATH:/usr/local/go/bin

# Create shadowmesh user
echo "[4/8] Creating shadowmesh user..."
useradd -r -s /bin/bash -m -d /var/lib/shadowmesh shadowmesh || true

# Create directories
echo "[5/8] Creating directories..."
mkdir -p /etc/shadowmesh
mkdir -p /var/lib/shadowmesh/keys
mkdir -p /var/log/shadowmesh
chown -R shadowmesh:shadowmesh /var/lib/shadowmesh /var/log/shadowmesh

# Configure firewall
echo "[6/8] Configuring firewall..."
ufw --force enable
ufw allow 22/tcp   # SSH
ufw allow 8443/tcp # WebSocket Secure
ufw allow 8080/tcp # HTTP API

# Configure PostgreSQL
echo "[7/8] Configuring PostgreSQL..."
sudo -u postgres psql -c "CREATE DATABASE shadowmesh;" 2>/dev/null || true
sudo -u postgres psql -c "CREATE USER shadowmesh WITH ENCRYPTED PASSWORD 'changeme';" 2>/dev/null || true
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE shadowmesh TO shadowmesh;" 2>/dev/null || true

# Configure Redis
echo "[8/8] Configuring Redis..."
if ! grep -q "^requirepass" /etc/redis/redis.conf; then
    echo "requirepass changeme" >> /etc/redis/redis.conf
    systemctl restart redis-server
fi

echo
echo "=== Installation Complete ==="
echo
echo "Installed software:"
echo "  Go: $(/usr/local/go/bin/go version)"
echo "  PostgreSQL: $(sudo -u postgres psql --version)"
echo "  Redis: $(redis-cli --version)"
echo "  Nginx: $(nginx -v 2>&1)"
echo
echo "Next steps:"
echo "  1. Deploy discovery node binary"
echo "  2. Configure systemd service"
echo "  3. Start discovery node"
echo
