#!/bin/bash
#
# ShadowMesh Relay Server Installation Script
# For Ubuntu 20.04+ / Debian 11+
#
# Usage: curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/scripts/install-relay.sh | sudo bash
#

set -e

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║       ShadowMesh Relay Server Installer                  ║"
echo "║       Post-Quantum VPN Network                            ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "ERROR: Please run as root (sudo)"
  exit 1
fi

# Detect OS
OS="$(uname -s)"
ARCH="$(uname -m)"

echo "Detected: $OS $ARCH"

if [ "$OS" != "Linux" ]; then
  echo "ERROR: This script only supports Linux"
  exit 1
fi

# Check for required commands
for cmd in curl git; do
  if ! command -v $cmd &> /dev/null; then
    echo "Installing $cmd..."
    apt-get update -qq
    apt-get install -y $cmd
  fi
done

# Install Go if not present
if ! command -v go &> /dev/null; then
  echo "Go not found. Installing Go 1.21.5..."
  cd /tmp
  wget -q https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
  export PATH=$PATH:/usr/local/go/bin
  rm go1.21.5.linux-amd64.tar.gz

  # Add Go to system PATH
  if ! grep -q "/usr/local/go/bin" /etc/profile; then
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
  fi

  echo "✅ Go $(go version | awk '{print $3}') installed"
else
  echo "✅ Go already installed: $(go version | awk '{print $3}')"
fi

# Clone/update repository
INSTALL_DIR="/opt/shadowmesh"
echo ""
echo "Installing to $INSTALL_DIR..."

if [ -d "$INSTALL_DIR" ]; then
  echo "Updating existing installation..."
  cd "$INSTALL_DIR"
  git fetch origin
  git reset --hard origin/main
else
  echo "Cloning repository..."
  git clone https://github.com/CG-8663/shadowmesh.git "$INSTALL_DIR"
  cd "$INSTALL_DIR"
fi

# Build relay server
echo ""
echo "Building relay server..."
export PATH=$PATH:/usr/local/go/bin
make build-relay

# Install binary
echo "Installing binary to /usr/local/bin..."
cp build/shadowmesh-relay /usr/local/bin/
chmod +x /usr/local/bin/shadowmesh-relay

# Create system user
if ! id shadowmesh &> /dev/null; then
  echo "Creating shadowmesh system user..."
  useradd --system --no-create-home --shell /bin/false shadowmesh
fi

# Create directories
echo "Creating configuration directories..."
mkdir -p /etc/shadowmesh
mkdir -p /var/lib/shadowmesh/keys
chown -R shadowmesh:shadowmesh /var/lib/shadowmesh
chmod 700 /var/lib/shadowmesh/keys

# Generate relay identity
echo ""
echo "Generating relay identity..."
sudo -u shadowmesh shadowmesh-relay --gen-keys --config /etc/shadowmesh/config.yaml || true

# Create default configuration
if [ ! -f /etc/shadowmesh/config.yaml ]; then
  echo "Creating default configuration..."
  cat > /etc/shadowmesh/config.yaml <<EOF
server:
  listen_addr: "0.0.0.0:8443"
  tls:
    enabled: true
    cert_file: "/etc/shadowmesh/relay-cert.pem"
    key_file: "/etc/shadowmesh/relay-key.pem"

limits:
  max_clients: 1000
  handshake_timeout: 30
  heartbeat_interval: 30
  heartbeat_timeout: 90
  max_frame_size: 65536
  read_buffer_size: 4096
  write_buffer_size: 4096

identity:
  keys_dir: "/var/lib/shadowmesh/keys"
  signing_key: "/var/lib/shadowmesh/keys/signing_key.json"
  relay_id: ""
  auto_generate: true

logging:
  level: "info"
  format: "text"
  output_file: ""
EOF

  chown root:shadowmesh /etc/shadowmesh/config.yaml
  chmod 640 /etc/shadowmesh/config.yaml
  echo "✅ Configuration created: /etc/shadowmesh/config.yaml"
fi

# Generate self-signed certificates if TLS files don't exist
if [ ! -f /etc/shadowmesh/relay-cert.pem ] || [ ! -f /etc/shadowmesh/relay-key.pem ]; then
  echo ""
  echo "Generating self-signed TLS certificates..."

  # Get server IP address
  SERVER_IP=$(curl -s ifconfig.me || curl -s icanhazip.com || echo "0.0.0.0")
  echo "Detected server IP: $SERVER_IP"

  # Generate private key
  openssl genrsa -out /etc/shadowmesh/relay-key.pem 4096 2>/dev/null

  # Create certificate with server IP in SAN
  cat > /tmp/relay-cert.cnf <<EOF
[req]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = v3_req

[dn]
C=US
ST=Cloud
L=Server
O=ShadowMesh
CN=$SERVER_IP

[v3_req]
subjectAltName = @alt_names
extendedKeyUsage = serverAuth

[alt_names]
IP.1 = $SERVER_IP
IP.2 = 0.0.0.0
DNS.1 = localhost
EOF

  # Generate self-signed certificate
  openssl req -new -x509 -days 365 -key /etc/shadowmesh/relay-key.pem \
    -out /etc/shadowmesh/relay-cert.pem \
    -config /tmp/relay-cert.cnf \
    -extensions v3_req 2>/dev/null

  rm /tmp/relay-cert.cnf

  chmod 600 /etc/shadowmesh/relay-key.pem
  chmod 644 /etc/shadowmesh/relay-cert.pem
  chown root:shadowmesh /etc/shadowmesh/relay-key.pem /etc/shadowmesh/relay-cert.pem

  echo "✅ Self-signed certificate generated"
  echo "   Certificate: /etc/shadowmesh/relay-cert.pem"
  echo "   Private key: /etc/shadowmesh/relay-key.pem"
  echo ""
  echo "⚠️  WARNING: Self-signed certificate for testing only!"
  echo "   For production, use Let's Encrypt or a proper CA certificate."
fi

# Create systemd service
echo ""
echo "Creating systemd service..."
cat > /etc/systemd/system/shadowmesh-relay.service <<'SYSTEMD_EOF'
[Unit]
Description=ShadowMesh Post-Quantum VPN Relay Server
Documentation=https://github.com/CG-8663/shadowmesh
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=shadowmesh
Group=shadowmesh
ExecStart=/usr/local/bin/shadowmesh-relay --config /etc/shadowmesh/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=shadowmesh-relay

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/shadowmesh
CapabilityBoundingSet=

[Install]
WantedBy=multi-user.target
SYSTEMD_EOF

systemctl daemon-reload

# Configure firewall (if ufw is present)
if command -v ufw &> /dev/null; then
  echo ""
  echo "Configuring firewall..."
  ufw allow 8443/tcp comment "ShadowMesh Relay"
  echo "✅ Firewall rule added: allow 8443/tcp"
fi

# Display relay ID
if [ -f /var/lib/shadowmesh/keys/relay_id.txt ]; then
  RELAY_ID=$(cat /var/lib/shadowmesh/keys/relay_id.txt)
else
  # Generate it now
  sudo -u shadowmesh shadowmesh-relay --gen-keys --config /etc/shadowmesh/config.yaml 2>/dev/null || true
  RELAY_ID=$(cat /var/lib/shadowmesh/keys/relay_id.txt 2>/dev/null || echo "ERROR")
fi

echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║       Installation Complete!                              ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "📋 Relay Information:"
echo "   Relay ID: $RELAY_ID"
echo "   Server IP: $SERVER_IP"
echo "   Listen Port: 8443"
echo "   WebSocket URL: wss://$SERVER_IP:8443/ws"
echo ""
echo "📁 Configuration:"
echo "   Config: /etc/shadowmesh/config.yaml"
echo "   Keys: /var/lib/shadowmesh/keys/"
echo "   Binary: /usr/local/bin/shadowmesh-relay"
echo ""
echo "🎯 Next Steps:"
echo ""
echo "1. Review configuration:"
echo "   sudo nano /etc/shadowmesh/config.yaml"
echo ""
echo "2. Enable and start service:"
echo "   sudo systemctl enable shadowmesh-relay"
echo "   sudo systemctl start shadowmesh-relay"
echo ""
echo "3. Check status:"
echo "   sudo systemctl status shadowmesh-relay"
echo "   sudo journalctl -u shadowmesh-relay -f"
echo ""
echo "4. Test health endpoint:"
echo "   curl -k https://$SERVER_IP:8443/health"
echo ""
echo "5. Configure clients to connect:"
echo "   relay_url: wss://$SERVER_IP:8443/ws"
echo ""
echo "📖 Documentation:"
echo "   https://github.com/CG-8663/shadowmesh/blob/main/STAGE_TESTING.md"
echo ""
