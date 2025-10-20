package mqtt

import (
	"context"
	"mqtt-modbus-bridge/pkg/modbus"
)

// PublisherInterface defines the contract for device diagnostic publishing
// This allows the diagnostics package to depend on an interface rather than concrete Publisher
type PublisherInterface interface {
	// PublishDeviceDiagnosticDiscovery publishes discovery configuration for per-device diagnostic sensor
	PublishDeviceDiagnosticDiscovery(ctx context.Context, deviceID string, deviceInfo *DeviceInfo) error

	// PublishDeviceDiagnosticState publishes state for per-device diagnostic sensor
	PublishDeviceDiagnosticState(ctx context.Context, deviceID string, metrics *DeviceMetrics) error
}

// SensorPublisher handles sensor-related MQTT publishing
// Interface Segregation Principle - only sensor operations
type SensorPublisher interface {
	// PublishSensorDiscovery publishes Home Assistant discovery config for a sensor
	PublishSensorDiscovery(ctx context.Context, result *modbus.CommandResult, deviceInfo *DeviceInfo) error

	// PublishSensorState publishes sensor state value to MQTT
	PublishSensorState(ctx context.Context, result *modbus.CommandResult) error

	// PublishAllDiscoveries publishes discovery configs for all sensors
	PublishAllDiscoveries(ctx context.Context, results []*modbus.CommandResult, deviceInfo *DeviceInfo) error
}

// StatusPublisher handles status-related MQTT publishing
// Interface Segregation Principle - only status operations
type StatusPublisher interface {
	// PublishStatus publishes a status message (online/offline)
	PublishStatus(ctx context.Context, status string) error

	// PublishStatusOnline publishes online status
	PublishStatusOnline(ctx context.Context) error

	// PublishStatusOffline publishes offline status
	PublishStatusOffline(ctx context.Context) error
}

// DiagnosticPublisher handles diagnostic-related MQTT publishing
// Interface Segregation Principle - only diagnostic operations
type DiagnosticPublisher interface {
	// PublishDiagnostic publishes a diagnostic message with code
	PublishDiagnostic(ctx context.Context, code int, message string) error

	// PublishDiagnosticDiscovery publishes Home Assistant discovery for diagnostics
	PublishDiagnosticDiscovery(ctx context.Context) error

	// PublishDeviceDiagnosticDiscovery publishes discovery for device-specific diagnostics
	PublishDeviceDiagnosticDiscovery(ctx context.Context, deviceID string, deviceInfo *DeviceInfo) error

	// PublishDeviceDiagnosticState publishes diagnostic state for a device
	PublishDeviceDiagnosticState(ctx context.Context, deviceID string, metrics *DeviceMetrics) error
}

// ConnectionManager handles MQTT connection lifecycle
// Interface Segregation Principle - only connection operations
type ConnectionManager interface {
	// Connect establishes connection to MQTT broker
	Connect(ctx context.Context) error

	// Disconnect closes connection to MQTT broker
	Disconnect()
}

// HAPublisher is a composite interface for components that need all publishing capabilities
// Use this when you need the full publisher (e.g., main application)
type HAPublisher interface {
	SensorPublisher
	StatusPublisher
	DiagnosticPublisher
	ConnectionManager
}

// Verify Publisher implements all interfaces at compile time
var (
	_ SensorPublisher     = (*Publisher)(nil)
	_ StatusPublisher     = (*Publisher)(nil)
	_ DiagnosticPublisher = (*Publisher)(nil)
	_ ConnectionManager   = (*Publisher)(nil)
	_ HAPublisher         = (*Publisher)(nil)
	_ PublisherInterface  = (*Publisher)(nil)
)
