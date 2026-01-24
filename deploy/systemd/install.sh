#!/bin/bash
# Installation script for markata-go systemd services
# Run as root: sudo ./install.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SITE_DIR="${SITE_DIR:-/var/www/mysite}"
SITE_URL="${SITE_URL:-https://example.com}"
GO_VERSION="${GO_VERSION:-1.22.0}"
SERVICE_USER="${SERVICE_USER:-markata}"

echo -e "${GREEN}markata-go systemd Installation Script${NC}"
echo "========================================"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: Please run as root (sudo ./install.sh)${NC}"
    exit 1
fi

# Prompt for configuration
read -p "Site directory [$SITE_DIR]: " input
SITE_DIR="${input:-$SITE_DIR}"

read -p "Site URL [$SITE_URL]: " input
SITE_URL="${input:-$SITE_URL}"

read -p "Service user [$SERVICE_USER]: " input
SERVICE_USER="${input:-$SERVICE_USER}"

echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  Site directory: $SITE_DIR"
echo "  Site URL: $SITE_URL"
echo "  Service user: $SERVICE_USER"
echo ""
read -p "Continue? [y/N] " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

echo ""
echo -e "${GREEN}Step 1: Creating service user...${NC}"
if id "$SERVICE_USER" &>/dev/null; then
    echo "  User '$SERVICE_USER' already exists"
else
    useradd -r -s /bin/false -d "$SITE_DIR" "$SERVICE_USER"
    echo "  Created user '$SERVICE_USER'"
fi

echo ""
echo -e "${GREEN}Step 2: Creating site directory...${NC}"
mkdir -p "$SITE_DIR"/{public,.go}
chown -R "$SERVICE_USER:$SERVICE_USER" "$SITE_DIR"
echo "  Created $SITE_DIR"

echo ""
echo -e "${GREEN}Step 3: Installing Go (if needed)...${NC}"
if command -v go &>/dev/null; then
    echo "  Go already installed: $(go version)"
else
    echo "  Installing Go $GO_VERSION..."
    curl -sSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xzf -
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh
    source /etc/profile.d/go.sh
    echo "  Installed Go $GO_VERSION"
fi

echo ""
echo -e "${GREEN}Step 4: Installing markata-go...${NC}"
sudo -u "$SERVICE_USER" bash -c "
    export HOME=$SITE_DIR
    export GOPATH=$SITE_DIR/.go
    export PATH=\$PATH:/usr/local/go/bin
    go install github.com/WaylonWalker/markata-go/cmd/markata-go@latest
"
echo "  Installed markata-go"

echo ""
echo -e "${GREEN}Step 5: Installing systemd services...${NC}"

# Create service files with correct paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Update and install markata-go.service
sed -e "s|/var/www/mysite|$SITE_DIR|g" \
    -e "s|User=markata|User=$SERVICE_USER|g" \
    -e "s|Group=markata|Group=$SERVICE_USER|g" \
    -e "s|https://example.com|$SITE_URL|g" \
    "$SCRIPT_DIR/markata-go.service" > /etc/systemd/system/markata-go.service
echo "  Installed markata-go.service"

# Update and install markata-go-watch.service
sed -e "s|/var/www/mysite|$SITE_DIR|g" \
    -e "s|User=markata|User=$SERVICE_USER|g" \
    -e "s|Group=markata|Group=$SERVICE_USER|g" \
    -e "s|https://example.com|$SITE_URL|g" \
    "$SCRIPT_DIR/markata-go-watch.service" > /etc/systemd/system/markata-go-watch.service
echo "  Installed markata-go-watch.service"

echo ""
echo -e "${GREEN}Step 6: Reloading systemd...${NC}"
systemctl daemon-reload
echo "  Reloaded systemd"

echo ""
echo -e "${GREEN}Installation complete!${NC}"
echo ""
echo "Next steps:"
echo ""
echo "1. Copy your site content to: $SITE_DIR"
echo ""
echo "2. For one-time builds (with nginx/caddy serving static files):"
echo "   sudo systemctl enable markata-go"
echo "   sudo systemctl start markata-go"
echo ""
echo "3. For watch mode (auto-rebuild on changes):"
echo "   sudo systemctl enable markata-go-watch"
echo "   sudo systemctl start markata-go-watch"
echo ""
echo "4. Check status:"
echo "   sudo systemctl status markata-go"
echo "   sudo journalctl -u markata-go -f"
echo ""
echo "5. Configure your web server (nginx/caddy) to serve: $SITE_DIR/public"
echo ""
