# Reactive Power Calculation

## Overview

This document explains how reactive power is calculated in the MQTT-Modbus Bridge application.

## Background

Reactive power (Q) is an important electrical parameter that cannot be measured directly from a single Modbus register. Instead, it must be calculated from other power measurements:

- **Active Power (P)**: Real power measured in Watts (W) - from register `power_active`
- **Apparent Power (S)**: Total power measured in Volt-Amperes (VA) - from register `power_apparent`
- **Reactive Power (Q)**: Imaginary power measured in Volt-Amperes Reactive (VAR) - calculated

## Formula

The reactive power is calculated using the following formula:

```
Q = √(S² - P²)
```

Where:
- Q = Reactive power (VAR)
- S = Apparent power (VA)
- P = Active power (W)

## Implementation

### Group-Based Calculation

When the `instant_group` is executed, it reads multiple registers in a single Modbus query for efficiency. This includes:
- Voltage
- Current
- Active Power (`power_active`)
- Apparent Power (`power_apparent`)
- Frequency
- Power Factor

After the group is executed and results are parsed, the reactive power is calculated using the already-read values of active and apparent power. This approach:

1. **Eliminates duplicate reads**: No need to read `power_active` and `power_apparent` again
2. **Improves performance**: Calculation happens instantly after group execution
3. **Maintains consistency**: All values are from the same Modbus query timestamp

### Code Flow

```
1. Execute instant_group → Read all instant registers in one query
2. Parse results → Extract power_active and power_apparent values
3. Calculate reactive power → Q = √(S² - P²)
4. Publish all results → Including calculated reactive power
```

### Error Handling

The implementation includes validation to prevent mathematical errors:

```go
if S*S < P*P {
    Q = 0.0  // Set to 0 if apparent < active (measurement error)
    logger.LogWarn("Apparent power less than active power")
} else {
    Q = math.Sqrt(S*S - P*P)
}
```

## Configuration

In `config-sample.yaml`, the reactive power register is defined with a virtual address:

```yaml
power_reactive:
  name: "Reactive Power"
  address: 0x0000  # Virtual address - calculated from other values
  unit: "var"
  device_class: "reactive_power"
  state_class: "measurement"
  ha_topic: "sensor/energy_meter/power_reactive"
```

The `address: 0x0000` indicates this is a virtual/calculated register, not read directly from the device.

## Exclusion from Group

The `power_reactive` register is explicitly excluded from the `instant_group` during group creation (see `main.go` line 157):

```go
if !app.isEnergyRegister(name) && name != "power_reactive" {
    instantRegisterNames = append(instantRegisterNames, name)
}
```

This is because:
1. It doesn't correspond to a physical Modbus register
2. It's calculated after the group execution
3. Including it would cause parsing errors (no physical register to read)

## Testing

The reactive power calculation is tested in:

**Integration Test**: `tests/integration/groups_integration_test.go`

```go
func TestReactivePowerCalculationFromGroupResults(t *testing.T) {
    groupResults := map[string]float64{
        "power_active":   2000.0, // 2000 W
        "power_apparent": 2300.0, // 2300 VA
    }
    
    // Expected: Q = √(2300² - 2000²) ≈ 1135.78 VAR
}
```

## Benefits

### Performance
- **Single Modbus Query**: All instant values read together
- **No Duplicate Reads**: Reuses already-read power values
- **Instant Calculation**: Mathematical operation is fast

### Accuracy
- **Temporal Consistency**: All values from same timestamp
- **Reduced Network Traffic**: One query instead of three
- **Error Validation**: Handles measurement errors gracefully

### Home Assistant Integration
- Reactive power appears as a standard sensor
- Updates together with other instant measurements
- Properly classified with `device_class: reactive_power`

## Future Enhancements

Potential improvements:
1. Add configurable validation thresholds
2. Track calculation errors separately
3. Support alternative calculation methods (e.g., from current and voltage)
4. Add historical trend analysis

## Related Files

- `src/main.go` - Main calculation logic in `calculateAndPublishReactivePower()`
- `src/pkg/modbus/reactive_power_command.go` - Reactive power command implementation
- `tests/integration/groups_integration_test.go` - Integration tests
- `config-sample.yaml` - Configuration example

## References

- [Reactive Power - Wikipedia](https://en.wikipedia.org/wiki/AC_power#Reactive_power)
- [Power Triangle](https://en.wikipedia.org/wiki/AC_power#Power_triangle)
