package ysf

// YSF Viterbi Convolutional Decoder
// Based on YSFConvolution.cpp from MMDVM_CM by Jonathan Naylor G4KLX
// Implements rate 1/2, constraint length K=5 convolutional code

const (
	numStatesD2 = 8
	numStates   = 16
	metricM     = 2
	constraintK = 5
	maxDecisions = 180
)

var (
	branchTable1 = []uint8{0, 0, 0, 0, 1, 1, 1, 1}
	branchTable2 = []uint8{0, 1, 1, 0, 0, 1, 1, 0}
)

// YSFConvolution implements Viterbi decoding for YSF
type YSFConvolution struct {
	metrics1   []uint16
	metrics2   []uint16
	oldMetrics []uint16
	newMetrics []uint16
	decisions  []uint64
	dp         int // decision pointer
}

// NewYSFConvolution creates a new Viterbi decoder
func NewYSFConvolution() *YSFConvolution {
	return &YSFConvolution{
		metrics1:  make([]uint16, numStates),
		metrics2:  make([]uint16, numStates),
		decisions: make([]uint64, maxDecisions),
	}
}

// Start initializes the decoder state
func (c *YSFConvolution) Start() {
	// Clear metrics
	for i := range c.metrics1 {
		c.metrics1[i] = 0
		c.metrics2[i] = 0
	}

	c.oldMetrics = c.metrics1
	c.newMetrics = c.metrics2
	c.dp = 0
}

// Decode processes two soft-decision bits through the Viterbi decoder
func (c *YSFConvolution) Decode(s0, s1 uint8) {
	if c.dp >= maxDecisions {
		return
	}

	c.decisions[c.dp] = 0

	for i := uint8(0); i < numStatesD2; i++ {
		j := i * 2

		// Calculate branch metric
		metric := uint16((branchTable1[i] ^ s0) + (branchTable2[i] ^ s1))

		// Path 0
		m0 := c.oldMetrics[i] + metric
		m1 := c.oldMetrics[i+numStatesD2] + (metricM - metric)
		var decision0 uint8
		if m0 >= m1 {
			decision0 = 1
			c.newMetrics[j+0] = m1
		} else {
			decision0 = 0
			c.newMetrics[j+0] = m0
		}

		// Path 1
		m0 = c.oldMetrics[i] + (metricM - metric)
		m1 = c.oldMetrics[i+numStatesD2] + metric
		var decision1 uint8
		if m0 >= m1 {
			decision1 = 1
			c.newMetrics[j+1] = m1
		} else {
			decision1 = 0
			c.newMetrics[j+1] = m0
		}

		// Store decisions
		c.decisions[c.dp] |= (uint64(decision1) << (j + 1)) | (uint64(decision0) << j)
	}

	c.dp++

	// Swap metrics
	c.oldMetrics, c.newMetrics = c.newMetrics, c.oldMetrics
}

// Chainback traces back through the trellis to recover decoded bits
func (c *YSFConvolution) Chainback(out []byte, nBits uint) {
	state := uint32(0)

	for nBits > 0 {
		nBits--
		c.dp--

		if c.dp < 0 {
			break
		}

		i := state >> (9 - constraintK)
		bit := uint8(c.decisions[c.dp]>>i) & 1
		state = (uint32(bit) << 7) | (state >> 1)

		writeBit(out, nBits, bit != 0)
	}
}

// Encode performs convolutional encoding
func (c *YSFConvolution) Encode(in []byte, out []byte, nBits uint) {
	var d1, d2, d3, d4 uint8
	k := uint(0)

	for i := uint(0); i < nBits; i++ {
		var d uint8
		if readBit(in, i) {
			d = 1
		}

		g1 := (d + d3 + d4) & 1
		g2 := (d + d1 + d2 + d4) & 1

		d4 = d3
		d3 = d2
		d2 = d1
		d1 = d

		writeBit(out, k, g1 != 0)
		k++
		writeBit(out, k, g2 != 0)
		k++
	}
}

// Bit manipulation helpers
var bitMaskTable = []byte{0x80, 0x40, 0x20, 0x10, 0x08, 0x04, 0x02, 0x01}

func writeBit(p []byte, i uint, b bool) {
	if b {
		p[i>>3] |= bitMaskTable[i&7]
	} else {
		p[i>>3] &= ^bitMaskTable[i&7]
	}
}

func readBit(p []byte, i uint) bool {
	return (p[i>>3] & bitMaskTable[i&7]) != 0
}
