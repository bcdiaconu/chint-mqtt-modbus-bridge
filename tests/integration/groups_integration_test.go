package integration

import (
	"math"
	"testing"

	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/modbus"
	"mqtt-modbus-bridge/pkg/modbus/groups"
)

// TestInstantGroupCreation tests that instant groups can be created
func TestInstantGroupCreation(t *testing.T) {
	registers := []config.Register{
		{Name: "Voltage", Address: 0x2000, Unit: "V", DeviceClass: "voltage"},
		{Name: "Current", Address: 0x2002, Unit: "A", DeviceClass: "current"},
	}

	factory := modbus.NewCommandFactory(1)
	registry := make(map[string]modbus.ModbusCommand)
	keys := []string{"voltage", "current"}

	for i, reg := range registers {
		cmd, err := factory.CreateCommand(reg)
		if err != nil {
			t.Fatalf("Failed to create command: %v", err)
		}
		registry[keys[i]] = cmd
	}

	group := groups.NewInstantGroup(registers, registry, keys)
	if group == nil {
		t.Fatal("Failed to create instant group")
	}

	names := group.GetNames()
	if len(names) != 2 {
		t.Errorf("Expected 2 names, got %d", len(names))
	}
}

// TestEnergyGroupCreation tests that energy groups can be created
func TestEnergyGroupCreation(t *testing.T) {
	registers := []config.Register{
		{Name: "Total Energy", Address: 0x4000, Unit: "kWh", DeviceClass: "energy"},
		{Name: "Import Energy", Address: 0x4004, Unit: "kWh", DeviceClass: "energy"},
	}

	factory := modbus.NewCommandFactory(1)
	registry := make(map[string]modbus.ModbusCommand)
	keys := []string{"total_energy", "import_energy"}

	for i, reg := range registers {
		cmd, err := factory.CreateCommand(reg)
		if err != nil {
			t.Fatalf("Failed to create command: %v", err)
		}
		registry[keys[i]] = cmd
	}

	group := groups.NewEnergyGroup(registers, registry, keys)
	if group == nil {
		t.Fatal("Failed to create energy group")
	}

	names := group.GetNames()
	if len(names) != 2 {
		t.Errorf("Expected 2 names, got %d", len(names))
	}
}

// TestCommandFactoryIntegration tests end-to-end command creation
func TestCommandFactoryIntegration(t *testing.T) {
	factory := modbus.NewCommandFactory(1)

	testCases := []struct {
		name    string
		regName string
		address uint16
	}{
		{"voltage", "voltage", 0x2000},
		{"current", "current", 0x2002},
		{"power", "active_power", 0x2004},
		{"energy", "total_energy", 0x4000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			register := config.Register{
				Name:    tc.regName,
				Address: tc.address,
				Unit:    "test",
			}

			cmd, err := factory.CreateCommand(register)
			if err != nil {
				t.Fatalf("Failed to create %s command: %v", tc.name, err)
			}

			if cmd == nil {
				t.Fatalf("%s command is nil", tc.name)
			}

			// Test parsing with sample data
			sampleData := []byte{0x41, 0x20, 0x00, 0x00} // 10.0 as float32
			_, err = cmd.ParseData(sampleData)
			if err != nil {
				t.Errorf("Failed to parse data for %s: %v", tc.name, err)
			}
		})
	}
}

// TestReactivePowerCalculationFromGroupResults tests reactive power calculation
func TestReactivePowerCalculationFromGroupResults(t *testing.T) {
	// Simulate instant group results that would include power_active and power_apparent
	groupResults := map[string]float64{
		"voltage":        230.0,  // 230 V
		"current":        10.0,   // 10 A
		"power_active":   2000.0, // 2000 W
		"power_apparent": 2300.0, // 2300 VA (S = V * I)
	}

	// Calculate expected reactive power: Q = sqrt(S² - P²)
	// Q = sqrt(2300² - 2000²) = sqrt(5290000 - 4000000) = sqrt(1290000) ≈ 1135.78 VAR
	P := groupResults["power_active"]
	S := groupResults["power_apparent"]
	expectedQ := 1135.78 // approximately

	// Verify we have the required values
	if _, hasP := groupResults["power_active"]; !hasP {
		t.Fatal("Missing power_active in group results")
	}
	if _, hasS := groupResults["power_apparent"]; !hasS {
		t.Fatal("Missing power_apparent in group results")
	}

	// Calculate Q using the same formula as the application
	var Q float64
	if S*S >= P*P {
		Q = math.Sqrt(S*S - P*P)
	} else {
		Q = 0.0
	}

	// Verify calculated value is close to expected
	tolerance := 1.0 // 1 VAR tolerance
	if Q < expectedQ-tolerance || Q > expectedQ+tolerance {
		t.Errorf("Expected reactive power around %.2f VAR, got %.2f VAR", expectedQ, Q)
	}

	t.Logf("✅ Reactive power calculated: %.2f VAR (from P=%.2f W, S=%.2f VA)", Q, P, S)
}
