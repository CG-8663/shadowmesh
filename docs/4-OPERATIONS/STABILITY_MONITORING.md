# ShadowMesh Stability Monitoring Guide

## Overview

The stability monitoring script provides 24-hour continuous health checks for ShadowMesh tunnels with automated alerting and JSON logging for analysis.

## Features

- **Continuous Health Checks**: Runs automated checks every 60 seconds (configurable)
- **JSON Structured Logging**: Machine-parsable logs for integration with monitoring tools
- **Real-Time Alerts**: Automated alerts for packet loss, high RTT, process failures
- **Comprehensive Metrics**: Process status, TUN device, connectivity, UDP statistics, system resources
- **Summary Reports**: Detailed uptime statistics and performance summaries
- **Graceful Shutdown**: Press Ctrl+C anytime to generate summary report

## Quick Start

### Basic Usage (24 hours, default settings)

```bash
cd /Volumes/BACKUPDISK/webcode/shadowmesh
./scripts/stability-monitor.sh
```

### Custom Duration (e.g., 1 hour test)

```bash
MONITOR_DURATION_HOURS=1 ./scripts/stability-monitor.sh
```

### Custom Check Interval (e.g., every 30 seconds)

```bash
CHECK_INTERVAL_SECONDS=30 ./scripts/stability-monitor.sh
```

## Configuration

Configure via environment variables:

```bash
# Connection settings
export REMOTE_TUNNEL_IP="10.10.10.5"        # Philippines tunnel IP
export LOCAL_TUNNEL_IP="10.10.10.3"         # UK tunnel IP
export TUN_DEVICE="chr001"                  # TUN device name
export LOG_FILE="$HOME/shadowmesh/l3-v11-rtt-fixed.log"

# Monitoring settings
export MONITOR_DURATION_HOURS=24            # Total monitoring duration
export CHECK_INTERVAL_SECONDS=60            # Time between checks

# Alert thresholds
export ALERT_THRESHOLD_PACKET_LOSS=10       # Alert if packet loss >10%
export ALERT_THRESHOLD_RTT_MS=500           # Alert if RTT >500ms

# Output
export OUTPUT_JSON="/var/log/shadowmesh/stability-monitor.json"

# Run monitoring
./scripts/stability-monitor.sh
```

## Monitored Metrics

### 1. Process Status
- Process running (PID, uptime)
- CPU usage percentage
- Memory usage (MB)
- Process health

### 2. TUN Device Status
- Device state (UP, DOWN, UNKNOWN)
- IP address configuration
- MTU settings

### 3. Connectivity Tests
- ICMP ping to remote tunnel IP
- Packet loss percentage
- RTT measurements (min/avg/max)

### 4. UDP Statistics
- Frames sent/received counts
- Recent RTT samples from logs
- Frame loss detection
- Send/receive performance metrics

### 5. System Resources
- Disk usage percentage
- Available disk space
- Alerts for disk usage >90%

## JSON Log Format

Each health check generates a JSON log entry:

```json
{
  "timestamp": "2025-11-07T10:30:45.123Z",
  "level": "INFO",
  "message": "Health check completed",
  "component": "stability-monitor",
  "check_num": 42,
  "process_status": "up",
  "process_pid": 12345,
  "process_uptime": "5-12:34:56",
  "process_mem_mb": 45.2,
  "process_cpu_pct": 2.5,
  "tun_status": "UNKNOWN",
  "tun_ip": "10.10.10.3",
  "tun_mtu": 1500,
  "ping_status": "success",
  "packet_loss": 0,
  "avg_rtt_ms": 245.5,
  "min_rtt_ms": 240.2,
  "max_rtt_ms": 250.8,
  "udp_send_count": 1234567,
  "udp_recv_count": 1234560,
  "udp_latest_rtt": "245ms",
  "frames_lost": 0,
  "disk_usage_pct": 45,
  "disk_available": "120G",
  "check_duration_ms": 1250
}
```

## Alerts

### Error Alerts (Critical Issues)

Triggered for:
- ShadowMesh process not running
- TUN device not found
- Connectivity test complete failure
- High packet loss (>10% by default)

### Warning Alerts (Non-Critical Issues)

Triggered for:
- TUN device in unexpected state
- Partial packet loss (<10% but >0%)
- High RTT (>500ms by default)
- UDP frame loss detected
- Disk usage >90%

## Usage Examples

### 1-Hour Stability Test

```bash
# Quick 1-hour test with frequent checks (every 30 seconds)
MONITOR_DURATION_HOURS=1 \
CHECK_INTERVAL_SECONDS=30 \
./scripts/stability-monitor.sh
```

### 24-Hour Production Monitoring

```bash
# Full 24-hour monitoring with default 60-second interval
MONITOR_DURATION_HOURS=24 \
OUTPUT_JSON="/var/log/shadowmesh/stability-24h.json" \
./scripts/stability-monitor.sh
```

### High-Frequency Monitoring (Real-Time)

```bash
# Check every 10 seconds for 1 hour
MONITOR_DURATION_HOURS=1 \
CHECK_INTERVAL_SECONDS=10 \
./scripts/stability-monitor.sh
```

### Custom Alert Thresholds

```bash
# Strict thresholds for production
ALERT_THRESHOLD_PACKET_LOSS=5    # Alert at 5% loss
ALERT_THRESHOLD_RTT_MS=300       # Alert at 300ms RTT
MONITOR_DURATION_HOURS=24 \
./scripts/stability-monitor.sh
```

## Running as Background Service

### Using nohup

```bash
# Run in background and redirect output
nohup ./scripts/stability-monitor.sh > stability-monitor.out 2>&1 &

# View output
tail -f stability-monitor.out

# Check if running
ps aux | grep stability-monitor

# Stop monitoring
pkill -f stability-monitor.sh
```

### Using screen

```bash
# Start screen session
screen -S shadowmesh-monitor

# Run monitoring
./scripts/stability-monitor.sh

# Detach: Press Ctrl+A then D
# Reattach: screen -r shadowmesh-monitor
# Kill session: screen -X -S shadowmesh-monitor quit
```

### Using tmux

```bash
# Start tmux session
tmux new -s monitor

# Run monitoring
./scripts/stability-monitor.sh

# Detach: Press Ctrl+B then D
# Reattach: tmux attach -t monitor
# Kill session: tmux kill-session -t monitor
```

## Analyzing Results

### Summary Statistics

The monitoring script generates a summary report at the end:

```
========================================
  Stability Monitoring Summary
========================================

Duration: 24.00 hours (Target: 24h)
Total Checks: 1440
Successful: 1438
Failed: 2
Warnings: 5
Uptime: 99.86%

Average Packet Loss: 0.1%
Average RTT: 245.3ms
Total UDP Frames Lost: 12

JSON Log: /var/log/shadowmesh/stability-monitor.json
```

### Parsing JSON Logs with jq

```bash
# View all health checks
cat /var/log/shadowmesh/stability-monitor.json | jq '.'

# Filter by level
cat stability-monitor.json | jq 'select(.level == "ERROR")'

# Extract connectivity metrics
cat stability-monitor.json | jq '{timestamp, packet_loss, avg_rtt_ms}'

# Calculate average RTT across all checks
cat stability-monitor.json | jq -s '[.[].avg_rtt_ms] | add/length'

# Find maximum RTT spike
cat stability-monitor.json | jq -s 'max_by(.avg_rtt_ms) | {timestamp, avg_rtt_ms}'

# Count errors and warnings
echo "Errors: $(cat stability-monitor.json | jq 'select(.level == "ERROR")' | wc -l)"
echo "Warnings: $(cat stability-monitor.json | jq 'select(.level == "WARN")' | wc -l)"

# Check for packet loss events
cat stability-monitor.json | jq 'select(.packet_loss > 0) | {timestamp, packet_loss, avg_rtt_ms}'

# Plot uptime over time
cat stability-monitor.json | jq -r '[.timestamp, .ping_status] | @csv'
```

### Integration with Grafana

Use Grafana Loki to visualize metrics:

```yaml
# promtail.yml
scrape_configs:
  - job_name: shadowmesh-stability
    static_configs:
      - targets:
          - localhost
        labels:
          job: shadowmesh-stability
          __path__: /var/log/shadowmesh/stability-monitor.json
    pipeline_stages:
      - json:
          expressions:
            level: level
            message: message
            packet_loss: packet_loss
            avg_rtt_ms: avg_rtt_ms
            process_cpu_pct: process_cpu_pct
            process_mem_mb: process_mem_mb
```

Create Grafana dashboard with panels:
- Uptime percentage (gauge)
- Packet loss over time (graph)
- RTT distribution (histogram)
- CPU/Memory usage (graph)
- Error/Warning count (stat)

## Troubleshooting

### Script Won't Start

```bash
# Check if script is executable
ls -la scripts/stability-monitor.sh
# If not: chmod +x scripts/stability-monitor.sh

# Check dependencies
which ping jq bc awk grep

# Run with debug output
bash -x ./scripts/stability-monitor.sh
```

### No Connectivity Detected

```bash
# Verify tunnel is running
ps aux | grep shadowmesh

# Check TUN device
ip link show chr001
ip addr show chr001

# Manual ping test
ping -c 3 10.10.10.5

# Check log file exists
ls -la $HOME/shadowmesh/l3-v11-rtt-fixed.log
```

### High Packet Loss Alerts

Possible causes:
- Network congestion (Starlink satellite handover)
- Buffer overflow (increase adaptive buffer size)
- CPU saturation (check process_cpu_pct in logs)
- Kernel dropping packets (check dmesg for errors)

Mitigation:
```bash
# Check kernel logs
dmesg | grep -i "chr001\|tun\|dropped"

# Increase buffer size in ShadowMesh config
# Monitor with increased check frequency
CHECK_INTERVAL_SECONDS=10 ./scripts/stability-monitor.sh
```

### Disk Space Alerts

```bash
# Check current disk usage
df -h /var/log/shadowmesh

# Clean old log files
find /var/log/shadowmesh -name "*.json" -mtime +7 -delete

# Rotate logs manually
mv stability-monitor.json stability-monitor.json.1
```

## Production Deployment

### Systemd Service

Create `/etc/systemd/system/shadowmesh-monitor.service`:

```ini
[Unit]
Description=ShadowMesh Stability Monitor
After=network.target shadowmesh.service
Requires=shadowmesh.service

[Service]
Type=simple
User=shadowmesh
Group=shadowmesh
WorkingDirectory=/opt/shadowmesh
Environment="MONITOR_DURATION_HOURS=168"
Environment="CHECK_INTERVAL_SECONDS=60"
Environment="OUTPUT_JSON=/var/log/shadowmesh/stability-monitor.json"
ExecStart=/opt/shadowmesh/scripts/stability-monitor.sh
Restart=on-failure
RestartSec=30s

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable shadowmesh-monitor.service
sudo systemctl start shadowmesh-monitor.service

# View logs
sudo journalctl -u shadowmesh-monitor -f

# Check status
sudo systemctl status shadowmesh-monitor
```

### Alerting Integration

#### Email Alerts via sendmail

Modify the `send_alert()` function in the script:

```bash
send_alert() {
    local severity="$1"
    local message="$2"

    if [ "$severity" = "ERROR" ]; then
        echo "$message" | mail -s "ShadowMesh Alert: $severity" admin@example.com
    fi

    # ... existing logging code ...
}
```

#### Slack Webhooks

```bash
send_alert() {
    local severity="$1"
    local message="$2"
    local webhook="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"

    if [ "$severity" = "ERROR" ]; then
        curl -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"[ShadowMesh] $message\"}" \
            "$webhook"
    fi

    # ... existing logging code ...
}
```

#### PagerDuty Integration

```bash
send_alert() {
    local severity="$1"
    local message="$2"
    local pagerduty_key="YOUR_INTEGRATION_KEY"

    if [ "$severity" = "ERROR" ]; then
        curl -X POST https://events.pagerduty.com/v2/enqueue \
            -H 'Content-Type: application/json' \
            -d "{
                \"routing_key\": \"$pagerduty_key\",
                \"event_action\": \"trigger\",
                \"payload\": {
                    \"summary\": \"$message\",
                    \"severity\": \"error\",
                    \"source\": \"shadowmesh-monitor\"
                }
            }"
    fi

    # ... existing logging code ...
}
```

## Performance Impact

The monitoring script has minimal impact on system performance:

- **CPU Usage**: <0.1% average (ping and shell commands only)
- **Memory**: ~2-5MB (bash process + temporary buffers)
- **Disk I/O**: ~1KB per check (JSON log entry)
- **Network**: ~500 bytes per check (5 ICMP packets)

For 60-second check interval over 24 hours:
- Total checks: 1,440
- Total log size: ~1.4MB
- Total network overhead: ~720KB

## Best Practices

1. **Set Appropriate Thresholds**: Adjust alert thresholds based on network conditions
   - Starlink networks: Use higher RTT thresholds (500-800ms)
   - Terrestrial networks: Use lower thresholds (100-300ms)

2. **Monitor Disk Space**: Ensure log directory has sufficient space
   - 24h monitoring: ~1.5MB per day
   - Weekly monitoring: ~10MB per week

3. **Archive Old Logs**: Implement log retention policy
   ```bash
   # Keep last 30 days, delete older
   find /var/log/shadowmesh -name "*.json" -mtime +30 -delete
   ```

4. **Review Summary Reports**: Check uptime percentage and failure patterns
   - >99.9% uptime: Excellent
   - 99-99.9% uptime: Good
   - <99% uptime: Investigate issues

5. **Correlate with External Events**: Compare downtime with:
   - Starlink satellite handovers
   - Network maintenance windows
   - System updates/reboots

## Validation Testing

Before production deployment, validate the monitoring script:

```bash
# Test 1: 5-minute quick test
MONITOR_DURATION_HOURS=0.083 CHECK_INTERVAL_SECONDS=10 ./scripts/stability-monitor.sh

# Test 2: Verify JSON output
cat /var/log/shadowmesh/stability-monitor.json | jq empty
# Should return nothing (valid JSON)

# Test 3: Trigger alerts manually
# Stop ShadowMesh process temporarily
sudo pkill shadowmesh
# Monitor should detect and alert within check interval
# Restart process: sudo systemctl start shadowmesh

# Test 4: Check graceful shutdown
# Run monitoring in foreground, press Ctrl+C
# Verify summary report is generated

# Test 5: Verify disk space alerts
# Temporarily set low threshold
ALERT_THRESHOLD_DISK_USAGE=10 ./scripts/stability-monitor.sh
```

## Related Documentation

- [Logging Guide](LOGGING_GUIDE.md) - Structured logging infrastructure
- [Performance Testing](PERFORMANCE_TESTING.md) - Throughput and latency benchmarks
- [Deployment Guide](DEPLOYMENT.md) - Production deployment procedures
- [Troubleshooting Guide](TROUBLESHOOTING.md) - Common issues and solutions
