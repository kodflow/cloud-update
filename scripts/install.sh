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
        OS=$ID
        VER=$VERSION_ID
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

# Detect init system
detect_init() {
    if [ -d /run/systemd/system ]; then
        INIT_SYSTEM="systemd"
    elif [ -f /sbin/openrc ]; then
        INIT_SYSTEM="openrc"
    elif [ -f /etc/init.d/rc ]; then
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
    
    # Copy binary
    if [ -f "./$BINARY_NAME" ]; then
        cp "./$BINARY_NAME" "$INSTALL_DIR/"
        chmod 755 "$INSTALL_DIR/$BINARY_NAME"
    elif [ -f "/app/$BINARY_NAME" ]; then
        cp "/app/$BINARY_NAME" "$INSTALL_DIR/"
        chmod 755 "$INSTALL_DIR/$BINARY_NAME"
    else
        echo -e "${RED}Binary not found!${NC}"
        exit 1
    fi
    
    # Create default config if not exists
    if [ ! -f "$CONFIG_DIR/config.env" ]; then
        cat > "$CONFIG_DIR/config.env" <<EOF
CLOUD_UPDATE_PORT=9999
CLOUD_UPDATE_SECRET=${CLOUD_UPDATE_SECRET:-change-me-in-production}
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
    
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
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
    
    # Check service installation
    case "$INIT_SYSTEM" in
        systemd)
            if systemctl list-unit-files | grep -q "$SERVICE_NAME"; then
                echo -e "${GREEN}✓ Systemd service is installed${NC}"
            else
                echo -e "${RED}✗ Systemd service not found${NC}"
                return 1
            fi
            ;;
        openrc)
            if rc-status -a 2>/dev/null | grep -q "$SERVICE_NAME" || [ -f "/etc/init.d/$SERVICE_NAME" ]; then
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
    install_binary
    
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
        *)
            echo -e "${YELLOW}No init system detected, skipping service installation${NC}"
            ;;
    esac
    
    # Run tests if requested
    if [ "$1" = "--test" ]; then
        test_installation
        exit $?
    fi
    
    echo -e "${GREEN}=== Installation Complete ===${NC}"
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
}

# Run main function
main "$@"