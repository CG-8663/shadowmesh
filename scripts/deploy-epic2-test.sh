#!/bin/bash
# Deploy Epic 2 to Production Endpoints for UDP P2P Testing

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ShadowMesh Epic 2 - Production Deploy${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Configuration (UPDATE THESE WITH YOUR ENDPOINTS)
ENDPOINT1_HOST="${ENDPOINT1_HOST:-user@endpoint1}"  # Belgium RPi or similar
ENDPOINT2_HOST="${ENDPOINT2_HOST:-user@endpoint2}"  # UK VPS or similar

echo -e "${YELLOW}Deployment Configuration:${NC}"
echo "  Endpoint 1: $ENDPOINT1_HOST"
echo "  Endpoint 2: $ENDPOINT2_HOST"
echo "  Relay Server: ws://94.237.121.21:9545"
echo ""

read -p "Continue with deployment? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled"
    exit 1
fi

# Build binaries for Linux
echo -e "${YELLOW}[1/6] Building Linux binaries...${NC}"

# ARM64 for Raspberry Pi
echo "  Building ARM64 binary..."
GOOS=linux GOARCH=arm64 go build -o bin/shadowmesh-daemon-arm64 cmd/shadowmesh-daemon/main.go
echo -e "${GREEN}  ✅ ARM64 binary built${NC}"

# AMD64 for VPS
echo "  Building AMD64 binary..."
GOOS=linux GOARCH=amd64 go build -o bin/shadowmesh-daemon-amd64 cmd/shadowmesh-daemon/main.go
echo -e "${GREEN}  ✅ AMD64 binary built${NC}"

echo ""

# Deploy to Endpoint 1
echo -e "${YELLOW}[2/6] Deploying to Endpoint 1...${NC}"
scp bin/shadowmesh-daemon-arm64 ${ENDPOINT1_HOST}:/tmp/shadowmesh-daemon
scp configs/endpoint1-udp-test.yaml ${ENDPOINT1_HOST}:/tmp/shadowmesh-config.yaml
echo -e "${GREEN}  ✅ Endpoint 1 files deployed${NC}"
echo ""

# Deploy to Endpoint 2
echo -e "${YELLOW}[3/6] Deploying to Endpoint 2...${NC}"
scp bin/shadowmesh-daemon-amd64 ${ENDPOINT2_HOST}:/tmp/shadowmesh-daemon
scp configs/endpoint2-udp-test.yaml ${ENDPOINT2_HOST}:/tmp/shadowmesh-config.yaml
echo -e "${GREEN}  ✅ Endpoint 2 files deployed${NC}"
echo ""

# Setup Endpoint 1
echo -e "${YELLOW}[4/6] Setting up Endpoint 1...${NC}"
ssh ${ENDPOINT1_HOST} << 'EOF'
# Create TAP device
sudo ip tuntap add mode tap tap0 2>/dev/null || true
sudo ip link set tap0 up
sudo ip addr flush dev tap0
sudo ip addr add 10.10.10.3/24 dev tap0

# Stop existing daemon if running
sudo pkill shadowmesh-daemon || true
sleep 1

# Start new daemon
sudo /tmp/shadowmesh-daemon -config /tmp/shadowmesh-config.yaml > /tmp/shadowmesh.log 2>&1 &
sleep 2

echo "Endpoint 1 status:"
tail -20 /tmp/shadowmesh.log
EOF
echo -e "${GREEN}  ✅ Endpoint 1 configured and started${NC}"
echo ""

# Setup Endpoint 2
echo -e "${YELLOW}[5/6] Setting up Endpoint 2...${NC}"
ssh ${ENDPOINT2_HOST} << 'EOF'
# Create TAP device
sudo ip tuntap add mode tap tap1 2>/dev/null || true
sudo ip link set tap1 up
sudo ip addr flush dev tap1
sudo ip addr add 10.10.10.4/24 dev tap1

# Stop existing daemon if running
sudo pkill shadowmesh-daemon || true
sleep 1

# Start new daemon
sudo /tmp/shadowmesh-daemon -config /tmp/shadowmesh-config.yaml > /tmp/shadowmesh.log 2>&1 &
sleep 2

echo "Endpoint 2 status:"
tail -20 /tmp/shadowmesh.log
EOF
echo -e "${GREEN}  ✅ Endpoint 2 configured and started${NC}"
echo ""

# Test connection
echo -e "${YELLOW}[6/6] Testing connection...${NC}"
echo ""

echo "Initiating connection from Endpoint 1..."
ssh ${ENDPOINT1_HOST} "curl -X POST http://localhost:9090/connect 2>/dev/null" || true
sleep 3
echo ""

echo "Endpoint 1 logs (UDP P2P attempt):"
echo "========================================"
ssh ${ENDPOINT1_HOST} "tail -30 /tmp/shadowmesh.log"
echo ""

echo "Endpoint 2 logs (waiting for connection):"
echo "========================================"
ssh ${ENDPOINT2_HOST} "tail -30 /tmp/shadowmesh.log"
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Testing Commands${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Check Endpoint 1 status:"
echo "  ssh ${ENDPOINT1_HOST} 'curl http://localhost:9090/status'"
echo ""
echo "Check Endpoint 2 status:"
echo "  ssh ${ENDPOINT2_HOST} 'curl http://localhost:9091/status'"
echo ""
echo "Test connectivity (from Endpoint 1):"
echo "  ssh ${ENDPOINT1_HOST} 'ping -c 3 10.10.10.4'"
echo ""
echo "Test connectivity (from Endpoint 2):"
echo "  ssh ${ENDPOINT2_HOST} 'ping -c 3 10.10.10.3'"
echo ""
echo "Run iperf3 test:"
echo "  ssh ${ENDPOINT2_HOST} 'iperf3 -s -B 10.10.10.4 &'"
echo "  ssh ${ENDPOINT1_HOST} 'iperf3 -c 10.10.10.4 -B 10.10.10.3 -t 30 -P 4'"
echo ""
echo "View live logs:"
echo "  ssh ${ENDPOINT1_HOST} 'tail -f /tmp/shadowmesh.log'"
echo "  ssh ${ENDPOINT2_HOST} 'tail -f /tmp/shadowmesh.log'"
echo ""

echo -e "${GREEN}✅ Epic 2 deployment complete!${NC}"
echo ""
echo "Look for these log messages to verify UDP P2P:"
echo "  - 'Attempting direct UDP P2P connection...'"
echo "  - 'NAT type is compatible with direct P2P...'"
echo "  - 'UDP hole punching...' (attempt)"
echo "  - '✅ Direct UDP P2P connection established' (success)"
echo "  - 'Falling back to relay mode...' (fallback)"
