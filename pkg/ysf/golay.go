package ysf

// Golay(24,12) encoder/decoder
// Based on Golay24128.cpp from MMDVM_CM by Jonathan Naylor G4KLX

// Encode24128 encodes 12-bit data into 24-bit Golay codeword
func Encode24128(data uint32) uint32 {
	// Truncate to 12 bits
	data &= 0xFFF

	// Look up in encoding table
	if data < uint32(len(encodingTable23127)) {
		code := encodingTable23127[data]
		// Add parity bit
		parity := uint32(0)
		temp := code
		for temp != 0 {
			parity ^= temp & 1
			temp >>= 1
		}
		return (code << 1) | parity
	}
	return 0
}

// Decode24128 decodes 24-bit Golay codeword (from byte array)
func Decode24128(bytes []byte) uint32 {
	if len(bytes) < 3 {
		return 0
	}

	code := (uint32(bytes[0]) << 16) | (uint32(bytes[1]) << 8) | uint32(bytes[2])
	return Decode24128Code(code)
}

// Decode24128Code decodes a 24-bit Golay codeword
func Decode24128Code(code uint32) uint32 {
	// Remove parity bit
	code >>= 1
	code &= 0x7FFFFF // 23 bits

	// Try to find syndrome
	syndrome := calculateSyndrome23(code)
	if syndrome == 0 {
		// No errors
		return code >> 11
	}

	// Check if syndrome is correctable (weight <= 3)
	weight := hammingWeight(syndrome)
	if weight <= 3 {
		// Error in parity bits only
		return code >> 11
	}

	// Try to find error pattern
	for i := uint32(0); i < 12; i++ {
		// Try single bit error in data
		testCode := code ^ (1 << (23 - i))
		testSyndrome := calculateSyndrome23(testCode)
		if testSyndrome == 0 {
			return testCode >> 11
		}
	}

	// Use minimum distance decoding
	minDist := 24
	bestMatch := uint32(0)

	for i := uint32(0); i < 4096; i++ {
		encoded := encodingTable23127[i]
		dist := hammingDistance23(code, encoded)
		if dist < minDist {
			minDist = dist
			bestMatch = i
		}
		if dist == 0 {
			break
		}
	}

	return bestMatch
}

// calculateSyndrome23 calculates syndrome for 23-bit Golay code
func calculateSyndrome23(code uint32) uint32 {
	syndrome := uint32(0)
	for i := uint32(0); i < 12; i++ {
		if code&(1<<(22-i)) != 0 {
			syndrome ^= syndromeTable[i]
		}
	}
	// Include parity bits
	syndrome ^= (code & 0x7FF)
	return syndrome
}

// hammingWeight returns number of 1 bits
func hammingWeight(x uint32) int {
	count := 0
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

// hammingDistance23 calculates Hamming distance between two 23-bit values
func hammingDistance23(a, b uint32) int {
	return hammingWeight((a ^ b) & 0x7FFFFF)
}

// Golay(23,12) generator polynomial: x^11 + x^10 + x^6 + x^5 + x^4 + x^2 + 1
// Syndrome lookup table for error correction
var syndromeTable = []uint32{
	0x400, 0x200, 0x100, 0x080, 0x040, 0x020,
	0x010, 0x008, 0x004, 0x002, 0x001, 0x600,
}

// Golay(23,12) encoding table - encodes 12-bit data to 23-bit code
// This is a partial table - full table would have 4096 entries
var encodingTable23127 = []uint32{
	0x000000, 0x0018EA, 0x00293E, 0x0031D4, 0x004A96, 0x00527C, 0x0063A8, 0x007B42,
	0x008DC6, 0x00952C, 0x00A4F8, 0x00BC12, 0x00C750, 0x00DFBA, 0x00EE6E, 0x00F684,
	0x010366, 0x011B8C, 0x012A58, 0x0132B2, 0x0149F0, 0x01511A, 0x0160CE, 0x017824,
	0x018EA0, 0x01964A, 0x01A79E, 0x01BF74, 0x01C436, 0x01DCDC, 0x01ED08, 0x01F5E2,
	0x0206CC, 0x021E26, 0x022FF2, 0x023718, 0x024C5A, 0x0254B0, 0x026564, 0x027D8E,
	0x028B0A, 0x0293E0, 0x02A234, 0x02BADE, 0x02C19C, 0x02D976, 0x02E8A2, 0x02F048,
	0x0305AA, 0x031D40, 0x032C94, 0x03347E, 0x034F3C, 0x0357D6, 0x036602, 0x037EE8,
	0x03886C, 0x039086, 0x03A152, 0x03B9B8, 0x03C2FA, 0x03DA10, 0x03EBC4, 0x03F32E,
	// Additional entries would continue...
	// For brevity, including first 64 entries. Full implementation would need all 4096.
}

func init() {
	// Generate full encoding table at runtime if not complete
	if len(encodingTable23127) < 4096 {
		// Generate remaining entries using polynomial multiplication
		newTable := make([]uint32, 4096)
		copy(newTable, encodingTable23127)

		// Generator polynomial coefficients
		gen := uint32(0xC75) // x^11 + x^10 + x^6 + x^5 + x^4 + x^2 + 1

		for i := len(encodingTable23127); i < 4096; i++ {
			data := uint32(i)
			code := data << 11

			// Polynomial division
			for j := 11; j >= 0; j-- {
				if code&(1<<(j+11)) != 0 {
					code ^= gen << j
				}
			}

			newTable[i] = (data << 11) | code
		}
		encodingTable23127 = newTable
	}
}
