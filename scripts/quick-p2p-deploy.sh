#!/bin/bash
# Quick P2P Deployment Script
# Deploys to UK VPS and Belgium RPi for immediate testing

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════╗"
echo "║  ShadowMesh P2P Quick Deploy                   ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""

# TODO: Replace these with your actual values
UK_VPS_IP="YOUR_UK_VPS_IP"
UK_VPS_USER="YOUR_USERNAME"
RPI_IP="YOUR_RPI_IP"
RPI_USER="pi"

if [ "$UK_VPS_IP" == "YOUR_UK_VPS_IP" ]; then
    echo -e "${RED}Error: Please edit this script and set UK_VPS_IP, UK_VPS_USER, and RPI_IP${NC}"
    echo "Edit: $0"
    exit 1
fi

echo -e "${GREEN}Configuration:${NC}"
echo "  UK VPS:     $UK_VPS_USER@$UK_VPS_IP (10.10.10.3)"
echo "  Belgium RPi: $RPI_USER@$RPI_IP (10.10.10.4)"
echo ""

# Build fresh binaries
echo -e "${YELLOW}Building fresh binaries...${NC}"
make build

# Deploy to UK VPS
echo ""
echo -e "${BLUE}=== Deploying to UK VPS ===${NC}"
echo "Creating directories..."
ssh $UK_VPS_USER@$UK_VPS_IP "sudo mkdir -p /opt/shadowmesh /etc/shadowmesh /var/log/shadowmesh && sudo chown -R $UK_VPS_USER:$UK_VPS_USER /opt/shadowmesh"

echo "Uploading daemon..."
scp build/shadowmesh-daemon $UK_VPS_USER@$UK_VPS_IP:/opt/shadowmesh/
ssh $UK_VPS_USER@$UK_VPS_IP "chmod +x /opt/shadowmesh/shadowmesh-daemon"

echo "Uploading config..."
# Update config with actual IP
sed "s/<UK_VPS_IP>/$UK_VPS_IP/g" configs/vps-uk-listener.yaml > /tmp/vps-config.yaml
scp /tmp/vps-config.yaml $UK_VPS_USER@$UK_VPS_IP:/etc/shadowmesh/config.yaml
rm /tmp/vps-config.yaml

echo -e "${GREEN}✓ UK VPS deployment complete${NC}"

# Deploy to Belgium RPi
echo ""
echo -e "${BLUE}=== Deploying to Belgium RPi ===${NC}"
echo "Creating directories..."
ssh $RPI_USER@$RPI_IP "sudo mkdir -p /opt/shadowmesh /etc/shadowmesh /var/log/shadowmesh && sudo chown -R $RPI_USER:$RPI_USER /opt/shadowmesh"

echo "Uploading daemon..."
scp build/shadowmesh-daemon $RPI_USER@$RPI_IP:/opt/shadowmesh/
ssh $RPI_USER@$RPI_IP "chmod +x /opt/shadowmesh/shadowmesh-daemon"

echo "Uploading config..."
# Update config with actual IP
sed "s/<UK_VPS_IP>/$UK_VPS_IP/g" configs/rpi-belgium-connector.yaml > /tmp/rpi-config.yaml
scp /tmp/rpi-config.yaml $RPI_USER@$RPI_IP:/etc/shadowmesh/config.yaml
rm /tmp/rpi-config.yaml

echo -e "${GREEN}✓ Belgium RPi deployment complete${NC}"

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════╗"
echo "║  Deployment Complete!                          ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Next: Start the daemons${NC}"
echo ""
echo "1. Terminal 1 - UK VPS (Listener):"
echo "   ${BLUE}ssh $UK_VPS_USER@$UK_VPS_IP${NC}"
echo "   ${BLUE}sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml${NC}"
echo ""
echo "2. Terminal 2 - Belgium RPi (Connector):"
echo "   ${BLUE}ssh $RPI_USER@$RPI_IP${NC}"
echo "   ${BLUE}sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml${NC}"
echo ""
echo "3. Test the tunnel:"
echo "   UK VPS:  ${GREEN}ping 10.10.10.4${NC}"
echo "   RPi:     ${GREEN}ping 10.10.10.3${NC}"
echo ""
