package codec

// AMBE codec conversion tables and functions
// Based on ModeConv.cpp from MMDVM_CM

// DMR frame AMBE bit positions
// DMR uses 3 AMBE frames per 33-byte payload
// Each AMBE frame has A (24 bits), B (23 bits), and C (25 bits) fields

var (
	// DMR_A_TABLE maps 24 AMBE A bits to positions in DMR frame
	DMR_A_TABLE = []uint{
		0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44,
		48, 52, 56, 60, 64, 68, 1, 5, 9, 13, 17, 21,
	}

	// DMR_B_TABLE maps 23 AMBE B bits to positions in DMR frame
	DMR_B_TABLE = []uint{
		25, 29, 33, 37, 41, 45, 49, 53, 57, 61, 65, 69,
		2, 6, 10, 14, 18, 22, 26, 30, 34, 38, 42,
	}

	// DMR_C_TABLE maps 25 AMBE C bits to positions in DMR frame
	DMR_C_TABLE = []uint{
		46, 50, 54, 58, 62, 66, 70, 3, 7, 11, 15, 19, 23,
		27, 31, 35, 39, 43, 47, 51, 55, 59, 63, 67, 71,
	}

	// INTERLEAVE_TABLE_26_4 is used for YSF VCH interleaving
	// Maps 104 bits (26 symbols * 4 bits) for interleaving
	// Row-major pattern as per MMDVMHost YSFPayload.cpp line 67-73
	INTERLEAVE_TABLE_26_4 = []uint{
		0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56, 60, 64, 68, 72, 76, 80, 84, 88, 92, 96, 100,
		1, 5, 9, 13, 17, 21, 25, 29, 33, 37, 41, 45, 49, 53, 57, 61, 65, 69, 73, 77, 81, 85, 89, 93, 97, 101,
		2, 6, 10, 14, 18, 22, 26, 30, 34, 38, 42, 46, 50, 54, 58, 62, 66, 70, 74, 78, 82, 86, 90, 94, 98, 102,
		3, 7, 11, 15, 19, 23, 27, 31, 35, 39, 43, 47, 51, 55, 59, 63, 67, 71, 75, 79, 83, 87, 91, 95, 99, 103,
	}

	// WHITENING_DATA is XORed with YSF VCH data for scrambling
	WHITENING_DATA = []byte{
		0x93, 0xD7, 0x51, 0x21, 0x9C, 0x2F, 0x6C, 0xD0, 0xEF, 0x0F,
		0xF8, 0x3D, 0xF1, 0x73, 0x20, 0x94, 0xED, 0x1E, 0x7C, 0xD8,
	}
)

// Bit manipulation helpers
const (
	bitMask0 = 0x80
	bitMask1 = 0x40
	bitMask2 = 0x20
	bitMask3 = 0x10
	bitMask4 = 0x08
	bitMask5 = 0x04
	bitMask6 = 0x02
	bitMask7 = 0x01
)

var bitMaskTable = []byte{bitMask0, bitMask1, bitMask2, bitMask3, bitMask4, bitMask5, bitMask6, bitMask7}

// readBit reads a bit from a byte array at the specified bit position
func readBit(data []byte, pos uint) bool {
	bytePos := pos >> 3
	bitPos := pos & 7
	if int(bytePos) >= len(data) {
		return false
	}
	return (data[bytePos] & bitMaskTable[bitPos]) != 0
}

// writeBit writes a bit to a byte array at the specified bit position
func writeBit(data []byte, pos uint, value bool) {
	bytePos := pos >> 3
	bitPos := pos & 7
	if int(bytePos) >= len(data) {
		return
	}
	if value {
		data[bytePos] |= bitMaskTable[bitPos]
	} else {
		data[bytePos] &= ^bitMaskTable[bitPos]
	}
}

// extractSingleAMBEFromDMR extracts a single AMBE frame from a 9-byte DMR mini-frame
// Returns (a, b, c) AMBE parameters
func extractSingleAMBEFromDMR(dmrFrame []byte) (a, b, c uint32) {
	var mask uint32 = 0x800000
	for i := uint(0); i < 24; i++ {
		aPos := DMR_A_TABLE[i]
		if readBit(dmrFrame, aPos) {
			a |= mask
		}
		mask >>= 1
	}

	mask = 0x400000
	for i := uint(0); i < 23; i++ {
		bPos := DMR_B_TABLE[i]
		if readBit(dmrFrame, bPos) {
			b |= mask
		}
		mask >>= 1
	}

	mask = 0x1000000
	for i := uint(0); i < 25; i++ {
		cPos := DMR_C_TABLE[i]
		if readBit(dmrFrame, cPos) {
			c |= mask
		}
		mask >>= 1
	}

	return
}

// extractAMBEFromDMR extracts 3 AMBE frames from a DMR voice frame
// DMR frame is 33 bytes containing 3 AMBE frames
// Returns three sets of (a, b, c) AMBE parameters
func extractAMBEFromDMR(dmrFrame []byte) (a1, b1, c1, a2, b2, c2, a3, b3, c3 uint32) {
	// Extract first AMBE frame
	var mask uint32 = 0x800000
	for i := uint(0); i < 24; i++ {
		a1Pos := DMR_A_TABLE[i]
		a2Pos := a1Pos + 72
		// Skip the reserved sync/embedded region (bytes 13-19 => bits 104-159)
		// Any second-frame bit that lands at >=104 must be moved past the region by +48
		if a2Pos >= 108 {
			a2Pos += 48
		}
		a3Pos := a1Pos + 192

		if readBit(dmrFrame, a1Pos) {
			a1 |= mask
		}
		if readBit(dmrFrame, a2Pos) {
			a2 |= mask
		}
		if readBit(dmrFrame, a3Pos) {
			a3 |= mask
		}
		mask >>= 1
	}

	mask = 0x400000
	for i := uint(0); i < 23; i++ {
		b1Pos := DMR_B_TABLE[i]
		b2Pos := b1Pos + 72
		// Skip reserved region for second frame as above
		if b2Pos >= 108 {
			b2Pos += 48
		}
		b3Pos := b1Pos + 192

		if readBit(dmrFrame, b1Pos) {
			b1 |= mask
		}
		if readBit(dmrFrame, b2Pos) {
			b2 |= mask
		}
		if readBit(dmrFrame, b3Pos) {
			b3 |= mask
		}
		mask >>= 1
	}

	mask = 0x1000000
	for i := uint(0); i < 25; i++ {
		c1Pos := DMR_C_TABLE[i]
		c2Pos := c1Pos + 72
		// Skip reserved region for second frame as above
		if c2Pos >= 108 {
			c2Pos += 48
		}
		c3Pos := c1Pos + 192

		if readBit(dmrFrame, c1Pos) {
			c1 |= mask
		}
		if readBit(dmrFrame, c2Pos) {
			c2 |= mask
		}
		if readBit(dmrFrame, c3Pos) {
			c3 |= mask
		}
		mask >>= 1
	}

	return
}

// insertAMBEToDMR inserts AMBE parameters into a DMR voice frame
func insertAMBEToDMR(dmrFrame []byte, a, b, c uint32) {
	var mask uint32 = 0x800000
	for i := uint(0); i < 24; i++ {
		aPos := DMR_A_TABLE[i]
		writeBit(dmrFrame, aPos, (a&mask) != 0)
		mask >>= 1
	}

	mask = 0x400000
	for i := uint(0); i < 23; i++ {
		bPos := DMR_B_TABLE[i]
		writeBit(dmrFrame, bPos, (b&mask) != 0)
		mask >>= 1
	}

	mask = 0x1000000
	for i := uint(0); i < 25; i++ {
		cPos := DMR_C_TABLE[i]
		writeBit(dmrFrame, cPos, (c&mask) != 0)
		mask >>= 1
	}
}

// insertInterleavedAMBEToDMR inserts 3 interleaved AMBE frames into a DMR voice frame
// This matches MMDVM_CM's frame structure where:
// - Frame 1 bits are at base table positions
// - Frame 2 bits are at table positions + 72 (with adjustment if >= 108)
// - Frame 3 bits are at table positions + 192
func insertInterleavedAMBEToDMR(dmrFrame []byte, a1, b1, c1, a2, b2, c2, a3, b3, c3 uint32) {
	// Insert A parameters for all 3 frames
	var mask uint32 = 0x800000
	for i := uint(0); i < 24; i++ {
		a1Pos := DMR_A_TABLE[i]
		a2Pos := a1Pos + 72
		// Align with MMDVM bit numbering: skip when >=108 by +48
		if a2Pos >= 108 {
			a2Pos += 48
		}
		a3Pos := a1Pos + 192

		writeBit(dmrFrame, a1Pos, (a1&mask) != 0)
		writeBit(dmrFrame, a2Pos, (a2&mask) != 0)
		writeBit(dmrFrame, a3Pos, (a3&mask) != 0)
		mask >>= 1
	}

	// Insert B parameters for all 3 frames
	mask = 0x400000
	for i := uint(0); i < 23; i++ {
		b1Pos := DMR_B_TABLE[i]
		b2Pos := b1Pos + 72
		if b2Pos >= 108 {
			b2Pos += 48
		}
		b3Pos := b1Pos + 192

		writeBit(dmrFrame, b1Pos, (b1&mask) != 0)
		writeBit(dmrFrame, b2Pos, (b2&mask) != 0)
		writeBit(dmrFrame, b3Pos, (b3&mask) != 0)
		mask >>= 1
	}

	// Insert C parameters for all 3 frames
	mask = 0x1000000
	for i := uint(0); i < 25; i++ {
		c1Pos := DMR_C_TABLE[i]
		c2Pos := c1Pos + 72
		if c2Pos >= 108 {
			c2Pos += 48
		}
		c3Pos := c1Pos + 192

		writeBit(dmrFrame, c1Pos, (c1&mask) != 0)
		writeBit(dmrFrame, c2Pos, (c2&mask) != 0)
		writeBit(dmrFrame, c3Pos, (c3&mask) != 0)
		mask >>= 1
	}
}

// ambeToYSF converts AMBE parameters (a, b, c) to YSF VCH format
// Returns 13-byte YSF VCH frame
func ambeToYSF(a, b, datC uint32) []byte {
	vch := make([]byte, 13)
	ysfFrame := make([]byte, 13)

	datA := a >> 12

	// Apply PRNG to b parameter
	b ^= (prngTable[datA] >> 1)

	datB := b >> 11

	// Encode dat_a (12 bits) into 36 bits (tripled)
	// Extract MSB first: bit 11 down to bit 0
	for i := uint(0); i < 12; i++ {
		bit := (datA >> (11 - i)) & 0x01
		writeBit(vch, 3*i+0, bit != 0)
		writeBit(vch, 3*i+1, bit != 0)
		writeBit(vch, 3*i+2, bit != 0)
	}

	// Encode dat_b (12 bits) into 36 bits (tripled)
	// Extract MSB first: bit 11 down to bit 0
	for i := uint(0); i < 12; i++ {
		bit := (datB >> (11 - i)) & 0x01
		writeBit(vch, 3*(i+12)+0, bit != 0)
		writeBit(vch, 3*(i+12)+1, bit != 0)
		writeBit(vch, 3*(i+12)+2, bit != 0)
	}

	// Encode first 3 bits of dat_c into 9 bits (tripled)
	// dat_c is 25 bits, extract MSB first: bit 24 down to bit 22
	for i := uint(0); i < 3; i++ {
		bit := (datC >> (24 - i)) & 0x01
		writeBit(vch, 3*(i+24)+0, bit != 0)
		writeBit(vch, 3*(i+24)+1, bit != 0)
		writeBit(vch, 3*(i+24)+2, bit != 0)
	}

	// Encode remaining 22 bits of dat_c (bits 21 down to 0)
	for i := uint(0); i < 22; i++ {
		bit := (datC >> (21 - i)) & 0x01
		writeBit(vch, i+81, bit != 0)
	}

	writeBit(vch, 103, false)

	// Scramble (whiten) the data
	for i := 0; i < 13; i++ {
		vch[i] ^= WHITENING_DATA[i]
	}

	// Interleave
	for i := uint(0); i < 104; i++ {
		n := INTERLEAVE_TABLE_26_4[i]
		s := readBit(vch, i)
		writeBit(ysfFrame, n, s)
	}

	return ysfFrame
}

// ysfToAMBE converts YSF VCH format to AMBE parameters
// Returns (dat_a, dat_b, dat_c)
func ysfToAMBE(ysfVCH []byte, offset uint) (uint32, uint32, uint32) {
	vch := make([]byte, 13)

	// Deinterleave
	for i := uint(0); i < 104; i++ {
		n := INTERLEAVE_TABLE_26_4[i]
		s := readBit(ysfVCH, offset+n)
		writeBit(vch, i, s)
	}

	// Descramble (unwhiten)
	for i := 0; i < 13; i++ {
		vch[i] ^= WHITENING_DATA[i]
	}

	// Extract dat_a (12 bits from tripled encoding)
	// CRITICAL: Only read the MIDDLE bit (bit 1), not majority vote!
	// This matches MMDVM_CM YSF2DMR/ModeConv.cpp lines 624-628
	var datA uint32
	for i := uint(0); i < 12; i++ {
		datA <<= 1
		if readBit(vch, 3*i+1) {
			datA |= 0x01
		}
	}

	// Extract dat_b (12 bits from tripled encoding)
	// CRITICAL: Only read the MIDDLE bit (bit 1), not majority vote!
	// This matches MMDVM_CM YSF2DMR/ModeConv.cpp lines 630-634
	var datB uint32
	for i := uint(0); i < 12; i++ {
		datB <<= 1
		if readBit(vch, 3*(i+12)+1) {
			datB |= 0x01
		}
	}

	// Extract dat_c (25 bits: first 3 bits tripled + 22 direct)
	// CRITICAL: Only read the MIDDLE bit (bit 1) of tripled bits, not majority vote!
	// This matches MMDVM_CM YSF2DMR/ModeConv.cpp lines 636-640
	var datC uint32
	for i := uint(0); i < 3; i++ {
		datC <<= 1
		if readBit(vch, 3*(i+24)+1) {
			datC |= 0x01
		}
	}

	for i := uint(0); i < 22; i++ {
		datC <<= 1
		if readBit(vch, i+81) {
			datC |= 0x01
		}
	}

	// YSF stores 12-bit parameter values for A and B
	// These need to be treated as DATA and re-encoded with Golay for DMR
	// We cannot reconstruct the original encoded values because YSF is lossy

	// Return the raw 12-bit parameter values - converter.go will Golay-encode them
	return datA, datB, datC
}
