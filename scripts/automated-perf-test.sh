#!/bin/bash
# Automated Performance Test Suite for ShadowMesh
# Runs comprehensive iperf3 tests and collects metrics

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ShadowMesh Automated Performance Test${NC}"
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

# Parse command line arguments
MODE=""
PEER_IP=""
DURATION=30
PARALLEL=4

while [[ $# -gt 0 ]]; do
    case $1 in
        --server)
            MODE="server"
            shift
            ;;
        --client)
            MODE="client"
            PEER_IP="$2"
            shift 2
            ;;
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --parallel)
            PARALLEL="$2"
            shift 2
            ;;
        *)
            echo "Usage: $0 [--server | --client PEER_IP] [--duration SECONDS] [--parallel STREAMS]"
            echo ""
            echo "Examples:"
            echo "  Server: $0 --server"
            echo "  Client: $0 --client 10.0.0.2 --duration 30 --parallel 4"
            exit 1
            ;;
    esac
done

# Auto-detect mode if not specified
if [ -z "$MODE" ]; then
    LOCAL_IP=$(ip addr show tap0 2>/dev/null | grep "inet " | awk '{print $2}' | cut -d'/' -f1)

    if [ -z "$LOCAL_IP" ]; then
        print_error "TAP device tap0 not found. Is ShadowMesh daemon running?"
        exit 1
    fi

    echo "Detected local IP: $LOCAL_IP"
    echo ""
    echo "Select mode:"
    echo "  1) Server - Run iperf3 server (receiver)"
    echo "  2) Client - Run iperf3 client (sender)"
    echo ""
    read -p "Enter choice [1-2]: " CHOICE

    case $CHOICE in
        1)
            MODE="server"
            ;;
        2)
            MODE="client"
            read -p "Enter peer IP address: " PEER_IP
            if [ -z "$PEER_IP" ]; then
                print_error "Peer IP required for client mode"
                exit 1
            fi
            ;;
        *)
            print_error "Invalid choice"
            exit 1
            ;;
    esac
fi

# Timestamp for log files
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="./perf-results"
mkdir -p "$LOG_DIR"

if [ "$MODE" = "server" ]; then
    # ==================== SERVER MODE ====================

    print_info "Running in SERVER mode..."
    echo ""

    # Kill existing iperf3 servers
    print_info "Checking for existing iperf3 servers..."
    if pgrep -x iperf3 > /dev/null; then
        print_warning "Killing existing iperf3 processes..."
        sudo pkill iperf3
        sleep 1
    fi

    # Start iperf3 server
    PORT=5202
    print_info "Starting iperf3 server on port $PORT..."
    print_success "Server ready. Waiting for client connections..."
    echo ""
    echo -e "${GREEN}On the client, run:${NC}"
    echo -e "${GREEN}  ./scripts/automated-perf-test.sh --client $(ip addr show tap0 2>/dev/null | grep "inet " | awk '{print $2}' | cut -d'/' -f1)${NC}"
    echo ""

    # Run server with JSON output for logging
    iperf3 -s -p $PORT --json --logfile "$LOG_DIR/server_${TIMESTAMP}.json" | tee "$LOG_DIR/server_${TIMESTAMP}.log"

else
    # ==================== CLIENT MODE ====================

    print_info "Running in CLIENT mode..."
    print_info "Target server: $PEER_IP"
    print_info "Test duration: ${DURATION}s"
    print_info "Parallel streams: $PARALLEL"
    echo ""

    # Check connectivity
    print_info "Testing connectivity to $PEER_IP..."
    if ! ping -c 3 -W 2 $PEER_IP > /dev/null 2>&1; then
        print_error "Cannot ping $PEER_IP - check tunnel connection"
        exit 1
    fi
    print_success "Connectivity OK"

    # Collect system info before test
    print_info "Collecting system information..."

    SYS_INFO_FILE="$LOG_DIR/sysinfo_${TIMESTAMP}.txt"

    {
        echo "=== System Information ==="
        echo "Timestamp: $(date)"
        echo "Hostname: $(hostname)"
        echo "Kernel: $(uname -r)"
        echo ""
        echo "=== Network Configuration ==="
        echo "TAP device:"
        ip addr show tap0 2>/dev/null || echo "tap0 not found"
        echo ""
        echo "=== TCP Configuration ==="
        echo "Congestion control: $(sysctl -n net.ipv4.tcp_congestion_control)"
        echo "TCP rmem: $(sysctl -n net.ipv4.tcp_rmem)"
        echo "TCP wmem: $(sysctl -n net.ipv4.tcp_wmem)"
        echo "Window scaling: $(sysctl -n net.ipv4.tcp_window_scaling)"
        echo ""
        echo "=== ShadowMesh Status ==="
        curl -s http://127.0.0.1:9090/status | python3 -m json.tool 2>/dev/null || echo "Failed to get status"
        echo ""
    } > "$SYS_INFO_FILE"

    print_success "System info saved to $SYS_INFO_FILE"

    # Run iperf3 tests
    PORT=5202

    echo ""
    print_info "==================================================="
    print_info "Test 1: Baseline TCP test ($PARALLEL streams)"
    print_info "==================================================="

    RESULT_FILE="$LOG_DIR/client_tcp_${TIMESTAMP}.json"

    iperf3 -c $PEER_IP -p $PORT \
        -t $DURATION \
        -P $PARALLEL \
        --json \
        --logfile "$RESULT_FILE" | tee "$LOG_DIR/client_tcp_${TIMESTAMP}.log"

    # Parse results
    if [ -f "$RESULT_FILE" ]; then
        print_info "Parsing results..."

        THROUGHPUT_SENDER=$(python3 -c "import json; data=json.load(open('$RESULT_FILE')); print(f\"{data['end']['sum_sent']['bits_per_second']/1000000:.2f}\")" 2>/dev/null || echo "N/A")
        THROUGHPUT_RECEIVER=$(python3 -c "import json; data=json.load(open('$RESULT_FILE')); print(f\"{data['end']['sum_received']['bits_per_second']/1000000:.2f}\")" 2>/dev/null || echo "N/A")
        RETRANSMITS=$(python3 -c "import json; data=json.load(open('$RESULT_FILE')); print(data['end']['sum_sent']['retransmits'])" 2>/dev/null || echo "N/A")

        echo ""
        print_success "========== TEST RESULTS =========="
        echo -e "${GREEN}Throughput (sender):   ${THROUGHPUT_SENDER} Mbps${NC}"
        echo -e "${GREEN}Throughput (receiver): ${THROUGHPUT_RECEIVER} Mbps${NC}"
        echo -e "${GREEN}Retransmissions:       ${RETRANSMITS}${NC}"
        echo -e "${GREEN}==================================${NC}"
        echo ""

        # Save summary
        SUMMARY_FILE="$LOG_DIR/summary_${TIMESTAMP}.txt"
        {
            echo "ShadowMesh Performance Test Summary"
            echo "Date: $(date)"
            echo "Peer: $PEER_IP"
            echo "Duration: ${DURATION}s"
            echo "Parallel streams: $PARALLEL"
            echo ""
            echo "Results:"
            echo "  Throughput (sender):   ${THROUGHPUT_SENDER} Mbps"
            echo "  Throughput (receiver): ${THROUGHPUT_RECEIVER} Mbps"
            echo "  Retransmissions:       ${RETRANSMITS}"
        } > "$SUMMARY_FILE"

        print_success "Summary saved to $SUMMARY_FILE"
    fi

    # Optional: Single stream test
    echo ""
    read -p "Run single-stream test for comparison? [y/N]: " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "==================================================="
        print_info "Test 2: Single stream TCP test"
        print_info "==================================================="

        iperf3 -c $PEER_IP -p $PORT \
            -t $DURATION \
            --json \
            --logfile "$LOG_DIR/client_single_${TIMESTAMP}.json" | tee "$LOG_DIR/client_single_${TIMESTAMP}.log"
    fi

    # Check for WebSocket errors in daemon logs
    echo ""
    print_info "Checking for WebSocket buffer errors..."

    if journalctl -u shadowmesh --since "1 minute ago" 2>/dev/null | grep -i "buffer full" > /dev/null; then
        print_error "WebSocket buffer errors detected! Check daemon logs:"
        print_warning "  journalctl -u shadowmesh --since '1 minute ago' | grep 'buffer full'"
    else
        print_success "No buffer errors detected"
    fi

    echo ""
    print_success "All tests complete!"
    print_info "Results saved to: $LOG_DIR/"
    echo ""

    ls -lh "$LOG_DIR"/*${TIMESTAMP}*
fi
