package ysf

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/logger"
)

// YSFNetwork handles YSF network communication
// Based on YSFNetwork.cpp from MMDVM_CM (client mode only)
type YSFNetwork struct {
	callsign      string
	serverAddr    string
	serverPort    int
	conn          *net.UDPConn
	serverUDPAddr *net.UDPAddr
	logger        *logger.Logger
	debug         bool

	// Polling
	pollInterval time.Duration
	lastPoll     time.Time

	// Buffers
	rxBuffer chan []byte
	txMutex  sync.Mutex

	// Poll/Unlink messages
	pollMsg   []byte
	unlinkMsg []byte

	// Running state
	running bool
	mu      sync.RWMutex
}

const (
	// DefaultPollInterval is the default time between poll messages
	DefaultPollInterval = 5 * time.Second

	// RxBufferSize is the size of the receive buffer channel
	RxBufferSize = 100
)

// NetworkConfig holds YSF network configuration
type NetworkConfig struct {
	Callsign     string
	ServerAddr   string
	ServerPort   int
	PollInterval time.Duration
	Debug        bool
}

// NewYSFNetwork creates a new YSF network client
func NewYSFNetwork(cfg NetworkConfig, log *logger.Logger) *YSFNetwork {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = DefaultPollInterval
	}

	// Pad callsign to 10 characters
	callsign := padCallsign(cfg.Callsign)

	// Create poll message: "YSFP" + callsign (14 bytes total)
	pollMsg := make([]byte, 14)
	copy(pollMsg[0:4], []byte("YSFP"))
	copy(pollMsg[4:14], []byte(callsign))

	// Create unlink message: "YSFU" + callsign (14 bytes total)
	unlinkMsg := make([]byte, 14)
	copy(unlinkMsg[0:4], []byte("YSFU"))
	copy(unlinkMsg[4:14], []byte(callsign))

	return &YSFNetwork{
		callsign:     callsign,
		serverAddr:   cfg.ServerAddr,
		serverPort:   cfg.ServerPort,
		logger:       log,
		debug:        cfg.Debug,
		pollInterval: cfg.PollInterval,
		rxBuffer:     make(chan []byte, RxBufferSize),
		pollMsg:      pollMsg,
		unlinkMsg:    unlinkMsg,
	}
}

// Open opens the YSF network connection
func (n *YSFNetwork) Open() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		return fmt.Errorf("YSF network already open")
	}

	// Resolve server address
	serverAddr := fmt.Sprintf("%s:%d", n.serverAddr, n.serverPort)
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve server address: %w", err)
	}
	n.serverUDPAddr = udpAddr

	// Create UDP connection (bind to any local port)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return fmt.Errorf("failed to create UDP socket: %w", err)
	}
	n.conn = conn

	n.running = true
	n.logger.Info("YSF network opened",
		logger.String("server", serverAddr),
		logger.String("callsign", TrimCallsign(n.callsign)))

	return nil
}

// Start starts the YSF network client
func (n *YSFNetwork) Start(ctx context.Context) error {
	if err := n.Open(); err != nil {
		return err
	}

	// Start receive goroutine
	go n.receiveLoop(ctx)

	// Start poll goroutine
	go n.pollLoop(ctx)

	return nil
}

// receiveLoop handles incoming UDP packets
func (n *YSFNetwork) receiveLoop(ctx context.Context) {
	buffer := make([]byte, 200) // Large enough for YSF frames

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Set read deadline to allow context checking
		if err := n.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
			n.logger.Error("Failed to set read deadline", logger.Error(err))
			continue
		}

		length, remoteAddr, err := n.conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Normal timeout, check context
			}
			n.logger.Error("Error reading from UDP socket", logger.Error(err))
			continue
		}

		// Verify packet is from our server (relaxed check - same IP, any port)
		if remoteAddr.IP.String() != n.serverUDPAddr.IP.String() {
			if n.debug {
				n.logger.Debug("Received packet from unexpected IP address",
					logger.String("addr", remoteAddr.String()),
					logger.String("expected", n.serverUDPAddr.String()))
			}
			continue
		}

		// Make a copy of the received data
		data := make([]byte, length)
		copy(data, buffer[:length])

		if n.debug {
			n.logger.Debug("YSF packet received",
				logger.Int("length", length),
				logger.String("from", remoteAddr.String()))
		}

		// Send to buffer
		select {
		case n.rxBuffer <- data:
		default:
			n.logger.Warn("RX buffer full, dropping packet")
		}
	}
}

// pollLoop sends periodic poll messages
func (n *YSFNetwork) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(n.pollInterval)
	defer ticker.Stop()

	// Send initial poll immediately
	if err := n.WritePoll(); err != nil {
		n.logger.Error("Failed to send initial poll", logger.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			// Send unlink before closing
			if err := n.WriteUnlink(); err != nil {
				n.logger.Error("Failed to send unlink", logger.Error(err))
			}
			return
		case <-ticker.C:
			if err := n.WritePoll(); err != nil {
				n.logger.Error("Failed to send poll", logger.Error(err))
			}
		}
	}
}

// Read reads a YSF frame from the receive buffer
// Returns nil if no data available
func (n *YSFNetwork) Read() []byte {
	select {
	case data := <-n.rxBuffer:
		return data
	default:
		return nil
	}
}

// Write writes a YSF frame to the server
func (n *YSFNetwork) Write(data []byte) error {
	n.txMutex.Lock()
	defer n.txMutex.Unlock()

	if !n.running {
		return fmt.Errorf("YSF network not open")
	}

	if len(data) != YSFFrameLength {
		return fmt.Errorf("invalid YSF frame length: %d (expected %d)", len(data), YSFFrameLength)
	}

	if n.debug {
		n.logger.Debug("YSF frame sent", logger.Int("length", len(data)))
	}

	_, err := n.conn.WriteToUDP(data, n.serverUDPAddr)
	if err != nil {
		return fmt.Errorf("failed to write to UDP socket: %w", err)
	}

	return nil
}

// WritePoll sends a poll message to the server
func (n *YSFNetwork) WritePoll() error {
	n.txMutex.Lock()
	defer n.txMutex.Unlock()

	if !n.running {
		return fmt.Errorf("YSF network not open")
	}

	if n.debug {
		n.logger.Debug("Sending poll message")
	}

	_, err := n.conn.WriteToUDP(n.pollMsg, n.serverUDPAddr)
	if err != nil {
		return fmt.Errorf("failed to send poll: %w", err)
	}

	n.lastPoll = time.Now()
	return nil
}

// WriteUnlink sends an unlink message to the server
func (n *YSFNetwork) WriteUnlink() error {
	n.txMutex.Lock()
	defer n.txMutex.Unlock()

	if !n.running {
		return fmt.Errorf("YSF network not open")
	}

	n.logger.Info("Sending unlink message")

	_, err := n.conn.WriteToUDP(n.unlinkMsg, n.serverUDPAddr)
	if err != nil {
		return fmt.Errorf("failed to send unlink: %w", err)
	}

	return nil
}

// Close closes the YSF network connection
func (n *YSFNetwork) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.running {
		return nil
	}

	n.running = false

	if n.conn != nil {
		if err := n.conn.Close(); err != nil {
			return fmt.Errorf("failed to close UDP socket: %w", err)
		}
	}

	n.logger.Info("YSF network closed")
	return nil
}

// GetCallsign returns the configured callsign
func (n *YSFNetwork) GetCallsign() string {
	return n.callsign
}
