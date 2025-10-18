package unit

import (
	"mqtt-modbus-bridge/pkg/config"
	"testing"
)

func TestConfig_DeviceBasedStructure(t *testing.T) {
	// Test YAML parsing with device-based segregated structure
	yamlContent := `
version: "2.1"

mqtt:
  broker: "test.mosquitto.org"
  port: 1883
  username: "test"
  password: "test"
  client_id: "test-client"
  gateway:
    mac: "AABBCCDDEEFF"
    cmd_topic: "test/cmd"
    data_topic: "test/data"

homeassistant:
  discovery_prefix: "homeassistant"
  status_topic: "bridge/status"
  diagnostic_topic: "bridge/diagnostic"

modbus:
  poll_interval: 1000
  register_delay: 50
  energy_delay: 5000
  timeout: 5000
  republish_interval: 24

devices:
  meter_1:
    metadata:
      name: "Test Meter 1"
      manufacturer: "TestCorp"
      model: "TM-100"
      enabled: true
    
    rtu:
      slave_id: 11
      poll_interval: 2000
    
    homeassistant:
      device_id: "test_meter_1"
      manufacturer: "TestCorp Inc."
      model: "TM-100 Advanced"
    
    modbus:
      register_groups:
        instant:
          name: "Instant Values"
          function_code: 0x03
          start_address: 0x2000
          register_count: 10
          enabled: true
          registers:
            - key: "voltage"
              name: "Voltage"
              offset: 0
              unit: "V"
              device_class: "voltage"
              state_class: "measurement"
              ha_topic: "meter1/voltage"
  
  meter_2:
    metadata:
      name: "Test Meter 2"
      manufacturer: "TestCorp"
      model: "TM-200"
      enabled: true
    
    rtu:
      slave_id: 12
    
    homeassistant:
      device_id: "test_meter_2"
    
    modbus:
      register_groups:
        instant:
          name: "Instant Values"
          function_code: 0x03
          start_address: 0x2000
          register_count: 10
          enabled: true
          registers:
            - key: "current"
              name: "Current"
              offset: 0
              unit: "A"
              device_class: "current"
              state_class: "measurement"
              ha_topic: "meter2/current"

logging:
  level: "info"
`

	// Parse the YAML
	cfg, err := config.LoadConfigFromString(yamlContent)
	if err != nil {
		t.Fatalf("Failed to parse V2.1 config: %v", err)
	}

	// Verify version
	if cfg.Version != "2.1" {
		t.Errorf("Expected version 2.1, got %s", cfg.Version)
	}

	// Verify devices exist
	if len(cfg.Devices) != 2 {
		t.Fatalf("Expected 2 devices, got %d", len(cfg.Devices))
	}

	// Test meter_1 structure
	meter1, exists := cfg.Devices["meter_1"]
	if !exists {
		t.Fatal("meter_1 device not found")
	}

	// Verify metadata
	if meter1.Metadata.Name != "Test Meter 1" {
		t.Errorf("Expected name 'Test Meter 1', got '%s'", meter1.Metadata.Name)
	}
	if meter1.Metadata.Manufacturer != "TestCorp" {
		t.Errorf("Expected manufacturer 'TestCorp', got '%s'", meter1.Metadata.Manufacturer)
	}
	if meter1.Metadata.Model != "TM-100" {
		t.Errorf("Expected model 'TM-100', got '%s'", meter1.Metadata.Model)
	}
	if !meter1.Metadata.Enabled {
		t.Error("Expected device to be enabled")
	}

	// Verify RTU config
	if meter1.RTU.SlaveID != 11 {
		t.Errorf("Expected slave_id 11, got %d", meter1.RTU.SlaveID)
	}
	if meter1.RTU.PollInterval != 2000 {
		t.Errorf("Expected poll_interval 2000, got %d", meter1.RTU.PollInterval)
	}

	// Verify Home Assistant config
	if meter1.HomeAssistant == nil {
		t.Fatal("HomeAssistant config should not be nil")
	}
	if meter1.HomeAssistant.DeviceID != "test_meter_1" {
		t.Errorf("Expected device_id 'test_meter_1', got '%s'", meter1.HomeAssistant.DeviceID)
	}
	if meter1.HomeAssistant.Manufacturer != "TestCorp Inc." {
		t.Errorf("Expected HA manufacturer 'TestCorp Inc.', got '%s'", meter1.HomeAssistant.Manufacturer)
	}

	// Verify Modbus config
	if len(meter1.Modbus.RegisterGroups) != 1 {
		t.Fatalf("Expected 1 register group, got %d", len(meter1.Modbus.RegisterGroups))
	}
	instantGroup, exists := meter1.Modbus.RegisterGroups["instant"]
	if !exists {
		t.Fatal("instant group not found")
	}
	if len(instantGroup.Registers) != 1 {
		t.Errorf("Expected 1 register in instant group, got %d", len(instantGroup.Registers))
	}

	// Test meter_2 (with minimal HA config)
	meter2, exists := cfg.Devices["meter_2"]
	if !exists {
		t.Fatal("meter_2 device not found")
	}
	if meter2.RTU.SlaveID != 12 {
		t.Errorf("Expected slave_id 12, got %d", meter2.RTU.SlaveID)
	}
	// Verify HA config inherits from metadata
	if meter2.HomeAssistant.Manufacturer != "" {
		t.Errorf("Expected empty HA manufacturer override, got '%s'", meter2.HomeAssistant.Manufacturer)
	}
}

func TestDevice_GetterMethods(t *testing.T) {
	device := config.Device{
		Metadata: config.DeviceMetadata{
			Name:         "Test Device",
			Manufacturer: "TestCorp",
			Model:        "TD-100",
			Enabled:      true,
		},
		RTU: config.RTUConfig{
			SlaveID:      42,
			PollInterval: 3000,
		},
		HomeAssistant: &config.HADeviceConfig{
			DeviceID:     "test_device_001",
			Manufacturer: "TestCorp Industries",
			Model:        "TD-100 Pro",
		},
		Modbus: config.ModbusDeviceConfig{
			RegisterGroups: map[string]config.RegisterGroup{
				"test": {
					Name: "Test Group",
				},
			},
		},
	}

	// Test getter methods
	if device.GetName() != "Test Device" {
		t.Errorf("GetName() = %s, want 'Test Device'", device.GetName())
	}

	if device.GetSlaveID() != 42 {
		t.Errorf("GetSlaveID() = %d, want 42", device.GetSlaveID())
	}

	if device.GetPollInterval() != 3000 {
		t.Errorf("GetPollInterval() = %d, want 3000", device.GetPollInterval())
	}

	if !device.IsEnabled() {
		t.Error("IsEnabled() = false, want true")
	}

	if device.GetHADeviceName() != "Test Device" {
		t.Errorf("GetHADeviceName() = %s, want 'Test Device'", device.GetHADeviceName())
	}

	if device.GetHAManufacturer() != "TestCorp Industries" {
		t.Errorf("GetHAManufacturer() = %s, want 'TestCorp Industries'", device.GetHAManufacturer())
	}

	if device.GetHAModel() != "TD-100 Pro" {
		t.Errorf("GetHAModel() = %s, want 'TD-100 Pro'", device.GetHAModel())
	}

	if device.GetHADeviceID("my_device") != "test_device_001" {
		t.Errorf("GetHADeviceID() = %s, want 'test_device_001'", device.GetHADeviceID("my_device"))
	}
}

func TestDevice_HAFallbacks(t *testing.T) {
	// Test with nil HomeAssistant config
	device := config.Device{
		Metadata: config.DeviceMetadata{
			Name:         "Test Device",
			Manufacturer: "TestCorp",
			Model:        "TD-100",
			Enabled:      true,
		},
		RTU: config.RTUConfig{
			SlaveID: 42,
		},
		HomeAssistant: nil, // No HA config
		Modbus: config.ModbusDeviceConfig{
			RegisterGroups: map[string]config.RegisterGroup{},
		},
	}

	// Should fallback to metadata
	if device.GetHAManufacturer() != "TestCorp" {
		t.Errorf("GetHAManufacturer() = %s, want 'TestCorp' (from metadata)", device.GetHAManufacturer())
	}

	if device.GetHAModel() != "TD-100" {
		t.Errorf("GetHAModel() = %s, want 'TD-100' (from metadata)", device.GetHAModel())
	}

	// Test with empty HA overrides
	device.HomeAssistant = &config.HADeviceConfig{
		DeviceID: "test_001",
		// Manufacturer and Model empty - should fallback to metadata
	}

	if device.GetHAManufacturer() != "TestCorp" {
		t.Errorf("GetHAManufacturer() = %s, want 'TestCorp' (fallback)", device.GetHAManufacturer())
	}

	if device.GetHAModel() != "TD-100" {
		t.Errorf("GetHAModel() = %s, want 'TD-100' (fallback)", device.GetHAModel())
	}

	// Test device_id fallback to deviceKey
	device.HomeAssistant = nil // No HA config
	if device.GetHADeviceID("energy_meter_1") != "energy_meter_1" {
		t.Errorf("GetHADeviceID() = %s, want 'energy_meter_1' (fallback to deviceKey)", device.GetHADeviceID("energy_meter_1"))
	}

	// Test device_id from config takes precedence
	device.HomeAssistant = &config.HADeviceConfig{
		DeviceID: "custom_id_123",
	}
	if device.GetHADeviceID("energy_meter_1") != "custom_id_123" {
		t.Errorf("GetHADeviceID() = %s, want 'custom_id_123' (from config)", device.GetHADeviceID("energy_meter_1"))
	}
}

func TestDevice_Validation(t *testing.T) {
	tests := []struct {
		name        string
		device      config.Device
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid device",
			device: config.Device{
				Metadata: config.DeviceMetadata{
					Name:    "Valid Device",
					Enabled: true,
				},
				RTU: config.RTUConfig{
					SlaveID: 11,
				},
				Modbus: config.ModbusDeviceConfig{
					RegisterGroups: map[string]config.RegisterGroup{
						"test": {
							Name:          "Test Group",
							FunctionCode:  0x03,
							StartAddress:  0x2000,
							RegisterCount: 10,
							Enabled:       true,
							Registers: []config.GroupRegister{
								{Key: "voltage", Name: "Voltage", Offset: 0, Unit: "V"},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Missing metadata name",
			device: config.Device{
				Metadata: config.DeviceMetadata{
					Name:    "", // Empty name
					Enabled: true,
				},
				RTU: config.RTUConfig{
					SlaveID: 11,
				},
				Modbus: config.ModbusDeviceConfig{
					RegisterGroups: map[string]config.RegisterGroup{
						"test": {Name: "Test"},
					},
				},
			},
			expectError: true,
			errorMsg:    "metadata.name is required",
		},
		{
			name: "Invalid slave_id (0)",
			device: config.Device{
				Metadata: config.DeviceMetadata{
					Name:    "Test Device",
					Enabled: true,
				},
				RTU: config.RTUConfig{
					SlaveID: 0, // Invalid
				},
				Modbus: config.ModbusDeviceConfig{
					RegisterGroups: map[string]config.RegisterGroup{
						"test": {Name: "Test"},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid rtu.slave_id",
		},
		{
			name: "Invalid slave_id (>247)",
			device: config.Device{
				Metadata: config.DeviceMetadata{
					Name:    "Test Device",
					Enabled: true,
				},
				RTU: config.RTUConfig{
					SlaveID: 250, // Too high
				},
				Modbus: config.ModbusDeviceConfig{
					RegisterGroups: map[string]config.RegisterGroup{
						"test": {Name: "Test"},
					},
				},
			},
			expectError: true,
			errorMsg:    "rtu.slave_id",
		},
		{
			name: "No register groups",
			device: config.Device{
				Metadata: config.DeviceMetadata{
					Name:    "Test Device",
					Enabled: true,
				},
				RTU: config.RTUConfig{
					SlaveID: 11,
				},
				Modbus: config.ModbusDeviceConfig{
					RegisterGroups: map[string]config.RegisterGroup{}, // Empty
				},
			},
			expectError: true,
			errorMsg:    "no modbus.register_groups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.device.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidateDevices_DuplicateSlaveID(t *testing.T) {
	devices := map[string]config.Device{
		"meter_1": {
			Metadata: config.DeviceMetadata{
				Name:    "Meter 1",
				Enabled: true,
			},
			RTU: config.RTUConfig{
				SlaveID: 11,
			},
			Modbus: config.ModbusDeviceConfig{
				RegisterGroups: map[string]config.RegisterGroup{
					"test": {
						Name:          "Test",
						FunctionCode:  0x03,
						StartAddress:  0x2000,
						RegisterCount: 10,
						Enabled:       true,
						Registers: []config.GroupRegister{
							{Key: "test", Name: "Test", Offset: 0, Unit: "V"},
						},
					},
				},
			},
		},
		"meter_2": {
			Metadata: config.DeviceMetadata{
				Name:    "Meter 2",
				Enabled: true,
			},
			RTU: config.RTUConfig{
				SlaveID: 11, // Duplicate!
			},
			Modbus: config.ModbusDeviceConfig{
				RegisterGroups: map[string]config.RegisterGroup{
					"test": {
						Name:          "Test",
						FunctionCode:  0x03,
						StartAddress:  0x2000,
						RegisterCount: 10,
						Enabled:       true,
						Registers: []config.GroupRegister{
							{Key: "test", Name: "Test", Offset: 0, Unit: "V"},
						},
					},
				},
			},
		},
	}

	err := config.ValidateDevices(devices)
	if err == nil {
		t.Error("Expected error for duplicate slave_id, got nil")
	}
	if !containsSubstring(err.Error(), "duplicate") && !containsSubstring(err.Error(), "slave_id") {
		t.Errorf("Expected duplicate slave_id error, got: %v", err)
	}
}

func TestValidateDevices_UniqueDeviceKeys(t *testing.T) {
	// Test that device keys in the map are unique (Go maps guarantee this)
	// and that we properly log them
	devices := map[string]config.Device{
		"energy_meter_1": {
			Metadata: config.DeviceMetadata{
				Name:    "Energy Meter 1",
				Enabled: true,
			},
			RTU: config.RTUConfig{
				SlaveID: 11,
			},
			Modbus: config.ModbusDeviceConfig{
				RegisterGroups: map[string]config.RegisterGroup{
					"test": {
						Name:          "Test",
						FunctionCode:  0x03,
						StartAddress:  0x2000,
						RegisterCount: 10,
						Enabled:       true,
						Registers: []config.GroupRegister{
							{Key: "test", Name: "Test", Offset: 0, Unit: "V"},
						},
					},
				},
			},
		},
		"energy_meter_2": {
			Metadata: config.DeviceMetadata{
				Name:    "Energy Meter 2",
				Enabled: true,
			},
			RTU: config.RTUConfig{
				SlaveID: 12,
			},
			Modbus: config.ModbusDeviceConfig{
				RegisterGroups: map[string]config.RegisterGroup{
					"test": {
						Name:          "Test",
						FunctionCode:  0x03,
						StartAddress:  0x2000,
						RegisterCount: 10,
						Enabled:       true,
						Registers: []config.GroupRegister{
							{Key: "test", Name: "Test", Offset: 0, Unit: "V"},
						},
					},
				},
			},
		},
		"inverter_1": {
			Metadata: config.DeviceMetadata{
				Name:    "Solar Inverter",
				Enabled: true,
			},
			RTU: config.RTUConfig{
				SlaveID: 20,
			},
			Modbus: config.ModbusDeviceConfig{
				RegisterGroups: map[string]config.RegisterGroup{
					"test": {
						Name:          "Test",
						FunctionCode:  0x03,
						StartAddress:  0x2000,
						RegisterCount: 10,
						Enabled:       true,
						Registers: []config.GroupRegister{
							{Key: "test", Name: "Test", Offset: 0, Unit: "V"},
						},
					},
				},
			},
		},
	}

	err := config.ValidateDevices(devices)
	if err != nil {
		t.Errorf("Expected no error for valid unique devices, got: %v", err)
	}

	// Verify we have exactly 3 devices with unique keys
	if len(devices) != 3 {
		t.Errorf("Expected 3 devices, got: %d", len(devices))
	}

	// Verify specific keys exist
	expectedKeys := []string{"energy_meter_1", "energy_meter_2", "inverter_1"}
	for _, key := range expectedKeys {
		if _, exists := devices[key]; !exists {
			t.Errorf("Expected device key '%s' to exist", key)
		}
	}
}

func TestDeviceID_Fallback(t *testing.T) {
	tests := []struct {
		name           string
		device         config.Device
		deviceKey      string
		expectedID     string
		expectFallback bool
	}{
		{
			name: "Explicit device_id",
			device: config.Device{
				HomeAssistant: &config.HADeviceConfig{
					DeviceID: "custom_device_123",
				},
			},
			deviceKey:      "energy_meter_1",
			expectedID:     "custom_device_123",
			expectFallback: false,
		},
		{
			name: "Fallback to device key (nil HA config)",
			device: config.Device{
				HomeAssistant: nil,
			},
			deviceKey:      "energy_meter_2",
			expectedID:     "energy_meter_2",
			expectFallback: true,
		},
		{
			name: "Fallback to device key (empty device_id)",
			device: config.Device{
				HomeAssistant: &config.HADeviceConfig{
					DeviceID:     "",
					Manufacturer: "TestCorp",
				},
			},
			deviceKey:      "inverter_1",
			expectedID:     "inverter_1",
			expectFallback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualID := tt.device.GetHADeviceID(tt.deviceKey)
			if actualID != tt.expectedID {
				t.Errorf("GetHADeviceID(%s) = %s, want %s", tt.deviceKey, actualID, tt.expectedID)
			}

			if tt.expectFallback && actualID != tt.deviceKey {
				t.Errorf("Expected fallback to device key %s, got %s", tt.deviceKey, actualID)
			}
		})
	}
}

func TestValidateDevices_DuplicateHADeviceID(t *testing.T) {
	tests := []struct {
		name        string
		devices     map[string]config.Device
		expectError bool
		errorMsg    string
	}{
		{
			name: "Duplicate explicit device_id",
			devices: map[string]config.Device{
				"meter_1": {
					Metadata: config.DeviceMetadata{
						Name:    "Meter 1",
						Enabled: true,
					},
					RTU: config.RTUConfig{
						SlaveID: 11,
					},
					HomeAssistant: &config.HADeviceConfig{
						DeviceID: "same_id_123",
					},
					Modbus: config.ModbusDeviceConfig{
						RegisterGroups: map[string]config.RegisterGroup{
							"test": {
								Name:          "Test",
								FunctionCode:  0x03,
								StartAddress:  0x2000,
								RegisterCount: 10,
								Enabled:       true,
								Registers: []config.GroupRegister{
									{Key: "test", Name: "Test", Offset: 0, Unit: "V"},
								},
							},
						},
					},
				},
				"meter_2": {
					Metadata: config.DeviceMetadata{
						Name:    "Meter 2",
						Enabled: true,
					},
					RTU: config.RTUConfig{
						SlaveID: 12,
					},
					HomeAssistant: &config.HADeviceConfig{
						DeviceID: "same_id_123", // Duplicate!
					},
					Modbus: config.ModbusDeviceConfig{
						RegisterGroups: map[string]config.RegisterGroup{
							"test": {
								Name:          "Test",
								FunctionCode:  0x03,
								StartAddress:  0x2000,
								RegisterCount: 10,
								Enabled:       true,
								Registers: []config.GroupRegister{
									{Key: "test", Name: "Test", Offset: 0, Unit: "V"},
								},
							},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "duplicate Home Assistant device_id",
		},
		{
			name: "Device key conflicts with explicit device_id",
			devices: map[string]config.Device{
				"meter_1": {
					Metadata: config.DeviceMetadata{
						Name:    "Meter 1",
						Enabled: true,
					},
					RTU: config.RTUConfig{
						SlaveID: 11,
					},
					// No HomeAssistant config - will use "meter_1" as device_id
					Modbus: config.ModbusDeviceConfig{
						RegisterGroups: map[string]config.RegisterGroup{
							"test": {
								Name:          "Test",
								FunctionCode:  0x03,
								StartAddress:  0x2000,
								RegisterCount: 10,
								Enabled:       true,
								Registers: []config.GroupRegister{
									{Key: "test", Name: "Test", Offset: 0, Unit: "V"},
								},
							},
						},
					},
				},
				"meter_2": {
					Metadata: config.DeviceMetadata{
						Name:    "Meter 2",
						Enabled: true,
					},
					RTU: config.RTUConfig{
						SlaveID: 12,
					},
					HomeAssistant: &config.HADeviceConfig{
						DeviceID: "meter_1", // Conflicts with meter_1's device key!
					},
					Modbus: config.ModbusDeviceConfig{
						RegisterGroups: map[string]config.RegisterGroup{
							"test": {
								Name:          "Test",
								FunctionCode:  0x03,
								StartAddress:  0x2000,
								RegisterCount: 10,
								Enabled:       true,
								Registers: []config.GroupRegister{
									{Key: "test", Name: "Test", Offset: 0, Unit: "V"},
								},
							},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "duplicate Home Assistant device_id",
		},
		{
			name: "All unique device IDs",
			devices: map[string]config.Device{
				"meter_1": {
					Metadata: config.DeviceMetadata{
						Name:    "Meter 1",
						Enabled: true,
					},
					RTU: config.RTUConfig{
						SlaveID: 11,
					},
					HomeAssistant: &config.HADeviceConfig{
						DeviceID: "chint_meter_001",
					},
					Modbus: config.ModbusDeviceConfig{
						RegisterGroups: map[string]config.RegisterGroup{
							"test": {
								Name:          "Test",
								FunctionCode:  0x03,
								StartAddress:  0x2000,
								RegisterCount: 10,
								Enabled:       true,
								Registers: []config.GroupRegister{
									{Key: "test", Name: "Test", Offset: 0, Unit: "V"},
								},
							},
						},
					},
				},
				"meter_2": {
					Metadata: config.DeviceMetadata{
						Name:    "Meter 2",
						Enabled: true,
					},
					RTU: config.RTUConfig{
						SlaveID: 12,
					},
					// No HomeAssistant config - will use "meter_2"
					Modbus: config.ModbusDeviceConfig{
						RegisterGroups: map[string]config.RegisterGroup{
							"test": {
								Name:          "Test",
								FunctionCode:  0x03,
								StartAddress:  0x2000,
								RegisterCount: 10,
								Enabled:       true,
								Registers: []config.GroupRegister{
									{Key: "test", Name: "Test", Offset: 0, Unit: "V"},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateDevices(tt.devices)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function to create a basic valid device for testing calculated values
func createTestDevice(registers []config.GroupRegister, calculatedValues []config.CalculatedValue) config.Device {
	return config.Device{
		Metadata: config.DeviceMetadata{
			Name:    "Test Device",
			Enabled: true,
		},
		RTU: config.RTUConfig{
			SlaveID: 1,
		},
		Modbus: config.ModbusDeviceConfig{
			RegisterGroups: map[string]config.RegisterGroup{
				"instant": {
					Name:          "Instant",
					FunctionCode:  0x03,
					StartAddress:  0x2000,
					RegisterCount: uint16(len(registers) * 2), // 2 registers per value (float32)
					Registers:     registers,
				},
			},
		},
		CalculatedValues: calculatedValues,
	}
}

func TestDeviceValidate_CalculatedValues(t *testing.T) {
	tests := []struct {
		name          string
		device        config.Device
		wantError     bool
		errorContains string
	}{
		{
			name: "Valid calculated value",
			device: createTestDevice(
				[]config.GroupRegister{
					{Key: "power_active", Offset: 0},
					{Key: "power_reactive", Offset: 4},
				},
				[]config.CalculatedValue{
					{
						Key:     "power_apparent",
						Formula: "sqrt(power_active^2 + power_reactive^2)",
					},
				},
			),
			wantError: false,
		},
		{
			name: "Calculated value with missing variable",
			device: createTestDevice(
				[]config.GroupRegister{
					{Key: "power_active", Offset: 0},
				},
				[]config.CalculatedValue{
					{
						Key:     "power_apparent",
						Formula: "sqrt(power_active^2 + power_reactive^2)",
					},
				},
			),
			wantError:     true,
			errorContains: "references unknown register 'power_reactive'",
		},
		{
			name: "Calculated value with duplicate key",
			device: createTestDevice(
				[]config.GroupRegister{
					{Key: "power_active", Offset: 0},
					{Key: "power_apparent", Offset: 4},
				},
				[]config.CalculatedValue{
					{
						Key:     "power_apparent", // Duplicate!
						Formula: "sqrt(power_active^2 + power_reactive^2)",
					},
				},
			),
			wantError:     true,
			errorContains: "conflicts with register",
		},
		{
			name: "Calculated value with empty formula",
			device: createTestDevice(
				[]config.GroupRegister{
					{Key: "power_active", Offset: 0},
				},
				[]config.CalculatedValue{
					{
						Key:     "power_apparent",
						Formula: "",
					},
				},
			),
			wantError:     true,
			errorContains: "has no formula",
		},
		{
			name: "Calculated value with invalid formula syntax",
			device: createTestDevice(
				[]config.GroupRegister{
					{Key: "power_active", Offset: 0},
				},
				[]config.CalculatedValue{
					{
						Key:     "power_apparent",
						Formula: "sqrt(power_active",
					},
				},
			),
			wantError:     true,
			errorContains: "invalid formula",
		},
		{
			name: "Multiple calculated values - all valid",
			device: createTestDevice(
				[]config.GroupRegister{
					{Key: "voltage", Offset: 0},
					{Key: "current", Offset: 4},
					{Key: "power_active", Offset: 8},
				},
				[]config.CalculatedValue{
					{
						Key:     "power_apparent",
						Formula: "voltage * current",
					},
					{
						Key:     "power_factor",
						Formula: "power_active / power_apparent",
					},
				},
			),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.device.Validate()

			if tt.wantError {
				if err == nil {
					t.Errorf("Device.Validate() expected error but got none")
					return
				}
				if tt.errorContains != "" && !containsSubstring(err.Error(), tt.errorContains) {
					t.Errorf("Device.Validate() error = %v, want error containing %q", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Device.Validate() unexpected error = %v", err)
			}
		})
	}
}
