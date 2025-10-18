package protocol

import (
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
)

// ComputeHMAC computes the HMAC-SHA1 signature for OpenBridge packets
// This is used to authenticate packets between OpenBridge systems
func ComputeHMAC(data []byte, passphrase string) []byte {
	h := hmac.New(sha1.New, []byte(passphrase))
	h.Write(data)
	return h.Sum(nil)
}

// VerifyHMAC verifies the HMAC-SHA1 signature for OpenBridge packets
func VerifyHMAC(data []byte, signature []byte, passphrase string) bool {
	expected := ComputeHMAC(data, passphrase)
	return hmac.Equal(expected, signature)
}

// AddOpenBridgeHMAC adds an HMAC-SHA1 signature to a DMRD packet for OpenBridge protocol
// The HMAC is computed over the standard 53-byte DMRD packet and appended as a 20-byte signature
func (p *DMRDPacket) AddOpenBridgeHMAC(passphrase string) error {
	// First encode the standard packet (without HMAC)
	p.HMAC = nil
	data, err := p.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode packet for HMAC: %w", err)
	}

	// Compute HMAC over the 53-byte packet
	p.HMAC = ComputeHMAC(data, passphrase)
	return nil
}

// VerifyOpenBridgeHMAC verifies the HMAC-SHA1 signature of an OpenBridge DMRD packet
func (p *DMRDPacket) VerifyOpenBridgeHMAC(passphrase string) bool {
	if len(p.HMAC) == 0 {
		return false
	}

	// Encode the packet without HMAC
	savedHMAC := p.HMAC
	p.HMAC = nil
	data, err := p.Encode()
	p.HMAC = savedHMAC

	if err != nil {
		return false
	}

	// Verify the HMAC
	return VerifyHMAC(data, p.HMAC, passphrase)
}
