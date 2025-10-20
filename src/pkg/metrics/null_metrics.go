package metrics

import "time"

// NullMetrics is a zero-overhead no-op implementation of MetricsCollector.
// Use this when metrics are disabled (metrics_port = 0) to avoid any
// performance overhead from metrics collection.
//
// All methods are no-ops and will be optimized away by the compiler.
type NullMetrics struct{}

// NewNullMetrics creates a new NullMetrics instance
func NewNullMetrics() *NullMetrics {
	return &NullMetrics{}
}

// IncrementModbusReads is a no-op
func (nm *NullMetrics) IncrementModbusReads() {}

// IncrementModbusErrors is a no-op
func (nm *NullMetrics) IncrementModbusErrors() {}

// IncrementMQTTPublishes is a no-op
func (nm *NullMetrics) IncrementMQTTPublishes() {}

// IncrementMQTTErrors is a no-op
func (nm *NullMetrics) IncrementMQTTErrors() {}

// SetGatewayStatus is a no-op
func (nm *NullMetrics) SetGatewayStatus(online bool) {}

// ObserveModbusReadDuration is a no-op
func (nm *NullMetrics) ObserveModbusReadDuration(duration time.Duration) {}

// StartMetricsServer is a no-op (always returns nil)
func (nm *NullMetrics) StartMetricsServer(port int) error {
	return nil
}

// Compile-time verification that NullMetrics implements MetricsCollector
var _ MetricsCollector = (*NullMetrics)(nil)
