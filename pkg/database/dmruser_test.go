package database

import (
	"os"
	"testing"

	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

func TestDMRUser_FullName(t *testing.T) {
	tests := []struct {
		name      string
		user      DMRUser
		expected  string
	}{
		{
			name: "Both names present",
			user: DMRUser{
				FirstName: "John",
				LastName:  "Doe",
			},
			expected: "John Doe",
		},
		{
			name: "Only first name",
			user: DMRUser{
				FirstName: "John",
			},
			expected: "John",
		},
		{
			name: "Only last name",
			user: DMRUser{
				LastName: "Doe",
			},
			expected: "Doe",
		},
		{
			name:     "No names",
			user:     DMRUser{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.FullName()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDMRUser_Location(t *testing.T) {
	tests := []struct {
		name     string
		user     DMRUser
		expected string
	}{
		{
			name: "All fields present",
			user: DMRUser{
				City:    "Seattle",
				State:   "WA",
				Country: "USA",
			},
			expected: "Seattle, WA, USA",
		},
		{
			name: "City and state only",
			user: DMRUser{
				City:  "Seattle",
				State: "WA",
			},
			expected: "Seattle, WA",
		},
		{
			name: "Country only",
			user: DMRUser{
				Country: "USA",
			},
			expected: "USA",
		},
		{
			name:     "No location",
			user:     DMRUser{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.Location()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDMRUserRepository_Upsert(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_dmr_user_upsert.db"
	defer os.Remove(dbPath)

	cfg := Config{Path: dbPath}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewDMRUserRepository(db.GetDB())

	// Create a user
	user := &DMRUser{
		RadioID:   3138617,
		Callsign:  "K7ABC",
		FirstName: "John",
		LastName:  "Doe",
		City:      "Seattle",
		State:     "WA",
		Country:   "USA",
	}

	err = repo.Upsert(user)
	if err != nil {
		t.Fatalf("Failed to upsert user: %v", err)
	}

	// Retrieve the user
	retrieved, err := repo.GetByRadioID(3138617)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if retrieved.Callsign != "K7ABC" {
		t.Errorf("Expected callsign K7ABC, got %s", retrieved.Callsign)
	}

	// Update the user
	user.FirstName = "Jane"
	err = repo.Upsert(user)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Retrieve again
	retrieved, err = repo.GetByRadioID(3138617)
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}

	if retrieved.FirstName != "Jane" {
		t.Errorf("Expected first name Jane, got %s", retrieved.FirstName)
	}
}

func TestDMRUserRepository_GetByCallsign(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_dmr_user_callsign.db"
	defer os.Remove(dbPath)

	cfg := Config{Path: dbPath}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewDMRUserRepository(db.GetDB())

	// Create a user
	user := &DMRUser{
		RadioID:  3138617,
		Callsign: "K7ABC",
	}

	err = repo.Upsert(user)
	if err != nil {
		t.Fatalf("Failed to upsert user: %v", err)
	}

	// Retrieve by callsign
	retrieved, err := repo.GetByCallsign("K7ABC")
	if err != nil {
		t.Fatalf("Failed to get user by callsign: %v", err)
	}

	if retrieved.RadioID != 3138617 {
		t.Errorf("Expected radio ID 3138617, got %d", retrieved.RadioID)
	}
}

func TestDMRUserRepository_Count(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_dmr_user_count.db"
	defer os.Remove(dbPath)

	cfg := Config{Path: dbPath}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewDMRUserRepository(db.GetDB())

	// Initially should be 0
	count, err := repo.Count()
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 users, got %d", count)
	}

	// Add some users
	users := []DMRUser{
		{RadioID: 1, Callsign: "K7ABC"},
		{RadioID: 2, Callsign: "K7DEF"},
		{RadioID: 3, Callsign: "K7GHI"},
	}

	for _, u := range users {
		user := u
		if err := repo.Upsert(&user); err != nil {
			t.Fatalf("Failed to upsert user: %v", err)
		}
	}

	// Count should be 3
	count, err = repo.Count()
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 users, got %d", count)
	}
}

func TestDMRUserRepository_UpsertBatch(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_dmr_user_batch.db"
	defer os.Remove(dbPath)

	cfg := Config{Path: dbPath}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewDMRUserRepository(db.GetDB())

	// Create batch of users
	users := make([]DMRUser, 100)
	for i := 0; i < 100; i++ {
		users[i] = DMRUser{
			RadioID:  uint32(i + 1),
			Callsign: "TEST",
		}
	}

	// Upsert in batches
	err = repo.UpsertBatch(users, 10)
	if err != nil {
		t.Fatalf("Failed to upsert batch: %v", err)
	}

	// Verify count
	count, err := repo.Count()
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}
	if count != 100 {
		t.Errorf("Expected 100 users, got %d", count)
	}
}
