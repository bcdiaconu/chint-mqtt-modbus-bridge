package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/internal/config"
)

// BaseCommand base implementation for all commands
// Template Method Pattern combined with Strategy Pattern
type BaseCommand struct {
	register config.Register
	address  uint16
	slaveID  uint8
}

// NewBaseCommand creates a base command
func NewBaseCommand(register config.Register, slaveID uint8) *BaseCommand {
	return &BaseCommand{
		register: register,
		address:  register.Address,
		slaveID:  slaveID,
	}
}

// Execute common implementation for executing Modbus command
func (c *BaseCommand) Execute(ctx context.Context, gateway Gateway) ([]byte, error) {
	// Function Code 03 - Read Holding Registers
	// Read 2 registers (4 bytes) for float32
	// Use atomic send/receive to prevent racing conditions
	response, err := gateway.SendCommandAndWaitForResponse(ctx, c.slaveID, 0x03, c.address, 2, 5)
	if err != nil {
		return nil, fmt.Errorf("error executing command: %w", err)
	}

	return response, nil
}

// GetUnit returns the unit of measurement
func (c *BaseCommand) GetUnit() string {
	return c.register.Unit
}

// GetTopic returns the Home Assistant topic
func (c *BaseCommand) GetTopic() string {
	return c.register.HATopic
}

// GetName returns the register name
func (c *BaseCommand) GetName() string {
	return c.register.Name
}

// GetDeviceClass returns the device class
func (c *BaseCommand) GetDeviceClass() string {
	return c.register.DeviceClass
}

// GetStateClass returns the state class
func (c *BaseCommand) GetStateClass() string {
	return c.register.StateClass
}

// ParseData default implementation for parsing data
func (c *BaseCommand) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	// Default implementation for float32 - BigEndian (standard Modbus)
	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	return float64(value), nil
}
