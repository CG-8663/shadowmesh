#!/bin/bash
# ShadowMesh 24-Hour Stability Monitoring Script
# Continuous health checks with JSON logging and automated alerts

set -e

# Configuration
REMOTE_TUNNEL_IP="${REMOTE_TUNNEL_IP:-10.10.10.5}"  # Philippines tunnel IP
LOCAL_TUNNEL_IP="${LOCAL_TUNNEL_IP:-10.10.10.3}"    # UK tunnel IP
TUN_DEVICE="${TUN_DEVICE:-chr001}"
LOG_FILE="${LOG_FILE:-$HOME/shadowmesh/l3-v11-rtt-fixed.log}"
MONITOR_DURATION_HOURS="${MONITOR_DURATION_HOURS:-24}"
CHECK_INTERVAL_SECONDS="${CHECK_INTERVAL_SECONDS:-60}"
ALERT_THRESHOLD_PACKET_LOSS="${ALERT_THRESHOLD_PACKET_LOSS:-10}"  # Alert if >10% loss
ALERT_THRESHOLD_RTT_MS="${ALERT_THRESHOLD_RTT_MS:-500}"           # Alert if RTT >500ms
OUTPUT_JSON="${OUTPUT_JSON:-/var/log/shadowmesh/stability-monitor.json}"

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Create output directory
mkdir -p "$(dirname "$OUTPUT_JSON")"

# Initialize counters
TOTAL_CHECKS=0
SUCCESSFUL_CHECKS=0
FAILED_CHECKS=0
WARNINGS=0
START_TIME=$(date +%s)
END_TIME=$((START_TIME + MONITOR_DURATION_HOURS * 3600))

# JSON logging function
log_json() {
    local timestamp=$(date -u '+%Y-%m-%dT%H:%M:%S.%3NZ')
    local level="$1"
    local message="$2"
    shift 2
    local fields="$*"

    echo "{\"timestamp\":\"$timestamp\",\"level\":\"$level\",\"message\":\"$message\",\"component\":\"stability-monitor\",$fields}" >> "$OUTPUT_JSON"
}

# Alert function
send_alert() {
    local severity="$1"
    local message="$2"
    local details="$3"

    log_json "$severity" "$message" "$details"

    if [ "$severity" = "ERROR" ]; then
        echo -e "${RED}[ALERT] $message${NC}" >&2
        ((FAILED_CHECKS++))
    elif [ "$severity" = "WARN" ]; then
        echo -e "${YELLOW}[WARNING] $message${NC}" >&2
        ((WARNINGS++))
    fi
}

# Health check function
perform_health_check() {
    local check_num=$1
    local timestamp=$(date -u '+%Y-%m-%dT%H:%M:%S.%3NZ')
    local check_start=$(date +%s%3N)

    echo -e "${BOLD}[Check $check_num/$((MONITOR_DURATION_HOURS * 3600 / CHECK_INTERVAL_SECONDS))] $(date '+%Y-%m-%d %H:%M:%S')${NC}"

    # Check 1: Process Status
    local process_status="down"
    local process_pid=""
    local process_uptime=""
    local process_mem_mb=0
    local process_cpu_pct=0

    if pgrep -f shadowmesh-l3-v11-rtt-fixed > /dev/null; then
        process_status="up"
        process_pid=$(pgrep -f shadowmesh-l3-v11-rtt-fixed)
        process_uptime=$(ps -p "$process_pid" -o etime= | tr -d ' ')
        process_mem_mb=$(ps -p "$process_pid" -o rss= | awk '{printf "%.1f", $1/1024}')
        process_cpu_pct=$(ps -p "$process_pid" -o %cpu= | tr -d ' ')
        echo -e "  Process: ${GREEN}✓ Running${NC} (PID: $process_pid, Uptime: $process_uptime)"
    else
        send_alert "ERROR" "ShadowMesh process not running" "\"check_num\":$check_num"
        echo -e "  Process: ${RED}✗ Not Running${NC}"
        return 1
    fi

    # Check 2: TUN Device Status
    local tun_status="down"
    local tun_ip=""
    local tun_mtu=0

    if ip link show "$TUN_DEVICE" &> /dev/null; then
        tun_status=$(ip link show "$TUN_DEVICE" | grep -oP 'state \K\w+' || echo "UNKNOWN")
        tun_ip=$(ip addr show "$TUN_DEVICE" | grep -oP 'inet \K[\d.]+' || echo "")
        tun_mtu=$(ip link show "$TUN_DEVICE" | grep -oP 'mtu \K\d+' || echo "0")

        if [ "$tun_status" = "UNKNOWN" ] || [ "$tun_status" = "UP" ]; then
            echo -e "  TUN Device: ${GREEN}✓ $TUN_DEVICE${NC} ($tun_ip)"
        else
            send_alert "WARN" "TUN device in unexpected state: $tun_status" "\"check_num\":$check_num,\"tun_status\":\"$tun_status\""
            echo -e "  TUN Device: ${YELLOW}⚠ $TUN_DEVICE (state: $tun_status)${NC}"
        fi
    else
        send_alert "ERROR" "TUN device not found: $TUN_DEVICE" "\"check_num\":$check_num"
        echo -e "  TUN Device: ${RED}✗ $TUN_DEVICE not found${NC}"
        return 1
    fi

    # Check 3: Connectivity Test
    local ping_status="failed"
    local packet_loss=100
    local avg_rtt_ms=0
    local min_rtt_ms=0
    local max_rtt_ms=0

    if timeout 5 ping -c 5 -W 2 "$REMOTE_TUNNEL_IP" &> /tmp/ping_test_$$; then
        packet_loss=$(grep 'packet loss' /tmp/ping_test_$$ | grep -oP '\d+(?=% packet loss)' || echo "0")
        avg_rtt_ms=$(grep 'rtt min/avg/max' /tmp/ping_test_$$ | grep -oP 'avg/\K[\d.]+' || echo "0")
        min_rtt_ms=$(grep 'rtt min/avg/max' /tmp/ping_test_$$ | grep -oP 'min/\K[\d.]+' || echo "0")
        max_rtt_ms=$(grep 'rtt min/avg/max' /tmp/ping_test_$$ | grep -oP 'max/\K[\d.]+' || echo "0")

        if [ "$packet_loss" -eq 0 ]; then
            ping_status="success"
            echo -e "  Connectivity: ${GREEN}✓ Success${NC} (RTT: ${avg_rtt_ms}ms avg, ${min_rtt_ms}ms min, ${max_rtt_ms}ms max)"
        elif [ "$packet_loss" -lt "$ALERT_THRESHOLD_PACKET_LOSS" ]; then
            ping_status="partial"
            send_alert "WARN" "Partial packet loss detected: ${packet_loss}%" "\"check_num\":$check_num,\"packet_loss\":$packet_loss"
            echo -e "  Connectivity: ${YELLOW}⚠ Partial${NC} (Loss: ${packet_loss}%, RTT: ${avg_rtt_ms}ms)"
        else
            send_alert "ERROR" "High packet loss: ${packet_loss}%" "\"check_num\":$check_num,\"packet_loss\":$packet_loss"
            echo -e "  Connectivity: ${RED}✗ High Loss${NC} (Loss: ${packet_loss}%)"
        fi

        # Check RTT threshold
        if [ "$(echo "$avg_rtt_ms > $ALERT_THRESHOLD_RTT_MS" | bc)" -eq 1 ]; then
            send_alert "WARN" "High RTT detected: ${avg_rtt_ms}ms" "\"check_num\":$check_num,\"rtt_ms\":$avg_rtt_ms"
        fi
    else
        send_alert "ERROR" "Connectivity test failed - no response" "\"check_num\":$check_num"
        echo -e "  Connectivity: ${RED}✗ Failed${NC}"
    fi
    rm -f /tmp/ping_test_$$

    # Check 4: UDP Statistics from Logs
    local udp_send_count=0
    local udp_recv_count=0
    local udp_latest_rtt=""
    local frames_lost=0

    if [ -f "$LOG_FILE" ]; then
        udp_send_count=$(grep 'PROFILE-UDP-SEND' "$LOG_FILE" | tail -1 | grep -oP 'Seq=\K\d+' || echo "0")
        udp_recv_count=$(grep 'PROFILE-UDP-RECV' "$LOG_FILE" | tail -1 | grep -oP 'Seq=\K\d+' || echo "0")
        udp_latest_rtt=$(grep '\[RTT\]' "$LOG_FILE" | tail -1 | grep -oP 'avg \K[0-9.]+[µms]+' || echo "N/A")
        frames_lost=$(grep 'Detected.*lost frames' "$LOG_FILE" | tail -10 | grep -oP 'Detected \K\d+' | awk '{sum+=$1} END {print sum}' || echo "0")

        echo -e "  UDP Stats: Sent: $(printf "%'d" $udp_send_count), Recv: $(printf "%'d" $udp_recv_count), RTT: $udp_latest_rtt"

        if [ "$frames_lost" -gt 0 ]; then
            send_alert "WARN" "UDP frame loss detected in last 10 samples" "\"check_num\":$check_num,\"frames_lost\":$frames_lost"
        fi
    else
        echo -e "  UDP Stats: ${YELLOW}⚠ Log file not found${NC}"
    fi

    # Check 5: System Resources
    local disk_usage_pct=$(df -h "$(dirname "$LOG_FILE")" | awk 'NR==2 {print $5}' | tr -d '%')
    local disk_available=$(df -h "$(dirname "$LOG_FILE")" | awk 'NR==2 {print $4}')

    echo -e "  Resources: CPU: ${process_cpu_pct}%, Memory: ${process_mem_mb}MB, Disk: ${disk_usage_pct}% (${disk_available} free)"

    if [ "$disk_usage_pct" -gt 90 ]; then
        send_alert "WARN" "Disk usage critical: ${disk_usage_pct}%" "\"check_num\":$check_num,\"disk_usage_pct\":$disk_usage_pct"
    fi

    # Calculate check duration
    local check_end=$(date +%s%3N)
    local check_duration_ms=$((check_end - check_start))

    # Log complete health check result as JSON
    log_json "INFO" "Health check completed" \
        "\"check_num\":$check_num,\"process_status\":\"$process_status\",\"process_pid\":$process_pid,\"process_uptime\":\"$process_uptime\",\"process_mem_mb\":$process_mem_mb,\"process_cpu_pct\":$process_cpu_pct,\"tun_status\":\"$tun_status\",\"tun_ip\":\"$tun_ip\",\"tun_mtu\":$tun_mtu,\"ping_status\":\"$ping_status\",\"packet_loss\":$packet_loss,\"avg_rtt_ms\":$avg_rtt_ms,\"min_rtt_ms\":$min_rtt_ms,\"max_rtt_ms\":$max_rtt_ms,\"udp_send_count\":$udp_send_count,\"udp_recv_count\":$udp_recv_count,\"udp_latest_rtt\":\"$udp_latest_rtt\",\"frames_lost\":$frames_lost,\"disk_usage_pct\":$disk_usage_pct,\"disk_available\":\"$disk_available\",\"check_duration_ms\":$check_duration_ms"

    ((SUCCESSFUL_CHECKS++))
    echo ""
}

# Signal handler for graceful shutdown
cleanup() {
    echo ""
    echo -e "${BOLD}========================================${NC}"
    echo -e "${BOLD}  Monitoring Interrupted${NC}"
    echo -e "${BOLD}========================================${NC}"
    generate_summary
    exit 0
}

trap cleanup SIGINT SIGTERM

# Generate summary report
generate_summary() {
    local end_time=$(date +%s)
    local duration_seconds=$((end_time - START_TIME))
    local duration_hours=$(echo "scale=2; $duration_seconds / 3600" | bc)
    local uptime_pct=0

    if [ "$TOTAL_CHECKS" -gt 0 ]; then
        uptime_pct=$(echo "scale=2; ($SUCCESSFUL_CHECKS * 100) / $TOTAL_CHECKS" | bc)
    fi

    echo -e "${BOLD}========================================${NC}"
    echo -e "${BOLD}  Stability Monitoring Summary${NC}"
    echo -e "${BOLD}========================================${NC}"
    echo ""
    echo -e "Duration: ${BOLD}${duration_hours} hours${NC} (Target: ${MONITOR_DURATION_HOURS}h)"
    echo -e "Total Checks: ${BOLD}${TOTAL_CHECKS}${NC}"
    echo -e "Successful: ${GREEN}${SUCCESSFUL_CHECKS}${NC}"
    echo -e "Failed: ${RED}${FAILED_CHECKS}${NC}"
    echo -e "Warnings: ${YELLOW}${WARNINGS}${NC}"
    echo -e "Uptime: ${BOLD}${uptime_pct}%${NC}"
    echo ""

    # Parse JSON log for statistics
    if [ -f "$OUTPUT_JSON" ]; then
        local avg_packet_loss=$(grep '"packet_loss"' "$OUTPUT_JSON" | grep -oP '"packet_loss":\K\d+' | awk '{sum+=$1; count++} END {if(count>0) printf "%.1f", sum/count; else print "0"}')
        local avg_rtt=$(grep '"avg_rtt_ms"' "$OUTPUT_JSON" | grep -oP '"avg_rtt_ms":\K[\d.]+' | awk '{sum+=$1; count++} END {if(count>0) printf "%.1f", sum/count; else print "0"}')
        local total_frames_lost=$(grep '"frames_lost"' "$OUTPUT_JSON" | grep -oP '"frames_lost":\K\d+' | awk '{sum+=$1} END {print sum}')

        echo -e "Average Packet Loss: ${BOLD}${avg_packet_loss}%${NC}"
        echo -e "Average RTT: ${BOLD}${avg_rtt}ms${NC}"
        echo -e "Total UDP Frames Lost: ${BOLD}${total_frames_lost}${NC}"
        echo ""
        echo -e "JSON Log: ${BLUE}${OUTPUT_JSON}${NC}"
    fi

    # Log summary as JSON
    log_json "INFO" "Stability monitoring completed" \
        "\"duration_hours\":$duration_hours,\"total_checks\":$TOTAL_CHECKS,\"successful_checks\":$SUCCESSFUL_CHECKS,\"failed_checks\":$FAILED_CHECKS,\"warnings\":$WARNINGS,\"uptime_pct\":$uptime_pct"

    # Overall status
    if [ "$FAILED_CHECKS" -eq 0 ] && [ "$WARNINGS" -eq 0 ]; then
        echo -e "${GREEN}${BOLD}✓ Tunnel stability: EXCELLENT${NC}"
        exit 0
    elif [ "$FAILED_CHECKS" -eq 0 ]; then
        echo -e "${YELLOW}${BOLD}⚠ Tunnel stability: GOOD (with warnings)${NC}"
        exit 0
    elif [ "$uptime_pct" = "$(echo "$uptime_pct >= 95" | bc)" ]; then
        echo -e "${YELLOW}${BOLD}⚠ Tunnel stability: ACCEPTABLE${NC}"
        exit 1
    else
        echo -e "${RED}${BOLD}✗ Tunnel stability: POOR${NC}"
        exit 1
    fi
}

# Main monitoring loop
echo -e "${BOLD}========================================${NC}"
echo -e "${BOLD}  ShadowMesh Stability Monitor${NC}"
echo -e "${BOLD}========================================${NC}"
echo ""
echo -e "Start Time: $(date '+%Y-%m-%d %H:%M:%S %Z')"
echo -e "Duration: ${MONITOR_DURATION_HOURS} hours"
echo -e "Check Interval: ${CHECK_INTERVAL_SECONDS} seconds"
echo -e "Remote Tunnel IP: ${REMOTE_TUNNEL_IP}"
echo -e "Local Tunnel IP: ${LOCAL_TUNNEL_IP}"
echo -e "TUN Device: ${TUN_DEVICE}"
echo -e "JSON Log: ${OUTPUT_JSON}"
echo ""
echo -e "Alerts triggered for:"
echo -e "  - Packet loss > ${ALERT_THRESHOLD_PACKET_LOSS}%"
echo -e "  - RTT > ${ALERT_THRESHOLD_RTT_MS}ms"
echo ""
echo -e "Press Ctrl+C to stop monitoring and generate summary"
echo ""

# Log monitoring start
log_json "INFO" "Stability monitoring started" \
    "\"duration_hours\":$MONITOR_DURATION_HOURS,\"check_interval_seconds\":$CHECK_INTERVAL_SECONDS,\"remote_tunnel_ip\":\"$REMOTE_TUNNEL_IP\",\"local_tunnel_ip\":\"$LOCAL_TUNNEL_IP\",\"tun_device\":\"$TUN_DEVICE\""

# Run monitoring loop
while [ "$(date +%s)" -lt "$END_TIME" ]; do
    ((TOTAL_CHECKS++))

    if ! perform_health_check "$TOTAL_CHECKS"; then
        # Health check failed - log but continue monitoring
        echo -e "${RED}Health check failed - continuing monitoring${NC}"
    fi

    # Sleep until next check
    sleep "$CHECK_INTERVAL_SECONDS"
done

# Generate final summary
generate_summary
