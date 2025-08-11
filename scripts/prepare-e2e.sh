#\!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
E2E_DIR="src/test/e2e"
BINARY_NAME="cloud-update"

echo -e "${YELLOW}▶ Preparing E2E test environment...${NC}"

# Check if we're in the project root
if [ \! -f "go.mod" ]; then
    echo -e "${RED}✗ Must run from project root${NC}"
    exit 1
fi

# Build the binary for Linux (containers need Linux binary)
echo -e "${YELLOW}▶ Building Linux binary for E2E tests...${NC}"
GOOS=linux GOARCH=amd64 go build -o "${E2E_DIR}/${BINARY_NAME}" ./src/cmd/cloud-update

# Make the binary executable
chmod +x "${E2E_DIR}/${BINARY_NAME}"

# Set environment variables for E2E tests
export E2E_SECRET="test-secret-key-for-e2e-testing-purposes-only"
export CLOUD_UPDATE_SECRET="test-secret-key-for-e2e-testing-purposes-only"
export CLOUD_UPDATE_LOG_LEVEL="debug"

# Clean up any existing containers
echo -e "${YELLOW}▶ Cleaning up existing containers...${NC}"
cd "${E2E_DIR}"
docker compose down --remove-orphans 2>/dev/null || true

# Clean Docker cache for problematic images
echo -e "${YELLOW}▶ Cleaning Docker cache...${NC}"
docker rmi cloud-update-alpine cloud-update-ubuntu cloud-update-debian 2>/dev/null || true

echo -e "${GREEN}✓ E2E environment prepared${NC}"