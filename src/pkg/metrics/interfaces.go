package metrics

import "time"

// MetricsCollector defines the interface for collecting application metrics.
// This abstraction allows for different implementations (e.g., Prometheus, StatsD, NullMetrics)
// and follows the Dependency Inversion Principle.
//
// Implementations:
//   - PrometheusMetrics: Full-featured Prometheus metrics with HTTP server
//   - NullMetrics: Zero-overhead no-op implementation when metrics are disabled
type MetricsCollector interface {
	// IncrementModbusReads increments the counter for successful Modbus read operations
	IncrementModbusReads()

	// IncrementModbusErrors increments the counter for failed Modbus read operations
	IncrementModbusErrors()

	// IncrementMQTTPublishes increments the counter for successful MQTT publish operations
	IncrementMQTTPublishes()

	// IncrementMQTTErrors increments the counter for failed MQTT publish operations
	IncrementMQTTErrors()

	// SetGatewayStatus sets the current gateway connection status
	// Parameters:
	//   - online: true if gateway is connected, false otherwise
	SetGatewayStatus(online bool)

	// ObserveModbusReadDuration records the duration of a Modbus read operation
	// Parameters:
	//   - duration: time taken to complete the Modbus read
	ObserveModbusReadDuration(duration time.Duration)

	// StartMetricsServer starts an HTTP server to expose metrics (optional for some implementations)
	// Parameters:
	//   - port: HTTP port to listen on (0 disables the server)
	// Returns:
	//   - error: nil on success, error if server fails to start
	StartMetricsServer(port int) error
}

// Compile-time verification that PrometheusMetrics implements MetricsCollector
var _ MetricsCollector = (*PrometheusMetrics)(nil)
