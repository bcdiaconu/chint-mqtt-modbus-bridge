package config

import (
	"fmt"
	"mqtt-modbus-bridge/pkg/logger"
)

// Device represents a Modbus device on the RTU bus
// Each device has a unique slave ID and its own register groups
// Used in configuration version 2.1+
type Device struct {
	Name           string                   `yaml:"name"`                    // Display name (e.g., "Energy Meter 1")
	SlaveID        uint8                    `yaml:"slave_id"`                // Modbus device ID (1-247)
	Manufacturer   string                   `yaml:"manufacturer,omitempty"`  // Device manufacturer
	Model          string                   `yaml:"model,omitempty"`         // Device model
	PollInterval   int                      `yaml:"poll_interval,omitempty"` // Override global poll interval (ms)
	Enabled        bool                     `yaml:"enabled"`                 // Enable/disable this device
	RegisterGroups map[string]RegisterGroup `yaml:"register_groups"`         // Groups for this device (instant, energy, status, etc.)
}

// Validate validates the device configuration
func (d *Device) Validate() error {
	if d.Name == "" {
		return fmt.Errorf("device name is required")
	}
	if d.SlaveID == 0 {
		return fmt.Errorf("device '%s' has invalid slave_id (must be 1-247)", d.Name)
	}
	if d.SlaveID > 247 {
		return fmt.Errorf("device '%s' has slave_id %d (max is 247)", d.Name, d.SlaveID)
	}
	if len(d.RegisterGroups) == 0 {
		return fmt.Errorf("device '%s' has no register groups", d.Name)
	}

	// Validate each register group
	for groupName, group := range d.RegisterGroups {
		// Set group name if not provided
		if group.Name == "" {
			group.Name = groupName
			d.RegisterGroups[groupName] = group
		}

		// Inherit slave_id from device if group doesn't have one
		if group.SlaveID == 0 {
			group.SlaveID = d.SlaveID
			d.RegisterGroups[groupName] = group
		}

		// Validate the group
		if err := group.Validate(); err != nil {
			return fmt.Errorf("device '%s', group '%s': %w", d.Name, groupName, err)
		}
	}

	return nil
}

// GetPollInterval returns the device-specific poll interval or the default
func (d *Device) GetPollInterval(defaultInterval int) int {
	if d.PollInterval > 0 {
		return d.PollInterval
	}
	return defaultInterval
}

// ValidateDevices validates all devices and checks for conflicts
func ValidateDevices(devices map[string]Device) error {
	if len(devices) == 0 {
		return fmt.Errorf("at least one device is required")
	}

	// Track used slave IDs to detect conflicts
	usedSlaveIDs := make(map[uint8]string)

	for deviceKey, device := range devices {
		if err := device.Validate(); err != nil {
			return fmt.Errorf("device '%s': %w", deviceKey, err)
		}

		// Check for duplicate slave IDs
		if existingDevice, exists := usedSlaveIDs[device.SlaveID]; exists {
			return fmt.Errorf("duplicate slave_id %d: used by both '%s' and '%s'",
				device.SlaveID, existingDevice, device.Name)
		}
		usedSlaveIDs[device.SlaveID] = device.Name
	}

	return nil
}

// ConvertDevicesToGroups converts device-based config (V2.1) to flat groups (V2.0) for backward compatibility
func ConvertDevicesToGroups(devices map[string]Device) map[string]RegisterGroup {
	groups := make(map[string]RegisterGroup)

	for deviceKey, device := range devices {
		if !device.Enabled {
			logger.LogDebug("Skipping disabled device: %s", device.Name)
			continue
		}

		for groupName, group := range device.RegisterGroups {
			// Create a unique key: deviceKey_groupName
			uniqueKey := fmt.Sprintf("%s_%s", deviceKey, groupName)

			// Ensure group has device's slave_id
			if group.SlaveID == 0 {
				group.SlaveID = device.SlaveID
			}

			// Ensure group has a name
			if group.Name == "" {
				group.Name = fmt.Sprintf("%s - %s", device.Name, groupName)
			}

			groups[uniqueKey] = group

			logger.LogDebug("Converted device '%s' group '%s' -> '%s' (slave_id: %d)",
				device.Name, groupName, uniqueKey, group.SlaveID)
		}
	}

	return groups
}

// GetAllRegisters extracts all registers from all devices for backward compatibility
func GetAllRegistersFromDevices(devices map[string]Device) map[string]Register {
	allGroups := ConvertDevicesToGroups(devices)
	return ConvertGroupsToRegisters(allGroups)
}
