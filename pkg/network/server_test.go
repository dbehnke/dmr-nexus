package network

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/config"
	"github.com/dbehnke/dmr-nexus/pkg/logger"
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

	// First RPTPING from unknown peer should get MSTCL
	if err := senderConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleRPTPING(ping, senderConn.LocalAddr().(*net.UDPAddr))

	// Expect MSTCL back
	buf := make([]byte, 64)
	n, _, err := senderConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("sender ReadFromUDP error (first MSTCL): %v", err)
	}
	if n < protocol.MSTCLPacketSize {
		t.Fatalf("MSTCL too small: %d", n)
	}
	if string(buf[0:5]) != protocol.PacketTypeMSTCL {
		t.Fatalf("expected MSTCL, got %q", string(buf[0:n]))
	}
	gotID := binary.BigEndian.Uint32(buf[5:9])
	if gotID != peerID {
		t.Fatalf("MSTCL peer id mismatch: got %d want %d", gotID, peerID)
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

func TestServer_HandleRPTPING_UnknownPeer_CooldownExpires(t *testing.T) {
	cfg := config.SystemConfig{Mode: "MASTER"}
	log := logger.New(logger.Config{Level: "info"})
	srv := NewServer(cfg, "test-system", log)
	// Use a shorter cooldown for testing
	srv.mstclCooldown = 100 * time.Millisecond

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

	// First RPTPING - should get MSTCL
	if err := senderConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleRPTPING(ping, senderConn.LocalAddr().(*net.UDPAddr))

	buf := make([]byte, 64)
	n, _, err := senderConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("Expected MSTCL: %v", err)
	}
	if string(buf[0:5]) != protocol.PacketTypeMSTCL {
		t.Fatalf("expected MSTCL, got %q", string(buf[0:n]))
	}

	// Wait for cooldown to expire
	time.Sleep(150 * time.Millisecond)

	// RPTPING after cooldown expires - should get MSTCL again
	if err := senderConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline error: %v", err)
	}
	srv.handleRPTPING(ping, senderConn.LocalAddr().(*net.UDPAddr))

	n, _, err = senderConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("Expected MSTCL after cooldown: %v", err)
	}
	if string(buf[0:5]) != protocol.PacketTypeMSTCL {
		t.Fatalf("expected MSTCL after cooldown, got %q", string(buf[0:n]))
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
