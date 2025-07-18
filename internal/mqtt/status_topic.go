package mqtt

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/internal/config"
	"mqtt-modbus-bridge/internal/logger"
	"mqtt-modbus-bridge/internal/modbus"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// StatusTopic handles status-related publishing (online/offline)
type StatusTopic struct {
	config *config.HAConfig
}

// NewStatusTopic creates a new status topic handler
func NewStatusTopic(config *config.HAConfig) *StatusTopic {
	return &StatusTopic{
		config: config,
	}
}

// PublishDiscovery for status topic (not applicable)
func (s *StatusTopic) PublishDiscovery(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	// Status topics don't need discovery configuration
	return nil
}

// PublishState publishes status (online/offline)
func (s *StatusTopic) PublishState(ctx context.Context, client mqtt.Client, result *modbus.CommandResult) error {
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	// Validate the result before publishing
	if err := s.ValidateData(result); err != nil {
		return fmt.Errorf("invalid status data: %w", err)
	}

	// For status, we expect the result.Value to be interpreted as status
	// 1.0 = online, 0.0 = offline
	var payload string
	if result.Value > 0 {
		payload = "online"
	} else {
		payload = "offline"
	}

	token := client.Publish(s.config.StatusTopic, 0, true, payload)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		if token.Error() != nil {
			return fmt.Errorf("error publishing status: %w", token.Error())
		}
	}

	return nil
}

// GetTopicPrefix returns the topic prefix for status topic
func (s *StatusTopic) GetTopicPrefix() string {
	return "status"
}

// ValidateData validates status data
func (s *StatusTopic) ValidateData(result *modbus.CommandResult) error {
	// Status values should be 0 or 1
	if result.Value < 0 || result.Value > 1 {
		return fmt.Errorf("invalid status value: %.3f (expected 0 or 1)", result.Value)
	}
	return nil
}

// PublishOnline publishes online status
func (s *StatusTopic) PublishOnline(ctx context.Context, client mqtt.Client) error {
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	token := client.Publish(s.config.StatusTopic, 0, true, "online")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		if token.Error() != nil {
			return fmt.Errorf("error publishing online status: %w", token.Error())
		}
	}

	logger.LogDebug("📡 Published bridge status: online")
	return nil
}

// PublishOffline publishes offline status
func (s *StatusTopic) PublishOffline(ctx context.Context, client mqtt.Client) error {
	if !client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	token := client.Publish(s.config.StatusTopic, 0, true, "offline")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		if token.Error() != nil {
			return fmt.Errorf("error publishing offline status: %w", token.Error())
		}
	}

	logger.LogDebug("📡 Published bridge status: offline")
	return nil
}
