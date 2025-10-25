package network

import (
	"context"
	"net"
	"sync"
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
	defer func() {
		if err := serverConn.Close(); err != nil {
			t.Logf("serverConn.Close error: %v", err)
		}
	}()

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
	if err := serverConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Logf("SetReadDeadline error: %v", err)
		return
	}
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

	// Send RPTACK response with salt
	salt := []byte{0x01, 0x02, 0x03, 0x04}
	ackPacket := &protocol.RPTACKPacket{
		RepeaterID: 312000,
		Salt:       salt,
	}
	ackData, _ := ackPacket.Encode()
	if _, err := serverConn.WriteToUDP(ackData, clientAddr); err != nil {
		t.Fatalf("WriteToUDP error: %v", err)
	}

	// Wait for RPTK (key exchange)
	if err := serverConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Logf("SetReadDeadline error: %v", err)
		return
	}
	n, _, err = serverConn.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Failed to receive RPTK: %v", err)
	}

	if n >= protocol.RPTKPacketSize && string(buffer[0:4]) == "RPTK" {
		// Send RPTACK for RPTK
		if _, err := serverConn.WriteToUDP(ackData, clientAddr); err != nil {
			t.Fatalf("WriteToUDP error: %v", err)
		}
	}

	// Wait for RPTC (configuration)
	if err := serverConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Logf("SetReadDeadline error: %v", err)
		if _, addr, err := serverConn.ReadFromUDP(buffer); err == nil && string(buffer[0:4]) == "RPTL" {
			if _, err := serverConn.WriteToUDP(ackData, addr); err != nil {
				t.Logf("WriteToUDP error: %v", err)
				return
			}
		}
	}
	n, _, err = serverConn.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("Failed to receive RPTC: %v", err)
	}

	if n >= protocol.RPTCPacketSize && string(buffer[0:4]) == "RPTC" {
		// Send RPTACK for RPTC
		if _, err := serverConn.WriteToUDP(ackData, clientAddr); err != nil {
			t.Fatalf("WriteToUDP error: %v", err)
		}
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
	defer func() {
		if err := serverConn.Close(); err != nil {
			t.Logf("serverConn.Close error: %v", err)
		}
	}()

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
		salt := []byte{0x01, 0x02, 0x03, 0x04}
		ackPacketWithSalt := &protocol.RPTACKPacket{RepeaterID: 312000, Salt: salt}
		ackDataWithSalt, _ := ackPacketWithSalt.Encode()
		ackPacket := &protocol.RPTACKPacket{RepeaterID: 312000}
		ackData, _ := ackPacket.Encode()

		// Handle RPTL - send RPTACK with salt
		if err := serverConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			t.Logf("SetReadDeadline error: %v", err)
			return
		}
		if _, addr, err := serverConn.ReadFromUDP(buffer); err == nil && string(buffer[0:4]) == "RPTL" {
			if _, err := serverConn.WriteToUDP(ackDataWithSalt, addr); err != nil {
				t.Logf("WriteToUDP error: %v", err)
				return
			}
		}

		// Handle RPTK
		if err := serverConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			t.Logf("SetReadDeadline error: %v", err)
			return
		}
		if n, addr, err := serverConn.ReadFromUDP(buffer); err == nil && n >= 4 && string(buffer[0:4]) == "RPTK" {
			if _, err := serverConn.WriteToUDP(ackData, addr); err != nil {
				t.Logf("WriteToUDP error: %v", err)
				return
			}
			_ = n // Mark as used
		}

		// Handle RPTC
		if err := serverConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			t.Logf("SetReadDeadline error: %v", err)
			return
		}
		if n, addr, err := serverConn.ReadFromUDP(buffer); err == nil && n >= 4 && string(buffer[0:4]) == "RPTC" {
			if _, err := serverConn.WriteToUDP(ackData, addr); err != nil {
				t.Logf("WriteToUDP error: %v", err)
				return
			}
			_ = n // Mark as used
		}
	}()

	// Start client
	go func() {
		if err := client.Start(ctx); err != nil {
			t.Logf("client.Start error: %v", err)
		}
	}()
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
	if err := serverConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		t.Logf("SetReadDeadline error: %v", err)
	}

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
	defer func() {
		if err := serverConn.Close(); err != nil {
			t.Logf("serverConn.Close error: %v", err)
		}
	}()

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

	// Helper to handle auth sequence and emit client addr when done
	clientAddrCh := make(chan *net.UDPAddr, 1)
	go func() {
		buffer := make([]byte, 1024)
		salt := []byte{0x01, 0x02, 0x03, 0x04}
		ackPacketWithSalt := &protocol.RPTACKPacket{RepeaterID: 312000, Salt: salt}
		ackDataWithSalt, _ := ackPacketWithSalt.Encode()
		ackPacket := &protocol.RPTACKPacket{RepeaterID: 312000}
		ackData, _ := ackPacket.Encode()
		var finalAddr *net.UDPAddr

		// Handle RPTL - send RPTACK with salt
		if err := serverConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			t.Logf("SetReadDeadline error: %v", err)
			return
		}
		if _, addr, err := serverConn.ReadFromUDP(buffer); err == nil && string(buffer[0:4]) == "RPTL" {
			finalAddr = addr
			if _, err := serverConn.WriteToUDP(ackDataWithSalt, addr); err != nil {
				t.Logf("WriteToUDP error: %v", err)
				return
			}
		}

		// Handle RPTK
		if err := serverConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			t.Logf("SetReadDeadline error: %v", err)
			return
		}
		if n, addr, err := serverConn.ReadFromUDP(buffer); err == nil && n >= 4 && string(buffer[0:4]) == "RPTK" {
			if _, err := serverConn.WriteToUDP(ackData, addr); err != nil {
				t.Logf("WriteToUDP error: %v", err)
				return
			}
			_ = n // Mark as used
		}

		// Handle RPTC
		if err := serverConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			t.Logf("SetReadDeadline error: %v", err)
			return
		}
		if n, addr, err := serverConn.ReadFromUDP(buffer); err == nil && n >= 4 && string(buffer[0:4]) == "RPTC" {
			if _, err := serverConn.WriteToUDP(ackData, addr); err != nil {
				t.Logf("WriteToUDP error: %v", err)
				return
			}
			_ = n // Mark as used
			// Handshake complete - emit client addr
			if finalAddr != nil {
				clientAddrCh <- finalAddr
			} else {
				clientAddrCh <- addr
			}
		}
	}()

	// Start client
	errChan := make(chan error, 1)
	go func() {
		errChan <- client.Start(ctx)
	}()
	// Wait for handshake to complete and get the client address
	var clientAddr *net.UDPAddr
	select {
	case clientAddr = <-clientAddrCh:
		// proceed
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for client address after handshake")
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
	if _, err := serverConn.WriteToUDP(dmrdData, clientAddr); err != nil {
		t.Logf("WriteToUDP error: %v", err)
	}

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

	// Teardown client cleanly
	cancel()
	<-errChan
}

func TestClient_Race(t *testing.T) {
	// Create a mock UDP server
	serverAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0, // Let OS assign port
	}
	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer func() {
		if err := serverConn.Close(); err != nil {
			t.Logf("serverConn.Close error: %v", err)
		}
	}()
	actualServerPort := serverConn.LocalAddr().(*net.UDPAddr).Port

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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Mock server handles authentication handshake
	go func() {
		buffer := make([]byte, 1024)
		salt := []byte{0x01, 0x02, 0x03, 0x04}
		ackPacketWithSalt := &protocol.RPTACKPacket{RepeaterID: 312000, Salt: salt}
		ackDataWithSalt, _ := ackPacketWithSalt.Encode()
		ackPacket := &protocol.RPTACKPacket{RepeaterID: 312000}
		ackData, _ := ackPacket.Encode()
		for step := 0; step < 3; step++ {
			if err := serverConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
				t.Logf("SetReadDeadline error: %v", err)
				return
			}
			n, addr, err := serverConn.ReadFromUDP(buffer)
			if err != nil {
				t.Logf("ReadFromUDP error: %v", err)
				return
			}
			if n >= 4 {
				// Send RPTACK with salt for RPTL, without salt for RPTK and RPTC
				if string(buffer[0:4]) == "RPTL" {
					if _, err := serverConn.WriteToUDP(ackDataWithSalt, addr); err != nil {
						t.Logf("WriteToUDP error: %v", err)
						return
					}
				} else if string(buffer[0:4]) == "RPTK" || string(buffer[0:4]) == "RPTC" {
					if _, err := serverConn.WriteToUDP(ackData, addr); err != nil {
						t.Logf("WriteToUDP error: %v", err)
						return
					}
				}
			}
		}
	}()

	// Start client in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- client.Start(ctx)
	}()

	// Wait for client to connect
	time.Sleep(500 * time.Millisecond)

	// Simulate concurrent SendDMRD and handler registration
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			packet := &protocol.DMRDPacket{
				Sequence:      byte(idx),
				SourceID:      3120000 + uint32(idx),
				DestinationID: 3100,
				RepeaterID:    312000,
				Timeslot:      1,
				CallType:      0,
				StreamID:      12345 + uint32(idx),
				Payload:       make([]byte, 33),
			}
			_ = client.SendDMRD(packet)
		}(i)
		go func(idx int) {
			defer wg.Done()
			client.OnDMRD(func(packet *protocol.DMRDPacket) {
				// No-op handler
			})
		}(i)
	}
	wg.Wait()
	cancel()
	select {
	case <-errChan:
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for client shutdown")
	}
}
