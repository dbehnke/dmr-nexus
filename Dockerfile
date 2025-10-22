# Build stage for frontend
FROM node:24-alpine AS frontend-builder

# Use repository root for build context (package.json and src live at repo root)
WORKDIR /app
# Copy package files from repo root
COPY frontend ./frontend
WORKDIR /app/frontend
RUN npm ci --production=false

# Build the frontend. Output (vite) typically goes to ./dist
RUN npm run build

# Build stage for backend
FROM golang:1.25-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy frontend build from previous stage

# Copy built frontend from frontend-builder's /app/frontend/dist into pkg/web/frontend/dist
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Diagnostic check: ensure pkg/web/frontend/dist exists and list contents (fail fast with helpful output)
RUN if [ -d ./frontend/dist ]; then echo "frontend/dist contents:" && ls -la ./frontend/dist; else echo "ERROR: frontend/dist not found in backend-builder" && ls -la . && false; fi

# Build the application with version information
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# If VERSION/GIT_COMMIT/BUILD_TIME not passed as build-args, compute them here and run the docker build helper
COPY scripts/docker-build-embed.sh /app/scripts/docker-build-embed.sh
# Run the build helper and pass build ARGs through; if the ARGs were not provided
# the helper will compute values (but computing requires .git to be in the context).
RUN chmod +x /app/scripts/docker-build-embed.sh && \
    VERSION="${VERSION}" GIT_COMMIT="${GIT_COMMIT}" BUILD_TIME="${BUILD_TIME}" /app/scripts/docker-build-embed.sh

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 dmr && \
    adduser -D -u 1000 -G dmr dmr

WORKDIR /app

# Copy binary from builder
COPY --from=backend-builder /app/bin/dmr-nexus /usr/local/bin/dmr-nexus

# Copy sample configuration
COPY --from=backend-builder /app/configs/*.sample.yaml /etc/dmr-nexus/

# Create directories
RUN mkdir -p /var/log/dmr-nexus && \
    chown -R dmr:dmr /var/log/dmr-nexus

# Switch to non-root user
USER dmr

# Expose ports
EXPOSE 62031/udp 8080/tcp 9090/tcp

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/usr/local/bin/dmr-nexus"]
CMD ["--config", "/etc/dmr-nexus/config.yaml"]
