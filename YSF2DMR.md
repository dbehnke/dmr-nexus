# YSF2DMR Bridge Implementation Plan

## Overview

Implementation of a YSF (System Fusion) to DMR bridge utility based on the MMDVM_CM/YSF2DMR C++ implementation. This provides a standalone command-line utility that bridges YSF reflectors/servers to DMR networks, with all core functionality implemented as reusable libraries that can be integrated into dmr-nexus for future dynamic YSF bridges.

## Goals

1. **Standalone Utility**: Command-line tool for dedicated YSF↔DMR bridging
2. **Reusable Libraries**: All functionality in packages that dmr-nexus can use internally
3. **Shared Infrastructure**: Leverage existing dmr-nexus database, config, and logging
4. **Modern Go Architecture**: Context-based, concurrent, with graceful shutdown

## Architecture

### Data Flow

```
YSF Reflector ←UDP→ YSF Network Client → Codec Converter → Bridge Logic → DMR Network Client ←UDP→ DMR Server
                          (pkg/ysf)       (pkg/codec)     (pkg/ysf2dmr)     (pkg/network)
```

### Components

#### 1. YSF Protocol Library (`pkg/ysf/`)

Implements the YSF network protocol for connecting to YSF reflectors:

- **Network Client**: UDP client that connects to YSF servers
  - Sends periodic poll messages (keep-alive)
  - Receives/sends 155-byte YSF frames
  - Handles unlink on shutdown

- **FICH Handler**: Frame Information Channel encoding/decoding
  - Golay(20,8) error correction
  - Frame type identification (Header, Voice, Terminator)
  - Data type identification (VD Mode 1/2, Voice FR)

- **Payload Processor**: YSF frame payload handling
  - Extract source/destination callsigns
  - Extract AMBE voice data
  - Insert AMBE voice data
  - Handle VD Mode 2 data channel

- **Constants & Types**: YSF protocol definitions
  - Frame structure (155 bytes total)
  - Callsign length (10 characters)
  - Frame types and data types
  - Sync patterns

#### 2. Codec Converter (`pkg/codec/`)

Handles bidirectional audio conversion between DMR and YSF:

- **Ring Buffers**: Separate buffers for DMR→YSF and YSF→DMR conversion
- **Frame Synchronization**: Manages timing differences (DMR: 55ms, YSF: 90ms)
- **AMBE Transcoding**: Converts between DMR and YSF AMBE formats
- **Stream States**: Tracks headers, data, and end-of-transmission

#### 3. Bridge Logic (`pkg/ysf2dmr/`)

Main bridge implementation and supporting infrastructure:

- **Configuration**: YAML-based configuration with validation
  - YSF server settings (address, port, callsign)
  - DMR server settings (address, port, credentials)
  - Database path and sync settings
  - Logging configuration

- **DMR ID Lookup**: Callsign ↔ DMR ID mapping
  - Uses existing dmr-nexus RadioID database
  - Handles callsign suffixes (strips -N, /M, etc.)
  - Fallback to default ID if not found

- **Bridge Core**: Orchestrates YSF↔DMR audio routing
  - Manages stream state (active/inactive)
  - Converts YSF callsigns to DMR IDs
  - Routes audio through codec converter
  - Handles headers and terminators

#### 4. Command-Line Utility (`cmd/ysf2dmr/`)

Standalone executable:

- **Main Entry Point**: Application initialization and lifecycle
  - Configuration loading
  - Database initialization
  - RadioID syncing (optional)
  - Signal handling for graceful shutdown

- **Example Configuration**: Well-documented YAML config template
- **README**: Complete usage documentation

### Integration with dmr-nexus

#### Shared Components

1. **Database**: Uses same SQLite database and schema
   - DMRUser table for RadioID lookups
   - Shared dmrid.dat sync mechanism

2. **Logging**: Uses dmr-nexus logger package
   - Consistent log format
   - Component-based logging

3. **Configuration**: Uses viper for YAML config (like dmr-nexus)
   - Environment variable support
   - Default value handling

4. **DMR Protocol**: Will use existing DMR network client
   - PEER mode for connecting to DMR servers
   - Authentication and keep-alive handling

## Implementation Status

### ✅ Completed

1. **YSF Protocol Implementation** (`pkg/ysf/`)
   - [x] Constants and type definitions
   - [x] FICH encoding/decoding with Golay(20,8)
   - [x] Payload processing
   - [x] Network client with UDP communication
   - [x] Unit tests

2. **Codec Converter** (`pkg/codec/`)
   - [x] Ring buffer implementation
   - [x] DMR→YSF conversion
   - [x] YSF→DMR conversion
   - [x] Frame state management

3. **Bridge Logic** (`pkg/ysf2dmr/`)
   - [x] Configuration loader with validation
   - [x] DMR ID lookup using database
   - [x] Main bridge orchestration
   - [x] YSF→DMR processing loop with codec conversion
   - [x] DMR→YSF processing loop with codec conversion
   - [x] Bidirectional audio flow complete

4. **DMR Client Integration** ✨ NEW
   - [x] Integrated dmr-nexus PEER mode client
   - [x] DMR packet transmission (headers, voice, terminators)
   - [x] DMR packet reception handler
   - [x] Stream ID management and sequence numbering
   - [x] YSF→DMR voice bridging with timing synchronization
   - [x] DMR→YSF voice bridging with timing synchronization

5. **Command-Line Utility** (`cmd/ysf2dmr/`)
   - [x] Main application entry point
   - [x] Example configuration file
   - [x] Comprehensive README

6. **Infrastructure Updates**
   - [x] Added FLCO type to protocol package
   - [x] Added logger field functions (Uint32, Float64, Uint)

### 🚧 Pending

1. **Testing**
   - [ ] Integration tests with real YSF reflector
   - [ ] Integration tests with real DMR server
   - [ ] Codec conversion tests
   - [ ] End-to-end bridge tests with actual voice traffic

2. **Documentation**
   - [ ] Add to main dmr-nexus README
   - [ ] Architecture diagrams
   - [ ] Troubleshooting guide

3. **Optimization**
   - [ ] Fine-tune timing for frame synchronization
   - [ ] Optimize codec conversion buffering
   - [ ] Add metrics for monitoring

### 🔮 Future Enhancements

1. **WiresX Support**
   - DTMF decoding for dynamic TG changes
   - WiresX command processing
   - Dynamic talkgroup switching

2. **APRS/GPS Features**
   - Extract GPS from YSF frames
   - Send location to APRS-IS
   - Include location in DMR transmissions

3. **Integration into dmr-nexus**
   - Add YSF bridge type to dmr-nexus config
   - Dynamic YSF bridges per talkgroup
   - Web dashboard for YSF bridge management

4. **Multiple YSF Connections**
   - Support multiple simultaneous YSF servers
   - Per-connection talkgroup mapping
   - Load balancing across servers

## Configuration Reference

### YSF Section

```yaml
ysf:
  callsign: "N0CALL"        # Your callsign (required)
  suffix: "ND"              # YSF suffix (default: ND)
  server_address: "ysf.example.com"  # YSF server (required)
  server_port: 42000        # YSF port (required)
  hang_time: 1000          # Stream hang time in ms
  debug: false             # Enable protocol debug
```

### DMR Section

```yaml
dmr:
  id: 1234567              # Your DMR ID (required)
  callsign: "N0CALL"       # Your callsign (required)
  startup_tg: 9990         # Initial talkgroup (required)
  startup_private: false   # false=TG, true=private
  server_address: "dmr.example.com"  # DMR server (required)
  server_port: 62031       # DMR port (required)
  password: "PASSWORD"     # DMR password (required)
  color_code: 1           # Color code (default: 1)
  # PEER mode configuration
  rx_freq: 435000000      # RX frequency in Hz
  tx_freq: 435000000      # TX frequency in Hz
  tx_power: 1             # Power in watts
  latitude: 0.0           # Location
  longitude: 0.0
  height: 0               # Height in meters
  location: "Unknown"
  description: "YSF2DMR Bridge"
  url: ""
  jitter: 500             # Jitter buffer in ms
  debug: false            # Enable protocol debug
```

### Database Section

```yaml
dmrid:
  database_path: "data/dmr-nexus.db"  # SQLite DB path
  sync_enabled: true      # Auto-sync from radioid.net
  sync_interval: "24h"    # Sync frequency
```

## Technical Details

### YSF Frame Structure

```
Offset  Length  Description
------  ------  -----------
0       4       Signature "YSFD"
4       10      Gateway callsign (space-padded)
14      10      Source callsign (space-padded)
24      10      Destination callsign (space-padded)
34      1       Frame counter
35      120     Payload (FICH + voice/data)
------  ------
Total:  155 bytes
```

### FICH Structure

The FICH (Frame Information Channel Header) is encoded with Golay(20,8):

- **FI** (2 bits): Frame Information (Header=0, Communication=1, Terminator=2)
- **CS** (2 bits): Communication Type / Channel ID
- **CM** (2 bits): Call Mode
- **BN** (1 bit): Block Number
- **BT** (1 bit): Block Type

### Callsign to DMR ID Mapping

1. Extract callsign from YSF frame source field
2. Clean callsign (trim spaces, uppercase)
3. Look up in RadioID database (exact match)
4. If not found, try base callsign (strip suffix like -N, /M)
5. If still not found, use default DMR ID from config

### Audio Conversion

- **YSF AMBE**: 2 AMBE frames per YSF payload (90ms frame time)
- **DMR AMBE**: Voice data in 33-byte payload (55ms frame time)
- **Conversion**: Ring buffers handle timing differences

## Building and Testing

### Build

```bash
# Build standalone utility
go build -o ysf2dmr ./cmd/ysf2dmr

# Run tests
go test ./pkg/ysf/... ./pkg/codec/... ./pkg/ysf2dmr/...
```

### Validate Configuration

```bash
./ysf2dmr -validate -config config.yaml
```

### Run

```bash
./ysf2dmr -config config.yaml
```

## References

### Source Material

- **MMDVM_CM/YSF2DMR**: Original C++ implementation by G4KLX, CA6JAU, EA7EE
- **YSF Protocol**: System Fusion digital voice protocol by Yaesu
- **dmr-nexus**: Existing DMR infrastructure and database

### Related Documentation

- [cmd/ysf2dmr/README.md](cmd/ysf2dmr/README.md) - User documentation
- [pkg/protocol/](pkg/protocol/) - DMR protocol implementation
- [pkg/database/](pkg/database/) - Database schema and repositories

## License

Follows the same license as dmr-nexus and maintains compatibility with MMDVM_CM's GPL license.

## Credits

- Original YSF2DMR: Jonathan Naylor (G4KLX), Andy Uribe (CA6JAU), Manuel Sanchez (EA7EE)
- Go implementation: dmr-nexus project
- Integration with dmr-nexus infrastructure
