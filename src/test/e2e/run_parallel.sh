#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🚀 Starting E2E tests in parallel...${NC}"

# Start time
START_TIME=$(date +%s)

# Create temporary files for output
ALPINE_LOG=$(mktemp)
UBUNTU_LOG=$(mktemp)
DEBIAN_LOG=$(mktemp)

# Cleanup function
cleanup() {
    rm -f "$ALPINE_LOG" "$UBUNTU_LOG" "$DEBIAN_LOG"
    docker compose -f src/test/e2e/docker-compose.yml down --volumes --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

# Run tests in parallel with output to temp files
echo "⏳ Running Alpine (OpenRC) test..."
./src/test/e2e/test_distro.sh alpine > "$ALPINE_LOG" 2>&1 &
PID_ALPINE=$!

echo "⏳ Running Ubuntu (systemd) test..."
./src/test/e2e/test_distro.sh ubuntu > "$UBUNTU_LOG" 2>&1 &
PID_UBUNTU=$!

echo "⏳ Running Debian (sysvinit) test..."
./src/test/e2e/test_distro.sh debian > "$DEBIAN_LOG" 2>&1 &
PID_DEBIAN=$!

# Wait and check results
FAILED=0
RESULTS=""

# Check Alpine
if wait $PID_ALPINE; then
    RESULTS="${RESULTS}${GREEN}✅ Alpine (OpenRC) test passed${NC}\n"
else
    RESULTS="${RESULTS}${RED}❌ Alpine (OpenRC) test failed${NC}\n"
    echo -e "${RED}Alpine test output:${NC}"
    cat "$ALPINE_LOG"
    FAILED=1
fi

# Check Ubuntu
if wait $PID_UBUNTU; then
    RESULTS="${RESULTS}${GREEN}✅ Ubuntu (systemd) test passed${NC}\n"
else
    RESULTS="${RESULTS}${RED}❌ Ubuntu (systemd) test failed${NC}\n"
    echo -e "${RED}Ubuntu test output:${NC}"
    cat "$UBUNTU_LOG"
    FAILED=1
fi

# Check Debian
if wait $PID_DEBIAN; then
    RESULTS="${RESULTS}${GREEN}✅ Debian (sysvinit) test passed${NC}\n"
else
    RESULTS="${RESULTS}${RED}❌ Debian (sysvinit) test failed${NC}\n"
    echo -e "${RED}Debian test output:${NC}"
    cat "$DEBIAN_LOG"
    FAILED=1
fi

# End time
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Print summary
echo -e "\n${YELLOW}📊 Test Results (${DURATION}s):${NC}"
echo -e "$RESULTS"

if [ $FAILED -eq 1 ]; then
    echo -e "${RED}❌ Some E2E tests failed${NC}"
    exit 1
fi

echo -e "${GREEN}✅ All E2E tests passed successfully!${NC}"