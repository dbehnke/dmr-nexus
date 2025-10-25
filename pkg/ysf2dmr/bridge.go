package ysf2dmr

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/codec"
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

	// State
	currentSrcID uint32
	currentDstID uint32
	currentFlco  protocol.FLCO
	streamActive bool
	ysfFrames    uint
	dmrFrames    uint

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
	// TODO: This needs to be implemented similar to dmr-nexus client mode
	// For now, this is a placeholder
	b.logger.Info("DMR client initialization pending implementation")

	// Start bridge loops
	go b.ysfToDMRLoop(ctx)
	go b.dmrToYSFLoop(ctx)

	return nil
}

// ysfToDMRLoop handles YSF -> DMR conversion
func (b *Bridge) ysfToDMRLoop(ctx context.Context) {
	ysfTicker := time.NewTicker(time.Duration(codec.YSFFramePer) * time.Millisecond)
	defer ysfTicker.Stop()

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
		}
	}
}

// dmrToYSFLoop handles DMR -> YSF conversion
func (b *Bridge) dmrToYSFLoop(ctx context.Context) {
	dmrTicker := time.NewTicker(time.Duration(codec.DMRFramePer) * time.Millisecond)
	defer dmrTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-dmrTicker.C:
			// TODO: Read DMR frames from client and convert to YSF
			// This needs DMR client implementation
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

// sendDMRPacket sends a DMR packet (placeholder)
func (b *Bridge) sendDMRPacket(packet *protocol.DMRDPacket) error {
	// TODO: Implement once DMR client is ready
	if b.dmrClient != nil {
		// Send packet via DMR client
	}
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

	// Close DMR client
	if b.dmrClient != nil {
		// TODO: Close DMR client
	}

	b.logger.Info("YSF2DMR bridge stopped")
	return nil
}
