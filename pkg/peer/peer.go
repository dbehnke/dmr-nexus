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

	// Dynamic subscription state
	Subscriptions *SubscriptionState

	// Repeat mode - when enabled, peer receives all traffic regardless of subscriptions
	RepeatMode bool

	mu sync.RWMutex
}

// Snapshot is a read-only view of a Peer suitable for API responses
type Snapshot struct {
	ID            uint32    `json:"id"`
	Address       string    `json:"address"`
	State         string    `json:"state"`
	Callsign      string    `json:"callsign"`
	Location      string    `json:"location"`
	ConnectedAt   time.Time `json:"connected_at"`
	LastHeard     time.Time `json:"last_heard"`
	PacketsRx     uint64    `json:"packets_rx"`
	BytesRx       uint64    `json:"bytes_rx"`
	PacketsTx     uint64    `json:"packets_tx"`
	BytesTx       uint64    `json:"bytes_tx"`
	Subscriptions struct {
		TS1 []uint32 `json:"ts1,omitempty"`
		TS2 []uint32 `json:"ts2,omitempty"`
	} `json:"subscriptions,omitempty"`
}

// Snapshot returns a consistent read-only snapshot of the peer's state
func (p *Peer) Snapshot(includeSubscriptions bool) Snapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	snap := Snapshot{
		ID:          p.ID,
		State:       p.State.String(),
		Callsign:    p.Callsign,
		Location:    p.Location,
		ConnectedAt: p.ConnectedAt,
		LastHeard:   p.LastHeard,
		PacketsRx:   p.PacketsReceived,
		BytesRx:     p.BytesReceived,
		PacketsTx:   p.PacketsSent,
		BytesTx:     p.BytesSent,
	}
	if p.Address != nil {
		snap.Address = p.Address.String()
	}
	if includeSubscriptions && p.Subscriptions != nil {
		// Use existing getters to provide active talkgroups only
		snap.Subscriptions.TS1 = p.Subscriptions.GetTalkgroups(1)
		snap.Subscriptions.TS2 = p.Subscriptions.GetTalkgroups(2)
	}
	return snap
}

// NewPeer creates a new peer with the given ID and address
func NewPeer(id uint32, addr *net.UDPAddr) *Peer {
	return &Peer{
		ID:            id,
		Address:       addr,
		State:         StateDisconnected,
		Subscriptions: NewSubscriptionState(),
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

	// Parse and update subscription options from Description field
	optionsStr := ExtractOptionsFromDescription(config.Description)
	if optionsStr != "" {
		if opts, err := ParseOptions(optionsStr); err == nil {
			// Update subscriptions (ignoring errors for backward compatibility)
			_ = p.Subscriptions.Update(opts)
		}
	}
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

// HasSubscription checks if the peer has a dynamic subscription for the given talkgroup
func (p *Peer) HasSubscription(tgid uint32, timeslot int) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Convert int timeslot to uint8 for subscription check
	var ts uint8
	switch timeslot {
	case 1:
		ts = 1
	case 2:
		ts = 2
	default:
		return false
	}

	return p.Subscriptions.HasTalkgroup(tgid, ts)
}

// GetSubscriptions returns the current subscription state (read-only)
func (p *Peer) GetSubscriptions() *SubscriptionState {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.Subscriptions
}

// UpdateSubscriptions updates the peer's subscription state
func (p *Peer) UpdateSubscriptions(opts *SubscriptionOptions) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.Subscriptions.Update(opts)
}

// SetRepeatMode enables or disables repeat mode for this peer
func (p *Peer) SetRepeatMode(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.RepeatMode = enabled
}

// GetRepeatMode returns whether repeat mode is enabled for this peer
func (p *Peer) GetRepeatMode() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.RepeatMode
}
