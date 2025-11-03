#!/bin/bash
# Deploy ShadowMesh from GitHub to a single machine
# Usage: ./deploy-from-github.sh <host> <user> <mode> [peer-ip]
#
# Examples:
#   ./deploy-from-github.sh 123.45.67.89 root listener
#   ./deploy-from-github.sh 192.168.1.100 pi connector 123.45.67.89

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Parse arguments
if [ $# -lt 3 ]; then
    echo "Usage: $0 <host> <user> <mode> [peer-ip]"
    echo ""
    echo "Arguments:"
    echo "  host      - IP address of the machine to deploy to"
    echo "  user      - SSH username"
    echo "  mode      - 'listener' or 'connector'"
    echo "  peer-ip   - Required if mode=connector (peer's IP address)"
    echo ""
    echo "Examples:"
    echo "  # Deploy to UK VPS as listener"
    echo "  $0 123.45.67.89 root listener"
    echo ""
    echo "  # Deploy to Belgium RPi as connector"
    echo "  $0 192.168.1.100 pi connector 123.45.67.89"
    echo ""
    exit 1
fi

HOST=$1
USER=$2
MODE=$3
PEER_IP=${4:-""}

# Validate mode
if [ "$MODE" != "listener" ] && [ "$MODE" != "connector" ]; then
    echo -e "${RED}Error: mode must be 'listener' or 'connector'${NC}"
    exit 1
fi

# Check peer-ip for connector mode
if [ "$MODE" == "connector" ] && [ -z "$PEER_IP" ]; then
    echo -e "${RED}Error: peer-ip required for connector mode${NC}"
    echo "Usage: $0 $HOST $USER connector <peer-ip>"
    exit 1
fi

# Determine TAP IP based on mode
if [ "$MODE" == "listener" ]; then
    TAP_IP="10.10.10.3"
    CONFIG_FILE="vps-uk-listener.yaml"
else
    TAP_IP="10.10.10.4"
    CONFIG_FILE="rpi-belgium-connector.yaml"
fi

echo -e "${BLUE}╔════════════════════════════════════════════════╗"
echo "║  ShadowMesh Deployment from GitHub             ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Configuration:${NC}"
echo "  Host:      $USER@$HOST"
echo "  Mode:      $MODE"
echo "  TAP IP:    $TAP_IP"
if [ "$MODE" == "connector" ]; then
    echo "  Peer IP:   $PEER_IP"
fi
echo "  GitHub:    https://github.com/CG-8663/shadowmesh"
echo ""

# Test SSH connectivity
echo -e "${BLUE}Testing SSH connection...${NC}"
if ! ssh -o ConnectTimeout=5 $USER@$HOST "echo 'SSH OK'" > /dev/null 2>&1; then
    echo -e "${RED}✗ Cannot connect to $HOST${NC}"
    echo "Please check:"
    echo "  - SSH credentials"
    echo "  - Network connectivity"
    echo "  - Host IP address"
    exit 1
fi
echo -e "${GREEN}✓ SSH connection OK${NC}"

# Deploy
echo ""
echo -e "${BLUE}Deploying to $HOST...${NC}"
echo ""

ssh $USER@$HOST "bash -s" <<'ENDSSH'
set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}Step 1: Installing dependencies...${NC}"

# Check and install git
if ! which git > /dev/null 2>&1; then
    echo "Installing git..."
    sudo apt-get update -qq
    sudo apt-get install -y git
fi
echo -e "${GREEN}✓ Git installed${NC}"

# Check and install Go
if ! which go > /dev/null 2>&1; then
    echo "Installing Go 1.21.5..."
    wget -q https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
    rm go1.21.5.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin
fi
echo -e "${GREEN}✓ Go installed${NC}"

echo ""
echo -e "${BLUE}Step 2: Cloning/updating repository...${NC}"

# Clone or update repository
if [ -d ~/shadowmesh ]; then
    cd ~/shadowmesh
    git fetch origin
    git reset --hard origin/main
    git pull origin main
    echo -e "${GREEN}✓ Repository updated${NC}"
else
    git clone https://github.com/CG-8663/shadowmesh.git ~/shadowmesh
    cd ~/shadowmesh
    echo -e "${GREEN}✓ Repository cloned${NC}"
fi

echo ""
echo -e "${BLUE}Step 3: Building shadowmesh-daemon...${NC}"

cd ~/shadowmesh
export PATH=$PATH:/usr/local/go/bin
make build

if [ ! -f build/shadowmesh-daemon ]; then
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Build successful${NC}"

echo ""
echo -e "${BLUE}Step 4: Installing...${NC}"

# Create directories
sudo mkdir -p /opt/shadowmesh /etc/shadowmesh /var/log/shadowmesh /var/lib/shadowmesh/keys
sudo chown -R $USER:$USER /opt/shadowmesh /var/log/shadowmesh /var/lib/shadowmesh

# Install binary
sudo cp build/shadowmesh-daemon /opt/shadowmesh/
sudo chmod +x /opt/shadowmesh/shadowmesh-daemon
echo -e "${GREEN}✓ Binary installed${NC}"

# Note: Config will be installed by parent script
echo -e "${GREEN}✓ Directories created${NC}"

echo ""
echo -e "${GREEN}Deployment complete!${NC}"
ENDSSH

# Install configuration
echo ""
echo -e "${BLUE}Step 5: Installing configuration...${NC}"

if [ "$MODE" == "connector" ]; then
    # Create config with actual peer IP
    ssh $USER@$HOST "cd ~/shadowmesh && sed 's/<UK_VPS_IP>/$PEER_IP/g' configs/$CONFIG_FILE > /tmp/config.yaml && sudo cp /tmp/config.yaml /etc/shadowmesh/config.yaml && rm /tmp/config.yaml"
else
    # Copy config as-is for listener
    ssh $USER@$HOST "sudo cp ~/shadowmesh/configs/$CONFIG_FILE /etc/shadowmesh/config.yaml"
fi

ssh $USER@$HOST "sudo chown root:root /etc/shadowmesh/config.yaml && sudo chmod 600 /etc/shadowmesh/config.yaml"
echo -e "${GREEN}✓ Configuration installed${NC}"

# Verify configuration
echo ""
echo -e "${BLUE}Step 6: Verifying configuration...${NC}"
ssh $USER@$HOST "sudo /opt/shadowmesh/shadowmesh-daemon --show-config --config /etc/shadowmesh/config.yaml" || {
    echo -e "${RED}✗ Configuration invalid${NC}"
    exit 1
}

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════╗"
echo "║  Deployment Complete!                          ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}To start the daemon:${NC}"
echo "  ssh $USER@$HOST"
echo "  sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml"
echo ""

if [ "$MODE" == "listener" ]; then
    echo -e "${YELLOW}This machine is configured as LISTENER${NC}"
    echo "  - Will accept connections on port 8443"
    echo "  - TAP IP: $TAP_IP"
    echo ""
else
    echo -e "${YELLOW}This machine is configured as CONNECTOR${NC}"
    echo "  - Will connect to: $PEER_IP:8443"
    echo "  - TAP IP: $TAP_IP"
    echo ""
fi

echo -e "${BLUE}Next steps:${NC}"
if [ "$MODE" == "listener" ]; then
    echo "  1. Start this daemon (listener)"
    echo "  2. Deploy connector to peer: $0 <peer-host> <peer-user> connector $HOST"
    echo "  3. Start connector daemon"
    echo "  4. Test: ping $TAP_IP"
else
    echo "  1. Ensure listener is running on $PEER_IP"
    echo "  2. Start this daemon (connector)"
    echo "  3. Wait for 'Tunnel established' message"
    echo "  4. Test: ping 10.10.10.3"
fi
echo ""
