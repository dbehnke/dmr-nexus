package peer

import (
	"net"
	"sync"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/protocol"
)

// ConnectionState represents the state of a peer connection
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateRPTLReceived
	StateAuthenticated
	StateConfigReceived
	StateConnected
)

// String returns the string representation of the connection state
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateRPTLReceived:
		return "rptl_received"
	case StateAuthenticated:
		return "authenticated"
	case StateConfigReceived:
		return "config_received"
	case StateConnected:
		return "connected"
	default:
		return "unknown"
	}
}

// Peer represents a connected repeater or peer system
type Peer struct {
	ID      uint32
	Address *net.UDPAddr
	State   ConnectionState

	// Configuration from RPTC packet
	Callsign    string
	RXFreq      string
	TXFreq      string
	TXPower     string
	ColorCode   string
	Latitude    string
	Longitude   string
	Height      string
	Location    string
	Description string
	URL         string
	SoftwareID  string
	PackageID   string

	// Connection tracking
	ConnectedAt time.Time
	LastHeard   time.Time
	Salt        []byte

	// Statistics
	PacketsReceived uint64
	BytesReceived   uint64
	PacketsSent     uint64
	BytesSent       uint64

	mu sync.RWMutex
}

// NewPeer creates a new peer with the given ID and address
func NewPeer(id uint32, addr *net.UDPAddr) *Peer {
	return &Peer{
		ID:      id,
		Address: addr,
		State:   StateDisconnected,
	}
}

// SetState updates the peer's connection state
func (p *Peer) SetState(state ConnectionState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.State = state
}

// GetState returns the peer's current connection state
func (p *Peer) GetState() ConnectionState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.State
}

// UpdateLastHeard updates the last heard timestamp to now
func (p *Peer) UpdateLastHeard() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.LastHeard = time.Now()
}

// GetLastHeard returns the last heard timestamp
func (p *Peer) GetLastHeard() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.LastHeard
}

// IsTimedOut checks if the peer has timed out based on the given duration
func (p *Peer) IsTimedOut(timeout time.Duration) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// If never heard, consider timed out
	if p.LastHeard.IsZero() {
		return true
	}
	
	return time.Since(p.LastHeard) > timeout
}

// SetConnected marks the peer as connected and sets the connection time
func (p *Peer) SetConnected() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.State = StateConnected
	p.ConnectedAt = time.Now()
}

// GetConnectedAt returns the connection timestamp
func (p *Peer) GetConnectedAt() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ConnectedAt
}

// SetConfig updates the peer's configuration from an RPTC packet
func (p *Peer) SetConfig(config *protocol.RPTCPacket) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.Callsign = config.Callsign
	p.RXFreq = config.RXFreq
	p.TXFreq = config.TXFreq
	p.TXPower = config.TXPower
	p.ColorCode = config.ColorCode
	p.Latitude = config.Latitude
	p.Longitude = config.Longitude
	p.Height = config.Height
	p.Location = config.Location
	p.Description = config.Description
	p.URL = config.URL
	p.SoftwareID = config.SoftwareID
	p.PackageID = config.PackageID
}

// IncrementPacketsReceived increments the packets received counter
func (p *Peer) IncrementPacketsReceived() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.PacketsReceived++
}

// AddBytesReceived adds to the bytes received counter
func (p *Peer) AddBytesReceived(bytes uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.BytesReceived += bytes
}

// IncrementPacketsSent increments the packets sent counter
func (p *Peer) IncrementPacketsSent() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.PacketsSent++
}

// AddBytesSent adds to the bytes sent counter
func (p *Peer) AddBytesSent(bytes uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.BytesSent += bytes
}

// GetUptime returns the peer's uptime duration
func (p *Peer) GetUptime() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.ConnectedAt.IsZero() {
		return 0
	}
	
	return time.Since(p.ConnectedAt)
}
