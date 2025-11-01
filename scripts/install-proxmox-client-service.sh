#!/bin/bash
set -e

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║   ShadowMesh Proxmox Client Service Installer            ║"
echo "║   Installs client as systemd background service          ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "ERROR: This script must be run as root (use sudo)"
    exit 1
fi

# Check if binary exists
if [ ! -f /usr/local/bin/shadowmesh-client ]; then
    echo "ERROR: shadowmesh-client binary not found at /usr/local/bin/shadowmesh-client"
    echo "Please build and install it first:"
    echo "  cd /opt/shadowmesh && make build-client"
    echo "  sudo cp build/shadowmesh-client /usr/local/bin/"
    exit 1
fi

# Check if config exists
if [ ! -f /etc/shadowmesh/config.yaml ]; then
    echo "ERROR: Config file not found at /etc/shadowmesh/config.yaml"
    echo "Please create the config file first."
    exit 1
fi

# Create directories
echo "Creating directories..."
mkdir -p /var/lib/shadowmesh
chmod 700 /var/lib/shadowmesh

# Create systemd service
echo "Creating systemd service..."
cat > /etc/systemd/system/shadowmesh-client.service << 'EOF'
[Unit]
Description=ShadowMesh Post-Quantum VPN Client
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Environment="PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
WorkingDirectory=/var/lib/shadowmesh
ExecStartPre=/sbin/modprobe tun
ExecStart=/usr/local/bin/shadowmesh-client -config /etc/shadowmesh/config.yaml
ExecStartPost=/bin/sleep 2
ExecStartPost=/sbin/ip addr add 10.10.10.2/24 dev tap0
ExecStartPost=/sbin/ip link set tap0 up
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=false
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/var/lib/shadowmesh /var/log /dev/net/tun

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd
echo "Reloading systemd..."
systemctl daemon-reload

# Enable and start service
echo "Enabling and starting service..."
systemctl enable shadowmesh-client
systemctl restart shadowmesh-client

echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║              Service Installation Complete!               ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "Waiting 5 seconds for client to connect..."
sleep 5

echo ""
echo "Service Status:"
systemctl status shadowmesh-client --no-pager -l | tail -20
echo ""
echo "Useful Commands:"
echo "  sudo systemctl status shadowmesh-client    # Check status"
echo "  sudo systemctl restart shadowmesh-client   # Restart client"
echo "  sudo systemctl stop shadowmesh-client      # Stop client"
echo "  sudo journalctl -u shadowmesh-client -f    # View live logs"
echo "  ip addr show tap0                          # Show TAP device"
echo ""
