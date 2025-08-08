#!/bin/bash
# Complete quality validation script
# This script ensures ALL quality checks pass

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🔍 Starting complete quality validation...${NC}"

# Track if we have any issues
ISSUES_FOUND=0

# Step 1: Check Go formatting
echo -e "${YELLOW}1. Checking Go formatting...${NC}"
UNFMT=$(gofmt -l .)
if [ -n "$UNFMT" ]; then
    echo -e "${RED}✗ Unformatted Go files found:${NC}"
    echo "$UNFMT"
    ISSUES_FOUND=1
else
    echo -e "${GREEN}✓ Go formatting OK${NC}"
fi

# Step 2: Check imports
echo -e "${YELLOW}2. Checking Go imports...${NC}"
if command -v goimports > /dev/null 2>&1; then
    UNIMP=$(goimports -l -local github.com/kodflow/cloud-update .)
    if [ -n "$UNIMP" ]; then
        echo -e "${RED}✗ Import issues found:${NC}"
        echo "$UNIMP"
        ISSUES_FOUND=1
    else
        echo -e "${GREEN}✓ Go imports OK${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  goimports not installed, skipping${NC}"
fi

# Step 3: Run golangci-lint
echo -e "${YELLOW}3. Running golangci-lint...${NC}"
if command -v golangci-lint > /dev/null 2>&1; then
    if golangci-lint run --timeout=5m ./src/... > /tmp/lint-output.txt 2>&1; then
        echo -e "${GREEN}✓ golangci-lint passed${NC}"
    else
        echo -e "${RED}✗ Linter issues found:${NC}"
        cat /tmp/lint-output.txt
        ISSUES_FOUND=1
    fi
else
    echo -e "${YELLOW}⚠️  golangci-lint not installed, skipping${NC}"
fi

# Step 4: Security scan with gosec
echo -e "${YELLOW}4. Running security scan...${NC}"
if command -v gosec > /dev/null 2>&1; then
    if gosec -quiet -conf .gosec.json ./src/... > /tmp/gosec-output.txt 2>&1; then
        echo -e "${GREEN}✓ Security scan passed${NC}"
    else
        echo -e "${YELLOW}⚠️  Security warnings (non-blocking):${NC}"
        cat /tmp/gosec-output.txt | head -20
    fi
else
    echo -e "${YELLOW}⚠️  gosec not installed, skipping${NC}"
fi

# Step 5: Check Bazel build files
echo -e "${YELLOW}5. Validating Bazel configuration...${NC}"
if bazel query //src/... --output=label > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Bazel configuration valid${NC}"
else
    echo -e "${RED}✗ Bazel configuration errors${NC}"
    ISSUES_FOUND=1
fi

# Step 6: Run unit tests
echo -e "${YELLOW}6. Running unit tests...${NC}"
if bazel test //src/internal/... //src/cmd/... --test_output=errors --cache_test_results=no > /tmp/test-output.txt 2>&1; then
    echo -e "${GREEN}✓ Unit tests passed${NC}"
else
    echo -e "${RED}✗ Test failures:${NC}"
    cat /tmp/test-output.txt | tail -20
    ISSUES_FOUND=1
fi

# Step 7: Check Go module consistency
echo -e "${YELLOW}7. Checking Go module consistency...${NC}"
go mod tidy
if git diff --exit-code go.mod go.sum > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Go modules consistent${NC}"
else
    echo -e "${YELLOW}⚠️  go.mod/go.sum were updated by go mod tidy${NC}"
fi

# Final result
echo ""
if [ $ISSUES_FOUND -eq 0 ]; then
    echo -e "${GREEN}✅ QUALITY VALIDATION PASSED - Code is pristine!${NC}"
    echo -e "${GREEN}The codebase meets all quality standards.${NC}"
    exit 0
else
    echo -e "${RED}❌ QUALITY VALIDATION FAILED - Issues need to be fixed${NC}"
    echo -e "${YELLOW}Run 'make fix' to automatically fix some issues${NC}"
    exit 1
fi