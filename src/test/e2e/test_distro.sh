#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check argument
if [ $# -ne 1 ]; then
    echo "Usage: $0 <alpine|ubuntu|debian>"
    exit 1
fi

DISTRO=$1

# Set default environment variables if not already set
export E2E_SECRET="${E2E_SECRET:-test-secret-key-for-e2e-testing-purposes-only}"
export CLOUD_UPDATE_SECRET="${CLOUD_UPDATE_SECRET:-test-secret-key-for-e2e-testing-purposes-only}"
export CLOUD_UPDATE_LOG_LEVEL="${CLOUD_UPDATE_LOG_LEVEL:-debug}"

# Test configuration
COMPOSE_FILE="src/test/e2e/docker-compose.yml"
case $DISTRO in
    alpine)
        CONTAINER="cloud-update-alpine"
        PORT="${E2E_PORT_ALPINE:-8081}"
        ;;
    ubuntu)
        CONTAINER="cloud-update-ubuntu"
        PORT="${E2E_PORT_UBUNTU:-8082}"
        ;;
    debian)
        CONTAINER="cloud-update-debian"
        PORT="${E2E_PORT_DEBIAN:-8083}"
        ;;
    *)
        echo -e "${RED}Invalid distribution: $DISTRO${NC}"
        echo "Use: alpine, ubuntu, or debian"
        exit 1
        ;;
esac

# Function to test health endpoint
test_health_endpoint() {
    echo -e "${YELLOW}Testing health endpoint on port $PORT...${NC}"
    
    for i in {1..30}; do
        if curl -sf "http://localhost:$PORT/health" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Health endpoint responding${NC}"
            return 0
        fi
        echo -n "."
        sleep 1
    done
    
    echo ""
    echo -e "${RED}✗ Health endpoint not responding${NC}"
    return 1
}

# Function to test webhook endpoint
test_webhook_endpoint() {
    echo -e "${YELLOW}Testing webhook endpoint...${NC}"
    
    # Include timestamp in payload
    TIMESTAMP=$(date +%s)
    PAYLOAD='{"action":"update","timestamp":'$TIMESTAMP'}'
    SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "${E2E_SECRET:-test-secret-key-for-e2e-testing-purposes-only}" | cut -d' ' -f2)
    
    RESPONSE=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -H "X-Cloud-Update-Signature: sha256=$SIGNATURE" \
        -d "$PAYLOAD" \
        "http://localhost:$PORT/webhook" 2>&1) || true
    
    if echo "$RESPONSE" | grep -q "accepted\|success\|ok"; then
        echo -e "${GREEN}✓ Webhook endpoint working${NC}"
        return 0
    else
        echo -e "${RED}✗ Webhook endpoint failed${NC}"
        echo "Response: $RESPONSE"
        return 1
    fi
}

# Main test flow
main() {
    echo -e "${GREEN}=== Testing $DISTRO ===${NC}"
    
    # Clean up existing container
    echo -e "${YELLOW}Cleaning up existing container...${NC}"
    DOCKER_COMPOSE_FILE="${COMPOSE_FILE}"
    docker compose -f $DOCKER_COMPOSE_FILE stop $DISTRO 2>/dev/null || true
    docker compose -f $DOCKER_COMPOSE_FILE rm -f $DISTRO 2>/dev/null || true
    
    # Build Linux binary if needed
    if [ ! -f "bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update" ]; then
        echo -e "${YELLOW}Building Linux binary with Bazel...${NC}"
        bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 //src/cmd/cloud-update:cloud-update
    fi
    
    # Create dist directory and copy binary
    mkdir -p dist/cloud-update_linux_amd64_v1
    cp -f bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update dist/cloud-update_linux_amd64_v1/cloud-update
    chmod 755 dist/cloud-update_linux_amd64_v1/cloud-update
    
    # Build and start container
    echo -e "${YELLOW}Building and starting $DISTRO container...${NC}"
    docker compose -f $COMPOSE_FILE up -d --build $DISTRO
    
    # Wait for container to initialize
    echo -e "${YELLOW}Waiting for service to start...${NC}"
    sleep 5
    
    # Check if container is running
    if ! docker ps | grep -q $CONTAINER; then
        echo -e "${RED}✗ Container $CONTAINER is not running${NC}"
        echo -e "${YELLOW}Container logs:${NC}"
        docker logs $CONTAINER 2>&1 | tail -20
        docker compose -f $COMPOSE_FILE stop $DISTRO
        exit 1
    fi
    
    echo -e "${GREEN}✓ Container is running${NC}"
    
    # Check if the service is installed correctly
    echo -e "${YELLOW}Checking installation...${NC}"
    if docker exec $CONTAINER test -f /opt/cloud-update/cloud-update; then
        echo -e "${GREEN}✓ Cloud-update binary installed${NC}"
    else
        echo -e "${RED}✗ Cloud-update binary not found${NC}"
        docker logs $CONTAINER 2>&1 | tail -20
        docker compose -f $COMPOSE_FILE stop $DISTRO
        exit 1
    fi
    
    # Test health endpoint
    if ! test_health_endpoint; then
        echo -e "${YELLOW}Container logs:${NC}"
        docker logs $CONTAINER 2>&1 | tail -20
        docker compose -f $COMPOSE_FILE stop $DISTRO
        exit 1
    fi
    
    # Test webhook endpoint
    if ! test_webhook_endpoint; then
        docker compose -f $COMPOSE_FILE stop $DISTRO
        exit 1
    fi
    
    # Clean up
    echo -e "${YELLOW}Cleaning up...${NC}"
    docker compose -f $COMPOSE_FILE stop $DISTRO
    docker compose -f $COMPOSE_FILE rm -f $DISTRO
    
    echo -e "${GREEN}✓ All tests passed for $DISTRO${NC}"
}

# Run main function
main "$@"