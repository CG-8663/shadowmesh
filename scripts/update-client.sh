#!/bin/bash
set -e

echo "Updating ShadowMesh client to latest version..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "ERROR: This script must be run as root (use sudo)"
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)
echo "Detected architecture: $ARCH"

# Ensure Go is in PATH
export PATH=$PATH:/usr/local/go/bin
if ! command -v go &> /dev/null; then
    echo "ERROR: Go not found. Please install Go first."
    exit 1
fi

echo "Using Go: $(go version)"

# Stop service
echo "Stopping shadowmesh-client service..."
systemctl stop shadowmesh-client

# Navigate to repo
cd /opt/shadowmesh

# Pull latest changes
echo "Pulling latest code..."
git fetch origin
git reset --hard origin/main

# Build client
echo "Building updated client..."
export PATH=$PATH:/usr/local/go/bin
make build-client

# Install binary
echo "Installing updated binary..."
cp bin/shadowmesh-client /usr/local/bin/shadowmesh-client
chmod +x /usr/local/bin/shadowmesh-client

# Start service
echo "Starting shadowmesh-client service..."
systemctl start shadowmesh-client

echo ""
echo "✅ Client updated successfully!"
echo ""
echo "Check status: sudo systemctl status shadowmesh-client"
echo "View logs: sudo journalctl -u shadowmesh-client -f"
