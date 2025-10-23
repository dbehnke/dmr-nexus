package network

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/bridge"
	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/peer"
	"github.com/dbehnke/dmr-nexus/pkg/protocol"
)

// Server represents a UDP server for MASTER mode
type Server struct {
	config          config.SystemConfig
	systemName      string // Name of this system (from config key)
	log             *logger.Logger
	conn            *net.UDPConn
	peerManager     *peer.PeerManager
	router          *bridge.Router
	pingTimeout     time.Duration
	cleanupInterval time.Duration
	regACL          *peer.ACL
	subACL          *peer.ACL
	tg1ACL          *peer.ACL
	tg2ACL          *peer.ACL
	// started is closed once the UDP listener is bound and ready
	started chan struct{}

	// Optional hooks for events
	onPeerConnected    func(id uint32, callsign string, addr string)
	onPeerDisconnected func(id uint32)

	// Mute map: streamID -> expiry of mute (2s idle or until terminator)
	mutedStreams   map[uint32]time.Time
	mutedStreamsMu sync.Mutex

	// Subscriber location tracking for private calls: radioID -> subscriberLocation
	subscriberLocations   map[uint32]*subscriberLocation
	subscriberLocationsMu sync.RWMutex
}

// subscriberLocation tracks where a subscriber (radio) was last seen
type subscriberLocation struct {
	peerID   uint32    // Which peer the subscriber is behind
	lastSeen time.Time // When we last saw traffic from this subscriber
}

// CleanupMutedStreamsOnce runs a single cleanup pass for mutedStreams (for testing)
func (s *Server) CleanupMutedStreamsOnce(now time.Time) {
	s.mutedStreamsMu.Lock()
	defer s.mutedStreamsMu.Unlock()
	for streamID, expiry := range s.mutedStreams {
		if now.After(expiry) {
			delete(s.mutedStreams, streamID)
		}
	}
}

// NewServer creates a new UDP server for MASTER mode
func NewServer(cfg config.SystemConfig, systemName string, log *logger.Logger) *Server {
	return &Server{
		config:              cfg,
		systemName:          systemName,
		log:                 log.WithComponent("network.server"),
		peerManager:         peer.NewPeerManager(),
		pingTimeout:         30 * time.Second, // Default timeout
		cleanupInterval:     10 * time.Second, // Default cleanup interval
		started:             make(chan struct{}),
		mutedStreams:        make(map[uint32]time.Time),
		subscriberLocations: make(map[uint32]*subscriberLocation),
	}
}

// WithPeerManager injects a shared peer manager (instead of using the internal one)
func (s *Server) WithPeerManager(pm *peer.PeerManager) *Server {
	s.peerManager = pm
	return s
}

// WithRouter injects a bridge router for routing packets between systems
func (s *Server) WithRouter(r *bridge.Router) *Server {
	s.router = r
	return s
}

// SetPeerEventHandlers sets optional callbacks for peer events
func (s *Server) SetPeerEventHandlers(onConnect func(id uint32, callsign string, addr string), onDisconnect func(id uint32)) {
	s.onPeerConnected = onConnect
	s.onPeerDisconnected = onDisconnect
}

// Start starts the server and begins accepting connections
func (s *Server) Start(ctx context.Context) error {
	// Parse ACLs if enabled
	if s.config.UseACL {
		if s.config.RegACL != "" {
			acl, err := peer.ParseACL(s.config.RegACL)
			if err != nil {
				return fmt.Errorf("failed to parse REG_ACL: %w", err)
			}
			s.regACL = acl
		}

		if s.config.SubACL != "" {
			acl, err := peer.ParseACL(s.config.SubACL)
			if err != nil {
				return fmt.Errorf("failed to parse SUB_ACL: %w", err)
			}
			s.subACL = acl
		}

		if s.config.TG1ACL != "" {
			acl, err := peer.ParseACL(s.config.TG1ACL)
			if err != nil {
				return fmt.Errorf("failed to parse TG1_ACL: %w", err)
			}
			s.tg1ACL = acl
		}

		if s.config.TG2ACL != "" {
			acl, err := peer.ParseACL(s.config.TG2ACL)
			if err != nil {
				return fmt.Errorf("failed to parse TG2_ACL: %w", err)
			}
			s.tg2ACL = acl
		}
	}

	// Create local UDP address
	localAddr := &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: s.config.Port,
	}

	// Create UDP connection
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}
	s.conn = conn
	// Signal that the server is ready to accept packets
	select {
	case <-s.started: // already closed
	default:
		close(s.started)
	}
	defer func() {
		_ = s.conn.Close()
	}()

	s.log.Info("Server started",
		logger.String("addr", conn.LocalAddr().String()),
		logger.Int("max_peers", s.config.MaxPeers))

	// Start goroutines for receiving and cleanup
	errChan := make(chan error, 2)

	go func() {
		errChan <- s.receiveLoop(ctx)
	}()

	go func() {
		errChan <- s.cleanupLoop(ctx)
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// WaitStarted blocks until the server UDP listener is bound or the context is canceled.
func (s *Server) WaitStarted(ctx context.Context) error {
	select {
	case <-s.started:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Addr returns the local UDP address the server is bound to. It should be called after WaitStarted.
func (s *Server) Addr() (*net.UDPAddr, error) {
	if s.conn == nil {
		return nil, fmt.Errorf("server not started")
	}
	addr := s.conn.LocalAddr()
	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("not a UDP address")
	}
	return udpAddr, nil
}

// receiveLoop continuously receives and processes packets
func (s *Server) receiveLoop(ctx context.Context) error {
	buffer := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Set read deadline to allow context checking
		if err := s.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
			s.log.Warn("Failed to set read deadline", logger.Error(err))
			continue
		}
		n, addr, err := s.conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			s.log.Error("Failed to read from UDP", logger.Error(err))
			continue
		}

		// Process packet
		go s.handlePacket(buffer[:n], addr)
	}
}

// handlePacket processes a received packet
func (s *Server) handlePacket(data []byte, addr *net.UDPAddr) {
	if len(data) == 0 {
		// Empty UDP packets can happen (spurious wake-ups, etc.) - ignore silently
		return
	}
	if len(data) < 4 {
		s.log.Debug("Packet too small", logger.Int("size", len(data)))
		return
	}

	// Get packet type - HomeBrew protocol has variable length packet type identifiers
	// Try to match from longest to shortest: 7 chars, 6 chars, 5 chars, 4 chars
	var packetType string
	if len(data) >= 7 {
		check7 := string(data[0:7])
		if check7 == protocol.PacketTypeRPTPING || check7 == protocol.PacketTypeMSTPONG {
			packetType = check7
		}
	}
	if packetType == "" && len(data) >= 6 {
		check6 := string(data[0:6])
		if check6 == protocol.PacketTypeRPTACK {
			packetType = check6
		}
	}
	if packetType == "" && len(data) >= 5 {
		check5 := string(data[0:5])
		if check5 == protocol.PacketTypeMSTCL || check5 == protocol.PacketTypeRPTCL {
			packetType = check5
		}
	}
	if packetType == "" {
		// Default to 4-char packet types (DMRD, RPTL, RPTK, RPTC)
		packetType = string(data[0:4])
	}

	// For debugging, get raw header
	headerLen := 7
	if len(data) < 7 {
		headerLen = len(data)
	}

	s.log.Debug("Received packet",
		logger.String("type", packetType),
		logger.String("addr", addr.String()),
		logger.Int("size", len(data)),
		logger.String("raw_header", string(data[0:headerLen])))

	switch packetType {
	case protocol.PacketTypeDMRD:
		s.handleDMRD(data, addr)
	case protocol.PacketTypeRPTL:
		s.handleRPTL(data, addr)
	case protocol.PacketTypeRPTK:
		s.handleRPTK(data, addr)
	case protocol.PacketTypeRPTC:
		s.handleRPTC(data, addr)
	case protocol.PacketTypeRPTO:
		s.handleRPTO(data, addr)
	case protocol.PacketTypeRPTCL:
		s.handleRPTCL(data, addr)
	case protocol.PacketTypeRPTPING:
		s.handleRPTPING(data, addr)
	case protocol.PacketTypeMSTCL:
		s.handleMSTCL(data, addr)
	default:
		s.log.Debug("Unknown packet type",
			logger.String("type", packetType),
			logger.String("addr", addr.String()))
	}
}

// handleRPTL handles login requests from peers
func (s *Server) handleRPTL(data []byte, addr *net.UDPAddr) {
	rptl, err := protocol.ParseRPTL(data)
	if err != nil {
		s.log.Error("Failed to parse RPTL", logger.Error(err))
		return
	}

	s.log.Info("Received RPTL",
		logger.Int("peer_id", int(rptl.RepeaterID)),
		logger.String("addr", addr.String()))

	// Check REG_ACL
	if s.config.UseACL && s.regACL != nil {
		if !s.regACL.Check(rptl.RepeaterID) {
			s.log.Warn("Peer denied by REG_ACL",
				logger.Int("peer_id", int(rptl.RepeaterID)))
			s.sendMSTCL(rptl.RepeaterID, addr)
			return
		}
	}

	// Add or update peer
	p := s.peerManager.AddPeer(rptl.RepeaterID, addr)
	p.SetState(peer.StateRPTLReceived)
	p.UpdateLastHeard()

	// Send RPTACK
	s.sendRPTACK(rptl.RepeaterID, addr)
}

// handleRPTK handles key exchange from peers
func (s *Server) handleRPTK(data []byte, addr *net.UDPAddr) {
	rptk, err := protocol.ParseRPTK(data)
	if err != nil {
		s.log.Error("Failed to parse RPTK", logger.Error(err))
		return
	}

	s.log.Info("Received RPTK",
		logger.Int("peer_id", int(rptk.RepeaterID)),
		logger.String("addr", addr.String()))

	// Get peer
	p := s.peerManager.GetPeer(rptk.RepeaterID)
	if p == nil {
		s.log.Warn("RPTK from unknown peer", logger.Int("peer_id", int(rptk.RepeaterID)))
		return
	}

	// Store challenge for verification (in real implementation, verify it)
	p.Salt = rptk.Challenge
	p.SetState(peer.StateAuthenticated)
	p.UpdateLastHeard()

	// Send RPTACK
	s.sendRPTACK(rptk.RepeaterID, addr)
}

// handleRPTC handles configuration from peers
func (s *Server) handleRPTC(data []byte, addr *net.UDPAddr) {
	rptc, err := protocol.ParseRPTC(data)
	if err != nil {
		s.log.Error("Failed to parse RPTC", logger.Error(err))
		return
	}

	s.log.Info("Received RPTC",
		logger.Int("peer_id", int(rptc.RepeaterID)),
		logger.String("callsign", rptc.Callsign),
		logger.String("location", rptc.Location))

	// Get peer
	p := s.peerManager.GetPeer(rptc.RepeaterID)
	if p == nil {
		s.log.Warn("RPTC from unknown peer", logger.Int("peer_id", int(rptc.RepeaterID)))
		return
	}

	// Update peer configuration
	p.SetConfig(rptc)
	p.SetConnected()
	p.UpdateLastHeard()

	s.log.Info("Peer connected",
		logger.Int("peer_id", int(rptc.RepeaterID)),
		logger.String("callsign", rptc.Callsign))

	// Hook: peer connected
	if s.onPeerConnected != nil {
		s.onPeerConnected(rptc.RepeaterID, rptc.Callsign, addr.String())
	}

	// Send RPTACK
	// The client enters DMR_CONF state and expects RPTACK to trigger setup_connection()
	s.sendRPTACK(rptc.RepeaterID, addr)
}

// handleRPTO handles OPTIONS packets from peers
func (s *Server) handleRPTO(data []byte, addr *net.UDPAddr) {
	// RPTO packet format: "RPTO" + 4 byte repeater ID + OPTIONS string
	if len(data) < 8 {
		s.log.Debug("RPTO packet too small", logger.Int("size", len(data)))
		return
	}

	// Extract repeater ID (bytes 4-8)
	peerID := binary.BigEndian.Uint32(data[4:8])

	// Get peer
	p := s.peerManager.GetPeer(peerID)
	if p == nil {
		s.log.Warn("RPTO from unknown peer", logger.Int("peer_id", int(peerID)))
		return
	}

	// Extract OPTIONS string (everything after the 8-byte header)
	var optionsStr string
	if len(data) > 8 {
		optionsStr = string(data[8:])
	}

	s.log.Info("Received RPTO",
		logger.Int("peer_id", int(peerID)),
		logger.String("options", optionsStr))

	// Parse and update peer subscriptions if OPTIONS provided
	if optionsStr != "" {
		if opts, err := peer.ParseOptions(optionsStr); err == nil {
			if p.Subscriptions != nil {
				if err := p.Subscriptions.Update(opts); err != nil {
					s.log.Warn("Failed to update peer subscriptions",
						logger.Int("peer_id", int(peerID)),
						logger.Error(err))
				} else {
					s.log.Debug("Updated peer subscriptions from RPTO",
						logger.Int("peer_id", int(peerID)),
						logger.Int("ts1_count", len(opts.TS1)),
						logger.Int("ts2_count", len(opts.TS2)))
				}
			}
		} else {
			s.log.Warn("Failed to parse OPTIONS",
				logger.Int("peer_id", int(peerID)),
				logger.String("options", optionsStr),
				logger.Error(err))
		}
	}

	// Update last heard
	p.UpdateLastHeard()

	// Send RPTACK to acknowledge OPTIONS
	s.sendRPTACK(peerID, addr)
}

// handleRPTPING handles keepalive pings from peers
func (s *Server) handleRPTPING(data []byte, addr *net.UDPAddr) {
	if len(data) < protocol.RPTPINGPacketSize {
		return
	}

	// Extract repeater ID from ping
	peerID := binary.BigEndian.Uint32(data[7:11])

	// Get peer
	p := s.peerManager.GetPeer(peerID)
	if p == nil {
		// Unknown peer - send MSTCL to force disconnect
		// This can happen if server restarts or client crashes
		s.log.Debug("Received RPTPING from unknown peer, sending MSTCL",
			logger.Uint64("peer_id", uint64(peerID)),
			logger.String("addr", addr.String()))
		s.sendMSTCL(peerID, addr)
		return
	}

	s.log.Debug("Received RPTPING",
		logger.Uint64("peer_id", uint64(peerID)),
		logger.String("addr", addr.String()))

	// Update last heard
	p.UpdateLastHeard()

	// Send MSTPONG response
	s.sendMSTPONG(peerID, addr)
}

// handleRPTCL handles disconnect requests from peers (peer-initiated)
func (s *Server) handleRPTCL(data []byte, addr *net.UDPAddr) {
	if len(data) < protocol.RPTCLPacketSize {
		return
	}

	// Extract repeater ID (bytes 5-9 after "RPTCL")
	peerID := binary.BigEndian.Uint32(data[5:9])

	s.log.Info("Peer disconnect (RPTCL)",
		logger.Uint64("peer_id", uint64(peerID)),
		logger.String("addr", addr.String()))

	// Clear subscriber locations for this peer
	s.clearSubscriberLocationsForPeer(peerID)

	// Remove peer
	s.peerManager.RemovePeer(peerID)

	// Hook: peer disconnected
	if s.onPeerDisconnected != nil {
		s.onPeerDisconnected(peerID)
	}
}

// handleMSTCL handles disconnect requests from peers
func (s *Server) handleMSTCL(data []byte, addr *net.UDPAddr) {
	if len(data) < protocol.MSTCLPacketSize {
		return
	}

	// Extract repeater ID
	peerID := binary.BigEndian.Uint32(data[5:9])

	s.log.Info("Peer disconnect",
		logger.Int("peer_id", int(peerID)),
		logger.String("addr", addr.String()))

	// Clear subscriber locations for this peer
	s.clearSubscriberLocationsForPeer(peerID)

	// Remove peer
	s.peerManager.RemovePeer(peerID)

	// Hook: peer disconnected
	if s.onPeerDisconnected != nil {
		s.onPeerDisconnected(peerID)
	}
}

// handleDMRD handles DMR data packets
func (s *Server) handleDMRD(data []byte, addr *net.UDPAddr) {
	dmrd, err := protocol.ParseDMRD(data)
	if err != nil {
		s.log.Debug("Failed to parse DMRD", logger.Error(err))
		return
	}

	// Get peer
	p := s.peerManager.GetPeerByAddress(addr)
	if p == nil {
		s.log.Debug("DMRD from unknown peer", logger.String("addr", addr.String()))
		return
	}

	// Update stats
	p.UpdateLastHeard()
	p.IncrementPacketsReceived()
	p.AddBytesReceived(uint64(len(data)))

	// Check SUB_ACL
	if s.config.UseACL && s.subACL != nil {
		if !s.subACL.Check(dmrd.SourceID) {
			s.log.Debug("Transmission denied by SUB_ACL",
				logger.Int("src_id", int(dmrd.SourceID)))
			return
		}
	}

	// Track subscriber location for private call routing
	// Always update location on every DMRD packet to keep it fresh
	s.trackSubscriberLocation(dmrd.SourceID, p.ID)

	// Handle private calls if enabled
	if s.config.PrivateCallsEnabled && dmrd.CallType == protocol.CallTypePrivate {
		s.handlePrivateCall(dmrd, data, p)
		return
	}

	// Check TG ACL based on timeslot (for group calls only)
	timeslot := dmrd.Timeslot
	if s.config.UseACL {
		if timeslot == 1 && s.tg1ACL != nil {
			if !s.tg1ACL.Check(dmrd.DestinationID) {
				s.log.Debug("Talkgroup denied by TG1_ACL",
					logger.Int("tg", int(dmrd.DestinationID)))
				return
			}
		} else if timeslot == 2 && s.tg2ACL != nil {
			if !s.tg2ACL.Check(dmrd.DestinationID) {
				s.log.Debug("Talkgroup denied by TG2_ACL",
					logger.Int("tg", int(dmrd.DestinationID)))
				return
			}
		}
	}

	// Process bridge activation/deactivation if router is configured
	if s.router != nil {
		// Special handling for TG 777 - enable "repeat everything" mode
		if dmrd.DestinationID == 777 {
			p.SetRepeatMode(true)

			s.log.Info("Peer enabled repeat-all mode",
				logger.Int("peer_id", int(p.ID)),
				logger.String("callsign", p.Callsign))

			// Don't process this as a normal talkgroup
			return
		}

		// Special handling for TG 4000 - disconnect from all dynamic subscriptions AND disable repeat mode
		if dmrd.DestinationID == 4000 {
			// Disable repeat mode
			p.SetRepeatMode(false)

			// Remove from all dynamic bridges
			bridgeCount := s.router.RemoveSubscriberFromAllDynamicBridges(p.ID)

			// Clear all dynamic subscriptions from peer
			var subCount int
			if p.Subscriptions != nil {
				subCount = p.Subscriptions.ClearAllDynamic()
			}

			s.log.Info("Peer disconnected from all dynamic talkgroups and disabled repeat mode",
				logger.Int("peer_id", int(p.ID)),
				logger.String("callsign", p.Callsign),
				logger.Int("dynamic_bridges", bridgeCount),
				logger.Int("dynamic_subscriptions", subCount))

			// Don't process this as a normal talkgroup
			return
		}

		// Touch the transmitting peer's subscription to this talkgroup
		// AddDynamic returns true if this is a NEW subscription (first key-up)
		// First key-up subscribes but doesn't forward audio (subscription activation)
		// Uses the peer's AutoTTL from OPTIONS, or unlimited if not set
		isNewSubscription := false
		if p.Subscriptions != nil {
			isNewSubscription = p.Subscriptions.AddDynamic(dmrd.DestinationID, uint8(dmrd.Timeslot))
		}

		// Create/update dynamic bridge for dashboard visibility
		// This doesn't affect forwarding logic - it's just for tracking/display
		// Bridges are now timeslot-agnostic
		s.router.GetOrCreateDynamicBridge(dmrd.DestinationID)

		// If this is the first key-up (new subscription), mark this stream muted
		if isNewSubscription {
			// Mute for the duration of this transmission: until voice terminator or 2s idle
			s.mutedStreams[dmrd.StreamID] = time.Now().Add(2 * time.Second)
			s.log.Info("Peer subscribed to talkgroup (first key-up muted for this transmission)",
				logger.Int("peer_id", int(p.ID)),
				logger.String("callsign", p.Callsign),
				logger.Int("tg", int(dmrd.DestinationID)),
				logger.Int("ts", dmrd.Timeslot),
				logger.Uint64("stream", uint64(dmrd.StreamID)))
			// Do not forward this frame
			return
		}

		s.log.Debug("Dynamic bridge activity",
			logger.Int("peer_id", int(p.ID)),
			logger.Int("tg", int(dmrd.DestinationID)),
			logger.Int("ts", dmrd.Timeslot),
			logger.Int("src", int(dmrd.SourceID)))

		// Check if this TGID should activate any static bridge rules
		activated := s.router.ProcessActivation(dmrd.DestinationID)
		if len(activated) > 0 {
			for bridgeName, rules := range activated {
				for _, rule := range rules {
					s.log.Info("Bridge rule activated",
						logger.String("bridge", bridgeName),
						logger.String("system", rule.System),
						logger.Int("tg", rule.TGID),
						logger.Int("ts", rule.Timeslot))
				}
			}
		}

		// Check if this TGID should deactivate any static bridge rules
		deactivated := s.router.ProcessDeactivation(dmrd.DestinationID)
		if len(deactivated) > 0 {
			for bridgeName, rules := range deactivated {
				for _, rule := range rules {
					s.log.Info("Bridge rule deactivated",
						logger.String("bridge", bridgeName),
						logger.String("system", rule.System),
						logger.Int("tg", rule.TGID),
						logger.Int("ts", rule.Timeslot))
				}
			}
		}

		// Update or clear stream mute based on frames
		if _, muted := s.mutedStreams[dmrd.StreamID]; muted {
			// Extend mute window with activity
			s.mutedStreams[dmrd.StreamID] = time.Now().Add(2 * time.Second)
			// If this is a terminator frame, unmute by deleting
			if dmrd.FrameType == protocol.FrameTypeVoiceTerminator {
				delete(s.mutedStreams, dmrd.StreamID)
			}
			// Suppress forwarding while muted
			return
		}

		// Route packet using bridge rules and dynamic bridges
		targets := s.router.RoutePacket(dmrd, s.systemName)

		// Forward to dynamically subscribed peers
		dynamicTargets := s.findDynamicSubscribers(dmrd.DestinationID, uint8(dmrd.Timeslot), p.ID)

		if len(targets) > 0 || len(dynamicTargets) > 0 {
			s.log.Debug("Routing DMRD packet",
				logger.Int("src", int(dmrd.SourceID)),
				logger.Int("dst", int(dmrd.DestinationID)),
				logger.Int("ts", dmrd.Timeslot),
				logger.Int("static_targets", len(targets)),
				logger.Int("dynamic_targets", len(dynamicTargets)))
		}

		// Forward to dynamic subscribers
		if len(dynamicTargets) > 0 {
			s.forwardToDynamicSubscribers(dmrd, data, dynamicTargets)
		}
	}

	// Forward to other peers if repeat is enabled
	if s.config.Repeat {
		s.forwardDMRD(dmrd, data, p.ID)
	}
}

// findDynamicSubscribers finds all peers that are subscribed to a talkgroup on ANY timeslot
// (timeslot-agnostic for dynamic bridges) or have repeat mode enabled, excluding the source peer
func (s *Server) findDynamicSubscribers(tgid uint32, timeslot uint8, sourcePeerID uint32) []*peer.Peer {
	allPeers := s.peerManager.GetAllPeers()
	subscribers := make([]*peer.Peer, 0)

	s.log.Debug("Finding dynamic subscribers (timeslot-agnostic)",
		logger.Int("tg", int(tgid)),
		logger.Int("source_ts", int(timeslot)),
		logger.Int("source_peer", int(sourcePeerID)),
		logger.Int("total_peers", len(allPeers)))

	for _, p := range allPeers {
		// Skip source peer
		if p.ID == sourcePeerID {
			s.log.Debug("Skipping source peer", logger.Int("peer_id", int(p.ID)))
			continue
		}

		// Only consider connected peers
		if p.GetState() != peer.StateConnected {
			s.log.Debug("Skipping non-connected peer",
				logger.Int("peer_id", int(p.ID)),
				logger.String("state", p.GetState().String()))
			continue
		}

		// Check if peer has repeat mode enabled (receives all traffic)
		if p.GetRepeatMode() {
			s.log.Debug("Adding peer in repeat mode",
				logger.Int("peer_id", int(p.ID)))
			subscribers = append(subscribers, p)
			continue
		}

		// Check if peer is subscribed to this talkgroup on ANY timeslot (timeslot-agnostic)
		if p.Subscriptions != nil {
			isSubscribed := p.Subscriptions.IsSubscribedToTalkgroup(tgid)
			s.log.Debug("Checking peer subscription (any timeslot)",
				logger.Int("peer_id", int(p.ID)),
				logger.Int("tg", int(tgid)),
				logger.Bool("is_subscribed", isSubscribed))
			if isSubscribed {
				subscribers = append(subscribers, p)
			}
		} else {
			s.log.Debug("Peer has no subscriptions", logger.Int("peer_id", int(p.ID)))
		}
	}

	s.log.Debug("Found subscribers",
		logger.Int("tg", int(tgid)),
		logger.Int("count", len(subscribers)))

	return subscribers
}

// handlePrivateCall handles routing of private (unit-to-unit) calls
func (s *Server) handlePrivateCall(dmrd *protocol.DMRDPacket, data []byte, sourcePeer *peer.Peer) {
	s.log.Debug("Handling private call",
		logger.Int("src", int(dmrd.SourceID)),
		logger.Int("dst", int(dmrd.DestinationID)),
		logger.Int("ts", dmrd.Timeslot),
		logger.Int("source_peer", int(sourcePeer.ID)))

	// Look up where the destination subscriber is located
	targetPeer, found := s.lookupSubscriberLocation(dmrd.DestinationID)

	if !found {
		// Destination not found or stale
		s.log.Debug("Private call destination not found",
			logger.Int("dst", int(dmrd.DestinationID)),
			logger.Int("src", int(dmrd.SourceID)))
		return
	}

	// Don't send back to the source peer
	if targetPeer.ID == sourcePeer.ID {
		s.log.Debug("Private call destination is on same peer as source, not forwarding",
			logger.Int("peer_id", int(targetPeer.ID)))
		return
	}

	s.log.Info("Routing private call",
		logger.Int("src", int(dmrd.SourceID)),
		logger.Int("dst", int(dmrd.DestinationID)),
		logger.Int("ts", dmrd.Timeslot),
		logger.Int("source_peer", int(sourcePeer.ID)),
		logger.Int("target_peer", int(targetPeer.ID)),
		logger.String("target_callsign", targetPeer.Callsign))

	// Forward the packet to the target peer
	_, err := s.conn.WriteToUDP(data, targetPeer.Address)
	if err != nil {
		s.log.Error("Failed to forward private call",
			logger.Int("target_peer", int(targetPeer.ID)),
			logger.Error(err))
		return
	}

	// Update stats
	targetPeer.IncrementPacketsSent()
	targetPeer.AddBytesSent(uint64(len(data)))
}

// countTalkgroupSubscribers counts how many peers are subscribed to a talkgroup (any timeslot)
func (s *Server) countTalkgroupSubscribers(tgid uint32) int {
	allPeers := s.peerManager.GetAllPeers()
	count := 0

	for _, p := range allPeers {
		if p.GetState() != peer.StateConnected {
			continue
		}

		if p.Subscriptions != nil && p.Subscriptions.IsSubscribedToTalkgroup(tgid) {
			count++
		}
	}

	return count
}

// forwardToDynamicSubscribers forwards a DMRD packet to dynamic subscribers
func (s *Server) forwardToDynamicSubscribers(_ *protocol.DMRDPacket, data []byte, targetPeers []*peer.Peer) {
	for _, targetPeer := range targetPeers {
		// Send packet
		_, err := s.conn.WriteToUDP(data, targetPeer.Address)
		if err != nil {
			s.log.Error("Failed to forward DMRD to dynamic subscriber",
				logger.Int("peer_id", int(targetPeer.ID)),
				logger.Error(err))
			continue
		}

		// Update stats
		targetPeer.IncrementPacketsSent()
		targetPeer.AddBytesSent(uint64(len(data)))
	}
}

// forwardDMRD forwards a DMRD packet to all other connected peers
func (s *Server) forwardDMRD(_ *protocol.DMRDPacket, data []byte, sourcePeerID uint32) {
	peers := s.peerManager.GetAllPeers()
	for _, p := range peers {
		// Don't send back to source
		if p.ID == sourcePeerID {
			continue
		}

		// Only send to fully connected peers
		if p.GetState() != peer.StateConnected {
			continue
		}

		// Send packet
		_, err := s.conn.WriteToUDP(data, p.Address)
		if err != nil {
			s.log.Error("Failed to forward DMRD",
				logger.Int("peer_id", int(p.ID)),
				logger.Error(err))
			continue
		}

		// Update stats
		p.IncrementPacketsSent()
		p.AddBytesSent(uint64(len(data)))
	}
}

// sendRPTACK sends an acknowledgement to a peer
func (s *Server) sendRPTACK(peerID uint32, addr *net.UDPAddr) {
	ack := &protocol.RPTACKPacket{
		RepeaterID: peerID,
	}
	data, err := ack.Encode()
	if err != nil {
		s.log.Error("Failed to encode RPTACK", logger.Error(err))
		return
	}

	_, err = s.conn.WriteToUDP(data, addr)
	if err != nil {
		s.log.Error("Failed to send RPTACK", logger.Error(err))
	}
}

// sendMSTPONG sends a pong response to a peer
func (s *Server) sendMSTPONG(peerID uint32, addr *net.UDPAddr) {
	pong := make([]byte, protocol.MSTPONGPacketSize)
	copy(pong[0:7], protocol.PacketTypeMSTPONG)
	binary.BigEndian.PutUint32(pong[7:11], peerID)

	_, err := s.conn.WriteToUDP(pong, addr)
	if err != nil {
		s.log.Debug("Failed to send MSTPONG", logger.Error(err))
	}
}

// sendMSTNAK sends a negative acknowledgement to an unknown peer
//
//nolint:unused
func (s *Server) sendMSTNAK(peerID uint32, addr *net.UDPAddr) {
	nak := make([]byte, protocol.MSTNAKPacketSize)
	copy(nak[0:6], protocol.PacketTypeMSTNAK)
	binary.BigEndian.PutUint32(nak[6:10], peerID)

	_, err := s.conn.WriteToUDP(nak, addr)
	if err != nil {
		s.log.Debug("Failed to send MSTNAK", logger.Error(err))
	}
}

// Reference the method to avoid unusedfunc diagnostics in editors/tools that
// warn about unused methods even when they are intentionally kept for future use.
var _ = (*Server).sendMSTNAK

// sendMSTCL sends a close/deny message to a peer
func (s *Server) sendMSTCL(peerID uint32, addr *net.UDPAddr) {
	cl := make([]byte, protocol.MSTCLPacketSize)
	copy(cl[0:5], protocol.PacketTypeMSTCL)
	binary.BigEndian.PutUint32(cl[5:9], peerID)

	_, err := s.conn.WriteToUDP(cl, addr)
	if err != nil {
		s.log.Debug("Failed to send MSTCL", logger.Error(err))
	}
}

// cleanupLoop periodically cleans up timed out peers
func (s *Server) cleanupLoop(ctx context.Context) error {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Cleanup timed out peers
			removed := s.peerManager.CleanupTimedOutPeers(s.pingTimeout)
			if removed > 0 {
				s.log.Info("Cleaned up timed out peers", logger.Int("count", removed))
			}

			// Cleanup inactive dynamic bridges (5 minutes of no subscribers)
			if s.router != nil {
				removedBridges := s.router.CleanupInactiveDynamicBridges(5*time.Minute, s.countTalkgroupSubscribers)
				if len(removedBridges) > 0 {
					s.log.Info("Cleaned up inactive dynamic bridges",
						logger.Int("count", len(removedBridges)))
				}
			}
			// Cleanup expired muted streams (idle > 2s)
			now := time.Now()
			for streamID, expiry := range s.mutedStreams {
				if now.After(expiry) {
					delete(s.mutedStreams, streamID)
				}
			}

			// Cleanup stale subscriber locations (not seen for 15 minutes)
			s.cleanupStaleSubscriberLocations(15 * time.Minute)
		}
	}
}

// trackSubscriberLocation records where a subscriber (radio) was last seen
func (s *Server) trackSubscriberLocation(radioID uint32, peerID uint32) {
	s.subscriberLocationsMu.Lock()
	defer s.subscriberLocationsMu.Unlock()

	s.subscriberLocations[radioID] = &subscriberLocation{
		peerID:   peerID,
		lastSeen: time.Now(),
	}
}

// lookupSubscriberLocation finds which peer a subscriber is behind
// Returns the peer and true if found, or nil and false if not found or stale
func (s *Server) lookupSubscriberLocation(radioID uint32) (*peer.Peer, bool) {
	s.subscriberLocationsMu.RLock()
	loc, exists := s.subscriberLocations[radioID]
	s.subscriberLocationsMu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if location is still fresh (within 15 minutes)
	if time.Since(loc.lastSeen) > 15*time.Minute {
		return nil, false
	}

	// Look up the peer
	p := s.peerManager.GetPeer(loc.peerID)
	if p == nil {
		return nil, false
	}

	// Only return connected peers
	if p.GetState() != peer.StateConnected {
		return nil, false
	}

	return p, true
}

// cleanupStaleSubscriberLocations removes subscriber locations not seen within the TTL
func (s *Server) cleanupStaleSubscriberLocations(ttl time.Duration) {
	s.subscriberLocationsMu.Lock()
	defer s.subscriberLocationsMu.Unlock()

	now := time.Now()
	for radioID, loc := range s.subscriberLocations {
		if now.Sub(loc.lastSeen) > ttl {
			delete(s.subscriberLocations, radioID)
		}
	}
}

// clearSubscriberLocationsForPeer removes all subscriber locations associated with a peer
func (s *Server) clearSubscriberLocationsForPeer(peerID uint32) {
	s.subscriberLocationsMu.Lock()
	defer s.subscriberLocationsMu.Unlock()

	for radioID, loc := range s.subscriberLocations {
		if loc.peerID == peerID {
			delete(s.subscriberLocations, radioID)
		}
	}
}

// Verify challenge (used during authentication)
// verifyChallenge would verify the authentication challenge. Currently unused.
