package gateway

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/logger"
	"mqtt-modbus-bridge/pkg/recovery"
	"time"
)

// CircuitBreakerGateway wraps a Gateway with circuit breaker pattern
// Provides fast-fail behavior when gateway is consistently unavailable
type CircuitBreakerGateway struct {
	gateway        Gateway
	circuitBreaker *recovery.CircuitBreaker
	lastLogTime    time.Time // Track last log time to avoid spam
}

// NewCircuitBreakerGateway creates a new gateway with circuit breaker
func NewCircuitBreakerGateway(gw Gateway, config recovery.CircuitBreakerConfig) *CircuitBreakerGateway {
	cb := recovery.NewCircuitBreaker(config)

	logger.LogInfo("🔌 Circuit breaker initialized for gateway (MaxFailures: %d, Timeout: %s)",
		config.MaxFailures, config.Timeout)

	return &CircuitBreakerGateway{
		gateway:        gw,
		circuitBreaker: cb,
		lastLogTime:    time.Now(),
	}
}

// Connect delegates to the underlying gateway
func (cbg *CircuitBreakerGateway) Connect(ctx context.Context) error {
	return cbg.gateway.Connect(ctx)
}

// Disconnect delegates to the underlying gateway
func (cbg *CircuitBreakerGateway) Disconnect() {
	cbg.gateway.Disconnect()
}

// SendCommand delegates to the underlying gateway
func (cbg *CircuitBreakerGateway) SendCommand(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16) error {
	return cbg.gateway.SendCommand(ctx, slaveID, functionCode, address, count)
} // WaitForResponse delegates to the underlying gateway
func (cbg *CircuitBreakerGateway) WaitForResponse(ctx context.Context, timeout int) ([]byte, error) {
	return cbg.gateway.WaitForResponse(ctx, timeout)
}

// SendCommandAndWaitForResponse wraps the gateway call with circuit breaker
func (cbg *CircuitBreakerGateway) SendCommandAndWaitForResponse(
	ctx context.Context,
	slaveID uint8,
	functionCode uint8,
	address uint16,
	count uint16,
	timeoutSeconds int,
) ([]byte, error) {
	var result []byte
	var callErr error

	// Execute through circuit breaker
	err := cbg.circuitBreaker.Call(func() error {
		result, callErr = cbg.gateway.SendCommandAndWaitForResponse(
			ctx, slaveID, functionCode, address, count, timeoutSeconds,
		)
		return callErr
	})

	// Log state changes (avoid spam by logging at most once per minute)
	cbg.logStateIfChanged()

	if err != nil {
		// Circuit breaker rejected the call or the call failed
		return nil, err
	}

	return result, nil
}

// SendDiagnosticCommand delegates to the underlying gateway
func (cbg *CircuitBreakerGateway) SendDiagnosticCommand(ctx context.Context) error {
	return cbg.gateway.SendDiagnosticCommand(ctx)
}

// IsConnected delegates to the underlying gateway
func (cbg *CircuitBreakerGateway) IsConnected() bool {
	return cbg.gateway.IsConnected()
}

// GetCircuitBreakerStats returns current circuit breaker statistics
func (cbg *CircuitBreakerGateway) GetCircuitBreakerStats() recovery.CircuitBreakerStats {
	return cbg.circuitBreaker.GetStats()
}

// ResetCircuitBreaker manually resets the circuit breaker
func (cbg *CircuitBreakerGateway) ResetCircuitBreaker() {
	logger.LogInfo("🔄 Manually resetting circuit breaker")
	cbg.circuitBreaker.Reset()
}

// logStateIfChanged logs circuit breaker state changes (rate-limited)
func (cbg *CircuitBreakerGateway) logStateIfChanged() {
	state := cbg.circuitBreaker.GetState()

	// Log state changes or once per minute
	if time.Since(cbg.lastLogTime) > time.Minute {
		switch state {
		case recovery.StateClosed:
			logger.LogDebug("🟢 Circuit breaker: CLOSED (normal operation)")
		case recovery.StateOpen:
			stats := cbg.circuitBreaker.GetStats()
			logger.LogWarn("🔴 Circuit breaker: OPEN (failures: %d, fast-failing requests)", stats.Failures)
		case recovery.StateHalfOpen:
			logger.LogInfo("🟡 Circuit breaker: HALF-OPEN (testing recovery)")
		}
		cbg.lastLogTime = time.Now()
	}
}

// GetState returns the current circuit breaker state (for monitoring)
func (cbg *CircuitBreakerGateway) GetState() recovery.CircuitState {
	return cbg.circuitBreaker.GetState()
}

// String provides a string representation for debugging
func (cbg *CircuitBreakerGateway) String() string {
	stats := cbg.circuitBreaker.GetStats()
	return fmt.Sprintf("CircuitBreakerGateway{%s}", stats.String())
}
