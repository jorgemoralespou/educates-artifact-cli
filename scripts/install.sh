#!/bin/bash

# Installation script for artifact-cli
# This script downloads and installs the latest release

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}$1${NC}"
}

# Configuration
REPO="jorgemoralespou/educates-artifact-cli"
BINARY_NAME="artifact-cli"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    armv7l)
        ARCH="arm"
        ;;
    *)
        print_error "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Map OS names
case $OS in
    darwin)
        OS="darwin"
        ;;
    linux)
        OS="linux"
        ;;
    mingw*|cygwin*|msys*)
        OS="windows"
        BINARY_NAME="${BINARY_NAME}.exe"
        ;;
    *)
        print_error "Unsupported OS: $OS"
        exit 1
        ;;
esac

print_header "artifact-cli Installer"
print_status "Detected OS: $OS, Architecture: $ARCH"

# Get latest release
print_status "Fetching latest release information..."
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest")
VERSION=$(echo "$LATEST_RELEASE" | grep '"tag_name"' | cut -d'"' -f4)
DOWNLOAD_URL=$(echo "$LATEST_RELEASE" | grep '"browser_download_url"' | grep "${OS}_${ARCH}" | cut -d'"' -f4)

if [ -z "$DOWNLOAD_URL" ]; then
    print_error "Could not find download URL for $OS/$ARCH"
    print_status "Available releases:"
    echo "$LATEST_RELEASE" | grep '"browser_download_url"' | cut -d'"' -f4
    exit 1
fi

print_status "Latest version: $VERSION"
print_status "Download URL: $DOWNLOAD_URL"

# Create temporary directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Download and extract
print_status "Downloading artifact-cli..."
cd "$TEMP_DIR"
curl -L -o artifact-cli.tar.gz "$DOWNLOAD_URL"
tar -xzf artifact-cli.tar.gz

# Check if binary exists
if [ ! -f "$BINARY_NAME" ]; then
    print_error "Binary not found in downloaded archive"
    ls -la
    exit 1
fi

# Make binary executable
chmod +x "$BINARY_NAME"

# Test binary
print_status "Testing binary..."
./"$BINARY_NAME" --version

# Install binary
print_status "Installing to $INSTALL_DIR..."
if [ "$OS" = "windows" ]; then
    # Windows installation
    INSTALL_DIR="$HOME/bin"
    mkdir -p "$INSTALL_DIR"
    cp "$BINARY_NAME" "$INSTALL_DIR/"
    print_warning "Please add $INSTALL_DIR to your PATH"
else
    # Unix-like installation
    if [ -w "$INSTALL_DIR" ]; then
        cp "$BINARY_NAME" "$INSTALL_DIR/"
    else
        print_status "Using sudo to install to $INSTALL_DIR"
        sudo cp "$BINARY_NAME" "$INSTALL_DIR/"
    fi
fi

print_status "Installation completed successfully!"
print_status "You can now use 'artifact-cli --help' to get started"

# Show usage examples
print_header "Quick Start"
echo "  artifact-cli push ghcr.io/my-user/my-app:1.0.0 -f ./my-app"
echo "  artifact-cli pull ghcr.io/my-user/my-app:1.0.0 -o ./restored-app"
echo ""
echo "For more information, visit: https://github.com/$REPO"
