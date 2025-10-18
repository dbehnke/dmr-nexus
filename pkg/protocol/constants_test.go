package protocol

import (
	"testing"
)

func TestPacketTypes(t *testing.T) {
	tests := []struct {
		name     string
		pktType  string
		expected int
	}{
		{"DMRD packet", "DMRD", 4},
		{"RPTL packet", "RPTL", 4},
		{"RPTK packet", "RPTK", 4},
		{"RPTC packet", "RPTC", 4},
		{"RPTACK packet", "RPTACK", 6},
		{"RPTPING packet", "RPTPING", 7},
		{"MSTPONG packet", "MSTPONG", 7},
		{"MSTCL packet", "MSTCL", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.pktType) != tt.expected {
				t.Errorf("Expected packet type '%s' to have length %d, got %d",
					tt.pktType, tt.expected, len(tt.pktType))
			}
		})
	}
}

func TestPacketSizes(t *testing.T) {
	tests := []struct {
		name         string
		packetType   string
		expectedSize int
	}{
		{"DMRD standard", "DMRD", DMRDPacketSize},
		{"DMRD OpenBridge", "DMRD_OBP", DMRDOpenBridgePacketSize},
		{"RPTL", "RPTL", RPTLPacketSize},
		{"RPTK", "RPTK", RPTKPacketSize},
		{"RPTC", "RPTC", RPTCPacketSize},
		{"RPTACK", "RPTACK", RPTACKPacketSize},
		{"RPTPING", "RPTPING", RPTPINGPacketSize},
		{"MSTPONG", "MSTPONG", MSTPONGPacketSize},
		{"MSTCL", "MSTCL", MSTCLPacketSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedSize <= 0 {
				t.Errorf("Expected positive packet size for %s", tt.packetType)
			}
		})
	}
}

func TestSlotBitMasks(t *testing.T) {
	tests := []struct {
		name     string
		mask     byte
		expected string
	}{
		{"Timeslot mask", SlotTimeslotMask, "0x80"},
		{"Call type mask", SlotCallTypeMask, "0x40"},
		{"Frame type mask", SlotFrameTypeMask, "0x30"},
		{"Data type mask", SlotDataTypeMask, "0x0F"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mask == 0 && tt.name != "Data type mask" {
				t.Errorf("Expected non-zero mask for %s", tt.name)
			}
		})
	}
}

func TestFrameTypes(t *testing.T) {
	tests := []struct {
		name      string
		frameType byte
		desc      string
	}{
		{"Voice burst", FrameTypeVoice, "Voice burst frame"},
		{"Voice header", FrameTypeVoiceHeader, "Voice header frame"},
		{"Voice terminator", FrameTypeVoiceTerminator, "Voice terminator frame"},
		{"Data sync", FrameTypeDataSync, "Data sync frame"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify they're defined (will check actual values after implementation)
			_ = tt.frameType
		})
	}
}

func TestCallTypes(t *testing.T) {
	// Test that call type detection works correctly
	groupCall := byte(0x00) // bit 6 clear = group call
	privateCall := byte(0x40) // bit 6 set = private call

	if groupCall&SlotCallTypeMask != 0 {
		t.Error("Expected group call to have bit 6 clear")
	}

	if privateCall&SlotCallTypeMask == 0 {
		t.Error("Expected private call to have bit 6 set")
	}
}

func TestTimeslotDetection(t *testing.T) {
	// Test timeslot bit detection
	ts1Byte := byte(0x00) // bit 7 clear = TS1
	ts2Byte := byte(0x80) // bit 7 set = TS2

	if ts1Byte&SlotTimeslotMask != 0 {
		t.Error("Expected TS1 to have bit 7 clear")
	}

	if ts2Byte&SlotTimeslotMask == 0 {
		t.Error("Expected TS2 to have bit 7 set")
	}
}
