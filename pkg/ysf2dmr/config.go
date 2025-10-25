package ysf2dmr

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config represents the YSF2DMR bridge configuration
type Config struct {
	YSF     YSFConfig     `mapstructure:"ysf"`
	DMR     DMRConfig     `mapstructure:"dmr"`
	DMRID   DMRIDConfig   `mapstructure:"dmrid"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// YSFConfig holds YSF network configuration
type YSFConfig struct {
	Callsign      string `mapstructure:"callsign"`
	Suffix        string `mapstructure:"suffix"`
	ServerAddress string `mapstructure:"server_address"`
	ServerPort    int    `mapstructure:"server_port"`
	HangTime      int    `mapstructure:"hang_time"` // milliseconds
	Debug         bool   `mapstructure:"debug"`
}

// DMRConfig holds DMR network configuration
type DMRConfig struct {
	ID             uint32  `mapstructure:"id"`
	Callsign       string  `mapstructure:"callsign"`
	StartupTG      uint32  `mapstructure:"startup_tg"`
	StartupPrivate bool    `mapstructure:"startup_private"`
	ServerAddress  string  `mapstructure:"server_address"`
	ServerPort     int     `mapstructure:"server_port"`
	Password       string  `mapstructure:"password"`
	ColorCode      int     `mapstructure:"color_code"`
	RXFreq         uint32  `mapstructure:"rx_freq"`
	TXFreq         uint32  `mapstructure:"tx_freq"`
	TXPower        int     `mapstructure:"tx_power"`
	Latitude       float64 `mapstructure:"latitude"`
	Longitude      float64 `mapstructure:"longitude"`
	Height         int     `mapstructure:"height"`
	Location       string  `mapstructure:"location"`
	Description    string  `mapstructure:"description"`
	URL            string  `mapstructure:"url"`
	Jitter         int     `mapstructure:"jitter"`
	Debug          bool    `mapstructure:"debug"`
}

// DMRIDConfig holds DMR ID database configuration
type DMRIDConfig struct {
	DatabasePath string        `mapstructure:"database_path"`
	SyncEnabled  bool          `mapstructure:"sync_enabled"`
	SyncInterval time.Duration `mapstructure:"sync_interval"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load loads configuration from file
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
		viper.AddConfigPath("/etc/ysf2dmr")
	}

	// Environment variables
	viper.SetEnvPrefix("YSF2DMR")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			return nil, fmt.Errorf("config file not found: %w", err)
		} else if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file does not exist: %w", err)
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
	// YSF defaults
	viper.SetDefault("ysf.suffix", "ND")
	viper.SetDefault("ysf.hang_time", 1000)
	viper.SetDefault("ysf.debug", false)

	// DMR defaults
	viper.SetDefault("dmr.color_code", 1)
	viper.SetDefault("dmr.rx_freq", 435000000)
	viper.SetDefault("dmr.tx_freq", 435000000)
	viper.SetDefault("dmr.tx_power", 1)
	viper.SetDefault("dmr.latitude", 0.0)
	viper.SetDefault("dmr.longitude", 0.0)
	viper.SetDefault("dmr.height", 0)
	viper.SetDefault("dmr.location", "Unknown")
	viper.SetDefault("dmr.description", "YSF2DMR Bridge")
	viper.SetDefault("dmr.url", "")
	viper.SetDefault("dmr.jitter", 500)
	viper.SetDefault("dmr.startup_private", false)
	viper.SetDefault("dmr.debug", false)

	// DMRID defaults
	viper.SetDefault("dmrid.database_path", "data/dmr-nexus.db")
	viper.SetDefault("dmrid.sync_enabled", true)
	viper.SetDefault("dmrid.sync_interval", "24h")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
}

// validate validates the configuration
func validate(cfg *Config) error {
	// YSF validation
	if cfg.YSF.Callsign == "" {
		return fmt.Errorf("ysf.callsign is required")
	}
	if cfg.YSF.ServerAddress == "" {
		return fmt.Errorf("ysf.server_address is required")
	}
	if cfg.YSF.ServerPort <= 0 || cfg.YSF.ServerPort > 65535 {
		return fmt.Errorf("ysf.server_port must be between 1 and 65535")
	}
	if cfg.YSF.HangTime < 0 {
		return fmt.Errorf("ysf.hang_time must be >= 0")
	}

	// DMR validation
	if cfg.DMR.ID == 0 {
		return fmt.Errorf("dmr.id is required")
	}
	if cfg.DMR.Callsign == "" {
		return fmt.Errorf("dmr.callsign is required")
	}
	if cfg.DMR.ServerAddress == "" {
		return fmt.Errorf("dmr.server_address is required")
	}
	if cfg.DMR.ServerPort <= 0 || cfg.DMR.ServerPort > 65535 {
		return fmt.Errorf("dmr.server_port must be between 1 and 65535")
	}
	if cfg.DMR.Password == "" {
		return fmt.Errorf("dmr.password is required")
	}
	if cfg.DMR.StartupTG == 0 {
		return fmt.Errorf("dmr.startup_tg is required")
	}
	if cfg.DMR.ColorCode < 0 || cfg.DMR.ColorCode > 15 {
		return fmt.Errorf("dmr.color_code must be between 0 and 15")
	}

	// DMRID validation
	if cfg.DMRID.DatabasePath == "" {
		return fmt.Errorf("dmrid.database_path is required")
	}

	// Logging validation
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[cfg.Logging.Level] {
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error")
	}

	validFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validFormats[cfg.Logging.Format] {
		return fmt.Errorf("logging.format must be one of: text, json")
	}

	return nil
}
