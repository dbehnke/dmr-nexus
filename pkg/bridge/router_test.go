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

func TestRouter_RoutePacket_WithPeerSubscriptions(t *testing.T) {
	router := NewRouter()

	// Register some peers
	router.RegisterPeer(312000, "PEER-312000")
	router.RegisterPeer(312001, "PEER-312001")

	// Set up subscription checker - PEER-312001 has subscription for TG 3100 TS1
	router.SetSubscriptionChecker(func(peerID uint32, tgid uint32, timeslot int) bool {
		return peerID == 312001 && tgid == 3100 && timeslot == 1
	})

	// Create a packet for TG 3100
	packet := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      312000,
		DestinationID: 3100,
		RepeaterID:    312000,
		Timeslot:      1,
		FrameType:     protocol.FrameTypeVoiceHeader,
		StreamID:      12345,
	}

	// Route the packet - should include peer with dynamic subscription
	targets := router.RoutePacket(packet, "PEER-312000")

	// Should include PEER-312001 (has subscription for TG 3100 TS1)
	found := false
	for _, target := range targets {
		if target == "PEER-312001" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should route to PEER-312001 with dynamic subscription")
	}

	// Should not include source peer
	for _, target := range targets {
		if target == "PEER-312000" {
			t.Error("Should not route back to source peer")
		}
	}
}

func TestRouter_RoutePacket_SubscriptionNoMatch(t *testing.T) {
	router := NewRouter()

	router.RegisterPeer(312000, "PEER-312000")
	router.RegisterPeer(312001, "PEER-312001")

	// Set up subscription checker - PEER-312001 has subscription for TG 3100 TS1 only
	router.SetSubscriptionChecker(func(peerID uint32, tgid uint32, timeslot int) bool {
		return peerID == 312001 && tgid == 3100 && timeslot == 1
	})

	// Create a packet for TG 9999 (no subscriptions)
	packet := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      312000,
		DestinationID: 9999,
		RepeaterID:    312000,
		Timeslot:      1,
		FrameType:     protocol.FrameTypeVoiceHeader,
		StreamID:      12345,
	}

	targets := router.RoutePacket(packet, "PEER-312000")

	// Should not include any peers (no subscriptions for TG 9999)
	if len(targets) != 0 {
		t.Errorf("Should have no targets, got %d", len(targets))
	}
}

func TestRouter_RoutePacket_SubscriptionWrongTimeslot(t *testing.T) {
	router := NewRouter()

	router.RegisterPeer(312000, "PEER-312000")
	router.RegisterPeer(312001, "PEER-312001")

	// Set up subscription checker - PEER-312001 has subscription for TG 3100 TS1 only
	router.SetSubscriptionChecker(func(peerID uint32, tgid uint32, timeslot int) bool {
		return peerID == 312001 && tgid == 3100 && timeslot == 1
	})

	// Create a packet for TG 3100 but on TS2 (subscription is for TS1)
	packet := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      312000,
		DestinationID: 3100,
		RepeaterID:    312000,
		Timeslot:      2,
		FrameType:     protocol.FrameTypeVoiceHeader,
		StreamID:      12345,
	}

	targets := router.RoutePacket(packet, "PEER-312000")

	// Should not include PEER-312001 (subscription is for TS1, not TS2)
	for _, target := range targets {
		if target == "PEER-312001" {
			t.Error("Should not route to PEER-312001 on wrong timeslot")
		}
	}
}

func TestRouter_RoutePacket_CombinesBridgeAndSubscriptions(t *testing.T) {
	router := NewRouter()

	router.RegisterPeer(312000, "PEER-312000")
	router.RegisterPeer(312001, "PEER-312001")

	// Set up subscription checker - PEER-312001 has subscription for TG 3100 TS1
	router.SetSubscriptionChecker(func(peerID uint32, tgid uint32, timeslot int) bool {
		return peerID == 312001 && tgid == 3100 && timeslot == 1
	})

	// Add a static bridge rule for TG 3100
	bridge := NewBridgeRuleSet("NATIONWIDE")
	bridge.AddRule(&BridgeRule{
		System:   "STATIC-SYSTEM",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
	})
	router.AddBridge(bridge)

	// Create a packet for TG 3100 TS1
	packet := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      312000,
		DestinationID: 3100,
		RepeaterID:    312000,
		Timeslot:      1,
		FrameType:     protocol.FrameTypeVoiceHeader,
		StreamID:      12345,
	}

	targets := router.RoutePacket(packet, "PEER-312000")

	// Should include both: STATIC-SYSTEM (bridge) and PEER-312001 (subscription)
	hasStatic := false
	hasDynamic := false
	for _, target := range targets {
		if target == "STATIC-SYSTEM" {
			hasStatic = true
		}
		if target == "PEER-312001" {
			hasDynamic = true
		}
	}

	if !hasStatic {
		t.Error("Should include STATIC-SYSTEM from bridge rules")
	}
	if !hasDynamic {
		t.Error("Should include PEER-312001 from dynamic subscription")
	}
}

func TestRouter_RegisterUnregisterPeer(t *testing.T) {
	router := NewRouter()

	// Register a peer
	router.RegisterPeer(312000, "PEER-312000")

	// Verify it's registered
	router.mu.RLock()
	systemName, exists := router.peerIDToSystemName[312000]
	router.mu.RUnlock()

	if !exists {
		t.Error("Peer should be registered")
	}
	if systemName != "PEER-312000" {
		t.Errorf("System name = %s, want PEER-312000", systemName)
	}

	// Unregister the peer
	router.UnregisterPeer(312000)

	// Verify it's unregistered
	router.mu.RLock()
	_, exists = router.peerIDToSystemName[312000]
	router.mu.RUnlock()

	if exists {
		t.Error("Peer should be unregistered")
	}
}

func TestRouter_SetSubscriptionChecker(t *testing.T) {
	router := NewRouter()

	// Initially no checker
	if router.subscriptionChecker != nil {
		t.Error("Should have no subscription checker initially")
	}

	// Set a checker
	called := false
	router.SetSubscriptionChecker(func(peerID uint32, tgid uint32, timeslot int) bool {
		called = true
		return false
	})

	// Verify checker was set
	if router.subscriptionChecker == nil {
		t.Error("Subscription checker should be set")
	}

	// Call the checker
	router.subscriptionChecker(312000, 3100, 1)
	if !called {
		t.Error("Subscription checker should have been called")
	}
}

func TestRouter_GetAllDynamicBridges_Sorted(t *testing.T) {
	router := NewRouter()

	// Create dynamic bridges in random order
	tgids := []uint32{9999, 1000, 5555, 2222, 7777}
	for _, tgid := range tgids {
		router.GetOrCreateDynamicBridge(tgid)
	}

	// Get all dynamic bridges
	bridges := router.GetAllDynamicBridges()

	// Verify they are sorted by TGID
	if len(bridges) != len(tgids) {
		t.Fatalf("Expected %d bridges, got %d", len(tgids), len(bridges))
	}

	for i := 1; i < len(bridges); i++ {
		if bridges[i-1].TGID >= bridges[i].TGID {
			t.Errorf("Bridges not sorted: TGID %d at index %d should come before TGID %d at index %d",
				bridges[i-1].TGID, i-1, bridges[i].TGID, i)
		}
	}

	// Verify the expected order: 1000, 2222, 5555, 7777, 9999
	expectedOrder := []uint32{1000, 2222, 5555, 7777, 9999}
	for i, expectedTGID := range expectedOrder {
		if bridges[i].TGID != expectedTGID {
			t.Errorf("Expected TGID %d at index %d, got %d", expectedTGID, i, bridges[i].TGID)
		}
	}
}
