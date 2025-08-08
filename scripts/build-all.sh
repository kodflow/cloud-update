#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Platforms to build
PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

# Get version
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

echo -e "${YELLOW}🔨 Building cloud-update for all platforms...${NC}"
echo "Version: ${VERSION}"

# Create dist directory
mkdir -p dist

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
  OS=$(echo $platform | cut -d/ -f1)
  ARCH=$(echo $platform | cut -d/ -f2)
  
  echo -e "${YELLOW}Building for ${OS}/${ARCH}...${NC}"
  
  # Set output name
  OUTPUT="dist/cloud-update-${OS}-${ARCH}"
  if [ "$OS" = "windows" ]; then
    OUTPUT="${OUTPUT}.exe"
  fi
  
  # Build with Bazel
  bazel build --config=ci \
    --platforms=@io_bazel_rules_go//go/toolchain:${OS}_${ARCH} \
    //src/cmd/cloud-update:cloud-update \
    --workspace_status_command="echo BUILD_VERSION ${VERSION}" 2>/dev/null || {
      echo -e "${RED}✗ Failed to build for ${OS}/${ARCH}${NC}"
      continue
    }
  
  # Copy the binary
  if [ "$OS" = "windows" ]; then
    cp bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update.exe "$OUTPUT" 2>/dev/null || \
    cp bazel-bin/src/cmd/cloud-update/windows_${ARCH}_pure_stripped/cloud-update.exe "$OUTPUT" 2>/dev/null || {
      echo -e "${RED}✗ Could not find binary for ${OS}/${ARCH}${NC}"
      continue
    }
  else
    cp bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update "$OUTPUT" 2>/dev/null || \
    cp bazel-bin/src/cmd/cloud-update/${OS}_${ARCH}_pure_stripped/cloud-update "$OUTPUT" 2>/dev/null || {
      echo -e "${RED}✗ Could not find binary for ${OS}/${ARCH}${NC}"
      continue
    }
  fi
  
  chmod +x "$OUTPUT" 2>/dev/null
  echo -e "${GREEN}✓ Built ${OUTPUT}${NC}"
done

# Create symlinks for common use cases
echo -e "${YELLOW}Creating convenience symlinks...${NC}"

# Linux binary for Docker (prefer amd64 for compatibility)
if [ -f "dist/cloud-update-linux-amd64" ]; then
  ln -sf cloud-update-linux-amd64 dist/cloud-update-linux
  echo -e "${GREEN}✓ Created dist/cloud-update-linux -> cloud-update-linux-amd64${NC}"
elif [ -f "dist/cloud-update-linux-arm64" ]; then
  ln -sf cloud-update-linux-arm64 dist/cloud-update-linux
  echo -e "${GREEN}✓ Created dist/cloud-update-linux -> cloud-update-linux-arm64${NC}"
fi

# Local binary based on current OS/ARCH
CURRENT_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
CURRENT_ARCH=$(uname -m)

# Map Darwin to darwin and x86_64/aarch64 to amd64/arm64
if [ "$CURRENT_OS" = "darwin" ]; then
  CURRENT_OS="darwin"
fi

if [ "$CURRENT_ARCH" = "x86_64" ]; then
  CURRENT_ARCH="amd64"
elif [ "$CURRENT_ARCH" = "aarch64" ] || [ "$CURRENT_ARCH" = "arm64" ]; then
  CURRENT_ARCH="arm64"
fi

LOCAL_BINARY="dist/cloud-update-${CURRENT_OS}-${CURRENT_ARCH}"
if [ -f "$LOCAL_BINARY" ]; then
  cp "$LOCAL_BINARY" ./cloud-update
  chmod +x ./cloud-update
  echo -e "${GREEN}✓ Created ./cloud-update for local use (${CURRENT_OS}/${CURRENT_ARCH})${NC}"
fi

echo -e "${GREEN}✅ Build complete!${NC}"
echo ""
echo "Available binaries:"
ls -lh dist/cloud-update-* | awk '{print "  " $9 " (" $5 ")"}'