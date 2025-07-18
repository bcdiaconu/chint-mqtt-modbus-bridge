package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"mqtt-modbus-bridge/internal/config"
	"mqtt-modbus-bridge/internal/modbus"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// CurrentTopic handles current sensor publishing
type CurrentTopic struct {
	config *config.HAConfig
}

// NewCurrentTopic creates a new current topic handler
func NewCurrentTopic(config *config.HAConfig) *CurrentTopic {
	return &CurrentTopic{
		config: config,
	}
}

// PublishDiscovery publishes current sensor discovery configuration
func (c *CurrentTopic) PublishDiscovery(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Extract sensor name from topic
	sensorName := extractSensorName(result.Topic)

	// Topic for discovery
	discoveryTopic := fmt.Sprintf("%s/sensor/%s_%s/config",
		c.config.DiscoveryPrefix, c.config.DeviceID, sensorName)

	// Configuration for the current sensor
	config := SensorConfig{
		Name:              result.Name,
		UniqueID:          fmt.Sprintf("%s_%s", c.config.DeviceID, sensorName),
		StateTopic:        result.Topic + "/state",
		UnitOfMeasurement: result.Unit,
		DeviceClass:       result.DeviceClass,
		StateClass:        result.StateClass,
		Device: DeviceInfo{
			Name:         c.config.DeviceName,
			Identifiers:  []string{c.config.DeviceID},
			Manufacturer: c.config.Manufacturer,
			Model:        c.config.Model,
		},
		ValueTemplate:       "{{ value_json.value }}",
		AvailabilityTopic:   c.config.StatusTopic,
		PayloadAvailable:    "online",
		PayloadNotAvailable: "offline",
	}

	// Serialize configuration
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializing current configuration: %w", err)
	}

	// Publish configuration
	token := client.Publish(discoveryTopic, 0, true, configJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing current discovery: %w", token.Error())
	}

	return nil
}

// PublishState publishes current sensor state
func (c *CurrentTopic) PublishState(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Validate the result before publishing
	if err := c.ValidateData(result); err != nil {
		return fmt.Errorf("invalid current data: %w", err)
	}

	// State topic
	stateTopic := result.Topic + "/state"

	// Current sensor data
	sensorData := SensorState{
		Value:     result.Value,
		Unit:      result.Unit,
		Timestamp: time.Now(),
	}

	// Serialize data
	dataJSON, err := json.Marshal(sensorData)
	if err != nil {
		return fmt.Errorf("error serializing current data: %w", err)
	}

	// Publish state
	token := client.Publish(stateTopic, 0, false, dataJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing current state: %w", token.Error())
	}

	return nil
}

// GetTopicPrefix returns the topic prefix for current topic
func (c *CurrentTopic) GetTopicPrefix() string {
	return "current"
}

// ValidateData validates current sensor data before publishing
func (c *CurrentTopic) ValidateData(result *modbus.CommandResult) error {
	// Check for invalid numeric values
	if math.IsNaN(result.Value) {
		return fmt.Errorf("current value is NaN for sensor %s", result.Name)
	}

	if math.IsInf(result.Value, 0) {
		return fmt.Errorf("current value is infinite for sensor %s", result.Name)
	}

	// Current-specific validation - typical range for residential/commercial
	if result.Value < 0 || result.Value > 1000 {
		return fmt.Errorf("current value out of reasonable bounds: %.3f A (expected 0-1000A)", result.Value)
	}

	// Check required fields
	if result.Name == "" {
		return fmt.Errorf("current sensor name is empty")
	}

	if result.Topic == "" {
		return fmt.Errorf("current sensor topic is empty")
	}

	return nil
}

// GetCurrentLoad returns the load assessment based on current reading
func (c *CurrentTopic) GetCurrentLoad(value float64) string {
	switch {
	case value < 1:
		return "minimal"
	case value < 10:
		return "low"
	case value < 50:
		return "moderate"
	case value < 200:
		return "high"
	default:
		return "critical"
	}
}

// IsCurrentNormal checks if current is within normal operating range
func (c *CurrentTopic) IsCurrentNormal(value float64) bool {
	return value >= 0 && value <= 400
}
