#!/bin/bash
# Production Deployment: UK VPS + Belgium RPi P2P Testing
# This script pulls latest from GitHub and deploys to both machines

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════╗"
echo "║  ShadowMesh Production Deployment              ║"
echo "║  Epic 2: Direct P2P Testing                    ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""

# TODO: Set these to your actual values
UK_VPS_IP="YOUR_UK_VPS_IP"
UK_VPS_USER="YOUR_USERNAME"
UK_VPS_PORT="22"

RPI_IP="YOUR_RPI_IP"
RPI_USER="pi"
RPI_PORT="22"

# Check configuration
if [ "$UK_VPS_IP" == "YOUR_UK_VPS_IP" ]; then
    echo -e "${RED}Error: Please set UK_VPS_IP, UK_VPS_USER, and RPI_IP in this script${NC}"
    echo "Edit: $0"
    echo ""
    echo "Example:"
    echo "  UK_VPS_IP=\"123.45.67.89\""
    echo "  UK_VPS_USER=\"root\""
    echo "  RPI_IP=\"192.168.1.100\""
    exit 1
fi

echo -e "${GREEN}Deployment Configuration:${NC}"
echo "  GitHub Repo: https://github.com/CG-8663/shadowmesh"
echo "  UK VPS:      $UK_VPS_USER@$UK_VPS_IP (Listener, 10.10.10.3)"
echo "  Belgium RPi: $RPI_USER@$RPI_IP (Connector, 10.10.10.4)"
echo ""

# Function to deploy to a machine
deploy_machine() {
    local HOST=$1
    local USER=$2
    local PORT=$3
    local CONFIG_FILE=$4
    local MACHINE_NAME=$5
    local TAP_IP=$6

    echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
    echo -e "${BLUE}Deploying to $MACHINE_NAME${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}"

    # Check SSH connectivity
    echo "Testing SSH connection..."
    if ! ssh -p $PORT -o ConnectTimeout=5 $USER@$HOST "echo 'SSH OK'" > /dev/null 2>&1; then
        echo -e "${RED}✗ Cannot connect to $HOST${NC}"
        echo "Please check:"
        echo "  - SSH credentials"
        echo "  - Network connectivity"
        echo "  - Firewall settings"
        return 1
    fi
    echo -e "${GREEN}✓ SSH connection OK${NC}"

    # Check if git is installed
    echo "Checking git installation..."
    if ! ssh -p $PORT $USER@$HOST "which git" > /dev/null 2>&1; then
        echo -e "${YELLOW}Git not found, installing...${NC}"
        ssh -p $PORT $USER@$HOST "sudo apt-get update && sudo apt-get install -y git"
    fi
    echo -e "${GREEN}✓ Git available${NC}"

    # Check if golang is installed
    echo "Checking Go installation..."
    if ! ssh -p $PORT $USER@$HOST "which go" > /dev/null 2>&1; then
        echo -e "${YELLOW}Go not found, installing...${NC}"
        ssh -p $PORT $USER@$HOST "wget -q https://go.dev/dl/go1.21.5.linux-amd64.tar.gz && \
                                    sudo rm -rf /usr/local/go && \
                                    sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz && \
                                    rm go1.21.5.linux-amd64.tar.gz"
        ssh -p $PORT $USER@$HOST "echo 'export PATH=\$PATH:/usr/local/go/bin' >> ~/.bashrc"
    fi
    echo -e "${GREEN}✓ Go available${NC}"

    # Clone or update repository
    echo "Setting up repository..."
    ssh -p $PORT $USER@$HOST "
        if [ -d ~/shadowmesh ]; then
            cd ~/shadowmesh
            git fetch origin
            git reset --hard origin/main
            git pull origin main
        else
            git clone https://github.com/CG-8663/shadowmesh.git ~/shadowmesh
        fi
    "
    echo -e "${GREEN}✓ Repository updated${NC}"

    # Build daemon
    echo "Building shadowmesh-daemon..."
    ssh -p $PORT $USER@$HOST "
        cd ~/shadowmesh
        export PATH=\$PATH:/usr/local/go/bin
        make build
    " || {
        echo -e "${RED}✗ Build failed${NC}"
        return 1
    }
    echo -e "${GREEN}✓ Build successful${NC}"

    # Create directories
    echo "Creating directories..."
    ssh -p $PORT $USER@$HOST "
        sudo mkdir -p /opt/shadowmesh /etc/shadowmesh /var/log/shadowmesh /var/lib/shadowmesh/keys
        sudo chown -R $USER:$USER /opt/shadowmesh /var/log/shadowmesh /var/lib/shadowmesh
    "
    echo -e "${GREEN}✓ Directories created${NC}"

    # Install binary
    echo "Installing daemon..."
    ssh -p $PORT $USER@$HOST "
        sudo cp ~/shadowmesh/build/shadowmesh-daemon /opt/shadowmesh/
        sudo chmod +x /opt/shadowmesh/shadowmesh-daemon
    "
    echo -e "${GREEN}✓ Daemon installed${NC}"

    # Install configuration
    echo "Installing configuration..."
    if [ "$MACHINE_NAME" == "Belgium RPi" ]; then
        # Update peer address with actual UK VPS IP
        ssh -p $PORT $USER@$HOST "
            sed 's/<UK_VPS_IP>/$UK_VPS_IP/g' ~/shadowmesh/configs/$CONFIG_FILE > /tmp/config.yaml
            sudo cp /tmp/config.yaml /etc/shadowmesh/config.yaml
            sudo chown root:root /etc/shadowmesh/config.yaml
            sudo chmod 600 /etc/shadowmesh/config.yaml
            rm /tmp/config.yaml
        "
    else
        ssh -p $PORT $USER@$HOST "
            sudo cp ~/shadowmesh/configs/$CONFIG_FILE /etc/shadowmesh/config.yaml
            sudo chown root:root /etc/shadowmesh/config.yaml
            sudo chmod 600 /etc/shadowmesh/config.yaml
        "
    fi
    echo -e "${GREEN}✓ Configuration installed${NC}"

    # Verify config
    echo "Verifying configuration..."
    ssh -p $PORT $USER@$HOST "sudo /opt/shadowmesh/shadowmesh-daemon --show-config --config /etc/shadowmesh/config.yaml" || {
        echo -e "${RED}✗ Configuration invalid${NC}"
        return 1
    }

    echo -e "${GREEN}✓ $MACHINE_NAME deployment complete!${NC}"
    echo ""
}

# Deploy to UK VPS
deploy_machine "$UK_VPS_IP" "$UK_VPS_USER" "$UK_VPS_PORT" "vps-uk-listener.yaml" "UK VPS" "10.10.10.3"

# Deploy to Belgium RPi
deploy_machine "$RPI_IP" "$RPI_USER" "$RPI_PORT" "rpi-belgium-connector.yaml" "Belgium RPi" "10.10.10.4"

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════╗"
echo "║  Deployment Complete!                          ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo ""
echo "1. ${BLUE}Start UK VPS (Listener):${NC}"
echo "   ssh $UK_VPS_USER@$UK_VPS_IP"
echo "   sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml"
echo ""
echo "2. ${BLUE}Start Belgium RPi (Connector):${NC}"
echo "   ssh $RPI_USER@$RPI_IP"
echo "   sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml"
echo ""
echo "3. ${BLUE}Test the encrypted tunnel:${NC}"
echo "   From UK VPS:  ping 10.10.10.4"
echo "   From RPi:     ping 10.10.10.3"
echo ""
echo "4. ${BLUE}Monitor logs:${NC}"
echo "   tail -f /var/log/shadowmesh/daemon.log"
echo ""
echo -e "${GREEN}Expected behavior:${NC}"
echo "  ✓ Connection established"
echo "  ✓ PQC handshake complete (10-30 seconds)"
echo "  ✓ Tunnel established message"
echo "  ✓ Bidirectional ping works"
echo "  ✓ Stats show 0 errors"
echo ""
