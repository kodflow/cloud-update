#!/bin/bash
# Automatic build validation script
# This script MUST pass after every code modification

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🔍 Starting build validation...${NC}"

# Step 1: Clean dependencies
echo -e "${YELLOW}1. Cleaning Go dependencies...${NC}"
go mod tidy
echo -e "${GREEN}✓ Dependencies cleaned${NC}"

# Step 2: Run Bazel unit tests
echo -e "${YELLOW}2. Running unit tests with Bazel...${NC}"
bazel test //src/internal/... //src/cmd/... --test_output=errors --cache_test_results=no
echo -e "${GREEN}✓ Unit tests passed${NC}"

# Step 3: Check Go compilation
echo -e "${YELLOW}3. Checking Go compilation...${NC}"
cd src && go build ./... && cd ..
echo -e "${GREEN}✓ Go compilation successful${NC}"

# Step 4: Validate BUILD.bazel files
echo -e "${YELLOW}4. Validating Bazel configuration...${NC}"
bazel query //src/... --output=label > /dev/null
echo -e "${GREEN}✓ Bazel configuration valid${NC}"

echo -e "${GREEN}✅ BUILD VALIDATION SUCCESSFUL - All checks passed!${NC}"
echo -e "${GREEN}The build is green and ready for further development.${NC}"