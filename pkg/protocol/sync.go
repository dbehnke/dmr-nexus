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

// BuildEmbeddedLC builds embedded LC fragment for non-sync voice frames
// The full LC is fragmented across multiple voice frames (A-F)
// For basic functionality, we use a simplified embedded LC pattern
func BuildEmbeddedLC(srcID, dstID uint32, flco FLCO, fragment int) []byte {
	// Return 7 bytes to match the sync pattern area
	// DMR embedded LC follows the same structure as full LC:
	// Byte 0: FLCO
	// Bytes 1-3: Destination ID (24-bit big-endian)
	// Bytes 4-6: Source ID (24-bit big-endian)
	// Fragments carry this data in sequence across voice frames A-F
	lc := make([]byte, 7)

	// Map fragment to LC byte positions
	// Fragment 0: LC bytes 0-3 (FLCO + dest[23:16] + dest[15:8] + dest[7:0])
	// Fragment 1: LC bytes 4-6 + byte 7 start (src[23:16] + src[15:8] + src[7:0] + options)
	// etc.

	switch fragment {
	case 0:
		// Embedded LC fragment 0: FLCO and destination ID
		lc[0] = byte(flco) & 0x0F // only lower nibble is writable in byte 13
		lc[1] = byte(dstID >> 16)
		lc[2] = byte(dstID >> 8)
		lc[3] = byte(dstID)
	case 1:
		// Embedded LC fragment 1: Source ID
		lc[0] = byte(srcID >> 16)
		lc[1] = byte(srcID >> 8)
		lc[2] = byte(srcID)
	case 2, 3, 4, 5:
		// Later fragments would carry options/FEC
		// For now, leave as zeros
	}

	// Respect nibble-protection like sync patterns do:
	// - Byte 0 (frame[13]) only lower nibble is used (upper nibble preserved)
	// - Bytes 1..5 (frame[14..18]) full bytes are used
	// - Byte 6 (frame[19]) only upper nibble is used; we keep lower nibble at 0
	// Ensure our last byte doesn't set the protected lower nibble
	lc[6] &= 0x00

	return lc
}

// InsertEmbeddedLC inserts embedded LC signalling into a DMR voice frame
// Embedded signalling occupies the same bytes as sync (13-19) but with different content
func InsertEmbeddedLC(frame []byte, srcID, dstID uint32, flco FLCO, voiceSeq int) {
	if len(frame) < 20 {
		return
	}

	// Voice sequence A-F maps to fragments 0-5
	embeddedData := BuildEmbeddedLC(srcID, dstID, flco, voiceSeq)

	// Debug first few insertions to validate masking and content
	// Note: keep this lightweight; comment out if too chatty
	fmt.Printf("BEFORE embedded: bytes[13-19] = %02X %02X %02X %02X %02X %02X %02X\n",
		frame[13], frame[14], frame[15], frame[16], frame[17], frame[18], frame[19])

	// Insert embedded data at bytes 13-19 with the same masking as sync
	// The mask preserves the outer nibbles of bytes 13 and 19
	for i := 0; i < 7; i++ {
		frame[i+13] = (frame[i+13] & ^SYNC_MASK[i]) | embeddedData[i]
	}

	fmt.Printf("AFTER  embedded: bytes[13-19] = %02X %02X %02X %02X %02X %02X %02X\n",
		frame[13], frame[14], frame[15], frame[16], frame[17], frame[18], frame[19])
}
