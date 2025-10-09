package groups

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/modbus"
)

const energyRegisterDataSize = 4 // Each value is 4 bytes (float32)

type EnergyGroup struct {
	Registers       []config.Register
	CommandRegistry map[string]modbus.ModbusCommand // Maps register KEY (from config) to its command
	RegisterKeys    []string                        // Config keys in same order as Registers
}

func NewEnergyGroup(registers []config.Register, commands map[string]modbus.ModbusCommand, keys []string) *EnergyGroup {
	return &EnergyGroup{
		Registers:       registers,
		CommandRegistry: commands,
		RegisterKeys:    keys,
	}
}

func (g *EnergyGroup) Execute(ctx context.Context, gateway modbus.Gateway, slaveID uint8) ([]byte, error) {
	if len(g.Registers) == 0 {
		return nil, nil
	}
	minAddr := g.Registers[0].Address
	maxAddr := g.Registers[0].Address
	for _, reg := range g.Registers {
		if reg.Address < minAddr {
			minAddr = reg.Address
		}
		if reg.Address > maxAddr {
			maxAddr = reg.Address
		}
	}
	// Calculate register count: addresses are in register units (each address = 1 register = 2 bytes)
	// Each float32 value occupies 2 consecutive Modbus registers
	// maxAddr points to the start of the last value, so we need to add 2 more registers for it
	regCount := (maxAddr - minAddr) + 2
	return gateway.SendCommandAndWaitForResponse(ctx, slaveID, 0x03, minAddr, regCount, 5)
}

func (g *EnergyGroup) ParseResults(rawData []byte) (map[string]float64, error) {
	results := make(map[string]float64)

	// Find minimum address for offset calculation
	minAddr := g.Registers[0].Address
	for _, reg := range g.Registers {
		if reg.Address < minAddr {
			minAddr = reg.Address
		}
	}

	// Parse each register using its corresponding command's ParseData method
	for i, reg := range g.Registers {
		// Get the config key for this register
		configKey := g.RegisterKeys[i]

		// Get the command for this register from the registry using the config key
		cmd, exists := g.CommandRegistry[configKey]
		if !exists {
			return nil, fmt.Errorf("no command found for register key: %s (name: %s)", configKey, reg.Name)
		}

		// Calculate offset in the raw data based on register address
		offset := int((reg.Address - minAddr) * 2) // 2 bytes per register

		// Extract the data slice for this register (4 bytes for float32)
		if offset+energyRegisterDataSize > len(rawData) {
			return nil, fmt.Errorf("insufficient data for register %s at offset %d", reg.Name, offset)
		}
		dataSlice := rawData[offset : offset+energyRegisterDataSize]

		// Use the command's ParseData method to parse the value
		value, err := cmd.ParseData(dataSlice)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", reg.Name, err)
		}

		// Store result using config key for consistency
		results[configKey] = value
	}

	return results, nil
}

func (g *EnergyGroup) GetNames() []string {
	names := make([]string, len(g.Registers))
	for i, reg := range g.Registers {
		names[i] = reg.Name
	}
	return names
}

var _ GroupStrategy = (*EnergyGroup)(nil)
