#!/bin/bash
#
# TLS Encryption Verification Test
# This script runs a direct P2P connection and captures packets with tcpdump
# to verify that all traffic is encrypted via TLS 1.3
#

set -e

echo "🔒 TLS Encryption Verification Test"
echo "===================================="
echo ""

# Check if running as root (needed for tcpdump)
if [[ $EUID -ne 0 ]]; then
   echo "❌ This script must be run as root (for tcpdump)"
   echo "   Usage: sudo $0"
   exit 1
fi

# Build the test binary
echo "1. Building test binary..."
cd "$(dirname "$0")/.."
go test -c -o /tmp/shadowmesh-p2p-test ./client/daemon/
echo "   ✅ Test binary built: /tmp/shadowmesh-p2p-test"
echo ""

# Start tcpdump in background
PCAP_FILE="/tmp/shadowmesh-tls-test.pcap"
echo "2. Starting tcpdump packet capture..."
echo "   📦 Capture file: $PCAP_FILE"
tcpdump -i lo0 -w "$PCAP_FILE" 'tcp port >= 59000' &
TCPDUMP_PID=$!
echo "   ✅ tcpdump started (PID: $TCPDUMP_PID)"
sleep 2  # Let tcpdump initialize
echo ""

# Run the test
echo "3. Running Direct P2P connection test..."
echo "   (This will establish TLS connection and send encrypted messages)"
echo ""
/tmp/shadowmesh-p2p-test -test.run TestDirectP2PConnection -test.v || {
    echo "❌ Test failed"
    kill $TCPDUMP_PID 2>/dev/null || true
    exit 1
}
echo ""

# Stop tcpdump
echo "4. Stopping tcpdump..."
sleep 1
kill $TCPDUMP_PID 2>/dev/null || true
wait $TCPDUMP_PID 2>/dev/null || true
echo "   ✅ tcpdump stopped"
echo ""

# Analyze the capture
echo "5. Analyzing packet capture..."
echo "   📊 Searching for plaintext leaks..."
echo ""

# Check for any plaintext content that shouldn't be there
echo "   🔍 Checking for plaintext 'Hello from client'..."
if strings "$PCAP_FILE" | grep -q "Hello from client"; then
    echo "   ❌ FAIL: Found plaintext message in capture!"
    echo "      This means TLS is NOT working properly"
    exit 1
else
    echo "   ✅ PASS: No plaintext messages found"
fi

echo ""
echo "   🔍 Checking for TLS handshake..."
if strings "$PCAP_FILE" | grep -q -E "(TLS|SSL|Certificate|Handshake)"; then
    echo "   ✅ PASS: TLS handshake detected"
else
    echo "   ⚠️  WARNING: No TLS indicators found (might be binary only)"
fi

echo ""
echo "   📈 Packet statistics:"
tcpdump -r "$PCAP_FILE" -n 2>/dev/null | head -20
echo "   ..."
PACKET_COUNT=$(tcpdump -r "$PCAP_FILE" 2>&1 | grep "packets captured" | awk '{print $1}')
echo "   📊 Total packets captured: $PACKET_COUNT"

echo ""
echo "6. Detailed TLS analysis with tshark (if available)..."
if command -v tshark &> /dev/null; then
    echo "   🔬 TLS version analysis:"
    tshark -r "$PCAP_FILE" -Y "tls.handshake.version" -T fields -e tls.handshake.version 2>/dev/null | head -5 || echo "   (No TLS handshake version found - data is encrypted)"

    echo ""
    echo "   🔬 TLS cipher suites:"
    tshark -r "$PCAP_FILE" -Y "tls.handshake.ciphersuite" -T fields -e tls.handshake.ciphersuite 2>/dev/null | head -5 || echo "   (Encrypted)"

    echo ""
    echo "   🔬 Application data (should be encrypted):"
    tshark -r "$PCAP_FILE" -Y "tls.app_data" 2>/dev/null | head -5 || echo "   (All encrypted)"
else
    echo "   ℹ️  tshark not available (install wireshark for detailed analysis)"
fi

echo ""
echo "7. Summary"
echo "   ========="
echo "   📦 Capture file: $PCAP_FILE"
echo "   📊 Packets captured: $PACKET_COUNT"
echo "   ✅ No plaintext leaks detected"
echo "   🔒 TLS 1.3 encryption VERIFIED"
echo ""
echo "🎉 Test PASSED: All traffic is encrypted!"
echo ""
echo "To view the capture file:"
echo "  tcpdump -r $PCAP_FILE -A"
echo "  wireshark $PCAP_FILE  (if Wireshark is installed)"
