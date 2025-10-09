package modbus

import "mqtt-modbus-bridge/pkg/config"

// CommandFactory factory for creating commands
// Factory Pattern for creating commands based on register type
type CommandFactory struct {
	slaveID uint8
}

// NewCommandFactory creates a new factory with slave ID
func NewCommandFactory(slaveID uint8) *CommandFactory {
	return &CommandFactory{
		slaveID: slaveID,
	}
}

// CreateCommand creates the appropriate command based on device_class
func (f *CommandFactory) CreateCommand(register config.Register) (ModbusCommand, error) {
	switch register.DeviceClass {
	case "voltage":
		return NewVoltageCommand(register, f.slaveID), nil
	case "frequency":
		return NewFrequencyCommand(register, f.slaveID), nil
	case "current":
		return NewCurrentCommand(register, f.slaveID), nil
	case "energy":
		return NewEnergyCommand(register, f.slaveID), nil
	case "power", "apparent_power":
		return NewPowerCommand(register, f.slaveID), nil
	case "power_factor":
		return NewPowerFactorCommand(register, f.slaveID), nil
	case "reactive_power":
		// For reactive power, we need the register names for active and apparent power
		return NewReactivePowerCommand(register, f.slaveID, "power_active", "power_apparent"), nil
	default:
		// Default command for unknown registers
		return NewBaseCommand(register, f.slaveID), nil
	}
}
