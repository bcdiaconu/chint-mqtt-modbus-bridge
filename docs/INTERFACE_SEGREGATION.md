# MQTT Publisher Interface Segregation

## Overview

The MQTT publisher has been refactored to follow the **Interface Segregation Principle (ISP)**. Instead of a single monolithic interface, we now have four domain-specific interfaces that components can depend on based on their needs.

## Interfaces

### 1. SensorPublisher

**Purpose**: Publishing sensor data and discovery configurations

**Methods**:

- `PublishSensorDiscovery()` - Publish HA discovery config for a sensor
- `PublishSensorState()` - Publish sensor state value
- `PublishAllDiscoveries()` - Publish all sensor discovery configs

**Use Case**: Components that only need to publish sensor data (e.g., polling service)

```go
type PollingService struct {
    sensorPub mqtt.SensorPublisher  // Only depends on sensor operations
}
```

### 2. StatusPublisher

**Purpose**: Publishing online/offline status

**Methods**:

- `PublishStatus()` - Publish arbitrary status
- `PublishStatusOnline()` - Mark bridge as online
- `PublishStatusOffline()` - Mark bridge as offline

**Use Case**: Components that manage bridge availability (e.g., heartbeat service, connection manager)

```go
type HeartbeatService struct {
    statusPub mqtt.StatusPublisher  // Only depends on status operations
}
```

### 3. DiagnosticPublisher

**Purpose**: Publishing diagnostic information

**Methods**:

- `PublishDiagnostic()` - Publish diagnostic message with error code
- `PublishDiagnosticDiscovery()` - Publish HA discovery for diagnostics
- `PublishDeviceDiagnosticDiscovery()` - Publish discovery for device diagnostics
- `PublishDeviceDiagnosticState()` - Publish device diagnostic state

**Use Case**: Components that report errors and metrics (e.g., error handlers, monitoring services)

```go
type ErrorHandler struct {
    diagPub mqtt.DiagnosticPublisher  // Only depends on diagnostic operations
}
```

### 4. ConnectionManager

**Purpose**: Managing MQTT connection lifecycle

**Methods**:

- `Connect()` - Establish MQTT connection
- `Disconnect()` - Close MQTT connection

**Use Case**: Components that manage MQTT lifecycle (e.g., initialization, shutdown)

```go
type ConnectionService struct {
    connMgr mqtt.ConnectionManager  // Only depends on connection operations
}
```

### 5. HAPublisher (Composite)

**Purpose**: Full publisher interface combining all domain interfaces

**Composition**:

```go
type HAPublisher interface {
    SensorPublisher
    StatusPublisher
    DiagnosticPublisher
    ConnectionManager
}
```

**Use Case**: Main application or facades that need all functionality

```go
type Application struct {
    publisher mqtt.HAPublisher  // Needs all operations
}
```

## Benefits

### 1. **Reduced Coupling**

Components only depend on the operations they actually use. A polling service doesn't need to know about connection management or diagnostics.

### 2. **Easier Testing**

Mock implementations can be simpler:

```go
type MockSensorPublisher struct {
    publishedCount int
}

func (m *MockSensorPublisher) PublishSensorState(ctx, result) error {
    m.publishedCount++
    return nil
}

// Only need 3 methods instead of 13
```

### 3. **Better API Discovery**

IDE autocomplete shows only relevant methods based on the interface being used.

### 4. **Compile-Time Safety**

Type system ensures components can't accidentally call methods they shouldn't:

```go
// Compile error: statusPub.PublishSensorState undefined
func (s *StatusService) badMethod(ctx context.Context) {
    s.statusPub.PublishSensorState(ctx, result)  // ❌ Won't compile
}
```

### 5. **Flexible Evolution**

Each interface can evolve independently without affecting components that don't use it.

## Implementation Notes

### Current State

The `Publisher` struct implements ALL interfaces:

```go
var (
    _ SensorPublisher     = (*Publisher)(nil)
    _ StatusPublisher     = (*Publisher)(nil)
    _ DiagnosticPublisher = (*Publisher)(nil)
    _ ConnectionManager   = (*Publisher)(nil)
    _ HAPublisher         = (*Publisher)(nil)
)
```

### Migration Strategy

1. ✅ **Phase 1** (Complete): Define interfaces
2. **Phase 2** (Future): Update service constructors to accept specific interfaces
3. **Phase 3** (Future): Update tests to use interface mocks

### Example Migration

**Before**:

```go
type PollingService struct {
    publisher *mqtt.Publisher  // Depends on concrete type
}

func NewPollingService(pub *mqtt.Publisher) *PollingService {
    return &PollingService{publisher: pub}
}
```

**After**:

```go
type PollingService struct {
    sensorPub mqtt.SensorPublisher  // Depends on interface
}

func NewPollingService(pub mqtt.SensorPublisher) *PollingService {
    return &PollingService{sensorPub: pub}
}
```

## Architectural Principles Applied

1. **Interface Segregation Principle (ISP)**: Clients should not be forced to depend on interfaces they don't use
2. **Dependency Inversion Principle (DIP)**: Depend on abstractions (interfaces), not concretions
3. **Single Responsibility Principle (SRP)**: Each interface has one clear responsibility
4. **Composition over Inheritance**: HAPublisher composes other interfaces

## Testing

Run interface tests:

```bash
go test ./pkg/mqtt/... -run TestInterface
```

All tests verify:

- ✅ Publisher implements all interfaces
- ✅ Interface segregation works correctly
- ✅ Composite interface includes all methods
- ✅ Type assertions work as expected

## Future Enhancements

1. Create interface adapters for legacy code
2. Add metrics interfaces for observability
3. Consider splitting large interfaces further if needed
4. Document interface compatibility guarantees
