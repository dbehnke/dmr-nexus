package config

import (
	"fmt"
	"strings"
)

// validate validates the configuration
func validate(cfg *Config) error {
	// Validate global config
	if cfg.Global.PingTime <= 0 {
		return fmt.Errorf("global.ping_time must be positive")
	}
	if cfg.Global.MaxMissed <= 0 {
		return fmt.Errorf("global.max_missed must be positive")
	}

	// Validate web config
	if cfg.Web.Enabled {
		if cfg.Web.Port <= 0 || cfg.Web.Port > 65535 {
			return fmt.Errorf("web.port must be between 1 and 65535")
		}
	}

	// Validate MQTT config
	if cfg.MQTT.Enabled {
		if cfg.MQTT.Broker == "" {
			return fmt.Errorf("mqtt.broker is required when mqtt is enabled")
		}
	}

	// Validate systems
	for name, sys := range cfg.Systems {
		if !sys.Enabled {
			continue
		}

		// Validate mode
		mode := strings.ToUpper(sys.Mode)
		if mode != "MASTER" && mode != "PEER" && mode != "OPENBRIDGE" {
			return fmt.Errorf("system %s: invalid mode %s (must be MASTER, PEER, or OPENBRIDGE)", name, sys.Mode)
		}

		// Validate port
		if sys.Port <= 0 || sys.Port > 65535 {
			return fmt.Errorf("system %s: port must be between 1 and 65535", name)
		}

		// Mode-specific validation
		switch mode {
		case "MASTER":
			if sys.Passphrase == "" {
				return fmt.Errorf("system %s: passphrase is required for MASTER mode", name)
			}
			if sys.MaxPeers <= 0 {
				return fmt.Errorf("system %s: max_peers must be positive", name)
			}

		case "PEER":
			if sys.MasterIP == "" {
				return fmt.Errorf("system %s: master_ip is required for PEER mode", name)
			}
			if sys.MasterPort <= 0 || sys.MasterPort > 65535 {
				return fmt.Errorf("system %s: master_port must be between 1 and 65535", name)
			}
			if sys.Passphrase == "" {
				return fmt.Errorf("system %s: passphrase is required for PEER mode", name)
			}
			if sys.RadioID <= 0 {
				return fmt.Errorf("system %s: radio_id is required for PEER mode", name)
			}

		case "OPENBRIDGE":
			if sys.TargetIP == "" {
				return fmt.Errorf("system %s: target_ip is required for OPENBRIDGE mode", name)
			}
			if sys.TargetPort <= 0 || sys.TargetPort > 65535 {
				return fmt.Errorf("system %s: target_port must be between 1 and 65535", name)
			}
			if sys.NetworkID <= 0 {
				return fmt.Errorf("system %s: network_id is required for OPENBRIDGE mode", name)
			}
			if sys.Passphrase == "" {
				return fmt.Errorf("system %s: passphrase is required for OPENBRIDGE mode", name)
			}
		}

		// Validate ACLs if enabled
		if sys.UseACL || cfg.Global.UseACL {
			// Just basic format check for now
			acls := []string{sys.RegACL, sys.SubACL, sys.TG1ACL, sys.TG2ACL, sys.TGACL}
			for _, acl := range acls {
				if acl != "" {
					if !strings.HasPrefix(acl, "PERMIT:") && !strings.HasPrefix(acl, "DENY:") {
						return fmt.Errorf("system %s: ACL must start with PERMIT: or DENY:", name)
					}
				}
			}
		}
	}

	// Validate bridge rules
	for bridgeName, rules := range cfg.Bridges {
		for i, rule := range rules {
			if rule.System == "" {
				return fmt.Errorf("bridge %s rule %d: system is required", bridgeName, i)
			}
			if _, exists := cfg.Systems[rule.System]; !exists {
				return fmt.Errorf("bridge %s rule %d: system %s not found", bridgeName, i, rule.System)
			}
			if rule.TGID <= 0 {
				return fmt.Errorf("bridge %s rule %d: tgid must be positive", bridgeName, i)
			}
			if rule.Timeslot != 1 && rule.Timeslot != 2 {
				return fmt.Errorf("bridge %s rule %d: timeslot must be 1 or 2", bridgeName, i)
			}
			if rule.ToType != "" && rule.ToType != "ON" && rule.ToType != "OFF" {
				return fmt.Errorf("bridge %s rule %d: to_type must be ON or OFF", bridgeName, i)
			}
		}
	}

	return nil
}
