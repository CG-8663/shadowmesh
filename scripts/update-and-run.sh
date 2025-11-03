#!/bin/bash
# Update from GitHub and run locally
# Run this script ON each machine (VPS or RPi)
#
# Usage on UK VPS:    ./update-and-run.sh listener
# Usage on Belgium RPi: ./update-and-run.sh connector YOUR_VPS_IP

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

MODE=$1
PEER_IP=${2:-""}

if [ -z "$MODE" ]; then
    echo "Usage: $0 <mode> [peer-ip]"
    echo ""
    echo "Examples:"
    echo "  # On UK VPS (listener)"
    echo "  $0 listener"
    echo ""
    echo "  # On Belgium RPi (connector)"
    echo "  $0 connector 123.45.67.89"
    echo ""
    exit 1
fi

if [ "$MODE" != "listener" ] && [ "$MODE" != "connector" ]; then
    echo -e "${RED}Error: mode must be 'listener' or 'connector'${NC}"
    exit 1
fi

if [ "$MODE" == "connector" ] && [ -z "$PEER_IP" ]; then
    echo -e "${RED}Error: peer-ip required for connector mode${NC}"
    echo "Usage: $0 connector YOUR_VPS_IP"
    exit 1
fi

echo -e "${BLUE}╔════════════════════════════════════════════════╗"
echo "║  ShadowMesh - Update & Run                     ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""

# Check if we're in a shadowmesh directory
if [ ! -f "go.mod" ] || ! grep -q "shadowmesh" go.mod; then
    echo -e "${RED}Error: Must run from shadowmesh directory${NC}"
    echo "Expected: cd /opt/shadowmesh && sudo ./scripts/update-and-run.sh $MODE"
    echo "Or:       cd ~/shadowmesh && ./scripts/update-and-run.sh $MODE"
    exit 1
fi

# Determine configuration
if [ "$MODE" == "listener" ]; then
    TAP_IP="10.10.10.3"
    CONFIG_FILE="configs/vps-uk-listener.yaml"
else
    TAP_IP="10.10.10.4"
    CONFIG_FILE="configs/rpi-belgium-connector.yaml"
fi

echo -e "${GREEN}Configuration:${NC}"
echo "  Mode:      $MODE"
echo "  TAP IP:    $TAP_IP"
if [ "$MODE" == "connector" ]; then
    echo "  Peer IP:   $PEER_IP"
fi
echo ""

# Step 1: Update from GitHub
echo -e "${BLUE}Step 1: Updating from GitHub...${NC}"
git fetch origin
git reset --hard origin/main
git pull origin main
echo -e "${GREEN}✓ Repository updated${NC}"
echo ""

# Step 2: Build
echo -e "${BLUE}Step 2: Building...${NC}"
# Ensure Go is in PATH
export PATH=$PATH:/usr/local/go/bin
make build
if [ ! -f "build/shadowmesh-daemon" ]; then
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Build successful${NC}"
echo ""

# Step 3: Create config
echo -e "${BLUE}Step 3: Preparing configuration...${NC}"
if [ "$MODE" == "connector" ]; then
    # Replace peer IP in config
    sed "s/<UK_VPS_IP>/$PEER_IP/g" $CONFIG_FILE > /tmp/shadowmesh-config.yaml
    CONFIG_PATH="/tmp/shadowmesh-config.yaml"
else
    CONFIG_PATH="$CONFIG_FILE"
fi
echo -e "${GREEN}✓ Configuration ready${NC}"
echo ""

# Step 4: Show what will run
echo -e "${BLUE}Step 4: Ready to start daemon${NC}"
echo ""
echo -e "${YELLOW}About to run:${NC}"
echo "  sudo ./build/shadowmesh-daemon --config $CONFIG_PATH"
echo ""

if [ "$MODE" == "listener" ]; then
    echo -e "${GREEN}This machine will:${NC}"
    echo "  - Listen on port 8443"
    echo "  - Accept incoming P2P connections"
    echo "  - Create TAP device chr-001 with IP $TAP_IP"
else
    echo -e "${GREEN}This machine will:${NC}"
    echo "  - Connect to $PEER_IP:8443"
    echo "  - Establish P2P encrypted tunnel"
    echo "  - Create TAP device chr-001 with IP $TAP_IP"
fi
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${YELLOW}Note: TAP device creation requires root${NC}"
    echo "Starting with sudo..."
    echo ""
fi

# Step 5: Run daemon
echo -e "${BLUE}Step 5: Starting daemon...${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
echo ""

sudo ./build/shadowmesh-daemon --config $CONFIG_PATH

# Cleanup
if [ -f "/tmp/shadowmesh-config.yaml" ]; then
    rm /tmp/shadowmesh-config.yaml
fi
