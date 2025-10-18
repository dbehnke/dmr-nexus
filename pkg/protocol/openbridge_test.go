package protocol

import (
	"bytes"
	"testing"
)

func TestComputeHMAC(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		passphrase string
		wantLen    int
	}{
		{
			name:       "Standard DMRD packet",
			data:       make([]byte, DMRDPacketSize),
			passphrase: "testpass",
			wantLen:    20, // HMAC-SHA1 produces 20 bytes
		},
		{
			name:       "Empty passphrase",
			data:       make([]byte, DMRDPacketSize),
			passphrase: "",
			wantLen:    20,
		},
		{
			name:       "Long passphrase",
			data:       make([]byte, DMRDPacketSize),
			passphrase: "this_is_a_very_long_passphrase_for_testing",
			wantLen:    20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hmac := ComputeHMAC(tt.data, tt.passphrase)
			if len(hmac) != tt.wantLen {
				t.Errorf("ComputeHMAC() returned %d bytes, want %d", len(hmac), tt.wantLen)
			}
		})
	}
}

func TestComputeHMAC_Consistency(t *testing.T) {
	// Test that same input produces same output
	data := make([]byte, DMRDPacketSize)
	copy(data[0:4], []byte("DMRD"))
	passphrase := "testpass"

	hmac1 := ComputeHMAC(data, passphrase)
	hmac2 := ComputeHMAC(data, passphrase)

	if !bytes.Equal(hmac1, hmac2) {
		t.Error("ComputeHMAC() should produce consistent results for same input")
	}
}

func TestComputeHMAC_Uniqueness(t *testing.T) {
	// Test that different inputs produce different outputs
	data1 := make([]byte, DMRDPacketSize)
	data2 := make([]byte, DMRDPacketSize)
	copy(data1[0:4], []byte("DMRD"))
	copy(data2[0:4], []byte("DMRD"))
	data1[4] = 0x01
	data2[4] = 0x02
	passphrase := "testpass"

	hmac1 := ComputeHMAC(data1, passphrase)
	hmac2 := ComputeHMAC(data2, passphrase)

	if bytes.Equal(hmac1, hmac2) {
		t.Error("ComputeHMAC() should produce different results for different input")
	}
}

func TestVerifyHMAC(t *testing.T) {
	data := make([]byte, DMRDPacketSize)
	copy(data[0:4], []byte("DMRD"))
	passphrase := "testpass"

	// Compute valid HMAC
	validHMAC := ComputeHMAC(data, passphrase)

	tests := []struct {
		name       string
		data       []byte
		hmac       []byte
		passphrase string
		want       bool
	}{
		{
			name:       "Valid HMAC",
			data:       data,
			hmac:       validHMAC,
			passphrase: passphrase,
			want:       true,
		},
		{
			name:       "Invalid HMAC",
			data:       data,
			hmac:       make([]byte, 20), // All zeros
			passphrase: passphrase,
			want:       false,
		},
		{
			name:       "Wrong passphrase",
			data:       data,
			hmac:       validHMAC,
			passphrase: "wrongpass",
			want:       false,
		},
		{
			name:       "Modified data",
			data:       append([]byte{}, data...),
			hmac:       validHMAC,
			passphrase: passphrase,
			want:       false,
		},
	}

	// Modify data for the last test
	tests[3].data[4] = 0xFF

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyHMAC(tt.data, tt.hmac, tt.passphrase); got != tt.want {
				t.Errorf("VerifyHMAC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDMRDPacket_AddOpenBridgeHMAC(t *testing.T) {
	packet := &DMRDPacket{
		Sequence:      0x01,
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    312000,
		Timeslot:      Timeslot1,
		CallType:      CallTypeGroup,
		FrameType:     FrameTypeVoice,
		DataType:      0x00,
		StreamID:      1,
		Payload:       make([]byte, 33),
	}

	passphrase := "testpass"

	// Add HMAC
	err := packet.AddOpenBridgeHMAC(passphrase)
	if err != nil {
		t.Fatalf("AddOpenBridgeHMAC() failed: %v", err)
	}

	// Verify HMAC was added
	if len(packet.HMAC) != 20 {
		t.Errorf("Expected HMAC length 20, got %d", len(packet.HMAC))
	}

	// Encode packet
	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	// Should be OpenBridge size
	if len(data) != DMRDOpenBridgePacketSize {
		t.Errorf("Expected packet size %d, got %d", DMRDOpenBridgePacketSize, len(data))
	}

	// Verify HMAC is valid
	if !VerifyHMAC(data[:DMRDPacketSize], packet.HMAC, passphrase) {
		t.Error("HMAC verification failed")
	}
}

func TestDMRDPacket_VerifyOpenBridgeHMAC(t *testing.T) {
	packet := &DMRDPacket{
		Sequence:      0x01,
		SourceID:      3120001,
		DestinationID: 3100,
		RepeaterID:    312000,
		Timeslot:      Timeslot1,
		CallType:      CallTypeGroup,
		FrameType:     FrameTypeVoice,
		DataType:      0x00,
		StreamID:      1,
		Payload:       make([]byte, 33),
	}

	passphrase := "testpass"

	// Add HMAC
	err := packet.AddOpenBridgeHMAC(passphrase)
	if err != nil {
		t.Fatalf("AddOpenBridgeHMAC() failed: %v", err)
	}

	tests := []struct {
		name       string
		packet     *DMRDPacket
		passphrase string
		want       bool
	}{
		{
			name:       "Valid HMAC",
			packet:     packet,
			passphrase: passphrase,
			want:       true,
		},
		{
			name:       "Wrong passphrase",
			packet:     packet,
			passphrase: "wrongpass",
			want:       false,
		},
		{
			name: "No HMAC",
			packet: &DMRDPacket{
				Sequence:      0x01,
				SourceID:      3120001,
				DestinationID: 3100,
				RepeaterID:    312000,
				Timeslot:      Timeslot1,
				CallType:      CallTypeGroup,
				FrameType:     FrameTypeVoice,
				DataType:      0x00,
				StreamID:      1,
				Payload:       make([]byte, 33),
			},
			passphrase: passphrase,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.packet.VerifyOpenBridgeHMAC(tt.passphrase); got != tt.want {
				t.Errorf("VerifyOpenBridgeHMAC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenBridgePacket_RoundTrip(t *testing.T) {
	// Create an OpenBridge packet
	original := &DMRDPacket{
		Sequence:      0x42,
		SourceID:      3120001,
		DestinationID: 91, // Brandmeister worldwide
		RepeaterID:    312000,
		Timeslot:      Timeslot1,
		CallType:      CallTypeGroup,
		FrameType:     FrameTypeVoiceHeader,
		DataType:      0x01,
		StreamID:      0x12345678,
		Payload:       make([]byte, 33),
	}

	// Fill payload with test data
	for i := 0; i < 33; i++ {
		original.Payload[i] = byte(i)
	}

	passphrase := "password"

	// Add HMAC
	err := original.AddOpenBridgeHMAC(passphrase)
	if err != nil {
		t.Fatalf("AddOpenBridgeHMAC() failed: %v", err)
	}

	// Encode
	data, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	// Parse back
	parsed := &DMRDPacket{}
	err = parsed.Parse(data)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Verify all fields
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
	if parsed.FrameType != original.FrameType {
		t.Errorf("FrameType mismatch: got %d, want %d", parsed.FrameType, original.FrameType)
	}
	if parsed.StreamID != original.StreamID {
		t.Errorf("StreamID mismatch: got %d, want %d", parsed.StreamID, original.StreamID)
	}
	if !bytes.Equal(parsed.Payload, original.Payload) {
		t.Error("Payload mismatch")
	}
	if !bytes.Equal(parsed.HMAC, original.HMAC) {
		t.Error("HMAC mismatch")
	}

	// Verify HMAC
	if !parsed.VerifyOpenBridgeHMAC(passphrase) {
		t.Error("HMAC verification failed after round trip")
	}
}

func TestOpenBridgePacket_NetworkID(t *testing.T) {
	// Test that network ID can be encoded in repeater ID field
	// OpenBridge uses this field for network identification
	packet := &DMRDPacket{
		Sequence:      0x01,
		SourceID:      3120001,
		DestinationID: 91,
		RepeaterID:    3129999, // Network ID
		Timeslot:      Timeslot1,
		CallType:      CallTypeGroup,
		FrameType:     FrameTypeVoice,
		DataType:      0x00,
		StreamID:      1,
		Payload:       make([]byte, 33),
	}

	passphrase := "password"
	err := packet.AddOpenBridgeHMAC(passphrase)
	if err != nil {
		t.Fatalf("AddOpenBridgeHMAC() failed: %v", err)
	}

	// Encode and parse
	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Encode() failed: %v", err)
	}

	parsed := &DMRDPacket{}
	err = parsed.Parse(data)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Verify network ID is preserved
	if parsed.RepeaterID != 3129999 {
		t.Errorf("Network ID not preserved: got %d, want %d", parsed.RepeaterID, 3129999)
	}
}

func TestOpenBridgePacket_BothSlots(t *testing.T) {
	// Test that both timeslots work with OpenBridge
	tests := []struct {
		name     string
		timeslot int
		callType int
	}{
		{"TS1 Group", Timeslot1, CallTypeGroup},
		{"TS2 Group", Timeslot2, CallTypeGroup},
		{"TS1 Private", Timeslot1, CallTypePrivate},
		{"TS2 Private", Timeslot2, CallTypePrivate},
	}

	passphrase := "password"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := &DMRDPacket{
				Sequence:      0x01,
				SourceID:      3120001,
				DestinationID: 3100,
				RepeaterID:    312000,
				Timeslot:      tt.timeslot,
				CallType:      tt.callType,
				FrameType:     FrameTypeVoice,
				DataType:      0x00,
				StreamID:      1,
				Payload:       make([]byte, 33),
			}

			err := packet.AddOpenBridgeHMAC(passphrase)
			if err != nil {
				t.Fatalf("AddOpenBridgeHMAC() failed: %v", err)
			}

			// Encode and parse
			data, err := packet.Encode()
			if err != nil {
				t.Fatalf("Encode() failed: %v", err)
			}

			parsed := &DMRDPacket{}
			err = parsed.Parse(data)
			if err != nil {
				t.Fatalf("Parse() failed: %v", err)
			}

			// Verify fields
			if parsed.Timeslot != tt.timeslot {
				t.Errorf("Timeslot mismatch: got %d, want %d", parsed.Timeslot, tt.timeslot)
			}
			if parsed.CallType != tt.callType {
				t.Errorf("CallType mismatch: got %d, want %d", parsed.CallType, tt.callType)
			}

			// Verify HMAC
			if !parsed.VerifyOpenBridgeHMAC(passphrase) {
				t.Error("HMAC verification failed")
			}
		})
	}
}
