package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Global  GlobalConfig            `mapstructure:"global"`
	Server  ServerConfig            `mapstructure:"server"`
	Web     WebConfig               `mapstructure:"web"`
	Systems map[string]SystemConfig `mapstructure:"systems"`
	Bridges map[string][]BridgeRule `mapstructure:"bridges"`
	MQTT    MQTTConfig              `mapstructure:"mqtt"`
	Logging LoggingConfig           `mapstructure:"logging"`
	Metrics MetricsConfig           `mapstructure:"metrics"`
}

// GlobalConfig holds global DMR configuration
type GlobalConfig struct {
	PingTime          int    `mapstructure:"ping_time"`           // Seconds between pings
	MaxMissed         int    `mapstructure:"max_missed"`          // Max missed pings before timeout
	UseACL            bool   `mapstructure:"use_acl"`             // Enable ACL processing
	RegACL            string `mapstructure:"reg_acl"`             // Registration ACL
	SubACL            string `mapstructure:"sub_acl"`             // Subscriber ACL
	TG1ACL            string `mapstructure:"tg1_acl"`             // Talkgroup timeslot 1 ACL
	TG2ACL            string `mapstructure:"tg2_acl"`             // Talkgroup timeslot 2 ACL
	PrivateCallsEnabled bool `mapstructure:"private_calls_enabled"` // Enable private call routing
}

// ServerConfig holds server identification
type ServerConfig struct {
	Name        string `mapstructure:"name"`
	Description string `mapstructure:"description"`
}

// WebConfig holds web dashboard configuration
type WebConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	AuthRequired bool   `mapstructure:"auth_required"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
}

// SystemConfig represents a single DMR system (MASTER, PEER, or OPENBRIDGE)
type SystemConfig struct {
	Mode    string `mapstructure:"mode"` // MASTER, PEER, OPENBRIDGE
	Enabled bool   `mapstructure:"enabled"`

	// Common fields
	IP         string `mapstructure:"ip"`
	Port       int    `mapstructure:"port"`
	Passphrase string `mapstructure:"passphrase"`

	// MASTER mode specific
	Repeat              bool `mapstructure:"repeat"`
	MaxPeers            int  `mapstructure:"max_peers"`
	PrivateCallsEnabled bool `mapstructure:"private_calls_enabled"` // Enable private call routing

	// PEER mode specific
	Loose       bool    `mapstructure:"loose"`
	MasterIP    string  `mapstructure:"master_ip"`
	MasterPort  int     `mapstructure:"master_port"`
	Callsign    string  `mapstructure:"callsign"`
	RadioID     int     `mapstructure:"radio_id"`
	RXFreq      int     `mapstructure:"rx_freq"`
	TXFreq      int     `mapstructure:"tx_freq"`
	TXPower     int     `mapstructure:"tx_power"`
	ColorCode   int     `mapstructure:"color_code"`
	Latitude    float64 `mapstructure:"latitude"`
	Longitude   float64 `mapstructure:"longitude"`
	Height      int     `mapstructure:"height"`
	Location    string  `mapstructure:"location"`
	Description string  `mapstructure:"description"`
	URL         string  `mapstructure:"url"`
	SoftwareID  string  `mapstructure:"software_id"`
	PackageID   string  `mapstructure:"package_id"`

	// OPENBRIDGE mode specific
	TargetIP   string `mapstructure:"target_ip"`
	TargetPort int    `mapstructure:"target_port"`
	NetworkID  int    `mapstructure:"network_id"`
	BothSlots  bool   `mapstructure:"both_slots"`

	// Common settings
	GroupHangtime int    `mapstructure:"group_hangtime"` // Seconds
	UseACL        bool   `mapstructure:"use_acl"`
	RegACL        string `mapstructure:"reg_acl"`
	SubACL        string `mapstructure:"sub_acl"`
	TG1ACL        string `mapstructure:"tg1_acl"`
	TG2ACL        string `mapstructure:"tg2_acl"`
	TGACL         string `mapstructure:"tg_acl"` // For OPENBRIDGE
}

// BridgeRule represents a conference bridge routing rule
type BridgeRule struct {
	System   string `mapstructure:"system"`
	TGID     int    `mapstructure:"tgid"`
	Timeslot int    `mapstructure:"timeslot"`
	Active   bool   `mapstructure:"active"`
	On       []int  `mapstructure:"on"`      // TGIDs that activate
	Off      []int  `mapstructure:"off"`     // TGIDs that deactivate
	Timeout  int    `mapstructure:"timeout"` // Minutes
	ToType   string `mapstructure:"to_type"` // ON or OFF
}

// MQTTConfig holds MQTT client configuration
type MQTTConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Broker      string `mapstructure:"broker"`
	TopicPrefix string `mapstructure:"topic_prefix"`
	ClientID    string `mapstructure:"client_id"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	QoS         byte   `mapstructure:"qos"`
	Retained    bool   `mapstructure:"retained"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled    bool             `mapstructure:"enabled"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
}

// PrometheusConfig holds Prometheus metrics configuration
type PrometheusConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

// Load loads configuration from file and environment variables
func Load(configFile string) (*Config, error) {
	// Set defaults
	setDefaults()

	// Set config file
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath("/etc/dmr-nexus")
	}

	// Environment variables
	viper.SetEnvPrefix("DMR")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found is OK, use defaults
		} else if os.IsNotExist(err) {
			// File explicitly specified but doesn't exist - that's also OK
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal to struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Global defaults
	viper.SetDefault("global.ping_time", 5)
	viper.SetDefault("global.max_missed", 3)
	viper.SetDefault("global.use_acl", true)
	viper.SetDefault("global.reg_acl", "PERMIT:ALL")
	viper.SetDefault("global.sub_acl", "DENY:1")
	viper.SetDefault("global.tg1_acl", "PERMIT:ALL")
	viper.SetDefault("global.tg2_acl", "PERMIT:ALL")
	viper.SetDefault("global.private_calls_enabled", false)

	// Server defaults
	viper.SetDefault("server.name", "DMR-Nexus")
	viper.SetDefault("server.description", "Go DMR Server")

	// Web defaults
	viper.SetDefault("web.enabled", true)
	viper.SetDefault("web.host", "0.0.0.0")
	viper.SetDefault("web.port", 8080)
	viper.SetDefault("web.auth_required", false)

	// MQTT defaults
	viper.SetDefault("mqtt.enabled", false)
	viper.SetDefault("mqtt.topic_prefix", "dmr/nexus")
	viper.SetDefault("mqtt.client_id", "dmr-nexus")
	viper.SetDefault("mqtt.qos", 1)
	viper.SetDefault("mqtt.retained", false)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 7)

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.prometheus.enabled", true)
	viper.SetDefault("metrics.prometheus.port", 9090)
	viper.SetDefault("metrics.prometheus.path", "/metrics")
}
