package network

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/peer"
	"github.com/dbehnke/dmr-nexus/pkg/protocol"
)

func TestServer_New(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:       "MASTER",
		Port:       62031,
		Passphrase: "test",
	}

	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)

	if srv == nil {
		t.Fatal("NewServer returned nil")
	}

	if srv.config.Port != 62031 {
		t.Errorf("Expected port 62031, got %d", srv.config.Port)
	}
}

func TestServer_StartStop(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:       "MASTER",
		Port:       0, // Use any available port
		Passphrase: "test",
	}

	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start(ctx)
	}()

	// Wait for server to report started
	if err := srv.WaitStarted(ctx); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	// Cancel context to stop server
	cancel()

	// Wait for server to stop
	err := <-errChan
	if err != nil && err != context.Canceled {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestServer_HandleRPTL(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:       "MASTER",
		Port:       0,
		Passphrase: "test",
		RegACL:     "PERMIT:ALL",
	}

	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server
	go func() {
		if err := srv.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("srv.Start error: %v", err)
		}
	}()
	if err := srv.WaitStarted(ctx); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	// Get the actual port the server is listening on
	serverAddr, err := srv.Addr()
	if err != nil {
		t.Fatalf("Addr error: %v", err)
	}

	// Create client connection
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer func() {
		if err := clientConn.Close(); err != nil {
			t.Logf("clientConn.Close error: %v", err)
		}
	}()

	// Send RPTL packet
	rptl := &protocol.RPTLPacket{
		RepeaterID: 312000,
	}
	data, err := rptl.Encode()
	if err != nil {
		t.Fatalf("Failed to encode RPTL: %v", err)
	}

	_, err = clientConn.Write(data)
	if err != nil {
		t.Fatalf("Failed to send RPTL: %v", err)
	}

	// Wait for RPTACK response
	if err := clientConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	buffer := make([]byte, 1024)
	n, err := clientConn.Read(buffer)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if err != nil {
		t.Fatalf("Failed to receive RPTACK: %v", err)
	}

	if n < protocol.RPTACKPacketSize {
		t.Fatalf("Response too small: %d bytes", n)
	}

	if string(buffer[0:6]) != protocol.PacketTypeRPTACK {
		t.Errorf("Expected RPTACK, got %s", string(buffer[0:6]))
	}

	// Verify peer was added
	time.Sleep(100 * time.Millisecond)
	if srv.peerManager.Count() != 1 {
		t.Errorf("Expected 1 peer, got %d", srv.peerManager.Count())
	}

	peer := srv.peerManager.GetPeer(312000)
	if peer == nil {
		t.Fatal("Peer not found in manager")
	}
}

func TestServer_HandleRPTK(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:       "MASTER",
		Port:       0,
		Passphrase: "test",
		RegACL:     "PERMIT:ALL",
	}

	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server
	go func() {
		if err := srv.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("srv.Start error: %v", err)
		}
	}()
	if err := srv.WaitStarted(ctx); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	serverAddr, err := srv.Addr()
	if err != nil {
		t.Fatalf("Addr error: %v", err)
	}
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer func() {
		if err := clientConn.Close(); err != nil {
			t.Logf("clientConn.Close error: %v", err)
		}
	}()

	// Send RPTL first
	rptl := &protocol.RPTLPacket{RepeaterID: 312000}
	data, _ := rptl.Encode()
	if _, err := clientConn.Write(data); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	// Read RPTACK
	buffer := make([]byte, 1024)
	if err := clientConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	if _, err := clientConn.Read(buffer); err != nil {
		t.Fatalf("Read error: %v", err)
	}

	// Send RPTK
	challenge := make([]byte, 32)
	for i := range challenge {
		challenge[i] = byte(i)
	}
	rptk := &protocol.RPTKPacket{
		RepeaterID: 312000,
		Challenge:  challenge,
	}
	data, _ = rptk.Encode()
	if _, err := clientConn.Write(data); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	// Read RPTACK response
	if err := clientConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	n, err := clientConn.Read(buffer)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if err != nil {
		t.Fatalf("Failed to receive RPTACK after RPTK: %v", err)
	}

	if string(buffer[0:6]) != protocol.PacketTypeRPTACK {
		t.Errorf("Expected RPTACK after RPTK, got %s", string(buffer[0:n]))
	}
}

func TestServer_HandleRPTC(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:       "MASTER",
		Port:       0,
		Passphrase: "test",
		RegACL:     "PERMIT:ALL",
	}

	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server
	go func() {
		if err := srv.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("srv.Start error: %v", err)
		}
	}()
	if err := srv.WaitStarted(ctx); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	serverAddr, err := srv.Addr()
	if err != nil {
		t.Fatalf("Addr error: %v", err)
	}
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer func() {
		if err := clientConn.Close(); err != nil {
			t.Logf("clientConn.Close error: %v", err)
		}
	}()

	buffer := make([]byte, 1024)

	// Send RPTL
	rptl := &protocol.RPTLPacket{RepeaterID: 312000}
	data, _ := rptl.Encode()
	if _, err := clientConn.Write(data); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if err := clientConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	if _, err := clientConn.Read(buffer); err != nil {
		t.Fatalf("Read error: %v", err)
	}

	// Send RPTK
	challenge := make([]byte, 32)
	rptk := &protocol.RPTKPacket{RepeaterID: 312000, Challenge: challenge}
	data, _ = rptk.Encode()
	if _, err := clientConn.Write(data); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if err := clientConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	if _, err := clientConn.Read(buffer); err != nil {
		t.Fatalf("Read error: %v", err)
	}

	// Send RPTC
	rptc := &protocol.RPTCPacket{
		RepeaterID:  312000,
		Callsign:    "W1ABC",
		Location:    "Boston, MA",
		Description: "Test Repeater",
	}
	data, _ = rptc.Encode()
	if _, err := clientConn.Write(data); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	// Read RPTACK response
	if err := clientConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	n, err := clientConn.Read(buffer)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if err != nil {
		t.Fatalf("Failed to receive RPTACK after RPTC: %v", err)
	}

	if string(buffer[0:6]) != protocol.PacketTypeRPTACK {
		t.Errorf("Expected RPTACK after RPTC, got %s", string(buffer[0:n]))
	}

	// Verify peer config was updated
	time.Sleep(100 * time.Millisecond)
	peer := srv.peerManager.GetPeer(312000)
	if peer == nil {
		t.Fatal("Peer not found")
	}

	if peer.Callsign != "W1ABC" {
		t.Errorf("Expected callsign W1ABC, got %s", peer.Callsign)
	}
}

func TestServer_ACLDeny(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:       "MASTER",
		Port:       0,
		Passphrase: "test",
		UseACL:     true,
		RegACL:     "DENY:312000", // Deny specific peer
	}

	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start(ctx)
	}()
	if err := srv.WaitStarted(ctx); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	serverAddr, err := srv.Addr()
	if err != nil {
		t.Fatalf("Addr error: %v", err)
	}
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer func() {
		_ = clientConn.Close()
		cancel()
		// ensure Start exits before test returns
		<-errChan
	}()

	// Send RPTL packet
	rptl := &protocol.RPTLPacket{RepeaterID: 312000}
	data, _ := rptl.Encode()
	if _, err := clientConn.Write(data); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	// Should receive MSTCL (deny) instead of RPTACK
	buffer := make([]byte, 1024)
	if err := clientConn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	_, err = clientConn.Read(buffer)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if err != nil {
		t.Fatalf("Failed to receive response: %v", err)
	}

	// Should be MSTCL (deny)
	if string(buffer[0:5]) == protocol.PacketTypeMSTCL {
		// Expected - peer was denied
	} else if string(buffer[0:6]) == protocol.PacketTypeRPTACK {
		t.Error("Expected MSTCL (deny), got RPTACK (should have been denied by ACL)")
	}

	// Verify peer was NOT added
	time.Sleep(100 * time.Millisecond)
	peer := srv.peerManager.GetPeer(312000)
	if peer != nil {
		t.Error("Peer should not be in manager (denied by ACL)")
	}
}

func TestServer_PeerTimeout(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:       "MASTER",
		Port:       0,
		Passphrase: "test",
		RegACL:     "PERMIT:ALL",
	}

	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)
	srv.pingTimeout = 200 * time.Millisecond     // Short timeout for testing
	srv.cleanupInterval = 100 * time.Millisecond // Frequent cleanup for testing

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start server
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start(ctx)
	}()
	if err := srv.WaitStarted(ctx); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	serverAddr, err := srv.Addr()
	if err != nil {
		t.Fatalf("Addr error: %v", err)
	}
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer func() {
		_ = clientConn.Close()
		cancel()
		<-errChan
	}()

	// Send RPTL to register peer
	rptl := &protocol.RPTLPacket{RepeaterID: 312000}
	data, _ := rptl.Encode()
	if _, err := clientConn.Write(data); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	// Wait for peer to be added
	time.Sleep(50 * time.Millisecond)

	if srv.peerManager.Count() != 1 {
		t.Fatalf("Expected 1 peer, got %d", srv.peerManager.Count())
	}

	// Wait for timeout cleanup (pingTimeout + cleanupInterval + buffer)
	time.Sleep(400 * time.Millisecond)

	// Peer should be removed due to timeout
	if srv.peerManager.Count() != 0 {
		t.Errorf("Expected 0 peers after timeout, got %d", srv.peerManager.Count())
	}
}

// Additional coverage: verify DMRD forwarding when Repeat is enabled
func TestServer_ForwardDMRD_RepeatEnabled(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:   "MASTER",
		Repeat: true,
	}
	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)

	// Bind a UDP socket for the server without starting background loops
	srvAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	serverConn, err := net.ListenUDP("udp", srvAddr)
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	srv.conn = serverConn
	defer func() { _ = serverConn.Close() }()

	// Destination peer (should receive forwarded DMRD)
	destConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("dest ListenUDP error: %v", err)
	}
	defer func() { _ = destConn.Close() }()
	destPeer := srv.peerManager.AddPeer(222, destConn.LocalAddr().(*net.UDPAddr))
	destPeer.SetConnected()

	// Source peer (should be excluded)
	srcAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 65000}
	srcPeer := srv.peerManager.AddPeer(111, srcAddr)
	srcPeer.SetConnected()

	// Prepare a DMRD packet
	dmrd := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    111,
		Timeslot:      1,
		CallType:      0,
		StreamID:      12345,
		Payload:       make([]byte, 33),
	}
	data, err := dmrd.Encode()
	if err != nil {
		t.Fatalf("Encode DMRD error: %v", err)
	}

	// Trigger forwarding
	if err := destConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.forwardDMRD(dmrd, data, srcPeer.ID)

	// Expect to receive the forwarded packet on destination
	buf := make([]byte, 2048)
	n, _, err := destConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("dest ReadFromUDP error: %v", err)
	}
	if n != len(data) {
		t.Fatalf("forward size mismatch: got %d want %d", n, len(data))
	}
}

// Additional coverage: RPTPING should generate MSTPONG to the sender
func TestServer_HandleRPTPING_SendsMSTPONG(t *testing.T) {
	cfg := config.SystemConfig{Mode: "MASTER"}
	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)

	// Bind server UDP socket
	serverConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	srv.conn = serverConn
	defer func() { _ = serverConn.Close() }()

	// Sender socket (peer)
	senderConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("sender ListenUDP error: %v", err)
	}
	defer func() { _ = senderConn.Close() }()

	// Register the peer so handleRPTPING finds it
	peerID := uint32(312000)
	p := srv.peerManager.AddPeer(peerID, senderConn.LocalAddr().(*net.UDPAddr))
	p.SetConnected()

	// Craft an RPTPING packet (7-byte type + repeater id at [7:11])
	ping := make([]byte, protocol.RPTPINGPacketSize)
	copy(ping[0:7], protocol.PacketTypeRPTPING)
	binary.BigEndian.PutUint32(ping[7:11], peerID)

	if err := senderConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	// Call handler directly with sender address
	srv.handleRPTPING(ping, senderConn.LocalAddr().(*net.UDPAddr))

	// Expect MSTPONG back
	buf := make([]byte, 64)
	n, _, err := senderConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("sender ReadFromUDP error: %v", err)
	}
	if n < protocol.MSTPONGPacketSize {
		t.Fatalf("pong too small: %d", n)
	}
	if string(buf[0:7]) != protocol.PacketTypeMSTPONG {
		t.Fatalf("expected MSTPONG, got %q", string(buf[0:n]))
	}
	gotID := binary.BigEndian.Uint32(buf[7:11])
	if gotID != peerID {
		t.Fatalf("MSTPONG peer id mismatch: got %d want %d", gotID, peerID)
	}
}

func TestServer_HandleRPTPING_UnknownPeer_CooldownBehavior(t *testing.T) {
	cfg := config.SystemConfig{Mode: "MASTER"}
	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)

	// Bind server UDP socket
	serverConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	srv.conn = serverConn
	defer func() { _ = serverConn.Close() }()

	// Sender socket (peer)
	senderConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("sender ListenUDP error: %v", err)
	}
	defer func() { _ = senderConn.Close() }()

	// Use an unknown peer ID (not registered)
	peerID := uint32(999999)

	// Craft an RPTPING packet
	ping := make([]byte, protocol.RPTPINGPacketSize)
	copy(ping[0:7], protocol.PacketTypeRPTPING)
	binary.BigEndian.PutUint32(ping[7:11], peerID)

	// First RPTPING from unknown peer should get MSTNAK
	if err := senderConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleRPTPING(ping, senderConn.LocalAddr().(*net.UDPAddr))

	// Expect MSTNAK back
	buf := make([]byte, 64)
	n, _, err := senderConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("sender ReadFromUDP error (first MSTNAK): %v", err)
	}
	if n < protocol.MSTNAKPacketSize {
		t.Fatalf("MSTNAK too small: %d", n)
	}
	if string(buf[0:6]) != protocol.PacketTypeMSTNAK {
		t.Fatalf("expected MSTNAK, got %q", string(buf[0:n]))
	}
	gotID := binary.BigEndian.Uint32(buf[6:10])
	if gotID != peerID {
		t.Fatalf("MSTNAK peer id mismatch: got %d want %d", gotID, peerID)
	}

	// Second RPTPING from same unknown peer should be silently ignored (no response)
	if err := senderConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleRPTPING(ping, senderConn.LocalAddr().(*net.UDPAddr))

	// Should timeout (no response)
	_, _, err = senderConn.ReadFromUDP(buf)
	if err == nil {
		t.Fatal("Expected timeout (no response), but got a response")
	}
	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Fatalf("Expected timeout error, got: %v", err)
	}

	// Third RPTPING should also be ignored
	if err := senderConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleRPTPING(ping, senderConn.LocalAddr().(*net.UDPAddr))

	_, _, err = senderConn.ReadFromUDP(buf)
	if err == nil {
		t.Fatal("Expected timeout (no response), but got a response")
	}
	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Fatalf("Expected timeout error, got: %v", err)
	}
}

// DMRD from unknown peer should get MSTNAK once, then be ignored during cooldown
func TestServer_HandleDMRD_UnknownPeer_CooldownBehavior(t *testing.T) {
	cfg := config.SystemConfig{Mode: "MASTER"}
	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)

	// Bind server UDP socket
	serverConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	srv.conn = serverConn
	defer func() { _ = serverConn.Close() }()

	// Sender socket (unknown peer)
	senderConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("sender ListenUDP error: %v", err)
	}
	defer func() { _ = senderConn.Close() }()

	peerID := uint32(999888)

	// Prepare a DMRD packet with RepeaterID set to peerID
	dmrd := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    peerID,
		Timeslot:      1,
		CallType:      0,
		StreamID:      12345,
		Payload:       make([]byte, 33),
	}
	data, err := dmrd.Encode()
	if err != nil {
		t.Fatalf("Encode DMRD error: %v", err)
	}

	// First DMRD - expect MSTNAK
	if err := senderConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleDMRD(data, senderConn.LocalAddr().(*net.UDPAddr))

	buf := make([]byte, 64)
	n, _, err := senderConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("sender ReadFromUDP error (first MSTNAK): %v", err)
	}
	if n < protocol.MSTNAKPacketSize {
		t.Fatalf("MSTNAK too small: %d", n)
	}
	if string(buf[0:6]) != protocol.PacketTypeMSTNAK {
		t.Fatalf("expected MSTNAK, got %q", string(buf[0:n]))
	}
	gotID := binary.BigEndian.Uint32(buf[6:10])
	if gotID != peerID {
		t.Fatalf("MSTNAK peer id mismatch: got %d want %d", gotID, peerID)
	}

	// Second DMRD immediately - should be silently ignored (no response)
	if err := senderConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleDMRD(data, senderConn.LocalAddr().(*net.UDPAddr))

	_, _, err = senderConn.ReadFromUDP(buf)
	if err == nil {
		t.Fatal("Expected timeout (no response), but got a response")
	}
	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Fatalf("Expected timeout error, got: %v", err)
	}
}

func TestServer_HandleDMRD_UnknownPeer_CooldownExpires(t *testing.T) {
	cfg := config.SystemConfig{Mode: "MASTER"}
	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)
	// shorten cooldown for test
	srv.mstNakCooldown = 100 * time.Millisecond

	// Bind server UDP socket
	serverConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	srv.conn = serverConn
	defer func() { _ = serverConn.Close() }()

	// Sender socket (unknown peer)
	senderConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("sender ListenUDP error: %v", err)
	}
	defer func() { _ = senderConn.Close() }()

	peerID := uint32(777666)

	// Prepare a DMRD packet with RepeaterID set to peerID
	dmrd := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    peerID,
		Timeslot:      1,
		CallType:      0,
		StreamID:      12345,
		Payload:       make([]byte, 33),
	}
	data, err := dmrd.Encode()
	if err != nil {
		t.Fatalf("Encode DMRD error: %v", err)
	}

	// First DMRD - should get MSTNAK
	if err := senderConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleDMRD(data, senderConn.LocalAddr().(*net.UDPAddr))
	buf := make([]byte, 64)
	n, _, err := senderConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("sender ReadFromUDP error (first MSTNAK): %v", err)
	}
	if string(buf[0:6]) != protocol.PacketTypeMSTNAK {
		t.Fatalf("expected MSTNAK, got %q", string(buf[0:n]))
	}

	// Second DMRD immediately - should be ignored
	if err := senderConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleDMRD(data, senderConn.LocalAddr().(*net.UDPAddr))
	_, _, err = senderConn.ReadFromUDP(buf)
	if err == nil {
		t.Fatal("Expected timeout (no response), but got a response")
	}

	// Wait for cooldown to expire
	time.Sleep(150 * time.Millisecond)

	// After cooldown, another DMRD should get MSTNAK again
	if err := senderConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleDMRD(data, senderConn.LocalAddr().(*net.UDPAddr))
	n, _, err = senderConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("sender ReadFromUDP error (MSTNAK after cooldown): %v", err)
	}
	if string(buf[0:6]) != protocol.PacketTypeMSTNAK {
		t.Fatalf("expected MSTNAK after cooldown, got %q", string(buf[0:n]))
	}
}

func TestServer_HandleRPTPING_UnknownPeer_CooldownExpires(t *testing.T) {
	cfg := config.SystemConfig{Mode: "MASTER"}
	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)
	// Use a shorter cooldown for testing
	srv.mstNakCooldown = 100 * time.Millisecond

	// Bind server UDP socket
	serverConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	srv.conn = serverConn
	defer func() { _ = serverConn.Close() }()

	// Sender socket (peer)
	senderConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("sender ListenUDP error: %v", err)
	}
	defer func() { _ = senderConn.Close() }()

	peerID := uint32(888888)

	// Craft an RPTPING packet
	ping := make([]byte, protocol.RPTPINGPacketSize)
	copy(ping[0:7], protocol.PacketTypeRPTPING)
	binary.BigEndian.PutUint32(ping[7:11], peerID)

	// First RPTPING - should get MSTNAK
	if err := senderConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleRPTPING(ping, senderConn.LocalAddr().(*net.UDPAddr))

	buf := make([]byte, 64)
	n, _, err := senderConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("Expected MSTNAK: %v", err)
	}
	if string(buf[0:6]) != protocol.PacketTypeMSTNAK {
		t.Fatalf("expected MSTNAK, got %q", string(buf[0:n]))
	}

	// Wait for cooldown to expire
	time.Sleep(150 * time.Millisecond)

	// RPTPING after cooldown expires - should get MSTNAK again
	if err := senderConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleRPTPING(ping, senderConn.LocalAddr().(*net.UDPAddr))

	n, _, err = senderConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("Expected MSTNAK after cooldown: %v", err)
	}
	if string(buf[0:6]) != protocol.PacketTypeMSTNAK {
		t.Fatalf("expected MSTNAK after cooldown, got %q", string(buf[0:n]))
	}
}

func TestStreamMuteFirstTransmission_AAA(t *testing.T) {
	// Arrange
	cfg := config.SystemConfig{
		Mode:       "MASTER",
		Port:       0,
		Passphrase: "test",
	}
	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)
	streamID := uint32(12345)
	// Simulate AddDynamic returning true (first key-up)
	srv.mutedStreams[streamID] = time.Now().Add(2 * time.Second)

	// Act
	muted, exists := srv.mutedStreams[streamID]

	// Assert
	if !exists {
		t.Errorf("Stream should be muted on first transmission")
	}
	if time.Until(muted) < time.Second {
		t.Errorf("Mute expiry should be at least 2s from now")
	}

	// Simulate cleanup after 2s
	time.Sleep(2100 * time.Millisecond)
	srv.CleanupMutedStreamsOnce(time.Now())
	_, stillMuted := srv.mutedStreams[streamID]
	if stillMuted {
		t.Errorf("Stream mute should be cleaned up after idle timeout")
	}
}

// TestServer_PrivateCallRouting tests private call routing between two peers
func TestServer_PrivateCallRouting(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:                "MASTER",
		Port:                0,
		Passphrase:          "test",
		PrivateCallsEnabled: true,
		UseACL:              false,
	}

	log := logger.New(logger.Config{Level: "debug"})
	srv := NewServer(cfg, "test-system", log)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start server
	go func() {
		if err := srv.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("srv.Start error: %v", err)
		}
	}()
	if err := srv.WaitStarted(ctx); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	serverAddr, err := srv.Addr()
	if err != nil {
		t.Fatalf("Addr error: %v", err)
	}

	// Create two peer connections
	peer1ID := uint32(312001)
	peer2ID := uint32(312002)
	radio1ID := uint32(3120001) // Radio behind peer1
	radio2ID := uint32(3120002) // Radio behind peer2

	// Connect peer 1
	conn1, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create peer1 connection: %v", err)
	}
	defer func() {
		if err := conn1.Close(); err != nil {
			t.Logf("conn1.Close error: %v", err)
		}
	}()

	// Connect peer 2
	conn2, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create peer2 connection: %v", err)
	}
	defer func() {
		if err := conn2.Close(); err != nil {
			t.Logf("conn2.Close error: %v", err)
		}
	}()

	// Perform full connection handshake for peer1
	if err := connectPeer(conn1, peer1ID, "PEER1"); err != nil {
		t.Fatalf("Failed to connect peer1: %v", err)
	}

	// Perform full connection handshake for peer2
	if err := connectPeer(conn2, peer2ID, "PEER2"); err != nil {
		t.Fatalf("Failed to connect peer2: %v", err)
	}

	// Wait for peers to be connected
	time.Sleep(200 * time.Millisecond)

	// Verify peers are connected
	p1 := srv.peerManager.GetPeer(peer1ID)
	if p1 == nil {
		t.Fatal("Peer1 not found in manager")
	}
	p2 := srv.peerManager.GetPeer(peer2ID)
	if p2 == nil {
		t.Fatal("Peer2 not found in manager")
	}

	// Send a group call from radio1 to establish its location
	dmrdGroup := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      radio1ID,
		DestinationID: 3100, // Group call
		RepeaterID:    peer1ID,
		Timeslot:      1,
		CallType:      protocol.CallTypeGroup,
		FrameType:     protocol.FrameTypeVoiceHeader,
		DataType:      0,
		StreamID:      1001,
		Payload:       make([]byte, 33),
	}
	data1, err := dmrdGroup.Encode()
	if err != nil {
		t.Fatalf("Failed to encode DMRD: %v", err)
	}
	t.Logf("Sending from conn1 (peer1): radio=%d, peer=%d", radio1ID, peer1ID)
	if _, err := conn1.Write(data1); err != nil {
		t.Fatalf("Failed to send group call: %v", err)
	}

	// Small delay to ensure packets are processed separately
	time.Sleep(100 * time.Millisecond)

	// Send a group call from radio2 to establish its location
	dmrdGroup2 := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      radio2ID,
		DestinationID: 3100, // Group call
		RepeaterID:    peer2ID,
		Timeslot:      1,
		CallType:      protocol.CallTypeGroup,
		FrameType:     protocol.FrameTypeVoiceHeader,
		DataType:      0,
		StreamID:      1002,
		Payload:       make([]byte, 33),
	}
	data2, err := dmrdGroup2.Encode()
	if err != nil {
		t.Fatalf("Failed to encode DMRD: %v", err)
	}
	t.Logf("Sending from conn2 (peer2): radio=%d, peer=%d", radio2ID, peer2ID)
	if _, err := conn2.Write(data2); err != nil {
		t.Fatalf("Failed to send group call: %v", err)
	}

	// Wait for location tracking to update
	time.Sleep(500 * time.Millisecond)

	// Debug: check the subscriber map directly
	srv.subscriberLocationsMu.RLock()
	t.Logf("Subscriber locations map: %+v", srv.subscriberLocations)
	srv.subscriberLocationsMu.RUnlock()

	// Verify subscriber locations are tracked
	loc1, found1 := srv.lookupSubscriberLocation(radio1ID)
	if !found1 {
		t.Fatalf("Radio1 location not tracked: found=%v", found1)
	}
	if loc1.ID != peer1ID {
		t.Fatalf("Radio1 location incorrect: expected peerID=%d, got=%d", peer1ID, loc1.ID)
	}

	loc2, found2 := srv.lookupSubscriberLocation(radio2ID)
	if !found2 {
		t.Fatalf("Radio2 location not tracked: found=%v", found2)
	}
	if loc2.ID != peer2ID {
		t.Fatalf("Radio2 location incorrect: expected peerID=%d, got=%d", peer2ID, loc2.ID)
	}

	// Now send a private call from radio1 to radio2
	dmrdPrivate := &protocol.DMRDPacket{
		Sequence:      2,
		SourceID:      radio1ID,
		DestinationID: radio2ID, // Private call to radio2
		RepeaterID:    peer1ID,
		Timeslot:      1,
		CallType:      protocol.CallTypePrivate, // Private call
		FrameType:     protocol.FrameTypeVoiceHeader,
		DataType:      0,
		StreamID:      1003,
		Payload:       make([]byte, 33),
	}
	privateData, _ := dmrdPrivate.Encode()
	if _, err := conn1.Write(privateData); err != nil {
		t.Fatalf("Failed to send private call: %v", err)
	}

	// Read from peer2's connection to verify it received the private call
	buffer := make([]byte, 1024)
	if err := conn2.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}

	n, err := conn2.Read(buffer)
	if err != nil {
		t.Fatalf("Peer2 should receive private call: %v", err)
	}

	// Verify it's a DMRD packet
	if string(buffer[0:4]) != protocol.PacketTypeDMRD {
		t.Errorf("Expected DMRD packet, got %s", string(buffer[0:4]))
	}

	// Parse and verify it's the private call
	received, err := protocol.ParseDMRD(buffer[:n])
	if err != nil {
		t.Fatalf("Failed to parse received DMRD: %v", err)
	}

	if received.SourceID != radio1ID {
		t.Errorf("Expected source %d, got %d", radio1ID, received.SourceID)
	}
	if received.DestinationID != radio2ID {
		t.Errorf("Expected destination %d, got %d", radio2ID, received.DestinationID)
	}
	if received.CallType != protocol.CallTypePrivate {
		t.Errorf("Expected private call type, got %d", received.CallType)
	}

	// Verify peer1 does not receive the packet back
	if err := conn1.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	_, err = conn1.Read(buffer)
	if err == nil {
		t.Error("Peer1 should not receive its own private call back")
	}
}

// TestServer_PrivateCallDisabled tests that private calls are not routed when disabled
func TestServer_PrivateCallDisabled(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:                "MASTER",
		Port:                0,
		Passphrase:          "test",
		PrivateCallsEnabled: false, // Disabled
		UseACL:              false,
	}

	log := logger.New(logger.Config{Level: "debug"})
	srv := NewServer(cfg, "test-system", log)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start server
	go func() {
		if err := srv.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("srv.Start error: %v", err)
		}
	}()
	if err := srv.WaitStarted(ctx); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	serverAddr, err := srv.Addr()
	if err != nil {
		t.Fatalf("Addr error: %v", err)
	}

	peer1ID := uint32(312001)
	peer2ID := uint32(312002)
	radio1ID := uint32(3120001)
	radio2ID := uint32(3120002)

	conn1, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create peer1 connection: %v", err)
	}
	defer func() {
		if err := conn1.Close(); err != nil {
			t.Logf("conn1.Close error: %v", err)
		}
	}()

	conn2, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create peer2 connection: %v", err)
	}
	defer func() {
		if err := conn2.Close(); err != nil {
			t.Logf("conn2.Close error: %v", err)
		}
	}()

	if err := connectPeer(conn1, peer1ID, "PEER1"); err != nil {
		t.Fatalf("Failed to connect peer1: %v", err)
	}
	if err := connectPeer(conn2, peer2ID, "PEER2"); err != nil {
		t.Fatalf("Failed to connect peer2: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Send private call with feature disabled
	dmrdPrivate := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      radio1ID,
		DestinationID: radio2ID,
		RepeaterID:    peer1ID,
		Timeslot:      1,
		CallType:      protocol.CallTypePrivate,
		FrameType:     protocol.FrameTypeVoiceHeader,
		DataType:      0,
		StreamID:      1001,
		Payload:       make([]byte, 33),
	}
	privateData, _ := dmrdPrivate.Encode()
	if _, err := conn1.Write(privateData); err != nil {
		t.Fatalf("Failed to send private call: %v", err)
	}

	// Peer2 should NOT receive the private call since feature is disabled
	buffer := make([]byte, 1024)
	if err := conn2.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}

	_, err = conn2.Read(buffer)
	if err == nil {
		t.Error("Peer2 should not receive private call when feature is disabled")
	}
}

// TestServer_PrivateCallUnknownDestination tests routing when destination is unknown
func TestServer_PrivateCallUnknownDestination(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:                "MASTER",
		Port:                0,
		Passphrase:          "test",
		PrivateCallsEnabled: true,
		UseACL:              false,
	}

	log := logger.New(logger.Config{Level: "debug"})
	srv := NewServer(cfg, "test-system", log)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		if err := srv.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("srv.Start error: %v", err)
		}
	}()
	if err := srv.WaitStarted(ctx); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}

	serverAddr, err := srv.Addr()
	if err != nil {
		t.Fatalf("Addr error: %v", err)
	}

	peer1ID := uint32(312001)
	radio1ID := uint32(3120001)
	unknownRadio := uint32(9999999)

	conn1, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create peer1 connection: %v", err)
	}
	defer func() {
		if err := conn1.Close(); err != nil {
			t.Logf("conn1.Close error: %v", err)
		}
	}()

	if err := connectPeer(conn1, peer1ID, "PEER1"); err != nil {
		t.Fatalf("Failed to connect peer1: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Send private call to unknown destination
	dmrdPrivate := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      radio1ID,
		DestinationID: unknownRadio,
		RepeaterID:    peer1ID,
		Timeslot:      1,
		CallType:      protocol.CallTypePrivate,
		FrameType:     protocol.FrameTypeVoiceHeader,
		DataType:      0,
		StreamID:      1001,
		Payload:       make([]byte, 33),
	}
	privateData, _ := dmrdPrivate.Encode()
	if _, err := conn1.Write(privateData); err != nil {
		t.Fatalf("Failed to send private call: %v", err)
	}

	// Should not receive anything back (call dropped)
	buffer := make([]byte, 1024)
	if err := conn1.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}

	_, err = conn1.Read(buffer)
	if err == nil {
		t.Error("Should not receive response for private call to unknown destination")
	}
}

// TestServer_SubscriberLocationCleanup tests that stale subscriber locations are cleaned up
func TestServer_SubscriberLocationCleanup(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:                "MASTER",
		Port:                0,
		PrivateCallsEnabled: true,
	}

	log := logger.New(logger.Config{Level: "debug"})
	srv := NewServer(cfg, "test-system", log)

	radioID := uint32(3120001)
	peerID := uint32(312001)

	// Track a subscriber location
	srv.trackSubscriberLocation(radioID, peerID)

	// Verify it exists
	_, found := srv.subscriberLocations[radioID]
	if !found {
		t.Fatal("Subscriber location should be tracked")
	}

	// Manually set the last seen time to 20 minutes ago
	srv.subscriberLocationsMu.Lock()
	srv.subscriberLocations[radioID].lastSeen = time.Now().Add(-20 * time.Minute)
	srv.subscriberLocationsMu.Unlock()

	// Run cleanup with 15 minute TTL
	srv.cleanupStaleSubscriberLocations(15 * time.Minute)

	// Verify it's cleaned up
	_, found = srv.subscriberLocations[radioID]
	if found {
		t.Error("Stale subscriber location should be cleaned up")
	}
}

// connectPeer performs a full connection handshake for a peer
func connectPeer(conn *net.UDPConn, peerID uint32, callsign string) error {
	buffer := make([]byte, 1024)

	// Send RPTL
	rptl := &protocol.RPTLPacket{RepeaterID: peerID}
	data, _ := rptl.Encode()
	if _, err := conn.Write(data); err != nil {
		return err
	}

	// Read RPTACK
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	if _, err := conn.Read(buffer); err != nil {
		return err
	}

	// Send RPTK
	challenge := make([]byte, 32)
	rptk := &protocol.RPTKPacket{
		RepeaterID: peerID,
		Challenge:  challenge,
	}
	data, _ = rptk.Encode()
	if _, err := conn.Write(data); err != nil {
		return err
	}

	// Read RPTACK
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	if _, err := conn.Read(buffer); err != nil {
		return err
	}

	// Send RPTC
	rptc := &protocol.RPTCPacket{
		RepeaterID:  peerID,
		Callsign:    callsign,
		RXFreq:      "449000000",
		TXFreq:      "444000000",
		TXPower:     "25",
		ColorCode:   "1",
		Latitude:    "42.3601",
		Longitude:   "-71.0589",
		Height:      "75",
		Location:    "Test",
		Description: "Test Peer",
		URL:         "http://test.local",
		SoftwareID:  "DMR-Nexus",
		PackageID:   "DMR-Nexus",
	}
	data, _ = rptc.Encode()
	if _, err := conn.Write(data); err != nil {
		return err
	}

	// Read RPTACK
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	if _, err := conn.Read(buffer); err != nil {
		return err
	}

	return nil
}

// Ensure that DMRD packets from peers that are NOT fully connected are ignored
func TestServer_IgnoreDMRDFromNonConnectedPeer(t *testing.T) {
	cfg := config.SystemConfig{
		Mode: "MASTER",
	}
	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)

	// Create a sender socket to represent the peer's address
	senderConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("sender ListenUDP error: %v", err)
	}
	defer func() { _ = senderConn.Close() }()

	// Bind a server UDP socket so MSTNAK/MSTPONG can be sent in tests
	serverConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("server ListenUDP error: %v", err)
	}
	srv.conn = serverConn
	defer func() { _ = serverConn.Close() }()

	// Add peer to manager but DO NOT mark as connected (leave as disconnected / partial state)
	peerID := uint32(312000)
	p := srv.peerManager.AddPeer(peerID, senderConn.LocalAddr().(*net.UDPAddr))

	// Sanity: ensure peer is not connected
	if p.GetState() == peer.StateConnected {
		t.Fatalf("test setup: peer should not be connected")
	}

	// Prepare a DMRD packet coming from this peer
	dmrd := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    peerID,
		Timeslot:      1,
		CallType:      0,
		StreamID:      12345,
		Payload:       make([]byte, 33),
	}
	data, err := dmrd.Encode()
	if err != nil {
		t.Fatalf("Encode DMRD error: %v", err)
	}

	// Pre-conditions: counters should be zero and LastHeard zero
	if p.PacketsReceived != 0 || p.BytesReceived != 0 || !p.GetLastHeard().IsZero() {
		t.Fatalf("precondition failed: peer counters/last-heard not zero")
	}

	// Call handler as if packet was received from the sender address
	srv.handleDMRD(data, senderConn.LocalAddr().(*net.UDPAddr))

	// Post-condition: since peer is not connected, stats and last-heard should be unchanged
	if p.PacketsReceived != 0 {
		t.Fatalf("expected 0 PacketsReceived, got %d", p.PacketsReceived)
	}
	if p.BytesReceived != 0 {
		t.Fatalf("expected 0 BytesReceived, got %d", p.BytesReceived)
	}
	if !p.GetLastHeard().IsZero() {
		t.Fatalf("expected LastHeard to be zero/time zero, got %v", p.GetLastHeard())
	}
}

// Known but non-connected peer should receive MSTNAK on DMRD and be tracked in rejectedPeers
func TestServer_HandleDMRD_KnownNonConnectedPeer_ReceivesMSTNAK(t *testing.T) {
	cfg := config.SystemConfig{Mode: "MASTER"}
	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)

	// Bind server UDP socket so sendMSTNAK can write
	serverConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	srv.conn = serverConn
	defer func() { _ = serverConn.Close() }()

	// Sender socket (peer)
	senderConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("sender ListenUDP error: %v", err)
	}
	defer func() { _ = senderConn.Close() }()

	peerID := uint32(555111)

	// Add peer to manager but DO NOT mark connected
	p := srv.peerManager.AddPeer(peerID, senderConn.LocalAddr().(*net.UDPAddr))
	if p.GetState() == peer.StateConnected {
		t.Fatalf("test setup: peer should not be connected")
	}

	// Prepare DMRD packet
	dmrd := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    peerID,
		Timeslot:      1,
		CallType:      0,
		StreamID:      12345,
		Payload:       make([]byte, 33),
	}
	data, err := dmrd.Encode()
	if err != nil {
		t.Fatalf("Encode DMRD error: %v", err)
	}

	// First DMRD - expect MSTNAK
	if err := senderConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleDMRD(data, senderConn.LocalAddr().(*net.UDPAddr))

	buf := make([]byte, 64)
	n, _, err := senderConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("sender ReadFromUDP error (first MSTNAK): %v", err)
	}
	if n < protocol.MSTNAKPacketSize {
		t.Fatalf("MSTNAK too small: %d", n)
	}
	if string(buf[0:6]) != protocol.PacketTypeMSTNAK {
		t.Fatalf("expected MSTNAK, got %q", string(buf[0:n]))
	}
	gotID := binary.BigEndian.Uint32(buf[6:10])
	if gotID != peerID {
		t.Fatalf("MSTNAK peer id mismatch: got %d want %d", gotID, peerID)
	}

	// Verify rejectedPeers recorded entry
	key := peerKey(peerID, senderConn.LocalAddr().(*net.UDPAddr))
	srv.rejectedPeersMu.Lock()
	_, exists := srv.rejectedPeers[key]
	srv.rejectedPeersMu.Unlock()
	if !exists {
		t.Fatalf("expected rejectedPeers to contain key %s", key)
	}

	// Second DMRD immediately should be ignored (no response)
	if err := senderConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleDMRD(data, senderConn.LocalAddr().(*net.UDPAddr))
	_, _, err = senderConn.ReadFromUDP(buf)
	if err == nil {
		t.Fatal("Expected timeout (no response), but got a response")
	}
	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Fatalf("Expected timeout error, got: %v", err)
	}
}
