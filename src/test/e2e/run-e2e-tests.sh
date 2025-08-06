#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_msg() {
    color=$1
    shift
    printf "${color}%s${NC}\n" "$*"
}

print_msg "$BLUE" "=== Cloud Update E2E Tests ==="
print_msg "$YELLOW" "Testing on multiple Linux distributions..."

# Clean up previous runs
print_msg "$YELLOW" "Cleaning up previous containers..."
docker compose -f test/e2e/docker-compose.yml down --remove-orphans || true

# Build and start services
print_msg "$GREEN" "Building Docker images..."
docker compose -f test/e2e/docker-compose.yml build

print_msg "$GREEN" "Starting services..."
docker compose -f test/e2e/docker-compose.yml up -d alpine ubuntu debian rockylinux fedora arch opensuse

# Wait for services to be healthy
print_msg "$YELLOW" "Waiting for services to be healthy..."
TIMEOUT=60
ELAPSED=0

while [ $ELAPSED -lt $TIMEOUT ]; do
    ALL_HEALTHY=true
    
    for DISTRO in alpine ubuntu debian rockylinux fedora arch opensuse; do
        HEALTH=$(docker inspect cloud-update-$DISTRO --format='{{.State.Health.Status}}' 2>/dev/null || echo "not-found")
        if [ "$HEALTH" != "healthy" ]; then
            ALL_HEALTHY=false
            break
        fi
    done
    
    if [ "$ALL_HEALTHY" = "true" ]; then
        print_msg "$GREEN" "All services are healthy!"
        break
    fi
    
    printf "."
    sleep 2
    ELAPSED=$((ELAPSED + 2))
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    print_msg "$RED" "Timeout waiting for services to be healthy"
    docker compose -f test/e2e/docker-compose.yml logs
    docker compose -f test/e2e/docker-compose.yml down
    exit 1
fi

# Run tests for each distribution
FAILED=0

print_msg "$BLUE" "\n=== Testing Alpine Linux ==="
if E2E_BASE_URL=http://localhost:9991 E2E_SECRET=test-secret-key-for-e2e go test -v ./test/e2e/...; then
    print_msg "$GREEN" "✓ Alpine tests passed"
else
    print_msg "$RED" "✗ Alpine tests failed"
    FAILED=$((FAILED + 1))
fi

print_msg "$BLUE" "\n=== Testing Ubuntu ==="
if E2E_BASE_URL=http://localhost:9992 E2E_SECRET=test-secret-key-for-e2e go test -v ./test/e2e/...; then
    print_msg "$GREEN" "✓ Ubuntu tests passed"
else
    print_msg "$RED" "✗ Ubuntu tests failed"
    FAILED=$((FAILED + 1))
fi

print_msg "$BLUE" "\n=== Testing Debian ==="
if E2E_BASE_URL=http://localhost:9993 E2E_SECRET=test-secret-key-for-e2e go test -v ./test/e2e/...; then
    print_msg "$GREEN" "✓ Debian tests passed"
else
    print_msg "$RED" "✗ Debian tests failed"
    FAILED=$((FAILED + 1))
fi

print_msg "$BLUE" "\n=== Testing Rocky Linux ==="
if E2E_BASE_URL=http://localhost:9994 E2E_SECRET=test-secret-key-for-e2e go test -v ./test/e2e/...; then
    print_msg "$GREEN" "✓ Rocky Linux tests passed"
else
    print_msg "$RED" "✗ Rocky Linux tests failed"
    FAILED=$((FAILED + 1))
fi

print_msg "$BLUE" "\n=== Testing Fedora ==="
if E2E_BASE_URL=http://localhost:9995 E2E_SECRET=test-secret-key-for-e2e go test -v ./test/e2e/...; then
    print_msg "$GREEN" "✓ Fedora tests passed"
else
    print_msg "$RED" "✗ Fedora tests failed"
    FAILED=$((FAILED + 1))
fi

print_msg "$BLUE" "\n=== Testing Arch Linux ==="
if E2E_BASE_URL=http://localhost:9996 E2E_SECRET=test-secret-key-for-e2e go test -v ./test/e2e/...; then
    print_msg "$GREEN" "✓ Arch Linux tests passed"
else
    print_msg "$RED" "✗ Arch Linux tests failed"
    FAILED=$((FAILED + 1))
fi

print_msg "$BLUE" "\n=== Testing openSUSE ==="
if E2E_BASE_URL=http://localhost:9997 E2E_SECRET=test-secret-key-for-e2e go test -v ./test/e2e/...; then
    print_msg "$GREEN" "✓ openSUSE tests passed"
else
    print_msg "$RED" "✗ openSUSE tests failed"
    FAILED=$((FAILED + 1))
fi

# Show logs if requested
if [ "$1" = "--logs" ]; then
    print_msg "$YELLOW" "\n=== Service Logs ==="
    docker compose -f test/e2e/docker-compose.yml logs
fi

# Clean up
print_msg "$YELLOW" "\nCleaning up..."
docker compose -f test/e2e/docker-compose.yml down

# Summary
print_msg "$BLUE" "\n=== Test Summary ==="
if [ $FAILED -eq 0 ]; then
    print_msg "$GREEN" "All E2E tests passed successfully! ✓"
    exit 0
else
    print_msg "$RED" "$FAILED distribution(s) failed tests ✗"
    exit 1
fi