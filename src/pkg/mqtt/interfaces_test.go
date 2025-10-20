package mqtt

import (
	"context"
	"mqtt-modbus-bridge/pkg/modbus"
	"testing"
)

// MockSensorPublisher demonstrates using only SensorPublisher interface
type MockSensorPublisher struct {
	publishedCount int
}

func (m *MockSensorPublisher) PublishSensorDiscovery(ctx context.Context, result *modbus.CommandResult, deviceInfo *DeviceInfo) error {
	m.publishedCount++
	return nil
}

func (m *MockSensorPublisher) PublishSensorState(ctx context.Context, result *modbus.CommandResult) error {
	m.publishedCount++
	return nil
}

func (m *MockSensorPublisher) PublishAllDiscoveries(ctx context.Context, results []*modbus.CommandResult, deviceInfo *DeviceInfo) error {
	m.publishedCount += len(results)
	return nil
}

// TestInterfaceSegregation demonstrates Interface Segregation Principle
func TestInterfaceSegregation(t *testing.T) {
	// Components only depend on interfaces they need
	mock := &MockSensorPublisher{}

	// Function that only needs sensor publishing
	useSensorPublisher := func(sp SensorPublisher) error {
		// Can only call sensor methods
		return sp.PublishSensorState(context.Background(), nil)
	}

	if err := useSensorPublisher(mock); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if mock.publishedCount != 1 {
		t.Errorf("Expected 1 publish, got %d", mock.publishedCount)
	}
}

// TestPublisherImplementsAllInterfaces verifies Publisher implements all interfaces
func TestPublisherImplementsAllInterfaces(t *testing.T) {
	// This test ensures Publisher implements all interfaces
	// The actual verification happens at compile time via the var declarations in interfaces.go

	var _ SensorPublisher = (*Publisher)(nil)
	var _ StatusPublisher = (*Publisher)(nil)
	var _ DiagnosticPublisher = (*Publisher)(nil)
	var _ ConnectionManager = (*Publisher)(nil)
	var _ HAPublisher = (*Publisher)(nil)

	t.Log("✅ Publisher implements all required interfaces")
}

// Example: Service that only needs status publishing
type StatusService struct {
	statusPub StatusPublisher
}

func NewStatusService(sp StatusPublisher) *StatusService {
	return &StatusService{statusPub: sp}
}

func (s *StatusService) SetOnline(ctx context.Context) error {
	return s.statusPub.PublishStatusOnline(ctx)
}

// Example: Service that only needs diagnostic publishing
type DiagnosticService struct {
	diagPub DiagnosticPublisher
}

func NewDiagnosticService(dp DiagnosticPublisher) *DiagnosticService {
	return &DiagnosticService{diagPub: dp}
}

func (d *DiagnosticService) ReportError(ctx context.Context, code int, msg string) error {
	return d.diagPub.PublishDiagnostic(ctx, code, msg)
}

// TestServicesDependOnSpecificInterfaces demonstrates dependency injection
func TestServicesDependOnSpecificInterfaces(t *testing.T) {
	// In real usage, we pass the full Publisher to services
	// but they only see the interface they need

	// Services get only what they need
	t.Log("✅ Services can depend on specific interfaces")
	t.Log("   - StatusService depends only on StatusPublisher")
	t.Log("   - DiagnosticService depends only on DiagnosticPublisher")
	t.Log("   - This follows Interface Segregation Principle")
}

// TestInterfaceComposition tests HAPublisher composite interface
func TestInterfaceComposition(t *testing.T) {
	// HAPublisher composes all interfaces
	// This is useful for the main application that needs everything

	var _ SensorPublisher = (HAPublisher)(nil)
	var _ StatusPublisher = (HAPublisher)(nil)
	var _ DiagnosticPublisher = (HAPublisher)(nil)
	var _ ConnectionManager = (HAPublisher)(nil)

	t.Log("✅ HAPublisher correctly composes all domain interfaces")
}
