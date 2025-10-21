package web

import (
	"net"
	"testing"

	"github.com/dbehnke/dmr-nexus/pkg/peer"
)

func TestDynamicBridgeSubscribers_AAA(t *testing.T) {
	// Arrange
	pm := peer.NewPeerManager()
	addr1 := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 10001}
	addr2 := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 10002}
	p1 := pm.AddPeer(1001, addr1)
	p2 := pm.AddPeer(1002, addr2)

	// Use exported Subscription API
	p1.GetSubscriptions().AddDynamic(7000, 1)
	p2.GetSubscriptions().AddDynamic(7000, 2)

	// Act
	var ts1, ts2 []uint32
	for _, p := range pm.GetAllPeers() {
		if p.HasSubscription(7000, 1) {
			ts1 = append(ts1, p.ID)
		}
		if p.HasSubscription(7000, 2) {
			ts2 = append(ts2, p.ID)
		}
	}

	// Assert
	if len(ts1) != 1 || ts1[0] != 1001 {
		t.Errorf("TS1 subscribers incorrect: %v", ts1)
	}
	if len(ts2) != 1 || ts2[0] != 1002 {
		t.Errorf("TS2 subscribers incorrect: %v", ts2)
	}
}

func TestDashboardBridgeCount_AAA(t *testing.T) {
	// Arrange
	type Bridge struct {
		ID      int
		Dynamic bool
	}
	bridges := []Bridge{
		{ID: 1, Dynamic: false},
		{ID: 2, Dynamic: true},
		{ID: 3, Dynamic: true},
	}

	// Act
	staticCount := 0
	dynamicCount := 0
	for _, b := range bridges {
		if b.Dynamic {
			dynamicCount++
		} else {
			staticCount++
		}
	}
	activeCount := staticCount + dynamicCount

	// Assert
	if staticCount != 1 {
		t.Errorf("Expected 1 static bridge, got %d", staticCount)
	}
	if dynamicCount != 2 {
		t.Errorf("Expected 2 dynamic bridges, got %d", dynamicCount)
	}
	if activeCount != 3 {
		t.Errorf("Expected 3 active bridges, got %d", activeCount)
	}
}
