#!/bin/bash
# ShadowMesh Remote Test via MCP Server
# Run tests on Mac Studio M1 Max for faster execution

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

MCP_URL="http://100.113.157.118:3000/mcp"
PROJECT_PATH="/Volumes/backupdisk/WebCode/shadowmesh"

echo -e "${BLUE}╔════════════════════════════════════════════════╗"
echo "║    ShadowMesh - Remote Tests via MCP           ║"
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

# Run tests via MCP
echo -e "${GREEN}Running tests on Mac Studio...${NC}"
echo ""

RESPONSE=$(curl -s -X POST "$MCP_URL" \
    -H "Content-Type: application/json" \
    -d "{
        \"jsonrpc\":\"2.0\",
        \"id\":1,
        \"method\":\"tools/call\",
        \"params\":{
            \"name\":\"shell:test\",
            \"arguments\":{
                \"type\":\"go\",
                \"path\":\"$PROJECT_PATH\"
            }
        }
    }")

echo "$RESPONSE" | jq -r '.result.stdout' 2>/dev/null || echo "$RESPONSE"

# Check test results
if echo "$RESPONSE" | grep -q '"error"'; then
    echo ""
    echo -e "${RED}❌ Tests failed${NC}"
    exit 1
else
    echo ""
    echo -e "${GREEN}✅ All tests passed${NC}"
fi
