# Diagnostics Package

This package provides device-level diagnostic tracking and health monitoring for MQTT-Modbus bridge devices.

## Architecture

The diagnostics package follows **Dependency Inversion Principle** by depending on interfaces rather than concrete implementations:

- **DeviceManager**: Manages device metrics, state calculation, and publishing logic
- **PublisherInterface** (from mqtt package): Interface for publishing diagnostic data

## Components

### DeviceManager

Stateful manager that:

- Tracks metrics per device (success rate, errors, response times)
- Calculates device health state (operational/warning/error/offline)
- Publishes diagnostic state based on intervals and state changes
- Thread-safe with mutex protection

**Public API:**

- `NewDeviceManager()` - Constructor with dependency injection
- `RecordSuccess(deviceID, responseTime)` - Track successful reads
- `RecordError(deviceID, errorMsg)` - Track failed reads
- `StartDiagnosticsLoop(ctx)` - Start periodic publishing
- `PublishDiscoveryForAllDevices(ctx)` - Publish HA discovery configs
- `GetMetrics(deviceID)` - Get metrics for testing/debugging

## Device States

1. **operational**: Success rate > 90%, no consecutive errors
2. **warning**: Success rate 80-90% OR 3-4 consecutive errors
3. **error**: Success rate < 80% OR 5+ consecutive errors
4. **offline**: No response for 30+ seconds

## Publishing Strategy

- **Immediate**: On state change (if enabled in config)
- **Periodic**: Based on current state
  - operational: 60s
  - warning: 30s
  - error: 5s
  - offline: 60s

## Dependencies

- **mqtt.PublisherInterface**: For publishing diagnostic data (interface)
- **mqtt.DeviceMetrics**: Metrics structure (owned by mqtt package)
- **config**: Configuration structures
- **logger**: Logging utilities

## Design Principles

- **Single Responsibility**: Only handles device diagnostics
- **Dependency Inversion**: Depends on PublisherInterface, not concrete Publisher
- **Encapsulation**: Internal state (maps, mutex) hidden from consumers
- **Thread Safety**: Mutex protection for concurrent access

## Usage Example

```go
import (
    "mqtt-modbus-bridge/pkg/diagnostics"
    "mqtt-modbus-bridge/pkg/mqtt"
    "mqtt-modbus-bridge/pkg/config"
)

// Create manager
manager := diagnostics.NewDeviceManager(
    publisher, // mqtt.PublisherInterface
    &cfg.HomeAssistant.DeviceDiagnostics,
    cfg.Devices,
)

// Start publishing loop
go manager.StartDiagnosticsLoop(ctx)

// Publish discovery configs
manager.PublishDiscoveryForAllDevices(ctx)

// Record device activity
manager.RecordSuccess("device1", 50*time.Millisecond)
manager.RecordError("device2", "timeout error")
```

## Why Separate Package?

Originally, `DeviceDiagnosticManager` was in the `mqtt` package, but it was moved to its own package because:

1. **Different responsibilities**: Topic handlers are stateless publishers, manager is stateful business logic
2. **Avoid circular dependencies**: Manager uses Publisher, so it can't be in same package
3. **Consistency**: Other components (gateway, executor) are also separate packages
4. **Testability**: Can mock PublisherInterface for unit testing
5. **Separation of concerns**: MQTT package handles publishing, diagnostics handles health tracking

## Related Packages

- **mqtt**: Contains topic handlers and Publisher implementation
- **mqtt/device_diagnostic_topic.go**: Stateless handler for publishing diagnostic messages
- **config**: Configuration structures for diagnostic settings
