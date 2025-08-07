#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
DOCKER_COMPOSE_FILE="src/test/e2e/docker-compose.yml"
DISTRIBUTIONS="alpine ubuntu debian rockylinux fedora arch opensuse"
FAILED_TESTS=0

echo -e "${GREEN}=== Starting E2E Tests ===${NC}"

# Build binary for Linux
echo -e "${YELLOW}Building Linux binary with Bazel...${NC}"
bazel build //src/cmd/cloud-update:cloud-update --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64

# Prepare dist directory for Docker
rm -rf dist/cloud-update_linux_amd64_v1
mkdir -p dist/cloud-update_linux_amd64_v1

# Find the correct binary path (Bazel creates different paths for different platforms)
BINARY_PATH=$(find bazel-bin/src/cmd/cloud-update -name "cloud-update" -type f | grep -E "linux.*amd64" | head -1)
if [ -z "$BINARY_PATH" ]; then
    # Fallback to the standard path
    BINARY_PATH="bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update"
fi

echo "Copying binary from: $BINARY_PATH"
cp "$BINARY_PATH" dist/cloud-update_linux_amd64_v1/cloud-update
chmod 755 dist/cloud-update_linux_amd64_v1/cloud-update

# Copy install script to dist
cp scripts/install.sh dist/cloud-update_linux_amd64_v1/
chmod 755 dist/cloud-update_linux_amd64_v1/install.sh

# Start services
echo -e "${YELLOW}Starting Docker services...${NC}"
docker compose -f "$DOCKER_COMPOSE_FILE" down --volumes --remove-orphans 2>/dev/null || true
docker compose -f "$DOCKER_COMPOSE_FILE" build --no-cache

# Test each distribution
for DISTRO in $DISTRIBUTIONS; do
    echo -e "\n${YELLOW}Testing $DISTRO...${NC}"
    
    # Start the container
    docker compose -f "$DOCKER_COMPOSE_FILE" up -d "$DISTRO"
    
    # Wait for container to be ready
    echo "Waiting for $DISTRO to start..."
    sleep 5
    
    # Copy install script to container
    docker cp scripts/install.sh "cloud-update-$DISTRO:/tmp/install.sh"
    
    # Run installation test
    echo "Running installation test on $DISTRO..."
    if docker exec "cloud-update-$DISTRO" bash -c "cd /app && chmod +x /tmp/install.sh && /tmp/install.sh --test"; then
        echo -e "${GREEN}✓ $DISTRO: Installation test passed${NC}"
        
        # Test HTTP endpoints
        echo "Testing HTTP endpoints on $DISTRO..."
        
        # Get port based on distro
        case "$DISTRO" in
            alpine) PORT=9991 ;;
            ubuntu) PORT=9992 ;;
            debian) PORT=9993 ;;
            rockylinux) PORT=9994 ;;
            fedora) PORT=9995 ;;
            arch) PORT=9996 ;;
            opensuse) PORT=9997 ;;
        esac
        
        # Test health endpoint
        if curl -f "http://localhost:$PORT/health" >/dev/null 2>&1; then
            echo -e "${GREEN}✓ $DISTRO: Health endpoint working${NC}"
        else
            echo -e "${RED}✗ $DISTRO: Health endpoint failed${NC}"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
        
        # Test webhook endpoint with HMAC
        PAYLOAD='{"action":"update","job_id":"test-123"}'
        SECRET="test-secret-key-for-e2e"
        SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" -binary | base64)
        
        RESPONSE=$(curl -s -X POST "http://localhost:$PORT/webhook" \
            -H "Content-Type: application/json" \
            -H "X-Cloud-Update-Signature: sha256=$SIGNATURE" \
            -d "$PAYLOAD" -w "\n%{http_code}" | tail -1)
        
        if [ "$RESPONSE" = "200" ]; then
            echo -e "${GREEN}✓ $DISTRO: Webhook endpoint working${NC}"
        else
            echo -e "${RED}✗ $DISTRO: Webhook endpoint failed (HTTP $RESPONSE)${NC}"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    else
        echo -e "${RED}✗ $DISTRO: Installation test failed${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    
    # Stop the container
    docker compose -f "$DOCKER_COMPOSE_FILE" stop "$DISTRO"
done

# Cleanup
echo -e "\n${YELLOW}Cleaning up...${NC}"
docker compose -f "$DOCKER_COMPOSE_FILE" down --volumes --remove-orphans

# Report results
echo -e "\n${GREEN}=== E2E Test Results ===${NC}"
if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}$FAILED_TESTS tests failed${NC}"
    exit 1
fi