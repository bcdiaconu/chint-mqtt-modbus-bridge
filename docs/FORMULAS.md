# Formula-Based Calculations

## Overview

The MQTT-Modbus Bridge supports formula-based calculations for derived register values. Calculated values are defined in a separate `calculated_values` section within each device configuration, making it clear that these are computed after all Modbus reads complete.

## Configuration Structure

### Device Organization

```yaml
devices:
  energy_meter_mains:
    metadata:
      name: "Energy Meter Mains"
      enabled: true
    
    rtu:
      slave_id: 11
    
    modbus:
      register_groups:
        instant:
          # Physical registers read from Modbus
          registers:
            - key: "power_active"
              offset: 0
            - key: "power_apparent"
              offset: 36
        energy:
          # Energy counters
          registers:
            - key: "energy_total"
              offset: 0
    
    # Calculated values - executed AFTER all Modbus reads
    calculated_values:
      - key: "power_reactive"
        name: "Reactive Power"
        unit: "var"
        formula: "sqrt(power_apparent^2 - power_active^2)"
        scale_factor: 1000
        device_class: "reactive_power"
        state_class: "measurement"
```

## Key Concepts

### Execution Order

1. **Modbus Reads**: All `register_groups` are read from Modbus devices
2. **Calculated Values**: Formulas are evaluated using values from step 1
3. **MQTT Publishing**: Both read and calculated values are published

This separation ensures:

- ✅ All dependencies are available when formulas execute
- ✅ Clear distinction between physical and calculated registers
- ✅ Simplified configuration structure

### Calculated Value Fields

- **`key`**: Unique identifier for this calculated value
- **`name`**: Human-readable display name
- **`unit`**: Unit of measurement (e.g., "var", "VA", "W")
- **`formula`**: Mathematical expression (see Formula Syntax below)
- **`scale_factor`**: Multiplier applied to final result (optional, default: 1.0)
- **`device_class`**: Home Assistant device class
- **`state_class`**: Home Assistant state class  
- **`min`/`max`**: Validation bounds (optional)

## Configuration Example

### DDSU666-H: Calculate Reactive Power from Active and Apparent Power

The DDSU666-H energy meter provides active power (P) and apparent power (S) directly from Modbus registers. Reactive power (Q) can be calculated using the formula: **Q = √(S² - P²)**

```yaml
devices:
  energy_meter_mains:
    modbus:
      register_groups:
        instant:
          registers:
            - key: "power_active"
              offset: 24
              unit: "W"
              scale_factor: 1000  # kW → W
            
            - key: "power_apparent"
              offset: 36
              unit: "VA"
              scale_factor: 1000  # kVA → VA
    
    calculated_values:
      - key: "power_reactive"
        name: "Reactive Power"
        unit: "var"
        formula: "sqrt(power_apparent^2 - power_active^2)"
        scale_factor: 1000  # kvar → var
        device_class: "reactive_power"
        state_class: "measurement"
```

### DDSU666: Calculate Apparent Power from Active and Reactive Power

The DDSU666 energy meter provides active power (P) and reactive power (Q) directly from Modbus registers. Apparent power (S) can be calculated using the formula: **S = √(P² + Q²)**

```yaml
devices:
  energy_meter_lights:
    modbus:
      register_groups:
        instant:
          registers:
            - key: "power_active"
              offset: 8
              unit: "W"
              scale_factor: 1000.0  # kW → W
            
            - key: "power_reactive"
              offset: 12
              unit: "var"
              scale_factor: 1000.0  # kvar → var
    
    calculated_values:
      - key: "power_apparent"
        name: "Apparent Power"
        unit: "VA"
        formula: "sqrt(power_active^2 + power_reactive^2)"
        device_class: "apparent_power"
        state_class: "measurement"
```

## Formula Syntax

### Expression Evaluator

The expression evaluator supports:

- **Basic operators**: `+`, `-`, `*`, `/`
- **Power operator**: `^` (e.g., `power_active^2`)
- **Functions**:
  - `sqrt()` - Square root
  - `abs()` - Absolute value

### Variables

Variables in formulas reference register keys from the same device. The system automatically resolves variable names to their current values.

**For example:**

```yaml
formula: "sqrt(power_active^2 + power_reactive^2)"
```

The evaluator will:

1. Look up the current value of `power_active` (from Modbus reads)
2. Look up the current value of `power_reactive` (from Modbus reads)
3. Substitute these values into the formula
4. Calculate and return the result

**Important**: Variable names must exactly match register `key` values (case-sensitive).

### Mathematical Operations

#### Basic Arithmetic

```yaml
# Addition
formula: "voltage_L1 + voltage_L2 + voltage_L3"

# Subtraction
formula: "power_imported - power_exported"

# Multiplication
formula: "voltage * current"

# Division
formula: "energy_total / 1000"  # Convert Wh to kWh
```

#### Power Operations

```yaml
# Square a value
formula: "current^2"

# Cube a value
formula: "voltage^3"

# Square root (using function)
formula: "sqrt(power_active^2 + power_reactive^2)"
```

#### Functions

```yaml
# Square root
formula: "sqrt(value1^2 + value2^2)"

# Absolute value
formula: "abs(power_reactive)"

# Combining functions
formula: "sqrt(abs(power_apparent^2 - power_active^2))"
```

### Operator Precedence

The expression evaluator follows standard mathematical precedence:

1. **Functions**: `sqrt()`, `abs()` (highest precedence)
2. **Power**: `^`
3. **Multiplication/Division**: `*`, `/`
4. **Addition/Subtraction**: `+`, `-` (lowest precedence)

Example evaluation order for `sqrt(power_active^2 + power_reactive^2)`:

1. Calculate `power_active^2`
2. Calculate `power_reactive^2`
3. Add the squared values
4. Take the square root

## Execution Flow

When the bridge starts and polls devices:

1. **Modbus Read Phase**: All `register_groups` are read from Modbus
   - Values are stored with prefixed keys (e.g., `energy_meter_mains_power_active`)
   - Read values are cached for 5 minutes

2. **Calculation Phase**: All `calculated_values` are executed
   - Formula variables are resolved from cached Modbus values
   - Mathematical expressions are evaluated
   - Scale factors are applied to results

3. **Publishing Phase**: Both read and calculated values are published to MQTT

### Example Execution

Given configuration:

```yaml
devices:
  energy_meter_lights:
    modbus:
      register_groups:
        instant:
          registers:
            - key: "power_active"
              offset: 8
              scale_factor: 1000.0  # Reads 0.398 kW
            
            - key: "power_reactive"
              offset: 12
              scale_factor: 1000.0  # Reads 0.150 kvar
    
    calculated_values:
      - key: "power_apparent"
        formula: "sqrt(power_active^2 + power_reactive^2)"
```

Execution steps:

1. **Modbus Read**:
   - Read `power_active` → 0.398 kW × 1000 = 398.0 W
   - Read `power_reactive` → 0.150 kvar × 1000 = 150.0 var

2. **Calculation**:
   - Substitute: `sqrt(398.0^2 + 150.0^2)`
   - Evaluate: `sqrt(158404.0 + 22500.0)` = `sqrt(180904.0)` = 425.3 VA

3. **Publish**:
   - `energy_meter_lights/power_active` → 398.0 W
   - `energy_meter_lights/power_reactive` → 150.0 var
   - `energy_meter_lights/power_apparent` → 425.3 VA

## Implementation Details

### Register Key Resolution

Within formulas, you reference registers by their simple `key` value (without device prefix):

```yaml
devices:
  energy_meter_lights:
    modbus:
      register_groups:
        instant:
          registers:
            - key: "power_active"     # Stored as "energy_meter_lights_power_active"
            - key: "power_reactive"   # Stored as "energy_meter_lights_power_reactive"
    
    calculated_values:
      - key: "power_apparent"
        # Formula uses simple key names (automatically resolved to prefixed keys)
        formula: "sqrt(power_active^2 + power_reactive^2)"
```

The formula system automatically:

- Prefixes variable names with the device key
- Resolves `power_active` → `energy_meter_lights_power_active`
- Fetches the current value from cache

### Dependency Caching

The system uses a 5-minute cache for register values. Benefits:

- **Performance**: Calculated values use cached Modbus reads
- **Consistency**: All calculations use the same snapshot of data
- **Efficiency**: No redundant Modbus reads

### Error Handling

The formula evaluator handles various error conditions:

- **Missing variables**: Error if a register referenced in the formula doesn't exist
- **Unavailable data**: Error if a required register value hasn't been read yet
- **Invalid formula syntax**: Error if the formula cannot be parsed
- **Mathematical errors**: Error for operations like division by zero or square root of negative numbers

## Use Cases

### Power Triangle Calculations

```yaml
calculated_values:
  # Given P and S, calculate Q
  - key: "power_reactive"
    formula: "sqrt(power_apparent^2 - power_active^2)"

  # Given P and Q, calculate S
  - key: "power_apparent"
    formula: "sqrt(power_active^2 + power_reactive^2)"

  # Given Q and S, calculate P
  - key: "power_active"
    formula: "sqrt(power_apparent^2 - power_reactive^2)"
```

### Energy Aggregation

```yaml
calculated_values:
  # Total energy across phases
  - key: "energy_total"
    formula: "energy_L1 + energy_L2 + energy_L3"
    unit: "kWh"
```

### Unit Conversion

```yaml
calculated_values:
  # Convert Wh to kWh
  - key: "energy_kwh"
    formula: "energy_wh / 1000"
    unit: "kWh"
```

### Custom Calculations

```yaml
calculated_values:
  # Power factor from P and S
  - key: "power_factor_calc"
    formula: "power_active / power_apparent"
    unit: ""
  
  # Efficiency calculation
  - key: "efficiency"
    formula: "(power_output / power_input) * 100"
    unit: "%"
```

## Best Practices

1. **Organize by device**: Keep calculated values in the `calculated_values` section of each device

2. **Use meaningful key names**: Choose descriptive keys that indicate what is being calculated

3. **Apply scale factors carefully**: Scale factors are applied to the final result after formula evaluation

4. **Validate formulas**: Test formulas with known values to ensure correct results

5. **Handle edge cases**: Consider adding validation rules (`min`, `max`) to catch unrealistic calculated values

6. **Document complex formulas**: Add comments in configuration explaining the physics or logic behind calculations

## Migrating from Hardcoded Commands

### Before (Hardcoded ReactivePowerCommand)

```go
// reactive_power_command.go
func (c *ReactivePowerCommand) ExecuteCommand(ctx context.Context, gateway Gateway) (*CommandResult, error) {
    // Fetch active power
    activePowerResult, _ := c.executor.ExecuteCommand(ctx, "power_active")
    activeP := activePowerResult.Value

    // Fetch apparent power
    apparentPowerResult, _ := c.executor.ExecuteCommand(ctx, "power_apparent")
    apparentS := apparentPowerResult.Value

    // Calculate reactive power
    reactiveQ := math.Sqrt(apparentS*apparentS - activeP*activeP)
    
    return &CommandResult{Value: reactiveQ}, nil
}
```

### After (Formula-Based Configuration)

```yaml
devices:
  energy_meter_mains:
    calculated_values:
      - key: "power_reactive"
        name: "Reactive Power"
        unit: "var"
        formula: "sqrt(power_apparent^2 - power_active^2)"
        scale_factor: 1000
        device_class: "reactive_power"
        state_class: "measurement"
```

Benefits:

- ✅ No custom Go code required
- ✅ Configuration-driven behavior
- ✅ Easily adaptable to different devices
- ✅ Automatic caching and error handling
- ✅ Consistent with other registers

## Troubleshooting

### Formula Not Evaluating

**Symptom**: Calculated value returns error or zero value

**Possible causes**:

1. Missing register values - check that all registers referenced in formula are defined and readable
1. Syntax error in formula - verify formula syntax matches supported operations
1. Mathematical error - check for division by zero or sqrt of negative numbers
1. Variables not yet available - calculated values need Modbus reads to complete first

**Solution**: Enable debug logging to see detailed evaluation steps:

```yaml
logging:
  level: "debug"
```

### Incorrect Calculated Values

**Symptom**: Formula calculates but result doesn't match expected value

**Possible causes**:

1. Variable names don't match register keys exactly (case-sensitive)
2. Scale factor applied incorrectly
3. Register values not updated (using stale cache)
4. Formula logic error

**Solution**:

- Verify variable names match `key` values exactly
- Check if scale factors on input registers affect calculation
- Review formula logic with known test values
- Enable debug logging to see actual substituted values

### Performance Issues

**Symptom**: Slow response when reading calculated values

**Possible causes**:

1. Complex formulas with many operations
2. Cache timeout too short causing repeated Modbus reads
3. Many calculated values per device

**Solution**:

- Simplify formulas where possible
- Increase cache timeout for stable readings
- Consider if all calculated values are necessary

## Future Enhancements

Planned improvements to the formula system:

- [ ] Additional mathematical functions (sin, cos, tan, log, exp)
- [ ] Conditional expressions (if/then/else)
- [ ] Time-based calculations (rate of change, averages)
- [ ] Support for constants in formulas
- [ ] Formula validation at configuration load time
- [ ] Dependency graph visualization tool
