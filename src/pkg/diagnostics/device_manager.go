package diagnostics

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/logger"
	"mqtt-modbus-bridge/pkg/mqtt"
	"sync"
	"time"
)

// DeviceManager manages device diagnostic metrics and publishing
// Follows Single Responsibility Principle - only handles device diagnostics
type DeviceManager struct {
	publisher   mqtt.PublisherInterface // Use interface to avoid tight coupling
	config      *config.DeviceDiagnosticsConfig
	devices     map[string]config.Device       // Device configurations for discovery
	metrics     map[string]*mqtt.DeviceMetrics // Metrics tracked per device
	lastState   map[string]string              // Last published state per device
	lastPublish map[string]time.Time           // Last publish time per device
	mu          sync.RWMutex                   // Mutex for concurrent access
}

// NewDeviceManager creates a new device diagnostic manager
func NewDeviceManager(publisher mqtt.PublisherInterface, diagnosticConfig *config.DeviceDiagnosticsConfig, devices map[string]config.Device) *DeviceManager {
	manager := &DeviceManager{
		publisher:   publisher,
		config:      diagnosticConfig,
		devices:     devices,
		metrics:     make(map[string]*mqtt.DeviceMetrics),
		lastState:   make(map[string]string),
		lastPublish: make(map[string]time.Time),
	}

	// Initialize metrics for all enabled devices
	for deviceID, device := range devices {
		if device.Metadata.Enabled {
			manager.metrics[deviceID] = &mqtt.DeviceMetrics{
				CurrentState: "operational", // Start optimistic
			}
		}
	}

	return manager
}

// RecordSuccess records a successful device read with response time
func (m *DeviceManager) RecordSuccess(deviceID string, responseTime time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics, exists := m.metrics[deviceID]
	if !exists {
		metrics = &mqtt.DeviceMetrics{}
		m.metrics[deviceID] = metrics
	}

	now := time.Now()
	metrics.LastReadTime = now
	metrics.LastSuccessTime = now
	metrics.ConsecutiveErrors = 0
	metrics.TotalReads++
	metrics.SuccessfulReads++
	metrics.TotalResponseTime += responseTime
}

// RecordError records a failed device read with error message
func (m *DeviceManager) RecordError(deviceID string, errorMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics, exists := m.metrics[deviceID]
	if !exists {
		metrics = &mqtt.DeviceMetrics{}
		m.metrics[deviceID] = metrics
	}

	now := time.Now()
	metrics.LastReadTime = now
	metrics.ConsecutiveErrors++
	metrics.TotalReads++
	metrics.FailedReads++
	metrics.LastError = errorMsg
	metrics.LastErrorTime = now
}

// StartDiagnosticsLoop starts the periodic device diagnostics publishing loop
func (m *DeviceManager) StartDiagnosticsLoop(ctx context.Context) {
	// Start with a small delay to let devices initialize
	time.Sleep(5 * time.Second)

	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	logger.LogInfo("📊 Device diagnostics loop started")

	for {
		select {
		case <-ctx.Done():
			logger.LogDebug("📊 Device diagnostics loop stopped")
			return
		case <-ticker.C:
			m.publishDiagnostics(ctx)
		}
	}
}

// PublishDiscoveryForAllDevices publishes discovery messages for all enabled devices
func (m *DeviceManager) PublishDiscoveryForAllDevices(ctx context.Context) error {
	for deviceKey, device := range m.devices {
		if !device.Metadata.Enabled {
			continue
		}

		// Get Home Assistant device ID (using device key as fallback)
		haDeviceID := deviceKey
		if device.HomeAssistant.DeviceID != "" {
			haDeviceID = device.HomeAssistant.DeviceID
		}

		// Build device info
		deviceInfo := &mqtt.DeviceInfo{
			Name:         device.Metadata.Name,
			Identifiers:  []string{haDeviceID},
			Manufacturer: device.Metadata.Manufacturer,
			Model:        device.Metadata.Model,
		}

		// Publish device diagnostic discovery
		if err := m.publisher.PublishDeviceDiagnosticDiscovery(ctx, haDeviceID, deviceInfo); err != nil {
			logger.LogWarn("⚠️ Error publishing device diagnostic discovery for %s: %v", deviceKey, err)
		} else {
			logger.LogDebug("📊 Published device diagnostic discovery for %s", deviceKey)
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// publishDiagnostics publishes diagnostic state for all devices based on configuration
func (m *DeviceManager) publishDiagnostics(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	thresholds := &m.config.Thresholds

	for deviceID, metrics := range m.metrics {
		// Calculate new state based on metrics
		newState := mqtt.CalculateDeviceState(metrics, thresholds)

		// Get last published state
		lastState := m.lastState[deviceID]
		lastPublish := m.lastPublish[deviceID]

		// Determine if we should publish
		shouldPublish := false

		// 1. State changed - publish immediately
		if newState != lastState {
			shouldPublish = true
			if m.config.PublishOnStateChange {
				logger.LogInfo("📊 Device %s state changed: %s → %s", deviceID, lastState, newState)
			}
		} else {
			// 2. Periodic publish based on current state interval
			interval := m.getDiagnosticIntervalForState(newState)
			if time.Since(lastPublish) >= interval {
				shouldPublish = true
			}
		}

		if shouldPublish {
			// Update current state in metrics
			metrics.CurrentState = newState

			// Publish diagnostic state
			if err := m.publisher.PublishDeviceDiagnosticState(ctx, deviceID, metrics); err != nil {
				logger.LogWarn("⚠️ Error publishing device diagnostic for %s: %v", deviceID, err)
			} else {
				// Update tracking (note: we're in RLock, this is safe as we only modify for current deviceID)
				m.lastState[deviceID] = newState
				m.lastPublish[deviceID] = time.Now()
			}
		}
	}
}

// getDiagnosticIntervalForState returns the publish interval for a given device state
func (m *DeviceManager) getDiagnosticIntervalForState(state string) time.Duration {
	intervals := &m.config.Intervals

	switch state {
	case "operational":
		return time.Duration(intervals.Operational) * time.Second
	case "warning":
		return time.Duration(intervals.Warning) * time.Second
	case "error":
		return time.Duration(intervals.Error) * time.Second
	case "offline":
		return time.Duration(intervals.Offline) * time.Second
	default:
		return 60 * time.Second // Default fallback
	}
}

// GetMetrics returns a copy of metrics for a specific device (for testing/debugging)
func (m *DeviceManager) GetMetrics(deviceID string) (*mqtt.DeviceMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics, exists := m.metrics[deviceID]
	if !exists {
		return nil, fmt.Errorf("device %s not found", deviceID)
	}

	// Return a copy to prevent external modification
	metricsCopy := *metrics
	return &metricsCopy, nil
}
