#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Platforms to build - Linux only for cloud-init usage
PLATFORMS=(
  "linux/amd64"     # x86_64 standard
  "linux/arm64"     # ARM 64-bit (AWS Graviton, RPi 4)
  "linux/386"       # x86 32-bit legacy systems
  "linux/arm"       # ARM 32-bit (RPi 3 and older)
  "linux/ppc64le"   # PowerPC 64-bit LE (IBM POWER)
  "linux/s390x"     # IBM Z mainframes
  "linux/mips64le"  # MIPS 64-bit LE
  "linux/riscv64"   # RISC-V 64-bit
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
  
  # Build with Bazel
  bazel build --config=ci \
    --platforms=@io_bazel_rules_go//go/toolchain:${OS}_${ARCH} \
    //src/cmd/cloud-update:cloud-update \
    --workspace_status_command="echo BUILD_VERSION ${VERSION}" 2>/dev/null || {
      echo -e "${RED}✗ Failed to build for ${OS}/${ARCH}${NC}"
      continue
    }
  
  # Copy the binary
  cp bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update "$OUTPUT" 2>/dev/null || \
  cp bazel-bin/src/cmd/cloud-update/${OS}_${ARCH}_pure_stripped/cloud-update "$OUTPUT" 2>/dev/null || {
    echo -e "${RED}✗ Could not find binary for ${OS}/${ARCH}${NC}"
    continue
  }
  
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

# Create main binary link (prefer amd64 for testing)
if [ -f "dist/cloud-update-linux-amd64" ]; then
  cp "dist/cloud-update-linux-amd64" ./cloud-update
  chmod +x ./cloud-update
  echo -e "${GREEN}✓ Created ./cloud-update for local testing${NC}"
fi

echo -e "${GREEN}✅ Build complete!${NC}"
echo ""
echo "Available binaries:"
ls -lh dist/cloud-update-* | awk '{print "  " $9 " (" $5 ")"}'