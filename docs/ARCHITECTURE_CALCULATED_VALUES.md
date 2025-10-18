# Architecture: Calculated Values

## Overview

Starting with version 2.1, the MQTT-Modbus Bridge uses a clean separation between physical Modbus registers and calculated/derived values.

## Configuration Structure

```yaml
devices:
  device_name:
    metadata:
      # Device identification
    
    rtu:
      # Physical layer (slave_id, etc.)
    
    modbus:
      register_groups:
        # Physical registers READ from Modbus
        instant:
          registers:
            - key: "voltage"
            - key: "current"
            - key: "power_active"
        
        energy:
          registers:
            - key: "energy_total"
    
    calculated_values:
      # Virtual registers CALCULATED from Modbus values
      - key: "power_apparent"
        formula: "sqrt(power_active^2 + power_reactive^2)"
```

## Execution Flow

```md
┌─────────────────────────────────────────────────────────┐
│ 1. MODBUS READ PHASE                                    │
│    ┌───────────────┐                                    │
│    │ Read Groups   │ → instant, energy, etc.            │
│    └───────────────┘                                    │
│    Values stored in cache with prefixed keys            │
│    Example: "energy_meter_mains_power_active"           │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│ 2. CALCULATION PHASE                                    │
│    ┌───────────────┐                                    │
│    │ Evaluate      │ → Use cached Modbus values         │
│    │ Formulas      │    Apply math expressions          │
│    └───────────────┘    Apply scale factors             │
│    Values stored with same key prefix structure         │
│    Example: "energy_meter_mains_power_reactive"         │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│ 3. PUBLISHING PHASE                                     │
│    ┌───────────────┐                                    │
│    │ Publish to    │ → Both read AND calculated values  │
│    │ MQTT          │    Published to Home Assistant     │
│    └───────────────┘                                    │
└─────────────────────────────────────────────────────────┘
```

## Command Types

The system now has **3 distinct command types**:

### 1. Single Register Commands

Read a single Modbus register (2 bytes for float32).

**Examples**: VoltageCommand, CurrentCommand, PowerCommand

```go
// Reads 1 register at specified address
func (c *VoltageCommand) Execute(ctx context.Context, gateway Gateway) ([]byte, error) {
    return gateway.ReadHoldingRegisters(c.slaveID, c.register.Address, 2)
}
```

**Configuration**:

```yaml
- key: "voltage"
  offset: 0      # Single float32 at offset 0
  unit: "V"
```

### 2. Group Register Commands

Read multiple contiguous registers as a group for efficiency.

**Example**: GroupExecutor

```go
// Reads N registers starting at group.start_address
func (g *GroupExecutor) ExecuteGroup(group RegisterGroup) (map[string]*CommandResult, error) {
    data := gateway.ReadHoldingRegisters(slaveID, startAddress, registerCount)
    // Parse each register from the group data
}
```

**Configuration**:

```yaml
instant:
  start_address: 0x2000
  register_count: 34
  registers:
    - key: "voltage"
      offset: 0
    - key: "current"
      offset: 4
```

### 3. Calculated Commands

Compute values using mathematical formulas after Modbus reads complete.

**Example**: CalculatedCommand

```go
// No Modbus read - evaluates formula with cached values
func (c *CalculatedCommand) ExecuteCommand(ctx context.Context, gateway Gateway) (*CommandResult, error) {
    // 1. Fetch dependency values from cache
    variables := map[string]float64{
        "power_active": cachedValue1,
        "power_reactive": cachedValue2,
    }
    
    // 2. Evaluate formula
    result := evaluator.Evaluate(formula, variables)
    
    // 3. Apply scale factor
    return result * scaleFactor
}
```

**Configuration**:

```yaml
calculated_values:
  - key: "power_apparent"
    formula: "sqrt(power_active^2 + power_reactive^2)"
    unit: "VA"
```

## Benefits of Separation

### 1. **Clear Responsibility**

- `modbus.register_groups`: Physical hardware interaction
- `calculated_values`: Pure computation/mathematics
- No confusion about what is read vs calculated

### 2. **Execution Guarantees**

- Calculated values ALWAYS run after Modbus reads
- No dependency resolution needed - all values available
- Predictable, deterministic execution order

### 3. **Simplified Configuration**

**Before** (mixed in register groups):

```yaml
registers:
  - key: "power_active"
    offset: 24          # Real Modbus register
  - key: "power_reactive"
    offset: -1          # Wait, is this real or calculated?
    formula: "..."
    depends_on: [...]   # Complex dependency tracking
```

**After** (clear separation):

```yaml
modbus:
  register_groups:
    instant:
      registers:
        - key: "power_active"
          offset: 24    # Always a real Modbus register

calculated_values:
  - key: "power_reactive"
    formula: "..."      # Always calculated, always runs last
```

### 4. **No Dependency Hell**

**Old approach**:

- Need to track `depends_on` arrays
- Risk of circular dependencies
- Complex resolution order

**New approach**:

- All Modbus values available before calculations start
- Simple sequential execution
- No circular dependency risk

### 5. **Better Performance**

- Group reads optimize Modbus communication
- Single pass through calculated values
- Efficient cache usage (5 minute TTL)

## Implementation Details

### Register Key Prefixing

All registers (both read and calculated) use the same prefixing scheme:

```md
{device_key}_{register_key}
```

**Examples**:

- `energy_meter_mains_voltage`
- `energy_meter_mains_power_active`
- `energy_meter_mains_power_reactive` (calculated)

### Formula Variable Resolution

Within formulas, use simple register keys (without device prefix):

```yaml
calculated_values:
  - key: "power_apparent"
    # Use simple names - system adds device prefix automatically
    formula: "sqrt(power_active^2 + power_reactive^2)"
```

The system internally:

1. Parses formula to extract variables: `["power_active", "power_reactive"]`
2. Adds device prefix: `["energy_meter_mains_power_active", "energy_meter_mains_power_reactive"]`
3. Fetches values from cache
4. Substitutes into formula
5. Evaluates and applies scale factor

### Scale Factor Application

Scale factors work consistently for both types:

**Modbus registers** (applied during read):

```yaml
- key: "power_active"
  offset: 24
  scale_factor: 1000    # Device reports kW, convert to W
  # Read: 0.398 kW → Stored: 398.0 W
```

**Calculated values** (applied after formula evaluation):

```yaml
- key: "power_reactive"
  formula: "sqrt(power_apparent^2 - power_active^2)"
  scale_factor: 1000    # Result in kvar, convert to var
  # Calculate: 0.150 kvar → Stored: 150.0 var
```

## Migration Guide

### From V2.0 to V2.1

**Step 1**: Identify calculated registers (those with `offset: -1` or `formula`)

**Step 2**: Move them from `modbus.register_groups.{group}.registers` to `calculated_values`

**Step 3**: Remove `offset` field (not needed for calculated values)

**Step 4**: Remove `depends_on` field (implicit - all Modbus values available)

**Example**:

**Before (V2.0)**:

```yaml
modbus:
  register_groups:
    instant:
      registers:
        - key: "power_active"
          offset: 24
        - key: "power_reactive"
          offset: -1
          formula: "sqrt(power_apparent^2 - power_active^2)"
          depends_on: ["power_apparent", "power_active"]
```

**After (V2.1)**:

```yaml
modbus:
  register_groups:
    instant:
      registers:
        - key: "power_active"
          offset: 24

calculated_values:
  - key: "power_reactive"
    formula: "sqrt(power_apparent^2 - power_active^2)"
    unit: "var"
    device_class: "reactive_power"
    state_class: "measurement"
```

## Future Enhancements

With this architecture, we can easily add:

1. **Time-based calculations**

   ```yaml
   - key: "power_rate_of_change"
     formula: "(power_now - power_5min_ago) / 300"
   ```

2. **Conditional calculations**

   ```yaml
   - key: "net_power"
     formula: "if(power_grid > 0, power_grid - power_solar, 0)"
   ```

3. **Statistical functions**

   ```yaml
   - key: "power_average"
     formula: "avg(power_active, 60)"  # 60 second average
   ```

4. **Cross-device calculations**

   ```yaml
   - key: "total_consumption"
     formula: "energy_meter_mains.power + energy_meter_lights.power"
   ```

All of these would continue to work with the same clean separation principle!
