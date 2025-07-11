package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mqtt-modbus-bridge/internal/config"
	"mqtt-modbus-bridge/internal/modbus"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Publisher responsible for publishing data to Home Assistant
// Single Responsibility Principle - only handles publishing to HA
type Publisher struct {
	client     mqtt.Client
	config     *config.HAConfig
	mqttConfig *config.MQTTConfig
}

// NewPublisher creates a new publisher for Home Assistant
func NewPublisher(cfg *config.MQTTConfig, haCfg *config.HAConfig) *Publisher {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.Broker, cfg.Port))
	opts.SetClientID(cfg.ClientID + "_ha_publisher")
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetAutoReconnect(true)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	publisher := &Publisher{
		config:     haCfg,
		mqttConfig: cfg,
	}

	// Callback for connection
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Printf("✅ HA Publisher connected to MQTT broker")
	})

	// Callback for disconnection
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("❌ HA Publisher disconnected: %v", err)
	})

	publisher.client = mqtt.NewClient(opts)
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
		log.Printf("🔄 Attempting to connect HA publisher to MQTT broker (attempt %d)...", attempt)
		
		if token := p.client.Connect(); token.Wait() && token.Error() != nil {
			log.Printf("❌ HA Publisher connection failed (attempt %d): %v", attempt, token.Error())
			log.Printf("⏳ Retrying in %.0f seconds...", retryDelay.Seconds())
			
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
		log.Printf("🔌 HA Publisher connection token successful, waiting for connection establishment...")
		
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
			log.Printf("✅ HA Publisher successfully connected to MQTT broker after %d attempts", attempt)
			return nil
		}

		// Connection establishment timeout
		log.Printf("⏰ HA Publisher connection establishment timeout (attempt %d)", attempt)
		log.Printf("⏳ Retrying in %.0f seconds...", retryDelay.Seconds())
		
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

// PublishSensorDiscovery publishes discovery configuration for a sensor
func (p *Publisher) PublishSensorDiscovery(ctx context.Context, result *modbus.CommandResult) error {
	if !p.client.IsConnected() {
		return fmt.Errorf("publisher is not connected")
	}

	// Extract sensor name from topic
	sensorName := extractSensorName(result.Topic)

	// Topic for discovery
	discoveryTopic := fmt.Sprintf("%s/sensor/%s_%s/config",
		p.config.DiscoveryPrefix, p.config.DeviceID, sensorName)

	// Configuration for the sensor
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
		return fmt.Errorf("error serializing configuration: %w", err)
	}

	// Only log discovery publishing once to avoid spam
	// log.Printf("📡 Publishing discovery for %s: %s", result.Name, discoveryTopic)

	// Publish configuration
	token := p.client.Publish(discoveryTopic, 0, true, configJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing discovery: %w", token.Error())
	}

	return nil
}

// PublishSensorState publishes the state of a sensor
func (p *Publisher) PublishSensorState(ctx context.Context, result *modbus.CommandResult) error {
	if !p.client.IsConnected() {
		return fmt.Errorf("publisher is not connected")
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

	// Reduced verbosity - only log state publishing occasionally or on errors
	// log.Printf("📊 Publishing state for %s: %.3f %s", result.Name, result.Value, result.Unit)

	// Publish state
	token := p.client.Publish(stateTopic, 0, false, dataJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing state: %w", token.Error())
	}

	return nil
}

// PublishAllDiscoveries publishes discovery configurations for all sensors
func (p *Publisher) PublishAllDiscoveries(ctx context.Context, results []*modbus.CommandResult) error {
	for _, result := range results {
		if err := p.PublishSensorDiscovery(ctx, result); err != nil {
			log.Printf("❌ Error publishing discovery for %s: %v", result.Name, err)
			continue
		}

		// Small pause between publications
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// PublishStatus publishes the bridge status (online/offline) to Home Assistant
func (p *Publisher) PublishStatus(ctx context.Context, status string) error {
	if !p.client.IsConnected() {
		return fmt.Errorf("publisher not connected")
	}

	payload := status // "online" or "offline"

	token := p.client.Publish(p.config.StatusTopic, 0, true, payload)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		if token.Error() != nil {
			return fmt.Errorf("error publishing status: %w", token.Error())
		}
	}

	log.Printf("📡 Published bridge status: %s", status)
	return nil
}

// PublishDiagnostic publishes diagnostic information to Home Assistant
func (p *Publisher) PublishDiagnostic(ctx context.Context, code int, message string) error {
	if !p.client.IsConnected() {
		return fmt.Errorf("publisher not connected")
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

	token := p.client.Publish(p.config.DiagnosticTopic, 0, false, payload)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		if token.Error() != nil {
			return fmt.Errorf("error publishing diagnostic: %w", token.Error())
		}
	}

	log.Printf("🔧 Published diagnostic: [%d] %s", code, message)
	return nil
}

// PublishStatusOnline publishes "online" status - convenience method
func (p *Publisher) PublishStatusOnline(ctx context.Context) error {
	return p.PublishStatus(ctx, "online")
}

// PublishStatusOffline publishes "offline" status - convenience method
func (p *Publisher) PublishStatusOffline(ctx context.Context) error {
	return p.PublishStatus(ctx, "offline")
}

// PublishDiagnosticDiscovery publishes discovery configuration for diagnostic sensor
func (p *Publisher) PublishDiagnosticDiscovery(ctx context.Context) error {
	if !p.client.IsConnected() {
		return fmt.Errorf("publisher is not connected")
	}

	// Topic for diagnostic sensor discovery
	discoveryTopic := fmt.Sprintf("%s/sensor/%s_diagnostic/config",
		p.config.DiscoveryPrefix, p.config.DeviceID)

	// Configuration for the diagnostic sensor
	config := SensorConfig{
		Name:              "Diagnostic",
		UniqueID:          fmt.Sprintf("%s_diagnostic", p.config.DeviceID),
		StateTopic:        p.config.DiagnosticTopic,
		UnitOfMeasurement: "",
		DeviceClass:       "enum",
		StateClass:        "",
		Device: DeviceInfo{
			Name:         p.config.DeviceName,
			Identifiers:  []string{p.config.DeviceID},
			Manufacturer: p.config.Manufacturer,
			Model:        p.config.Model,
		},
		ValueTemplate:          "{{ value_json.message }}",
		AvailabilityTopic:      p.config.StatusTopic,
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

	log.Printf("📡 Publishing diagnostic discovery: %s", discoveryTopic)

	// Publish configuration
	token := p.client.Publish(discoveryTopic, 0, true, configJSON)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing diagnostic discovery: %w", token.Error())
	}

	return nil
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

// extractSensorName extracts the sensor name from the topic
func extractSensorName(topic string) string {
	// Extract the last part from the topic
	// E.g.: "sensor/energy_meter/voltage" -> "voltage"
	parts := []rune(topic)
	lastSlash := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash != -1 && lastSlash < len(parts)-1 {
		return string(parts[lastSlash+1:])
	}

	return topic
}
