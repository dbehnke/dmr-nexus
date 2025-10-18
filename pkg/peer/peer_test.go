package peer

import (
	"net"
	"testing"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/protocol"
)

func TestPeer_New(t *testing.T) {
	addr := &net.UDPAddr{
		IP:   net.ParseIP("192.168.1.100"),
		Port: 62031,
	}

	peer := NewPeer(312000, addr)

	if peer == nil {
		t.Fatal("NewPeer returned nil")
	}

	if peer.ID != 312000 {
		t.Errorf("Expected peer ID 312000, got %d", peer.ID)
	}

	if peer.Address.String() != addr.String() {
		t.Errorf("Expected address %s, got %s", addr.String(), peer.Address.String())
	}

	if peer.State != StateDisconnected {
		t.Errorf("Expected initial state StateDisconnected, got %v", peer.State)
	}

	if !peer.ConnectedAt.IsZero() {
		t.Error("ConnectedAt should be zero for new peer")
	}

	if !peer.LastHeard.IsZero() {
		t.Error("LastHeard should be zero for new peer")
	}
}

func TestPeer_SetState(t *testing.T) {
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}
	peer := NewPeer(312000, addr)

	states := []ConnectionState{
		StateRPTLReceived,
		StateAuthenticated,
		StateConfigReceived,
		StateConnected,
	}

	for _, state := range states {
		peer.SetState(state)
		if peer.GetState() != state {
			t.Errorf("Expected state %v, got %v", state, peer.GetState())
		}
	}
}

func TestPeer_UpdateLastHeard(t *testing.T) {
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}
	peer := NewPeer(312000, addr)

	before := time.Now()
	time.Sleep(1 * time.Millisecond)
	peer.UpdateLastHeard()
	time.Sleep(1 * time.Millisecond)
	after := time.Now()

	lastHeard := peer.GetLastHeard()
	if lastHeard.Before(before) || lastHeard.After(after) {
		t.Errorf("LastHeard %v not between %v and %v", lastHeard, before, after)
	}
}

func TestPeer_IsTimedOut(t *testing.T) {
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}
	peer := NewPeer(312000, addr)

	// New peer with no LastHeard should timeout
	if !peer.IsTimedOut(1 * time.Second) {
		t.Error("New peer should timeout immediately")
	}

	// Update LastHeard and check timeout
	peer.UpdateLastHeard()
	if peer.IsTimedOut(5 * time.Second) {
		t.Error("Peer should not timeout within 5 seconds")
	}

	// Wait and check timeout
	time.Sleep(10 * time.Millisecond)
	if !peer.IsTimedOut(5 * time.Millisecond) {
		t.Error("Peer should timeout after 5ms")
	}
}

func TestPeer_SetConnected(t *testing.T) {
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}
	peer := NewPeer(312000, addr)

	before := time.Now()
	peer.SetConnected()
	after := time.Now()

	if peer.GetState() != StateConnected {
		t.Errorf("Expected state StateConnected, got %v", peer.GetState())
	}

	connectedAt := peer.GetConnectedAt()
	if connectedAt.Before(before) || connectedAt.After(after) {
		t.Errorf("ConnectedAt %v not between %v and %v", connectedAt, before, after)
	}
}

func TestPeer_SetConfig(t *testing.T) {
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}
	peer := NewPeer(312000, addr)

	config := &protocol.RPTCPacket{
		RepeaterID:  312000,
		Callsign:    "W1ABC",
		RXFreq:      "449000000",
		TXFreq:      "444000000",
		TXPower:     "25",
		ColorCode:   "1",
		Latitude:    "38.0000",
		Longitude:   "-095.0000",
		Height:      "75",
		Location:    "Boston, MA",
		Description: "Test Repeater",
		URL:         "https://w1abc.org",
		SoftwareID:  "DMR-Nexus",
		PackageID:   "v1.0.0",
	}

	peer.SetConfig(config)

	if peer.Callsign != "W1ABC" {
		t.Errorf("Expected callsign W1ABC, got %s", peer.Callsign)
	}

	if peer.Location != "Boston, MA" {
		t.Errorf("Expected location 'Boston, MA', got %s", peer.Location)
	}

	if peer.Description != "Test Repeater" {
		t.Errorf("Expected description 'Test Repeater', got %s", peer.Description)
	}
}

func TestPeer_UpdateStats(t *testing.T) {
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}
	peer := NewPeer(312000, addr)

	// Initial stats should be zero
	if peer.PacketsReceived != 0 {
		t.Errorf("Expected 0 packets received, got %d", peer.PacketsReceived)
	}

	if peer.BytesReceived != 0 {
		t.Errorf("Expected 0 bytes received, got %d", peer.BytesReceived)
	}

	// Update stats
	peer.IncrementPacketsReceived()
	peer.AddBytesReceived(53) // Standard DMRD packet

	if peer.PacketsReceived != 1 {
		t.Errorf("Expected 1 packet received, got %d", peer.PacketsReceived)
	}

	if peer.BytesReceived != 53 {
		t.Errorf("Expected 53 bytes received, got %d", peer.BytesReceived)
	}

	// Update multiple times
	for i := 0; i < 10; i++ {
		peer.IncrementPacketsReceived()
		peer.AddBytesReceived(53)
	}

	if peer.PacketsReceived != 11 {
		t.Errorf("Expected 11 packets received, got %d", peer.PacketsReceived)
	}

	if peer.BytesReceived != 583 {
		t.Errorf("Expected 583 bytes received, got %d", peer.BytesReceived)
	}
}

func TestPeer_GetUptime(t *testing.T) {
	addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 62031}
	peer := NewPeer(312000, addr)

	// No uptime before connected
	uptime := peer.GetUptime()
	if uptime != 0 {
		t.Errorf("Expected 0 uptime for unconnected peer, got %v", uptime)
	}

	// Set connected and check uptime
	peer.SetConnected()
	time.Sleep(10 * time.Millisecond)

	uptime = peer.GetUptime()
	if uptime < 10*time.Millisecond || uptime > 100*time.Millisecond {
		t.Errorf("Expected uptime between 10-100ms, got %v", uptime)
	}
}

func TestConnectionState_String(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{StateDisconnected, "disconnected"},
		{StateRPTLReceived, "rptl_received"},
		{StateAuthenticated, "authenticated"},
		{StateConfigReceived, "config_received"},
		{StateConnected, "connected"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.state.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.state.String())
			}
		})
	}
}
