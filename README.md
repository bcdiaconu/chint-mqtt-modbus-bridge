# chint-mqtt-modbus-bridge

A robust and production-ready bridge for integrating Chint energy meters (Modbus RTU) with Home Assistant, using a PUSR USR-DR164 Modbus-MQTT gateway. This project enables seamless, real-time monitoring and automation of energy data in Home Assistant via MQTT Discovery. Designed for reliability, flexibility, and easy extension.

**Supported Hardware:**

- **Energy Meters:**
  - [Chint DDSU666-H](docs/DDSU666-H.md) - Comprehensive single-phase meter with advanced features
  - [Chint DDSU666](docs/DDSU666.md) - Simplified single-phase meter (basic model)
- **Gateway:** PUSR USR-DR164 (Modbus RTU <-> MQTT)

---

A robust bridge between USR-DR164 Modbus-MQTT Gateway and Home Assistant, implemented in Go with SOLID principles.

## Features

### Core Features

- **SOLID Architecture**: Follows all SOLID principles for easy maintenance
- **Strategy Pattern**: Each Modbus register type has its own parsing strategy
- **Home Assistant Integration**: Auto-discovery and automatic sensor publishing
- **Automatic Retry Logic**: Infinite retry with configurable delay if MQTT broker is unavailable
- **External Configuration**: Complete configuration through YAML file
- **Comprehensive Logging**: Detailed monitoring of all operations
- **Graceful Shutdown**: Safe shutdown with complete cleanup

### Multi-Device Support (V2.1+)

- **Device-Based Configuration**: Organize multiple Modbus devices with segregated metadata, RTU, Modbus, and Home Assistant sections
- **Unique Device Validation**: Automatic validation of device keys, slave IDs, and Home Assistant device IDs
- **Flexible Device IDs**: Optional `homeassistant.device_id` with automatic fallback to device key
- **Per-Device Settings**: Individual poll intervals, manufacturer/model overrides, and register groups per device
- **Scalable Architecture**: Support for multiple energy meters, inverters, or other Modbus devices on the same RTU bus

See [Configuration Documentation](docs/CONFIG.md) for details.

## Project Structure

```md
├── src/                           # Source code directory
│   ├── main.go                        # CLI application entry point (creates executable)
│   ├── pkg/                           # Public packages (renamed from internal for testing)
│   │   ├── config/                    # Configuration management
│   │   │   └── config.go
│   │   ├── modbus/                    # Command Pattern for Modbus operations
│   │   │   ├── interfaces.go          # ModbusCommand interface and Gateway interface
│   │   │   ├── types.go               # Common types (CommandResult, CommandError)
│   │   │   ├── executor.go            # Command executor implementation
│   │   │   ├── factory.go             # Command factory for creating commands
│   │   │   ├── base_command.go        # Base command with common functionality
│   │   │   ├── voltage_command.go     # Voltage reading command
│   │   │   ├── frequency_command.go   # Frequency reading command
│   │   │   ├── current_command.go     # Current reading command
│   │   │   ├── energy_command.go      # Energy reading command
│   │   │   ├── power_command.go       # Power reading command
│   │   │   ├── power_factor_command.go       # Power factor reading command
│   │   │   ├── reactive_power_command.go     # Reactive power calculation command
│   │   │   └── groups/                # Grouped register reading
│   │   │       ├── group_strategy.go  # GroupStrategy interface
│   │   │       ├── instant_group.go   # Instant values group (voltage, current, power, frequency)
│   │   │       ├── energy_group.go    # Energy values group
│   │   │       └── group_executor.go  # Group execution orchestrator
│   │   ├── gateway/                   # USR-DR164 MQTT Gateway
│   │   │   └── gateway.go
│   │   ├── mqtt/                      # MQTT topic management
│   │   │   ├── publisher.go           # MQTT publisher
│   │   │   ├── topics.go              # Topic definitions
│   │   │   └── *_topic.go             # Individual topic handlers
│   │   └── logger/                    # Logging utilities
│   │       └── logger.go
│   ├── main.go                        # Application initialization
│   ├── go.mod                         # Go module definition
│   └── go.sum                         # Go dependencies
├── tests/                             # Tests (separate module from src)
│   ├── unit/                          # Unit tests
│   │   ├── modbus_commands_test.go    # Command parsing tests
│   │   ├── config_test.go             # Configuration tests
│   │   └── factory_test.go            # Factory pattern tests
│   ├── integration/                   # Integration tests
│   │   └── groups_integration_test.go # Group execution tests
│   ├── go.mod                         # Test module definition
│   └── README.md                      # Testing documentation
├── .github/                           # GitHub Actions workflows
│   └── workflows/
│       └── go-tests.yml               # CI/CD pipeline
├── config-sample.yaml                 # Sample configuration file
├── run-tests.sh                       # Test runner (bash)
├── run-tests.ps1                      # Test runner (PowerShell)
└── README.md                          # This file
```

## SOLID Principles Implementation

### 1. Single Responsibility Principle (SRP)

- `CommandExecutor`: Only executes Modbus commands
- `USRGateway`: Only handles MQTT communication with gateway
- `Publisher`: Only publishes to Home Assistant
- `Config`: Only manages configuration

### 2. Open/Closed Principle (OCP)

- `ModbusCommand`: Interface open for extensions, closed for modifications
- New register types can be added without modifying existing code

### 3. Liskov Substitution Principle (LSP)

- All commands implement `ModbusCommand` and are interchangeable
- `BaseCommand` can replace any specific command

### 4. Interface Segregation Principle (ISP)

- `Gateway`: Minimal interface for gateway communication
- `ModbusCommand`: Specific interface for Modbus commands

### 5. Dependency Inversion Principle (DIP)

- `CommandExecutor` depends on `Gateway` interface, not implementation
- `CommandFactory` creates commands based on interface

## Installation

```bash
# Clone the repository
git clone <repo-url>
cd chint-mqtt-modbus-bridge

# Navigate to source directory
cd src

# Install dependencies
go mod tidy

# Compile application
go build -o mqtt-modbus-bridge .

# Copy binary to binaries location
cp mqtt-modbus-bridge /usr/local/bin/
```

## Testing

The project includes comprehensive unit and integration tests organized in separate directories.

### Test Structure

Tests are organized into two categories:

- **Unit Tests** (`tests/unit/`): Test individual components in isolation
  - `modbus_commands_test.go` - Command parsing and validation tests
  - `config_test.go` - Configuration loading and validation tests
  - `factory_test.go` - Factory pattern and command creation tests

- **Integration Tests** (`tests/integration/`): Test component interactions
  - `groups_integration_test.go` - Group execution and reactive power calculation tests

### Running Tests

#### All Tests

```bash
# Using convenience scripts (recommended)
./run-tests.sh      # Linux/macOS
./run-tests.ps1     # Windows PowerShell

# Or manually
cd tests
go test ./... -v
```

#### Specific Test Categories

```bash
# Unit tests only
cd tests
go test ./unit/... -v

# Integration tests only
cd tests
go test ./integration/... -v
```

#### Test Coverage

```bash
# Generate coverage report
cd tests
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# View coverage per package
go test ./unit/... -cover
go test ./integration/... -cover

# Detailed coverage information
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out
```

### Continuous Integration

GitHub Actions automatically runs all tests on every push and pull request.

The CI/CD workflow includes:

- ✅ Unit tests with coverage reporting
- ✅ Integration tests with coverage reporting
- ✅ Code linting with golangci-lint
- ✅ Security scanning with gosec
- ✅ Build verification
- ✅ Format checking

View the workflow status in the **Actions** tab of the repository.

### Test Coverage Summary

Current test coverage:

- **Unit Tests**: 22 tests covering command parsing, factory patterns, and configuration
- **Integration Tests**: 4 tests covering group execution and reactive power calculation
- **Total**: 26 tests, all passing ✅

For more detailed testing documentation, see [tests/README.md](tests/README.md).

## Documentation

Comprehensive documentation is available in the `docs/` directory:

### Getting Started

- **[Configuration Reference](docs/CONFIG.md)** - Complete configuration format documentation (V2.0 and V2.1)
- **[Multi-Device Support](docs/MULTI_DEVICE.md)** - Setting up multiple Modbus devices (V2.1+)
- **[Migration Guide](docs/MIGRATION.md)** - Upgrading from V1, V2.0, or single-device to multi-device

### Technical Reference

- **[Validation Rules](docs/VALIDATION.md)** - Configuration validation rules and examples
- **[CRC Implementation](docs/CRC.md)** - Modbus CRC calculation details
- **[Function Codes](docs/FUNCTION_CODE.md)** - Supported Modbus function codes
- **[Reactive Power Calculation](docs/REACTIVE_POWER_CALCULATION.md)** - Power calculations

## Configuration

Create a configuration file in one of these locations:

- `/etc/mqtt-modbus-bridge/config.yaml`
- `/etc/mqtt-modbus-bridge.yaml`
- `./config.yaml`

You can find a sample configuration file in the project directory as `config-sample.yaml`. Copy this file to your desired location and modify it according to your setup requirements.

## Usage

```bash
# Run with default configuration
./mqtt-modbus-bridge

# Run with custom configuration
./mqtt-modbus-bridge /path/to/config.yaml

# Check that it's running correctly
./mqtt-modbus-bridge 2>&1 | tee mqtt-modbus-bridge.log
```

## Monitoring

The application logs all operations:

```md
🚀 Starting MQTT-Modbus Bridge...
✅ Strategy registered: voltage (Voltage)
✅ Strategy registered: frequency (Frequency)
✅ Gateway connected to MQTT broker
📡 Gateway subscribed to: D4AD20B75646/data
✅ HA Publisher connected to MQTT broker
🔍 Publishing discovery configurations for Home Assistant...
📡 Publishing discovery for Voltage: homeassistant/sensor/energy_meter_001_voltage/config
✅ MQTT-Modbus Bridge started successfully
📊 Starting register reading...
📤 Sending command: 01030800000200C5C3 to D4AD20B75646/cmd
📥 Response received: 010304436666664E6C
📈 Voltage: 230.600 V
📊 Publishing state for Voltage: 230.600 V
```

## Home Assistant Integration

The bridge automatically publishes sensors in Home Assistant through MQTT Discovery:

### Energy Sensors

```yaml
# Sensor automatically created in Home Assistant
sensor:
  - platform: mqtt
    name: "Voltage"
    state_topic: "sensor/energy_meter/voltage/state"
    unit_of_measurement: "V"
    device_class: "voltage"
    value_template: "{{ value_json.value }}"
    availability_topic: "modbus-bridge/status"
    payload_available: "online"
    payload_not_available: "offline"
```

### Diagnostic and Status Monitoring

The bridge also creates a diagnostic sensor that appears in the Home Assistant logbook:

- **Diagnostic Sensor**: Shows error codes and human-readable messages (hidden by default, can be enabled manually)
- **Availability Status**: All sensors show as "unavailable" when the gateway is offline  
- **Logbook Integration**: Status changes and errors are logged with timestamps

The diagnostic sensor is configured with `entity_category: "diagnostic"`, which means it won't appear in the main interface but can be found in the device settings and enabled if needed.

**Diagnostic Messages Include:**

- Gateway connection status
- Modbus communication errors
- Configuration errors
- Recovery notifications

**MQTT Topics:**

- Status: `modbus-bridge/status` (online/offline)
- Diagnostic: `modbus-bridge/diagnostic` (error codes with timestamps)

## MQTT Broker Connection & Retry Logic

The application implements robust connection handling for MQTT broker connectivity:

### Automatic Retry Configuration

If the MQTT broker is not available at startup, the application will retry connecting indefinitely at configurable intervals:

```yaml
mqtt:
  broker: "localhost"
  port: 1883
  username: "mqtt"
  password: "mqtt_password"
  client_id: "modbus-bridge"
  retry_delay: 5000  # Retry delay in milliseconds (default: 5000ms = 5 seconds)
```

### Connection Behavior

- **Startup**: If broker is unavailable, application logs attempts and retries every `retry_delay` milliseconds
- **Runtime**: Built-in auto-reconnect handles temporary disconnections
- **Logging**: Each connection attempt is logged with attempt number and timing
- **Graceful**: Application can be stopped anytime during retry attempts with Ctrl+C

### Example Output

```text
🔄 Attempting to connect gateway to MQTT broker (attempt 1)...
❌ Gateway connection failed (attempt 1): network Error : dial tcp 127.0.0.1:1883: connectex: No connection could be made because the target machine actively refused it.
⏳ Retrying in 5 seconds...
🔄 Attempting to connect HA publisher to MQTT broker (attempt 1)...
❌ HA Publisher connection failed (attempt 1): network Error : dial tcp 127.0.0.1:1883: connectex: No connection could be made because the target machine actively refused it.
⏳ Retrying in 5 seconds...
...
✅ Gateway successfully connected to MQTT broker after 3 attempts
✅ HA Publisher successfully connected to MQTT broker after 3 attempts
```

## MQTT Filtering and Data Processing

The bridge implements sophisticated MQTT message filtering and processing to ensure data reliability and accuracy:

### Message Filtering

The gateway automatically filters incoming MQTT messages to process only valid input data:

- **Topic Filtering**: Only processes messages from the configured `data_topic` (e.g., `D4AD20B75646/data`)
- **Protocol Filtering**: Validates Modbus RTU response format:
  - Checks for valid function code (0x03 - Read Holding Registers)
  - Verifies minimum message length (5 bytes)
  - Extracts payload data starting from byte position 3
  - Ignores malformed or incomplete messages

### Response Processing

```go
// Example: Processing a valid Modbus response
// Input:  [01 03 04 43 66 66 66 4E 6C] (9 bytes)
// Output: [43 66 66 66] (4 bytes of actual data)
```

### Error Handling

- **Channel Management**: Uses buffered channels to prevent blocking on response processing
- **Overflow Protection**: Automatically discards responses when processing queue is full
- **Logging**: Comprehensive logging of all filtering decisions and data transformations

## Data Validation

The bridge implements multi-layer data validation to ensure data quality and system reliability:

### Input Validation

**Numeric Validation:**

- Checks for `NaN` (Not a Number) values
- Detects infinite values (`+Inf`, `-Inf`)
- Validates minimum data length requirements (4 bytes for float32)

**Field Validation:**

- Ensures required fields (name, topic, unit) are present
- Validates topic format and structure
- Checks device class and state class values

### Range Validation

**Configuration-Based Limits:**

```yaml
registers:
  voltage:
    min: 100.0      # Minimum acceptable voltage
    max: 300.0      # Maximum acceptable voltage
```

**Dynamic Quality Assessment:**

- **Voltage Quality**: Categorizes readings as excellent (220-240V), good (210-250V), acceptable (180-270V), or poor
- **Stability Checks**: Identifies unstable voltage conditions outside normal ranges
- **Threshold Alerts**: Logs warnings when values exceed configured thresholds

### Validation Error Handling

**Error Responses:**

```text
❌ Validation failed: voltage value 350.5 V above maximum threshold 300.0 V
❌ Validation failed: current value is NaN for sensor Current_L1
❌ Validation failed: sensor name is empty
```

**Recovery Actions:**

- Skips publishing invalid data to prevent Home Assistant errors
- Maintains diagnostic information for troubleshooting
- Continues processing other valid sensors
- Logs detailed error information for analysis

### Topic-Specific Validation

Each sensor type implements custom validation rules:

- **Voltage**: Range validation (100-300V), stability assessment
- **Current**: Non-negative validation, overload detection
- **Power**: Calculation validation, power factor correlation
- **Energy**: Monotonic increase validation, consumption rate limits
- **Frequency**: Grid frequency validation (45-65Hz)

## Grouped Register Reading (Performance Optimization)

The bridge implements a **Group Strategy Pattern** for reading multiple Modbus registers in a single query, significantly improving performance and reducing communication overhead.

### How It Works

Instead of querying each register individually, the group strategy:

1. **Calculates Address Range**: Determines the minimum and maximum register addresses in the group
2. **Single Query**: Sends one Modbus read command for the entire range
3. **Smart Parsing**: Uses each register's specific `ParseData()` method to parse its portion of the response
4. **Preserves Validation**: All validation logic from individual commands is maintained

### Performance Improvement

**Before (Individual Queries):**

- 6 separate queries for instant values (voltage, current, power_active, power_apparent, power_factor, frequency)
- 3 separate queries for energy values (energy_total, energy_imported, energy_exported)
- Delays between each query
- **Total: ~9 Modbus queries**

**After (Grouped Queries):**

- 1 query for all instant values
- 1 query for all energy values  
- 1 query for calculated values (power_reactive)
- **Total: ~3 Modbus queries**

**Result: ~67% reduction in Modbus traffic!**

### Register Groups

#### InstantGroup

Reads instant measurement values in a single query:

- Voltage
- Current
- Active Power
- Apparent Power
- Power Factor
- Frequency

#### EnergyGroup

Reads energy accumulation values in a single query:

- Total Energy
- Imported Energy
- Exported Energy

### Implementation Details

The group executor:

1. **Registry-Based**: Maintains a map of register names to their commands
2. **Offset Calculation**: Calculates each register's offset in the raw data: `offset = (registerAddress - minAddress) * 2`
3. **Command Reuse**: Uses existing `ParseData()` methods from individual commands
4. **Automatic Fallback**: If group reading fails, automatically falls back to individual register reads

### Example Log Output

```text
✅ Instant group created with 6 registers: [voltage current power_active power_apparent power_factor frequency]
✅ Energy group created with 3 registers: [energy_total energy_imported energy_exported]
📊 Using InstantGroup for optimized batch reading
✅ 📊 Normal: Successfully read and published 6 registers in single query
⚡ Using EnergyGroup for optimized batch reading
✅ ⚡ Energy: Successfully read and published 3 registers in single query
```

### Architecture

```text
GroupStrategy (Interface)
    ├── Execute(ctx, gateway, slaveID) -> rawData
    ├── ParseResults(rawData) -> map[string]float64
    └── GetNames() -> []string

InstantGroup (Implementation)
    ├── Registers: []config.Register
    ├── CommandRegistry: map[string]ModbusCommand
    └── Uses commands' ParseData() for each register

EnergyGroup (Implementation)
    ├── Registers: []config.Register
    ├── CommandRegistry: map[string]ModbusCommand
    └── Uses commands' ParseData() for each register

GroupExecutor
    ├── CreateInstantGroup()
    ├── CreateEnergyGroup()
    └── ExecuteGroup()
```

### Benefits

1. **Performance**: Dramatically reduced query count and communication overhead
2. **Code Reuse**: Leverages existing `ParseData()` methods - no code duplication
3. **Maintainability**: Changes to parsing logic automatically apply to grouped reads
4. **Type Safety**: Full compile-time type checking via Go interfaces
5. **Reliability**: Automatic fallback to individual reads if group reading fails
6. **Compatibility**: Works with existing configuration - no changes needed

### Monitoring Performance

To see the optimization in action, set logging level to `trace`:

```yaml
# config.yaml
logging:
  level: "trace"
```

Look for these log messages:

- `📊 Using InstantGroup for optimized batch reading` - Grouped read initiated
- `✅ Successfully read and published X registers in single query` - Grouped read succeeded
- `⚠️ Group read failed, falling back to individual register reads` - Fallback triggered (rare)

## Development

To add a new register type:

1. Create a new command file in `internal/modbus/` (e.g., `temperature_command.go`)
2. Implement the `ModbusCommand` interface
3. Update `CommandFactory.CreateCommand()` in `factory.go`
4. Add the register in `config.yaml`

```go
// Example new command - temperature_command.go
package modbus

import (
    "encoding/binary"
    "fmt"
    "math"
    "mqtt-modbus-bridge/internal/config"
)

type TemperatureCommand struct {
    *BaseCommand
}

func NewTemperatureCommand(register config.Register, slaveID uint8) *TemperatureCommand {
    return &TemperatureCommand{
        BaseCommand: NewBaseCommand(register, slaveID),
    }
}

func (c *TemperatureCommand) ParseData(rawData []byte) (float64, error) {
    // Temperature-specific implementation
    if len(rawData) < 4 {
        return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
    }
    
    bits := binary.BigEndian.Uint32(rawData[:4])
    value := math.Float32frombits(bits)
    
    return float64(value), nil
}
```

### Command Pattern Benefits

The new file structure provides:

- **Single Responsibility**: Each command file handles one register type
- **Open/Closed Principle**: Add new commands without modifying existing code
- **Easy Testing**: Each command can be tested independently
- **Clear Organization**: Interfaces, types, and implementations are clearly separated
- **Maintainability**: Changes to one command don't affect others

## Build and Compilation

```bash
# Test compilation
cd src
go build .

# Run linter checks
go fmt ./...
go vet ./...
```

## License

MIT License
