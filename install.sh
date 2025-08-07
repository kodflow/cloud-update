#!/bin/sh
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Installation directories
INSTALL_DIR="/opt/cloud-update"
CONFIG_DIR="/etc/cloud-update"
BINARY_NAME="cloud-update"

# Detect init system
detect_init_system() {
    if [ -f /sbin/openrc ]; then
        echo "openrc"
    elif [ -d /run/systemd/system ]; then
        echo "systemd"
    elif [ -f /sbin/init ] && [ ! -L /sbin/init ]; then
        echo "sysvinit"
    else
        echo "unknown"
    fi
}

# Detect distribution
detect_distro() {
    if [ -f /etc/alpine-release ]; then
        echo "alpine"
    elif [ -f /etc/os-release ]; then
        . /etc/os-release
        case "${ID:-unknown}" in
            ubuntu) echo "ubuntu" ;;
            debian) echo "debian" ;;
            rhel|centos|fedora) echo "rhel" ;;
            suse|opensuse*) echo "suse" ;;
            arch) echo "arch" ;;
            *) echo "unknown" ;;
        esac
    else
        echo "unknown"
    fi
}

# Print colored message
print_msg() {
    color=$1
    shift
    printf "${color}%s${NC}\n" "$*"
}

# Check if running as root
check_root() {
    if [ "$(id -u)" != "0" ]; then
        print_msg "$RED" "Error: This script must be run as root"
        exit 1
    fi
}

# Build the binary
build_binary() {
    print_msg "$GREEN" "Building $BINARY_NAME..."
    
    if ! command -v go >/dev/null 2>&1; then
        print_msg "$RED" "Error: Go is not installed"
        print_msg "$YELLOW" "Please install Go first: https://golang.org/dl/"
        exit 1
    fi
    
    # Check if we're in the right directory
    if [ ! -f "src/cmd/cloud-update/main.go" ]; then
        print_msg "$RED" "Error: Cannot find src/cmd/cloud-update/main.go"
        print_msg "$YELLOW" "Please run this script from the repository root"
        exit 1
    fi
    
    go build -o "$BINARY_NAME" ./src/cmd/cloud-update
    
    if [ ! -f "$BINARY_NAME" ]; then
        print_msg "$RED" "Error: Build failed"
        exit 1
    fi
    
    print_msg "$GREEN" "Build successful"
}

# Install binary and configuration
install_files() {
    print_msg "$GREEN" "Installing files..."
    
    # Create directories with secure permissions
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    chmod 755 "$INSTALL_DIR"
    chmod 700 "$CONFIG_DIR"
    
    # Copy binary
    cp "$BINARY_NAME" "$INSTALL_DIR/"
    chmod 755 "$INSTALL_DIR/$BINARY_NAME"
    
    # Create configuration file if it doesn't exist
    if [ ! -f "$CONFIG_DIR/config.env" ]; then
        print_msg "$GREEN" "Creating configuration file..."
        GENERATED_DATE=$(date)
        GENERATED_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | base64 | tr -d '\n')
        cat > "$CONFIG_DIR/config.env" << EOF
# Cloud Update Configuration
# Generated on: $GENERATED_DATE

# Port on which the HTTP server will listen
CLOUD_UPDATE_PORT=9999

# Secret key for webhook signature validation (HMAC SHA256)
# IMPORTANT: This is a generated secret key
CLOUD_UPDATE_SECRET=$GENERATED_SECRET

# Log level (debug, info, warn, error)
CLOUD_UPDATE_LOG_LEVEL=info
EOF
        # Set secure permissions (readable only by owner)
        chmod 600 "$CONFIG_DIR/config.env"
        chown root:root "$CONFIG_DIR/config.env"
        
        print_msg "$YELLOW" "Configuration file created at $CONFIG_DIR/config.env"
        print_msg "$YELLOW" "IMPORTANT: Edit it and set your CLOUD_UPDATE_SECRET"
        print_msg "$RED" "File permissions set to 600 (owner read/write only)"
    else
        print_msg "$YELLOW" "Configuration file already exists at $CONFIG_DIR/config.env"
        # Ensure existing file has secure permissions
        chmod 600 "$CONFIG_DIR/config.env"
    fi
}

# Install init script based on system
install_init_script() {
    INIT_SYSTEM=$(detect_init_system)
    
    print_msg "$GREEN" "Detected init system: $INIT_SYSTEM"
    
    case "$INIT_SYSTEM" in
        systemd)
            print_msg "$GREEN" "Installing systemd service..."
            if [ ! -f "src/init/systemd/cloud-update.service" ]; then
                print_msg "$RED" "Error: Cannot find src/init/systemd/cloud-update.service"
                print_msg "$YELLOW" "Please run this script from the repository root"
                exit 1
            fi
            cp src/init/systemd/cloud-update.service /etc/systemd/system/
            systemctl daemon-reload
            print_msg "$GREEN" "Systemd service installed"
            print_msg "$YELLOW" "To enable: systemctl enable cloud-update"
            print_msg "$YELLOW" "To start: systemctl start cloud-update"
            ;;
            
        openrc)
            print_msg "$GREEN" "Installing OpenRC service..."
            if [ -f "src/init/openrc/cloud-update" ]; then
                cp src/init/openrc/cloud-update /etc/init.d/
                chmod 755 /etc/init.d/cloud-update
                print_msg "$GREEN" "OpenRC service installed"
                print_msg "$YELLOW" "To enable: rc-update add cloud-update default"
                print_msg "$YELLOW" "To start: rc-service cloud-update start"
            else
                print_msg "$RED" "Error: OpenRC service file not found"
                exit 1
            fi
            ;;
            
        sysvinit)
            print_msg "$GREEN" "Installing SysVinit service..."
            if [ -f "src/init/sysvinit/cloud-update" ]; then
                cp src/init/sysvinit/cloud-update /etc/init.d/
                chmod 755 /etc/init.d/cloud-update
                update-rc.d cloud-update defaults 2>/dev/null || true
                print_msg "$GREEN" "SysVinit service installed"
                print_msg "$YELLOW" "To start: service cloud-update start"
            else
                print_msg "$RED" "Error: SysVinit service file not found"
                exit 1
            fi
            ;;
            
        *)
            print_msg "$YELLOW" "Warning: Unknown init system"
            print_msg "$YELLOW" "You'll need to manually configure the service to start on boot"
            ;;
    esac
}

# Generate secret key
generate_secret() {
    if command -v openssl >/dev/null 2>&1; then
        SECRET=$(openssl rand -hex 32)
    elif command -v dd >/dev/null 2>&1; then
        SECRET=$(dd if=/dev/urandom bs=32 count=1 2>/dev/null | hexdump -e '32/1 "%02x"')
    else
        print_msg "$YELLOW" "Warning: Cannot generate secret automatically"
        print_msg "$YELLOW" "Please manually set CLOUD_UPDATE_SECRET in $CONFIG_DIR/config.env"
        return
    fi
    
    print_msg "$GREEN" "Generated secret key: $SECRET"
    print_msg "$YELLOW" "Add this to $CONFIG_DIR/config.env:"
    print_msg "$YELLOW" "CLOUD_UPDATE_SECRET=$SECRET"
}

# Main installation
main() {
    print_msg "$GREEN" "=== Cloud Update Installation Script ==="
    
    check_root
    
    DISTRO=$(detect_distro)
    print_msg "$GREEN" "Detected distribution: $DISTRO"
    
    build_binary
    install_files
    install_init_script
    
    print_msg "$GREEN" "\n=== Installation Complete ==="
    print_msg "$YELLOW" "\nNext steps:"
    print_msg "$YELLOW" "1. Generate a secret key:"
    generate_secret
    print_msg "$YELLOW" "\n2. Edit configuration:"
    print_msg "$YELLOW" "   nano $CONFIG_DIR/config.env"
    print_msg "$YELLOW" "\n3. Start the service (see commands above)"
    print_msg "$GREEN" "\n=== Done ==="
}

# Run main function
main "$@"