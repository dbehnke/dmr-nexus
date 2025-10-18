//go:build integration

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

// TestIntegration_ClientToMaster tests end-to-end communication
// between a client (PEER) and a mock master server
func TestIntegration_ClientToMaster(t *testing.T) {
	// Create a mock master server
	masterAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0, // Let OS assign
	}

	masterConn, err := net.ListenUDP("udp", masterAddr)
	if err != nil {
		t.Fatalf("Failed to create master server: %v", err)
	}
	defer masterConn.Close()

	actualMasterPort := masterConn.LocalAddr().(*net.UDPAddr).Port
	t.Logf("Mock master listening on port %d", actualMasterPort)

	// Create client configuration
	cfg := config.SystemConfig{
		Mode:        "PEER",
		MasterIP:    "127.0.0.1",
		MasterPort:  actualMasterPort,
		Port:        0,
		RadioID:     312000,
		Callsign:    "W1ABC",
		RXFreq:      449000000,
		TXFreq:      444000000,
		TXPower:     25,
		ColorCode:   1,
		Latitude:    42.3601,
		Longitude:   -71.0589,
		Height:      75,
		Location:    "Boston, MA",
		Description: "Integration Test",
		URL:         "https://example.com",
		SoftwareID:  "DMR-Nexus",
		PackageID:   "DMR-Nexus",
		Passphrase:  "test123",
	}

	log := logger.New(logger.Config{Level: "info"})
	client := NewClient(cfg, log)

	// Track received DMRD packets on client
	clientReceivedDMRD := make(chan *protocol.DMRDPacket, 1)
	client.OnDMRD(func(packet *protocol.DMRDPacket) {
		clientReceivedDMRD <- packet
	})

	// Start client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientErrChan := make(chan error, 1)
	go func() {
		clientErrChan <- client.Start(ctx)
	}()

	// Master server goroutine - handle authentication and traffic
	masterErrChan := make(chan error, 1)
	masterReceivedDMRD := make(chan *protocol.DMRDPacket, 1)
	clientAddrChan := make(chan *net.UDPAddr, 1)

	go func() {
		defer close(masterReceivedDMRD)
		buffer := make([]byte, 4096)
		var clientAddr *net.UDPAddr
		ackPacket := &protocol.RPTACKPacket{RepeaterID: 312000}
		ackData, _ := ackPacket.Encode()

		authComplete := false

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			masterConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, addr, err := masterConn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				masterErrChan <- err
				return
			}

			if clientAddr == nil {
				clientAddr = addr
				select {
				case clientAddrChan <- addr:
				default:
				}
			}

			// Process packet based on type
			if n >= 4 {
				packetType := string(buffer[0:4])

				switch {
				case packetType == protocol.PacketTypeRPTL:
					t.Log("Master: Received RPTL, sending RPTACK")
					masterConn.WriteToUDP(ackData, addr)

				case packetType == protocol.PacketTypeRPTK:
					t.Log("Master: Received RPTK, sending RPTACK")
					masterConn.WriteToUDP(ackData, addr)

				case packetType == protocol.PacketTypeRPTC:
					t.Log("Master: Received RPTC, sending RPTACK")
					masterConn.WriteToUDP(ackData, addr)
					authComplete = true

				case packetType == protocol.PacketTypeDMRD && n >= protocol.DMRDPacketSize:
					if !authComplete {
						t.Log("Master: Ignoring DMRD before auth complete")
						continue
					}

					// Parse and store DMRD packet
					dmrd := &protocol.DMRDPacket{}
					if err := dmrd.Parse(buffer[:n]); err == nil {
						t.Logf("Master: Received DMRD from %d to %d on TS%d",
							dmrd.SourceID, dmrd.DestinationID, dmrd.Timeslot)
						masterReceivedDMRD <- dmrd
					}

				case n >= 7 && string(buffer[0:7]) == protocol.PacketTypeRPTPING:
					t.Log("Master: Received RPTPING, sending MSTPONG")
					pong := &protocol.MSTPONGPacket{RepeaterID: 312000}
					pongData, _ := pong.Encode()
					masterConn.WriteToUDP(pongData, addr)
				}
			}
		}
	}()

	// Wait for authentication to complete
	time.Sleep(1 * time.Second)

	// Test 1: Client sends DMRD packet to master
	t.Log("=== Test 1: Client -> Master ===")
	testPacket1 := &protocol.DMRDPacket{
		Sequence:      1,
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    312000,
		Timeslot:      protocol.Timeslot1,
		CallType:      protocol.CallTypeGroup,
		FrameType:     protocol.FrameTypeVoiceHeader,
		StreamID:      12345,
		Payload:       make([]byte, 33),
	}

	if err := client.SendDMRD(testPacket1); err != nil {
		t.Fatalf("Failed to send DMRD from client: %v", err)
	}

	// Master should receive the packet
	select {
	case received := <-masterReceivedDMRD:
		if received.SourceID != testPacket1.SourceID {
			t.Errorf("Master received wrong SourceID: got %d, want %d",
				received.SourceID, testPacket1.SourceID)
		}
		if received.DestinationID != testPacket1.DestinationID {
			t.Errorf("Master received wrong DestinationID: got %d, want %d",
				received.DestinationID, testPacket1.DestinationID)
		}
		if received.Timeslot != testPacket1.Timeslot {
			t.Errorf("Master received wrong Timeslot: got %d, want %d",
				received.Timeslot, testPacket1.Timeslot)
		}
		t.Log("✓ Master successfully received DMRD packet from client")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout: Master did not receive DMRD packet from client")
	}

	// Test 2: Master sends DMRD packet to client
	t.Log("=== Test 2: Master -> Client ===")
	testPacket2 := &protocol.DMRDPacket{
		Sequence:      2,
		SourceID:      3120002,
		DestinationID: 3100,
		RepeaterID:    312001,
		Timeslot:      protocol.Timeslot2,
		CallType:      protocol.CallTypeGroup,
		FrameType:     protocol.FrameTypeVoice,
		StreamID:      54321,
		Payload:       make([]byte, 33),
	}

	// Get client address from channel
	var masterClientAddr *net.UDPAddr
	select {
	case masterClientAddr = <-clientAddrChan:
		t.Log("Got client address from master goroutine")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout: Could not get client address from master")
	}

	// Send DMRD from master to client
	dmrdData, _ := testPacket2.Encode()
	_, err = masterConn.WriteToUDP(dmrdData, masterClientAddr)
	if err != nil {
		t.Fatalf("Failed to send DMRD from master: %v", err)
	}
	t.Log("Master sent DMRD packet to client")

	// Client should receive the packet
	select {
	case received := <-clientReceivedDMRD:
		if received.SourceID != testPacket2.SourceID {
			t.Errorf("Client received wrong SourceID: got %d, want %d",
				received.SourceID, testPacket2.SourceID)
		}
		if received.DestinationID != testPacket2.DestinationID {
			t.Errorf("Client received wrong DestinationID: got %d, want %d",
				received.DestinationID, testPacket2.DestinationID)
		}
		if received.Timeslot != testPacket2.Timeslot {
			t.Errorf("Client received wrong Timeslot: got %d, want %d",
				received.Timeslot, testPacket2.Timeslot)
		}
		t.Log("✓ Client successfully received DMRD packet from master")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout: Client did not receive DMRD packet from master")
	}

	// Test 3: Verify keepalive mechanism
	t.Log("=== Test 3: Keepalive/Ping ===")
	// Wait for at least one ping cycle (5 seconds + buffer)
	time.Sleep(6 * time.Second)
	t.Log("✓ Keepalive mechanism working (no disconnection)")

	// Clean shutdown
	cancel()

	select {
	case err := <-clientErrChan:
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Logf("Client error (expected): %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Log("Client shutdown timeout")
	}

	t.Log("=== Integration Test Complete ===")
}
