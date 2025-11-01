#!/bin/bash
#
# ShadowMesh UpCloud VM Deployment Script
# Uses upctl CLI to create and configure relay server
#
# Prerequisites:
#   - upctl CLI installed (https://github.com/UpCloudLtd/upcloud-cli)
#   - API token configured: upctl config set --key username=YOUR_USERNAME token=YOUR_TOKEN
#

set -e

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║       ShadowMesh UpCloud Relay Deployment                ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Configuration
HOSTNAME="${1:-shadowmesh-relay}"
ZONE="${2:-de-fra1}"  # Default: Frankfurt
PLAN="1xCPU-2GB"      # 1 vCPU, 2 GB RAM, 50 GB disk
OS="Ubuntu 22.04"

# Check if upctl is installed
if ! command -v upctl &> /dev/null; then
    echo "ERROR: upctl CLI not found"
    echo ""
    echo "Install with:"
    echo "  macOS: brew install upctl"
    echo "  Linux: https://github.com/UpCloudLtd/upcloud-cli/releases"
    echo ""
    exit 1
fi

# Check if upctl is configured
if ! upctl account show &> /dev/null; then
    echo "ERROR: upctl not configured"
    echo ""
    echo "Configure with your API token:"
    echo "  upctl config set --key username=YOUR_USERNAME token=YOUR_TOKEN"
    echo ""
    exit 1
fi

echo "✅ upctl CLI configured"
echo ""
echo "Deployment Configuration:"
echo "  Hostname: $HOSTNAME"
echo "  Zone: $ZONE"
echo "  Plan: $PLAN"
echo "  OS: $OS"
echo ""

# List available zones
echo "Available zones:"
upctl zone list | grep -E "(Zone|$ZONE)" | head -10
echo ""

read -p "Continue with deployment? (y/N): " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled"
    exit 0
fi

# Generate SSH key if needed
SSH_KEY_PATH="$HOME/.ssh/shadowmesh_relay_ed25519"
if [ ! -f "$SSH_KEY_PATH" ]; then
    echo "Generating SSH key..."
    ssh-keygen -t ed25519 -f "$SSH_KEY_PATH" -N "" -C "shadowmesh-relay"
    echo "✅ SSH key generated: $SSH_KEY_PATH"
fi

# Get or upload SSH key to UpCloud
SSH_KEY_NAME="shadowmesh-relay-key"
echo "Checking for existing SSH key in UpCloud..."

if ! upctl sshkey list | grep -q "$SSH_KEY_NAME"; then
    echo "Uploading SSH key to UpCloud..."
    upctl sshkey create \
        --title "$SSH_KEY_NAME" \
        --ssh-key "$(cat ${SSH_KEY_PATH}.pub)"
    echo "✅ SSH key uploaded"
else
    echo "✅ SSH key already exists in UpCloud"
fi

# Create firewall rules for post-deployment
FIREWALL_RULES=$(cat <<'EOF'
#!/bin/bash
# Allow SSH
ufw allow 22/tcp
# Allow ShadowMesh relay
ufw allow 8443/tcp
# Enable firewall
echo "y" | ufw enable
EOF
)

# Create cloud-init user data
CLOUD_INIT=$(cat <<'EOF'
#cloud-config
package_update: true
package_upgrade: true
packages:
  - curl
  - git
  - ufw

runcmd:
  - |
    # Configure firewall
    ufw allow 22/tcp
    ufw allow 8443/tcp
    echo "y" | ufw enable
  - |
    # Install ShadowMesh relay
    curl -sSL https://raw.githubusercontent.com/CG-8663/shadowmesh/main/scripts/install-relay.sh | bash
    systemctl enable shadowmesh-relay
    systemctl start shadowmesh-relay
  - |
    # Log completion
    echo "ShadowMesh relay installation completed at $(date)" >> /var/log/shadowmesh-install.log

final_message: "ShadowMesh Relay Server is ready!"
EOF
)

# Save cloud-init to temp file
TEMP_CLOUD_INIT="/tmp/shadowmesh-cloud-init.yaml"
echo "$CLOUD_INIT" > "$TEMP_CLOUD_INIT"

echo ""
echo "Creating UpCloud VM..."
echo "This may take 2-3 minutes..."
echo ""

# Create the server
SERVER_UUID=$(upctl server create \
    --hostname "$HOSTNAME" \
    --zone "$ZONE" \
    --plan "$PLAN" \
    --os "$OS" \
    --ssh-keys "$SSH_KEY_NAME" \
    --user-data "$(cat $TEMP_CLOUD_INIT)" \
    --enable-ipv4 \
    --format json | jq -r '.uuid')

if [ -z "$SERVER_UUID" ]; then
    echo "ERROR: Failed to create server"
    rm -f "$TEMP_CLOUD_INIT"
    exit 1
fi

echo "✅ Server created: $SERVER_UUID"
echo ""
echo "Waiting for server to start..."

# Wait for server to be in 'started' state
MAX_WAIT=120
ELAPSED=0
while [ $ELAPSED -lt $MAX_WAIT ]; do
    STATE=$(upctl server show "$SERVER_UUID" --format json | jq -r '.state')
    if [ "$STATE" == "started" ]; then
        break
    fi
    echo -n "."
    sleep 5
    ELAPSED=$((ELAPSED + 5))
done

if [ "$STATE" != "started" ]; then
    echo ""
    echo "ERROR: Server failed to start within $MAX_WAIT seconds"
    exit 1
fi

echo ""
echo "✅ Server started"
echo ""

# Get server details
SERVER_INFO=$(upctl server show "$SERVER_UUID" --format json)
SERVER_IP=$(echo "$SERVER_INFO" | jq -r '.ip_addresses[] | select(.access == "public" and .family == "IPv4") | .address')
SERVER_HOSTNAME=$(echo "$SERVER_INFO" | jq -r '.hostname')

rm -f "$TEMP_CLOUD_INIT"

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║       UpCloud VM Deployed Successfully!                   ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "📋 Server Information:"
echo "   UUID: $SERVER_UUID"
echo "   Hostname: $SERVER_HOSTNAME"
echo "   Public IP: $SERVER_IP"
echo "   Zone: $ZONE"
echo "   Plan: $PLAN"
echo ""
echo "🔐 SSH Access:"
echo "   ssh root@$SERVER_IP -i $SSH_KEY_PATH"
echo ""
echo "⏳ Installation Progress:"
echo "   The relay server is being installed via cloud-init (2-3 minutes)"
echo "   Monitor progress:"
echo "     ssh root@$SERVER_IP -i $SSH_KEY_PATH 'tail -f /var/log/shadowmesh-install.log'"
echo ""
echo "🎯 After Installation Completes:"
echo ""
echo "   WebSocket URL: wss://$SERVER_IP:8443/ws"
echo ""
echo "   Test health:"
echo "     curl -k https://$SERVER_IP:8443/health"
echo ""
echo "   Check status:"
echo "     ssh root@$SERVER_IP -i $SSH_KEY_PATH 'systemctl status shadowmesh-relay'"
echo ""
echo "   View logs:"
echo "     ssh root@$SERVER_IP -i $SSH_KEY_PATH 'journalctl -u shadowmesh-relay -f'"
echo ""
echo "📖 Next Steps:"
echo "   1. Wait for installation to complete (2-3 minutes)"
echo "   2. Verify relay is running: curl -k https://$SERVER_IP:8443/health"
echo "   3. Configure client with URL: wss://$SERVER_IP:8443/ws"
echo "   4. See DISTRIBUTED_TESTING.md for full testing guide"
echo ""
