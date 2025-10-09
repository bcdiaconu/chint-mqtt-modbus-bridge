package unit

import (
	"testing"

	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/modbus"
)

// TestCommandFactoryCreation tests factory instantiation
func TestCommandFactoryCreation(t *testing.T) {
	slaveID := uint8(1)
	factory := modbus.NewCommandFactory(slaveID)

	if factory == nil {
		t.Fatal("Factory creation returned nil")
	}
}

// TestFactoryCreateAllCommandTypes tests creating all supported command types
func TestFactoryCreateAllCommandTypes(t *testing.T) {
	testCases := []struct {
		name         string
		registerName string
		address      uint16
		expectError  bool
	}{
		{
			name:         "voltage_command",
			registerName: "voltage",
			address:      0x2000,
			expectError:  false,
		},
		{
			name:         "current_command",
			registerName: "current",
			address:      0x2002,
			expectError:  false,
		},
		{
			name:         "active_power_command",
			registerName: "active_power",
			address:      0x2004,
			expectError:  false,
		},
		{
			name:         "reactive_power_command",
			registerName: "reactive_power",
			address:      0x2006,
			expectError:  false,
		},
		{
			name:         "power_factor_command",
			registerName: "power_factor",
			address:      0x200A,
			expectError:  false,
		},
		{
			name:         "frequency_command",
			registerName: "frequency",
			address:      0x200E,
			expectError:  false,
		},
		{
			name:         "total_energy_command",
			registerName: "total_energy",
			address:      0x4000,
			expectError:  false,
		},
		{
			name:         "import_energy_command",
			registerName: "import_energy",
			address:      0x4004,
			expectError:  false,
		},
		{
			name:         "export_energy_command",
			registerName: "export_energy",
			address:      0x400A,
			expectError:  false,
		},
		{
			name:         "unknown_register_falls_back_to_base",
			registerName: "unknown_type",
			address:      0x9999,
			expectError:  false, // Factory returns BaseCommand for unknown types
		},
	}

	factory := modbus.NewCommandFactory(1)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			register := config.Register{
				Name:        tc.registerName,
				Address:     tc.address,
				Unit:        "test",
				DeviceClass: tc.registerName, // Use register name as device_class for testing
			}

			cmd, err := factory.CreateCommand(register)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for register %s, got nil", tc.registerName)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for register %s: %v", tc.registerName, err)
				}
				if cmd == nil {
					t.Errorf("Command is nil for register %s", tc.registerName)
				}
			}
		})
	}
}

// TestFactoryCommandUniqueness tests that factory creates different instances
func TestFactoryCommandUniqueness(t *testing.T) {
	factory := modbus.NewCommandFactory(1)
	register := config.Register{
		Name:    "voltage",
		Address: 0x2000,
		Unit:    "V",
	}

	cmd1, err1 := factory.CreateCommand(register)
	cmd2, err2 := factory.CreateCommand(register)

	if err1 != nil || err2 != nil {
		t.Fatal("Error creating commands")
	}

	// Commands should be different instances
	if cmd1 == cmd2 {
		t.Error("Factory returned same instance instead of creating new ones")
	}
}

// TestBaseCommandProperties tests base command functionality
func TestBaseCommandProperties(t *testing.T) {
	register := config.Register{
		Name:    "test_voltage",
		Address: 0x2000,
		Unit:    "V",
	}

	cmd := modbus.NewVoltageCommand(register, 1)

	// Test that command stores register info
	// This is implicit - command should use register for parsing
	testData := []byte{0x43, 0x5C, 0x40, 0x00} // 220.25V

	value, err := cmd.ParseData(testData)
	if err != nil {
		t.Fatalf("ParseData failed: %v", err)
	}

	if value < 220.0 || value > 221.0 {
		t.Errorf("Expected value around 220.25, got %.2f", value)
	}
}

// TestCommandDataValidation tests data validation in commands
func TestCommandDataValidation(t *testing.T) {
	testCases := []struct {
		name        string
		cmdType     string
		data        []byte
		expectError bool
	}{
		{
			name:        "valid_voltage_data",
			cmdType:     "voltage",
			data:        []byte{0x43, 0x5C, 0x40, 0x00},
			expectError: false,
		},
		{
			name:        "insufficient_voltage_data",
			cmdType:     "voltage",
			data:        []byte{0x43, 0x5C},
			expectError: true,
		},
		{
			name:        "empty_data",
			cmdType:     "voltage",
			data:        []byte{},
			expectError: true,
		},
		{
			name:        "valid_current_data",
			cmdType:     "current",
			data:        []byte{0x41, 0x7C, 0x00, 0x00},
			expectError: false,
		},
		{
			name:        "insufficient_current_data",
			cmdType:     "current",
			data:        []byte{0x41},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			register := config.Register{
				Name:    tc.cmdType,
				Address: 0x2000,
				Unit:    "test",
			}

			var cmd modbus.ModbusCommand
			var err error

			switch tc.cmdType {
			case "voltage":
				cmd = modbus.NewVoltageCommand(register, 1)
			case "current":
				cmd = modbus.NewCurrentCommand(register, 1)
			default:
				t.Fatalf("Unknown command type: %s", tc.cmdType)
			}

			_, err = cmd.ParseData(tc.data)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestPowerConversion tests power conversion from kW to W
func TestPowerConversion(t *testing.T) {
	register := config.Register{
		Name:    "active_power",
		Address: 0x2004,
		Unit:    "W",
	}

	cmd := modbus.NewPowerCommand(register, 1)

	// Test data: 0x40, 0x5D, 0x47, 0xAE represents ~3.4575 kW as float32
	testData := []byte{0x40, 0x5D, 0x47, 0xAE}

	value, err := cmd.ParseData(testData)
	if err != nil {
		t.Fatalf("ParseData failed: %v", err)
	}

	// Value should be converted to W (multiplied by 1000)
	// Actual value is ~3457.5 W
	expectedMin := 3457.0
	expectedMax := 3458.0

	if value < expectedMin || value > expectedMax {
		t.Errorf("Expected value between %.0f and %.0f W, got %.2f", expectedMin, expectedMax, value)
	}
}

// TestReactivePowerCalculation tests reactive power calculation logic
func TestReactivePowerCalculation(t *testing.T) {
	// Note: Reactive power doesn't use ParseData - it calculates from active and apparent power
	// This test verifies the command can be created with proper configuration
	register := config.Register{
		Name:        "reactive_power",
		Address:     0x2006,
		Unit:        "VAr",
		DeviceClass: "reactive_power",
	}

	cmd := modbus.NewReactivePowerCommand(register, 1, "active_power", "apparent_power")

	if cmd == nil {
		t.Fatal("Failed to create reactive power command")
	}

	// Reactive power command is created successfully
	// The actual calculation is tested in integration tests
	// since it requires executor and other commands
}
