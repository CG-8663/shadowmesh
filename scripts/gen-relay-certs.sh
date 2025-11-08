#!/bin/bash
# Generate self-signed TLS certificates for ShadowMesh Relay Server

set -e

CERT_DIR="/etc/shadowmesh"
CERT_FILE="$CERT_DIR/relay.crt"
KEY_FILE="$CERT_DIR/relay.key"

echo "Generating self-signed TLS certificate for relay server..."

# Create directory if it doesn't exist
mkdir -p $CERT_DIR

# Generate self-signed certificate (valid for 365 days)
openssl req -x509 -newkey rsa:4096 -sha256 -days 365 -nodes \
  -keyout $KEY_FILE \
  -out $CERT_FILE \
  -subj "/CN=shadowmesh-relay" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:100.115.193.115,IP:192.168.1.111,IP:10.10.10.3"

# Set permissions
chmod 600 $KEY_FILE
chmod 644 $CERT_FILE

echo "✅ TLS certificate generated:"
echo "   Certificate: $CERT_FILE"
echo "   Private Key: $KEY_FILE"

# Display certificate info
echo ""
echo "Certificate details:"
openssl x509 -in $CERT_FILE -noout -subject -dates -fingerprint
