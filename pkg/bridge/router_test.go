package bridge

import (
	"testing"

	"github.com/dbehnke/dmr-nexus/pkg/protocol"
)

func TestRouter_New(t *testing.T) {
	router := NewRouter()
	if router == nil {
		t.Fatal("NewRouter returned nil")
	}
}

func TestRouter_AddBridge(t *testing.T) {
	router := NewRouter()

	bridge := NewBridgeRuleSet("NATIONWIDE")
	router.AddBridge(bridge)

	if len(router.bridges) != 1 {
		t.Errorf("Expected 1 bridge, got %d", len(router.bridges))
	}
}

func TestRouter_GetBridge(t *testing.T) {
	router := NewRouter()

	bridge1 := NewBridgeRuleSet("NATIONWIDE")
	bridge2 := NewBridgeRuleSet("REGIONAL")

	router.AddBridge(bridge1)
	router.AddBridge(bridge2)

	result := router.GetBridge("NATIONWIDE")
	if result == nil {
		t.Fatal("GetBridge returned nil for NATIONWIDE")
	}
	if result.Name != "NATIONWIDE" {
		t.Errorf("Expected bridge name NATIONWIDE, got %s", result.Name)
	}

	result = router.GetBridge("NONEXISTENT")
	if result != nil {
		t.Error("GetBridge should return nil for non-existent bridge")
	}
}

func TestRouter_RoutePacket(t *testing.T) {
	router := NewRouter()

	// Create a bridge with two systems
	bridge := NewBridgeRuleSet("NATIONWIDE")
	rule1 := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
	}
	rule2 := &BridgeRule{
		System:   "SYSTEM2",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
	}
	bridge.AddRule(rule1)
	bridge.AddRule(rule2)
	router.AddBridge(bridge)

	// Create a DMRD packet
	packet := &protocol.DMRDPacket{
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    312000,
		Timeslot:      1,
		CallType:      protocol.CallTypeGroup,
		StreamID:      12345,
	}

	// Route from SYSTEM1 - should route to SYSTEM2
	targets := router.RoutePacket(packet, "SYSTEM1")

	if len(targets) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(targets))
	}

	if targets[0] != "SYSTEM2" {
		t.Errorf("Expected target SYSTEM2, got %s", targets[0])
	}
}

func TestRouter_RoutePacket_NoMatch(t *testing.T) {
	router := NewRouter()

	// Create a bridge
	bridge := NewBridgeRuleSet("NATIONWIDE")
	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
	}
	bridge.AddRule(rule)
	router.AddBridge(bridge)

	// Create a packet with non-matching TGID
	packet := &protocol.DMRDPacket{
		SourceID:      3120001,
		DestinationID: 9999,
		RepeaterID:    312000,
		Timeslot:      1,
		CallType:      protocol.CallTypeGroup,
		StreamID:      12345,
	}

	// Route from SYSTEM1 - should route to no systems
	targets := router.RoutePacket(packet, "SYSTEM1")

	if len(targets) != 0 {
		t.Errorf("Expected 0 targets, got %d", len(targets))
	}
}

func TestRouter_RoutePacket_DuplicateStream(t *testing.T) {
	router := NewRouter()

	// Create a bridge
	bridge := NewBridgeRuleSet("NATIONWIDE")
	rule1 := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
	}
	rule2 := &BridgeRule{
		System:   "SYSTEM2",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
	}
	bridge.AddRule(rule1)
	bridge.AddRule(rule2)
	router.AddBridge(bridge)

	// Create a packet
	packet := &protocol.DMRDPacket{
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    312000,
		Timeslot:      1,
		CallType:      protocol.CallTypeGroup,
		StreamID:      12345,
	}

	// First time - should route
	targets := router.RoutePacket(packet, "SYSTEM1")
	if len(targets) != 1 {
		t.Fatalf("Expected 1 target on first route, got %d", len(targets))
	}

	// Second time with same stream from same system - should not route (duplicate)
	targets = router.RoutePacket(packet, "SYSTEM1")
	if len(targets) != 0 {
		t.Errorf("Expected 0 targets on duplicate, got %d", len(targets))
	}
}

func TestRouter_RoutePacket_StreamTerminator(t *testing.T) {
	router := NewRouter()

	// Create a bridge
	bridge := NewBridgeRuleSet("NATIONWIDE")
	rule1 := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
	}
	rule2 := &BridgeRule{
		System:   "SYSTEM2",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
	}
	bridge.AddRule(rule1)
	bridge.AddRule(rule2)
	router.AddBridge(bridge)

	// Create a voice header packet
	packet := &protocol.DMRDPacket{
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    312000,
		Timeslot:      1,
		CallType:      protocol.CallTypeGroup,
		StreamID:      12345,
		FrameType:     protocol.FrameTypeVoiceHeader,
	}

	// Route voice header
	targets := router.RoutePacket(packet, "SYSTEM1")
	if len(targets) != 1 {
		t.Fatalf("Expected 1 target for header, got %d", len(targets))
	}

	// Create a voice terminator packet
	packet.FrameType = protocol.FrameTypeVoiceTerminator

	// Route terminator - should route and end stream
	targets = router.RoutePacket(packet, "SYSTEM1")
	if len(targets) != 0 {
		t.Errorf("Expected 0 targets for duplicate in same call, got %d", len(targets))
	}

	// Verify stream is no longer active
	if router.streamTracker.IsActive(12345) {
		t.Error("Stream should not be active after terminator")
	}
}

func TestRouter_ProcessActivation(t *testing.T) {
	router := NewRouter()

	// Create a bridge with activation rules
	bridge := NewBridgeRuleSet("NATIONWIDE")
	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   false,
		On:       []int{3100},
	}
	bridge.AddRule(rule)
	router.AddBridge(bridge)

	// Process activation
	activated := router.ProcessActivation(3100)

	if len(activated) == 0 {
		t.Error("Expected some rules to be activated")
	}

	if !rule.Active {
		t.Error("Rule should be activated")
	}
}

func TestRouter_ProcessDeactivation(t *testing.T) {
	router := NewRouter()

	// Create a bridge with deactivation rules
	bridge := NewBridgeRuleSet("NATIONWIDE")
	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
		Off:      []int{3101},
	}
	bridge.AddRule(rule)
	router.AddBridge(bridge)

	// Process deactivation
	deactivated := router.ProcessDeactivation(3101)

	if len(deactivated) == 0 {
		t.Error("Expected some rules to be deactivated")
	}

	if rule.Active {
		t.Error("Rule should be deactivated")
	}
}

func TestRouter_GetActiveBridges(t *testing.T) {
	router := NewRouter()

	// Create bridges
	bridge1 := NewBridgeRuleSet("NATIONWIDE")
	bridge2 := NewBridgeRuleSet("REGIONAL")

	rule1 := &BridgeRule{System: "SYSTEM1", TGID: 3100, Timeslot: 1, Active: true}
	rule2 := &BridgeRule{System: "SYSTEM2", TGID: 3200, Timeslot: 1, Active: false}

	bridge1.AddRule(rule1)
	bridge2.AddRule(rule2)

	router.AddBridge(bridge1)
	router.AddBridge(bridge2)

	// Get active bridges
	active := router.GetActiveBridges()

	// NATIONWIDE should be active (has active rule), REGIONAL should not
	if len(active) != 1 {
		t.Fatalf("Expected 1 active bridge, got %d", len(active))
	}

	if active[0].Name != "NATIONWIDE" {
		t.Errorf("Expected NATIONWIDE bridge, got %s", active[0].Name)
	}
}

func TestRouter_CleanupStreams(t *testing.T) {
	router := NewRouter()

	// Create a bridge
	bridge := NewBridgeRuleSet("NATIONWIDE")
	rule := &BridgeRule{System: "SYSTEM1", TGID: 3100, Timeslot: 1, Active: true}
	bridge.AddRule(rule)
	router.AddBridge(bridge)

	// Track a stream
	packet := &protocol.DMRDPacket{
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    312000,
		Timeslot:      1,
		CallType:      protocol.CallTypeGroup,
		StreamID:      12345,
	}

	router.RoutePacket(packet, "SYSTEM1")

	// Verify stream is active
	if !router.streamTracker.IsActive(12345) {
		t.Error("Stream should be active")
	}

	// Cleanup streams immediately (0 duration)
	router.CleanupStreams(0)

	// Stream should be cleaned up
	if router.streamTracker.IsActive(12345) {
		t.Error("Stream should be cleaned up")
	}
}
