//go:build integration
// +build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/dbehnke/dmr-nexus/internal/testhelpers"
	"github.com/dbehnke/dmr-nexus/pkg/metrics"
	"github.com/dbehnke/dmr-nexus/pkg/mqtt"
	"github.com/dbehnke/dmr-nexus/pkg/protocol"
)

// TestMQTTEventPublishing tests MQTT event publishing functionality
func TestMQTTEventPublishing(t *testing.T) {
	suite := testhelpers.NewIntegrationSuite(t)
	defer suite.Cleanup()

	// Create MQTT publisher (disabled for testing)
	config := mqtt.Config{
		Enabled:     false,
		TopicPrefix: "dmr/test",
	}
	publisher := mqtt.New(config, suite.Logger)

	// Test publishing peer connect event
	connectEvent := mqtt.PeerConnectEvent{
		PeerID:    312000,
		Callsign:  "W1ABC",
		Timestamp: time.Now(),
	}

	err := publisher.PublishPeerConnect(connectEvent)
	if err != nil {
		t.Errorf("Failed to publish peer connect event: %v", err)
	}

	// Test publishing traffic event
	trafficEvent := mqtt.TrafficEvent{
		SourceID:  123456,
		DestID:    3100,
		Timeslot:  1,
		StreamID:  12345678,
		Timestamp: time.Now(),
	}

	err = publisher.PublishTraffic(trafficEvent)
	if err != nil {
		t.Errorf("Failed to publish traffic event: %v", err)
	}

	// Test publishing bridge event
	bridgeEvent := mqtt.BridgeEvent{
		BridgeName: "NATIONWIDE",
		System:     "MASTER-1",
		TGID:       3100,
		Timeslot:   1,
		Active:     true,
		Timestamp:  time.Now(),
	}

	err = publisher.PublishBridgeChange(bridgeEvent)
	if err != nil {
		t.Errorf("Failed to publish bridge event: %v", err)
	}
}

// TestMetricsCollection tests metrics collection functionality
func TestMetricsCollection(t *testing.T) {
	suite := testhelpers.NewIntegrationSuite(t)
	defer suite.Cleanup()

	collector := metrics.NewCollector()

	// Simulate peer connections
	for i := 0; i < 5; i++ {
		collector.PeerConnected(uint32(312000 + i))
	}

	// Check peer metrics
	if collector.GetTotalPeers() != 5 {
		t.Errorf("Expected 5 total peers, got %d", collector.GetTotalPeers())
	}
	if collector.GetActivePeers() != 5 {
		t.Errorf("Expected 5 active peers, got %d", collector.GetActivePeers())
	}

	// Simulate packet traffic
	for i := 0; i < 100; i++ {
		collector.PacketReceived("DMRD")
		collector.PacketSent("DMRD")
		collector.BytesReceived(53)
		collector.BytesSent(53)
	}

	if collector.GetPacketsReceived() != 100 {
		t.Errorf("Expected 100 packets received, got %d", collector.GetPacketsReceived())
	}
	if collector.GetBytesReceived() != 5300 {
		t.Errorf("Expected 5300 bytes received, got %d", collector.GetBytesReceived())
	}

	// Simulate stream activity
	streamID := uint32(12345678)
	collector.StreamStarted(streamID)
	if collector.GetActiveStreams() != 1 {
		t.Errorf("Expected 1 active stream, got %d", collector.GetActiveStreams())
	}

	collector.StreamEnded(streamID)
	if collector.GetActiveStreams() != 0 {
		t.Errorf("Expected 0 active streams, got %d", collector.GetActiveStreams())
	}

	// Simulate bridge routing
	for i := 0; i < 10; i++ {
		collector.BridgeRouted("NATIONWIDE", "MASTER-1", 3100)
	}

	if collector.GetBridgeRoutes() != 10 {
		t.Errorf("Expected 10 bridge routes, got %d", collector.GetBridgeRoutes())
	}

	// Test peer disconnection
	collector.PeerDisconnected(312000)
	if collector.GetActivePeers() != 4 {
		t.Errorf("Expected 4 active peers after disconnect, got %d", collector.GetActivePeers())
	}
}

// TestMockPeerPackets tests mock peer packet handling
func TestMockPeerPackets(t *testing.T) {
	suite := testhelpers.NewIntegrationSuite(t)
	defer suite.Cleanup()

	peer := suite.CreateMockPeer(312000, "password", "W1ABC")

	// Test RPTL packet creation
	rptl := &protocol.RPTLPacket{
		RepeaterID: peer.PeerID,
	}
	data, err := rptl.Encode()
	if err != nil {
		t.Fatalf("Failed to encode RPTL: %v", err)
	}
	if len(data) != protocol.RPTLPacketSize {
		t.Errorf("Expected RPTL size %d, got %d", protocol.RPTLPacketSize, len(data))
	}

	// Test DMRD packet creation
	payload := make([]byte, 33)
	for i := range payload {
		payload[i] = byte(i)
	}

	dmrd := &protocol.DMRDPacket{
		Sequence:      0,
		SourceID:      123456,
		DestinationID: 3100,
		RepeaterID:    peer.PeerID,
		Timeslot:      protocol.Timeslot1,
		CallType:      protocol.CallTypeGroup,
		StreamID:      12345678,
		Payload:       payload,
	}

	data, err = dmrd.Encode()
	if err != nil {
		t.Fatalf("Failed to encode DMRD: %v", err)
	}
	if len(data) != protocol.DMRDPacketSize {
		t.Errorf("Expected DMRD size %d, got %d", protocol.DMRDPacketSize, len(data))
	}

	// Parse it back
	parsed := &protocol.DMRDPacket{}
	err = parsed.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse DMRD: %v", err)
	}

	if parsed.SourceID != dmrd.SourceID {
		t.Errorf("Source ID mismatch: expected %d, got %d", dmrd.SourceID, parsed.SourceID)
	}
	if parsed.DestinationID != dmrd.DestinationID {
		t.Errorf("Dest ID mismatch: expected %d, got %d", dmrd.DestinationID, parsed.DestinationID)
	}
	if parsed.StreamID != dmrd.StreamID {
		t.Errorf("Stream ID mismatch: expected %d, got %d", dmrd.StreamID, parsed.StreamID)
	}
}

// TestMultipleMockPeers tests handling multiple mock peers
func TestMultipleMockPeers(t *testing.T) {
	suite := testhelpers.NewIntegrationSuite(t)
	defer suite.Cleanup()

	// Create multiple peers
	numPeers := 10
	peers := make([]*testhelpers.MockPeer, numPeers)
	for i := 0; i < numPeers; i++ {
		callsign := fmt.Sprintf("PEER%d", i)
		peers[i] = suite.CreateMockPeer(
			uint32(312000+i),
			"password",
			callsign,
		)
	}

	// Verify all peers were created
	if len(suite.MockPeers) != numPeers {
		t.Errorf("Expected %d mock peers, got %d", numPeers, len(suite.MockPeers))
	}

	// Each peer should have unique ID
	ids := make(map[uint32]bool)
	for _, peer := range peers {
		if ids[peer.PeerID] {
			t.Errorf("Duplicate peer ID: %d", peer.PeerID)
		}
		ids[peer.PeerID] = true
	}

	if len(ids) != numPeers {
		t.Errorf("Expected %d unique peer IDs, got %d", numPeers, len(ids))
	}
}

// TestMetricsConcurrency tests concurrent metrics updates
func TestMetricsConcurrency(t *testing.T) {
	suite := testhelpers.NewIntegrationSuite(t)
	defer suite.Cleanup()

	collector := metrics.NewCollector()

	// Simulate concurrent peer connections
	const maxConcurrentPeers = 50
	done := make(chan bool, maxConcurrentPeers)

	for i := 0; i < maxConcurrentPeers; i++ {
		go func(id int) {
			peerID := uint32(312000 + id)
			collector.PeerConnected(peerID)

			// Simulate some activity
			for j := 0; j < 10; j++ {
				collector.PacketReceived("DMRD")
				collector.BytesReceived(53)
			}

			collector.PeerDisconnected(peerID)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < maxConcurrentPeers; i++ {
		<-done
	}

	// Verify metrics
	expectedPackets := uint64(maxConcurrentPeers * 10)
	expectedBytes := uint64(maxConcurrentPeers * 10 * 53)

	if collector.GetPacketsReceived() != expectedPackets {
		t.Errorf("Expected %d packets, got %d", expectedPackets, collector.GetPacketsReceived())
	}

	if collector.GetBytesReceived() != expectedBytes {
		t.Errorf("Expected %d bytes, got %d", expectedBytes, collector.GetBytesReceived())
	}

	// All peers should be disconnected
	if collector.GetActivePeers() != 0 {
		t.Errorf("Expected 0 active peers, got %d", collector.GetActivePeers())
	}
}

// TestIntegrationSuite_WaitForAdvanced tests advanced WaitFor scenarios
func TestIntegrationSuite_WaitForAdvanced(t *testing.T) {
	suite := testhelpers.NewIntegrationSuite(t)
	defer suite.Cleanup()

	collector := metrics.NewCollector()

	// Start background goroutine that will eventually meet condition
	go func() {
		time.Sleep(100 * time.Millisecond)
		for i := 0; i < 10; i++ {
			collector.PacketReceived("DMRD")
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Wait for packets to be received
	condition := func() bool {
		return collector.GetPacketsReceived() >= 10
	}

	success := suite.WaitFor(condition, 2*time.Second, "10 packets received")
	if !success {
		t.Error("WaitFor failed: expected 10 packets to be received")
	}

	if collector.GetPacketsReceived() < 10 {
		t.Errorf("Expected at least 10 packets, got %d", collector.GetPacketsReceived())
	}
}
