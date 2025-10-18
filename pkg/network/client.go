package network

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/protocol"
)

// ConnectionState represents the state of the peer connection
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateRPTLSent
	StateAuthenticated
	StateConfigSent
	StateConnected
)

// Client represents a UDP client for PEER mode
type Client struct {
	config      config.SystemConfig
	log         *logger.Logger
	conn        *net.UDPConn
	masterAddr  *net.UDPAddr
	state       ConnectionState
	stateMu     sync.RWMutex
	salt        []byte
	dmrdHandler func(*protocol.DMRDPacket)
	handlerMu   sync.RWMutex
	lastPing    time.Time
	lastPingMu  sync.RWMutex
}

// NewClient creates a new UDP client for PEER mode
func NewClient(cfg config.SystemConfig, log *logger.Logger) *Client {
	return &Client{
		config:   cfg,
		log:      log.WithComponent("network.client"),
		state:    StateDisconnected,
		lastPing: time.Now(),
	}
}

// Start starts the client and connects to the master
func (c *Client) Start(ctx context.Context) error {
	// Resolve master address
	masterAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.config.MasterIP, c.config.MasterPort))
	if err != nil {
		return fmt.Errorf("failed to resolve master address: %w", err)
	}
	c.masterAddr = masterAddr

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

	c.log.Info("Client started",
		logger.String("master", c.masterAddr.String()),
		logger.String("local", conn.LocalAddr().String()))

	// Start authentication
	if err := c.authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Start goroutines for receiving and keepalive
	errChan := make(chan error, 2)

	go func() {
		errChan <- c.receiveLoop(ctx)
	}()

	go func() {
		errChan <- c.keepaliveLoop(ctx)
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// authenticate performs the authentication handshake with the master
func (c *Client) authenticate() error {
	// Step 1: Send RPTL (login request)
	c.log.Info("Sending RPTL (login request)", logger.Int("radio_id", int(c.config.RadioID)))

	rptl := &protocol.RPTLPacket{
		RepeaterID: uint32(c.config.RadioID),
	}
	data, err := rptl.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode RPTL: %w", err)
	}

	_, err = c.conn.WriteToUDP(data, c.masterAddr)
	if err != nil {
		return fmt.Errorf("failed to send RPTL: %w", err)
	}

	c.setState(StateRPTLSent)

	// Wait for RPTACK
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 1024)
	n, _, err := c.conn.ReadFromUDP(buffer)
	if err != nil {
		return fmt.Errorf("failed to receive RPTACK: %w", err)
	}

	// Parse RPTACK
	if n >= protocol.RPTACKPacketSize && string(buffer[0:6]) == protocol.PacketTypeRPTACK {
		c.log.Info("Received RPTACK")
		c.setState(StateAuthenticated)
	} else {
		return fmt.Errorf("unexpected response to RPTL: %s", string(buffer[0:n]))
	}

	// Step 2: Send RPTK (key exchange)
	c.log.Info("Sending RPTK (key exchange)")

	// Generate salt for challenge
	c.salt = make([]byte, protocol.SaltLength)
	for i := range c.salt {
		c.salt[i] = byte(time.Now().UnixNano() % 256)
	}

	// Create challenge: SHA256(salt + passphrase)
	h := sha256.New()
	h.Write(c.salt)
	h.Write([]byte(c.config.Passphrase))
	challenge := h.Sum(nil)

	rptk := &protocol.RPTKPacket{
		RepeaterID: uint32(c.config.RadioID),
		Challenge:  challenge,
	}
	data, err = rptk.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode RPTK: %w", err)
	}

	_, err = c.conn.WriteToUDP(data, c.masterAddr)
	if err != nil {
		return fmt.Errorf("failed to send RPTK: %w", err)
	}

	// Wait for RPTACK
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, _, err = c.conn.ReadFromUDP(buffer)
	if err != nil {
		return fmt.Errorf("failed to receive RPTACK after RPTK: %w", err)
	}

	if n >= protocol.RPTACKPacketSize && string(buffer[0:6]) == protocol.PacketTypeRPTACK {
		c.log.Info("Received RPTACK after RPTK")
	} else {
		return fmt.Errorf("unexpected response to RPTK")
	}

	// Step 3: Send RPTC (configuration)
	c.log.Info("Sending RPTC (configuration)")

	rptc := &protocol.RPTCPacket{
		RepeaterID:  uint32(c.config.RadioID),
		Callsign:    c.config.Callsign,
		RXFreq:      fmt.Sprintf("%d", c.config.RXFreq),
		TXFreq:      fmt.Sprintf("%d", c.config.TXFreq),
		TXPower:     fmt.Sprintf("%d", c.config.TXPower),
		ColorCode:   fmt.Sprintf("%d", c.config.ColorCode),
		Latitude:    fmt.Sprintf("%.4f", c.config.Latitude),
		Longitude:   fmt.Sprintf("%.4f", c.config.Longitude),
		Height:      fmt.Sprintf("%d", c.config.Height),
		Location:    c.config.Location,
		Description: c.config.Description,
		URL:         c.config.URL,
		SoftwareID:  c.config.SoftwareID,
		PackageID:   c.config.PackageID,
	}
	data, err = rptc.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode RPTC: %w", err)
	}

	_, err = c.conn.WriteToUDP(data, c.masterAddr)
	if err != nil {
		return fmt.Errorf("failed to send RPTC: %w", err)
	}

	c.setState(StateConfigSent)

	// Wait for RPTACK
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, _, err = c.conn.ReadFromUDP(buffer)
	if err != nil {
		return fmt.Errorf("failed to receive RPTACK after RPTC: %w", err)
	}

	if n >= protocol.RPTACKPacketSize && string(buffer[0:6]) == protocol.PacketTypeRPTACK {
		c.log.Info("Received RPTACK after RPTC - Authentication complete")
		c.setState(StateConnected)
	} else {
		return fmt.Errorf("unexpected response to RPTC")
	}

	// Clear read deadline for normal operation
	c.conn.SetReadDeadline(time.Time{})

	return nil
}

// receiveLoop continuously receives and processes packets
func (c *Client) receiveLoop(ctx context.Context) error {
	buffer := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Set read deadline to allow context checking
		c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, _, err := c.conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return fmt.Errorf("read error: %w", err)
		}

		// Process received packet
		c.handlePacket(buffer[:n])
	}
}

// handlePacket processes a received packet
func (c *Client) handlePacket(data []byte) {
	if len(data) < 4 {
		return
	}

	// Check packet type
	switch {
	case len(data) >= protocol.DMRDPacketSize && string(data[0:4]) == protocol.PacketTypeDMRD:
		// DMRD packet
		packet := &protocol.DMRDPacket{}
		if err := packet.Parse(data); err != nil {
			c.log.Error("Failed to parse DMRD packet", logger.Error(err))
			return
		}

		c.log.Debug("Received DMRD packet",
			logger.Int("src", int(packet.SourceID)),
			logger.Int("dst", int(packet.DestinationID)),
			logger.Int("ts", packet.Timeslot))

		// Call DMRD handler
		c.handlerMu.RLock()
		handler := c.dmrdHandler
		c.handlerMu.RUnlock()

		if handler != nil {
			handler(packet)
		}

	case len(data) >= protocol.MSTPONGPacketSize && string(data[0:7]) == protocol.PacketTypeMSTPONG:
		// MSTPONG packet
		c.log.Debug("Received MSTPONG")
		c.updateLastPing()

	case len(data) >= protocol.MSTCLPacketSize && string(data[0:5]) == protocol.PacketTypeMSTCL:
		// MSTCL - master closing connection
		c.log.Warn("Received MSTCL - master closing connection")
		c.setState(StateDisconnected)

	default:
		c.log.Debug("Received unknown packet type", logger.String("type", string(data[0:4])))
	}
}

// keepaliveLoop sends periodic RPTPING packets
func (c *Client) keepaliveLoop(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if c.getState() != StateConnected {
				continue
			}

			// Send RPTPING
			ping := &protocol.RPTPINGPacket{
				RepeaterID: uint32(c.config.RadioID),
			}
			data, err := ping.Encode()
			if err != nil {
				c.log.Error("Failed to encode RPTPING", logger.Error(err))
				continue
			}

			_, err = c.conn.WriteToUDP(data, c.masterAddr)
			if err != nil {
				c.log.Error("Failed to send RPTPING", logger.Error(err))
				continue
			}

			c.log.Debug("Sent RPTPING")
		}
	}
}

// SendDMRD sends a DMRD packet to the master
func (c *Client) SendDMRD(packet *protocol.DMRDPacket) error {
	if c.getState() != StateConnected {
		return fmt.Errorf("not connected to master")
	}

	data, err := packet.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode DMRD: %w", err)
	}

	_, err = c.conn.WriteToUDP(data, c.masterAddr)
	if err != nil {
		return fmt.Errorf("failed to send DMRD: %w", err)
	}

	c.log.Debug("Sent DMRD packet",
		logger.Int("src", int(packet.SourceID)),
		logger.Int("dst", int(packet.DestinationID)),
		logger.Int("ts", packet.Timeslot))

	return nil
}

// OnDMRD sets the handler for received DMRD packets
func (c *Client) OnDMRD(handler func(*protocol.DMRDPacket)) {
	c.handlerMu.Lock()
	c.dmrdHandler = handler
	c.handlerMu.Unlock()
}

// GetSalt returns the authentication salt (for testing)
func (c *Client) GetSalt() string {
	if c.salt == nil {
		return ""
	}
	return hex.EncodeToString(c.salt)
}

// Helper methods for state management
func (c *Client) setState(state ConnectionState) {
	c.stateMu.Lock()
	c.state = state
	c.stateMu.Unlock()
}

func (c *Client) getState() ConnectionState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

func (c *Client) updateLastPing() {
	c.lastPingMu.Lock()
	c.lastPing = time.Now()
	c.lastPingMu.Unlock()
}
