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

	// FICH (Frame Information Channel Header) configuration
	FICHCallSign      byte   `mapstructure:"fich_callsign"`       // 0=use radio ID, 1=use callsign
	FICHCallMode      byte   `mapstructure:"fich_callmode"`       // 0=group, 1=individual
	FICHFrameTotal    byte   `mapstructure:"fich_frametotal"`     // Frame total (0-7)
	FICHMessageRoute  byte   `mapstructure:"fich_messageroute"`   // 0=local, 1=network
	FICHVOIP          byte   `mapstructure:"fich_voip"`           // 0=off, 1=on
	FICHDataType      byte   `mapstructure:"fich_datatype"`       // 0=voice/data, 1=data, 2=voice
	FICHSQLType       byte   `mapstructure:"fich_sqltype"`        // 0=off, 1=on
	FICHSQLCode       byte   `mapstructure:"fich_sqlcode"`        // SQL code (0-255)
	YSFRadioID        string `mapstructure:"ysf_radioid"`         // YSF radio ID (5 characters)
	YSFDT1            []byte `mapstructure:"ysf_dt1"`             // DT1 data (10 bytes)
	YSFDT2            []byte `mapstructure:"ysf_dt2"`             // DT2 data (10 bytes)
}

// DMRConfig holds DMR network configuration
type DMRConfig struct {
	ID             uint32  `mapstructure:"id"`
	Callsign       string  `mapstructure:"callsign"`
	StartupTG      uint32  `mapstructure:"startup_tg"`
	StartupPrivate bool    `mapstructure:"startup_private"`
	StartupPTT     bool    `mapstructure:"startup_ptt"` // Send PTT (dummy frame) on startup to activate talkgroup
	Timeslot       int     `mapstructure:"timeslot"`
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

	// FICH defaults (matching MMDVM_CM defaults)
	viper.SetDefault("ysf.fich_callsign", 0)      // Use radio ID
	viper.SetDefault("ysf.fich_callmode", 0)      // Group call
	viper.SetDefault("ysf.fich_frametotal", 7)    // Frame total 7
	viper.SetDefault("ysf.fich_messageroute", 0)  // Local
	viper.SetDefault("ysf.fich_voip", 0)          // Off
	viper.SetDefault("ysf.fich_datatype", 2)      // Voice
	viper.SetDefault("ysf.fich_sqltype", 0)       // Off
	viper.SetDefault("ysf.fich_sqlcode", 0)       // Code 0
	viper.SetDefault("ysf.ysf_radioid", "*****")  // Default radio ID
	viper.SetDefault("ysf.ysf_dt1", []byte{0x31, 0x22, 0x62, 0x5F, 0x29, 0x00, 0x00, 0x00, 0x00, 0x00})
	viper.SetDefault("ysf.ysf_dt2", []byte{0x00, 0x00, 0x00, 0x00, 0x6C, 0x20, 0x1C, 0x20, 0x03, 0x08})

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
	viper.SetDefault("dmr.startup_ptt", true) // Send PTT on startup by default
	viper.SetDefault("dmr.timeslot", 2)
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
	if cfg.DMR.Timeslot != 1 && cfg.DMR.Timeslot != 2 {
		return fmt.Errorf("dmr.timeslot must be 1 or 2")
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
