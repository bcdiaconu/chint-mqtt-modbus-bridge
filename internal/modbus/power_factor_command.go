package modbus

import (
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/internal/config"
)

// PowerFactorCommand Command for reading power factor
type PowerFactorCommand struct {
	*BaseCommand
}

// NewPowerFactorCommand creates a power factor Command
func NewPowerFactorCommand(register config.Register, slaveID uint8) *PowerFactorCommand {
	return &PowerFactorCommand{
		BaseCommand: NewBaseCommand(register, slaveID),
	}
}

// ParseData parses data for power factor
func (c *PowerFactorCommand) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	// Always return absolute value for power factor
	return math.Abs(float64(value)), nil
}
