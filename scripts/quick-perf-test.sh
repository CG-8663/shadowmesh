#!/bin/bash
# ShadowMesh v11 Phase 3 - Quick Performance Test Script
# Tests ICMP, TCP, and UDP performance locally or between servers

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY="./shadowmesh-l3-v11-phase3-darwin-arm64"
KEYDIR="./keys-test"
BACKBONE="http://209.151.148.121:8080"
TUN_DEVICE="smtest0"

# Test parameters
ICMP_COUNT=100
IPERF_DURATION=30

# Functions
print_header() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_error "Please run as root (sudo)"
        exit 1
    fi
}

check_binary() {
    local os=$(uname -s)
    local arch=$(uname -m)

    if [ "$os" == "Darwin" ] && [ "$arch" == "arm64" ]; then
        BINARY="./shadowmesh-l3-v11-phase3-darwin-arm64"
    elif [ "$os" == "Linux" ] && [ "$arch" == "x86_64" ]; then
        BINARY="./shadowmesh-l3-v11-phase3-amd64"
    elif [ "$os" == "Linux" ] && [ "$arch" == "aarch64" ]; then
        BINARY="./shadowmesh-l3-v11-phase3-arm64"
    else
        print_error "Unsupported platform: $os $arch"
        exit 1
    fi

    if [ ! -f "$BINARY" ]; then
        print_error "Binary not found: $BINARY"
        print_warning "Please run this script from the shadowmesh directory"
        exit 1
    fi

    print_success "Using binary: $BINARY"
}

generate_keys() {
    print_header "Generating Test Keys"

    if [ -d "$KEYDIR" ]; then
        print_warning "Keys directory exists, skipping generation"
        PEER_ID=$(cat "$KEYDIR/peer_id.txt")
        print_success "Peer ID: $PEER_ID"
        return
    fi

    mkdir -p "$KEYDIR"
    $BINARY -generate-keys -keydir "$KEYDIR"

    PEER_ID=$(cat "$KEYDIR/peer_id.txt")
    print_success "Generated Peer ID: $PEER_ID"
}

test_local_mode() {
    print_header "Test 1: Local Mode (TUN Device Creation)"

    print_warning "This test validates TUN device creation and IP configuration"

    # Get local IP
    if [ "$(uname -s)" == "Darwin" ]; then
        LOCAL_IP=$(ifconfig | grep "inet " | grep -v 127.0.0.1 | head -1 | awk '{print $2}')
    else
        LOCAL_IP=$(ip -4 addr show | grep inet | grep -v 127.0.0.1 | head -1 | awk '{print $2}' | cut -d/ -f1)
    fi

    if [ -z "$LOCAL_IP" ]; then
        LOCAL_IP="127.0.0.1"
    fi

    print_success "Using IP: $LOCAL_IP"

    # Start in background
    $BINARY \
        -keydir "$KEYDIR" \
        -backbone "$BACKBONE" \
        -ip "$LOCAL_IP" \
        -tun "$TUN_DEVICE" \
        -tun-ip "10.100.0.1" \
        -tun-netmask "24" \
        -port 9443 \
        -udp-port 9444 > /tmp/shadowmesh-test.log 2>&1 &

    SHADOWMESH_PID=$!

    sleep 5

    # Check if process is running
    if ps -p $SHADOWMESH_PID > /dev/null; then
        print_success "ShadowMesh running (PID: $SHADOWMESH_PID)"
    else
        print_error "ShadowMesh failed to start"
        print_error "Log output:"
        cat /tmp/shadowmesh-test.log
        return 1
    fi

    # Check TUN device
    if ifconfig "$TUN_DEVICE" > /dev/null 2>&1; then
        print_success "TUN device created: $TUN_DEVICE"
        ifconfig "$TUN_DEVICE" | grep "inet " | head -1
    else
        print_error "TUN device not found"
        print_error "Log output:"
        cat /tmp/shadowmesh-test.log
        kill $SHADOWMESH_PID 2>/dev/null || true
        return 1
    fi

    # Test local ping
    print_warning "Testing local TUN interface (3 packets)..."
    if ping -c 3 10.100.0.1 > /dev/null 2>&1; then
        print_success "Local ping successful"
    else
        print_warning "Local ping failed (may need routing configuration)"
    fi

    # Show logs
    print_warning "ShadowMesh output:"
    tail -20 /tmp/shadowmesh-test.log

    # Cleanup
    print_warning "Stopping ShadowMesh..."
    kill $SHADOWMESH_PID 2>/dev/null || true
    sleep 2

    print_success "Test 1 complete"
}

show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --local           Run local TUN device test"
    echo "  --cleanup         Cleanup processes and devices"
    echo "  -h, --help        Show this help"
    echo ""
    echo "Examples:"
    echo "  sudo ./scripts/quick-perf-test.sh --local"
    echo "  sudo ./scripts/quick-perf-test.sh --cleanup"
    echo ""
}

cleanup() {
    print_header "Cleanup"

    # Kill any running ShadowMesh processes
    pkill -f shadowmesh-l3-v11-phase3 2>/dev/null || true

    # Remove TUN device (if on Linux)
    if [ "$(uname -s)" == "Linux" ]; then
        ip link del "$TUN_DEVICE" 2>/dev/null || true
    fi

    print_success "Cleanup complete"
}

# Main
main() {
    print_header "ShadowMesh v11 Phase 3 Performance Test"

    if [ $# -eq 0 ]; then
        show_help
        exit 0
    fi

    check_root
    check_binary

    # Parse arguments
    case "$1" in
        --local)
            generate_keys
            test_local_mode
            cleanup
            ;;
        --cleanup)
            cleanup
            ;;
        -h|--help)
            show_help
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac

    print_header "Test Complete"
}

# Trap cleanup on exit
trap cleanup EXIT

main "$@"
