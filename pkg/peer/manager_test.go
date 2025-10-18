package peer

import (
	"net"
	"testing"
	"time"
)

func TestPeerManager_New(t *testing.T) {
	mgr := NewPeerManager()

	if mgr == nil {
		t.Fatal("NewPeerManager returned nil")
	}

	count := mgr.Count()
	if count != 0 {
		t.Errorf("Expected 0 peers, got %d", count)
	}
}

func TestPeerManager_AddPeer(t *testing.T) {
	mgr := NewPeerManager()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}

	peer := mgr.AddPeer(312000, addr)

	if peer == nil {
		t.Fatal("AddPeer returned nil")
	}

	if peer.ID != 312000 {
		t.Errorf("Expected peer ID 312000, got %d", peer.ID)
	}

	if mgr.Count() != 1 {
		t.Errorf("Expected 1 peer, got %d", mgr.Count())
	}
}

func TestPeerManager_GetPeer(t *testing.T) {
	mgr := NewPeerManager()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}

	// Add a peer
	added := mgr.AddPeer(312000, addr)

	// Get the peer
	retrieved := mgr.GetPeer(312000)

	if retrieved == nil {
		t.Fatal("GetPeer returned nil")
	}

	if retrieved.ID != added.ID {
		t.Errorf("Expected peer ID %d, got %d", added.ID, retrieved.ID)
	}

	// Try to get non-existent peer
	notFound := mgr.GetPeer(999999)
	if notFound != nil {
		t.Error("Expected nil for non-existent peer")
	}
}

func TestPeerManager_GetPeerByAddress(t *testing.T) {
	mgr := NewPeerManager()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}

	// Add a peer
	added := mgr.AddPeer(312000, addr)

	// Get peer by address
	retrieved := mgr.GetPeerByAddress(addr)

	if retrieved == nil {
		t.Fatal("GetPeerByAddress returned nil")
	}

	if retrieved.ID != added.ID {
		t.Errorf("Expected peer ID %d, got %d", added.ID, retrieved.ID)
	}

	// Try to get peer with different address
	otherAddr := &net.UDPAddr{IP: net.ParseIP("192.168.1.101"), Port: 62031}
	notFound := mgr.GetPeerByAddress(otherAddr)
	if notFound != nil {
		t.Error("Expected nil for non-existent address")
	}
}

func TestPeerManager_RemovePeer(t *testing.T) {
	mgr := NewPeerManager()
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}

	// Add a peer
	mgr.AddPeer(312000, addr)

	if mgr.Count() != 1 {
		t.Errorf("Expected 1 peer before removal, got %d", mgr.Count())
	}

	// Remove the peer
	mgr.RemovePeer(312000)

	if mgr.Count() != 0 {
		t.Errorf("Expected 0 peers after removal, got %d", mgr.Count())
	}

	// Verify peer is gone
	peer := mgr.GetPeer(312000)
	if peer != nil {
		t.Error("Expected peer to be removed")
	}
}

func TestPeerManager_GetAllPeers(t *testing.T) {
	mgr := NewPeerManager()

	// Add multiple peers
	ids := []uint32{312000, 312001, 312002}
	for _, id := range ids {
		addr := &net.UDPAddr{
			IP:   net.ParseIP("192.168.1.100"),
			Port: int(id),
		}
		mgr.AddPeer(id, addr)
	}

	// Get all peers
	peers := mgr.GetAllPeers()

	if len(peers) != len(ids) {
		t.Errorf("Expected %d peers, got %d", len(ids), len(peers))
	}

	// Verify all peers are present
	for _, id := range ids {
		found := false
		for _, peer := range peers {
			if peer.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Peer %d not found in GetAllPeers", id)
		}
	}
}

func TestPeerManager_Count(t *testing.T) {
	mgr := NewPeerManager()

	if mgr.Count() != 0 {
		t.Errorf("Expected 0 peers initially, got %d", mgr.Count())
	}

	// Add peers
	for i := uint32(1); i <= 5; i++ {
		addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: int(62030 + i)}
		mgr.AddPeer(312000+i, addr)

		if mgr.Count() != int(i) {
			t.Errorf("Expected %d peers, got %d", i, mgr.Count())
		}
	}

	// Remove peers
	for i := uint32(1); i <= 5; i++ {
		mgr.RemovePeer(312000 + i)

		if mgr.Count() != int(5-i) {
			t.Errorf("Expected %d peers after removal, got %d", 5-i, mgr.Count())
		}
	}
}

func TestPeerManager_CleanupTimedOutPeers(t *testing.T) {
	mgr := NewPeerManager()

	// Add peers
	addr1 := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}
	peer1 := mgr.AddPeer(312000, addr1)

	addr2 := &net.UDPAddr{IP: net.ParseIP("192.168.1.101"), Port: 62031}
	mgr.AddPeer(312001, addr2)

	// Wait a bit first
	time.Sleep(20 * time.Millisecond)

	// Update LastHeard for peer1 only AFTER the wait
	peer1.UpdateLastHeard()

	// Cleanup with 10ms timeout should remove peer2 but not peer1
	removed := mgr.CleanupTimedOutPeers(10 * time.Millisecond)

	if removed != 1 {
		t.Errorf("Expected 1 peer removed, got %d", removed)
	}

	if mgr.Count() != 1 {
		t.Errorf("Expected 1 peer remaining, got %d", mgr.Count())
	}

	if mgr.GetPeer(312000) == nil {
		t.Error("Expected peer1 to remain")
	}

	if mgr.GetPeer(312001) != nil {
		t.Error("Expected peer2 to be removed")
	}
}

func TestPeerManager_Concurrent(t *testing.T) {
	mgr := NewPeerManager()

	// Test concurrent operations
	done := make(chan bool)

	// Add peers concurrently
	for i := 0; i < 10; i++ {
		go func(id uint32) {
			addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: int(62030 + id)}
			mgr.AddPeer(312000+id, addr)
			done <- true
		}(uint32(i))
	}

	// Wait for all adds
	for i := 0; i < 10; i++ {
		<-done
	}

	if mgr.Count() != 10 {
		t.Errorf("Expected 10 peers after concurrent adds, got %d", mgr.Count())
	}

	// Get peers concurrently
	for i := 0; i < 10; i++ {
		go func(id uint32) {
			_ = mgr.GetPeer(312000 + id)
			done <- true
		}(uint32(i))
	}

	// Wait for all gets
	for i := 0; i < 10; i++ {
		<-done
	}

	// Remove peers concurrently
	for i := 0; i < 10; i++ {
		go func(id uint32) {
			mgr.RemovePeer(312000 + id)
			done <- true
		}(uint32(i))
	}

	// Wait for all removes
	for i := 0; i < 10; i++ {
		<-done
	}

	if mgr.Count() != 0 {
		t.Errorf("Expected 0 peers after concurrent removes, got %d", mgr.Count())
	}
}

func TestPeerManager_AddPeer_UpdatesExisting(t *testing.T) {
	mgr := NewPeerManager()
	addr1 := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}

	// Add peer first time
	peer1 := mgr.AddPeer(312000, addr1)
	peer1.Callsign = "W1ABC"

	// Add same peer again with different address
	addr2 := &net.UDPAddr{IP: net.ParseIP("192.168.1.101"), Port: 62032}
	mgr.AddPeer(312000, addr2)

	// Should still have only 1 peer
	if mgr.Count() != 1 {
		t.Errorf("Expected 1 peer, got %d", mgr.Count())
	}

	// Get updated peer
	updated := mgr.GetPeer(312000)
	if updated == nil {
		t.Fatal("Expected to find peer after update")
	}

	// Verify address was updated
	if updated.Address.String() != addr2.String() {
		t.Errorf("Expected address to be updated to %s, got %s", addr2.String(), updated.Address.String())
	}

	// Verify it's the same peer (callsign preserved)
	if updated.Callsign != "W1ABC" {
		t.Error("Expected callsign to be preserved when updating peer")
	}
}
