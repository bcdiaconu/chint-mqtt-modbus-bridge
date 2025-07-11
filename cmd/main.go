package main

import (
	"context"
	"fmt"
	"log"
	"mqtt-modbus-bridge/internal/config"
	"mqtt-modbus-bridge/internal/homeassistant"
	"mqtt-modbus-bridge/internal/modbus"
	"mqtt-modbus-bridge/internal/mqtt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Diagnostic error codes
const (
	DiagnosticOK               = 0
	DiagnosticMQTTDisconnected = 1001
	DiagnosticModbusTimeout    = 1002
	DiagnosticModbusError      = 1003
	DiagnosticConfigError      = 1004
	DiagnosticGatewayError     = 1005
)

// Application main application class
// Facade Pattern - simplified interface for complex system
type Application struct {
	config     *config.Config
	gateway    *mqtt.USRGateway
	executor   *modbus.CommandExecutor
	publisher  *homeassistant.Publisher
	strategies map[string]modbus.CommandStrategy
	mu         sync.Mutex // Mutex for synchronizing access to the gateway

	// Gateway status tracking
	consecutiveErrors int
	isGatewayOnline   bool
	lastErrorTime     time.Time

	// Grace period for offline status - avoid oscillation for temporary errors
	errorGracePeriod   time.Duration // Waiting time before marking as offline
	firstErrorTime     time.Time     // First error in the current sequence
	statusSetToOffline bool          // Flag to avoid repeatedly setting offline status

	// Performance tracking for cleaner output
	lastSummaryTime time.Time
	successfulReads int
	errorReads      int
}

// NewApplication creates a new application instance
func NewApplication(configPath string) (*Application, error) {
	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}

	// Create gateway
	gateway := mqtt.NewUSRGateway(&cfg.MQTT)

	// Create command executor
	executor := modbus.NewCommandExecutor(gateway, &cfg.Modbus)

	// Create publisher for Home Assistant
	publisher := homeassistant.NewPublisher(&cfg.MQTT, &cfg.HomeAssistant)

	app := &Application{
		config:     cfg,
		gateway:    gateway,
		executor:   executor,
		publisher:  publisher,
		strategies: make(map[string]modbus.CommandStrategy),
		// Initialize gateway status tracking
		consecutiveErrors: 0,
		isGatewayOnline:   true,
		lastErrorTime:     time.Time{},
		// Initialize grace period tracking - 15 seconds grace before marking offline
		errorGracePeriod:   15 * time.Second,
		firstErrorTime:     time.Time{},
		statusSetToOffline: false,
		// Initialize performance tracking
		lastSummaryTime: time.Now(),
		successfulReads: 0,
		errorReads:      0,
	}

	// Register strategies
	if err := app.registerStrategies(); err != nil {
		return nil, fmt.Errorf("error registering strategies: %w", err)
	}

	return app, nil
}

// registerStrategies registers all strategies from configuration
// Factory Pattern for creating strategies
func (app *Application) registerStrategies() error {
	factory := modbus.NewStrategyFactory(app.config.Modbus.SlaveID)

	for name, register := range app.config.Registers {
		strategy, err := factory.CreateStrategy(register)
		if err != nil {
			return fmt.Errorf("error creating strategy %s: %w", name, err)
		}

		app.executor.RegisterStrategy(name, strategy)
		app.strategies[name] = strategy
		log.Printf("✅ Strategy registered: %s (%s)", name, register.Name)
	}

	// Set executor for reactive power strategies that need it
	for name, strategy := range app.strategies {
		if reactivePowerStrategy, ok := strategy.(*modbus.ReactivePowerStrategy); ok {
			reactivePowerStrategy.SetExecutor(app.executor)
			log.Printf("🔧 Executor set for reactive power strategy: %s", name)
		}
	}

	return nil
}

// Start starts the application
func (app *Application) Start(ctx context.Context) error {
	log.Printf("🚀 Starting MQTT-Modbus Bridge...")

	// Connect gateway
	if err := app.gateway.Connect(ctx); err != nil {
		return fmt.Errorf("error connecting gateway: %w", err)
	}

	// Connect publisher
	if err := app.publisher.Connect(ctx); err != nil {
		return fmt.Errorf("error connecting publisher: %w", err)
	}

	// Publish discovery configurations for Home Assistant
	if err := app.publishDiscoveryConfigs(ctx); err != nil {
		log.Printf("⚠️ Error publishing discovery configs: %v", err)
		app.publisher.PublishDiagnostic(ctx, DiagnosticConfigError, fmt.Sprintf("Discovery config error: %v", err))
	}

	// Publish online status
	if err := app.publisher.PublishStatusOnline(ctx); err != nil {
		log.Printf("⚠️ Error publishing online status: %v", err)
	} else {
		app.publisher.PublishDiagnostic(ctx, DiagnosticOK, "MQTT-Modbus Bridge started successfully")
	}

	// Start separate polling loops for different register types
	go app.mainLoopNormalRegisters(ctx)
	go app.mainLoopEnergyRegisters(ctx)

	log.Printf("✅ MQTT-Modbus Bridge started successfully")
	log.Printf("🔇 Verbose logging reduced - Summary reports every 30 seconds")
	return nil
}

// Stop stops the application
func (app *Application) Stop() {
	log.Printf("🛑 Stopping MQTT-Modbus Bridge...")

	// Publish offline status before disconnecting
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.publisher.PublishStatusOffline(ctx); err != nil {
		log.Printf("⚠️ Error publishing offline status: %v", err)
	} else {
		app.publisher.PublishDiagnostic(ctx, DiagnosticOK, "MQTT-Modbus Bridge stopped gracefully")
	}

	app.gateway.Disconnect()
	app.publisher.Disconnect()

	log.Printf("✅ MQTT-Modbus Bridge stopped")
}

// mainLoopNormalRegisters polling loop for normal registers (voltage, current, power, etc.)
func (app *Application) mainLoopNormalRegisters(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(app.config.Modbus.RegisterDelay) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			app.readNormalRegisters(ctx)
		}
	}
}

// mainLoopEnergyRegisters polling loop for energy registers (kWh meters)
func (app *Application) mainLoopEnergyRegisters(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(app.config.Modbus.EnergyDelay) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			app.readEnergyRegisters(ctx)
		}
	}
}

// isEnergyRegister checks if a register is an energy register (kWh meter)
func (app *Application) isEnergyRegister(name string) bool {
	energyRegisters := []string{"energy_total", "energy_imported", "energy_exported"}
	for _, energyReg := range energyRegisters {
		if name == energyReg {
			return true
		}
	}
	return false
}

// readNormalRegisters reads normal registers (voltage, current, power, frequency, power factor)
func (app *Application) readNormalRegisters(ctx context.Context) {
	for name := range app.strategies {
		if !app.isEnergyRegister(name) {
			app.readSingleRegister(ctx, name, "📊 Normal")
		}
	}
}

// readEnergyRegisters reads energy registers (kWh meters)
func (app *Application) readEnergyRegisters(ctx context.Context) {
	for name := range app.strategies {
		if app.isEnergyRegister(name) {
			app.readSingleRegister(ctx, name, "⚡ Energy")
		}
	}
}

// readSingleRegister reads a single register and publishes to Home Assistant
func (app *Application) readSingleRegister(ctx context.Context, name string, logPrefix string) {
	// Lock to prevent concurrent access to gateway
	app.mu.Lock()
	defer app.mu.Unlock()

	result, err := app.executor.ExecuteCommand(ctx, name)
	if err != nil {
		app.errorReads++

		// Only log errors occasionally to avoid spam
		if app.consecutiveErrors == 0 || app.consecutiveErrors%10 == 0 {
			log.Printf("❌ %s execution error %s: %v", logPrefix, name, err)
		}

		// Track consecutive errors for gateway status
		app.handleGatewayError(ctx)

		// Publish diagnostic error
		errorMsg := fmt.Sprintf("Register %s read error: %v", name, err)
		if ctxErr := app.publisher.PublishDiagnostic(ctx, DiagnosticModbusError, errorMsg); ctxErr != nil {
			log.Printf("⚠️ Error publishing diagnostic: %v", ctxErr)
		}
		return
	}

	// Reset error counter on successful read
	app.handleGatewaySuccess(ctx)
	app.successfulReads++

	// Only show detailed logs every 30 seconds or for important changes
	shouldLog := time.Since(app.lastSummaryTime) >= 30*time.Second

	if shouldLog {
		log.Printf("� Summary - Success: %d, Errors: %d, Last 30s", app.successfulReads, app.errorReads)
		log.Printf("�📈 %s %s: %.3f %s", logPrefix, result.Name, result.Value, result.Unit)
		app.lastSummaryTime = time.Now()
		app.successfulReads = 0
		app.errorReads = 0
	}

	// Publish state to Home Assistant
	if err := app.publisher.PublishSensorState(ctx, result); err != nil {
		log.Printf("❌ %s state publication error %s: %v", logPrefix, result.Name, err)

		// Publish diagnostic error
		errorMsg := fmt.Sprintf("State publication error for %s: %v", result.Name, err)
		if ctxErr := app.publisher.PublishDiagnostic(ctx, DiagnosticMQTTDisconnected, errorMsg); ctxErr != nil {
			log.Printf("⚠️ Error publishing diagnostic: %v", ctxErr)
		}
	}
}

// handleGatewayError manages error counting and offline status with grace period
func (app *Application) handleGatewayError(ctx context.Context) {
	app.consecutiveErrors++
	app.lastErrorTime = time.Now()

	// If this is the first error in the sequence, record the time
	if app.firstErrorTime.IsZero() {
		app.firstErrorTime = time.Now()
		log.Printf("⚠️ First error detected, starting grace period of %.0f seconds", app.errorGracePeriod.Seconds())
	}

	// Check if we're still in grace period
	timeSinceFirstError := time.Since(app.firstErrorTime)
	if timeSinceFirstError < app.errorGracePeriod {
		// Still in grace period - don't change status to offline yet
		log.Printf("🕐 Error %d in grace period (%.1fs/%.0fs) - keeping status online",
			app.consecutiveErrors, timeSinceFirstError.Seconds(), app.errorGracePeriod.Seconds())
		return
	}

	// Grace period expired - set status to offline if not already done
	if app.isGatewayOnline && !app.statusSetToOffline {
		app.isGatewayOnline = false
		app.statusSetToOffline = true
		log.Printf("🔴 Grace period expired - App marked as OFFLINE after %d errors over %.1f seconds",
			app.consecutiveErrors, timeSinceFirstError.Seconds())

		// Publish offline status to ensure MQTT broker has correct state
		if err := app.publisher.PublishStatusOffline(ctx); err != nil {
			log.Printf("⚠️ Error publishing offline status: %v", err)
		}
	}
}

// handleGatewaySuccess resets error counter and changes status to online when functionality resumes
func (app *Application) handleGatewaySuccess(ctx context.Context) {
	// Reset error counter and grace period tracking
	app.consecutiveErrors = 0
	app.firstErrorTime = time.Time{}
	app.statusSetToOffline = false

	// If gateway was offline, mark it back online
	if !app.isGatewayOnline {
		app.isGatewayOnline = true
		log.Printf("🟢 App marked as ONLINE - functionality restored")

		// Publish online status
		if err := app.publisher.PublishStatusOnline(ctx); err != nil {
			log.Printf("⚠️ Error publishing online status: %v", err)
		}

		// Publish recovery diagnostic
		if err := app.publisher.PublishDiagnostic(ctx, DiagnosticOK, "Functionality restored - app back online"); err != nil {
			log.Printf("⚠️ Error publishing recovery diagnostic: %v", err)
		}
	}
}

// publishDiscoveryConfigs publishes discovery configurations for Home Assistant
func (app *Application) publishDiscoveryConfigs(ctx context.Context) error {
	log.Printf("🔍 Publishing discovery configurations for Home Assistant...")

	// Create mock results for discovery
	var results []*modbus.CommandResult
	for name, strategy := range app.strategies {
		result := &modbus.CommandResult{
			Strategy:    name,
			Name:        strategy.GetName(),
			Value:       0, // Mock value
			Unit:        strategy.GetUnit(),
			Topic:       strategy.GetTopic(),
			DeviceClass: strategy.GetDeviceClass(),
			StateClass:  strategy.GetStateClass(),
		}
		results = append(results, result)
	}

	// Publish sensor discoveries
	if err := app.publisher.PublishAllDiscoveries(ctx, results); err != nil {
		return err
	}

	// Publish diagnostic sensor discovery
	if err := app.publisher.PublishDiagnosticDiscovery(ctx); err != nil {
		log.Printf("⚠️ Error publishing diagnostic discovery: %v", err)
		// Don't return error - this is not critical
	}

	return nil
}

func main() {
	// Configure logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT and SIGTERM for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Configuration path from command arguments or default
	configPath := ""
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// Create application
	app, err := NewApplication(configPath)
	if err != nil {
		log.Fatalf("❌ Application creation error: %v", err)
	}

	// Start application
	if err := app.Start(ctx); err != nil {
		log.Fatalf("❌ Application start error: %v", err)
	}

	// Wait for stop signal
	<-sigChan
	log.Printf("📢 Stop signal received...")

	// Stop application
	app.Stop()
}
