package modbus

import (
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/internal/config"
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

	// Convert from KW/KVA to W/VA (multiply by 1000)
	convertedValue := float64(value) * 1000.0

	return convertedValue, nil
}
