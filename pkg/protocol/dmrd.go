package protocol

import (
	"encoding/binary"
	"fmt"
)

// DMRDPacket represents a DMR data packet
type DMRDPacket struct {
	Sequence      byte   // Sequence number
	SourceID      uint32 // Source subscriber ID (24-bit)
	DestinationID uint32 // Destination ID - talkgroup or subscriber (24-bit)
	RepeaterID    uint32 // Repeater/Peer ID
	Timeslot      int    // 1 or 2
	CallType      int    // 0=group, 1=private
	FrameType     byte   // Frame type (voice, header, terminator, data)
	DataType      byte   // Data type / voice sequence (lower 4 bits)
	StreamID      uint32 // Stream identifier
	Payload       []byte // 33 bytes of voice/data payload
	HMAC          []byte // 20 bytes HMAC-SHA1 (OpenBridge only)
}

// Parse parses a DMRD packet from raw bytes
func (p *DMRDPacket) Parse(data []byte) error {
	// Validate packet size
	if len(data) != DMRDPacketSize && len(data) != DMRDOpenBridgePacketSize {
		return fmt.Errorf("invalid DMRD packet size: %d (expected %d or %d)",
			len(data), DMRDPacketSize, DMRDOpenBridgePacketSize)
	}

	// Validate signature
	if string(data[0:4]) != PacketTypeDMRD {
		return fmt.Errorf("invalid DMRD signature: %s", string(data[0:4]))
	}

	// Parse fields
	p.Sequence = data[DMRDOffsetSeq]

	// Parse 24-bit IDs (big-endian, stored in upper 24 bits of uint32)
	p.SourceID = uint32(data[DMRDOffsetSrcID])<<16 |
		uint32(data[DMRDOffsetSrcID+1])<<8 |
		uint32(data[DMRDOffsetSrcID+2])

	p.DestinationID = uint32(data[DMRDOffsetDstID])<<16 |
		uint32(data[DMRDOffsetDstID+1])<<8 |
		uint32(data[DMRDOffsetDstID+2])

	// Parse 32-bit repeater ID (big-endian)
	p.RepeaterID = binary.BigEndian.Uint32(data[DMRDOffsetRptID : DMRDOffsetRptID+4])

	// Parse slot byte
	slotByte := data[DMRDOffsetSlot]

	// Extract timeslot (bit 7)
	if slotByte&SlotTimeslotMask != 0 {
		p.Timeslot = Timeslot2
	} else {
		p.Timeslot = Timeslot1
	}

	// Extract call type (bit 6)
	if slotByte&SlotCallTypeMask != 0 {
		p.CallType = CallTypePrivate
	} else {
		p.CallType = CallTypeGroup
	}

	// Extract frame type (bits 4-5)
	p.FrameType = (slotByte & SlotFrameTypeMask) >> 4

	// Extract data type (bits 0-3)
	p.DataType = slotByte & SlotDataTypeMask

	// Parse stream ID (big-endian)
	p.StreamID = binary.BigEndian.Uint32(data[DMRDOffsetStreamID : DMRDOffsetStreamID+4])

	// Copy payload (33 bytes)
	p.Payload = make([]byte, 33)
	copy(p.Payload, data[DMRDOffsetPayload:DMRDOffsetPayload+33])

	// Copy HMAC if present (OpenBridge)
	if len(data) == DMRDOpenBridgePacketSize {
		p.HMAC = make([]byte, 20)
		copy(p.HMAC, data[DMRDOffsetHMAC:DMRDOffsetHMAC+20])
	}

	return nil
}

// Encode encodes the DMRD packet to raw bytes
func (p *DMRDPacket) Encode() ([]byte, error) {
	// Determine packet size
	size := DMRDPacketSize
	if len(p.HMAC) > 0 {
		size = DMRDOpenBridgePacketSize
	}

	data := make([]byte, size)

	// Write signature
	copy(data[0:4], []byte(PacketTypeDMRD))

	// Write sequence
	data[DMRDOffsetSeq] = p.Sequence

	// Write 24-bit source ID (big-endian)
	data[DMRDOffsetSrcID] = byte(p.SourceID >> 16)
	data[DMRDOffsetSrcID+1] = byte(p.SourceID >> 8)
	data[DMRDOffsetSrcID+2] = byte(p.SourceID)

	// Write 24-bit destination ID (big-endian)
	data[DMRDOffsetDstID] = byte(p.DestinationID >> 16)
	data[DMRDOffsetDstID+1] = byte(p.DestinationID >> 8)
	data[DMRDOffsetDstID+2] = byte(p.DestinationID)

	// Write 32-bit repeater ID (big-endian)
	binary.BigEndian.PutUint32(data[DMRDOffsetRptID:DMRDOffsetRptID+4], p.RepeaterID)

	// Build slot byte
	var slotByte byte = 0

	// Set timeslot bit
	if p.Timeslot == Timeslot2 {
		slotByte |= SlotTimeslotMask
	}

	// Set call type bit
	if p.CallType == CallTypePrivate {
		slotByte |= SlotCallTypeMask
	}

	// Set frame type bits
	slotByte |= (p.FrameType << 4) & SlotFrameTypeMask

	// Set data type bits
	slotByte |= p.DataType & SlotDataTypeMask

	data[DMRDOffsetSlot] = slotByte

	// Write stream ID (big-endian)
	binary.BigEndian.PutUint32(data[DMRDOffsetStreamID:DMRDOffsetStreamID+4], p.StreamID)

	// Copy payload
	if len(p.Payload) >= 33 {
		copy(data[DMRDOffsetPayload:DMRDOffsetPayload+33], p.Payload[:33])
	} else {
		copy(data[DMRDOffsetPayload:], p.Payload)
	}

	// Copy HMAC if present
	if len(p.HMAC) > 0 && size == DMRDOpenBridgePacketSize {
		copy(data[DMRDOffsetHMAC:DMRDOffsetHMAC+20], p.HMAC[:20])
	}

	return data, nil
}

// ParseDMRD parses a DMRD packet from raw bytes
func ParseDMRD(data []byte) (*DMRDPacket, error) {
	p := &DMRDPacket{}
	err := p.Parse(data)
	return p, err
}
