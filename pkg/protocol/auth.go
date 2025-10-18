package protocol

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// RPTLPacket represents a login request from a peer
type RPTLPacket struct {
	RepeaterID uint32
}

// Parse parses an RPTL packet from raw bytes
func (p *RPTLPacket) Parse(data []byte) error {
	if len(data) != RPTLPacketSize {
		return fmt.Errorf("invalid RPTL packet size: %d (expected %d)", len(data), RPTLPacketSize)
	}

	if string(data[0:4]) != PacketTypeRPTL {
		return fmt.Errorf("invalid RPTL signature: %s", string(data[0:4]))
	}

	p.RepeaterID = binary.BigEndian.Uint32(data[4:8])
	return nil
}

// Encode encodes the RPTL packet to raw bytes
func (p *RPTLPacket) Encode() ([]byte, error) {
	data := make([]byte, RPTLPacketSize)
	copy(data[0:4], []byte(PacketTypeRPTL))
	binary.BigEndian.PutUint32(data[4:8], p.RepeaterID)
	return data, nil
}

// RPTKPacket represents a key exchange packet
type RPTKPacket struct {
	RepeaterID uint32
	Challenge  []byte // 32 bytes
}

// Parse parses an RPTK packet from raw bytes
func (p *RPTKPacket) Parse(data []byte) error {
	if len(data) != RPTKPacketSize {
		return fmt.Errorf("invalid RPTK packet size: %d (expected %d)", len(data), RPTKPacketSize)
	}

	if string(data[0:4]) != PacketTypeRPTK {
		return fmt.Errorf("invalid RPTK signature: %s", string(data[0:4]))
	}

	p.RepeaterID = binary.BigEndian.Uint32(data[4:8])
	p.Challenge = make([]byte, ChallengeLength)
	copy(p.Challenge, data[8:8+ChallengeLength])
	return nil
}

// Encode encodes the RPTK packet to raw bytes
func (p *RPTKPacket) Encode() ([]byte, error) {
	data := make([]byte, RPTKPacketSize)
	copy(data[0:4], []byte(PacketTypeRPTK))
	binary.BigEndian.PutUint32(data[4:8], p.RepeaterID)
	
	if len(p.Challenge) >= ChallengeLength {
		copy(data[8:8+ChallengeLength], p.Challenge[:ChallengeLength])
	} else {
		copy(data[8:], p.Challenge)
	}
	
	return data, nil
}

// RPTCPacket represents a configuration packet from a peer
type RPTCPacket struct {
	RepeaterID  uint32
	Callsign    string
	RXFreq      string
	TXFreq      string
	TXPower     string
	ColorCode   string
	Latitude    string
	Longitude   string
	Height      string
	Location    string
	Description string
	Slots       string
	URL         string
	SoftwareID  string
	PackageID   string
}

// Parse parses an RPTC packet from raw bytes
func (p *RPTCPacket) Parse(data []byte) error {
	if len(data) != RPTCPacketSize {
		return fmt.Errorf("invalid RPTC packet size: %d (expected %d)", len(data), RPTCPacketSize)
	}

	if string(data[0:4]) != PacketTypeRPTC {
		return fmt.Errorf("invalid RPTC signature: %s", string(data[0:4]))
	}

	p.RepeaterID = binary.BigEndian.Uint32(data[4:8])

	// Parse fixed-length string fields (trim spaces and nulls)
	p.Callsign = strings.TrimSpace(string(data[8:16]))
	p.RXFreq = strings.TrimSpace(string(data[16:25]))
	p.TXFreq = strings.TrimSpace(string(data[25:34]))
	p.TXPower = strings.TrimSpace(string(data[34:36]))
	p.ColorCode = strings.TrimSpace(string(data[36:38]))
	p.Latitude = strings.TrimSpace(string(data[38:46]))
	p.Longitude = strings.TrimSpace(string(data[46:55]))
	p.Height = strings.TrimSpace(string(data[55:58]))
	p.Location = strings.TrimSpace(string(data[58:78]))
	p.Description = strings.TrimSpace(string(data[78:97]))
	p.Slots = strings.TrimSpace(string(data[97:98]))
	p.URL = strings.TrimSpace(string(data[98:222]))
	p.SoftwareID = strings.TrimSpace(string(data[222:262]))
	p.PackageID = strings.TrimSpace(string(data[262:302]))

	return nil
}

// Encode encodes the RPTC packet to raw bytes
func (p *RPTCPacket) Encode() ([]byte, error) {
	data := make([]byte, RPTCPacketSize)
	copy(data[0:4], []byte(PacketTypeRPTC))
	binary.BigEndian.PutUint32(data[4:8], p.RepeaterID)

	// Helper to copy string to fixed-width field
	copyField := func(dst []byte, src string) {
		for i := range dst {
			if i < len(src) {
				dst[i] = src[i]
			} else {
				dst[i] = ' '
			}
		}
	}

	copyField(data[8:16], p.Callsign)
	copyField(data[16:25], p.RXFreq)
	copyField(data[25:34], p.TXFreq)
	copyField(data[34:36], p.TXPower)
	copyField(data[36:38], p.ColorCode)
	copyField(data[38:46], p.Latitude)
	copyField(data[46:55], p.Longitude)
	copyField(data[55:58], p.Height)
	copyField(data[58:78], p.Location)
	copyField(data[78:97], p.Description)
	copyField(data[97:98], p.Slots)
	copyField(data[98:222], p.URL)
	copyField(data[222:262], p.SoftwareID)
	copyField(data[262:302], p.PackageID)

	return data, nil
}

// RPTACKPacket represents an acknowledgement from master
type RPTACKPacket struct {
	RepeaterID uint32
}

// Parse parses an RPTACK packet from raw bytes
func (p *RPTACKPacket) Parse(data []byte) error {
	if len(data) != RPTACKPacketSize {
		return fmt.Errorf("invalid RPTACK packet size: %d (expected %d)", len(data), RPTACKPacketSize)
	}

	if string(data[0:6]) != PacketTypeRPTACK {
		return fmt.Errorf("invalid RPTACK signature: %s", string(data[0:6]))
	}

	p.RepeaterID = binary.BigEndian.Uint32(data[6:10])
	return nil
}

// Encode encodes the RPTACK packet to raw bytes
func (p *RPTACKPacket) Encode() ([]byte, error) {
	data := make([]byte, RPTACKPacketSize)
	copy(data[0:6], []byte(PacketTypeRPTACK))
	binary.BigEndian.PutUint32(data[6:10], p.RepeaterID)
	return data, nil
}

// RPTPINGPacket represents a keepalive ping from peer
type RPTPINGPacket struct {
	RepeaterID uint32
}

// Parse parses an RPTPING packet from raw bytes
func (p *RPTPINGPacket) Parse(data []byte) error {
	if len(data) != RPTPINGPacketSize {
		return fmt.Errorf("invalid RPTPING packet size: %d (expected %d)", len(data), RPTPINGPacketSize)
	}

	if string(data[0:7]) != PacketTypeRPTPING {
		return fmt.Errorf("invalid RPTPING signature: %s", string(data[0:7]))
	}

	p.RepeaterID = binary.BigEndian.Uint32(data[7:11])
	return nil
}

// Encode encodes the RPTPING packet to raw bytes
func (p *RPTPINGPacket) Encode() ([]byte, error) {
	data := make([]byte, RPTPINGPacketSize)
	copy(data[0:7], []byte(PacketTypeRPTPING))
	binary.BigEndian.PutUint32(data[7:11], p.RepeaterID)
	return data, nil
}

// MSTPONGPacket represents a keepalive pong from master
type MSTPONGPacket struct {
	RepeaterID uint32
}

// Parse parses an MSTPONG packet from raw bytes
func (p *MSTPONGPacket) Parse(data []byte) error {
	if len(data) != MSTPONGPacketSize {
		return fmt.Errorf("invalid MSTPONG packet size: %d (expected %d)", len(data), MSTPONGPacketSize)
	}

	if string(data[0:7]) != PacketTypeMSTPONG {
		return fmt.Errorf("invalid MSTPONG signature: %s", string(data[0:7]))
	}

	p.RepeaterID = binary.BigEndian.Uint32(data[7:11])
	return nil
}

// Encode encodes the MSTPONG packet to raw bytes
func (p *MSTPONGPacket) Encode() ([]byte, error) {
	data := make([]byte, MSTPONGPacketSize)
	copy(data[0:7], []byte(PacketTypeMSTPONG))
	binary.BigEndian.PutUint32(data[7:11], p.RepeaterID)
	return data, nil
}

// MSTCLPacket represents a close/disconnect packet from master
type MSTCLPacket struct {
	RepeaterID uint32
}

// Parse parses an MSTCL packet from raw bytes
func (p *MSTCLPacket) Parse(data []byte) error {
	if len(data) != MSTCLPacketSize {
		return fmt.Errorf("invalid MSTCL packet size: %d (expected %d)", len(data), MSTCLPacketSize)
	}

	if string(data[0:5]) != PacketTypeMSTCL {
		return fmt.Errorf("invalid MSTCL signature: %s", string(data[0:5]))
	}

	p.RepeaterID = binary.BigEndian.Uint32(data[5:9])
	return nil
}

// Encode encodes the MSTCL packet to raw bytes
func (p *MSTCLPacket) Encode() ([]byte, error) {
	data := make([]byte, MSTCLPacketSize)
	copy(data[0:5], []byte(PacketTypeMSTCL))
	binary.BigEndian.PutUint32(data[5:9], p.RepeaterID)
	return data, nil
}

// Helper functions for parsing packets

// ParseRPTL parses an RPTL packet from raw bytes
func ParseRPTL(data []byte) (*RPTLPacket, error) {
	p := &RPTLPacket{}
	err := p.Parse(data)
	return p, err
}

// ParseRPTK parses an RPTK packet from raw bytes
func ParseRPTK(data []byte) (*RPTKPacket, error) {
	p := &RPTKPacket{}
	err := p.Parse(data)
	return p, err
}

// ParseRPTC parses an RPTC packet from raw bytes
func ParseRPTC(data []byte) (*RPTCPacket, error) {
	p := &RPTCPacket{}
	err := p.Parse(data)
	return p, err
}

// ParseRPTACK parses an RPTACK packet from raw bytes
func ParseRPTACK(data []byte) (*RPTACKPacket, error) {
	p := &RPTACKPacket{}
	err := p.Parse(data)
	return p, err
}

// ParseRPTPING parses an RPTPING packet from raw bytes
func ParseRPTPING(data []byte) (*RPTPINGPacket, error) {
	p := &RPTPINGPacket{}
	err := p.Parse(data)
	return p, err
}

// ParseMSTPONG parses an MSTPONG packet from raw bytes
func ParseMSTPONG(data []byte) (*MSTPONGPacket, error) {
	p := &MSTPONGPacket{}
	err := p.Parse(data)
	return p, err
}

// ParseMSTCL parses an MSTCL packet from raw bytes
func ParseMSTCL(data []byte) (*MSTCLPacket, error) {
	p := &MSTCLPacket{}
	err := p.Parse(data)
	return p, err
}
