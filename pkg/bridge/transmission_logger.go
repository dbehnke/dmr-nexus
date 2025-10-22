package bridge

import (
	"sync"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/database"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

// TransmissionLogger logs DMR transmissions to the database
type TransmissionLogger struct {
	repo          *database.TransmissionRepository
	logger        *logger.Logger
	activeStreams map[uint32]*activeStream
	mu            sync.RWMutex
}

// activeStream tracks an ongoing transmission
type activeStream struct {
	streamID    uint32
	radioID     uint32
	talkgroupID uint32
	timeslot    int
	repeaterID  uint32
	startTime   time.Time
	lastSeen    time.Time
	packetCount int
}

// NewTransmissionLogger creates a new transmission logger
func NewTransmissionLogger(repo *database.TransmissionRepository, log *logger.Logger) *TransmissionLogger {
	return &TransmissionLogger{
		repo:          repo,
		logger:        log,
		activeStreams: make(map[uint32]*activeStream),
	}
}

// LogPacket logs a DMR packet, tracking streams and creating transmission records
func (tl *TransmissionLogger) LogPacket(streamID, radioID, talkgroupID, repeaterID uint32, timeslot int, isTerminator bool) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	now := time.Now()

	// Get or create active stream
	stream, exists := tl.activeStreams[streamID]
	if !exists {
		// New stream - create tracking entry
		stream = &activeStream{
			streamID:    streamID,
			radioID:     radioID,
			talkgroupID: talkgroupID,
			timeslot:    timeslot,
			repeaterID:  repeaterID,
			startTime:   now,
			lastSeen:    now,
			packetCount: 1,
		}
		tl.activeStreams[streamID] = stream
		tl.logger.Debug("Started tracking stream",
			logger.Any("stream_id", streamID),
			logger.Any("radio_id", radioID),
			logger.Any("talkgroup_id", talkgroupID))
	} else {
		// Existing stream - update
		stream.lastSeen = now
		stream.packetCount++
	}

	// If terminator, save to database and remove from active tracking
	if isTerminator {
		duration := stream.lastSeen.Sub(stream.startTime).Seconds()

		// Only save transmissions that are at least 0.5 seconds long
		// Very short transmissions are likely spurious or duplicate packets
		if duration >= 0.5 {
			tx := &database.Transmission{
				RadioID:     stream.radioID,
				TalkgroupID: stream.talkgroupID,
				Timeslot:    stream.timeslot,
				Duration:    duration,
				StreamID:    stream.streamID,
				StartTime:   stream.startTime,
				EndTime:     stream.lastSeen,
				RepeaterID:  stream.repeaterID,
				PacketCount: stream.packetCount,
			}

			if err := tl.repo.Create(tx); err != nil {
				tl.logger.Error("Failed to save transmission",
					logger.Error(err),
					logger.Any("stream_id", streamID))
			} else {
				tl.logger.Debug("Saved transmission",
					logger.Any("stream_id", streamID),
					logger.Any("radio_id", stream.radioID),
					logger.Any("talkgroup_id", stream.talkgroupID),
					logger.Any("duration", duration))
			}
		} else {
			tl.logger.Debug("Skipped saving very short transmission",
				logger.Any("stream_id", streamID),
				logger.Any("duration", duration),
				logger.Any("packet_count", stream.packetCount))
		}

		delete(tl.activeStreams, streamID)
	}
}

// CleanupStaleStreams removes streams that haven't seen activity recently
// Should be called periodically
func (tl *TransmissionLogger) CleanupStaleStreams(maxAge time.Duration) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	now := time.Now()
	for streamID, stream := range tl.activeStreams {
		// Only cleanup streams where the last packet was seen more than maxAge ago
		// This prevents cleaning up streams that just started (which would create <1s duplicates)
		timeSinceLastPacket := now.Sub(stream.lastSeen)
		if timeSinceLastPacket > maxAge {
			// Stream is stale - save it and remove from tracking
			duration := stream.lastSeen.Sub(stream.startTime).Seconds()

			// Only save transmissions that are at least 0.5 seconds long
			if duration >= 0.5 {
				tx := &database.Transmission{
					RadioID:     stream.radioID,
					TalkgroupID: stream.talkgroupID,
					Timeslot:    stream.timeslot,
					Duration:    duration,
					StreamID:    stream.streamID,
					StartTime:   stream.startTime,
					EndTime:     stream.lastSeen,
					RepeaterID:  stream.repeaterID,
					PacketCount: stream.packetCount,
				}

				if err := tl.repo.Create(tx); err != nil {
					tl.logger.Error("Failed to save stale transmission",
						logger.Error(err),
						logger.Any("stream_id", streamID))
				} else {
					tl.logger.Debug("Saved stale transmission",
						logger.Any("stream_id", streamID),
						logger.Any("radio_id", stream.radioID),
						logger.Any("duration", duration),
						logger.Any("time_since_last_packet", timeSinceLastPacket))
				}
			} else {
				tl.logger.Debug("Skipped saving very short stale stream",
					logger.Any("stream_id", streamID),
					logger.Any("duration", duration))
			}

			delete(tl.activeStreams, streamID)
		}
	}
}

// GetActiveStreamCount returns the number of currently active streams
func (tl *TransmissionLogger) GetActiveStreamCount() int {
	tl.mu.RLock()
	defer tl.mu.RUnlock()
	return len(tl.activeStreams)
}
