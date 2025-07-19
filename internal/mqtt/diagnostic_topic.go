package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"mqtt-modbus-bridge/internal/config"
	"mqtt-modbus-bridge/internal/logger"
	"mqtt-modbus-bridge/internal/modbus"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// DiagnosticTopic handles Home Assistant diagnostic sensor publishing
// DiagnosticTopic handles diagnostic-related publishing
type DiagnosticTopic struct {
	config *config.HAConfig
}

// NewDiagnosticTopic creates a new diagnostic topic handler
func NewDiagnosticTopic(config *config.HAConfig) *DiagnosticTopic {
	return &DiagnosticTopic{
		config: config,
	}
}

// PublishDiscovery publishes diagnostic discovery configuration
func (d *DiagnosticTopic) PublishDiscovery(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Topic for diagnostic sensor discovery
	discoveryTopic := fmt.Sprintf("%s/sensor/%s_diagnostic/config",
		d.config.DiscoveryPrefix, d.config.DeviceID)

	// Configuration for the diagnostic sensor
	config := SensorConfig{
		Name:              "Diagnostic",
		UniqueID:          fmt.Sprintf("%s_diagnostic", d.config.DeviceID),
		StateTopic:        d.config.DiagnosticTopic,
		UnitOfMeasurement: "",
		DeviceClass:       "enum",
		StateClass:        "",
		Device: DeviceInfo{
			Name:         d.config.DeviceName,
			Identifiers:  []string{d.config.DeviceID},
			Manufacturer: d.config.Manufacturer,
			Model:        d.config.Model,
		},
		ValueTemplate:          "{{ value_json.message }}",
		AvailabilityTopic:      d.config.StatusTopic,
		PayloadAvailable:       "online",
		PayloadNotAvailable:    "offline",
		JSONAttributesTemplate: "{{ value_json | tojson }}",
		EntityCategory:         "diagnostic",
	}

	// Serialize configuration
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializing diagnostic configuration: %w", err)
	}

	logger.LogDebug("📡 Publishing diagnostic discovery: %s", discoveryTopic)

	// Publish configuration
	token := client.Publish(discoveryTopic, 0, true, configJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing diagnostic discovery: %w", token.Error())
	}

	return nil
}

// PublishState publishes diagnostic state
func (d *DiagnosticTopic) PublishState(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	// Validate the result before publishing
	if err := d.ValidateData(result, nil); err != nil {
		return fmt.Errorf("invalid diagnostic data: %w", err)
	}

	// For diagnostics, we expect the result to contain diagnostic information
	// The Value field contains the diagnostic code, and Name contains the message
	diagnostic := map[string]interface{}{
		"code":      int(result.Value),
		"message":   result.Name,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(diagnostic)
	if err != nil {
		return fmt.Errorf("error marshaling diagnostic: %w", err)
	}

	token := client.Publish(d.config.DiagnosticTopic, 0, false, payload)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		if token.Error() != nil {
			return fmt.Errorf("error publishing diagnostic: %w", token.Error())
		}
	}

	return nil
}

// GetTopicPrefix returns the topic prefix for diagnostic topic
func (d *DiagnosticTopic) GetTopicPrefix() string {
	return "diagnostic"
}

// ValidateData validates diagnostic data
func (d *DiagnosticTopic) ValidateData(result *modbus.CommandResult, register *config.Register) error {
	// Diagnostic code should be a valid integer
	if result.Value < 0 || result.Value > 9999 {
		return fmt.Errorf("invalid diagnostic code: %.0f", result.Value)
	}

	// Message should not be empty
	if result.Name == "" {
		return fmt.Errorf("diagnostic message is empty")
	}

	return nil
}

// PublishDiagnostic publishes diagnostic information with code and message
func (d *DiagnosticTopic) PublishDiagnostic(ctx context.Context, client mqtt.Client, code int, message string) error {
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	diagnostic := map[string]interface{}{
		"code":      code,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(diagnostic)
	if err != nil {
		return fmt.Errorf("error marshaling diagnostic: %w", err)
	}

	token := client.Publish(d.config.DiagnosticTopic, 0, false, payload)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		if token.Error() != nil {
			return fmt.Errorf("error publishing diagnostic: %w", token.Error())
		}
	}

	logger.LogDebug("🔧 Published diagnostic: [%d] %s", code, message)
	return nil
}
