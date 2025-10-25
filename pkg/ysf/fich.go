package ysf

import (
	"fmt"
)

// FICH encoding/decoding based on YSFFICH.cpp from MMDVM_CM
// Uses Golay(24,12), Viterbi convolution, and CRC-CCITT

const (
	ysfSyncLengthBytes = 5
)

// Interleave table for FICH bits
var interleaveTable = []uint{
	0, 40, 80, 120, 160,
	2, 42, 82, 122, 162,
	4, 44, 84, 124, 164,
	6, 46, 86, 126, 166,
	8, 48, 88, 128, 168,
	10, 50, 90, 130, 170,
	12, 52, 92, 132, 172,
	14, 54, 94, 134, 174,
	16, 56, 96, 136, 176,
	18, 58, 98, 138, 178,
	20, 60, 100, 140, 180,
	22, 62, 102, 142, 182,
	24, 64, 104, 144, 184,
	26, 66, 106, 146, 186,
	28, 68, 108, 148, 188,
	30, 70, 110, 150, 190,
	32, 72, 112, 152, 192,
	34, 74, 114, 154, 194,
	36, 76, 116, 156, 196,
	38, 78, 118, 158, 198,
}

// Encode encodes FICH data into the payload
func (f *YSFFICH) Encode(payload []byte) error {
	if len(payload) < ysfSyncLengthBytes+25 {
		return fmt.Errorf("payload too short for FICH encoding: %d", len(payload))
	}

	// Skip sync bytes
	bytes := payload[ysfSyncLengthBytes:]

	// Create 6-byte FICH data
	fich := make([]byte, 6)
	fich[0] = ((f.FI & 0x03) << 6) | ((f.CS & 0x03) << 4) | ((f.CM & 0x03) << 2) | (f.BN & 0x03)
	fich[1] = ((f.BT & 0x03) << 6) | ((f.FN & 0x07) << 3) | (f.FT & 0x07)
	fich[2] = ((f.MR & 0x03) << 3) | ((f.VoIP & 0x01) << 2) | (f.DT & 0x03)
	if f.Dev != 0 {
		fich[2] |= 0x40
	}
	fich[3] = f.SQ & 0x7F
	if f.SQL != 0 {
		fich[3] |= 0x80
	}

	// Add CRC
	AddCCITT162(fich)

	// Encode with Golay(24,12)
	b0 := ((uint32(fich[0]) << 4) & 0xFF0) | ((uint32(fich[1]) >> 4) & 0x00F)
	b1 := ((uint32(fich[1]) << 8) & 0xF00) | ((uint32(fich[2]) >> 0) & 0x0FF)
	b2 := ((uint32(fich[3]) << 4) & 0xFF0) | ((uint32(fich[4]) >> 4) & 0x00F)
	b3 := ((uint32(fich[4]) << 8) & 0xF00) | ((uint32(fich[5]) >> 0) & 0x0FF)

	c0 := Encode24128(b0)
	c1 := Encode24128(b1)
	c2 := Encode24128(b2)
	c3 := Encode24128(b3)

	// Pack into byte array for convolution
	conv := make([]byte, 13)
	conv[0] = byte((c0 >> 16) & 0xFF)
	conv[1] = byte((c0 >> 8) & 0xFF)
	conv[2] = byte((c0 >> 0) & 0xFF)
	conv[3] = byte((c1 >> 16) & 0xFF)
	conv[4] = byte((c1 >> 8) & 0xFF)
	conv[5] = byte((c1 >> 0) & 0xFF)
	conv[6] = byte((c2 >> 16) & 0xFF)
	conv[7] = byte((c2 >> 8) & 0xFF)
	conv[8] = byte((c2 >> 0) & 0xFF)
	conv[9] = byte((c3 >> 16) & 0xFF)
	conv[10] = byte((c3 >> 8) & 0xFF)
	conv[11] = byte((c3 >> 0) & 0xFF)
	conv[12] = 0x00

	// Convolutional encoding
	convolved := make([]byte, 25)
	convolution := NewYSFConvolution()
	convolution.Encode(conv, convolved, 100)

	// Interleave and write to payload
	j := uint(0)
	for i := uint(0); i < 100; i++ {
		n := interleaveTable[i]

		s0 := readBit(convolved, j)
		j++
		s1 := readBit(convolved, j)
		j++

		writeBit(bytes, n, s0)
		writeBit(bytes, n+1, s1)
	}

	return nil
}

// Decode decodes FICH data from the payload
func (f *YSFFICH) Decode(payload []byte) (bool, error) {
	if len(payload) < ysfSyncLengthBytes+25 {
		return false, fmt.Errorf("payload too short for FICH decoding: %d", len(payload))
	}

	// Skip sync bytes
	bytes := payload[ysfSyncLengthBytes:]

	// Initialize Viterbi decoder
	viterbi := NewYSFConvolution()
	viterbi.Start()

	// Deinterleave and feed to Viterbi
	for i := uint(0); i < 100; i++ {
		n := interleaveTable[i]
		var s0, s1 uint8

		if readBit(bytes, n) {
			s0 = 1
		}
		if readBit(bytes, n+1) {
			s1 = 1
		}

		viterbi.Decode(s0, s1)
	}

	// Chainback to get decoded bits
	output := make([]byte, 13)
	viterbi.Chainback(output, 96)

	// Decode Golay(24,12) codes
	b0 := Decode24128(output[0:3])
	b1 := Decode24128(output[3:6])
	b2 := Decode24128(output[6:9])
	b3 := Decode24128(output[9:12])

	// Reconstruct FICH bytes
	fich := make([]byte, 6)
	fich[0] = byte((b0 >> 4) & 0xFF)
	fich[1] = byte(((b0 << 4) & 0xF0) | ((b1 >> 8) & 0x0F))
	fich[2] = byte((b1 >> 0) & 0xFF)
	fich[3] = byte((b2 >> 4) & 0xFF)
	fich[4] = byte(((b2 << 4) & 0xF0) | ((b3 >> 8) & 0x0F))
	fich[5] = byte((b3 >> 0) & 0xFF)

	// Check CRC
	if !CheckCCITT162(fich) {
		return false, nil
	}

	// Parse FICH fields
	f.FI = (fich[0] >> 6) & 0x03
	f.CS = (fich[0] >> 4) & 0x03
	f.CM = (fich[0] >> 2) & 0x03
	f.BN = fich[0] & 0x03
	f.BT = (fich[1] >> 6) & 0x03
	f.FN = (fich[1] >> 3) & 0x07
	f.FT = fich[1] & 0x07
	f.DT = fich[2] & 0x03
	f.MR = (fich[2] >> 3) & 0x03
	if fich[2]&0x40 != 0 {
		f.Dev = 1
	} else {
		f.Dev = 0
	}
	if fich[2]&0x04 != 0 {
		f.VoIP = 1
	} else {
		f.VoIP = 0
	}
	if fich[3]&0x80 != 0 {
		f.SQL = 1
	} else {
		f.SQL = 0
	}
	f.SQ = fich[3] & 0x7F

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
