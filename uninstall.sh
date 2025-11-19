#!/bin/bash
# ShadowMesh Uninstaller
# Usage: curl -sSL https://raw.githubusercontent.com/cg-8663/shadowmesh/main/uninstall.sh | sudo bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_info() { echo -e "${YELLOW}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }

if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ Please run with sudo${NC}"
    exit 1
fi

echo ""
echo "════════════════════════════════════════════════════════"
echo "ShadowMesh Uninstaller"
echo "════════════════════════════════════════════════════════"
echo ""
echo "This will remove:"
echo "  • ShadowMesh daemon service"
echo "  • Binary at /usr/local/bin/shadowmesh-daemon"
echo "  • Configuration at /etc/shadowmesh/"
echo "  • TAP interface chr001"
echo ""
read -p "Continue? [y/N]: " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo "Uninstall cancelled"
    exit 0
fi
echo ""

# Stop and disable services
print_info "Stopping services..."
systemctl stop shadowmesh-daemon 2>/dev/null || true
systemctl disable shadowmesh-daemon 2>/dev/null || true
systemctl stop shadowmesh 2>/dev/null || true
systemctl disable shadowmesh 2>/dev/null || true
systemctl stop shadowmesh-client 2>/dev/null || true
systemctl disable shadowmesh-client 2>/dev/null || true
print_success "Services stopped"

# Remove service files
print_info "Removing service files..."
rm -f /etc/systemd/system/shadowmesh-daemon.service
rm -f /etc/systemd/system/shadowmesh.service
rm -f /etc/systemd/system/shadowmesh-client.service
systemctl daemon-reload
print_success "Service files removed"

# Kill any running processes
print_info "Killing running processes..."
pkill -9 shadowmesh 2>/dev/null || true
sleep 1
print_success "Processes killed"

# Remove TAP interface
print_info "Removing TAP interface..."
ip link delete chr001 2>/dev/null || true
print_success "TAP interface removed"

# Remove binary
print_info "Removing binary..."
rm -f /usr/local/bin/shadowmesh-daemon
print_success "Binary removed"

# Remove configuration (ask first)
echo ""
read -p "Remove configuration files? [y/N]: " remove_config
if [[ "$remove_config" =~ ^[Yy]$ ]]; then
    rm -rf /etc/shadowmesh
    print_success "Configuration removed"
else
    print_info "Configuration kept at /etc/shadowmesh/"
fi

echo ""
echo "════════════════════════════════════════════════════════"
echo "Uninstall Complete!"
echo "════════════════════════════════════════════════════════"
echo ""
echo "Note: Repository at /opt/shadowmesh was NOT removed"
echo "To remove it: sudo rm -rf /opt/shadowmesh"
echo ""
