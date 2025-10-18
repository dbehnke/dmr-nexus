package peer

import (
	"fmt"
	"strconv"
	"strings"
)

// ACLAction defines whether to permit or deny
type ACLAction int

const (
	ACLPermit ACLAction = iota
	ACLDeny
)

// String returns the string representation of the ACL action
func (a ACLAction) String() string {
	switch a {
	case ACLPermit:
		return "PERMIT"
	case ACLDeny:
		return "DENY"
	default:
		return "UNKNOWN"
	}
}

// RuleType defines the type of ACL rule
type RuleType int

const (
	RuleTypeAll RuleType = iota
	RuleTypeSingle
	RuleTypeRange
)

// ACLRule represents a single rule in an ACL
type ACLRule struct {
	Type  RuleType
	ID    uint32 // For RuleTypeSingle
	Start uint32 // For RuleTypeRange
	End   uint32 // For RuleTypeRange
}

// String returns the string representation of the rule
func (r ACLRule) String() string {
	switch r.Type {
	case RuleTypeAll:
		return "ALL"
	case RuleTypeSingle:
		return fmt.Sprintf("%d", r.ID)
	case RuleTypeRange:
		return fmt.Sprintf("%d-%d", r.Start, r.End)
	default:
		return "UNKNOWN"
	}
}

// Matches checks if the given ID matches this rule
func (r ACLRule) Matches(id uint32) bool {
	switch r.Type {
	case RuleTypeAll:
		return true
	case RuleTypeSingle:
		return r.ID == id
	case RuleTypeRange:
		return id >= r.Start && id <= r.End
	default:
		return false
	}
}

// ACL represents an Access Control List
type ACL struct {
	Action ACLAction
	Rules  []ACLRule
}

// String returns the string representation of the ACL
func (a *ACL) String() string {
	var rules []string
	for _, rule := range a.Rules {
		rules = append(rules, rule.String())
	}
	return fmt.Sprintf("%s:%s", a.Action.String(), strings.Join(rules, ","))
}

// Check checks if the given ID is allowed by this ACL
func (a *ACL) Check(id uint32) bool {
	// Check if ID matches any rule
	matches := false
	for _, rule := range a.Rules {
		if rule.Matches(id) {
			matches = true
			break
		}
	}

	// Apply action
	if a.Action == ACLPermit {
		// For PERMIT, allow only if matches
		return matches
	} else {
		// For DENY, allow only if does NOT match
		return !matches
	}
}

// ParseACL parses an ACL string in the format "ACTION:RULE[,RULE]..."
// Examples: "PERMIT:ALL", "DENY:1", "PERMIT:3100-3199", "DENY:1,1000-2000,4500"
func ParseACL(rule string) (*ACL, error) {
	if rule == "" {
		return nil, fmt.Errorf("empty ACL rule")
	}

	// Split action and rules
	parts := strings.SplitN(rule, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ACL format: missing colon")
	}

	// Parse action
	var action ACLAction
	switch strings.ToUpper(parts[0]) {
	case "PERMIT":
		action = ACLPermit
	case "DENY":
		action = ACLDeny
	default:
		return nil, fmt.Errorf("invalid ACL action: %s", parts[0])
	}

	// Parse rules
	acl := &ACL{
		Action: action,
		Rules:  make([]ACLRule, 0),
	}

	ruleStrings := strings.Split(parts[1], ",")
	for _, ruleStr := range ruleStrings {
		ruleStr = strings.TrimSpace(ruleStr)
		if ruleStr == "" {
			continue
		}

		// Check for ALL
		if strings.ToUpper(ruleStr) == "ALL" {
			acl.Rules = append(acl.Rules, ACLRule{Type: RuleTypeAll})
			continue
		}

		// Check for range
		if strings.Contains(ruleStr, "-") {
			rangeParts := strings.Split(ruleStr, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", ruleStr)
			}

			start, err := strconv.ParseUint(strings.TrimSpace(rangeParts[0]), 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid range start: %s", rangeParts[0])
			}

			end, err := strconv.ParseUint(strings.TrimSpace(rangeParts[1]), 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid range end: %s", rangeParts[1])
			}

			if start > end {
				return nil, fmt.Errorf("invalid range: start (%d) > end (%d)", start, end)
			}

			acl.Rules = append(acl.Rules, ACLRule{
				Type:  RuleTypeRange,
				Start: uint32(start),
				End:   uint32(end),
			})
			continue
		}

		// Parse single ID
		id, err := strconv.ParseUint(ruleStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid ID: %s", ruleStr)
		}

		acl.Rules = append(acl.Rules, ACLRule{
			Type: RuleTypeSingle,
			ID:   uint32(id),
		})
	}

	if len(acl.Rules) == 0 {
		return nil, fmt.Errorf("no rules specified")
	}

	return acl, nil
}
