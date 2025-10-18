package bridge

import (
	"fmt"
	"sync"
	"time"
)

// TimerManager manages timeout timers for bridge rules
type TimerManager struct {
	timers map[string]*time.Timer
	mu     sync.RWMutex
}

// NewTimerManager creates a new timer manager
func NewTimerManager() *TimerManager {
	return &TimerManager{
		timers: make(map[string]*time.Timer),
	}
}

// ruleKey generates a unique key for a rule
func ruleKey(rule *BridgeRule) string {
	return fmt.Sprintf("%s:%d:%d", rule.System, rule.TGID, rule.Timeslot)
}

// SetTimeout sets a timeout for a rule (in minutes as specified in config)
// When the timeout expires, the rule will be deactivated
func (tm *TimerManager) SetTimeout(rule *BridgeRule) {
	if rule.Timeout <= 0 {
		return // No timeout configured
	}

	duration := time.Duration(rule.Timeout) * time.Minute
	tm.SetTimeoutWithCallback(rule, duration, func(r *BridgeRule) {
		r.Deactivate()
	})
}

// SetTimeoutWithCallback sets a timeout with a custom callback
func (tm *TimerManager) SetTimeoutWithCallback(rule *BridgeRule, duration time.Duration, callback func(*BridgeRule)) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	key := ruleKey(rule)

	// Clear existing timer if present
	if existingTimer, exists := tm.timers[key]; exists {
		existingTimer.Stop()
	}

	// Create new timer
	timer := time.AfterFunc(duration, func() {
		callback(rule)
		tm.mu.Lock()
		delete(tm.timers, key)
		tm.mu.Unlock()
	})

	tm.timers[key] = timer
}

// ClearTimeout clears the timeout for a rule
func (tm *TimerManager) ClearTimeout(rule *BridgeRule) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	key := ruleKey(rule)
	if timer, exists := tm.timers[key]; exists {
		timer.Stop()
		delete(tm.timers, key)
	}
}

// RefreshTimeout refreshes the timeout for a rule
func (tm *TimerManager) RefreshTimeout(rule *BridgeRule) {
	// Simply set the timeout again, which will clear the old one
	tm.SetTimeout(rule)
}

// HasTimer checks if a rule has an active timer
func (tm *TimerManager) HasTimer(rule *BridgeRule) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	key := ruleKey(rule)
	_, exists := tm.timers[key]
	return exists
}

// StopAll stops all active timers
func (tm *TimerManager) StopAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, timer := range tm.timers {
		timer.Stop()
	}

	tm.timers = make(map[string]*time.Timer)
}
