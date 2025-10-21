package scheduler

import (
	"context"
	"mqtt-modbus-bridge/pkg/builder"
	"mqtt-modbus-bridge/pkg/logger"
	"mqtt-modbus-bridge/pkg/modbus"
	"sync"
	"time"
)

// GroupScheduler manages independent polling for each register group
// Each group can have its own poll_interval
type GroupScheduler struct {
	executor         builder.ExecutorInterface
	groupIntervals   map[string]time.Duration // groupKey -> poll interval
	lastExecutions   map[string]time.Time     // groupKey -> last execution time
	mu               sync.RWMutex             // Protect maps
	executionMutex   sync.Mutex               // Ensures only one group executes at a time (prevents concurrent Modbus requests)
	minCheckInterval time.Duration            // How often to check for groups that need execution
}

// NewGroupScheduler creates a new group scheduler
func NewGroupScheduler(executor builder.ExecutorInterface, groupIntervals map[string]int) *GroupScheduler {
	scheduler := &GroupScheduler{
		executor:       executor,
		groupIntervals: make(map[string]time.Duration),
		lastExecutions: make(map[string]time.Time),
	}

	// Convert intervals from milliseconds to time.Duration
	minInterval := time.Duration(0)
	for groupKey, intervalMs := range groupIntervals {
		interval := time.Duration(intervalMs) * time.Millisecond
		scheduler.groupIntervals[groupKey] = interval

		// Find minimum interval for check frequency
		if minInterval == 0 || interval < minInterval {
			minInterval = interval
		}

		logger.LogInfo("📅 Scheduled group '%s' with interval: %v (%d ms)", groupKey, interval, intervalMs)
	}

	// Check at most every 100ms or at 1/10 of minimum interval
	if minInterval > 0 {
		scheduler.minCheckInterval = minInterval / 10
		if scheduler.minCheckInterval < 100*time.Millisecond {
			scheduler.minCheckInterval = 100 * time.Millisecond
		}
	} else {
		scheduler.minCheckInterval = 100 * time.Millisecond
	}

	logger.LogInfo("📅 Group scheduler initialized with %d groups (check interval: %v)",
		len(groupIntervals), scheduler.minCheckInterval)

	return scheduler
}

// Start begins the group polling scheduler
func (s *GroupScheduler) Start(ctx context.Context, callback func(context.Context, map[string]*modbus.CommandResult)) {
	ticker := time.NewTicker(s.minCheckInterval)
	defer ticker.Stop()

	logger.LogInfo("🔄 Group scheduler started (check interval: %v)", s.minCheckInterval)

	for {
		select {
		case <-ctx.Done():
			logger.LogDebug("🔄 Group scheduler stopped")
			return
		case <-ticker.C:
			s.checkAndExecuteGroups(ctx, callback)
		}
	}
}

// checkAndExecuteGroups checks which groups are due and executes them
func (s *GroupScheduler) checkAndExecuteGroups(ctx context.Context, callback func(context.Context, map[string]*modbus.CommandResult)) {
	now := time.Now()

	s.mu.RLock()
	groupsToExecute := make([]string, 0)

	for groupKey, interval := range s.groupIntervals {
		lastExec, exists := s.lastExecutions[groupKey]

		// Execute if never executed OR if interval has passed
		if !exists || now.Sub(lastExec) >= interval {
			groupsToExecute = append(groupsToExecute, groupKey)
		}
	}
	s.mu.RUnlock()

	// Execute groups that are due (sequentially, one at a time)
	if len(groupsToExecute) > 0 {
		logger.LogTrace("⏰ Groups due for execution: %v", groupsToExecute)

		for _, groupKey := range groupsToExecute {
			s.executeGroup(ctx, groupKey, callback)
		}
	}
}

// executeGroup executes a single register group
func (s *GroupScheduler) executeGroup(ctx context.Context, groupKey string, callback func(context.Context, map[string]*modbus.CommandResult)) {
	// CRITICAL: Lock to ensure only one group executes at a time
	// This prevents concurrent Modbus requests that would cause race conditions
	s.executionMutex.Lock()
	defer s.executionMutex.Unlock()

	startTime := time.Now()

	logger.LogTrace("🔄 Executing group '%s'...", groupKey)

	// Execute the group strategy
	results, err := s.executor.ExecuteGroup(ctx, groupKey)

	executionTime := time.Since(startTime)

	// Update last execution time (even if failed, to avoid retry storms)
	s.mu.Lock()
	s.lastExecutions[groupKey] = startTime
	s.mu.Unlock()

	if err != nil {
		logger.LogError("❌ Group '%s' execution failed after %v: %v", groupKey, executionTime, err)
		return
	}

	if len(results) > 0 {
		logger.LogTrace("✅ Group '%s' executed successfully in %v (%d registers)",
			groupKey, executionTime, len(results))

		// Call callback with results
		if callback != nil {
			callback(ctx, results)
		}
	}
}

// GetNextExecutionTimes returns when each group will execute next (for debugging)
func (s *GroupScheduler) GetNextExecutionTimes() map[string]time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	next := make(map[string]time.Time)
	for groupKey, interval := range s.groupIntervals {
		if lastExec, exists := s.lastExecutions[groupKey]; exists {
			next[groupKey] = lastExec.Add(interval)
		} else {
			next[groupKey] = time.Now() // Will execute immediately
		}
	}
	return next
}
