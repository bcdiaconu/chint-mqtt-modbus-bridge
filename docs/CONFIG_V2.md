# Configuration V2 - Group-Based Modbus Commands

## Overview

Configuration V2 introduces a **group-based approach** to Modbus register reading. Instead of defining individual register addresses, you define **register groups** that map directly to Modbus commands. This leverages the automatic CRC calculation from `modbus.BuildModbusCommand()`.

## Benefits

### ✅ Explicit Command Definition
- **Before (V1)**: Individual addresses, implicit grouping
- **After (V2)**: Explicit groups with `slave_id`, `function_code`, `start_address`, `register_count`

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

```
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

```
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

```
Formula: offset = (address - group_start_address) * 2
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

## Migration from V1

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

- ✅ Slave ID is set
- ✅ Function code is set (defaults to 0x03)
- ✅ Register count is set
- ✅ Offsets are within range
- ✅ Register keys are unique
- ✅ Calculated register dependencies exist

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

- [CRC Implementation](CRC.md) - Details about automatic CRC calculation
- [Modbus Protocol](../README.md#modbus) - Modbus RTU specification
- [Configuration V1](../config-sample.yaml) - Original configuration format

## License

Part of the CHINT MQTT-Modbus Bridge project.
