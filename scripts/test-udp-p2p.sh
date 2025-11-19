#!/bin/bash
# ShadowMesh Epic 2 - UDP P2P Testing Script
# Tests direct UDP P2P connection with relay fallback

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ShadowMesh Epic 2 - UDP P2P Test${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if running as root (needed for TAP devices)
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}Error: This script must be run as root (for TAP device creation)${NC}"
   echo "Usage: sudo $0"
   exit 1
fi

# Create TAP devices
echo -e "${YELLOW}[1/6] Creating TAP devices...${NC}"
if ! ip tuntap add mode tap tap0 2>/dev/null; then
    echo -e "${YELLOW}  tap0 already exists${NC}"
fi
if ! ip tuntap add mode tap tap1 2>/dev/null; then
    echo -e "${YELLOW}  tap1 already exists${NC}"
fi
ip link set tap0 up
ip link set tap1 up
echo -e "${GREEN}  ✅ TAP devices created${NC}"
echo ""

# Assign IP addresses
echo -e "${YELLOW}[2/6] Assigning IP addresses...${NC}"
ip addr flush dev tap0
ip addr flush dev tap1
ip addr add 10.10.10.3/24 dev tap0
ip addr add 10.10.10.4/24 dev tap1
echo -e "${GREEN}  ✅ IP addresses assigned${NC}"
echo ""

# Start Endpoint 1 daemon
echo -e "${YELLOW}[3/6] Starting Endpoint 1 daemon (tap0, 10.10.10.3)...${NC}"
./bin/shadowmesh-daemon -config configs/endpoint1-udp-test.yaml &
DAEMON1_PID=$!
sleep 2
echo -e "${GREEN}  ✅ Daemon 1 started (PID: $DAEMON1_PID)${NC}"
echo ""

# Start Endpoint 2 daemon
echo -e "${YELLOW}[4/6] Starting Endpoint 2 daemon (tap1, 10.10.10.4)...${NC}"
./bin/shadowmesh-daemon -config configs/endpoint2-udp-test.yaml &
DAEMON2_PID=$!
sleep 2
echo -e "${GREEN}  ✅ Daemon 2 started (PID: $DAEMON2_PID)${NC}"
echo ""

# Connection test
echo -e "${YELLOW}[5/6] Testing connection scenarios...${NC}"
echo ""

# Scenario 1: UDP P2P attempt (will fail on localhost)
echo -e "${BLUE}Scenario 1: UDP Hole Punching Attempt${NC}"
echo "Expected: UDP will fail (loopback limitation), should fallback to relay"
echo ""
# TODO: Need to connect from endpoint 1 to endpoint 2
# This requires the peer address to be known

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}[6/6] Cleaning up...${NC}"
    if [[ ! -z "$DAEMON1_PID" ]]; then
        kill $DAEMON1_PID 2>/dev/null || true
    fi
    if [[ ! -z "$DAEMON2_PID" ]]; then
        kill $DAEMON2_PID 2>/dev/null || true
    fi
    ip link delete tap0 2>/dev/null || true
    ip link delete tap1 2>/dev/null || true
    echo -e "${GREEN}  ✅ Cleanup complete${NC}"
}

trap cleanup EXIT

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Manual Testing Instructions:${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Daemons are running. To test connection:"
echo ""
echo "Terminal 1 (Endpoint 1 status):"
echo "  curl http://localhost:9090/status"
echo ""
echo "Terminal 2 (Endpoint 2 status):"
echo "  curl http://localhost:9091/status"
echo ""
echo "Connect endpoint 1 to endpoint 2 via relay:"
echo "  curl -X POST http://localhost:9090/connect"
echo ""
echo "Connect endpoint 2 to endpoint 1 via relay:"
echo "  curl -X POST http://localhost:9091/connect"
echo ""
echo "Test connectivity with ping:"
echo "  ping -c 3 10.10.10.4  # From endpoint 1"
echo "  ping -c 3 10.10.10.3  # From endpoint 2"
echo ""
echo -e "${YELLOW}Note: UDP hole punching won't work on localhost (loopback).${NC}"
echo -e "${YELLOW}For real UDP P2P test, use two separate machines/VMs.${NC}"
echo ""
echo "Press Ctrl+C to stop daemons and cleanup..."
echo ""

# Keep script running
wait
