package services

import (
	"context"
	"mqtt-modbus-bridge/pkg/builder"
	"mqtt-modbus-bridge/pkg/health"
	"mqtt-modbus-bridge/pkg/logger"
	"time"
)

// HeartbeatService manages periodic status heartbeats
// Single Responsibility: Send periodic online status to keep Home Assistant informed
type HeartbeatService struct {
	publisher     builder.PublisherInterface
	healthMonitor *health.GatewayHealthMonitor
	interval      time.Duration
}

// NewHeartbeatService creates a new heartbeat service
func NewHeartbeatService(
	publisher builder.PublisherInterface,
	healthMonitor *health.GatewayHealthMonitor,
	interval time.Duration,
) *HeartbeatService {
	return &HeartbeatService{
		publisher:     publisher,
		healthMonitor: healthMonitor,
		interval:      interval,
	}
}

// Start begins the heartbeat loop
func (s *HeartbeatService) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	logger.LogInfo("💓 Heartbeat service started with interval: %v", s.interval)

	for {
		select {
		case <-ctx.Done():
			logger.LogDebug("🔇 Heartbeat service stopped")
			return
		case <-ticker.C:
			s.sendHeartbeat(ctx)
		}
	}
}

// sendHeartbeat sends a status heartbeat if gateway is online
func (s *HeartbeatService) sendHeartbeat(ctx context.Context) {
	// Only send heartbeat if we're currently marked as online
	if !s.healthMonitor.IsOnline() {
		logger.LogDebug("💔 Skipping heartbeat - gateway is offline")
		return
	}

	if err := s.publisher.PublishStatusOnline(ctx); err != nil {
		logger.LogError("⚠️ Heartbeat failed: %v", err)
		return
	}

	logger.LogDebug("💓 Heartbeat sent: online")

	// Also send diagnostic heartbeat to keep sensor alive
	if diagErr := s.publisher.PublishDiagnostic(ctx, 0, "MQTT-Modbus Bridge running"); diagErr != nil {
		logger.LogDebug("⚠️ Diagnostic heartbeat failed: %v", diagErr)
	}
}

// SendImmediateHeartbeat sends a heartbeat immediately (useful for startup)
func (s *HeartbeatService) SendImmediateHeartbeat(ctx context.Context) error {
	s.sendHeartbeat(ctx)
	return nil
}
