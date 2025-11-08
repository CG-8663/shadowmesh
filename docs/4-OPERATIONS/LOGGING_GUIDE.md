# ShadowMesh Structured Logging Guide

## Overview

ShadowMesh uses structured JSON logging for production-ready logging with built-in log rotation. Logs are written in JSON format for easy parsing, monitoring, and analysis.

## Features

- **Structured JSON Output**: All logs are JSON formatted for easy parsing
- **Log Levels**: DEBUG, INFO, WARN, ERROR, FATAL
- **Automatic Log Rotation**: Rotates logs when they reach configurable size
- **Contextual Fields**: Add global and per-log fields for rich context
- **Stack Traces**: Automatic stack traces for ERROR and FATAL levels
- **Thread-Safe**: Concurrent logging from multiple goroutines
- **Zero External Dependencies**: Built on Go standard library

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/shadowmesh/shadowmesh/pkg/logging"
)

func main() {
    // Initialize logger
    logger, err := logging.NewLogger("shadowmesh", logging.INFO, "/var/log/shadowmesh/app.log")
    if err != nil {
        log.Fatal(err)
    }
    defer logger.Close()

    // Simple logging
    logger.Info("Application started")
    logger.Warn("Configuration file not found, using defaults")
    logger.Error("Failed to connect to peer")
}
```

### With Contextual Fields

```go
// Log with additional context
logger.Info("Connection established", logging.Fields{
    "peer_id": "8c53bab8368ba57586d1f69fa1f750d597aa04ed",
    "ip": "100.126.75.74",
    "port": 8443,
    "duration_ms": 250,
})

// Add global fields (e.g., node identity)
logger.WithFields(logging.Fields{
    "node_id": "shadowmesh-node-01",
    "version": "v11",
    "region": "uk-london",
})

logger.Info("Tunnel established")
// Output includes node_id, version, region in every log
```

### Log Levels

```go
logger.Debug("Detailed debugging information")         // Only shown in DEBUG level
logger.Info("Informational message")                   // Normal operations
logger.Warn("Warning: retry attempt 3/5")             // Potential issues
logger.Error("Error processing packet")                // Errors (with stack trace)
logger.Fatal("Critical failure, shutting down")        // Exits program (with stack trace)
```

### Formatted Logging

```go
logger.Infof("Received %d bytes from peer %s", 1024, peerID)
logger.Errorf("Connection timeout after %d attempts", retryCount)
```

## Configuration

### Log Rotation

```go
logger := logging.NewLogger("shadowmesh", logging.INFO, "/var/log/shadowmesh/app.log")

// Configure rotation (default: 100MB, 10 backups)
logger.SetMaxFileSize(50 * 1024 * 1024) // 50MB
logger.SetMaxBackups(5)                  // Keep 5 backup files

// When log reaches 50MB, it will rotate:
// app.log          -> Current log
// app.log.1        -> Previous log (most recent backup)
// app.log.2        -> Older backup
// ...
// app.log.5        -> Oldest backup (deleted when new rotation occurs)
```

### Log Levels

```go
logger.SetLevel(logging.DEBUG)  // Show all logs
logger.SetLevel(logging.INFO)   // Show INFO, WARN, ERROR, FATAL
logger.SetLevel(logging.WARN)   // Show WARN, ERROR, FATAL
logger.SetLevel(logging.ERROR)  // Show ERROR, FATAL only
```

## JSON Log Format

All logs are written as single-line JSON objects:

```json
{
  "timestamp": "2025-11-07T10:30:45.123456Z",
  "level": "INFO",
  "message": "UDP tunnel established",
  "component": "p2p",
  "caller": "udp_connection.go:197",
  "fields": {
    "peer_id": "8c53bab8368ba57586d1f69fa1f750d597aa04ed",
    "remote_ip": "100.126.75.74",
    "remote_port": 9444,
    "rtt_ms": 245
  }
}
```

### Fields Explanation

- `timestamp`: ISO 8601 UTC timestamp with nanosecond precision
- `level`: Log level (DEBUG, INFO, WARN, ERROR, FATAL)
- `message`: Human-readable log message
- `component`: Logger component name (e.g., "p2p", "tun", "udp")
- `caller`: Source file and line number
- `fields`: Additional contextual data
- `stack_trace`: Stack trace (ERROR and FATAL only)

## Integration with ShadowMesh

### Replace Standard log Package

**Before:**
```go
import "log"

log.Println("Starting ShadowMesh...")
log.Printf("Connected to peer %s", peerID)
```

**After:**
```go
import "github.com/shadowmesh/shadowmesh/pkg/logging"

logger := logging.GetDefaultLogger()
logger.Info("Starting ShadowMesh...")
logger.Infof("Connected to peer %s", peerID)
```

### Component-Specific Loggers

```go
// Create component-specific loggers
p2pLogger, _ := logging.NewLogger("p2p", logging.INFO, "/var/log/shadowmesh/p2p.log")
tunLogger, _ := logging.NewLogger("tun", logging.DEBUG, "/var/log/shadowmesh/tun.log")
udpLogger, _ := logging.NewLogger("udp", logging.INFO, "/var/log/shadowmesh/udp.log")

// Each logger writes to separate files with component name in logs
p2pLogger.Info("P2P connection established")
tunLogger.Debug("TUN packet received", logging.Fields{"size": 1500})
udpLogger.Info("UDP frame sent", logging.Fields{"seq": 12345})
```

## Production Deployment

### Systemd Service Logging

```bash
# Run shadowmesh with logging to file
sudo ./shadowmesh-l3-v11-rtt-fixed-amd64 \
  -keydir ./keys \
  -backbone http://209.151.148.121:8080 \
  -ip 100.115.193.115 \
  -port 8443 \
  -udp-port 9443 \
  -tun chr001 \
  -tun-ip 10.10.10.3 \
  -tun-netmask 24 \
  -connect 8c53bab8368ba57586d1f69fa1f750d597aa04ed \
  -log-level INFO \
  -log-file /var/log/shadowmesh/tunnel.log
```

### Log Directory Structure

```
/var/log/shadowmesh/
├── tunnel.log           # Current log
├── tunnel.log.1         # Last rotation
├── tunnel.log.2         # Previous rotation
├── tunnel.log.3
├── tunnel.log.4
└── tunnel.log.5
```

### Log Rotation Policy

```bash
# Create log directory
sudo mkdir -p /var/log/shadowmesh
sudo chown shadowmesh:shadowmesh /var/log/shadowmesh

# Automatic rotation happens when:
# - Log file reaches max size (default: 100MB)
# - Keeps last 10 backups (configurable)
# - Old backups are automatically deleted
```

## Monitoring and Analysis

### Parse JSON Logs with jq

```bash
# View logs in human-readable format
cat /var/log/shadowmesh/tunnel.log | jq '.'

# Filter by log level
cat tunnel.log | jq 'select(.level == "ERROR")'

# Extract specific fields
cat tunnel.log | jq '{timestamp, message, fields}'

# Count errors by component
cat tunnel.log | jq 'select(.level == "ERROR") | .component' | sort | uniq -c

# Find high RTT events
cat tunnel.log | jq 'select(.fields.rtt_ms > 500)'

# Real-time monitoring
tail -f tunnel.log | jq '.'
```

### Centralized Logging (ELK Stack)

```bash
# Filebeat configuration for ShadowMesh logs
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/shadowmesh/*.log
  json.keys_under_root: true
  json.add_error_key: true

output.elasticsearch:
  hosts: ["localhost:9200"]
  index: "shadowmesh-%{+yyyy.MM.dd}"
```

### Grafana Loki Integration

```yaml
# promtail.yml
scrape_configs:
  - job_name: shadowmesh
    static_configs:
      - targets:
          - localhost
        labels:
          job: shadowmesh
          __path__: /var/log/shadowmesh/*.log
    pipeline_stages:
      - json:
          expressions:
            level: level
            message: message
            component: component
```

## Performance Considerations

- **Minimal Overhead**: JSON marshaling adds <1µs per log entry
- **Async Writes**: Logs are written to disk asynchronously
- **Buffer Size**: Default 8KB buffer reduces syscalls
- **Rotation**: No downtime during log rotation

## Best Practices

### 1. Use Appropriate Log Levels

```go
// DEBUG: Detailed information for debugging (disabled in production)
logger.Debug("Packet header parsed", logging.Fields{"header": header})

// INFO: Important state changes and operational events
logger.Info("Connection established", logging.Fields{"peer_id": peerID})

// WARN: Potential issues that don't affect functionality
logger.Warn("Retry attempt", logging.Fields{"attempt": 3, "max": 5})

// ERROR: Errors that affect functionality (includes stack trace)
logger.Error("Failed to send packet", logging.Fields{"error": err.Error()})

// FATAL: Critical errors requiring shutdown (includes stack trace, exits program)
logger.Fatal("Database connection lost")
```

### 2. Add Contextual Fields

```go
// Bad
logger.Info("Packet received")

// Good
logger.Info("Packet received", logging.Fields{
    "peer_id": peerID,
    "size": len(packet),
    "seq": sequenceNum,
})
```

### 3. Use Global Fields for Identity

```go
logger.WithFields(logging.Fields{
    "node_id": nodeID,
    "peer_id": myPeerID,
    "version": version,
})
// All subsequent logs will include these fields
```

### 4. Structured Errors

```go
// Bad
logger.Error(fmt.Sprintf("Failed to connect: %v", err))

// Good
logger.Error("Failed to connect", logging.Fields{
    "error": err.Error(),
    "peer_id": peerID,
    "ip": remoteIP,
    "port": remotePort,
})
```

## Troubleshooting

### Logs Not Appearing

```bash
# Check log file permissions
ls -la /var/log/shadowmesh/

# Check if process has write permissions
sudo -u shadowmesh touch /var/log/shadowmesh/test.log

# Check disk space
df -h /var/log
```

### High Log Volume

```go
// Increase log level in production
logger.SetLevel(logging.INFO)  // Disable DEBUG logs

// Reduce logging frequency
if count%100 == 0 {
    logger.Info("Progress", logging.Fields{"count": count})
}
```

### Log Rotation Not Working

```bash
# Check file size
ls -lh /var/log/shadowmesh/tunnel.log

# Verify rotation settings in code
logger.SetMaxFileSize(100 * 1024 * 1024)  // 100MB
logger.SetMaxBackups(10)
```

## Migration from Standard log Package

```go
// Step 1: Initialize structured logger in main.go
import "github.com/shadowmesh/shadowmesh/pkg/logging"

func main() {
    // Initialize global logger
    logging.InitDefaultLogger("shadowmesh", logging.INFO, "/var/log/shadowmesh/app.log")
    defer logging.GetDefaultLogger().Close()

    // Your code...
}

// Step 2: Replace log.Printf with logging calls
// Old: log.Printf("Connection established to %s", peerID)
// New: logging.Infof("Connection established to %s", peerID)

// Step 3: Add contextual fields where useful
logging.Info("Connection established", logging.Fields{"peer_id": peerID})
```

## Example: Complete Main Function

```go
package main

import (
    "flag"
    "github.com/shadowmesh/shadowmesh/pkg/logging"
)

func main() {
    logFile := flag.String("log-file", "/var/log/shadowmesh/tunnel.log", "Log file path")
    logLevel := flag.String("log-level", "INFO", "Log level (DEBUG, INFO, WARN, ERROR)")
    flag.Parse()

    // Parse log level
    level := logging.INFO
    switch *logLevel {
    case "DEBUG":
        level = logging.DEBUG
    case "WARN":
        level = logging.WARN
    case "ERROR":
        level = logging.ERROR
    }

    // Initialize logger
    logger, err := logging.NewLogger("shadowmesh", level, *logFile)
    if err != nil {
        panic(err)
    }
    defer logger.Close()

    // Configure rotation
    logger.SetMaxFileSize(100 * 1024 * 1024) // 100MB
    logger.SetMaxBackups(10)

    // Add global fields
    logger.WithFields(logging.Fields{
        "version": "v11",
        "build": "20251107",
    })

    // Application code
    logger.Info("ShadowMesh started")

    // ... rest of application ...
}
```

## Additional Resources

- [Go Standard Library log package](https://pkg.go.dev/log)
- [JSON Log Format Best Practices](https://www.honeycomb.io/blog/structured-logging-and-your-team)
- [Grafana Loki Documentation](https://grafana.com/docs/loki/latest/)
- [ELK Stack for Log Analysis](https://www.elastic.co/what-is/elk-stack)
