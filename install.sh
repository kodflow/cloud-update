#!/bin/sh
# Cloud Update Installation Script
# POSIX-compliant script that works on all Linux distributions
# Usage: curl -sSL https://raw.githubusercontent.com/kodflow/cloud-update/main/install.sh | sudo sh

set -e

# Colors (if terminal supports it)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    NC=''
fi

# Configuration
GITHUB_REPO="kodflow/cloud-update"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="cloud-update"

# Functions
log_info() {
    printf "${GREEN}[INFO]${NC} %s\n" "$1"
}

log_warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

log_error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1"
    exit 1
}

# Detect architecture
detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        armv7l|armv6l|arm)
            echo "arm"
            ;;
        i686|i386)
            echo "386"
            ;;
        ppc64le)
            echo "ppc64le"
            ;;
        s390x)
            echo "s390x"
            ;;
        mips64le)
            echo "mips64le"
            ;;
        riscv64)
            echo "riscv64"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            ;;
    esac
}

# Check if running as root
check_root() {
    if [ "$(id -u)" != "0" ]; then
        log_error "This script must be run as root (use sudo)"
    fi
}

# Download binary
download_binary() {
    ARCH=$(detect_arch)
    BINARY_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/${BINARY_NAME}-linux-${ARCH}"
    
    log_info "Downloading ${BINARY_NAME} for linux/${ARCH}..."
    
    # Try wget first, then curl
    if command -v wget >/dev/null 2>&1; then
        wget -q -O "${INSTALL_DIR}/${BINARY_NAME}" "$BINARY_URL" || \
            log_error "Failed to download binary"
    elif command -v curl >/dev/null 2>&1; then
        curl -sSL -o "${INSTALL_DIR}/${BINARY_NAME}" "$BINARY_URL" || \
            log_error "Failed to download binary"
    else
        log_error "Neither wget nor curl found. Please install one of them."
    fi
    
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    log_info "Binary downloaded to ${INSTALL_DIR}/${BINARY_NAME}"
}

# Install service using the binary's built-in installer
install_service() {
    log_info "Installing service using built-in installer..."
    
    # Use the binary's setup command which handles all init systems
    if "${INSTALL_DIR}/${BINARY_NAME}" --setup; then
        log_info "Service installation completed successfully"
    else
        log_error "Service installation failed"
    fi
}

# Main installation
main() {
    log_info "🚀 Starting Cloud Update installation..."
    
    # Check prerequisites
    check_root
    
    # Download binary
    download_binary
    
    # Verify download
    if "${INSTALL_DIR}/${BINARY_NAME}" version >/dev/null 2>&1; then
        VERSION=$("${INSTALL_DIR}/${BINARY_NAME}" version 2>/dev/null || echo 'unknown')
        log_info "✅ Binary verified - Version: ${VERSION}"
    else
        log_error "Downloaded binary is not working correctly"
    fi
    
    # Install service using the binary's built-in setup
    install_service
    
    log_info "🎉 Installation complete!"
    log_info ""
    log_info "📍 Binary location: ${INSTALL_DIR}/${BINARY_NAME}"
    log_info "📄 Configuration: /etc/cloud-update/config.env"
    log_info ""
    log_info "🔧 Next steps:"
    log_info "1. Edit the secret: sudo nano /etc/cloud-update/config.env"
    log_info "2. Start service: sudo systemctl start cloud-update"
    log_info "3. Enable at boot: sudo systemctl enable cloud-update"
    log_info "4. Check status: sudo systemctl status cloud-update"
    log_info ""
    log_info "📖 Documentation: https://github.com/${GITHUB_REPO}#readme"
}

# Run main function
main "$@"