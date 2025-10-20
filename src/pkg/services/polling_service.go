package services

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/builder"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/diagnostics"
	"mqtt-modbus-bridge/pkg/health"
	"mqtt-modbus-bridge/pkg/logger"
	"mqtt-modbus-bridge/pkg/modbus"
	"time"
)

// PollingService encapsulates the polling loop logic
// Single Responsibility: Execute strategies and coordinate publishing
type PollingService struct {
	executor          builder.ExecutorInterface
	publisher         builder.PublisherInterface
	healthMonitor     *health.GatewayHealthMonitor
	diagnosticManager *diagnostics.DeviceManager
	config            *config.Config

	// Performance tracking
	successfulReads int
	errorReads      int
	lastSummaryTime time.Time
	lastPublishTime map[string]time.Time
}

// NewPollingService creates a new polling service
func NewPollingService(
	executor builder.ExecutorInterface,
	publisher builder.PublisherInterface,
	healthMonitor *health.GatewayHealthMonitor,
	diagnosticManager *diagnostics.DeviceManager,
	cfg *config.Config,
) *PollingService {
	return &PollingService{
		executor:          executor,
		publisher:         publisher,
		healthMonitor:     healthMonitor,
		diagnosticManager: diagnosticManager,
		config:            cfg,
		lastPublishTime:   make(map[string]time.Time),
		lastSummaryTime:   time.Now(),
	}
}

// Start begins the polling loop
func (s *PollingService) Start(ctx context.Context, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	logger.LogInfo("🔄 Polling service started with interval: %v", pollInterval)

	for {
		select {
		case <-ctx.Done():
			logger.LogDebug("🔄 Polling service stopped")
			return
		case <-ticker.C:
			logger.LogDebug("🔄 Polling tick - executing all strategies...")
			s.ExecuteAndPublish(ctx)
		}
	}
}

// ExecuteAndPublish executes all strategies and publishes results
func (s *PollingService) ExecuteAndPublish(ctx context.Context) {
	// Track start time for response time measurement
	startTime := time.Now()

	// Execute all strategies (groups first, then calculated)
	results, err := s.executor.ExecuteAll(ctx)

	responseTime := time.Since(startTime)

	if err != nil {
		s.handleExecutionError(ctx, err)
		return
	}

	s.handleExecutionSuccess(ctx, results, responseTime)
}

// handleExecutionError handles errors during strategy execution
func (s *PollingService) handleExecutionError(ctx context.Context, err error) {
	s.errorReads++
	s.recordError(ctx)

	logger.LogError("❌ Strategy execution error: %v", err)

	// Update metrics for all devices (error) - if diagnostic manager is enabled
	if s.diagnosticManager != nil {
		for deviceID := range s.config.Devices {
			if s.config.Devices[deviceID].Metadata.Enabled {
				s.diagnosticManager.RecordError(deviceID, fmt.Sprintf("Strategy execution error: %v", err))
			}
		}
	}

	// Publish diagnostic
	errorMsg := fmt.Sprintf("Strategy execution error: %v", err)
	if diagErr := s.publisher.PublishDiagnostic(ctx, 3, errorMsg); diagErr != nil {
		logger.LogError("⚠️ Error publishing diagnostic: %v", diagErr)
	}
}

// handleExecutionSuccess handles successful strategy execution
func (s *PollingService) handleExecutionSuccess(ctx context.Context, results map[string]*modbus.CommandResult, responseTime time.Duration) {
	// Update metrics for all enabled devices (if diagnostic manager is enabled)
	if s.diagnosticManager != nil {
		for deviceID, device := range s.config.Devices {
			if device.Metadata.Enabled {
				s.diagnosticManager.RecordSuccess(deviceID, responseTime)
			}
		}
	}

	// Success - mark gateway as healthy
	s.successfulReads += len(results)
	s.recordSuccess(ctx)

	// Print summary every 30 seconds
	shouldLog := time.Since(s.lastSummaryTime) >= 30*time.Second

	if shouldLog {
		logger.LogInfo("📊 Summary - Success: %d, Errors: %d, Last 30s", s.successfulReads, s.errorReads)
		s.lastSummaryTime = time.Now()
		s.successfulReads = 0
		s.errorReads = 0
	}

	// Publish each result to Home Assistant
	s.publishResults(ctx, results, shouldLog)
}

// publishResults publishes all results to MQTT
func (s *PollingService) publishResults(ctx context.Context, results map[string]*modbus.CommandResult, shouldLog bool) {
	for key, result := range results {
		logger.LogTrace("📊 %s: %.3f %s", result.Name, result.Value, result.Unit)

		// Publish to Home Assistant
		if pubErr := s.publisher.PublishSensorState(ctx, result); pubErr != nil {
			logger.LogError("⚠️ Error publishing sensor state for %s: %v", key, pubErr)
		} else {
			// Update last publish time for successful publications
			s.lastPublishTime[key] = time.Now()
		}
	}

	if shouldLog {
		logger.LogDebug("✅ Successfully executed and published %d strategies", len(results))
	}
}

// recordError records a gateway error
func (s *PollingService) recordError(ctx context.Context) {
	shouldMarkOffline := s.healthMonitor.RecordError()

	// If this is first error, log grace period start
	if s.healthMonitor.GetConsecutiveErrors() == 1 {
		logger.LogWarn("⚠️ First error detected, starting grace period")
	}

	// Check if we're still in grace period
	if s.healthMonitor.IsInGracePeriod() {
		logger.LogDebug("🕐 Error %d in grace period (%.1fs elapsed) - keeping status online",
			s.healthMonitor.GetConsecutiveErrors(),
			s.healthMonitor.GetTimeSinceFirstError().Seconds())
		return
	}

	// Grace period expired - set status to offline if needed
	if shouldMarkOffline && s.healthMonitor.IsOnline() {
		s.healthMonitor.MarkOffline()
		logger.LogError("🔴 Grace period expired - Gateway marked as OFFLINE after %d errors over %.1f seconds",
			s.healthMonitor.GetConsecutiveErrors(),
			s.healthMonitor.GetTimeSinceFirstError().Seconds())

		// Publish offline status
		if err := s.publisher.PublishStatusOffline(ctx); err != nil {
			logger.LogError("⚠️ Error publishing offline status: %v", err)
		}
	}
}

// recordSuccess records a successful operation
func (s *PollingService) recordSuccess(ctx context.Context) {
	s.healthMonitor.RecordSuccess()

	// If gateway was offline, mark it back online
	if !s.healthMonitor.IsOnline() {
		s.healthMonitor.MarkOnline()
		logger.LogInfo("🟢 Gateway marked as ONLINE - functionality restored")

		// Publish online status
		if err := s.publisher.PublishStatusOnline(ctx); err != nil {
			logger.LogError("⚠️ Error publishing online status: %v", err)
		}

		// Publish recovery diagnostic
		if err := s.publisher.PublishDiagnostic(ctx, 0, "Functionality restored - gateway back online"); err != nil {
			logger.LogError("⚠️ Error publishing recovery diagnostic: %v", err)
		}
	}
}

// GetLastPublishTime returns the last publish time for a key
func (s *PollingService) GetLastPublishTime(key string) (time.Time, bool) {
	t, ok := s.lastPublishTime[key]
	return t, ok
}

// GetPerformanceStats returns current performance statistics
func (s *PollingService) GetPerformanceStats() (successfulReads, errorReads int, lastSummary time.Time) {
	return s.successfulReads, s.errorReads, s.lastSummaryTime
}
