#!/bin/bash
# ShadowMesh Epic 2 - Connection Logic Test (No TAP devices required)
# Tests UDP hole punching attempt and relay fallback without TAP device setup

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ShadowMesh Epic 2 - Connection Test${NC}"
echo -e "${BLUE}(Simplified - No TAP Devices)${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Create minimal test configs without TAP devices
echo -e "${YELLOW}[1/4] Creating test configurations...${NC}"

cat > /tmp/shadowmesh-test1.yaml <<EOF
daemon:
  listen_address: "127.0.0.1:19090"
  log_level: "info"

network:
  tap_device: "tap99"  # Won't be created, just for config
  local_ip: "10.10.10.3/24"

encryption:
  key: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

peer:
  address: ""
  id: "test-endpoint1"

nat:
  enabled: true
  stun_server: "stun.l.google.com:19302"

relay:
  enabled: true
  server: "ws://94.237.121.21:9545"
EOF

cat > /tmp/shadowmesh-test2.yaml <<EOF
daemon:
  listen_address: "127.0.0.1:19091"
  log_level: "info"

network:
  tap_device: "tap98"  # Won't be created, just for config
  local_ip: "10.10.10.4/24"

encryption:
  key: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

peer:
  address: ""
  id: "test-endpoint2"

nat:
  enabled: true
  stun_server: "stun.l.google.com:19302"

relay:
  enabled: true
  server: "ws://94.237.121.21:9545"
EOF

echo -e "${GREEN}  ✅ Test configurations created${NC}"
echo ""

# Test relay server connectivity
echo -e "${YELLOW}[2/4] Testing relay server connectivity...${NC}"
if curl -s -o /dev/null -w "%{http_code}" http://94.237.121.21:9545/health | grep -q "200"; then
    echo -e "${GREEN}  ✅ Relay server is reachable${NC}"
else
    echo -e "${RED}  ❌ Relay server is NOT reachable${NC}"
    echo -e "${RED}     This test requires the production relay server${NC}"
    exit 1
fi
echo ""

# Start endpoint 1 daemon (will fail on TAP device but continue)
echo -e "${YELLOW}[3/4] Starting test daemons...${NC}"
echo -e "${BLUE}Note: TAP device errors are expected and can be ignored${NC}"
echo ""

./bin/shadowmesh-daemon -config /tmp/shadowmesh-test1.yaml > /tmp/daemon1.log 2>&1 &
DAEMON1_PID=$!
sleep 2

./bin/shadowmesh-daemon -config /tmp/shadowmesh-test2.yaml > /tmp/daemon2.log 2>&1 &
DAEMON2_PID=$!
sleep 2

echo -e "${GREEN}  ✅ Daemons started (PIDs: $DAEMON1_PID, $DAEMON2_PID)${NC}"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}[4/4] Cleaning up...${NC}"
    if [[ ! -z "$DAEMON1_PID" ]]; then
        kill $DAEMON1_PID 2>/dev/null || true
    fi
    if [[ ! -z "$DAEMON2_PID" ]]; then
        kill $DAEMON2_PID 2>/dev/null || true
    fi
    rm -f /tmp/shadowmesh-test1.yaml /tmp/shadowmesh-test2.yaml
    echo -e "${GREEN}  ✅ Cleanup complete${NC}"
}

trap cleanup EXIT

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Status${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check daemon 1 status
echo "Endpoint 1 Status:"
if curl -s http://localhost:19090/status 2>/dev/null | grep -q "state"; then
    echo -e "${GREEN}  ✅ Daemon 1 API responding${NC}"
    curl -s http://localhost:19090/status | python3 -m json.tool
else
    echo -e "${RED}  ❌ Daemon 1 not responding${NC}"
    echo "Logs:"
    tail -20 /tmp/daemon1.log
fi
echo ""

# Check daemon 2 status
echo "Endpoint 2 Status:"
if curl -s http://localhost:19091/status 2>/dev/null | grep -q "state"; then
    echo -e "${GREEN}  ✅ Daemon 2 API responding${NC}"
    curl -s http://localhost:19091/status | python3 -m json.tool
else
    echo -e "${RED}  ❌ Daemon 2 not responding${NC}"
    echo "Logs:"
    tail -20 /tmp/daemon2.log
fi
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Connection Test${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

echo "Attempting connection from Endpoint 1 to relay..."
echo ""

if curl -X POST http://localhost:19090/connect 2>/dev/null; then
    echo ""
    echo -e "${GREEN}✅ Connection initiated successfully${NC}"
else
    echo ""
    echo -e "${RED}❌ Connection failed${NC}"
fi

echo ""
echo "Daemon 1 Logs (last 30 lines):"
echo "========================================"
tail -30 /tmp/daemon1.log
echo ""

echo "Daemon 2 Logs (last 30 lines):"
echo "========================================"
tail -30 /tmp/daemon2.log
echo ""

echo -e "${YELLOW}Press Ctrl+C to stop test and cleanup...${NC}"
wait
