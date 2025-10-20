package gateway

import (
	"context"
	"errors"
	"mqtt-modbus-bridge/pkg/recovery"
	"testing"
	"time"
)

// MockGateway is a mock implementation of Gateway for testing
type MockGateway struct {
	failCount    int
	shouldFail   bool
	callCount    int
	connected    bool
	lastSlaveID  uint8
	lastFuncCode uint8
	lastAddress  uint16
	lastCount    uint16
}

func NewMockGateway() *MockGateway {
	return &MockGateway{
		connected: true,
	}
}

func (m *MockGateway) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *MockGateway) Disconnect() {
	m.connected = false
}

func (m *MockGateway) SendCommand(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16) error {
	return nil
}

func (m *MockGateway) WaitForResponse(ctx context.Context, timeout int) ([]byte, error) {
	return []byte{}, nil
}

func (m *MockGateway) SendCommandAndWaitForResponse(
	ctx context.Context,
	slaveID uint8,
	functionCode uint8,
	address uint16,
	count uint16,
	timeoutSeconds int,
) ([]byte, error) {
	m.callCount++
	m.lastSlaveID = slaveID
	m.lastFuncCode = functionCode
	m.lastAddress = address
	m.lastCount = count

	if m.shouldFail {
		m.failCount++
		return nil, errors.New("mock gateway error")
	}

	return []byte{0x01, 0x02, 0x03, 0x04}, nil
}

func (m *MockGateway) SendDiagnosticCommand(ctx context.Context) error {
	return nil
}

func (m *MockGateway) IsConnected() bool {
	return m.connected
}

// TestCircuitBreakerNormalOperation tests normal circuit breaker behavior
func TestCircuitBreakerNormalOperation(t *testing.T) {
	mock := NewMockGateway()
	config := recovery.CircuitBreakerConfig{
		MaxFailures:      3,
		Timeout:          1 * time.Second,
		HalfOpenMaxTries: 2,
	}
	cbGateway := NewCircuitBreakerGateway(mock, config)

	ctx := context.Background()

	// Should succeed on normal operation
	data, err := cbGateway.SendCommandAndWaitForResponse(ctx, 1, 0x03, 0x2000, 2, 5)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if len(data) != 4 {
		t.Errorf("Expected 4 bytes, got %d", len(data))
	}
	if mock.callCount != 1 {
		t.Errorf("Expected 1 call, got %d", mock.callCount)
	}

	// Verify circuit is closed
	if !cbGateway.circuitBreaker.IsClosed() {
		t.Error("Expected circuit to be closed")
	}
}

// TestCircuitBreakerOpens tests that circuit opens after max failures
func TestCircuitBreakerOpens(t *testing.T) {
	mock := NewMockGateway()
	mock.shouldFail = true
	config := recovery.CircuitBreakerConfig{
		MaxFailures:      3,
		Timeout:          1 * time.Second,
		HalfOpenMaxTries: 2,
	}
	cbGateway := NewCircuitBreakerGateway(mock, config)

	ctx := context.Background()

	// Fail MaxFailures times
	for i := 0; i < 3; i++ {
		_, err := cbGateway.SendCommandAndWaitForResponse(ctx, 1, 0x03, 0x2000, 2, 5)
		if err == nil {
			t.Errorf("Expected error on failure %d", i+1)
		}
	}

	if mock.callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", mock.callCount)
	}

	// Circuit should be open now
	if !cbGateway.circuitBreaker.IsOpen() {
		t.Error("Expected circuit to be open after max failures")
	}

	// Next call should be rejected without calling the mock
	beforeCount := mock.callCount
	_, err := cbGateway.SendCommandAndWaitForResponse(ctx, 1, 0x03, 0x2000, 2, 5)
	if err == nil {
		t.Error("Expected error when circuit is open")
	}
	if mock.callCount != beforeCount {
		t.Error("Expected no calls to mock when circuit is open")
	}
}

// TestCircuitBreakerRecovery tests circuit recovery after timeout
func TestCircuitBreakerRecovery(t *testing.T) {
	mock := NewMockGateway()
	mock.shouldFail = true
	config := recovery.CircuitBreakerConfig{
		MaxFailures:      2,
		Timeout:          100 * time.Millisecond, // Short timeout for test
		HalfOpenMaxTries: 2,
	}
	cbGateway := NewCircuitBreakerGateway(mock, config)

	ctx := context.Background()

	// Fail to open circuit
	for i := 0; i < 2; i++ {
		_, _ = cbGateway.SendCommandAndWaitForResponse(ctx, 1, 0x03, 0x2000, 2, 5)
	}

	if !cbGateway.circuitBreaker.IsOpen() {
		t.Error("Expected circuit to be open")
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open and allow test request
	mock.shouldFail = false // Fix the mock

	// Need HalfOpenMaxTries (2) successful calls to fully close circuit
	// The circuit closes when halfOpenAttempts >= halfOpenMaxTries (after 2nd success)
	for i := 0; i < 3; i++ {
		_, err := cbGateway.SendCommandAndWaitForResponse(ctx, 1, 0x03, 0x2000, 2, 5)
		if err != nil {
			t.Errorf("Expected success call %d, got: %v", i+1, err)
		}
	}

	// Circuit should be closed after successful recovery
	if !cbGateway.circuitBreaker.IsClosed() {
		state := cbGateway.GetState()
		stats := cbGateway.GetCircuitBreakerStats()
		t.Errorf("Expected circuit to be closed after successful recovery, got state: %s, halfOpenAttempts: %d",
			state, stats.HalfOpenAttempts)
	}
}

// TestCircuitBreakerStats tests statistics retrieval
func TestCircuitBreakerStats(t *testing.T) {
	mock := NewMockGateway()
	config := recovery.CircuitBreakerConfig{
		MaxFailures:      3,
		Timeout:          1 * time.Second,
		HalfOpenMaxTries: 2,
	}
	cbGateway := NewCircuitBreakerGateway(mock, config)

	stats := cbGateway.GetCircuitBreakerStats()
	if stats.State != recovery.StateClosed {
		t.Errorf("Expected initial state CLOSED, got %s", stats.State)
	}
	if stats.Failures != 0 {
		t.Errorf("Expected 0 failures, got %d", stats.Failures)
	}
}

// TestCircuitBreakerGetState tests state retrieval
func TestCircuitBreakerGetState(t *testing.T) {
	mock := NewMockGateway()
	config := recovery.CircuitBreakerConfig{
		MaxFailures:      2,
		Timeout:          1 * time.Second,
		HalfOpenMaxTries: 2,
	}
	cbGateway := NewCircuitBreakerGateway(mock, config)

	// Initial state should be closed
	if cbGateway.GetState() != recovery.StateClosed {
		t.Error("Expected initial state to be CLOSED")
	}

	// Fail to open circuit
	ctx := context.Background()
	mock.shouldFail = true
	for i := 0; i < 2; i++ {
		_, _ = cbGateway.SendCommandAndWaitForResponse(ctx, 1, 0x03, 0x2000, 2, 5)
	}

	// State should be open
	if cbGateway.GetState() != recovery.StateOpen {
		t.Error("Expected state to be OPEN after failures")
	}
}
