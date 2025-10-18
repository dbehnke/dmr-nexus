package network

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/protocol"
)

// OpenBridgeClient represents a UDP client for OPENBRIDGE mode
// OpenBridge is a stateless protocol that uses HMAC-SHA1 for authentication
// on every packet. It's designed for DMR+ and Brandmeister connectivity.
type OpenBridgeClient struct {
	config      config.SystemConfig
	log         *logger.Logger
	conn        *net.UDPConn
	targetAddr  *net.UDPAddr
	dmrdHandler func(*protocol.DMRDPacket)
	handlerMu   sync.RWMutex
}

// NewOpenBridgeClient creates a new OpenBridge client
func NewOpenBridgeClient(cfg config.SystemConfig, log *logger.Logger) *OpenBridgeClient {
	return &OpenBridgeClient{
		config: cfg,
		log:    log.WithComponent("network.openbridge"),
	}
}

// Start starts the OpenBridge client
func (c *OpenBridgeClient) Start(ctx context.Context) error {
	// Resolve target address
	targetAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.config.TargetIP, c.config.TargetPort))
	if err != nil {
		return fmt.Errorf("failed to resolve target address: %w", err)
	}
	c.targetAddr = targetAddr

	// Create local UDP address
	localAddr := &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: c.config.Port,
	}

	// Create UDP connection
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}
	c.conn = conn
	defer c.conn.Close()

	c.log.Info("OpenBridge client started",
		logger.String("target", c.targetAddr.String()),
		logger.String("local", conn.LocalAddr().String()),
		logger.Int("network_id", c.config.NetworkID),
		logger.Bool("both_slots", c.config.BothSlots))

	// Start receive loop
	errChan := make(chan error, 1)
	go func() {
		errChan <- c.receiveLoop(ctx)
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// receiveLoop receives and processes incoming packets
func (c *OpenBridgeClient) receiveLoop(ctx context.Context) error {
	buf := make([]byte, 2048)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, addr, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.log.Error("Failed to read from UDP", logger.Error(err))
			continue
		}

		// Process packet
		go c.handlePacket(buf[:n], addr)
	}
}

// handlePacket processes a received packet
func (c *OpenBridgeClient) handlePacket(data []byte, addr *net.UDPAddr) {
	// OpenBridge only handles DMRD packets
	if len(data) != protocol.DMRDOpenBridgePacketSize {
		c.log.Debug("Received non-OpenBridge packet",
			logger.Int("size", len(data)),
			logger.String("from", addr.String()))
		return
	}

	// Check signature
	if string(data[0:4]) != protocol.PacketTypeDMRD {
		c.log.Debug("Received non-DMRD packet",
			logger.String("from", addr.String()))
		return
	}

	// Parse DMRD packet
	packet := &protocol.DMRDPacket{}
	if err := packet.Parse(data); err != nil {
		c.log.Error("Failed to parse DMRD packet",
			logger.Error(err),
			logger.String("from", addr.String()))
		return
	}

	// Verify HMAC
	if !packet.VerifyOpenBridgeHMAC(c.config.Passphrase) {
		c.log.Warn("HMAC verification failed",
			logger.String("from", addr.String()),
			logger.Uint64("src", uint64(packet.SourceID)),
			logger.Uint64("dst", uint64(packet.DestinationID)))
		return
	}

	c.log.Debug("Received DMRD packet",
		logger.Uint64("src", uint64(packet.SourceID)),
		logger.Uint64("dst", uint64(packet.DestinationID)),
		logger.Int("ts", packet.Timeslot),
		logger.Uint64("stream", uint64(packet.StreamID)))

	// Call handler if set
	c.handlerMu.RLock()
	handler := c.dmrdHandler
	c.handlerMu.RUnlock()

	if handler != nil {
		handler(packet)
	}
}

// SendDMRD sends a DMRD packet with OpenBridge HMAC
func (c *OpenBridgeClient) SendDMRD(packet *protocol.DMRDPacket) error {
	// Apply BothSlots filtering
	// OpenBridge typically only uses TS1 for group calls
	// TS2 is only used for private calls unless both_slots is enabled
	if !c.config.BothSlots {
		// Allow TS1 for any call type
		// Allow TS2 only for private calls
		if packet.Timeslot == protocol.Timeslot2 && packet.CallType == protocol.CallTypeGroup {
			c.log.Debug("Filtering TS2 group call (both_slots=false)")
			return nil
		}
	}

	// Set network ID in repeater ID field
	packet.RepeaterID = uint32(c.config.NetworkID)

	// Add HMAC
	if err := packet.AddOpenBridgeHMAC(c.config.Passphrase); err != nil {
		return fmt.Errorf("failed to add HMAC: %w", err)
	}

	// Encode packet
	data, err := packet.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode packet: %w", err)
	}

	// Send to target
	_, err = c.conn.WriteToUDP(data, c.targetAddr)
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	c.log.Debug("Sent DMRD packet",
		logger.Uint64("src", uint64(packet.SourceID)),
		logger.Uint64("dst", uint64(packet.DestinationID)),
		logger.Int("ts", packet.Timeslot),
		logger.Uint64("stream", uint64(packet.StreamID)))

	return nil
}

// SetDMRDHandler sets the handler for received DMRD packets
func (c *OpenBridgeClient) SetDMRDHandler(handler func(*protocol.DMRDPacket)) {
	c.handlerMu.Lock()
	defer c.handlerMu.Unlock()
	c.dmrdHandler = handler
}

// Stop stops the OpenBridge client
func (c *OpenBridgeClient) Stop() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
