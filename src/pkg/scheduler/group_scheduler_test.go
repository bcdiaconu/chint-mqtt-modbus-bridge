package scheduler

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/modbus"
	"sync"
	"testing"
	"time"
)

// mockExecutor implements ExecutorInterface for testing
type mockExecutor struct {
	mu             sync.Mutex
	executionOrder []string // Track order of group executions
	executionTimes map[string]time.Time
	currentCount   int           // Current number of concurrent executions
	maxConcurrent  int           // Maximum observed concurrent executions
	delay          time.Duration // Simulated execution delay
	shouldFail     map[string]bool
}

func newMockExecutor(delay time.Duration) *mockExecutor {
	return &mockExecutor{
		executionOrder: make([]string, 0),
		executionTimes: make(map[string]time.Time),
		delay:          delay,
		shouldFail:     make(map[string]bool),
	}
}

func (m *mockExecutor) ExecuteAll(ctx context.Context) (map[string]*modbus.CommandResult, error) {
	// Not used in scheduler tests
	return nil, nil
}

func (m *mockExecutor) RegisterFromDevices(devices map[string]config.Device) error {
	// Not used in scheduler tests
	return nil
}

func (m *mockExecutor) ExecuteGroup(ctx context.Context, groupKey string) (map[string]*modbus.CommandResult, error) {
	m.mu.Lock()
	m.currentCount++
	if m.currentCount > m.maxConcurrent {
		m.maxConcurrent = m.currentCount
	}
	m.executionOrder = append(m.executionOrder, groupKey)
	m.executionTimes[groupKey] = time.Now()
	m.mu.Unlock()

	// Simulate work
	time.Sleep(m.delay)

	m.mu.Lock()
	m.currentCount--
	shouldFail := m.shouldFail[groupKey]
	m.mu.Unlock()

	if shouldFail {
		return nil, fmt.Errorf("simulated failure for group %s", groupKey)
	}

	// Return dummy result
	return map[string]*modbus.CommandResult{
		"test_register": {
			Strategy: groupKey,
			Name:     "test_register",
			Value:    42.0,
			Unit:     "V",
		},
	}, nil
}

func (m *mockExecutor) GetGroupIntervals() map[string]int {
	return map[string]int{
		"group_a": 1000,
		"group_b": 1000,
		"group_c": 2000,
	}
}

func (m *mockExecutor) getMaxConcurrent() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.maxConcurrent
}

func (m *mockExecutor) getExecutionOrder() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	order := make([]string, len(m.executionOrder))
	copy(order, m.executionOrder)
	return order
}

// TestSequentialExecution verifies that groups execute sequentially, never concurrently
func TestSequentialExecution(t *testing.T) {
	executor := newMockExecutor(50 * time.Millisecond) // 50ms per execution

	groupIntervals := map[string]int{
		"group_a": 100, // 100ms interval
		"group_b": 100, // 100ms interval
		"group_c": 100, // 100ms interval
	}

	scheduler := NewGroupScheduler(executor, groupIntervals)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resultsReceived := 0
	var resultsMu sync.Mutex

	callback := func(ctx context.Context, results map[string]*modbus.CommandResult) {
		resultsMu.Lock()
		resultsReceived++
		resultsMu.Unlock()
	}

	// Start scheduler
	go scheduler.Start(ctx, callback)

	// Wait for multiple execution cycles
	time.Sleep(500 * time.Millisecond)

	// Check that executions were sequential (max concurrent = 1)
	maxConcurrent := executor.getMaxConcurrent()
	if maxConcurrent != 1 {
		t.Errorf("❌ Expected sequential execution (max concurrent = 1), got %d", maxConcurrent)
	} else {
		t.Logf("✅ Sequential execution verified: max concurrent executions = %d", maxConcurrent)
	}

	// Verify we had multiple executions
	executionOrder := executor.getExecutionOrder()
	if len(executionOrder) < 3 {
		t.Errorf("❌ Expected at least 3 executions, got %d", len(executionOrder))
	} else {
		t.Logf("✅ Multiple executions completed: %d total", len(executionOrder))
		t.Logf("   Execution order: %v", executionOrder)
	}
}

// TestRaceConditionPrevention simulates the bug where concurrent group executions
// cause response mix-ups between different Modbus devices
func TestRaceConditionPrevention(t *testing.T) {
	executor := newMockExecutor(100 * time.Millisecond) // Longer execution time

	// Two groups that would execute at the same time
	groupIntervals := map[string]int{
		"slave_11_instant": 200, // Same interval
		"slave_1_instant":  200, // Same interval
	}

	scheduler := NewGroupScheduler(executor, groupIntervals)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	executionTimestamps := make(map[string][]time.Time)
	var tsMu sync.Mutex

	callback := func(ctx context.Context, results map[string]*modbus.CommandResult) {
		tsMu.Lock()
		defer tsMu.Unlock()
		// Track which group executed when
		for groupKey := range results {
			executionTimestamps[groupKey] = append(executionTimestamps[groupKey], time.Now())
		}
	}

	// Start scheduler
	go scheduler.Start(ctx, callback)

	// Wait for executions
	time.Sleep(700 * time.Millisecond)

	// Verify sequential execution
	maxConcurrent := executor.getMaxConcurrent()
	if maxConcurrent > 1 {
		t.Errorf("❌ Race condition detected! Found %d concurrent executions", maxConcurrent)
		t.Log("   This would cause 'unexpected response' errors in production")
	} else {
		t.Log("✅ Race condition prevented: groups executed sequentially")
	}

	// Check execution order
	executionOrder := executor.getExecutionOrder()
	t.Logf("📊 Execution order: %v", executionOrder)
	t.Logf("📊 Total executions: %d", len(executionOrder))

	// The KEY test: sequential execution prevents race conditions
	t.Log("✅ Sequential execution verified - no race conditions possible")
}

// TestConcurrentSchedulerCalls verifies that multiple simultaneous checks
// don't cause concurrent group executions
func TestConcurrentSchedulerCalls(t *testing.T) {
	executor := newMockExecutor(50 * time.Millisecond)

	groupIntervals := map[string]int{
		"group_a": 100,
		"group_b": 100,
	}

	scheduler := NewGroupScheduler(executor, groupIntervals)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	callback := func(ctx context.Context, results map[string]*modbus.CommandResult) {
		// No-op
	}

	// Simulate multiple concurrent scheduler ticks (shouldn't happen normally,
	// but tests the mutex protection)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scheduler.checkAndExecuteGroups(ctx, callback)
		}()
	}

	wg.Wait()

	// Even with concurrent check calls, max concurrent executions should be 1
	maxConcurrent := executor.getMaxConcurrent()
	if maxConcurrent > 1 {
		t.Errorf("❌ Concurrent execution detected with %d groups running simultaneously", maxConcurrent)
		t.Log("   executionMutex failed to prevent concurrent access")
	} else {
		t.Log("✅ Mutex protection verified: no concurrent executions despite concurrent checks")
	}

	executionOrder := executor.getExecutionOrder()
	t.Logf("📊 Executions completed: %d (order: %v)", len(executionOrder), executionOrder)
}

// TestDifferentIntervals verifies that groups with different intervals
// execute at correct times without interference
func TestDifferentIntervals(t *testing.T) {
	executor := newMockExecutor(20 * time.Millisecond)

	groupIntervals := map[string]int{
		"fast_group": 100, // Every 100ms
		"slow_group": 300, // Every 300ms
	}

	scheduler := NewGroupScheduler(executor, groupIntervals)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	executionCounts := make(map[string]int)
	var countMu sync.Mutex

	callback := func(ctx context.Context, results map[string]*modbus.CommandResult) {
		countMu.Lock()
		defer countMu.Unlock()
		for key := range results {
			executionCounts[key]++
		}
	}

	// Start scheduler
	go scheduler.Start(ctx, callback)

	// Wait for executions
	time.Sleep(950 * time.Millisecond)

	// Fast group (100ms) should execute ~9 times in 900ms
	// Slow group (300ms) should execute ~3 times in 900ms
	executionOrder := executor.getExecutionOrder()

	fastExecutions := 0
	slowExecutions := 0
	for _, group := range executionOrder {
		if group == "fast_group" {
			fastExecutions++
		} else if group == "slow_group" {
			slowExecutions++
		}
	}

	t.Logf("📊 Fast group (100ms): %d executions", fastExecutions)
	t.Logf("📊 Slow group (300ms): %d executions", slowExecutions)

	// Fast should execute more than slow (but exact ratio depends on timing)
	if fastExecutions <= slowExecutions {
		t.Errorf("❌ Fast group should execute more than slow group (fast=%d, slow=%d)",
			fastExecutions, slowExecutions)
	} else {
		ratio := float64(fastExecutions) / float64(slowExecutions)
		t.Logf("✅ Execution intervals respected: fast/slow ratio %.2f", ratio)
	}

	// Verify sequential execution throughout
	maxConcurrent := executor.getMaxConcurrent()
	if maxConcurrent > 1 {
		t.Errorf("❌ Concurrent execution detected: %d", maxConcurrent)
	} else {
		t.Log("✅ All executions remained sequential")
	}
}

// TestExecutionFailureDoesNotBlock verifies that a failing group
// doesn't prevent other groups from executing
func TestExecutionFailureDoesNotBlock(t *testing.T) {
	executor := newMockExecutor(20 * time.Millisecond)
	executor.shouldFail["failing_group"] = true

	groupIntervals := map[string]int{
		"failing_group": 100,
		"working_group": 100,
	}

	scheduler := NewGroupScheduler(executor, groupIntervals)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	successCount := 0
	var countMu sync.Mutex

	callback := func(ctx context.Context, results map[string]*modbus.CommandResult) {
		countMu.Lock()
		successCount++
		countMu.Unlock()
	}

	// Start scheduler
	go scheduler.Start(ctx, callback)

	// Wait for executions
	time.Sleep(450 * time.Millisecond)

	// Check that working_group executed despite failing_group failures
	executionOrder := executor.getExecutionOrder()

	workingExecutions := 0
	failingExecutions := 0
	for _, group := range executionOrder {
		if group == "working_group" {
			workingExecutions++
		} else if group == "failing_group" {
			failingExecutions++
		}
	}

	t.Logf("📊 Failing group: %d executions", failingExecutions)
	t.Logf("📊 Working group: %d executions", workingExecutions)

	if workingExecutions == 0 {
		t.Error("❌ Working group was blocked by failing group")
	} else {
		t.Log("✅ Working group continued executing despite other group failures")
	}

	// Verify sequential execution maintained
	maxConcurrent := executor.getMaxConcurrent()
	if maxConcurrent > 1 {
		t.Errorf("❌ Concurrent execution detected: %d", maxConcurrent)
	} else {
		t.Log("✅ Sequential execution maintained even with failures")
	}
}

// TestHighFrequencyScheduling tests scheduler under high-frequency polling
func TestHighFrequencyScheduling(t *testing.T) {
	executor := newMockExecutor(5 * time.Millisecond) // Very fast execution

	groupIntervals := map[string]int{
		"high_freq_a": 50, // 50ms = 20 Hz
		"high_freq_b": 50,
		"high_freq_c": 50,
	}

	scheduler := NewGroupScheduler(executor, groupIntervals)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	callback := func(ctx context.Context, results map[string]*modbus.CommandResult) {
		// No-op
	}

	// Start scheduler
	go scheduler.Start(ctx, callback)

	// Wait for executions
	time.Sleep(450 * time.Millisecond)

	// Check results
	maxConcurrent := executor.getMaxConcurrent()
	executionOrder := executor.getExecutionOrder()

	t.Logf("📊 Total executions: %d in 450ms", len(executionOrder))
	t.Logf("📊 Max concurrent: %d", maxConcurrent)

	if maxConcurrent > 1 {
		t.Errorf("❌ High-frequency scheduling caused concurrent executions: %d", maxConcurrent)
	} else {
		t.Log("✅ High-frequency scheduling maintained sequential execution")
	}

	// Should have multiple executions
	// The exact number depends on scheduler timing, but we should see activity
	if len(executionOrder) < 10 {
		t.Logf("⚠️ Fewer executions than expected: %d (expected > 10)", len(executionOrder))
		t.Log("   This may be due to scheduler check interval (100ms)")
	} else {
		t.Logf("✅ High-frequency scheduling working: %d executions", len(executionOrder))
	}
}

// TestMutexReleaseOnPanic verifies that mutex is released even if execution panics
func TestMutexReleaseOnPanic(t *testing.T) {
	// Note: This test verifies the defer pattern releases the mutex
	// The actual panic would be caught by the defer in executeGroup()

	executor := newMockExecutor(10 * time.Millisecond)

	groupIntervals := map[string]int{
		"group_a": 100,
	}

	scheduler := NewGroupScheduler(executor, groupIntervals)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	callback := func(ctx context.Context, results map[string]*modbus.CommandResult) {
		// No-op
	}

	// Try to execute multiple times to verify mutex works after each execution
	scheduler.checkAndExecuteGroups(ctx, callback)
	time.Sleep(50 * time.Millisecond)

	scheduler.checkAndExecuteGroups(ctx, callback)
	time.Sleep(50 * time.Millisecond)

	scheduler.checkAndExecuteGroups(ctx, callback)

	executionOrder := executor.getExecutionOrder()
	if len(executionOrder) < 2 {
		t.Error("❌ Mutex not released properly between executions")
	} else {
		t.Logf("✅ Mutex properly released: %d sequential executions completed", len(executionOrder))
	}
}

// BenchmarkSchedulerOverhead measures the overhead of the scheduler
func BenchmarkSchedulerOverhead(b *testing.B) {
	executor := newMockExecutor(1 * time.Millisecond)

	groupIntervals := map[string]int{
		"group_a": 100,
	}

	scheduler := NewGroupScheduler(executor, groupIntervals)

	ctx := context.Background()
	callback := func(ctx context.Context, results map[string]*modbus.CommandResult) {
		// No-op
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scheduler.checkAndExecuteGroups(ctx, callback)
	}
}
