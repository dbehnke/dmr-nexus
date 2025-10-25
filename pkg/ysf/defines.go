package ysf

// YSF protocol constants and definitions
// Based on YSFDefines.h from MMDVM_CM/YSF2DMR

const (
	// YSFCallsignLength is the length of YSF callsign fields
	YSFCallsignLength = 10

	// YSFFrameLength is the total length of a YSF frame
	YSFFrameLength = 155

	// YSFSyncLength is the length of the sync pattern
	YSFSyncLength = 5

	// YSFHeaderLength is the length of the YSF header
	YSFHeaderLength = 120

	// YSFFICHLength is the length of the FICH
	YSFFICHLength = 6
)

// Frame Information (FI) values
const (
	YSFFIHeader        = 0x00 // Header
	YSFFICommunication = 0x01 // Voice/Data
	YSFFITerminator    = 0x02 // Terminator
	YSFFITestFrame     = 0x03 // Test frame
)

// Data Type (DT) values
const (
	YSFDTVDMode1      = 0x00 // Voice/Data Mode 1
	YSFDTDataFullRate = 0x01 // Data Full Rate
	YSFDTVDMode2      = 0x02 // Voice/Data Mode 2
	YSFDTVoiceFR      = 0x03 // Voice Full Rate
)

// Sync patterns
var (
	YSFSyncBytes = []byte{0xD4, 0x71, 0xC9, 0x63, 0x4D}
)

// YSFFrame represents a complete YSF frame
type YSFFrame struct {
	Signature []byte // 4 bytes: "YSFD"
	Gateway   string // 10 bytes: Gateway callsign
	Source    string // 10 bytes: Source callsign
	Dest      string // 10 bytes: Destination callsign
	Counter   byte   // 1 byte: Frame counter
	Payload   []byte // 120 bytes: FICH + payload data
}

// YSFFICH represents the Frame Information Channel Header
type YSFFICH struct {
	FI          byte // Frame Information
	CS          byte // Communication Type / Channel ID
	CM          byte // Call Mode
	BN          byte // Block Number
	BT          byte // Block Type
	FN          byte // Frame Number
	FT          byte // Frame Total
	Dev         byte // Device Type
	MR          byte // Message Route
	VoIP        byte // VoIP flag
	DT          byte // Data Type
	SQL         byte // SQL Type
	SQ          byte // SQL Code
}

// NewYSFFrame creates a new YSF frame with default values
func NewYSFFrame() *YSFFrame {
	return &YSFFrame{
		Signature: []byte("YSFD"),
		Gateway:   padCallsign(""),
		Source:    padCallsign(""),
		Dest:      padCallsign("ALL"),
		Counter:   0,
		Payload:   make([]byte, YSFHeaderLength),
	}
}

// padCallsign pads a callsign to YSFCallsignLength with spaces
func padCallsign(cs string) string {
	if len(cs) > YSFCallsignLength {
		return cs[:YSFCallsignLength]
	}
	for len(cs) < YSFCallsignLength {
		cs += " "
	}
	return cs
}

// TrimCallsign removes trailing spaces from a callsign
func TrimCallsign(cs string) string {
	for len(cs) > 0 && cs[len(cs)-1] == ' ' {
		cs = cs[:len(cs)-1]
	}
	return cs
}
