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

# Stop and remove service
remove_service() {
    INIT_SYSTEM=$(detect_init_system)
    
    print_msg "$GREEN" "Detected init system: $INIT_SYSTEM"
    
    case "$INIT_SYSTEM" in
        systemd)
            print_msg "$GREEN" "Removing systemd service..."
            systemctl stop cloud-update 2>/dev/null || true
            systemctl disable cloud-update 2>/dev/null || true
            rm -f /etc/systemd/system/cloud-update.service
            systemctl daemon-reload
            print_msg "$GREEN" "Systemd service removed"
            ;;
            
        openrc)
            print_msg "$GREEN" "Removing OpenRC service..."
            rc-service cloud-update stop 2>/dev/null || true
            rc-update del cloud-update 2>/dev/null || true
            rm -f /etc/init.d/cloud-update
            print_msg "$GREEN" "OpenRC service removed"
            ;;
            
        sysvinit)
            print_msg "$GREEN" "Removing SysVinit service..."
            service cloud-update stop 2>/dev/null || true
            update-rc.d -f cloud-update remove 2>/dev/null || true
            rm -f /etc/init.d/cloud-update
            print_msg "$GREEN" "SysVinit service removed"
            ;;
            
        *)
            print_msg "$YELLOW" "Warning: Unknown init system"
            ;;
    esac
}

# Remove files
remove_files() {
    print_msg "$GREEN" "Removing installation files..."
    
    # Remove binary
    if [ -d "$INSTALL_DIR" ]; then
        rm -rf "$INSTALL_DIR"
        print_msg "$GREEN" "Removed $INSTALL_DIR"
    fi
    
    # Ask about configuration
    if [ -d "$CONFIG_DIR" ]; then
        printf "${YELLOW}Remove configuration directory $CONFIG_DIR? (y/N): ${NC}"
        read -r response
        case "$response" in
            [yY][eE][sS]|[yY])
                rm -rf "$CONFIG_DIR"
                print_msg "$GREEN" "Removed $CONFIG_DIR"
                ;;
            *)
                print_msg "$YELLOW" "Configuration directory kept at $CONFIG_DIR"
                ;;
        esac
    fi
}

# Main uninstallation
main() {
    print_msg "$GREEN" "=== Cloud Update Uninstallation Script ==="
    
    check_root
    
    printf "${YELLOW}Are you sure you want to uninstall cloud-update? (y/N): ${NC}"
    read -r response
    case "$response" in
        [yY][eE][sS]|[yY])
            remove_service
            remove_files
            print_msg "$GREEN" "\n=== Uninstallation Complete ==="
            ;;
        *)
            print_msg "$YELLOW" "Uninstallation cancelled"
            exit 0
            ;;
    esac
}

# Run main function
main "$@"