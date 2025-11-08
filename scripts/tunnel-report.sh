#!/bin/bash
# ShadowMesh Tunnel Health & Performance Report
# Run this on UK client (100.115.193.115) to test tunnel to Philippines

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Configuration
REMOTE_TUNNEL_IP="10.10.10.5"  # Philippines tunnel IP
LOCAL_TUNNEL_IP="10.10.10.3"   # UK tunnel IP
TUN_DEVICE="chr001"
LOG_FILE="$HOME/shadowmesh/l3-v11-rtt-fixed.log"

echo -e "${BOLD}========================================${NC}"
echo -e "${BOLD}  ShadowMesh Tunnel Health Report${NC}"
echo -e "${BOLD}========================================${NC}"
echo ""
echo -e "Report Time: $(date '+%Y-%m-%d %H:%M:%S %Z')"
echo -e "Client: UK (100.115.193.115)"
echo -e "Remote: Philippines (100.126.75.74)"
echo ""

# Test 1: Process Status
echo -e "${BOLD}[1/8] Process Status${NC}"
if pgrep -f shadowmesh-l3-v11-rtt-fixed > /dev/null; then
    PID=$(pgrep -f shadowmesh-l3-v11-rtt-fixed)
    UPTIME=$(ps -p $PID -o etime= | tr -d ' ')
    MEM=$(ps -p $PID -o rss= | awk '{printf "%.1f MB", $1/1024}')
    CPU=$(ps -p $PID -o %cpu= | tr -d ' ')
    echo -e "  Status: ${GREEN}✓ Running${NC}"
    echo -e "  PID: $PID"
    echo -e "  Uptime: $UPTIME"
    echo -e "  Memory: $MEM"
    echo -e "  CPU: ${CPU}%"
else
    echo -e "  Status: ${RED}✗ Not Running${NC}"
    exit 1
fi
echo ""

# Test 2: TUN Device Status
echo -e "${BOLD}[2/8] TUN Device Status${NC}"
if ip link show $TUN_DEVICE &> /dev/null; then
    STATE=$(ip link show $TUN_DEVICE | grep -oP 'state \K\w+')
    IP=$(ip addr show $TUN_DEVICE | grep -oP 'inet \K[\d.]+')
    MTU=$(ip link show $TUN_DEVICE | grep -oP 'mtu \K\d+')

    if [ "$STATE" = "UNKNOWN" ] || [ "$STATE" = "UP" ]; then
        echo -e "  Device: ${GREEN}✓ $TUN_DEVICE${NC}"
    else
        echo -e "  Device: ${YELLOW}⚠ $TUN_DEVICE (state: $STATE)${NC}"
    fi
    echo -e "  IP Address: $IP"
    echo -e "  MTU: $MTU bytes"
else
    echo -e "  Device: ${RED}✗ $TUN_DEVICE not found${NC}"
fi
echo ""

# Test 3: Routing Table
echo -e "${BOLD}[3/8] Routing Configuration${NC}"
if ip route | grep -q "$TUN_DEVICE"; then
    ROUTES=$(ip route | grep "$TUN_DEVICE" | head -3)
    echo -e "  Routes: ${GREEN}✓ Configured${NC}"
    echo "$ROUTES" | while read line; do
        echo -e "    $line"
    done
else
    echo -e "  Routes: ${YELLOW}⚠ No routes found${NC}"
fi
echo ""

# Test 4: Connectivity Test (Ping)
echo -e "${BOLD}[4/8] Tunnel Connectivity${NC}"
echo -n "  Testing ping to $REMOTE_TUNNEL_IP... "

if timeout 5 ping -c 3 -W 2 $REMOTE_TUNNEL_IP &> /tmp/ping_test.txt; then
    LOSS=$(grep 'packet loss' /tmp/ping_test.txt | grep -oP '\d+(?=% packet loss)')
    AVG_RTT=$(grep 'rtt min/avg/max' /tmp/ping_test.txt | grep -oP 'avg/\K[\d.]+' || echo "N/A")

    if [ "$LOSS" = "0" ]; then
        echo -e "${GREEN}✓ Success${NC}"
        echo -e "  Packet Loss: ${GREEN}0%${NC}"
        if [ "$AVG_RTT" != "N/A" ]; then
            echo -e "  Average RTT: ${GREEN}${AVG_RTT}ms${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ Partial${NC}"
        echo -e "  Packet Loss: ${YELLOW}${LOSS}%${NC}"
    fi
else
    echo -e "${RED}✗ Failed${NC}"
    echo -e "  Status: ${RED}No response from remote tunnel${NC}"
fi
rm -f /tmp/ping_test.txt
echo ""

# Test 5: RTT Measurements from Logs
echo -e "${BOLD}[5/8] UDP Echo RTT Measurements${NC}"
if [ -f "$LOG_FILE" ]; then
    RECENT_RTT=$(grep '\[RTT\]' "$LOG_FILE" | tail -5)

    if [ -n "$RECENT_RTT" ]; then
        echo -e "  Recent RTT samples:"
        echo "$RECENT_RTT" | while read line; do
            RTT=$(echo "$line" | grep -oP 'Peer.*?: \K[0-9.]+[µm]s')
            AVG=$(echo "$line" | grep -oP 'avg \K[0-9.]+[µms]+')
            if [ -n "$RTT" ]; then
                echo -e "    Last: ${GREEN}$RTT${NC}  |  Avg: ${BLUE}$AVG${NC}"
            fi
        done | tail -1

        # Get latest average
        LATEST_AVG=$(echo "$RECENT_RTT" | tail -1 | grep -oP 'avg \K[0-9.]+[µms]+')
        if [ -n "$LATEST_AVG" ]; then
            echo -e "  Current Average: ${BOLD}${BLUE}$LATEST_AVG${NC}"
        fi
    else
        echo -e "  Status: ${YELLOW}⚠ No RTT measurements in log${NC}"
    fi
else
    echo -e "  Status: ${YELLOW}⚠ Log file not found${NC}"
fi
echo ""

# Test 6: Adaptive Buffer Status
echo -e "${BOLD}[6/8] Adaptive Buffer Status${NC}"
if [ -f "$LOG_FILE" ]; then
    BUFFER_INFO=$(grep '\[ADAPTIVE-BUFFER\]' "$LOG_FILE" | tail -3)

    if [ -n "$BUFFER_INFO" ]; then
        LINK_TYPE=$(echo "$BUFFER_INFO" | grep -oP 'Detected \K(satellite|terrestrial) link' | tail -1)
        BUFFER_SIZE=$(echo "$BUFFER_INFO" | grep -oP 'buffer: \d+ → \K\d+' | tail -1)
        BUFFER_INIT=$(echo "$BUFFER_INFO" | grep -oP 'Initial buffer size: \K\d+' | tail -1)

        if [ -n "$LINK_TYPE" ]; then
            if [[ "$LINK_TYPE" == "satellite"* ]]; then
                echo -e "  Link Type: ${GREEN}Satellite${NC}"
            else
                echo -e "  Link Type: ${BLUE}Terrestrial${NC}"
            fi
        fi

        if [ -n "$BUFFER_SIZE" ]; then
            echo -e "  Buffer Size: ${BOLD}${BUFFER_SIZE} packets${NC}"
        elif [ -n "$BUFFER_INIT" ]; then
            echo -e "  Buffer Size: ${BOLD}${BUFFER_INIT} packets${NC} (initial)"
        fi

        # Show recommendation
        RECOMMENDATION=$(grep 'Recommendation:' "$LOG_FILE" | tail -1 | cut -d':' -f2-)
        if [ -n "$RECOMMENDATION" ]; then
            echo -e "  ${YELLOW}Note:${RECOMMENDATION}${NC}"
        fi
    else
        echo -e "  Status: ${YELLOW}⚠ No buffer info in log${NC}"
    fi
else
    echo -e "  Status: ${YELLOW}⚠ Log file not found${NC}"
fi
echo ""

# Test 7: Frame Statistics
echo -e "${BOLD}[7/8] UDP Frame Statistics${NC}"
if [ -f "$LOG_FILE" ]; then
    SEND_COUNT=$(grep 'PROFILE-UDP-SEND' "$LOG_FILE" | tail -1 | grep -oP 'Seq=\K\d+')
    RECV_COUNT=$(grep 'PROFILE-UDP-RECV' "$LOG_FILE" | tail -1 | grep -oP 'Seq=\K\d+')

    if [ -n "$SEND_COUNT" ]; then
        SEND_FORMATTED=$(printf "%'d" $SEND_COUNT)
        echo -e "  Frames Sent: ${GREEN}${SEND_FORMATTED}${NC}"
    fi

    if [ -n "$RECV_COUNT" ]; then
        RECV_FORMATTED=$(printf "%'d" $RECV_COUNT)
        echo -e "  Frames Received: ${GREEN}${RECV_FORMATTED}${NC}"
    fi

    # Check for packet loss
    LOSS_DETECTED=$(grep 'Detected.*lost frames' "$LOG_FILE" | tail -5)
    if [ -n "$LOSS_DETECTED" ]; then
        TOTAL_LOST=$(echo "$LOSS_DETECTED" | grep -oP 'Detected \K\d+' | awk '{sum+=$1} END {print sum}')
        echo -e "  Packet Loss: ${YELLOW}${TOTAL_LOST} frames detected${NC}"
    else
        echo -e "  Packet Loss: ${GREEN}✓ None detected${NC}"
    fi

    # Recent performance
    RECENT_PERF=$(grep 'PROFILE-UDP-SEND' "$LOG_FILE" | tail -1)
    if [ -n "$RECENT_PERF" ]; then
        TOTAL_TIME=$(echo "$RECENT_PERF" | grep -oP 'Total=\K[0-9.]+[µm]s')
        FRAME_SIZE=$(echo "$RECENT_PERF" | grep -oP 'FrameSize=\K\d+')
        if [ -n "$TOTAL_TIME" ] && [ -n "$FRAME_SIZE" ]; then
            echo -e "  Recent Send: ${FRAME_SIZE} bytes in ${TOTAL_TIME}"
        fi
    fi
else
    echo -e "  Status: ${YELLOW}⚠ Log file not found${NC}"
fi
echo ""

# Test 8: Connection Duration
echo -e "${BOLD}[8/8] Connection Health${NC}"
if [ -f "$LOG_FILE" ]; then
    # Get first and last timestamp
    FIRST_TS=$(head -20 "$LOG_FILE" | grep -oP '^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}' | head -1)
    LAST_TS=$(tail -20 "$LOG_FILE" | grep -oP '^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}' | tail -1)

    if [ -n "$FIRST_TS" ] && [ -n "$LAST_TS" ]; then
        FIRST_EPOCH=$(date -d "$FIRST_TS" +%s 2>/dev/null || echo "0")
        LAST_EPOCH=$(date -d "$LAST_TS" +%s 2>/dev/null || echo "0")

        if [ "$FIRST_EPOCH" != "0" ] && [ "$LAST_EPOCH" != "0" ]; then
            DURATION=$((LAST_EPOCH - FIRST_EPOCH))
            HOURS=$((DURATION / 3600))
            MINUTES=$(((DURATION % 3600) / 60))

            echo -e "  Connection Age: ${GREEN}${HOURS}h ${MINUTES}m${NC}"
            echo -e "  Started: $FIRST_TS"
            echo -e "  Last Activity: $LAST_TS"
        fi
    fi

    # Check for recent errors
    ERRORS=$(grep -i 'error\|failed\|timeout' "$LOG_FILE" | tail -3)
    if [ -n "$ERRORS" ]; then
        echo -e "  Recent Issues: ${YELLOW}⚠ Found${NC}"
        echo "$ERRORS" | while read line; do
            SHORT=$(echo "$line" | cut -c1-80)
            echo -e "    ${YELLOW}${SHORT}...${NC}"
        done
    else
        echo -e "  Health: ${GREEN}✓ No recent errors${NC}"
    fi
fi
echo ""

# Summary
echo -e "${BOLD}========================================${NC}"
echo -e "${BOLD}  Summary${NC}"
echo -e "${BOLD}========================================${NC}"

PASS_COUNT=0
WARN_COUNT=0
FAIL_COUNT=0

# Count results
if pgrep -f shadowmesh-l3-v11-rtt-fixed > /dev/null; then ((PASS_COUNT++)); else ((FAIL_COUNT++)); fi
if ip link show $TUN_DEVICE &> /dev/null; then ((PASS_COUNT++)); else ((FAIL_COUNT++)); fi
if timeout 5 ping -c 1 -W 2 $REMOTE_TUNNEL_IP &> /dev/null; then ((PASS_COUNT++)); else ((WARN_COUNT++)); fi

TOTAL_TESTS=8

echo -e "  Tests Passed: ${GREEN}${PASS_COUNT}${NC}"
if [ $WARN_COUNT -gt 0 ]; then
    echo -e "  Warnings: ${YELLOW}${WARN_COUNT}${NC}"
fi
if [ $FAIL_COUNT -gt 0 ]; then
    echo -e "  Tests Failed: ${RED}${FAIL_COUNT}${NC}"
fi

echo ""
if [ $FAIL_COUNT -eq 0 ] && [ $WARN_COUNT -eq 0 ]; then
    echo -e "${GREEN}${BOLD}✓ Tunnel is healthy and operational${NC}"
    exit 0
elif [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${YELLOW}${BOLD}⚠ Tunnel is operational with warnings${NC}"
    exit 0
else
    echo -e "${RED}${BOLD}✗ Tunnel has critical issues${NC}"
    exit 1
fi
