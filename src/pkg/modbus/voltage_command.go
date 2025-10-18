package modbus

import (
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/pkg/config"
)

// VoltageCommand command for reading voltage
type VoltageCommand struct {
	*BaseCommand
}

// NewVoltageCommand creates a voltage command
func NewVoltageCommand(register config.Register, slaveID uint8) *VoltageCommand {
	return &VoltageCommand{
		BaseCommand: NewBaseCommand(register, slaveID),
	}
}

// ParseData parses data for voltage
func (c *VoltageCommand) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	// Convert bytes to float32 (IEEE 754)
	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	// Apply scale factor from configuration
	return float64(value) * c.register.ScaleFactor, nil
}
