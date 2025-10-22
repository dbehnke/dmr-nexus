# DMR-Nexus Docker Deployment

This guide covers running DMR-Nexus in Docker with automatic version management based on git tags.

## Quick Start

### 1. Build the Image

The easiest way to build with automatic versioning:

```bash
./build-docker.sh
```

This will:
- Automatically detect the git version (tag + commit hash)
- Build the Docker image with version labels
- Tag as both `dmr-nexus:latest` and `dmr-nexus:<version>`

### 2. Run with Docker Compose

```bash
# Copy sample config
cp configs/config.sample.yaml config.yaml

# Edit config.yaml as needed
vim config.yaml

# Start the service
./docker-compose.sh up -d

# View logs
./docker-compose.sh logs -f

# Stop the service
./docker-compose.sh down
```

## Manual Docker Build

If you prefer manual control:

```bash
# Build with auto-detected version
docker build \
  --build-arg VERSION=auto \
  --build-arg GIT_COMMIT=auto \
  --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t dmr-nexus:latest \
  .

# Or specify version manually
docker build \
  --build-arg VERSION=v1.0.0 \
  --build-arg GIT_COMMIT=abc1234 \
  --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t dmr-nexus:v1.0.0 \
  .
```

## Running Manually

```bash
# Create directories
mkdir -p data logs

# Run container
docker run -d \
  --name dmr-nexus \
  --restart unless-stopped \
  -p 62031:62031/udp \
  -p 8080:8080 \
  -p 9090:9090 \
  -v $(pwd)/config.yaml:/etc/dmr-nexus/config.yaml:ro \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/logs:/var/log/dmr-nexus \
  -e TZ=America/New_York \
  dmr-nexus:latest
```

## Docker Compose Configuration

The `docker-compose.yml` includes:

### Ports
- `62031/udp` - DMR peer protocol
- `8080/tcp` - Web UI and API
- `9090/tcp` - Prometheus metrics

### Volumes
- `./config.yaml` - Configuration file (read-only)
- `dmr-data` - Persistent data (database, etc.)
- `dmr-logs` - Log files

### Environment Variables
- `TZ` - Timezone (default: UTC)
- `VERSION` - Build version (auto-detected from git)
- `GIT_COMMIT` - Git commit hash (auto-detected)
- `BUILD_TIME` - Build timestamp (auto-generated)

## Version Management

The build system automatically uses git to determine the version:

```bash
# If you have tags
git tag v1.0.0
./build-docker.sh
# → Builds as dmr-nexus:v1.0.0

# If no tags (uses commit hash)
./build-docker.sh
# → Builds as dmr-nexus:abc1234

# If you have uncommitted changes
./build-docker.sh
# → Builds as dmr-nexus:v1.0.0-dirty
```

## Checking Version

```bash
# Check version in running container
docker exec dmr-nexus /usr/local/bin/dmr-nexus --version

# Or from logs
docker logs dmr-nexus 2>&1 | grep "version"
```

## Health Checks

The container includes automatic health checking:

```bash
# Check health status
docker ps
# Look for "(healthy)" in STATUS column

# View health check logs
docker inspect dmr-nexus | jq '.[0].State.Health'
```

## Troubleshooting

### View logs
```bash
./docker-compose.sh logs -f dmr-nexus
```

### Access container shell
```bash
docker exec -it dmr-nexus /bin/sh
```

### Restart service
```bash
./docker-compose.sh restart
```

### Rebuild after code changes
```bash
./build-docker.sh
./docker-compose.sh up -d --force-recreate
```

## Production Deployment

For production, consider:

1. **Use specific version tags** instead of `latest`
2. **Enable Prometheus metrics** and monitoring
3. **Set up log rotation** for the logs volume
4. **Configure resource limits** in docker-compose.yml:
   ```yaml
   services:
     dmr-nexus:
       deploy:
         resources:
           limits:
             cpus: '2'
             memory: 1G
           reservations:
             cpus: '1'
             memory: 512M
   ```
5. **Use secrets** for sensitive config instead of volume mounts
6. **Set up automated backups** of the data volume

## Multi-Architecture Builds

To build for multiple architectures:

```bash
# Build for linux/amd64 and linux/arm64
docker buildx create --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=auto \
  --build-arg GIT_COMMIT=auto \
  --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t dmr-nexus:latest \
  --push \
  .
```
