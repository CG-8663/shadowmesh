#!/bin/bash
set -e

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║   ShadowMesh Raspberry Pi Client Auto-Installer          ║"
echo "║   Post-Quantum DPN Client with Auto-Configuration        ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "ERROR: This script must be run as root (use sudo)"
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)
echo "Detected architecture: $ARCH"

# Install dependencies
echo ""
echo "Installing dependencies..."
apt-get update
apt-get install -y git build-essential curl wget iproute2

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo ""
    echo "Installing Go 1.21..."
    GO_VERSION="1.21.6"

    case "$ARCH" in
        armv6l|armv7l)
            GO_ARCH="armv6l"
            ;;
        aarch64|arm64)
            GO_ARCH="arm64"
            ;;
        x86_64)
            GO_ARCH="amd64"
            ;;
        *)
            echo "ERROR: Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    cd /tmp
    wget -q https://golang.org/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go${GO_VERSION}.linux-${GO_ARCH}.tar.gz
    rm go${GO_VERSION}.linux-${GO_ARCH}.tar.gz

    # Add Go to PATH
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile

    echo "Go installed: $(go version)"
else
    echo "Go already installed: $(go version)"
fi

# Ensure Go is in PATH for this script
export PATH=$PATH:/usr/local/go/bin

# Load TUN kernel module
echo ""
echo "Loading TUN/TAP kernel module..."
modprobe tun
echo "tun" >> /etc/modules-load.d/shadowmesh.conf

# Clone or update repository
REPO_DIR="/opt/shadowmesh"
if [ -d "$REPO_DIR" ]; then
    echo ""
    echo "Updating existing repository..."
    cd "$REPO_DIR"
    git fetch origin
    git reset --hard origin/main
else
    echo ""
    echo "Cloning ShadowMesh repository..."
    git clone https://github.com/CG-8663/shadowmesh.git "$REPO_DIR"
    cd "$REPO_DIR"
fi

# Build client
echo ""
echo "Building ShadowMesh client..."
make build-client

# Install binary
echo ""
echo "Installing client binary..."
cp build/shadowmesh-client /usr/local/bin/shadowmesh-client
chmod +x /usr/local/bin/shadowmesh-client

# Create directories
echo ""
echo "Creating configuration directories..."
mkdir -p /etc/shadowmesh
mkdir -p /var/lib/shadowmesh
chmod 700 /var/lib/shadowmesh

# Create configuration file with chr001 device and auto IP assignment
echo ""
echo "Creating configuration file..."
cat > /etc/shadowmesh/config.yaml << 'EOF'
relay:
  url: "wss://83.136.252.52:8443/ws"
  tls_skip_verify: true
  reconnect_interval: 5s
  max_reconnects: 10
  handshake_timeout: 30s

tap:
  name: "chr001"
  mtu: 1500
  ip_addr: "10.10.10.3"
  netmask: "255.255.255.0"

crypto:
  enable_key_rotation: true
  key_rotation_interval: 1h

identity:
  keys_dir: "/var/lib/shadowmesh"
  private_key_file: "signing_key.bin"
  client_id_file: "client_id.txt"
  auto_generate: true

logging:
  level: "info"
  format: "text"
  output_file: "/var/log/shadowmesh-client.log"
EOF

chmod 600 /etc/shadowmesh/config.yaml

# Create systemd service
echo ""
echo "Creating systemd service..."
cat > /etc/systemd/system/shadowmesh-client.service << 'EOF'
[Unit]
Description=ShadowMesh Post-Quantum DPN Client
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Environment="PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
WorkingDirectory=/var/lib/shadowmesh
ExecStartPre=/sbin/modprobe tun
ExecStart=/usr/local/bin/shadowmesh-client -config /etc/shadowmesh/config.yaml
ExecStartPost=/bin/sleep 2
ExecStartPost=/sbin/ip addr add 10.10.10.3/24 dev chr001
ExecStartPost=/sbin/ip link set chr001 up
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=false
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/var/lib/shadowmesh /var/log /dev/net/tun

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd
echo ""
echo "Reloading systemd..."
systemctl daemon-reload

# Enable and start service
echo ""
echo "Enabling and starting ShadowMesh client service..."
systemctl enable shadowmesh-client
systemctl restart shadowmesh-client

# Wait for service to start
echo ""
echo "Waiting for client to initialize..."
sleep 5

# Check status
echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║              Installation Complete!                       ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "Service Status:"
systemctl status shadowmesh-client --no-pager -l | tail -20
echo ""
echo "Network Configuration:"
ip addr show chr001 2>/dev/null || echo "chr001 device not yet created (wait a moment)"
echo ""
echo "Useful Commands:"
echo "  sudo systemctl status shadowmesh-client    # Check status"
echo "  sudo systemctl restart shadowmesh-client   # Restart client"
echo "  sudo systemctl stop shadowmesh-client      # Stop client"
echo "  sudo journalctl -u shadowmesh-client -f    # View live logs"
echo "  ping 10.10.10.2                            # Ping Proxmox client"
echo "  ip addr show chr001                        # Show TAP device"
echo ""
echo "The client is now running in the background!"
echo "You can SSH into this machine and run ping tests."
echo ""
