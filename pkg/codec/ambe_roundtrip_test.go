package codec

import (
	"testing"
)

// TestAMBERoundtrip verifies that insert and extract are inverse operations
func TestAMBERoundtrip(t *testing.T) {
	// Test values
	origA := uint32(0xF32E06)
	origB := uint32(0x7C99ED)
	origC := uint32(0x5CEC4B)

	// Insert into 9-byte frame
	frame := make([]byte, 9)
	insertAMBEToDMR(frame, origA, origB, origC)

	t.Logf("Original: A=%06X B=%06X C=%06X", origA, origB, origC)
	t.Logf("Frame: %02X", frame)

	// Extract back
	gotA, gotB, gotC := extractSingleAMBEFromDMR(frame)

	t.Logf("Extracted: A=%06X B=%06X C=%06X", gotA, gotB, gotC)

	// Verify round-trip
	if gotA != origA {
		t.Errorf("A mismatch: want %06X, got %06X", origA, gotA)
	}
	if gotB != origB {
		t.Errorf("B mismatch: want %06X, got %06X", origB, gotB)
	}
	if gotC != origC {
		t.Errorf("C mismatch: want %06X, got %06X", origC, gotC)
	}
}
