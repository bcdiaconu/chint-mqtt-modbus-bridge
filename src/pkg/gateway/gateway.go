package gateway

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/logger"
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
	commandMutex sync.Mutex // Synchronize command/response pairs
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
		logger.LogInfo("Gateway connected to MQTT broker")

		// Subscribe to data topic
		if token := client.Subscribe(cfg.Gateway.DataTopic, 0, gateway.onMessage); token.Wait() && token.Error() != nil {
			logger.LogError("Error subscribing to %s: %v", cfg.Gateway.DataTopic, token.Error())
		} else {
			logger.LogInfo("Gateway subscribed to: %s", cfg.Gateway.DataTopic)
		}
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		gateway.mu.Lock()
		gateway.connected = false
		gateway.mu.Unlock()
		logger.LogError("Gateway disconnected: %v", err)
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
		logger.LogDebug("Attempting to connect gateway to MQTT broker (attempt %d)...", attempt)

		if token := g.client.Connect(); token.Wait() && token.Error() != nil {
			logger.LogError("Gateway connection failed (attempt %d): %v", attempt, token.Error())
			logger.LogInfo("Retrying in %.0f seconds...", retryDelay.Seconds())
			// Wait for retry delay or context cancellation
			select {
			case <-ctx.Done():
				return fmt.Errorf("connection cancelled: %w", ctx.Err())
			case <-time.After(retryDelay):
				attempt++
				continue
			}
		}

		// Connection successful, wait for full connection establishment
		logger.LogDebug("Gateway connection token successful, waiting for connection establishment...")

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
			logger.LogInfo("Gateway successfully connected to MQTT broker after %d attempts", attempt)
			return nil
		}

		// Connection establishment timeout - retry with better error handling
		logger.LogWarn("Gateway connection establishment timeout (attempt %d)", attempt)
		logger.LogInfo("Retrying in %.0f seconds...", retryDelay.Seconds())

		// Ensure clean disconnection before retry
		if g.client.IsConnected() {
			g.client.Disconnect(250)
			time.Sleep(250 * time.Millisecond) // Wait for clean disconnection
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("connection cancelled during timeout: %w", ctx.Err())
		case <-time.After(retryDelay):
			attempt++
			continue
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
	// Check connection state with proper locking
	g.mu.RLock()
	connected := g.connected && g.client != nil && g.client.IsConnected()
	g.mu.RUnlock()

	if !connected {
		return fmt.Errorf("gateway is not connected")
	}

	// Clear response channel to avoid stale responses
	select {
	case <-g.responseChan:
		// Drained one stale response
	default:
		// Channel was already empty
	}

	// Create Modbus RTU command as binary data
	command := g.buildModbusCommand(slaveID, functionCode, address, count)

	// Send command with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Send command
	logger.LogDebug("Gateway sending modbus command: %02X to topic: %s", command, g.config.Gateway.CmdTopic)

	token := g.client.Publish(g.config.Gateway.CmdTopic, 0, false, command)

	// Wait for publish completion with context timeout
	select {
	case <-ctx.Done():
		return fmt.Errorf("command send timeout: %w", ctx.Err())
	default:
		if token.Wait() && token.Error() != nil {
			return fmt.Errorf("error publishing command: %w", token.Error())
		}
	}

	return nil
}

// WaitForResponse waits for response from gateway - implements modbus.Gateway interface
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

// Disconnect closes the gateway connection
func (g *USRGateway) Disconnect() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.connected {
		g.connected = false
		if g.client != nil && g.client.IsConnected() {
			g.client.Disconnect(250)
		}
	}
}

// Close closes the gateway connection
func (g *USRGateway) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.connected {
		g.connected = false
		if g.client != nil && g.client.IsConnected() {
			g.client.Disconnect(250)
		}
	}

	// Close the response channel safely
	if g.responseChan != nil {
		close(g.responseChan)
		g.responseChan = nil
	}
}

// onMessage handles incoming MQTT messages
func (g *USRGateway) onMessage(client mqtt.Client, msg mqtt.Message) {
	data := msg.Payload()
	logger.LogTrace("Gateway received message on %s: %02X", msg.Topic(), data)

	// Extract useful data from Modbus response
	if len(data) >= 5 {
		// Skip Slave ID, Function Code and Byte Count
		// For Read Holding Registers, data starts at position 3
		if data[1] == 0x03 { // Function Code 03
			byteCount := int(data[2])
			if len(data) >= 3+byteCount {
				actualData := data[3 : 3+byteCount]
				logger.LogDebug("Gateway received response: %02X", actualData)

				// Send data to channel
				select {
				case g.responseChan <- actualData:
				default:
					logger.LogWarn("Response channel full, response ignored")
				}
			}
		}
	}
}

// buildModbusCommand builds a Modbus RTU command as binary data
func (g *USRGateway) buildModbusCommand(slaveID uint8, functionCode uint8, address uint16, count uint16) []byte {
	cmd := make([]byte, 8)
	cmd[0] = slaveID
	cmd[1] = functionCode
	cmd[2] = byte(address >> 8)   // Address High
	cmd[3] = byte(address & 0xFF) // Address Low
	cmd[4] = byte(count >> 8)     // Count High
	cmd[5] = byte(count & 0xFF)   // Count Low

	// Calculate CRC16
	crc := g.calculateCRC16(cmd[:6])
	cmd[6] = byte(crc & 0xFF) // CRC Low
	cmd[7] = byte(crc >> 8)   // CRC High

	return cmd
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

// SendDiagnosticCommand sends a diagnostic command to test gateway connectivity
func (g *USRGateway) SendDiagnosticCommand(ctx context.Context) error {
	if !g.IsConnected() {
		return fmt.Errorf("gateway not connected")
	}

	// Send a simple read input registers command to test connectivity
	// This is a basic Modbus command that most devices support
	diagnosticCommand := g.buildModbusCommand(11, 3, 0, 2) // Read 2 registers starting at address 0

	// logger.LogInfo("🔍 Sending diagnostic command to test gateway connectivity...")
	// logger.LogDebug("📤 Diagnostic command: %02X", diagnosticCommand)

	if token := g.client.Publish(g.config.Gateway.CmdTopic, 0, false, diagnosticCommand); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to send diagnostic command: %v", token.Error())
	}

	// Wait for response with longer timeout
	select {
	case response := <-g.responseChan:
		// logger.LogInfo("✅ Diagnostic response received: %02X", response)
		_ = response // Use the response variable
		return nil
	case <-time.After(10 * time.Second):
		// logger.LogError("❌ Diagnostic command timeout - gateway may not be responding")
		return fmt.Errorf("diagnostic command timeout")
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}
}

// SendCommandAndWaitForResponse sends a command and waits for response atomically
// This prevents racing conditions between multiple commands
func (g *USRGateway) SendCommandAndWaitForResponse(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16, timeoutSeconds int) ([]byte, error) {
	// Lock to ensure only one command/response cycle at a time
	g.commandMutex.Lock()
	defer g.commandMutex.Unlock()

	// Send command
	if err := g.SendCommand(ctx, slaveID, functionCode, address, count); err != nil {
		return nil, err
	}

	// Wait for response
	response, err := g.WaitForResponse(ctx, timeoutSeconds)
	if err != nil {
		return nil, err
	}

	// Add small delay between commands to prevent gateway overload
	time.Sleep(50 * time.Millisecond)

	return response, nil
}
