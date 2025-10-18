package modbus

import (
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/pkg/config"
)

// PowerCommand Command for reading power
type PowerCommand struct {
	*BaseCommand
}

// NewPowerCommand creates a power Command
func NewPowerCommand(register config.Register, slaveID uint8) *PowerCommand {
	return &PowerCommand{
		BaseCommand: NewBaseCommand(register, slaveID),
	}
}

// ParseData parses data for power
func (c *PowerCommand) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	// Apply scale factor from configuration (default 1.0 if not specified)
	convertedValue := float64(value) * c.register.ScaleFactor

	return convertedValue, nil
}
