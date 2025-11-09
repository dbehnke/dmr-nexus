package protocol

import "fmt"

// DMR Voice Sync patterns and embedded signalling
// Based on DMRDefines.h and Sync.cpp from MMDVMHost
// https://github.com/g4klx/MMDVMHost

// Voice sync patterns - 7 bytes inserted at bytes 13-19 with masking
// MS (Mobile Station) sourced patterns - used for repeater mode
// BS (Base Station) sourced patterns - used for network/master mode
var (
	// MS_SOURCED_AUDIO_SYNC is the voice sync pattern for MS mode (repeater to network)
	MS_SOURCED_AUDIO_SYNC = []byte{0x07, 0xF7, 0xD5, 0xDD, 0x57, 0xDF, 0xD0}

	// BS_SOURCED_AUDIO_SYNC is the voice sync pattern for BS mode (network to repeater)
	BS_SOURCED_AUDIO_SYNC = []byte{0x07, 0x55, 0xFD, 0x7D, 0xF7, 0x5F, 0x70}

	// MS_SOURCED_DATA_SYNC is the data sync pattern
	MS_SOURCED_DATA_SYNC = []byte{0x0D, 0x5D, 0x7F, 0x77, 0xFD, 0x75, 0x70}

	// SYNC_MASK protects the outer nibbles of bytes 13 and 19
	SYNC_MASK = []byte{0x0F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xF0}
)

// InsertVoiceSync inserts the voice sync pattern into a DMR voice frame
// The sync pattern occupies bytes 13-19 (7 bytes) with nibble masking
// This matches MMDVMHost's Sync::addDMRAudioSync implementation
func InsertVoiceSync(frame []byte, timeslot int) {
	if len(frame) < 20 {
		return
	}

	// Use MS-sourced audio sync pattern for both timeslots
	// In MS mode, the same pattern is used regardless of timeslot
	syncPattern := MS_SOURCED_AUDIO_SYNC

	// Debug: log bytes 13-19 before sync insertion (first time only)
	fmt.Printf("BEFORE sync: bytes[13-19] = %02X %02X %02X %02X %02X %02X %02X\n",
		frame[13], frame[14], frame[15], frame[16], frame[17], frame[18], frame[19])

	// Apply sync pattern at bytes 13-19 with masking
	// The mask preserves the outer nibbles (4 bits) of bytes 13 and 19
	for i := 0; i < 7; i++ {
		frame[i+13] = (frame[i+13] & ^SYNC_MASK[i]) | syncPattern[i]
	}

	// Debug: log result
	fmt.Printf("AFTER sync: bytes[13-19] = %02X %02X %02X %02X %02X %02X %02X\n",
		frame[13], frame[14], frame[15], frame[16], frame[17], frame[18], frame[19])
}

// EmbeddedLCEncoder maintains state for encoding embedded LC across a superframe
type EmbeddedLCEncoder struct {
	raw  [128]bool // Interleaved/encoded 128-bit buffer
	data [72]bool  // 72-bit LC data (FLCO, features, src, dst)
}

// NewEmbeddedLCEncoder creates an encoder for the given LC parameters
func NewEmbeddedLCEncoder(srcID, dstID uint32, flco FLCO) *EmbeddedLCEncoder {
	enc := &EmbeddedLCEncoder{}
	enc.encodeLC(srcID, dstID, flco)
	return enc
}

// encodeLC encodes the 72-bit LC data with Hamming(16,11,4) and parity checks
// Following DMREmbeddedData::encodeEmbeddedData from MMDVMHost
func (e *EmbeddedLCEncoder) encodeLC(srcID, dstID uint32, flco FLCO) {
	// Build the 72-bit LC: FLCO(6) + Features(3) + SrcID(24) + DstID(24) + Reserved(15)
	// For now, use the provided FLCO and zeros for features and reserved
	e.data[0] = (flco & 0x20) != 0
	e.data[1] = (flco & 0x10) != 0
	e.data[2] = (flco & 0x08) != 0
	e.data[3] = (flco & 0x04) != 0
	e.data[4] = (flco & 0x02) != 0
	e.data[5] = (flco & 0x01) != 0

	// Features (3 bits): PF, R, O (all false for now)
	e.data[6] = false // PF (Priority/Emergency)
	e.data[7] = false // R (Reserved)
	e.data[8] = false // O (OVCM)

	// SrcID (24 bits, MSB first)
	for i := 0; i < 24; i++ {
		e.data[9+i] = (srcID & (1 << (23 - i))) != 0
	}

	// DstID (24 bits, MSB first)
	for i := 0; i < 24; i++ {
		e.data[33+i] = (dstID & (1 << (23 - i))) != 0
	}

	// Reserved (15 bits) - all false
	for i := 57; i < 72; i++ {
		e.data[i] = false
	}

	// Compute 5-bit CRC on the data
	crc := computeFiveBitCRC(e.data[:])

	// Create working buffer with data and CRC embedded at specific positions
	var workBuf [128]bool

	// Copy data bits into working buffer at designated positions
	// Rows: 0-10, 16-26, 32-41, 48-57, 64-73, 80-89, 96-105
	b := 0
	for a := 0; a < 11; a++ {
		workBuf[a] = e.data[b]
		b++
	}
	for a := 16; a < 27; a++ {
		workBuf[a] = e.data[b]
		b++
	}
	for a := 32; a < 42; a++ {
		workBuf[a] = e.data[b]
		b++
	}
	for a := 48; a < 58; a++ {
		workBuf[a] = e.data[b]
		b++
	}
	for a := 64; a < 74; a++ {
		workBuf[a] = e.data[b]
		b++
	}
	for a := 80; a < 90; a++ {
		workBuf[a] = e.data[b]
		b++
	}
	for a := 96; a < 106; a++ {
		workBuf[a] = e.data[b]
		b++
	}

	// Insert the 5-bit CRC at positions 106, 90, 74, 58, 42
	workBuf[106] = (crc & 0x01) != 0
	workBuf[90] = (crc & 0x02) != 0
	workBuf[74] = (crc & 0x04) != 0
	workBuf[58] = (crc & 0x08) != 0
	workBuf[42] = (crc & 0x10) != 0

	// Apply Hamming(16,11,4) to each 16-bit row except the last
	for a := 0; a < 112; a += 16 {
		applyHamming16114(workBuf[a : a+16])
	}

	// Add parity bits for each column (row 7)
	for a := 0; a < 16; a++ {
		parity := workBuf[a+0] != workBuf[a+16] != workBuf[a+32] != workBuf[a+48] != workBuf[a+64] != workBuf[a+80] != workBuf[a+96]
		workBuf[a+112] = parity
	}

	// Pack downwards in columns to create the raw interleaved buffer
	b = 0
	for a := 0; a < 128; a++ {
		e.raw[a] = workBuf[b]
		b += 16
		if b > 127 {
			b -= 127
		}
	}
}

// GetFragment returns the 32-bit embedded data fragment for the given voice frame
// n is the fragment index: 0 (frame B), 1 (frame C), 2 (frame D), 3 (frame E), 4 (frame F)
// Returns the fragment as 5 bytes (with nibble masking) and the LCSS value
// Each fragment uses 32 bits from the 128-bit raw buffer, with the last fragment wrapping
func (e *EmbeddedLCEncoder) GetFragment(n int) (fragment [5]byte, lcss byte) {
	if n < 0 || n >= 5 {
		return [5]byte{}, 0
	}

	// Calculate the starting bit position in the 128-bit raw buffer
	// Fragments are distributed across the buffer: each uses 32 bits, but only
	// 128 bits total exist (5*32=160 > 128), so we modulo wrap
	startBit := (n * 32) % 128

	// Extract 32 bits from the raw buffer (with wrapping)
	// Pack into bits array with 4-bit padding at start and end
	var bits [40]bool
	for i := 0; i < 32; i++ {
		bitIdx := (startBit + i) % 128
		bits[i+4] = e.raw[bitIdx]
	}

	// Convert bits to 5 bytes
	for i := 0; i < 5; i++ {
		fragment[i] = bitsToByte(bits[i*8 : (i+1)*8])
	}

	// Determine LCSS based on fragment position
	// LCSS sequence: 1 (first), 3 (middle), 3 (middle), 3 (middle), 2 (last)
	switch n {
	case 0:
		lcss = 1 // First block (frame B)
	case 4:
		lcss = 2 // Last block (frame F)
	default:
		lcss = 3 // Middle blocks (frames C, D, E)
	}

	return fragment, lcss
}

// InsertEmbeddedFragment inserts a 32-bit embedded LC fragment into a DMR voice frame
// The fragment occupies the nibble-masked region in bytes 13-18
// Following DMREMB::getData and DMREmbeddedData::getData from MMDVMHost
func InsertEmbeddedFragment(frame []byte, fragment [5]byte, lcss byte) {
	if len(frame) < 20 {
		return
	}

	// Insert the 5-byte fragment into bytes 13-18 with nibble masking
	// Byte 13: preserve high nibble, insert low nibble from fragment[0] low nibble
	// Bytes 14-17: full bytes from fragment[1-4]
	// Byte 18: insert high nibble from fragment[4] high nibble, preserve low nibble
	frame[13] = (frame[13] & 0xF0) | (fragment[0] & 0x0F)
	frame[14] = fragment[1]
	frame[15] = fragment[2]
	frame[16] = fragment[3]
	frame[17] = fragment[4]
	frame[18] = (frame[18] & 0x0F) | (fragment[4] & 0xF0)

	// Note: The LCSS (Link Control Start/Stop) bits are encoded separately
	// in the EMB (Embedded Signalling) via QR(16,7,6) coding.
	// The EMB occupies part of bytes 13 and 18-19 and requires separate encoding.
	// For now, we just insert the LC fragment; full EMB encoding will be added later.
	_ = lcss
}

// Helper: compute 5-bit CRC for DMR embedded data
// This is a simplified CRC-5 for the 72-bit LC data
func computeFiveBitCRC(data []bool) byte {
	// DMR uses a specific CRC-5 polynomial
	// Polynomial: x^5 + x^4 + x^2 + 1 (0x15)
	const poly = 0x15
	var crc byte = 0

	for _, bit := range data {
		if bit {
			crc ^= 0x10
		}
		if (crc & 0x10) != 0 {
			crc = ((crc << 1) & 0x1F) ^ poly
		} else {
			crc = (crc << 1) & 0x1F
		}
	}

	return crc
}

// Helper: apply Hamming(16,11,4) encoding to a 16-bit block
// Input: 11 data bits in positions 0-10
// Output: 5 parity bits computed and placed in positions 11-15
func applyHamming16114(block []bool) {
	// Hamming(16,11,4) generator matrix for DMR
	// Standard configuration matching MMDVMHost's CHamming::encode16114
	p0 := block[0] != block[1] != block[2] != block[3] != block[5] != block[7] != block[8]
	p1 := block[1] != block[2] != block[3] != block[4] != block[6] != block[8] != block[9]
	p2 := block[2] != block[3] != block[4] != block[5] != block[7] != block[9] != block[10]
	p3 := block[0] != block[1] != block[2] != block[4] != block[6] != block[7] != block[10]
	p4 := block[0] != block[1] != block[3] != block[5] != block[6] != block[8] != block[9] != block[10]

	block[11] = p0
	block[12] = p1
	block[13] = p2
	block[14] = p3
	block[15] = p4
}

// Helper: convert 8 bool bits to a byte (MSB first)
func bitsToByte(bits []bool) byte {
	var b byte
	for i := 0; i < 8 && i < len(bits); i++ {
		if bits[i] {
			b |= 1 << (7 - i)
		}
	}
	return b
}
