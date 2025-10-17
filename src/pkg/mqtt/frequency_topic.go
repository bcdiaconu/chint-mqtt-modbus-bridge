package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/modbus"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// FrequencyTopic handles frequency sensor publishing
type FrequencyTopic struct {
	config *config.HAConfig
}

// NewFrequencyTopic creates a new frequency topic handler
func NewFrequencyTopic(config *config.HAConfig) *FrequencyTopic {
	return &FrequencyTopic{
		config: config,
	}
}

// PublishDiscovery publishes frequency sensor discovery configuration
func (f *FrequencyTopic) PublishDiscovery(ctx context.Context, client mqtt.Client, result *modbus.CommandResult, deviceInfo *DeviceInfo) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Extract sensor name from topic
	sensorName := extractSensorName(result.Topic)

	// Use device info if provided, otherwise fall back to deprecated global config
	var device DeviceInfo
	if deviceInfo != nil {
		device = *deviceInfo
	} else {
		device = DeviceInfo{
			Name:         f.config.DeviceName,
			Identifiers:  []string{f.config.DeviceID},
			Manufacturer: f.config.Manufacturer,
			Model:        f.config.Model,
		}
	}

	// Topic for discovery
	discoveryTopic := fmt.Sprintf("%s/sensor/%s_%s/config",
		f.config.DiscoveryPrefix, device.Identifiers[0], sensorName)

	// Configuration for the frequency sensor
	config := SensorConfig{
		Name:                result.Name,
		UniqueID:            fmt.Sprintf("%s_%s", device.Identifiers[0], sensorName),
		StateTopic:          result.Topic + "/state",
		UnitOfMeasurement:   result.Unit,
		DeviceClass:         result.DeviceClass,
		StateClass:          result.StateClass,
		Device:              device,
		ValueTemplate:       "{{ value_json.value }}",
		AvailabilityTopic:   f.config.StatusTopic,
		PayloadAvailable:    "online",
		PayloadNotAvailable: "offline",
	}

	// Serialize configuration
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializing frequency configuration: %w", err)
	}

	// Publish configuration
	token := client.Publish(discoveryTopic, 0, true, configJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing frequency discovery: %w", token.Error())
	}

	return nil
}

// PublishState publishes frequency sensor state
func (f *FrequencyTopic) PublishState(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Validate the result before publishing
	if err := f.ValidateData(result, nil); err != nil {
		return fmt.Errorf("invalid frequency data: %w", err)
	}

	// State topic
	stateTopic := result.Topic + "/state"

	// Frequency sensor data
	sensorData := SensorState{
		Value:     result.Value,
		Unit:      result.Unit,
		Timestamp: time.Now(),
	}

	// Serialize data
	dataJSON, err := json.Marshal(sensorData)
	if err != nil {
		return fmt.Errorf("error serializing frequency data: %w", err)
	}

	// Publish state
	token := client.Publish(stateTopic, 0, false, dataJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing frequency state: %w", token.Error())
	}

	return nil
}

// GetTopicPrefix returns the topic prefix for frequency topic
func (f *FrequencyTopic) GetTopicPrefix() string {
	return "frequency"
}

// ValidateData validates frequency sensor data before publishing
func (f *FrequencyTopic) ValidateData(result *modbus.CommandResult, register *config.Register) error {
	// Check for invalid numeric values (NaN, Inf)
	if math.IsNaN(result.Value) {
		return fmt.Errorf("frequency value is NaN for sensor %s", result.Name)
	}

	if math.IsInf(result.Value, 0) {
		return fmt.Errorf("frequency value is infinite for sensor %s", result.Name)
	}

	// Check required fields
	if result.Name == "" {
		return fmt.Errorf("frequency sensor name is empty")
	}

	if result.Topic == "" {
		return fmt.Errorf("frequency sensor topic is empty")
	}

	// Apply min/max validation from register config if specified
	if register != nil {
		if register.Min != nil && result.Value < *register.Min {
			return fmt.Errorf("frequency value %.3f Hz below minimum threshold %.3f Hz", result.Value, *register.Min)
		}
		if register.Max != nil && result.Value > *register.Max {
			return fmt.Errorf("frequency value %.3f Hz above maximum threshold %.3f Hz", result.Value, *register.Max)
		}
	}

	return nil
}

// GetFrequencyStability returns stability assessment of the frequency reading
func (f *FrequencyTopic) GetFrequencyStability(value float64) string {
	deviation := math.Abs(value - 50.0) // Assuming 50Hz standard
	switch {
	case deviation <= 0.1:
		return "excellent"
	case deviation <= 0.3:
		return "good"
	case deviation <= 0.5:
		return "acceptable"
	default:
		return "unstable"
	}
}

// IsFrequencyStable checks if frequency is within stable grid range
func (f *FrequencyTopic) IsFrequencyStable(value float64) bool {
	return value >= 49.5 && value <= 50.5
}

// GetGridStandard returns the grid standard assessment
func (f *FrequencyTopic) GetGridStandard(value float64) string {
	switch {
	case value >= 49.5 && value <= 50.5:
		return "EU_50Hz"
	case value >= 59.5 && value <= 60.5:
		return "US_60Hz"
	default:
		return "unknown"
	}
}
