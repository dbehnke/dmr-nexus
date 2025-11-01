package ysf

// CRC-CCITT (0x1021) implementation for YSF FICH validation
// Based on CRC.cpp from MMDVM_CM

const (
	crcCCITT16Poly = 0x1021
)

// AddCCITT162 adds CRC-CCITT checksum to data (last 2 bytes)
func AddCCITT162(data []byte) {
	if len(data) < 2 {
		return
	}

	crc := calculateCCITT162(data[:len(data)-2])
	data[len(data)-2] = byte(crc >> 8)
	data[len(data)-1] = byte(crc & 0xFF)
}

// CheckCCITT162 verifies CRC-CCITT checksum
func CheckCCITT162(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	crc := calculateCCITT162(data[:len(data)-2])
	return byte(crc>>8) == data[len(data)-2] && byte(crc&0xFF) == data[len(data)-1]
}

// calculateCCITT162 calculates CRC-CCITT for given data
func calculateCCITT162(data []byte) uint16 {
	crc := uint16(0xFFFF)

	for _, b := range data {
		crc ^= uint16(b) << 8

		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ crcCCITT16Poly
			} else {
				crc <<= 1
			}
		}
	}

	return ^crc
}
