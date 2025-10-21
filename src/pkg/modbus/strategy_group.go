package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/errors"
	"mqtt-modbus-bridge/pkg/gateway"
	"mqtt-modbus-bridge/pkg/logger"
	"strings"
)

// extractSensorKey extracts the sensor key from a full key (device_key_sensor_key)
// Example: "energy_meter_lights_power_active" -> "power_active"
func extractSensorKey(fullKey string) string {
	// Find the last underscore followed by a known sensor type
	// Common sensor keys: voltage, current, power_active, power_reactive, etc.
	parts := strings.Split(fullKey, "_")
	if len(parts) >= 2 {
		// Take the last 2 parts for compound keys like "power_active"
		// or just last part for simple keys like "voltage"
		if len(parts) >= 3 {
			// Check if last 2 parts form a known compound key
			lastTwo := strings.Join(parts[len(parts)-2:], "_")
			knownKeys := []string{"power_active", "power_reactive", "power_apparent", "power_factor",
				"energy_total", "energy_imported", "energy_exported"}
			for _, known := range knownKeys {
				if lastTwo == known {
					return lastTwo
				}
			}
		}
		// Otherwise return just the last part
		return parts[len(parts)-1]
	}
	return fullKey
}

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

	// Log group execution start for debugging sequential execution
	logger.LogTrace("🔄 Executing group '%s' (Slave %d, Addr 0x%04X, Count %d)",
		s.groupKey, s.slaveID, s.groupConfig.StartAddress, s.groupConfig.RegisterCount)

	// Read the entire group in one Modbus transaction
	// NOTE: SendCommandAndWaitForResponse uses commandMutex to ensure
	// SEQUENTIAL execution - no overlap between different slaves or groups
	data, err := s.gateway.SendCommandAndWaitForResponse(
		ctx,
		s.slaveID,
		s.groupConfig.FunctionCode,
		s.groupConfig.StartAddress,
		s.groupConfig.RegisterCount,
		15, // 15 second timeout (increased from 5s due to slow gateway/device response times)
	)
	if err != nil {
		logger.LogWarn("❌ Group '%s' (Slave %d) read failed: %v", s.groupKey, s.slaveID, err)
		modbusErr := errors.NewModbusError("read_register_group", err, s.slaveID, s.groupKey)
		modbusErr.FunctionCode = s.groupConfig.FunctionCode
		modbusErr.Address = s.groupConfig.StartAddress
		return nil, modbusErr
	}

	logger.LogTrace("✅ Group '%s' (Slave %d) read successful (%d bytes)", s.groupKey, s.slaveID, len(data))

	expectedBytes := int(s.groupConfig.RegisterCount) * 2 // Each register is 2 bytes
	if len(data) != expectedBytes {
		modbusErr := errors.NewModbusError("parse_register_group",
			fmt.Errorf("expected %d bytes for group '%s', got %d bytes", expectedBytes, s.groupKey, len(data)),
			s.slaveID, s.groupKey)
		modbusErr.Address = s.groupConfig.StartAddress
		return nil, modbusErr
	}

	// Parse each register from the group data
	for _, regWithKey := range s.registers {
		reg := regWithKey.Register
		offset := reg.Address - s.groupConfig.StartAddress // Offset in registers
		byteOffset := int(offset) * 2                      // Offset in bytes

		// Ensure we have enough data
		if byteOffset+4 > len(data) {
			modbusErr := errors.NewModbusError("parse_register_offset",
				fmt.Errorf("register '%s' offset %d exceeds group data length %d", regWithKey.Key, byteOffset, len(data)),
				s.slaveID, regWithKey.Key)
			modbusErr.Address = reg.Address
			return nil, modbusErr
		}

		// Extract 4 bytes for float32
		registerData := data[byteOffset : byteOffset+4]
		bits := binary.BigEndian.Uint32(registerData)
		rawValue := math.Float32frombits(bits)

		// Apply scale factor
		value := float64(rawValue) * reg.ScaleFactor

		// Apply absolute value if configured
		if reg.ApplyAbs {
			value = math.Abs(value)
		}

		// Extract just the sensor key from the full key (device_key_sensor_key)
		sensorKey := extractSensorKey(regWithKey.Key)

		// Create result
		result := &CommandResult{
			Strategy:    "group_register",
			Name:        reg.Name,
			Value:       value,
			Unit:        reg.Unit,
			Topic:       reg.HATopic,
			SensorKey:   sensorKey,
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
