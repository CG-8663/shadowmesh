#!/bin/bash
# Test direct P2P connection between two machines (no relay)
# This tests the core Epic 2 functionality: direct encrypted mesh

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════╗"
echo "║    ShadowMesh Direct P2P Test                  ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""
echo "This test validates Epic 2: Core Networking & Direct P2P"
echo "Testing direct encrypted tunnel WITHOUT relay server"
echo ""

# Architecture explanation
echo -e "${YELLOW}Architecture (Direct P2P Mode):${NC}"
echo ""
echo "  Machine A (MacBook Pro)        Machine B (Mac Studio)"
echo "  ┌──────────────────┐          ┌──────────────────┐"
echo "  │  tap0: 10.0.0.2  │◄────────►│  tap0: 10.0.0.3  │"
echo "  │                  │   WSS    │                  │"
echo "  │  shadowmesh-     │  Direct  │  shadowmesh-     │"
echo "  │  daemon          │  P2P     │  daemon          │"
echo "  │                  │  Tunnel  │                  │"
echo "  │  ML-KEM-1024     │          │  ML-KEM-1024     │"
echo "  │  ChaCha20        │          │  ChaCha20        │"
echo "  └──────────────────┘          └──────────────────┘"
echo ""
echo "Data flow: App → TAP → Encrypt → WSS → Decrypt → TAP → App"
echo ""

# Test prerequisites
echo -e "${GREEN}Checking prerequisites...${NC}"
echo "--------------------------------------"

# Check if running as root (required for TAP devices)
if [ "$EUID" -ne 0 ]; then 
    echo -e "${YELLOW}⚠ Not running as root${NC}"
    echo "TAP device creation requires root privileges"
    echo "Run with: sudo $0"
    exit 1
fi

# Check if tap driver exists (macOS)
if [[ "$OSTYPE" == "darwin"* ]]; then
    if ! kextstat | grep -q "tap"; then
        echo -e "${YELLOW}⚠ TAP driver not loaded${NC}"
        echo "Install TunTap for macOS: https://sourceforge.net/projects/tuntaposx/"
        exit 1
    fi
    echo "✓ TAP driver loaded"
fi

# Check if shadowmesh-daemon exists
if [ ! -f "./build/shadowmesh-daemon" ]; then
    echo -e "${YELLOW}⚠ shadowmesh-daemon not built${NC}"
    echo "Run: make build"
    exit 1
fi
echo "✓ shadowmesh-daemon built"

echo ""
echo -e "${GREEN}Test Configuration${NC}"
echo "--------------------------------------"
echo "This machine will:"
echo "  1. Create TAP device (tap0)"
echo "  2. Configure IP: 10.0.0.2/24"
echo "  3. Generate PQC keys"
echo "  4. Start daemon in listening mode"
echo ""
echo "On the second machine, run:"
echo "  sudo ./build/shadowmesh-daemon --connect <this-machine-ip>"
echo ""
echo "Then test connectivity:"
echo "  ping 10.0.0.3  # From this machine"
echo "  ping 10.0.0.2  # From other machine"
echo ""

# Create test config
cat > /tmp/shadowmesh-test.yaml << 'YAML'
local_id: "test-node-1"
private_key_path: "/tmp/shadowmesh-test-key"
tap:
  device_name: "tap0"
  ip_address: "10.0.0.2"
  netmask: "255.255.255.0"
  mtu: 1500
relay_url: ""  # Direct P2P, no relay
peers: []
log_level: "debug"
log_file: "/tmp/shadowmesh-test.log"
YAML

echo -e "${GREEN}Created test configuration:${NC}"
cat /tmp/shadowmesh-test.yaml
echo ""

echo -e "${BLUE}Ready to start daemon?${NC}"
echo "This will:"
echo "  - Create tap0 device"
echo "  - Assign IP 10.0.0.2"
echo "  - Listen for connections"
echo ""
echo "Press Ctrl+C to exit"
echo ""

# Start daemon (this will block)
# ./build/shadowmesh-daemon --config /tmp/shadowmesh-test.yaml

echo -e "${YELLOW}To start daemon:${NC}"
echo "  sudo ./build/shadowmesh-daemon --config /tmp/shadowmesh-test.yaml"
