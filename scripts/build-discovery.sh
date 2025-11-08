#!/bin/bash
set -e

echo "═══════════════════════════════════════════════════════════"
echo "  ShadowMesh Discovery Node Build Script"
echo "═══════════════════════════════════════════════════════════"
echo

# Create build directory
mkdir -p build

# Download dependencies
echo "[1/3] Downloading dependencies..."
go mod download
go mod tidy

# Build for Linux (deployment target)
echo "[2/3] Building Linux binary (amd64)..."
GOOS=linux GOARCH=amd64 go build -o build/shadowmesh-discovery ./cmd/discovery/

# Build for current platform (local testing)
echo "[3/3] Building local binary..."
go build -o build/shadowmesh-discovery-local ./cmd/discovery/

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "  Build Complete!"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "Output:"
echo "  Linux binary (for deployment):  build/shadowmesh-discovery"
echo "  Local binary (for testing):     build/shadowmesh-discovery-local"
echo ""
echo "Binary size:"
ls -lh build/shadowmesh-discovery* | awk '{print "  " $9 ": " $5}'
echo ""
echo "Next steps:"
echo "  1. Test locally: ./build/shadowmesh-discovery-local -generate-config"
echo "  2. Deploy: ./scripts/deploy-discovery-node.sh <server-ip> <region>"
echo ""
