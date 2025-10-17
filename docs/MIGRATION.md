# Migration Guide

## Overview

This guide helps you migrate your configuration between different versions:

- **V1 → V2.0**: Individual registers → Group-based configuration
- **V2.0 → V2.1**: Single device → Multiple devices with segregated sections
- **Single Device → Multi-Device**: Adding more devices to V2.1 configuration

## V1 to V2.0 Migration

### What Changed

V2.0 introduced **group-based register organization** with explicit Modbus commands instead of individual register addresses.

### Before (V1)

```yaml
registers:
  voltage:
    name: "Voltage"
    address: 0x2000
    unit: "V"
    scale: 0.1
  
  current:
    name: "Current"
    address: 0x2002
    unit: "A"
    scale: 0.001
```

### After (V2.0)

```yaml
version: "2.0"

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
        offset: 0        # 0 bytes from start
        unit: "V"
        scale: 0.1
      
      - key: "current"
        name: "Current"
        offset: 4        # 4 bytes from start (2 registers)
        unit: "A"
        scale: 0.001
```

### Migration Steps

1. **Add version field**: `version: "2.0"`
2. **Create register_groups**: Organize registers into logical groups
3. **Add group parameters**: `slave_id`, `function_code`, `start_address`, `register_count`
4. **Convert addresses to offsets**: Calculate offset from group start address
5. **Convert register list**: Array of registers with `key`, `name`, `offset`, `unit`, `scale`

### Address to Offset Conversion

```math
\text{offset} = (\text{register\_address} - \text{group\_start\_address}) \times 2
```

Example:

```md
Group start_address: 0x2000
Register address:    0x2002
Offset:              (0x2002 - 0x2000) × 2 = 4 bytes
```

## V2.0 to V2.1 Migration

### Changes

V2.1 introduced **device-based organization** with four segregated configuration sections:

- `metadata`: Device identification
- `rtu`: RTU/Physical layer settings
- `modbus`: Modbus protocol (contains register_groups)
- `homeassistant`: Home Assistant integration (optional)

### Before (V2.0)

```yaml
version: "2.0"

modbus:
  slave_id: 11
  poll_interval: 5000

mqtt:
  broker: "192.168.1.100:1883"

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
        scale: 0.1
```

### After (V2.1)

```yaml
version: "2.1"

mqtt:
  broker: "192.168.1.100:1883"

global:
  poll_interval: 5000       # Default for all devices

devices:
  energy_meter:             # Device key
    metadata:
      name: "Energy Meter"
      enabled: true
    
    rtu:
      slave_id: 11
      # poll_interval: 1000  # Optional: override global
    
    # homeassistant section is optional
    # If omitted, device_id defaults to device key ("energy_meter")
    
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

### Steps to migrate

1. **Update version**: `version: "2.1"`
2. **Move poll_interval to global**: Create `global:` section with `poll_interval`
3. **Create devices map**: Add `devices:` with unique device keys
4. **Create device sections**:
   - `metadata`: Add `name`, optionally `manufacturer`, `model`, `enabled`
   - `rtu`: Move `slave_id`, optionally add per-device `poll_interval`
   - `modbus`: Move `register_groups` here
   - `homeassistant` (optional): Add `device_id`, `manufacturer`, `model` overrides
5. **Remove group-level slave_id**: Now in `devices.<device>.rtu.slave_id`

### slave_id Migration

**V2.0**: Each register_group had its own `slave_id`

```yaml
register_groups:
  instant:
    slave_id: 11        # ← Group-level
    # ...
```

**V2.1**: Device-level `slave_id` in rtu section

```yaml
devices:
  energy_meter:
    rtu:
      slave_id: 11      # ← Device-level
    modbus:
      register_groups:
        instant:
          # No slave_id here
```

## Single Device to Multi-Device

### Adding a Second Device

Starting configuration (V2.1 with one device):

```yaml
version: "2.1"

mqtt:
  broker: "192.168.1.100:1883"

global:
  poll_interval: 5000

devices:
  energy_meter_1:
    metadata:
      name: "Energy Meter 1"
      enabled: true
    
    rtu:
      slave_id: 11
    
    modbus:
      register_groups:
        instant:
          # ... register group configuration
```

Adding a second device:

```yaml
version: "2.1"

mqtt:
  broker: "192.168.1.100:1883"

global:
  poll_interval: 5000

devices:
  energy_meter_1:
    metadata:
      name: "Energy Meter 1"
      enabled: true
    
    rtu:
      slave_id: 11
    
    modbus:
      register_groups:
        instant:
          # ... register group configuration

  energy_meter_2:           # ← New device
    metadata:
      name: "Energy Meter 2"
      enabled: true
    
    rtu:
      slave_id: 12          # ← Must be unique!
    
    homeassistant:
      device_id: "meter_2"  # Optional: customize HA device ID
    
    modbus:
      register_groups:
        instant:
          # ... register group configuration (can be identical)
```

### Requirements for Multiple Devices

1. **Unique device keys**: `energy_meter_1`, `energy_meter_2` (automatic)
2. **Unique slave IDs**: `11`, `12` (validated)
3. **Unique HA device IDs**: Explicit or fallback to device key (validated)

## Common Migration Scenarios

### Scenario 1: Simple V1 to V2.1

If you have a simple V1 configuration with one device:

1. Update to V2.0 first (group-based)
2. Then update to V2.1 (device-based)

OR jump directly to V2.1:

```yaml
# V1
registers:
  voltage:
    name: "Voltage"
    address: 0x2000

# V2.1 (direct)
version: "2.1"
devices:
  energy_meter:
    metadata:
      name: "Energy Meter"
    rtu:
      slave_id: 11
    modbus:
      register_groups:
        instant:
          function_code: 0x03
          start_address: 0x2000
          register_count: 34
          registers:
            - key: "voltage"
              name: "Voltage"
              offset: 0
```

### Scenario 2: Adding Device-Specific Settings

Customize individual devices:

```yaml
devices:
  fast_meter:
    metadata:
      name: "Fast Polling Meter"
    rtu:
      slave_id: 11
      poll_interval: 1000    # ← Override global (fast polling)
    modbus:
      register_groups:
        instant:
          # ... high-priority measurements

  slow_meter:
    metadata:
      name: "Slow Polling Meter"
    rtu:
      slave_id: 12
      poll_interval: 10000   # ← Override global (slow polling)
    modbus:
      register_groups:
        daily:
          # ... low-priority measurements
```

### Scenario 3: Customizing Home Assistant Integration

```yaml
devices:
  meter_1:
    metadata:
      name: "Energy Meter 1"
      manufacturer: "Chint"
      model: "DTSU666-H"
    
    rtu:
      slave_id: 11
    
    homeassistant:
      device_id: "main_meter"           # Custom HA device ID
      manufacturer: "Chint Electric"     # Override for HA
      model: "DTSU666-H Three-Phase"    # More specific model
    
    modbus:
      register_groups:
        instant:
          # ... configuration
```

## Validation After Migration

### Run Validation

```powershell
# Validate configuration
.\mqtt-modbus-bridge.exe --validate-config
```

### Common Validation Errors

#### Duplicate slave_id

```log
Error: duplicate slave_id 11 found in devices: energy_meter_1, energy_meter_2
```

**Solution**: Assign unique slave IDs to each device:

```yaml
devices:
  energy_meter_1:
    rtu:
      slave_id: 11    # ← Unique
  energy_meter_2:
    rtu:
      slave_id: 12    # ← Unique
```

#### Duplicate HA device_id

```log
Error: duplicate homeassistant.device_id 'meter' found in devices: meter_1 (explicit), meter_2 (explicit)
```

**Solution**: Use unique device IDs:

```yaml
devices:
  meter_1:
    homeassistant:
      device_id: "meter_1"    # ← Unique
  meter_2:
    homeassistant:
      device_id: "meter_2"    # ← Unique
```

#### Device key conflicts with explicit device_id

```log
Error: duplicate homeassistant.device_id 'energy_meter' found in devices: energy_meter (fallback), meter_2 (explicit)
```

**Solution**: Rename device key or change explicit device_id:

```yaml
devices:
  energy_meter:
    # No homeassistant section - uses "energy_meter" as device_id
  meter_2:
    homeassistant:
      device_id: "meter_2"    # ← Don't use "energy_meter"
```

## Rollback

If you need to rollback:

1. **Keep backup**: Always backup `config.yaml` before migration
2. **Restore backup**: Copy backup file back to `config.yaml`
3. **Restart application**: The application will use the old configuration

## Testing

After migration:

1. **Validate configuration**: Run with `--validate-config`
2. **Test connectivity**: Check MQTT broker connection
3. **Verify Home Assistant**: Check devices appear correctly
4. **Monitor logs**: Watch for errors or warnings
5. **Test each device**: Verify all devices communicate correctly

## See Also

- [Multi-Device Support](MULTI_DEVICE.md) - Detailed V2.1 device configuration
- [Configuration Reference](CONFIG.md) - Complete configuration documentation
- [Validation Rules](VALIDATION.md) - Validation rules and examples

## License

This project is licensed under the MIT License.
