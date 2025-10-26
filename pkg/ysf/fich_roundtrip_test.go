package ysf

import (
	"testing"
)

// TestFICHRoundtrip tests that we can encode and decode FICH correctly
func TestFICHRoundtrip(t *testing.T) {
	// Create a FICH with known values
	original := &YSFFICH{
		FI:   1,
		CS:   2,
		CM:   0,
		BN:   0,
		BT:   1,
		FN:   3,
		FT:   7,
		DT:   2,
		MR:   0,
		Dev:  0,
		VoIP: 1,
		SQ:   0x25,
		SQL:  0,
	}

	// Create payload buffer with sync bytes
	payload := make([]byte, 155)
	// Set sync pattern
	payload[0] = 0xD4
	payload[1] = 0x71
	payload[2] = 0xC9
	payload[3] = 0x63
	payload[4] = 0x4D

	// Encode FICH into payload
	err := original.Encode(payload)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Now decode it back
	decoded := &YSFFICH{}
	valid, err := decoded.Decode(payload)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if !valid {
		t.Fatal("Decode returned invalid (CRC failed)")
	}

	// Compare all fields
	if decoded.FI != original.FI {
		t.Errorf("FI mismatch: got %d, want %d", decoded.FI, original.FI)
	}
	if decoded.CS != original.CS {
		t.Errorf("CS mismatch: got %d, want %d", decoded.CS, original.CS)
	}
	if decoded.CM != original.CM {
		t.Errorf("CM mismatch: got %d, want %d", decoded.CM, original.CM)
	}
	if decoded.BN != original.BN {
		t.Errorf("BN mismatch: got %d, want %d", decoded.BN, original.BN)
	}
	if decoded.BT != original.BT {
		t.Errorf("BT mismatch: got %d, want %d", decoded.BT, original.BT)
	}
	if decoded.FN != original.FN {
		t.Errorf("FN mismatch: got %d, want %d", decoded.FN, original.FN)
	}
	if decoded.FT != original.FT {
		t.Errorf("FT mismatch: got %d, want %d", decoded.FT, original.FT)
	}
	if decoded.DT != original.DT {
		t.Errorf("DT mismatch: got %d, want %d", decoded.DT, original.DT)
	}
	if decoded.MR != original.MR {
		t.Errorf("MR mismatch: got %d, want %d", decoded.MR, original.MR)
	}
	if decoded.Dev != original.Dev {
		t.Errorf("Dev mismatch: got %d, want %d", decoded.Dev, original.Dev)
	}
	if decoded.VoIP != original.VoIP {
		t.Errorf("VoIP mismatch: got %d, want %d", decoded.VoIP, original.VoIP)
	}
	if decoded.SQ != original.SQ {
		t.Errorf("SQ mismatch: got %d, want %d", decoded.SQ, original.SQ)
	}
	if decoded.SQL != original.SQL {
		t.Errorf("SQL mismatch: got %d, want %d", decoded.SQL, original.SQL)
	}
}
