package ysf2dmr

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/codec"
	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/network"
	"github.com/dbehnke/dmr-nexus/pkg/protocol"
	"github.com/dbehnke/dmr-nexus/pkg/ysf"
)

// Bridge implements the YSF to DMR bridge
type Bridge struct {
	config    *Config
	ysfNet    *ysf.YSFNetwork
	dmrClient *network.Client
	converter *codec.Converter
	lookup    *Lookup
	logger    *logger.Logger

	// State for YSF -> DMR
	currentSrcID     uint32
	currentDstID     uint32
	currentFlco      protocol.FLCO
	streamActive     bool
	ysfFrames        uint
	dmrSeqNum        byte
	dmrStreamID      uint32
	lastFICH         *ysf.YSFFICH // Last valid FICH for fallback when CRC fails
	lastYSFFrameTime time.Time    // Last time a YSF frame was received (for watchdog)
	// FICH decode noise control
	invalidFICHCount   uint32    // number of invalid FICH seen since last log
	lastInvalidFICHLog time.Time // last time we emitted an invalid FICH summary

	// State for DMR -> YSF
	dmrRxActive  bool
	dmrFrames    uint
	dmrRxSrcID   uint32
	dmrRxDstID   uint32
	ysfTxFrameNo uint // YSF transmission frame number for DT1/DT2 sequencing

	// Sync
	mu sync.RWMutex
}

// NewBridge creates a new YSF2DMR bridge
func NewBridge(cfg *Config, lookup *Lookup, log *logger.Logger) *Bridge {
	b := &Bridge{
		config:       cfg,
		lookup:       lookup,
		logger:       log,
		converter:    codec.NewConverter(),
		currentDstID: cfg.DMR.StartupTG,
	}
	// Enable short-lived codec debug sampling (first few frames only)
	b.converter.SetDebugLogger(log)
	return b
}

// Start starts the YSF2DMR bridge
func (b *Bridge) Start(ctx context.Context) error {
	// Initialize YSF network
	ysfCfg := ysf.NetworkConfig{
		Callsign:     b.config.YSF.Callsign,
		ServerAddr:   b.config.YSF.ServerAddress,
		ServerPort:   b.config.YSF.ServerPort,
		PollInterval: 5 * time.Second,
		Debug:        b.config.YSF.Debug,
	}

	b.ysfNet = ysf.NewYSFNetwork(ysfCfg, b.logger.WithComponent("ysf"))
	if err := b.ysfNet.Start(ctx); err != nil {
		return fmt.Errorf("failed to start YSF network: %w", err)
	}

	b.logger.Info("YSF network started",
		logger.String("server", fmt.Sprintf("%s:%d", b.config.YSF.ServerAddress, b.config.YSF.ServerPort)),
		logger.String("callsign", b.config.YSF.Callsign))

	// Initialize DMR client (PEER mode)
	dmrSysCfg := config.SystemConfig{
		Mode:        "PEER",
		Enabled:     true,
		IP:          "0.0.0.0",
		Port:        0, // Let system assign port
		Passphrase:  b.config.DMR.Password,
		MasterIP:    b.config.DMR.ServerAddress,
		MasterPort:  b.config.DMR.ServerPort,
		Callsign:    b.config.DMR.Callsign,
		RadioID:     int(b.config.DMR.ID),
		RXFreq:      int(b.config.DMR.RXFreq),
		TXFreq:      int(b.config.DMR.TXFreq),
		TXPower:     b.config.DMR.TXPower,
		ColorCode:   b.config.DMR.ColorCode,
		Latitude:    b.config.DMR.Latitude,
		Longitude:   b.config.DMR.Longitude,
		Height:      b.config.DMR.Height,
		Location:    b.config.DMR.Location,
		Description: b.config.DMR.Description,
		URL:         b.config.DMR.URL,
		SoftwareID:  "YSF2DMR",
		PackageID:   "YSF2DMR Bridge",
	}

	b.dmrClient = network.NewClient(dmrSysCfg, b.logger.WithComponent("dmr"))

	// Set up DMR packet handler
	b.dmrClient.OnDMRD(b.handleDMRPacket)

	// Start DMR client in background
	go func() {
		if err := b.dmrClient.Start(ctx); err != nil && err != context.Canceled {
			b.logger.Error("DMR client error", logger.Error(err))
		}
	}()

	b.logger.Info("DMR client started",
		logger.String("server", fmt.Sprintf("%s:%d", b.config.DMR.ServerAddress, b.config.DMR.ServerPort)),
		logger.Uint32("dmr_id", b.config.DMR.ID))

	// Send startup PTT if configured
	if b.config.DMR.StartupPTT && b.config.DMR.StartupTG > 0 {
		go b.sendStartupPTT(ctx)
	}

	// Start bridge loops
	go b.ysfToDMRLoop(ctx)
	go b.dmrToYSFLoop(ctx)

	return nil
}

// ysfToDMRLoop handles YSF -> DMR conversion
func (b *Bridge) ysfToDMRLoop(ctx context.Context) {
	// YSF frame processing ticker
	ysfTicker := time.NewTicker(10 * time.Millisecond)
	defer ysfTicker.Stop()

	// DMR frame transmission ticker
	dmrTicker := time.NewTicker(time.Duration(codec.DMRFramePer) * time.Millisecond)
	defer dmrTicker.Stop()

	// Watchdog ticker to detect stalled streams (check every 500ms)
	watchdogTicker := time.NewTicker(500 * time.Millisecond)
	defer watchdogTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-watchdogTicker.C:
			// Check for stream inactivity timeout (3 seconds)
			b.mu.Lock()
			if b.streamActive && !b.lastYSFFrameTime.IsZero() {
				inactiveTime := time.Since(b.lastYSFFrameTime)
				if inactiveTime > 3*time.Second {
					b.logger.Warn("YSF stream timeout - no frames received",
						logger.Float64("inactive_seconds", inactiveTime.Seconds()))
					b.mu.Unlock()
					// Force termination
					if err := b.handleYSFTerminator(); err != nil {
						b.logger.Error("Failed to force stream termination", logger.Error(err))
					}
					continue
				}
			}
			b.mu.Unlock()

		case <-ysfTicker.C:
			// Read YSF frames from network
			data := b.ysfNet.Read()
			if data == nil {
				continue
			}

			// Check frame length
			if len(data) != ysf.YSFFrameLength {
				// Silently ignore poll/keepalive frames (14 bytes: YSFP/YSFU + callsign)
				if len(data) == 14 && len(data) >= 4 {
					tag := string(data[0:4])
					if tag == "YSFP" || tag == "YSFU" {
						// Poll acknowledgment or unlink, ignore silently
						continue
					}
				}
				// Log unexpected frame lengths
				b.logger.Warn("Invalid YSF frame length",
					logger.Int("length", len(data)))
				continue
			}

			// Process YSF frame
			if err := b.processYSFFrame(data); err != nil {
				b.logger.Error("Failed to process YSF frame",
					logger.Error(err))
			}

		case <-dmrTicker.C:
			// Get converted DMR frame from codec
			frame := make([]byte, 33)
			frameType := b.converter.GetDMR(frame)

			switch frameType {
			case codec.TagHeader:
				// Send DMR header
				b.logger.Info("Sending DMR header")
				if err := b.sendDMRHeader(); err != nil {
					b.logger.Error("Failed to send DMR header", logger.Error(err))
				}

			case codec.TagData:
				// Send DMR voice frame
				b.logger.Debug("Sending DMR voice frame")
				if err := b.sendDMRVoice(frame); err != nil {
					b.logger.Error("Failed to send DMR voice", logger.Error(err))
				}

			case codec.TagEOT:
				// Send DMR terminator
				b.logger.Info("Sending DMR terminator")
				if err := b.sendDMRTerminator(); err != nil {
					b.logger.Error("Failed to send DMR terminator", logger.Error(err))
				}
			}
		}
	}
}

// dmrToYSFLoop handles DMR -> YSF conversion
func (b *Bridge) dmrToYSFLoop(ctx context.Context) {
	ysfTicker := time.NewTicker(time.Duration(codec.YSFFramePer) * time.Millisecond)
	defer ysfTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ysfTicker.C:
			// Get converted YSF frame from codec
			frame := make([]byte, ysf.YSFHeaderLength)
			frameType := b.converter.GetYSF(frame)

			switch frameType {
			case codec.TagHeader:
				// Send YSF header
				if err := b.sendYSFHeader(); err != nil {
					b.logger.Error("Failed to send YSF header", logger.Error(err))
				}

			case codec.TagData:
				// Send YSF voice frame
				if err := b.sendYSFVoice(frame); err != nil {
					b.logger.Error("Failed to send YSF voice", logger.Error(err))
				}

			case codec.TagEOT:
				// Send YSF terminator
				if err := b.sendYSFTerminator(); err != nil {
					b.logger.Error("Failed to send YSF terminator", logger.Error(err))
				}
			}
		}
	}
}

// detectTerminator tries to detect a terminator frame by examining payload patterns
// when FICH CRC fails. Terminators often have silence/null patterns in voice data.
func (b *Bridge) detectTerminator(payload []byte) bool {
	// Check if we have enough data
	if len(payload) < 120 {
		return false
	}

	// Skip FICH area (first 30 bytes after sync) and look at voice/data payload
	// Terminators often have long sequences of zeros or repeated patterns
	voiceStart := 30
	voiceEnd := 120

	zeroCount := 0
	for i := voiceStart; i < voiceEnd && i < len(payload); i++ {
		if payload[i] == 0x00 {
			zeroCount++
		}
	}

	// If more than 80% of voice payload is zeros, likely a terminator
	threshold := int(float64(voiceEnd-voiceStart) * 0.8)
	return zeroCount > threshold
}

// processYSFFrame processes a received YSF frame
func (b *Bridge) processYSFFrame(data []byte) error {
	// Verify signature
	if string(data[0:4]) != "YSFD" {
		return fmt.Errorf("invalid YSF signature: %s", string(data[0:4]))
	}

	// Extract payload (after header)
	payload := data[35:]

	// Decode FICH
	fich := &ysf.YSFFICH{}
	valid, err := fich.Decode(payload)
	if err != nil {
		return fmt.Errorf("failed to decode FICH: %w", err)
	}

	// If FICH CRC failed, use last valid FICH and increment frame number
	// This matches MMDVMHost behavior - YSF frames often have corrupted FICH
	if !valid {
		// Seed lastFICH from best-effort parsed fields if we don't have one yet
		if b.lastFICH == nil {
			b.lastFICH = fich
			b.logger.Warn("Invalid FICH CRC on first frame; seeding from parsed fields",
				logger.Int("fi", int(fich.GetFI())),
				logger.Int("dt", int(fich.GetDT())),
				logger.Int("fn", int(fich.GetFN())),
				logger.Int("ft", int(fich.GetFT())))
			// Continue processing with the seeded FICH - don't drop the frame!
		} else {
			// Throttle noisy per-frame logs by aggregating invalid FICH counts
			now := time.Now()
			b.invalidFICHCount++
			if now.Sub(b.lastInvalidFICHLog) >= time.Second {
				b.logger.Debug("Invalid FICH CRCs in window",
					logger.Uint32("count", b.invalidFICHCount))
				b.lastInvalidFICHLog = now
				b.invalidFICHCount = 0
			}
			// Check if this might be a terminator by looking at payload patterns
			// Terminator frames often have all-zero or specific patterns
			isLikelyTerminator := b.detectTerminator(payload)

			if isLikelyTerminator {
				b.logger.Debug("Detected likely terminator frame despite invalid FICH")
				return b.handleYSFTerminator()
			}

			// Use last valid FICH and increment frame number
			fich = b.lastFICH
			// Ensure we treat subsequent frames as communication, not repeated headers
			fich.SetFI(ysf.YSFFICommunication)
			fich.SetDT(ysf.YSFDTVDMode2)
			fn := fich.GetFN() + 1
			ft := fich.GetFT()
			if fn > ft {
				fn = 0
			}
			fich.SetFN(fn)
			// Avoid per-frame log spam here; summary is emitted periodically above
		}
	} else {
		// Valid FICH - save it for future use
		b.lastFICH = fich
	}

	fi := fich.GetFI()
	dt := fich.GetDT()

	// Process based on frame type
	switch fi {
	case ysf.YSFFIHeader:
		return b.handleYSFHeader(data, payload)

	case ysf.YSFFITerminator:
		return b.handleYSFTerminator()

	case ysf.YSFFICommunication:
		if dt == ysf.YSFDTVDMode2 {
			// If stream not active, synthesize a stream start (we joined mid-transmission)
			if !b.streamActive {
				b.logger.Info("Starting YSF stream from mid-transmission (no header seen)")
				if err := b.handleYSFHeader(data, payload); err != nil {
					b.logger.Error("Failed to synthesize stream start", logger.Error(err))
					return err
				}
			}
			return b.handleYSFVoice(payload)
		} else {
			b.logger.Debug("Skipping non-VD Mode 2 communication frame",
				logger.Int("dt", int(dt)))
		}
	default:
		b.logger.Debug("Skipping unknown frame type",
			logger.Int("fi", int(fi)))
	}

	return nil
}

// handleYSFHeader processes a YSF header frame
func (b *Bridge) handleYSFHeader(data []byte, payload []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Reset FICH adaptive decode at the start of a new transmission
	ysf.ResetFICHAdaptation()

	// Extract source callsign from frame
	sourceCallsign := string(data[14:24])
	sourceCallsign = ysf.TrimCallsign(sourceCallsign)

	// Look up DMR ID for source
	srcID := b.lookup.FindID(sourceCallsign)
	if srcID == 0 {
		// Use a default ID if lookup fails
		srcID = b.config.DMR.ID
		b.logger.Warn("Could not find DMR ID for callsign, using default",
			logger.String("callsign", sourceCallsign),
			logger.Uint32("default_id", srcID))
	}

	b.currentSrcID = srcID
	b.streamActive = true
	b.ysfFrames = 0
	b.dmrSeqNum = 0
	b.dmrStreamID = rand.Uint32()
	b.lastYSFFrameTime = time.Now() // Start watchdog timer

	// Set up FLCO
	if b.config.DMR.StartupPrivate {
		b.currentFlco = protocol.FLCOUserUser
	} else {
		b.currentFlco = protocol.FLCOGroup
	}

	b.logger.Info("YSF voice stream started",
		logger.String("source", sourceCallsign),
		logger.Uint32("src_id", srcID),
		logger.Uint32("dst_id", b.currentDstID),
		logger.String("call_type", b.currentFlco.String()))

	// Initialize converter
	b.converter.PutYSFHeader()

	// Set DMR stream metadata for sync/embedded signalling
	b.converter.SetDMRStreamMetadata(
		b.config.DMR.Timeslot,
		srcID,
		b.currentDstID,
		uint8(b.currentFlco),
	)

	return nil
}

// handleYSFTerminator processes a YSF terminator frame
func (b *Bridge) handleYSFTerminator() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.streamActive {
		return nil
	}

	duration := float64(b.ysfFrames) / 10.0
	b.logger.Info("YSF voice stream ended",
		logger.Float64("duration_seconds", duration),
		logger.Uint("frames", b.ysfFrames))

	// Add hang time dummy frames to prevent abrupt cutoff
	// YSF frames are 100ms each (90ms + overhead)
	// MMDVM_CM calculates: extraFrames = (hangTime / 100ms) - ysfFrames - 2
	hangTimeMS := b.config.YSF.HangTime
	if hangTimeMS > 0 {
		extraFrames := (hangTimeMS / 100) - int(b.ysfFrames) - 2
		if extraFrames > 0 {
			b.logger.Debug("Adding hang time dummy frames",
				logger.Int("count", extraFrames))
			for i := 0; i < extraFrames; i++ {
				b.converter.PutDummyYSF()
			}
		}
	}

	b.streamActive = false
	b.ysfFrames = 0
	b.lastYSFFrameTime = time.Time{} // Reset watchdog timer

	// Send EOT to converter
	b.converter.PutYSFEOT()

	return nil
}

// handleYSFVoice processes a YSF voice frame
func (b *Bridge) handleYSFVoice(payload []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.streamActive {
		return nil
	}

	// Update watchdog timer
	b.lastYSFFrameTime = time.Now()

	// Extract voice data and send to converter
	b.converter.PutYSF(payload)
	b.ysfFrames++

	return nil
}

// handleDMRPacket processes a received DMR packet
func (b *Bridge) handleDMRPacket(packet *protocol.DMRDPacket) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if this is for our talkgroup
	if packet.DestinationID != b.currentDstID {
		return
	}

	// Determine frame type
	dataType := packet.DataType & 0x0F

	// Voice LC Header (start of transmission)
	if dataType == 0x01 {
		b.dmrRxActive = true
		b.dmrFrames = 0
		b.dmrRxSrcID = packet.SourceID
		b.dmrRxDstID = packet.DestinationID

		// Get source callsign
		srcCallsign := b.lookup.FindCallsign(packet.SourceID)
		if srcCallsign == "" {
			srcCallsign = fmt.Sprintf("%d", packet.SourceID)
		}

		b.logger.Info("DMR voice stream started",
			logger.String("source", srcCallsign),
			logger.Uint32("src_id", packet.SourceID),
			logger.Uint32("dst_id", packet.DestinationID))

		b.converter.PutDMRHeader()
		return
	}

	// Terminator
	if dataType == 0x02 {
		if b.dmrRxActive {
			duration := float64(b.dmrFrames) / 16.667
			b.logger.Info("DMR voice stream ended",
				logger.Float64("duration_seconds", duration),
				logger.Uint("frames", b.dmrFrames))

			b.converter.PutDMREOT()
			b.dmrRxActive = false
			b.dmrFrames = 0
		}
		return
	}

	// Voice frames
	if b.dmrRxActive && (dataType == 0x00 || (dataType >= 0x03 && dataType <= 0x0A)) {
		b.converter.PutDMR(packet.Payload)
		b.dmrFrames++
	}
}

// sendDMRHeader sends a DMR header packet
func (b *Bridge) sendDMRHeader() error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Convert FLCO to CallType: FLCOGroup (0x00) -> CallTypeGroup (0), FLCOUserUser (0x03) -> CallTypePrivate (1)
	callType := protocol.CallTypeGroup
	if b.currentFlco == protocol.FLCOUserUser {
		callType = protocol.CallTypePrivate
	}

	// Build proper Voice LC Header payload
	payload := protocol.BuildVoiceLCHeader(b.currentSrcID, b.currentDstID, b.currentFlco)

	packet := &protocol.DMRDPacket{
		Sequence:      b.dmrSeqNum,
		SourceID:      b.currentSrcID,
		DestinationID: b.currentDstID,
		RepeaterID:    b.config.DMR.ID,
		Timeslot:      b.config.DMR.Timeslot,
		CallType:      callType,
		FrameType:     protocol.FrameTypeVoiceHeader,
		DataType:      0x01, // Voice LC Header
		StreamID:      b.dmrStreamID,
		Payload:       payload,
	}

	b.dmrSeqNum++

	// Debug: log header details and first bytes of LC payload
	if b.dmrClient != nil {
		b.logger.Debug("Sending DMR Voice LC Header",
			logger.Uint32("src_id", packet.SourceID),
			logger.Uint32("dst_id", packet.DestinationID),
			logger.Int("ts", packet.Timeslot),
			logger.Int("call_type", packet.CallType),
			logger.Uint32("stream_id", packet.StreamID),
			logger.String("lc_bytes", fmt.Sprintf("%02X %02X %02X %02X %02X %02X %02X %02X %02X",
				packet.Payload[0], packet.Payload[1], packet.Payload[2], packet.Payload[3],
				packet.Payload[4], packet.Payload[5], packet.Payload[6], packet.Payload[7], packet.Payload[8])),
		)
	}

	if b.dmrClient != nil {
		return b.dmrClient.SendDMRD(packet)
	}
	return nil
}

// sendDMRVoice sends a DMR voice packet
func (b *Bridge) sendDMRVoice(voiceData []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Calculate voice sequence (0-5 repeating, with special handling for sync)
	voiceSeq := byte((b.dmrSeqNum - 1) % 6)
	var dataType byte
	if voiceSeq == 0 {
		dataType = 0x00 // Voice Sync
	} else {
		dataType = 0x03 + voiceSeq - 1 // 0x03-0x07 for frames A-F
	}

	// Convert FLCO to CallType: FLCOGroup (0x00) -> CallTypeGroup (0), FLCOUserUser (0x03) -> CallTypePrivate (1)
	callType := protocol.CallTypeGroup
	if b.currentFlco == protocol.FLCOUserUser {
		callType = protocol.CallTypePrivate
	}

	packet := &protocol.DMRDPacket{
		Sequence:      b.dmrSeqNum,
		SourceID:      b.currentSrcID,
		DestinationID: b.currentDstID,
		RepeaterID:    b.config.DMR.ID,
		Timeslot:      b.config.DMR.Timeslot,
		CallType:      callType,
		FrameType:     protocol.FrameTypeVoice,
		DataType:      dataType,
		StreamID:      b.dmrStreamID,
		Payload:       make([]byte, 33),
	}

	// Copy voice data
	if len(voiceData) > 0 {
		copy(packet.Payload, voiceData)
	}

	b.dmrSeqNum++

	// Debug: log voice frame sequencing
	b.logger.Debug("Sending DMR Voice",
		logger.Uint32("src_id", packet.SourceID),
		logger.Uint32("dst_id", packet.DestinationID),
		logger.Int("ts", packet.Timeslot),
		logger.Int("call_type", packet.CallType),
		logger.Int("data_type", int(packet.DataType)),
		logger.Int("seq", int(packet.Sequence)),
	)

	if b.dmrClient != nil {
		return b.dmrClient.SendDMRD(packet)
	}
	return nil
}

// sendDMRTerminator sends a DMR terminator packet
func (b *Bridge) sendDMRTerminator() error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Convert FLCO to CallType: FLCOGroup (0x00) -> CallTypeGroup (0), FLCOUserUser (0x03) -> CallTypePrivate (1)
	callType := protocol.CallTypeGroup
	if b.currentFlco == protocol.FLCOUserUser {
		callType = protocol.CallTypePrivate
	}

	// Build proper terminator payload with LC data
	payload := protocol.BuildVoiceTerminatorPayload(b.currentSrcID, b.currentDstID, b.currentFlco)

	packet := &protocol.DMRDPacket{
		Sequence:      b.dmrSeqNum,
		SourceID:      b.currentSrcID,
		DestinationID: b.currentDstID,
		RepeaterID:    b.config.DMR.ID,
		Timeslot:      b.config.DMR.Timeslot,
		CallType:      callType,
		FrameType:     protocol.FrameTypeVoiceTerminator,
		DataType:      0x02, // Terminator with LC
		StreamID:      b.dmrStreamID,
		Payload:       payload,
	}

	b.dmrSeqNum++

	// Debug: log terminator details and first bytes of LC payload
	if b.dmrClient != nil {
		b.logger.Debug("Sending DMR Voice Terminator",
			logger.Uint32("src_id", packet.SourceID),
			logger.Uint32("dst_id", packet.DestinationID),
			logger.Int("ts", packet.Timeslot),
			logger.Int("call_type", packet.CallType),
			logger.Uint32("stream_id", packet.StreamID),
			logger.String("lc_bytes", fmt.Sprintf("%02X %02X %02X %02X %02X %02X %02X %02X %02X",
				packet.Payload[0], packet.Payload[1], packet.Payload[2], packet.Payload[3],
				packet.Payload[4], packet.Payload[5], packet.Payload[6], packet.Payload[7], packet.Payload[8])),
		)
	}

	if b.dmrClient != nil {
		return b.dmrClient.SendDMRD(packet)
	}
	return nil
}

// sendYSFHeader sends a YSF header frame
func (b *Bridge) sendYSFHeader() error {
	b.ysfTxFrameNo = 0 // Reset frame number for new transmission

	frame := ysf.NewYSFFrame()

	// Set gateway callsign
	frame.Gateway = ysf.TrimCallsign(b.config.YSF.Callsign)

	// Set source callsign from DMR ID
	srcCallsign := b.lookup.FindCallsign(b.dmrRxSrcID)
	if srcCallsign == "" {
		srcCallsign = fmt.Sprintf("%d", b.dmrRxSrcID)
	}
	frame.Source = srcCallsign

	// Set destination
	frame.Dest = "ALL"

	// Build frame
	ysfData := make([]byte, ysf.YSFFrameLength)
	copy(ysfData[0:4], frame.Signature)
	copy(ysfData[4:14], []byte(frame.Gateway))
	copy(ysfData[14:24], []byte(frame.Source))
	copy(ysfData[24:34], []byte(frame.Dest))
	ysfData[34] = 0 // Frame counter

	// Set FICH for header with configuration
	fich := &ysf.YSFFICH{}
	fich.SetFI(ysf.YSFFIHeader)
	fich.SetCS(b.config.YSF.FICHCallSign)
	fich.SetCM(b.config.YSF.FICHCallMode)
	fich.SetBN(0)
	fich.SetBT(0)
	fich.SetFN(0)
	fich.SetFT(b.config.YSF.FICHFrameTotal)
	fich.SetDev(0)
	fich.SetMR(b.config.YSF.FICHMessageRoute)
	fich.SetVoIP(b.config.YSF.FICHVOIP)
	fich.SetDT(b.config.YSF.FICHDataType)
	fich.SetSQL(b.config.YSF.FICHSQLType)
	fich.SetSQ(b.config.YSF.FICHSQLCode)
	_ = fich.Encode(ysfData[35:])

	// Debug: log YSF header being sent
	b.logger.Info("Sending YSF Header",
		logger.String("gateway", frame.Gateway),
		logger.String("source", frame.Source),
		logger.String("dest", frame.Dest),
	)

	return b.ysfNet.Write(ysfData)
}

// sendYSFVoice sends a YSF voice frame
func (b *Bridge) sendYSFVoice(voiceData []byte) error {
	frame := ysf.NewYSFFrame()

	// Set callsigns
	frame.Gateway = ysf.TrimCallsign(b.config.YSF.Callsign)
	srcCallsign := b.lookup.FindCallsign(b.dmrRxSrcID)
	if srcCallsign == "" {
		srcCallsign = fmt.Sprintf("%d", b.dmrRxSrcID)
	}
	frame.Source = srcCallsign
	dstCallsign := b.lookup.FindCallsign(b.dmrRxDstID)
	if dstCallsign == "" {
		dstCallsign = fmt.Sprintf("TG %d", b.dmrRxDstID)
	}
	frame.Dest = "ALL"

	// Calculate frame number within transmission (0-7, wraps)
	fn := b.ysfTxFrameNo % (uint(b.config.YSF.FICHFrameTotal) + 1)

	// Build frame
	ysfData := make([]byte, ysf.YSFFrameLength)
	copy(ysfData[0:4], frame.Signature)
	copy(ysfData[4:14], []byte(frame.Gateway))
	copy(ysfData[14:24], []byte(frame.Source))
	copy(ysfData[24:34], []byte(frame.Dest))
	ysfData[34] = byte((b.ysfTxFrameNo & 0x7F) << 1) // Frame counter

	// Copy voice payload (120 bytes) produced by converter into payload area
	if len(voiceData) > 0 {
		copy(ysfData[35:], voiceData)
	}

	// Write FICH after copying voice data - FICH is interleaved with voice payload
	fichVoice := &ysf.YSFFICH{}
	fichVoice.SetFI(ysf.YSFFICommunication)
	fichVoice.SetCS(b.config.YSF.FICHCallSign)
	fichVoice.SetCM(b.config.YSF.FICHCallMode)
	fichVoice.SetBN(byte(fn))
	fichVoice.SetBT(byte(b.config.YSF.FICHFrameTotal))
	fichVoice.SetFN(byte(fn))
	fichVoice.SetFT(b.config.YSF.FICHFrameTotal)
	fichVoice.SetDev(0)
	fichVoice.SetMR(b.config.YSF.FICHMessageRoute)
	fichVoice.SetVoIP(b.config.YSF.FICHVOIP)
	fichVoice.SetDT(b.config.YSF.FICHDataType)
	fichVoice.SetSQL(b.config.YSF.FICHSQLType)
	fichVoice.SetSQ(b.config.YSF.FICHSQLCode)
	_ = fichVoice.Encode(ysfData[35:])

	// Write VD Mode 2 data (DT1/DT2 and callsigns) at specific frame numbers
	// This matches MMDVM_CM behavior
	payload := ysf.NewYSFPayload()
	var dch [10]byte

	switch fn {
	case 0:
		// Frame 0: Radio ID (first 5 bytes asterisks, last 5 bytes radio ID)
		for i := 0; i < 5; i++ {
			dch[i] = '*'
		}
		radioID := b.config.YSF.YSFRadioID
		if len(radioID) > 5 {
			radioID = radioID[:5]
		}
		copy(dch[5:], radioID)
		_ = payload.WriteVDMode2Data(ysfData[35:], dch[:])

	case 1:
		// Frame 1: Source callsign (padded to 10 bytes)
		src := srcCallsign
		if len(src) > 10 {
			src = src[:10]
		}
		copy(dch[:], src)
		_ = payload.WriteVDMode2Data(ysfData[35:], dch[:])

	case 2:
		// Frame 2: Destination callsign (padded to 10 bytes)
		dst := dstCallsign
		if len(dst) > 10 {
			dst = dst[:10]
		}
		copy(dch[:], dst)
		_ = payload.WriteVDMode2Data(ysfData[35:], dch[:])

	case 5:
		// Frame 5: Radio ID (repeated)
		for i := 0; i < 5; i++ {
			dch[i] = ' '
		}
		radioID := b.config.YSF.YSFRadioID
		if len(radioID) > 5 {
			radioID = radioID[:5]
		}
		copy(dch[5:], radioID)
		_ = payload.WriteVDMode2Data(ysfData[35:], dch[:])

	case 6:
		// Frame 6: DT1 data
		if len(b.config.YSF.YSFDT1) >= 10 {
			copy(dch[:], b.config.YSF.YSFDT1[:10])
		} else {
			copy(dch[:], b.config.YSF.YSFDT1)
		}
		_ = payload.WriteVDMode2Data(ysfData[35:], dch[:])

	case 7:
		// Frame 7: DT2 data
		if len(b.config.YSF.YSFDT2) >= 10 {
			copy(dch[:], b.config.YSF.YSFDT2[:10])
		} else {
			copy(dch[:], b.config.YSF.YSFDT2)
		}
		_ = payload.WriteVDMode2Data(ysfData[35:], dch[:])

	default:
		// Other frames: empty data
		_ = payload.WriteVDMode2Data(ysfData[35:], dch[:])
	}

	b.ysfTxFrameNo++

	// Debug: log YSF voice being sent
	b.logger.Debug("Sending YSF Voice",
		logger.String("gateway", frame.Gateway),
		logger.String("source", frame.Source),
		logger.String("dest", frame.Dest),
		logger.Int("frame_no", int(fn)),
		logger.Int("payload_len", len(voiceData)),
	)

	return b.ysfNet.Write(ysfData)
}

// sendYSFTerminator sends a YSF terminator frame
func (b *Bridge) sendYSFTerminator() error {
	frame := ysf.NewYSFFrame()

	// Set callsigns
	frame.Gateway = ysf.TrimCallsign(b.config.YSF.Callsign)
	srcCallsign := b.lookup.FindCallsign(b.dmrRxSrcID)
	if srcCallsign == "" {
		srcCallsign = fmt.Sprintf("%d", b.dmrRxSrcID)
	}
	frame.Source = srcCallsign
	frame.Dest = "ALL"

	// Build frame
	ysfData := make([]byte, ysf.YSFFrameLength)
	copy(ysfData[0:4], frame.Signature)
	copy(ysfData[4:14], []byte(frame.Gateway))
	copy(ysfData[14:24], []byte(frame.Source))
	copy(ysfData[24:34], []byte(frame.Dest))
	ysfData[34] = 0 // Frame counter

	// Set FICH for terminator with configuration
	fich := &ysf.YSFFICH{}
	fich.SetFI(ysf.YSFFITerminator)
	fich.SetCS(b.config.YSF.FICHCallSign)
	fich.SetCM(b.config.YSF.FICHCallMode)
	fich.SetBN(0)
	fich.SetBT(0)
	fich.SetFN(0)
	fich.SetFT(b.config.YSF.FICHFrameTotal)
	fich.SetDev(0)
	fich.SetMR(b.config.YSF.FICHMessageRoute)
	fich.SetVoIP(b.config.YSF.FICHVOIP)
	fich.SetDT(b.config.YSF.FICHDataType)
	fich.SetSQL(b.config.YSF.FICHSQLType)
	fich.SetSQ(b.config.YSF.FICHSQLCode)
	_ = fich.Encode(ysfData[35:])

	// Debug: log YSF terminator being sent
	b.logger.Debug("Sending YSF Terminator",
		logger.String("gateway", frame.Gateway),
		logger.String("source", frame.Source),
		logger.String("dest", frame.Dest),
	)

	return b.ysfNet.Write(ysfData)
}

// sendStartupPTT sends a PTT (dummy DMR frame) on startup to activate the talkgroup
// This waits for the DMR client to be connected before sending
func (b *Bridge) sendStartupPTT(ctx context.Context) {
	// Wait for DMR client to be connected
	// Check connection status every 100ms for up to 30 seconds
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			b.logger.Warn("Timeout waiting for DMR connection, startup PTT not sent")
			return
		case <-ticker.C:
			// Check if DMR client is connected
			// The DMR client doesn't expose a public IsConnected() method,
			// so we'll wait a few seconds and then try to send
			// For now, just wait 5 seconds after startup
			time.Sleep(5 * time.Second)

			// Determine FLCO based on StartupPrivate setting
			flco := protocol.FLCOGroup
			if b.config.DMR.StartupPrivate {
				flco = protocol.FLCOUserUser
			}

			// Send the dummy DMR frame to activate the talkgroup
			if err := b.SendDummyDMR(b.config.DMR.ID, b.config.DMR.StartupTG, flco); err != nil {
				b.logger.Error("Failed to send startup PTT",
					logger.Error(err),
					logger.Uint32("tg", b.config.DMR.StartupTG))
			} else {
				callType := "TG"
				if b.config.DMR.StartupPrivate {
					callType = "Private"
				}
				b.logger.Info("Sent startup PTT to activate talkgroup",
					logger.String("call_type", callType),
					logger.Uint32("dst_id", b.config.DMR.StartupTG),
					logger.Uint32("src_id", b.config.DMR.ID))
			}
			return
		}
	}
}

// SendDummyDMR sends a dummy DMR frame (PTT/Unlink signal)
// This sends a minimal DMR transmission (header + terminator) to signal presence or unlink
func (b *Bridge) SendDummyDMR(srcID, dstID uint32, flco protocol.FLCO) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.dmrClient == nil {
		return fmt.Errorf("DMR client not initialized")
	}

	// Generate stream ID
	streamID := rand.Uint32()

	// Convert FLCO to CallType
	callType := protocol.CallTypeGroup
	if flco == protocol.FLCOUserUser {
		callType = protocol.CallTypePrivate
	}

	// Send Voice LC Header
	headerPayload := protocol.BuildVoiceLCHeader(srcID, dstID, flco)
	headerPacket := &protocol.DMRDPacket{
		Sequence:      0,
		SourceID:      srcID,
		DestinationID: dstID,
		RepeaterID:    b.config.DMR.ID,
		Timeslot:      b.config.DMR.Timeslot,
		CallType:      callType,
		FrameType:     protocol.FrameTypeVoiceHeader,
		DataType:      0x01, // Voice LC Header
		StreamID:      streamID,
		Payload:       headerPayload,
	}

	// Send header 3 times (as per DMR spec)
	for i := byte(0); i < 3; i++ {
		headerPacket.Sequence = i
		if err := b.dmrClient.SendDMRD(headerPacket); err != nil {
			return fmt.Errorf("failed to send DMR dummy header: %w", err)
		}
	}

	// Send Voice Terminator
	termPayload := protocol.BuildVoiceTerminatorPayload(srcID, dstID, flco)
	termPacket := &protocol.DMRDPacket{
		Sequence:      3,
		SourceID:      srcID,
		DestinationID: dstID,
		RepeaterID:    b.config.DMR.ID,
		Timeslot:      b.config.DMR.Timeslot,
		CallType:      callType,
		FrameType:     protocol.FrameTypeVoiceTerminator,
		DataType:      0x02, // Terminator with LC
		StreamID:      streamID,
		Payload:       termPayload,
	}

	if err := b.dmrClient.SendDMRD(termPacket); err != nil {
		return fmt.Errorf("failed to send DMR dummy terminator: %w", err)
	}

	b.logger.Debug("Sent DMR dummy frame (PTT/Unlink)",
		logger.Uint32("src_id", srcID),
		logger.Uint32("dst_id", dstID),
		logger.String("flco", flco.String()),
		logger.Uint32("stream_id", streamID))

	return nil
}

// Stop stops the bridge
func (b *Bridge) Stop() error {
	b.logger.Info("Stopping YSF2DMR bridge")

	// Close YSF network
	if b.ysfNet != nil {
		if err := b.ysfNet.Close(); err != nil {
			b.logger.Error("Failed to close YSF network", logger.Error(err))
		}
	}

	// DMR client will be closed by context cancellation

	b.logger.Info("YSF2DMR bridge stopped")
	return nil
}
