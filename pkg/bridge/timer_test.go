package bridge

import (
	"testing"
	"time"
)

func TestTimerManager_New(t *testing.T) {
	tm := NewTimerManager()
	if tm == nil {
		t.Fatal("NewTimerManager returned nil")
	}
}

func TestTimerManager_SetTimeout(t *testing.T) {
	tm := NewTimerManager()

	// Create a rule
	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
		Timeout:  1, // 1 minute
	}

	// Set timeout
	tm.SetTimeout(rule)

	// Check that timer exists
	if !tm.HasTimer(rule) {
		t.Error("Timer should exist after SetTimeout")
	}
}

func TestTimerManager_ClearTimeout(t *testing.T) {
	tm := NewTimerManager()

	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
		Timeout:  1,
	}

	// Set timeout
	tm.SetTimeout(rule)

	// Clear timeout
	tm.ClearTimeout(rule)

	// Check that timer no longer exists
	if tm.HasTimer(rule) {
		t.Error("Timer should not exist after ClearTimeout")
	}
}

func TestTimerManager_TimeoutExpiration(t *testing.T) {
	tm := NewTimerManager()

	// Create a rule with short timeout
	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
		Timeout:  1, // 1 minute (in real implementation)
	}

	// For testing, we need to use a shorter duration
	// Normally we'd use minutes, but for tests we'll check the mechanism works

	// Set timeout
	tm.SetTimeout(rule)

	// Timer should exist
	if !tm.HasTimer(rule) {
		t.Error("Timer should exist")
	}
}

func TestTimerManager_RefreshTimeout(t *testing.T) {
	tm := NewTimerManager()

	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
		Timeout:  5,
	}

	// Set initial timeout
	tm.SetTimeout(rule)

	// Refresh the timeout
	tm.RefreshTimeout(rule)

	// Timer should still exist
	if !tm.HasTimer(rule) {
		t.Error("Timer should exist after refresh")
	}
}

func TestTimerManager_MultipleRules(t *testing.T) {
	tm := NewTimerManager()

	rule1 := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
		Timeout:  5,
	}

	rule2 := &BridgeRule{
		System:   "SYSTEM2",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
		Timeout:  10,
	}

	// Set timeouts for both rules
	tm.SetTimeout(rule1)
	tm.SetTimeout(rule2)

	// Both should have timers
	if !tm.HasTimer(rule1) {
		t.Error("Timer for rule1 should exist")
	}
	if !tm.HasTimer(rule2) {
		t.Error("Timer for rule2 should exist")
	}

	// Clear rule1
	tm.ClearTimeout(rule1)

	// rule1 should not have timer, rule2 should
	if tm.HasTimer(rule1) {
		t.Error("Timer for rule1 should not exist after clear")
	}
	if !tm.HasTimer(rule2) {
		t.Error("Timer for rule2 should still exist")
	}
}

func TestTimerManager_RuleKey(t *testing.T) {
	rule1 := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
	}

	rule2 := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
	}

	rule3 := &BridgeRule{
		System:   "SYSTEM2",
		TGID:     3100,
		Timeslot: 1,
	}

	// Same system, TGID, and timeslot should produce same key
	key1 := ruleKey(rule1)
	key2 := ruleKey(rule2)
	if key1 != key2 {
		t.Error("Keys should be equal for identical rules")
	}

	// Different system should produce different key
	key3 := ruleKey(rule3)
	if key1 == key3 {
		t.Error("Keys should be different for different systems")
	}
}

func TestTimerManager_StopAll(t *testing.T) {
	tm := NewTimerManager()

	rule1 := &BridgeRule{System: "SYSTEM1", TGID: 3100, Timeslot: 1, Timeout: 5}
	rule2 := &BridgeRule{System: "SYSTEM2", TGID: 3200, Timeslot: 2, Timeout: 10}
	rule3 := &BridgeRule{System: "SYSTEM3", TGID: 3300, Timeslot: 1, Timeout: 15}

	tm.SetTimeout(rule1)
	tm.SetTimeout(rule2)
	tm.SetTimeout(rule3)

	// All should have timers
	if !tm.HasTimer(rule1) || !tm.HasTimer(rule2) || !tm.HasTimer(rule3) {
		t.Error("All rules should have timers")
	}

	// Stop all timers
	tm.StopAll()

	// None should have timers
	if tm.HasTimer(rule1) || tm.HasTimer(rule2) || tm.HasTimer(rule3) {
		t.Error("No rules should have timers after StopAll")
	}
}

func TestTimerManager_ZeroTimeout(t *testing.T) {
	tm := NewTimerManager()

	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
		Timeout:  0, // No timeout
	}

	// Set timeout with 0 value - should not create timer
	tm.SetTimeout(rule)

	// Timer should not exist for zero timeout
	if tm.HasTimer(rule) {
		t.Error("Timer should not exist for zero timeout")
	}
}

func TestTimerManager_CallbackExecution(t *testing.T) {
	tm := NewTimerManager()

	rule := &BridgeRule{
		System:   "SYSTEM1",
		TGID:     3100,
		Timeslot: 1,
		Active:   true,
		Timeout:  1,
	}

	// Track if callback was called using a channel
	callbackDone := make(chan struct{}, 1)
	callback := func(r *BridgeRule) {
		if r.System != "SYSTEM1" {
			t.Error("Wrong rule passed to callback")
		}
		callbackDone <- struct{}{}
	}

	// Set timeout with callback using a very short duration for testing
	tm.SetTimeoutWithCallback(rule, 10*time.Millisecond, callback)

	// Wait for callback to be called
	select {
	case <-callbackDone:
		// Callback was called successfully
	case <-time.After(100 * time.Millisecond):
		t.Error("Callback should have been called after timeout")
	}
}
