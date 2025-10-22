package database

import (
	"os"
	"testing"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

func TestNewDB(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_dmr_nexus.db"
	defer func() { _ = os.Remove(dbPath) }()

	cfg := Config{Path: dbPath}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if db.db == nil {
		t.Error("Expected non-nil database connection")
	}
}

func TestNewDB_DefaultPath(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	defer func() { _ = os.Remove("dmr-nexus.db") }()

	cfg := Config{}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database with default path: %v", err)
	}
	defer func() { _ = db.Close() }()

	if db.db == nil {
		t.Error("Expected non-nil database connection")
	}
}

func TestTransmission_BeforeCreate(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_transmission_create.db"
	defer func() { _ = os.Remove(dbPath) }()

	cfg := Config{Path: dbPath}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create transmission without timestamps
	tx := &Transmission{
		RadioID:     1234567,
		TalkgroupID: 91,
		Timeslot:    1,
		Duration:    5.5,
		StreamID:    999,
		RepeaterID:  3001,
		PacketCount: 10,
	}

	repo := NewTransmissionRepository(db.GetDB())
	err = repo.Create(tx)
	if err != nil {
		t.Fatalf("Failed to create transmission: %v", err)
	}

	if tx.ID == 0 {
		t.Error("Expected non-zero ID after creation")
	}
	if tx.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set by hook")
	}
	if tx.StartTime.IsZero() {
		t.Error("Expected StartTime to be set by hook")
	}
	if tx.EndTime.IsZero() {
		t.Error("Expected EndTime to be set by hook")
	}
}

func TestTransmissionRepository_Create(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_repo_create.db"
	defer func() { _ = os.Remove(dbPath) }()

	cfg := Config{Path: dbPath}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewTransmissionRepository(db.GetDB())

	now := time.Now()
	tx := &Transmission{
		RadioID:     1234567,
		TalkgroupID: 91,
		Timeslot:    1,
		Duration:    5.5,
		StreamID:    12345,
		StartTime:   now,
		EndTime:     now.Add(5 * time.Second),
		RepeaterID:  3001,
		PacketCount: 10,
	}

	err = repo.Create(tx)
	if err != nil {
		t.Fatalf("Failed to create transmission: %v", err)
	}

	if tx.ID == 0 {
		t.Error("Expected non-zero ID after creation")
	}
}

func TestTransmissionRepository_GetRecent(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_get_recent.db"
	defer os.Remove(dbPath)

	cfg := Config{Path: dbPath}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewTransmissionRepository(db.GetDB())

	// Create multiple transmissions
	now := time.Now()
	for i := 0; i < 5; i++ {
		tx := &Transmission{
			RadioID:     uint32(1234560 + i),
			TalkgroupID: 91,
			Timeslot:    1,
			Duration:    float64(i),
			StreamID:    uint32(1000 + i),
			StartTime:   now.Add(time.Duration(i) * time.Minute),
			EndTime:     now.Add(time.Duration(i)*time.Minute + 5*time.Second),
			RepeaterID:  3001,
			PacketCount: 10,
		}
		if err := repo.Create(tx); err != nil {
			t.Fatalf("Failed to create transmission %d: %v", i, err)
		}
	}

	// Get recent 3
	transmissions, err := repo.GetRecent(3)
	if err != nil {
		t.Fatalf("Failed to get recent transmissions: %v", err)
	}

	if len(transmissions) != 3 {
		t.Errorf("Expected 3 transmissions, got %d", len(transmissions))
	}

	// Verify order (most recent first)
	if len(transmissions) >= 2 {
		if transmissions[0].StartTime.Before(transmissions[1].StartTime) {
			t.Error("Expected transmissions to be ordered by start_time DESC")
		}
	}
}

func TestTransmissionRepository_GetRecentPaginated(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_paginated.db"
	defer os.Remove(dbPath)

	cfg := Config{Path: dbPath}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewTransmissionRepository(db.GetDB())

	// Create 10 transmissions
	now := time.Now()
	for i := 0; i < 10; i++ {
		tx := &Transmission{
			RadioID:     uint32(1234560 + i),
			TalkgroupID: 91,
			Timeslot:    1,
			Duration:    float64(i),
			StreamID:    uint32(1000 + i),
			StartTime:   now.Add(time.Duration(i) * time.Minute),
			EndTime:     now.Add(time.Duration(i)*time.Minute + 5*time.Second),
			RepeaterID:  3001,
			PacketCount: 10,
		}
		if err := repo.Create(tx); err != nil {
			t.Fatalf("Failed to create transmission %d: %v", i, err)
		}
	}

	// Get first page
	transmissions, total, err := repo.GetRecentPaginated(1, 5)
	if err != nil {
		t.Fatalf("Failed to get paginated transmissions: %v", err)
	}

	if len(transmissions) != 5 {
		t.Errorf("Expected 5 transmissions on page 1, got %d", len(transmissions))
	}

	if total != 10 {
		t.Errorf("Expected total of 10, got %d", total)
	}

	// Get second page
	transmissions2, total2, err := repo.GetRecentPaginated(2, 5)
	if err != nil {
		t.Fatalf("Failed to get paginated transmissions page 2: %v", err)
	}

	if len(transmissions2) != 5 {
		t.Errorf("Expected 5 transmissions on page 2, got %d", len(transmissions2))
	}

	if total2 != 10 {
		t.Errorf("Expected total of 10 on page 2, got %d", total2)
	}
}

func TestTransmissionRepository_GetByRadioID(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_by_radio.db"
	defer os.Remove(dbPath)

	cfg := Config{Path: dbPath}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewTransmissionRepository(db.GetDB())

	now := time.Now()
	targetRadioID := uint32(1234567)

	// Create transmissions for target radio
	for i := 0; i < 3; i++ {
		tx := &Transmission{
			RadioID:     targetRadioID,
			TalkgroupID: 91,
			Timeslot:    1,
			Duration:    float64(i),
			StreamID:    uint32(1000 + i),
			StartTime:   now.Add(time.Duration(i) * time.Minute),
			EndTime:     now.Add(time.Duration(i)*time.Minute + 5*time.Second),
			RepeaterID:  3001,
			PacketCount: 10,
		}
		if err := repo.Create(tx); err != nil {
			t.Fatalf("Failed to create transmission %d: %v", i, err)
		}
	}

	// Create transmission for different radio
	otherTx := &Transmission{
		RadioID:     9999999,
		TalkgroupID: 91,
		Timeslot:    1,
		Duration:    1.0,
		StreamID:    9999,
		StartTime:   now,
		EndTime:     now.Add(5 * time.Second),
		RepeaterID:  3001,
		PacketCount: 10,
	}
	if err := repo.Create(otherTx); err != nil {
		t.Fatalf("Failed to create other transmission: %v", err)
	}

	// Query by target radio ID
	transmissions, err := repo.GetByRadioID(targetRadioID, 10)
	if err != nil {
		t.Fatalf("Failed to get transmissions by radio ID: %v", err)
	}

	if len(transmissions) != 3 {
		t.Errorf("Expected 3 transmissions for radio %d, got %d", targetRadioID, len(transmissions))
	}

	for _, tx := range transmissions {
		if tx.RadioID != targetRadioID {
			t.Errorf("Expected radio ID %d, got %d", targetRadioID, tx.RadioID)
		}
	}
}

func TestTransmissionRepository_GetByTalkgroup(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_by_tg.db"
	defer os.Remove(dbPath)

	cfg := Config{Path: dbPath}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewTransmissionRepository(db.GetDB())

	now := time.Now()
	targetTG := uint32(91)

	// Create transmissions for target talkgroup
	for i := 0; i < 2; i++ {
		tx := &Transmission{
			RadioID:     uint32(1234560 + i),
			TalkgroupID: targetTG,
			Timeslot:    1,
			Duration:    float64(i),
			StreamID:    uint32(1000 + i),
			StartTime:   now.Add(time.Duration(i) * time.Minute),
			EndTime:     now.Add(time.Duration(i)*time.Minute + 5*time.Second),
			RepeaterID:  3001,
			PacketCount: 10,
		}
		if err := repo.Create(tx); err != nil {
			t.Fatalf("Failed to create transmission %d: %v", i, err)
		}
	}

	// Create transmission for different talkgroup
	otherTx := &Transmission{
		RadioID:     1234567,
		TalkgroupID: 3100,
		Timeslot:    1,
		Duration:    1.0,
		StreamID:    9999,
		StartTime:   now,
		EndTime:     now.Add(5 * time.Second),
		RepeaterID:  3001,
		PacketCount: 10,
	}
	if err := repo.Create(otherTx); err != nil {
		t.Fatalf("Failed to create other transmission: %v", err)
	}

	// Query by target talkgroup
	transmissions, err := repo.GetByTalkgroup(targetTG, 10)
	if err != nil {
		t.Fatalf("Failed to get transmissions by talkgroup: %v", err)
	}

	if len(transmissions) != 2 {
		t.Errorf("Expected 2 transmissions for TG %d, got %d", targetTG, len(transmissions))
	}

	for _, tx := range transmissions {
		if tx.TalkgroupID != targetTG {
			t.Errorf("Expected talkgroup ID %d, got %d", targetTG, tx.TalkgroupID)
		}
	}
}

func TestTransmissionRepository_DeleteOlderThan(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_delete_old.db"
	defer os.Remove(dbPath)

	cfg := Config{Path: dbPath}
	db, err := NewDB(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := NewTransmissionRepository(db.GetDB())

	now := time.Now()

	// Create old transmission
	oldTx := &Transmission{
		RadioID:     1234567,
		TalkgroupID: 91,
		Timeslot:    1,
		Duration:    1.0,
		StreamID:    1000,
		StartTime:   now.Add(-48 * time.Hour),
		EndTime:     now.Add(-48*time.Hour + 5*time.Second),
		RepeaterID:  3001,
		PacketCount: 10,
	}
	if err := repo.Create(oldTx); err != nil {
		t.Fatalf("Failed to create old transmission: %v", err)
	}

	// Create recent transmission
	recentTx := &Transmission{
		RadioID:     1234568,
		TalkgroupID: 91,
		Timeslot:    1,
		Duration:    1.0,
		StreamID:    1001,
		StartTime:   now.Add(-1 * time.Hour),
		EndTime:     now.Add(-1*time.Hour + 5*time.Second),
		RepeaterID:  3001,
		PacketCount: 10,
	}
	if err := repo.Create(recentTx); err != nil {
		t.Fatalf("Failed to create recent transmission: %v", err)
	}

	// Delete transmissions older than 24 hours
	deleted, err := repo.DeleteOlderThan(now.Add(-24 * time.Hour))
	if err != nil {
		t.Fatalf("Failed to delete old transmissions: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 deletion, got %d", deleted)
	}

	// Verify recent transmission still exists
	transmissions, err := repo.GetRecent(10)
	if err != nil {
		t.Fatalf("Failed to get remaining transmissions: %v", err)
	}

	if len(transmissions) != 1 {
		t.Errorf("Expected 1 remaining transmission, got %d", len(transmissions))
	}
}
