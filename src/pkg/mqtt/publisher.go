package mqtt

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/logger"
	"mqtt-modbus-bridge/pkg/modbus"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// Publisher responsible for publishing data to Home Assistant using Topic Pattern
// Single Responsibility Principle - only handles publishing coordination
type Publisher struct {
	client     paho.Client
	config     *config.HAConfig
	mqttConfig *config.MQTTConfig
	context    *TopicContext
}

// NewPublisher creates a new publisher for Home Assistant
func NewPublisher(cfg *config.MQTTConfig, haCfg *config.HAConfig) *Publisher {
	opts := paho.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.Broker, cfg.Port))
	opts.SetClientID(cfg.ClientID + "_ha_publisher")
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetAutoReconnect(true)

	// Use keep_alive from config, default to 60 seconds if not specified
	keepAlive := cfg.KeepAlive
	if keepAlive == 0 {
		keepAlive = 60
	}
	opts.SetKeepAlive(time.Duration(keepAlive) * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	// Set Last Will and Testament to automatically mark as offline on disconnect
	opts.SetWill(haCfg.StatusTopic, "offline", 1, true)

	publisher := &Publisher{
		config:     haCfg,
		mqttConfig: cfg,
		context:    NewTopicContext(haCfg, cfg),
	}

	// Callback for connection
	opts.SetOnConnectHandler(func(client paho.Client) {
		logger.LogInfo("HA Publisher connected to MQTT broker")
		// Immediately publish online status when connected
		if token := client.Publish(haCfg.StatusTopic, 1, true, "online"); token.Wait() && token.Error() != nil {
			logger.LogWarn("Error publishing online status on connect: %v", token.Error())
		}
	})

	// Callback for disconnection
	opts.SetConnectionLostHandler(func(client paho.Client, err error) {
		logger.LogError("HA Publisher disconnected: %v", err)
	})

	publisher.client = paho.NewClient(opts)
	return publisher
}

// Connect connects the publisher to the broker with infinite retry
func (p *Publisher) Connect(ctx context.Context) error {
	retryDelay := time.Duration(p.mqttConfig.RetryDelay) * time.Millisecond
	if retryDelay == 0 {
		retryDelay = 5000 * time.Millisecond // Default 5 seconds
	}

	attempt := 1
	for {
		logger.LogDebug("🔄 Attempting to connect HA publisher to MQTT broker (attempt %d)...", attempt)

		if token := p.client.Connect(); token.Wait() && token.Error() != nil {
			logger.LogError("❌ HA Publisher connection failed (attempt %d): %v", attempt, token.Error())
			logger.LogInfo("⏳ Retrying in %.0f seconds...", retryDelay.Seconds())

			// Wait for retry delay or context cancellation
			select {
			case <-ctx.Done():
				return fmt.Errorf("HA publisher connection cancelled: %w", ctx.Err())
			case <-time.After(retryDelay):
				attempt++
				continue
			}
		}

		// Connection successful, wait for full connection establishment
		logger.LogDebug("🔌 HA Publisher connection token successful, waiting for connection establishment...")

		// Wait for connection with timeout
		connected := false
		for i := 0; i < 50; i++ {
			if p.client.IsConnected() {
				connected = true
				break
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("HA publisher connection cancelled during establishment: %w", ctx.Err())
			case <-time.After(100 * time.Millisecond):
			}
		}

		if connected {
			logger.LogInfo("✅ HA Publisher successfully connected to MQTT broker after %d attempts", attempt)
			return nil
		}

		// Connection establishment timeout
		logger.LogWarn("⏰ HA Publisher connection establishment timeout (attempt %d)", attempt)
		logger.LogInfo("⏳ Retrying in %.0f seconds...", retryDelay.Seconds())

		select {
		case <-ctx.Done():
			return fmt.Errorf("HA publisher connection cancelled during timeout: %w", ctx.Err())
		case <-time.After(retryDelay):
			attempt++
			continue
		}
	}
}

// Disconnect disconnects the publisher
func (p *Publisher) Disconnect() {
	if p.client.IsConnected() {
		p.client.Disconnect(250)
	}
}

// PublishSensorDiscovery publishes discovery configuration for a sensor using topic pattern
// deviceInfo contains the Home Assistant device information (nil for backward compatibility with global device)
func (p *Publisher) PublishSensorDiscovery(ctx context.Context, result *modbus.CommandResult, deviceInfo *DeviceInfo) error {
	// Determine topic type based on device class
	topicType := p.getTopicTypeFromDeviceClass(result.DeviceClass)
	handler := p.context.GetHandler(topicType)
	return handler.PublishDiscovery(ctx, p.client, result, deviceInfo)
}

// PublishSensorState publishes the state of a sensor using topic pattern
func (p *Publisher) PublishSensorState(ctx context.Context, result *modbus.CommandResult) error {
	// Determine topic type based on device class
	topicType := p.getTopicTypeFromDeviceClass(result.DeviceClass)
	handler := p.context.GetHandler(topicType)

	// Debug log: name, value, and full topic for debugging
	logger.LogDebug("📤 Publishing '%s' = %.2f %s → %s",
		result.Name, result.Value, result.Unit, result.Topic)

	return handler.PublishState(ctx, p.client, result)
}

// getTopicTypeFromDeviceClass maps device class to topic type
func (p *Publisher) getTopicTypeFromDeviceClass(deviceClass string) string {
	switch deviceClass {
	case "voltage":
		return "voltage"
	case "current":
		return "current"
	case "frequency":
		return "frequency"
	case "power", "apparent_power", "reactive_power":
		return "power"
	case "power_factor":
		return "power_factor"
	case "energy":
		return "energy"
	default:
		return "sensor" // fallback to generic sensor
	}
}

// PublishAllDiscoveries publishes discovery configurations for all sensors
// deviceInfo contains the Home Assistant device information (nil for backward compatibility)
func (p *Publisher) PublishAllDiscoveries(ctx context.Context, results []*modbus.CommandResult, deviceInfo *DeviceInfo) error {
	for _, result := range results {
		if err := p.PublishSensorDiscovery(ctx, result, deviceInfo); err != nil {
			logger.LogError("❌ Error publishing discovery for %s: %v", result.Name, err)
			continue
		}

		// Small pause between publications
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// PublishStatus publishes the bridge status using topic pattern
func (p *Publisher) PublishStatus(ctx context.Context, status string) error {
	handler := p.context.GetHandler("status").(*StatusTopic)

	if status == "online" {
		return handler.PublishOnline(ctx, p.client)
	} else {
		return handler.PublishOffline(ctx, p.client)
	}
}

// PublishDiagnostic publishes diagnostic information using topic pattern
func (p *Publisher) PublishDiagnostic(ctx context.Context, code int, message string) error {
	handler := p.context.GetHandler("diagnostic").(*DiagnosticTopic)
	return handler.PublishDiagnostic(ctx, p.client, code, message)
}

// PublishStatusOnline publishes "online" status - convenience method
func (p *Publisher) PublishStatusOnline(ctx context.Context) error {
	return p.PublishStatus(ctx, "online")
}

// PublishStatusOffline publishes "offline" status - convenience method
func (p *Publisher) PublishStatusOffline(ctx context.Context) error {
	return p.PublishStatus(ctx, "offline")
}

// PublishDiagnosticDiscovery publishes discovery configuration for diagnostic sensor using topic pattern
// This is for the bridge-level diagnostic sensor (not per-device)
func (p *Publisher) PublishDiagnosticDiscovery(ctx context.Context) error {
	handler := p.context.GetHandler("diagnostic")
	// Create a dummy result for discovery
	dummyResult := &modbus.CommandResult{
		Name: "Diagnostic",
	}

	// Create DeviceInfo for the bridge itself using constants
	bridgeDeviceInfo := &DeviceInfo{
		Name:         config.BridgeDeviceName,
		Identifiers:  []string{config.BridgeDeviceID},
		Manufacturer: config.BridgeDeviceManufacturer,
		Model:        config.BridgeDeviceModel,
	}

	return handler.PublishDiscovery(ctx, p.client, dummyResult, bridgeDeviceInfo)
}

// SensorConfig configuration for a Home Assistant sensor
type SensorConfig struct {
	Name                   string     `json:"name"`
	UniqueID               string     `json:"unique_id"`
	StateTopic             string     `json:"state_topic"`
	UnitOfMeasurement      string     `json:"unit_of_measurement,omitempty"`
	DeviceClass            string     `json:"device_class,omitempty"`
	StateClass             string     `json:"state_class,omitempty"`
	Device                 DeviceInfo `json:"device"`
	ValueTemplate          string     `json:"value_template"`
	AvailabilityTopic      string     `json:"availability_topic"`
	AvailabilityMode       string     `json:"availability_mode,omitempty"`
	PayloadAvailable       string     `json:"payload_available"`
	PayloadNotAvailable    string     `json:"payload_not_available"`
	JSONAttributesTemplate string     `json:"json_attributes_template,omitempty"`
	EntityCategory         string     `json:"entity_category,omitempty"`
}

// DeviceInfo information about the device
type DeviceInfo struct {
	Name         string   `json:"name"`
	Identifiers  []string `json:"identifiers"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
}

// SensorState state of a sensor
type SensorState struct {
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
}
