package peer

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Constants for subscription limits
const (
	MaxStaticTalkgroups = 50   // Maximum static talkgroups per timeslot
	MaxAutoStaticTTL    = 3600 // Maximum TTL in seconds (1 hour)
	DefaultAutoTTL      = 600  // Default TTL in seconds (10 minutes)
)

// SubscriptionOptions represents parsed OPTIONS from a peer
type SubscriptionOptions struct {
	TS1      []uint32 // Static talkgroups for timeslot 1
	TS2      []uint32 // Static talkgroups for timeslot 2
	Auto     int      // Auto-static TTL in seconds
	DropAll  bool     // Clear all static talkgroups
	UnlinkTS uint8    // Unlink specific timeslot (1 or 2)
}

// SubscriptionState tracks dynamic talkgroup subscriptions for a peer
type SubscriptionState struct {
	TS1         map[uint32]time.Time // Talkgroup -> expiry time for TS1
	TS2         map[uint32]time.Time // Talkgroup -> expiry time for TS2
	AutoTTL     time.Duration        // Auto-static TTL
	LastUpdated time.Time            // Last update timestamp
	mu          sync.RWMutex
}

// NewSubscriptionState creates a new subscription state
func NewSubscriptionState() *SubscriptionState {
	return &SubscriptionState{
		TS1: make(map[uint32]time.Time),
		TS2: make(map[uint32]time.Time),
	}
}

// Update updates the subscription state with new options
func (s *SubscriptionState) Update(opts *SubscriptionOptions) error {
	if err := validateOptions(opts); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.LastUpdated = now

	// Handle DROP=ALL
	if opts.DropAll {
		s.TS1 = make(map[uint32]time.Time)
		s.TS2 = make(map[uint32]time.Time)
		s.AutoTTL = 0
		return nil
	}

	// Handle UNLINK
	switch opts.UnlinkTS {
	case 1:
		s.TS1 = make(map[uint32]time.Time)
	case 2:
		s.TS2 = make(map[uint32]time.Time)
	}

	// Update auto TTL
	if opts.Auto > 0 {
		s.AutoTTL = time.Duration(opts.Auto) * time.Second
	}

	// Set expiry time for talkgroups
	var expiryTime time.Time
	if s.AutoTTL > 0 {
		expiryTime = now.Add(s.AutoTTL)
	}
	// Zero time means no expiry (static)

	// Update TS1 talkgroups
	if len(opts.TS1) > 0 {
		for _, tgid := range opts.TS1 {
			s.TS1[tgid] = expiryTime
		}
	}

	// Update TS2 talkgroups
	if len(opts.TS2) > 0 {
		for _, tgid := range opts.TS2 {
			s.TS2[tgid] = expiryTime
		}
	}

	return nil
}

// HasTalkgroup checks if a talkgroup is in the subscription for the given timeslot
func (s *SubscriptionState) HasTalkgroup(tgid uint32, timeslot uint8) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tgMap map[uint32]time.Time
	switch timeslot {
	case 1:
		tgMap = s.TS1
	case 2:
		tgMap = s.TS2
	default:
		return false
	}

	expiryTime, exists := tgMap[tgid]
	if !exists {
		return false
	}

	// Static (zero) or unlimited dynamic (sentinel) are always active
	if expiryTime.IsZero() || expiryTime.Unix() == 1 {
		return true
	}

	// TTL-based dynamic: check if expired
	if time.Now().After(expiryTime) {
		return false
	}

	return true
}

// GetTalkgroups returns all active talkgroups for the given timeslot
func (s *SubscriptionState) GetTalkgroups(timeslot uint8) []uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tgMap map[uint32]time.Time
	switch timeslot {
	case 1:
		tgMap = s.TS1
	case 2:
		tgMap = s.TS2
	default:
		return []uint32{}
	}

	now := time.Now()
	result := make([]uint32, 0, len(tgMap))

	for tgid, expiryTime := range tgMap {
		// Keep static (zero) and unlimited dynamic (sentinel)
		if expiryTime.IsZero() || expiryTime.Unix() == 1 {
			result = append(result, tgid)
			continue
		}

		// Keep TTL-based if not expired
		if now.Before(expiryTime) {
			result = append(result, tgid)
		}
	}

	return result
}

// IsExpired checks if the subscription has expired based on TTL
func (s *SubscriptionState) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If no TTL set, never expires
	if s.AutoTTL == 0 {
		return false
	}

	// Check if last update is older than TTL
	return time.Since(s.LastUpdated) > s.AutoTTL
}

// Clear clears all subscription state
func (s *SubscriptionState) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TS1 = make(map[uint32]time.Time)
	s.TS2 = make(map[uint32]time.Time)
	s.AutoTTL = 0
	s.LastUpdated = time.Time{}
}

// CleanupExpired removes expired talkgroups from the subscription
func (s *SubscriptionState) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Clean TS1
	for tgid, expiryTime := range s.TS1 {
		if !expiryTime.IsZero() && now.After(expiryTime) {
			delete(s.TS1, tgid)
		}
	}

	// Clean TS2
	for tgid, expiryTime := range s.TS2 {
		if !expiryTime.IsZero() && now.After(expiryTime) {
			delete(s.TS2, tgid)
		}
	}
}

// AddDynamic adds a dynamic talkgroup subscription
// Only allows one dynamic TG per timeslot - clears other dynamic TGs in the same slot
// Uses the peer's AutoTTL setting (from OPTIONS) or unlimited if AutoTTL is 0
// Returns true if this is a NEW subscription (first key-up), false if already subscribed
func (s *SubscriptionState) AddDynamic(tgid uint32, timeslot uint8) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	var tgMap map[uint32]time.Time
	switch timeslot {
	case 1:
		tgMap = s.TS1
	case 2:
		tgMap = s.TS2
	default:
		return false
	}

	// Check if already subscribed to this TG (return false = not new)
	if expiry, exists := tgMap[tgid]; exists {
		// Already subscribed - just extend/refresh the TTL
		// Only update if it's a dynamic subscription (has non-zero expiry)
		// Static subscriptions (zero time) are never refreshed
		if !expiry.IsZero() && s.AutoTTL > 0 {
			tgMap[tgid] = time.Now().Add(s.AutoTTL)
		}
		s.LastUpdated = time.Now()
		return false // Not a new subscription
	}

	// Clear all OTHER dynamic subscriptions in this timeslot (keep static ones)
	// Static subscriptions have time.Time{} (zero value) as the marker
	// Dynamic subscriptions have a sentinel value of time.Unix(1, 0) if unlimited
	for existingTGID, existingExpiry := range tgMap {
		// Keep static subscriptions (zero time) - those come from RPTC OPTIONS
		// Remove all other subscriptions (both TTL-based and unlimited dynamic)
		if existingTGID != tgid && !existingExpiry.IsZero() {
			delete(tgMap, existingTGID)
		}
	}

	// Add new subscription
	// Use sentinel value time.Unix(1, 0) for unlimited dynamic subscriptions
	// This distinguishes them from static subscriptions (zero time)
	var expiryTime time.Time
	if s.AutoTTL > 0 {
		expiryTime = time.Now().Add(s.AutoTTL)
	} else {
		// Unlimited dynamic subscription - use sentinel value
		expiryTime = time.Unix(1, 0)
	}

	tgMap[tgid] = expiryTime
	s.LastUpdated = time.Now()
	return true // This is a new subscription
}

// TouchDynamic extends the TTL for a dynamic subscription
func (s *SubscriptionState) TouchDynamic(tgid uint32, timeslot uint8, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var tgMap map[uint32]time.Time
	switch timeslot {
	case 1:
		tgMap = s.TS1
	case 2:
		tgMap = s.TS2
	default:
		return
	}

	if _, exists := tgMap[tgid]; exists {
		tgMap[tgid] = time.Now().Add(ttl)
		s.LastUpdated = time.Now()
	}
}

// ClearAllDynamic removes all dynamic subscriptions while keeping static ones (from RPTC OPTIONS)
// This is used when a peer transmits on the special disconnect TG (4000)
// Static subscriptions have time.Time{} (zero value)
// Dynamic subscriptions have either future time (TTL) or time.Unix(1, 0) (unlimited sentinel)
func (s *SubscriptionState) ClearAllDynamic() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0

	// Clear TS1 dynamic subscriptions (non-zero expiry time)
	// Keep static subscriptions (zero time value)
	for tgid, expiryTime := range s.TS1 {
		if !expiryTime.IsZero() {
			delete(s.TS1, tgid)
			count++
		}
	}

	// Clear TS2 dynamic subscriptions (non-zero expiry time)
	// Keep static subscriptions (zero time value)
	for tgid, expiryTime := range s.TS2 {
		if !expiryTime.IsZero() {
			delete(s.TS2, tgid)
			count++
		}
	}

	s.LastUpdated = time.Now()
	return count
}

// IsSubscribed checks if the peer is subscribed to a specific talkgroup/timeslot
// Returns true if subscribed (either static or dynamic and not expired)
func (s *SubscriptionState) IsSubscribed(tgid uint32, timeslot uint8) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tgMap map[uint32]time.Time
	switch timeslot {
	case 1:
		tgMap = s.TS1
	case 2:
		tgMap = s.TS2
	default:
		return false
	}

	expiryTime, exists := tgMap[tgid]
	if !exists {
		return false
	}

	// Static subscription (zero time) always valid
	if expiryTime.IsZero() {
		return true
	}

	// Unlimited dynamic subscription sentinel (Unix epoch + 1 nanosecond)
	if expiryTime.Unix() == 1 {
		return true
	}

	// TTL-based dynamic subscription - check if expired
	return time.Now().Before(expiryTime)
}

// IsSubscribedToTalkgroup checks if the peer is subscribed to a talkgroup on ANY timeslot
// This is used for timeslot-agnostic dynamic bridges
func (s *SubscriptionState) IsSubscribedToTalkgroup(tgid uint32) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()

	// Check TS1
	if expiryTime, exists := s.TS1[tgid]; exists {
		// Static (zero) or unlimited dynamic (sentinel) or not expired TTL
		if expiryTime.IsZero() || expiryTime.Unix() == 1 || now.Before(expiryTime) {
			return true
		}
	}

	// Check TS2
	if expiryTime, exists := s.TS2[tgid]; exists {
		// Static (zero) or unlimited dynamic (sentinel) or not expired TTL
		if expiryTime.IsZero() || expiryTime.Unix() == 1 || now.Before(expiryTime) {
			return true
		}
	}

	return false
}

// ParseOptions parses an OPTIONS string into SubscriptionOptions
// Format: TS1=3100,3101;TS2=91;AUTO=600;DROP=ALL;UNLINK=TS1
func ParseOptions(input string) (*SubscriptionOptions, error) {
	opts := &SubscriptionOptions{
		TS1: []uint32{},
		TS2: []uint32{},
	}

	// Trim null bytes from input (common in binary protocol packets)
	input = strings.Trim(input, "\x00")
	
	if input == "" {
		return opts, nil
	}

	// Split by semicolon
	pairs := strings.Split(input, ";")

	for _, pair := range pairs {
		// Split by equals
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 {
			continue // Skip invalid pairs
		}

		key := strings.ToUpper(strings.TrimSpace(parts[0]))
		// Trim whitespace and null bytes from value
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\x00")

		switch key {
		case "TS1":
			tgs, err := parseTalkgroupList(value)
			if err != nil {
				return nil, fmt.Errorf("invalid TS1 value: %w", err)
			}
			opts.TS1 = tgs

		case "TS2":
			tgs, err := parseTalkgroupList(value)
			if err != nil {
				return nil, fmt.Errorf("invalid TS2 value: %w", err)
			}
			opts.TS2 = tgs

		case "AUTO":
			ttl, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid AUTO value: %w", err)
			}
			opts.Auto = ttl

		case "DROP":
			if strings.ToUpper(value) == "ALL" {
				opts.DropAll = true
			}

		case "UNLINK":
			ts := strings.ToUpper(value)
			switch ts {
			case "TS1":
				opts.UnlinkTS = 1
			case "TS2":
				opts.UnlinkTS = 2
			}
		}
	}

	// Validate the options before returning
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	return opts, nil
}

// parseTalkgroupList parses a comma-separated list of talkgroup IDs
func parseTalkgroupList(input string) ([]uint32, error) {
	// Trim null bytes from input (common in binary protocol packets)
	input = strings.Trim(input, "\x00")
	
	if input == "" {
		return []uint32{}, nil
	}

	parts := strings.Split(input, ",")
	result := make([]uint32, 0, len(parts))

	for _, part := range parts {
		// Trim whitespace and null bytes from the part
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "\x00")
		if part == "" {
			continue
		}

		tgid, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid talkgroup ID '%s': %w", part, err)
		}

		result = append(result, uint32(tgid))
	}

	return result, nil
}

// ExtractOptionsFromDescription extracts OPTIONS string from a description field
// Expected format: "description text | OPTIONS: TS1=3100,3101"
func ExtractOptionsFromDescription(description string) string {
	// Find "OPTIONS:" (case insensitive)
	idx := strings.Index(strings.ToUpper(description), "OPTIONS:")
	if idx == -1 {
		return ""
	}

	// Extract everything after "OPTIONS:"
	optionsStart := idx + len("OPTIONS:")
	if optionsStart >= len(description) {
		return ""
	}

	return strings.TrimSpace(description[optionsStart:])
}

// validateOptions validates subscription options
func validateOptions(opts *SubscriptionOptions) error {
	if opts == nil {
		return fmt.Errorf("options cannot be nil")
	}

	// Check TS1 limit
	if len(opts.TS1) > MaxStaticTalkgroups {
		return fmt.Errorf("too many TS1 talkgroups: %d (max %d)", len(opts.TS1), MaxStaticTalkgroups)
	}

	// Check TS2 limit
	if len(opts.TS2) > MaxStaticTalkgroups {
		return fmt.Errorf("too many TS2 talkgroups: %d (max %d)", len(opts.TS2), MaxStaticTalkgroups)
	}

	// Check AUTO value
	if opts.Auto < 0 {
		return fmt.Errorf("AUTO value cannot be negative: %d", opts.Auto)
	}
	if opts.Auto > MaxAutoStaticTTL {
		return fmt.Errorf("AUTO value too large: %d (max %d)", opts.Auto, MaxAutoStaticTTL)
	}

	return nil
}
