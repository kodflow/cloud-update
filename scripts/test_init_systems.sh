#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Load environment variables
if [ -f .env.test ]; then
    export $(cat .env.test | grep -v '^#' | xargs)
fi

# Test configuration
COMPOSE_FILE="src/test/e2e/docker-compose-init.yml"
SYSTEMS=("alpine-openrc" "fedora-systemd" "debian-sysvinit")
PORTS=("9991" "9992" "9993")

# Function to check if service is running
check_service_running() {
    local container=$1
    local init_system=$2
    local check_cmd=""
    
    case $init_system in
        openrc)
            check_cmd="rc-service cloud-update status"
            ;;
        systemd)
            check_cmd="systemctl is-active cloud-update"
            ;;
        sysvinit)
            check_cmd="service cloud-update status"
            ;;
    esac
    
    echo -e "${YELLOW}Checking if service is running in $container...${NC}"
    if docker exec $container sh -c "$check_cmd" 2>/dev/null | grep -q "running\|active"; then
        echo -e "${GREEN}✓ Service is running${NC}"
        return 0
    else
        echo -e "${RED}✗ Service is not running${NC}"
        docker exec $container sh -c "$check_cmd" 2>&1 || true
        return 1
    fi
}

# Function to test health endpoint
test_health_endpoint() {
    local port=$1
    local system=$2
    
    echo -e "${YELLOW}Testing health endpoint for $system on port $port...${NC}"
    
    # Wait for service to be ready
    for i in {1..30}; do
        if curl -sf "http://localhost:$port/health" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Health endpoint responding${NC}"
            return 0
        fi
        echo -n "."
        sleep 1
    done
    
    echo -e "${RED}✗ Health endpoint not responding after 30 seconds${NC}"
    return 1
}

# Function to test webhook endpoint
test_webhook_endpoint() {
    local port=$1
    local system=$2
    
    echo -e "${YELLOW}Testing webhook endpoint for $system...${NC}"
    
    # Prepare test payload
    PAYLOAD='{"action":"update","job_id":"test-123"}'
    SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$E2E_SECRET" | cut -d' ' -f2)
    
    # Send webhook request
    RESPONSE=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -H "X-Signature: $SIGNATURE" \
        -d "$PAYLOAD" \
        "http://localhost:$port/webhook" 2>&1) || true
    
    if echo "$RESPONSE" | grep -q "success\|ok\|200"; then
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
    echo -e "${GREEN}=== Testing Init Systems ===${NC}"
    
    # Clean up any existing containers
    echo -e "${YELLOW}Cleaning up existing containers...${NC}"
    docker compose -f $COMPOSE_FILE down --volumes --remove-orphans 2>/dev/null || true
    
    # Build Linux binary with Bazel if not exists
    if [ ! -f "bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update" ]; then
        echo -e "${YELLOW}Building Linux binary with Bazel...${NC}"
        bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 //src/cmd/cloud-update:cloud-update
    fi
    
    # Create dist directory and copy binary
    mkdir -p dist/cloud-update_linux_amd64_v1
    cp bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update dist/cloud-update_linux_amd64_v1/cloud-update
    
    # Build and start all containers
    echo -e "${YELLOW}Building and starting containers...${NC}"
    docker compose -f $COMPOSE_FILE up -d --build
    
    # Wait for containers to initialize
    echo -e "${YELLOW}Waiting for containers to initialize...${NC}"
    sleep 10
    
    # Test each init system
    FAILED=0
    for i in ${!SYSTEMS[@]}; do
        SYSTEM=${SYSTEMS[$i]}
        PORT=${PORTS[$i]}
        INIT_TYPE=$(echo $SYSTEM | cut -d'-' -f2)
        CONTAINER="cloud-update-$INIT_TYPE"
        
        echo ""
        echo -e "${GREEN}=== Testing $SYSTEM ===${NC}"
        
        # Check if container is running
        if ! docker ps | grep -q $CONTAINER; then
            echo -e "${RED}✗ Container $CONTAINER is not running${NC}"
            docker logs $CONTAINER 2>&1 | tail -20
            FAILED=1
            continue
        fi
        
        # Check if service is running
        if ! check_service_running $CONTAINER $INIT_TYPE; then
            FAILED=1
            continue
        fi
        
        # Test health endpoint
        if ! test_health_endpoint $PORT $SYSTEM; then
            FAILED=1
            continue
        fi
        
        # Test webhook endpoint
        if ! test_webhook_endpoint $PORT $SYSTEM; then
            FAILED=1
            continue
        fi
        
        echo -e "${GREEN}✓ All tests passed for $SYSTEM${NC}"
    done
    
    # Clean up
    echo ""
    echo -e "${YELLOW}Cleaning up...${NC}"
    docker compose -f $COMPOSE_FILE down --volumes --remove-orphans
    
    # Report results
    echo ""
    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}=== All init system tests passed! ===${NC}"
        exit 0
    else
        echo -e "${RED}=== Some init system tests failed ===${NC}"
        exit 1
    fi
}

# Run main function
main "$@"