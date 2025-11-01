package ysf

// Golay(24,12) encoder/decoder
// Based on Golay24128.cpp from MMDVM_CM by Jonathan Naylor G4KLX

// Encode24128 encodes 12-bit data into 24-bit Golay codeword (with parity)
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

// Encode23127 encodes 12-bit data into 23-bit Golay codeword (without parity)
func Encode23127(data uint32) uint32 {
	// Truncate to 12 bits
	data &= 0xFFF

	// Look up in encoding table
	if data < uint32(len(encodingTable23127)) {
		return encodingTable23127[data]
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
// Uses syndrome-based decoding for error correction
func Decode24128Code(code uint32) uint32 {
	// Extract the 23-bit code (remove parity bit)
	code23 := (code >> 1) & 0x7FFFFF

	// Try syndrome decoding - XOR with all valid codewords
	// and find the one with minimum Hamming distance
	minDistance := 24
	bestData := uint32(0)

	// Search through all 4096 possible data values
	for data := uint32(0); data < 4096; data++ {
		var validCode uint32
		if data < uint32(len(encodingTable23127)) {
			validCode = encodingTable23127[data]
		} else {
			// Shouldn't happen since table has all 4096 entries after init
			validCode = generateGolay23(data)
		}

		// Calculate Hamming distance
		distance := hammingWeight((code23 ^ validCode) & 0x7FFFFF)

		if distance < minDistance {
			minDistance = distance
			bestData = data

			// If we found a perfect match, we can stop
			if distance == 0 {
				break
			}
		}
	}

	// Golay(24,12) can correct up to 3 bit errors
	// If distance > 3, the data may be unreliable but we return the best match
	return bestData
}

// generateGolay23 generates Golay(23,12) codeword for given 12-bit data
func generateGolay23(data uint32) uint32 {
	// Generator polynomial: x^11 + x^10 + x^6 + x^5 + x^4 + x^2 + 1
	data &= 0xFFF
	code := data << 11

	// XOR with generator polynomial for each bit position
	gen := uint32(0xC75) // Binary: 110001110101

	for i := 11; i >= 0; i-- {
		if code&(1<<(i+11)) != 0 {
			code ^= gen << i
		}
	}

	// Result is data bits concatenated with parity bits
	return (data << 11) | (code & 0x7FF)
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

// Golay(23,12) encoding table - encodes 12-bit data to 23-bit code
// Fully generated at init to ensure consistency with generator polynomial
var encodingTable23127 []uint32

func init() {
	// Generate full encoding table at runtime
	encodingTable23127 = make([]uint32, 4096)
	for i := 0; i < 4096; i++ {
		encodingTable23127[i] = generateGolay23(uint32(i))
	}
}
