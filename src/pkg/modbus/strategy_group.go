package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/gateway"
)

// GroupRegisterStrategy reads multiple contiguous registers as a group
type GroupRegisterStrategy struct {
	groupKey    string
	groupConfig config.RegisterGroup
	registers   []RegisterWithKey // Registers in this group with their keys
	slaveID     uint8
	gateway     gateway.Gateway
	cache       *ValueCache
}

// RegisterWithKey pairs a register key with its configuration
type RegisterWithKey struct {
	Key      string
	Register config.Register
}

// NewGroupRegisterStrategy creates a new group register strategy
func NewGroupRegisterStrategy(
	groupKey string,
	groupConfig config.RegisterGroup,
	registers []RegisterWithKey,
	slaveID uint8,
	gateway gateway.Gateway,
	cache *ValueCache,
) *GroupRegisterStrategy {
	return &GroupRegisterStrategy{
		groupKey:    groupKey,
		groupConfig: groupConfig,
		registers:   registers,
		slaveID:     slaveID,
		gateway:     gateway,
		cache:       cache,
	}
}

// GetRegisters returns all registers in this group
func (s *GroupRegisterStrategy) GetRegisters() []RegisterWithKey {
	return s.registers
}

// Execute reads all registers in the group and returns multiple results
func (s *GroupRegisterStrategy) Execute(ctx context.Context) (map[string]*CommandResult, error) {
	results := make(map[string]*CommandResult)

	// Read the entire group in one Modbus transaction
	data, err := s.gateway.SendCommandAndWaitForResponse(
		ctx,
		s.slaveID,
		s.groupConfig.FunctionCode,
		s.groupConfig.StartAddress,
		s.groupConfig.RegisterCount,
		5, // 5 second timeout
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read register group '%s' at address 0x%04X: %w",
			s.groupKey, s.groupConfig.StartAddress, err)
	}

	expectedBytes := int(s.groupConfig.RegisterCount) * 2 // Each register is 2 bytes
	if len(data) != expectedBytes {
		return nil, fmt.Errorf("expected %d bytes for group '%s', got %d bytes",
			expectedBytes, s.groupKey, len(data))
	}

	// Parse each register from the group data
	for _, regWithKey := range s.registers {
		reg := regWithKey.Register
		offset := reg.Address - s.groupConfig.StartAddress // Offset in registers
		byteOffset := int(offset) * 2                      // Offset in bytes

		// Ensure we have enough data
		if byteOffset+4 > len(data) {
			return nil, fmt.Errorf("register '%s' offset %d exceeds group data length %d",
				regWithKey.Key, byteOffset, len(data))
		}

		// Extract 4 bytes for float32
		registerData := data[byteOffset : byteOffset+4]
		bits := binary.BigEndian.Uint32(registerData)
		rawValue := math.Float32frombits(bits)

		// Apply scale factor
		value := float64(rawValue) * reg.ScaleFactor

		// Create result
		result := &CommandResult{
			Strategy:    "group_register",
			Name:        reg.Name,
			Value:       value,
			Unit:        reg.Unit,
			Topic:       reg.HATopic,
			DeviceClass: reg.DeviceClass,
			StateClass:  reg.StateClass,
			RawData:     registerData,
		}

		results[regWithKey.Key] = result

		// Cache individual result
		if s.cache != nil {
			s.cache.Set(regWithKey.Key, result)
		}
	}

	return results, nil
}

// GetKey returns the group key
func (s *GroupRegisterStrategy) GetKey() string {
	return s.groupKey
}

// GetRegisterInfo returns nil (group doesn't have single register info)
func (s *GroupRegisterStrategy) GetRegisterInfo() config.Register {
	return config.Register{}
}
