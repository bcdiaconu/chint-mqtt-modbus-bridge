package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/logger"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// DeviceDiagnosticTopic handles per-device diagnostic sensor publishing
type DeviceDiagnosticTopic struct {
	config  *config.HAConfig
	factory *TopicFactory
}

// NewDeviceDiagnosticTopic creates a new device diagnostic topic handler
func NewDeviceDiagnosticTopic(config *config.HAConfig) *DeviceDiagnosticTopic {
	return &DeviceDiagnosticTopic{
		config:  config,
		factory: NewTopicFactory(config.DiscoveryPrefix),
	}
}

// DeviceMetrics represents metrics tracked per device
type DeviceMetrics struct {
	LastReadTime      time.Time
	LastSuccessTime   time.Time
	ConsecutiveErrors int
	TotalReads        int64
	SuccessfulReads   int64
	FailedReads       int64
	TotalResponseTime time.Duration
	LastError         string
	LastErrorTime     time.Time
	CurrentState      string // operational, warning, error, offline
}

// DeviceDiagnosticState represents the state payload for device diagnostic sensor
type DeviceDiagnosticState struct {
	State             string  `json:"state"`
	LastRead          string  `json:"last_read,omitempty"`
	LastSuccess       string  `json:"last_success,omitempty"`
	ConsecutiveErrors int     `json:"consecutive_errors"`
	TotalReads        int64   `json:"total_reads"`
	SuccessfulReads   int64   `json:"successful_reads"`
	FailedReads       int64   `json:"failed_reads"`
	SuccessRate       float64 `json:"success_rate"`
	AvgResponseMs     int64   `json:"avg_response_ms,omitempty"`
	LastError         string  `json:"last_error,omitempty"`
	LastErrorTime     string  `json:"last_error_time,omitempty"`
}

// PublishDiscovery publishes discovery configuration for device diagnostic sensor
func (d *DeviceDiagnosticTopic) PublishDiscovery(ctx context.Context, client mqtt.Client, deviceID string, deviceInfo *DeviceInfo) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Build topics
	discoveryTopic := d.factory.BuildDeviceDiagnosticDiscoveryTopic(deviceID)
	stateTopic := d.factory.BuildDeviceDiagnosticStateTopic(deviceID)
	uniqueID := d.factory.BuildDeviceDiagnosticUniqueID(deviceID)

	// Configuration for the device diagnostic sensor
	sensorConfig := SensorConfig{
		Name:                   deviceInfo.Name + " Diagnostic",
		UniqueID:               uniqueID,
		StateTopic:             stateTopic,
		DeviceClass:            "enum",
		Device:                 *deviceInfo,
		ValueTemplate:          "{{ value_json.state }}",
		AvailabilityTopic:      d.config.StatusTopic,
		AvailabilityMode:       "latest",
		PayloadAvailable:       "online",
		PayloadNotAvailable:    "offline",
		JSONAttributesTemplate: "{{ value_json | tojson }}",
		EntityCategory:         "diagnostic",
	}

	// Serialize configuration
	configJSON, err := json.Marshal(sensorConfig)
	if err != nil {
		return fmt.Errorf("error serializing device diagnostic configuration: %w", err)
	}

	logger.LogDebug("📡 Publishing device diagnostic discovery for %s: %s", deviceID, discoveryTopic)

	// Publish configuration with retain
	token := client.Publish(discoveryTopic, 0, true, configJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing device diagnostic discovery: %w", token.Error())
	}

	return nil
}

// PublishState publishes device diagnostic state
func (d *DeviceDiagnosticTopic) PublishState(ctx context.Context, client mqtt.Client, deviceID string, metrics *DeviceMetrics) error {
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	// Calculate success rate
	successRate := 0.0
	if metrics.TotalReads > 0 {
		successRate = (float64(metrics.SuccessfulReads) / float64(metrics.TotalReads)) * 100.0
	}

	// Calculate average response time
	avgResponseMs := int64(0)
	if metrics.SuccessfulReads > 0 {
		avgResponseMs = metrics.TotalResponseTime.Milliseconds() / metrics.SuccessfulReads
	}

	// Build state payload
	state := DeviceDiagnosticState{
		State:             metrics.CurrentState,
		ConsecutiveErrors: metrics.ConsecutiveErrors,
		TotalReads:        metrics.TotalReads,
		SuccessfulReads:   metrics.SuccessfulReads,
		FailedReads:       metrics.FailedReads,
		SuccessRate:       successRate,
		AvgResponseMs:     avgResponseMs,
	}

	// Add timestamps if available
	if !metrics.LastReadTime.IsZero() {
		state.LastRead = metrics.LastReadTime.Format(time.RFC3339)
	}
	if !metrics.LastSuccessTime.IsZero() {
		state.LastSuccess = metrics.LastSuccessTime.Format(time.RFC3339)
	}
	if metrics.LastError != "" {
		state.LastError = metrics.LastError
		if !metrics.LastErrorTime.IsZero() {
			state.LastErrorTime = metrics.LastErrorTime.Format(time.RFC3339)
		}
	}

	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("error marshaling device diagnostic state: %w", err)
	}

	stateTopic := d.factory.BuildDeviceDiagnosticStateTopic(deviceID)
	token := client.Publish(stateTopic, 0, false, payload)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		if token.Error() != nil {
			return fmt.Errorf("error publishing device diagnostic state: %w", token.Error())
		}
	}

	logger.LogDebug("📊 Published device diagnostic for %s: state=%s, success_rate=%.1f%%",
		deviceID, metrics.CurrentState, successRate)

	return nil
}

// CalculateDeviceState determines device state based on metrics and thresholds
func CalculateDeviceState(metrics *DeviceMetrics, thresholds *config.DiagnosticThresholdsConfig) string {
	// Check if device is offline (no successful reads within timeout)
	if !metrics.LastSuccessTime.IsZero() {
		timeSinceSuccess := time.Since(metrics.LastSuccessTime).Seconds()
		if timeSinceSuccess > float64(thresholds.OfflineTimeout) {
			return "offline"
		}
	} else if metrics.TotalReads > 0 {
		// If we have reads but never succeeded, and it's been more than timeout
		if !metrics.LastReadTime.IsZero() {
			timeSinceLastRead := time.Since(metrics.LastReadTime).Seconds()
			if timeSinceLastRead > float64(thresholds.OfflineTimeout) {
				return "offline"
			}
		}
	}

	// Calculate success rate
	successRate := 0.0
	if metrics.TotalReads > 0 {
		successRate = (float64(metrics.SuccessfulReads) / float64(metrics.TotalReads)) * 100.0
	}

	// Check for error state
	if successRate < thresholds.ErrorSuccessRate || metrics.ConsecutiveErrors >= thresholds.ErrorConsecutiveErrors {
		return "error"
	}

	// Check for warning state
	if successRate < thresholds.WarningSuccessRate || metrics.ConsecutiveErrors >= thresholds.WarningConsecutiveErrors {
		return "warning"
	}

	// Everything is good
	return "operational"
}
