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

// SensorTopic handles sensor-related publishing
type SensorTopic struct {
	config *config.HAConfig
}

// NewSensorTopic creates a new sensor topic handler
func NewSensorTopic(config *config.HAConfig) *SensorTopic {
	return &SensorTopic{
		config: config,
	}
}

// PublishDiscovery publishes sensor discovery configuration
func (s *SensorTopic) PublishDiscovery(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Extract sensor name from topic
	sensorName := extractSensorName(result.Topic)

	// Topic for discovery
	discoveryTopic := fmt.Sprintf("%s/sensor/%s_%s/config",
		s.config.DiscoveryPrefix, s.config.DeviceID, sensorName)

	// Configuration for the sensor
	config := SensorConfig{
		Name:              result.Name,
		UniqueID:          fmt.Sprintf("%s_%s", s.config.DeviceID, sensorName),
		StateTopic:        result.Topic + "/state",
		UnitOfMeasurement: result.Unit,
		DeviceClass:       result.DeviceClass,
		StateClass:        result.StateClass,
		Device: DeviceInfo{
			Name:         s.config.DeviceName,
			Identifiers:  []string{s.config.DeviceID},
			Manufacturer: s.config.Manufacturer,
			Model:        s.config.Model,
		},
		ValueTemplate:       "{{ value_json.value }}",
		AvailabilityTopic:   s.config.StatusTopic,
		PayloadAvailable:    "online",
		PayloadNotAvailable: "offline",
	}

	// Serialize configuration
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializing configuration: %w", err)
	}

	// Publish configuration
	token := client.Publish(discoveryTopic, 0, true, configJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing discovery: %w", token.Error())
	}

	return nil
}

// PublishState publishes sensor state
func (s *SensorTopic) PublishState(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Validate the result before publishing
	if err := s.ValidateData(result, nil); err != nil {
		return fmt.Errorf("invalid sensor data: %w", err)
	}

	// State topic
	stateTopic := result.Topic + "/state"

	// Sensor data
	sensorData := SensorState{
		Value:     result.Value,
		Unit:      result.Unit,
		Timestamp: time.Now(),
	}

	// Serialize data
	dataJSON, err := json.Marshal(sensorData)
	if err != nil {
		return fmt.Errorf("error serializing data: %w", err)
	}

	// Publish state
	token := client.Publish(stateTopic, 0, false, dataJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing state: %w", token.Error())
	}

	return nil
}

// GetTopicPrefix returns the topic prefix for sensor topic
func (s *SensorTopic) GetTopicPrefix() string {
	return "sensor"
}

// ValidateData validates sensor data before publishing
func (s *SensorTopic) ValidateData(result *modbus.CommandResult, register *config.Register) error {
	// Check for invalid numeric values
	if math.IsNaN(result.Value) {
		return fmt.Errorf("value is NaN for sensor %s", result.Name)
	}

	if math.IsInf(result.Value, 0) {
		return fmt.Errorf("value is infinite for sensor %s", result.Name)
	}

	// Check for reasonable bounds based on device class
	switch result.DeviceClass {
	case "voltage":
		if result.Value < 0 || result.Value > 1000 {
			return fmt.Errorf("voltage value out of reasonable bounds: %.3f", result.Value)
		}
	case "current":
		if result.Value < 0 || result.Value > 1000 {
			return fmt.Errorf("current value out of reasonable bounds: %.3f", result.Value)
		}
	case "frequency":
		if result.Value < 40 || result.Value > 70 {
			return fmt.Errorf("frequency value out of reasonable bounds: %.3f", result.Value)
		}
	case "power", "apparent_power":
		if result.Value < -100000 || result.Value > 100000 {
			return fmt.Errorf("power value out of reasonable bounds: %.3f", result.Value)
		}
	case "power_factor":
		if result.Value < 0 || result.Value > 1 {
			return fmt.Errorf("power factor value out of reasonable bounds: %.3f", result.Value)
		}
	case "energy":
		if result.Value < 0 || result.Value > 999999999 {
			return fmt.Errorf("energy value out of reasonable bounds: %.3f", result.Value)
		}
	}

	// Check required fields
	if result.Name == "" {
		return fmt.Errorf("sensor name is empty")
	}

	if result.Topic == "" {
		return fmt.Errorf("sensor topic is empty")
	}

	return nil
}
