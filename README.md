# chint-mqtt-modbus-bridge

A robust and production-ready bridge for integrating a Chint DDSU666-H energy meter (Modbus RTU) with Home Assistant, using a PUSR USR-DR164 Modbus-MQTT gateway. This project enables seamless, real-time monitoring and automation of energy data in Home Assistant via MQTT Discovery. Designed for reliability, flexibility, and easy extension.

**Supported Hardware:**

- **Energy Meter:** Chint DDSU666-H (Modbus RTU)
- **Gateway:** PUSR USR-DR164 (Modbus RTU <-> MQTT)

---

A robust bridge between USR-DR164 Modbus-MQTT Gateway and Home Assistant, implemented in Go with SOLID principles.

## Features

- **SOLID Architecture**: Follows all SOLID principles for easy maintenance
- **Strategy Pattern**: Each Modbus register type has its own parsing strategy
- **Home Assistant Integration**: Auto-discovery and automatic sensor publishing
- **Automatic Retry Logic**: Infinite retry with configurable delay if MQTT broker is unavailable
- **External Configuration**: Complete configuration through YAML file
- **Comprehensive Logging**: Detailed monitoring of all operations
- **Graceful Shutdown**: Safe shutdown with complete cleanup

## Architecture

```
├── cmd/
│   └── main.go                    # Main application
├── internal/
│   ├── config/                    # Configuration management
│   │   └── config.go
│   ├── modbus/                    # Strategy Pattern for Modbus commands
│   │   ├── strategy.go            # Interfaces and executor
│   │   └── strategies.go          # Concrete implementations
│   ├── mqtt/                      # USR-DR164 Gateway
│   │   └── gateway.go
│   └── homeassistant/             # Home Assistant Publisher
│       └── publisher.go
└── config.yaml                   # Application configuration
```

## SOLID Principles Implementation

### 1. Single Responsibility Principle (SRP)

- `CommandExecutor`: Only executes Modbus commands
- `USRGateway`: Only handles MQTT communication with gateway
- `Publisher`: Only publishes to Home Assistant
- `Config`: Only manages configuration

### 2. Open/Closed Principle (OCP)

- `CommandStrategy`: Interface open for extensions, closed for modifications
- New register types can be added without modifying existing code

### 3. Liskov Substitution Principle (LSP)

- All strategies implement `CommandStrategy` and are interchangeable
- `BaseStrategy` can replace any specific strategy

### 4. Interface Segregation Principle (ISP)

- `Gateway`: Minimal interface for gateway communication
- `CommandStrategy`: Specific interface for Modbus commands

### 5. Dependency Inversion Principle (DIP)

- `CommandExecutor` depends on `Gateway` interface, not implementation
- `StrategyFactory` creates strategies based on interface

## Installation

```bash
# Clone the repository
git clone <repo-url>
cd mqtt-modbus-bridge

# Install dependencies
go mod tidy

# Compile application
go build -o mqtt-modbus-bridge ./cmd/main.go

# Copy binary to binaries location
cp mqtt-modbus-bridge /usr/local/bin/
```

## Configuration

Create a configuration file in one of these locations:

- `/etc/mqtt-modbus-bridge/config.yaml`
- `/etc/mqtt-modbus-bridge.yaml`
- `./config.yam`

You can find a sample configuration file in the project directory as `config-sample.yaml`.

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

```
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

## Development

To add a new register type:

1. Create a new strategy in `strategies.go`
2. Update `StrategyFactory.CreateStrategy()`
3. Add the register in `config.yaml`

```go
// Example new strategy
type TemperatureStrategy struct {
    *BaseStrategy
}

func (s *TemperatureStrategy) ParseData(rawData []byte) (float64, error) {
    // Temperature-specific implementation
}
```

## Testing

```bash
# Test compilation
go build ./...

# Run tests (when added)
go test ./...

# Check formatting
go fmt ./...

# Check with go vet
go vet ./...
```

## License

MIT License
