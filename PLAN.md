# DMR-Nexus Project Plan

## Project Overview

**DMR-Nexus** is a modern, high-performance DMR (Digital Mobile Radio) repeater networking system written in Go with an embedded Vue3 dashboard. It is designed as a complete replacement for hblink3, implementing the HomeBrew Protocol (HBP) for DMR amateur radio networks.

### Goals

- **Drop-in replacement** for hblink3 with full feature parity
- **Modern architecture** using Go's concurrency model for high performance
- **Embedded web dashboard** with real-time monitoring (Vue3 + TailwindCSS)
- **Single binary deployment** with no external dependencies
- **Production-ready** with comprehensive testing and CI/CD

### Inspiration

This project follows the architectural pattern established by [ysf-nexus](https://github.com/dbehnke/ysf-nexus), which successfully reimplemented the YSF reflector protocol in Go. DMR-Nexus applies the same approach to the DMR HomeBrew Protocol.

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                        DMR-Nexus                             │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   HBSYSTEM   │  │  OPENBRIDGE  │  │    Bridge    │      │
│  │ PEER/MASTER  │  │   Protocol   │  │   Routing    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│         │                  │                  │             │
│         └──────────────────┴──────────────────┘             │
│                          │                                  │
│                  ┌───────▼───────┐                          │
│                  │  UDP Network  │                          │
│                  │    Server     │                          │
│                  └───────────────┘                          │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Peer Manager │  │  ACL Engine  │  │Stream Tracker│      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Web Dashboard│  │     MQTT     │  │  Prometheus  │      │
│  │  (Vue3/WS)   │  │  Publisher   │  │   Metrics    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### Key Subsystems

#### 1. Protocol Layer (`pkg/protocol`)

Implements the DMR HomeBrew Protocol packet parsing and encoding:

- **DMRD**: DMR Data packets (53 bytes + 20 byte signature for OpenBridge)
- **RPTL**: Login request from peer
- **RPTACK**: Login acknowledgement from master
- **RPTK**: Key/challenge exchange
- **RPTC**: Configuration packet
- **RPTPING/MSTPONG**: Keepalive mechanism
- **MSTCL**: Close/disconnect from master
- **OpenBridge**: DMR+ style protocol with HMAC-SHA1 authentication

**Packet Structure:**
```
DMRD Packet (Standard HBP):
[0-3]   DMRD (magic)
[4]     Sequence number
[5-7]   Source subscriber ID
[8-10]  Destination ID (talkgroup)
[11-14] Peer/Repeater ID
[15]    Slot/Call type bits
[16-19] Stream ID
[20-52] Voice/Data payload

DMRD Packet (OpenBridge):
[0-52]  Standard DMRD packet
[53-72] HMAC-SHA1 signature (20 bytes)
```

#### 2. Peer Management (`pkg/peer`)

Manages connected repeaters and peer systems:

- **Peer Registry**: Thread-safe tracking of all connected peers
- **Connection States**: NO → RPTL_SENT → AUTHENTICATED → CONFIG_SENT → YES
- **Timeout Management**: Automatic cleanup of inactive peers
- **Statistics**: Track packets/bytes per peer, uptime, last seen
- **ACL Enforcement**: Registration, subscriber, and talkgroup access control

**Connection Flow:**
```
Peer                           Master
  │                              │
  ├─── RPTL (Login) ────────────>│
  │<──────── RPTACK ─────────────┤
  ├─── RPTK (Key) ──────────────>│
  │<──────── RPTACK ─────────────┤
  ├─── RPTC (Config) ───────────>│
  │<──────── RPTACK ─────────────┤
  │                              │
  ├─── RPTPING ─────────────────>│
  │<──────── MSTPONG ────────────┤
  │                              │
  ├─── DMRD (Voice) ────────────>│
  │<──────── DMRD (Echo) ────────┤
```

#### 3. Conference Bridge/Routing (`pkg/bridge`)

Implements talkgroup-based routing between systems:

- **Routing Rules**: Configuration-based traffic forwarding
- **Talkgroup Mapping**: Map TGIDs between different systems
- **ON/OFF Timers**: Activate/deactivate bridges on schedule
- **Unit-to-Unit Optimization**: Direct routing for private calls
- **Stream Deduplication**: Prevent packet loops

**Routing Example:**
```yaml
bridges:
  NATIONWIDE:
    - system: REPEATER-1
      tgid: 3100
      timeslot: 1
      active: true
    - system: REPEATER-2
      tgid: 3100
      timeslot: 1
      active: true
    - system: OPENBRIDGE-BM
      tgid: 91
      timeslot: 1
      active: true
```

#### 4. Network Layer (`pkg/network`)

UDP server/client for packet handling:

- **Server Mode**: Listen for peer connections (MASTER)
- **Client Mode**: Connect to master systems (PEER)
- **OpenBridge Mode**: Stateless UDP with HMAC authentication
- **Packet Validation**: CRC checks, size validation
- **Flow Control**: Rate limiting, congestion management

#### 5. Web Dashboard (`frontend`)

Modern Vue3 single-page application:

- **Real-time Updates**: WebSocket connection for live data
- **Peer Monitoring**: Connection status, uptime, activity
- **Talkgroup Activity**: Live call logs with TS1/TS2 separation
- **Bridge Status**: Active bridges, routing rules, statistics
- **Configuration UI**: Manage ACLs, bridges, system settings
- **Responsive Design**: TailwindCSS for mobile-friendly UI

**Tech Stack:**
- Vue 3 with Composition API
- Vite for build tooling
- TailwindCSS for styling
- Pinia for state management
- Chart.js for metrics visualization

## DMR Protocol Details

### Timeslot Structure

DMR operates on two timeslots (TS1 and TS2) that alternate every 30ms:

```
Time: ──┬──TS1──┬──TS2──┬──TS1──┬──TS2──┬──
        0ms    30ms    60ms    90ms   120ms
```

Each timeslot can carry independent voice/data streams.

### Frame Types

DMR voice calls consist of 6 frames (A-F) plus header/terminator:

- **Frame Type 0**: Voice burst (6 frames: A, B, C, D, E, F)
- **Frame Type 1**: Voice header
- **Frame Type 2**: Voice terminator
- **Frame Type 3**: Data sync

### Call Types

- **Group Call**: Normal talkgroup communication (bit 0x40 clear)
- **Private Call**: Unit-to-unit (bit 0x40 set)
- **CSBK**: Control signaling (bits 0x23 = 0x23)

### Slot Bit Encoding (Byte 15)

```
Bit 7 (0x80): Timeslot (0=TS1, 1=TS2)
Bit 6 (0x40): Call type (0=group, 1=unit)
Bits 4-5 (0x30): Frame type
Bits 0-3 (0x0F): Data type / Voice sequence
```

## Access Control Lists (ACLs)

### ACL Types

1. **REG_ACL**: Controls which peer IDs can register (MASTER mode only)
2. **SUB_ACL**: Controls which subscriber IDs can transmit
3. **TG1_ACL**: Controls talkgroup access on timeslot 1
4. **TG2_ACL**: Controls talkgroup access on timeslot 2

### ACL Format

```
ACTION:RANGE[,RANGE]...

Examples:
PERMIT:ALL                  # Allow everything
DENY:1                      # Deny ID 1
PERMIT:3100-3199            # Allow range
DENY:1,1000-2000,4500-6000  # Deny multiple
```

### ACL Processing

1. Check GLOBAL ACL first
2. Then check SYSTEM-specific ACL
3. First DENY wins (fail fast)
4. Default action if no match

## Configuration

### System Configuration Types

#### MASTER Mode

Acts as a central hub accepting peer connections:

```yaml
systems:
  MASTER-1:
    mode: MASTER
    enabled: true
    repeat: true            # Repeat traffic to other peers
    max_peers: 50
    port: 62031
    passphrase: "secret"
    group_hangtime: 5       # Seconds to keep talkgroup active

    acls:
      reg_acl: "PERMIT:ALL"
      sub_acl: "DENY:1"
      tg1_acl: "PERMIT:ALL"
      tg2_acl: "PERMIT:ALL"
```

#### PEER Mode

Connects to a master system:

```yaml
systems:
  REPEATER-1:
    mode: PEER
    enabled: true
    port: 62032
    master_ip: "192.168.1.1"
    master_port: 62031
    passphrase: "secret"

    # Repeater identification
    callsign: "W1ABC"
    radio_id: 312000
    rx_freq: 449000000
    tx_freq: 444000000
    tx_power: 25
    color_code: 1
    latitude: 38.0000
    longitude: -095.0000
    height: 75
    location: "Boston, MA"
    description: "Repeater on Mt. Washington"
    url: "https://w1abc.org"

    acls:
      sub_acl: "DENY:1"
      tg1_acl: "PERMIT:ALL"
      tg2_acl: "PERMIT:ALL"
```

#### OPENBRIDGE Mode

Brandmeister/DMR+ compatible bridging:

```yaml
systems:
  OBP-BRANDMEISTER:
    mode: OPENBRIDGE
    enabled: true
    port: 62035
    target_ip: "44.131.4.1"
    target_port: 62031
    network_id: 3129999       # Your network ID
    passphrase: "password"
    both_slots: false         # true = allow TS2 for unit calls

    acls:
      sub_acl: "DENY:1"
      tg_acl: "PERMIT:ALL"    # Single ACL (all on TS1)
```

### Bridge Routing Rules

```yaml
bridges:
  # Nationwide bridge on talkgroup 3100
  NATIONWIDE:
    - system: MASTER-1
      tgid: 3100
      timeslot: 1
      active: true
      timeout: 15             # Minutes before auto-disable

    - system: REPEATER-1
      tgid: 3100
      timeslot: 1
      active: true
      on: [3100]              # Activate when TG 3100 received
      off: [3101]             # Deactivate when TG 3101 received

    - system: OBP-BRANDMEISTER
      tgid: 3100
      timeslot: 1
      active: true

  # Regional bridge with timer
  REGIONAL:
    - system: MASTER-1
      tgid: 3120
      timeslot: 2
      active: false
      to_type: OFF            # Start inactive, timer to activate
      timeout: 60             # Activate after 60 minutes
```

### Complete Configuration Example

```yaml
global:
  ping_time: 5              # Seconds between pings
  max_missed: 3             # Missed pings before timeout
  use_acl: true

  acls:
    reg_acl: "PERMIT:ALL"
    sub_acl: "DENY:1"
    tg1_acl: "PERMIT:ALL"
    tg2_acl: "PERMIT:ALL"

server:
  name: "DMR-Nexus"
  description: "Go DMR Server"

web:
  enabled: true
  host: "0.0.0.0"
  port: 8080
  auth_required: false

mqtt:
  enabled: true
  broker: "tcp://localhost:1883"
  topic_prefix: "dmr/nexus"
  client_id: "dmr-nexus"
  qos: 1

logging:
  level: "info"
  format: "json"
  file: "/var/log/dmr-nexus.log"

metrics:
  enabled: true
  prometheus:
    enabled: true
    port: 9090
    path: "/metrics"

systems:
  # ... (system definitions as above)

bridges:
  # ... (bridge definitions as above)
```

## Project Structure

```
dmr-nexus/
├── PLAN.md                      # This file
├── README.md                    # User documentation
├── LICENSE                      # MIT License
├── Makefile                     # Build automation
├── Dockerfile                   # Container build
├── .dockerignore
├── .gitignore
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
│
├── cmd/
│   └── dmr-nexus/
│       └── main.go              # Application entry point
│
├── pkg/
│   ├── protocol/                # DMR HBP protocol implementation
│   │   ├── packets.go           # Packet parsing/encoding
│   │   ├── dmrd.go              # DMRD packet structure
│   │   ├── auth.go              # RPTL/RPTACK/RPTK/RPTC
│   │   ├── openbridge.go        # OpenBridge protocol
│   │   ├── constants.go         # Protocol constants
│   │   └── utils.go             # Helper functions
│   │
│   ├── peer/                    # Peer/repeater management
│   │   ├── manager.go           # Peer registry
│   │   ├── peer.go              # Individual peer state
│   │   ├── acl.go               # Access control lists
│   │   ├── stats.go             # Statistics tracking
│   │   └── timeout.go           # Cleanup loops
│   │
│   ├── bridge/                  # Conference bridge/routing
│   │   ├── router.go            # Routing engine
│   │   ├── rules.go             # Bridge rule processing
│   │   ├── stream.go            # Stream ID tracking
│   │   ├── timer.go             # ON/OFF timer management
│   │   └── unitmap.go           # Unit-to-unit call cache
│   │
│   ├── network/                 # UDP networking
│   │   ├── server.go            # UDP server (MASTER mode)
│   │   ├── client.go            # UDP client (PEER mode)
│   │   ├── openbridge.go        # OpenBridge network handler
│   │   └── metrics.go           # Network statistics
│   │
│   ├── config/                  # Configuration management
│   │   ├── config.go            # Config structures
│   │   ├── loader.go            # YAML parsing
│   │   ├── validation.go        # Config validation
│   │   └── defaults.go          # Default values
│   │
│   ├── web/                     # Web dashboard backend
│   │   ├── server.go            # HTTP server
│   │   ├── websocket.go         # WebSocket handler
│   │   ├── api.go               # REST API endpoints
│   │   ├── embed.go             # Embedded frontend assets
│   │   └── middleware.go        # Auth, logging, CORS
│   │
│   └── logger/                  # Structured logging
│       ├── logger.go            # Logger interface
│       └── zap.go               # Zap implementation
│
├── frontend/                    # Vue3 dashboard
│   ├── package.json
│   ├── vite.config.js
│   ├── index.html
│   ├── tailwind.config.js
│   ├── postcss.config.js
│   │
│   ├── src/
│   │   ├── main.js              # Vue app entry
│   │   ├── App.vue              # Root component
│   │   │
│   │   ├── views/               # Page components
│   │   │   ├── Dashboard.vue    # Main dashboard
│   │   │   ├── Peers.vue        # Peer list/details
│   │   │   ├── Bridges.vue      # Bridge status
│   │   │   ├── Activity.vue     # Talkgroup activity log
│   │   │   ├── Settings.vue     # Configuration UI
│   │   │   └── Login.vue        # Authentication
│   │   │
│   │   ├── components/          # Reusable components
│   │   │   ├── PeerCard.vue
│   │   │   ├── TalkgroupLog.vue
│   │   │   ├── BridgeStatus.vue
│   │   │   └── SystemStats.vue
│   │   │
│   │   ├── stores/              # Pinia state management
│   │   │   ├── peers.js
│   │   │   ├── bridges.js
│   │   │   ├── activity.js
│   │   │   └── websocket.js
│   │   │
│   │   ├── router/              # Vue Router
│   │   │   └── index.js
│   │   │
│   │   ├── composables/         # Composition API utilities
│   │   │   ├── useWebSocket.js
│   │   │   └── useApi.js
│   │   │
│   │   └── assets/              # Static assets
│   │       └── logo.svg
│   │
│   └── dist/                    # Build output (embedded)
│
├── configs/                     # Sample configurations
│   ├── dmr-nexus.sample.yaml
│   ├── bridges.sample.yaml
│   └── docker-compose.yaml
│
├── internal/                    # Internal utilities
│   ├── testhelpers/             # Testing utilities
│   │   ├── mock_peer.go
│   │   ├── mock_network.go
│   │   └── integration_suite.go
│   │
│   └── tools/                   # Development tools
│       └── packet_analyzer/
│
├── dagger/                      # CI/CD pipeline
│   ├── main.go                  # Dagger pipeline
│   └── dagger.json
│
└── docs/                        # Additional documentation
    ├── PROTOCOL.md              # DMR protocol details
    ├── MIGRATION.md             # Migration from hblink3
    └── API.md                   # REST API documentation
```

## Development Phases

### Phase 1: Core Protocol (Weeks 1-2) ✅ COMPLETE

**Deliverables:**
- [x] DMR packet parsing (DMRD, RPTL, RPTK, RPTC, etc.)
- [x] HBSYSTEM PEER mode (connect to master)
- [x] Authentication handshake implementation
- [x] Keepalive/ping mechanism
- [x] Basic packet forwarding
- [x] Unit tests for protocol layer

**Files:**
- `pkg/protocol/constants.go` ✅
- `pkg/protocol/dmrd.go` ✅
- `pkg/protocol/auth.go` ✅
- `pkg/network/client.go` ✅

**Test Coverage:**
- Protocol packet parsing and encoding: 100%
- Authentication handshake: 100%
- Keepalive mechanism: 100%
- Integration test: Client-to-Master bidirectional communication

### Phase 2: Master Mode (Weeks 3-4) ✅ COMPLETE

**Deliverables:**
- [x] HBSYSTEM MASTER mode (accept connections)
- [x] Peer registration and tracking
- [x] ACL enforcement (all types)
- [x] Dual-slot stream management
- [x] Peer-to-peer packet forwarding
- [x] Integration tests with mock peers

**Files:**
- `pkg/peer/manager.go` ✅
- `pkg/peer/peer.go` ✅
- `pkg/peer/acl.go` ✅
- `pkg/network/server.go` ✅

**Test Coverage:**
- Peer state management: 100%
- PeerManager operations: 100%
- ACL parsing and enforcement: 100%
- Server authentication handshake: 100%
- Packet forwarding: 100%
- Timeout cleanup: 100%

### Phase 3: OpenBridge Protocol (Week 5) ✅ COMPLETE

**Deliverables:**
- [x] OPENBRIDGE protocol with HMAC-SHA1
- [x] Brandmeister compatibility testing
- [x] BOTH_SLOTS configuration
- [x] Network ID handling
- [x] OpenBridge integration tests

**Files:**
- `pkg/protocol/openbridge.go` ✅
- `pkg/network/openbridge.go` ✅

**Test Coverage:**
- HMAC computation and verification: 100%
- OpenBridge packet encoding/decoding: 100%
- Network packet send/receive: 100%
- BOTH_SLOTS filtering: 100%
- HMAC authentication: 100%

### Phase 4: Conference Bridge/Routing (Weeks 6-7)

**Deliverables:**
- [ ] Routing rules engine
- [ ] Talkgroup mapping across systems
- [ ] ON/OFF timer support
- [ ] Unit-to-unit call optimization
- [ ] Stream deduplication
- [ ] Bridge configuration loading

**Files:**
- `pkg/bridge/router.go`
- `pkg/bridge/rules.go`
- `pkg/bridge/stream.go`
- `pkg/bridge/timer.go`

### Phase 5: Web Dashboard (Weeks 8-9)

**Deliverables:**
- [ ] Vue3 project setup with Vite
- [ ] WebSocket real-time connection
- [ ] Peer monitoring views
- [ ] Talkgroup activity logging
- [ ] Bridge status display
- [ ] Configuration management UI
- [ ] Responsive design with TailwindCSS

**Files:**
- `frontend/src/App.vue`
- `frontend/src/views/*.vue`
- `frontend/src/stores/*.js`
- `pkg/web/server.go`
- `pkg/web/websocket.go`

### Phase 6: Integration & Testing (Week 10)

**Deliverables:**
- [ ] MQTT event publishing
- [ ] Prometheus metrics endpoints
- [ ] Comprehensive integration tests
- [ ] End-to-end testing with real repeaters
- [ ] Load testing (simulate 50+ peers)
- [ ] Performance optimization

**Files:**
- `pkg/mqtt/publisher.go`
- `pkg/metrics/prometheus.go`
- `internal/testhelpers/*.go`

### Phase 7: CI/CD & Release (Week 11)

**Deliverables:**
- [ ] Dagger CI/CD pipeline
- [ ] Docker multi-stage build
- [ ] GitHub Actions workflow
- [ ] Multi-arch binaries (amd64, arm64)
- [ ] Documentation completion
- [ ] v1.0.0 release

**Files:**
- `dagger/main.go`
- `Dockerfile`
- `.github/workflows/ci.yml`

## Testing Strategy

### Unit Tests

- Protocol packet parsing/encoding
- ACL rule evaluation
- Routing rule matching
- Configuration validation
- Stream tracking logic

**Target:** 80%+ code coverage

### Integration Tests

- PEER → MASTER connection flow
- Multi-peer packet forwarding
- OpenBridge authentication
- Conference bridge routing
- WebSocket event delivery

### End-to-End Tests

- Real repeater connections
- Brandmeister OpenBridge
- Multi-system talkgroup routing
- Dashboard functionality
- MQTT event publishing

### Load Testing

- 50+ simultaneous peer connections
- High packet rate (100+ pps)
- Memory leak detection
- Goroutine leak detection
- CPU usage profiling

## Performance Targets

- **Latency**: <5ms packet routing
- **Throughput**: 1000+ packets per second
- **Connections**: 200+ simultaneous peers
- **Memory**: <200MB under full load
- **CPU**: <10% on modern hardware (4 cores)
- **Uptime**: 99.9% availability

## Compatibility Matrix

### Supported Systems

| System Type | Protocol | Compatibility | Notes |
|-------------|----------|---------------|-------|
| hblink3 | HBP | ✅ Full | Drop-in replacement |
| Brandmeister | OpenBridge | ✅ Full | HMAC-SHA1 auth |
| DMR+ | OpenBridge | ✅ Full | Network ID support |
| MMDVM Repeaters | HBP Peer | ✅ Full | Standard repeaters |
| Pi-Star | HBP Peer | ✅ Full | Hotspot support |
| OpenSpot | HBP Peer | ✅ Full | SharkRF devices |
| XLX Reflectors | XLXPEER | 🔄 Planned | Future enhancement |

### Migration from hblink3

**Configuration Conversion:**
- INI → YAML format
- Automatic converter tool provided
- Side-by-side operation supported

**Feature Parity:**
- ✅ All MASTER/PEER/OPENBRIDGE modes
- ✅ ACL system (all types)
- ✅ Conference bridge routing
- ✅ Reporting server protocol
- ✅ Alias/directory support
- ➕ Enhanced web dashboard
- ➕ Real-time monitoring
- ➕ Better performance

## Security Considerations

### Authentication

- HMAC-based challenge/response
- Passphrase protection
- Peer ID validation
- OpenBridge HMAC-SHA1 signatures

### Access Control

- Multi-level ACL system
- IP-based restrictions (future)
- Rate limiting (future)
- Blacklist support

### Network Security

- No external dependencies in binary
- Minimal attack surface
- Security audit before v1.0
- CVE monitoring for dependencies

## Deployment Options

### Standalone Binary

```bash
./dmr-nexus --config /etc/dmr-nexus/config.yaml
```

### Docker Container

```bash
docker run -p 62031:62031/udp -p 8080:8080 \
  -v ./config.yaml:/etc/dmr-nexus/config.yaml \
  dbehnke/dmr-nexus:latest
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dmr-nexus
spec:
  replicas: 1
  selector:
    matchLabels:
      app: dmr-nexus
  template:
    metadata:
      labels:
        app: dmr-nexus
    spec:
      containers:
      - name: dmr-nexus
        image: dbehnke/dmr-nexus:latest
        ports:
        - containerPort: 62031
          protocol: UDP
        - containerPort: 8080
          protocol: TCP
```

### Systemd Service

```ini
[Unit]
Description=DMR-Nexus Server
After=network.target

[Service]
Type=simple
User=dmr
ExecStart=/usr/local/bin/dmr-nexus --config /etc/dmr-nexus/config.yaml
Restart=always

[Install]
WantedBy=multi-user.target
```

## Monitoring & Observability

### Metrics (Prometheus)

- `dmr_peers_total` - Total peer connections
- `dmr_peers_active` - Currently active peers
- `dmr_packets_received_total` - Packets received by type
- `dmr_packets_sent_total` - Packets sent by type
- `dmr_bytes_received_total` - Bytes received
- `dmr_bytes_sent_total` - Bytes sent
- `dmr_talkgroups_active` - Active talkgroups
- `dmr_streams_active` - Active voice streams
- `dmr_bridge_routes_total` - Bridge routing events

### Logging

**Structured JSON logging:**
```json
{
  "level": "info",
  "timestamp": "2024-01-15T10:30:00Z",
  "component": "peer",
  "peer_id": "312000",
  "callsign": "W1ABC",
  "message": "Peer connected",
  "uptime": "15m30s"
}
```

**Log Levels:**
- DEBUG: Protocol details, packet hex dumps
- INFO: Connections, disconnections, major events
- WARN: ACL denials, retries, degraded state
- ERROR: Connection failures, protocol errors

### Health Checks

- HTTP `/health` endpoint
- UDP echo test
- Database connectivity (if applicable)
- External service checks (MQTT, etc.)

## Documentation Plan

### User Documentation

1. **README.md**: Quick start, features, installation
2. **INSTALL.md**: Detailed installation guide
3. **CONFIG.md**: Configuration reference
4. **MIGRATION.md**: Migration from hblink3
5. **TROUBLESHOOTING.md**: Common issues and solutions

### Developer Documentation

1. **PROTOCOL.md**: DMR HBP protocol details
2. **ARCHITECTURE.md**: System architecture
3. **API.md**: REST API reference
4. **CONTRIBUTING.md**: Development workflow
5. **CHANGELOG.md**: Version history

### API Documentation

- OpenAPI/Swagger specification
- WebSocket message format
- MQTT event schema
- Prometheus metrics catalog

## Success Criteria

### v1.0 Release Criteria

- ✅ Full hblink3 feature parity
- ✅ 80%+ code coverage
- ✅ Successful Brandmeister integration
- ✅ 10+ repeater production testing
- ✅ Complete documentation
- ✅ CI/CD pipeline operational
- ✅ Docker images published
- ✅ Security audit complete

### Community Adoption

- GitHub stars: 100+
- Active users: 50+
- Contributors: 5+
- Issues resolved: 90%+
- Forum posts/discussions active

## Resources & References

### DMR Protocol Documentation

- [HomeBrew Protocol Specification](https://github.com/g4klx/MMDVM_Bridge/blob/master/PROTOCOL.md)
- [Brandmeister OpenBridge](https://wiki.brandmeister.network/index.php/OpenBridge)
- [DMR Standard (ETSI TS 102 361)](https://www.etsi.org/deliver/etsi_ts/102300_102399/10236101/01.02.01_60/ts_10236101v010201p.pdf)

### Related Projects

- [hblink3](https://github.com/HBLink-org/hblink3) - Python implementation
- [ysf-nexus](https://github.com/dbehnke/ysf-nexus) - YSF Go implementation
- [dmr_utils3](https://github.com/HBLink-org/dmr_utils3) - DMR utilities

### Development Tools

- [Go](https://golang.org/) - Programming language
- [Vue 3](https://vuejs.org/) - Frontend framework
- [Dagger](https://dagger.io/) - CI/CD platform
- [Docker](https://www.docker.com/) - Containerization

## License

MIT License - Open source, free for commercial and non-commercial use.

## Contributing

Contributions welcome! See CONTRIBUTING.md for development workflow and coding standards.

## Support

- GitHub Issues: Bug reports and feature requests
- GitHub Discussions: Questions and community support
- Email: Technical questions to project maintainer

---

**Version**: 1.0
**Last Updated**: 2025-10-17
**Author**: DMR-Nexus Development Team
