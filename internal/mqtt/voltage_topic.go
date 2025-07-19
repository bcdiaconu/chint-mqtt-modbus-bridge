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

// VoltageTopic handles voltage sensor publishing
type VoltageTopic struct {
	config *config.HAConfig
}

// NewVoltageTopic creates a new voltage topic handler
func NewVoltageTopic(config *config.HAConfig) *VoltageTopic {
	return &VoltageTopic{
		config: config,
	}
}

// PublishDiscovery publishes voltage sensor discovery configuration
func (v *VoltageTopic) PublishDiscovery(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Extract sensor name from topic
	sensorName := extractSensorName(result.Topic)

	// Topic for discovery
	discoveryTopic := fmt.Sprintf("%s/sensor/%s_%s/config",
		v.config.DiscoveryPrefix, v.config.DeviceID, sensorName)

	// Configuration for the voltage sensor
	config := SensorConfig{
		Name:              result.Name,
		UniqueID:          fmt.Sprintf("%s_%s", v.config.DeviceID, sensorName),
		StateTopic:        result.Topic + "/state",
		UnitOfMeasurement: result.Unit,
		DeviceClass:       result.DeviceClass,
		StateClass:        result.StateClass,
		Device: DeviceInfo{
			Name:         v.config.DeviceName,
			Identifiers:  []string{v.config.DeviceID},
			Manufacturer: v.config.Manufacturer,
			Model:        v.config.Model,
		},
		ValueTemplate:       "{{ value_json.value }}",
		AvailabilityTopic:   v.config.StatusTopic,
		PayloadAvailable:    "online",
		PayloadNotAvailable: "offline",
	}

	// Serialize configuration
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializing voltage configuration: %w", err)
	}

	// Publish configuration
	token := client.Publish(discoveryTopic, 0, true, configJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing voltage discovery: %w", token.Error())
	}

	return nil
}

// PublishState publishes voltage sensor state
func (v *VoltageTopic) PublishState(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Validate the result before publishing
	if err := v.ValidateData(result, nil); err != nil {
		return fmt.Errorf("invalid voltage data: %w", err)
	}

	// State topic
	stateTopic := result.Topic + "/state"

	// Voltage sensor data
	sensorData := SensorState{
		Value:     result.Value,
		Unit:      result.Unit,
		Timestamp: time.Now(),
	}

	// Serialize data
	dataJSON, err := json.Marshal(sensorData)
	if err != nil {
		return fmt.Errorf("error serializing voltage data: %w", err)
	}

	// Publish state
	token := client.Publish(stateTopic, 0, false, dataJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing voltage state: %w", token.Error())
	}

	return nil
}

// GetTopicPrefix returns the topic prefix for voltage topic
func (v *VoltageTopic) GetTopicPrefix() string {
	return "voltage"
}

// ValidateData validates voltage sensor data before publishing
func (v *VoltageTopic) ValidateData(result *modbus.CommandResult, register *config.Register) error {
	// Check for invalid numeric values (NaN, Inf)
	if math.IsNaN(result.Value) {
		return fmt.Errorf("voltage value is NaN for sensor %s", result.Name)
	}

	if math.IsInf(result.Value, 0) {
		return fmt.Errorf("voltage value is infinite for sensor %s", result.Name)
	}

	// Check required fields
	if result.Name == "" {
		return fmt.Errorf("voltage sensor name is empty")
	}

	if result.Topic == "" {
		return fmt.Errorf("voltage sensor topic is empty")
	}

	// Apply min/max validation from register config if specified
	if register != nil {
		if register.Min != nil && result.Value < *register.Min {
			return fmt.Errorf("voltage value %.3f V below minimum threshold %.3f V", result.Value, *register.Min)
		}
		if register.Max != nil && result.Value > *register.Max {
			return fmt.Errorf("voltage value %.3f V above maximum threshold %.3f V", result.Value, *register.Max)
		}
	}

	return nil
}

// GetVoltageQuality returns a quality assessment of the voltage reading
func (v *VoltageTopic) GetVoltageQuality(value float64) string {
	switch {
	case value >= 220 && value <= 240:
		return "excellent"
	case value >= 210 && value <= 250:
		return "good"
	case value >= 180 && value <= 270:
		return "acceptable"
	default:
		return "poor"
	}
}

// IsVoltageStable checks if voltage is within stable range
func (v *VoltageTopic) IsVoltageStable(value float64) bool {
	return value >= 210 && value <= 250
}
