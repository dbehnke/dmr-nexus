# YSF2DMR Bridge

A standalone command-line utility that bridges YSF (System Fusion) networks to DMR networks. This bridge allows YSF users to communicate with DMR talkgroups and vice versa.

## Features

- **Bidirectional Audio Bridge**: Connects YSF reflectors/servers to DMR networks
- **Automatic DMR ID Lookup**: Uses RadioID database to map callsigns to DMR IDs
- **Shared Database**: Reuses the dmr-nexus database infrastructure
- **Modern Go Architecture**: Context-based cancellation, graceful shutdown
- **Configurable**: YAML-based configuration
- **Reusable Libraries**: All core functionality is in libraries that can be integrated into dmr-nexus

## Architecture

The YSF2DMR bridge consists of several reusable packages:

- **pkg/ysf**: YSF protocol implementation (network, FICH, payload)
- **pkg/codec**: Audio codec conversion between DMR and YSF
- **pkg/ysf2dmr**: Bridge logic, configuration, and DMR ID lookups
- **cmd/ysf2dmr**: Command-line utility

## Installation

### From Source

```bash
# Build the binary
cd cmd/ysf2dmr
go build -o ysf2dmr

# Or use make from the project root
make build-ysf2dmr
```

## Configuration

Create a `config.yaml` file (see [config.yaml](config.yaml) for a complete example):

```yaml
ysf:
  callsign: "N0CALL"
  suffix: "ND"
  server_address: "ysf.example.com"
  server_port: 42000
  hang_time: 1000

dmr:
  id: 1234567
  callsign: "N0CALL"
  startup_tg: 9990
  startup_private: false
  server_address: "dmr.example.com"
  server_port: 62031
  password: "PASSWORD"
  color_code: 1
  # ... more DMR settings

dmrid:
  database_path: "data/dmr-nexus.db"
  sync_enabled: true
  sync_interval: "24h"

logging:
  level: "info"
  format: "text"
```

### Configuration Options

#### YSF Configuration

- `callsign`: Your callsign as it appears on YSF (required)
- `suffix`: Suffix for YSF identification (default: "ND")
- `server_address`: YSF reflector/server address (required)
- `server_port`: YSF server port (required)
- `hang_time`: Keep stream active after last voice frame (ms, default: 1000)
- `debug`: Enable YSF protocol debug logging (default: false)

#### DMR Configuration

- `id`: Your DMR ID (required)
- `callsign`: Your callsign for DMR (required)
- `startup_tg`: Talkgroup to connect to on startup (required)
- `startup_private`: false for TG calls, true for private calls (default: false)
- `server_address`: DMR server address (required)
- `server_port`: DMR server port (required)
- `password`: DMR server password (required)
- `color_code`: DMR color code (default: 1)
- `rx_freq`, `tx_freq`: Radio frequencies in Hz (required for PEER mode)
- `tx_power`: TX power in watts (default: 1)
- `latitude`, `longitude`, `height`: Location information
- `location`, `description`, `url`: Additional info
- `jitter`: Jitter buffer in milliseconds (default: 500)
- `debug`: Enable DMR protocol debug logging (default: false)

#### DMR ID Database

- `database_path`: Path to SQLite database (default: "data/dmr-nexus.db")
- `sync_enabled`: Auto-sync from radioid.net (default: true)
- `sync_interval`: Sync frequency (default: "24h")

## Usage

### Basic Usage

```bash
# Run with default config file (config.yaml in current directory)
./ysf2dmr

# Run with custom config file
./ysf2dmr -config /path/to/config.yaml

# Validate configuration without starting
./ysf2dmr -validate

# Show version information
./ysf2dmr -version
```

### Running as a Service

#### systemd (Linux)

Create `/etc/systemd/system/ysf2dmr.service`:

```ini
[Unit]
Description=YSF2DMR Bridge
After=network.target

[Service]
Type=simple
User=ysf2dmr
WorkingDirectory=/opt/ysf2dmr
ExecStart=/opt/ysf2dmr/ysf2dmr -config /etc/ysf2dmr/config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable ysf2dmr
sudo systemctl start ysf2dmr
sudo systemctl status ysf2dmr
```

## How It Works

### Data Flow

1. **YSF → DMR**:
   - Receive YSF voice frames from YSF reflector
   - Extract source callsign from YSF frame
   - Look up DMR ID from callsign using RadioID database
   - Convert YSF AMBE audio to DMR AMBE format
   - Send DMR packets to DMR network

2. **DMR → YSF**:
   - Receive DMR voice packets from DMR network
   - Look up callsign from DMR ID using RadioID database
   - Convert DMR AMBE audio to YSF AMBE format
   - Send YSF frames to YSF reflector

### Callsign to DMR ID Mapping

The bridge uses the RadioID database (same as dmr-nexus) to map YSF callsigns to DMR IDs:

- Database is automatically synced from radioid.net
- Callsign suffixes (e.g., `-N`, `/M`) are stripped for lookup
- If no DMR ID is found, the configured default ID is used

## Limitations

### Current Status

This is an initial implementation with some limitations:

- **No WiresX Support**: WiresX commands are not implemented
- **No APRS/GPS**: Location features not implemented
- **No MMDVM Host**: Only works with network servers, not local MMDVM
- **DMR Client Pending**: Full DMR PEER mode client needs integration

### Future Enhancements

- Complete DMR PEER mode client integration
- WiresX command processing for dynamic TG changes
- APRS/GPS location features
- Support for multiple simultaneous YSF connections
- Web dashboard integration
- Integration into dmr-nexus for dynamic YSF bridges

## Troubleshooting

### Enable Debug Logging

Set logging level to `debug` in config:

```yaml
logging:
  level: "debug"
```

Enable protocol-specific debug:

```yaml
ysf:
  debug: true

dmr:
  debug: true
```

### Common Issues

**No DMR ID found for callsign**:
- Ensure RadioID database is synced (`sync_enabled: true`)
- Check if callsign is registered at radioid.net
- Bridge will use default DMR ID if lookup fails

**Cannot connect to YSF server**:
- Verify server address and port
- Check firewall rules
- Ensure YSF server is accepting connections

**Cannot connect to DMR server**:
- Verify server address, port, and password
- Check that your DMR ID is authorized
- Ensure network connectivity

## Development

### Running Tests

```bash
# Test YSF protocol
go test ./pkg/ysf/...

# Test codec converter
go test ./pkg/codec/...

# Test YSF2DMR bridge
go test ./pkg/ysf2dmr/...

# All tests
go test ./...
```

### Building from Source

```bash
# Clone repository
git clone https://github.com/dbehnke/dmr-nexus.git
cd dmr-nexus

# Build YSF2DMR
cd cmd/ysf2dmr
go build

# Or use Makefile
cd ../..
make build-ysf2dmr
```

## License

This project follows the same license as dmr-nexus. See the main repository for details.

## Credits

Based on the YSF2DMR implementation from MMDVM_CM by:
- Jonathan Naylor (G4KLX)
- Andy Uribe (CA6JAU)
- Manuel Sanchez (EA7EE)

Reimplemented in Go with modern architecture and integrated with dmr-nexus infrastructure.

## Support

For issues, questions, or contributions:
- GitHub Issues: https://github.com/dbehnke/dmr-nexus/issues
- Main Documentation: See dmr-nexus README
