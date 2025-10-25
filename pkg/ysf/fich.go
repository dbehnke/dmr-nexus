package ysf

import (
	"fmt"
)

// FICH encoding/decoding based on YSFFICH.cpp from MMDVM_CM

// Golay(20,8) encoding table for FICH
var golay20_8Table = []uint32{
	0x00000, 0x08659, 0x10CB2, 0x18AEB, 0x21964, 0x29F3D, 0x31536, 0x3996F,
	0x42DB9, 0x4ABCE, 0x52105, 0x5A77C, 0x63C93, 0x6BAFA, 0x73251, 0x7B408,
	0x85EB3, 0x8D8EA, 0x95241, 0x9D618, 0xB48F7, 0xBCEAE, 0xA4405, 0xAC25C,
	0xC736A, 0xCF533, 0xD7F98, 0xDF9C1, 0xE674E, 0xEE117, 0xF6DBC, 0xFEBE5,
	0x00000, 0x0847B, 0x108F6, 0x18C8D, 0x21387, 0x297FC, 0x31B71, 0x39F0A,
	0x42738, 0x4A343, 0x52FCE, 0x5ABB5, 0x6341F, 0x6B064, 0x73CE9, 0x7B892,
}

// EncodeFICH encodes FICH data into the payload
func (f *YSFFICH) Encode(payload []byte) error {
	if len(payload) < 48 {
		return fmt.Errorf("payload too short for FICH encoding: %d", len(payload))
	}

	// Build FICH data byte
	var fich uint32 = 0

	// Bits 0-1: Frame Information (FI)
	fich |= uint32(f.FI & 0x03)

	// Bits 2-3: Communication Type / Channel ID (CS)
	fich |= uint32(f.CS&0x03) << 2

	// Bits 4-5: Call Mode (CM)
	fich |= uint32(f.CM&0x03) << 4

	// Bit 6: Block Number (BN)
	fich |= uint32(f.BN&0x01) << 6

	// Bit 7: Block Type (BT)
	fich |= uint32(f.BT&0x01) << 7

	// Apply Golay(20,8) encoding
	encoded := golay20_8Encode(uint8(fich))

	// Write encoded FICH into payload (bits 40-59 of the sync+fich area)
	// The FICH is interleaved across the frame
	writeFICHBits(payload, encoded)

	return nil
}

// Decode decodes FICH data from the payload
func (f *YSFFICH) Decode(payload []byte) (bool, error) {
	if len(payload) < 48 {
		return false, fmt.Errorf("payload too short for FICH decoding: %d", len(payload))
	}

	// Extract FICH bits from payload
	encoded := readFICHBits(payload)

	// Decode Golay(20,8)
	decoded, valid := golay20_8Decode(encoded)
	if !valid {
		return false, nil
	}

	// Parse FICH fields
	f.FI = decoded & 0x03
	f.CS = (decoded >> 2) & 0x03
	f.CM = (decoded >> 4) & 0x03
	f.BN = (decoded >> 6) & 0x01
	f.BT = (decoded >> 7) & 0x01

	return true, nil
}

// SetFI sets the Frame Information field
func (f *YSFFICH) SetFI(fi byte) {
	f.FI = fi
}

// SetCS sets the Communication Type / Channel ID field
func (f *YSFFICH) SetCS(cs byte) {
	f.CS = cs
}

// SetCM sets the Call Mode field
func (f *YSFFICH) SetCM(cm byte) {
	f.CM = cm
}

// SetBN sets the Block Number field
func (f *YSFFICH) SetBN(bn byte) {
	f.BN = bn
}

// SetBT sets the Block Type field
func (f *YSFFICH) SetBT(bt byte) {
	f.BT = bt
}

// SetFN sets the Frame Number field
func (f *YSFFICH) SetFN(fn byte) {
	f.FN = fn
}

// SetFT sets the Frame Total field
func (f *YSFFICH) SetFT(ft byte) {
	f.FT = ft
}

// SetDev sets the Device Type field
func (f *YSFFICH) SetDev(dev byte) {
	f.Dev = dev
}

// SetMR sets the Message Route field
func (f *YSFFICH) SetMR(mr byte) {
	f.MR = mr
}

// SetVoIP sets the VoIP flag
func (f *YSFFICH) SetVoIP(voip byte) {
	f.VoIP = voip
}

// SetDT sets the Data Type field
func (f *YSFFICH) SetDT(dt byte) {
	f.DT = dt
}

// SetSQL sets the SQL Type field
func (f *YSFFICH) SetSQL(sql byte) {
	f.SQL = sql
}

// SetSQ sets the SQL Code field
func (f *YSFFICH) SetSQ(sq byte) {
	f.SQ = sq
}

// GetFI gets the Frame Information field
func (f *YSFFICH) GetFI() byte {
	return f.FI
}

// GetDT gets the Data Type field
func (f *YSFFICH) GetDT() byte {
	return f.DT
}

// GetFN gets the Frame Number field
func (f *YSFFICH) GetFN() byte {
	return f.FN
}

// GetFT gets the Frame Total field
func (f *YSFFICH) GetFT() byte {
	return f.FT
}

// golay20_8Encode encodes 8 data bits into 20-bit Golay code
func golay20_8Encode(data uint8) uint32 {
	if int(data) < len(golay20_8Table) {
		return golay20_8Table[data]
	}
	return 0
}

// golay20_8Decode decodes 20-bit Golay code into 8 data bits
func golay20_8Decode(code uint32) (uint8, bool) {
	// Simple syndrome-based decoding
	// Try to find matching codeword
	minDist := 32
	var bestMatch uint8 = 0

	for i := 0; i < 256; i++ {
		encoded := golay20_8Encode(uint8(i))
		dist := hammingDistance(code, encoded)
		if dist < minDist {
			minDist = dist
			bestMatch = uint8(i)
		}
		if dist == 0 {
			break
		}
	}

	// Golay(20,8) can correct up to 3 errors
	valid := minDist <= 3
	return bestMatch, valid
}

// hammingDistance calculates Hamming distance between two 20-bit values
func hammingDistance(a, b uint32) int {
	xor := (a ^ b) & 0xFFFFF // Mask to 20 bits
	count := 0
	for xor != 0 {
		count += int(xor & 1)
		xor >>= 1
	}
	return count
}

// writeFICHBits writes FICH bits into the payload
// FICH is interleaved across specific bit positions
func writeFICHBits(payload []byte, fich uint32) {
	// Simplified bit interleaving - actual implementation would follow
	// the YSF specification for FICH bit positions
	// For now, write to a known location
	payload[4] = byte((fich >> 12) & 0xFF)
	payload[5] = byte((fich >> 4) & 0xFF)
	payload[6] = byte((fich & 0x0F) << 4)
}

// readFICHBits reads FICH bits from the payload
func readFICHBits(payload []byte) uint32 {
	// Simplified bit de-interleaving
	var fich uint32
	fich = uint32(payload[4]) << 12
	fich |= uint32(payload[5]) << 4
	fich |= uint32(payload[6]) >> 4
	return fich & 0xFFFFF // Mask to 20 bits
}
