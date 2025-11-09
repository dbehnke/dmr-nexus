package codec

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/dbehnke/dmr-nexus/pkg/logger"
	"github.com/dbehnke/dmr-nexus/pkg/protocol"
	"github.com/dbehnke/dmr-nexus/pkg/ysf"
)

// Converter handles bidirectional audio conversion between DMR and YSF
// Based on ModeConv.cpp from MMDVM_CM

const (
	// Frame tags for state machine
	TagHeader = 0x00
	TagData   = 0x01
	TagEOT    = 0x02

	// DMR frame timing
	DMRFramePer = 55 // milliseconds between DMR frames

	// YSF frame timing
	YSFFramePer = 90 // milliseconds between YSF frames

	// Ring buffer size
	BufferSize = 1000
)

var (
	// DMR silence frame (9 bytes of silence data)
	DMRSilence = []byte{0xB9, 0xE8, 0x81, 0x52, 0x61, 0x73, 0x00, 0x2A, 0x6B}

	// YSF silence frame (13 bytes of silence data)
	// Keep default zeroed silence for now; DMR->YSF path will be refined later
	YSFSilence = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

// Converter performs DMR <-> YSF codec conversion
type Converter struct {
	// DMR -> YSF conversion
	ysfBuffer ringBuffer
	ysfN      uint

	// YSF -> DMR conversion
	dmrBuffer ringBuffer
	dmrN      uint

	// optional debug logger (short-lived sampling)
	dbgLog   *logger.Logger
	dbgCount int
	dbgMax   int

	// additional debug counters for mini and full DMR payloads
	dbgMiniCount int
	dbgMiniMax   int
	dbgDMRCount  int
	dbgDMRMax    int

	// Stream metadata for DMR sync/embedded signalling
	dmrTimeslot int
	dmrSrcID    uint32
	dmrDstID    uint32
	dmrFLCO     uint8 // FLCO value

	// Embedded LC encoder - created once per stream
	embeddedEncoder *protocol.EmbeddedLCEncoder

	// Feature toggles
	embeddedEnabled bool

	// one-time debug flags
	dbgEmbeddedLogged bool

	mu sync.Mutex
}

// NewConverter creates a new codec converter
func NewConverter() *Converter {
	c := &Converter{
		ysfBuffer:       newRingBuffer(BufferSize),
		dmrBuffer:       newRingBuffer(BufferSize),
		dbgMax:          10,
		dbgMiniMax:      10,
		dbgDMRMax:       12, // Increased to see 2 full superframes (6 frames each)
		embeddedEnabled: true,
	}

	// Optional env flag to disable embedded LC for troubleshooting
	if v := os.Getenv("YSF2DMR_DISABLE_EMBEDDED"); v == "1" || v == "true" || v == "TRUE" {
		c.embeddedEnabled = false
	}

	return c
}

// SetDebugLogger enables short-lived debug sampling logs for YSF->DMR conversion.
// It logs the first few AMBE parameter samples (datA/datB/datC) and intermediate
// values to help diagnose bit-mapping issues.
func (c *Converter) SetDebugLogger(log *logger.Logger) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if log != nil {
		c.dbgLog = log.WithComponent("codec")
		c.dbgCount = 0
	} else {
		c.dbgLog = nil
	}
}

// SetEmbeddedEnabled enables or disables insertion of embedded LC signalling
// into non-sync voice frames. Useful for A/B testing of audio garble issues.
func (c *Converter) SetEmbeddedEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.embeddedEnabled = enabled
}

// SetDMRStreamMetadata sets the stream metadata for DMR sync/embedded signalling
// and initializes the embedded LC encoder for the new stream
func (c *Converter) SetDMRStreamMetadata(timeslot int, srcID, dstID uint32, flco uint8) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dmrTimeslot = timeslot
	c.dmrSrcID = srcID
	c.dmrDstID = dstID
	c.dmrFLCO = flco

	// Create embedded LC encoder for this stream
	if c.embeddedEnabled {
		c.embeddedEncoder = protocol.NewEmbeddedLCEncoder(srcID, dstID, protocol.FLCO(flco))
	}
}

// PutDMR adds a DMR voice frame for conversion to YSF
func (c *Converter) PutDMR(frame []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Extract AMBE parameters from DMR frame (33 bytes contains 3 AMBE frames)
	a1, b1, c1, a2, b2, c2, a3, b3, c3 := extractAMBEFromDMR(frame)

	// Convert each AMBE frame to YSF format and add to buffer
	ysfFrame1 := ambeToYSF(a1, b1, c1)
	c.ysfBuffer.addData(TagData, ysfFrame1)

	ysfFrame2 := ambeToYSF(a2, b2, c2)
	c.ysfBuffer.addData(TagData, ysfFrame2)

	ysfFrame3 := ambeToYSF(a3, b3, c3)
	c.ysfBuffer.addData(TagData, ysfFrame3)
}

// PutDMRHeader adds a DMR header for conversion to YSF
func (c *Converter) PutDMRHeader() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ysfBuffer.addData(TagHeader, nil)
	c.ysfN = 0
}

// GetYSF retrieves a YSF voice frame converted from DMR
// Returns the frame type (TagHeader, TagData, TagEOT) and the frame data
// YSF frames contain 5 VCH sections, so we need to collect 5 AMBE frames from the buffer
func (c *Converter) GetYSF(frame []byte) uint {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ysfBuffer.isEmpty() {
		return 0xFF // No data
	}

	// We need 5 AMBE frames to create one YSF frame (5 VCH sections)
	var ysfVCH [5][]byte

	for i := 0; i < 5; i++ {
		if c.ysfBuffer.isEmpty() {
			return 0xFF // Not enough data
		}

		tag, data := c.ysfBuffer.getData()

		if tag == TagHeader {
			if i == 0 {
				c.ysfN = 0
				return TagHeader
			}
			return 0xFF // Unexpected header mid-frame
		}

		if tag == TagEOT {
			if i == 0 {
				return TagEOT
			}
			return 0xFF // Unexpected EOT mid-frame
		}

		if tag == TagData {
			// Data should be a 13-byte YSF VCH frame
			if len(data) >= 13 {
				ysfVCH[i] = data
			} else {
				return 0xFF // Invalid data
			}
		} else {
			return 0xFF // Invalid tag
		}
	}

	// Now pack all 5 VCH sections contiguously (legacy behavior)
	// Each VCH is 13 bytes = 104 bits; minimal buffer 65 bytes
	if len(frame) < 65 {
		return 0xFF
	}

	// Clear the frame first
	for i := range frame[:65] {
		frame[i] = 0
	}

	// Copy each VCH section contiguously into the frame
	for i := 0; i < 5; i++ {
		copy(frame[i*13:(i+1)*13], ysfVCH[i])
	}

	c.ysfN++
	return TagData
}

// PutYSF adds a YSF voice frame for conversion to DMR
// YSF frame contains 5 VCH sections, each containing AMBE data
func (c *Converter) PutYSF(frame []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// YSF frame contains 5 VCH (Voice Channel) sections
	// Each VCH section is 13 bytes (104 bits) of AMBE data
	// We need to extract AMBE parameters from each VCH section
	// and convert them to DMR format

	// Expect full YSF payload (120 bytes). We operate after SYNC(5) + FICH(6)
	if len(frame) < ysf.YSFHeaderLength { // 120 bytes payload
		return
	}

	// The input frame here is the full 120-byte YSF payload, which includes:
	// [0:5]   SYNC (5 bytes)
	// [5:11]  FICH (6 bytes)
	// [>=11]  Voice/Data area
	payload := frame

	// Process each of the 5 VCH sections in the YSF frame.
	// According to MMDVM_CM ModeConv:
	// - Skip SYNC (5 bytes) and FICH (6 bytes)
	// - First VCH starts at an additional 40 bits (5 bytes) after that
	// Therefore, from the start of the 120-byte payload, the first VCH starts at:
	//   (5 + 6) bytes + 40 bits = 11 bytes + 5 bytes = 16 bytes => 128 bits
	// Subsequent VCHs are 144 bits (18 bytes) apart.
	baseBits := uint(ysf.YSFSyncLength+ysf.YSFFICHLength) * 8
	for i := uint(0); i < 5; i++ {
		offset := baseBits + 40 + i*144 // bit offset within payload

		// Extract AMBE parameters from YSF VCH format (12-bit parameter values)
		datA, datB, datC := ysfToAMBE(payload, offset)

		// YSF stores lossy 12-bit parameter values - we need to re-encode with Golay
		// This won't perfectly match the original DMR->YSF->DMR round trip,
		// but it's the best we can do with YSF's lossy format

		// Encode dat_a with Golay(24,12) to get 24-bit value
		encA := ysf.Encode24128(datA)

		// Encode dat_b with Golay(23,12) - returns 23-bit value
		encB := ysf.Encode23127(datB)

		// Apply PRNG to B for DMR storage
		// PRNG table has 24-bit values, encB is 23 bits
		// This matches MMDVM_CM YSF2DMR/ModeConv.cpp lines 664-665
		pRNG := prngTable[datA]
		bOut := encB ^ pRNG

		// dat_c remains as-is (25 bits)
		ambeC := datC

		// Short-lived debug sampling
		if c.dbgLog != nil && c.dbgCount < c.dbgMax {
			c.dbgLog.Debug("YSF->DMR AMBE sample",
				logger.Int("vch_index", int(i)),
				logger.Uint32("datA_12", datA),
				logger.String("encA_24", fmtHex24(encA)),
				logger.Uint32("datB_12", datB),
				logger.String("encB_23", fmtHex23(encB)),
				logger.String("prng_23", fmtHex23(pRNG)),
				logger.String("bOut_23", fmtHex23(bOut)),
				logger.Uint32("datC_25", datC),
			)
			c.dbgCount++
		}

		ambeA := encA
		ambeB := bOut

		// Create a mini DMR frame with just this one AMBE frame
		// DMR format uses 9 bytes per AMBE frame
		dmrFrame := make([]byte, 9)
		insertAMBEToDMR(dmrFrame, ambeA, ambeB, ambeC)

		// Debug: log the 9-byte mini AMBE frame
		if c.dbgLog != nil && c.dbgMiniCount < c.dbgMiniMax {
			c.dbgLog.Debug("AMBE mini-frame",
				logger.Int("vch_index", int(i)),
				logger.String("bytes", fmtHexBytes(dmrFrame, 9)),
			)
			c.dbgMiniCount++
		}

		// Add to buffer
		c.dmrBuffer.addData(TagData, dmrFrame)
	}
}

// Helpers to format fixed-width hex for debug
func fmtHex24(v uint32) string { // 24-bit value
	return fmt.Sprintf("%06X", v&0xFFFFFF)
}
func fmtHex23(v uint32) string { // 23-bit value
	return fmt.Sprintf("%06X", v&0x7FFFFF)
}

// fmtHexBytes formats a byte slice into a space-separated hex string, up to max bytes
func fmtHexBytes(b []byte, max int) string {
	if max <= 0 || max > len(b) {
		max = len(b)
	}
	var sb strings.Builder
	for i := 0; i < max; i++ {
		if i > 0 {
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "%02X", b[i])
	}
	return sb.String()
}

// PutYSFHeader adds a YSF header for conversion to DMR
func (c *Converter) PutYSFHeader() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.dmrBuffer.addData(TagHeader, nil)
	c.dmrN = 0
}

// PutYSFEOT adds a YSF end-of-transmission for conversion to DMR
// This adds filler silence frames to align to 5-frame boundaries (YSF transmits in groups of 5 VCH)
func (c *Converter) PutYSFEOT() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// YSF transmits voice in groups of 5 VCH sections per frame
	// Add filler frames to complete the group
	fill := 5 - (c.dmrN % 5)
	if fill < 5 {
		for i := uint(0); i < fill; i++ {
			// Add DMR silence frame (9 bytes)
			c.dmrBuffer.addData(TagData, DMRSilence)
			c.dmrN++
		}
	}

	c.dmrBuffer.addData(TagEOT, nil)
}

// PutDMREOT adds a DMR end-of-transmission for conversion to YSF
// This adds filler silence frames to align to 5-frame boundaries
func (c *Converter) PutDMREOT() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add filler frames to complete a 5-frame group
	fill := 5 - (c.ysfN % 5)
	if fill < 5 {
		for i := uint(0); i < fill; i++ {
			// Add YSF silence frame (13 bytes)
			c.ysfBuffer.addData(TagData, YSFSilence)
			c.ysfN++
		}
	}

	c.ysfBuffer.addData(TagEOT, nil)
}

// PutDummyYSF adds dummy YSF silence frames for hangtime
// Each YSF frame contains 5 VCH sections, so we add 5 DMR silence frames
func (c *Converter) PutDummyYSF() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add 5 DMR silence frames (one YSF frame worth)
	for i := 0; i < 5; i++ {
		c.dmrBuffer.addData(TagData, DMRSilence)
		c.dmrN++
	}
}

// PutDummyDMR adds dummy DMR silence frames for hangtime
func (c *Converter) PutDummyDMR() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add one YSF silence frame
	c.ysfBuffer.addData(TagData, YSFSilence)
	c.ysfN++
}

// GetDMR retrieves a DMR voice frame converted from YSF
// Returns the frame type (TagHeader, TagData, TagEOT) and the frame data
// DMR frames contain 3 AMBE frames, so we need to collect 3 frames from the buffer
func (c *Converter) GetDMR(frame []byte) uint {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.dmrBuffer.isEmpty() {
		return 0xFF // No data
	}

	// We need 3 AMBE frames to create one DMR frame
	var ambeFrames [3]struct {
		a, b, c uint32
	}

	for i := 0; i < 3; i++ {
		if c.dmrBuffer.isEmpty() {
			return 0xFF // Not enough data
		}

		tag, data := c.dmrBuffer.getData()

		if tag == TagHeader {
			if i == 0 {
				c.dmrN = 0
				return TagHeader
			}
			return 0xFF // Unexpected header mid-frame
		}

		if tag == TagEOT {
			if i == 0 {
				return TagEOT
			}
			return 0xFF // Unexpected EOT mid-frame
		}

		if tag == TagData && len(data) >= 9 {
			// Extract AMBE parameters from the 9-byte mini DMR frame
			ambeFrames[i].a, ambeFrames[i].b, ambeFrames[i].c = extractSingleAMBEFromDMR(data)

			// Debug: verify extraction
			if c.dbgLog != nil && c.dbgDMRCount == 0 && i == 0 {
				c.dbgLog.Debug("Extracted from mini-frame",
					logger.String("bytes", fmtHexBytes(data, 9)),
					logger.String("a", fmtHex24(ambeFrames[i].a)),
					logger.String("b", fmtHex23(ambeFrames[i].b)),
					logger.Uint32("c", ambeFrames[i].c),
				)
			}
		} else {
			return 0xFF // Invalid data
		}
	}

	// Now pack all 3 AMBE frames into a single 33-byte DMR frame
	// The 3 AMBE frames are interleaved in the DMR frame
	if len(frame) < 33 {
		return 0xFF
	}

	// Clear the frame first
	for i := range frame[:33] {
		frame[i] = 0
	}

	// Interleave all 3 AMBE frames into the DMR frame
	// Frame 1 at base positions, Frame 2 at +72 bits, Frame 3 at +192 bits
	insertInterleavedAMBEToDMR(frame, ambeFrames[0].a, ambeFrames[0].b, ambeFrames[0].c,
		ambeFrames[1].a, ambeFrames[1].b, ambeFrames[1].c,
		ambeFrames[2].a, ambeFrames[2].b, ambeFrames[2].c)

	// Debug: log which mini-frames we're using
	if c.dbgLog != nil && c.dbgDMRCount < c.dbgDMRMax {
		c.dbgLog.Debug("Interleaving 3 AMBE frames",
			logger.String("frame0", fmtHexBytes([]byte{
				byte(ambeFrames[0].a >> 16), byte(ambeFrames[0].a >> 8), byte(ambeFrames[0].a),
			}, 3)),
			logger.String("frame1", fmtHexBytes([]byte{
				byte(ambeFrames[1].a >> 16), byte(ambeFrames[1].a >> 8), byte(ambeFrames[1].a),
			}, 3)),
			logger.String("frame2", fmtHexBytes([]byte{
				byte(ambeFrames[2].a >> 16), byte(ambeFrames[2].a >> 8), byte(ambeFrames[2].a),
			}, 3)),
		)
	}

	// Insert DMR voice sync or embedded signalling
	// Voice sync pattern is inserted every 6th frame (when dmrN % 6 == 0)
	// Other frames get embedded LC signalling
	// Ensure reserved sync/embedded region (bytes 13-19) starts clean with no AMBE bits
	for i := 13; i <= 19; i++ {
		frame[i] = 0x00
	}
	voiceSeq := int(c.dmrN % 6)
	if c.dbgLog != nil && !c.dbgEmbeddedLogged {
		c.dbgLog.Debug("Embedded LC setting",
			logger.Bool("enabled", c.embeddedEnabled))
		c.dbgEmbeddedLogged = true
	}
	if voiceSeq == 0 {
		// Insert voice sync pattern for this timeslot
		protocol.InsertVoiceSync(frame, c.dmrTimeslot)
	} else if c.embeddedEnabled && c.embeddedEncoder != nil {
		// Insert embedded LC fragment for frames B-F (voiceSeq 1-5)
		// The encoder expects fragment indices 0-4 for frames B-F
		fragment, lcss := c.embeddedEncoder.GetFragment(voiceSeq - 1)
		protocol.InsertEmbeddedFragment(frame, fragment, lcss)
	}

	// Debug: log first few full 33-byte DMR payloads
	if c.dbgLog != nil && c.dbgDMRCount < c.dbgDMRMax {
		c.dbgLog.Debug("DMR 33-byte payload",
			logger.Uint("seq", c.dmrN),
			logger.String("bytes", fmtHexBytes(frame[:33], 33)),
		)
		c.dbgDMRCount++
	}

	c.dmrN++
	return TagData
}

// ringBuffer is a simple ring buffer for frame data
type ringBuffer struct {
	data []bufferEntry
	head int
	tail int
	size int
	mu   sync.Mutex
}

type bufferEntry struct {
	tag  uint
	data []byte
}

func newRingBuffer(size int) ringBuffer {
	return ringBuffer{
		data: make([]bufferEntry, size),
		size: size,
	}
}

func (rb *ringBuffer) addData(tag uint, data []byte) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	// Copy data to avoid mutations
	var dataCopy []byte
	if data != nil {
		dataCopy = make([]byte, len(data))
		copy(dataCopy, data)
	}

	rb.data[rb.tail] = bufferEntry{
		tag:  tag,
		data: dataCopy,
	}

	rb.tail = (rb.tail + 1) % rb.size

	// Handle overflow
	if rb.tail == rb.head {
		rb.head = (rb.head + 1) % rb.size
	}
}

func (rb *ringBuffer) getData() (uint, []byte) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.isEmpty() {
		return 0xFF, nil
	}

	entry := rb.data[rb.head]
	rb.head = (rb.head + 1) % rb.size

	return entry.tag, entry.data
}

func (rb *ringBuffer) isEmpty() bool {
	return rb.head == rb.tail
}
