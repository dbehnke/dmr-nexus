package protocol

import (
	"bytes"
	"testing"
)

func TestDMRDPacket_Parse(t *testing.T) {
	// Create a sample DMRD packet
	data := make([]byte, DMRDPacketSize)
	copy(data[0:4], []byte("DMRD")) // Signature
	data[4] = 0x01                  // Sequence
	data[5] = 0x31
	data[6] = 0x20
	data[7] = 0x01 // Source ID: 3219457
	data[8] = 0x00
	data[9] = 0x0C
	data[10] = 0x1C // Destination ID: 3100
	// Repeater ID: 312000 (0x0004C2C0 in hex)
	data[11] = 0x00
	data[12] = 0x04
	data[13] = 0xC2
	data[14] = 0xC0
	data[15] = 0x00 // Slot byte (TS1, group call)
	data[16] = 0x00
	data[17] = 0x00
	data[18] = 0x00
	data[19] = 0x01 // Stream ID
	// Payload bytes 20-52 (33 bytes of voice data - leave as zeros for test)

	packet := &DMRDPacket{}
	err := packet.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse DMRD packet: %v", err)
	}

	// Verify parsed values
	if packet.Sequence != 0x01 {
		t.Errorf("Expected sequence 0x01, got 0x%02X", packet.Sequence)
	}

	if packet.SourceID != 3219457 {
		t.Errorf("Expected source ID 3219457, got %d", packet.SourceID)
	}

	if packet.DestinationID != 3100 {
		t.Errorf("Expected destination ID 3100, got %d", packet.DestinationID)
	}

	if packet.RepeaterID != 312000 {
		t.Errorf("Expected repeater ID 312000, got %d", packet.RepeaterID)
	}

	if packet.Timeslot != Timeslot1 {
		t.Errorf("Expected timeslot 1, got %d", packet.Timeslot)
	}

	if packet.CallType != CallTypeGroup {
		t.Errorf("Expected group call type, got %d", packet.CallType)
	}

	if packet.StreamID != 1 {
		t.Errorf("Expected stream ID 1, got %d", packet.StreamID)
	}

	if len(packet.Payload) != 33 {
		t.Errorf("Expected payload length 33, got %d", len(packet.Payload))
	}
}

func TestDMRDPacket_Parse_InvalidSize(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"Too small", 10},
		{"Too large (not OpenBridge)", 60},
		{"Empty", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.size)
			packet := &DMRDPacket{}
			err := packet.Parse(data)
			if err == nil {
				t.Error("Expected error for invalid packet size")
			}
		})
	}
}

func TestDMRDPacket_Parse_InvalidSignature(t *testing.T) {
	data := make([]byte, DMRDPacketSize)
	copy(data[0:4], []byte("XXXX")) // Invalid signature

	packet := &DMRDPacket{}
	err := packet.Parse(data)
	if err == nil {
		t.Error("Expected error for invalid signature")
	}
}

func TestDMRDPacket_Encode(t *testing.T) {
	packet := &DMRDPacket{
		Sequence:      0x05,
		SourceID:      3219457,
		DestinationID: 3100,
		RepeaterID:    312000,
		Timeslot:      Timeslot2,
		CallType:      CallTypeGroup,
		FrameType:     FrameTypeVoice,
		StreamID:      12345,
		Payload:       make([]byte, 33),
	}

	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Failed to encode DMRD packet: %v", err)
	}

	if len(data) != DMRDPacketSize {
		t.Errorf("Expected encoded size %d, got %d", DMRDPacketSize, len(data))
	}

	// Verify signature
	if !bytes.Equal(data[0:4], []byte("DMRD")) {
		t.Error("Invalid signature in encoded packet")
	}

	// Verify sequence
	if data[4] != 0x05 {
		t.Errorf("Expected sequence 0x05, got 0x%02X", data[4])
	}

	// Verify timeslot bit (should be set for TS2)
	if data[15]&SlotTimeslotMask == 0 {
		t.Error("Expected timeslot bit to be set for TS2")
	}
}

func TestDMRDPacket_RoundTrip(t *testing.T) {
	// Create original packet
	original := &DMRDPacket{
		Sequence:      0x42,
		SourceID:      1234567,
		DestinationID: 9876,
		RepeaterID:    312999,
		Timeslot:      Timeslot1,
		CallType:      CallTypePrivate,
		FrameType:     FrameTypeVoiceHeader,
		StreamID:      0xABCDEF01,
		Payload:       []byte("test payload data here 123456789!"),
	}

	// Encode
	data, err := original.Encode()
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Parse
	parsed := &DMRDPacket{}
	err = parsed.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Compare
	if parsed.Sequence != original.Sequence {
		t.Errorf("Sequence mismatch: got %d, want %d", parsed.Sequence, original.Sequence)
	}
	if parsed.SourceID != original.SourceID {
		t.Errorf("SourceID mismatch: got %d, want %d", parsed.SourceID, original.SourceID)
	}
	if parsed.DestinationID != original.DestinationID {
		t.Errorf("DestinationID mismatch: got %d, want %d", parsed.DestinationID, original.DestinationID)
	}
	if parsed.RepeaterID != original.RepeaterID {
		t.Errorf("RepeaterID mismatch: got %d, want %d", parsed.RepeaterID, original.RepeaterID)
	}
	if parsed.Timeslot != original.Timeslot {
		t.Errorf("Timeslot mismatch: got %d, want %d", parsed.Timeslot, original.Timeslot)
	}
	if parsed.CallType != original.CallType {
		t.Errorf("CallType mismatch: got %d, want %d", parsed.CallType, original.CallType)
	}
	if parsed.StreamID != original.StreamID {
		t.Errorf("StreamID mismatch: got %d, want %d", parsed.StreamID, original.StreamID)
	}
	if !bytes.Equal(parsed.Payload, original.Payload) {
		t.Error("Payload mismatch")
	}
}

func TestDMRDPacket_Timeslot(t *testing.T) {
	tests := []struct {
		name     string
		timeslot int
		slotByte byte
		expectTS int
	}{
		{"TS1", Timeslot1, 0x00, Timeslot1},
		{"TS2", Timeslot2, 0x80, Timeslot2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, DMRDPacketSize)
			copy(data[0:4], []byte("DMRD"))
			data[15] = tt.slotByte

			packet := &DMRDPacket{}
			err := packet.Parse(data)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if packet.Timeslot != tt.expectTS {
				t.Errorf("Expected timeslot %d, got %d", tt.expectTS, packet.Timeslot)
			}
		})
	}
}

func TestDMRDPacket_CallType(t *testing.T) {
	tests := []struct {
		name       string
		callType   int
		slotByte   byte
		expectType int
	}{
		{"Group call", CallTypeGroup, 0x00, CallTypeGroup},
		{"Private call", CallTypePrivate, 0x40, CallTypePrivate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, DMRDPacketSize)
			copy(data[0:4], []byte("DMRD"))
			data[15] = tt.slotByte

			packet := &DMRDPacket{}
			err := packet.Parse(data)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if packet.CallType != tt.expectType {
				t.Errorf("Expected call type %d, got %d", tt.expectType, packet.CallType)
			}
		})
	}
}
