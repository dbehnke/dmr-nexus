package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestLoad_UsesDefaults_WhenNoFile(t *testing.T) {
	// Reset viper to avoid cross-test pollution
	viper.Reset()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// Spot-check a few defaults
	if cfg.Web.Enabled != true {
		t.Errorf("expected Web.Enabled default true, got %v", cfg.Web.Enabled)
	}
	if cfg.Web.Port != 8080 {
		t.Errorf("expected Web.Port default 8080, got %d", cfg.Web.Port)
	}
	if cfg.Global.PingTime != 5 {
		t.Errorf("expected Global.PingTime default 5, got %d", cfg.Global.PingTime)
	}
	if cfg.Global.UseACL != true {
		t.Errorf("expected Global.UseACL default true, got %v", cfg.Global.UseACL)
	}
	if cfg.Logging.Level == "" {
		t.Errorf("expected Logging.Level to be set (default info)")
	}
	if cfg.Metrics.Prometheus.Port != 9090 {
		t.Errorf("expected Prometheus.Port default 9090, got %d", cfg.Metrics.Prometheus.Port)
	}
}

func TestValidate_Errors(t *testing.T) {
	t.Run("invalid global ping_time", func(t *testing.T) {
		cfg := &Config{Global: GlobalConfig{PingTime: 0, MaxMissed: 1}, Web: WebConfig{Enabled: false}}
		if err := validate(cfg); err == nil {
			t.Fatal("expected error for non-positive global.ping_time")
		}
	})

	t.Run("invalid web port when enabled", func(t *testing.T) {
		cfg := &Config{
			Global: GlobalConfig{PingTime: 1, MaxMissed: 1},
			Web:    WebConfig{Enabled: true, Port: 70000},
		}
		if err := validate(cfg); err == nil {
			t.Fatal("expected error for invalid web.port out of range")
		}
	})

	t.Run("peer system missing master_ip", func(t *testing.T) {
		cfg := &Config{
			Global: GlobalConfig{PingTime: 1, MaxMissed: 1},
			Systems: map[string]SystemConfig{
				"peer1": {Enabled: true, Mode: "PEER", Port: 62031, MasterPort: 62031, Passphrase: "x", RadioID: 1},
			},
		}
		if err := validate(cfg); err == nil {
			t.Fatal("expected error for PEER without master_ip")
		}
	})

	t.Run("invalid ACL prefix", func(t *testing.T) {
		cfg := &Config{
			Global: GlobalConfig{PingTime: 1, MaxMissed: 1},
			Systems: map[string]SystemConfig{
				"m1": {Enabled: true, Mode: "MASTER", Port: 62031, Passphrase: "x", MaxPeers: 1, UseACL: true, RegACL: "ALLOW:1"},
			},
		}
		if err := validate(cfg); err == nil {
			t.Fatal("expected error for ACL not starting with PERMIT: or DENY:")
		}
	})

	t.Run("bridge references unknown system", func(t *testing.T) {
		cfg := &Config{
			Global:  GlobalConfig{PingTime: 1, MaxMissed: 1},
			Systems: map[string]SystemConfig{"m1": {Enabled: true, Mode: "MASTER", Port: 1234, Passphrase: "x", MaxPeers: 1}},
			Bridges: map[string][]BridgeRule{
				"b1": {{System: "nope", TGID: 3100, Timeslot: 1}},
			},
		}
		if err := validate(cfg); err == nil {
			t.Fatal("expected error for bridge system not found")
		}
	})
}
