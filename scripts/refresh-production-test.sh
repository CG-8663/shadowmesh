#!/bin/bash
# Refresh ShadowMesh Production Test
# Clean install on shadowmesh-001 (UK Proxmox) and shadowmesh-002 (Belgium RPi)

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════╗"
echo "║  ShadowMesh Production Test - Refresh         ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""

# Infrastructure via Tailscale
SHADOWMESH_001_TS="100.115.193.115"  # UK Proxmox
SHADOWMESH_002_TS="100.90.48.10"     # Belgium RPi
SSH_USER="pxcghost"

# Local paths
BUILD_DIR="./build"
RELAY_BIN="shadowmesh-relay-linux-amd64"

# Remote paths
REMOTE_DIR="/opt/shadowmesh"
REMOTE_CONFIG="/etc/shadowmesh"

echo -e "${GREEN}Infrastructure:${NC}"
echo "  shadowmesh-001 (UK Proxmox):     $SHADOWMESH_001_TS"
echo "  shadowmesh-002 (Belgium RPi):    $SHADOWMESH_002_TS"
echo "  TAP Network:                     10.10.10.0/24"
echo ""

# Function to clean and deploy
clean_and_deploy() {
    local HOST=$1
    local MACHINE_NAME=$2
    local BINARY=$3

    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Refreshing $MACHINE_NAME ($HOST)${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    # Step 1: Stop any running processes
    echo -e "${YELLOW}1. Stopping any running ShadowMesh processes...${NC}"
    ssh $SSH_USER@$HOST "sudo pkill -f shadowmesh || true" || true
    sleep 2

    # Step 2: Clean up old installation
    echo -e "${YELLOW}2. Cleaning up old installation...${NC}"
    ssh $SSH_USER@$HOST "sudo rm -rf $REMOTE_DIR/* /var/lib/shadowmesh/keys/* /var/log/shadowmesh/* || true"

    # Step 3: Create directories
    echo -e "${YELLOW}3. Creating directories...${NC}"
    ssh $SSH_USER@$HOST "sudo mkdir -p $REMOTE_DIR $REMOTE_CONFIG /var/lib/shadowmesh/keys /var/log/shadowmesh"

    # Step 4: Upload binary
    echo -e "${YELLOW}4. Uploading binary ($BINARY)...${NC}"
    scp $BUILD_DIR/$BINARY $SSH_USER@$HOST:/tmp/shadowmesh-binary
    ssh $SSH_USER@$HOST "sudo mv /tmp/shadowmesh-binary $REMOTE_DIR/shadowmesh && sudo chmod +x $REMOTE_DIR/shadowmesh"

    echo -e "${GREEN}✅ $MACHINE_NAME refreshed${NC}"
    echo ""
}

# Check binaries exist
if [ ! -f "$BUILD_DIR/$RELAY_BIN" ]; then
    echo -e "${RED}Error: Relay binary not found${NC}"
    echo "Run: make build-relay"
    exit 1
fi

echo -e "${YELLOW}Starting fresh deployment...${NC}"
echo ""

# Deploy to both machines
clean_and_deploy $SHADOWMESH_001_TS "shadowmesh-001 (UK Proxmox)" "shadowmesh-relay-linux-amd64"
clean_and_deploy $SHADOWMESH_002_TS "shadowmesh-002 (Belgium RPi)" "shadowmesh-relay-linux-arm64"

echo -e "${GREEN}╔════════════════════════════════════════════════╗"
echo "║  Deployment Complete!                          ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo ""
echo "1. Configure and start relay server on shadowmesh-001:"
echo "   ssh $SSH_USER@$SHADOWMESH_001_TS"
echo "   sudo /opt/shadowmesh/shadowmesh"
echo ""
echo "2. Connect from shadowmesh-002:"
echo "   ssh $SSH_USER@$SHADOWMESH_002_TS"
echo "   sudo /opt/shadowmesh/shadowmesh"
echo ""
echo "3. Test connectivity:"
echo "   ping 10.10.10.3  # From shadowmesh-002"
echo "   ping 10.10.10.4  # From shadowmesh-001"
echo ""
