package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/errors"
	"mqtt-modbus-bridge/pkg/gateway"
)

// SingleRegisterStrategy reads a single Modbus register (float32, 2 registers)
type SingleRegisterStrategy struct {
	*BaseStrategy
}

// NewSingleRegisterStrategy creates a new single register strategy
func NewSingleRegisterStrategy(key string, register config.Register, slaveID uint8, gw gateway.Gateway, cache *ValueCache) *SingleRegisterStrategy {
	return &SingleRegisterStrategy{
		BaseStrategy: NewBaseStrategy(key, register, slaveID, gw, cache),
	}
}

// Execute reads a single register from Modbus and returns the result
func (s *SingleRegisterStrategy) Execute(ctx context.Context) (*CommandResult, error) {
	// Check cache first
	if s.cache != nil {
		if cached, found := s.cache.Get(s.key); found {
			return cached, nil
		}
	}

	// Read 2 registers (4 bytes) for float32 using function code 0x03
	data, err := s.gateway.SendCommandAndWaitForResponse(
		ctx,
		s.slaveID,
		0x03, // Read Holding Registers
		s.register.Address,
		2, // 2 registers for float32
		5, // 5 second timeout
	)
	if err != nil {
		modbusErr := errors.NewModbusError("read_single_register", err, s.slaveID, s.key)
		modbusErr.FunctionCode = 0x03
		modbusErr.Address = s.register.Address
		return nil, modbusErr
	}

	if len(data) != 4 {
		modbusErr := errors.NewModbusError("parse_single_register",
			fmt.Errorf("expected 4 bytes, got %d bytes", len(data)),
			s.slaveID, s.key)
		modbusErr.Address = s.register.Address
		return nil, modbusErr
	}

	// Parse float32 (Big Endian)
	bits := binary.BigEndian.Uint32(data)
	rawValue := math.Float32frombits(bits)

	// Apply scale factor
	value := float64(rawValue) * s.register.ScaleFactor

	// Create result
	result := &CommandResult{
		Strategy:    "single_register",
		Name:        s.register.Name,
		Value:       value,
		Unit:        s.register.Unit,
		Topic:       s.register.HATopic,
		SensorKey:   s.key, // Use the full key as sensor key
		DeviceClass: s.register.DeviceClass,
		StateClass:  s.register.StateClass,
		RawData:     data,
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set(s.key, result)
	}

	return result, nil
}
