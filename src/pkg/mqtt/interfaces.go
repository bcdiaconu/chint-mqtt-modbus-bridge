package mqtt

import "context"

// PublisherInterface defines the contract for device diagnostic publishing
// This allows the diagnostics package to depend on an interface rather than concrete Publisher
type PublisherInterface interface {
	// PublishDeviceDiagnosticDiscovery publishes discovery configuration for per-device diagnostic sensor
	PublishDeviceDiagnosticDiscovery(ctx context.Context, deviceID string, deviceInfo *DeviceInfo) error

	// PublishDeviceDiagnosticState publishes state for per-device diagnostic sensor
	PublishDeviceDiagnosticState(ctx context.Context, deviceID string, metrics *DeviceMetrics) error
}
