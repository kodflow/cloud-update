#!/bin/sh
# Cloud Update Installation Script
# POSIX-compliant script that works on all Linux distributions
# Usage: curl -sSL https://raw.githubusercontent.com/kodflow/cloud-update/main/install.sh | sh

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
SERVICE_DIR="/etc/systemd/system"
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

# Detect init system
detect_init() {
    if [ -d /run/systemd/system ]; then
        echo "systemd"
    elif [ -f /sbin/openrc ]; then
        echo "openrc"
    elif [ -f /etc/init.d/cron ] && [ ! -d /run/systemd/system ]; then
        echo "sysvinit"
    else
        echo "unknown"
    fi
}

# Check if running as root
check_root() {
    if [ "$(id -u)" != "0" ]; then
        log_error "This script must be run as root"
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
    log_info "Binary installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

# Install systemd service
install_systemd() {
    log_info "Installing systemd service..."
    
    cat > "${SERVICE_DIR}/${BINARY_NAME}.service" <<EOF
[Unit]
Description=Cloud Update Agent
After=network.target

[Service]
Type=simple
User=root
EnvironmentFile=-/etc/default/${BINARY_NAME}
ExecStart=${INSTALL_DIR}/${BINARY_NAME} serve
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
    
    # Create default environment file
    cat > "/etc/default/${BINARY_NAME}" <<EOF
# Cloud Update Configuration
# Set your webhook secret here
CLOUD_UPDATE_SECRET=change-me-$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')
CLOUD_UPDATE_PORT=9999
CLOUD_UPDATE_LOG_LEVEL=info
EOF
    
    chmod 600 "/etc/default/${BINARY_NAME}"
    
    systemctl daemon-reload
    log_info "Systemd service installed"
    log_warn "Please edit /etc/default/${BINARY_NAME} to set your webhook secret"
}

# Install OpenRC service
install_openrc() {
    log_info "Installing OpenRC service..."
    
    cat > "/etc/init.d/${BINARY_NAME}" <<'EOF'
#!/sbin/openrc-run

name="Cloud Update"
description="Cloud Update Agent"
command="/usr/local/bin/cloud-update"
command_args="serve"
command_background=true
pidfile="/run/${RC_SVCNAME}.pid"
start_stop_daemon_args="--env CLOUD_UPDATE_SECRET=${CLOUD_UPDATE_SECRET:-change-me}"

depend() {
    need net
    after firewall
}

start_pre() {
    checkpath --directory --owner root:root --mode 0755 /var/log/cloud-update
    checkpath --directory --owner root:root --mode 0755 /var/lib/cloud-update
}
EOF
    
    chmod +x "/etc/init.d/${BINARY_NAME}"
    
    # Create configuration file
    cat > "/etc/conf.d/${BINARY_NAME}" <<EOF
# Cloud Update Configuration
CLOUD_UPDATE_SECRET="change-me-$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')"
CLOUD_UPDATE_PORT="9999"
CLOUD_UPDATE_LOG_LEVEL="info"
EOF
    
    chmod 600 "/etc/conf.d/${BINARY_NAME}"
    
    log_info "OpenRC service installed"
    log_warn "Please edit /etc/conf.d/${BINARY_NAME} to set your webhook secret"
}

# Install sysvinit service
install_sysvinit() {
    log_info "Installing SysVinit service..."
    
    cat > "/etc/init.d/${BINARY_NAME}" <<'EOF'
#!/bin/sh
### BEGIN INIT INFO
# Provides:          cloud-update
# Required-Start:    $network $remote_fs
# Required-Stop:     $network $remote_fs
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Cloud Update Agent
# Description:       Cloud Update system update agent
### END INIT INFO

PATH=/sbin:/usr/sbin:/bin:/usr/bin:/usr/local/bin
DESC="Cloud Update Agent"
NAME=cloud-update
DAEMON=/usr/local/bin/$NAME
DAEMON_ARGS="serve"
PIDFILE=/var/run/$NAME.pid
SCRIPTNAME=/etc/init.d/$NAME

# Load configuration
[ -r /etc/default/$NAME ] && . /etc/default/$NAME

# Export environment variables
export CLOUD_UPDATE_SECRET
export CLOUD_UPDATE_PORT
export CLOUD_UPDATE_LOG_LEVEL

. /lib/lsb/init-functions

do_start() {
    start-stop-daemon --start --quiet --pidfile $PIDFILE \
        --make-pidfile --background \
        --exec $DAEMON -- $DAEMON_ARGS \
        || return 2
}

do_stop() {
    start-stop-daemon --stop --quiet --retry=TERM/30/KILL/5 \
        --pidfile $PIDFILE --name $NAME
    RETVAL="$?"
    rm -f $PIDFILE
    return "$RETVAL"
}

case "$1" in
    start)
        log_daemon_msg "Starting $DESC" "$NAME"
        do_start
        case "$?" in
            0|1) log_end_msg 0 ;;
            2) log_end_msg 1 ;;
        esac
        ;;
    stop)
        log_daemon_msg "Stopping $DESC" "$NAME"
        do_stop
        case "$?" in
            0|1) log_end_msg 0 ;;
            2) log_end_msg 1 ;;
        esac
        ;;
    restart|force-reload)
        log_daemon_msg "Restarting $DESC" "$NAME"
        do_stop
        case "$?" in
            0|1)
                do_start
                case "$?" in
                    0) log_end_msg 0 ;;
                    1) log_end_msg 1 ;;
                    *) log_end_msg 1 ;;
                esac
                ;;
            *)
                log_end_msg 1
                ;;
        esac
        ;;
    status)
        status_of_proc "$DAEMON" "$NAME" && exit 0 || exit $?
        ;;
    *)
        echo "Usage: $SCRIPTNAME {start|stop|status|restart|force-reload}" >&2
        exit 3
        ;;
esac
EOF
    
    chmod +x "/etc/init.d/${BINARY_NAME}"
    
    # Create default configuration
    cat > "/etc/default/${BINARY_NAME}" <<EOF
# Cloud Update Configuration
CLOUD_UPDATE_SECRET="change-me-$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')"
CLOUD_UPDATE_PORT="9999"
CLOUD_UPDATE_LOG_LEVEL="info"
EOF
    
    chmod 600 "/etc/default/${BINARY_NAME}"
    
    # Enable service
    if command -v update-rc.d >/dev/null 2>&1; then
        update-rc.d "${BINARY_NAME}" defaults
    elif command -v chkconfig >/dev/null 2>&1; then
        chkconfig --add "${BINARY_NAME}"
    fi
    
    log_info "SysVinit service installed"
    log_warn "Please edit /etc/default/${BINARY_NAME} to set your webhook secret"
}

# Create necessary directories
create_directories() {
    log_info "Creating directories..."
    mkdir -p /var/log/cloud-update
    mkdir -p /var/lib/cloud-update
    chmod 755 /var/log/cloud-update
    chmod 755 /var/lib/cloud-update
}

# Main installation
main() {
    log_info "Starting Cloud Update installation..."
    
    # Check prerequisites
    check_root
    
    # Create directories
    create_directories
    
    # Download binary
    download_binary
    
    # Install service based on init system
    INIT_SYSTEM=$(detect_init)
    case "$INIT_SYSTEM" in
        systemd)
            install_systemd
            log_info "To start the service: systemctl start ${BINARY_NAME}"
            log_info "To enable at boot: systemctl enable ${BINARY_NAME}"
            ;;
        openrc)
            install_openrc
            log_info "To start the service: rc-service ${BINARY_NAME} start"
            log_info "To enable at boot: rc-update add ${BINARY_NAME} default"
            ;;
        sysvinit)
            install_sysvinit
            log_info "To start the service: service ${BINARY_NAME} start"
            ;;
        *)
            log_warn "Unknown init system. Service not installed."
            log_warn "You can run the binary manually: ${INSTALL_DIR}/${BINARY_NAME} serve"
            ;;
    esac
    
    log_info "Installation complete!"
    log_info "Binary location: ${INSTALL_DIR}/${BINARY_NAME}"
    log_info "Version: $(${INSTALL_DIR}/${BINARY_NAME} version 2>/dev/null || echo 'unknown')"
}

# Run main function
main "$@"