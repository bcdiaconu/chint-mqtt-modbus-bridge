package integration

import (
	"context"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/diagnostics"
	"mqtt-modbus-bridge/pkg/mqtt"
	"testing"
	"time"
)

// MockPublisher implements mqtt.PublisherInterface for testing
type MockPublisher struct {
	discoveryPublished map[string]bool
	statePublished     map[string]int
}

func NewMockPublisher() *MockPublisher {
	return &MockPublisher{
		discoveryPublished: make(map[string]bool),
		statePublished:     make(map[string]int),
	}
}

func (m *MockPublisher) PublishDeviceDiagnosticDiscovery(ctx context.Context, deviceID string, deviceInfo *mqtt.DeviceInfo) error {
	m.discoveryPublished[deviceID] = true
	return nil
}

func (m *MockPublisher) PublishDeviceDiagnosticState(ctx context.Context, deviceID string, metrics *mqtt.DeviceMetrics) error {
	m.statePublished[deviceID]++
	return nil
}

// getTestConfig returns a minimal test configuration
func getTestConfig() *config.DeviceDiagnosticsConfig {
	return &config.DeviceDiagnosticsConfig{
		Enabled:              true,
		PublishOnStateChange: true,
	}
}

// TestDeviceManagerCreation tests basic manager creation
func TestDeviceManagerCreation(t *testing.T) {
	publisher := NewMockPublisher()
	diagnosticConfig := getTestConfig()

	devices := map[string]config.Device{
		"test_device": {
			Metadata: config.DeviceMetadata{
				Name:         "Test Device",
				Manufacturer: "Test Manufacturer",
				Model:        "Test Model",
				Enabled:      true,
			},
		},
	}

	manager := diagnostics.NewDeviceManager(publisher, diagnosticConfig, devices)

	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}

	// Verify metrics were initialized
	metrics, err := manager.GetMetrics("test_device")
	if err != nil {
		t.Fatalf("Expected metrics to be initialized for test_device, got error: %v", err)
	}

	if metrics.CurrentState != "operational" {
		t.Errorf("Expected initial state to be 'operational', got '%s'", metrics.CurrentState)
	}
}

// TestRecordSuccess tests recording successful reads
func TestRecordSuccess(t *testing.T) {
	publisher := NewMockPublisher()
	diagnosticConfig := getTestConfig()

	devices := map[string]config.Device{
		"test_device": {
			Metadata: config.DeviceMetadata{
				Name:    "Test Device",
				Enabled: true,
			},
		},
	}

	manager := diagnostics.NewDeviceManager(publisher, diagnosticConfig, devices)

	// Record success
	manager.RecordSuccess("test_device", 50*time.Millisecond)

	// Verify metrics
	metrics, err := manager.GetMetrics("test_device")
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}

	if metrics.TotalReads != 1 {
		t.Errorf("Expected TotalReads=1, got %d", metrics.TotalReads)
	}

	if metrics.SuccessfulReads != 1 {
		t.Errorf("Expected SuccessfulReads=1, got %d", metrics.SuccessfulReads)
	}

	if metrics.ConsecutiveErrors != 0 {
		t.Errorf("Expected ConsecutiveErrors=0, got %d", metrics.ConsecutiveErrors)
	}
}

// TestRecordError tests recording failed reads
func TestRecordError(t *testing.T) {
	publisher := NewMockPublisher()
	diagnosticConfig := getTestConfig()

	devices := map[string]config.Device{
		"test_device": {
			Metadata: config.DeviceMetadata{
				Name:    "Test Device",
				Enabled: true,
			},
		},
	}

	manager := diagnostics.NewDeviceManager(publisher, diagnosticConfig, devices)

	// Record error
	manager.RecordError("test_device", "test error")

	// Verify metrics
	metrics, err := manager.GetMetrics("test_device")
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}

	if metrics.TotalReads != 1 {
		t.Errorf("Expected TotalReads=1, got %d", metrics.TotalReads)
	}

	if metrics.FailedReads != 1 {
		t.Errorf("Expected FailedReads=1, got %d", metrics.FailedReads)
	}

	if metrics.ConsecutiveErrors != 1 {
		t.Errorf("Expected ConsecutiveErrors=1, got %d", metrics.ConsecutiveErrors)
	}

	if metrics.LastError != "test error" {
		t.Errorf("Expected LastError='test error', got '%s'", metrics.LastError)
	}
}

// TestPublishDiscovery tests discovery publishing for all devices
func TestPublishDiscovery(t *testing.T) {
	publisher := NewMockPublisher()
	diagnosticConfig := getTestConfig()

	devices := map[string]config.Device{
		"device1": {
			Metadata: config.DeviceMetadata{
				Name:         "Device 1",
				Manufacturer: "Manufacturer 1",
				Model:        "Model 1",
				Enabled:      true,
			},
		},
		"device2": {
			Metadata: config.DeviceMetadata{
				Name:    "Device 2",
				Enabled: true,
			},
		},
		"device3": {
			Metadata: config.DeviceMetadata{
				Name:    "Device 3",
				Enabled: false, // Disabled device
			},
		},
	}

	manager := diagnostics.NewDeviceManager(publisher, diagnosticConfig, devices)
	ctx := context.Background()

	// Publish discovery
	err := manager.PublishDiscoveryForAllDevices(ctx)
	if err != nil {
		t.Fatalf("Failed to publish discovery: %v", err)
	}

	// Verify only enabled devices published
	if !publisher.discoveryPublished["device1"] {
		t.Error("Expected device1 discovery to be published")
	}

	if !publisher.discoveryPublished["device2"] {
		t.Error("Expected device2 discovery to be published")
	}

	if publisher.discoveryPublished["device3"] {
		t.Error("Expected device3 discovery NOT to be published (disabled)")
	}
}

// TestNilHomeAssistantConfig tests handling of nil HomeAssistant config
func TestNilHomeAssistantConfig(t *testing.T) {
	publisher := NewMockPublisher()
	diagnosticConfig := getTestConfig()

	devices := map[string]config.Device{
		"test_device": {
			Metadata: config.DeviceMetadata{
				Name:         "Test Device",
				Manufacturer: "Test Mfg",
				Model:        "Test Model",
				Enabled:      true,
			},
			HomeAssistant: nil, // Explicitly nil
		},
	}

	manager := diagnostics.NewDeviceManager(publisher, diagnosticConfig, devices)
	ctx := context.Background()

	// This should not panic
	err := manager.PublishDiscoveryForAllDevices(ctx)
	if err != nil {
		t.Fatalf("Failed to publish discovery with nil HomeAssistant: %v", err)
	}

	if !publisher.discoveryPublished["test_device"] {
		t.Error("Expected test_device discovery to be published even with nil HomeAssistant")
	}
}
