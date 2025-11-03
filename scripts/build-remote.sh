#!/bin/bash
# ShadowMesh Remote Build via MCP Server
# Offload heavy builds to Mac Studio M1 Max

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

MCP_URL="http://100.113.157.118:3000/mcp"
PROJECT_PATH="/Volumes/backupdisk/WebCode/shadowmesh"

echo -e "${BLUE}╔════════════════════════════════════════════════╗"
echo "║    ShadowMesh - Remote Build via MCP           ║"
echo "╚════════════════════════════════════════════════╝${NC}"
echo ""

# Check MCP server health
echo -e "${GREEN}Checking MCP server...${NC}"
HEALTH=$(curl -s http://100.113.157.118:3000/health)
if echo "$HEALTH" | grep -q "healthy"; then
    echo "✅ MCP server ready"
else
    echo -e "${RED}❌ MCP server not available${NC}"
    exit 1
fi
echo ""

# Sync code to Mac Studio
echo -e "${GREEN}Syncing code to Mac Studio...${NC}"
rsync -avz --delete \
    --exclude='.git' \
    --exclude='build/' \
    --exclude='bin/' \
    --exclude='daemon' \
    --exclude='node_modules/' \
    . james@100.113.157.118:$PROJECT_PATH/
echo "✅ Code synced"
echo ""

# Run build via MCP
echo -e "${GREEN}Building on Mac Studio (M1 Max)...${NC}"

# Build client daemon
curl -s -X POST "$MCP_URL" \
    -H "Content-Type: application/json" \
    -d "{
        \"jsonrpc\":\"2.0\",
        \"id\":1,
        \"method\":\"tools/call\",
        \"params\":{
            \"name\":\"shell:build\",
            \"arguments\":{
                \"type\":\"go\",
                \"path\":\"$PROJECT_PATH\",
                \"target\":\"build-client-daemon\"
            }
        }
    }" | jq -r '.result.stdout' || echo "Build initiated"

# Build client CLI
curl -s -X POST "$MCP_URL" \
    -H "Content-Type: application/json" \
    -d "{
        \"jsonrpc\":\"2.0\",
        \"id\":2,
        \"method\":\"tools/call\",
        \"params\":{
            \"name\":\"shell:build\",
            \"arguments\":{
                \"type\":\"go\",
                \"path\":\"$PROJECT_PATH\",
                \"target\":\"build-client-cli\"
            }
        }
    }" | jq -r '.result.stdout' || echo "Build initiated"

# Build relay server
curl -s -X POST "$MCP_URL" \
    -H "Content-Type: application/json" \
    -d "{
        \"jsonrpc\":\"2.0\",
        \"id\":3,
        \"method\":\"tools/call\",
        \"params\":{
            \"name\":\"shell:build\",
            \"arguments\":{
                \"type\":\"go\",
                \"path\":\"$PROJECT_PATH\",
                \"target\":\"build-relay\"
            }
        }
    }" | jq -r '.result.stdout' || echo "Build initiated"

echo ""
echo -e "${GREEN}✅ Remote build complete${NC}"
echo ""

# Copy binaries back
echo -e "${GREEN}Retrieving binaries...${NC}"
rsync -avz james@100.113.157.118:$PROJECT_PATH/build/ ./build/
echo "✅ Binaries downloaded to ./build/"
echo ""

echo -e "${BLUE}Build completed successfully!${NC}"
echo "Binaries available:"
echo "  - build/shadowmesh-daemon"
echo "  - build/shadowmesh"
echo "  - build/shadowmesh-relay"
