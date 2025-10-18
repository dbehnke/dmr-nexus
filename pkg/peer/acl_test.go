package peer

import (
	"testing"
)

func TestACL_Parse_Simple(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		wantErr  bool
		action   ACLAction
		numRules int
	}{
		{
			name:     "Permit all",
			rule:     "PERMIT:ALL",
			action:   ACLPermit,
			numRules: 1,
		},
		{
			name:     "Deny all",
			rule:     "DENY:ALL",
			action:   ACLDeny,
			numRules: 1,
		},
		{
			name:     "Permit single ID",
			rule:     "PERMIT:312000",
			action:   ACLPermit,
			numRules: 1,
		},
		{
			name:     "Deny single ID",
			rule:     "DENY:1",
			action:   ACLDeny,
			numRules: 1,
		},
		{
			name:     "Permit range",
			rule:     "PERMIT:3100-3199",
			action:   ACLPermit,
			numRules: 1,
		},
		{
			name:     "Deny multiple",
			rule:     "DENY:1,1000-2000,4500",
			action:   ACLDeny,
			numRules: 3,
		},
		{
			name:    "Invalid format no colon",
			rule:    "PERMIT_ALL",
			wantErr: true,
		},
		{
			name:    "Invalid action",
			rule:    "ALLOW:ALL",
			wantErr: true,
		},
		{
			name:    "Empty rule",
			rule:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acl, err := ParseACL(tt.rule)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if acl.Action != tt.action {
				t.Errorf("Expected action %v, got %v", tt.action, acl.Action)
			}

			if len(acl.Rules) != tt.numRules {
				t.Errorf("Expected %d rules, got %d", tt.numRules, len(acl.Rules))
			}
		})
	}
}

func TestACL_Check_SingleID(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		id       uint32
		expected bool
	}{
		{
			name:     "Permit all - allow ID",
			rule:     "PERMIT:ALL",
			id:       312000,
			expected: true,
		},
		{
			name:     "Deny all - deny ID",
			rule:     "DENY:ALL",
			id:       312000,
			expected: false,
		},
		{
			name:     "Permit specific - allow match",
			rule:     "PERMIT:312000",
			id:       312000,
			expected: true,
		},
		{
			name:     "Permit specific - deny non-match",
			rule:     "PERMIT:312000",
			id:       312001,
			expected: false,
		},
		{
			name:     "Deny specific - deny match",
			rule:     "DENY:1",
			id:       1,
			expected: false,
		},
		{
			name:     "Deny specific - allow non-match",
			rule:     "DENY:1",
			id:       312000,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acl, err := ParseACL(tt.rule)
			if err != nil {
				t.Fatalf("Failed to parse ACL: %v", err)
			}

			result := acl.Check(tt.id)
			if result != tt.expected {
				t.Errorf("Check(%d) = %v, expected %v", tt.id, result, tt.expected)
			}
		})
	}
}

func TestACL_Check_Range(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		id       uint32
		expected bool
	}{
		{
			name:     "Permit range - allow in range",
			rule:     "PERMIT:3100-3199",
			id:       3150,
			expected: true,
		},
		{
			name:     "Permit range - allow start",
			rule:     "PERMIT:3100-3199",
			id:       3100,
			expected: true,
		},
		{
			name:     "Permit range - allow end",
			rule:     "PERMIT:3100-3199",
			id:       3199,
			expected: true,
		},
		{
			name:     "Permit range - deny below",
			rule:     "PERMIT:3100-3199",
			id:       3099,
			expected: false,
		},
		{
			name:     "Permit range - deny above",
			rule:     "PERMIT:3100-3199",
			id:       3200,
			expected: false,
		},
		{
			name:     "Deny range - deny in range",
			rule:     "DENY:1000-2000",
			id:       1500,
			expected: false,
		},
		{
			name:     "Deny range - allow outside",
			rule:     "DENY:1000-2000",
			id:       3000,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acl, err := ParseACL(tt.rule)
			if err != nil {
				t.Fatalf("Failed to parse ACL: %v", err)
			}

			result := acl.Check(tt.id)
			if result != tt.expected {
				t.Errorf("Check(%d) = %v, expected %v", tt.id, result, tt.expected)
			}
		})
	}
}

func TestACL_Check_Multiple(t *testing.T) {
	// Test multiple rules in one ACL
	acl, err := ParseACL("DENY:1,1000-2000,4500-6000")
	if err != nil {
		t.Fatalf("Failed to parse ACL: %v", err)
	}

	tests := []struct {
		id       uint32
		expected bool
	}{
		{1, false},        // Denied by first rule
		{1500, false},     // Denied by range 1000-2000
		{5000, false},     // Denied by range 4500-6000
		{3000, true},      // Allowed (not in any deny rule)
		{312000, true},    // Allowed (not in any deny rule)
		{999, true},       // Allowed (just before range)
		{2001, true},      // Allowed (just after range)
		{4499, true},      // Allowed (just before range)
		{6001, true},      // Allowed (just after range)
	}

	for _, tt := range tests {
		result := acl.Check(tt.id)
		if result != tt.expected {
			t.Errorf("Check(%d) = %v, expected %v", tt.id, result, tt.expected)
		}
	}
}

func TestACL_Parse_InvalidRanges(t *testing.T) {
	tests := []struct {
		name string
		rule string
	}{
		{
			name: "Invalid range format",
			rule: "PERMIT:3100-3199-3200",
		},
		{
			name: "Non-numeric ID",
			rule: "PERMIT:ABC",
		},
		{
			name: "Non-numeric range",
			rule: "PERMIT:ABC-DEF",
		},
		{
			name: "Inverted range",
			rule: "PERMIT:3199-3100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseACL(tt.rule)
			if err == nil {
				t.Error("Expected error for invalid ACL, got nil")
			}
		})
	}
}

func TestACLAction_String(t *testing.T) {
	tests := []struct {
		action   ACLAction
		expected string
	}{
		{ACLPermit, "PERMIT"},
		{ACLDeny, "DENY"},
	}

	for _, tt := range tests {
		if tt.action.String() != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, tt.action.String())
		}
	}
}

func TestACLRule_String(t *testing.T) {
	tests := []struct {
		name     string
		rule     ACLRule
		expected string
	}{
		{
			name:     "All rule",
			rule:     ACLRule{Type: RuleTypeAll},
			expected: "ALL",
		},
		{
			name:     "Single ID",
			rule:     ACLRule{Type: RuleTypeSingle, ID: 312000},
			expected: "312000",
		},
		{
			name:     "Range",
			rule:     ACLRule{Type: RuleTypeRange, Start: 3100, End: 3199},
			expected: "3100-3199",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.rule.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.rule.String())
			}
		})
	}
}

func TestACL_String(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		expected string
	}{
		{
			name:     "Permit all",
			rule:     "PERMIT:ALL",
			expected: "PERMIT:ALL",
		},
		{
			name:     "Deny single",
			rule:     "DENY:1",
			expected: "DENY:1",
		},
		{
			name:     "Permit range",
			rule:     "PERMIT:3100-3199",
			expected: "PERMIT:3100-3199",
		},
		{
			name:     "Deny multiple",
			rule:     "DENY:1,1000-2000,4500",
			expected: "DENY:1,1000-2000,4500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acl, err := ParseACL(tt.rule)
			if err != nil {
				t.Fatalf("Failed to parse ACL: %v", err)
			}

			result := acl.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
