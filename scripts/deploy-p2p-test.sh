#!/bin/bash
# Deploy ShadowMesh for Direct P2P Testing
# UK VPS (listener) <-> Belgium Raspberry Pi (connector)

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════╗"
echo "║  ShadowMesh Direct P2P Deployment             ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""

# Configuration
UK_VPS_IP="<REPLACE_WITH_UK_VPS_IP>"
UK_VPS_USER="<REPLACE_WITH_UK_VPS_USER>"
UK_VPS_PORT="22"

RPI_IP="<REPLACE_WITH_RPI_IP>"
RPI_USER="pi"
RPI_PORT="22"

# Paths
LOCAL_BUILD_DIR="./build"
LOCAL_CONFIG_DIR="./configs"

REMOTE_INSTALL_DIR="/opt/shadowmesh"
REMOTE_CONFIG_DIR="/etc/shadowmesh"
REMOTE_KEYS_DIR="/var/lib/shadowmesh/keys"
REMOTE_LOG_DIR="/var/log/shadowmesh"

# Check if we have the required configuration
if [ "$UK_VPS_IP" == "<REPLACE_WITH_UK_VPS_IP>" ]; then
    echo -e "${RED}Error: Please edit this script and set UK_VPS_IP${NC}"
    echo "Edit: $0"
    exit 1
fi

if [ "$RPI_IP" == "<REPLACE_WITH_RPI_IP>" ]; then
    echo -e "${RED}Error: Please edit this script and set RPI_IP${NC}"
    echo "Edit: $0"
    exit 1
fi

# Check if binaries exist
if [ ! -f "$LOCAL_BUILD_DIR/shadowmesh-daemon" ]; then
    echo -e "${YELLOW}Building binaries first...${NC}"
    make build
fi

echo -e "${GREEN}Deployment Configuration:${NC}"
echo "  UK VPS (Listener):  $UK_VPS_USER@$UK_VPS_IP"
echo "  Belgium RPi (Connector): $RPI_USER@$RPI_IP"
echo "  TAP Device: chr-001"
echo "  UK VPS IP: 10.10.10.3/24"
echo "  RPi IP: 10.10.10.4/24"
echo ""

# Function to deploy to a machine
deploy_to_machine() {
    local HOST=$1
    local USER=$2
    local PORT=$3
    local CONFIG_FILE=$4
    local MACHINE_NAME=$5

    echo -e "${BLUE}Deploying to $MACHINE_NAME ($HOST)...${NC}"

    # Create directories
    echo "  Creating directories..."
    ssh -p $PORT $USER@$HOST "sudo mkdir -p $REMOTE_INSTALL_DIR $REMOTE_CONFIG_DIR $REMOTE_KEYS_DIR $REMOTE_LOG_DIR && \
                               sudo chown -R $USER:$USER $REMOTE_INSTALL_DIR && \
                               sudo chmod 755 $REMOTE_INSTALL_DIR"

    # Copy binary
    echo "  Uploading shadowmesh-daemon..."
    scp -P $PORT $LOCAL_BUILD_DIR/shadowmesh-daemon $USER@$HOST:$REMOTE_INSTALL_DIR/

    # Set executable permissions
    ssh -p $PORT $USER@$HOST "chmod +x $REMOTE_INSTALL_DIR/shadowmesh-daemon"

    # Copy configuration
    echo "  Uploading configuration..."

    # Update config file with actual IPs before uploading
    if [ "$MACHINE_NAME" == "Belgium RPi" ]; then
        # Replace placeholder with actual UK VPS IP
        sed "s/<UK_VPS_IP>/$UK_VPS_IP/g" $CONFIG_FILE > /tmp/rpi-config-temp.yaml
        scp -P $PORT /tmp/rpi-config-temp.yaml $USER@$HOST:$REMOTE_CONFIG_DIR/config.yaml
        rm /tmp/rpi-config-temp.yaml
    else
        scp -P $PORT $CONFIG_FILE $USER@$HOST:$REMOTE_CONFIG_DIR/config.yaml
    fi

    # Set config permissions
    ssh -p $PORT $USER@$HOST "sudo chown -R root:root $REMOTE_CONFIG_DIR && \
                               sudo chmod 600 $REMOTE_CONFIG_DIR/config.yaml"

    echo -e "${GREEN}  ✓ Deployed to $MACHINE_NAME${NC}"
    echo ""
}

# Deploy to UK VPS (listener)
deploy_to_machine "$UK_VPS_IP" "$UK_VPS_USER" "$UK_VPS_PORT" \
                  "$LOCAL_CONFIG_DIR/vps-uk-listener.yaml" "UK VPS"

# Deploy to Belgium RPi (connector)
deploy_to_machine "$RPI_IP" "$RPI_USER" "$RPI_PORT" \
                  "$LOCAL_CONFIG_DIR/rpi-belgium-connector.yaml" "Belgium RPi"

echo -e "${GREEN}╔════════════════════════════════════════════════╗"
echo "║  Deployment Complete!                          ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo ""
echo "1. Start UK VPS (Listener):"
echo "   ssh $UK_VPS_USER@$UK_VPS_IP"
echo "   sudo $REMOTE_INSTALL_DIR/shadowmesh-daemon --config $REMOTE_CONFIG_DIR/config.yaml"
echo ""
echo "2. Start Belgium RPi (Connector):"
echo "   ssh $RPI_USER@$RPI_IP"
echo "   sudo $REMOTE_INSTALL_DIR/shadowmesh-daemon --config $REMOTE_CONFIG_DIR/config.yaml"
echo ""
echo "3. Test encrypted tunnel:"
echo "   From UK VPS:  ping 10.10.10.4"
echo "   From RPi:     ping 10.10.10.3"
echo ""
echo "4. Check logs:"
echo "   tail -f $REMOTE_LOG_DIR/daemon.log"
echo ""
echo -e "${BLUE}Expected Behavior:${NC}"
echo "  - UK VPS listens on port 8443"
echo "  - RPi connects to UK VPS"
echo "  - PQC handshake (ML-KEM-1024 + ML-DSA-87)"
echo "  - Encrypted tunnel established"
echo "  - Ping should work over 10.10.10.x network"
echo ""
