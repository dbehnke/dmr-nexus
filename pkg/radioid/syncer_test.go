package radioid

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/dbehnke/dmr-nexus/pkg/database"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

func TestSyncer_parseCSV(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_radioid_syncer.db"
	defer os.Remove(dbPath)

	cfg := database.Config{Path: dbPath}
	db, err := database.NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := database.NewDMRUserRepository(db.GetDB())
	syncer := NewSyncer(repo, log)

	// Create test CSV data
	csvData := `RADIO_ID,CALLSIGN,FIRST_NAME,LAST_NAME,CITY,STATE,COUNTRY
3138617,K7ABC,John,Doe,Seattle,WA,USA
3200449,W7XYZ,Jane,Smith,Portland,OR,USA
1234567,VE3TEST,Bob,Johnson,Toronto,ON,Canada`

	reader := strings.NewReader(csvData)
	users, err := syncer.parseCSV(reader)
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(users) != 3 {
		t.Errorf("Expected 3 users, got %d", len(users))
	}

	// Check first user
	if users[0].RadioID != 3138617 {
		t.Errorf("Expected radio ID 3138617, got %d", users[0].RadioID)
	}
	if users[0].Callsign != "K7ABC" {
		t.Errorf("Expected callsign K7ABC, got %s", users[0].Callsign)
	}
	if users[0].FirstName != "John" {
		t.Errorf("Expected first name John, got %s", users[0].FirstName)
	}
	if users[0].City != "Seattle" {
		t.Errorf("Expected city Seattle, got %s", users[0].City)
	}
}

func TestSyncer_parseCSV_InvalidData(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_radioid_syncer_invalid.db"
	defer os.Remove(dbPath)

	cfg := database.Config{Path: dbPath}
	db, err := database.NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := database.NewDMRUserRepository(db.GetDB())
	syncer := NewSyncer(repo, log)

	// CSV with invalid radio ID and short lines
	csvData := `RADIO_ID,CALLSIGN,FIRST_NAME,LAST_NAME,CITY,STATE,COUNTRY
invalid,K7ABC,John,Doe,Seattle,WA,USA
3138617,K7DEF,Jane,Smith,Portland,OR,USA
short,line
1234567,VE3TEST,Bob,Johnson,Toronto,ON,Canada`

	reader := strings.NewReader(csvData)
	users, err := syncer.parseCSV(reader)
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should only have 2 valid users (invalid radio ID and short line are skipped)
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestNewSyncer(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_radioid_new_syncer.db"
	defer os.Remove(dbPath)

	cfg := database.Config{Path: dbPath}
	db, err := database.NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := database.NewDMRUserRepository(db.GetDB())
	syncer := NewSyncer(repo, log)

	if syncer == nil {
		t.Error("Expected non-nil syncer")
	}
	if syncer.repo == nil {
		t.Error("Expected non-nil repo in syncer")
	}
	if syncer.logger == nil {
		t.Error("Expected non-nil logger in syncer")
	}
	if syncer.client == nil {
		t.Error("Expected non-nil HTTP client in syncer")
	}
}

func TestSyncer_Start_Cancellation(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_radioid_start.db"
	defer os.Remove(dbPath)

	cfg := database.Config{Path: dbPath}
	db, err := database.NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := database.NewDMRUserRepository(db.GetDB())
	syncer := NewSyncer(repo, log)

	// Create a context that we'll cancel immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Start should return quickly when context is cancelled
	syncer.Start(ctx)
	// If we get here without hanging, the test passes
}
