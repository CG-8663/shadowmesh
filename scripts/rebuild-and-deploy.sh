#!/bin/bash
# Rebuild and Deploy ShadowMesh after Buffer Optimizations
# Rebuilds daemon and relay binaries and redeploys to active systems

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ShadowMesh Rebuild & Deploy${NC}"
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
echo "  1. Pull latest changes from GitHub"
echo "  2. Rebuild daemon binary with 2MB WebSocket buffers"
echo "  3. Rebuild relay server binary"
echo "  4. Deploy to local system"
echo "  5. (Optional) Deploy relay server to remote host"
echo ""
read -p "Continue? [y/N]: " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_info "Aborted by user"
    exit 0
fi

echo ""
print_info "Step 1: Pulling latest changes from GitHub..."
git pull origin main
print_success "Repository updated"

echo ""
print_info "Step 2: Building daemon binary..."
go build -o bin/shadowmesh-daemon ./cmd/shadowmesh-daemon/
if [ -f "bin/shadowmesh-daemon" ]; then
    chmod +x bin/shadowmesh-daemon
    print_success "Daemon binary built: bin/shadowmesh-daemon"
else
    print_error "Failed to build daemon binary"
    exit 1
fi

echo ""
print_info "Step 3: Building relay server binary..."
go build -o bin/relay-server ./cmd/relay-server/
if [ -f "bin/relay-server" ]; then
    chmod +x bin/relay-server

    # Also build Linux version for deployment
    print_info "Building Linux relay server binary..."
    GOOS=linux GOARCH=amd64 go build -o bin/relay-server-linux ./cmd/relay-server/
    chmod +x bin/relay-server-linux

    print_success "Relay server binaries built"
else
    print_error "Failed to build relay server binary"
    exit 1
fi

echo ""
print_info "Step 4: Installing daemon locally..."
sudo cp bin/shadowmesh-daemon /usr/local/bin/
print_success "Daemon installed to /usr/local/bin/shadowmesh-daemon"

echo ""
print_warning "Restart ShadowMesh daemon to use new binary:"
echo "  If running as service: sudo systemctl restart shadowmesh"
echo "  If running manually: Stop daemon (Ctrl+C) and restart"

echo ""
read -p "Deploy relay server to remote host? [y/N]: " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo ""
    read -p "Enter relay server SSH host (e.g., root@94.237.121.21): " RELAY_HOST

    if [ -z "$RELAY_HOST" ]; then
        print_warning "No host specified, skipping relay deployment"
    else
        print_info "Deploying relay server to ${RELAY_HOST}..."

        # Upload binary to home directory (avoid /tmp space issues)
        scp bin/relay-server-linux "${RELAY_HOST}:~/relay-server"

        # Install and restart
        ssh "${RELAY_HOST}" << 'ENDSSH'
sudo mv ~/relay-server /usr/local/bin/relay-server
sudo chmod +x /usr/local/bin/relay-server
echo "Checking for running relay server..."
if pgrep -x "relay-server" > /dev/null; then
    echo "Stopping existing relay server..."
    sudo pkill relay-server
    sleep 2
fi
echo "Starting relay server..."
nohup sudo /usr/local/bin/relay-server -port 9545 > /var/log/relay-server.log 2>&1 &
sleep 1
if pgrep -x "relay-server" > /dev/null; then
    echo "Relay server started successfully"
else
    echo "ERROR: Relay server failed to start"
    exit 1
fi
ENDSSH

        if [ $? -eq 0 ]; then
            print_success "Relay server deployed to ${RELAY_HOST}"
        else
            print_error "Relay deployment failed"
        fi
    fi
fi

echo ""
print_success "Build and deployment complete!"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. Apply TCP optimizations (if not done already):"
echo "     ${GREEN}sudo ./scripts/optimize-tcp-performance.sh${NC}"
echo ""
echo "  2. Restart ShadowMesh connections:"
echo "     ${GREEN}curl -X POST http://127.0.0.1:9090/disconnect${NC}"
echo "     ${GREEN}curl -X POST http://127.0.0.1:9090/connect -H \"Content-Type: application/json\" -d '{\"peer_address\": \"PEER_IP:9001\", \"use_relay\": true}'${NC}"
echo ""
echo "  3. Re-run iperf3 test:"
echo "     ${GREEN}iperf3 -c 10.0.0.X -t 30 -P 4${NC}"
echo ""
echo -e "${BLUE}Expected improvements:${NC}"
echo "  - No more \"send buffer full\" errors"
echo "  - 30-40 Mbps throughput (vs 13.4 Mbps)"
echo "  - <500 retransmissions (vs 1,797)"
echo "  - 80-95% bandwidth utilization"
