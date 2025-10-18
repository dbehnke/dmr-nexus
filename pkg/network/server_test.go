package network

import (
	"context"
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
	srv := NewServer(cfg, log)

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
	srv := NewServer(cfg, log)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start(ctx)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

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
	srv := NewServer(cfg, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server
	go srv.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Get the actual port the server is listening on
	if srv.conn == nil {
		t.Fatal("Server connection is nil")
	}
	serverAddr := srv.conn.LocalAddr().(*net.UDPAddr)

	// Create client connection
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer clientConn.Close()

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
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)
	n, err := clientConn.Read(buffer)
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
	srv := NewServer(cfg, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server
	go srv.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	serverAddr := srv.conn.LocalAddr().(*net.UDPAddr)
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	// Send RPTL first
	rptl := &protocol.RPTLPacket{RepeaterID: 312000}
	data, _ := rptl.Encode()
	clientConn.Write(data)

	// Read RPTACK
	buffer := make([]byte, 1024)
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	clientConn.Read(buffer)

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
	clientConn.Write(data)

	// Read RPTACK response
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := clientConn.Read(buffer)
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
	srv := NewServer(cfg, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server
	go srv.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	serverAddr := srv.conn.LocalAddr().(*net.UDPAddr)
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	buffer := make([]byte, 1024)

	// Send RPTL
	rptl := &protocol.RPTLPacket{RepeaterID: 312000}
	data, _ := rptl.Encode()
	clientConn.Write(data)
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	clientConn.Read(buffer)

	// Send RPTK
	challenge := make([]byte, 32)
	rptk := &protocol.RPTKPacket{RepeaterID: 312000, Challenge: challenge}
	data, _ = rptk.Encode()
	clientConn.Write(data)
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	clientConn.Read(buffer)

	// Send RPTC
	rptc := &protocol.RPTCPacket{
		RepeaterID:  312000,
		Callsign:    "W1ABC",
		Location:    "Boston, MA",
		Description: "Test Repeater",
	}
	data, _ = rptc.Encode()
	clientConn.Write(data)

	// Read RPTACK response
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := clientConn.Read(buffer)
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
	srv := NewServer(cfg, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server
	go srv.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	serverAddr := srv.conn.LocalAddr().(*net.UDPAddr)
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	// Send RPTL packet
	rptl := &protocol.RPTLPacket{RepeaterID: 312000}
	data, _ := rptl.Encode()
	clientConn.Write(data)

	// Should receive MSTCL (deny) instead of RPTACK
	buffer := make([]byte, 1024)
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = clientConn.Read(buffer)
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
	srv := NewServer(cfg, log)
	srv.pingTimeout = 200 * time.Millisecond     // Short timeout for testing
	srv.cleanupInterval = 100 * time.Millisecond // Frequent cleanup for testing

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start server
	go srv.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	serverAddr := srv.conn.LocalAddr().(*net.UDPAddr)
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	// Send RPTL to register peer
	rptl := &protocol.RPTLPacket{RepeaterID: 312000}
	data, _ := rptl.Encode()
	clientConn.Write(data)

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
