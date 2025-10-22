# Pure Go SQLite Implementation

## Overview

DMR-Nexus uses a **pure Go** SQLite implementation (`modernc.org/sqlite`) instead of the traditional CGO-based `mattn/go-sqlite3`. This provides several benefits for deployment and portability.

## Why Pure Go SQLite?

### Advantages

1. **No CGO Required**
   - Simpler build process
   - No C compiler needed
   - Faster compilation
   - Smaller Docker images

2. **Better Cross-Compilation**
   - Easy to build for different platforms
   - No platform-specific C dependencies
   - Consistent behavior across OS/architectures

3. **Easier Deployment**
   - Static binaries with no external dependencies
   - No need to install SQLite libraries on target system
   - Alpine Docker images work without glibc

4. **Same Performance**
   - modernc.org/sqlite is fully compatible with SQLite
   - Implements the same database engine
   - Performance is comparable for most workloads

### Trade-offs

- Slightly larger binary size (embedded SQLite)
- Pure Go implementation may be slightly slower in some edge cases
- For DMR-Nexus use case (logging transmissions), performance is excellent

## Implementation Details

### Database Driver

```go
// pkg/database/db.go
import (
    "gorm.io/driver/sqlite"
    _ "modernc.org/sqlite"  // Pure Go SQLite driver
)

// Using explicit Dialector to ensure pure Go driver is used
dialector := sqlite.Dialector{
    DriverName: "sqlite",
    DSN:        cfg.Path,
}

db, err := gorm.Open(dialector, &gorm.Config{
    Logger: gormLog,
})
```

### Build Configuration

All builds use `CGO_ENABLED=0`:

**Dockerfile:**
```dockerfile
RUN CGO_ENABLED=0 go build \
    -ldflags "..." \
    -o /app/bin/dmr-nexus \
    ./cmd/dmr-nexus
```

**Makefile:**
```makefile
build:
    @CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dmr-nexus
```

## Verification

### Check Your Build

Verify that your build doesn't require CGO:

```bash
# Check if binary is static
file bin/dmr-nexus
# Should show: statically linked

# Check for CGO dependencies
ldd bin/dmr-nexus 2>&1 | grep -i "not a dynamic"
# Should show: not a dynamic executable

# Build explicitly without CGO
CGO_ENABLED=0 go build ./cmd/dmr-nexus
```

### Test Database Functionality

```bash
# Run database tests without CGO
CGO_ENABLED=0 go test -v ./pkg/database/...

# All tests should pass
```

## Dependencies

The pure Go implementation is provided by:

```go
// go.mod
require (
    gorm.io/driver/sqlite v1.6.0
    gorm.io/gorm v1.31.0
    modernc.org/sqlite v1.39.1  // Pure Go SQLite
)
```

## Migration from CGO SQLite

If you previously built with CGO enabled:

1. **Clean rebuild required:**
   ```bash
   make clean
   make build
   ```

2. **Existing databases are compatible:**
   - SQLite database format is the same
   - No migration needed
   - Just rebuild the application

3. **Docker images:**
   - Rebuild with `./build-docker.sh`
   - New images will use pure Go implementation
   - No data migration needed

## Troubleshooting

### Error: "Binary was compiled with 'CGO_ENABLED=0'"

This error indicates the old CGO-based driver is being used. Solutions:

1. **Ensure you're using the updated code:**
   ```bash
   git pull
   go mod tidy
   ```

2. **Clean and rebuild:**
   ```bash
   make clean
   make build
   ```

3. **Verify imports in database/db.go:**
   ```go
   import (
       "gorm.io/driver/sqlite"
       _ "modernc.org/sqlite"  // This import is required
   )
   ```

### Database Performance

For DMR-Nexus workloads (transmission logging):
- Pure Go SQLite performs excellently
- Typically < 1ms for inserts
- Query performance is more than sufficient
- No performance issues observed

If you need absolute maximum performance:
- Consider CGO build (requires C compiler)
- Change back to `mattn/go-sqlite3`
- Most users won't need this

## References

- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) - Pure Go SQLite
- [GORM SQLite Driver](https://gorm.io/docs/connecting_to_the_database.html#SQLite)
- [CGO_ENABLED environment variable](https://pkg.go.dev/cmd/cgo)
