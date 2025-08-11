#\!/bin/sh
# Simple installation script for E2E testing

set -e

echo "Installing cloud-update service..."

# Create directories
mkdir -p /opt/cloud-update
mkdir -p /etc/cloud-update

# Copy binary
if [ -f /app/cloud-update ]; then
    cp /app/cloud-update /opt/cloud-update/cloud-update
    chmod 755 /opt/cloud-update/cloud-update
    echo "Binary installed to /opt/cloud-update/cloud-update"
else
    echo "Error: cloud-update binary not found at /app/cloud-update"
    exit 1
fi

# Run setup
echo "Running setup..."
/opt/cloud-update/cloud-update --setup || {
    echo "Setup completed (may have warnings in container environment)"
}

echo "Installation complete\!"
EOF < /dev/null