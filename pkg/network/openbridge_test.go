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

func TestOpenBridgeClient_New(t *testing.T) {
	log := logger.New(logger.Config{Level: "info"})
	cfg := config.SystemConfig{
		Mode:       "OPENBRIDGE",
		Port:       62035,
		TargetIP:   "127.0.0.1",
		TargetPort: 62036,
		NetworkID:  3129999,
		Passphrase: "password",
		BothSlots:  false,
	}

	client := NewOpenBridgeClient(cfg, log)
	if client == nil {
		t.Fatal("NewOpenBridgeClient() returned nil")
	}

	if client.config.NetworkID != 3129999 {
		t.Errorf("Expected network ID 3129999, got %d", client.config.NetworkID)
	}
}

func TestOpenBridgeClient_SendDMRD(t *testing.T) {
	log := logger.New(logger.Config{Level: "debug"})

	// Create a mock server
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to resolve server address: %v", err)
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		t.Fatalf("Failed to create server connection: %v", err)
	}
	defer serverConn.Close()

	serverPort := serverConn.LocalAddr().(*net.UDPAddr).Port

	// Create OpenBridge client
	cfg := config.SystemConfig{
		Mode:       "OPENBRIDGE",
		Port:       0, // Let OS assign port
		TargetIP:   "127.0.0.1",
		TargetPort: serverPort,
		NetworkID:  3129999,
		Passphrase: "password",
		BothSlots:  false,
	}

	client := NewOpenBridgeClient(cfg, log)

	// Start client in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientReady := make(chan error, 1)
	go func() {
		clientReady <- client.Start(ctx)
	}()

	// Wait for client to be ready
	time.Sleep(100 * time.Millisecond)

	// Create a DMRD packet
	packet := &protocol.DMRDPacket{
		Sequence:      0x01,
		SourceID:      3120001,
		DestinationID: 91, // Brandmeister worldwide
		RepeaterID:    uint32(cfg.NetworkID),
		Timeslot:      protocol.Timeslot1,
		CallType:      protocol.CallTypeGroup,
		FrameType:     protocol.FrameTypeVoice,
		DataType:      0x00,
		StreamID:      12345,
		Payload:       make([]byte, 33),
	}

	// Send packet
	err = client.SendDMRD(packet)
	if err != nil {
		t.Fatalf("SendDMRD() failed: %v", err)
	}

	// Receive on server side
	buf := make([]byte, 1024)
	serverConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := serverConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("Failed to receive packet: %v", err)
	}

	// Should be OpenBridge size
	if n != protocol.DMRDOpenBridgePacketSize {
		t.Errorf("Expected packet size %d, got %d", protocol.DMRDOpenBridgePacketSize, n)
	}

	// Parse received packet
	received := &protocol.DMRDPacket{}
	err = received.Parse(buf[:n])
	if err != nil {
		t.Fatalf("Failed to parse received packet: %v", err)
	}

	// Verify HMAC
	if !received.VerifyOpenBridgeHMAC(cfg.Passphrase) {
		t.Error("HMAC verification failed")
	}

	// Verify packet fields
	if received.SourceID != packet.SourceID {
		t.Errorf("Source ID mismatch: got %d, want %d", received.SourceID, packet.SourceID)
	}
	if received.DestinationID != packet.DestinationID {
		t.Errorf("Destination ID mismatch: got %d, want %d", received.DestinationID, packet.DestinationID)
	}
	if received.RepeaterID != uint32(cfg.NetworkID) {
		t.Errorf("Network ID mismatch: got %d, want %d", received.RepeaterID, cfg.NetworkID)
	}

	cancel()
}

func TestOpenBridgeClient_ReceiveDMRD(t *testing.T) {
	log := logger.New(logger.Config{Level: "debug"})

	// Create OpenBridge client
	cfg := config.SystemConfig{
		Mode:       "OPENBRIDGE",
		Port:       0, // Let OS assign port
		TargetIP:   "127.0.0.1",
		TargetPort: 62037, // Doesn't matter for this test
		NetworkID:  3129999,
		Passphrase: "password",
		BothSlots:  false,
	}

	client := NewOpenBridgeClient(cfg, log)

	// Set up packet handler
	receivedPacket := make(chan *protocol.DMRDPacket, 1)
	client.SetDMRDHandler(func(p *protocol.DMRDPacket) {
		receivedPacket <- p
	})

	// Start client
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		client.Start(ctx)
	}()

	// Wait for client to be ready
	time.Sleep(100 * time.Millisecond)

	// Get client's actual port
	clientAddr := client.conn.LocalAddr().(*net.UDPAddr)

	// Create a mock sender
	senderConn, err := net.DialUDP("udp", nil, clientAddr)
	if err != nil {
		t.Fatalf("Failed to create sender connection: %v", err)
	}
	defer senderConn.Close()

	// Create and send a DMRD packet with HMAC
	packet := &protocol.DMRDPacket{
		Sequence:      0x42,
		SourceID:      3120001,
		DestinationID: 91,
		RepeaterID:    uint32(cfg.NetworkID),
		Timeslot:      protocol.Timeslot1,
		CallType:      protocol.CallTypeGroup,
		FrameType:     protocol.FrameTypeVoice,
		DataType:      0x00,
		StreamID:      54321,
		Payload:       make([]byte, 33),
	}

	// Add HMAC
	err = packet.AddOpenBridgeHMAC(cfg.Passphrase)
	if err != nil {
		t.Fatalf("AddOpenBridgeHMAC() failed: %v", err)
	}

	// Encode and send
	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	_, err = senderConn.Write(data)
	if err != nil {
		t.Fatalf("Failed to send packet: %v", err)
	}

	// Wait for packet to be received
	select {
	case received := <-receivedPacket:
		// Verify packet fields
		if received.Sequence != packet.Sequence {
			t.Errorf("Sequence mismatch: got %d, want %d", received.Sequence, packet.Sequence)
		}
		if received.SourceID != packet.SourceID {
			t.Errorf("Source ID mismatch: got %d, want %d", received.SourceID, packet.SourceID)
		}
		if received.StreamID != packet.StreamID {
			t.Errorf("Stream ID mismatch: got %d, want %d", received.StreamID, packet.StreamID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for packet")
	}

	cancel()
}

func TestOpenBridgeClient_RejectInvalidHMAC(t *testing.T) {
	log := logger.New(logger.Config{Level: "debug"})

	cfg := config.SystemConfig{
		Mode:       "OPENBRIDGE",
		Port:       0,
		TargetIP:   "127.0.0.1",
		TargetPort: 62038,
		NetworkID:  3129999,
		Passphrase: "password",
		BothSlots:  false,
	}

	client := NewOpenBridgeClient(cfg, log)

	// Set up packet handler
	receivedPacket := make(chan *protocol.DMRDPacket, 1)
	client.SetDMRDHandler(func(p *protocol.DMRDPacket) {
		receivedPacket <- p
	})

	// Start client
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		client.Start(ctx)
	}()

	// Wait for client to be ready
	time.Sleep(100 * time.Millisecond)

	// Get client's actual port
	clientAddr := client.conn.LocalAddr().(*net.UDPAddr)

	// Create a mock sender
	senderConn, err := net.DialUDP("udp", nil, clientAddr)
	if err != nil {
		t.Fatalf("Failed to create sender connection: %v", err)
	}
	defer senderConn.Close()

	// Create packet with wrong passphrase
	packet := &protocol.DMRDPacket{
		Sequence:      0x42,
		SourceID:      3120001,
		DestinationID: 91,
		RepeaterID:    uint32(cfg.NetworkID),
		Timeslot:      protocol.Timeslot1,
		CallType:      protocol.CallTypeGroup,
		FrameType:     protocol.FrameTypeVoice,
		DataType:      0x00,
		StreamID:      54321,
		Payload:       make([]byte, 33),
	}

	// Add HMAC with WRONG passphrase
	err = packet.AddOpenBridgeHMAC("wrongpassword")
	if err != nil {
		t.Fatalf("AddOpenBridgeHMAC() failed: %v", err)
	}

	// Encode and send
	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	_, err = senderConn.Write(data)
	if err != nil {
		t.Fatalf("Failed to send packet: %v", err)
	}

	// Packet should be rejected (not received)
	select {
	case <-receivedPacket:
		t.Fatal("Packet with invalid HMAC should have been rejected")
	case <-time.After(1 * time.Second):
		// Expected - packet was rejected
	}

	cancel()
}

func TestOpenBridgeClient_BothSlots(t *testing.T) {
	log := logger.New(logger.Config{Level: "debug"})

	tests := []struct {
		name      string
		bothSlots bool
		timeslot  int
		callType  int
		shouldSend bool
	}{
		{"TS1 with both_slots=false", false, protocol.Timeslot1, protocol.CallTypeGroup, true},
		{"TS2 with both_slots=false", false, protocol.Timeslot2, protocol.CallTypeGroup, false},
		{"TS1 with both_slots=true", true, protocol.Timeslot1, protocol.CallTypeGroup, true},
		{"TS2 with both_slots=true", true, protocol.Timeslot2, protocol.CallTypeGroup, true},
		{"TS2 private with both_slots=false", false, protocol.Timeslot2, protocol.CallTypePrivate, true},
		{"TS2 private with both_slots=true", true, protocol.Timeslot2, protocol.CallTypePrivate, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("Failed to resolve server address: %v", err)
			}

			serverConn, err := net.ListenUDP("udp", serverAddr)
			if err != nil {
				t.Fatalf("Failed to create server connection: %v", err)
			}
			defer serverConn.Close()

			serverPort := serverConn.LocalAddr().(*net.UDPAddr).Port

			// Create OpenBridge client
			cfg := config.SystemConfig{
				Mode:       "OPENBRIDGE",
				Port:       0,
				TargetIP:   "127.0.0.1",
				TargetPort: serverPort,
				NetworkID:  3129999,
				Passphrase: "password",
				BothSlots:  tt.bothSlots,
			}

			client := NewOpenBridgeClient(cfg, log)

			// Start client
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			go func() {
				client.Start(ctx)
			}()

			time.Sleep(100 * time.Millisecond)

			// Create packet
			packet := &protocol.DMRDPacket{
				Sequence:      0x01,
				SourceID:      3120001,
				DestinationID: 91,
				RepeaterID:    uint32(cfg.NetworkID),
				Timeslot:      tt.timeslot,
				CallType:      tt.callType,
				FrameType:     protocol.FrameTypeVoice,
				DataType:      0x00,
				StreamID:      12345,
				Payload:       make([]byte, 33),
			}

			// Send packet
			err = client.SendDMRD(packet)
			if err != nil && tt.shouldSend {
				t.Fatalf("SendDMRD() failed: %v", err)
			}

			// Try to receive
			buf := make([]byte, 1024)
			serverConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			_, _, err = serverConn.ReadFromUDP(buf)

			if tt.shouldSend && err != nil {
				t.Errorf("Expected to receive packet but got error: %v", err)
			}
			if !tt.shouldSend && err == nil {
				t.Error("Expected packet to be filtered but it was sent")
			}

			cancel()
		})
	}
}
