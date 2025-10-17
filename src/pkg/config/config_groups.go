package config

import (
	"fmt"
	"mqtt-modbus-bridge/pkg/logger"
)

// RegisterGroup defines a contiguous block of Modbus registers to read in one command
// Used in configuration version 2.0+
type RegisterGroup struct {
	Name          string          `yaml:"name"`
	SlaveID       uint8           `yaml:"slave_id"`       // Modbus device ID
	FunctionCode  uint8           `yaml:"function_code"`  // Modbus function (0x03, 0x04, etc.)
	StartAddress  uint16          `yaml:"start_address"`  // First register address
	RegisterCount uint16          `yaml:"register_count"` // Number of 16-bit registers
	Enabled       bool            `yaml:"enabled"`        // Enable/disable this group
	Registers     []GroupRegister `yaml:"registers"`      // Registers in this group
}

// GroupRegister defines a register within a group
type GroupRegister struct {
	Key           string  `yaml:"key"`    // Unique identifier (e.g., "voltage")
	Name          string  `yaml:"name"`   // Display name
	Offset        int     `yaml:"offset"` // Byte offset from group start
	Unit          string  `yaml:"unit"`
	DeviceClass   string  `yaml:"device_class"`
	StateClass    string  `yaml:"state_class"`
	HATopic       string  `yaml:"ha_topic,omitempty"` // Optional: Auto-constructed if not provided in v2.1
	Min           float64 `yaml:"min,omitempty"`
	Max           float64 `yaml:"max,omitempty"`
	MaxKwhPerHour float64 `yaml:"max_kwh_per_hour,omitempty"`
}

// CalculatedRegister defines a virtual register calculated from other registers
type CalculatedRegister struct {
	Name        string   `yaml:"name"`
	Unit        string   `yaml:"unit"`
	DeviceClass string   `yaml:"device_class"`
	StateClass  string   `yaml:"state_class"`
	HATopic     string   `yaml:"ha_topic"`
	Formula     string   `yaml:"formula"`    // Formula for calculation
	DependsOn   []string `yaml:"depends_on"` // Register keys this depends on
}

// Validate validates the register group configuration
func (g *RegisterGroup) Validate() error {
	if g.SlaveID == 0 {
		return fmt.Errorf("slave_id is required for register group '%s'", g.Name)
	}
	if g.FunctionCode == 0 {
		return fmt.Errorf("function_code is required for register group '%s'", g.Name)
	}
	if g.RegisterCount == 0 {
		return fmt.Errorf("register_count is required for register group '%s'", g.Name)
	}
	if len(g.Registers) == 0 {
		return fmt.Errorf("no registers defined for group '%s'", g.Name)
	}

	// Validate that offsets are within the read range
	maxBytes := int(g.RegisterCount) * 2 // Each register is 2 bytes
	for _, reg := range g.Registers {
		if reg.Offset < 0 {
			return fmt.Errorf("register '%s' has negative offset", reg.Key)
		}
		if reg.Offset+4 > maxBytes { // +4 because float32 is 4 bytes
			return fmt.Errorf("register '%s' offset %d exceeds group range (max %d bytes)",
				reg.Key, reg.Offset, maxBytes)
		}
	}

	return nil
}

// ValidateGroups validates all register groups and their dependencies
func ValidateGroups(groups map[string]RegisterGroup, calculated map[string]CalculatedRegister) error {
	if len(groups) == 0 {
		return fmt.Errorf("at least one register group is required")
	}

	for name, group := range groups {
		if err := group.Validate(); err != nil {
			return fmt.Errorf("invalid register group '%s': %w", name, err)
		}
	}

	// Validate calculated registers dependencies
	for key, calc := range calculated {
		for _, dep := range calc.DependsOn {
			found := false
			for _, group := range groups {
				for _, reg := range group.Registers {
					if reg.Key == dep {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				return fmt.Errorf("calculated register '%s' depends on unknown register '%s'", key, dep)
			}
		}
	}

	return nil
}

// ConvertGroupsToRegisters converts RegisterGroups to old-style Registers for backward compatibility
func ConvertGroupsToRegisters(groups map[string]RegisterGroup) map[string]Register {
	registers := make(map[string]Register)

	for _, group := range groups {
		for _, reg := range group.Registers {
			// Calculate actual address from group start + offset
			// Safe conversion: offset is in bytes, divide by 2 to get register offset
			offsetInRegisters := reg.Offset / 2

			// Validate that the address calculation won't overflow uint16
			if offsetInRegisters < 0 || offsetInRegisters > 0xFFFF {
				// Skip invalid register (offset out of range)
				continue
			}

			// Calculate final address and check for overflow
			finalAddress := int(group.StartAddress) + offsetInRegisters
			if finalAddress > 0xFFFF {
				// Skip invalid register (address overflow)
				continue
			}

			// Safe conversion after validation
			// #nosec G115 -- Validated above that offsetInRegisters fits in uint16
			address := group.StartAddress + uint16(offsetInRegisters)

			// Create pointers for optional fields (only if non-zero)
			var minPtr, maxPtr, maxKwhPtr *float64
			if reg.Min != 0 {
				minPtr = &reg.Min
			}
			if reg.Max != 0 {
				maxPtr = &reg.Max
			}
			if reg.MaxKwhPerHour != 0 {
				maxKwhPtr = &reg.MaxKwhPerHour
			}

			registers[reg.Key] = Register{
				Name:          reg.Name,
				Address:       address,
				Unit:          reg.Unit,
				DeviceClass:   reg.DeviceClass,
				StateClass:    reg.StateClass,
				HATopic:       reg.HATopic,
				Min:           minPtr,
				Max:           maxPtr,
				MaxKwhPerHour: maxKwhPtr,
			}

			logger.LogDebug("Converted group register '%s' (group: %s, offset: %d) -> address 0x%04X",
				reg.Key, group.Name, reg.Offset, address)
		}
	}

	return registers
}
