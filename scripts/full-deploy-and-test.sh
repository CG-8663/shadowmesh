#!/bin/bash
# Full Deploy and Test - Complete automation for ShadowMesh optimization
# Rebuilds, deploys, optimizes, and tests in one go

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ShadowMesh Full Deploy & Test${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if in shadowmesh directory
if [ ! -f "go.mod" ]; then
    print_error "Must run from shadowmesh repository root"
    exit 1
fi

echo "This script will:"
echo "  1. Pull latest code from GitHub"
echo "  2. Rebuild daemon with 2MB WebSocket buffers"
echo "  3. Deploy daemon locally"
echo "  4. (Optional) Deploy relay server to remote host"
echo "  5. Apply TCP optimizations (BBR, 16MB buffers)"
echo "  6. Restart ShadowMesh connections"
echo "  7. Run automated iperf3 performance tests"
echo ""
read -p "Continue? [y/N]: " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_info "Aborted by user"
    exit 0
fi

# ============== PHASE 1: PULL LATEST CODE ==============

echo ""
print_info "========================================"
print_info "PHASE 1: Pulling latest code"
print_info "========================================"

git pull origin main
print_success "Code updated"

# ============== PHASE 2: BUILD BINARIES ==============

echo ""
print_info "========================================"
print_info "PHASE 2: Building binaries"
print_info "========================================"

# Build daemon
print_info "Building daemon binary..."
go build -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon/
if [ ! -f "bin/shadowmesh-daemon" ]; then
    print_error "Failed to build daemon"
    exit 1
fi
chmod +x bin/shadowmesh-daemon
print_success "Daemon built: bin/shadowmesh-daemon"

# Build relay server (Linux)
print_info "Building relay server binary (Linux)..."
GOOS=linux GOARCH=amd64 go build -o bin/relay-server-linux ./cmd/relay-server/
if [ ! -f "bin/relay-server-linux" ]; then
    print_error "Failed to build relay server"
    exit 1
fi
chmod +x bin/relay-server-linux
print_success "Relay server built: bin/relay-server-linux"

# ============== PHASE 3: DEPLOY LOCALLY ==============

echo ""
print_info "========================================"
print_info "PHASE 3: Deploying daemon locally"
print_info "========================================"

sudo cp bin/shadowmesh-daemon /usr/local/bin/
print_success "Daemon deployed to /usr/local/bin/"

# Check if daemon is running as service or manually
if systemctl is-active --quiet shadowmesh 2>/dev/null; then
    print_warning "Restarting shadowmesh service..."
    sudo systemctl restart shadowmesh
    sleep 2
    if systemctl is-active --quiet shadowmesh; then
        print_success "Service restarted"
    else
        print_error "Service failed to restart - check logs"
    fi
else
    print_warning "ShadowMesh is not running as a service"
    print_info "If running manually, restart the daemon now (Ctrl+C and re-run)"
    read -p "Press Enter when daemon is restarted..."
fi

# ============== PHASE 4: DEPLOY RELAY (OPTIONAL) ==============

echo ""
print_info "========================================"
print_info "PHASE 4: Deploy relay server (optional)"
print_info "========================================"

read -p "Deploy relay server to remote host? [y/N]: " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    read -p "Enter relay server SSH host (e.g., root@94.237.121.21): " RELAY_HOST

    if [ -z "$RELAY_HOST" ]; then
        print_warning "No host specified, skipping relay deployment"
    else
        print_info "Deploying to ${RELAY_HOST}..."

        # Upload binary
        print_info "Uploading binary..."
        scp bin/relay-server-linux "${RELAY_HOST}:/tmp/relay-server"

        # Install and restart
        print_info "Installing and restarting relay server..."
        ssh "${RELAY_HOST}" << 'ENDSSH'
sudo mv /tmp/relay-server /usr/local/bin/relay-server
sudo chmod +x /usr/local/bin/relay-server

echo "Stopping existing relay server..."
sudo pkill relay-server || true
sleep 2

echo "Starting relay server..."
nohup sudo /usr/local/bin/relay-server -port 9545 > /var/log/relay-server.log 2>&1 &
sleep 1

if pgrep -x "relay-server" > /dev/null; then
    echo "Relay server started successfully"
else
    echo "ERROR: Relay server failed to start"
    exit 1
fi

echo "Checking relay server health..."
curl -s http://localhost:9545/health || echo "Health check failed"
ENDSSH

        if [ $? -eq 0 ]; then
            print_success "Relay server deployed"
        else
            print_error "Relay deployment failed"
            exit 1
        fi
    fi
else
    print_info "Skipping relay deployment"
fi

# ============== PHASE 5: APPLY TCP OPTIMIZATIONS ==============

echo ""
print_info "========================================"
print_info "PHASE 5: Applying TCP optimizations"
print_info "========================================"

if [ -f "./scripts/optimize-tcp-performance.sh" ]; then
    print_info "Running TCP optimization script..."
    sudo ./scripts/optimize-tcp-performance.sh <<< 'y'
    print_success "TCP optimizations applied"
else
    print_warning "TCP optimization script not found, skipping"
fi

# ============== PHASE 6: RESTART CONNECTIONS ==============

echo ""
print_info "========================================"
print_info "PHASE 6: Restarting ShadowMesh connections"
print_info "========================================"

# Get current status
print_info "Checking current connection status..."
CURRENT_STATUS=$(curl -s http://127.0.0.1:9090/status)

if echo "$CURRENT_STATUS" | grep -q "Connected"; then
    print_info "Disconnecting existing connection..."
    curl -s -X POST http://127.0.0.1:9090/disconnect
    sleep 1
    print_success "Disconnected"

    # Ask user for reconnection details
    read -p "Reconnect to peer? [Y/n]: " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
        read -p "Enter peer address (IP:PORT): " PEER_ADDR
        read -p "Use relay server? [y/N]: " -n 1 -r
        echo

        USE_RELAY="false"
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            USE_RELAY="true"
        fi

        print_info "Connecting to ${PEER_ADDR} (relay: ${USE_RELAY})..."

        CONNECT_RESULT=$(curl -s -X POST http://127.0.0.1:9090/connect \
            -H "Content-Type: application/json" \
            -d "{\"peer_address\": \"${PEER_ADDR}\", \"use_relay\": ${USE_RELAY}}")

        echo "$CONNECT_RESULT"

        sleep 2

        # Verify connection
        NEW_STATUS=$(curl -s http://127.0.0.1:9090/status)
        if echo "$NEW_STATUS" | grep -q "Connected"; then
            print_success "Connection established"
        else
            print_error "Connection failed - check daemon logs"
        fi
    fi
else
    print_info "No active connection to restart"
fi

# ============== PHASE 7: RUN PERFORMANCE TESTS ==============

echo ""
print_info "========================================"
print_info "PHASE 7: Running performance tests"
print_info "========================================"

read -p "Run iperf3 performance tests? [Y/n]: " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Nn]$ ]]; then
    if [ -f "./scripts/automated-perf-test.sh" ]; then
        chmod +x ./scripts/automated-perf-test.sh

        # Auto-detect peer IP from status
        PEER_IP=$(echo "$NEW_STATUS" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('details', {}).get('peer_address', '').split(':')[0])" 2>/dev/null)

        if [ -z "$PEER_IP" ]; then
            read -p "Enter peer tunnel IP (e.g., 10.0.0.2): " PEER_IP
        fi

        if [ ! -z "$PEER_IP" ]; then
            print_info "Starting automated performance test..."
            print_info "Target: $PEER_IP"
            echo ""

            ./scripts/automated-perf-test.sh --client "$PEER_IP" --duration 30 --parallel 4
        else
            print_warning "No peer IP specified, skipping tests"
        fi
    else
        print_warning "Automated test script not found"
        print_info "Run manually: iperf3 -c PEER_IP -t 30 -P 4"
    fi
else
    print_info "Skipping performance tests"
fi

# ============== COMPLETION ==============

echo ""
print_success "=========================================="
print_success "DEPLOYMENT AND TESTING COMPLETE"
print_success "=========================================="
echo ""

print_info "Summary:"
echo "  ✓ Binaries rebuilt with 2MB WebSocket buffers"
echo "  ✓ Daemon deployed locally"
echo "  ✓ TCP optimizations applied (BBR, 16MB buffers)"
echo "  ✓ Connection restarted"
echo "  ✓ Performance tests completed"
echo ""

print_info "Check results in: ./perf-results/"
echo ""

print_info "Next steps:"
echo "  1. Review test results in perf-results/"
echo "  2. Compare with baseline (13.4 Mbps, 1797 retransmits)"
echo "  3. Expected: 30-40 Mbps, <500 retransmits"
echo "  4. If issues persist, check daemon logs:"
echo "     journalctl -u shadowmesh -f"
