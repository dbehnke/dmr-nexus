package bridge

import (
	"os"
	"testing"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/database"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

func TestTransmissionLogger_LogPacket(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_tx_logger.db"
	defer os.Remove(dbPath)

	db, err := database.NewDB(database.Config{Path: dbPath}, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := database.NewTransmissionRepository(db.GetDB())
	txLogger := NewTransmissionLogger(repo, log)

	// Simulate a transmission with multiple packets
	streamID := uint32(12345)
	radioID := uint32(1234567)
	talkgroupID := uint32(91)
	timeslot := 1
	repeaterID := uint32(3001)

	// Log start packet
	txLogger.LogPacket(streamID, radioID, talkgroupID, repeaterID, timeslot, false)

	// Check active stream count
	if count := txLogger.GetActiveStreamCount(); count != 1 {
		t.Errorf("Expected 1 active stream, got %d", count)
	}

	// Log more packets
	time.Sleep(10 * time.Millisecond)
	txLogger.LogPacket(streamID, radioID, talkgroupID, repeaterID, timeslot, false)
	time.Sleep(10 * time.Millisecond)
	txLogger.LogPacket(streamID, radioID, talkgroupID, repeaterID, timeslot, false)

	// Log terminator packet
	time.Sleep(10 * time.Millisecond)
	txLogger.LogPacket(streamID, radioID, talkgroupID, repeaterID, timeslot, true)

	// Check that stream was saved and removed from active
	if count := txLogger.GetActiveStreamCount(); count != 0 {
		t.Errorf("Expected 0 active streams after terminator, got %d", count)
	}

	// Verify transmission was saved to database
	transmissions, err := repo.GetRecent(1)
	if err != nil {
		t.Fatalf("Failed to get transmissions: %v", err)
	}

	if len(transmissions) != 1 {
		t.Fatalf("Expected 1 transmission, got %d", len(transmissions))
	}

	tx := transmissions[0]
	if tx.RadioID != radioID {
		t.Errorf("Expected radio ID %d, got %d", radioID, tx.RadioID)
	}
	if tx.TalkgroupID != talkgroupID {
		t.Errorf("Expected talkgroup ID %d, got %d", talkgroupID, tx.TalkgroupID)
	}
	if tx.Timeslot != timeslot {
		t.Errorf("Expected timeslot %d, got %d", timeslot, tx.Timeslot)
	}
	if tx.StreamID != streamID {
		t.Errorf("Expected stream ID %d, got %d", streamID, tx.StreamID)
	}
	if tx.PacketCount != 4 {
		t.Errorf("Expected packet count 4, got %d", tx.PacketCount)
	}
	if tx.Duration <= 0 {
		t.Errorf("Expected positive duration, got %f", tx.Duration)
	}
}

func TestTransmissionLogger_MultipleStreams(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_tx_logger_multi.db"
	defer os.Remove(dbPath)

	db, err := database.NewDB(database.Config{Path: dbPath}, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := database.NewTransmissionRepository(db.GetDB())
	txLogger := NewTransmissionLogger(repo, log)

	// Start two streams simultaneously
	stream1 := uint32(11111)
	stream2 := uint32(22222)

	txLogger.LogPacket(stream1, 1000001, 91, 3001, 1, false)
	txLogger.LogPacket(stream2, 1000002, 92, 3001, 2, false)

	if count := txLogger.GetActiveStreamCount(); count != 2 {
		t.Errorf("Expected 2 active streams, got %d", count)
	}

	// End first stream
	txLogger.LogPacket(stream1, 1000001, 91, 3001, 1, true)

	if count := txLogger.GetActiveStreamCount(); count != 1 {
		t.Errorf("Expected 1 active stream after ending first, got %d", count)
	}

	// End second stream
	txLogger.LogPacket(stream2, 1000002, 92, 3001, 2, true)

	if count := txLogger.GetActiveStreamCount(); count != 0 {
		t.Errorf("Expected 0 active streams after ending both, got %d", count)
	}

	// Verify both transmissions were saved
	transmissions, err := repo.GetRecent(10)
	if err != nil {
		t.Fatalf("Failed to get transmissions: %v", err)
	}

	if len(transmissions) != 2 {
		t.Fatalf("Expected 2 transmissions, got %d", len(transmissions))
	}
}

func TestTransmissionLogger_CleanupStaleStreams(t *testing.T) {
	log := logger.New(logger.Config{Level: "error"})
	dbPath := "/tmp/test_tx_logger_cleanup.db"
	defer os.Remove(dbPath)

	db, err := database.NewDB(database.Config{Path: dbPath}, log)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	repo := database.NewTransmissionRepository(db.GetDB())
	txLogger := NewTransmissionLogger(repo, log)

	// Start a stream
	streamID := uint32(99999)
	txLogger.LogPacket(streamID, 1000001, 91, 3001, 1, false)

	if count := txLogger.GetActiveStreamCount(); count != 1 {
		t.Errorf("Expected 1 active stream, got %d", count)
	}

	// Wait a bit then cleanup with short max age
	time.Sleep(50 * time.Millisecond)
	txLogger.CleanupStaleStreams(10 * time.Millisecond)

	// Stream should be cleaned up
	if count := txLogger.GetActiveStreamCount(); count != 0 {
		t.Errorf("Expected 0 active streams after cleanup, got %d", count)
	}

	// Verify transmission was saved
	transmissions, err := repo.GetRecent(1)
	if err != nil {
		t.Fatalf("Failed to get transmissions: %v", err)
	}

	if len(transmissions) != 1 {
		t.Fatalf("Expected 1 transmission after cleanup, got %d", len(transmissions))
	}
}
