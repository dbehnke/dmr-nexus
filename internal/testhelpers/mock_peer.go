package testhelpers

import (
	"net"
	"sync"
	"time"

	"github.com/dbehnke/dmr-nexus/pkg/protocol"
)

// MockPeer simulates a DMR peer/repeater for testing
type MockPeer struct {
	PeerID     uint32
	Passphrase string
	Callsign   string
	conn       *net.UDPConn
	masterAddr *net.UDPAddr
	mu         sync.RWMutex
	packets    [][]byte
	closed     bool
}

// NewMockPeer creates a new mock peer
func NewMockPeer(peerID uint32, passphrase string, callsign string) *MockPeer {
	return &MockPeer{
		PeerID:     peerID,
		Passphrase: passphrase,
		Callsign:   callsign,
		packets:    make([][]byte, 0),
	}
}

// Connect connects the mock peer to a master
func (m *MockPeer) Connect(masterAddr string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	addr, err := net.ResolveUDPAddr("udp", masterAddr)
	if err != nil {
		return err
	}
	m.masterAddr = addr

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	m.conn = conn

	return nil
}

// SendRPTL sends a login packet to the master
func (m *MockPeer) SendRPTL() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.conn == nil {
		return nil
	}

	packet := &protocol.RPTLPacket{
		RepeaterID: m.PeerID,
	}
	data, err := packet.Encode()
	if err != nil {
		return err
	}

	_, err = m.conn.Write(data)
	return err
}

// SendDMRD sends a DMRD packet
func (m *MockPeer) SendDMRD(sourceID, destID uint32, timeslot uint8, streamID uint32, seq uint8, payload []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.conn == nil {
		return nil
	}

	ts := protocol.Timeslot1
	if timeslot == 2 {
		ts = protocol.Timeslot2
	}

	packet := &protocol.DMRDPacket{
		Sequence:      seq,
		SourceID:      sourceID,
		DestinationID: destID,
		RepeaterID:    m.PeerID,
		Timeslot:      ts,
		CallType:      protocol.CallTypeGroup,
		FrameType:     0,
		DataType:      0,
		StreamID:      streamID,
		Payload:       payload,
	}
	data, err := packet.Encode()
	if err != nil {
		return err
	}

	_, err = m.conn.Write(data)
	return err
}

// ReceivePacket receives a packet from the master
func (m *MockPeer) ReceivePacket(timeout time.Duration) ([]byte, error) {
	m.mu.RLock()
	conn := m.conn
	m.mu.RUnlock()

	if conn == nil {
		return nil, nil
	}

	conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	packet := make([]byte, n)
	copy(packet, buf[:n])

	m.mu.Lock()
	m.packets = append(m.packets, packet)
	m.mu.Unlock()

	return packet, nil
}

// GetReceivedPackets returns all received packets
func (m *MockPeer) GetReceivedPackets() [][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	packets := make([][]byte, len(m.packets))
	copy(packets, m.packets)
	return packets
}

// Close closes the mock peer connection
func (m *MockPeer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	if m.conn != nil {
		return m.conn.Close()
	}
	return nil
}

// IsConnected returns whether the peer is connected
func (m *MockPeer) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conn != nil && !m.closed
}
