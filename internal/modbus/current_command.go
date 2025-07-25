package modbus

import (
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/internal/config"
)

// CurrentCommand command for reading current
type CurrentCommand struct {
	*BaseCommand
}

// NewCurrentCommand creates a current command
func NewCurrentCommand(register config.Register, slaveID uint8) *CurrentCommand {
	return &CurrentCommand{
		BaseCommand: NewBaseCommand(register, slaveID),
	}
}

// ParseData parses data for current
func (c *CurrentCommand) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	return float64(value), nil
}
