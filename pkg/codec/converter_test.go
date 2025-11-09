package codec

import (
	"testing"

	"github.com/dbehnke/dmr-nexus/pkg/ysf"
)

// Test that PutYSF extracts VCHs from bit offsets 40 + 144*i
// Uses a round-trip through ambeToYSF/ysfToAMBE expectations
func TestPutYSF_ExtractionOffsets(t *testing.T) {
	c := NewConverter()

	// Build payload with 5 VCH generated from known AMBE parameters
	payload := make([]byte, ysf.YSFHeaderLength)
	// VCH placement within the 120-byte payload (includes SYNC+FICH):
	// Start at 16 bytes (after 5 bytes SYNC + 6 bytes FICH + 40 bits), then +18 bytes each
	positions := []int{16, 34, 52, 70, 88}

	// Choose deterministic AMBE params (within bit-lengths)
	// datA: 12-bit, datB: 12-bit, datC: 25-bit
	datAs := []uint32{0x001, 0x123, 0x456, 0x789, 0xABC & 0xFFF}
	datBs := []uint32{0xABC & 0xFFF, 0x0F0, 0x00F, 0x5A5 & 0xFFF, 0x321}
	datCs := []uint32{0x1ABCDE & 0x1FFFFFF, 0x1, 0x155555 & 0x1FFFFFF, 0x0AAAAA & 0x1FFFFFF, 0x1FFFFF}

	for i := 0; i < 5; i++ {
		encA := ysf.Encode24128(datAs[i])
		// For DMR->YSF direction, B in the DMR frame is stored as encB ^ (prng>>1)
		// ambeToYSF expects that and will XOR with (prng>>1) internally to recover encB
		encB := ysf.Encode23127(datBs[i])
		pRNG := prngTable[datAs[i]] >> 1
		bIn := (encB >> 0) ^ pRNG
		v := ambeToYSF(encA, bIn, datCs[i])
		copy(payload[positions[i]:positions[i]+13], v)
	}

	// Feed into converter
	c.PutYSF(payload)

	// We expect 5 mini DMR frames in the dmrBuffer
	for i := 0; i < 5; i++ {
		tag, data := c.dmrBuffer.getData()
		if tag != TagData {
			t.Fatalf("expected TagData from dmrBuffer, got %d", tag)
		}
		if len(data) != 9 {
			t.Fatalf("expected 9-byte mini DMR, got %d bytes", len(data))
		}

		// Extract AMBE parameters from mini DMR
		a, b, c1, _, _, _, _, _, _ := extractAMBEFromDMR(data)

		// Compute expected encoded values per PutYSF pipeline
		// Note: YSF carries only the upper 12 bits derived from encoded A/B
		encA := ysf.Encode24128(datAs[i])
		encB := ysf.Encode23127(datBs[i])
		txA12 := (encA >> 12) & 0xFFF
		txB12 := (encB >> 11) & 0xFFF
		// In YSF->DMR direction, the converter applies PRNG without shifting
		pRNG := prngTable[datAs[i]]
		expA := ysf.Encode24128(txA12)
		expB := (ysf.Encode23127(txB12) ^ pRNG) & 0x7FFFFF
		expC := datCs[i]

		if a != expA {
			t.Fatalf("frame %d: AMBE-A mismatch: got %06X want %06X", i, a, expA)
		}
		if b != expB {
			t.Fatalf("frame %d: AMBE-B mismatch: got %06X want %06X", i, b, expB)
		}
		if c1 != expC {
			t.Fatalf("frame %d: AMBE-C mismatch: got %08X want %08X", i, c1, expC)
		}
	}
}
