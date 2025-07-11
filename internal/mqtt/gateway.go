package mqtt

import (
	"context"
	"fmt"
	"log"
	"mqtt-modbus-bridge/internal/config"
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

	// Callback for connection
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		gateway.mu.Lock()
		gateway.connected = true
		gateway.mu.Unlock()

		log.Printf("✅ Gateway connected to MQTT broker")

		// Subscribe to data topic
		if token := client.Subscribe(cfg.Gateway.DataTopic, 0, gateway.onMessage); token.Wait() && token.Error() != nil {
			log.Printf("❌ Error subscribing to %s: %v", cfg.Gateway.DataTopic, token.Error())
		} else {
			log.Printf("📡 Gateway subscribed to: %s", cfg.Gateway.DataTopic)
		}
	})

	// Callback for disconnection
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		gateway.mu.Lock()
		gateway.connected = false
		gateway.mu.Unlock()
		log.Printf("❌ Gateway disconnected: %v", err)
	})

	gateway.client = mqtt.NewClient(opts)
	return gateway
}

// Connect connects the gateway to broker
func (g *USRGateway) Connect(ctx context.Context) error {
	if token := g.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("error connecting MQTT gateway: %w", token.Error())
	}

	// Wait for connection
	for i := 0; i < 50; i++ {
		if g.IsConnected() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	return fmt.Errorf("timeout connecting gateway")
}

// Disconnect disconnects the gateway
func (g *USRGateway) Disconnect() {
	g.mu.Lock()
	g.connected = false
	g.mu.Unlock()

	if g.client.IsConnected() {
		g.client.Disconnect(250)
	}
	close(g.responseChan)
}

// SendCommand sends a Modbus command through MQTT
func (g *USRGateway) SendCommand(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16) error {
	if !g.IsConnected() {
		return fmt.Errorf("gateway is not connected")
	}

	// Build Modbus RTU command
	command := buildModbusCommand(slaveID, functionCode, address, count)

	// Only log command details in debug mode or occasionally
	// log.Printf("📤 Sending command: %s to %s", hex.EncodeToString(command), g.config.Gateway.CmdTopic)

	// Publish command as raw bytes (USR-DR164 expects binary data)
	token := g.client.Publish(g.config.Gateway.CmdTopic, 0, false, command)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing command: %w", token.Error())
	}

	return nil
}

// WaitForResponse waits for response from gateway
func (g *USRGateway) WaitForResponse(ctx context.Context, timeoutSeconds int) ([]byte, error) {
	timeout := time.Duration(timeoutSeconds) * time.Second

	select {
	case response := <-g.responseChan:
		return response, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for response (%d seconds)", timeoutSeconds)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// IsConnected checks if gateway is connected
func (g *USRGateway) IsConnected() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.connected && g.client.IsConnected()
}

// onMessage callback for received MQTT messages
func (g *USRGateway) onMessage(client mqtt.Client, msg mqtt.Message) {
	data := msg.Payload()
	// Only log response details in debug mode or occasionally
	// log.Printf("📥 Response received: %s", hex.EncodeToString(data))

	// Extract useful data from Modbus response
	if len(data) >= 5 {
		// Skip Slave ID, Function Code and Byte Count
		// For Read Holding Registers, data starts at position 3
		if data[1] == 0x03 { // Function Code 03
			byteCount := int(data[2])
			if len(data) >= 3+byteCount {
				actualData := data[3 : 3+byteCount]

				// Send data to channel
				select {
				case g.responseChan <- actualData:
				default:
					log.Printf("⚠️ Response channel full, response ignored")
				}
			}
		}
	}
}

// buildModbusCommand builds a Modbus RTU command
func buildModbusCommand(slaveID uint8, functionCode uint8, address uint16, count uint16) []byte {
	cmd := make([]byte, 8)
	cmd[0] = slaveID
	cmd[1] = functionCode
	cmd[2] = byte(address >> 8)   // Address High
	cmd[3] = byte(address & 0xFF) // Address Low
	cmd[4] = byte(count >> 8)     // Count High
	cmd[5] = byte(count & 0xFF)   // Count Low

	// Calculate CRC16
	crc := calculateCRC16(cmd[:6])
	cmd[6] = byte(crc & 0xFF) // CRC Low
	cmd[7] = byte(crc >> 8)   // CRC High

	return cmd
}

// calculateCRC16 calculates CRC16 for Modbus RTU
func calculateCRC16(data []byte) uint16 {
	var crc uint16 = 0xFFFF

	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 == 1 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}

	return crc
}
