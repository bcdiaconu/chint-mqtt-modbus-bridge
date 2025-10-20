package health

import (
	"mqtt-modbus-bridge/pkg/recovery"
	"sync"
	"time"
)

// GatewayHealthMonitor tracks gateway online/offline status and integrates with error recovery
// Extracted from Application to follow Single Responsibility Principle
type GatewayHealthMonitor struct {
	isOnline         bool
	lastErrorTime    time.Time
	errorManager     *recovery.ErrorRecoveryManager
	mu               sync.RWMutex
}

// NewGatewayHealthMonitor creates a new gateway health monitor
func NewGatewayHealthMonitor(gracePeriod time.Duration) *GatewayHealthMonitor {
	return &GatewayHealthMonitor{
		isOnline:      true,
		lastErrorTime: time.Time{},
		errorManager:  recovery.NewErrorRecoveryManager(gracePeriod),
	}
}

// IsOnline returns whether the gateway is currently marked as online
func (m *GatewayHealthMonitor) IsOnline() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isOnline
}

// RecordSuccess records a successful gateway operation
func (m *GatewayHealthMonitor) RecordSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.errorManager.RecordSuccess()
	m.isOnline = true
}

// RecordError records a gateway error and returns whether it should be marked offline
func (m *GatewayHealthMonitor) RecordError() (shouldMarkOffline bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.lastErrorTime = time.Now()
	m.errorManager.RecordError()
	
	return m.errorManager.ShouldMarkOffline()
}

// MarkOffline explicitly marks the gateway as offline
func (m *GatewayHealthMonitor) MarkOffline() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.isOnline = false
	m.errorManager.MarkAsOffline()
}

// MarkOnline explicitly marks the gateway as online
func (m *GatewayHealthMonitor) MarkOnline() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.isOnline = true
	m.errorManager.Reset()
}

// GetConsecutiveErrors returns the current count of consecutive errors
func (m *GatewayHealthMonitor) GetConsecutiveErrors() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorManager.GetConsecutiveErrors()
}

// GetLastErrorTime returns the time of the last error
func (m *GatewayHealthMonitor) GetLastErrorTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastErrorTime
}

// IsInGracePeriod returns true if currently in error grace period
func (m *GatewayHealthMonitor) IsInGracePeriod() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorManager.IsInGracePeriod()
}

// GetTimeSinceFirstError returns duration since first error in current sequence
func (m *GatewayHealthMonitor) GetTimeSinceFirstError() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorManager.GetTimeSinceFirstError()
}
