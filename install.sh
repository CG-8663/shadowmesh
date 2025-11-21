#!/bin/bash
# ShadowMesh One-Line Installer
# Usage: curl -sSL https://raw.githubusercontent.com/cg-8663/shadowmesh/main/install.sh | bash
# Or:    curl -sSL https://get.shadowmesh.io | bash

set -e

VERSION="0.1.0-epic2"
REPO_URL="https://github.com/cg-8663/shadowmesh.git"
INSTALL_DIR="/opt/shadowmesh"
RELAY_IP="94.237.121.21"  # Default UpCloud relay
RELAY_PORT="9545"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║       ShadowMesh Installer v${VERSION}          ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }
print_info() { echo -e "${YELLOW}ℹ️  $1${NC}"; }

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    print_error "Please run with sudo"
    echo "Usage: curl -sSL https://raw.githubusercontent.com/cg-8663/shadowmesh/main/install.sh | sudo bash"
    exit 1
fi

print_header

# Auto-detect node or prompt
echo "Detecting node configuration..."
echo ""

HOSTNAME=$(hostname)
CURRENT_IP=$(hostname -I | awk '{print $1}')

print_info "Hostname: $HOSTNAME"
print_info "Primary IP: $CURRENT_IP"
echo ""

# Auto-detect based on hostname or IP
if [[ "$HOSTNAME" == *"shadowmesh-001"* ]] || [[ "$HOSTNAME" == *"uk"* ]] || [[ "$CURRENT_IP" == "100.86.59.47" ]]; then
    NODE_NAME="UK Client"
    NODE_IP="10.10.10.3/24"
    AUTO_DETECTED=true
elif [[ "$HOSTNAME" == *"shadowmesh-002"* ]] || [[ "$HOSTNAME" == *"belgium"* ]] || [[ "$HOSTNAME" == *"raspi"* ]] || [[ "$CURRENT_IP" == "100.90.48.10" ]]; then
    NODE_NAME="Belgium Client"
    NODE_IP="10.10.10.4/24"
    AUTO_DETECTED=true
elif [[ "$HOSTNAME" == *"chronara"* ]] || [[ "$HOSTNAME" == *"vm111"* ]] || [[ "$CURRENT_IP" == *"10.10.10.5"* ]]; then
    NODE_NAME="Chronara API Client"
    NODE_IP="10.10.10.5/24"
    AUTO_DETECTED=true
elif [[ "$HOSTNAME" == *"shadowmesh-004"* ]] || [[ "$HOSTNAME" == *"chronara-pi-client-ph"* ]] || [[ "$HOSTNAME" == *"philippines"* ]] || [[ "$HOSTNAME" == *"-ph"* ]] || [[ "$CURRENT_IP" == "100.87.142.44" ]]; then
    NODE_NAME="Philippines Client"
    NODE_IP="10.10.10.6/24"
    AUTO_DETECTED=true
else
    AUTO_DETECTED=false
fi

if [ "$AUTO_DETECTED" = true ]; then
    echo "Auto-detected: $NODE_NAME"
    echo "  Mesh IP: $NODE_IP"
    echo "  Relay: $RELAY_IP:$RELAY_PORT"
    echo ""
    read -p "Is this correct? [Y/n]: " confirm
    if [[ "$confirm" =~ ^[Nn]$ ]]; then
        AUTO_DETECTED=false
    fi
fi

if [ "$AUTO_DETECTED" = false ]; then
    echo "Manual configuration:"
    echo ""
    echo "Select node type:"
    echo "  1) UK Client (10.10.10.3)"
    echo "  2) Belgium Client (10.10.10.4)"
    echo "  3) Chronara API Client (10.10.10.5)"
    echo "  4) Philippines Client (10.10.10.6)"
    echo "  5) Custom"
    read -p "Choice [1-5]: " choice

    case $choice in
        1) NODE_NAME="UK Client"; NODE_IP="10.10.10.3/24" ;;
        2) NODE_NAME="Belgium Client"; NODE_IP="10.10.10.4/24" ;;
        3) NODE_NAME="Chronara API Client"; NODE_IP="10.10.10.5/24" ;;
        4) NODE_NAME="Philippines Client"; NODE_IP="10.10.10.6/24" ;;
        5)
            read -p "Node name: " NODE_NAME
            read -p "Mesh IP (x.x.x.x/24): " NODE_IP
            read -p "Relay IP [$RELAY_IP]: " custom_relay
            if [ -n "$custom_relay" ]; then
                RELAY_IP="$custom_relay"
            fi
            ;;
        *) print_error "Invalid choice"; exit 1 ;;
    esac
fi

echo ""
echo "════════════════════════════════════════════════════════"
echo "Installation Summary:"
echo "  Node: $NODE_NAME"
echo "  Mesh IP: $NODE_IP"
echo "  Relay: $RELAY_IP:$RELAY_PORT"
echo "════════════════════════════════════════════════════════"
echo ""

# Check for non-interactive mode (via environment variable or stdin redirection)
if [ -n "$SHADOWMESH_AUTO_INSTALL" ] || [ ! -t 0 ]; then
    echo "Auto-confirming installation (non-interactive mode)"
    proceed="y"
else
    read -p "Proceed with installation? [Y/n]: " proceed
fi

if [[ "$proceed" =~ ^[Nn]$ ]]; then
    echo "Installation cancelled"
    exit 0
fi
echo ""

# Step 0: Cleanup old installations
print_info "Step 0/8: Cleaning up old installations..."
systemctl stop shadowmesh 2>/dev/null || true
systemctl stop shadowmesh-client 2>/dev/null || true
systemctl stop shadowmesh-daemon 2>/dev/null || true
systemctl stop shadowmesh-relay 2>/dev/null || true
systemctl disable shadowmesh 2>/dev/null || true
systemctl disable shadowmesh-client 2>/dev/null || true
systemctl disable shadowmesh-daemon 2>/dev/null || true
systemctl disable shadowmesh-relay 2>/dev/null || true
pkill -9 shadowmesh 2>/dev/null || true
sleep 2
rm -f /etc/systemd/system/shadowmesh.service
rm -f /etc/systemd/system/shadowmesh-client.service
rm -f /etc/systemd/system/shadowmesh-daemon.service
rm -f /etc/systemd/system/shadowmesh-relay.service
ip link delete chr001 2>/dev/null || true
ip link delete chr-api 2>/dev/null || true
print_success "Old installations cleaned"
echo ""

# Step 1: Install dependencies
print_info "Step 1/8: Installing dependencies..."
if command -v apt-get &> /dev/null; then
    apt-get update -qq
    apt-get install -y -qq git curl wget openssl iptables > /dev/null 2>&1
elif command -v yum &> /dev/null; then
    yum install -y git curl wget openssl iptables > /dev/null 2>&1
fi
print_success "Dependencies installed"
echo ""

# Step 2: Install Go if needed
if ! command -v go &> /dev/null; then
    print_info "Step 2/8: Installing Go 1.23.3..."
    cd /tmp
    wget -q https://go.dev/dl/go1.23.3.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.23.3.linux-amd64.tar.gz
    rm go1.23.3.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
    print_success "Go installed: $(go version)"
else
    print_info "Step 2/8: Go already installed: $(go version)"
fi
echo ""

# Step 3: Clone repository
print_info "Step 3/8: Cloning ShadowMesh repository..."
if [ -d "$INSTALL_DIR" ]; then
    print_info "Repository exists, updating..."
    cd "$INSTALL_DIR"
    git config --global --add safe.directory "$INSTALL_DIR" 2>/dev/null || true
    git fetch origin > /dev/null 2>&1
    git reset --hard origin/main > /dev/null 2>&1
else
    git clone "$REPO_URL" "$INSTALL_DIR" > /dev/null 2>&1
    cd "$INSTALL_DIR"
fi
print_success "Repository ready"
echo ""

# Step 4: Build binary
print_info "Step 4/8: Building ShadowMesh daemon..."
export PATH=$PATH:/usr/local/go/bin
cd "$INSTALL_DIR"
go build -ldflags="-s -w" -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon/ 2>&1 | grep -v "go: downloading" || true
print_success "Binary built"
echo ""

# Step 5: Install binary
print_info "Step 5/8: Installing binary..."
cp bin/shadowmesh-daemon /usr/local/bin/
chmod +x /usr/local/bin/shadowmesh-daemon
print_success "Binary installed to /usr/local/bin/shadowmesh-daemon"
echo ""

# Step 6: Create configuration
print_info "Step 6/8: Creating configuration..."
mkdir -p /etc/shadowmesh

# Generate encryption key (or use shared key for mesh)
ENC_KEY="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

cat > /etc/shadowmesh/daemon.yaml << EOF
# ShadowMesh Configuration
# Node: $NODE_NAME
# Generated: $(date)

daemon:
  listen_address: "127.0.0.1:9090"
  log_level: "info"

network:
  tap_device: "chr001"
  local_ip: "$NODE_IP"

encryption:
  key: "$ENC_KEY"

peer:
  address: "$RELAY_IP:$RELAY_PORT"

nat:
  enabled: true
  stun_server: "stun.l.google.com:19302"

max_throughput:
  enabled: true
  socket:
    send_buffer_kb: 256
    recv_buffer_kb: 512
  batching:
    enabled: true
    max_batch_size: 10
    max_batch_bytes: 9000
    coalesce_timeout_ms: 2
  compression:
    enabled: true
    mode: "adaptive"
    compression_level: 1
EOF

print_success "Configuration created at /etc/shadowmesh/daemon.yaml"
echo ""

# Step 7: Create systemd service
print_info "Step 7/8: Creating systemd service..."

# Create new service
cat > /etc/systemd/system/shadowmesh-daemon.service << 'EOF'
[Unit]
Description=ShadowMesh Daemon
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/shadowmesh-daemon /etc/shadowmesh/daemon.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable shadowmesh-daemon
systemctl start shadowmesh-daemon
sleep 3

print_success "Service created and started"
echo ""

# Step 8: Verify installation
print_info "Step 8/8: Verifying installation..."
echo ""

if systemctl is-active --quiet shadowmesh-daemon; then
    print_success "Daemon is running"
else
    print_error "Daemon failed to start"
    echo ""
    journalctl -u shadowmesh-daemon -n 20 --no-pager
    exit 1
fi

# Check TAP interface
sleep 2
if ip addr show chr001 &> /dev/null; then
    print_success "TAP interface chr001 created"
    ip addr show chr001 | grep -E "inet |mtu"
else
    print_info "TAP interface not yet created (check logs)"
fi

echo ""
echo "════════════════════════════════════════════════════════"
echo "Installation Complete!"
echo "════════════════════════════════════════════════════════"
echo ""
echo "Node: $NODE_NAME"
echo "Mesh IP: $NODE_IP"
echo "Relay: $RELAY_IP:$RELAY_PORT"
echo ""
echo "Management Commands:"
echo "  Status:  sudo systemctl status shadowmesh-daemon"
echo "  Restart: sudo systemctl restart shadowmesh-daemon"
echo "  Logs:    sudo journalctl -u shadowmesh-daemon -f"
echo "  Network: ip addr show chr001"
echo ""
echo "Test connectivity:"
echo "  ping 10.10.10.3  # UK"
echo "  ping 10.10.10.4  # Belgium"
echo "  ping 10.10.10.5  # Chronara API"
echo ""
