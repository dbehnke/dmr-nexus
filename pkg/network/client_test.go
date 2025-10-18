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

func TestClient_New(t *testing.T) {
	cfg := config.SystemConfig{
		Mode:       "PEER",
		MasterIP:   "127.0.0.1",
		MasterPort: 62031,
		Port:       62032,
		RadioID:    312000,
		Passphrase: "test",
	}

	log := logger.New(logger.Config{Level: "info"})
	client := NewClient(cfg, log)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.config.RadioID != 312000 {
		t.Errorf("Expected radio ID 312000, got %d", client.config.RadioID)
	}
}

func TestClient_Connect(t *testing.T) {
	// Create a mock UDP server
	serverAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0, // Let OS assign port
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer serverConn.Close()

	actualServerPort := serverConn.LocalAddr().(*net.UDPAddr).Port

	// Create client configuration
	cfg := config.SystemConfig{
		Mode:       "PEER",
		MasterIP:   "127.0.0.1",
		MasterPort: actualServerPort,
		Port:       0, // Let OS assign port
		RadioID:    312000,
		Passphrase: "test",
	}

	log := logger.New(logger.Config{Level: "debug"})
	client := NewClient(cfg, log)

	// Start client in background
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- client.Start(ctx)
	}()

	// Wait a bit for client to start
	time.Sleep(100 * time.Millisecond)

	// Mock server should receive RPTL packet
	buffer := make([]byte, 1024)
	serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, clientAddr, err := serverConn.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Mock server failed to receive packet: %v", err)
	}

	// Verify it's an RPTL packet
	if n < protocol.RPTLPacketSize {
		t.Fatalf("Expected at least %d bytes, got %d", protocol.RPTLPacketSize, n)
	}

	packet := &protocol.RPTLPacket{}
	err = packet.Parse(buffer[:n])
	if err != nil {
		t.Fatalf("Failed to parse RPTL packet: %v", err)
	}

	if packet.RepeaterID != 312000 {
		t.Errorf("Expected repeater ID 312000, got %d", packet.RepeaterID)
	}

	// Send RPTACK response
	ackPacket := &protocol.RPTACKPacket{RepeaterID: 312000}
	ackData, _ := ackPacket.Encode()
	serverConn.WriteToUDP(ackData, clientAddr)

	// Wait for RPTK (key exchange)
	serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, _, err = serverConn.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Failed to receive RPTK: %v", err)
	}

	if n >= protocol.RPTKPacketSize && string(buffer[0:4]) == "RPTK" {
		// Send RPTACK for RPTK
		serverConn.WriteToUDP(ackData, clientAddr)
	}

	// Wait for RPTC (configuration)
	serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, _, err = serverConn.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Failed to receive RPTC: %v", err)
	}

	if n >= protocol.RPTCPacketSize && string(buffer[0:4]) == "RPTC" {
		// Send RPTACK for RPTC
		serverConn.WriteToUDP(ackData, clientAddr)
	}

	// Give client time to process final ACK
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop client
	cancel()

	// Wait for client to finish
	select {
	case err := <-errChan:
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Logf("Client error (expected): %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Test timeout")
	}
}

func TestClient_SendDMRD(t *testing.T) {
	// Create a mock UDP server
	serverAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer serverConn.Close()

	actualServerPort := serverConn.LocalAddr().(*net.UDPAddr).Port

	// Create client
	cfg := config.SystemConfig{
		Mode:       "PEER",
		MasterIP:   "127.0.0.1",
		MasterPort: actualServerPort,
		Port:       0,
		RadioID:    312000,
		Passphrase: "test",
	}

	log := logger.New(logger.Config{Level: "debug"})
	client := NewClient(cfg, log)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Helper to handle auth sequence
	go func() {
		buffer := make([]byte, 1024)
		ackPacket := &protocol.RPTACKPacket{RepeaterID: 312000}
		ackData, _ := ackPacket.Encode()

		// Handle RPTL
		serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		if _, addr, err := serverConn.ReadFromUDP(buffer); err == nil && string(buffer[0:4]) == "RPTL" {
			serverConn.WriteToUDP(ackData, addr)
		}

		// Handle RPTK
		serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		if n, addr, err := serverConn.ReadFromUDP(buffer); err == nil && n >= 4 && string(buffer[0:4]) == "RPTK" {
			serverConn.WriteToUDP(ackData, addr)
			_ = n // Mark as used
		}

		// Handle RPTC
		serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		if n, addr, err := serverConn.ReadFromUDP(buffer); err == nil && n >= 4 && string(buffer[0:4]) == "RPTC" {
			serverConn.WriteToUDP(ackData, addr)
			_ = n // Mark as used
		}
	}()

	// Start client
	go client.Start(ctx)
	time.Sleep(500 * time.Millisecond)

	// Create DMRD packet to send
	dmrdPacket := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    312000,
		Timeslot:      1,
		CallType:      0,
		StreamID:      12345,
		Payload:       make([]byte, 33),
	}

	// Send packet through client
	err = client.SendDMRD(dmrdPacket)
	if err != nil {
		t.Fatalf("Failed to send DMRD packet: %v", err)
	}

	// Verify server receives it
	buffer := make([]byte, 1024)
	serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))

	// Read packets until we get a DMRD (skip RPTL, RPTK, RPTC, RPTPING)
	for i := 0; i < 10; i++ {
		n, _, err := serverConn.ReadFromUDP(buffer)
		if err != nil {
			t.Fatalf("Mock server failed to receive packet: %v", err)
		}

		if n >= protocol.DMRDPacketSize && string(buffer[0:4]) == "DMRD" {
			// Found DMRD packet
			receivedPacket := &protocol.DMRDPacket{}
			err = receivedPacket.Parse(buffer[:n])
			if err != nil {
				t.Fatalf("Failed to parse DMRD packet: %v", err)
			}

			if receivedPacket.SourceID != dmrdPacket.SourceID {
				t.Errorf("SourceID mismatch: got %d, want %d", receivedPacket.SourceID, dmrdPacket.SourceID)
			}
			if receivedPacket.DestinationID != dmrdPacket.DestinationID {
				t.Errorf("DestinationID mismatch: got %d, want %d", receivedPacket.DestinationID, dmrdPacket.DestinationID)
			}
			return // Test passed
		}
	}

	t.Fatal("Did not receive DMRD packet from client")
}

func TestClient_ReceiveDMRD(t *testing.T) {
	// Create a mock UDP server
	serverAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer serverConn.Close()

	actualServerPort := serverConn.LocalAddr().(*net.UDPAddr).Port

	// Create client with packet handler
	cfg := config.SystemConfig{
		Mode:       "PEER",
		MasterIP:   "127.0.0.1",
		MasterPort: actualServerPort,
		Port:       0,
		RadioID:    312000,
		Passphrase: "test",
	}

	log := logger.New(logger.Config{Level: "debug"})
	client := NewClient(cfg, log)

	// Set up packet receiver
	receivedPackets := make(chan *protocol.DMRDPacket, 1)
	client.OnDMRD(func(packet *protocol.DMRDPacket) {
		receivedPackets <- packet
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Helper to handle auth sequence
	var clientAddr *net.UDPAddr
	go func() {
		buffer := make([]byte, 1024)
		ackPacket := &protocol.RPTACKPacket{RepeaterID: 312000}
		ackData, _ := ackPacket.Encode()

		// Handle RPTL
		serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		if _, addr, err := serverConn.ReadFromUDP(buffer); err == nil && string(buffer[0:4]) == "RPTL" {
			clientAddr = addr
			serverConn.WriteToUDP(ackData, addr)
		}

		// Handle RPTK
		serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		if n, addr, err := serverConn.ReadFromUDP(buffer); err == nil && n >= 4 && string(buffer[0:4]) == "RPTK" {
			serverConn.WriteToUDP(ackData, addr)
			_ = n // Mark as used
		}

		// Handle RPTC
		serverConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		if n, addr, err := serverConn.ReadFromUDP(buffer); err == nil && n >= 4 && string(buffer[0:4]) == "RPTC" {
			serverConn.WriteToUDP(ackData, addr)
			_ = n // Mark as used
		}
	}()

	// Start client
	go client.Start(ctx)
	time.Sleep(600 * time.Millisecond)

	if clientAddr == nil {
		t.Fatal("Failed to get client address from auth sequence")
	}

	// Send DMRD packet from server to client
	dmrdPacket := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      3120002,
		DestinationID: 3100,
		RepeaterID:    312001,
		Timeslot:      1,
		CallType:      0,
		StreamID:      54321,
		Payload:       make([]byte, 33),
	}

	dmrdData, _ := dmrdPacket.Encode()
	serverConn.WriteToUDP(dmrdData, clientAddr)

	// Wait for client to receive and process
	select {
	case received := <-receivedPackets:
		if received.SourceID != dmrdPacket.SourceID {
			t.Errorf("SourceID mismatch: got %d, want %d", received.SourceID, dmrdPacket.SourceID)
		}
		if received.StreamID != dmrdPacket.StreamID {
			t.Errorf("StreamID mismatch: got %d, want %d", received.StreamID, dmrdPacket.StreamID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for DMRD packet")
	}
}
