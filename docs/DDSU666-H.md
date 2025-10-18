# CHINT DDSU666-H Single-Phase Energy Meter

## Overview

The DDSU666-H is a comprehensive single-phase energy meter with advanced measurement and communication features.

## Key Features

- **Single-phase AC measurement**: Voltage, current, power (active, reactive, apparent), frequency
- **Power factor calculation**: Real-time power factor monitoring
- **Bidirectional energy metering**: Separate tracking of imported and exported energy
- **Modbus RTU communication**: Standard protocol for industrial automation
- **High accuracy**: Class 1.0 accuracy for active energy measurement

## Technical Specifications

| Parameter | Value |
|-----------|-------|
| Manufacturer | Chint Electric Co. |
| Model | DDSU666-H |
| Rated Voltage | 230V AC |
| Rated Current | Up to 100A |
| Frequency | 50/60 Hz |
| Communication | Modbus RTU (RS485) |
| Protocol | RTU over MQTT (via USR gateway) |
| **Default Slave ID** | **11** (factory default, configurable) |

**Note**: The DDSU666-H comes with a factory default Modbus slave ID of **11**. This can be changed using the meter's configuration interface if needed.

## Register Map

All register values are stored as **32-bit floating point** (2 Modbus registers each).

### Instant Measurements (0x2000 - 0x2021)

| Address | Parameter | Unit | Description | Offset (bytes) |
|---------|-----------|------|-------------|----------------|
| 0x2000 | Voltage | V | Line voltage | 0 |
| 0x2002 | Current | A | Line current | 4 |
| 0x2006 | Active Power | W | Real power consumption/generation (Modbus: kW) | 12 |
| 0x2012 | Apparent Power | VA | Total power (Modbus: kVA) | 36 |
| 0x2018 | Power Factor | - | Ratio of active to apparent power (-1.0 to 1.0) | 48 |
| 0x2020 | Frequency | Hz | Grid frequency | 64 |

**Note**: The meter reports power values in kW/kVA via Modbus. The configuration uses `scale_factor: 1000` to convert these to W/VA for Home Assistant.

### Energy Counters (0x4000 - 0x4015)

| Address | Parameter | Unit | Description | Offset (bytes) |
|---------|-----------|------|-------------|----------------|
| 0x4000 | Total Active Energy | kWh | Cumulative energy (import + export) | 0 |
| 0x400A | Imported Energy | kWh | Energy consumed from grid | 20 |
| 0x4014 | Exported Energy | kWh | Energy exported to grid (solar, etc.) | 40 |

## Modbus Configuration

### Function Codes

- **0x03**: Read Holding Registers (used for all reads)

### Optimized Reading Strategy

The DDSU666-H supports reading multiple consecutive registers in a single Modbus command:

#### Group 1: Instant Measurements

- **Start Address**: 0x2000
- **Register Count**: 34 (covers 0x2000-0x2021)
- **Byte Count**: 68 bytes
- **Benefit**: Reads all real-time values in one command

#### Group 2: Energy Counters

- **Start Address**: 0x4000
- **Register Count**: 22 (covers 0x4000-0x4015)
- **Byte Count**: 44 bytes
- **Benefit**: Reads all energy counters in one command

This strategy minimizes Modbus traffic and reduces polling time.

## Scale Factor / Unit Conversion

The DDSU666-H reports power values in **kW/kVA** via Modbus, not W/VA. To display values correctly in Home Assistant with standard units (W, VA), you must use the `scale_factor` parameter.

### Power Value Conversion

| Modbus Value | Unit | Scale Factor | Displayed Value | Unit |
|--------------|------|--------------|-----------------|------|
| Active Power | kW | 1000 | Active Power | W |
| Apparent Power | kVA | 1000 | Apparent Power | VA |
| Reactive Power | kvar | 1000 | Reactive Power | var |

**Example:**

- Modbus reads: `2.5` (kW)
- With `scale_factor: 1000`: `2500` (W)
- Home Assistant displays: "2500 W" or "2.5 kW" (depending on unit settings)

### Other Values (No Conversion Needed)

| Parameter | Unit | Scale Factor | Notes |
|-----------|------|--------------|-------|
| Voltage | V | 1.0 (default) | Direct reading |
| Current | A | 1.0 (default) | Direct reading |
| Power Factor | - | 1.0 (default) | Dimensionless (-1.0 to 1.0) |
| Frequency | Hz | 1.0 (default) | Direct reading |
| Energy | kWh | 1.0 (default) | Already in kWh |

**Note**: If `scale_factor` is omitted from configuration, it defaults to `1.0` (no conversion).

## Configuration Example

```yaml
energy_meter_mains:
  metadata:
    name: "Energy Meter Mains"
    manufacturer: "Chint Electric Co."
    model: "DDSU666-H Single-Phase Meter"
    enabled: true
  
  rtu:
    slave_id: 11
    poll_interval: 1000
  
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
            device_class: "voltage"
            state_class: "measurement"
            min: 100.0
            max: 300.0
          
          - key: "power_active"
            name: "Active Power"
            offset: 12
            unit: "W"
            scale_factor: 1000       # Device reports kW, convert to W
            device_class: "power"
            state_class: "measurement"
            min: -50000.0
            max: 50000.0
          
          - key: "power_apparent"
            name: "Apparent Power"
            offset: 36
            unit: "VA"
            scale_factor: 1000       # Device reports kVA, convert to VA
            device_class: "apparent_power"
            state_class: "measurement"
            min: 0.0
            max: 100000.0
          # ... additional registers
      
      energy:
        name: "Energy Counters"
        function_code: 0x03
        start_address: 0x4000
        register_count: 22
        enabled: true
        registers:
          - key: "energy_total"
            name: "Total Active Energy"
            offset: 0
            unit: "kWh"
            device_class: "energy"
            state_class: "total_increasing"
            max_kwh_per_hour: 20.0
          # ... additional registers
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

### Apparent Power (VA)

- **Min**: 0.0 VA
- **Max**: 100,000 VA

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
- **Imported Energy Change**: Max 64.0 kWh/hour (high load detection)
- **Exported Energy Change**: Max 3.0 kWh/hour (solar export limit)

## Home Assistant Integration

The meter integrates with Home Assistant using MQTT discovery:

### Device Class Mappings

- **voltage** → Voltage sensor
- **current** → Current sensor
- **power** → Power sensor (Active Power)
- **apparent_power** → Apparent Power sensor
- **reactive_power** → Reactive Power sensor
- **power_factor** → Power Factor sensor
- **frequency** → Frequency sensor
- **energy** → Energy sensor (cumulative)

### State Classes

- **measurement**: For instantaneous values (voltage, current, power, etc.)
- **total_increasing**: For cumulative energy counters

## Troubleshooting

### No Data Received

1. Check slave ID matches meter configuration (default: 11)
2. Verify RS485 wiring (A to A, B to B)
3. Check baud rate (typical: 9600, 19200)
4. Verify Modbus timeout settings

### Incorrect Values

1. Check byte order (ABCD vs DCBA for float32)
2. Verify start address offsets
3. Ensure register count covers all needed registers

### Communication Errors

1. Reduce poll interval if frequent timeouts
2. Check RS485 termination resistors
3. Verify cable length (max 1200m for RS485)

## References

- Manufacturer: [Chint Electric Co.](https://www.chint.com/)
- Product Manual: DDSU666-H User Manual
- Modbus Protocol: [Modbus RTU Specification](https://modbus.org/)
