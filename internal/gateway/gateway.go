package gateway

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/internal/config"
	"mqtt-modbus-bridge/internal/logger"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// USRGateway implementation of USR-DR164 gateway
// Single Responsibility Principle - only handles MQTT communication with gateway
type USRGateway struct {
	client       mqtt.Client
	config       *config.MQTTConfig
	responseChan chan []byte
	mu           sync.RWMutex
	connected    bool
}

// NewUSRGateway creates a new USR-DR164 gateway
func NewUSRGateway(cfg *config.MQTTConfig) *USRGateway {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.Broker, cfg.Port))
	opts.SetClientID(cfg.ClientID + "_gateway")
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetAutoReconnect(true)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	gateway := &USRGateway{
		config:       cfg,
		responseChan: make(chan []byte, 10),
	}

	// Connection status callbacks
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		gateway.mu.Lock()
		gateway.connected = true
		gateway.mu.Unlock()
		logger.LogInfo("✅ Gateway connected to MQTT broker")

		// Subscribe to data topic
		if token := client.Subscribe(cfg.Gateway.DataTopic, 0, gateway.onMessage); token.Wait() && token.Error() != nil {
			logger.LogError("❌ Error subscribing to %s: %v", cfg.Gateway.DataTopic, token.Error())
		} else {
			logger.LogInfo("📡 Gateway subscribed to: %s", cfg.Gateway.DataTopic)
		}
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		gateway.mu.Lock()
		gateway.connected = false
		gateway.mu.Unlock()
		logger.LogError("❌ Gateway disconnected: %v", err)
	})

	gateway.client = mqtt.NewClient(opts)
	return gateway
}

// Connect connects the gateway to broker with infinite retry
func (g *USRGateway) Connect(ctx context.Context) error {
	retryDelay := time.Duration(g.config.RetryDelay) * time.Millisecond
	if retryDelay == 0 {
		retryDelay = 5000 * time.Millisecond // Default 5 seconds
	}

	attempt := 1
	for {
		logger.LogDebug("🔄 Attempting to connect gateway to MQTT broker (attempt %d)...", attempt)

		if token := g.client.Connect(); token.Wait() && token.Error() != nil {
			logger.LogError("❌ Gateway connection failed (attempt %d): %v", attempt, token.Error())
			logger.LogInfo("⏳ Retrying in %.0f seconds...", retryDelay.Seconds())
			attempt++
			select {
			case <-ctx.Done():
				return fmt.Errorf("connection cancelled: %w", ctx.Err())
			case <-time.After(retryDelay):
				continue
			}
		}

		// Connection successful, wait for full connection establishment
		logger.LogDebug("🔌 Gateway connection token successful, waiting for connection establishment...")

		// Wait for connection with timeout
		connected := false
		for i := 0; i < 50; i++ {
			if g.IsConnected() {
				connected = true
				break
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("connection cancelled during establishment: %w", ctx.Err())
			case <-time.After(100 * time.Millisecond):
			}
		}

		if connected {
			logger.LogInfo("✅ Gateway successfully connected to MQTT broker after %d attempts", attempt)
			return nil
		}

		// Connection not established, retry
		logger.LogWarn("⚠️ Gateway connection not fully established, retrying...")
		if g.client.IsConnected() {
			g.client.Disconnect(250)
		}
		attempt++
		select {
		case <-ctx.Done():
			return fmt.Errorf("connection cancelled: %w", ctx.Err())
		case <-time.After(retryDelay):
		}
	}
}

// IsConnected checks if the gateway is connected
func (g *USRGateway) IsConnected() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.connected && g.client.IsConnected()
}

// SendCommand sends a Modbus command through MQTT - implements modbus.Gateway interface
func (g *USRGateway) SendCommand(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16) error {
	if !g.IsConnected() {
		return fmt.Errorf("gateway not connected")
	}

	// Clear channel
	for len(g.responseChan) > 0 {
		<-g.responseChan
	}

	// Create Modbus RTU command in hex format
	command := g.createModbusCommand(slaveID, functionCode, address, count)

	// Send command
	logger.LogDebug("📤 Gateway sending modbus command: %s", command)
	if token := g.client.Publish(g.config.Gateway.CmdTopic, 0, false, []byte(command)); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to send command: %v", token.Error())
	}

	return nil
}

// WaitForResponse waits for response from gateway - implements modbus.Gateway interface
func (g *USRGateway) WaitForResponse(ctx context.Context, timeout int) ([]byte, error) {
	timeoutDuration := time.Duration(timeout) * time.Millisecond
	if timeoutDuration == 0 {
		timeoutDuration = 5000 * time.Millisecond // Default 5 seconds
	}

	select {
	case response := <-g.responseChan:
		logger.LogDebug("📥 Gateway received response: %s", string(response))
		return response, nil
	case <-time.After(timeoutDuration):
		return nil, fmt.Errorf("gateway response timeout")
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	}
}

// Disconnect closes the gateway connection
func (g *USRGateway) Disconnect() {
	g.Close()
}

// Close closes the gateway connection
func (g *USRGateway) Close() {
	if g.IsConnected() {
		g.client.Disconnect(250)
	}
}

// onMessage handles incoming MQTT messages
func (g *USRGateway) onMessage(client mqtt.Client, msg mqtt.Message) {
	logger.LogDebug("📨 Gateway received message on %s: %s", msg.Topic(), string(msg.Payload()))

	// USR-DR164 sends hex response directly
	response := string(msg.Payload())

	// Validate that it's a hex string
	if g.isValidHexResponse(response) {
		logger.LogDebug("📥 Gateway received hex response: %s", response)
		select {
		case g.responseChan <- []byte(response):
		default:
			logger.LogWarn("⚠️ Gateway response channel full, discarding message")
		}
	} else {
		logger.LogWarn("⚠️ Gateway received invalid hex response: %s", response)
	}
}

// isValidHexResponse checks if the response is a valid hex string
func (g *USRGateway) isValidHexResponse(response string) bool {
	// Remove any whitespace
	response = strings.TrimSpace(response)

	// Check if it's a valid hex string (even length, hex characters only)
	if len(response) < 6 || len(response)%2 != 0 {
		return false
	}

	for _, c := range response {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}

	return true
}

// createModbusCommand creates a Modbus RTU command in hex format
func (g *USRGateway) createModbusCommand(slaveID uint8, functionCode uint8, address uint16, count uint16) string {
	// Create Modbus RTU frame
	frame := make([]byte, 8)

	// Modbus RTU frame format:
	// [0] Slave ID
	// [1] Function Code
	// [2-3] Starting Address (Big Endian)
	// [4-5] Quantity (Big Endian)
	// [6-7] CRC (Little Endian)

	frame[0] = slaveID
	frame[1] = functionCode
	frame[2] = byte(address >> 8)   // Address high byte
	frame[3] = byte(address & 0xFF) // Address low byte
	frame[4] = byte(count >> 8)     // Count high byte
	frame[5] = byte(count & 0xFF)   // Count low byte

	// Calculate CRC16
	crc := g.calculateCRC16(frame[:6])
	frame[6] = byte(crc & 0xFF) // CRC low byte
	frame[7] = byte(crc >> 8)   // CRC high byte

	// Convert to hex string
	return fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X",
		frame[0], frame[1], frame[2], frame[3],
		frame[4], frame[5], frame[6], frame[7])
}

// calculateCRC16 calculates CRC16 for Modbus RTU
func (g *USRGateway) calculateCRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)

	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc >>= 1
				crc ^= 0xA001
			} else {
				crc >>= 1
			}
		}
	}

	return crc
}
