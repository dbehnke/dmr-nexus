package bridge

import (
	"sync"
)

// BridgeRule represents a single routing rule for a conference bridge
type BridgeRule struct {
	System   string // System name to route to/from
	TGID     int    // Talkgroup ID
	Timeslot int    // Timeslot (1 or 2)
	Active   bool   // Whether this rule is currently active
	On       []int  // TGIDs that activate this rule
	Off      []int  // TGIDs that deactivate this rule
	Timeout  int    // Minutes before auto-disable (if >0)

	mu sync.RWMutex
}

// Matches checks if this rule matches the given TGID and timeslot
func (r *BridgeRule) Matches(tgid uint32, timeslot int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.Active {
		return false
	}

	return int(tgid) == r.TGID && timeslot == r.Timeslot
}

// ShouldActivate checks if this rule should be activated by the given TGID
func (r *BridgeRule) ShouldActivate(tgid uint32) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.On) == 0 {
		return false
	}

	tgidInt := int(tgid)
	for _, activationTGID := range r.On {
		if activationTGID == tgidInt {
			return true
		}
	}

	return false
}

// ShouldDeactivate checks if this rule should be deactivated by the given TGID
func (r *BridgeRule) ShouldDeactivate(tgid uint32) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.Off) == 0 {
		return false
	}

	tgidInt := int(tgid)
	for _, deactivationTGID := range r.Off {
		if deactivationTGID == tgidInt {
			return true
		}
	}

	return false
}

// Activate activates this rule
func (r *BridgeRule) Activate() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Active = true
}

// Deactivate deactivates this rule
func (r *BridgeRule) Deactivate() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Active = false
}

// BridgeRuleSet represents a named set of bridge rules
type BridgeRuleSet struct {
	Name  string
	Rules []*BridgeRule
	mu    sync.RWMutex
}

// NewBridgeRuleSet creates a new bridge rule set
func NewBridgeRuleSet(name string) *BridgeRuleSet {
	return &BridgeRuleSet{
		Name:  name,
		Rules: make([]*BridgeRule, 0),
	}
}

// AddRule adds a rule to this rule set
func (brs *BridgeRuleSet) AddRule(rule *BridgeRule) {
	brs.mu.Lock()
	defer brs.mu.Unlock()
	brs.Rules = append(brs.Rules, rule)
}

// GetRulesForSystem returns all rules for a specific system
func (brs *BridgeRuleSet) GetRulesForSystem(system string) []*BridgeRule {
	brs.mu.RLock()
	defer brs.mu.RUnlock()

	result := make([]*BridgeRule, 0)
	for _, rule := range brs.Rules {
		if rule.System == system {
			result = append(result, rule)
		}
	}

	return result
}

// GetMatchingRules returns all active rules that match the given TGID and timeslot,
// excluding the source system to prevent loops
func (brs *BridgeRuleSet) GetMatchingRules(tgid uint32, timeslot int, excludeSystem string) []*BridgeRule {
	brs.mu.RLock()
	defer brs.mu.RUnlock()

	result := make([]*BridgeRule, 0)
	for _, rule := range brs.Rules {
		if rule.System == excludeSystem {
			continue
		}
		if rule.Matches(tgid, timeslot) {
			result = append(result, rule)
		}
	}

	return result
}

// ProcessActivation processes activation for the given TGID
// Returns the list of rules that were activated
func (brs *BridgeRuleSet) ProcessActivation(tgid uint32) []*BridgeRule {
	brs.mu.RLock()
	defer brs.mu.RUnlock()

	activated := make([]*BridgeRule, 0)
	for _, rule := range brs.Rules {
		if rule.ShouldActivate(tgid) {
			rule.Activate()
			activated = append(activated, rule)
		}
	}

	return activated
}

// ProcessDeactivation processes deactivation for the given TGID
// Returns the list of rules that were deactivated
func (brs *BridgeRuleSet) ProcessDeactivation(tgid uint32) []*BridgeRule {
	brs.mu.RLock()
	defer brs.mu.RUnlock()

	deactivated := make([]*BridgeRule, 0)
	for _, rule := range brs.Rules {
		if rule.ShouldDeactivate(tgid) {
			rule.Deactivate()
			deactivated = append(deactivated, rule)
		}
	}

	return deactivated
}

// BridgeRuleSnapshot is a read-only snapshot of a BridgeRule
type BridgeRuleSnapshot struct {
	System   string `json:"system"`
	TGID     int    `json:"tgid"`
	Timeslot int    `json:"timeslot"`
	Active   bool   `json:"active"`
}

// BridgeRuleSetSnapshot is a read-only snapshot of a BridgeRuleSet
type BridgeRuleSetSnapshot struct {
	Name  string               `json:"name"`
	Rules []BridgeRuleSnapshot `json:"rules"`
}

// Snapshot returns a snapshot of the rule set and all rules
func (brs *BridgeRuleSet) Snapshot() BridgeRuleSetSnapshot {
	brs.mu.RLock()
	defer brs.mu.RUnlock()

	out := BridgeRuleSetSnapshot{Name: brs.Name, Rules: make([]BridgeRuleSnapshot, 0, len(brs.Rules))}
	for _, rule := range brs.Rules {
		rule.mu.RLock()
		out.Rules = append(out.Rules, BridgeRuleSnapshot{
			System:   rule.System,
			TGID:     rule.TGID,
			Timeslot: rule.Timeslot,
			Active:   rule.Active,
		})
		rule.mu.RUnlock()
	}
	return out
}
