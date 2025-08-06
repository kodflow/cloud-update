#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🔧 Preparing E2E tests...${NC}"

# Detect Docker architecture
DOCKER_ARCH=$(docker version --format '{{.Server.Arch}}' 2>/dev/null || echo "amd64")
echo "Docker architecture detected: ${DOCKER_ARCH}"

# Map Docker arch to our naming convention
if [ "$DOCKER_ARCH" = "x86_64" ]; then
  DOCKER_ARCH="amd64"
elif [ "$DOCKER_ARCH" = "aarch64" ]; then
  DOCKER_ARCH="arm64"
fi

# Binary path
BINARY_PATH="dist/cloud-update-linux-${DOCKER_ARCH}"

# Check if we need to build
if [ ! -f "$BINARY_PATH" ]; then
  echo -e "${YELLOW}Binary for linux/${DOCKER_ARCH} not found. Building...${NC}"
  
  # Build only the needed binary
  bazel build --config=ci \
    --platforms=@io_bazel_rules_go//go/toolchain:linux_${DOCKER_ARCH} \
    //src/cmd/cloud-update:cloud-update 2>/dev/null || {
      echo -e "${RED}✗ Failed to build for linux/${DOCKER_ARCH}${NC}"
      exit 1
    }
  
  # Copy the binary
  mkdir -p dist
  cp bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update "$BINARY_PATH" 2>/dev/null || \
  cp bazel-bin/src/cmd/cloud-update/linux_${DOCKER_ARCH}_pure_stripped/cloud-update "$BINARY_PATH" 2>/dev/null || {
    echo -e "${RED}✗ Could not find binary${NC}"
    exit 1
  }
  
  chmod +x "$BINARY_PATH"
  echo -e "${GREEN}✓ Built ${BINARY_PATH}${NC}"
else
  echo -e "${GREEN}✓ Binary ${BINARY_PATH} already exists${NC}"
fi

# Create symlink for Docker to use
ln -sf "cloud-update-linux-${DOCKER_ARCH}" dist/cloud-update-docker
echo -e "${GREEN}✓ Created dist/cloud-update-docker -> cloud-update-linux-${DOCKER_ARCH}${NC}"

# Copy to test directory
rm -f src/test/e2e/cloud-update 2>/dev/null
cp "$BINARY_PATH" src/test/e2e/cloud-update
chmod +x src/test/e2e/cloud-update
echo -e "${GREEN}✓ Copied binary to src/test/e2e/cloud-update${NC}"

echo -e "${GREEN}✅ E2E tests are ready!${NC}"