package modbus

import (
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/pkg/config"
)

// FrequencyCommand Command for reading frequency
type FrequencyCommand struct {
	*BaseCommand
}

// NewFrequencyCommand creates a frequency Command
func NewFrequencyCommand(register config.Register, slaveID uint8) *FrequencyCommand {
	return &FrequencyCommand{
		BaseCommand: NewBaseCommand(register, slaveID),
	}
}

// ParseData parses data for frequency
func (c *FrequencyCommand) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	return float64(value), nil
}
