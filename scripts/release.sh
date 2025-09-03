#!/bin/bash

# Release script for artifact-cli
# This script helps create releases manually

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "Not in a git repository"
    exit 1
fi

# Check if there are uncommitted changes
if ! git diff-index --quiet HEAD --; then
    print_error "You have uncommitted changes. Please commit or stash them first."
    exit 1
fi

# Get the current version from git tags
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
print_status "Current version: $CURRENT_VERSION"

# Ask for new version
read -p "Enter new version (e.g., v1.0.0): " NEW_VERSION

# Validate version format
if [[ ! $NEW_VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    print_error "Invalid version format. Use semantic versioning (e.g., v1.0.0)"
    exit 1
fi

# Check if version already exists
if git tag -l | grep -q "^$NEW_VERSION$"; then
    print_error "Version $NEW_VERSION already exists"
    exit 1
fi

print_status "Creating release $NEW_VERSION"

# Update CHANGELOG.md
print_status "Updating CHANGELOG.md"
sed -i.bak "s/## \[Unreleased\]/## \[Unreleased\]\n\n## \[${NEW_VERSION#v}\] - $(date +%Y-%m-%d)/" CHANGELOG.md
rm CHANGELOG.md.bak

# Commit changes
print_status "Committing changes"
git add CHANGELOG.md
git commit -m "chore: prepare release $NEW_VERSION"

# Create tag
print_status "Creating tag $NEW_VERSION"
git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"

# Push changes and tags
print_status "Pushing changes and tags"
git push origin main
git push origin "$NEW_VERSION"

print_status "Release $NEW_VERSION created successfully!"
print_warning "The GitHub Actions workflow will now build and publish the release artifacts."

# Optional: Run GoReleaser locally for testing
read -p "Do you want to run GoReleaser locally for testing? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_status "Running GoReleaser locally..."
    if command -v goreleaser &> /dev/null; then
        goreleaser release --snapshot --clean
    else
        print_warning "GoReleaser not found. Install it with: go install github.com/goreleaser/goreleaser@latest"
    fi
fi

print_status "Done!"
