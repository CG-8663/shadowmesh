#!/bin/bash
set -e

# ShadowMesh 4-Region Discovery Backbone Deployment
# Deploys to: London (existing), NYC, Singapore, Sydney
# Naming: shadowmesh-discovery-{loc} (nyc, sin, syd)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  ShadowMesh 4-Region Discovery Backbone Deployment${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

if ! command -v upctl &> /dev/null; then
    echo -e "${RED}Error: upctl not found. Please install UpCloud CLI.${NC}"
    exit 1
fi

# Check UpCloud authentication
if ! upctl account show &> /dev/null; then
    echo -e "${RED}Error: UpCloud CLI not authenticated. Run: upctl config${NC}"
    exit 1
fi

# Get current credits
CREDITS=$(upctl account show | grep Credits | awk '{print $2}')
echo -e "${GREEN}✓ UpCloud authenticated. Credits: $CREDITS${NC}"

# Check SSH key
if [ ! -f ~/.ssh/shadowmesh_relay_ed25519 ]; then
    echo -e "${RED}Error: SSH key not found at ~/.ssh/shadowmesh_relay_ed25519${NC}"
    exit 1
fi
echo -e "${GREEN}✓ SSH key found${NC}"

# Deployment configuration (no associative arrays for bash 3 compatibility)
ZONES="us-nyc1 sg-sin1 au-syd1"
PLAN="1xCPU-2GB"
SSH_KEY=$(cat ~/.ssh/shadowmesh_relay_ed25519.pub)

echo
echo -e "${YELLOW}Deployment Plan:${NC}"
echo -e "  ${BLUE}Regions:${NC}"
echo -e "    - New York (us-nyc1) → shadowmesh-discovery-nyc"
echo -e "    - Singapore (sg-sin1) → shadowmesh-discovery-sin"
echo -e "    - Sydney (au-syd1) → shadowmesh-discovery-syd"
echo -e "  ${BLUE}Plan:${NC} $PLAN (~€11/month per server)"
echo -e "  ${BLUE}Total Cost:${NC} ~€33/month for 3 new servers"
echo -e "  ${BLUE}Existing:${NC} London (shadowmesh-relay-lon, already deployed)"
echo

# Confirm deployment
read -p "Deploy 3 new discovery nodes? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo -e "${YELLOW}Deployment cancelled.${NC}"
    exit 0
fi

echo
echo -e "${GREEN}Starting deployment...${NC}"
echo

# Cloud-init script for discovery nodes
cat > /tmp/discovery-cloud-init.sh << 'CLOUD_INIT_EOF'
#!/bin/bash
set -e

# Update system
apt-get update
apt-get upgrade -y

# Install dependencies
apt-get install -y \
    curl \
    wget \
    git \
    build-essential \
    ufw \
    nginx \
    redis-server \
    postgresql \
    postgresql-contrib

# Install Go 1.21+
cd /tmp
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
export PATH=$PATH:/usr/local/go/bin

# Create shadowmesh user
useradd -r -s /bin/bash -m -d /var/lib/shadowmesh shadowmesh

# Create directories
mkdir -p /etc/shadowmesh
mkdir -p /var/lib/shadowmesh/keys
mkdir -p /var/log/shadowmesh
chown -R shadowmesh:shadowmesh /var/lib/shadowmesh /var/log/shadowmesh

# Configure firewall
ufw --force enable
ufw allow 22/tcp   # SSH
ufw allow 8443/tcp # WebSocket Secure
ufw allow 8080/tcp # HTTP API

# Configure PostgreSQL
sudo -u postgres psql -c "CREATE DATABASE shadowmesh;"
sudo -u postgres psql -c "CREATE USER shadowmesh WITH ENCRYPTED PASSWORD 'changeme';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE shadowmesh TO shadowmesh;"

# Configure Redis
sed -i 's/^# requirepass .*/requirepass changeme/' /etc/redis/redis.conf
systemctl restart redis-server

# Create placeholder for discovery binary (will upload later)
touch /usr/local/bin/shadowmesh-discovery
chmod +x /usr/local/bin/shadowmesh-discovery

echo "Cloud-init complete. Discovery node ready for software deployment."
CLOUD_INIT_EOF

# Function to get location code from zone
get_location_code() {
    case $1 in
        us-nyc1) echo "nyc" ;;
        sg-sin1) echo "sin" ;;
        au-syd1) echo "syd" ;;
        uk-lon1) echo "lon" ;;
        *) echo "unknown" ;;
    esac
}

# Function to get region name from zone
get_region_name() {
    case $1 in
        us-nyc1) echo "North America (New York)" ;;
        sg-sin1) echo "Asia-Pacific (Singapore)" ;;
        au-syd1) echo "Australia (Sydney)" ;;
        uk-lon1) echo "Europe (London)" ;;
        *) echo "Unknown" ;;
    esac
}

# Function to deploy a server
deploy_server() {
    local ZONE=$1
    local LOC_CODE=$(get_location_code "$ZONE")
    local REGION_NAME=$(get_region_name "$ZONE")
    local HOSTNAME="shadowmesh-discovery-${LOC_CODE}"

    echo -e "${BLUE}──────────────────────────────────────────────────────${NC}"
    echo -e "${YELLOW}Deploying: ${HOSTNAME} (${REGION_NAME})${NC}"
    echo -e "${BLUE}──────────────────────────────────────────────────────${NC}"

    # Check if server already exists
    if upctl server list | grep -q "$HOSTNAME"; then
        echo -e "${YELLOW}⚠ Server $HOSTNAME already exists. Skipping...${NC}"
        return 0
    fi

    # Create server
    echo "Creating server..."
    SERVER_UUID=$(upctl server create \
        --hostname "$HOSTNAME" \
        --zone "$ZONE" \
        --plan "$PLAN" \
        --os "01000000-0000-4000-8000-000020080100" \
        --ssh-keys ~/.ssh/shadowmesh_relay_ed25519.pub \
        --user-data file:///tmp/discovery-cloud-init.sh \
        --wait | grep UUID | awk '{print $2}')

    if [ -z "$SERVER_UUID" ]; then
        echo -e "${RED}✗ Failed to create server $HOSTNAME${NC}"
        return 1
    fi

    echo -e "${GREEN}✓ Server created: $SERVER_UUID${NC}"

    # Wait for server to be fully operational
    echo "Waiting for server to initialize (60 seconds)..."
    sleep 60

    # Get server IP
    SERVER_IP=$(upctl server show "$SERVER_UUID" | grep "IPv4:" | awk '{print $2}')
    echo -e "${GREEN}✓ Server IP: $SERVER_IP${NC}"

    # Save deployment info
    cat >> /tmp/shadowmesh-deployment.txt << EOF
Region: $REGION_NAME
Zone: $ZONE
Hostname: $HOSTNAME
UUID: $SERVER_UUID
IP: $SERVER_IP
SSH: ssh -i ~/.ssh/shadowmesh_relay_ed25519 root@$SERVER_IP

EOF

    echo -e "${GREEN}✓ Deployment complete: $HOSTNAME${NC}"
    echo
}

# Deploy to each region
> /tmp/shadowmesh-deployment.txt
echo "ShadowMesh 4-Region Backbone Deployment - $(date)" > /tmp/shadowmesh-deployment.txt
echo "================================================================" >> /tmp/shadowmesh-deployment.txt
echo >> /tmp/shadowmesh-deployment.txt

for ZONE in $ZONES; do
    deploy_server "$ZONE"
done

# Summary
echo
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}         Deployment Complete!${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo
echo -e "${YELLOW}Deployed Servers:${NC}"
upctl server list | grep shadowmesh
echo
echo -e "${YELLOW}Next Steps:${NC}"
echo "1. Build discovery node binary:"
echo "   cd $PROJECT_ROOT"
echo "   make build-discovery"
echo
echo "2. Deploy software to servers:"
echo "   ./scripts/deploy-discovery-software.sh"
echo
echo "3. Test inter-region connectivity:"
echo "   ./scripts/test-backbone-connectivity.sh"
echo
echo -e "${YELLOW}Deployment details saved to:${NC} /tmp/shadowmesh-deployment.txt"
cat /tmp/shadowmesh-deployment.txt
echo
echo -e "${GREEN}Credits remaining:${NC}"
upctl account show | grep Credits
echo
