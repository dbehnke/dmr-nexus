## Compose Build Helper

This project includes a helper script located at `scripts/compose-build`. 
This script is designed to build Docker Compose images using git-derived 
information such as `VERSION`, `GIT_COMMIT`, and `BUILD_TIME`.

### Usage

To use the script, simply run:

```sh
scripts/compose-build
```

The script will create a temporary `.env` file, run `docker compose build`, 
and then restore the original `.env` file if it exists.

# DMR-Nexus

A modern, high-performance DMR (Digital Mobile Radio) repeater networking system written in Go with an embedded Vue3 dashboard. Drop-in replacement for hblink3.

## Features

- **Full HomeBrew Protocol Support**: PEER, MASTER, and OPENBRIDGE modes
- **High Performance**: Handle 200+ simultaneous peer connections using Go's goroutines
- **Web Dashboard**: Real-time monitoring with Vue3, WebSocket updates, and TailwindCSS
- **Dynamic Talkgroup Subscriptions**: Automatic on-demand subscriptions with configurable TTL
- **Timeslot-Agnostic Bridges**: TG bridging works across TS1 and TS2 automatically
- **Conference Bridge**: Talkgroup-based routing between multiple systems
- **Special Talkgroups**: TG 777 (monitor all), TG 4000 (disconnect all dynamic)
- **Single Binary**: All features packaged in one executable with embedded frontend
- **Docker Ready**: Easy deployment with containerization support
- **MQTT Integration**: Real-time events for connect/disconnect/talk actions
- **Prometheus Metrics**: Production-ready observability
- **Comprehensive Testing**: Unit and integration tests with high coverage
- **Modern CI/CD**: Dagger-powered containerized pipeline

## Quick Start

### Prerequisites

- Go 1.21 or later
- Docker (optional)

### Installation

```bash
# Clone the repository
git clone https://github.com/dbehnke/dmr-nexus.git
cd dmr-nexus

# Build the binary
make build

# Run with default configuration
./bin/dmr-nexus --config configs/dmr-nexus.sample.yaml
```

### Docker Deployment

```bash
# Build Docker image
docker build -t dmr-nexus .

# Run container
docker run -p 62031:62031/udp -p 8080:8080 \
  -v ./config.yaml:/etc/dmr-nexus/config.yaml \
  dmr-nexus
```

## Configuration

Create a `config.yaml` file:

```yaml
global:
  ping_time: 5
  max_missed: 3
  use_acl: true

server:
  name: "DMR-Nexus"
  description: "Go DMR Server"

web:
  enabled: true
  port: 8080
  auth_required: false

systems:
  MASTER-1:
    mode: MASTER
    enabled: true
    port: 62031
    passphrase: "changeme"
    max_peers: 50

  REPEATER-1:
    mode: PEER
    enabled: true
    master_ip: "192.168.1.1"
    master_port: 62031
    passphrase: "changeme"
    callsign: "W1ABC"
    radio_id: 312000

  OBP-BRANDMEISTER:
    mode: OPENBRIDGE
    enabled: true
    target_ip: "44.131.4.1"
    target_port: 62031
    network_id: 3129999
    passphrase: "password"

bridges:
  NATIONWIDE:
    - system: MASTER-1
      tgid: 3100
      timeslot: 1
      active: true
    - system: OBP-BRANDMEISTER
      tgid: 91
      timeslot: 1
      active: true
```

## Web Dashboard

Access the web dashboard at `http://localhost:8080` to view:

- **Live Peer Connections**: Real-time status and activity
- **Talkgroup Activity**: History of transmissions with callsigns (TS1/TS2)
- **Bridge Status**: Active bridges and routing configuration
- **System Metrics**: Connection counts, packet rates, uptime statistics
- **Configuration**: Web-based settings management

## DMR Protocol Support

### System Modes

- **MASTER**: Act as a central hub accepting peer connections
- **PEER**: Connect to a master system
- **OPENBRIDGE**: Brandmeister/DMR+ style bridging

### Packet Types

- **DMRD**: DMR data packets (voice/data transmission)
- **RPTL/RPTACK**: Login/authentication
- **RPTK/RPTC**: Key exchange and configuration
- **RPTPING/MSTPONG**: Keepalive mechanism
- **MSTCL**: Connection close

### Access Control

- **REG_ACL**: Peer registration control (MASTER mode)
- **SUB_ACL**: Subscriber ID filtering
- **TG1_ACL**: Talkgroup access on timeslot 1
- **TG2_ACL**: Talkgroup access on timeslot 2

## Conference Bridge System

Route talkgroups between different systems:

```yaml
bridges:
  NATIONWIDE:
    - system: REPEATER-1
      tgid: 3100
      timeslot: 1
      active: true
      on: [3100]      # Activate on TG 3100
      off: [3101]     # Deactivate on TG 3101
      timeout: 15     # Auto-disable after 15 minutes
```

## Special Talkgroups

DMR-Nexus includes special administrative talkgroups for managing dynamic subscriptions:

### TG 777 - Monitor All Mode

**TG 777** enables "parrot mode" or "monitor all" functionality.

- **Purpose**: Receive ALL traffic from all talkgroups regardless of subscriptions
- **Usage**: Key up on TG 777 (any timeslot) to enable
- **Effect**: 
  - Peer enters "repeat mode"
  - Receives all DMRD packets from all talkgroups
  - Bypasses normal subscription filtering
  - Remains active until disabled with TG 4000
- **Use Case**: Network monitoring, troubleshooting, dispatch operations

**Example:**
1. Peer keys up on TG 777
2. Peer now receives traffic from TG 7000, 8000, 9000, etc. simultaneously
3. Peer keys up on TG 4000 to disable and return to normal operation

### TG 4000 - Disconnect All Dynamic

**TG 4000** is a special administrative talkgroup that resets dynamic subscriptions.

- **Purpose**: Immediately unsubscribe from all dynamic talkgroups and disable repeat mode
- **Usage**: Key up on TG 4000 (any timeslot)
- **Effect**: 
  - Removes peer from all dynamic bridges
  - Clears all dynamic subscriptions (TTL-based subscriptions)
  - Disables TG 777 repeat mode if enabled
  - Preserves static subscriptions configured in peer OPTIONS
- **Use Case**: Clean slate without waiting for TTL expiration

**Example:**
1. Peer transmits on TG 7000, 8000, 9000 (creates dynamic subscriptions)
2. Peer enables TG 777 monitor mode
3. Peer keys up TG 4000
4. Peer is now only subscribed to static talkgroups from their configuration
5. TG 777 repeat mode is disabled

See [docs/TALKGROUP_4000.md](docs/TALKGROUP_4000.md) for detailed information.

## Dynamic Talkgroup Subscriptions

DMR-Nexus features an intelligent dynamic subscription system that automatically manages talkgroup access based on transmission activity.

### How It Works

**First Key-Up = Subscription Activation**
- When a peer transmits on a talkgroup for the first time, it subscribes to that talkgroup
- The first transmission does NOT forward audio (subscription key-up only)
- Subsequent transmissions forward normally

**One Talkgroup Per Timeslot**
- Each timeslot (TS1/TS2) can only have one active dynamic subscription at a time
- Transmitting on a new talkgroup automatically unsubscribes from the previous talkgroup in that timeslot
- TS1 and TS2 subscriptions are independent

**Timeslot-Agnostic Bridges**
- Dynamic bridges work across timeslots automatically
- Example: Client 1 on TG 7000 TS1 can talk to Client 2 on TG 7000 TS2
- The server bridges traffic between different timeslot configurations

**Subscription TTL (Time-To-Live)**
- Subscription lifetime is controlled by the peer's `AUTO` setting in OPTIONS
- Example: `OPTIONS=AUTO=600` sets a 10-minute TTL
- If no `AUTO` is specified, subscriptions are unlimited until switching talkgroups
- TTL is refreshed on each transmission

**Bridge Cleanup**
- Dynamic bridges are automatically removed after 5 minutes of having zero subscribers
- Based on actual peer subscriptions, not cached lists
- Cleanup runs every 10 seconds

### Configuration Examples

**DroidStar Client Configuration:**
```
OPTIONS=AUTO=600  # 10-minute subscription TTL
```

**Server Configuration:**
```yaml
systems:
  MASTER-1:
    mode: MASTER
    enabled: true
    port: 62031
    passphrase: "changeme"
    repeat: false  # Use dynamic bridges instead of repeating to all peers
    max_peers: 50
```

### Usage Example

**Scenario: Two clients switching between talkgroups**

1. **Client 1 keys up on TG 7000 TS2:**
   - First transmission: Subscribes to TG 7000 TS2, no audio forwarded
   - Bridge "7000" created (timeslot-agnostic)
   
2. **Client 2 keys up on TG 7000 TS1:**
   - First transmission: Subscribes to TG 7000 TS1, no audio forwarded
   - Now subscribed to same TG but different timeslot
   
3. **Client 1 transmits again on TG 7000:**
   - Audio forwarded to Client 2 (bridge works across TS1 and TS2)
   
4. **Client 1 keys up on TG 8000 TS2:**
   - Automatically unsubscribed from TG 7000 TS2
   - Subscribed to TG 8000 TS2 (first key-up, no audio)
   - Client 2 still on TG 7000
   
5. **After 5 minutes of no subscribers on TG 8000:**
   - Bridge "8000" automatically cleaned up
   
6. **Client 1 keys up on TG 777:**
   - Enters monitor-all mode
   - Receives traffic from all talkgroups
   
7. **Client 1 keys up on TG 4000:**
   - All dynamic subscriptions cleared
   - Monitor-all mode disabled
   - Returns to clean state

## MQTT Integration

Real-time events published to MQTT:

```json
// Peer connection
{
  "type": "peer_connect",
  "peer_id": "312000",
  "callsign": "W1ABC",
  "timestamp": "2024-01-15T10:30:00Z"
}

// Talkgroup activity
{
  "type": "talk_start",
  "peer_id": "312000",
  "subscriber_id": "3120001",
  "talkgroup": "3100",
  "timeslot": 1,
  "timestamp": "2024-01-15T10:31:00Z"
}
```

## Development

### Building from Source

```bash
# Install dependencies
make deps

# Build binary
make build

# Run tests
make test

# Run with live reload (requires air)
make dev
```

### CI/CD Pipeline (Dagger)

DMR-Nexus uses [Dagger](https://dagger.io) for containerized, reproducible CI/CD:

```bash
# Run complete CI pipeline locally
dagger call ci --source=.

# Individual pipeline steps
dagger call test --source=.   # Run tests
dagger call lint --source=.   # Lint code
dagger call build --source=.  # Build binary
```

## Performance

DMR-Nexus is designed for high performance:

- **Connections**: 200+ simultaneous peers
- **Latency**: <5ms packet routing
- **Throughput**: 1000+ packets per second
- **Memory Usage**: <200MB under full load
- **CPU Usage**: <10% on modern hardware

## Migration from hblink3

DMR-Nexus is designed as a drop-in replacement for hblink3:

1. **Install DMR-Nexus** using instructions above
2. **Convert configuration** from hblink3 INI to YAML format (tool provided)
3. **Update bridge rules** to YAML format
4. **Test configuration** with `dmr-nexus --config config.yaml --validate`
5. **Switch over** by stopping hblink3 and starting dmr-nexus

See [MIGRATION.md](docs/MIGRATION.md) for detailed migration guide.

## Documentation

- **[PLAN.md](PLAN.md)**: Comprehensive project plan and architecture
- **[PROTOCOL.md](docs/PROTOCOL.md)**: DMR HomeBrew Protocol details
- **[CONFIG.md](docs/CONFIG.md)**: Configuration reference
- **[API.md](docs/API.md)**: REST API documentation
- **[MIGRATION.md](docs/MIGRATION.md)**: Migration from hblink3

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Development setup
- Coding standards
- Testing requirements
- Pull request process

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Related Projects

- [hblink3](https://github.com/HBLink-org/hblink3) - Original Python implementation
- [ysf-nexus](https://github.com/dbehnke/ysf-nexus) - YSF reflector in Go
- [dmr_utils3](https://github.com/HBLink-org/dmr_utils3) - DMR utilities library

## Support

- **Issues**: [GitHub Issues](https://github.com/dbehnke/dmr-nexus/issues)
- **Discussions**: [GitHub Discussions](https://github.com/dbehnke/dmr-nexus/discussions)
- **Email**: Technical questions to project maintainer

## Acknowledgments

- Thanks to the hblink3 project for the original Python implementation
- DMR community for protocol development and testing

## Building with embedded frontend

If you want a single binary that contains the built Vue3 frontend (no external `frontend/dist` required at runtime), use the embed-aware Makefile target:

```bash
# Build an embedded binary (builds the frontend, copies artifacts, then builds the Go binary)
make build-embed

# The resulting binary will be at:
./bin/dmr-nexus
```

Notes:
- `make build-embed` runs the frontend build and then copies `frontend/dist` into `pkg/web/frontend/dist` so the `//go:embed` pattern used by the server can pick up the files at compile time.
- The Dockerfile's backend build stage copies the frontend `dist` from the `frontend-builder` stage into the backend build context and runs a `go build -tags=embed`, so building the Docker image will also produce an embedded binary.
- If you prefer not to embed, continue to use `make build` which keeps static assets on the filesystem (the server will serve `frontend/dist` from disk if present).
- Go community for excellent networking libraries

## Current Status

🚧 **In Active Development** 🚧

DMR-Nexus is currently in early development. See [PLAN.md](PLAN.md) for roadmap and progress.

### Implemented

- ✅ Project structure and build system
- ✅ Comprehensive planning and documentation
- 🔄 DMR protocol implementation (in progress)
- 🔄 MASTER/PEER modes (in progress)

### Planned

- ⏳ OpenBridge protocol
- ⏳ Conference bridge routing
- ⏳ Web dashboard
- ⏳ MQTT integration
- ⏳ v1.0.0 release

---

**73!** 📻
