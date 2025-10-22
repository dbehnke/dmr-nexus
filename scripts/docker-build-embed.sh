#!/bin/sh
set -eu

echo "Preparing to build embedded binary inside Docker builder"

# Compute version info using git (repo is available in the build context)
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

echo "Computed VERSION=$VERSION GIT_COMMIT=$GIT_COMMIT BUILD_TIME=$BUILD_TIME"

# Copy frontend build into pkg/web so go:embed can match the files
rm -rf pkg/web/frontend/dist || true
mkdir -p pkg/web/frontend
cp -a frontend/dist pkg/web/frontend/

echo "Running go build with ldflags"
CGO_ENABLED=0 go build -tags=embed \
  -ldflags "-X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT} -X main.buildTime=${BUILD_TIME} -s -w" \
  -o /app/bin/dmr-nexus ./cmd/dmr-nexus

echo "Build complete: /app/bin/dmr-nexus"

exit 0
