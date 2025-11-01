#!/bin/bash
# ShadowMesh Client Installer
# One-line install: curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/scripts/install-client.sh | sudo bash

set -e

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║       ShadowMesh Client Installer                        ║"
echo "║       Post-Quantum DPN Network                            ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ This script must be run as root (use sudo)"
    exit 1
fi

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

echo "Detected: $OS $ARCH"

# Check for required tools
if ! command -v git &> /dev/null; then
    echo "Installing git..."
    if [ -f /etc/debian_version ]; then
        apt-get update -qq && apt-get install -y -qq git curl
    elif [ -f /etc/redhat-release ]; then
        yum install -y git curl
    else
        echo "❌ Please install git manually"
        exit 1
    fi
fi

# Install Go if not present
if ! command -v go &> /dev/null; then
    echo "Go not found. Installing Go 1.21.5..."
    
    if [ "$ARCH" = "x86_64" ]; then
        GO_ARCH="amd64"
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
        GO_ARCH="arm64"
    else
        echo "❌ Unsupported architecture: $ARCH"
        exit 1
    fi
    
    GO_VERSION="1.21.5"
    GO_TAR="go${GO_VERSION}.${OS}-${GO_ARCH}.tar.gz"
    
    curl -sL "https://go.dev/dl/${GO_TAR}" -o "/tmp/${GO_TAR}"
    tar -C /usr/local -xzf "/tmp/${GO_TAR}"
    rm "/tmp/${GO_TAR}"
    
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    
    GO_INSTALLED_VERSION=$(go version | awk '{print $3}')
    echo "✅ Go ${GO_INSTALLED_VERSION} installed"
else
    export PATH=$PATH:/usr/local/go/bin
    echo "✅ Go already installed: $(go version | awk '{print $3}')"
fi

# Clone or update repository
INSTALL_DIR="/opt/shadowmesh"
echo ""
echo "Installing to ${INSTALL_DIR}..."

if [ -d "${INSTALL_DIR}/.git" ]; then
    echo "Updating existing installation..."
    cd "${INSTALL_DIR}"
    git pull origin main
else
    echo "Cloning repository..."
    rm -rf "${INSTALL_DIR}"
    git clone https://github.com/CG-8663/shadowmesh.git "${INSTALL_DIR}"
    cd "${INSTALL_DIR}"
fi

# Build client
echo ""
echo "Building client..."
make build-client

# Install binary
echo "Installing binary..."
cp bin/shadowmesh-client /usr/local/bin/
chmod +x /usr/local/bin/shadowmesh-client

# Create config directory
mkdir -p /etc/shadowmesh

# Create example config if it doesn't exist
if [ ! -f /etc/shadowmesh/config.yaml ]; then
    echo "Creating example configuration..."
    cat > /etc/shadowmesh/config.yaml << 'EOF'
# ShadowMesh Client Configuration
# Replace with your relay server URL
relay_url: "wss://YOUR_RELAY_IP:8443/ws"
tls_verify: false
log_level: "info"
interface: "tun0"
EOF
    echo "⚠️  Please edit /etc/shadowmesh/config.yaml with your relay server URL"
fi

# Load TUN module
if ! lsmod | grep -q "^tun"; then
    echo "Loading TUN kernel module..."
    modprobe tun
    echo "tun" >> /etc/modules-load.d/shadowmesh.conf
fi

echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║              Installation Complete! ✅                    ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "Client installed at: /usr/local/bin/shadowmesh-client"
echo "Configuration: /etc/shadowmesh/config.yaml"
echo ""
echo "Next steps:"
echo "1. Edit config: nano /etc/shadowmesh/config.yaml"
echo "2. Set your relay URL: relay_url: \"wss://YOUR_RELAY_IP:8443/ws\""
echo "3. Run client: shadowmesh-client -config /etc/shadowmesh/config.yaml"
echo ""
echo "For systemd service setup, see: ${INSTALL_DIR}/docs/CLIENT_SETUP.md"
echo ""
