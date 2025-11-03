#!/bin/bash
#
# Redact Sensitive Information Before Publishing
# This script redacts public IPs, usernames, and other sensitive data
#

set -e

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║     ShadowMesh Documentation Redaction Script             ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Backup original files
BACKUP_DIR="docs-backup-$(date +%Y%m%d-%H%M%S)"
echo "Creating backup in $BACKUP_DIR..."
cp -r docs "$BACKUP_DIR"

# Files to redact
FILES=$(find docs -type f -name "*.md")

echo "Redacting sensitive information..."
echo ""

for file in $FILES; do
    echo "Processing: $file"

    # Replace public IPs with placeholders
    sed -i.bak 's/83\.136\.252\.52/<RELAY_PUBLIC_IP>/g' "$file"
    sed -i.bak 's/80\.229\.0\.71/<CLIENT_PUBLIC_IP>/g' "$file"

    # Replace ISP names with generic terms
    sed -i.bak 's/Plusnet ISP/[UK ISP]/g' "$file"
    sed -i.bak 's/Plusnet/[UK ISP]/g' "$file"
    sed -i.bak 's/STARLINK/[Satellite ISP]/g' "$file"
    sed -i.bak 's/Starlink/[Satellite ISP]/g' "$file"

    # Fix incorrect location references (Philippines → UK)
    sed -i.bak 's/Philippines/UK/g' "$file"
    sed -i.bak 's/Aparri, UK/UK/g' "$file"  # Fix double replacement
    sed -i.bak 's/North Luzon, //' "$file"
    sed -i.bak 's/Aparri/UK Region/g' "$file"

    # Replace usernames if found
    sed -i.bak 's/pxcghost/<USERNAME>/g' "$file"
    sed -i.bak 's/drees/<USERNAME>/g' "$file"

    # Remove .bak files
    rm -f "${file}.bak"
done

echo ""
echo "✅ Redaction complete!"
echo ""
echo "Summary of changes:"
echo "  - Public IPs replaced with <RELAY_PUBLIC_IP> and <CLIENT_PUBLIC_IP>"
echo "  - ISP names replaced with [UK ISP] and [Satellite ISP]"
echo "  - Location details genericized"
echo "  - Usernames replaced with <USERNAME>"
echo ""
echo "Backup saved to: $BACKUP_DIR"
echo ""
echo "Please review the changes before committing:"
echo "  git diff docs/"
echo ""
