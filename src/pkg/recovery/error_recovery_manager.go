package recovery

import (
	"time"
)

// ErrorRecoveryManager manages error tracking and recovery logic for gateway operations
// Extracted from Application to follow Single Responsibility Principle
type ErrorRecoveryManager struct {
	consecutiveErrors  int
	firstErrorTime     time.Time
	errorGracePeriod   time.Duration
	statusSetToOffline bool
}

// NewErrorRecoveryManager creates a new error recovery manager
func NewErrorRecoveryManager(gracePeriod time.Duration) *ErrorRecoveryManager {
	if gracePeriod == 0 {
		gracePeriod = 15 * time.Second // Default grace period
	}
	
	return &ErrorRecoveryManager{
		consecutiveErrors:  0,
		firstErrorTime:     time.Time{},
		errorGracePeriod:   gracePeriod,
		statusSetToOffline: false,
	}
}

// RecordError records an error occurrence and returns whether grace period has expired
func (m *ErrorRecoveryManager) RecordError() bool {
	m.consecutiveErrors++
	
	// Record first error time if this is the start of an error sequence
	if m.firstErrorTime.IsZero() {
		m.firstErrorTime = time.Now()
	}
	
	// Check if grace period has expired
	return time.Since(m.firstErrorTime) >= m.errorGracePeriod
}

// RecordSuccess resets error tracking after a successful operation
func (m *ErrorRecoveryManager) RecordSuccess() {
	m.consecutiveErrors = 0
	m.firstErrorTime = time.Time{}
	m.statusSetToOffline = false
}

// GetConsecutiveErrors returns the current count of consecutive errors
func (m *ErrorRecoveryManager) GetConsecutiveErrors() int {
	return m.consecutiveErrors
}

// ShouldMarkOffline returns true if we should mark the gateway as offline
// This considers both the error count and grace period
func (m *ErrorRecoveryManager) ShouldMarkOffline() bool {
	if m.statusSetToOffline {
		return false // Already marked offline, don't repeat
	}
	
	// Mark offline if grace period has expired
	if !m.firstErrorTime.IsZero() && time.Since(m.firstErrorTime) >= m.errorGracePeriod {
		return true
	}
	
	return false
}

// MarkAsOffline sets the flag indicating we've already marked the status as offline
// This prevents repeated offline status publications
func (m *ErrorRecoveryManager) MarkAsOffline() {
	m.statusSetToOffline = true
}

// IsInGracePeriod returns true if we're currently in the grace period after first error
func (m *ErrorRecoveryManager) IsInGracePeriod() bool {
	if m.firstErrorTime.IsZero() {
		return false
	}
	return time.Since(m.firstErrorTime) < m.errorGracePeriod
}

// GetTimeSinceFirstError returns the duration since the first error in current sequence
func (m *ErrorRecoveryManager) GetTimeSinceFirstError() time.Duration {
	if m.firstErrorTime.IsZero() {
		return 0
	}
	return time.Since(m.firstErrorTime)
}

// Reset resets all error tracking state
func (m *ErrorRecoveryManager) Reset() {
	m.consecutiveErrors = 0
	m.firstErrorTime = time.Time{}
	m.statusSetToOffline = false
}
