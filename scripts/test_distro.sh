#!/bin/bash
set -e

# Load environment variables
if [ -f .env.test ]; then
    export $(cat .env.test | grep -v '^#' | xargs)
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check arguments
if [ $# -ne 1 ]; then
    echo "Usage: $0 <distribution>"
    echo "Available distributions: alpine ubuntu debian rockylinux fedora arch opensuse"
    exit 1
fi

DISTRO=$1

# Get port for this distribution
PORT_VAR="E2E_PORT_$(echo $DISTRO | tr '[:lower:]' '[:upper:]')"
PORT=${!PORT_VAR}

if [ -z "$PORT" ]; then
    echo -e "${RED}Unknown distribution: $DISTRO${NC}"
    exit 1
fi

echo -e "${GREEN}=== Testing $DISTRO ===${NC}"

# Build Linux binary if not exists
if [ ! -f "dist/cloud-update_linux_amd64_v1/cloud-update" ]; then
    echo -e "${YELLOW}Building Linux binary with Bazel...${NC}"
    bazel build //src/cmd/cloud-update:cloud-update --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64
    
    # Prepare dist directory
    rm -rf dist/cloud-update_linux_amd64_v1
    mkdir -p dist/cloud-update_linux_amd64_v1
    
    BINARY_PATH=$(find bazel-bin/src/cmd/cloud-update -name "cloud-update" -type f | grep -E "linux.*amd64" | head -1)
    if [ -z "$BINARY_PATH" ]; then
        BINARY_PATH="bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update"
    fi
    
    cp "$BINARY_PATH" dist/cloud-update_linux_amd64_v1/cloud-update
    chmod 755 dist/cloud-update_linux_amd64_v1/cloud-update
fi

# Copy install script
cp scripts/install.sh dist/cloud-update_linux_amd64_v1/
chmod 755 dist/cloud-update_linux_amd64_v1/install.sh

# Start the specific container
echo -e "${YELLOW}Starting $DISTRO container...${NC}"
docker compose -f "$DOCKER_COMPOSE_FILE" up -d --build "$DISTRO"

# Wait for container to be ready
echo "Waiting for $DISTRO to start..."
for i in {1..30}; do
    if docker exec "cloud-update-$DISTRO" echo "Container ready" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done

# Copy install script to container
docker cp scripts/install.sh "cloud-update-$DISTRO:/tmp/install.sh"

# Run installation test
echo "Running installation test on $DISTRO..."
if docker exec "cloud-update-$DISTRO" bash -c "cd /app && chmod +x /tmp/install.sh && /tmp/install.sh --test"; then
    echo -e "${GREEN}✓ $DISTRO: Installation test passed${NC}"
else
    echo -e "${RED}✗ $DISTRO: Installation test failed${NC}"
    docker compose -f "$DOCKER_COMPOSE_FILE" stop "$DISTRO"
    exit 1
fi

# Test health endpoint
echo "Testing health endpoint on $DISTRO..."
HEALTH_URL="${E2E_BASE_URL_PREFIX}:${PORT}/health"
if curl -f "$HEALTH_URL" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ $DISTRO: Health endpoint working${NC}"
else
    echo -e "${RED}✗ $DISTRO: Health endpoint failed${NC}"
    docker compose -f "$DOCKER_COMPOSE_FILE" stop "$DISTRO"
    exit 1
fi

# Test webhook endpoint with correct HMAC
echo "Testing webhook endpoint on $DISTRO..."
WEBHOOK_URL="${E2E_BASE_URL_PREFIX}:${PORT}/webhook"
PAYLOAD='{"action":"update","job_id":"test-123"}'

# Generate correct HMAC signature (in hex format, not base64)
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$E2E_SECRET" | cut -d' ' -f2)

RESPONSE=$(curl -s -X POST "$WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -H "X-Cloud-Update-Signature: sha256=$SIGNATURE" \
    -d "$PAYLOAD" \
    -w "\n%{http_code}" | tail -1)

if [ "$RESPONSE" = "200" ] || [ "$RESPONSE" = "202" ]; then
    echo -e "${GREEN}✓ $DISTRO: Webhook endpoint working${NC}"
else
    echo -e "${RED}✗ $DISTRO: Webhook endpoint failed (HTTP $RESPONSE)${NC}"
    
    # Debug: Show the actual response
    echo "Debug - Testing with curl:"
    curl -v -X POST "$WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -H "X-Cloud-Update-Signature: sha256=$SIGNATURE" \
        -d "$PAYLOAD"
    
    docker compose -f "$DOCKER_COMPOSE_FILE" stop "$DISTRO"
    exit 1
fi

# Stop the container
echo -e "${YELLOW}Stopping $DISTRO container...${NC}"
docker compose -f "$DOCKER_COMPOSE_FILE" stop "$DISTRO"

echo -e "${GREEN}✓ All tests passed for $DISTRO${NC}"