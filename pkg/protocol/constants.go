package protocol

// Packet type identifiers (4-7 byte ASCII strings)
const (
	PacketTypeDMRD    = "DMRD"
	PacketTypeRPTL    = "RPTL"
	PacketTypeRPTK    = "RPTK"
	PacketTypeRPTC    = "RPTC"
	PacketTypeRPTO    = "RPTO"    // OPTIONS packet
	PacketTypeRPTCL   = "RPTCL"
	PacketTypeRPTACK  = "RPTACK"
	PacketTypeRPTPING = "RPTPING"
	PacketTypeMSTPONG = "MSTPONG"
	PacketTypeMSTNAK  = "MSTNAK"
	PacketTypeMSTCL   = "MSTCL"
)

// Packet size constants (in bytes)
const (
	DMRDPacketSize           = 53  // Standard HBP DMRD packet
	DMRDPacketSizeDroidStar  = 55  // DroidStar/client variant (adds 2 bytes BER+RSSI)
	DMRDOpenBridgePacketSize = 73  // DMRD + 20 byte HMAC-SHA1 signature
	RPTLPacketSize           = 8   // Login request (RPTL + 4 byte repeater ID)
	RPTKPacketSize           = 40  // Key exchange (RPTK + 4 byte repeater ID + 32 byte challenge)
	RPTCPacketSize           = 302 // Configuration packet
	RPTCLPacketSize          = 9   // Close from peer (RPTCL + 4 byte repeater ID)
	RPTACKPacketSize         = 10  // Acknowledgement (RPTACK + 4 byte repeater ID)
	RPTPINGPacketSize        = 11  // Ping from peer (RPTPING + 4 byte repeater ID)
	MSTPONGPacketSize        = 11  // Pong from master (MSTPONG + 4 byte repeater ID)
	MSTNAKPacketSize         = 10  // Negative acknowledgement (MSTNAK + 4 byte repeater ID)
	MSTCLPacketSize          = 9   // Close connection (MSTCL + 4 byte repeater ID)
)

// Slot byte (byte 15) bit masks - DMR slot information encoding
const (
	SlotTimeslotMask  = 0x80 // Bit 7: Timeslot (0=TS1, 1=TS2)
	SlotCallTypeMask  = 0x40 // Bit 6: Call type (0=group, 1=unit/private)
	SlotFrameTypeMask = 0x30 // Bits 4-5: Frame type
	SlotDataTypeMask  = 0x0F // Bits 0-3: Data type / Voice sequence
)

// Frame types (extracted from bits 4-5 of slot byte)
const (
	FrameTypeVoice           = 0x00 // Voice burst (A-F frames)
	FrameTypeVoiceHeader     = 0x01 // Voice call header
	FrameTypeVoiceTerminator = 0x02 // Voice call terminator
	FrameTypeDataSync        = 0x03 // Data synchronization
)

// DMRD packet field offsets
const (
	DMRDOffsetSignature = 0  // 4 bytes: "DMRD"
	DMRDOffsetSeq       = 4  // 1 byte: Sequence number
	DMRDOffsetSrcID     = 5  // 3 bytes: Source subscriber ID
	DMRDOffsetDstID     = 8  // 3 bytes: Destination ID (talkgroup or subscriber)
	DMRDOffsetRptID     = 11 // 4 bytes: Repeater/Peer ID
	DMRDOffsetSlot      = 15 // 1 byte: Slot/Call type bits
	DMRDOffsetStreamID  = 16 // 4 bytes: Stream ID
	DMRDOffsetPayload   = 20 // 33 bytes: Voice/Data payload
	DMRDOffsetHMAC      = 53 // 20 bytes: HMAC-SHA1 (OpenBridge only)
)

// Authentication sequence constants
const (
	SaltLength      = 4  // Salt length for challenge
	ChallengeLength = 32 // Challenge length for RPTK
)

// Timeslot values
const (
	Timeslot1 = 1
	Timeslot2 = 2
)

// Call type values
const (
	CallTypeGroup   = 0 // Group/talkgroup call
	CallTypePrivate = 1 // Unit-to-unit/private call
)
