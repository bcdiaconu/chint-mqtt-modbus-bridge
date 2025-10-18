package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/logger"
	"mqtt-modbus-bridge/pkg/modbus"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// DiagnosticTopic handles Home Assistant diagnostic sensor publishing
// DiagnosticTopic handles diagnostic-related publishing
type DiagnosticTopic struct {
	config  *config.HAConfig
	factory *TopicFactory
}

// NewDiagnosticTopic creates a new diagnostic topic handler
func NewDiagnosticTopic(config *config.HAConfig) *DiagnosticTopic {
	return &DiagnosticTopic{
		config:  config,
		factory: NewTopicFactory(config.DiscoveryPrefix),
	}
}

// PublishDiscovery publishes diagnostic discovery configuration
func (d *DiagnosticTopic) PublishDiscovery(ctx context.Context, client mqtt.Client, result *modbus.CommandResult, deviceInfo *DeviceInfo) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Use device info if provided, otherwise fall back to bridge constants
	var device DeviceInfo
	if deviceInfo != nil {
		device = *deviceInfo
	} else {
		// Fallback to bridge device constants (should not happen in normal operation)
		device = DeviceInfo{
			Name:         config.BridgeDeviceName,
			Identifiers:  []string{config.BridgeDeviceID},
			Manufacturer: config.BridgeDeviceManufacturer,
			Model:        config.BridgeDeviceModel,
		}
	}

	// Topic for diagnostic sensor discovery
	logger.LogDebug("🔍 Bridge device info: Name='%s', ID='%s', Manufacturer='%s', Model='%s'",
		device.Name, device.Identifiers[0], device.Manufacturer, device.Model)

	// Build topics using factory
	deviceID := ExtractDeviceID(&device)
	discoveryTopic := d.factory.BuildDiagnosticDiscoveryTopic(deviceID)
	stateTopic := d.factory.BuildDiagnosticStateTopic(deviceID)
	uniqueID := d.factory.BuildDiagnosticUniqueID(deviceID)

	// Configuration for the diagnostic sensor
	config := SensorConfig{
		Name:                   "Diagnostic",
		UniqueID:               uniqueID,
		StateTopic:             stateTopic,
		UnitOfMeasurement:      "",
		DeviceClass:            "enum",
		StateClass:             "",
		Device:                 device,
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

	// Publish to Home Assistant state topic using bridge device ID constant
	bridgeDeviceID := config.BridgeDeviceID
	stateTopic := d.factory.BuildDiagnosticStateTopic(bridgeDeviceID)

	logger.LogDebug("🔧 📤 Publishing diagnostic to '%s': %s", stateTopic, message)

	token := client.Publish(stateTopic, 0, false, payload)
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
