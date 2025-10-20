package metrics

import (
	"sync"
	"time"

	"mqtt-modbus-bridge/pkg/logger"
)

// PerformanceTracker tracks operation performance metrics
type PerformanceTracker struct {
	successfulReads int
	errorReads      int
	lastSummaryTime time.Time
	summaryInterval time.Duration
	mu              sync.RWMutex
}

// PerformanceStats represents performance statistics
type PerformanceStats struct {
	SuccessfulReads int
	ErrorReads      int
	LastSummary     time.Time
	SuccessRate     float64
	ErrorRate       float64
}

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker(summaryInterval time.Duration) *PerformanceTracker {
	return &PerformanceTracker{
		successfulReads: 0,
		errorReads:      0,
		lastSummaryTime: time.Now(),
		summaryInterval: summaryInterval,
	}
}

// RecordSuccess records a successful operation
func (pt *PerformanceTracker) RecordSuccess() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.successfulReads++
}

// RecordSuccessBatch records multiple successful operations
func (pt *PerformanceTracker) RecordSuccessBatch(count int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.successfulReads += count
}

// RecordError records a failed operation
func (pt *PerformanceTracker) RecordError() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.errorReads++
}

// GetStats returns current performance statistics
func (pt *PerformanceTracker) GetStats() PerformanceStats {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	total := pt.successfulReads + pt.errorReads
	var successRate, errorRate float64

	if total > 0 {
		successRate = (float64(pt.successfulReads) / float64(total)) * 100.0
		errorRate = (float64(pt.errorReads) / float64(total)) * 100.0
	}

	return PerformanceStats{
		SuccessfulReads: pt.successfulReads,
		ErrorReads:      pt.errorReads,
		LastSummary:     pt.lastSummaryTime,
		SuccessRate:     successRate,
		ErrorRate:       errorRate,
	}
}

// ShouldPrintSummary checks if enough time has passed to print summary
func (pt *PerformanceTracker) ShouldPrintSummary() bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return time.Since(pt.lastSummaryTime) >= pt.summaryInterval
}

// PrintSummary prints performance summary and resets counters
func (pt *PerformanceTracker) PrintSummary() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if time.Since(pt.lastSummaryTime) < pt.summaryInterval {
		return
	}

	logger.LogInfo("📊 Summary - Success: %d, Errors: %d, Last %v",
		pt.successfulReads,
		pt.errorReads,
		pt.summaryInterval,
	)

	pt.lastSummaryTime = time.Now()
	pt.successfulReads = 0
	pt.errorReads = 0
}

// PrintSummaryIfNeeded prints summary only if interval has passed
func (pt *PerformanceTracker) PrintSummaryIfNeeded() {
	if pt.ShouldPrintSummary() {
		pt.PrintSummary()
	}
}

// Reset resets all counters and timers
func (pt *PerformanceTracker) Reset() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.successfulReads = 0
	pt.errorReads = 0
	pt.lastSummaryTime = time.Now()
}

// GetLastSummaryTime returns when the last summary was printed
func (pt *PerformanceTracker) GetLastSummaryTime() time.Time {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.lastSummaryTime
}

// GetSuccessCount returns the current success count
func (pt *PerformanceTracker) GetSuccessCount() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.successfulReads
}

// GetErrorCount returns the current error count
func (pt *PerformanceTracker) GetErrorCount() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.errorReads
}

// GetTotalCount returns the total number of operations
func (pt *PerformanceTracker) GetTotalCount() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.successfulReads + pt.errorReads
}
