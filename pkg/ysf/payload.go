package ysf

import (
	"fmt"
)

// YSFPayload handles YSF payload encoding and decoding
// Based on YSFPayload.cpp from MMDVM_CM

// YSFPayload represents YSF payload data
type YSFPayload struct {
	source string
	dest   string
}

// NewYSFPayload creates a new YSF payload processor
func NewYSFPayload() *YSFPayload {
	return &YSFPayload{}
}

// ProcessHeaderData processes header data from a YSF frame
// Returns true if valid header was processed
func (p *YSFPayload) ProcessHeaderData(payload []byte) (bool, error) {
	if len(payload) < YSFHeaderLength {
		return false, fmt.Errorf("payload too short: %d", len(payload))
	}

	// Extract CSD1 and CSD2 from VD Mode 2 data
	// CSD1 contains source callsign
	// CSD2 contains destination callsign
	csd1 := make([]byte, 20)
	csd2 := make([]byte, 20)

	// Read CSD1 (bytes 20-39 of payload after FICH)
	copy(csd1, payload[20:40])

	// Read CSD2 (bytes 40-59 of payload after FICH)
	copy(csd2, payload[40:60])

	// Extract callsigns from CSD data
	// First 10 bytes of CSD1 (after radio ID) contain source callsign
	p.source = string(csd1[10:20])
	p.source = TrimCallsign(p.source)

	// First 10 bytes of CSD2 contain destination callsign
	p.dest = string(csd2[0:10])
	p.dest = TrimCallsign(p.dest)

	return len(p.source) > 0, nil
}

// GetSource returns the source callsign from the last processed header
func (p *YSFPayload) GetSource() string {
	return p.source
}

// GetDest returns the destination callsign from the last processed header
func (p *YSFPayload) GetDest() string {
	return p.dest
}

// WriteHeader writes header information into a YSF payload
func (p *YSFPayload) WriteHeader(payload []byte, csd1, csd2 []byte) error {
	if len(payload) < YSFHeaderLength {
		return fmt.Errorf("payload too short: %d", len(payload))
	}

	if len(csd1) != 20 || len(csd2) != 20 {
		return fmt.Errorf("invalid CSD length: csd1=%d, csd2=%d", len(csd1), len(csd2))
	}

	// Write CSD1 (contains radio ID and source callsign)
	copy(payload[20:40], csd1)

	// Write CSD2 (contains destination callsign)
	copy(payload[40:60], csd2)

	return nil
}

// WriteVDMode2Data writes VD Mode 2 data into a YSF payload
func (p *YSFPayload) WriteVDMode2Data(payload []byte, data []byte) error {
	if len(payload) < YSFHeaderLength {
		return fmt.Errorf("payload too short: %d", len(payload))
	}

	if len(data) != 10 {
		return fmt.Errorf("invalid data length: %d (expected 10)", len(data))
	}

	// Write VD Mode 2 data into DCH (Data Channel) area
	// DCH is at specific positions in the YSF frame
	copy(payload[20:30], data)

	return nil
}

// ReadVDMode2Data reads VD Mode 2 data from a YSF payload
func (p *YSFPayload) ReadVDMode2Data(payload []byte) ([]byte, error) {
	if len(payload) < YSFHeaderLength {
		return nil, fmt.Errorf("payload too short: %d", len(payload))
	}

	data := make([]byte, 10)
	copy(data, payload[20:30])

	return data, nil
}

// ExtractAMBE extracts AMBE voice data from YSF payload
// Returns AMBE frames for codec conversion
func (p *YSFPayload) ExtractAMBE(payload []byte) ([][]byte, error) {
	if len(payload) < YSFHeaderLength {
		return nil, fmt.Errorf("payload too short: %d", len(payload))
	}

	// YSF uses AMBE encoding for voice
	// Extract AMBE frames from specific positions in the payload
	// VD Mode 2 contains voice data interleaved with data channel

	ambeFrames := make([][]byte, 2)

	// Extract first AMBE frame (49 bits = ~7 bytes)
	ambeFrames[0] = make([]byte, 9)
	copy(ambeFrames[0], payload[60:69])

	// Extract second AMBE frame
	ambeFrames[1] = make([]byte, 9)
	copy(ambeFrames[1], payload[90:99])

	return ambeFrames, nil
}

// InsertAMBE inserts AMBE voice data into YSF payload
func (p *YSFPayload) InsertAMBE(payload []byte, ambeFrames [][]byte) error {
	if len(payload) < YSFHeaderLength {
		return fmt.Errorf("payload too short: %d", len(payload))
	}

	if len(ambeFrames) != 2 {
		return fmt.Errorf("expected 2 AMBE frames, got %d", len(ambeFrames))
	}

	// Insert first AMBE frame
	if len(ambeFrames[0]) >= 9 {
		copy(payload[60:69], ambeFrames[0][:9])
	}

	// Insert second AMBE frame
	if len(ambeFrames[1]) >= 9 {
		copy(payload[90:99], ambeFrames[1][:9])
	}

	return nil
}
