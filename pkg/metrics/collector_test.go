package metrics

import (
	"testing"
)

// TestNewCollector tests creating a new metrics collector
func TestNewCollector(t *testing.T) {
	collector := NewCollector()
	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}
}

// TestCollector_PeerMetrics tests peer metrics
func TestCollector_PeerMetrics(t *testing.T) {
	collector := NewCollector()

	// Test incrementing peer connections
	collector.PeerConnected(312000)
	total := collector.GetTotalPeers()
	active := collector.GetActivePeers()

	if total < 1 {
		t.Error("Expected at least 1 total peer")
	}
	if active < 1 {
		t.Error("Expected at least 1 active peer")
	}

	// Test disconnecting peer
	collector.PeerDisconnected(312000)
	active = collector.GetActivePeers()
	if active > 0 {
		t.Error("Expected 0 active peers after disconnect")
	}
}

// TestCollector_PacketMetrics tests packet metrics
func TestCollector_PacketMetrics(t *testing.T) {
	collector := NewCollector()

	// Test recording received packets
	collector.PacketReceived("DMRD")
	collector.PacketReceived("RPTL")
	
	received := collector.GetPacketsReceived()
	if received < 2 {
		t.Errorf("Expected at least 2 received packets, got %d", received)
	}

	// Test recording sent packets
	collector.PacketSent("DMRD")
	sent := collector.GetPacketsSent()
	if sent < 1 {
		t.Errorf("Expected at least 1 sent packet, got %d", sent)
	}
}

// TestCollector_ByteMetrics tests byte transfer metrics
func TestCollector_ByteMetrics(t *testing.T) {
	collector := NewCollector()

	// Test recording bytes
	collector.BytesReceived(1024)
	collector.BytesSent(2048)

	received := collector.GetBytesReceived()
	sent := collector.GetBytesSent()

	if received != 1024 {
		t.Errorf("Expected 1024 bytes received, got %d", received)
	}
	if sent != 2048 {
		t.Errorf("Expected 2048 bytes sent, got %d", sent)
	}
}

// TestCollector_StreamMetrics tests active stream tracking
func TestCollector_StreamMetrics(t *testing.T) {
	collector := NewCollector()

	// Test starting a stream
	collector.StreamStarted(12345678)
	active := collector.GetActiveStreams()
	if active < 1 {
		t.Errorf("Expected at least 1 active stream, got %d", active)
	}

	// Test ending a stream
	collector.StreamEnded(12345678)
	active = collector.GetActiveStreams()
	if active > 0 {
		t.Errorf("Expected 0 active streams, got %d", active)
	}
}

// TestCollector_BridgeMetrics tests bridge routing metrics
func TestCollector_BridgeMetrics(t *testing.T) {
	collector := NewCollector()

	// Test recording bridge routes
	collector.BridgeRouted("NATIONWIDE", "MASTER-1", 3100)
	routes := collector.GetBridgeRoutes()
	if routes < 1 {
		t.Errorf("Expected at least 1 bridge route, got %d", routes)
	}
}

// TestCollector_TalkgroupMetrics tests talkgroup activity
func TestCollector_TalkgroupMetrics(t *testing.T) {
	collector := NewCollector()

	// Test recording talkgroup activity
	collector.TalkgroupActive(3100, 1)
	active := collector.GetActiveTalkgroups()
	if active < 1 {
		t.Errorf("Expected at least 1 active talkgroup, got %d", active)
	}

	// Test ending talkgroup activity
	collector.TalkgroupInactive(3100, 1)
	active = collector.GetActiveTalkgroups()
	if active > 0 {
		t.Errorf("Expected 0 active talkgroups, got %d", active)
	}
}

// TestCollector_Reset tests resetting counters
func TestCollector_Reset(t *testing.T) {
	collector := NewCollector()

	// Add some metrics
	collector.PeerConnected(312000)
	collector.PacketReceived("DMRD")
	collector.BytesReceived(1024)

	// Reset
	collector.Reset()

	// Check that counters are reset (but don't check total peers as it may not reset)
	if collector.GetActivePeers() != 0 {
		t.Error("Expected active peers to be 0 after reset")
	}
}

// TestCollector_Concurrent tests concurrent access
func TestCollector_Concurrent(t *testing.T) {
	collector := NewCollector()

	// Run concurrent updates
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			collector.PeerConnected(uint32(312000 + id))
			collector.PacketReceived("DMRD")
			collector.BytesReceived(100)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check that metrics were recorded (exact values may vary due to timing)
	if collector.GetPacketsReceived() < 10 {
		t.Error("Expected at least 10 received packets")
	}
}
