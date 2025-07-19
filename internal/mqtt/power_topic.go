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

// PowerTopic handles power sensor publishing (active, apparent, reactive)
type PowerTopic struct {
	config *config.HAConfig
}

// NewPowerTopic creates a new power topic handler
func NewPowerTopic(config *config.HAConfig) *PowerTopic {
	return &PowerTopic{
		config: config,
	}
}

// PublishDiscovery publishes power sensor discovery configuration
func (p *PowerTopic) PublishDiscovery(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Extract sensor name from topic
	sensorName := extractSensorName(result.Topic)

	// Topic for discovery
	discoveryTopic := fmt.Sprintf("%s/sensor/%s_%s/config",
		p.config.DiscoveryPrefix, p.config.DeviceID, sensorName)

	// Configuration for the power sensor
	config := SensorConfig{
		Name:              result.Name,
		UniqueID:          fmt.Sprintf("%s_%s", p.config.DeviceID, sensorName),
		StateTopic:        result.Topic + "/state",
		UnitOfMeasurement: result.Unit,
		DeviceClass:       result.DeviceClass,
		StateClass:        result.StateClass,
		Device: DeviceInfo{
			Name:         p.config.DeviceName,
			Identifiers:  []string{p.config.DeviceID},
			Manufacturer: p.config.Manufacturer,
			Model:        p.config.Model,
		},
		ValueTemplate:       "{{ value_json.value }}",
		AvailabilityTopic:   p.config.StatusTopic,
		PayloadAvailable:    "online",
		PayloadNotAvailable: "offline",
	}

	// Serialize configuration
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializing power configuration: %w", err)
	}

	// Publish configuration
	token := client.Publish(discoveryTopic, 0, true, configJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing power discovery: %w", token.Error())
	}

	return nil
}

// PublishState publishes power sensor state
func (p *PowerTopic) PublishState(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Validate the result before publishing
	if err := p.ValidateData(result, nil); err != nil {
		return fmt.Errorf("invalid power data: %w", err)
	}

	// State topic
	stateTopic := result.Topic + "/state"

	// Power sensor data
	sensorData := SensorState{
		Value:     result.Value,
		Unit:      result.Unit,
		Timestamp: time.Now(),
	}

	// Serialize data
	dataJSON, err := json.Marshal(sensorData)
	if err != nil {
		return fmt.Errorf("error serializing power data: %w", err)
	}

	// Publish state
	token := client.Publish(stateTopic, 0, false, dataJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing power state: %w", token.Error())
	}

	return nil
}

// GetTopicPrefix returns the topic prefix for power topic
func (p *PowerTopic) GetTopicPrefix() string {
	return "power"
}

// ValidateData validates power sensor data before publishing
func (p *PowerTopic) ValidateData(result *modbus.CommandResult, register *config.Register) error {
	// Check for invalid numeric values (NaN, Inf)
	if math.IsNaN(result.Value) {
		return fmt.Errorf("power value is NaN for sensor %s", result.Name)
	}

	if math.IsInf(result.Value, 0) {
		return fmt.Errorf("power value is infinite for sensor %s", result.Name)
	}

	// Check required fields
	if result.Name == "" {
		return fmt.Errorf("power sensor name is empty")
	}

	if result.Topic == "" {
		return fmt.Errorf("power sensor topic is empty")
	}

	// Apply min/max validation from register config if specified
	if register != nil {
		if register.Min != nil && result.Value < *register.Min {
			return fmt.Errorf("power value %.3f W below minimum threshold %.3f W", result.Value, *register.Min)
		}
		if register.Max != nil && result.Value > *register.Max {
			return fmt.Errorf("power value %.3f W above maximum threshold %.3f W", result.Value, *register.Max)
		}
	}

	return nil
}

// GetPowerDirection returns the power flow direction
func (p *PowerTopic) GetPowerDirection(value float64) string {
	switch {
	case value > 100:
		return "consuming"
	case value < -100:
		return "generating"
	default:
		return "balanced"
	}
}

// GetPowerLevel returns the power level assessment
func (p *PowerTopic) GetPowerLevel(value float64) string {
	absValue := math.Abs(value)
	switch {
	case absValue < 100:
		return "minimal"
	case absValue < 1000:
		return "low"
	case absValue < 5000:
		return "moderate"
	case absValue < 20000:
		return "high"
	default:
		return "very_high"
	}
}

// IsPowerBalanced checks if power is within balanced range
func (p *PowerTopic) IsPowerBalanced(value float64) bool {
	return math.Abs(value) <= 50 // Within 50W of zero
}

// CalculatePowerEfficiency calculates efficiency based on active vs apparent power
func (p *PowerTopic) CalculatePowerEfficiency(activePower, apparentPower float64) float64 {
	if apparentPower == 0 {
		return 0
	}
	return math.Abs(activePower) / apparentPower
}
