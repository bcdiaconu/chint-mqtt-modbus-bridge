package config

import (
	"fmt"
	"mqtt-modbus-bridge/pkg/logger"
	"mqtt-modbus-bridge/pkg/topics"
)

// Device represents a Modbus device on the RTU bus (Version 2.1+)
// Organized into 4 sections: metadata, rtu, modbus, homeassistant
type Device struct {
	Metadata         DeviceMetadata     `yaml:"metadata"`                    // Device metadata (name, manufacturer, model)
	RTU              RTUConfig          `yaml:"rtu"`                         // RTU/Physical layer configuration
	Modbus           ModbusDeviceConfig `yaml:"modbus"`                      // Modbus protocol layer
	HomeAssistant    *HADeviceConfig    `yaml:"homeassistant,omitempty"`     // Home Assistant integration (optional)
	CalculatedValues []CalculatedValue  `yaml:"calculated_values,omitempty"` // Calculated/derived values
}

// DeviceMetadata contains device identification and metadata
type DeviceMetadata struct {
	Name         string `yaml:"name"`                   // Display name (e.g., "Energy Meter 1")
	Manufacturer string `yaml:"manufacturer,omitempty"` // Device manufacturer
	Model        string `yaml:"model,omitempty"`        // Device model
	Enabled      bool   `yaml:"enabled"`                // Enable/disable this device
}

// RTUConfig contains RTU/Physical layer configuration
type RTUConfig struct {
	SlaveID      uint8 `yaml:"slave_id"`                // Modbus device ID (1-247)
	PollInterval int   `yaml:"poll_interval,omitempty"` // Override global poll interval (ms)
}

// ModbusDeviceConfig contains Modbus protocol layer configuration
type ModbusDeviceConfig struct {
	RegisterGroups map[string]RegisterGroup `yaml:"register_groups"` // Register groups (instant, energy, status, etc.)
}

// HADeviceConfig contains Home Assistant specific configuration
// All fields are optional and will use defaults if not specified
type HADeviceConfig struct {
	DeviceID     string `yaml:"device_id,omitempty"`    // Unique ID in HA (defaults to device key)
	Manufacturer string `yaml:"manufacturer,omitempty"` // HA manufacturer override (defaults to metadata.manufacturer)
	Model        string `yaml:"model,omitempty"`        // HA model override (defaults to metadata.model)
}

// CalculatedValue represents a value computed from other registers
// Calculated values are executed AFTER all Modbus reads complete
type CalculatedValue struct {
	Key         string   `yaml:"key"`                    // Unique key for this calculated value
	Name        string   `yaml:"name"`                   // Display name
	Unit        string   `yaml:"unit"`                   // Unit of measurement
	Formula     string   `yaml:"formula"`                // Mathematical expression
	ScaleFactor float64  `yaml:"scale_factor,omitempty"` // Multiplier applied to result (default: 1.0)
	DeviceClass string   `yaml:"device_class"`           // Home Assistant device class
	StateClass  string   `yaml:"state_class"`            // Home Assistant state class
	Min         *float64 `yaml:"min,omitempty"`          // Minimum valid value
	Max         *float64 `yaml:"max,omitempty"`          // Maximum valid value
}

// GetName returns the device name from metadata
func (d *Device) GetName() string {
	return d.Metadata.Name
}

// GetSlaveID returns the RTU slave ID
func (d *Device) GetSlaveID() uint8 {
	return d.RTU.SlaveID
}

// GetPollInterval returns the poll interval (0 if not set, use global)
func (d *Device) GetPollInterval() int {
	return d.RTU.PollInterval
}

// IsEnabled returns whether the device is enabled
func (d *Device) IsEnabled() bool {
	return d.Metadata.Enabled
}

// GetHADeviceName returns the Home Assistant device name (uses metadata.name)
func (d *Device) GetHADeviceName() string {
	return d.Metadata.Name
}

// GetHAManufacturer returns the HA manufacturer with fallback chain:
// 1. homeassistant.manufacturer (if specified)
// 2. metadata.manufacturer (if specified)
// 3. "Unknown" (default)
func (d *Device) GetHAManufacturer() string {
	if d.HomeAssistant != nil && d.HomeAssistant.Manufacturer != "" {
		return d.HomeAssistant.Manufacturer
	}
	if d.Metadata.Manufacturer != "" {
		return d.Metadata.Manufacturer
	}
	return "Unknown"
}

// GetHAModel returns the HA model with fallback chain:
// 1. homeassistant.model (if specified)
// 2. metadata.model (if specified)
// 3. "Modbus Device" (default)
func (d *Device) GetHAModel() string {
	if d.HomeAssistant != nil && d.HomeAssistant.Model != "" {
		return d.HomeAssistant.Model
	}
	if d.Metadata.Model != "" {
		return d.Metadata.Model
	}
	return "Modbus Device"
}

// GetHADeviceID returns the HA device ID with fallback chain:
// 1. homeassistant.device_id (if specified)
// 2. deviceKey (the unique device identifier from config)
func (d *Device) GetHADeviceID(deviceKey string) string {
	if d.HomeAssistant != nil && d.HomeAssistant.DeviceID != "" {
		return d.HomeAssistant.DeviceID
	}
	return deviceKey
}

// Validate validates the device configuration
func (d *Device) Validate() error {
	// Validate metadata
	if d.Metadata.Name == "" {
		return fmt.Errorf("device metadata.name is required")
	}

	// Validate RTU configuration
	if d.RTU.SlaveID == 0 {
		return fmt.Errorf("device '%s' has invalid rtu.slave_id (must be 1-247)", d.Metadata.Name)
	}
	if d.RTU.SlaveID > 247 {
		return fmt.Errorf("device '%s' has rtu.slave_id %d (max is 247)", d.Metadata.Name, d.RTU.SlaveID)
	}

	// Validate Modbus configuration
	if len(d.Modbus.RegisterGroups) == 0 {
		return fmt.Errorf("device '%s' has no modbus.register_groups", d.Metadata.Name)
	}

	// Track register keys across all groups to ensure uniqueness
	usedRegisterKeys := make(map[string]string) // register key -> group name

	// Validate each register group
	for groupName, group := range d.Modbus.RegisterGroups {
		// Set group name if not provided
		if group.Name == "" {
			group.Name = groupName
			d.Modbus.RegisterGroups[groupName] = group
		}

		// Inherit slave_id from device RTU config if group doesn't have one
		if group.SlaveID == 0 {
			group.SlaveID = d.RTU.SlaveID
			d.Modbus.RegisterGroups[groupName] = group
		}

		// Validate the group
		if err := group.Validate(); err != nil {
			return fmt.Errorf("device '%s', group '%s': %w", d.Metadata.Name, groupName, err)
		}

		// Check for duplicate register keys within this device
		for _, reg := range group.Registers {
			if reg.Key == "" {
				return fmt.Errorf("device '%s', group '%s': register key cannot be empty", d.Metadata.Name, groupName)
			}
			if existingGroup, exists := usedRegisterKeys[reg.Key]; exists {
				return fmt.Errorf("device '%s': duplicate register key '%s' found in groups '%s' and '%s'",
					d.Metadata.Name, reg.Key, existingGroup, groupName)
			}
			usedRegisterKeys[reg.Key] = groupName
		}
	}

	// Validate calculated values
	for i, calc := range d.CalculatedValues {
		// Validate key
		if calc.Key == "" {
			return fmt.Errorf("device '%s': calculated_values[%d] key cannot be empty", d.Metadata.Name, i)
		}

		// Check for duplicate keys (calculated value vs register keys)
		if existingGroup, exists := usedRegisterKeys[calc.Key]; exists {
			return fmt.Errorf("device '%s': calculated value key '%s' conflicts with register in group '%s'",
				d.Metadata.Name, calc.Key, existingGroup)
		}

		// Validate formula
		if calc.Formula == "" {
			return fmt.Errorf("device '%s': calculated value '%s' has no formula", d.Metadata.Name, calc.Key)
		}

		// Validate formula syntax and extract variables
		variables, err := ValidateFormula(calc.Formula)
		if err != nil {
			return fmt.Errorf("device '%s': calculated value '%s' has invalid formula: %w", d.Metadata.Name, calc.Key, err)
		}

		// Validate that all variables exist in this device's registers
		for _, varName := range variables {
			if _, exists := usedRegisterKeys[varName]; !exists {
				return fmt.Errorf("device '%s': calculated value '%s' references unknown register '%s' in formula",
					d.Metadata.Name, calc.Key, varName)
			}
		}

		// Mark this calculated value key as used
		usedRegisterKeys[calc.Key] = "calculated_values"
	}

	return nil
}

// ValidateDevices validates all devices and checks for conflicts
func ValidateDevices(devices map[string]Device) error {
	if len(devices) == 0 {
		return fmt.Errorf("at least one device is required")
	}

	// Track used slave IDs, device keys, and HA device IDs to detect conflicts
	usedSlaveIDs := make(map[uint8]string)
	usedHADeviceIDs := make(map[string]string) // HA device ID -> device key
	deviceKeys := make([]string, 0, len(devices))

	for deviceKey, device := range devices {
		// Validate device key is not empty
		if deviceKey == "" {
			return fmt.Errorf("device key cannot be empty")
		}
		deviceKeys = append(deviceKeys, deviceKey)

		if err := device.Validate(); err != nil {
			return fmt.Errorf("device '%s': %w", deviceKey, err)
		}

		// Check for duplicate slave IDs
		if existingDevice, exists := usedSlaveIDs[device.RTU.SlaveID]; exists {
			return fmt.Errorf("duplicate rtu.slave_id %d: used by both '%s' and '%s'",
				device.RTU.SlaveID, existingDevice, device.Metadata.Name)
		}
		usedSlaveIDs[device.RTU.SlaveID] = device.Metadata.Name

		// Get the effective HA device ID (explicit or fallback to device key)
		haDeviceID := device.GetHADeviceID(deviceKey)

		// Check for duplicate HA device IDs
		if existingDeviceKey, exists := usedHADeviceIDs[haDeviceID]; exists {
			return fmt.Errorf("duplicate Home Assistant device_id '%s': used by both device keys '%s' and '%s'",
				haDeviceID, existingDeviceKey, deviceKey)
		}
		usedHADeviceIDs[haDeviceID] = deviceKey
	}

	logger.LogDebug("✅ Validated %d devices with unique keys: %v", len(devices), deviceKeys)

	return nil
}

// ConvertDevicesToGroups converts device-based config (V2.1) to flat groups (V2.0) for backward compatibility
func ConvertDevicesToGroups(devices map[string]Device) map[string]RegisterGroup {
	groups := make(map[string]RegisterGroup)

	for deviceKey, device := range devices {
		if !device.Metadata.Enabled {
			logger.LogDebug("Skipping disabled device: %s", device.Metadata.Name)
			continue
		}

		// Get the HA device ID for this device
		haDeviceID := device.GetHADeviceID(deviceKey)

		for groupName, group := range device.Modbus.RegisterGroups {
			// Create a unique key: deviceKey_groupName
			uniqueKey := fmt.Sprintf("%s_%s", deviceKey, groupName)

			// Ensure group has device's slave_id from RTU config
			if group.SlaveID == 0 {
				group.SlaveID = device.RTU.SlaveID
			}

			// Ensure group has a name
			if group.Name == "" {
				group.Name = fmt.Sprintf("%s - %s", device.Metadata.Name, groupName)
			}

			// Construct HATopic for each register if not provided
			for i := range group.Registers {
				if group.Registers[i].HATopic == "" {
					// Auto-construct topic (discovery prefix is set in topics package)
					group.Registers[i].HATopic = topics.ConstructHATopic(haDeviceID, group.Registers[i].Key, group.Registers[i].DeviceClass)
					logger.LogDebug("Auto-constructed topic for %s/%s: %s",
						deviceKey, group.Registers[i].Key, group.Registers[i].HATopic)
				}
			}

			groups[uniqueKey] = group

			logger.LogDebug("Converted device '%s' group '%s' -> '%s' (slave_id: %d)",
				device.Metadata.Name, groupName, uniqueKey, group.SlaveID)
		}
	}

	return groups
}

// GetAllRegisters extracts all registers from all devices for backward compatibility
// Creates unique keys by combining device key with register key (deviceKey_registerKey)
func GetAllRegistersFromDevices(devices map[string]Device) map[string]Register {
	registers := make(map[string]Register)

	for deviceKey, device := range devices {
		if !device.Metadata.Enabled {
			logger.LogDebug("Skipping disabled device: %s", device.Metadata.Name)
			continue
		}

		// Get the HA device ID for this device
		haDeviceID := device.GetHADeviceID(deviceKey)

		// Convert each register group
		for _, group := range device.Modbus.RegisterGroups {
			// Ensure group has device's slave_id from RTU config
			if group.SlaveID == 0 {
				group.SlaveID = device.RTU.SlaveID
			}

			for _, reg := range group.Registers {
				// Calculate actual address from group start + offset
				offsetInRegisters := reg.Offset / 2

				// Validate that the address calculation won't overflow uint16
				if offsetInRegisters < 0 || offsetInRegisters > 0xFFFF {
					continue
				}

				// Calculate final address and check for overflow
				finalAddress := int(group.StartAddress) + offsetInRegisters
				if finalAddress > 0xFFFF {
					continue
				}

				// Safe conversion after validation
				// #nosec G115 -- Validated above that offsetInRegisters fits in uint16
				address := group.StartAddress + uint16(offsetInRegisters)

				// Construct HATopic if not provided
				haTopic := reg.HATopic
				if haTopic == "" {
					haTopic = topics.ConstructHATopic(haDeviceID, reg.Key, reg.DeviceClass)
				} // Create pointers for optional fields (only if non-zero)
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

				// Default scale_factor to 1.0 if not specified
				scaleFactor := reg.ScaleFactor
				if scaleFactor == 0 {
					scaleFactor = 1.0
				}

				// Create unique key: deviceKey_registerKey
				uniqueKey := fmt.Sprintf("%s_%s", deviceKey, reg.Key)

				registers[uniqueKey] = Register{
					Name:          reg.Name,
					Address:       address,
					Unit:          reg.Unit,
					ScaleFactor:   scaleFactor,
					ApplyAbs:      reg.ApplyAbs, // Copy apply_abs flag
					Formula:       reg.Formula,
					DependsOn:     reg.DependsOn,
					DeviceClass:   reg.DeviceClass,
					StateClass:    reg.StateClass,
					HATopic:       haTopic,
					Min:           minPtr,
					Max:           maxPtr,
					MaxKwhPerHour: maxKwhPtr,
				}

				logger.LogDebug("Converted device '%s' register '%s' -> '%s' (address: 0x%04X, topic: %s)",
					deviceKey, reg.Key, uniqueKey, address, haTopic)
			}
		}

		// Process calculated values for this device
		for _, calc := range device.CalculatedValues {
			// Construct HATopic (discovery prefix is set in topics package)
			haTopic := topics.ConstructHATopic(haDeviceID, calc.Key, calc.DeviceClass) // Create pointers for optional fields
			var minPtr, maxPtr *float64
			if calc.Min != nil {
				minPtr = calc.Min
			}
			if calc.Max != nil {
				maxPtr = calc.Max
			}

			// Default scale_factor to 1.0 if not specified
			scaleFactor := calc.ScaleFactor
			if scaleFactor == 0 {
				scaleFactor = 1.0
			}

			// Create unique key: deviceKey_registerKey
			uniqueKey := fmt.Sprintf("%s_%s", deviceKey, calc.Key)

			registers[uniqueKey] = Register{
				Name:        calc.Name,
				Address:     0, // Calculated values have no Modbus address
				Unit:        calc.Unit,
				ScaleFactor: scaleFactor,
				Formula:     calc.Formula,
				DependsOn:   []string{}, // Will be extracted from formula
				DeviceClass: calc.DeviceClass,
				StateClass:  calc.StateClass,
				HATopic:     haTopic,
				Min:         minPtr,
				Max:         maxPtr,
			}

			logger.LogDebug("Converted device '%s' calculated value '%s' -> '%s' (formula: %s, topic: %s)",
				deviceKey, calc.Key, uniqueKey, calc.Formula, haTopic)
		}
	}

	return registers
}
