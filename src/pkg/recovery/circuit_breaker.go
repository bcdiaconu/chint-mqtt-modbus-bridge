package recovery

import (
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	// StateClosed - normal operation, requests pass through
	StateClosed CircuitState = iota
	// StateOpen - failing, requests blocked immediately
	StateOpen
	// StateHalfOpen - testing recovery, limited requests allowed
	StateHalfOpen
)

// String returns the string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker implements the circuit breaker pattern
// Prevents cascading failures by failing fast when a service is unavailable
type CircuitBreaker struct {
	// Configuration
	maxFailures      int           // Number of failures before opening circuit
	timeout          time.Duration // Time to wait before attempting recovery (half-open)
	halfOpenMaxTries int           // Number of test requests allowed in half-open state

	// State
	state            CircuitState
	failures         int
	lastFailureTime  time.Time
	lastStateChange  time.Time
	halfOpenAttempts int

	// Thread safety
	mu sync.RWMutex
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	MaxFailures      int           // Default: 5
	Timeout          time.Duration // Default: 30 seconds
	HalfOpenMaxTries int           // Default: 3
}

// NewCircuitBreaker creates a new circuit breaker with given configuration
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	// Set defaults if not specified
	if config.MaxFailures == 0 {
		config.MaxFailures = 5
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.HalfOpenMaxTries == 0 {
		config.HalfOpenMaxTries = 3
	}

	return &CircuitBreaker{
		maxFailures:      config.MaxFailures,
		timeout:          config.Timeout,
		halfOpenMaxTries: config.HalfOpenMaxTries,
		state:            StateClosed,
		lastStateChange:  time.Now(),
	}
}

// Call executes the given function if the circuit allows it
// Returns error if circuit is open or if the function fails
func (cb *CircuitBreaker) Call(fn func() error) error {
	// Check if we can proceed
	if err := cb.beforeCall(); err != nil {
		return err
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.afterCall(err)

	return err
}

// beforeCall checks if the call should be allowed
func (cb *CircuitBreaker) beforeCall() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Normal operation - allow call
		return nil

	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastFailureTime) > cb.timeout {
			// Transition to half-open state
			cb.state = StateHalfOpen
			cb.halfOpenAttempts = 0
			cb.lastStateChange = time.Now()
			return nil
		}
		// Circuit is open - reject call
		return fmt.Errorf("circuit breaker is OPEN (failed %d times, waiting %.0fs)",
			cb.failures, time.Until(cb.lastFailureTime.Add(cb.timeout)).Seconds())

	case StateHalfOpen:
		// Allow limited number of test requests
		if cb.halfOpenAttempts >= cb.halfOpenMaxTries {
			return fmt.Errorf("circuit breaker is HALF-OPEN (max test attempts reached)")
		}
		cb.halfOpenAttempts++
		return nil

	default:
		return fmt.Errorf("circuit breaker in unknown state")
	}
}

// afterCall records the result of the call
func (cb *CircuitBreaker) afterCall(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onFailure handles a failed call
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		// Check if we should open the circuit
		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
			cb.lastStateChange = time.Now()
		}

	case StateHalfOpen:
		// Failed during testing - reopen circuit
		cb.state = StateOpen
		cb.halfOpenAttempts = 0
		cb.lastStateChange = time.Now()
	}
}

// onSuccess handles a successful call
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		// Reset failure counter on success
		cb.failures = 0

	case StateHalfOpen:
		// Check if we've had enough successful tests
		if cb.halfOpenAttempts >= cb.halfOpenMaxTries {
			// Recovery confirmed - close circuit
			cb.state = StateClosed
			cb.failures = 0
			cb.halfOpenAttempts = 0
			cb.lastStateChange = time.Now()
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// GetLastFailureTime returns the time of the last failure
func (cb *CircuitBreaker) GetLastFailureTime() time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.lastFailureTime
}

// GetTimeSinceLastStateChange returns duration since last state change
func (cb *CircuitBreaker) GetTimeSinceLastStateChange() time.Duration {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return time.Since(cb.lastStateChange)
}

// IsOpen returns true if circuit is open
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == StateOpen
}

// IsClosed returns true if circuit is closed
func (cb *CircuitBreaker) IsClosed() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == StateClosed
}

// IsHalfOpen returns true if circuit is half-open
func (cb *CircuitBreaker) IsHalfOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == StateHalfOpen
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.halfOpenAttempts = 0
	cb.lastStateChange = time.Now()
}

// GetStats returns statistics about the circuit breaker
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:                    cb.state,
		Failures:                 cb.failures,
		LastFailureTime:          cb.lastFailureTime,
		LastStateChange:          cb.lastStateChange,
		HalfOpenAttempts:         cb.halfOpenAttempts,
		TimeSinceLastStateChange: time.Since(cb.lastStateChange),
	}
}

// CircuitBreakerStats holds statistics about the circuit breaker
type CircuitBreakerStats struct {
	State                    CircuitState
	Failures                 int
	LastFailureTime          time.Time
	LastStateChange          time.Time
	HalfOpenAttempts         int
	TimeSinceLastStateChange time.Duration
}

// String returns a string representation of the stats
func (s CircuitBreakerStats) String() string {
	return fmt.Sprintf("State: %s, Failures: %d, Last Failure: %s ago, Last State Change: %s ago",
		s.State,
		s.Failures,
		time.Since(s.LastFailureTime).Round(time.Second),
		s.TimeSinceLastStateChange.Round(time.Second))
}
