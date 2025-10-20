package mqtt

import (
	"context"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/modbus"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// TopicHandler defines the interface for different topic handlers
// Each handler publishes discovery and state for sensors
type TopicHandler interface {
	PublishDiscovery(ctx context.Context, client paho.Client, result *modbus.CommandResult, deviceInfo *DeviceInfo) error
	PublishState(ctx context.Context, client paho.Client, result *modbus.CommandResult) error
	GetTopicPrefix() string
	ValidateData(result *modbus.CommandResult, register *config.Register) error
}

// TopicContext manages the topic handlers
type TopicContext struct {
	handlers              map[string]TopicHandler
	deviceDiagnosticTopic *DeviceDiagnosticTopic // Separate handler for device diagnostics
	config                *config.HAConfig
	mqttConfig            *config.MQTTConfig
}

// NewTopicContext creates a new topic context with all handlers
func NewTopicContext(haCfg *config.HAConfig, mqttCfg *config.MQTTConfig) *TopicContext {
	ctx := &TopicContext{
		handlers:              make(map[string]TopicHandler),
		deviceDiagnosticTopic: NewDeviceDiagnosticTopic(haCfg),
		config:                haCfg,
		mqttConfig:            mqttCfg,
	}

	// Register all topic handlers
	ctx.handlers["voltage"] = NewVoltageTopic(haCfg)
	ctx.handlers["current"] = NewCurrentTopic(haCfg)
	ctx.handlers["frequency"] = NewFrequencyTopic(haCfg)
	ctx.handlers["power"] = NewPowerTopic(haCfg)
	ctx.handlers["power_factor"] = NewPowerFactorTopic(haCfg)
	ctx.handlers["energy"] = NewEnergyTopic(haCfg)
	ctx.handlers["sensor"] = NewSensorTopic(haCfg) // Keep as fallback
	ctx.handlers["status"] = NewStatusTopic(haCfg)
	ctx.handlers["diagnostic"] = NewDiagnosticTopic(haCfg)

	return ctx
}

// GetHandler returns the appropriate handler for a given topic type
func (tc *TopicContext) GetHandler(topicType string) TopicHandler {
	if handler, exists := tc.handlers[topicType]; exists {
		return handler
	}
	// Default to sensor handler
	return tc.handlers["sensor"]
}

// RegisterHandler allows registering custom topic handlers
func (tc *TopicContext) RegisterHandler(name string, handler TopicHandler) {
	tc.handlers[name] = handler
}

// GetDeviceDiagnosticTopic returns the device diagnostic topic handler
func (tc *TopicContext) GetDeviceDiagnosticTopic() *DeviceDiagnosticTopic {
	return tc.deviceDiagnosticTopic
}
