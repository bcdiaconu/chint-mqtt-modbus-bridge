# Modbus Function Codes - Why Per-Group, Not Global

## Problem with Global Function Code

Having a global `function_code` in the Modbus configuration is **inflexible** and **incorrect** because:

### 1. Different Register Types Need Different Functions

A typical Modbus device has multiple register types:

```yaml
# ❌ WRONG - Global function code
modbus:
  slave_id: 11
  function_code: 0x03  # What if we need 0x04 for some registers?
```

Common Modbus functions:
- **0x03** - Read Holding Registers (configuration, setpoints)
- **0x04** - Read Input Registers (sensor data, read-only)
- **0x06** - Write Single Register
- **0x10** - Write Multiple Registers

### 2. Real-World Example

Consider an energy meter:

```yaml
# ✅ CORRECT - Function code per group
register_groups:
  # Holding registers - configuration values
  config:
    function_code: 0x03  # Read Holding Registers
    start_address: 0x0000
    registers:
      - key: "ct_ratio"
        name: "CT Ratio"
  
  # Input registers - live measurements
  instant:
    function_code: 0x04  # Read Input Registers
    start_address: 0x2000
    registers:
      - key: "voltage"
        name: "Voltage"
      - key: "current"
        name: "Current"
  
  # Holding registers - energy counters
  energy:
    function_code: 0x03  # Read Holding Registers
    start_address: 0x4000
    registers:
      - key: "energy_total"
        name: "Total Energy"
```

### 3. Commands Generated

Each group generates its own command with the appropriate function code:

```
Config Group:  0B 03 0000 0002 XXXX  (Function 0x03)
                  ↑
                  └─ Read Holding Registers

Instant Group: 0B 04 2000 0022 XXXX  (Function 0x04)
                  ↑
                  └─ Read Input Registers

Energy Group:  0B 03 4000 0016 XXXX  (Function 0x03)
                  ↑
                  └─ Read Holding Registers
```

## Architecture Decision

### Function Code Belongs To:

1. ✅ **Register Group** - Each group is a Modbus command
2. ✅ **Individual Register** (if not grouped) - Each standalone read is a command
3. ❌ **Global Modbus Config** - Too restrictive

### Why This is Correct

A Modbus command is defined by:
```
Command = SlaveID + FunctionCode + StartAddress + Count + CRC
```

Since each **group = one command**, the function code is part of the group definition.

## Configuration Best Practices

### Default Slave ID (Global)

```yaml
modbus:
  slave_id: 11  # Default for all groups (can be overridden)
```

The global `slave_id` makes sense because:
- Most devices have one slave ID
- Can be overridden per group if needed
- Reduces configuration duplication

### Function Code (Per-Group)

```yaml
register_groups:
  my_group:
    slave_id: 11           # Use global default or override
    function_code: 0x03    # REQUIRED - specific to this command
    start_address: 0x2000
    register_count: 34
```

The per-group `function_code` is correct because:
- Each group is a distinct Modbus command
- Different groups may use different functions
- Makes the command structure explicit

## Code Implementation

### Group Validation

```go
func (g *RegisterGroup) Validate() error {
    if g.SlaveID == 0 {
        return fmt.Errorf("slave_id is required for register group %s", g.Name)
    }
    if g.FunctionCode == 0 {
        return fmt.Errorf("function_code is required for register group %s", g.Name)
    }
    // ... more validation
}
```

### Command Generation

```go
// Each group generates its own command
func (g *RegisterGroup) BuildCommand() []byte {
    return modbus.BuildModbusCommand(
        g.SlaveID,        // Can be group-specific or default
        g.FunctionCode,   // MUST be group-specific
        g.StartAddress,
        g.RegisterCount,
    )
}
```

## Common Function Code Reference

| Code | Name | Description | Use Case |
|------|------|-------------|----------|
| 0x01 | Read Coils | Read 1-2000 coil status | Digital outputs (ON/OFF) |
| 0x02 | Read Discrete Inputs | Read 1-2000 input status | Digital inputs (sensors) |
| 0x03 | Read Holding Registers | Read 1-125 registers | Configuration, setpoints |
| 0x04 | Read Input Registers | Read 1-125 registers | Sensor data (read-only) |
| 0x05 | Write Single Coil | Write one coil | Control single output |
| 0x06 | Write Single Register | Write one register | Set one parameter |
| 0x0F | Write Multiple Coils | Write multiple coils | Control multiple outputs |
| 0x10 | Write Multiple Registers | Write multiple registers | Set multiple parameters |

## Migration Guide

### From V1 (Implicit)

V1 configuration didn't specify function code - it was hardcoded:

```yaml
# V1 - function code hardcoded as 0x03 in code
registers:
  voltage:
    address: 0x2000
```

### To V2 (Explicit)

V2 makes it explicit and flexible:

```yaml
# V2 - function code explicit per group
register_groups:
  instant:
    function_code: 0x03  # Or 0x04, depending on device
    start_address: 0x2000
    register_count: 34
    registers:
      - key: "voltage"
        offset: 0
```

## Summary

| Aspect | Global Function Code | Per-Group Function Code |
|--------|---------------------|-------------------------|
| **Flexibility** | ❌ Limited to one function | ✅ Different functions per group |
| **Correctness** | ❌ Command = Group, not global | ✅ Each group = one command |
| **Use Cases** | ❌ Can't mix 0x03 and 0x04 | ✅ Can read both register types |
| **Configuration** | ❌ One size fits none | ✅ Precise control |
| **Real Devices** | ❌ Many devices need both | ✅ Matches device capabilities |

## Conclusion

**Function code MUST be at the group level** because:
1. Each group represents one Modbus command
2. Commands require a function code
3. Different groups may need different functions
4. This matches the Modbus protocol structure

The global `slave_id` makes sense as a default, but `function_code` does not - it's intrinsic to each command/group.
