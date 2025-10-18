# Build stage for frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build stage for backend
FROM golang:1.21-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy frontend build from previous stage
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Build the application
ARG VERSION=dev
ARG BUILD_TIME=unknown
RUN go build \
    -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -s -w" \
    -o /app/bin/dmr-nexus \
    ./cmd/dmr-nexus

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
