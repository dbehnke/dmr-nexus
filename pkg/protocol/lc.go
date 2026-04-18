package protocol

// BuildVoiceLCHeader builds a DMR Voice LC Header payload (33 bytes)
// This encodes the source ID, destination ID, and FLCO into the payload
func BuildVoiceLCHeader(srcID, dstID uint32, flco FLCO) []byte {
	payload := make([]byte, 33)

	// DMR Voice LC Header structure (simplified):
	// Bytes 0-8: Full LC data (9 bytes)
	// Bytes 9-32: Reed-Solomon FEC and padding

	// Build Full LC (9 bytes total)
	lc := make([]byte, 9)

	// Byte 0: FLCO (bits 5-0) and Feature Set ID
	lc[0] = byte(flco) & 0x3F

	// Bytes 1-3: Target Address (Destination ID, 24-bit big-endian)
	lc[1] = byte(dstID >> 16)
	lc[2] = byte(dstID >> 8)
	lc[3] = byte(dstID)

	// Bytes 4-6: Source Address (Source ID, 24-bit big-endian)
	lc[4] = byte(srcID >> 16)
	lc[5] = byte(srcID >> 8)
	lc[6] = byte(srcID)

	// Bytes 7-8: Reserved/Options (set to 0)
	lc[7] = 0x00
	lc[8] = 0x00

	// Copy LC into payload (rest is FEC/padding, left as zeros for now)
	copy(payload[0:9], lc)

	// TODO: Add proper Reed-Solomon FEC encoding
	// For basic functionality, the LC data in clear may work on some systems

	return payload
}

// BuildVoiceTerminatorPayload builds a DMR Voice Terminator payload (33 bytes)
func BuildVoiceTerminatorPayload(srcID, dstID uint32, flco FLCO) []byte {
	// Terminator uses same LC structure as header
	return BuildVoiceLCHeader(srcID, dstID, flco)
}

// ParseVoiceLCHeader parses a Voice LC Header payload to extract source/dest/FLCO
func ParseVoiceLCHeader(payload []byte) (srcID, dstID uint32, flco FLCO, ok bool) {
	if len(payload) < 9 {
		return 0, 0, 0, false
	}

	// Extract FLCO
	flco = FLCO(payload[0] & 0x3F)

	// Extract destination ID (24-bit big-endian)
	dstID = uint32(payload[1])<<16 | uint32(payload[2])<<8 | uint32(payload[3])

	// Extract source ID (24-bit big-endian)
	srcID = uint32(payload[4])<<16 | uint32(payload[5])<<8 | uint32(payload[6])

	return srcID, dstID, flco, true
}

// BuildEmbeddedData builds embedded signalling data for voice frames
// Returns 16 bytes of embedded data (LC + sync)
func BuildEmbeddedData(srcID, dstID uint32, flco FLCO, colorCode byte) []byte {
	data := make([]byte, 16)

	// Simplified: embed partial LC in voice frames
	// Bytes 0-3: Partial LC fragments (would normally be interleaved)
	data[0] = byte(flco) & 0x3F
	data[1] = byte(dstID >> 16)
	data[2] = byte(dstID >> 8)
	data[3] = byte(dstID)

	// Rest is padding/sync (simplified)
	for i := 4; i < 16; i++ {
		data[i] = 0x00
	}

	return data
}

// EncodeColorCode encodes the color code into the sync pattern
func EncodeColorCode(colorCode byte) uint32 {
	// DMR color code encoding (simplified)
	// In real DMR, this is part of the CACH/SYNC pattern
	return uint32(colorCode & 0x0F)
}

// BuildVoiceSyncPayload builds a voice sync payload with embedded LC
func BuildVoiceSyncPayload(srcID, dstID uint32, flco FLCO) []byte {
	payload := make([]byte, 33)

	// Voice sync frame includes embedded signalling
	// Simplified version - real DMR has complex interleaving
	embedded := BuildEmbeddedData(srcID, dstID, flco, 1)
	copy(payload[0:16], embedded)

	return payload
}
