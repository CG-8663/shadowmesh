#!/bin/bash
# TCP Performance Optimization Script for ShadowMesh
# Optimizes TCP for high-latency relay connections

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ShadowMesh TCP Performance Optimizer${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}This script must be run as root (use sudo)${NC}"
   exit 1
fi

# Check available disk space (warn if < 100MB free on /)
ROOT_AVAIL=$(df / | tail -1 | awk '{print $4}')
if [ "$ROOT_AVAIL" -lt 102400 ]; then
    print_warning "Low disk space: $(df -h / | tail -1 | awk '{print $4}') available"
    print_warning "Consider cleaning up /tmp or /var/log if issues occur"
fi

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo -e "${YELLOW}This script will optimize TCP settings for high-latency connections.${NC}"
echo ""
echo "Optimizations to be applied:"
echo "  1. Enable TCP BBR congestion control"
echo "  2. Increase TCP receive/send buffers to 16MB"
echo "  3. Enable TCP window scaling"
echo "  4. Optimize TCP keepalive settings"
echo "  5. Make changes persistent across reboots"
echo ""
read -p "Continue? [y/N]: " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_info "Aborted by user"
    exit 0
fi

echo ""
print_info "Step 1: Checking current TCP settings..."

# Display current settings
echo ""
echo "Current settings:"
echo "  Congestion control: $(sysctl -n net.ipv4.tcp_congestion_control)"
echo "  TCP rmem: $(sysctl -n net.ipv4.tcp_rmem)"
echo "  TCP wmem: $(sysctl -n net.ipv4.tcp_wmem)"
echo "  Window scaling: $(sysctl -n net.ipv4.tcp_window_scaling)"
echo ""

print_info "Step 2: Checking if BBR is available..."

# Check if BBR module is available
if modinfo tcp_bbr &>/dev/null; then
    print_success "TCP BBR module available"
else
    print_warning "TCP BBR module not found - may need kernel 4.9+"
    print_info "Checking kernel version..."
    KERNEL_VERSION=$(uname -r | cut -d. -f1)
    if [ "$KERNEL_VERSION" -lt 4 ]; then
        print_error "Kernel version too old for BBR (requires 4.9+)"
        print_info "Skipping BBR, will only optimize buffers"
        BBR_AVAILABLE=false
    else
        print_info "Kernel supports BBR, loading module..."
        BBR_AVAILABLE=true
    fi
fi

print_info "Step 3: Applying TCP optimizations..."

# Note: Skipping sysctl backup to avoid /tmp space issues on Raspberry Pi
# Current settings already displayed in Step 1 above

# Apply optimizations
echo ""
print_info "Applying runtime optimizations..."

# Enable TCP BBR if available
if [ "${BBR_AVAILABLE}" != "false" ]; then
    modprobe tcp_bbr 2>/dev/null || true
    sysctl -w net.core.default_qdisc=fq
    sysctl -w net.ipv4.tcp_congestion_control=bbr
    print_success "TCP BBR congestion control enabled"
else
    print_warning "Using default congestion control (cubic)"
fi

# Increase TCP buffer sizes (16MB max)
sysctl -w net.ipv4.tcp_rmem="4096 131072 16777216"
sysctl -w net.ipv4.tcp_wmem="4096 131072 16777216"
print_success "TCP buffers increased to 16MB"

# Increase core buffer limits
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
print_success "Core buffer limits increased"

# Enable TCP window scaling
sysctl -w net.ipv4.tcp_window_scaling=1
print_success "TCP window scaling enabled"

# Optimize TCP keepalive for faster dead connection detection
sysctl -w net.ipv4.tcp_keepalive_time=600
sysctl -w net.ipv4.tcp_keepalive_intvl=60
sysctl -w net.ipv4.tcp_keepalive_probes=5
print_success "TCP keepalive optimized"

# Enable TCP timestamps for better RTT measurement
sysctl -w net.ipv4.tcp_timestamps=1
print_success "TCP timestamps enabled"

# Enable selective acknowledgments
sysctl -w net.ipv4.tcp_sack=1
print_success "TCP SACK enabled"

# Increase max syn backlog
sysctl -w net.ipv4.tcp_max_syn_backlog=8192
print_success "TCP SYN backlog increased"

# Enable TCP fast open
sysctl -w net.ipv4.tcp_fastopen=3
print_success "TCP Fast Open enabled"

echo ""
print_info "Step 4: Making changes persistent..."

# Create or update /etc/sysctl.d/99-shadowmesh-tcp.conf
SYSCTL_FILE="/etc/sysctl.d/99-shadowmesh-tcp.conf"
print_info "Creating ${SYSCTL_FILE}..."

cat > "${SYSCTL_FILE}" <<EOF
# ShadowMesh TCP Performance Optimizations
# Generated on $(date)

# TCP BBR Congestion Control
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr

# TCP Buffer Sizes (16MB max)
net.ipv4.tcp_rmem=4096 131072 16777216
net.ipv4.tcp_wmem=4096 131072 16777216
net.core.rmem_max=16777216
net.core.wmem_max=16777216

# TCP Window Scaling
net.ipv4.tcp_window_scaling=1

# TCP Keepalive
net.ipv4.tcp_keepalive_time=600
net.ipv4.tcp_keepalive_intvl=60
net.ipv4.tcp_keepalive_probes=5

# TCP Features
net.ipv4.tcp_timestamps=1
net.ipv4.tcp_sack=1
net.ipv4.tcp_fastopen=3

# TCP Connection Limits
net.ipv4.tcp_max_syn_backlog=8192
EOF

print_success "Configuration saved to ${SYSCTL_FILE}"

# Load BBR module on boot
if [ "${BBR_AVAILABLE}" != "false" ]; then
    if ! grep -q "tcp_bbr" /etc/modules-load.d/modules.conf 2>/dev/null; then
        echo "tcp_bbr" >> /etc/modules-load.d/modules.conf
        print_success "TCP BBR module will load on boot"
    fi
fi

echo ""
print_info "Step 5: Verifying new settings..."
echo ""
echo "New settings:"
echo "  Congestion control: $(sysctl -n net.ipv4.tcp_congestion_control)"
echo "  TCP rmem: $(sysctl -n net.ipv4.tcp_rmem)"
echo "  TCP wmem: $(sysctl -n net.ipv4.tcp_wmem)"
echo "  Window scaling: $(sysctl -n net.ipv4.tcp_window_scaling)"
echo "  Core rmem_max: $(sysctl -n net.core.rmem_max)"
echo "  Core wmem_max: $(sysctl -n net.core.wmem_max)"
echo ""

print_success "TCP optimizations applied successfully!"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. Run this script on BOTH endpoints (10.0.0.1 and 10.0.0.2)"
echo "  2. Restart any existing connections"
echo "  3. Re-run iperf3 test to measure improvement:"
echo "     ${GREEN}iperf3 -c 10.0.0.X -t 30 -P 4${NC}"
echo ""
echo -e "${BLUE}Expected improvements:${NC}"
echo "  - 2-3x throughput increase with BBR"
echo "  - Fewer retransmissions"
echo "  - Better utilization of available bandwidth"
echo "  - More stable performance under varying latency"
echo ""

# Check if reboot is needed
if [ "${BBR_AVAILABLE}" != "false" ]; then
    print_warning "For full BBR functionality, a reboot is recommended"
    echo "  Run: sudo reboot"
fi
