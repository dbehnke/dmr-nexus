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
	bridges              map[string]*BridgeRuleSet
	streamTracker        *StreamTracker
	subscriptionChecker  PeerSubscriptionChecker
	peerIDToSystemName   map[uint32]string // Maps peer IDs to system names
	mu                   sync.RWMutex
}

// NewRouter creates a new router instance
func NewRouter() *Router {
	return &Router{
		bridges:            make(map[string]*BridgeRuleSet),
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
