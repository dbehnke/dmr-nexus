package network

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/peer"
	"github.com/dbehnke/dmr-nexus/pkg/protocol"
)

// Server represents a UDP server for MASTER mode
type Server struct {
	config          config.SystemConfig
	log             *logger.Logger
	conn            *net.UDPConn
	peerManager     *peer.PeerManager
	pingTimeout     time.Duration
	cleanupInterval time.Duration
	regACL          *peer.ACL
	subACL          *peer.ACL
	tg1ACL          *peer.ACL
	tg2ACL          *peer.ACL
}

// NewServer creates a new UDP server for MASTER mode
func NewServer(cfg config.SystemConfig, log *logger.Logger) *Server {
	return &Server{
		config:          cfg,
		log:             log.WithComponent("network.server"),
		peerManager:     peer.NewPeerManager(),
		pingTimeout:     30 * time.Second, // Default timeout
		cleanupInterval: 10 * time.Second, // Default cleanup interval
	}
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
	defer s.conn.Close()

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
		s.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
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
	if len(data) < 4 {
		s.log.Debug("Packet too small", logger.Int("size", len(data)))
		return
	}

	// Get packet type
	packetType := string(data[0:4])

	switch packetType {
	case protocol.PacketTypeDMRD:
		s.handleDMRD(data, addr)
	case protocol.PacketTypeRPTL:
		s.handleRPTL(data, addr)
	case protocol.PacketTypeRPTK:
		s.handleRPTK(data, addr)
	case protocol.PacketTypeRPTC:
		s.handleRPTC(data, addr)
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

	// Send RPTACK
	s.sendRPTACK(rptc.RepeaterID, addr)
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
		return
	}

	// Update last heard
	p.UpdateLastHeard()

	// Send MSTPONG response
	s.sendMSTPONG(peerID, addr)
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

	// Remove peer
	s.peerManager.RemovePeer(peerID)
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

	// Check TG ACL based on timeslot
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

	// Forward to other peers if repeat is enabled
	if s.config.Repeat {
		s.forwardDMRD(dmrd, data, p.ID)
	}
}

// forwardDMRD forwards a DMRD packet to all other connected peers
func (s *Server) forwardDMRD(dmrd *protocol.DMRDPacket, data []byte, sourcePeerID uint32) {
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
			removed := s.peerManager.CleanupTimedOutPeers(s.pingTimeout)
			if removed > 0 {
				s.log.Info("Cleaned up timed out peers", logger.Int("count", removed))
			}
		}
	}
}

// Verify challenge (used during authentication)
func (s *Server) verifyChallenge(peerID uint32, challenge []byte) bool {
	// In a real implementation, we would:
	// 1. Generate our own challenge based on salt + passphrase
	// 2. Compare with received challenge
	// For now, just accept all challenges after RPTL

	p := s.peerManager.GetPeer(peerID)
	if p == nil {
		return false
	}

	// Generate expected challenge
	h := sha256.New()
	h.Write(p.Salt)
	h.Write([]byte(s.config.Passphrase))
	expected := h.Sum(nil)

	// Compare (simplified - in real implementation would need exact match)
	_ = expected
	return true
}
