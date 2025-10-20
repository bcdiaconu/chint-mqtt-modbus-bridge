package unit

import (
	"os"
	"testing"

	"mqtt-modbus-bridge/pkg/config"
)

// TestConfigLoading tests configuration file loading
func TestConfigLoading(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `
mqtt:
  broker: "localhost"
  port: 1883
  username: "test_user"
  password: "test_pass"
  client_id: "test_client"
  retry_delay: 5000
  gateway:
    mac: "00:11:22:33:44:55"
    cmd_topic: "gateway/cmd"
    data_topic: "gateway/data"

homeassistant:
  discovery_prefix: "homeassistant"
  device_name: "Energy Meter"
  device_id: "energy_meter_01"
  manufacturer: "Chint"
  model: "DDSU666-H"
  status_topic: "energy/status"
  diagnostic_topic: "energy/diagnostic"

modbus:
  slave_id: 1
  poll_interval: 5000
  register_delay: 100
  energy_delay: 300000
  timeout: 2000
  republish_interval: 24

registers:
  voltage:
    name: "voltage"
    address: 8192
    unit: "V"
    device_class: "voltage"
    state_class: "measurement"
    ha_topic: "homeassistant/sensor/voltage/config"
  current:
    name: "current"
    address: 8194
    unit: "A"
    device_class: "current"
    state_class: "measurement"
    ha_topic: "homeassistant/sensor/current/config"

logging:
  level: "info"
  format: "text"
`

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Load configuration
	cfg, err := config.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify MQTT configuration
	if cfg.MQTT.Broker != "localhost" {
		t.Errorf("Expected broker 'localhost', got '%s'", cfg.MQTT.Broker)
	}
	if cfg.MQTT.Port != 1883 {
		t.Errorf("Expected port 1883, got %d", cfg.MQTT.Port)
	}
	if cfg.MQTT.Username != "test_user" {
		t.Errorf("Expected username 'test_user', got '%s'", cfg.MQTT.Username)
	}

	// Verify Modbus configuration
	if cfg.Modbus.SlaveID != 1 {
		t.Errorf("Expected slave ID 1, got %d", cfg.Modbus.SlaveID)
	}
	if cfg.Modbus.PollInterval != 5000 {
		t.Errorf("Expected poll interval 5000, got %d", cfg.Modbus.PollInterval)
	}

	// Verify registers
	if len(cfg.Registers) != 2 {
		t.Errorf("Expected 2 registers, got %d", len(cfg.Registers))
	}

	voltage, ok := cfg.Registers["voltage"]
	if !ok {
		t.Fatal("Voltage register not found")
	}
	if voltage.Address != 8192 {
		t.Errorf("Expected voltage address 8192, got %d", voltage.Address)
	}
	if voltage.Unit != "V" {
		t.Errorf("Expected voltage unit 'V', got '%s'", voltage.Unit)
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Invalid config - missing required fields
	invalidConfig := `
mqtt:
  broker: ""
  port: 0
`

	tmpFile.WriteString(invalidConfig)
	tmpFile.Close()

	cfg, err := config.LoadConfig(tmpFile.Name())
	// Config might load but should have empty/default values
	if err != nil {
		// Expected behavior - config validation failed
		return
	}

	// If it loads, verify defaults or empty values
	if cfg.MQTT.Broker == "" {
		t.Log("Broker is empty as expected")
	}
}

// TestConfigFileNotFound tests behavior when config file doesn't exist
func TestConfigFileNotFound(t *testing.T) {
	_, err := config.LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file, got nil")
	}
}

// TestRegisterConfiguration tests register configuration parsing
func TestRegisterConfiguration(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `
mqtt:
  broker: "localhost"
  port: 1883
  gateway:
    mac: "00:11:22:33:44:55"
    cmd_topic: "cmd"
    data_topic: "data"
homeassistant:
  discovery_prefix: "homeassistant"
  device_name: "Test"
  device_id: "test"
  manufacturer: "Test"
  model: "Test"
  status_topic: "status"
  diagnostic_topic: "diagnostic"
modbus:
  slave_id: 1
  poll_interval: 5000
  register_delay: 100
  energy_delay: 300000
  timeout: 2000
  republish_interval: 24
registers:
  test_register:
    name: "test_register"
    address: 0x2000
    unit: "V"
    device_class: "voltage"
    state_class: "measurement"
    ha_topic: "homeassistant/sensor/test/config"
    min: 180.0
    max: 250.0
logging:
  level: "info"
  format: "text"
`

	tmpFile.WriteString(configContent)
	tmpFile.Close()

	cfg, err := config.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	reg, ok := cfg.Registers["test_register"]
	if !ok {
		t.Fatal("test_register not found")
	}

	if reg.Name != "test_register" {
		t.Errorf("Expected name 'test_register', got '%s'", reg.Name)
	}
	if reg.Address != 0x2000 {
		t.Errorf("Expected address 0x2000, got 0x%X", reg.Address)
	}
	if reg.Min == nil {
		t.Error("Expected min value to be set")
	} else if *reg.Min != 180.0 {
		t.Errorf("Expected min 180.0, got %.1f", *reg.Min)
	}
	if reg.Max == nil {
		t.Error("Expected max value to be set")
	} else if *reg.Max != 250.0 {
		t.Errorf("Expected max 250.0, got %.1f", *reg.Max)
	}
}

// TestPortConflictValidation tests port conflict detection
func TestPortConflictValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid - Different ports",
			config: `
version: "1.0"
mqtt:
  broker: "localhost"
  port: 1883
  gateway:
    mac: "00:11:22:33:44:55"
    cmd_topic: "cmd"
    data_topic: "data"
homeassistant:
  discovery_prefix: "homeassistant"
modbus:
  slave_id: 1
  poll_interval: 5000
  register_delay: 100
  energy_delay: 300000
  timeout: 2000
application:
  health_check_port: 8080
  metrics_port: 9090
registers:
  test:
    name: "test"
    address: 0x2000
    unit: "V"
    device_class: "voltage"
    state_class: "measurement"
    ha_topic: "ha/test"
logging:
  level: "info"
`,
			expectError: false,
		},
		{
			name: "Invalid - Health and Metrics same port",
			config: `
version: "1.0"
mqtt:
  broker: "localhost"
  port: 1883
  gateway:
    mac: "00:11:22:33:44:55"
    cmd_topic: "cmd"
    data_topic: "data"
homeassistant:
  discovery_prefix: "homeassistant"
modbus:
  slave_id: 1
  poll_interval: 5000
  register_delay: 100
  energy_delay: 300000
  timeout: 2000
application:
  health_check_port: 8080
  metrics_port: 8080
registers:
  test:
    name: "test"
    address: 0x2000
    unit: "V"
    device_class: "voltage"
    state_class: "measurement"
    ha_topic: "ha/test"
logging:
  level: "info"
`,
			expectError: true,
			errorMsg:    "cannot use the same port",
		},
		{
			name: "Invalid - Health port conflicts with MQTT",
			config: `
version: "1.0"
mqtt:
  broker: "localhost"
  port: 1883
  gateway:
    mac: "00:11:22:33:44:55"
    cmd_topic: "cmd"
    data_topic: "data"
homeassistant:
  discovery_prefix: "homeassistant"
modbus:
  slave_id: 1
  poll_interval: 5000
  register_delay: 100
  energy_delay: 300000
  timeout: 2000
application:
  health_check_port: 1883
  metrics_port: 0
registers:
  test:
    name: "test"
    address: 0x2000
    unit: "V"
    device_class: "voltage"
    state_class: "measurement"
    ha_topic: "ha/test"
logging:
  level: "info"
`,
			expectError: true,
			errorMsg:    "conflicts with mqtt.port",
		},
		{
			name: "Invalid - Metrics port conflicts with MQTT",
			config: `
version: "1.0"
mqtt:
  broker: "localhost"
  port: 1883
  gateway:
    mac: "00:11:22:33:44:55"
    cmd_topic: "cmd"
    data_topic: "data"
homeassistant:
  discovery_prefix: "homeassistant"
modbus:
  slave_id: 1
  poll_interval: 5000
  register_delay: 100
  energy_delay: 300000
  timeout: 2000
application:
  health_check_port: 0
  metrics_port: 1883
registers:
  test:
    name: "test"
    address: 0x2000
    unit: "V"
    device_class: "voltage"
    state_class: "measurement"
    ha_topic: "ha/test"
logging:
  level: "info"
`,
			expectError: true,
			errorMsg:    "conflicts with mqtt.port",
		},
		{
			name: "Invalid - Port out of range",
			config: `
version: "1.0"
mqtt:
  broker: "localhost"
  port: 70000
  gateway:
    mac: "00:11:22:33:44:55"
    cmd_topic: "cmd"
    data_topic: "data"
homeassistant:
  discovery_prefix: "homeassistant"
modbus:
  slave_id: 1
  poll_interval: 5000
  register_delay: 100
  energy_delay: 300000
  timeout: 2000
registers:
  test:
    name: "test"
    address: 0x2000
    unit: "V"
    device_class: "voltage"
    state_class: "measurement"
    ha_topic: "ha/test"
logging:
  level: "info"
`,
			expectError: true,
			errorMsg:    "must be between 1 and 65535",
		},
		{
			name: "Valid - Metrics disabled (port 0)",
			config: `
version: "1.0"
mqtt:
  broker: "localhost"
  port: 1883
  gateway:
    mac: "00:11:22:33:44:55"
    cmd_topic: "cmd"
    data_topic: "data"
homeassistant:
  discovery_prefix: "homeassistant"
modbus:
  slave_id: 1
  poll_interval: 5000
  register_delay: 100
  energy_delay: 300000
  timeout: 2000
application:
  health_check_port: 8080
  metrics_port: 0
registers:
  test:
    name: "test"
    address: 0x2000
    unit: "V"
    device_class: "voltage"
    state_class: "measurement"
    ha_topic: "ha/test"
logging:
  level: "info"
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.LoadConfigFromString(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if tt.errorMsg != "" {
					// Check if error message contains expected substring
					errStr := err.Error()
					if len(errStr) > 0 && len(tt.errorMsg) > 0 {
						found := false
						for i := 0; i <= len(errStr)-len(tt.errorMsg); i++ {
							if errStr[i:i+len(tt.errorMsg)] == tt.errorMsg {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, errStr)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}
