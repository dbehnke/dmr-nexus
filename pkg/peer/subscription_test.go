package peer

import (
	"reflect"
	"testing"
	"time"
)

func TestParseOptions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *SubscriptionOptions
		wantErr bool
	}{
		{
			name:  "Empty string",
			input: "",
			want: &SubscriptionOptions{
				TS1:  []uint32{},
				TS2:  []uint32{},
				Auto: 0,
			},
			wantErr: false,
		},
		{
			name:  "Single TS1 talkgroup",
			input: "TS1=3100",
			want: &SubscriptionOptions{
				TS1:  []uint32{3100},
				TS2:  []uint32{},
				Auto: 0,
			},
			wantErr: false,
		},
		{
			name:  "Multiple TS1 talkgroups",
			input: "TS1=3100,3101,3102",
			want: &SubscriptionOptions{
				TS1:  []uint32{3100, 3101, 3102},
				TS2:  []uint32{},
				Auto: 0,
			},
			wantErr: false,
		},
		{
			name:  "Both timeslots",
			input: "TS1=3100,3101;TS2=91,92",
			want: &SubscriptionOptions{
				TS1:  []uint32{3100, 3101},
				TS2:  []uint32{91, 92},
				Auto: 0,
			},
			wantErr: false,
		},
		{
			name:  "With auto-static TTL",
			input: "TS1=3100;AUTO=600",
			want: &SubscriptionOptions{
				TS1:  []uint32{3100},
				TS2:  []uint32{},
				Auto: 600,
			},
			wantErr: false,
		},
		{
			name:  "With DROP=ALL",
			input: "DROP=ALL",
			want: &SubscriptionOptions{
				TS1:      []uint32{},
				TS2:      []uint32{},
				Auto:     0,
				DropAll:  true,
				UnlinkTS: 0,
			},
			wantErr: false,
		},
		{
			name:  "With UNLINK=TS1",
			input: "UNLINK=TS1",
			want: &SubscriptionOptions{
				TS1:      []uint32{},
				TS2:      []uint32{},
				Auto:     0,
				UnlinkTS: 1,
			},
			wantErr: false,
		},
		{
			name:  "With UNLINK=TS2",
			input: "UNLINK=TS2",
			want: &SubscriptionOptions{
				TS1:      []uint32{},
				TS2:      []uint32{},
				Auto:     0,
				UnlinkTS: 2,
			},
			wantErr: false,
		},
		{
			name:  "Complex example",
			input: "TS1=3100,3101;TS2=91;AUTO=600",
			want: &SubscriptionOptions{
				TS1:  []uint32{3100, 3101},
				TS2:  []uint32{91},
				Auto: 600,
			},
			wantErr: false,
		},
		{
			name:  "Case insensitive keys",
			input: "ts1=3100;ts2=91",
			want: &SubscriptionOptions{
				TS1:  []uint32{3100},
				TS2:  []uint32{91},
				Auto: 0,
			},
			wantErr: false,
		},
		{
			name:  "Whitespace handling",
			input: " TS1 = 3100 , 3101 ; TS2 = 91 ",
			want: &SubscriptionOptions{
				TS1:  []uint32{3100, 3101},
				TS2:  []uint32{91},
				Auto: 0,
			},
			wantErr: false,
		},
		{
			name:    "Invalid talkgroup ID",
			input:   "TS1=invalid",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Invalid AUTO value",
			input:   "AUTO=invalid",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Negative AUTO value",
			input:   "AUTO=-100",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "AUTO too large",
			input:   "AUTO=5000",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOptions(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractOptionsFromDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		want        string
	}{
		{
			name:        "OPTIONS in description",
			description: "My Pi-Star | OPTIONS: TS1=3100,3101;TS2=91",
			want:        "TS1=3100,3101;TS2=91",
		},
		{
			name:        "OPTIONS at start",
			description: "OPTIONS: TS1=3100",
			want:        "TS1=3100",
		},
		{
			name:        "OPTIONS at end",
			description: "Some description | OPTIONS: TS1=3100",
			want:        "TS1=3100",
		},
		{
			name:        "No OPTIONS",
			description: "Just a description",
			want:        "",
		},
		{
			name:        "Empty description",
			description: "",
			want:        "",
		},
		{
			name:        "Case insensitive OPTIONS",
			description: "My repeater | options: TS1=3100",
			want:        "TS1=3100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractOptionsFromDescription(tt.description)
			if got != tt.want {
				t.Errorf("ExtractOptionsFromDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubscriptionState_New(t *testing.T) {
	state := NewSubscriptionState()
	if state == nil {
		t.Fatal("NewSubscriptionState() returned nil")
	}
	if state.TS1 == nil || state.TS2 == nil {
		t.Error("NewSubscriptionState() did not initialize talkgroup maps")
	}
	if state.AutoTTL != 0 {
		t.Errorf("NewSubscriptionState() AutoTTL = %v, want 0", state.AutoTTL)
	}
}

func TestSubscriptionState_Update(t *testing.T) {
	state := NewSubscriptionState()

	// Initial update
	opts := &SubscriptionOptions{
		TS1:  []uint32{3100, 3101},
		TS2:  []uint32{91},
		Auto: 600,
	}

	err := state.Update(opts)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify TS1 talkgroups
	if len(state.TS1) != 2 {
		t.Errorf("TS1 count = %d, want 2", len(state.TS1))
	}
	if _, ok := state.TS1[3100]; !ok {
		t.Error("TS1 missing talkgroup 3100")
	}
	if _, ok := state.TS1[3101]; !ok {
		t.Error("TS1 missing talkgroup 3101")
	}

	// Verify TS2 talkgroups
	if len(state.TS2) != 1 {
		t.Errorf("TS2 count = %d, want 1", len(state.TS2))
	}
	if _, ok := state.TS2[91]; !ok {
		t.Error("TS2 missing talkgroup 91")
	}

	// Verify TTL
	if state.AutoTTL != 600*time.Second {
		t.Errorf("AutoTTL = %v, want %v", state.AutoTTL, 600*time.Second)
	}

	// Verify last updated is recent
	if time.Since(state.LastUpdated) > time.Second {
		t.Error("LastUpdated is not recent")
	}
}

func TestSubscriptionState_UpdateWithDropAll(t *testing.T) {
	state := NewSubscriptionState()

	// Add some talkgroups
	opts := &SubscriptionOptions{
		TS1: []uint32{3100, 3101},
		TS2: []uint32{91},
	}
	if err := state.Update(opts); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Drop all
	dropOpts := &SubscriptionOptions{
		DropAll: true,
	}
	err := state.Update(dropOpts)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify all cleared
	if len(state.TS1) != 0 {
		t.Errorf("TS1 count = %d, want 0 after DROP=ALL", len(state.TS1))
	}
	if len(state.TS2) != 0 {
		t.Errorf("TS2 count = %d, want 0 after DROP=ALL", len(state.TS2))
	}
}

func TestSubscriptionState_UpdateWithUnlink(t *testing.T) {
	state := NewSubscriptionState()

	// Add talkgroups to both timeslots
	opts := &SubscriptionOptions{
		TS1: []uint32{3100, 3101},
		TS2: []uint32{91, 92},
	}
	if err := state.Update(opts); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Unlink TS1
	unlinkOpts := &SubscriptionOptions{
		UnlinkTS: 1,
	}
	err := state.Update(unlinkOpts)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify TS1 cleared, TS2 intact
	if len(state.TS1) != 0 {
		t.Errorf("TS1 count = %d, want 0 after UNLINK=TS1", len(state.TS1))
	}
	if len(state.TS2) != 2 {
		t.Errorf("TS2 count = %d, want 2 after UNLINK=TS1", len(state.TS2))
	}

	// Unlink TS2
	unlinkOpts2 := &SubscriptionOptions{
		UnlinkTS: 2,
	}
	if err := state.Update(unlinkOpts2); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify TS2 cleared
	if len(state.TS2) != 0 {
		t.Errorf("TS2 count = %d, want 0 after UNLINK=TS2", len(state.TS2))
	}
}

func TestSubscriptionState_HasTalkgroup(t *testing.T) {
	state := NewSubscriptionState()

	opts := &SubscriptionOptions{
		TS1: []uint32{3100, 3101},
		TS2: []uint32{91},
	}
	if err := state.Update(opts); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	tests := []struct {
		name     string
		tgid     uint32
		timeslot uint8
		want     bool
	}{
		{"TS1 match", 3100, 1, true},
		{"TS1 match 2", 3101, 1, true},
		{"TS1 no match", 3102, 1, false},
		{"TS2 match", 91, 2, true},
		{"TS2 no match", 92, 2, false},
		{"Wrong timeslot", 3100, 2, false},
		{"Invalid timeslot", 3100, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := state.HasTalkgroup(tt.tgid, tt.timeslot)
			if got != tt.want {
				t.Errorf("HasTalkgroup(%d, %d) = %v, want %v", tt.tgid, tt.timeslot, got, tt.want)
			}
		})
	}
}

func TestSubscriptionState_GetTalkgroups(t *testing.T) {
	state := NewSubscriptionState()

	opts := &SubscriptionOptions{
		TS1: []uint32{3100, 3101, 3102},
		TS2: []uint32{91, 92},
	}
	if err := state.Update(opts); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	ts1 := state.GetTalkgroups(1)
	if len(ts1) != 3 {
		t.Errorf("GetTalkgroups(1) count = %d, want 3", len(ts1))
	}

	ts2 := state.GetTalkgroups(2)
	if len(ts2) != 2 {
		t.Errorf("GetTalkgroups(2) count = %d, want 2", len(ts2))
	}

	// Invalid timeslot should return empty
	ts3 := state.GetTalkgroups(3)
	if len(ts3) != 0 {
		t.Errorf("GetTalkgroups(3) count = %d, want 0", len(ts3))
	}
}

func TestSubscriptionState_IsExpired(t *testing.T) {
	state := NewSubscriptionState()

	// No TTL set - should not expire
	if state.IsExpired() {
		t.Error("IsExpired() = true for zero TTL, want false")
	}

	// Set TTL and recent update
	opts := &SubscriptionOptions{
		TS1:  []uint32{3100},
		Auto: 2, // 2 seconds
	}
	if err := state.Update(opts); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Should not be expired yet
	if state.IsExpired() {
		t.Error("IsExpired() = true for recent update, want false")
	}

	// Simulate old update
	state.LastUpdated = time.Now().Add(-5 * time.Second)

	// Should be expired now
	if !state.IsExpired() {
		t.Error("IsExpired() = false for old update, want true")
	}
}

func TestSubscriptionState_Clear(t *testing.T) {
	state := NewSubscriptionState()

	// Add some data
	opts := &SubscriptionOptions{
		TS1:  []uint32{3100, 3101},
		TS2:  []uint32{91},
		Auto: 600,
	}
	if err := state.Update(opts); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Clear
	state.Clear()

	// Verify cleared
	if len(state.TS1) != 0 {
		t.Errorf("TS1 count = %d after Clear(), want 0", len(state.TS1))
	}
	if len(state.TS2) != 0 {
		t.Errorf("TS2 count = %d after Clear(), want 0", len(state.TS2))
	}
	if state.AutoTTL != 0 {
		t.Errorf("AutoTTL = %v after Clear(), want 0", state.AutoTTL)
	}
}

func TestSubscriptionState_MaxTalkgroupsLimit(t *testing.T) {
	state := NewSubscriptionState()

	// Try to add more than max talkgroups
	tgs := make([]uint32, MaxStaticTalkgroups+1)
	for i := range tgs {
		tgs[i] = uint32(3100 + i)
	}

	opts := &SubscriptionOptions{
		TS1: tgs,
	}

	err := state.Update(opts)
	if err == nil {
		t.Error("Update() with too many talkgroups should return error")
	}
}

func TestSubscriptionState_ConcurrentAccess(t *testing.T) {
	state := NewSubscriptionState()

	// Initialize with some data
	opts := &SubscriptionOptions{
		TS1: []uint32{3100},
		TS2: []uint32{91},
	}
	if err := state.Update(opts); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Concurrent reads and writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				state.HasTalkgroup(3100, 1)
				state.GetTalkgroups(1)
				state.IsExpired()
			}
			done <- true
		}()
	}

	// Also do some concurrent writes
	go func() {
		for i := 0; i < 50; i++ {
			opts := &SubscriptionOptions{
				TS1: []uint32{3100 + uint32(i)},
			}
			_ = state.Update(opts) // Ignore error in concurrent test
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 11; i++ {
		<-done
	}
}

func TestValidateOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    *SubscriptionOptions
		wantErr bool
	}{
		{
			name: "Valid options",
			opts: &SubscriptionOptions{
				TS1:  []uint32{3100, 3101},
				TS2:  []uint32{91},
				Auto: 600,
			},
			wantErr: false,
		},
		{
			name: "Empty options",
			opts: &SubscriptionOptions{
				TS1:  []uint32{},
				TS2:  []uint32{},
				Auto: 0,
			},
			wantErr: false,
		},
		{
			name: "Too many TS1 talkgroups",
			opts: &SubscriptionOptions{
				TS1: make([]uint32, MaxStaticTalkgroups+1),
			},
			wantErr: true,
		},
		{
			name: "Too many TS2 talkgroups",
			opts: &SubscriptionOptions{
				TS2: make([]uint32, MaxStaticTalkgroups+1),
			},
			wantErr: true,
		},
		{
			name: "Invalid AUTO value",
			opts: &SubscriptionOptions{
				TS1:  []uint32{3100},
				Auto: MaxAutoStaticTTL + 1,
			},
			wantErr: true,
		},
		{
			name: "Negative AUTO value",
			opts: &SubscriptionOptions{
				TS1:  []uint32{3100},
				Auto: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptions(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
