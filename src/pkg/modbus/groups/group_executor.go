package groups

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/modbus"
)

// GroupExecutor executes grouped Modbus queries
// Now device-agnostic - uses slave_id from RegisterGroup
type GroupExecutor struct {
	gateway         modbus.Gateway
	commandRegistry map[string]modbus.ModbusCommand
}

// NewGroupExecutor creates a new group executor
// Note: slaveID is now obtained from each RegisterGroup, not passed globally
func NewGroupExecutor(gateway modbus.Gateway, commandRegistry map[string]modbus.ModbusCommand) *GroupExecutor {
	return &GroupExecutor{
		gateway:         gateway,
		commandRegistry: commandRegistry,
	}
}

// CreateInstantGroup creates an instant values group from register names
func (e *GroupExecutor) CreateInstantGroup(registerNames []string, allRegisters map[string]config.Register) (*InstantGroup, error) {
	registers := make([]config.Register, 0, len(registerNames))
	commands := make(map[string]modbus.ModbusCommand)
	keys := make([]string, 0, len(registerNames))

	for _, name := range registerNames {
		reg, exists := allRegisters[name]
		if !exists {
			return nil, fmt.Errorf("register %s not found in configuration", name)
		}

		cmd, exists := e.commandRegistry[name]
		if !exists {
			return nil, fmt.Errorf("command for register %s not found", name)
		}

		registers = append(registers, reg)
		commands[name] = cmd
		keys = append(keys, name) // Store the config key
	}

	return NewInstantGroup(registers, commands, keys), nil
}

// CreateEnergyGroup creates an energy values group from register names
func (e *GroupExecutor) CreateEnergyGroup(registerNames []string, allRegisters map[string]config.Register) (*EnergyGroup, error) {
	registers := make([]config.Register, 0, len(registerNames))
	commands := make(map[string]modbus.ModbusCommand)
	keys := make([]string, 0, len(registerNames))

	for _, name := range registerNames {
		reg, exists := allRegisters[name]
		if !exists {
			return nil, fmt.Errorf("register %s not found in configuration", name)
		}

		cmd, exists := e.commandRegistry[name]
		if !exists {
			return nil, fmt.Errorf("command for register %s not found", name)
		}

		registers = append(registers, reg)
		commands[name] = cmd
		keys = append(keys, name) // Store the config key
	}

	return NewEnergyGroup(registers, commands, keys), nil
}

// ExecuteGroup executes a group strategy and returns parsed results
// groupConfig: RegisterGroup configuration containing slave_id for this group
func (e *GroupExecutor) ExecuteGroup(ctx context.Context, group GroupStrategy, groupConfig config.RegisterGroup) (map[string]float64, error) {
	// Use slave_id from the RegisterGroup configuration
	slaveID := groupConfig.SlaveID
	if slaveID == 0 {
		return nil, fmt.Errorf("group '%s' has no slave_id", groupConfig.Name)
	}

	// Execute the grouped query
	rawData, err := group.Execute(ctx, e.gateway, slaveID)
	if err != nil {
		return nil, fmt.Errorf("error executing group query: %w", err)
	}

	// Parse the results using individual command parsers
	results, err := group.ParseResults(rawData)
	if err != nil {
		return nil, fmt.Errorf("error parsing group results: %w", err)
	}

	return results, nil
}
