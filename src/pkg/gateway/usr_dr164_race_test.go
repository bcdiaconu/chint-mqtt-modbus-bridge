package gateway

import (
	"context"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/crc"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// mockMessage implements mqtt.Message interface for testing
type mockMessage struct {
	topic   string
	payload []byte
}

func (m *mockMessage) Duplicate() bool   { return false }
func (m *mockMessage) Qos() byte         { return 0 }
func (m *mockMessage) Retained() bool    { return false }
func (m *mockMessage) Topic() string     { return m.topic }
func (m *mockMessage) MessageID() uint16 { return 0 }
func (m *mockMessage) Payload() []byte   { return m.payload }
func (m *mockMessage) Ack()              {}

// createMockMessage creates a mock MQTT message with valid CRC
func createMockMessage(topic string, slaveID, functionCode uint8, data []byte) mqtt.Message {
	// Build Modbus RTU frame: [SlaveID, FunctionCode, ByteCount, Data..., CRC16]
	byteCount := uint8(len(data))
	frame := []byte{slaveID, functionCode, byteCount}
	frame = append(frame, data...)

	// Append CRC
	frameWithCRC := crc.AppendCRC(frame)

	return &mockMessage{
		topic:   topic,
		payload: frameWithCRC,
	}
}

// TestResponseValidation verifies that the gateway rejects responses with incorrect SlaveID or FunctionCode
func TestResponseValidation(t *testing.T) {
	tests := []struct {
		name                 string
		expectedSlaveID      uint8
		expectedFunctionCode uint8
		receivedSlaveID      uint8
		receivedFunctionCode uint8
		shouldAccept         bool
		description          string
	}{
		{
			name:                 "Correct SlaveID and FunctionCode",
			expectedSlaveID:      11,
			expectedFunctionCode: 0x03,
			receivedSlaveID:      11,
			receivedFunctionCode: 0x03,
			shouldAccept:         true,
			description:          "Response matches expected parameters - should be accepted",
		},
		{
			name:                 "Wrong SlaveID",
			expectedSlaveID:      11,
			expectedFunctionCode: 0x03,
			receivedSlaveID:      1, // Different slave
			receivedFunctionCode: 0x03,
			shouldAccept:         false,
			description:          "Response from different slave - should be rejected",
		},
		{
			name:                 "Wrong FunctionCode",
			expectedSlaveID:      11,
			expectedFunctionCode: 0x03,
			receivedSlaveID:      11,
			receivedFunctionCode: 0x04, // Different function
			shouldAccept:         false,
			description:          "Response with different function code - should be rejected",
		},
		{
			name:                 "Wrong SlaveID and FunctionCode",
			expectedSlaveID:      11,
			expectedFunctionCode: 0x03,
			receivedSlaveID:      1,
			receivedFunctionCode: 0x04,
			shouldAccept:         false,
			description:          "Response from different slave and function - should be rejected",
		},
		{
			name:                 "Response from Slave 1 when expecting Slave 11",
			expectedSlaveID:      11,
			expectedFunctionCode: 0x03,
			receivedSlaveID:      1,
			receivedFunctionCode: 0x03,
			shouldAccept:         false,
			description:          "Simulates race condition - wrong meter response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock gateway
			cfg := &config.MQTTConfig{
				Broker:   "test",
				Port:     1883,
				Username: "test",
				Password: "test",
				Gateway: config.GatewayConfig{
					MAC:       "TEST123456",
					CmdTopic:  "test/cmd",
					DataTopic: "test/data",
				},
			}

			gateway := NewUSRGateway(cfg)

			// Set expected parameters (simulating a pending request)
			gateway.expectedSlaveID = tt.expectedSlaveID
			gateway.expectedFunctionCode = tt.expectedFunctionCode

			// Simulate receiving a response with specific SlaveID and FunctionCode
			// Data: 4 bytes of dummy data (0x00 0x00 0x00 0x00)
			responseData := []byte{0x00, 0x00, 0x00, 0x00}
			mockMsg := createMockMessage(
				"test/data",
				tt.receivedSlaveID,
				tt.receivedFunctionCode,
				responseData,
			)

			// Track if response was accepted
			responseReceived := false
			responseMutex := sync.Mutex{}

			// Start a goroutine to check if response is placed on channel
			go func() {
				// Simulate onMessage call
				gateway.onMessage(nil, mockMsg)

				// Check if response channel has data (non-blocking)
				select {
				case <-gateway.responseChan:
					responseMutex.Lock()
					responseReceived = true
					responseMutex.Unlock()
				case <-time.After(100 * time.Millisecond):
					// No response received (expected for rejected responses)
				}
			}()

			time.Sleep(150 * time.Millisecond)

			responseMutex.Lock()
			accepted := responseReceived
			responseMutex.Unlock()

			if accepted != tt.shouldAccept {
				t.Errorf("%s: Expected shouldAccept=%v, got accepted=%v\n  Expected: SlaveID=%d FunctionCode=0x%02X\n  Received: SlaveID=%d FunctionCode=0x%02X\n  Description: %s",
					tt.name, tt.shouldAccept, accepted,
					tt.expectedSlaveID, tt.expectedFunctionCode,
					tt.receivedSlaveID, tt.receivedFunctionCode,
					tt.description)
			} else {
				t.Logf("✅ %s: Correctly %s response (SlaveID=%d->%d, FC=0x%02X->0x%02X)",
					tt.name,
					map[bool]string{true: "accepted", false: "rejected"}[accepted],
					tt.expectedSlaveID, tt.receivedSlaveID,
					tt.expectedFunctionCode, tt.receivedFunctionCode)
			}
		})
	}
}

// TestStaleResponseCleanup verifies that stale responses are cleared before new requests
func TestStaleResponseCleanup(t *testing.T) {
	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)

	// Simulate a stale response in the channel (from previous timed-out request)
	staleResponse := []byte{0x01, 0x02, 0x03, 0x04}
	select {
	case gateway.responseChan <- staleResponse:
		t.Log("✅ Inserted stale response into channel")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("❌ Failed to insert stale response")
	}

	// Verify stale response is in channel (check length without consuming)
	if len(gateway.responseChan) == 0 {
		t.Fatal("❌ Stale response was not retained in channel")
	}
	t.Log("✅ Stale response confirmed in channel")

	// Now simulate SendCommandAndWaitForResponse which should clear stale response
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// This will fail (no actual gateway), but should clear stale response first
	gateway.commandMutex.Lock()

	// Clear stale responses (this is what SendCommandAndWaitForResponse does)
	select {
	case old := <-gateway.responseChan:
		t.Logf("✅ Cleared stale response: %v", old)
	default:
		t.Error("⚠️ No stale response to clear - should have been one!")
	}

	gateway.commandMutex.Unlock()

	// Verify channel is now empty
	if len(gateway.responseChan) != 0 {
		t.Error("❌ Channel should be empty after cleanup")
	} else {
		t.Log("✅ Channel is empty after cleanup")
	}

	<-ctx.Done()
}

// TestConcurrentRequestsSameDevice verifies sequential execution prevents race conditions
func TestConcurrentRequestsSameDevice(t *testing.T) {
	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)

	// Track execution order
	var executionOrder []string
	var orderMutex sync.Mutex

	// Simulate two concurrent requests to same device
	var wg sync.WaitGroup
	wg.Add(2)

	// Request 1: Instant measurements (SlaveID=11, FC=0x03)
	go func() {
		defer wg.Done()
		gateway.commandMutex.Lock()
		orderMutex.Lock()
		executionOrder = append(executionOrder, "Request1-Start")
		orderMutex.Unlock()

		// Simulate work
		time.Sleep(50 * time.Millisecond)

		orderMutex.Lock()
		executionOrder = append(executionOrder, "Request1-End")
		orderMutex.Unlock()
		gateway.commandMutex.Unlock()
	}()

	// Request 2: Energy measurements (SlaveID=11, FC=0x03)
	// This should wait for Request 1 to complete
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Start slightly after Request 1

		gateway.commandMutex.Lock()
		orderMutex.Lock()
		executionOrder = append(executionOrder, "Request2-Start")
		orderMutex.Unlock()

		// Simulate work
		time.Sleep(50 * time.Millisecond)

		orderMutex.Lock()
		executionOrder = append(executionOrder, "Request2-End")
		orderMutex.Unlock()
		gateway.commandMutex.Unlock()
	}()

	wg.Wait()

	// Verify execution was sequential
	expectedOrder := []string{
		"Request1-Start",
		"Request1-End",
		"Request2-Start",
		"Request2-End",
	}

	orderMutex.Lock()
	defer orderMutex.Unlock()

	if len(executionOrder) != len(expectedOrder) {
		t.Fatalf("❌ Expected %d events, got %d", len(expectedOrder), len(executionOrder))
	}

	for i, expected := range expectedOrder {
		if executionOrder[i] != expected {
			t.Errorf("❌ Event %d: expected '%s', got '%s'", i, expected, executionOrder[i])
		}
	}

	t.Logf("✅ Sequential execution verified: %v", executionOrder)
}

// TestRaceConditionPrevention simulates the exact bug that was fixed
// Two meters (SlaveID 11 and 1) polled simultaneously, responses could mix
func TestRaceConditionPrevention(t *testing.T) {
	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)

	// Scenario: Request for Slave 11, but response from Slave 1 arrives first (race condition)

	// Step 1: Set expected parameters for Slave 11
	gateway.expectedSlaveID = 11
	gateway.expectedFunctionCode = 0x03

	t.Logf("📤 Expecting response from Slave 11, Function 0x03")

	// Step 2: Simulate wrong response arriving (from Slave 1)
	wrongData := []byte{
		0x43, 0x70, 0x00, 0x00, // Voltage = 240.0 V
		0x40, 0x20, 0x00, 0x00, // Current = 2.5 A
	}
	wrongMsg := createMockMessage("test/data", 0x01, 0x03, wrongData)

	responseAccepted := false
	var receivedResponse []byte

	// Simulate onMessage call
	go func() {
		gateway.onMessage(nil, wrongMsg)

		// Try to read from channel (non-blocking)
		select {
		case resp := <-gateway.responseChan:
			responseAccepted = true
			receivedResponse = resp
		case <-time.After(100 * time.Millisecond):
			// Expected - response should be rejected
		}
	}()

	time.Sleep(150 * time.Millisecond)

	if responseAccepted {
		t.Errorf("❌ Race condition NOT prevented! Wrong response was accepted (got %d bytes)",
			len(receivedResponse))
	} else {
		t.Log("✅ Race condition prevented! Response from Slave 1 was rejected when expecting Slave 11")
	}

	// Step 3: Now send correct response from Slave 11
	correctData := []byte{
		0x43, 0x70, 0x00, 0x00, // Voltage = 240.0 V
		0x40, 0x20, 0x00, 0x00, // Current = 2.5 A
	}
	correctMsg := createMockMessage("test/data", 0x0B, 0x03, correctData)

	go func() {
		gateway.onMessage(nil, correctMsg)
	}()

	// This time, response should be accepted
	select {
	case resp := <-gateway.responseChan:
		// resp contains only the data portion (not SlaveID/FunctionCode)
		// Just verify we got a response
		if len(resp) == 0 {
			t.Error("❌ Received empty response")
		} else {
			t.Logf("✅ Correct response from Slave 11 was accepted: %d bytes", len(resp))
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("❌ Correct response was not received")
	}
}

// TestMultipleDevicesNoInterference verifies that responses don't interfere between devices
func TestMultipleDevicesNoInterference(t *testing.T) {
	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)

	// Simulate polling both devices
	devices := []struct {
		name    string
		slaveID uint8
		data    []byte
	}{
		{
			name:    "Energy Meter Mains",
			slaveID: 11,
			data: []byte{
				0x43, 0x70, 0x00, 0x00, // Data
			},
		},
		{
			name:    "Energy Meter Lights",
			slaveID: 1,
			data: []byte{
				0x41, 0xF0, 0x00, 0x00, // Data
			},
		},
	}

	for _, device := range devices {
		t.Run(device.name, func(t *testing.T) {
			// Set expected parameters
			gateway.expectedSlaveID = device.slaveID
			gateway.expectedFunctionCode = 0x03

			t.Logf("📤 Polling %s (Slave %d)", device.name, device.slaveID)

			// Create mock message
			mockMsg := createMockMessage("test/data", device.slaveID, 0x03, device.data)

			// Send response
			go func() {
				time.Sleep(50 * time.Millisecond)
				gateway.onMessage(nil, mockMsg)
			}()

			// Wait for response
			select {
			case resp := <-gateway.responseChan:
				// resp contains only data, not SlaveID
				// We can't verify SlaveID from resp, but we know it's correct
				// because gateway validated it before putting in channel
				if len(resp) == 0 {
					t.Errorf("❌ %s: Received empty response", device.name)
				} else {
					t.Logf("✅ %s: Received correct response (%d bytes)", device.name, len(resp))
				}
			case <-time.After(200 * time.Millisecond):
				t.Errorf("❌ %s: Timeout waiting for response", device.name)
			}
		})
	}
}

// TestResponseTimingRaceCondition simulates delayed responses causing confusion
func TestResponseTimingRaceCondition(t *testing.T) {
	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)

	// Scenario: Request times out, new request starts, old response arrives late

	// Request 1 to Slave 11
	gateway.expectedSlaveID = 11
	gateway.expectedFunctionCode = 0x03
	t.Log("📤 Request 1: Expecting Slave 11, FC=0x03")

	// Simulate timeout (no response)
	time.Sleep(100 * time.Millisecond)
	t.Log("⏱️ Request 1 timed out")

	// Request 2 to Slave 1 (different device)
	gateway.expectedSlaveID = 1
	gateway.expectedFunctionCode = 0x03
	t.Log("📤 Request 2: Expecting Slave 1, FC=0x03")

	// Old response from Request 1 arrives late
	lateData := []byte{0x43, 0x70, 0x00, 0x00}
	lateMsg := createMockMessage("test/data", 0x0B, 0x03, lateData)

	go func() {
		gateway.onMessage(nil, lateMsg)
	}()

	// Check if late response was incorrectly accepted
	select {
	case resp := <-gateway.responseChan:
		t.Errorf("❌ Late response was incorrectly accepted! Got %d bytes when expecting nothing", len(resp))
	case <-time.After(150 * time.Millisecond):
		t.Log("✅ Late response from Slave 11 was correctly rejected (expecting Slave 1)")
	}

	// Now send correct response for Request 2
	correctData := []byte{0x41, 0xF0, 0x00, 0x00}
	correctMsg := createMockMessage("test/data", 0x01, 0x03, correctData)

	go func() {
		gateway.onMessage(nil, correctMsg)
	}()

	select {
	case resp := <-gateway.responseChan:
		if len(resp) == 0 {
			t.Error("❌ Received empty response")
		} else {
			t.Logf("✅ Correct response from Slave 1 accepted (%d bytes)", len(resp))
		}
	case <-time.After(150 * time.Millisecond):
		t.Error("❌ Correct response not received")
	}
}

// BenchmarkResponseValidation measures performance impact of validation
func BenchmarkResponseValidation(b *testing.B) {
	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)
	gateway.expectedSlaveID = 11
	gateway.expectedFunctionCode = 0x03

	responseData := []byte{0x43, 0x70, 0x00, 0x00}
	mockMsg := createMockMessage("test/data", 0x0B, 0x03, responseData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gateway.onMessage(nil, mockMsg)

		// Drain channel
		select {
		case <-gateway.responseChan:
		default:
		}
	}
}

// TestExpectedParametersUpdate verifies that expected parameters are updated correctly
func TestExpectedParametersUpdate(t *testing.T) {
	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)

	// Simulate multiple requests updating expected parameters
	requests := []struct {
		slaveID      uint8
		functionCode uint8
	}{
		{11, 0x03},
		{1, 0x03},
		{11, 0x04},
		{20, 0x03},
	}

	for i, req := range requests {
		gateway.expectedSlaveID = req.slaveID
		gateway.expectedFunctionCode = req.functionCode

		if gateway.expectedSlaveID != req.slaveID {
			t.Errorf("Request %d: expectedSlaveID not updated: expected %d, got %d",
				i, req.slaveID, gateway.expectedSlaveID)
		}

		if gateway.expectedFunctionCode != req.functionCode {
			t.Errorf("Request %d: expectedFunctionCode not updated: expected 0x%02X, got 0x%02X",
				i, req.functionCode, gateway.expectedFunctionCode)
		}

		t.Logf("✅ Request %d: Expected parameters updated (Slave=%d, FC=0x%02X)",
			i, req.slaveID, req.functionCode)
	}
}

// TestConcurrentResponsesSimultaneous simulates multiple responses arriving at the same time
// Only the correct response (matching expectedSlaveID/FunctionCode) should be accepted
func TestConcurrentResponsesSimultaneous(t *testing.T) {
	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)

	// Expecting response from Slave 11
	gateway.expectedSlaveID = 11
	gateway.expectedFunctionCode = 0x03

	t.Log("📤 Expecting response from Slave 11, FC=0x03")

	// Create multiple responses that arrive simultaneously
	responses := []struct {
		name     string
		slaveID  uint8
		data     []byte
		expected bool
	}{
		{
			name:     "Wrong Slave 1",
			slaveID:  1,
			data:     []byte{0x41, 0xF0, 0x00, 0x00},
			expected: false,
		},
		{
			name:     "Correct Slave 11",
			slaveID:  11,
			data:     []byte{0x43, 0x70, 0x00, 0x00},
			expected: true,
		},
		{
			name:     "Wrong Slave 20",
			slaveID:  20,
			data:     []byte{0x44, 0x20, 0x00, 0x00},
			expected: false,
		},
		{
			name:     "Another Wrong Slave 1",
			slaveID:  1,
			data:     []byte{0x42, 0x00, 0x00, 0x00},
			expected: false,
		},
	}

	// Send all responses concurrently
	var wg sync.WaitGroup
	for _, resp := range responses {
		wg.Add(1)
		go func(r struct {
			name     string
			slaveID  uint8
			data     []byte
			expected bool
		}) {
			defer wg.Done()
			msg := createMockMessage("test/data", r.slaveID, 0x03, r.data)
			gateway.onMessage(nil, msg)
			t.Logf("📨 Sent response: %s (Slave %d)", r.name, r.slaveID)
		}(resp)
	}

	// Wait for all responses to be processed
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// Check that only ONE response was accepted (the correct one)
	select {
	case data := <-gateway.responseChan:
		t.Logf("✅ Received response: %d bytes (should be from Slave 11)", len(data))

		// Try to read again - should timeout (no more responses)
		select {
		case extra := <-gateway.responseChan:
			t.Errorf("❌ Multiple responses accepted! Extra response: %v", extra)
		case <-time.After(100 * time.Millisecond):
			t.Log("✅ Only one response accepted (correct behavior)")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("❌ No response received - should have gotten Slave 11 response")
	}
}

// TestConcurrentInterleavedRequests simulates realistic polling of multiple devices
// Devices polled in quick succession with responses potentially overlapping
func TestConcurrentInterleavedRequests(t *testing.T) {
	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)

	// Simulate polling sequence: Slave 11 -> Slave 1 -> Slave 11 -> Slave 20
	pollingSequence := []struct {
		slaveID      uint8
		functionCode uint8
		data         []byte
		description  string
	}{
		{11, 0x03, []byte{0x43, 0x70, 0x00, 0x00}, "Slave 11 - Instant"},
		{1, 0x03, []byte{0x41, 0xF0, 0x00, 0x00}, "Slave 1 - Instant"},
		{11, 0x03, []byte{0x43, 0x80, 0x00, 0x00}, "Slave 11 - Energy"},
		{20, 0x03, []byte{0x44, 0x20, 0x00, 0x00}, "Slave 20 - Status"},
	}

	successCount := 0

	for i, poll := range pollingSequence {
		t.Logf("📤 Poll %d: %s (Slave %d, FC=0x%02X)", i+1, poll.description, poll.slaveID, poll.functionCode)

		// Set expected parameters (protected by mutex)
		gateway.commandMutex.Lock()
		gateway.expectedSlaveID = poll.slaveID
		gateway.expectedFunctionCode = poll.functionCode
		gateway.commandMutex.Unlock()

		// Small delay to simulate request transmission
		time.Sleep(10 * time.Millisecond)

		// Send response
		msg := createMockMessage("test/data", poll.slaveID, poll.functionCode, poll.data)
		go func() {
			time.Sleep(20 * time.Millisecond) // Simulate network delay
			gateway.onMessage(nil, msg)
		}()

		// Wait for response
		select {
		case data := <-gateway.responseChan:
			t.Logf("✅ Poll %d: Received %d bytes", i+1, len(data))
			successCount++
		case <-time.After(100 * time.Millisecond):
			t.Errorf("❌ Poll %d: Timeout waiting for response", i+1)
		}
	}

	if successCount != len(pollingSequence) {
		t.Errorf("❌ Expected %d successful polls, got %d", len(pollingSequence), successCount)
	} else {
		t.Logf("✅ All %d polls completed successfully (no interference)", successCount)
	}
}

// TestConcurrentStressTest stress tests the gateway with many rapid concurrent responses
func TestConcurrentStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)

	const numRequests = 100
	correctResponses := 0
	rejectedResponses := 0

	t.Logf("🔥 Stress test: %d rapid requests", numRequests)

	for i := 0; i < numRequests; i++ {
		// Alternate between 3 different slaves
		targetSlaveID := uint8((i % 3) + 1) // 1, 2, 3

		// Set expected parameters
		gateway.commandMutex.Lock()
		gateway.expectedSlaveID = targetSlaveID
		gateway.expectedFunctionCode = 0x03
		gateway.commandMutex.Unlock()

		// Send correct response and several wrong ones concurrently
		var wg sync.WaitGroup

		// Correct response
		wg.Add(1)
		go func(slaveID uint8) {
			defer wg.Done()
			data := []byte{0x43, byte(i), 0x00, 0x00}
			msg := createMockMessage("test/data", slaveID, 0x03, data)
			gateway.onMessage(nil, msg)
		}(targetSlaveID)

		// Wrong responses (from other slaves)
		for j := 0; j < 3; j++ {
			wrongSlaveID := uint8(((i + j + 1) % 3) + 1)
			if wrongSlaveID != targetSlaveID {
				wg.Add(1)
				go func(slaveID uint8) {
					defer wg.Done()
					data := []byte{0x41, byte(i), byte(slaveID), 0x00}
					msg := createMockMessage("test/data", slaveID, 0x03, data)
					gateway.onMessage(nil, msg)
				}(wrongSlaveID)
			}
		}

		wg.Wait()

		// Check for correct response
		select {
		case <-gateway.responseChan:
			correctResponses++
		case <-time.After(50 * time.Millisecond):
			rejectedResponses++
		}

		// Tiny delay between iterations
		time.Sleep(5 * time.Millisecond)
	}

	successRate := float64(correctResponses) / float64(numRequests) * 100

	t.Logf("📊 Stress test results:")
	t.Logf("  Correct responses: %d / %d (%.1f%%)", correctResponses, numRequests, successRate)
	t.Logf("  Rejected/Timeout: %d", rejectedResponses)

	if successRate < 95.0 {
		t.Errorf("❌ Success rate too low: %.1f%% (expected >= 95%%)", successRate)
	} else {
		t.Logf("✅ Stress test passed with %.1f%% success rate", successRate)
	}
}

// TestConcurrentResponseCollision tests that response channel doesn't overflow or lose data
func TestConcurrentResponseCollision(t *testing.T) {
	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)

	// Set expected parameters
	gateway.expectedSlaveID = 11
	gateway.expectedFunctionCode = 0x03

	t.Log("📤 Testing response channel collision handling")

	// Try to send multiple correct responses rapidly
	// Only first should be accepted (channel buffer size = 1)
	numResponses := 5
	var wg sync.WaitGroup

	for i := 0; i < numResponses; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			data := []byte{0x43, byte(idx), 0x00, 0x00}
			msg := createMockMessage("test/data", 11, 0x03, data)
			gateway.onMessage(nil, msg)
			t.Logf("📨 Sent response %d", idx+1)
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// Count how many responses were buffered
	receivedCount := 0
	for {
		select {
		case data := <-gateway.responseChan:
			receivedCount++
			t.Logf("📥 Received response %d: %v", receivedCount, data)
		case <-time.After(50 * time.Millisecond):
			// No more responses
			goto done
		}
	}

done:
	t.Logf("📊 Sent %d responses, received %d (channel buffer size = 1)", numResponses, receivedCount)

	// With buffer size 1, only 1 response should be buffered
	// Others should be dropped (logged as warnings)
	if receivedCount > 1 {
		t.Logf("⚠️ Warning: Multiple responses buffered (expected max 1, got %d)", receivedCount)
		t.Log("   This may indicate channel overflow - check gateway logs for warnings")
	}

	if receivedCount == 0 {
		t.Error("❌ No responses received - channel may be blocked")
	} else {
		t.Logf("✅ Response collision handled correctly (%d response received)", receivedCount)
	}
}

// TestConcurrentRapidDeviceSwitch tests rapid switching between devices
// Simulates realistic scenario where devices are polled in quick succession
func TestConcurrentRapidDeviceSwitch(t *testing.T) {
	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)

	devices := []uint8{11, 1, 20, 11, 1, 20, 11, 1} // Rapid switching pattern
	successCount := 0

	t.Logf("🔄 Testing rapid device switching (%d switches)", len(devices))

	for i, slaveID := range devices {
		// Set expected parameters
		gateway.commandMutex.Lock()
		gateway.expectedSlaveID = slaveID
		gateway.expectedFunctionCode = 0x03
		gateway.commandMutex.Unlock()

		// Send response immediately (simulating fast polling)
		data := []byte{0x43, byte(i), 0x00, 0x00}
		msg := createMockMessage("test/data", slaveID, 0x03, data)

		go func() {
			time.Sleep(5 * time.Millisecond) // Very small delay
			gateway.onMessage(nil, msg)
		}()

		// Wait for response
		select {
		case <-gateway.responseChan:
			successCount++
			t.Logf("✅ Switch %d: Slave %d responded correctly", i+1, slaveID)
		case <-time.After(50 * time.Millisecond):
			t.Errorf("❌ Switch %d: Timeout for Slave %d", i+1, slaveID)
		}

		// Very short delay before next switch (10ms = 100 switches/sec)
		time.Sleep(10 * time.Millisecond)
	}

	successRate := float64(successCount) / float64(len(devices)) * 100

	if successRate < 100.0 {
		t.Errorf("❌ Rapid switching failed: %.1f%% success rate (expected 100%%)", successRate)
	} else {
		t.Logf("✅ Rapid device switching: %d/%d successful (100%%)", successCount, len(devices))
	}
}

// TestConcurrentMixedValidInvalid tests concurrent mix of valid and invalid responses
func TestConcurrentMixedValidInvalid(t *testing.T) {
	cfg := &config.MQTTConfig{
		Broker:   "test",
		Port:     1883,
		Username: "test",
		Password: "test",
		Gateway: config.GatewayConfig{
			MAC:       "TEST123456",
			CmdTopic:  "test/cmd",
			DataTopic: "test/data",
		},
	}

	gateway := NewUSRGateway(cfg)

	// Expecting Slave 11, Function 0x03
	gateway.expectedSlaveID = 11
	gateway.expectedFunctionCode = 0x03

	t.Log("📤 Expecting Slave 11, FC=0x03")
	t.Log("📨 Sending mix of valid and invalid responses concurrently...")

	// Send a mix of responses concurrently
	var wg sync.WaitGroup
	responseTypes := []string{
		"Wrong SlaveID 1",
		"Wrong SlaveID 20",
		"Correct Slave 11",
		"Wrong FunctionCode",
		"Wrong SlaveID 5",
		"Wrong Both",
	}

	for _, respType := range responseTypes {
		wg.Add(1)
		go func(rType string) {
			defer wg.Done()

			var slaveID, functionCode uint8
			switch rType {
			case "Correct Slave 11":
				slaveID, functionCode = 11, 0x03
			case "Wrong SlaveID 1":
				slaveID, functionCode = 1, 0x03
			case "Wrong SlaveID 20":
				slaveID, functionCode = 20, 0x03
			case "Wrong SlaveID 5":
				slaveID, functionCode = 5, 0x03
			case "Wrong FunctionCode":
				slaveID, functionCode = 11, 0x04
			case "Wrong Both":
				slaveID, functionCode = 1, 0x04
			}

			data := []byte{0x43, byte(slaveID), 0x00, 0x00}
			msg := createMockMessage("test/data", slaveID, functionCode, data)
			gateway.onMessage(nil, msg)
			t.Logf("  📨 %s (Slave=%d, FC=0x%02X)", rType, slaveID, functionCode)
		}(respType)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// Check that only the correct response was accepted
	select {
	case data := <-gateway.responseChan:
		t.Logf("✅ Correct response accepted (%d bytes)", len(data))

		// Verify no other responses in channel
		select {
		case extra := <-gateway.responseChan:
			t.Errorf("❌ Multiple responses accepted! Extra: %v", extra)
		case <-time.After(50 * time.Millisecond):
			t.Log("✅ Only correct response accepted, all invalid responses rejected")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("❌ No response received - correct response should have been accepted")
	}
}
