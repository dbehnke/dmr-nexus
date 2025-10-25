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
	currentSrcID   uint32
	currentDstID   uint32
	currentFlco    protocol.FLCO
	streamActive   bool
	ysfFrames      uint
	dmrSeqNum      byte
	dmrStreamID    uint32

	// State for DMR -> YSF
	dmrRxActive    bool
	dmrFrames      uint
	dmrRxSrcID     uint32
	dmrRxDstID     uint32

	// Sync
	mu sync.RWMutex
}

// NewBridge creates a new YSF2DMR bridge
func NewBridge(cfg *Config, lookup *Lookup, log *logger.Logger) *Bridge {
	return &Bridge{
		config:       cfg,
		lookup:       lookup,
		logger:       log,
		converter:    codec.NewConverter(),
		currentDstID: cfg.DMR.StartupTG,
	}
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
		Mode:       "PEER",
		Enabled:    true,
		IP:         "0.0.0.0",
		Port:       0, // Let system assign port
		Passphrase: b.config.DMR.Password,
		MasterIP:   b.config.DMR.ServerAddress,
		MasterPort: b.config.DMR.ServerPort,
		Callsign:   b.config.DMR.Callsign,
		RadioID:    int(b.config.DMR.ID),
		RXFreq:     int(b.config.DMR.RXFreq),
		TXFreq:     int(b.config.DMR.TXFreq),
		TXPower:    b.config.DMR.TXPower,
		ColorCode:  b.config.DMR.ColorCode,
		Latitude:   b.config.DMR.Latitude,
		Longitude:  b.config.DMR.Longitude,
		Height:     b.config.DMR.Height,
		Location:   b.config.DMR.Location,
		Description: b.config.DMR.Description,
		URL:        b.config.DMR.URL,
		SoftwareID: "YSF2DMR",
		PackageID:  "YSF2DMR Bridge",
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

	for {
		select {
		case <-ctx.Done():
			return

		case <-ysfTicker.C:
			// Read YSF frames from network
			data := b.ysfNet.Read()
			if data == nil {
				continue
			}

			if len(data) != ysf.YSFFrameLength {
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
				if err := b.sendDMRHeader(); err != nil {
					b.logger.Error("Failed to send DMR header", logger.Error(err))
				}

			case codec.TagData:
				// Send DMR voice frame
				if err := b.sendDMRVoice(frame); err != nil {
					b.logger.Error("Failed to send DMR voice", logger.Error(err))
				}

			case codec.TagEOT:
				// Send DMR terminator
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

	if !valid {
		b.logger.Debug("Invalid FICH, skipping frame")
		return nil
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
			return b.handleYSFVoice(payload)
		}
	}

	return nil
}

// handleYSFHeader processes a YSF header frame
func (b *Bridge) handleYSFHeader(data []byte, payload []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

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

	b.streamActive = false
	b.ysfFrames = 0

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

	packet := &protocol.DMRDPacket{
		Sequence:      b.dmrSeqNum,
		SourceID:      b.currentSrcID,
		DestinationID: b.currentDstID,
		RepeaterID:    b.config.DMR.ID,
		Timeslot:      2, // Always use timeslot 2
		CallType:      int(b.currentFlco),
		FrameType:     protocol.FrameTypeVoiceHeader,
		DataType:      0x01, // Voice LC Header
		StreamID:      b.dmrStreamID,
		Payload:       make([]byte, 33),
	}

	b.dmrSeqNum++

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
	dataType := byte(0x00) // Voice Sync/Voice
	if voiceSeq == 0 {
		dataType = 0x00 // Voice Sync
	} else {
		dataType = 0x03 + voiceSeq - 1 // 0x03-0x07 for frames A-F
	}

	packet := &protocol.DMRDPacket{
		Sequence:      b.dmrSeqNum,
		SourceID:      b.currentSrcID,
		DestinationID: b.currentDstID,
		RepeaterID:    b.config.DMR.ID,
		Timeslot:      2,
		CallType:      int(b.currentFlco),
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

	if b.dmrClient != nil {
		return b.dmrClient.SendDMRD(packet)
	}
	return nil
}

// sendDMRTerminator sends a DMR terminator packet
func (b *Bridge) sendDMRTerminator() error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	packet := &protocol.DMRDPacket{
		Sequence:      b.dmrSeqNum,
		SourceID:      b.currentSrcID,
		DestinationID: b.currentDstID,
		RepeaterID:    b.config.DMR.ID,
		Timeslot:      2,
		CallType:      int(b.currentFlco),
		FrameType:     protocol.FrameTypeVoiceTerminator,
		DataType:      0x02, // Terminator with LC
		StreamID:      b.dmrStreamID,
		Payload:       make([]byte, 33),
	}

	b.dmrSeqNum++

	if b.dmrClient != nil {
		return b.dmrClient.SendDMRD(packet)
	}
	return nil
}

// sendYSFHeader sends a YSF header frame
func (b *Bridge) sendYSFHeader() error {
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

	// Set FICH for header
	fich := &ysf.YSFFICH{}
	fich.SetFI(ysf.YSFFIHeader)
	fich.SetDT(ysf.YSFDTVDMode2)
	_ = fich.Encode(ysfData[35:])

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
	frame.Dest = "ALL"

	// Build frame
	ysfData := make([]byte, ysf.YSFFrameLength)
	copy(ysfData[0:4], frame.Signature)
	copy(ysfData[4:14], []byte(frame.Gateway))
	copy(ysfData[14:24], []byte(frame.Source))
	copy(ysfData[24:34], []byte(frame.Dest))
	ysfData[34] = 0 // Frame counter

	// Set FICH for communication
	fich := &ysf.YSFFICH{}
	fich.SetFI(ysf.YSFFICommunication)
	fich.SetDT(ysf.YSFDTVDMode2)
	_ = fich.Encode(ysfData[35:])

	// Copy voice data
	if len(voiceData) > 0 {
		copy(ysfData[35:], voiceData)
	}

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

	// Set FICH for terminator
	fich := &ysf.YSFFICH{}
	fich.SetFI(ysf.YSFFITerminator)
	fich.SetDT(ysf.YSFDTVDMode2)
	_ = fich.Encode(ysfData[35:])

	return b.ysfNet.Write(ysfData)
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
