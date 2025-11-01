#!/bin/bash
#
# Generate self-signed TLS certificates for ShadowMesh relay testing
#
# Usage: ./scripts/generate-test-certs.sh [output_dir]
#
# This script creates a self-signed certificate authority (CA) and a relay
# server certificate signed by that CA. These certificates are suitable for
# local testing only and should NOT be used in production.
#

set -e

# Default output directory
OUTPUT_DIR="${1:-./test-certs}"

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║       ShadowMesh Test Certificate Generator              ║"
echo "║       (For Development/Testing Only)                      ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"
cd "$OUTPUT_DIR"

echo "Output directory: $(pwd)"
echo ""

# Generate CA private key
echo "[1/6] Generating Certificate Authority (CA) private key..."
openssl genrsa -out ca-key.pem 4096 2>/dev/null

# Generate CA certificate
echo "[2/6] Generating CA certificate (10 year validity)..."
openssl req -new -x509 -days 3650 -key ca-key.pem -sha256 -out ca-cert.pem -subj "/C=US/ST=Test/L=Test/O=ShadowMesh Test CA/CN=ShadowMesh Test Root CA"

# Generate relay server private key
echo "[3/6] Generating relay server private key..."
openssl genrsa -out relay-key.pem 4096 2>/dev/null

# Generate relay server certificate signing request (CSR)
echo "[4/6] Generating relay server CSR..."
openssl req -new -key relay-key.pem -out relay-csr.pem -subj "/C=US/ST=Test/L=Test/O=ShadowMesh Relay/CN=localhost"

# Create extension file for Subject Alternative Names (SAN)
echo "[5/6] Creating certificate extensions..."
cat > relay-ext.cnf <<EOF
subjectAltName = DNS:localhost,DNS:*.localhost,IP:127.0.0.1,IP:0.0.0.0
extendedKeyUsage = serverAuth
EOF

# Sign relay server certificate with CA
echo "[6/6] Signing relay server certificate with CA..."
openssl x509 -req -days 365 -in relay-csr.pem -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out relay-cert.pem -sha256 -extfile relay-ext.cnf

# Set proper permissions
chmod 600 ca-key.pem relay-key.pem
chmod 644 ca-cert.pem relay-cert.pem

# Clean up temporary files
rm relay-csr.pem relay-ext.cnf ca-cert.srl 2>/dev/null || true

echo ""
echo "✅ Certificate generation complete!"
echo ""
echo "Generated files:"
echo "  📁 $(pwd)"
echo "  ├── ca-cert.pem      (CA certificate - install on clients)"
echo "  ├── ca-key.pem       (CA private key - keep secure)"
echo "  ├── relay-cert.pem   (Relay server certificate)"
echo "  └── relay-key.pem    (Relay server private key)"
echo ""
echo "Next steps:"
echo "  1. Configure relay server to use relay-cert.pem and relay-key.pem"
echo "  2. Install ca-cert.pem on all client machines as trusted CA"
echo "  3. Start relay server: sudo ./build/shadowmesh-relay"
echo ""
echo "⚠️  WARNING: These certificates are for TESTING ONLY"
echo "    Do NOT use in production environments!"
echo ""
