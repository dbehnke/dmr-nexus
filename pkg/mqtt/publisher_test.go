package mqtt

import (
	"context"
	"testing"
	"time"
)

// TestNewPublisher tests creating a new MQTT publisher
func TestNewPublisher(t *testing.T) {
	config := Config{
		Enabled:     true,
		Broker:      "tcp://localhost:1883",
		TopicPrefix: "dmr/test",
		ClientID:    "test-client",
		QoS:         1,
		Retained:    false,
	}

	pub := New(config, nil)
	if pub == nil {
		t.Fatal("Expected non-nil publisher")
	}

	if pub.config.Broker != config.Broker {
		t.Errorf("Expected broker %s, got %s", config.Broker, pub.config.Broker)
	}
}

// TestPublisher_Start tests starting the publisher (when disabled)
func TestPublisher_StartWhenDisabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	pub := New(config, nil)
	ctx := context.Background()

	err := pub.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error when disabled, got %v", err)
	}
}

// TestPublisher_Stop tests stopping the publisher
func TestPublisher_Stop(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	pub := New(config, nil)

	// Should not panic when stopping without starting
	pub.Stop()
}

// TestPublisher_PublishEvent tests publishing events
func TestPublisher_PublishEvent(t *testing.T) {
	config := Config{
		Enabled:     false,
		TopicPrefix: "dmr/test",
	}

	pub := New(config, nil)

	// Should not error when disabled
	event := PeerConnectEvent{
		PeerID:    312000,
		Callsign:  "W1ABC",
		Timestamp: time.Now(),
	}

	err := pub.PublishPeerConnect(event)
	if err != nil {
		t.Errorf("Expected no error when disabled, got %v", err)
	}
}

// TestPublisher_PublishTrafficEvent tests publishing traffic events
func TestPublisher_PublishTrafficEvent(t *testing.T) {
	config := Config{
		Enabled:     false,
		TopicPrefix: "dmr/test",
	}

	pub := New(config, nil)

	event := TrafficEvent{
		SourceID:    123456,
		DestID:      3100,
		Timeslot:    1,
		StreamID:    12345678,
		Timestamp:   time.Now(),
	}

	err := pub.PublishTraffic(event)
	if err != nil {
		t.Errorf("Expected no error when disabled, got %v", err)
	}
}

// TestPublisher_PublishBridgeEvent tests publishing bridge events
func TestPublisher_PublishBridgeEvent(t *testing.T) {
	config := Config{
		Enabled:     false,
		TopicPrefix: "dmr/test",
	}

	pub := New(config, nil)

	event := BridgeEvent{
		BridgeName: "NATIONWIDE",
		System:     "MASTER-1",
		TGID:       3100,
		Timeslot:   1,
		Active:     true,
		Timestamp:  time.Now(),
	}

	err := pub.PublishBridgeChange(event)
	if err != nil {
		t.Errorf("Expected no error when disabled, got %v", err)
	}
}

// TestTopicFormat tests topic formatting
func TestTopicFormat(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		suffix     string
		expected   string
	}{
		{
			name:     "simple topic",
			prefix:   "dmr/nexus",
			suffix:   "peers/connect",
			expected: "dmr/nexus/peers/connect",
		},
		{
			name:     "trailing slash in prefix",
			prefix:   "dmr/nexus/",
			suffix:   "peers/connect",
			expected: "dmr/nexus/peers/connect",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			suffix:   "peers/connect",
			expected: "peers/connect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				TopicPrefix: tt.prefix,
			}
			pub := New(config, nil)
			topic := pub.formatTopic(tt.suffix)
			if topic != tt.expected {
				t.Errorf("Expected topic %s, got %s", tt.expected, topic)
			}
		})
	}
}

// TestEventSerialization tests that events can be serialized to JSON
func TestEventSerialization(t *testing.T) {
	tests := []struct {
		name  string
		event interface{}
	}{
		{
			name: "PeerConnectEvent",
			event: PeerConnectEvent{
				PeerID:    312000,
				Callsign:  "W1ABC",
				Timestamp: time.Now(),
			},
		},
		{
			name: "PeerDisconnectEvent",
			event: PeerDisconnectEvent{
				PeerID:    312000,
				Callsign:  "W1ABC",
				Reason:    "timeout",
				Timestamp: time.Now(),
			},
		},
		{
			name: "TrafficEvent",
			event: TrafficEvent{
				SourceID:  123456,
				DestID:    3100,
				Timeslot:  1,
				StreamID:  12345678,
				Timestamp: time.Now(),
			},
		},
		{
			name: "BridgeEvent",
			event: BridgeEvent{
				BridgeName: "NATIONWIDE",
				System:     "MASTER-1",
				TGID:       3100,
				Timeslot:   1,
				Active:     true,
				Timestamp:  time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Enabled: false,
			}
			pub := New(config, nil)
			
			_, err := pub.serializeEvent(tt.event)
			if err != nil {
				t.Errorf("Failed to serialize %s: %v", tt.name, err)
			}
		})
	}
}
