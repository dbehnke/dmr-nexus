package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Test RPTL (Login Request) packet
func TestRPTLPacket_Parse(t *testing.T) {
	data := make([]byte, RPTLPacketSize)
	copy(data[0:4], []byte("RPTL"))
	// Repeater ID: 312000 (0x0004C2C0)
	data[4] = 0x00
	data[5] = 0x04
	data[6] = 0xC2
	data[7] = 0xC0

	packet := &RPTLPacket{}
	err := packet.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse RPTL packet: %v", err)
	}

	if packet.RepeaterID != 312000 {
		t.Errorf("Expected repeater ID 312000, got %d", packet.RepeaterID)
	}
}

func TestRPTLPacket_Encode(t *testing.T) {
	packet := &RPTLPacket{
		RepeaterID: 312000,
	}

	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Failed to encode RPTL packet: %v", err)
	}

	if len(data) != RPTLPacketSize {
		t.Errorf("Expected size %d, got %d", RPTLPacketSize, len(data))
	}

	if !bytes.Equal(data[0:4], []byte("RPTL")) {
		t.Error("Invalid signature in encoded packet")
	}
}

func TestRPTLPacket_RoundTrip(t *testing.T) {
	original := &RPTLPacket{RepeaterID: 999999}

	data, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	parsed := &RPTLPacket{}
	err = parsed.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.RepeaterID != original.RepeaterID {
		t.Errorf("RepeaterID mismatch: got %d, want %d", parsed.RepeaterID, original.RepeaterID)
	}
}

// Test RPTK (Key Exchange) packet
func TestRPTKPacket_Parse(t *testing.T) {
	data := make([]byte, RPTKPacketSize)
	copy(data[0:4], []byte("RPTK"))
	// Repeater ID: 312000
	data[4] = 0x00
	data[5] = 0x04
	data[6] = 0xC2
	data[7] = 0xC0
	// Challenge: 32 bytes starting at offset 8
	for i := 0; i < ChallengeLength; i++ {
		data[8+i] = byte(i)
	}

	packet := &RPTKPacket{}
	err := packet.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse RPTK packet: %v", err)
	}

	if packet.RepeaterID != 312000 {
		t.Errorf("Expected repeater ID 312000, got %d", packet.RepeaterID)
	}

	if len(packet.Challenge) != ChallengeLength {
		t.Errorf("Expected challenge length %d, got %d", ChallengeLength, len(packet.Challenge))
	}

	// Verify challenge content
	for i := 0; i < ChallengeLength; i++ {
		if packet.Challenge[i] != byte(i) {
			t.Errorf("Challenge byte %d: expected %d, got %d", i, byte(i), packet.Challenge[i])
			break
		}
	}
}

func TestRPTKPacket_Encode(t *testing.T) {
	challenge := make([]byte, ChallengeLength)
	for i := range challenge {
		challenge[i] = byte(i * 2)
	}

	packet := &RPTKPacket{
		RepeaterID: 312000,
		Challenge:  challenge,
	}

	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Failed to encode RPTK packet: %v", err)
	}

	if len(data) != RPTKPacketSize {
		t.Errorf("Expected size %d, got %d", RPTKPacketSize, len(data))
	}

	if !bytes.Equal(data[0:4], []byte("RPTK")) {
		t.Error("Invalid signature in encoded packet")
	}
}

// Test RPTC (Configuration) packet
func TestRPTCPacket_Parse(t *testing.T) {
	data := make([]byte, RPTCPacketSize)
	copy(data[0:4], []byte("RPTC"))
	// Repeater ID
	data[4] = 0x00
	data[5] = 0x04
	data[6] = 0xC2
	data[7] = 0xC0
	// Callsign (8 bytes at offset 8)
	copy(data[8:16], []byte("W1ABC   "))
	// Various config fields...
	// RX Freq (9 bytes at offset 16)
	copy(data[16:25], []byte("449000000"))
	// TX Freq (9 bytes at offset 25)
	copy(data[25:34], []byte("444000000"))

	packet := &RPTCPacket{}
	err := packet.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse RPTC packet: %v", err)
	}

	if packet.RepeaterID != 312000 {
		t.Errorf("Expected repeater ID 312000, got %d", packet.RepeaterID)
	}

	if packet.Callsign != "W1ABC" {
		t.Errorf("Expected callsign W1ABC, got %s", packet.Callsign)
	}

	if packet.RXFreq != "449000000" {
		t.Errorf("Expected RX freq 449000000, got %s", packet.RXFreq)
	}

	if packet.TXFreq != "444000000" {
		t.Errorf("Expected TX freq 444000000, got %s", packet.TXFreq)
	}
}

func TestRPTCPacket_Encode(t *testing.T) {
	packet := &RPTCPacket{
		RepeaterID:  312000,
		Callsign:    "W1ABC",
		RXFreq:      "449000000",
		TXFreq:      "444000000",
		TXPower:     "25",
		ColorCode:   "1",
		Latitude:    "42.3601",
		Longitude:   "-71.0589",
		Height:      "75",
		Location:    "Boston, MA",
		Description: "Test Repeater",
		URL:         "https://example.com",
		SoftwareID:  "DMR-Nexus",
		PackageID:   "DMR-Nexus",
	}

	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Failed to encode RPTC packet: %v", err)
	}

	if len(data) != RPTCPacketSize {
		t.Errorf("Expected size %d, got %d", RPTCPacketSize, len(data))
	}

	if !bytes.Equal(data[0:4], []byte("RPTC")) {
		t.Error("Invalid signature in encoded packet")
	}
}

// Test RPTACK (Acknowledgement) packet
func TestRPTACKPacket_Parse(t *testing.T) {
	data := make([]byte, RPTACKPacketSize)
	copy(data[0:6], []byte("RPTACK"))
	// Repeater ID
	data[6] = 0x00
	data[7] = 0x04
	data[8] = 0xC2
	data[9] = 0xC0

	packet := &RPTACKPacket{}
	err := packet.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse RPTACK packet: %v", err)
	}

	if packet.RepeaterID != 312000 {
		t.Errorf("Expected repeater ID 312000, got %d", packet.RepeaterID)
	}
}

func TestRPTACKPacket_Encode(t *testing.T) {
	packet := &RPTACKPacket{
		RepeaterID: 312000,
	}

	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Failed to encode RPTACK packet: %v", err)
	}

	if len(data) != RPTACKPacketSize {
		t.Errorf("Expected size %d, got %d", RPTACKPacketSize, len(data))
	}

	if !bytes.Equal(data[0:6], []byte("RPTACK")) {
		t.Error("Invalid signature in encoded packet")
	}
}

func TestRPTACKPacket_EncodeWithSalt(t *testing.T) {
	salt := []byte{0x01, 0x02, 0x03, 0x04}
	packet := &RPTACKPacket{
		RepeaterID: 312000,
		Salt:       salt,
	}

	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Failed to encode RPTACK packet with salt: %v", err)
	}

	if len(data) != RPTACKPacketSizeWithSalt {
		t.Errorf("Expected size %d, got %d", RPTACKPacketSizeWithSalt, len(data))
	}

	if !bytes.Equal(data[0:6], []byte("RPTACK")) {
		t.Error("Invalid signature in encoded packet")
	}

	if !bytes.Equal(data[6:10], salt) {
		t.Errorf("Expected salt %v, got %v", salt, data[6:10])
	}

	repeaterID := binary.BigEndian.Uint32(data[10:14])
	if repeaterID != 312000 {
		t.Errorf("Expected repeater ID 312000, got %d", repeaterID)
	}
}

func TestRPTACKPacket_ParseWithSalt(t *testing.T) {
	data := make([]byte, RPTACKPacketSizeWithSalt)
	copy(data[0:6], []byte("RPTACK"))
	// Salt
	data[6] = 0x01
	data[7] = 0x02
	data[8] = 0x03
	data[9] = 0x04
	// Repeater ID
	data[10] = 0x00
	data[11] = 0x04
	data[12] = 0xC2
	data[13] = 0xC0

	packet := &RPTACKPacket{}
	err := packet.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse RPTACK packet with salt: %v", err)
	}

	if packet.RepeaterID != 312000 {
		t.Errorf("Expected repeater ID 312000, got %d", packet.RepeaterID)
	}

	expectedSalt := []byte{0x01, 0x02, 0x03, 0x04}
	if !bytes.Equal(packet.Salt, expectedSalt) {
		t.Errorf("Expected salt %v, got %v", expectedSalt, packet.Salt)
	}
}

func TestRPTACKPacket_RoundTripWithSalt(t *testing.T) {
	salt := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	original := &RPTACKPacket{
		RepeaterID: 312000,
		Salt:       salt,
	}

	data, err := original.Encode()
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	parsed := &RPTACKPacket{}
	err = parsed.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if parsed.RepeaterID != original.RepeaterID {
		t.Errorf("RepeaterID mismatch: expected %d, got %d", original.RepeaterID, parsed.RepeaterID)
	}

	if !bytes.Equal(parsed.Salt, original.Salt) {
		t.Errorf("Salt mismatch: expected %v, got %v", original.Salt, parsed.Salt)
	}
}

// Test RPTPING (Keepalive ping) packet
func TestRPTPINGPacket_Parse(t *testing.T) {
	data := make([]byte, RPTPINGPacketSize)
	copy(data[0:7], []byte("RPTPING"))
	// Repeater ID
	data[7] = 0x00
	data[8] = 0x04
	data[9] = 0xC2
	data[10] = 0xC0

	packet := &RPTPINGPacket{}
	err := packet.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse RPTPING packet: %v", err)
	}

	if packet.RepeaterID != 312000 {
		t.Errorf("Expected repeater ID 312000, got %d", packet.RepeaterID)
	}
}

func TestRPTPINGPacket_Encode(t *testing.T) {
	packet := &RPTPINGPacket{
		RepeaterID: 312000,
	}

	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Failed to encode RPTPING packet: %v", err)
	}

	if len(data) != RPTPINGPacketSize {
		t.Errorf("Expected size %d, got %d", RPTPINGPacketSize, len(data))
	}

	if !bytes.Equal(data[0:7], []byte("RPTPING")) {
		t.Error("Invalid signature in encoded packet")
	}
}

// Test MSTPONG (Master pong response) packet
func TestMSTPONGPacket_Parse(t *testing.T) {
	data := make([]byte, MSTPONGPacketSize)
	copy(data[0:7], []byte("MSTPONG"))
	// Repeater ID
	data[7] = 0x00
	data[8] = 0x04
	data[9] = 0xC2
	data[10] = 0xC0

	packet := &MSTPONGPacket{}
	err := packet.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse MSTPONG packet: %v", err)
	}

	if packet.RepeaterID != 312000 {
		t.Errorf("Expected repeater ID 312000, got %d", packet.RepeaterID)
	}
}

func TestMSTPONGPacket_Encode(t *testing.T) {
	packet := &MSTPONGPacket{
		RepeaterID: 312000,
	}

	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Failed to encode MSTPONG packet: %v", err)
	}

	if len(data) != MSTPONGPacketSize {
		t.Errorf("Expected size %d, got %d", MSTPONGPacketSize, len(data))
	}

	if !bytes.Equal(data[0:7], []byte("MSTPONG")) {
		t.Error("Invalid signature in encoded packet")
	}
}

// Test MSTCL (Master close) packet
func TestMSTCLPacket_Parse(t *testing.T) {
	data := make([]byte, MSTCLPacketSize)
	copy(data[0:5], []byte("MSTCL"))
	// Repeater ID
	data[5] = 0x00
	data[6] = 0x04
	data[7] = 0xC2
	data[8] = 0xC0

	packet := &MSTCLPacket{}
	err := packet.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse MSTCL packet: %v", err)
	}

	if packet.RepeaterID != 312000 {
		t.Errorf("Expected repeater ID 312000, got %d", packet.RepeaterID)
	}
}

func TestMSTCLPacket_Encode(t *testing.T) {
	packet := &MSTCLPacket{
		RepeaterID: 312000,
	}

	data, err := packet.Encode()
	if err != nil {
		t.Fatalf("Failed to encode MSTCL packet: %v", err)
	}

	if len(data) != MSTCLPacketSize {
		t.Errorf("Expected size %d, got %d", MSTCLPacketSize, len(data))
	}

	if !bytes.Equal(data[0:5], []byte("MSTCL")) {
		t.Error("Invalid signature in encoded packet")
	}
}

// Test invalid packet sizes
func TestAuthPackets_InvalidSize(t *testing.T) {
	tests := []struct {
		name       string
		packetType string
		parse      func([]byte) error
	}{
		{"RPTL too small", "RPTL", func(d []byte) error { p := &RPTLPacket{}; return p.Parse(d) }},
		{"RPTK too small", "RPTK", func(d []byte) error { p := &RPTKPacket{}; return p.Parse(d) }},
		{"RPTC too small", "RPTC", func(d []byte) error { p := &RPTCPacket{}; return p.Parse(d) }},
		{"RPTACK too small", "RPTACK", func(d []byte) error { p := &RPTACKPacket{}; return p.Parse(d) }},
		{"RPTPING too small", "RPTPING", func(d []byte) error { p := &RPTPINGPacket{}; return p.Parse(d) }},
		{"MSTPONG too small", "MSTPONG", func(d []byte) error { p := &MSTPONGPacket{}; return p.Parse(d) }},
		{"MSTCL too small", "MSTCL", func(d []byte) error { p := &MSTCLPacket{}; return p.Parse(d) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte{0, 1, 2} // Too small
			err := tt.parse(data)
			if err == nil {
				t.Errorf("Expected error for %s with invalid size", tt.packetType)
			}
		})
	}
}
