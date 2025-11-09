package ysf

import (
	"testing"
)

func TestGolay24128RoundTrip(t *testing.T) {
	tests := []uint32{
		0x000, 0x001, 0x002, 0x010, 0x020, 0x040, 0x080, 0x100, 0x200, 0x400, 0x800,
		0x123, 0x456, 0x789, 0xABC, 0xDEF,
		0xFFF, // Max 12-bit value
	}

	for _, data := range tests {
		// Encode
		encoded := Encode24128(data)

		// Decode without errors
		decoded := Decode24128Code(encoded)

		if decoded != data {
			t.Errorf("Round trip failed for data %03X: encoded=%06X, decoded=%03X", data, encoded, decoded)
		}
	}
}

func TestGolay24128ErrorCorrection(t *testing.T) {
	data := uint32(0x123)
	encoded := Encode24128(data)

	// Test single bit error
	corrupted := encoded ^ (1 << 10)
	decoded := Decode24128Code(corrupted)
	if decoded != data {
		t.Errorf("Single bit error correction failed: data=%03X, decoded=%03X", data, decoded)
	}

	// Test double bit error
	corrupted = encoded ^ (1 << 10) ^ (1 << 15)
	decoded = Decode24128Code(corrupted)
	if decoded != data {
		t.Errorf("Double bit error correction failed: data=%03X, decoded=%03X", data, decoded)
	}

	// Test triple bit error
	corrupted = encoded ^ (1 << 5) ^ (1 << 10) ^ (1 << 15)
	decoded = Decode24128Code(corrupted)
	if decoded != data {
		t.Errorf("Triple bit error correction failed: data=%03X, decoded=%03X", data, decoded)
	}
}

func TestGenerateGolay23(t *testing.T) {
	// Test that generateGolay23 produces valid codewords
	for data := uint32(0); data < 100; data++ {
		code := generateGolay23(data)

		// Check that data portion matches
		extractedData := (code >> 11) & 0xFFF
		if extractedData != data {
			t.Errorf("Data mismatch: input=%03X, extracted=%03X, code=%06X", data, extractedData, code)
		}

		// Verify it decodes correctly
		// Add parity bit (even parity)
		parity := uint32(0)
		temp := code
		for temp != 0 {
			parity ^= temp & 1
			temp >>= 1
		}
		fullCode := (code << 1) | parity

		decoded := Decode24128Code(fullCode)
		if decoded != data {
			t.Errorf("Generated code doesn't decode: data=%03X, code=%06X, decoded=%03X", data, code, decoded)
		}
	}
}
