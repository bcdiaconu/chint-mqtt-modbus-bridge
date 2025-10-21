package builder

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/diagnostics"
	"mqtt-modbus-bridge/pkg/gateway"
	"mqtt-modbus-bridge/pkg/health"
	"mqtt-modbus-bridge/pkg/modbus"
	"mqtt-modbus-bridge/pkg/mqtt"
	"time"
)

// ApplicationBuilder provides a fluent interface for constructing Application instances
// Following Builder pattern to enable dependency injection and improve testability
type ApplicationBuilder struct {
	config            *config.Config
	gateway           GatewayInterface
	executor          ExecutorInterface
	publisher         PublisherInterface
	healthMonitor     *health.GatewayHealthMonitor
	diagnosticManager *diagnostics.DeviceManager
	errorGracePeriod  time.Duration
}

// GatewayInterface defines the contract for gateway implementations
// Enables mocking and testing
type GatewayInterface interface {
	Connect(ctx context.Context) error
	Disconnect()
	IsConnected() bool
	SendCommand(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16) error
	WaitForResponse(ctx context.Context, timeout int) ([]byte, error)
	SendCommandAndWaitForResponse(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16, timeoutSeconds int) ([]byte, error)
}

// ExecutorInterface defines the contract for strategy executors
// Enables mocking and testing
type ExecutorInterface interface {
	ExecuteAll(ctx context.Context) (map[string]*modbus.CommandResult, error)
	ExecuteGroup(ctx context.Context, groupKey string) (map[string]*modbus.CommandResult, error)
	RegisterFromDevices(devices map[string]config.Device) error
}

// PublisherInterface defines the contract for MQTT publishers
// Enables mocking and testing
type PublisherInterface interface {
	Connect(ctx context.Context) error
	Disconnect()
	PublishAllDiscoveries(ctx context.Context, results []*modbus.CommandResult, deviceInfo *mqtt.DeviceInfo) error
	PublishSensorState(ctx context.Context, result *modbus.CommandResult) error
	PublishStatusOnline(ctx context.Context) error
	PublishStatusOffline(ctx context.Context) error
	PublishDiagnostic(ctx context.Context, code int, message string) error
	PublishDiagnosticDiscovery(ctx context.Context) error
	PublishDeviceDiagnosticDiscovery(ctx context.Context, deviceID string, deviceInfo *mqtt.DeviceInfo) error
	PublishDeviceDiagnosticState(ctx context.Context, deviceID string, metrics *mqtt.DeviceMetrics) error
}

// NewApplicationBuilder creates a new builder with default configuration
func NewApplicationBuilder(cfg *config.Config) *ApplicationBuilder {
	return &ApplicationBuilder{
		config:           cfg,
		errorGracePeriod: 15 * time.Second, // Default grace period
	}
}

// WithGateway sets a custom gateway implementation
func (b *ApplicationBuilder) WithGateway(gw GatewayInterface) *ApplicationBuilder {
	b.gateway = gw
	return b
}

// WithExecutor sets a custom executor implementation
func (b *ApplicationBuilder) WithExecutor(exec ExecutorInterface) *ApplicationBuilder {
	b.executor = exec
	return b
}

// WithPublisher sets a custom publisher implementation
func (b *ApplicationBuilder) WithPublisher(pub PublisherInterface) *ApplicationBuilder {
	b.publisher = pub
	return b
}

// WithHealthMonitor sets a custom health monitor
func (b *ApplicationBuilder) WithHealthMonitor(monitor *health.GatewayHealthMonitor) *ApplicationBuilder {
	b.healthMonitor = monitor
	return b
}

// WithDiagnosticManager sets a custom diagnostic manager
func (b *ApplicationBuilder) WithDiagnosticManager(manager *diagnostics.DeviceManager) *ApplicationBuilder {
	b.diagnosticManager = manager
	return b
}

// WithErrorGracePeriod sets the error grace period
func (b *ApplicationBuilder) WithErrorGracePeriod(period time.Duration) *ApplicationBuilder {
	b.errorGracePeriod = period
	return b
}

// Build constructs the Application with all dependencies
// Creates default implementations for any missing dependencies
func (b *ApplicationBuilder) Build() (*Application, error) {
	// Validate required config
	if b.config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Create default implementations if not provided
	if b.gateway == nil {
		b.gateway = gateway.NewUSRGateway(&b.config.MQTT)
	}

	if b.executor == nil {
		b.executor = modbus.NewStrategyExecutor(b.gateway.(gateway.Gateway), b.config.HomeAssistant.DiscoveryPrefix)
	}

	if b.publisher == nil {
		b.publisher = mqtt.NewPublisher(&b.config.MQTT, &b.config.HomeAssistant)
	}

	if b.healthMonitor == nil {
		b.healthMonitor = health.NewGatewayHealthMonitor(b.errorGracePeriod)
	}

	// Diagnostic manager is optional and created only if enabled
	if b.diagnosticManager == nil && b.config.HomeAssistant.DeviceDiagnostics.Enabled {
		b.diagnosticManager = diagnostics.NewDeviceManager(
			b.publisher,
			&b.config.HomeAssistant.DeviceDiagnostics,
			b.config.Devices,
		)
	}

	// Build the application
	app := &Application{
		config:            b.config,
		gateway:           b.gateway,
		executor:          b.executor,
		publisher:         b.publisher,
		healthMonitor:     b.healthMonitor,
		diagnosticManager: b.diagnosticManager,
		lastPublishTime:   make(map[string]time.Time),
		lastSummaryTime:   time.Now(),
	}

	return app, nil
}

// Application represents the main application structure
// Extracted from main.go for better testability
type Application struct {
	config            *config.Config
	gateway           GatewayInterface
	executor          ExecutorInterface
	publisher         PublisherInterface
	healthMonitor     *health.GatewayHealthMonitor
	diagnosticManager *diagnostics.DeviceManager
	lastPublishTime   map[string]time.Time
	lastSummaryTime   time.Time
	successfulReads   int
	errorReads        int
}

// GetConfig returns the application configuration
func (app *Application) GetConfig() *config.Config {
	return app.config
}

// GetGateway returns the gateway interface
func (app *Application) GetGateway() GatewayInterface {
	return app.gateway
}

// GetExecutor returns the executor interface
func (app *Application) GetExecutor() ExecutorInterface {
	return app.executor
}

// GetPublisher returns the publisher interface
func (app *Application) GetPublisher() PublisherInterface {
	return app.publisher
}

// GetHealthMonitor returns the health monitor
func (app *Application) GetHealthMonitor() *health.GatewayHealthMonitor {
	return app.healthMonitor
}

// GetDiagnosticManager returns the diagnostic manager
func (app *Application) GetDiagnosticManager() *diagnostics.DeviceManager {
	return app.diagnosticManager
}

// GetLastPublishTime returns the last publish time map
func (app *Application) GetLastPublishTime() map[string]time.Time {
	return app.lastPublishTime
}

// GetLastSummaryTime returns the last summary time
func (app *Application) GetLastSummaryTime() time.Time {
	return app.lastSummaryTime
}

// SetLastSummaryTime sets the last summary time
func (app *Application) SetLastSummaryTime(t time.Time) {
	app.lastSummaryTime = t
}

// IncrementSuccessfulReads increments the successful reads counter
func (app *Application) IncrementSuccessfulReads() {
	app.successfulReads++
}

// IncrementErrorReads increments the error reads counter
func (app *Application) IncrementErrorReads() {
	app.errorReads++
}

// GetSuccessfulReads returns the successful reads count
func (app *Application) GetSuccessfulReads() int {
	return app.successfulReads
}

// GetErrorReads returns the error reads count
func (app *Application) GetErrorReads() int {
	return app.errorReads
}

// ResetPerformanceCounters resets the performance counters
func (app *Application) ResetPerformanceCounters() {
	app.successfulReads = 0
	app.errorReads = 0
}
