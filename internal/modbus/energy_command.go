package modbus

import (
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/internal/config"
)

// EnergyCommand Command for reading energy
type EnergyCommand struct {
	*BaseCommand
}

// NewEnergyCommand creates an energy Command
func NewEnergyCommand(register config.Register, slaveID uint8) *EnergyCommand {
	return &EnergyCommand{
		BaseCommand: NewBaseCommand(register, slaveID),
	}
}

// ParseData parses data for energy
func (c *EnergyCommand) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	return float64(value), nil
}
