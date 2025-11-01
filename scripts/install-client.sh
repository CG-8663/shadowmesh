#!/bin/bash
set -e

echo "================================"
echo "ShadowMesh Client Installer"
echo "================================"
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

# Install Go if not present
if ! command -v go &> /dev/null; then
  echo "Go not found. Installing..."
  case "$OS" in
    Linux)
      wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
      tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
      export PATH=$PATH:/usr/local/go/bin
      rm go1.21.5.linux-amd64.tar.gz
      ;;
    Darwin)
      echo "Please install Go via Homebrew: brew install go"
      exit 1
      ;;
  esac
fi

# Clone repository
INSTALL_DIR="/opt/shadowmesh"
echo "Installing to $INSTALL_DIR..."

if [ -d "$INSTALL_DIR" ]; then
  echo "Updating existing installation..."
  cd "$INSTALL_DIR"
  git pull
else
  git clone https://github.com/CG-8663/shadowmesh.git "$INSTALL_DIR"
  cd "$INSTALL_DIR"
fi

# Build client
echo "Building client..."
go build -o /usr/local/bin/shadowmesh-client ./client/daemon

# Set permissions
chmod +x /usr/local/bin/shadowmesh-client

# Create config directory
mkdir -p /root/.shadowmesh/keys
chmod 700 /root/.shadowmesh/keys

# Install systemd service (Linux only)
if [ "$OS" == "Linux" ] && command -v systemctl &> /dev/null; then
  echo "Installing systemd service..."
  cat > /etc/systemd/system/shadowmesh.service << 'SYSTEMD_EOF'
[Unit]
Description=ShadowMesh Post-Quantum VPN Client
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/shadowmesh-client
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
SYSTEMD_EOF

  systemctl daemon-reload
  echo "Service installed. Enable with: systemctl enable shadowmesh"
fi

echo ""
echo "================================"
echo "Installation Complete!"
echo "================================"
echo ""
echo "Next steps:"
echo "1. Generate keys: shadowmesh-client --gen-keys"
echo "2. Edit config: nano ~/.shadowmesh/config.yaml"
echo "3. Run client: shadowmesh-client"
echo ""
echo "Or enable service: systemctl enable --now shadowmesh"
