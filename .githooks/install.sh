#!/bin/bash
# Installation script for Git hooks
# This script sets up Git to use the hooks in .githooks directory

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get the root directory of the git repository
REPO_ROOT=$(git rev-parse --show-toplevel)
HOOKS_DIR="$REPO_ROOT/.githooks"

echo -e "${BLUE}═══════════════════════════════════════════${NC}"
echo -e "${BLUE}     🔧 Git Hooks Installation 🔧${NC}"
echo -e "${BLUE}═══════════════════════════════════════════${NC}"
echo ""

# Check if .githooks directory exists
if [ ! -d "$HOOKS_DIR" ]; then
    echo -e "${RED}✗ .githooks directory not found!${NC}"
    echo -e "${YELLOW}  Please ensure you're in the root of the repository${NC}"
    exit 1
fi

# Method 1: Configure Git to use .githooks directory (recommended)
echo -e "${YELLOW}▶ Configuring Git to use .githooks directory...${NC}"
git config core.hooksPath .githooks

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Git configured to use .githooks directory${NC}"
else
    echo -e "${RED}✗ Failed to configure Git hooks path${NC}"
    exit 1
fi

# Make hooks executable
echo -e "${YELLOW}▶ Making hooks executable...${NC}"
chmod +x "$HOOKS_DIR"/*.sh 2>/dev/null || true
chmod +x "$HOOKS_DIR"/pre-* 2>/dev/null || true
chmod +x "$HOOKS_DIR"/post-* 2>/dev/null || true
chmod +x "$HOOKS_DIR"/commit-msg 2>/dev/null || true
echo -e "${GREEN}✓ Hooks are now executable${NC}"

# List installed hooks
echo ""
echo -e "${YELLOW}▶ Installed hooks:${NC}"
for hook in "$HOOKS_DIR"/*; do
    if [ -f "$hook" ] && [ -x "$hook" ]; then
        basename="$(basename "$hook")"
        if [[ ! "$basename" == *.* ]]; then  # Exclude files with extensions
            echo -e "  ${GREEN}✓${NC} $basename"
        fi
    fi
done

# Test hook configuration
echo ""
echo -e "${YELLOW}▶ Testing configuration...${NC}"
CONFIGURED_PATH=$(git config --get core.hooksPath)
if [ "$CONFIGURED_PATH" = ".githooks" ]; then
    echo -e "${GREEN}✓ Git hooks path correctly configured${NC}"
else
    echo -e "${RED}✗ Git hooks path not correctly configured${NC}"
    echo -e "${YELLOW}  Expected: .githooks${NC}"
    echo -e "${YELLOW}  Got: $CONFIGURED_PATH${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}═══════════════════════════════════════════${NC}"
echo -e "${GREEN}    ✅ Git hooks installed successfully!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════${NC}"
echo ""
echo -e "${BLUE}The following hooks are now active:${NC}"
echo -e "  • ${YELLOW}pre-commit${NC}: Quick formatting and validation checks"
echo -e "  • ${YELLOW}pre-push${NC}: Full quality suite (format, lint, security, tests)"
echo ""
echo -e "${BLUE}To bypass hooks temporarily:${NC}"
echo -e "  • Commit: ${YELLOW}git commit --no-verify${NC}"
echo -e "  • Push: ${YELLOW}git push --no-verify${NC}"
echo ""
echo -e "${BLUE}To uninstall hooks:${NC}"
echo -e "  • Run: ${YELLOW}make hooks/uninstall${NC}"
echo -e "  • Or: ${YELLOW}git config --unset core.hooksPath${NC}"
echo ""