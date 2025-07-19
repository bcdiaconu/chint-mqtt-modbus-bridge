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

// PowerFactorTopic handles power factor sensor publishing
type PowerFactorTopic struct {
	config *config.HAConfig
}

// NewPowerFactorTopic creates a new power factor topic handler
func NewPowerFactorTopic(config *config.HAConfig) *PowerFactorTopic {
	return &PowerFactorTopic{
		config: config,
	}
}

// PublishDiscovery publishes power factor sensor discovery configuration
func (pf *PowerFactorTopic) PublishDiscovery(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Extract sensor name from topic
	sensorName := extractSensorName(result.Topic)

	// Topic for discovery
	discoveryTopic := fmt.Sprintf("%s/sensor/%s_%s/config",
		pf.config.DiscoveryPrefix, pf.config.DeviceID, sensorName)

	// Configuration for the power factor sensor
	config := SensorConfig{
		Name:              result.Name,
		UniqueID:          fmt.Sprintf("%s_%s", pf.config.DeviceID, sensorName),
		StateTopic:        result.Topic + "/state",
		UnitOfMeasurement: result.Unit,
		DeviceClass:       result.DeviceClass,
		StateClass:        result.StateClass,
		Device: DeviceInfo{
			Name:         pf.config.DeviceName,
			Identifiers:  []string{pf.config.DeviceID},
			Manufacturer: pf.config.Manufacturer,
			Model:        pf.config.Model,
		},
		ValueTemplate:       "{{ value_json.value | round(2) }}",
		AvailabilityTopic:   pf.config.StatusTopic,
		PayloadAvailable:    "online",
		PayloadNotAvailable: "offline",
	}

	// Serialize configuration
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializing power factor configuration: %w", err)
	}

	// Publish configuration
	token := client.Publish(discoveryTopic, 0, true, configJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing power factor discovery: %w", token.Error())
	}

	return nil
}

// PublishState publishes power factor sensor state
func (pf *PowerFactorTopic) PublishState(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Validate the result before publishing
	if err := pf.ValidateData(result, nil); err != nil {
		return fmt.Errorf("invalid power factor data: %w", err)
	}

	// State topic
	stateTopic := result.Topic + "/state"

	// Power factor sensor data with 2 decimal precision
	sensorData := SensorState{
		Value:     math.Round(result.Value*100) / 100, // Round to 2 decimal places
		Unit:      result.Unit,
		Timestamp: time.Now(),
	}

	// Serialize data
	dataJSON, err := json.Marshal(sensorData)
	if err != nil {
		return fmt.Errorf("error serializing power factor data: %w", err)
	}

	// Publish state
	token := client.Publish(stateTopic, 0, false, dataJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing power factor state: %w", token.Error())
	}

	return nil
}

// GetTopicPrefix returns the topic prefix for power factor topic
func (pf *PowerFactorTopic) GetTopicPrefix() string {
	return "power_factor"
}

// ValidateData validates power factor sensor data before publishing
func (pf *PowerFactorTopic) ValidateData(result *modbus.CommandResult, register *config.Register) error {
	// Check for invalid numeric values (NaN, Inf)
	if math.IsNaN(result.Value) {
		return fmt.Errorf("power factor value is NaN for sensor %s", result.Name)
	}

	if math.IsInf(result.Value, 0) {
		return fmt.Errorf("power factor value is infinite for sensor %s", result.Name)
	}

	// Check required fields
	if result.Name == "" {
		return fmt.Errorf("power factor sensor name is empty")
	}

	if result.Topic == "" {
		return fmt.Errorf("power factor sensor topic is empty")
	}

	// Apply min/max validation from register config if specified
	if register != nil {
		if register.Min != nil && result.Value < *register.Min {
			return fmt.Errorf("power factor value %.3f below minimum threshold %.3f", result.Value, *register.Min)
		}
		if register.Max != nil && result.Value > *register.Max {
			return fmt.Errorf("power factor value %.3f above maximum threshold %.3f", result.Value, *register.Max)
		}
	}

	return nil
}

// GetPowerFactorQuality returns quality assessment of the power factor
func (pf *PowerFactorTopic) GetPowerFactorQuality(value float64) string {
	switch {
	case value >= 0.95:
		return "excellent"
	case value >= 0.90:
		return "good"
	case value >= 0.80:
		return "acceptable"
	case value >= 0.70:
		return "poor"
	default:
		return "very_poor"
	}
}

// IsLeadingPowerFactor checks if power factor indicates leading load
func (pf *PowerFactorTopic) IsLeadingPowerFactor(value float64) bool {
	// This would require phase angle information, simplified check
	return value > 0.98
}

// GetEfficiencyImpact returns the efficiency impact assessment
func (pf *PowerFactorTopic) GetEfficiencyImpact(value float64) string {
	switch {
	case value >= 0.95:
		return "minimal_loss"
	case value >= 0.90:
		return "low_loss"
	case value >= 0.80:
		return "moderate_loss"
	case value >= 0.70:
		return "high_loss"
	default:
		return "severe_loss"
	}
}

// CalculateReactivePowerRatio calculates reactive power ratio from power factor
func (pf *PowerFactorTopic) CalculateReactivePowerRatio(powerFactor float64) float64 {
	if powerFactor <= 0 || powerFactor >= 1 {
		return 0
	}
	return math.Sqrt(1 - powerFactor*powerFactor)
}
