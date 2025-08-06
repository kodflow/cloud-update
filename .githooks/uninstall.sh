#!/bin/bash
# Uninstallation script for Git hooks
# This script removes the Git hooks configuration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}═══════════════════════════════════════════${NC}"
echo -e "${BLUE}     🔧 Git Hooks Uninstallation 🔧${NC}"
echo -e "${BLUE}═══════════════════════════════════════════${NC}"
echo ""

# Check current hooks configuration
CURRENT_PATH=$(git config --get core.hooksPath)

if [ -z "$CURRENT_PATH" ]; then
    echo -e "${YELLOW}▶ Git hooks are not configured (using default .git/hooks)${NC}"
elif [ "$CURRENT_PATH" != ".githooks" ]; then
    echo -e "${YELLOW}▶ Git hooks are configured to: $CURRENT_PATH${NC}"
    echo -e "${YELLOW}  Not managed by this script, skipping...${NC}"
    exit 0
else
    echo -e "${YELLOW}▶ Removing Git hooks configuration...${NC}"
    git config --unset core.hooksPath
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Git hooks configuration removed${NC}"
        echo -e "${GREEN}  Git will now use default .git/hooks directory${NC}"
    else
        echo -e "${RED}✗ Failed to remove Git hooks configuration${NC}"
        exit 1
    fi
fi

echo ""
echo -e "${GREEN}═══════════════════════════════════════════${NC}"
echo -e "${GREEN}    ✅ Git hooks uninstalled successfully!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════${NC}"
echo ""
echo -e "${BLUE}To reinstall hooks later:${NC}"
echo -e "  • Run: ${YELLOW}make hooks/install${NC}"
echo -e "  • Or: ${YELLOW}./.githooks/install.sh${NC}"
echo ""