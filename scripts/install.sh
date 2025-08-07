#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="cloud-update"
INSTALL_DIR="/opt/cloud-update"
CONFIG_DIR="/etc/cloud-update"
SERVICE_NAME="cloud-update"

# Detect OS and distribution
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=${ID:-unknown}
        VER=${VERSION_ID:-unknown}
    elif [ -f /etc/alpine-release ]; then
        OS="alpine"
        VER=$(cat /etc/alpine-release)
    elif [ -f /etc/debian_version ]; then
        OS="debian"
        VER=$(cat /etc/debian_version)
    elif [ -f /etc/redhat-release ]; then
        OS="rhel"
        VER=$(rpm -q --qf "%{VERSION}" $(rpm -q --whatprovides redhat-release))
    else
        echo -e "${RED}Cannot detect OS${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}Detected OS: $OS $VER${NC}"
}

# Detect if running in container
is_container() {
    # Check for Docker
    if [ -f /.dockerenv ]; then
        return 0
    fi
    
    # Check for cgroup indicators
    if grep -q 'docker\|containerd\|kubepods' /proc/1/cgroup 2>/dev/null; then
        return 0
    fi
    
    # Check for container specific files
    if [ -f /run/.containerenv ]; then
        return 0
    fi
    
    return 1
}

# Detect init system
detect_init() {
    # First check if we're in a container
    if is_container; then
        IN_CONTAINER=true
        echo -e "${YELLOW}Running in container environment${NC}"
    else
        IN_CONTAINER=false
    fi
    
    # Detect based on available init systems
    if [ -f /sbin/openrc ] || [ -f /usr/sbin/openrc ]; then
        INIT_SYSTEM="openrc"
    elif command -v systemctl >/dev/null 2>&1; then
        # If systemctl exists, we can use systemd (even in containers)
        INIT_SYSTEM="systemd"
        if [ "$IN_CONTAINER" = true ] && ! systemctl is-system-running >/dev/null 2>&1; then
            echo -e "${YELLOW}Systemd installed but not running (container mode)${NC}"
        fi
    elif [ -f /etc/init.d/rc ] || command -v update-rc.d >/dev/null 2>&1; then
        INIT_SYSTEM="sysvinit"
    else
        echo -e "${YELLOW}Warning: Could not detect init system${NC}"
        INIT_SYSTEM="none"
    fi
    
    echo -e "${GREEN}Detected init system: $INIT_SYSTEM${NC}"
}

# Install binary
install_binary() {
    echo -e "${GREEN}Installing $BINARY_NAME...${NC}"
    
    # Create directories
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    
    # If in test mode and binary already exists, skip copy
    if [ "$1" = "--test" ] && [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        echo -e "${YELLOW}Binary already installed, skipping copy${NC}"
    else
        # Copy binary
        if [ -f "./$BINARY_NAME" ]; then
            cp -f "./$BINARY_NAME" "$INSTALL_DIR/"
            chmod 755 "$INSTALL_DIR/$BINARY_NAME"
        elif [ -f "/app/$BINARY_NAME" ]; then
            cp -f "/app/$BINARY_NAME" "$INSTALL_DIR/"
            chmod 755 "$INSTALL_DIR/$BINARY_NAME"
        else
            echo -e "${RED}Binary not found!${NC}"
            exit 1
        fi
    fi
    
    # Create default config if not exists
    if [ ! -f "$CONFIG_DIR/config.env" ]; then
        cat > "$CONFIG_DIR/config.env" <<EOF
CLOUD_UPDATE_PORT=9999
CLOUD_UPDATE_SECRET=${CLOUD_UPDATE_SECRET:-$(openssl rand -hex 32 2>/dev/null || echo "PLEASE_SET_A_SECURE_SECRET")}
CLOUD_UPDATE_LOG_LEVEL=info
EOF
        chmod 600 "$CONFIG_DIR/config.env"
    fi
    
    echo -e "${GREEN}Binary installed to $INSTALL_DIR/$BINARY_NAME${NC}"
}

# Install systemd service
install_systemd() {
    echo -e "${GREEN}Installing systemd service...${NC}"
    
    cat > "/etc/systemd/system/$SERVICE_NAME.service" <<EOF
[Unit]
Description=Cloud Update Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$CONFIG_DIR/config.env
ExecStart=$INSTALL_DIR/$BINARY_NAME
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
    
    # Try to reload and enable if systemd is running
    if systemctl is-system-running >/dev/null 2>&1; then
        systemctl daemon-reload
        systemctl enable "$SERVICE_NAME"
        echo -e "${GREEN}Service enabled in systemd${NC}"
    elif [ "$IN_CONTAINER" = true ]; then
        # In container, create symlink manually for systemd
        ln -sf "/etc/systemd/system/$SERVICE_NAME.service" \
            "/etc/systemd/system/multi-user.target.wants/$SERVICE_NAME.service" 2>/dev/null || true
        echo -e "${YELLOW}Service configured for container systemd${NC}"
    else
        echo -e "${YELLOW}Systemd not running, service file created but not enabled${NC}"
    fi
    echo -e "${GREEN}Systemd service installed${NC}"
}

# Install OpenRC service
install_openrc() {
    echo -e "${GREEN}Installing OpenRC service...${NC}"
    
    cat > "/etc/init.d/$SERVICE_NAME" <<'EOF'
#!/sbin/openrc-run

name="Cloud Update Service"
command="/opt/cloud-update/cloud-update"
command_background=true
pidfile="/run/${RC_SVCNAME}.pid"
start_stop_daemon_args="--env-file /etc/cloud-update/config.env"

depend() {
    need net
}

start_pre() {
    checkpath --directory --owner root:root --mode 0755 /run
}
EOF
    
    chmod 755 "/etc/init.d/$SERVICE_NAME"
    rc-update add "$SERVICE_NAME" default
    echo -e "${GREEN}OpenRC service installed${NC}"
}

# Install SysVInit service
install_sysvinit() {
    echo -e "${GREEN}Installing SysVInit service...${NC}"
    
    cat > "/etc/init.d/$SERVICE_NAME" <<'EOF'
#!/bin/sh
### BEGIN INIT INFO
# Provides:          cloud-update
# Required-Start:    $network $remote_fs
# Required-Stop:     $network $remote_fs
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Cloud Update Service
### END INIT INFO

. /lib/lsb/init-functions

DAEMON=/opt/cloud-update/cloud-update
NAME=cloud-update
PIDFILE=/var/run/$NAME.pid

case "$1" in
    start)
        log_daemon_msg "Starting $NAME"
        start-stop-daemon --start --quiet --background \
            --make-pidfile --pidfile $PIDFILE \
            --exec $DAEMON
        log_end_msg $?
        ;;
    stop)
        log_daemon_msg "Stopping $NAME"
        start-stop-daemon --stop --quiet --pidfile $PIDFILE
        log_end_msg $?
        ;;
    restart)
        $0 stop
        $0 start
        ;;
    status)
        status_of_proc -p $PIDFILE $DAEMON $NAME && exit 0 || exit $?
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac
EOF
    
    chmod 755 "/etc/init.d/$SERVICE_NAME"
    update-rc.d "$SERVICE_NAME" defaults
    echo -e "${GREEN}SysVInit service installed${NC}"
}

# Test installation
test_installation() {
    echo -e "${YELLOW}Testing installation...${NC}"
    
    # Check if binary exists and is executable
    if [ -x "$INSTALL_DIR/$BINARY_NAME" ]; then
        echo -e "${GREEN}✓ Binary is installed and executable${NC}"
    else
        echo -e "${RED}✗ Binary not found or not executable${NC}"
        return 1
    fi
    
    # Check if config exists
    if [ -f "$CONFIG_DIR/config.env" ]; then
        echo -e "${GREEN}✓ Configuration file exists${NC}"
    else
        echo -e "${RED}✗ Configuration file not found${NC}"
        return 1
    fi
    
    # Test binary execution
    if CLOUD_UPDATE_SECRET=test "$INSTALL_DIR/$BINARY_NAME" --version >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Binary runs successfully${NC}"
        CLOUD_UPDATE_SECRET=test "$INSTALL_DIR/$BINARY_NAME" --version
    else
        echo -e "${RED}✗ Binary failed to run${NC}"
        return 1
    fi
    
    # Check service installation based on init system
    case "$INIT_SYSTEM" in
        systemd)
            if [ -f "/etc/systemd/system/$SERVICE_NAME.service" ]; then
                echo -e "${GREEN}✓ Systemd service file is installed${NC}"
                if systemctl is-system-running >/dev/null 2>&1; then
                    if systemctl list-unit-files | grep -q "$SERVICE_NAME"; then
                        echo -e "${GREEN}✓ Systemd service is enabled${NC}"
                    fi
                elif [ "$IN_CONTAINER" = true ]; then
                    if [ -L "/etc/systemd/system/multi-user.target.wants/$SERVICE_NAME.service" ]; then
                        echo -e "${GREEN}✓ Service configured for container systemd${NC}"
                    fi
                fi
            else
                echo -e "${RED}✗ Systemd service file not found${NC}"
                return 1
            fi
            ;;
        openrc)
            if [ -f "/etc/init.d/$SERVICE_NAME" ]; then
                echo -e "${GREEN}✓ OpenRC service is installed${NC}"
            else
                echo -e "${RED}✗ OpenRC service not found${NC}"
                return 1
            fi
            ;;
        sysvinit)
            if [ -f "/etc/init.d/$SERVICE_NAME" ]; then
                echo -e "${GREEN}✓ SysVInit service is installed${NC}"
            else
                echo -e "${RED}✗ SysVInit service not found${NC}"
                return 1
            fi
            ;;
        none)
            if [ "$IN_CONTAINER" = true ]; then
                echo -e "${YELLOW}⚠ Running in container without init system${NC}"
            else
                echo -e "${YELLOW}⚠ No init system detected${NC}"
            fi
            ;;
    esac
    
    echo -e "${GREEN}Installation test completed successfully!${NC}"
    return 0
}

# Main installation flow
main() {
    echo -e "${GREEN}=== Cloud Update Installation ===${NC}"
    
    # Check if running as root
    if [ "$EUID" -ne 0 ]; then
        echo -e "${RED}Please run as root${NC}"
        exit 1
    fi
    
    # Detect environment
    detect_os
    detect_init
    
    # Install binary
    install_binary "$1"
    
    # Install service based on init system
    case "$INIT_SYSTEM" in
        systemd)
            install_systemd
            ;;
        openrc)
            install_openrc
            ;;
        sysvinit)
            install_sysvinit
            ;;
        none)
            echo -e "${YELLOW}No init system detected, skipping service installation${NC}"
            if [ "$IN_CONTAINER" = true ]; then
                echo -e "${YELLOW}You can run the binary directly: $INSTALL_DIR/$BINARY_NAME${NC}"
            fi
            ;;
        *)
            echo -e "${YELLOW}Unknown init system, skipping service installation${NC}"
            ;;
    esac
    
    # Run tests if requested
    if [ "$1" = "--test" ]; then
        test_installation
        exit $?
    fi
    
    echo -e "${GREEN}=== Installation Complete ===${NC}"
    
    if [ "$INIT_SYSTEM" != "none" ]; then
        echo -e "Start the service with:"
        case "$INIT_SYSTEM" in
            systemd)
                echo "  systemctl start $SERVICE_NAME"
                ;;
            openrc)
                echo "  rc-service $SERVICE_NAME start"
                ;;
            sysvinit)
                echo "  service $SERVICE_NAME start"
                ;;
        esac
    elif [ "$IN_CONTAINER" = true ]; then
        echo -e "Run the service directly with:"
        echo "  source $CONFIG_DIR/config.env && $INSTALL_DIR/$BINARY_NAME"
    fi
}

# Run main function
main "$@"