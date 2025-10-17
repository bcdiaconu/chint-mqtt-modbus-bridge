# Multi-Device Support (V2.1)

## Overview

Configuration V2.1 introduces **device-based organization** for managing multiple Modbus devices on the same RTU bus. Each device has its own segregated configuration sections for metadata, RTU settings, Modbus registers, and Home Assistant integration.

## Device-Based Structure

### Four Segregated Sections

Each device in the `devices` map has four distinct configuration sections:

```yaml
devices:
  device_key:              # Unique device identifier (device key)
    metadata:              # Device identification and metadata
    rtu:                   # RTU/Physical layer configuration  
    modbus:                # Modbus protocol configuration
    homeassistant:         # Home Assistant integration (optional)
```

### 1. Metadata Section

Device identification and general information:

```yaml
metadata:
  name: "Energy Meter 1"          # Required: Human-readable device name
  manufacturer: "Chint"            # Optional: Device manufacturer
  model: "DTSU666-H"               # Optional: Device model
  enabled: true                    # Optional: Enable/disable device (default: true)
```

**Purpose**: Provides human-readable information and basic device management.

### 2. RTU Section

RTU/Physical layer configuration:

```yaml
rtu:
  slave_id: 11                     # Required: Modbus RTU slave ID (1-247, must be unique)
  poll_interval: 1000              # Optional: Override global poll_interval (milliseconds)
```

**Purpose**: Defines the physical Modbus RTU parameters for device communication.

### 3. Modbus Section

Modbus protocol configuration with register groups:

```yaml
modbus:
  register_groups:
    instant:
      name: "Instant Measurements"
      function_code: 0x03          # Read Holding Registers
      start_address: 0x2000
      register_count: 34
      enabled: true
      registers:
        - key: "voltage"
          name: "Voltage"
          offset: 0
          unit: "V"
          scale: 0.1
```

**Purpose**: Defines what data to read from the device and how to parse it.

### 4. Home Assistant Section (Optional)

Home Assistant integration settings:

```yaml
homeassistant:
  device_id: "chint_meter_1"       # Optional: HA device ID (defaults to device key)
  manufacturer: "Chint Electric"    # Optional: Override metadata.manufacturer
  model: "DTSU666-H Three-Phase"   # Optional: Override metadata.model
```

**Purpose**: Customizes Home Assistant device appearance. All fields are optional.

## Device Keys and Uniqueness

### Device Key

The device key is the unique identifier in the `devices` map:

```yaml
devices:
  energy_meter_1:        # ← Device key (must be unique)
    metadata:
      name: "Energy Meter 1"
```

**Requirements:**

- Must be unique across all devices
- Used as default Home Assistant device ID if `homeassistant.device_id` is not specified
- Cannot be changed after Home Assistant discovery (requires deleting and recreating device)

### Unique Identifiers Validation

The configuration validates three types of uniqueness:

1. **Device Keys**: Automatically unique (enforced by YAML map structure)
2. **RTU Slave IDs**: Must be unique across all devices (validated by `ValidateDevices()`)
3. **Home Assistant Device IDs**: Must be unique including fallbacks (validated by `ValidateDevices()`)

Example validation scenarios:

```yaml
# ✅ Valid: All unique
devices:
  meter_1:
    rtu:
      slave_id: 11
    homeassistant:
      device_id: "ha_meter_1"
      
  meter_2:
    rtu:
      slave_id: 12
    homeassistant:
      device_id: "ha_meter_2"

# ❌ Invalid: Duplicate slave_id
devices:
  meter_1:
    rtu:
      slave_id: 11        # ← Duplicate!
  meter_2:
    rtu:
      slave_id: 11        # ← Duplicate!

# ❌ Invalid: Duplicate HA device_id (explicit)
devices:
  meter_1:
    homeassistant:
      device_id: "meter"  # ← Duplicate!
  meter_2:
    homeassistant:
      device_id: "meter"  # ← Duplicate!

# ❌ Invalid: Duplicate HA device_id (device key conflicts with explicit)
devices:
  meter_1:               # ← Fallback to "meter_1"
    # No homeassistant section
  meter_2:
    homeassistant:
      device_id: "meter_1"  # ← Conflicts with meter_1's fallback!
```

## Home Assistant Device ID Fallback

### How It Works

The Home Assistant device ID follows this fallback chain:

```md
homeassistant.device_id (explicit)
    ↓ if not set
device_key (automatic fallback)
```

### Examples

```yaml
# Example 1: Explicit device_id
devices:
  energy_meter_1:
    homeassistant:
      device_id: "chint_meter_1"    # HA device_id = "chint_meter_1"

# Example 2: Fallback to device key
devices:
  energy_meter_2:
    # No homeassistant section         # HA device_id = "energy_meter_2"

# Example 3: Empty homeassistant section
devices:
  energy_meter_3:
    homeassistant:
      manufacturer: "Chint"           # HA device_id = "energy_meter_3"
      # device_id not set - fallback
```

### Code Implementation

```go
// GetHADeviceID returns the Home Assistant device ID with fallback to device key
func (d *Device) GetHADeviceID(deviceKey string) string {
    if d.HomeAssistant != nil && d.HomeAssistant.DeviceID != "" {
        return d.HomeAssistant.DeviceID
    }
    return deviceKey  // Fallback to device key
}
```

## Multiple Devices Example

### Complete Configuration

```yaml
version: "2.1"

mqtt:
  broker: "192.168.1.100:1883"
  username: "mqtt_user"
  password: "mqtt_password"
  
global:
  poll_interval: 5000        # Default for all devices

devices:
  # First Energy Meter
  energy_meter_1:
    metadata:
      name: "Energy Meter 1"
      manufacturer: "Chint"
      model: "DTSU666-H"
      enabled: true
    
    rtu:
      slave_id: 11           # Unique slave ID
      poll_interval: 1000    # Override global interval
    
    homeassistant:
      device_id: "chint_meter_1"
      manufacturer: "Chint Electric"
      model: "DTSU666-H Three-Phase"
    
    modbus:
      register_groups:
        instant:
          name: "Instant Measurements"
          function_code: 0x03
          start_address: 0x2000
          register_count: 34
          enabled: true
          registers:
            - key: "voltage"
              name: "Voltage"
              offset: 0
              unit: "V"
              scale: 0.1

  # Second Energy Meter
  energy_meter_2:
    metadata:
      name: "Energy Meter 2"
      manufacturer: "Chint"
      model: "DTSU666-H"
      enabled: true
    
    rtu:
      slave_id: 12           # Different slave ID
      # Uses global poll_interval
    
    # No homeassistant section - uses device key "energy_meter_2"
    
    modbus:
      register_groups:
        instant:
          name: "Instant Measurements"
          function_code: 0x03
          start_address: 0x2000
          register_count: 34
          enabled: true
          registers:
            - key: "voltage"
              name: "Voltage"
              offset: 0
              unit: "V"
              scale: 0.1
```

### Result in Home Assistant

This configuration creates two separate devices in Home Assistant:

1. **Device: "chint_meter_1"**
   - Name: "Energy Meter 1"
   - Manufacturer: "Chint Electric"
   - Model: "DTSU666-H Three-Phase"
   - Sensors: voltage, current, power, etc. (all from slave_id 11)

2. **Device: "energy_meter_2"**
   - Name: "Energy Meter 2"
   - Manufacturer: "Chint"
   - Model: "DTSU666-H"
   - Sensors: voltage, current, power, etc. (all from slave_id 12)

## Getter Methods

The `Device` struct provides getter methods with automatic fallbacks:

```go
// Device identification
func (d *Device) GetName() string
func (d *Device) GetHADeviceID(deviceKey string) string

// RTU configuration
func (d *Device) GetSlaveID() int
func (d *Device) GetPollInterval() int

// Home Assistant integration
func (d *Device) GetHAManufacturer() string  // homeassistant.manufacturer → metadata.manufacturer
func (d *Device) GetHAModel() string         // homeassistant.model → metadata.model
func (d *Device) IsEnabled() bool            // metadata.enabled (default: true)
```

### Fallback Chains

1. **HA Device ID**:

   ```md
   homeassistant.device_id → device_key
   ```

2. **HA Manufacturer**:

   ```md
   homeassistant.manufacturer → metadata.manufacturer → ""
   ```

3. **HA Model**:

   ```md
   homeassistant.model → metadata.model → ""
   ```

## Benefits

### Scalability

- Add unlimited devices without code changes
- Each device operates independently
- No interference between devices

### Organization

- Clear separation of concerns (metadata, RTU, Modbus, HA)
- Easy to understand and maintain
- Consistent structure across all devices

### Flexibility

- Per-device poll intervals
- Optional Home Assistant integration
- Custom manufacturer/model overrides
- Enable/disable individual devices

### Validation

- Automatic uniqueness checks
- Clear error messages
- Prevents configuration conflicts

## Migration from Single Device

See [Migration Guide](MIGRATION.md#single-device-to-multi-device) for step-by-step instructions.

## See Also

- [Configuration Reference](CONFIG.md) - Complete configuration documentation
- [Validation Rules](VALIDATION.md) - Detailed validation rules and examples
- [Migration Guide](MIGRATION.md) - Upgrading from V1 or V2.0

## License

This project is licensed under the MIT License.
