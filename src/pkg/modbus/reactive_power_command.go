package modbus

import (
	"context"
	"fmt"
	"math"
	"mqtt-modbus-bridge/pkg/config"
)

// ReactivePowerCommand Command for calculating reactive power
// Reactive power is calculated as: Q = sqrt(S² - P²) where S = apparent power, P = active power
type ReactivePowerCommand struct {
	*BaseCommand
	activePowerName   string
	apparentPowerName string
	executor          *CommandExecutor
}

// NewReactivePowerCommand creates a new Command for reactive power
func NewReactivePowerCommand(register config.Register, slaveID uint8, activePowerName, apparentPowerName string) *ReactivePowerCommand {
	return &ReactivePowerCommand{
		BaseCommand:       NewBaseCommand(register, slaveID),
		activePowerName:   activePowerName,
		apparentPowerName: apparentPowerName,
	}
}

// SetExecutor sets the executor to allow reading other values
func (c *ReactivePowerCommand) SetExecutor(executor *CommandExecutor) {
	c.executor = executor
}

// ExecuteCommand implements reading reactive power by calculation
func (c *ReactivePowerCommand) ExecuteCommand(ctx context.Context, gateway Gateway) (*CommandResult, error) {
	if c.executor == nil {
		return nil, fmt.Errorf("executor not set for reactive power calculation")
	}

	// Read active power
	activePowerResult, err := c.executor.ExecuteCommand(ctx, c.activePowerName)
	if err != nil {
		return nil, fmt.Errorf("error reading active power: %w", err)
	}

	// Read apparent power
	apparentPowerResult, err := c.executor.ExecuteCommand(ctx, c.apparentPowerName)
	if err != nil {
		return nil, fmt.Errorf("error reading apparent power: %w", err)
	}

	// Calculate reactive power: Q = sqrt(S² - P²)
	P := activePowerResult.Value   // Active power in W
	S := apparentPowerResult.Value // Apparent power in VA

	// Validation to avoid sqrt of negative number
	if S*S < P*P {
		// If apparent is less than active (theoretically impossible, but may occur due to measurement errors)
		// Set Q = 0
		Q := 0.0
		return &CommandResult{
			Strategy:    c.register.Name,
			Name:        c.register.Name,
			Value:       Q,
			Unit:        c.register.Unit,
			Topic:       c.register.HATopic,
			DeviceClass: c.register.DeviceClass,
			StateClass:  c.register.StateClass,
		}, nil
	}

	Q := math.Sqrt(S*S - P*P) // Reactive power in VAR

	return &CommandResult{
		Strategy:    c.register.Name,
		Name:        c.register.Name,
		Value:       Q,
		Unit:        c.register.Unit,
		Topic:       c.register.HATopic,
		DeviceClass: c.register.DeviceClass,
		StateClass:  c.register.StateClass,
	}, nil
}

// ParseData is not used for this Command because we calculate from other values
func (c *ReactivePowerCommand) ParseData(rawData []byte) (float64, error) {
	return 0, fmt.Errorf("ParseData not used for reactive power - use ExecuteCommand instead")
}
