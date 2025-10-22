#!/bin/bash
set -e

# Get version information from git
export VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
export GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
export BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "=========================================="
echo "DMR-Nexus Docker Compose"
echo "=========================================="
echo "Version:    $VERSION"
echo "Commit:     $GIT_COMMIT"
echo "Build Time: $BUILD_TIME"
echo "=========================================="
echo ""

# Forward all arguments to docker-compose
docker-compose "$@"
