#!/bin/bash
set -e

# Get version information from git
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "=========================================="
echo "Building DMR-Nexus Docker Image"
echo "=========================================="
echo "Version:    $VERSION"
echo "Commit:     $GIT_COMMIT"
echo "Build Time: $BUILD_TIME"
echo "CGO:        Disabled (pure Go SQLite)"
echo "=========================================="

# Build the Docker image
docker build \
    --build-arg VERSION="$VERSION" \
    --build-arg GIT_COMMIT="$GIT_COMMIT" \
    --build-arg BUILD_TIME="$BUILD_TIME" \
    -t "dmr-nexus:${VERSION}" \
    -t "dmr-nexus:latest" \
    .

echo ""
echo "=========================================="
echo "Build complete!"
echo "Image tags:"
echo "  - dmr-nexus:${VERSION}"
echo "  - dmr-nexus:latest"
echo "=========================================="
