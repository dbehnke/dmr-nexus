package bridge

import (
	"testing"
	"time"
)

func TestStreamTracker_New(t *testing.T) {
	tracker := NewStreamTracker()
	if tracker == nil {
		t.Fatal("NewStreamTracker returned nil")
	}
}

func TestStreamTracker_TrackStream(t *testing.T) {
	tracker := NewStreamTracker()

	// Track a new stream
	streamID := uint32(12345)
	system := "SYSTEM1"

	// First time seeing this stream should return true (new stream)
	if !tracker.TrackStream(streamID, system) {
		t.Error("Expected TrackStream to return true for new stream")
	}

	// Second time seeing the same stream from the same system should return false (duplicate)
	if tracker.TrackStream(streamID, system) {
		t.Error("Expected TrackStream to return false for duplicate stream from same system")
	}
}

func TestStreamTracker_TrackStreamDifferentSystems(t *testing.T) {
	tracker := NewStreamTracker()

	streamID := uint32(12345)
	system1 := "SYSTEM1"
	system2 := "SYSTEM2"

	// First system tracks the stream
	if !tracker.TrackStream(streamID, system1) {
		t.Error("Expected TrackStream to return true for system1")
	}

	// Second system tracks the same stream - should be allowed (not a duplicate)
	if !tracker.TrackStream(streamID, system2) {
		t.Error("Expected TrackStream to return true for system2")
	}

	// First system tries again - should be duplicate
	if tracker.TrackStream(streamID, system1) {
		t.Error("Expected TrackStream to return false for duplicate from system1")
	}
}

func TestStreamTracker_IsActive(t *testing.T) {
	tracker := NewStreamTracker()

	streamID := uint32(12345)
	system := "SYSTEM1"

	// Stream should not be active initially
	if tracker.IsActive(streamID) {
		t.Error("Expected stream to not be active initially")
	}

	// Track the stream
	tracker.TrackStream(streamID, system)

	// Stream should now be active
	if !tracker.IsActive(streamID) {
		t.Error("Expected stream to be active after tracking")
	}
}

func TestStreamTracker_EndStream(t *testing.T) {
	tracker := NewStreamTracker()

	streamID := uint32(12345)
	system := "SYSTEM1"

	// Track the stream
	tracker.TrackStream(streamID, system)

	// Verify it's active
	if !tracker.IsActive(streamID) {
		t.Error("Stream should be active")
	}

	// End the stream
	tracker.EndStream(streamID)

	// Stream should no longer be active
	if tracker.IsActive(streamID) {
		t.Error("Stream should not be active after ending")
	}

	// Should be able to track the same stream ID again after ending
	if !tracker.TrackStream(streamID, system) {
		t.Error("Should be able to track stream again after ending")
	}
}

func TestStreamTracker_CleanupOldStreams(t *testing.T) {
	tracker := NewStreamTracker()

	// Track some streams
	streamID1 := uint32(111)
	streamID2 := uint32(222)
	streamID3 := uint32(333)

	tracker.TrackStream(streamID1, "SYSTEM1")
	tracker.TrackStream(streamID2, "SYSTEM1")
	tracker.TrackStream(streamID3, "SYSTEM1")

	// Verify all are active
	if !tracker.IsActive(streamID1) || !tracker.IsActive(streamID2) || !tracker.IsActive(streamID3) {
		t.Error("All streams should be active")
	}

	// Clean up streams older than 1 second
	time.Sleep(1100 * time.Millisecond)
	tracker.CleanupOldStreams(1 * time.Second)

	// All streams should be cleaned up
	if tracker.IsActive(streamID1) || tracker.IsActive(streamID2) || tracker.IsActive(streamID3) {
		t.Error("All streams should be cleaned up after timeout")
	}
}

func TestStreamTracker_GetStreamSystems(t *testing.T) {
	tracker := NewStreamTracker()

	streamID := uint32(12345)
	system1 := "SYSTEM1"
	system2 := "SYSTEM2"

	// Track stream from two systems
	tracker.TrackStream(streamID, system1)
	tracker.TrackStream(streamID, system2)

	// Get systems that have seen this stream
	systems := tracker.GetStreamSystems(streamID)
	if len(systems) != 2 {
		t.Errorf("Expected 2 systems, got %d", len(systems))
	}

	// Verify both systems are present
	hasSystem1 := false
	hasSystem2 := false
	for _, sys := range systems {
		if sys == system1 {
			hasSystem1 = true
		}
		if sys == system2 {
			hasSystem2 = true
		}
	}

	if !hasSystem1 || !hasSystem2 {
		t.Error("Expected both SYSTEM1 and SYSTEM2 in results")
	}
}

func TestStreamTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewStreamTracker()

	// Test concurrent tracking from multiple goroutines
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			streamID := uint32(id)
			tracker.TrackStream(streamID, "SYSTEM1")
			tracker.IsActive(streamID)
			tracker.EndStream(streamID)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
