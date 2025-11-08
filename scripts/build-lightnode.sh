#!/bin/bash
set -e

echo "═══════════════════════════════════════════════════════════"
echo "  ShadowMesh Light Node Build Script"
echo "═══════════════════════════════════════════════════════════"
echo

# Create build directory
mkdir -p build

# Download dependencies
echo "[1/4] Downloading dependencies..."
go mod download
go mod tidy

# Build for Linux x86_64 (VPS)
echo "[2/4] Building Linux x86_64 binary..."
GOOS=linux GOARCH=amd64 go build -o build/shadowmesh-lightnode-linux-amd64 ./cmd/lightnode/

# Build for Linux ARM64 (Raspberry Pi 4)
echo "[3/4] Building Linux ARM64 binary (Raspberry Pi)..."
GOOS=linux GOARCH=arm64 go build -o build/shadowmesh-lightnode-linux-arm64 ./cmd/lightnode/

# Build for current platform (local testing)
echo "[4/4] Building local binary..."
go build -o build/shadowmesh-lightnode-local ./cmd/lightnode/

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "  Build Complete!"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "Output:"
echo "  Linux x86_64 (VPS):        build/shadowmesh-lightnode-linux-amd64"
echo "  Linux ARM64 (Raspberry Pi):build/shadowmesh-lightnode-linux-arm64"
echo "  Local binary:              build/shadowmesh-lightnode-local"
echo ""
echo "Binary sizes:"
ls -lh build/shadowmesh-lightnode* | awk '{print "  " $9 ": " $5}'
echo ""
echo "Usage:"
echo "  1. Generate keys:   ./shadowmesh-lightnode -generate-keys"
echo "  2. Start node:      ./shadowmesh-lightnode -backbone http://209.151.148.121:8080"
echo "  3. Connect to peer: ./shadowmesh-lightnode -connect <peer-id>"
echo "  4. Test video:      ./shadowmesh-lightnode -connect <peer-id> -test-video"
echo ""
