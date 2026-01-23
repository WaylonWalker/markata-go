#!/bin/bash
# Install markata-go
# Usage: curl -sSL https://waylonwalker.github.io/markata-go/install.sh | bash
#
# Environment variables:
#   INSTALL_DIR - Installation directory (default: /usr/local/bin or ~/.local/bin)
#   VERSION     - Specific version to install (default: latest)

set -e

REPO="WaylonWalker/markata-go"
BINARY_NAME="markata-go"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}==>${NC} $1"
}

warn() {
    echo -e "${YELLOW}Warning:${NC} $1"
}

error() {
    echo -e "${RED}Error:${NC} $1" >&2
    exit 1
}

# Detect OS
detect_os() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$OS" in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        freebsd) OS="freebsd" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) error "Unsupported operating system: $OS" ;;
    esac
    echo "$OS"
}

# Detect architecture
detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        armv7l|armv7) ARCH="armv7" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac
    echo "$ARCH"
}

# Determine install directory
get_install_dir() {
    if [ -n "$INSTALL_DIR" ]; then
        echo "$INSTALL_DIR"
        return
    fi

    # Try /usr/local/bin first if we have write access
    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
    elif [ -d "$HOME/.local/bin" ] || mkdir -p "$HOME/.local/bin" 2>/dev/null; then
        echo "$HOME/.local/bin"
    else
        error "Cannot determine install directory. Set INSTALL_DIR environment variable."
    fi
}

# Get the latest release version from GitHub
get_latest_version() {
    if [ -n "$VERSION" ]; then
        echo "$VERSION"
        return
    fi

    # Try to get the latest release tag
    local version
    if command -v curl &> /dev/null; then
        version=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
    elif command -v wget &> /dev/null; then
        version=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    if [ -z "$version" ]; then
        error "Could not determine latest version. Please set VERSION environment variable."
    fi

    echo "$version"
}

# Download and install
install() {
    local os=$(detect_os)
    local arch=$(detect_arch)
    local install_dir=$(get_install_dir)
    local version=$(get_latest_version)

    # Remove 'v' prefix for archive name if present
    local version_num="${version#v}"

    info "Installing $BINARY_NAME $version for $os/$arch"

    # Construct download URL
    # GoReleaser uses format: markata-go_0.1.0_linux_amd64.tar.gz
    local archive_ext="tar.gz"
    if [ "$os" = "windows" ]; then
        archive_ext="zip"
    fi

    local archive_name="${BINARY_NAME}_${version_num}_${os}_${arch}.${archive_ext}"
    local url="https://github.com/$REPO/releases/download/${version}/${archive_name}"

    info "Downloading from $url"

    # Create temp directory
    local tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    # Download
    if command -v curl &> /dev/null; then
        curl -sL "$url" -o "$tmp_dir/$archive_name" || error "Download failed. Check if the version exists."
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O "$tmp_dir/$archive_name" || error "Download failed. Check if the version exists."
    fi

    # Verify download succeeded
    if [ ! -f "$tmp_dir/$archive_name" ]; then
        error "Download failed"
    fi

    # Extract
    info "Extracting archive"
    cd "$tmp_dir"
    if [ "$archive_ext" = "zip" ]; then
        unzip -q "$archive_name" || error "Extraction failed"
    else
        tar xzf "$archive_name" || error "Extraction failed"
    fi

    # Find the binary (it might be in the root or a subdirectory)
    local binary_path
    if [ -f "$BINARY_NAME" ]; then
        binary_path="$BINARY_NAME"
    elif [ -f "${BINARY_NAME}.exe" ]; then
        binary_path="${BINARY_NAME}.exe"
    else
        error "Binary not found in archive"
    fi

    # Install
    info "Installing to $install_dir"
    if [ -w "$install_dir" ]; then
        mv "$binary_path" "$install_dir/"
        chmod +x "$install_dir/$BINARY_NAME"
    else
        warn "Need elevated permissions to install to $install_dir"
        sudo mv "$binary_path" "$install_dir/"
        sudo chmod +x "$install_dir/$BINARY_NAME"
    fi

    # Verify installation
    if command -v "$BINARY_NAME" &> /dev/null; then
        info "Successfully installed $BINARY_NAME"
        echo ""
        "$BINARY_NAME" version
    else
        warn "Installation complete, but $BINARY_NAME is not in PATH"
        echo ""
        echo "Add $install_dir to your PATH:"
        echo "  export PATH=\"\$PATH:$install_dir\""
        echo ""
        echo "Then verify with:"
        echo "  $BINARY_NAME version"
    fi
}

# Run installation
install
