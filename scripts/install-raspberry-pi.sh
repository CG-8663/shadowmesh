#!/bin/bash
# ShadowMesh Installation Script for Raspberry Pi
# Story 2-8: Direct P2P Integration Test
#
# This script installs ShadowMesh daemon on a Raspberry Pi

set -e  # Exit on error

echo "════════════════════════════════════════════════════════"
echo "ShadowMesh Daemon Installation for Raspberry Pi"
echo "════════════════════════════════════════════════════════"
echo ""

# Check if running on Raspberry Pi or Linux
if [[ ! -f /etc/os-release ]]; then
    echo "❌ Error: This script is for Linux systems only"
    exit 1
fi

# Check if running with sudo
if [[ $EUID -ne 0 ]]; then
   echo "❌ Error: This script must be run with sudo"
   echo "   Usage: sudo ./scripts/install-raspberry-pi.sh"
   exit 1
fi

# Get actual user (when run with sudo)
ACTUAL_USER=${SUDO_USER:-$USER}
ACTUAL_HOME=$(eval echo ~$ACTUAL_USER)

echo "Installing for user: $ACTUAL_USER"
echo "Home directory: $ACTUAL_HOME"
echo ""

# Step 1: Check for Go installation
echo "Step 1: Checking for Go installation..."

# Detect architecture
ARCH=$(uname -m)
echo "Detected architecture: $ARCH"

case "$ARCH" in
    aarch64|arm64)
        GO_ARCH="arm64"
        ;;
    armv7l|armv6l)
        GO_ARCH="armv6l"
        ;;
    x86_64|amd64)
        GO_ARCH="amd64"
        ;;
    *)
        echo "❌ Error: Unsupported architecture: $ARCH"
        echo "   Supported: aarch64/arm64, armv7l/armv6l, x86_64/amd64"
        exit 1
        ;;
esac

if ! command -v go &> /dev/null; then
    GO_VERSION="1.23.3"
    echo "⚠️  Go is not installed. Installing Go ${GO_VERSION} for $GO_ARCH..."

    # Use home directory instead of /tmp (which may be full)
    DOWNLOAD_DIR="$ACTUAL_HOME/.shadowmesh-install"
    mkdir -p "$DOWNLOAD_DIR"
    cd "$DOWNLOAD_DIR"

    # Download with progress and timeout
    echo "📥 Downloading Go (this may take a few minutes)..."
    GO_URL="https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    if ! wget --show-progress --timeout=60 "$GO_URL"; then
        echo "❌ Error: Failed to download Go from $GO_URL"
        echo "   Please check your internet connection and try again"
        exit 1
    fi

    echo "📦 Extracting Go..."
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"

    echo "🧹 Cleaning up..."
    cd /
    rm -rf "$DOWNLOAD_DIR"

    # Add to PATH for current user
    if ! grep -q "/usr/local/go/bin" "$ACTUAL_HOME/.bashrc"; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> "$ACTUAL_HOME/.bashrc"
    fi

    export PATH=$PATH:/usr/local/go/bin

    # Verify installation
    echo "🔍 Verifying Go installation..."
    if [[ -f /usr/local/go/bin/go ]]; then
        GO_VERSION=$(/usr/local/go/bin/go version 2>&1)
        if [[ $? -eq 0 ]]; then
            echo "✅ Go installed successfully: $GO_VERSION"
        else
            echo "❌ Error: Go binary exists but failed to run"
            echo "   Error: $GO_VERSION"
            exit 1
        fi
    else
        echo "❌ Error: Go binary not found at /usr/local/go/bin/go"
        echo "   Checking extraction..."
        ls -la /usr/local/go/ 2>&1 || echo "   /usr/local/go/ does not exist"
        exit 1
    fi
else
    GO_VERSION=$(go version)
    echo "✅ Go is already installed: $GO_VERSION"
fi

# Step 2: Install Git if needed
echo ""
echo "Step 2: Checking for Git installation..."
if ! command -v git &> /dev/null; then
    echo "⚠️  Git is not installed. Installing..."
    apt update -qq
    apt install -y git
    echo "✅ Git installed successfully"
else
    echo "✅ Git is already installed"
fi

# Step 3: Clone or update repository
echo ""
echo "Step 3: Setting up ShadowMesh repository..."
SHADOWMESH_DIR="$ACTUAL_HOME/shadowmesh"

if [[ -d "$SHADOWMESH_DIR" ]]; then
    echo "⚠️  Directory $SHADOWMESH_DIR already exists"
    read -p "Update existing installation? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cd "$SHADOWMESH_DIR"
        sudo -u $ACTUAL_USER git pull
        echo "✅ Repository updated"
    else
        echo "Skipping repository update"
    fi
else
    echo "📥 Cloning ShadowMesh repository..."
    sudo -u $ACTUAL_USER git clone https://github.com/yourusername/shadowmesh.git "$SHADOWMESH_DIR"
    echo "✅ Repository cloned"
fi

# Step 4: Build daemon
echo ""
echo "Step 4: Building ShadowMesh daemon..."
cd "$SHADOWMESH_DIR"

# Ensure Go is available
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$ACTUAL_HOME/go

# Download dependencies first
echo "📦 Downloading dependencies..."
sudo -u $ACTUAL_USER env PATH=$PATH GOPATH=$GOPATH /usr/local/go/bin/go mod download

# Build as actual user
echo "🔨 Compiling daemon..."
sudo -u $ACTUAL_USER env PATH=$PATH GOPATH=$GOPATH /usr/local/go/bin/go build -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon/

if [[ ! -f bin/shadowmesh-daemon ]]; then
    echo "❌ Error: Build failed"
    exit 1
fi

echo "✅ Daemon built successfully"

# Step 5: Install binary system-wide
echo ""
echo "Step 5: Installing daemon binary..."
cp bin/shadowmesh-daemon /usr/local/bin/
chmod +x /usr/local/bin/shadowmesh-daemon
echo "✅ Binary installed to /usr/local/bin/shadowmesh-daemon"

# Step 6: Create config directory
echo ""
echo "Step 6: Creating configuration directory..."
mkdir -p /etc/shadowmesh
echo "✅ Created /etc/shadowmesh/"

# Step 7: Generate encryption key
echo ""
echo "Step 7: Generating encryption key..."
ENCRYPTION_KEY=$(openssl rand -hex 32)
echo ""
echo "════════════════════════════════════════════════════════"
echo "IMPORTANT: Save this encryption key!"
echo "════════════════════════════════════════════════════════"
echo ""
echo "Encryption Key:"
echo "$ENCRYPTION_KEY"
echo ""
echo "⚠️  You MUST use the SAME key on BOTH Raspberry Pis!"
echo "⚠️  Copy this key before continuing!"
echo ""
read -p "Press Enter after you've saved the key..."

# Step 8: Create configuration file
echo ""
echo "Step 8: Creating configuration file..."

read -p "Enter local IP for this Pi (e.g., 10.0.0.1/24): " LOCAL_IP

cat > /etc/shadowmesh/daemon.yaml <<EOF
# ShadowMesh Daemon Configuration
# Generated: $(date)

daemon:
  listen_address: "127.0.0.1:9090"
  log_level: "info"

network:
  tap_device: "tap0"
  local_ip: "$LOCAL_IP"

encryption:
  key: "$ENCRYPTION_KEY"

peer:
  address: ""

nat:
  enabled: true
  stun_server: "stun.l.google.com:19302"
EOF

echo "✅ Configuration saved to /etc/shadowmesh/daemon.yaml"

# Step 9: Final instructions
echo ""
echo "════════════════════════════════════════════════════════"
echo "Installation Complete!"
echo "════════════════════════════════════════════════════════"
echo ""
echo "Next steps:"
echo ""
echo "1. Install ShadowMesh on the second Raspberry Pi using this script"
echo "2. Use the SAME encryption key on both Pis"
echo "3. Use different local IPs (e.g., 10.0.0.1 and 10.0.0.2)"
echo ""
echo "To start the daemon:"
echo "  sudo shadowmesh-daemon /etc/shadowmesh/daemon.yaml"
echo ""
echo "To connect to peer (on initiator):"
echo "  curl -X POST http://127.0.0.1:9090/connect \\"
echo "    -H \"Content-Type: application/json\" \\"
echo "    -d '{\"peer_address\": \"PEER_IP:9001\"}'"
echo ""
echo "To check status:"
echo "  curl http://127.0.0.1:9090/status"
echo ""
echo "For detailed testing instructions, see:"
echo "  $SHADOWMESH_DIR/docs/QUICK_START_TESTING.md"
echo ""
echo "════════════════════════════════════════════════════════"
