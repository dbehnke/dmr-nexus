package ysf

import (
	"testing"
)

func TestPadCallsign(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"KB3EFE", "KB3EFE    "},
		{"N0CALL", "N0CALL    "},
		{"", "          "},
		{"VERYLONGCALLSIGN", "VERYLONGCA"}, // Should truncate
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := padCallsign(tt.input)
			if result != tt.expected {
				t.Errorf("padCallsign(%q) = %q, want %q", tt.input, result, tt.expected)
			}
			if len(result) != YSFCallsignLength {
				t.Errorf("padCallsign(%q) length = %d, want %d", tt.input, len(result), YSFCallsignLength)
			}
		})
	}
}

func TestTrimCallsign(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"KB3EFE    ", "KB3EFE"},
		{"N0CALL", "N0CALL"},
		{"   ", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TrimCallsign(tt.input)
			if result != tt.expected {
				t.Errorf("TrimCallsign(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewYSFFrame(t *testing.T) {
	frame := NewYSFFrame()

	if string(frame.Signature) != "YSFD" {
		t.Errorf("Frame signature = %s, want YSFD", string(frame.Signature))
	}

	if len(frame.Gateway) != YSFCallsignLength {
		t.Errorf("Gateway length = %d, want %d", len(frame.Gateway), YSFCallsignLength)
	}

	if len(frame.Source) != YSFCallsignLength {
		t.Errorf("Source length = %d, want %d", len(frame.Source), YSFCallsignLength)
	}

	if len(frame.Dest) != YSFCallsignLength {
		t.Errorf("Dest length = %d, want %d", len(frame.Dest), YSFCallsignLength)
	}

	if len(frame.Payload) != YSFHeaderLength {
		t.Errorf("Payload length = %d, want %d", len(frame.Payload), YSFHeaderLength)
	}

	if TrimCallsign(frame.Dest) != "ALL" {
		t.Errorf("Default dest = %q, want ALL", TrimCallsign(frame.Dest))
	}
}

func TestFICHConstants(t *testing.T) {
	if YSFFIHeader != 0x00 {
		t.Errorf("YSFFIHeader = 0x%02x, want 0x00", YSFFIHeader)
	}

	if YSFFICommunication != 0x01 {
		t.Errorf("YSFFICommunication = 0x%02x, want 0x01", YSFFICommunication)
	}

	if YSFFITerminator != 0x02 {
		t.Errorf("YSFFITerminator = 0x%02x, want 0x02", YSFFITerminator)
	}
}

func TestDataTypeConstants(t *testing.T) {
	if YSFDTVDMode1 != 0x00 {
		t.Errorf("YSFDTVDMode1 = 0x%02x, want 0x00", YSFDTVDMode1)
	}

	if YSFDTVDMode2 != 0x02 {
		t.Errorf("YSFDTVDMode2 = 0x%02x, want 0x02", YSFDTVDMode2)
	}
}
