package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration
// Follows SRP - only responsible for configuration management
type Config struct {
	MQTT          MQTTConfig          `yaml:"mqtt"`
	HomeAssistant HAConfig            `yaml:"homeassistant"`
	Modbus        ModbusConfig        `yaml:"modbus"`
	Registers     map[string]Register `yaml:"registers"`
	Logging       LoggingConfig       `yaml:"logging"`
}

// MQTTConfig contains MQTT broker and gateway settings
type MQTTConfig struct {
	Broker     string        `yaml:"broker"`
	Port       int           `yaml:"port"`
	Username   string        `yaml:"username"`
	Password   string        `yaml:"password"`
	ClientID   string        `yaml:"client_id"`
	RetryDelay int           `yaml:"retry_delay"` // Delay between connection retries in milliseconds
	Gateway    GatewayConfig `yaml:"gateway"`
}

// GatewayConfig contains USR-DR164 gateway specific settings
type GatewayConfig struct {
	MAC       string `yaml:"mac"`
	CmdTopic  string `yaml:"cmd_topic"`
	DataTopic string `yaml:"data_topic"`
}

// HAConfig contains Home Assistant MQTT Discovery settings
type HAConfig struct {
	DiscoveryPrefix string `yaml:"discovery_prefix"`
	DeviceName      string `yaml:"device_name"`
	DeviceID        string `yaml:"device_id"`
	Manufacturer    string `yaml:"manufacturer"`
	Model           string `yaml:"model"`
	StatusTopic     string `yaml:"status_topic"`
	DiagnosticTopic string `yaml:"diagnostic_topic"`
}

// ModbusConfig contains Modbus device settings
type ModbusConfig struct {
	SlaveID       uint8 `yaml:"slave_id"`
	PollInterval  int   `yaml:"poll_interval"`
	RegisterDelay int   `yaml:"register_delay"`
	EnergyDelay   int   `yaml:"energy_delay"`
	Timeout       int   `yaml:"timeout"`
}

// Register represents a Modbus register configuration
// Used by Strategy Pattern implementations
type Register struct {
	Name        string `yaml:"name"`
	Address     uint16 `yaml:"address"`
	Unit        string `yaml:"unit"`
	DeviceClass string `yaml:"device_class"`
	StateClass  string `yaml:"state_class"`
	HATopic     string `yaml:"ha_topic"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level   string `yaml:"level"`
	File    string `yaml:"file"`
	MaxSize int    `yaml:"max_size"`
	MaxAge  int    `yaml:"max_age"`
}

// LoadConfig loads configuration from specified file
func LoadConfig(configPath string) (*Config, error) {
	// Try to find configuration file in different locations
	paths := []string{
		configPath,
		"/etc/mqtt-modbus-bridge/config.yaml",
		"/etc/mqtt-modbus-bridge.yaml",
		"./config.yaml",
	}

	var data []byte
	var err error
	var usedPath string

	for _, path := range paths {
		if path == "" {
			continue
		}
		data, err = os.ReadFile(path)
		if err == nil {
			usedPath = path
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("cannot read configuration file from any of the locations: %v. Last error: %w", paths, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing configuration from %s: %w", usedPath, err)
	}

	// Configuration validation
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration in %s: %w", usedPath, err)
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.MQTT.Broker == "" {
		return fmt.Errorf("MQTT broker is not specified")
	}
	if c.MQTT.Port <= 0 {
		return fmt.Errorf("MQTT port must be positive")
	}
	if c.MQTT.Gateway.MAC == "" {
		return fmt.Errorf("gateway MAC is not specified")
	}
	if c.Modbus.SlaveID == 0 {
		return fmt.Errorf("Modbus Slave ID must be specified")
	}
	if c.Modbus.PollInterval <= 0 {
		return fmt.Errorf("Modbus poll interval must be positive")
	}
	if c.Modbus.RegisterDelay < 0 {
		return fmt.Errorf("Modbus register delay must be non-negative")
	}
	if c.Modbus.EnergyDelay < 0 {
		return fmt.Errorf("Modbus energy delay must be non-negative")
	}
	if len(c.Registers) == 0 {
		return fmt.Errorf("no registers are defined")
	}
	if c.HomeAssistant.StatusTopic == "" {
		return fmt.Errorf("Home Assistant status topic is not specified")
	}
	if c.HomeAssistant.DiagnosticTopic == "" {
		return fmt.Errorf("Home Assistant diagnostic topic is not specified")
	}

	// Register validation
	for name, reg := range c.Registers {
		if reg.Name == "" {
			return fmt.Errorf("register %s has no name", name)
		}
		if reg.HATopic == "" {
			return fmt.Errorf("register %s has no Home Assistant topic", name)
		}
	}

	return nil
}
