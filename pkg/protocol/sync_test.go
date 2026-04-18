package protocol

import (
	"testing"
)

// TestEmbeddedLCEncoder_Basic validates the encoder creates and encodes properly
func TestEmbeddedLCEncoder_Basic(t *testing.T) {
	srcID := uint32(3120001)
	dstID := uint32(70777)
	flco := FLCOGroup

	enc := NewEmbeddedLCEncoder(srcID, dstID, flco)
	if enc == nil {
		t.Fatal("NewEmbeddedLCEncoder returned nil")
	}

	// Verify data buffer contains expected values
	// FLCO should be in first 6 bits (FLCO_GROUP = 0)
	for i := 0; i < 6; i++ {
		if enc.data[i] {
			t.Errorf("FLCO bit %d should be false for FLCOGroup, got true", i)
		}
	}

	// Features bits should be false
	if enc.data[6] || enc.data[7] || enc.data[8] {
		t.Error("Features bits should all be false")
	}

	// Verify source ID is encoded (bits 9-32)
	expectedSrc := srcID
	var decodedSrc uint32
	for i := 0; i < 24; i++ {
		if enc.data[9+i] {
			decodedSrc |= 1 << (23 - i)
		}
	}
	if decodedSrc != expectedSrc {
		t.Errorf("Source ID mismatch: expected %d, got %d", expectedSrc, decodedSrc)
	}

	// Verify destination ID is encoded (bits 33-56)
	expectedDst := dstID
	var decodedDst uint32
	for i := 0; i < 24; i++ {
		if enc.data[33+i] {
			decodedDst |= 1 << (23 - i)
		}
	}
	if decodedDst != expectedDst {
		t.Errorf("Destination ID mismatch: expected %d, got %d", expectedDst, decodedDst)
	}
}

// TestEmbeddedLCEncoder_GetFragment validates fragment extraction and LCSS
func TestEmbeddedLCEncoder_GetFragment(t *testing.T) {
	srcID := uint32(3120001)
	dstID := uint32(70777)
	flco := FLCOGroup

	enc := NewEmbeddedLCEncoder(srcID, dstID, flco)

	tests := []struct {
		name         string
		fragmentIdx  int
		expectedLCSS byte
	}{
		{"Fragment 0 (Frame B)", 0, 1},
		{"Fragment 1 (Frame C)", 1, 3},
		{"Fragment 2 (Frame D)", 2, 3},
		{"Fragment 3 (Frame E)", 3, 3},
		{"Fragment 4 (Frame F)", 4, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fragment, lcss := enc.GetFragment(tt.fragmentIdx)

			if lcss != tt.expectedLCSS {
				t.Errorf("LCSS mismatch for fragment %d: expected %d, got %d",
					tt.fragmentIdx, tt.expectedLCSS, lcss)
			}

			// Fragment should be non-zero (we have data encoded)
			allZero := true
			for _, b := range fragment {
				if b != 0 {
					allZero = false
					break
				}
			}
			if allZero {
				t.Error("Fragment is all zeros - expected encoded data")
			}
		})
	}
}

// TestEmbeddedLCEncoder_InvalidFragment validates boundary checking
func TestEmbeddedLCEncoder_InvalidFragment(t *testing.T) {
	enc := NewEmbeddedLCEncoder(3120001, 70777, FLCOGroup)

	tests := []struct {
		name        string
		fragmentIdx int
	}{
		{"Negative index", -1},
		{"Index too large", 5},
		{"Way too large", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fragment, lcss := enc.GetFragment(tt.fragmentIdx)

			// Should return zero fragment and zero LCSS
			if lcss != 0 {
				t.Errorf("Expected LCSS=0 for invalid index, got %d", lcss)
			}

			allZero := true
			for _, b := range fragment {
				if b != 0 {
					allZero = false
					break
				}
			}
			if !allZero {
				t.Error("Expected zero fragment for invalid index")
			}
		})
	}
}

// TestInsertEmbeddedFragment validates fragment insertion into frame
func TestInsertEmbeddedFragment(t *testing.T) {
	// Create a test frame (33 bytes)
	frame := make([]byte, 33)
	for i := range frame {
		frame[i] = 0xFF // Fill with 0xFF to test masking
	}

	// Create test fragment
	fragment := [5]byte{0x12, 0x34, 0x56, 0x78, 0x9A}
	lcss := byte(1)

	InsertEmbeddedFragment(frame, fragment, lcss)

	// Check byte 13: high nibble preserved (0xF), low nibble from fragment[0] (0x2)
	if frame[13] != 0xF2 {
		t.Errorf("Byte 13 mismatch: expected 0xF2, got 0x%02X", frame[13])
	}

	// Check bytes 14-17: full bytes from fragment[1-4]
	if frame[14] != 0x34 {
		t.Errorf("Byte 14 mismatch: expected 0x34, got 0x%02X", frame[14])
	}
	if frame[15] != 0x56 {
		t.Errorf("Byte 15 mismatch: expected 0x56, got 0x%02X", frame[15])
	}
	if frame[16] != 0x78 {
		t.Errorf("Byte 16 mismatch: expected 0x78, got 0x%02X", frame[16])
	}
	if frame[17] != 0x9A {
		t.Errorf("Byte 17 mismatch: expected 0x9A, got 0x%02X", frame[17])
	}

	// Check byte 18: low nibble preserved (0xF), high nibble from fragment[4] (0x9)
	if frame[18] != 0x9F {
		t.Errorf("Byte 18 mismatch: expected 0x9F, got 0x%02X", frame[18])
	}

	// Other bytes should be unchanged
	for i := 0; i < 13; i++ {
		if frame[i] != 0xFF {
			t.Errorf("Byte %d should be unchanged (0xFF), got 0x%02X", i, frame[i])
		}
	}
	for i := 19; i < 33; i++ {
		if frame[i] != 0xFF {
			t.Errorf("Byte %d should be unchanged (0xFF), got 0x%02X", i, frame[i])
		}
	}
}

// TestInsertEmbeddedFragment_ShortFrame validates handling of short frames
func TestInsertEmbeddedFragment_ShortFrame(t *testing.T) {
	// Create a frame that's too short
	frame := make([]byte, 19)
	for i := range frame {
		frame[i] = 0xFF
	}

	fragment := [5]byte{0x12, 0x34, 0x56, 0x78, 0x9A}
	lcss := byte(1)

	// Should not panic or corrupt memory
	InsertEmbeddedFragment(frame, fragment, lcss)

	// Frame should be unchanged since it's too short
	for i := range frame {
		if frame[i] != 0xFF {
			t.Errorf("Short frame was modified at byte %d", i)
		}
	}
}

// TestBitsToByte validates bit-to-byte conversion
func TestBitsToByte(t *testing.T) {
	tests := []struct {
		name     string
		bits     []bool
		expected byte
	}{
		{
			name:     "All zeros",
			bits:     []bool{false, false, false, false, false, false, false, false},
			expected: 0x00,
		},
		{
			name:     "All ones",
			bits:     []bool{true, true, true, true, true, true, true, true},
			expected: 0xFF,
		},
		{
			name:     "Alternating",
			bits:     []bool{true, false, true, false, true, false, true, false},
			expected: 0xAA,
		},
		{
			name:     "MSB only",
			bits:     []bool{true, false, false, false, false, false, false, false},
			expected: 0x80,
		},
		{
			name:     "LSB only",
			bits:     []bool{false, false, false, false, false, false, false, true},
			expected: 0x01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bitsToByte(tt.bits)
			if result != tt.expected {
				t.Errorf("Expected 0x%02X, got 0x%02X", tt.expected, result)
			}
		})
	}
}

// TestApplyHamming16114 validates Hamming encoding
func TestApplyHamming16114(t *testing.T) {
	// Test with all zeros - should produce valid parity bits
	block := make([]bool, 16)
	applyHamming16114(block)

	// All parity bits should be false for all-zero data
	for i := 11; i < 16; i++ {
		if block[i] {
			t.Errorf("Parity bit %d should be false for all-zero data", i)
		}
	}

	// Test with all ones in data positions
	block2 := make([]bool, 16)
	for i := 0; i < 11; i++ {
		block2[i] = true
	}
	applyHamming16114(block2)

	// Verify parity bits were computed (should not all be false)
	parityAllFalse := true
	for i := 11; i < 16; i++ {
		if block2[i] {
			parityAllFalse = false
			break
		}
	}
	if parityAllFalse {
		t.Error("Expected at least one parity bit to be true for all-ones data")
	}
}

// TestComputeFiveBitCRC validates CRC computation
func TestComputeFiveBitCRC(t *testing.T) {
	// Test with all zeros
	data1 := make([]bool, 72)
	crc1 := computeFiveBitCRC(data1)

	// CRC should be valid (5 bits max)
	if crc1 > 0x1F {
		t.Errorf("CRC should be 5 bits max, got 0x%02X", crc1)
	}

	// Test with different data should produce different CRC
	data2 := make([]bool, 72)
	data2[0] = true
	crc2 := computeFiveBitCRC(data2)

	if crc1 == crc2 {
		t.Error("Different data should produce different CRCs")
	}

	// CRC should be deterministic
	crc3 := computeFiveBitCRC(data2)
	if crc2 != crc3 {
		t.Error("CRC should be deterministic")
	}
}

// TestEmbeddedLCEncoder_FullIntegration validates end-to-end encoding
func TestEmbeddedLCEncoder_FullIntegration(t *testing.T) {
	srcID := uint32(3120001)
	dstID := uint32(70777)
	flco := FLCOGroup

	enc := NewEmbeddedLCEncoder(srcID, dstID, flco)

	// Create 6 DMR frames (superframe)
	frames := make([][]byte, 6)
	for i := range frames {
		frames[i] = make([]byte, 33)
	}

	// Frame 0 (A): gets voice sync, not embedded LC
	InsertVoiceSync(frames[0], 1)

	// Frames 1-5 (B-F): get embedded LC fragments
	for i := 1; i < 6; i++ {
		fragment, lcss := enc.GetFragment(i - 1)
		InsertEmbeddedFragment(frames[i], fragment, lcss)
	}

	// Verify frames are non-zero and different from each other
	for i := 0; i < 6; i++ {
		allZero := true
		for j := 13; j <= 18; j++ {
			if frames[i][j] != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			t.Errorf("Frame %d bytes 13-18 are all zero", i)
		}
	}

	// Verify each embedded frame is unique (except frames that wrap due to 128-bit limit)
	// With 128 bits and 5 fragments of 32 bits (160 total), we expect wrapping
	// Fragment 4 (bits 128-159 % 128 = 0-31) may equal Fragment 0
	for i := 1; i < 5; i++ {
		for j := i + 1; j < 6; j++ {
			// Skip comparison of frames that wrap (fragment 0 and fragment 4)
			if i == 1 && j == 5 {
				continue
			}

			same := true
			for k := 13; k <= 18; k++ {
				if frames[i][k] != frames[j][k] {
					same = false
					break
				}
			}
			if same {
				t.Errorf("Frames %d and %d have identical embedded regions", i, j)
			}
		}
	}
}
