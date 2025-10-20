package metrics

import (
	"sync"
)

// Collector collects DMR-Nexus metrics
type Collector struct {
	mu sync.RWMutex

	// Peer metrics
	totalPeers  uint64
	activePeers map[uint32]bool

	// Packet metrics
	packetsReceived uint64
	packetsSent     uint64
	bytesReceived   uint64
	bytesSent       uint64

	// Stream metrics
	activeStreams map[uint32]bool

	// Bridge metrics
	bridgeRoutes uint64

	// Talkgroup metrics
	activeTalkgroups map[string]bool // key: "tgid:timeslot"
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		activePeers:      make(map[uint32]bool),
		activeStreams:    make(map[uint32]bool),
		activeTalkgroups: make(map[string]bool),
	}
}

// PeerConnected records a peer connection
func (c *Collector) PeerConnected(peerID uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalPeers++
	c.activePeers[peerID] = true
}

// PeerDisconnected records a peer disconnection
func (c *Collector) PeerDisconnected(peerID uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.activePeers, peerID)
}

// PacketReceived records a received packet
func (c *Collector) PacketReceived(packetType string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.packetsReceived++
}

// PacketSent records a sent packet
func (c *Collector) PacketSent(packetType string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.packetsSent++
}

// BytesReceived records received bytes
func (c *Collector) BytesReceived(bytes uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.bytesReceived += bytes
}

// BytesSent records sent bytes
func (c *Collector) BytesSent(bytes uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.bytesSent += bytes
}

// StreamStarted records a stream start
func (c *Collector) StreamStarted(streamID uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.activeStreams[streamID] = true
}

// StreamEnded records a stream end
func (c *Collector) StreamEnded(streamID uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.activeStreams, streamID)
}

// BridgeRouted records a bridge routing event
func (c *Collector) BridgeRouted(bridgeName, system string, tgid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.bridgeRoutes++
}

// TalkgroupActive records a talkgroup becoming active
func (c *Collector) TalkgroupActive(tgid uint32, timeslot uint8) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := talkgroupKey(tgid, timeslot)
	c.activeTalkgroups[key] = true
}

// TalkgroupInactive records a talkgroup becoming inactive
func (c *Collector) TalkgroupInactive(tgid uint32, timeslot uint8) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := talkgroupKey(tgid, timeslot)
	delete(c.activeTalkgroups, key)
}

// Reset resets all metrics (useful for testing)
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.activePeers = make(map[uint32]bool)
	c.activeStreams = make(map[uint32]bool)
	c.activeTalkgroups = make(map[string]bool)
	// Note: We don't reset total counters like totalPeers, packetsReceived, etc.
	// as those are cumulative
}

// Getters for metrics

// GetTotalPeers returns total peer connections
func (c *Collector) GetTotalPeers() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.totalPeers
}

// GetActivePeers returns the number of active peers
func (c *Collector) GetActivePeers() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.activePeers)
}

// GetPacketsReceived returns total packets received
func (c *Collector) GetPacketsReceived() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.packetsReceived
}

// GetPacketsSent returns total packets sent
func (c *Collector) GetPacketsSent() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.packetsSent
}

// GetBytesReceived returns total bytes received
func (c *Collector) GetBytesReceived() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.bytesReceived
}

// GetBytesSent returns total bytes sent
func (c *Collector) GetBytesSent() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.bytesSent
}

// GetActiveStreams returns the number of active streams
func (c *Collector) GetActiveStreams() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.activeStreams)
}

// GetBridgeRoutes returns total bridge routing events
func (c *Collector) GetBridgeRoutes() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.bridgeRoutes
}

// GetActiveTalkgroups returns the number of active talkgroups
func (c *Collector) GetActiveTalkgroups() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.activeTalkgroups)
}

func talkgroupKey(tgid uint32, timeslot uint8) string {
	return string([]byte{
		byte(tgid >> 24),
		byte(tgid >> 16),
		byte(tgid >> 8),
		byte(tgid),
		timeslot,
	})
}
