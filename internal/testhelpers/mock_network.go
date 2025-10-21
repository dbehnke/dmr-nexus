package testhelpers

import (
	"net"
	"sync"
)

// MockNetwork simulates a network for testing
type MockNetwork struct {
	mu          sync.RWMutex
	listeners   map[string]*MockListener
	sentPackets []MockPacket
}

// MockPacket represents a packet sent on the mock network
type MockPacket struct {
	From string
	To   string
	Data []byte
}

// MockListener is a mock UDP listener
type MockListener struct {
	addr    *net.UDPAddr
	packets chan []byte
	closed  bool
	mu      sync.RWMutex
}

// NewMockNetwork creates a new mock network
func NewMockNetwork() *MockNetwork {
	return &MockNetwork{
		listeners:   make(map[string]*MockListener),
		sentPackets: make([]MockPacket, 0),
	}
}

// CreateListener creates a mock listener on the specified address
func (n *MockNetwork) CreateListener(addr string) (*MockListener, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	listener := &MockListener{
		addr:    udpAddr,
		packets: make(chan []byte, 100),
	}

	n.listeners[addr] = listener
	return listener, nil
}

// SendPacket records a packet being sent
func (n *MockNetwork) SendPacket(from, to string, data []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()

	packet := MockPacket{
		From: from,
		To:   to,
		Data: make([]byte, len(data)),
	}
	copy(packet.Data, data)

	n.sentPackets = append(n.sentPackets, packet)

	// Deliver to listener if it exists
	if listener, ok := n.listeners[to]; ok {
		select {
		case listener.packets <- packet.Data:
		default:
			// Buffer full, drop packet
		}
	}
}

// GetSentPackets returns all sent packets
func (n *MockNetwork) GetSentPackets() []MockPacket {
	n.mu.RLock()
	defer n.mu.RUnlock()

	packets := make([]MockPacket, len(n.sentPackets))
	copy(packets, n.sentPackets)
	return packets
}

// Close closes the mock network
func (n *MockNetwork) Close() {
	n.mu.Lock()
	defer n.mu.Unlock()

	for _, listener := range n.listeners {
		listener.Close()
	}
}

// Receive receives a packet from the listener
func (l *MockListener) Receive() ([]byte, error) {
	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return nil, net.ErrClosed
	}
	l.mu.RUnlock()

	select {
	case packet := <-l.packets:
		return packet, nil
	default:
		return nil, nil
	}
}

// Close closes the listener
func (l *MockListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true
	close(l.packets)
	return nil
}

// GetPacketCount returns the number of packets received
func (l *MockListener) GetPacketCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.packets)
}
