package protocol

import "testing"

func TestBuildVoiceLCHeader_MapsIDsAndFLCO(t *testing.T) {
	srcID := uint32(5300208)
	dstID := uint32(34000)

	// Group call
	payload := BuildVoiceLCHeader(srcID, dstID, FLCOGroup)

	if len(payload) != 33 {
		t.Fatalf("expected payload length 33, got %d", len(payload))
	}

	// Byte 0: FLCO in lower 6 bits
	if got := payload[0] & 0x3F; got != byte(FLCOGroup) {
		t.Errorf("expected FLCO %d, got %d", FLCOGroup, got)
	}

	// Bytes 1-3: Destination ID (24-bit big-endian)
	if payload[1] != byte(dstID>>16) || payload[2] != byte(dstID>>8) || payload[3] != byte(dstID) {
		t.Errorf("destination ID not encoded correctly: %02X %02X %02X", payload[1], payload[2], payload[3])
	}

	// Bytes 4-6: Source ID (24-bit big-endian)
	if payload[4] != byte(srcID>>16) || payload[5] != byte(srcID>>8) || payload[6] != byte(srcID) {
		t.Errorf("source ID not encoded correctly: %02X %02X %02X", payload[4], payload[5], payload[6])
	}

	// Bytes 7-8 should be zero in simplified form
	if payload[7] != 0x00 || payload[8] != 0x00 {
		t.Errorf("expected bytes 7-8 to be zero, got %02X %02X", payload[7], payload[8])
	}
}

func TestBuildVoiceTerminatorPayload_MatchesHeaderLayout(t *testing.T) {
	srcID := uint32(123456)
	dstID := uint32(7890)

	header := BuildVoiceLCHeader(srcID, dstID, FLCOUserUser)
	term := BuildVoiceTerminatorPayload(srcID, dstID, FLCOUserUser)

	if len(term) != 33 {
		t.Fatalf("expected payload length 33, got %d", len(term))
	}

	// First 9 bytes LC should match the header's LC with same inputs
	for i := 0; i < 9; i++ {
		if term[i] != header[i] {
			t.Fatalf("terminator LC mismatch at byte %d: %02X != %02X", i, term[i], header[i])
		}
	}
}

func TestParseVoiceLCHeader_RoundTrip(t *testing.T) {
	srcID := uint32(424242)
	dstID := uint32(3100)

	payload := BuildVoiceLCHeader(srcID, dstID, FLCOGroup)

	parsedSrc, parsedDst, flco, ok := ParseVoiceLCHeader(payload)
	if !ok {
		t.Fatalf("expected ok=true parsing LC header")
	}

	if parsedSrc != srcID {
		t.Errorf("source ID mismatch: got %d, want %d", parsedSrc, srcID)
	}

	if parsedDst != dstID {
		t.Errorf("destination ID mismatch: got %d, want %d", parsedDst, dstID)
	}

	if flco != FLCOGroup {
		t.Errorf("FLCO mismatch: got %d, want %d", flco, FLCOGroup)
	}
}
