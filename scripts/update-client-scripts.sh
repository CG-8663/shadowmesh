#!/bin/bash
#
# Update ShadowMesh Client Scripts from GitHub
# Run this on each client machine to get latest performance testing scripts
#

set -e

REPO_URL="https://github.com/CG-8663/shadowmesh.git"
INSTALL_DIR="/opt/shadowmesh"

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║       ShadowMesh Client Scripts Updater                   ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Check if git is installed
if ! command -v git &> /dev/null; then
    echo "Installing git..."
    sudo apt-get update
    sudo apt-get install -y git
fi

# Check if repo exists
if [ -d "$INSTALL_DIR/.git" ]; then
    echo "Updating existing repository..."
    cd "$INSTALL_DIR"

    # Stash any local changes
    sudo git stash

    # Pull latest changes
    sudo git pull origin main

    echo "✅ Repository updated to latest version"
else
    echo "Cloning repository..."
    sudo mkdir -p "$INSTALL_DIR"
    sudo git clone "$REPO_URL" "$INSTALL_DIR"

    echo "✅ Repository cloned successfully"
fi

# Make scripts executable
echo ""
echo "Making scripts executable..."
sudo chmod +x "$INSTALL_DIR"/scripts/*.sh

# List available scripts
echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║              Available Performance Scripts                 ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

echo "Test Scripts:"
echo "  $INSTALL_DIR/scripts/quick-perf-test.sh"
echo "    - Quick 5-minute performance validation"
echo ""
echo "  $INSTALL_DIR/scripts/compare-tailscale-shadowmesh.sh"
echo "    - Full A/B comparison with Tailscale (15 minutes)"
echo ""

# Install dependencies if needed
echo "Checking dependencies..."

# Check for speedtest-cli
if ! command -v speedtest-cli &> /dev/null; then
    read -p "Install speedtest-cli for baseline speed tests? (y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        sudo apt-get install -y speedtest-cli
        echo "✅ speedtest-cli installed"
    fi
fi

# Check for iperf3
if ! command -v iperf3 &> /dev/null; then
    read -p "Install iperf3 for throughput tests? (y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        sudo apt-get install -y iperf3
        echo "✅ iperf3 installed"
    fi
fi

# Check for jq (used by comparison script)
if ! command -v jq &> /dev/null; then
    echo "Installing jq for JSON processing..."
    sudo apt-get install -y jq
    echo "✅ jq installed"
fi

echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║                  Update Complete!                          ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "Repository location: $INSTALL_DIR"
echo "Git branch: $(cd $INSTALL_DIR && git branch --show-current)"
echo "Latest commit: $(cd $INSTALL_DIR && git log -1 --oneline)"
echo ""
echo "To run performance tests:"
echo "  cd $INSTALL_DIR"
echo "  ./scripts/quick-perf-test.sh"
echo "  ./scripts/compare-tailscale-shadowmesh.sh"
echo ""
