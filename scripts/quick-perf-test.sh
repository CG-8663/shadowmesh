#!/bin/bash
#
# ShadowMesh Quick Performance Test
# Run this from Proxmox (Philippines) to test Belgium Raspberry Pi
#

set -e

BELGIUM_IP="10.10.10.4"
TEST_NAME="Belgium-Philippines-$(date +%Y%m%d-%H%M%S)"

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║       ShadowMesh Quick Performance Test                   ║"
echo "║       Philippines (Starlink) → Belgium                   ║"
echo "║       Distance: ~15,000 km                                ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Create results directory
mkdir -p ~/shadowmesh-perf-results
cd ~/shadowmesh-perf-results
mkdir -p "$TEST_NAME"
cd "$TEST_NAME"

echo "Test: $TEST_NAME"
echo "Target: $BELGIUM_IP (Belgium Raspberry Pi)"
echo "Date: $(date)"
echo ""

# Test 1: Quick ping (30 packets)
echo "=== Test 1: Ping Test (30 packets) ==="
echo "Testing connectivity and latency..."
ping -c 30 -i 0.5 "$BELGIUM_IP" | tee ping-30.txt
echo ""

# Extract statistics
RTT_STATS=$(grep "rtt min/avg/max" ping-30.txt | cut -d'=' -f2)
echo "RTT Statistics: $RTT_STATS"
echo ""

# Test 2: Extended ping (100 packets for better statistics)
echo "=== Test 2: Extended Ping (100 packets) ==="
ping -c 100 -i 0.2 "$BELGIUM_IP" | tee ping-100.txt
echo ""

# Test 3: Large packet test
echo "=== Test 3: Large Packet Test (MTU 1472) ==="
echo "Testing with maximum non-fragmented packet size..."
ping -c 20 -s 1472 "$BELGIUM_IP" | tee ping-large.txt
echo ""

# Test 4: Check if iperf3 is available
if ! command -v iperf3 &> /dev/null; then
    echo "=== Installing iperf3 for throughput tests ==="
    echo "This requires sudo access..."
    sudo apt-get update
    sudo apt-get install -y iperf3
    echo ""
fi

# Test 5: TCP Throughput (if iperf3 server is running on Belgium)
echo "=== Test 5: TCP Throughput Test ==="
echo "Attempting to connect to iperf3 server on $BELGIUM_IP..."
echo "NOTE: Make sure iperf3 server is running on Belgium Pi:"
echo "      iperf3 -s -B $BELGIUM_IP"
echo ""
read -p "Is iperf3 server running on Belgium? (y/N): " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Running 30-second TCP throughput test..."
    iperf3 -c "$BELGIUM_IP" -t 30 -i 5 -J > tcp-throughput.json 2>&1

    if [ -f tcp-throughput.json ]; then
        THROUGHPUT=$(cat tcp-throughput.json | jq -r '.end.sum_sent.bits_per_second / 1000000' 2>/dev/null || echo "N/A")
        echo "Average Throughput: $THROUGHPUT Mbps"
    fi
    echo ""

    # Test 6: Parallel streams
    echo "=== Test 6: Parallel TCP Streams (4 streams) ==="
    iperf3 -c "$BELGIUM_IP" -t 30 -P 4 -i 5 -J > tcp-parallel.json 2>&1

    if [ -f tcp-parallel.json ]; then
        PARALLEL_THROUGHPUT=$(cat tcp-parallel.json | jq -r '.end.sum_sent.bits_per_second / 1000000' 2>/dev/null || echo "N/A")
        echo "Aggregate Throughput (4 streams): $PARALLEL_THROUGHPUT Mbps"
    fi
    echo ""
else
    echo "Skipping throughput tests. Run iperf3 server on Belgium first:"
    echo "  ssh user@$BELGIUM_IP"
    echo "  iperf3 -s -B $BELGIUM_IP"
    echo ""
fi

# Test 7: File transfer test (if SSH works)
echo "=== Test 7: SSH and File Transfer Test ==="
read -p "Test SSH file transfer? (y/N): " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    read -p "Enter SSH username for $BELGIUM_IP: " SSH_USER

    # Create test file
    echo "Creating 10MB test file..."
    dd if=/dev/zero of=test-10mb.bin bs=1M count=10 2>/dev/null

    echo "Transferring file to Belgium..."
    time scp test-10mb.bin ${SSH_USER}@${BELGIUM_IP}:/tmp/ 2>&1 | tee scp-transfer.txt

    echo "Cleaning up..."
    ssh ${SSH_USER}@${BELGIUM_IP} "rm -f /tmp/test-10mb.bin"
    rm -f test-10mb.bin
    echo ""
fi

# Generate summary report
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║              Generating Summary Report                     ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

cat > RESULTS.md <<EOF
# ShadowMesh Performance Test Results

**Test ID**: $TEST_NAME
**Date**: $(date)
**Route**: Philippines (Starlink) → Belgium (Raspberry Pi)
**Distance**: ~15,000 km
**Relay**: Frankfurt, Germany (83.136.252.52)

---

## Network Topology

\`\`\`
Proxmox VM (Philippines)  →  UpCloud Relay (Germany)  →  Raspberry Pi (Belgium)
North Luzon, Aparri          Frankfurt                   Europe
10.10.10.2                   83.136.252.52              10.10.10.4
Starlink Satellite           Datacenter                  Residential
\`\`\`

---

## Test Results

### Ping Test (30 packets)
\`\`\`
$(grep "packet" ping-30.txt | tail -1)
$(grep "rtt" ping-30.txt | tail -1)
\`\`\`

### Extended Ping (100 packets)
\`\`\`
$(grep "packet" ping-100.txt | tail -1)
$(grep "rtt" ping-100.txt | tail -1)
\`\`\`

### Large Packet Test (1472 bytes)
\`\`\`
$(grep "packet" ping-large.txt 2>/dev/null | tail -1 || echo "Not run")
$(grep "rtt" ping-large.txt 2>/dev/null | tail -1 || echo "Not run")
\`\`\`

### TCP Throughput
$(if [ -f tcp-throughput.json ]; then
    echo "- **Single Stream**: $(cat tcp-throughput.json | jq -r '.end.sum_sent.bits_per_second / 1000000' 2>/dev/null || echo 'N/A') Mbps"
else
    echo "- Not tested (iperf3 server not running)"
fi)

$(if [ -f tcp-parallel.json ]; then
    echo "- **Parallel (4 streams)**: $(cat tcp-parallel.json | jq -r '.end.sum_sent.bits_per_second / 1000000' 2>/dev/null || echo 'N/A') Mbps"
fi)

---

## Analysis

### Latency Analysis
$(awk '/rtt min\/avg\/max/ {
    split($4, rtt, "/");
    min=rtt[1]; avg=rtt[2]; max=rtt[3];
    split(max, maxval, " ");
    max=maxval[1];

    if (avg < 800) status="✅ Excellent";
    else if (avg < 1000) status="✅ Good";
    else if (avg < 2000) status="⚠️  Acceptable";
    else status="❌ Poor";

    print "- **Min RTT**: " min " ms";
    print "- **Avg RTT**: " avg " ms " status;
    print "- **Max RTT**: " max " ms";

    if (avg < 1000) {
        print "- **Assessment**: Excellent latency for 15,000 km over Starlink!";
    } else {
        print "- **Assessment**: Acceptable given satellite link and distance";
    }
}' ping-100.txt)

### Packet Loss
$(awk '/packet loss/ {
    split($6, loss, "%");
    if (loss[1] == 0) status="✅ Perfect";
    else if (loss[1] < 1) status="✅ Excellent";
    else if (loss[1] < 5) status="⚠️  Acceptable";
    else status="❌ Poor";

    print "- **Packet Loss**: " $6 " " status;
}' ping-100.txt)

### Throughput Analysis
$(if [ -f tcp-throughput.json ]; then
    MBPS=$(cat tcp-throughput.json | jq -r '.end.sum_sent.bits_per_second / 1000000' 2>/dev/null || echo 0)
    if awk -v mbps="$MBPS" 'BEGIN {exit !(mbps >= 20)}'; then
        echo "- **Throughput**: ✅ Excellent (>20 Mbps)"
    elif awk -v mbps="$MBPS" 'BEGIN {exit !(mbps >= 10)}'; then
        echo "- **Throughput**: ✅ Good (>10 Mbps)"
    elif awk -v mbps="$MBPS" 'BEGIN {exit !(mbps >= 5)}'; then
        echo "- **Throughput**: ⚠️  Acceptable (>5 Mbps)"
    else
        echo "- **Throughput**: ❌ Needs optimization (<5 Mbps)"
    fi
fi)

---

## Comparison to Expectations

| Metric | Expected (Starlink) | Measured | Status |
|--------|---------------------|----------|--------|
| **RTT** | 600-1000ms | $(awk '/rtt min\/avg\/max/ {split($4, rtt, "/"); print rtt[2] "ms"}' ping-100.txt) | $(awk '/rtt min\/avg\/max/ {split($4, rtt, "/"); if (rtt[2] < 1000) print "✅"; else print "⚠️ "}' ping-100.txt) |
| **Packet Loss** | <1% | $(grep "packet loss" ping-100.txt | awk '{print $6}') | $(awk '/packet loss/ {if ($6 == "0%") print "✅"; else print "⚠️ "}' ping-100.txt) |
| **Throughput** | 10-50 Mbps | $(if [ -f tcp-throughput.json ]; then cat tcp-throughput.json | jq -r '.end.sum_sent.bits_per_second / 1000000' 2>/dev/null; else echo "N/A"; fi) Mbps | TBD |

---

## Recommendations

1. **For Production Use**:
   - Current latency is $(awk '/rtt min\/avg\/max/ {split($4, rtt, "/"); if (rtt[2] < 800) print "acceptable"; else if (rtt[2] < 1000) print "borderline"; else print "high"}' ping-100.txt) for real-time applications
   - Best suited for: File transfers, SSH, non-latency-sensitive apps
   - Not recommended for: Gaming, real-time video calls (due to Starlink latency)

2. **Performance Optimization**:
   $(if [ -f tcp-throughput.json ]; then
       MBPS=$(cat tcp-throughput.json | jq -r '.end.sum_sent.bits_per_second / 1000000' 2>/dev/null)
       if awk -v mbps="$MBPS" 'BEGIN {exit !(mbps < 20)}'; then
           echo "- Consider enabling TCP BBR congestion control"
           echo "- Tune WebSocket buffer sizes"
           echo "- Investigate packet loss causes"
       else
           echo "- Throughput is good, no immediate optimization needed"
       fi
   fi)

3. **Next Steps**:
   - Run 24-hour stability test
   - Test with multiple concurrent connections
   - Compare against Tailscale baseline
   - Document for customer case studies

---

**Generated by ShadowMesh Performance Testing Suite**

_ShadowMesh: The world's first post-quantum VPN network_
EOF

echo "Results saved to:"
echo "  $(pwd)/RESULTS.md"
echo ""
cat RESULTS.md
echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║              Performance Testing Complete!                 ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "Results directory: $(pwd)"
echo ""
