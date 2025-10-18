package peer

import (
	"net"
	"sync"
	"time"
)

// PeerManager manages all connected peers in a thread-safe manner
type PeerManager struct {
	peers map[uint32]*Peer
	mu    sync.RWMutex
}

// NewPeerManager creates a new peer manager
func NewPeerManager() *PeerManager {
	return &PeerManager{
		peers: make(map[uint32]*Peer),
	}
}

// AddPeer adds a new peer or updates an existing peer's address
func (pm *PeerManager) AddPeer(id uint32, addr *net.UDPAddr) *Peer {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if peer already exists
	if peer, exists := pm.peers[id]; exists {
		// Update address if peer exists
		peer.Address = addr
		return peer
	}

	// Create new peer
	peer := NewPeer(id, addr)
	pm.peers[id] = peer
	return peer
}

// GetPeer retrieves a peer by ID
func (pm *PeerManager) GetPeer(id uint32) *Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.peers[id]
}

// GetPeerByAddress retrieves a peer by UDP address
func (pm *PeerManager) GetPeerByAddress(addr *net.UDPAddr) *Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, peer := range pm.peers {
		if peer.Address.String() == addr.String() {
			return peer
		}
	}

	return nil
}

// RemovePeer removes a peer by ID
func (pm *PeerManager) RemovePeer(id uint32) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.peers, id)
}

// GetAllPeers returns a slice of all peers
func (pm *PeerManager) GetAllPeers() []*Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]*Peer, 0, len(pm.peers))
	for _, peer := range pm.peers {
		peers = append(peers, peer)
	}

	return peers
}

// Count returns the number of connected peers
func (pm *PeerManager) Count() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.peers)
}

// CleanupTimedOutPeers removes peers that haven't been heard from in the given duration
// Returns the number of peers removed
func (pm *PeerManager) CleanupTimedOutPeers(timeout time.Duration) int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	removed := 0
	for id, peer := range pm.peers {
		if peer.IsTimedOut(timeout) {
			delete(pm.peers, id)
			removed++
		}
	}

	return removed
}
