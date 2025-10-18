# CHINT DDSU666 Single-Phase Energy Meter

## Overview

The DDSU666 is a simplified single-phase energy meter with essential measurement features. This is the basic model in the DDSU666 series, offering core functionality at a lower cost compared to the DDSU666-H variant.

## Key Features

- **Single-phase AC measurement**: Voltage, current, power (active & reactive), frequency
- **Power factor monitoring**: Real-time power factor calculation
- **Basic energy metering**: Total and reverse (exported) energy tracking
- **Modbus RTU communication**: Standard industrial protocol
- **Compact design**: Simplified register layout for faster polling

## Key Differences from DDSU666-H

| Feature | DDSU666 | DDSU666-H |
|---------|---------|-----------|
| Apparent Power | ❌ Not available | ✅ Available (0x2012) |
| Power Units | W, var | W, var |
| Energy Counters | 2 (Total, Reverse) | 3 (Total, Import, Export) |
| Register Layout | Simplified | Comprehensive |
| Price Point | Lower | Higher |
| Use Case | Basic monitoring | Advanced monitoring |

## Technical Specifications

| Parameter | Value |
|-----------|-------|
| Manufacturer | Chint Electric Co. |
| Model | DDSU666 |
| Rated Voltage | 230V AC |
| Rated Current | Up to 100A |
| Frequency | 50/60 Hz |
| Communication | Modbus RTU (RS485) |
| Protocol | RTU over MQTT (via USR gateway) |
| **Default Slave ID** | **1** (factory default, configurable) |

**Note**: The DDSU666 comes with a factory default Modbus slave ID of **1**. This can be changed using the meter's configuration interface if needed. This differs from the DDSU666-H which uses slave ID **11** by default.
| Protocol | RTU over MQTT (via USR gateway) |

## Register Map

All register values are stored as **32-bit floating point** (2 Modbus registers each).

### Instant Measurements (0x2000 - 0x200F)

| Address | Parameter | Unit | Description | Raw Unit | Scale Factor | Offset (bytes) |
|---------|-----------|------|-------------|----------|--------------|----------------|
| 0x2000 | Voltage (U) | V | Line voltage | V | 1.0 | 0 |
| 0x2002 | Current (I) | A | Line current | A | 1.0 | 4 |
| 0x2004 | Active Power (P) | W | Real power consumption/generation | kW | 1000.0 | 8 |
| 0x2006 | Reactive Power (Q) | var | Reactive power | kvar | 1000.0 | 12 |
| 0x200A | Power Factor | - | Ratio of active to apparent power (-1.0 to 1.0) | - | 1.0 | 20 |
| 0x200E | Frequency (Freq) | Hz | Grid frequency | Hz | 1.0 | 28 |

**Important**: The DDSU666 meter reports power values in **kW** and **kvar** at the Modbus level, but the bridge automatically converts these to **W** and **var** using a `scale_factor` of 1000.0 for consistency with DDSU666-H and Home Assistant conventions.

### Energy Counters (0x4000 - 0x400B)

| Address | Parameter | Unit | Description | Offset (bytes) |
|---------|-----------|------|-------------|----------------|
| 0x4000 | Total Active Energy (Ep) | kWh | Cumulative energy consumption | 0 |
| 0x400A | Reverse Active Energy (Ep Reverse) | kWh | Energy exported/returned to grid | 20 |

**Note**: This meter does not have a separate "imported energy" register like the DDSU666-H. The total energy (0x4000) represents net consumption.

## Modbus Configuration

### Function Codes

- **0x03**: Read Holding Registers (used for all reads)

### Optimized Reading Strategy

The DDSU666's simplified register layout allows for efficient polling:

#### Group 1: Instant Measurements

- **Start Address**: 0x2000
- **Register Count**: 16 (covers 0x2000-0x200F)
- **Byte Count**: 32 bytes
- **Registers Read**: U, I, P, Q, PF, Freq (all in one command)
- **Benefit**: All instant measurements in a single Modbus query

#### Group 2: Energy Counters

- **Start Address**: 0x4000
- **Register Count**: 12 (covers 0x4000-0x400B)
- **Byte Count**: 24 bytes
- **Registers Read**: Total energy, Reverse energy
- **Benefit**: Both energy counters in one command

**Total Commands per Poll**: Only **2 Modbus commands** (vs 3+ for DDSU666-H)

## Scale Factor / Unit Conversion

Like the DDSU666-H, the DDSU666 also reports power values in **kW/kvar** via Modbus. The configuration uses `scale_factor: 1000` to convert these to W/var for consistency with Home Assistant standards.

### Power Value Conversion

| Modbus Value | Unit | Scale Factor | Displayed Value | Unit |
|--------------|------|--------------|-----------------|------|
| Active Power | kW | 1000 | Active Power | W |
| Reactive Power | kvar | 1000 | Reactive Power | var |

**Example:**

- Modbus reads: `1.5` (kW)
- With `scale_factor: 1000`: `1500` (W)
- Home Assistant displays: "1500 W" or "1.5 kW"

### Configuration Consistency

Both DDSU666 and DDSU666-H use the **same scale factors** for power measurements, ensuring consistent behavior across devices:

```yaml
power_active:
  unit: "W"
  scale_factor: 1000    # kW → W

power_reactive:
  unit: "var"
  scale_factor: 1000    # kvar → var
```

This unified approach simplifies multi-device configurations.

## Configuration

The DDSU666 meter reports power values in **kW** and **kvar** (not W and var like the DDSU666-H). To maintain consistency across all devices and with Home Assistant conventions, the bridge uses a **scale_factor** parameter:

- **Scale Factor**: A multiplier applied to raw Modbus values to convert them to the desired unit
- **Default**: 1.0 (no conversion)
- **DDSU666 Power Values**: 1000.0 (converts kW → W, kvar → var)

### Example Configuration

```yaml
registers:
  - key: "power_active"
    unit: "W"
    scale_factor: 1000.0  # Converts kW to W
  - key: "voltage"
    unit: "V"
    scale_factor: 1.0     # No conversion needed (or omit for default)
```

This approach allows any Modbus device to report values in its native units while still presenting them correctly in Home Assistant.

## Configuration Example

```yaml
energy_meter_lights:
  metadata:
    name: "Energy Meter Lights"
    manufacturer: "Chint Electric Co."
    model: "DDSU666 Single-Phase Meter"
    enabled: true
  
  rtu:
    slave_id: 1
    poll_interval: 1000
  
  modbus:
    register_groups:
      instant:
        name: "Instant Measurements"
        function_code: 0x03
        start_address: 0x2000
        register_count: 16
        enabled: true
        registers:
          - key: "voltage"
            name: "Voltage"
            offset: 0
            unit: "V"
            device_class: "voltage"
            state_class: "measurement"
            min: 100.0
            max: 300.0
          
          - key: "current"
            name: "Current"
            offset: 4
            unit: "A"
            device_class: "current"
            state_class: "measurement"
            min: 0.0
            max: 100.0
          
          - key: "power_active"
            name: "Active Power"
            offset: 8
            unit: "W"
            scale_factor: 1000.0  # DDSU666 reports in kW, convert to W
            device_class: "power"
            state_class: "measurement"
            min: -50000.0
            max: 50000.0
          
          - key: "power_reactive"
            name: "Reactive Power"
            offset: 12
            unit: "var"
            scale_factor: 1000.0  # DDSU666 reports in kvar, convert to var
            device_class: "reactive_power"
            state_class: "measurement"
            min: -50000.0
            max: 50000.0
          
          - key: "power_factor"
            name: "Power Factor"
            offset: 20
            unit: ""
            device_class: "power_factor"
            state_class: "measurement"
            min: -1.0
            max: 1.0
          
          - key: "frequency"
            name: "Frequency"
            offset: 28
            unit: "Hz"
            device_class: "frequency"
            state_class: "measurement"
            min: 45.0
            max: 65.0
      
      energy:
        name: "Energy Counter"
        function_code: 0x03
        start_address: 0x4000
        register_count: 12
        enabled: true
        registers:
          - key: "energy_total"
            name: "Total Active Energy"
            offset: 0
            unit: "kWh"
            device_class: "energy"
            state_class: "total_increasing"
            max_kwh_per_hour: 20.0
          
          - key: "energy_exported"
            name: "Reverse Active Energy"
            offset: 20
            unit: "kWh"
            device_class: "energy"
            state_class: "total_increasing"
            max_kwh_per_hour: 3.0
```

## Validation Limits

### Voltage (V)

- **Min**: 100.0 V (below indicates power issue)
- **Max**: 300.0 V (above indicates overvoltage)
- **Normal Range**: 207V - 253V (230V ±10%)

### Current (A)

- **Min**: 0.0 A
- **Max**: 100.0 A (typical installation limit)

### Active Power (W)

- **Min**: -50,000 W (export scenario)
- **Max**: 50,000 W (import scenario)

### Reactive Power (var)

- **Min**: -50,000 var
- **Max**: 50,000 var

### Power Factor

- **Min**: -1.0 (leading, capacitive)
- **Max**: 1.0 (unity or lagging, inductive)
- **Ideal**: 1.0 (unity, maximum efficiency)

### Frequency (Hz)

- **Min**: 45.0 Hz (below indicates grid instability)
- **Max**: 65.0 Hz (above indicates grid instability)
- **Normal**: 50Hz ±1% or 60Hz ±1%

### Energy Counters (kWh)

- **Total Energy Change**: Max 20.0 kWh/hour (spike detection)
- **Exported Energy Change**: Max 3.0 kWh/hour (solar export scenarios)

## Home Assistant Integration

The meter integrates with Home Assistant using MQTT discovery:

### Device Class Mappings

- **voltage** → Voltage sensor
- **current** → Current sensor
- **power** → Power sensor (Active Power in W)
- **reactive_power** → Reactive Power sensor (in var)
- **power_factor** → Power Factor sensor
- **frequency** → Frequency sensor
- **energy** → Energy sensor (cumulative)

### State Classes

- **measurement**: For instantaneous values (voltage, current, power, etc.)
- **total_increasing**: For cumulative energy counters

### Topic Structure

```md
sensor/<device_id>/voltage
sensor/<device_id>/current
sensor/<device_id>/power_active
sensor/<device_id>/power_reactive
sensor/<device_id>/power_factor
sensor/<device_id>/frequency
sensor/<device_id>/energy_total
sensor/<device_id>/energy_exported
```

## Comparison: When to Use DDSU666 vs DDSU666-H

### Choose DDSU666 when

- ✅ You need basic power and energy monitoring
- ✅ Budget is a primary concern
- ✅ You don't need apparent power measurements
- ✅ Simple import/export tracking is sufficient
- ✅ Faster polling is desired (fewer registers to read)

### Choose DDSU666-H when

- ✅ You need comprehensive power quality analysis
- ✅ Apparent power measurement is required
- ✅ Detailed import/export separation is needed
- ✅ Advanced diagnostics are important
- ✅ Power factor calculations require apparent power

## Troubleshooting

### No Data Received

1. Check slave ID matches meter configuration
2. Verify RS485 wiring (A to A, B to B)
3. Check baud rate (typical: 9600, 19200)
4. Verify Modbus timeout settings

### Incorrect Values

1. Check byte order (ABCD vs DCBA for float32)
2. Verify start address offsets
3. Ensure register count covers all needed registers

### Missing Apparent Power

This is normal - DDSU666 does not provide apparent power. If needed:

```math
\text{Apparent Power (kVA)} \approx \sqrt{\text{Active Power}^2 + \text{Reactive Power}^2}
```

### Energy Counter Discrepancies

- DDSU666 has only "Total" and "Reverse" energy
- It does NOT have separate "Import" counter
- Total = Net consumption (import - export cumulative)

## References

- Manufacturer: [Chint Electric Co.](https://www.chint.com/)
- Product Manual: DDSU666 User Manual
- Modbus Protocol: [Modbus RTU Specification](https://modbus.org/)
- Related Model: [DDSU666-H Documentation](DDSU666-H.md)
