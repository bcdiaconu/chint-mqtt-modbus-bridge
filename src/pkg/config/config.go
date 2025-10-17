package config

import (
	"fmt"
	"mqtt-modbus-bridge/pkg/logger"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration
// Follows SRP - only responsible for configuration management
// Supports V1 (registers), V2.0 (register_groups), and V2.1 (devices) formats
type Config struct {
	Version        string                        `yaml:"version,omitempty"` // Configuration version (optional, default 1.0)
	MQTT           MQTTConfig                    `yaml:"mqtt"`
	HomeAssistant  HAConfig                      `yaml:"homeassistant"`
	Modbus         ModbusConfig                  `yaml:"modbus"`
	Registers      map[string]Register           `yaml:"registers,omitempty"`            // V1 format
	RegisterGroups map[string]RegisterGroup      `yaml:"register_groups,omitempty"`      // V2.0 format
	Devices        map[string]Device             `yaml:"devices,omitempty"`              // V2.1 format (recommended)
	CalculatedRegs map[string]CalculatedRegister `yaml:"calculated_registers,omitempty"` // V2.0+ format
	Logging        logger.LoggingConfig          `yaml:"logging"`
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
// Global settings for the bridge (discovery prefix and bridge-level topics)
// Per-device Home Assistant information is configured in Device struct
type HAConfig struct {
	DiscoveryPrefix string `yaml:"discovery_prefix"` // HA MQTT discovery prefix (e.g., "homeassistant")
	StatusTopic     string `yaml:"status_topic"`     // Bridge availability topic
	DiagnosticTopic string `yaml:"diagnostic_topic"` // Bridge diagnostics topic

	// DEPRECATED: These fields are now per-device in Device struct
	// Kept for backward compatibility with V2.0 configs
	DeviceName   string `yaml:"device_name,omitempty"`
	DeviceID     string `yaml:"device_id,omitempty"`
	Manufacturer string `yaml:"manufacturer,omitempty"`
	Model        string `yaml:"model,omitempty"`
}

// ModbusConfig contains Modbus device settings
type ModbusConfig struct {
	SlaveID           uint8 `yaml:"slave_id"`
	PollInterval      int   `yaml:"poll_interval"`
	RegisterDelay     int   `yaml:"register_delay"`
	EnergyDelay       int   `yaml:"energy_delay"`
	Timeout           int   `yaml:"timeout"`
	RepublishInterval int   `yaml:"republish_interval"` // Hours between forced republishing of energy sensors
}

// Register represents a Modbus register configuration
// Used by Strategy Pattern implementations
type Register struct {
	Name          string   `yaml:"name"`
	Address       uint16   `yaml:"address"`
	Unit          string   `yaml:"unit"`
	DeviceClass   string   `yaml:"device_class"`
	StateClass    string   `yaml:"state_class"`
	HATopic       string   `yaml:"ha_topic"`
	Min           *float64 `yaml:"min,omitempty"`              // Minimum valid value (optional)
	Max           *float64 `yaml:"max,omitempty"`              // Maximum valid value (optional)
	MaxKwhPerHour *float64 `yaml:"max_kwh_per_hour,omitempty"` // Maximum kWh change per hour for energy registers (optional)
}

// LoadConfig loads configuration from specified file with version detection
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
		// #nosec G304 - Paths are from a hardcoded list of safe configuration file locations
		data, err = os.ReadFile(path)
		if err == nil {
			usedPath = path
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("cannot read configuration file from any of the locations: %v. Last error: %w", paths, err)
	}

	// First, parse just the version to validate compatibility
	var versionCheck VersionInfo
	if err := yaml.Unmarshal(data, &versionCheck); err != nil {
		return nil, fmt.Errorf("error parsing configuration version from %s: %w", usedPath, err)
	}

	// If no version specified, assume V1 (backward compatibility)
	if versionCheck.Version == "" {
		logger.LogWarn("⚠️  No 'version' field in configuration, assuming legacy format (1.0)")
		versionCheck.Version = "1.0"
	}

	// Validate version compatibility
	if versionCheck.Version == "2.0" {
		if err := ValidateVersion(versionCheck.Version); err != nil {
			logger.LogError("❌ Configuration version incompatibility in %s: %v", usedPath, err)
			logger.LogError("   Current parser version: %s", CurrentVersion)
			logger.LogError("   Minimum compatible version: %s", MinCompatibleVersion)
			logger.LogError("   File version: %s", versionCheck.Version)
			return nil, err
		}
		logger.LogInfo("✅ Configuration version %s is compatible (parser version: %s)",
			versionCheck.Version, CurrentVersion)
	}

	// Parse the full configuration
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing configuration from %s: %w", usedPath, err)
	}

	// Set version if not present (backward compatibility)
	if config.Version == "" {
		config.Version = "1.0"
	}

	// Configuration validation
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration in %s: %w", usedPath, err)
	}

	logger.LogInfo("✅ Configuration loaded successfully from %s (version: %s)", usedPath, config.Version)
	return &config, nil
}

// LoadConfigFromString loads configuration from a YAML string (for testing)
func LoadConfigFromString(yamlContent string) (*Config, error) {
	data := []byte(yamlContent)

	// First, parse just the version to validate compatibility
	var versionCheck VersionInfo
	if err := yaml.Unmarshal(data, &versionCheck); err != nil {
		return nil, fmt.Errorf("error parsing configuration version: %w", err)
	}

	// If no version specified, assume V1 (backward compatibility)
	if versionCheck.Version == "" {
		versionCheck.Version = "1.0"
	}

	// Parse the full configuration
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing configuration: %w", err)
	}

	// Set version if not present (backward compatibility)
	if config.Version == "" {
		config.Version = "1.0"
	}

	// Configuration validation
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate checks if the configuration is valid
// Supports both V1 (registers) and V2 (register_groups) formats
func (c *Config) Validate() error {
	// Common validation
	if c.MQTT.Broker == "" {
		return fmt.Errorf("mqtt.broker is not specified")
	}
	if c.MQTT.Port <= 0 {
		return fmt.Errorf("mqtt.port must be positive")
	}
	if c.MQTT.Gateway.MAC == "" {
		return fmt.Errorf("mqtt.gateway.mac is not specified")
	}
	if c.Modbus.PollInterval <= 0 {
		return fmt.Errorf("modbus.poll_interval must be positive")
	}
	if c.Modbus.RegisterDelay < 0 {
		return fmt.Errorf("modbus.register_delay must be non-negative")
	}
	if c.Modbus.EnergyDelay < 0 {
		return fmt.Errorf("modbus.energy_delay must be non-negative")
	}
	if c.HomeAssistant.StatusTopic == "" {
		return fmt.Errorf("homeassistant.status_topic is not specified")
	}
	if c.HomeAssistant.DiagnosticTopic == "" {
		return fmt.Errorf("homeassistant.diagnostic_topic is not specified")
	}

	// Version-specific validation
	if c.Version == "2.1" {
		// V2.1 format validation (device-based)
		if c.Modbus.SlaveID == 0 {
			logger.LogWarn("⚠️  modbus.slave_id is 0 (devices specify their own slave_id)")
		}

		// Validate devices (V2.1 format - preferred)
		if len(c.Devices) > 0 {
			if err := ValidateDevices(c.Devices); err != nil {
				return err
			}

			// Convert devices to flat groups for backward compatibility
			if len(c.RegisterGroups) == 0 {
				c.RegisterGroups = ConvertDevicesToGroups(c.Devices)
				logger.LogInfo("✅ Converted %d devices to %d register groups",
					len(c.Devices), len(c.RegisterGroups))
			}
		} else if len(c.RegisterGroups) > 0 {
			// Fallback to V2.0 format (flat groups)
			logger.LogWarn("⚠️  Using V2.0 format (register_groups). Consider upgrading to V2.1 (devices)")
			if err := ValidateGroups(c.RegisterGroups, c.CalculatedRegs); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("V2.1 config requires either 'devices' or 'register_groups'")
		}

		// Convert groups to registers for backward compatibility with existing code
		if len(c.Registers) == 0 {
			c.Registers = ConvertGroupsToRegisters(c.RegisterGroups)
			logger.LogInfo("✅ Converted %d register groups to %d individual registers",
				len(c.RegisterGroups), len(c.Registers))
		}
	} else if c.Version == "2.0" {
		// V2.0 format validation (flat groups)
		if c.Modbus.SlaveID == 0 {
			logger.LogWarn("⚠️  modbus.slave_id is 0, groups must specify their own slave_id")
		}

		// Validate register groups (V2.0 format)
		if err := ValidateGroups(c.RegisterGroups, c.CalculatedRegs); err != nil {
			return err
		}

		// Convert groups to registers for backward compatibility with existing code
		if len(c.Registers) == 0 {
			c.Registers = ConvertGroupsToRegisters(c.RegisterGroups)
			logger.LogInfo("✅ Converted %d register groups to %d individual registers",
				len(c.RegisterGroups), len(c.Registers))
		}
	} else {
		// V1 format validation
		if c.Modbus.SlaveID == 0 {
			return fmt.Errorf("modbus.slave_id must be specified")
		}
		if len(c.Registers) == 0 {
			return fmt.Errorf("no registers are defined")
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
	}

	return nil
}
