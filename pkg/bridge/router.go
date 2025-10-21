package bridge

import (
	"sync"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/protocol"
)

// PeerSubscriptionChecker is a function that checks if a peer has a subscription
type PeerSubscriptionChecker func(peerID uint32, tgid uint32, timeslot int) bool

// Router manages conference bridge routing between systems
type Router struct {
	bridges             map[string]*BridgeRuleSet
	dynamicBridges      map[string]*DynamicBridge // key: "tgid:timeslot"
	streamTracker       *StreamTracker
	txLogger            *TransmissionLogger
	subscriptionChecker PeerSubscriptionChecker
	peerIDToSystemName  map[uint32]string // Maps peer IDs to system names
	mu                  sync.RWMutex
}

// DynamicBridge represents an automatically created bridge for a talkgroup
type DynamicBridge struct {
	TGID         uint32
	Timeslot     int
	CreatedAt    time.Time
	LastActivity time.Time
	Subscribers  map[uint32]bool // Peer IDs subscribed to this TG
	mu           sync.RWMutex
}

// NewRouter creates a new router instance
func NewRouter() *Router {
	return &Router{
		bridges:            make(map[string]*BridgeRuleSet),
		dynamicBridges:     make(map[string]*DynamicBridge),
		streamTracker:      NewStreamTracker(),
		peerIDToSystemName: make(map[uint32]string),
	}
}

// SetSubscriptionChecker sets the function to check peer subscriptions
func (r *Router) SetSubscriptionChecker(checker PeerSubscriptionChecker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subscriptionChecker = checker
}

// SetTransmissionLogger sets the transmission logger for the router
func (r *Router) SetTransmissionLogger(logger *TransmissionLogger) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.txLogger = logger
}

// RegisterPeer registers a peer ID to system name mapping
func (r *Router) RegisterPeer(peerID uint32, systemName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.peerIDToSystemName[peerID] = systemName
}

// UnregisterPeer removes a peer ID to system name mapping
func (r *Router) UnregisterPeer(peerID uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.peerIDToSystemName, peerID)
}

// AddBridge adds a bridge rule set to the router
func (r *Router) AddBridge(bridge *BridgeRuleSet) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bridges[bridge.Name] = bridge
}

// GetBridge retrieves a bridge by name
func (r *Router) GetBridge(name string) *BridgeRuleSet {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.bridges[name]
}

// RoutePacket routes a DMR packet based on bridge rules and peer subscriptions
// Returns a list of target systems to forward the packet to
func (r *Router) RoutePacket(packet *protocol.DMRDPacket, sourceSystem string) []string {
	// Log the transmission if logger is configured
	if r.txLogger != nil {
		isTerminator := packet.FrameType == protocol.FrameTypeVoiceTerminator
		r.txLogger.LogPacket(
			packet.StreamID,
			packet.SourceID,
			packet.DestinationID,
			packet.RepeaterID,
			packet.Timeslot,
			isTerminator,
		)
	}

	// Check if this is a terminator frame - end the stream after processing
	isTerminator := packet.FrameType == protocol.FrameTypeVoiceTerminator
	defer func() {
		if isTerminator {
			r.streamTracker.EndStream(packet.StreamID)
		}
	}()

	// Check for stream deduplication
	if !r.streamTracker.TrackStream(packet.StreamID, sourceSystem) {
		// Duplicate stream from this system - don't forward
		return []string{}
	}

	// Find matching bridge rules across all bridges
	targets := make([]string, 0)
	targetSet := make(map[string]bool) // Use set to avoid duplicates

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check static bridge rules
	for _, bridge := range r.bridges {
		matches := bridge.GetMatchingRules(packet.DestinationID, packet.Timeslot, sourceSystem)
		for _, rule := range matches {
			targetSet[rule.System] = true
		}
	}

	// Check dynamic peer subscriptions
	if r.subscriptionChecker != nil {
		for peerID, systemName := range r.peerIDToSystemName {
			// Skip the source system
			if systemName == sourceSystem {
				continue
			}

			// Check if this peer has a subscription for this talkgroup/timeslot
			if r.subscriptionChecker(peerID, packet.DestinationID, packet.Timeslot) {
				targetSet[systemName] = true
			}
		}
	}

	// Convert set to slice
	for target := range targetSet {
		targets = append(targets, target)
	}

	return targets
}

// ProcessActivation processes activation for the given TGID across all bridges
// Returns a map of bridge names to lists of activated rules
func (r *Router) ProcessActivation(tgid uint32) map[string][]*BridgeRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]*BridgeRule)

	for name, bridge := range r.bridges {
		activated := bridge.ProcessActivation(tgid)
		if len(activated) > 0 {
			result[name] = activated
		}
	}

	return result
}

// ProcessDeactivation processes deactivation for the given TGID across all bridges
// Returns a map of bridge names to lists of deactivated rules
func (r *Router) ProcessDeactivation(tgid uint32) map[string][]*BridgeRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]*BridgeRule)

	for name, bridge := range r.bridges {
		deactivated := bridge.ProcessDeactivation(tgid)
		if len(deactivated) > 0 {
			result[name] = deactivated
		}
	}

	return result
}

// GetActiveBridges returns all bridges that have at least one active rule
func (r *Router) GetActiveBridges() []*BridgeRuleSet {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*BridgeRuleSet, 0)

	for _, bridge := range r.bridges {
		hasActive := false
		bridge.mu.RLock()
		for _, rule := range bridge.Rules {
			rule.mu.RLock()
			if rule.Active {
				hasActive = true
				rule.mu.RUnlock()
				break
			}
			rule.mu.RUnlock()
		}
		bridge.mu.RUnlock()

		if hasActive {
			result = append(result, bridge)
		}
	}

	return result
}

// CleanupStreams removes old streams from the tracker
func (r *Router) CleanupStreams(maxAge time.Duration) {
	r.streamTracker.CleanupOldStreams(maxAge)
}

// GetOrCreateDynamicBridge gets or creates a dynamic bridge for a talkgroup
func (r *Router) GetOrCreateDynamicBridge(tgid uint32, timeslot int) *DynamicBridge {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := dynamicBridgeKey(tgid, timeslot)
	if bridge, exists := r.dynamicBridges[key]; exists {
		return bridge
	}

	// Create new dynamic bridge
	bridge := &DynamicBridge{
		TGID:         tgid,
		Timeslot:     timeslot,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Subscribers:  make(map[uint32]bool),
	}
	r.dynamicBridges[key] = bridge

	return bridge
}

// AddSubscriberToDynamicBridge adds a peer to a dynamic bridge's subscriber list
func (r *Router) AddSubscriberToDynamicBridge(tgid uint32, timeslot int, peerID uint32) {
	bridge := r.GetOrCreateDynamicBridge(tgid, timeslot)

	bridge.mu.Lock()
	bridge.Subscribers[peerID] = true
	bridge.LastActivity = time.Now()
	bridge.mu.Unlock()
}

// RemoveSubscriberFromDynamicBridge removes a peer from a dynamic bridge
func (r *Router) RemoveSubscriberFromDynamicBridge(tgid uint32, timeslot int, peerID uint32) {
	r.mu.RLock()
	key := dynamicBridgeKey(tgid, timeslot)
	bridge, exists := r.dynamicBridges[key]
	r.mu.RUnlock()

	if !exists {
		return
	}

	bridge.mu.Lock()
	delete(bridge.Subscribers, peerID)
	bridge.mu.Unlock()
}

// RemoveSubscriberFromAllDynamicBridges removes a peer from all dynamic bridges
// Returns the count of bridges the peer was removed from
func (r *Router) RemoveSubscriberFromAllDynamicBridges(peerID uint32) int {
	r.mu.RLock()
	bridges := make([]*DynamicBridge, 0, len(r.dynamicBridges))
	for _, bridge := range r.dynamicBridges {
		bridges = append(bridges, bridge)
	}
	r.mu.RUnlock()

	count := 0
	for _, bridge := range bridges {
		bridge.mu.Lock()
		if _, exists := bridge.Subscribers[peerID]; exists {
			delete(bridge.Subscribers, peerID)
			count++
		}
		bridge.mu.Unlock()
	}

	return count
}

// CleanupInactiveDynamicBridges removes dynamic bridges with no actual subscribers
// It checks if any peers are subscribed to each talkgroup (not the bridge's cached subscriber list)
// subscriberCountFunc should return the number of peers subscribed to the given talkgroup
func (r *Router) CleanupInactiveDynamicBridges(maxInactive time.Duration, subscriberCountFunc func(tgid uint32) int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	removed := make([]string, 0)

	for key, bridge := range r.dynamicBridges {
		bridge.mu.RLock()
		tgid := bridge.TGID
		lastActivity := bridge.LastActivity
		bridge.mu.RUnlock()

		// Get actual subscriber count from peer manager
		actualSubscriberCount := subscriberCountFunc(tgid)

		// Remove if no actual subscribers and inactive for the duration
		if actualSubscriberCount == 0 && now.Sub(lastActivity) > maxInactive {
			delete(r.dynamicBridges, key)
			removed = append(removed, key)
		}
	}

	return removed
}

// GetDynamicBridgeSubscribers returns the peer IDs subscribed to a dynamic bridge
func (r *Router) GetDynamicBridgeSubscribers(tgid uint32, timeslot int) []uint32 {
	r.mu.RLock()
	key := dynamicBridgeKey(tgid, timeslot)
	bridge, exists := r.dynamicBridges[key]
	r.mu.RUnlock()

	if !exists {
		return []uint32{}
	}

	bridge.mu.RLock()
	defer bridge.mu.RUnlock()

	result := make([]uint32, 0, len(bridge.Subscribers))
	for peerID := range bridge.Subscribers {
		result = append(result, peerID)
	}

	return result
}

// GetAllDynamicBridges returns a snapshot of all dynamic bridges
func (r *Router) GetAllDynamicBridges() []*DynamicBridge {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*DynamicBridge, 0, len(r.dynamicBridges))
	for _, bridge := range r.dynamicBridges {
		// Create a copy to avoid race conditions
		bridge.mu.RLock()
		bridgeCopy := &DynamicBridge{
			TGID:         bridge.TGID,
			Timeslot:     bridge.Timeslot,
			CreatedAt:    bridge.CreatedAt,
			LastActivity: bridge.LastActivity,
			Subscribers:  make(map[uint32]bool, len(bridge.Subscribers)),
		}
		for peerID := range bridge.Subscribers {
			bridgeCopy.Subscribers[peerID] = true
		}
		bridge.mu.RUnlock()

		result = append(result, bridgeCopy)
	}

	return result
}

// dynamicBridgeKey creates a unique key for a dynamic bridge
func dynamicBridgeKey(tgid uint32, timeslot int) string {
	return string([]byte{
		byte(tgid >> 24),
		byte(tgid >> 16),
		byte(tgid >> 8),
		byte(tgid),
		':',
		byte(timeslot),
	})
}
