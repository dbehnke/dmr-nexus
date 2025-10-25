package codec

import (
	"sync"
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

// Converter performs DMR <-> YSF codec conversion
type Converter struct {
	// DMR -> YSF conversion
	ysfBuffer ringBuffer
	ysfN      uint

	// YSF -> DMR conversion
	dmrBuffer ringBuffer
	dmrN      uint

	mu sync.Mutex
}

// NewConverter creates a new codec converter
func NewConverter() *Converter {
	return &Converter{
		ysfBuffer: newRingBuffer(BufferSize),
		dmrBuffer: newRingBuffer(BufferSize),
	}
}

// PutDMR adds a DMR voice frame for conversion to YSF
func (c *Converter) PutDMR(frame []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Extract AMBE data from DMR frame (33 bytes)
	// DMR frame contains AMBE voice data that needs to be converted to YSF format
	ambe := make([]byte, len(frame))
	copy(ambe, frame)

	// Add to buffer for YSF conversion
	c.ysfBuffer.addData(TagData, ambe)
}

// PutDMRHeader adds a DMR header for conversion to YSF
func (c *Converter) PutDMRHeader() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ysfBuffer.addData(TagHeader, nil)
	c.ysfN = 0
}

// PutDMREOT adds a DMR end-of-transmission for conversion to YSF
func (c *Converter) PutDMREOT() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ysfBuffer.addData(TagEOT, nil)
}

// GetYSF retrieves a YSF voice frame converted from DMR
// Returns the frame type (TagHeader, TagData, TagEOT) and the frame data
func (c *Converter) GetYSF(frame []byte) uint {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ysfBuffer.isEmpty() {
		return 0xFF // No data
	}

	tag, data := c.ysfBuffer.getData()

	if tag == TagHeader {
		c.ysfN = 0
		return TagHeader
	}

	if tag == TagEOT {
		return TagEOT
	}

	if tag == TagData {
		// Convert DMR AMBE to YSF AMBE format
		// Copy voice data (simplified - actual conversion may need reframing)
		if len(data) > 0 && len(frame) >= len(data) {
			copy(frame, data)
		}
		c.ysfN++
		return TagData
	}

	return 0xFF
}

// PutYSF adds a YSF voice frame for conversion to DMR
func (c *Converter) PutYSF(frame []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Extract AMBE data from YSF frame
	// YSF frame contains AMBE voice data that needs to be converted to DMR format
	ambe := make([]byte, len(frame))
	copy(ambe, frame)

	// Add to buffer for DMR conversion
	c.dmrBuffer.addData(TagData, ambe)
}

// PutYSFHeader adds a YSF header for conversion to DMR
func (c *Converter) PutYSFHeader() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.dmrBuffer.addData(TagHeader, nil)
	c.dmrN = 0
}

// PutYSFEOT adds a YSF end-of-transmission for conversion to DMR
func (c *Converter) PutYSFEOT() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.dmrBuffer.addData(TagEOT, nil)
}

// PutDummyYSF adds a dummy YSF frame for hangtime
func (c *Converter) PutDummyYSF() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add silence/dummy frame
	dummy := make([]byte, 120)
	c.dmrBuffer.addData(TagData, dummy)
}

// GetDMR retrieves a DMR voice frame converted from YSF
// Returns the frame type (TagHeader, TagData, TagEOT) and the frame data
func (c *Converter) GetDMR(frame []byte) uint {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.dmrBuffer.isEmpty() {
		return 0xFF // No data
	}

	tag, data := c.dmrBuffer.getData()

	if tag == TagHeader {
		c.dmrN = 0
		return TagHeader
	}

	if tag == TagEOT {
		return TagEOT
	}

	if tag == TagData {
		// Convert YSF AMBE to DMR AMBE format
		// Copy voice data (simplified - actual conversion may need reframing)
		if len(data) > 0 && len(frame) >= len(data) {
			copy(frame, data)
		}
		c.dmrN++
		return TagData
	}

	return 0xFF
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
