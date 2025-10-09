package tests

import (
	"testing"

	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/modbus"
	"mqtt-modbus-bridge/pkg/modbus/groups"
)

// TestInstantGroupExecution tests the instant values group strategy
func TestInstantGroupExecution(t *testing.T) {
	// Create mock registers for instant values
	registers := []config.Register{
		{Name: "voltage", Address: 0x2000, Count: 2, DataType: "float32", Unit: "V", Precision: 1},
		{Name: "current", Address: 0x2002, Count: 2, DataType: "float32", Unit: "A", Precision: 2},
		{Name: "active_power", Address: 0x2004, Count: 2, DataType: "float32", Unit: "W", Precision: 0},
		{Name: "frequency", Address: 0x200E, Count: 2, DataType: "float32", Unit: "Hz", Precision: 2},
	}

	// Create command registry
	factory := modbus.NewCommandFactory()
	registry := make(map[string]modbus.ModbusCommand)
	for _, reg := range registers {
		cmd := factory.CreateCommand(reg)
		if cmd != nil {
			registry[reg.Name] = cmd
		}
	}

	// Create instant group
	instantGroup := groups.NewInstantGroup(registers, registry)

	// Execute group to get request
	request := instantGroup.Execute()

	// Verify request covers all registers
	// Voltage at 0x2000, Frequency at 0x200E = range of 16 registers (0x10)
	expectedStart := uint16(0x2000)
	expectedCount := uint16(16) // 0x200E - 0x2000 + 2 = 14 + 2 = 16

	if request.StartAddr != expectedStart {
		t.Errorf("Expected start address 0x%X, got 0x%X", expectedStart, request.StartAddr)
	}

	if request.RegCount != expectedCount {
		t.Errorf("Expected register count %d, got %d", expectedCount, request.RegCount)
	}

	// Mock response data: 32 bytes (16 registers × 2 bytes)
	// Voltage: 220.5V (0x435C4000)
	// Current: 15.75A (0x417C0000)
	// Power: 3464.5W (0x45586800)
	// Unused registers: zeros
	// Frequency: 50.01Hz (0x42480A3D)
	mockData := make([]byte, 32)
	copy(mockData[0:4], []byte{0x43, 0x5C, 0x40, 0x00})   // voltage at offset 0
	copy(mockData[4:8], []byte{0x41, 0x7C, 0x00, 0x00})   // current at offset 4
	copy(mockData[8:12], []byte{0x45, 0x58, 0x68, 0x00})  // power at offset 8
	copy(mockData[28:32], []byte{0x42, 0x48, 0x0A, 0x3D}) // frequency at offset 28

	// Parse results
	results, err := instantGroup.ParseResults(mockData)
	if err != nil {
		t.Fatalf("Failed to parse group results: %v", err)
	}

	// Verify we got all 4 values
	if len(results) != 4 {
		t.Fatalf("Expected 4 results, got %d", len(results))
	}

	// Verify each value
	expectedValues := map[string]float64{
		"voltage":      220.25,
		"current":      15.75,
		"active_power": 3464.5,
		"frequency":    50.01,
	}

	for name, expectedValue := range expectedValues {
		found := false
		for _, result := range results {
			if result.Name == name {
				found = true
				if value, ok := result.Value.(float64); ok {
					tolerance := 1.0
					if value < expectedValue-tolerance || value > expectedValue+tolerance {
						t.Errorf("For %s: expected ~%.2f, got %.2f", name, expectedValue, value)
					}
				} else {
					t.Errorf("For %s: expected float64 value, got %T", name, result.Value)
				}
				break
			}
		}
		if !found {
			t.Errorf("Result for %s not found in parsed data", name)
		}
	}

	// Verify GetNames returns correct names
	names := instantGroup.GetNames()
	if len(names) != 4 {
		t.Errorf("Expected 4 names, got %d", len(names))
	}
}

// TestEnergyGroupExecution tests the energy values group strategy
func TestEnergyGroupExecution(t *testing.T) {
	registers := []config.Register{
		{Name: "total_energy", Address: 0x4000, Count: 4, DataType: "uint64", Unit: "kWh", Precision: 2, Scale: 0.01},
		{Name: "import_energy", Address: 0x4004, Count: 4, DataType: "uint64", Unit: "kWh", Precision: 2, Scale: 0.01},
		{Name: "export_energy", Address: 0x400A, Count: 4, DataType: "uint64", Unit: "kWh", Precision: 2, Scale: 0.01},
	}

	factory := modbus.NewCommandFactory()
	registry := make(map[string]modbus.ModbusCommand)
	for _, reg := range registers {
		cmd := factory.CreateCommand(reg)
		if cmd != nil {
			registry[reg.Name] = cmd
		}
	}

	energyGroup := groups.NewEnergyGroup(registers, registry)

	request := energyGroup.Execute()

	// Energy group: 0x4000 to 0x400A + 4 = 0x400E
	// Range: 14 registers (0x400A - 0x4000 + 4 = 10 + 4 = 14)
	expectedStart := uint16(0x4000)
	expectedCount := uint16(14)

	if request.StartAddr != expectedStart {
		t.Errorf("Expected start address 0x%X, got 0x%X", expectedStart, request.StartAddr)
	}

	if request.RegCount != expectedCount {
		t.Errorf("Expected register count %d, got %d", expectedCount, request.RegCount)
	}

	// Mock response: 28 bytes (14 registers × 2 bytes)
	// Total: 123456 (0x1E240) = 1234.56 kWh
	// Import: 100000 (0x186A0) = 1000.00 kWh
	// Export: 23456 (0x5BA0) = 234.56 kWh
	mockData := make([]byte, 28)
	copy(mockData[0:8], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0xE2, 0x40})   // total at offset 0
	copy(mockData[8:16], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x86, 0xA0})  // import at offset 8
	copy(mockData[20:28], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x5B, 0xA0}) // export at offset 20

	results, err := energyGroup.ParseResults(mockData)
	if err != nil {
		t.Fatalf("Failed to parse energy group results: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	expectedValues := map[string]float64{
		"total_energy":  1234.56,
		"import_energy": 1000.00,
		"export_energy": 234.56,
	}

	for name, expectedValue := range expectedValues {
		found := false
		for _, result := range results {
			if result.Name == name {
				found = true
				if value, ok := result.Value.(float64); ok {
					tolerance := 0.01
					if value < expectedValue-tolerance || value > expectedValue+tolerance {
						t.Errorf("For %s: expected %.2f, got %.2f", name, expectedValue, value)
					}
				}
				break
			}
		}
		if !found {
			t.Errorf("Result for %s not found in parsed data", name)
		}
	}
}

// TestGroupExecutorIntegration tests the complete group executor workflow
func TestGroupExecutorIntegration(t *testing.T) {
	// Create complete register configuration
	registers := []config.Register{
		{Name: "voltage", Address: 0x2000, Count: 2, DataType: "float32", Unit: "V"},
		{Name: "current", Address: 0x2002, Count: 2, DataType: "float32", Unit: "A"},
		{Name: "active_power", Address: 0x2004, Count: 2, DataType: "float32", Unit: "W"},
		{Name: "total_energy", Address: 0x4000, Count: 4, DataType: "uint64", Unit: "kWh", Scale: 0.01},
	}

	factory := modbus.NewCommandFactory()
	registry := make(map[string]modbus.ModbusCommand)
	for _, reg := range registers {
		cmd := factory.CreateCommand(reg)
		if cmd != nil {
			registry[reg.Name] = cmd
		}
	}

	executor := groups.NewGroupExecutor()

	// Create instant group
	instantRegs := []config.Register{registers[0], registers[1], registers[2]}
	instantGroup := groups.NewInstantGroup(instantRegs, registry)
	executor.AddGroup("instant", instantGroup)

	// Create energy group
	energyRegs := []config.Register{registers[3]}
	energyGroup := groups.NewEnergyGroup(energyRegs, registry)
	executor.AddGroup("energy", energyGroup)

	// Verify groups are registered
	groupNames := []string{"instant", "energy"}
	for _, name := range groupNames {
		group := executor.GetGroup(name)
		if group == nil {
			t.Errorf("Group '%s' not found in executor", name)
		}
	}

	// Execute instant group
	instantRequest := executor.ExecuteGroup("instant")
	if instantRequest == nil {
		t.Fatal("Instant group execution returned nil request")
	}

	if instantRequest.StartAddr != 0x2000 {
		t.Errorf("Expected instant group start at 0x2000, got 0x%X", instantRequest.StartAddr)
	}

	// Execute energy group
	energyRequest := executor.ExecuteGroup("energy")
	if energyRequest == nil {
		t.Fatal("Energy group execution returned nil request")
	}

	if energyRequest.StartAddr != 0x4000 {
		t.Errorf("Expected energy group start at 0x4000, got 0x%X", energyRequest.StartAddr)
	}
}

// TestGroupWithInvalidData tests error handling in group parsing
func TestGroupWithInvalidData(t *testing.T) {
	registers := []config.Register{
		{Name: "voltage", Address: 0x2000, Count: 2, DataType: "float32", Unit: "V"},
	}

	factory := modbus.NewCommandFactory()
	registry := make(map[string]modbus.ModbusCommand)
	for _, reg := range registers {
		registry[reg.Name] = factory.CreateCommand(reg)
	}

	group := groups.NewInstantGroup(registers, registry)

	// Invalid data: too short
	invalidData := []byte{0x43, 0x5C}

	_, err := group.ParseResults(invalidData)
	if err == nil {
		t.Error("Expected error for invalid data length, got nil")
	}
}
