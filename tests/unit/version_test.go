package unit

import (
	"mqtt-modbus-bridge/pkg/config"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigVersionValidation(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid version 2.0",
			configYAML: `version: "2.0"
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "test"
  gateway:
    mac: "test"
    cmd_topic: "cmd"
    data_topic: "data"
homeassistant:
  status_topic: "status"
  diagnostic_topic: "diag"
modbus:
  slave_id: 11
  poll_interval: 1000
register_groups:
  test:
    name: "Test Group"
    slave_id: 11
    function_code: 0x03
    start_address: 0x2000
    register_count: 10
    enabled: true
    registers:
      - key: "test_reg"
        name: "Test"
        offset: 0
        unit: "V"
logging:
  level: "info"
`,
			shouldError: false,
		},
		{
			name: "missing version defaults to 1.0",
			configYAML: `mqtt:
  broker: "localhost"
  port: 1883
  client_id: "test"
  gateway:
    mac: "test"
    cmd_topic: "cmd"
    data_topic: "data"
homeassistant:
  status_topic: "status"
  diagnostic_topic: "diag"
modbus:
  slave_id: 11
  poll_interval: 1000
registers:
  test:
    name: "Test"
    address: 0x2000
    unit: "V"
    ha_topic: "test"
logging:
  level: "info"
`,
			shouldError: false,
		},
		{
			name: "incompatible version 3.0",
			configYAML: `version: "3.0"
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "test"
  gateway:
    mac: "test"
    cmd_topic: "cmd"
    data_topic: "data"
homeassistant:
  status_topic: "status"
  diagnostic_topic: "diag"
modbus:
  slave_id: 11
  poll_interval: 1000
logging:
  level: "info"
`,
			shouldError: true,
			errorMsg:    "version", // Just check that version is mentioned in error
		},
		{
			name: "version 2.0 with missing register groups",
			configYAML: `version: "2.0"
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "test"
  gateway:
    mac: "test"
    cmd_topic: "cmd"
    data_topic: "data"
homeassistant:
  status_topic: "status"
  diagnostic_topic: "diag"
modbus:
  slave_id: 11
  poll_interval: 1000
logging:
  level: "info"
`,
			shouldError: true,
			errorMsg:    "at least one register group is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configPath, []byte(tt.configYAML), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config: %v", err)
			}

			// Try to load config
			cfg, err := config.LoadConfig(configPath)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if cfg == nil {
					t.Error("Expected config, got nil")
				}
			}
		})
	}
}

func TestConfigVersion2Conversion(t *testing.T) {
	configYAML := `version: "2.0"
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "test"
  gateway:
    mac: "test"
    cmd_topic: "cmd"
    data_topic: "data"
homeassistant:
  status_topic: "status"
  diagnostic_topic: "diag"
modbus:
  slave_id: 11
  poll_interval: 1000
register_groups:
  instant:
    name: "Instant Readings"
    slave_id: 11
    function_code: 0x03
    start_address: 0x2000
    register_count: 10
    enabled: true
    registers:
      - key: "voltage"
        name: "Voltage"
        offset: 0
        unit: "V"
        ha_topic: "voltage"
      - key: "current"
        name: "Current"
        offset: 4
        unit: "A"
        ha_topic: "current"
logging:
  level: "info"
`

	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify version
	if cfg.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", cfg.Version)
	}

	// Verify register groups exist
	if len(cfg.RegisterGroups) == 0 {
		t.Error("Expected register groups, got none")
	}

	// Verify conversion to registers happened
	if len(cfg.Registers) == 0 {
		t.Error("Expected registers after conversion, got none")
	}

	// Verify converted registers have correct addresses
	voltageReg, ok := cfg.Registers["voltage"]
	if !ok {
		t.Fatal("Voltage register not found after conversion")
	}
	expectedVoltageAddr := uint16(0x2000) // start + offset/2 = 0x2000 + 0/2
	if voltageReg.Address != expectedVoltageAddr {
		t.Errorf("Expected voltage address 0x%04X, got 0x%04X",
			expectedVoltageAddr, voltageReg.Address)
	}

	currentReg, ok := cfg.Registers["current"]
	if !ok {
		t.Fatal("Current register not found after conversion")
	}
	expectedCurrentAddr := uint16(0x2002) // start + offset/2 = 0x2000 + 4/2
	if currentReg.Address != expectedCurrentAddr {
		t.Errorf("Expected current address 0x%04X, got 0x%04X",
			expectedCurrentAddr, currentReg.Address)
	}
}

func TestVersionConstants(t *testing.T) {
	if config.CurrentVersion != "2.0" {
		t.Errorf("Expected CurrentVersion to be 2.0, got %s", config.CurrentVersion)
	}
	if config.MinCompatibleVersion != "2.0" {
		t.Errorf("Expected MinCompatibleVersion to be 2.0, got %s", config.MinCompatibleVersion)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
