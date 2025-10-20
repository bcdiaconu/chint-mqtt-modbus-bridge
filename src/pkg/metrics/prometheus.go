package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// PrometheusMetrics tracks application metrics in Prometheus format
type PrometheusMetrics struct {
	// Counters
	modbusReadsTotal   int64
	modbusErrorsTotal  int64
	mqttPublishesTotal int64
	mqttErrorsTotal    int64

	// Gauges
	gatewayStatus int64 // 1 = online, 0 = offline

	// Histograms (simplified - store sum and count for average)
	modbusReadDurationSum   float64
	modbusReadDurationCount int64

	mu sync.RWMutex
}

// NewPrometheusMetrics creates a new Prometheus metrics collector
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		gatewayStatus: 1, // Start as online
	}
}

// IncrementModbusReads increments the Modbus read counter
func (pm *PrometheusMetrics) IncrementModbusReads() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.modbusReadsTotal++
}

// IncrementModbusErrors increments the Modbus error counter
func (pm *PrometheusMetrics) IncrementModbusErrors() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.modbusErrorsTotal++
}

// IncrementMQTTPublishes increments the MQTT publish counter
func (pm *PrometheusMetrics) IncrementMQTTPublishes() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.mqttPublishesTotal++
}

// IncrementMQTTErrors increments the MQTT error counter
func (pm *PrometheusMetrics) IncrementMQTTErrors() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.mqttErrorsTotal++
}

// SetGatewayStatus sets the gateway status (1 = online, 0 = offline)
func (pm *PrometheusMetrics) SetGatewayStatus(online bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if online {
		pm.gatewayStatus = 1
	} else {
		pm.gatewayStatus = 0
	}
}

// ObserveModbusReadDuration records a Modbus read duration
func (pm *PrometheusMetrics) ObserveModbusReadDuration(duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	seconds := duration.Seconds()
	pm.modbusReadDurationSum += seconds
	pm.modbusReadDurationCount++
}

// GetMetricsText returns metrics in Prometheus text format
func (pm *PrometheusMetrics) GetMetricsText() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var avgReadDuration float64
	if pm.modbusReadDurationCount > 0 {
		avgReadDuration = pm.modbusReadDurationSum / float64(pm.modbusReadDurationCount)
	}

	return fmt.Sprintf(`# HELP modbus_reads_total Total number of Modbus read operations
# TYPE modbus_reads_total counter
modbus_reads_total %d

# HELP modbus_errors_total Total number of Modbus read errors
# TYPE modbus_errors_total counter
modbus_errors_total %d

# HELP mqtt_publishes_total Total number of MQTT publish operations
# TYPE mqtt_publishes_total counter
mqtt_publishes_total %d

# HELP mqtt_errors_total Total number of MQTT publish errors
# TYPE mqtt_errors_total counter
mqtt_errors_total %d

# HELP gateway_status Current gateway status (1 = online, 0 = offline)
# TYPE gateway_status gauge
gateway_status %d

# HELP modbus_read_duration_seconds Average Modbus read duration in seconds
# TYPE modbus_read_duration_seconds gauge
modbus_read_duration_seconds %.6f

# HELP modbus_read_duration_count Total number of Modbus read duration observations
# TYPE modbus_read_duration_count counter
modbus_read_duration_count %d
`,
		pm.modbusReadsTotal,
		pm.modbusErrorsTotal,
		pm.mqttPublishesTotal,
		pm.mqttErrorsTotal,
		pm.gatewayStatus,
		avgReadDuration,
		pm.modbusReadDurationCount,
	)
}

// ServeHTTP implements http.Handler interface for /metrics endpoint
func (pm *PrometheusMetrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, pm.GetMetricsText())
}

// StartMetricsServer starts an HTTP server on the given port to expose metrics
// Implements secure defaults with timeouts to prevent slowloris attacks
func (pm *PrometheusMetrics) StartMetricsServer(port int) error {
	http.Handle("/metrics", pm)
	addr := fmt.Sprintf(":%d", port)

	// Create server with secure timeout settings (gosec G114)
	server := &http.Server{
		Addr:              addr,
		ReadTimeout:       15 * time.Second, // Max time to read request
		ReadHeaderTimeout: 10 * time.Second, // Max time to read headers
		WriteTimeout:      15 * time.Second, // Max time to write response
		IdleTimeout:       60 * time.Second, // Max time for keep-alive connections
	}

	return server.ListenAndServe()
}

// GetStats returns current metric values
func (pm *PrometheusMetrics) GetStats() MetricStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var avgDuration float64
	if pm.modbusReadDurationCount > 0 {
		avgDuration = pm.modbusReadDurationSum / float64(pm.modbusReadDurationCount)
	}

	return MetricStats{
		ModbusReadsTotal:        pm.modbusReadsTotal,
		ModbusErrorsTotal:       pm.modbusErrorsTotal,
		MQTTPublishesTotal:      pm.mqttPublishesTotal,
		MQTTErrorsTotal:         pm.mqttErrorsTotal,
		GatewayOnline:           pm.gatewayStatus == 1,
		AvgModbusReadDuration:   avgDuration,
		ModbusReadDurationCount: pm.modbusReadDurationCount,
	}
}

// MetricStats represents current metric statistics
type MetricStats struct {
	ModbusReadsTotal        int64
	ModbusErrorsTotal       int64
	MQTTPublishesTotal      int64
	MQTTErrorsTotal         int64
	GatewayOnline           bool
	AvgModbusReadDuration   float64
	ModbusReadDurationCount int64
}
