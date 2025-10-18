package bridge

import (
	"sync"
	"time"
)

// StreamInfo tracks information about an active stream
type StreamInfo struct {
	StreamID  uint32
	Systems   map[string]bool // Systems that have seen this stream
	StartTime time.Time
}

// StreamTracker manages active DMR streams and prevents packet loops
type StreamTracker struct {
	streams map[uint32]*StreamInfo
	mu      sync.RWMutex
}

// NewStreamTracker creates a new stream tracker
func NewStreamTracker() *StreamTracker {
	return &StreamTracker{
		streams: make(map[uint32]*StreamInfo),
	}
}

// TrackStream tracks a stream from a specific system.
// Returns true if this is a new stream from this system (should forward),
// false if we've already seen this stream from this system (duplicate, don't forward).
func (st *StreamTracker) TrackStream(streamID uint32, system string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Get or create stream info
	info, exists := st.streams[streamID]
	if !exists {
		// New stream - create tracking entry
		info = &StreamInfo{
			StreamID:  streamID,
			Systems:   make(map[string]bool),
			StartTime: time.Now(),
		}
		st.streams[streamID] = info
	}

	// Check if this system has already seen this stream
	if info.Systems[system] {
		// Duplicate - we've already processed this stream from this system
		return false
	}

	// Mark that this system has now seen the stream
	info.Systems[system] = true
	return true
}

// IsActive checks if a stream is currently active
func (st *StreamTracker) IsActive(streamID uint32) bool {
	st.mu.RLock()
	defer st.mu.RUnlock()

	_, exists := st.streams[streamID]
	return exists
}

// EndStream marks a stream as ended and removes it from tracking
func (st *StreamTracker) EndStream(streamID uint32) {
	st.mu.Lock()
	defer st.mu.Unlock()

	delete(st.streams, streamID)
}

// GetStreamSystems returns the list of systems that have seen this stream
func (st *StreamTracker) GetStreamSystems(streamID uint32) []string {
	st.mu.RLock()
	defer st.mu.RUnlock()

	info, exists := st.streams[streamID]
	if !exists {
		return []string{}
	}

	systems := make([]string, 0, len(info.Systems))
	for system := range info.Systems {
		systems = append(systems, system)
	}

	return systems
}

// CleanupOldStreams removes streams that have been active longer than the given duration
func (st *StreamTracker) CleanupOldStreams(maxAge time.Duration) {
	st.mu.Lock()
	defer st.mu.Unlock()

	now := time.Now()
	for streamID, info := range st.streams {
		if now.Sub(info.StartTime) > maxAge {
			delete(st.streams, streamID)
		}
	}
}
