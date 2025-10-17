# Configuration V2 - Group-Based Modbus Commands

## Overview

Configuration V2 introduces a **group-based approach** to Modbus register reading. Version 2.1 adds **device-based organization** for multi-device setups.

### Version History

- **V2.0**: Group-based register organization with explicit Modbus commands
- **V2.1**: Multi-device support with segregated configuration sections

## Benefits

### ✅ Explicit Command Definition

- **Before (V1)**: Individual addresses, implicit grouping
- **After (V2.0)**: Explicit groups with `slave_id`, `function_code`, `start_address`, `register_count`
- **After (V2.1)**: Device-based organization with metadata, RTU, Modbus, and Home Assistant sections

### ✅ Multi-Device Support (V2.1+)

- Multiple devices on the same RTU bus with different slave IDs
- Each device has unique configuration and Home Assistant integration
- Automatic validation of device keys and Home Assistant device IDs

### ✅ Automatic CRC Calculation  

Each group maps directly to `modbus.BuildModbusCommand(slave_id, function_code, start_address, register_count)` with automatic CRC calculation.

### ✅ Clear Register Layout

Offsets are explicit in bytes from group start, making the memory layout crystal clear.

### ✅ Easy Maintenance

Changing a command? Just update the group parameters, CRC recalculates automatically!

## Configuration Structure

### Old Format (V1)

```yaml
registers:
  voltage:
    name: "Voltage"
    address: 0x2000  # Where is it in the Modbus command?
    # How many registers to read?
    # What's the command structure?
```

### New Format (V2)

```yaml
register_groups:
  instant:
    name: "Instant Measurements"
    slave_id: 11
    function_code: 0x03          # Read Holding Registers
    start_address: 0x2000
    register_count: 34           # Explicit!
    enabled: true
    registers:
      - key: "voltage"
        name: "Voltage"
        offset: 0                # 0 bytes from start (0x2000)
        unit: "V"
        
      - key: "current"  
        name: "Current"
        offset: 4                # 4 bytes from start (0x2002)
        unit: "A"
        
      - key: "frequency"
        name: "Frequency"
        offset: 64               # 64 bytes from start (0x2020)
        unit: "Hz"
```

### Device-Based Format (V2.1) - Multiple Devices

```yaml
version: "2.1"

devices:
  # First Energy Meter - Device Key: energy_meter_1
  energy_meter_1:
    # Device identification and metadata
    metadata:
      name: "Energy Meter 1"
      manufacturer: "Chint"
      model: "DTSU666-H"
      enabled: true
    
    # RTU/Physical layer configuration
    rtu:
      slave_id: 11              # Modbus RTU slave ID (1-247, must be unique)
      poll_interval: 1000       # Optional: override global poll_interval
    
    # Home Assistant integration (optional)
    homeassistant:
      device_id: "chint_meter_1"         # Optional: defaults to device key
      manufacturer: "Chint Electric"      # Optional: override metadata.manufacturer
      model: "DTSU666-H Three-Phase"     # Optional: override metadata.model
    
    # Modbus protocol configuration
    modbus:
      register_groups:
        instant:
          name: "Instant Readings"
          function_code: 0x03
          start_address: 0x2000
          register_count: 34
          enabled: true
          registers:
            - key: "voltage"
              name: "Voltage"
              offset: 0
              unit: "V"
              device_class: "voltage"
              state_class: "measurement"
              ha_topic: "meter1/voltage"
  
  # Second Energy Meter - Device Key: energy_meter_2
  energy_meter_2:
    metadata:
      name: "Energy Meter 2"
      manufacturer: "Chint"
      model: "DTSU666-H"
      enabled: true
    rtu:
      slave_id: 12              # Different slave ID!
    # homeassistant section omitted - will use "energy_meter_2" as device_id
    modbus:
      register_groups:
        instant:
          name: "Instant Readings"
          function_code: 0x03
          start_address: 0x2000
          register_count: 34
          enabled: true
          registers:
            - key: "voltage"
              name: "Voltage"
              offset: 0
              unit: "V"
```

## V2.1 Device Configuration Structure

### Four Segregated Sections

Each device in V2.1 is organized into **four distinct sections**:

#### 1. **metadata** - Device Identification

```yaml
metadata:
  name: "Energy Meter 1"        # Required: Display name
  manufacturer: "Chint"          # Optional: Device manufacturer
  model: "DTSU666-H"            # Optional: Device model
  enabled: true                  # Optional: Enable/disable device (default: true)
```

#### 2. **rtu** - RTU/Physical Layer

```yaml
rtu:
  slave_id: 11                   # Required: Modbus slave ID (1-247, must be unique)
  poll_interval: 1000            # Optional: Override global poll_interval (milliseconds)
```

#### 3. **homeassistant** - Home Assistant Integration (Optional)

```yaml
homeassistant:
  device_id: "chint_meter_001"   # Optional: HA device ID (defaults to device key)
  manufacturer: "Chint Electric" # Optional: Override metadata.manufacturer
  model: "DTSU666-H Pro"        # Optional: Override metadata.model
```

**Fallback Chain**:

- `device_id`: Defaults to device key (e.g., `energy_meter_1`)
- `manufacturer`: Falls back to `metadata.manufacturer` → "Unknown"
- `model`: Falls back to `metadata.model` → "Modbus Device"

#### 4. **modbus** - Modbus Protocol Configuration

```yaml
modbus:
  register_groups:
    instant:                     # Group name (can be anything: instant, energy, status, etc.)
      name: "Instant Readings"
      function_code: 0x03
      start_address: 0x2000
      register_count: 34
      enabled: true
      registers:
        - key: "voltage"
          name: "Voltage"
          offset: 0
          unit: "V"
```

### Device Keys and Uniqueness

#### Device Key

The **device key** is the unique identifier in the `devices:` map:

```yaml
devices:
  energy_meter_1:      # ← This is the device key (must be unique)
    metadata:
      name: "Energy Meter 1"
```

**Rules**:

- ✅ Must be unique across all devices
- ✅ Used as default `homeassistant.device_id` if not specified
- ✅ Used internally for logging and debugging
- ✅ Should be descriptive (e.g., `energy_meter_1`, `solar_inverter`, `heat_pump_main`)

#### Validation Rules

The system validates the following uniqueness constraints:

1. **Device Keys** - Must be unique (automatically enforced by YAML map structure)
2. **RTU Slave IDs** - Each device must have a unique `rtu.slave_id` (1-247)
3. **Home Assistant Device IDs** - The effective `device_id` must be unique across all devices

**Example of Invalid Configuration**:

```yaml
devices:
  meter_1:
    rtu:
      slave_id: 11
    # No homeassistant config - device_id = "meter_1"
  
  meter_2:
    rtu:
      slave_id: 12
    homeassistant:
      device_id: "meter_1"    # ❌ ERROR: Conflicts with meter_1's device key!
```

**Error Message**:

```log
duplicate Home Assistant device_id 'meter_1': used by both device keys 'meter_1' and 'meter_2'
```

#### Valid Examples

- **Example 1: All Explicit IDs**

```yaml
devices:
  energy_meter_1:
    homeassistant:
      device_id: "chint_meter_001"    # Explicit, unique
  energy_meter_2:
    homeassistant:
      device_id: "chint_meter_002"    # Explicit, unique
```

- **Example 2: Mixed Explicit and Default**

```yaml
devices:
  energy_meter_1:
    homeassistant:
      device_id: "custom_meter_id"    # Explicit
  energy_meter_2:
    # No homeassistant - uses "energy_meter_2" as device_id
```

- **Example 3: All Default IDs**

```yaml
devices:
  energy_meter_1:
    # Uses "energy_meter_1" as device_id
  energy_meter_2:
    # Uses "energy_meter_2" as device_id
  solar_inverter:
    # Uses "solar_inverter" as device_id
```

## How It Works

### 1. Group Definition

A register group defines a **single Modbus command**:

```yaml
instant:
  slave_id: 11              # Device ID
  function_code: 0x03       # Read Holding Registers
  start_address: 0x2000     # First register
  register_count: 34        # Number of 16-bit registers (0x22)
  enabled: true
```

This maps to:

```go
command := modbus.BuildModbusCommand(0x0B, 0x03, 0x2000, 0x0022)
// Result: 0B0320000022CEB9 (CRC calculated automatically!)
```

### 2. Register Offsets

Each register specifies its **byte offset** from the group start:

```yaml
registers:
  - key: "voltage"
    offset: 0        # Bytes 0-3 in response
    
  - key: "current"
    offset: 4        # Bytes 4-7 in response
    
  - key: "frequency"
    offset: 64       # Bytes 64-67 in response
```

### 3. Command Generation

The system automatically generates the Modbus command:

```go
// From configuration
group := config.RegisterGroups["instant"]

// Generate command with automatic CRC
command := modbus.BuildModbusCommand(
    group.SlaveID,        // 0x0B
    group.FunctionCode,   // 0x03
    group.StartAddress,   // 0x2000
    group.RegisterCount,  // 0x0022
)

// Send to gateway
gateway.SendCommand(ctx, command)
```

### 4. Response Parsing

Parse using offsets:

```go
// Response: 68 bytes of register data
response := []byte{...}

for _, reg := range group.Registers {
    // Extract 4 bytes at offset
    data := response[reg.Offset : reg.Offset+4]
    value := parseFloat32(data)
    results[reg.Key] = value
}
```

## Example Configuration

### Complete V2 Configuration

```yaml
modbus:
  slave_id: 11
  function_code: 0x03
  poll_interval: 1000
  timeout: 5

register_groups:
  # Instant measurements: voltage, current, power, frequency
  instant:
    name: "Instant Measurements"
    slave_id: 11
    function_code: 0x03
    start_address: 0x2000
    register_count: 34           # 0x2000 to 0x2021 (34 registers)
    enabled: true
    registers:
      - key: "voltage"
        name: "Voltage"
        offset: 0                # 0x2000
        unit: "V"
        device_class: "voltage"
        min: 100.0
        max: 300.0
        
      - key: "current"
        name: "Current"
        offset: 4                # 0x2002
        unit: "A"
        device_class: "current"
        min: 0.0
        max: 100.0
        
      - key: "power_active"
        name: "Active Power"
        offset: 12               # 0x2006
        unit: "W"
        device_class: "power"
        
      - key: "power_apparent"
        name: "Apparent Power"
        offset: 36               # 0x2012
        unit: "VA"
        device_class: "apparent_power"
        
      - key: "power_factor"
        name: "Power Factor"
        offset: 48               # 0x2018
        unit: ""
        device_class: "power_factor"
        
      - key: "frequency"
        name: "Frequency"
        offset: 64               # 0x2020
        unit: "Hz"
        device_class: "frequency"

  # Energy counters: kWh meters
  energy:
    name: "Energy Counters"
    slave_id: 11
    function_code: 0x03
    start_address: 0x4000
    register_count: 22           # 0x4000 to 0x4015 (22 registers)
    enabled: true
    registers:
      - key: "energy_total"
        name: "Active Energy"
        offset: 0                # 0x4000
        unit: "kWh"
        device_class: "energy"
        
      - key: "energy_imported"
        name: "Imported Energy"
        offset: 20               # 0x400A
        unit: "kWh"
        device_class: "energy"
        
      - key: "energy_exported"
        name: "Exported Energy"
        offset: 40               # 0x4014
        unit: "kWh"
        device_class: "energy"

# Calculated/virtual registers
calculated_registers:
  power_reactive:
    name: "Reactive Power"
    unit: "var"
    device_class: "reactive_power"
    formula: "sqrt(power_apparent^2 - power_active^2)"
    depends_on:
      - "power_apparent"
      - "power_active"
```

## Generated Commands

From the above configuration, these commands are generated:

### Instant Group Command

```md
Command: 0B 03 2000 0022 CE B9
         ││ │  │    │    └──┴─ CRC-16 (auto-calculated)
         ││ │  │    └───────── Count: 34 registers (0x0022)
         ││ │  └────────────── Start: 0x2000
         ││ └───────────────── Function: Read Holding Registers (0x03)
         │└─────────────────── Slave ID: 11 (0x0B)
         └──────────────────── Device address
```

**Hex**: `0B0320000022CEB9`

### Energy Group Command

```md
Command: 0B 03 4000 0016 D1 6E
         ││ │  │    │    └──┴─ CRC-16 (auto-calculated)
         ││ │  │    └───────── Count: 22 registers (0x0016)
         ││ │  └────────────── Start: 0x4000
         ││ └───────────────── Function: Read Holding Registers (0x03)
         │└─────────────────── Slave ID: 11 (0x0B)
         └──────────────────── Device address
```

**Hex**: `0B0340000016D16E`

## Offset Calculation

### How to Calculate Offsets

Given Modbus addresses, calculate byte offsets:

```math
\text{offset} = (\text{address} - \text{group\_start\_address}) \times 2
```

**Example**: Instant group starting at 0x2000

| Register | Address | Calculation | Offset (bytes) |
|----------|---------|-------------|----------------|
| Voltage | 0x2000 | (0x2000 - 0x2000) × 2 | 0 |
| Current | 0x2002 | (0x2002 - 0x2000) × 2 | 4 |
| Power | 0x2006 | (0x2006 - 0x2000) × 2 | 12 |
| Apparent | 0x2012 | (0x2012 - 0x2000) × 2 | 36 |
| PF | 0x2018 | (0x2018 - 0x2000) × 2 | 48 |
| Frequency | 0x2020 | (0x2020 - 0x2000) × 2 | 64 |

## Migration Guide

### Migration from V2.0 to V2.1 (Multi-Device Support)

V2.1 introduces device-based organization for multi-device setups. If you have a single device, this is optional but recommended for future scalability.

#### Before (V2.0) - Single Device

```yaml
version: "2.0"

modbus:
  slave_id: 11
  poll_interval: 1000

register_groups:
  instant:
    name: "Instant Measurements"
    slave_id: 11
    function_code: 0x03
    start_address: 0x2000
    register_count: 34
    enabled: true
    registers:
      - key: "voltage"
        name: "Voltage"
        offset: 0
        unit: "V"
```

#### After (V2.1) - Device-Based

```yaml
version: "2.1"

modbus:
  poll_interval: 1000       # Global default

devices:
  energy_meter:             # Device key (choose descriptive name)
    metadata:
      name: "Energy Meter"
      manufacturer: "Chint"
      model: "DTSU666-H"
      enabled: true
    
    rtu:
      slave_id: 11          # Moved from global modbus config
    
    # homeassistant section optional - defaults to device key "energy_meter"
    
    modbus:
      register_groups:      # Same structure as V2.0
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
```

#### Migration Steps (V2.0 → V2.1)

1. **Update version**: Change `version: "2.0"` to `version: "2.1"`

2. **Create devices section**: Wrap your config in a device:

   ```yaml
   devices:
     my_device:    # Choose a descriptive device key
   ```

3. **Add metadata section**:

   ```yaml
   metadata:
     name: "My Device Name"
     manufacturer: "Device Manufacturer"
     model: "Device Model"
     enabled: true
   ```

4. **Move slave_id to rtu section**:

   ```yaml
   rtu:
     slave_id: 11    # From modbus.slave_id
   ```

5. **Wrap register_groups in modbus section**:

   ```yaml
   modbus:
     register_groups:
       # Your existing groups here
   ```

6. **(Optional) Add Home Assistant config**:

   ```yaml
   homeassistant:
     device_id: "custom_id"      # Optional: defaults to device key
     manufacturer: "Override"     # Optional: override metadata
     model: "Custom Model"        # Optional: override metadata
   ```

#### Adding More Devices (V2.1)

To add a second device on the same RTU bus:

```yaml
devices:
  energy_meter_1:
    metadata:
      name: "Energy Meter 1"
    rtu:
      slave_id: 11        # First device
    modbus:
      register_groups:
        # ... groups for meter 1
  
  energy_meter_2:
    metadata:
      name: "Energy Meter 2"
    rtu:
      slave_id: 12        # Second device - different slave_id!
    modbus:
      register_groups:
        # ... groups for meter 2 (can be identical or different)
```

**Important**: Each device must have:

- ✅ Unique device key (`energy_meter_1`, `energy_meter_2`)
- ✅ Unique `rtu.slave_id` (1-247)
- ✅ Unique effective `homeassistant.device_id`

### Migration from V1

### Automatic Conversion

The system can automatically convert V2 config to V1 format:

```go
configV2 := LoadConfigV2("config-v2.yaml")
configV1 := configV2.ConvertToV1()
```

### Manual Migration Steps

1. **Identify register groups** - Group contiguous registers
2. **Calculate range** - Determine start address and count
3. **Calculate offsets** - Convert addresses to byte offsets
4. **Define groups** - Create register_groups section
5. **Test** - Verify commands match expected values

## Validation

The configuration validates:

### V2.0 Validation

- ✅ Slave ID is set
- ✅ Function code is set (defaults to 0x03)
- ✅ Register count is set
- ✅ Offsets are within range
- ✅ Register keys are unique
- ✅ Calculated register dependencies exist

### V2.1 Additional Validation

- ✅ **Device Keys**: Must be unique (automatically enforced by YAML map)
- ✅ **metadata.name**: Required for each device
- ✅ **rtu.slave_id**: Required, must be between 1-247, must be unique across all devices
- ✅ **modbus.register_groups**: At least one group required per device
- ✅ **Home Assistant device_id Uniqueness**: The effective `device_id` (explicit or defaulted to device key) must be unique across all devices

### Validation Examples

**Valid Configuration**:

```yaml
devices:
  meter_1:
    rtu:
      slave_id: 11
    # device_id defaults to "meter_1"
  
  meter_2:
    rtu:
      slave_id: 12
    homeassistant:
      device_id: "custom_meter_002"
```

✅ All device keys unique, all slave IDs unique, all device_ids unique

**Invalid - Duplicate slave_id**:

```yaml
devices:
  meter_1:
    rtu:
      slave_id: 11    # ❌
  meter_2:
    rtu:
      slave_id: 11    # ❌ ERROR: Duplicate!
```

❌ Error: `duplicate rtu.slave_id 11: used by both 'meter_1' and 'meter_2'`

**Invalid - Duplicate device_id**:

```yaml
devices:
  meter_1:
    # device_id defaults to "meter_1"
  
  meter_2:
    homeassistant:
      device_id: "meter_1"    # ❌ Conflicts with meter_1!
```

❌ Error: `duplicate Home Assistant device_id 'meter_1': used by both device keys 'meter_1' and 'meter_2'`

**Invalid - Missing required fields**:

```yaml
devices:
  meter_1:
    metadata:
      # name missing!    # ❌
    rtu:
      # slave_id missing!    # ❌
```

❌ Error: `device metadata.name is required`
❌ Error: `device has invalid rtu.slave_id (must be 1-247)`

## API Usage

### Loading Configuration

```go
import "mqtt-modbus-bridge/pkg/config"

// Load V2 configuration
cfg, err := config.LoadConfigV2("config-v2.yaml")
if err != nil {
    log.Fatal(err)
}

// Access groups
for name, group := range cfg.RegisterGroups {
    fmt.Printf("Group: %s\n", name)
    fmt.Printf("  Command will read %d registers from 0x%04X\n", 
        group.RegisterCount, group.StartAddress)
}
```

### Generating Commands

```go
import "mqtt-modbus-bridge/pkg/modbus"

// For each group, generate Modbus command
for name, group := range cfg.RegisterGroups {
    if !group.Enabled {
        continue
    }
    
    // Generate command with automatic CRC
    command := modbus.BuildModbusCommand(
        group.SlaveID,
        group.FunctionCode,
        group.StartAddress,
        group.RegisterCount,
    )
    
    fmt.Printf("%s: %02X\n", name, command)
}
```

Output:

```
instant: 0B0320000022CEB9
energy: 0B0340000016D16E
```

## Benefits Summary

| Feature | V1 (Old) | V2 (New) |
|---------|----------|----------|
| **CRC** | Manual calculation | Automatic via `BuildModbusCommand()` |
| **Command Structure** | Implicit | Explicit (slave_id, function_code, etc.) |
| **Register Layout** | Absolute addresses | Relative offsets |
| **Grouping** | Runtime logic | Configuration-driven |
| **Maintainability** | Change address = recalculate CRC | Change anything = auto-update |
| **Clarity** | Address only | Full command parameters |
| **Validation** | Runtime errors | Configuration validation |

## See Also

### Related Documentation

- **[Multi-Device Support](MULTI_DEVICE.md)** - Detailed V2.1 device-based configuration
- **[Migration Guide](MIGRATION.md)** - Upgrading between versions
- **[Validation Rules](VALIDATION.md)** - Configuration validation and error handling

### Technical References

- **[CRC Implementation](CRC.md)** - Automatic CRC calculation details
- **[Function Codes](FUNCTION_CODE.md)** - Supported Modbus function codes
- **[Reactive Power Calculation](REACTIVE_POWER_CALCULATION.md)** - Power calculations

### Main Documentation

- **[README](../README.md)** - Main project documentation
- **[Testing Documentation](../tests/README.md)** - Test suite documentation

## License

Part of the CHINT MQTT-Modbus Bridge project.
