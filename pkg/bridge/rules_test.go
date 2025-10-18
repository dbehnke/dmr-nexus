package bridge

import (
	"testing"
)

func TestBridgeRule_Matches(t *testing.T) {
	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
	}

	tests := []struct {
		name     string
		tgid     uint32
		timeslot int
		expected bool
	}{
		{"Exact match", 3100, 1, true},
		{"Wrong TGID", 3200, 1, false},
		{"Wrong timeslot", 3100, 2, false},
		{"Both wrong", 3200, 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Matches(tt.tgid, tt.timeslot)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBridgeRule_MatchesInactive(t *testing.T) {
	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   false,
	}

	// Even with matching TGID and timeslot, inactive rules shouldn't match
	if rule.Matches(3100, 1) {
		t.Error("Inactive rule should not match")
	}
}

func TestBridgeRule_ShouldActivate(t *testing.T) {
	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   false,
		On:       []int{3100, 3101},
	}

	tests := []struct {
		name     string
		tgid     uint32
		expected bool
	}{
		{"Activation TGID 3100", 3100, true},
		{"Activation TGID 3101", 3101, true},
		{"Non-activation TGID", 3200, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.ShouldActivate(tt.tgid)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBridgeRule_ShouldDeactivate(t *testing.T) {
	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
		Off:      []int{3101, 3102},
	}

	tests := []struct {
		name     string
		tgid     uint32
		expected bool
	}{
		{"Deactivation TGID 3101", 3101, true},
		{"Deactivation TGID 3102", 3102, true},
		{"Non-deactivation TGID", 3100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.ShouldDeactivate(tt.tgid)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBridgeRule_Activate(t *testing.T) {
	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   false,
	}

	if rule.Active {
		t.Error("Rule should start inactive")
	}

	rule.Activate()

	if !rule.Active {
		t.Error("Rule should be active after Activate()")
	}
}

func TestBridgeRule_Deactivate(t *testing.T) {
	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
	}

	if !rule.Active {
		t.Error("Rule should start active")
	}

	rule.Deactivate()

	if rule.Active {
		t.Error("Rule should be inactive after Deactivate()")
	}
}

func TestBridgeRuleSet_New(t *testing.T) {
	rules := NewBridgeRuleSet("NATIONWIDE")
	if rules == nil {
		t.Fatal("NewBridgeRuleSet returned nil")
	}

	if rules.Name != "NATIONWIDE" {
		t.Errorf("Expected name 'NATIONWIDE', got '%s'", rules.Name)
	}

	if len(rules.Rules) != 0 {
		t.Error("New rule set should have no rules")
	}
}

func TestBridgeRuleSet_AddRule(t *testing.T) {
	rules := NewBridgeRuleSet("NATIONWIDE")

	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
	}

	rules.AddRule(rule)

	if len(rules.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules.Rules))
	}

	if rules.Rules[0] != rule {
		t.Error("Added rule does not match")
	}
}

func TestBridgeRuleSet_GetRulesForSystem(t *testing.T) {
	rules := NewBridgeRuleSet("NATIONWIDE")

	rule1 := &BridgeRule{System: "SYSTEM1", TGID: 3100, Timeslot: 1, Active: true}
	rule2 := &BridgeRule{System: "SYSTEM1", TGID: 3100, Timeslot: 2, Active: true}
	rule3 := &BridgeRule{System: "SYSTEM2", TGID: 3100, Timeslot: 1, Active: true}

	rules.AddRule(rule1)
	rules.AddRule(rule2)
	rules.AddRule(rule3)

	system1Rules := rules.GetRulesForSystem("SYSTEM1")
	if len(system1Rules) != 2 {
		t.Errorf("Expected 2 rules for SYSTEM1, got %d", len(system1Rules))
	}

	system2Rules := rules.GetRulesForSystem("SYSTEM2")
	if len(system2Rules) != 1 {
		t.Errorf("Expected 1 rule for SYSTEM2, got %d", len(system2Rules))
	}

	system3Rules := rules.GetRulesForSystem("SYSTEM3")
	if len(system3Rules) != 0 {
		t.Errorf("Expected 0 rules for SYSTEM3, got %d", len(system3Rules))
	}
}

func TestBridgeRuleSet_GetMatchingRules(t *testing.T) {
	rules := NewBridgeRuleSet("NATIONWIDE")

	rule1 := &BridgeRule{System: "SYSTEM1", TGID: 3100, Timeslot: 1, Active: true}
	rule2 := &BridgeRule{System: "SYSTEM2", TGID: 3100, Timeslot: 1, Active: true}
	rule3 := &BridgeRule{System: "SYSTEM3", TGID: 3200, Timeslot: 1, Active: true}
	rule4 := &BridgeRule{System: "SYSTEM4", TGID: 3100, Timeslot: 1, Active: false}

	rules.AddRule(rule1)
	rules.AddRule(rule2)
	rules.AddRule(rule3)
	rules.AddRule(rule4)

	// Get matching rules for TGID 3100, Timeslot 1, excluding SYSTEM1
	matches := rules.GetMatchingRules(3100, 1, "SYSTEM1")

	// Should match SYSTEM2 only (SYSTEM3 is wrong TGID, SYSTEM4 is inactive)
	if len(matches) != 1 {
		t.Errorf("Expected 1 matching rule, got %d", len(matches))
	}

	if len(matches) > 0 && matches[0].System != "SYSTEM2" {
		t.Errorf("Expected matching rule for SYSTEM2, got %s", matches[0].System)
	}
}

func TestBridgeRuleSet_ProcessActivation(t *testing.T) {
	rules := NewBridgeRuleSet("NATIONWIDE")

	rule1 := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   false,
		On:       []int{3100},
	}

	rule2 := &BridgeRule{
		System:   "SYSTEM2",
		TGID:     3100,
		Timeslot: 1,
		Active:   false,
		On:       []int{3100},
	}

	rules.AddRule(rule1)
	rules.AddRule(rule2)

	// Process activation for TGID 3100
	activated := rules.ProcessActivation(3100)

	if len(activated) != 2 {
		t.Errorf("Expected 2 rules to be activated, got %d", len(activated))
	}

	if !rule1.Active || !rule2.Active {
		t.Error("Both rules should be activated")
	}
}

func TestBridgeRuleSet_ProcessDeactivation(t *testing.T) {
	rules := NewBridgeRuleSet("NATIONWIDE")

	rule1 := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
		Off:      []int{3101},
	}

	rule2 := &BridgeRule{
		System:   "SYSTEM2",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
		Off:      []int{3101},
	}

	rules.AddRule(rule1)
	rules.AddRule(rule2)

	// Process deactivation for TGID 3101
	deactivated := rules.ProcessDeactivation(3101)

	if len(deactivated) != 2 {
		t.Errorf("Expected 2 rules to be deactivated, got %d", len(deactivated))
	}

	if rule1.Active || rule2.Active {
		t.Error("Both rules should be deactivated")
	}
}
