package unit

import (
	"testing"

	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/modbus"
)

// TestVoltageCommandParsing tests voltage command data parsing
func TestVoltageCommandParsing(t *testing.T) {
	// Mock register configuration
	register := config.Register{
		Name:    "voltage",
		Address: 0x2000,
		Unit:    "V",
	}

	// Create voltage command
	cmd := modbus.NewVoltageCommand(register, 1)

	// Mock Modbus response: 220.25V (float32)
	// IEEE 754: 0x435C4000 = 220.25
	mockData := []byte{0x43, 0x5C, 0x40, 0x00}

	// Parse data
	value, err := cmd.ParseData(mockData)
	if err != nil {
		t.Fatalf("Failed to parse voltage data: %v", err)
	}

	// Value should be ~220.25V
	expectedValue := 220.25
	tolerance := 0.1
	if value < expectedValue-tolerance || value > expectedValue+tolerance {
		t.Errorf("Expected value ~%.2f, got %.2f", expectedValue, value)
	}
}

// TestCurrentCommandParsing tests current command data parsing
func TestCurrentCommandParsing(t *testing.T) {
	register := config.Register{
		Name:    "current",
		Address: 0x2002,
		Unit:    "A",
	}

	cmd := modbus.NewCurrentCommand(register, 1)

	// Mock response: 15.75A
	// IEEE 754: 0x417C0000 = 15.75
	mockData := []byte{0x41, 0x7C, 0x00, 0x00}

	value, err := cmd.ParseData(mockData)
	if err != nil {
		t.Fatalf("Failed to parse current data: %v", err)
	}

	expectedValue := 15.75
	tolerance := 0.01
	if value < expectedValue-tolerance || value > expectedValue+tolerance {
		t.Errorf("Expected value ~%.2f, got %.2f", expectedValue, value)
	}
}

// TestPowerCommandParsing tests power command data parsing
func TestPowerCommandParsing(t *testing.T) {
	register := config.Register{
		Name:    "active_power",
		Address: 0x2004,
		Unit:    "W",
	}

	cmd := modbus.NewPowerCommand(register, 1)

	// Mock response: 3.4645 kW (float32)
	// IEEE 754: 0x45586800 = 3464.5 -> converted to W: 3464500
	mockData := []byte{0x45, 0x58, 0x68, 0x00}

	value, err := cmd.ParseData(mockData)
	if err != nil {
		t.Fatalf("Failed to parse power data: %v", err)
	}

	// Power is converted from kW to W (multiplied by 1000)
	expectedValue := 3464500.0
	tolerance := 5000.0 // Allow 5kW tolerance for float32 precision
	if value < expectedValue-tolerance || value > expectedValue+tolerance {
		t.Errorf("Expected value ~%.0f, got %.0f", expectedValue, value)
	}
}

// TestFrequencyCommandParsing tests frequency command data parsing
func TestFrequencyCommandParsing(t *testing.T) {
	register := config.Register{
		Name:    "frequency",
		Address: 0x200E,
		Unit:    "Hz",
	}

	cmd := modbus.NewFrequencyCommand(register, 1)

	// Mock response: 50.01Hz
	// IEEE 754: 0x42480A3D = 50.01
	mockData := []byte{0x42, 0x48, 0x0A, 0x3D}

	value, err := cmd.ParseData(mockData)
	if err != nil {
		t.Fatalf("Failed to parse frequency data: %v", err)
	}

	expectedValue := 50.01
	tolerance := 0.05
	if value < expectedValue-tolerance || value > expectedValue+tolerance {
		t.Errorf("Expected value ~%.2f, got %.2f", expectedValue, value)
	}
}

// TestPowerFactorCommandParsing tests power factor command data parsing
func TestPowerFactorCommandParsing(t *testing.T) {
	register := config.Register{
		Name:    "power_factor",
		Address: 0x200A,
		Unit:    "",
	}

	cmd := modbus.NewPowerFactorCommand(register, 1)

	// Mock response: 0.92
	// IEEE 754: 0x3F6B851F = 0.92
	mockData := []byte{0x3F, 0x6B, 0x85, 0x1F}

	value, err := cmd.ParseData(mockData)
	if err != nil {
		t.Fatalf("Failed to parse power factor data: %v", err)
	}

	expectedValue := 0.92
	tolerance := 0.01
	if value < expectedValue-tolerance || value > expectedValue+tolerance {
		t.Errorf("Expected value ~%.2f, got %.2f", expectedValue, value)
	}
}

// TestEnergyCommandParsing tests energy command data parsing
func TestEnergyCommandParsing(t *testing.T) {
	register := config.Register{
		Name:    "total_energy",
		Address: 0x4000,
		Unit:    "kWh",
	}

	cmd := modbus.NewEnergyCommand(register, 1)

	// Mock response: 1234.56 kWh (float32)
	// IEEE 754: 0x449A51EC = 1234.56
	// Convert 1234.56 to IEEE 754: sign=0, exp=137 (10001001), mantissa=0x1A51EC
	mockData := []byte{0x44, 0x9A, 0x51, 0xEC}

	value, err := cmd.ParseData(mockData)
	if err != nil {
		t.Fatalf("Failed to parse energy data: %v", err)
	}

	expectedValue := 1234.56
	tolerance := 0.01
	if value < expectedValue-tolerance || value > expectedValue+tolerance {
		t.Errorf("Expected value ~%.2f, got %.2f", expectedValue, value)
	}
}

// TestInvalidDataLength tests error handling for invalid data length
func TestInvalidDataLength(t *testing.T) {
	register := config.Register{
		Name:    "voltage",
		Address: 0x2000,
		Unit:    "V",
	}

	cmd := modbus.NewVoltageCommand(register, 1)

	// Invalid data: only 2 bytes instead of 4
	mockData := []byte{0x43, 0x5C}

	_, err := cmd.ParseData(mockData)
	if err == nil {
		t.Error("Expected error for invalid data length, got nil")
	}
}

// TestCommandFactory tests the factory pattern for creating commands
func TestCommandFactory(t *testing.T) {
	testCases := []struct {
		name     string
		register config.Register
	}{
		{
			name:     "voltage",
			register: config.Register{Name: "voltage", Address: 0x2000},
		},
		{
			name:     "current",
			register: config.Register{Name: "current", Address: 0x2002},
		},
		{
			name:     "active_power",
			register: config.Register{Name: "active_power", Address: 0x2004},
		},
		{
			name:     "frequency",
			register: config.Register{Name: "frequency", Address: 0x200E},
		},
		{
			name:     "power_factor",
			register: config.Register{Name: "power_factor", Address: 0x200A},
		},
		{
			name:     "reactive_power",
			register: config.Register{Name: "reactive_power", Address: 0x2006},
		},
		{
			name:     "total_energy",
			register: config.Register{Name: "total_energy", Address: 0x4000},
		},
	}

	slaveID := uint8(1)
	factory := modbus.NewCommandFactory(slaveID)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := factory.CreateCommand(tc.register)
			if err != nil {
				t.Fatalf("Factory returned error for register %s: %v", tc.register.Name, err)
			}
			if cmd == nil {
				t.Fatalf("Factory returned nil for register %s", tc.register.Name)
			}
		})
	}
}
