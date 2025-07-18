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

// EnergyTopic handles energy sensor publishing (total, imported, exported)
type EnergyTopic struct {
	config *config.HAConfig
}

// NewEnergyTopic creates a new energy topic handler
func NewEnergyTopic(config *config.HAConfig) *EnergyTopic {
	return &EnergyTopic{
		config: config,
	}
}

// PublishDiscovery publishes energy sensor discovery configuration
func (e *EnergyTopic) PublishDiscovery(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Extract sensor name from topic
	sensorName := extractSensorName(result.Topic)

	// Topic for discovery
	discoveryTopic := fmt.Sprintf("%s/sensor/%s_%s/config",
		e.config.DiscoveryPrefix, e.config.DeviceID, sensorName)

	// Configuration for the energy sensor
	config := SensorConfig{
		Name:              result.Name,
		UniqueID:          fmt.Sprintf("%s_%s", e.config.DeviceID, sensorName),
		StateTopic:        result.Topic + "/state",
		UnitOfMeasurement: result.Unit,
		DeviceClass:       result.DeviceClass,
		StateClass:        result.StateClass,
		Device: DeviceInfo{
			Name:         e.config.DeviceName,
			Identifiers:  []string{e.config.DeviceID},
			Manufacturer: e.config.Manufacturer,
			Model:        e.config.Model,
		},
		ValueTemplate:       "{{ value_json.value | round(3) }}",
		AvailabilityTopic:   e.config.StatusTopic,
		PayloadAvailable:    "online",
		PayloadNotAvailable: "offline",
	}

	// Serialize configuration
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error serializing energy configuration: %w", err)
	}

	// Publish configuration
	token := client.Publish(discoveryTopic, 0, true, configJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing energy discovery: %w", token.Error())
	}

	return nil
}

// PublishState publishes energy sensor state
func (e *EnergyTopic) PublishState(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client is not connected")
	}

	// Validate the result before publishing
	if err := e.ValidateData(result); err != nil {
		return fmt.Errorf("invalid energy data: %w", err)
	}

	// State topic
	stateTopic := result.Topic + "/state"

	// Energy sensor data with 3 decimal precision
	sensorData := SensorState{
		Value:     math.Round(result.Value*1000) / 1000, // Round to 3 decimal places
		Unit:      result.Unit,
		Timestamp: time.Now(),
	}

	// Serialize data
	dataJSON, err := json.Marshal(sensorData)
	if err != nil {
		return fmt.Errorf("error serializing energy data: %w", err)
	}

	// Publish state
	token := client.Publish(stateTopic, 0, false, dataJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing energy state: %w", token.Error())
	}

	return nil
}

// GetTopicPrefix returns the topic prefix for energy topic
func (e *EnergyTopic) GetTopicPrefix() string {
	return "energy"
}

// ValidateData validates energy sensor data before publishing
func (e *EnergyTopic) ValidateData(result *modbus.CommandResult) error {
	// Check for invalid numeric values
	if math.IsNaN(result.Value) {
		return fmt.Errorf("energy value is NaN for sensor %s", result.Name)
	}

	if math.IsInf(result.Value, 0) {
		return fmt.Errorf("energy value is infinite for sensor %s", result.Name)
	}

	// Energy specific validation - must be non-negative and reasonable
	if result.Value < 0 {
		return fmt.Errorf("energy value cannot be negative: %.3f kWh", result.Value)
	}

	if result.Value > 999999999 {
		return fmt.Errorf("energy value out of reasonable bounds: %.3f kWh (expected 0-999,999,999 kWh)", result.Value)
	}

	// Check required fields
	if result.Name == "" {
		return fmt.Errorf("energy sensor name is empty")
	}

	if result.Topic == "" {
		return fmt.Errorf("energy sensor topic is empty")
	}

	return nil
}

// GetEnergyUsageLevel returns usage level assessment
func (e *EnergyTopic) GetEnergyUsageLevel(value float64) string {
	switch {
	case value < 100:
		return "minimal"
	case value < 1000:
		return "low"
	case value < 5000:
		return "moderate"
	case value < 20000:
		return "high"
	default:
		return "very_high"
	}
}

// CalculateDailyUsage estimates daily usage based on current reading
func (e *EnergyTopic) CalculateDailyUsage(currentValue, previousValue float64, hoursElapsed float64) float64 {
	if hoursElapsed <= 0 {
		return 0
	}
	return (currentValue - previousValue) * 24 / hoursElapsed
}

// CalculateMonthlyCost calculates monthly cost based on kWh and rate
func (e *EnergyTopic) CalculateMonthlyCost(kwhUsed float64, ratePerKwh float64) float64 {
	return kwhUsed * ratePerKwh
}

// IsEnergyIncreasing checks if energy reading is increasing (normal for imported energy)
func (e *EnergyTopic) IsEnergyIncreasing(currentValue, previousValue float64) bool {
	return currentValue > previousValue
}

// DetectEnergySpike detects unusual energy spikes
func (e *EnergyTopic) DetectEnergySpike(currentValue, previousValue float64, maxChangePercent float64) bool {
	if previousValue <= 0 {
		return false
	}

	change := math.Abs(currentValue - previousValue)
	changePercent := (change / previousValue) * 100

	return changePercent > maxChangePercent
}

// GetEnergyDirection returns energy flow direction
func (e *EnergyTopic) GetEnergyDirection(sensorName string) string {
	switch {
	case sensorName == "energy_imported":
		return "consuming"
	case sensorName == "energy_exported":
		return "generating"
	default:
		return "total"
	}
}
