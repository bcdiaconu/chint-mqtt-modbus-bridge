package gateway

import (
	"context"
	"encoding/binary"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/crc"
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

	// Track expected response for validation
	expectedSlaveID      uint8
	expectedFunctionCode uint8
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
	// Wait for connection with short retry (handle brief disconnections during auto-reconnect)
	maxWait := 3 * time.Second
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		g.mu.RLock()
		connected := g.connected && g.client != nil && g.client.IsConnected()
		g.mu.RUnlock()

		if connected {
			break
		}

		// Connection not ready, wait a bit before retrying
		time.Sleep(100 * time.Millisecond)
	}

	// Final connection check
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

	// Verify CRC of the received message
	if !crc.VerifyCRC(data) {
		logger.LogWarn("Received message with invalid CRC, ignoring: %02X", data)
		return
	}

	// Validate minimum message length
	if len(data) < 5 {
		logger.LogWarn("Received message too short (len=%d), ignoring: %02X", len(data), data)
		return
	}

	// Extract Slave ID and Function Code for validation
	receivedSlaveID := data[0]
	receivedFunctionCode := data[1]

	// Validate this is the response we're expecting (protect against out-of-order responses)
	g.commandMutex.Lock()
	expectedSlaveID := g.expectedSlaveID
	expectedFunctionCode := g.expectedFunctionCode
	g.commandMutex.Unlock()

	if receivedSlaveID != expectedSlaveID || receivedFunctionCode != expectedFunctionCode {
		logger.LogWarn("Received unexpected response (Slave=%d, Func=0x%02X) but expecting (Slave=%d, Func=0x%02X), ignoring",
			receivedSlaveID, receivedFunctionCode, expectedSlaveID, expectedFunctionCode)
		return
	}

	// Extract useful data from Modbus response
	// For Read Holding Registers (0x03), data starts at position 3
	if receivedFunctionCode == 0x03 { // Function Code 03
		byteCount := int(data[2])
		if len(data) >= 3+byteCount+2 { // +2 for CRC
			actualData := data[3 : 3+byteCount]
			logger.LogDebug("Gateway received valid response from Slave %d: %02X", receivedSlaveID, actualData)

			// Send data to channel (non-blocking to prevent deadlock)
			select {
			case g.responseChan <- actualData:
				// Successfully sent
			default:
				logger.LogWarn("Response channel full, response ignored (Slave=%d, Func=0x%02X)",
					receivedSlaveID, receivedFunctionCode)
			}
		} else {
			logger.LogWarn("Invalid byte count in response: expected %d bytes but message too short", byteCount)
		}
	} else {
		logger.LogWarn("Unsupported function code in response: 0x%02X", receivedFunctionCode)
	}
}

// buildModbusCommand builds a Modbus RTU command with CRC
func (g *USRGateway) buildModbusCommand(slaveID uint8, functionCode uint8, address uint16, count uint16) []byte {
	// Build the command without CRC
	command := make([]byte, 6)
	command[0] = slaveID
	command[1] = functionCode
	binary.BigEndian.PutUint16(command[2:4], address)
	binary.BigEndian.PutUint16(command[4:6], count)

	// Append CRC using crc package
	return crc.AppendCRC(command)
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
// This prevents racing conditions between multiple commands and ensures SEQUENTIAL execution
//
// CRITICAL: This method uses commandMutex to ensure:
// 1. Only ONE Modbus transaction at a time (no overlap between slaves/groups)
// 2. Each request gets its CORRECT response (validated by SlaveID + FunctionCode)
// 3. Stale responses from timed-out requests are cleared before new requests
//
// Execution flow:
//
//	Lock commandMutex → Clear stale responses → Set expected response params →
//	Send command → Wait for matching response → Unlock → 50ms delay
//
// This guarantees that register groups from different slaves (e.g., Slave 1 and Slave 11)
// are executed SEQUENTIALLY and cannot interfere with each other.
func (g *USRGateway) SendCommandAndWaitForResponse(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16, timeoutSeconds int) ([]byte, error) {
	// Lock to ensure only one command/response cycle at a time
	g.commandMutex.Lock()
	defer g.commandMutex.Unlock()

	// Clear any stale responses from channel before sending new command
	// This prevents receiving old responses for new requests
	select {
	case <-g.responseChan:
		logger.LogWarn("Cleared stale response from channel before new request")
	default:
		// Channel is empty, good to go
	}

	// Set expected response parameters for validation in onMessage
	g.expectedSlaveID = slaveID
	g.expectedFunctionCode = functionCode

	// Send command
	if err := g.SendCommand(ctx, slaveID, functionCode, address, count); err != nil {
		return nil, err
	}

	// Wait for response
	response, err := g.WaitForResponse(ctx, timeoutSeconds)

	if err != nil {
		// CRITICAL: After timeout, clear any stale response that might arrive late
		// Do this SYNCHRONOUSLY before releasing mutex to prevent next request
		// from consuming this stale response
		//
		// Small delay to allow onMessage() to process any fragments that arrived
		// just before timeout expired
		time.Sleep(50 * time.Millisecond)

		select {
		case staleResp := <-g.responseChan:
			logger.LogWarn("⚠️ Discarded late response after timeout: %d bytes from potential Slave %d",
				len(staleResp), g.expectedSlaveID)
		default:
			// No stale response in channel
		}

		return nil, err
	}

	// Add small delay between commands to prevent gateway overload
	time.Sleep(50 * time.Millisecond)

	return response, nil
}
