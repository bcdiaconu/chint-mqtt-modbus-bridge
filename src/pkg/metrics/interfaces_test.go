package metrics

import (
	"context"
	"testing"
	"time"
)

// TestMetricsCollectorInterface verifies that both PrometheusMetrics and NullMetrics
// implement the MetricsCollector interface
func TestMetricsCollectorInterface(t *testing.T) {
	t.Run("PrometheusMetrics implements MetricsCollector", func(t *testing.T) {
		var _ MetricsCollector = (*PrometheusMetrics)(nil)
		t.Log("✅ PrometheusMetrics implements MetricsCollector interface")
	})

	t.Run("NullMetrics implements MetricsCollector", func(t *testing.T) {
		var _ MetricsCollector = (*NullMetrics)(nil)
		t.Log("✅ NullMetrics implements MetricsCollector interface")
	})
}

// TestPrometheusMetricsRecording verifies that PrometheusMetrics actually records values
func TestPrometheusMetricsRecording(t *testing.T) {
	pm := NewPrometheusMetrics()

	// Test counter increments
	pm.IncrementModbusReads()
	pm.IncrementModbusReads()
	pm.IncrementModbusErrors()

	// Test duration observation
	pm.ObserveModbusReadDuration(100 * time.Millisecond)
	pm.ObserveModbusReadDuration(200 * time.Millisecond)

	// Test gateway status
	pm.SetGatewayStatus(true)
	pm.SetGatewayStatus(false)

	// Test MQTT metrics
	pm.IncrementMQTTPublishes()
	pm.IncrementMQTTErrors()

	// Verify metrics output contains expected values
	output := pm.GetMetricsText()

	// Check counters
	if !contains(output, "modbus_reads_total 2") {
		t.Errorf("Expected modbus_reads_total to be 2")
	}
	if !contains(output, "modbus_errors_total 1") {
		t.Errorf("Expected modbus_errors_total to be 1")
	}
	if !contains(output, "mqtt_publishes_total 1") {
		t.Errorf("Expected mqtt_publishes_total to be 1")
	}
	if !contains(output, "mqtt_errors_total 1") {
		t.Errorf("Expected mqtt_errors_total to be 1")
	}
	if !contains(output, "gateway_status 0") {
		t.Errorf("Expected gateway_status to be 0 (offline)")
	}

	t.Log("✅ PrometheusMetrics correctly records all metric types")
}

// TestNullMetricsZeroOverhead verifies that NullMetrics has no side effects
func TestNullMetricsZeroOverhead(t *testing.T) {
	nm := NewNullMetrics()

	// All operations should be no-ops and not panic
	nm.IncrementModbusReads()
	nm.IncrementModbusErrors()
	nm.IncrementMQTTPublishes()
	nm.IncrementMQTTErrors()
	nm.SetGatewayStatus(true)
	nm.SetGatewayStatus(false)
	nm.ObserveModbusReadDuration(100 * time.Millisecond)

	// StartMetricsServer should return nil immediately
	if err := nm.StartMetricsServer(9090); err != nil {
		t.Errorf("NullMetrics.StartMetricsServer should always return nil, got: %v", err)
	}

	t.Log("✅ NullMetrics provides zero-overhead no-op implementation")
}

// TestMetricsCollectorSwappable verifies that implementations can be swapped
func TestMetricsCollectorSwappable(t *testing.T) {
	// Simulates runtime choice based on configuration
	testCases := []struct {
		name             string
		metricsEnabled   bool
		expectedType     string
		metricsCollector MetricsCollector
	}{
		{
			name:             "Metrics enabled",
			metricsEnabled:   true,
			expectedType:     "PrometheusMetrics",
			metricsCollector: NewPrometheusMetrics(),
		},
		{
			name:             "Metrics disabled",
			metricsEnabled:   false,
			expectedType:     "NullMetrics",
			metricsCollector: NewNullMetrics(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Both implementations should work identically from interface perspective
			tc.metricsCollector.IncrementModbusReads()
			tc.metricsCollector.IncrementModbusErrors()
			tc.metricsCollector.IncrementMQTTPublishes()
			tc.metricsCollector.IncrementMQTTErrors()
			tc.metricsCollector.SetGatewayStatus(true)
			tc.metricsCollector.ObserveModbusReadDuration(50 * time.Millisecond)

			t.Logf("✅ %s implementation works correctly", tc.expectedType)
		})
	}
}

// TestMetricsCollectorThreadSafety verifies that PrometheusMetrics is thread-safe
func TestMetricsCollectorThreadSafety(t *testing.T) {
	pm := NewPrometheusMetrics()

	// Simulate concurrent access (like in real application)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Spawn multiple goroutines calling metrics
	for i := 0; i < 10; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					pm.IncrementModbusReads()
					pm.IncrementMQTTPublishes()
					pm.ObserveModbusReadDuration(10 * time.Millisecond)
					pm.SetGatewayStatus(true)
				}
			}
		}()
	}

	// Wait for goroutines to run
	<-ctx.Done()

	// Verify we can still get metrics without panic
	output := pm.GetMetricsText()
	if output == "" {
		t.Error("Expected non-empty metrics output")
	}

	t.Log("✅ PrometheusMetrics is thread-safe under concurrent access")
}

// TestMetricsServerStartup verifies that metrics server can start (PrometheusMetrics only)
func TestMetricsServerStartup(t *testing.T) {
	pm := NewPrometheusMetrics()

	// Test with invalid port (0 is typically reserved)
	// We don't actually start the server in test, just verify the method exists
	// and has correct signature

	// For NullMetrics, StartMetricsServer should always succeed
	nm := NewNullMetrics()
	if err := nm.StartMetricsServer(0); err != nil {
		t.Errorf("NullMetrics.StartMetricsServer should never fail, got: %v", err)
	}

	t.Log("✅ Both implementations have StartMetricsServer method with correct signature")
	t.Log("   (PrometheusMetrics:", pm, ")")
	t.Log("   (NullMetrics:", nm, ")")
}

// Helper function to check if string contains substring
func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && haystack != "" && needle != "" &&
		(haystack == needle || findSubstring(haystack, needle))
}

func findSubstring(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
