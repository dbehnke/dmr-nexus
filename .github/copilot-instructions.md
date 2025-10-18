# DMR-Nexus Copilot Instructions

This document provides guidance for GitHub Copilot when working on the DMR-Nexus project.

## Project Overview

DMR-Nexus is a modern, high-performance DMR (Digital Mobile Radio) repeater networking system written in Go with an embedded Vue3 dashboard. It implements the HomeBrew Protocol (HBP) for DMR amateur radio networks and is designed as a drop-in replacement for hblink3.

## Architecture

- **Language**: Go 1.21+
- **Protocol**: DMR HomeBrew Protocol (HBP) with PEER, MASTER, and OPENBRIDGE modes
- **Frontend**: Vue3 with TailwindCSS (embedded in binary)
- **Deployment**: Single binary with embedded web dashboard
- **Project Pattern**: Follows the architectural pattern of [ysf-nexus](https://github.com/dbehnke/ysf-nexus)

## Project Structure

```
dmr-nexus/
├── cmd/dmr-nexus/           # Main application entry point
├── pkg/
│   ├── config/              # Configuration management (Viper-based YAML)
│   ├── logger/              # Structured logging
│   ├── network/             # UDP network layer (server/client)
│   ├── peer/                # Peer management and ACL
│   └── protocol/            # DMR HomeBrew Protocol implementation
├── configs/                 # Sample configuration files
└── Makefile                 # Build automation
```

## Coding Standards

### General Go Guidelines

1. **Follow standard Go conventions**:
   - Use `gofmt` for formatting (enforced by `make fmt`)
   - Follow effective Go practices
   - Keep functions small and focused
   - Use meaningful variable names

2. **Error Handling**:
   - Always handle errors explicitly
   - Use `fmt.Errorf` or `errors.New` for error wrapping
   - Return errors rather than panicking unless it's truly unrecoverable

3. **Concurrency**:
   - Use goroutines for concurrent operations (this is a high-performance networking application)
   - Always use proper synchronization (mutex, channels, sync.Map)
   - Be mindful of goroutine leaks - ensure proper cleanup

4. **Testing**:
   - Write unit tests for all new functionality
   - Use table-driven tests where appropriate
   - Integration tests use build tag `integration`
   - Aim for high test coverage
   - Tests should be deterministic and not flaky

5. **Comments**:
   - Document all exported functions, types, and constants
   - Use GoDoc style comments
   - Explain complex algorithms or DMR protocol specifics

### Project-Specific Conventions

1. **Logging**:
   - Use the structured logger from `pkg/logger`
   - Example: `log.Info("message", logger.String("key", "value"))`
   - Log levels: Debug, Info, Warn, Error

2. **Configuration**:
   - Use Viper for configuration management
   - Configuration files are in YAML format
   - See `configs/dmr-nexus.sample.yaml` for structure

3. **Peer Management**:
   - Peer connections follow state machine: `StateDisconnected` → `StateRPTLReceived` → `StateAuthenticated` → `StateConfigReceived` → `StateConnected`
   - Always use thread-safe access to peer data structures

4. **Protocol Implementation**:
   - DMR packets are binary protocol - careful with byte ordering
   - DMRD packets are 53 bytes (standard HBP) or 73 bytes (OpenBridge with HMAC-SHA1)
   - Packet types: DMRD, RPTL, RPTACK, RPTK, RPTC, RPTPING, MSTPONG, MSTCL

5. **Naming Conventions**:
   - Peer/Repeater IDs are `uint32`
   - Talkgroups are `uint32`
   - Timeslots are `uint8` (1 or 2)
   - Connection states use enums (ConnectionState type)

## Build and Development

### Building
```bash
make build          # Build the binary
make deps           # Download dependencies
make clean          # Clean build artifacts
```

### Testing
```bash
make test                  # Run unit tests
make test-coverage         # Run tests with coverage report
make test-integration      # Run integration tests
```

### Code Quality
```bash
make lint           # Run golangci-lint (if installed)
make fmt            # Format code
make vet            # Run go vet
```

### Development
```bash
make dev            # Run with live reload (requires air)
make run            # Build and run
```

## Dependencies

- **github.com/spf13/viper**: Configuration management
- Standard library packages for networking, concurrency, and cryptography
- Minimal external dependencies by design

## Important Considerations

1. **Performance**:
   - This is a high-performance networking application designed to handle 200+ simultaneous connections
   - Be mindful of allocations in hot paths
   - Use connection pooling and buffer reuse where appropriate

2. **DMR Protocol Specifics**:
   - DMR is a binary amateur radio protocol
   - HomeBrew Protocol (HBP) is the network protocol used by repeaters
   - Packets must be byte-perfect - protocol compliance is critical

3. **Backwards Compatibility**:
   - Maintain compatibility with hblink3 configuration where possible
   - This is designed as a drop-in replacement

4. **Security**:
   - Handle authentication properly (passphrase-based with challenge-response)
   - ACLs control peer registration and talkgroup access
   - OpenBridge mode uses HMAC-SHA1 signatures

## Common Tasks

### Adding a New Packet Type
1. Define packet structure in `pkg/protocol`
2. Add parser and encoder functions
3. Update packet handler in network layer
4. Add unit tests for parsing/encoding
5. Document packet format in code comments

### Adding a New System Mode
1. Define mode constants in configuration
2. Implement protocol handler in `pkg/network`
3. Update peer manager if needed
4. Add tests for the new mode
5. Update configuration documentation

### Modifying Peer State Machine
1. Update `ConnectionState` enum in `pkg/peer/peer.go`
2. Update state transition logic
3. Add logging for state changes
4. Update tests to cover new states

## Testing Philosophy

- **Unit tests**: Test individual components in isolation
- **Integration tests**: Test component interactions (use `integration` build tag)
- **Table-driven tests**: Use for testing multiple scenarios
- **Mock external dependencies**: Network, time, etc.
- **Test coverage**: Aim for >80% coverage on core components

## Documentation

- **PLAN.md**: Comprehensive project plan and architecture details
- **README.md**: User-facing documentation and quick start
- Inline code comments for complex logic
- GoDoc comments for all exported symbols

## When in Doubt

- Refer to existing code patterns in the repository
- Follow Go standard library conventions
- Keep changes minimal and focused
- Write tests first when fixing bugs
- Ask for clarification on DMR protocol specifics if unsure
