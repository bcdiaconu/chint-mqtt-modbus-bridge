package main

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/gateway"
	"mqtt-modbus-bridge/pkg/logger"
	"mqtt-modbus-bridge/pkg/modbus"
	"mqtt-modbus-bridge/pkg/mqtt"
	"mqtt-modbus-bridge/pkg/topics"
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
	config    *config.Config
	gateway   *gateway.USRGateway
	executor  *modbus.StrategyExecutor
	publisher *mqtt.Publisher

	mu sync.Mutex // Mutex for synchronizing access to the gateway

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

	// Last publish tracking for forced republish
	lastPublishTime map[string]time.Time // Track last publish time per sensor
}

// NewApplication creates a new application instance
func NewApplication(configPath string) (*Application, error) {
	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}

	// Initialize logging with level
	logger.GlobalLogging = &cfg.Logging
	logger.LogStartup("🔧 Logging initialized with level: %s", cfg.Logging.Level)

	// Create gateway
	gatewayInstance := gateway.NewUSRGateway(&cfg.MQTT)

	// Create strategy executor with discovery prefix
	executor := modbus.NewStrategyExecutor(gatewayInstance, cfg.HomeAssistant.DiscoveryPrefix)

	// Create publisher for Home Assistant
	publisher := mqtt.NewPublisher(&cfg.MQTT, &cfg.HomeAssistant)

	app := &Application{
		config:    cfg,
		gateway:   gatewayInstance,
		executor:  executor,
		publisher: publisher,
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

		// Initialize last publish tracking
		lastPublishTime: make(map[string]time.Time),
	}

	// Register all strategies from devices
	if err := app.registerStrategies(); err != nil {
		return nil, fmt.Errorf("error registering strategies: %w", err)
	}

	return app, nil
}

// registerStrategies registers all strategies from device configuration
func (app *Application) registerStrategies() error {
	logger.LogInfo("🔧 Registering strategies from devices...")

	// Register from V2.1 device configuration
	if len(app.config.Devices) > 0 {
		return app.executor.RegisterFromDevices(app.config.Devices)
	}

	// V2.0 compatibility: convert old format to devices
	logger.LogInfo("⚠️ Using V2.0 configuration format (deprecated)")
	// TODO: Implement V2.0 compatibility if needed
	return fmt.Errorf("V2.0 format not yet supported with new strategy pattern")
}

// registerCommands registers all commands from configuration
// Factory Pattern for creating commands
// Old registerCommands and initializeGroups methods removed
// Now using registerStrategies() which calls executor.RegisterFromDevices()

// Start starts the application
func (app *Application) Start(ctx context.Context) error {
	logger.LogInfo("🚀 Starting MQTT-Modbus Bridge...")

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
		logger.LogError("⚠️ Error publishing discovery configs: %v", err)
		if diagErr := app.publisher.PublishDiagnostic(ctx, DiagnosticConfigError, fmt.Sprintf("Discovery config error: %v", err)); diagErr != nil {
			logger.LogError("⚠️ Error publishing diagnostic: %v", diagErr)
		}
	}

	// Publish online status
	if err := app.publisher.PublishStatusOnline(ctx); err != nil {
		logger.LogError("⚠️ Error publishing online status: %v", err)
	} else {
		if diagErr := app.publisher.PublishDiagnostic(ctx, DiagnosticOK, "MQTT-Modbus Bridge started successfully"); diagErr != nil {
			logger.LogError("⚠️ Error publishing diagnostic: %v", diagErr)
		}
	}

	// Start polling loop (unified for all register types)
	go app.mainLoopNormalRegisters(ctx)

	// Start heartbeat to maintain online status
	go app.heartbeatLoop(ctx)

	// Start forced republish loop for energy sensors
	go app.forcedRepublishLoop(ctx)

	logger.LogInfo("✅ MQTT-Modbus Bridge started successfully")
	logger.LogInfo("🔇 Verbose logging reduced - Summary reports every 30 seconds")
	return nil
}

// Stop stops the application
func (app *Application) Stop() {
	logger.LogInfo("🛑 Stopping MQTT-Modbus Bridge...")

	// Publish offline status before disconnecting
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.publisher.PublishStatusOffline(ctx); err != nil {
		logger.LogError("⚠️ Error publishing offline status: %v", err)
	} else {
		if diagErr := app.publisher.PublishDiagnostic(ctx, DiagnosticOK, "MQTT-Modbus Bridge stopped gracefully"); diagErr != nil {
			logger.LogError("⚠️ Error publishing diagnostic: %v", diagErr)
		}
	}

	app.gateway.Disconnect()
	app.publisher.Disconnect()

	logger.LogInfo("✅ MQTT-Modbus Bridge stopped")
}

// mainLoopNormalRegisters polling loop for normal registers (voltage, current, power, etc.)
func (app *Application) mainLoopNormalRegisters(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(app.config.Modbus.PollInterval) * time.Millisecond)
	defer ticker.Stop()

	logger.LogDebug("🔄 Registers polling started (interval: %dms)", app.config.Modbus.PollInterval)

	for {
		select {
		case <-ctx.Done():
			logger.LogDebug("🔄 Registers polling stopped")
			return
		case <-ticker.C:
			logger.LogDebug("🔄 Polling tick - executing all strategies...")
			app.executeAllStrategies(ctx)
		}
	}
}

// mainLoopEnergyRegisters - removed (now using unified polling)

// executeAllStrategies executes all registered strategies and publishes results
func (app *Application) executeAllStrategies(ctx context.Context) {
	// Execute all strategies (groups first, then calculated)
	results, err := app.executor.ExecuteAll(ctx)

	if err != nil {
		app.errorReads++
		app.handleGatewayError(ctx)
		logger.LogError("❌ Strategy execution error: %v", err)

		// Publish diagnostic
		errorMsg := fmt.Sprintf("Strategy execution error: %v", err)
		if diagErr := app.publisher.PublishDiagnostic(ctx, DiagnosticModbusError, errorMsg); diagErr != nil {
			logger.LogError("⚠️ Error publishing diagnostic: %v", diagErr)
		}
		return
	}

	// Success - publish all results
	app.successfulReads += len(results)
	app.handleGatewaySuccess(ctx)

	// Only show detailed logs every 30 seconds
	shouldLog := time.Since(app.lastSummaryTime) >= 30*time.Second

	if shouldLog {
		logger.LogInfo("📊 Summary - Success: %d, Errors: %d, Last 30s", app.successfulReads, app.errorReads)
		app.lastSummaryTime = time.Now()
		app.successfulReads = 0
		app.errorReads = 0
	}

	// Publish each result to Home Assistant
	for key, result := range results {
		logger.LogTrace("� %s: %.3f %s", result.Name, result.Value, result.Unit)

		// Publish to Home Assistant
		if pubErr := app.publisher.PublishSensorState(ctx, result); pubErr != nil {
			logger.LogError("⚠️ Error publishing sensor state for %s: %v", key, pubErr)
		} else {
			// Update last publish time for successful publications
			app.updateLastPublishTime(key)
		}
	}

	if shouldLog {
		logger.LogDebug("✅ Successfully executed and published %d strategies", len(results))
	}
}

// handleGatewayError manages error counting and offline status with grace period
func (app *Application) handleGatewayError(ctx context.Context) {
	app.consecutiveErrors++
	app.lastErrorTime = time.Now()

	// If this is the first error in the sequence, record the time
	if app.firstErrorTime.IsZero() {
		app.firstErrorTime = time.Now()
		logger.LogWarn("⚠️ First error detected, starting grace period of %.0f seconds", app.errorGracePeriod.Seconds())
	}

	// Check if we're still in grace period
	timeSinceFirstError := time.Since(app.firstErrorTime)
	if timeSinceFirstError < app.errorGracePeriod {
		// Still in grace period - don't change status to offline yet
		logger.LogDebug("🕐 Error %d in grace period (%.1fs/%.0fs) - keeping status online",
			app.consecutiveErrors, timeSinceFirstError.Seconds(), app.errorGracePeriod.Seconds())
		return
	}

	// Grace period expired - set status to offline if not already done
	if app.isGatewayOnline && !app.statusSetToOffline {
		app.isGatewayOnline = false
		app.statusSetToOffline = true
		logger.LogError("🔴 Grace period expired - App marked as OFFLINE after %d errors over %.1f seconds",
			app.consecutiveErrors, timeSinceFirstError.Seconds())

		// Publish offline status to ensure MQTT broker has correct state
		if err := app.publisher.PublishStatusOffline(ctx); err != nil {
			logger.LogError("⚠️ Error publishing offline status: %v", err)
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
		logger.LogInfo("🟢 App marked as ONLINE - functionality restored")

		// Publish online status
		if err := app.publisher.PublishStatusOnline(ctx); err != nil {
			logger.LogError("⚠️ Error publishing online status: %v", err)
		}

		// Publish recovery diagnostic
		if err := app.publisher.PublishDiagnostic(ctx, DiagnosticOK, "Functionality restored - app back online"); err != nil {
			logger.LogError("⚠️ Error publishing recovery diagnostic: %v", err)
		}
	}
}

// publishDiscoveryConfigs publishes discovery configurations for Home Assistant
func (app *Application) publishDiscoveryConfigs(ctx context.Context) error {
	logger.LogDebug("🔍 Publishing discovery configurations for Home Assistant...")

	// Check if using V2.1 (device-based) configuration
	if len(app.config.Devices) > 0 {
		// V2.1: Publish discovery per device
		return app.publishDiscoveryConfigsV21(ctx)
	}

	// V2.0/V1: Use global device (backward compatibility)
	return app.publishDiscoveryConfigsLegacy(ctx)
}

// publishDiscoveryConfigsV21 publishes discoveries for V2.1 device-based config
func (app *Application) publishDiscoveryConfigsV21(ctx context.Context) error {
	for deviceKey, device := range app.config.Devices {
		if !device.IsEnabled() {
			logger.LogDebug("⏭️ Skipping disabled device: %s", deviceKey)
			continue
		}

		// Build DeviceInfo for this Modbus device
		// Use device_id from homeassistant config, or deviceKey as fallback
		haDeviceID := device.GetHADeviceID(deviceKey)

		deviceInfo := &mqtt.DeviceInfo{
			Name:         device.GetHADeviceName(),
			Identifiers:  []string{haDeviceID},
			Manufacturer: device.GetHAManufacturer(),
			Model:        device.GetHAModel(),
		}

		logger.LogDebug("📡 Publishing discovery for device: %s (slave_id=%d)", device.GetName(), device.GetSlaveID())

		// Create mock results for this device's sensors
		var deviceResults []*modbus.CommandResult

		// Add register_groups sensors
		for _, group := range device.Modbus.RegisterGroups {
			for _, register := range group.Registers {
				// Construct the full HA topic path automatically with discovery prefix
				topic := topics.ConstructHATopic(app.config.HomeAssistant.DiscoveryPrefix, haDeviceID, register.Key, register.DeviceClass)

				result := &modbus.CommandResult{
					Strategy:    register.Key,
					Name:        register.Name,
					Value:       0, // Mock value
					Unit:        register.Unit,
					Topic:       topic,
					SensorKey:   register.Key, // Just the sensor key, not device_id_sensor_key
					DeviceClass: register.DeviceClass,
					StateClass:  register.StateClass,
				}
				deviceResults = append(deviceResults, result)
			}
		}

		// Add calculated_values sensors
		for _, calc := range device.CalculatedValues {
			// Construct the full HA topic path automatically with discovery prefix
			topic := topics.ConstructHATopic(app.config.HomeAssistant.DiscoveryPrefix, haDeviceID, calc.Key, calc.DeviceClass)

			result := &modbus.CommandResult{
				Strategy:    calc.Key,
				Name:        calc.Name,
				Value:       0, // Mock value
				Unit:        calc.Unit,
				Topic:       topic,
				SensorKey:   calc.Key, // Just the sensor key, not device_id_sensor_key
				DeviceClass: calc.DeviceClass,
				StateClass:  calc.StateClass,
			}
			deviceResults = append(deviceResults, result)
		}

		// Publish sensor discoveries for this device
		if err := app.publisher.PublishAllDiscoveries(ctx, deviceResults, deviceInfo); err != nil {
			logger.LogWarn("⚠️ Error publishing discoveries for device %s: %v", deviceKey, err)
			// Continue with other devices
		}

		// Small pause between devices
		time.Sleep(200 * time.Millisecond)
	}

	// Publish bridge-level diagnostic sensor discovery
	if err := app.publisher.PublishDiagnosticDiscovery(ctx); err != nil {
		logger.LogError("⚠️ Error publishing diagnostic discovery: %v", err)
	}

	return nil
}

// publishDiscoveryConfigsLegacy publishes discoveries for V2.0/V1 configs (backward compatibility)
func (app *Application) publishDiscoveryConfigsLegacy(ctx context.Context) error {
	// Get all strategies from executor to create mock results
	allStrategies := app.executor.GetAllStrategies()

	var results []*modbus.CommandResult
	for key, strategy := range allStrategies {
		// Check strategy type to handle different interfaces
		switch s := strategy.(type) {
		case *modbus.SingleRegisterStrategy:
			register := s.GetRegisterInfo()
			result := &modbus.CommandResult{
				Strategy:    key,
				Name:        register.Name,
				Value:       0,
				Unit:        register.Unit,
				Topic:       register.HATopic,
				SensorKey:   key,
				DeviceClass: register.DeviceClass,
				StateClass:  register.StateClass,
			}
			results = append(results, result)

		case *modbus.CalculatedRegisterStrategy:
			register := s.GetRegisterInfo()
			result := &modbus.CommandResult{
				Strategy:    key,
				Name:        register.Name,
				Value:       0,
				Unit:        register.Unit,
				Topic:       register.HATopic,
				SensorKey:   key,
				DeviceClass: register.DeviceClass,
				StateClass:  register.StateClass,
			}
			results = append(results, result)

		case *modbus.GroupRegisterStrategy:
			// Groups contain multiple registers, extract them
			for _, regWithKey := range s.GetRegisters() {
				result := &modbus.CommandResult{
					Strategy:    regWithKey.Key,
					Name:        regWithKey.Register.Name,
					Value:       0,
					Unit:        regWithKey.Register.Unit,
					Topic:       regWithKey.Register.HATopic,
					SensorKey:   regWithKey.Key,
					DeviceClass: regWithKey.Register.DeviceClass,
					StateClass:  regWithKey.Register.StateClass,
				}
				results = append(results, result)
			}
		}
	}

	// Publish sensor discoveries with nil deviceInfo (uses global config)
	if err := app.publisher.PublishAllDiscoveries(ctx, results, nil); err != nil {
		return err
	}

	// Publish diagnostic sensor discovery
	if err := app.publisher.PublishDiagnosticDiscovery(ctx); err != nil {
		logger.LogError("⚠️ Error publishing diagnostic discovery: %v", err)
		// Don't return error - this is not critical
	}

	return nil
}

func main() {
	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT and SIGTERM for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Parse command line arguments
	configPath := ""
	diagnosticMode := false

	for i, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Printf("Usage: %s [config_path] [--diagnostic]\n", os.Args[0])
			fmt.Printf("  config_path: Path to configuration file (optional)\n")
			fmt.Printf("  --diagnostic: Run diagnostic mode to test connectivity\n")
			return
		} else if arg == "--diagnostic" {
			diagnosticMode = true
		} else if i == 0 { // First argument is config path
			configPath = arg
		}
	}

	// Create application
	app, err := NewApplication(configPath)
	if err != nil {
		logger.LogError("Application creation error: %v", err)
		os.Exit(1)
	}

	// Run diagnostic mode if requested
	if diagnosticMode {
		logger.LogInfo("🔍 Running diagnostic mode...")

		// Connect gateway for diagnostic
		if err := app.gateway.Connect(ctx); err != nil {
			logger.LogError("Gateway connection error: %v", err)
			os.Exit(1)
		}

		// Connect publisher for diagnostic
		if err := app.publisher.Connect(ctx); err != nil {
			logger.LogError("Publisher connection error: %v", err)
			os.Exit(1)
		}

		// Run diagnostic tests
		if err := app.DiagnosticMode(ctx); err != nil {
			logger.LogError("Diagnostic failed: %v", err)
			os.Exit(1)
		}

		logger.LogInfo("✅ Diagnostic completed successfully")
		return
	}

	// Start application
	if err := app.Start(ctx); err != nil {
		logger.LogError("Application start error: %v", err)
		os.Exit(1)
	}

	// Wait for stop signal
	<-sigChan
	logger.LogInfo("📢 Stop signal received...")

	// Stop application
	app.Stop()
}

// heartbeatLoop sends periodic "online" status to maintain availability
func (app *Application) heartbeatLoop(ctx context.Context) {
	// Use heartbeat_interval from config, default to 20 seconds if not specified
	heartbeatInterval := app.config.MQTT.HeartbeatInterval
	if heartbeatInterval == 0 {
		heartbeatInterval = 20
	}

	ticker := time.NewTicker(time.Duration(heartbeatInterval) * time.Second)
	defer ticker.Stop()

	logger.LogInfo("💓 Heartbeat loop started with interval: %d seconds", heartbeatInterval)

	for {
		select {
		case <-ctx.Done():
			logger.LogDebug("🔇 Heartbeat loop stopped")
			return
		case <-ticker.C:
			// Only send heartbeat if we're currently marked as online
			if app.isGatewayOnline {
				if err := app.publisher.PublishStatusOnline(ctx); err != nil {
					logger.LogError("⚠️ Heartbeat failed: %v", err)
				} else {
					logger.LogDebug("💓 Heartbeat sent: online")
				}
			}
		}
	}
}

// updateLastPublishTime updates the last publish time for a sensor
func (app *Application) updateLastPublishTime(sensorName string) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.lastPublishTime[sensorName] = time.Now()
}

// forcedRepublishLoop periodically republishes energy values to prevent Home Assistant unavailable states
func (app *Application) forcedRepublishLoop(ctx context.Context) {
	// Get republish interval from config, default to 4 hours if not set
	republishHours := app.config.Modbus.RepublishInterval
	if republishHours <= 0 {
		republishHours = 4 // Default fallback
	}

	ticker := time.NewTicker(time.Duration(republishHours) * time.Hour)
	defer ticker.Stop()

	logger.LogInfo("📡 Started forced republish loop for energy sensors (every %d hours)", republishHours)

	for {
		select {
		case <-ctx.Done():
			logger.LogDebug("⏹️ Forced republish loop stopped")
			return
		case <-ticker.C:
			logger.LogInfo("🔄 Running forced republish for energy sensors (interval: %d hours)...", republishHours)
			app.forceRepublishEnergySensors(ctx)
		}
	}
}

// forceRepublishEnergySensors republishes energy sensor values to maintain Home Assistant availability
func (app *Application) forceRepublishEnergySensors(ctx context.Context) {
	energySensors := []string{
		"energy_total",
		"energy_imported",
		"energy_exported",
	}

	// Get republish interval from config, default to 4 hours if not set
	republishHours := app.config.Modbus.RepublishInterval
	if republishHours <= 0 {
		republishHours = 4 // Default fallback
	}

	// Only republish if it's been more than 75% of the republish interval since last publish
	threshold := time.Duration(float64(republishHours)*0.75) * time.Hour

	for _, sensorName := range energySensors {
		app.mu.Lock()
		lastPublish, exists := app.lastPublishTime[sensorName]
		app.mu.Unlock()

		if !exists || time.Since(lastPublish) > threshold {
			logger.LogInfo("🔄 Force republishing %s (last published: %v)", sensorName, lastPublish.Format("15:04:05"))

			// Execute strategy to get current value
			result, err := app.executor.GetResult(ctx, sensorName)
			if err != nil {
				logger.LogError("❌ Failed to force republish %s: %v", sensorName, err)
				continue
			}

			// Publish the result
			if pubErr := app.publisher.PublishSensorState(ctx, result); pubErr != nil {
				logger.LogError("⚠️ Error publishing forced sensor state for %s: %v", sensorName, pubErr)
			} else {
				app.updateLastPublishTime(sensorName)
			}
		}
	}
}

// DiagnosticMode runs diagnostic tests to help troubleshoot connectivity issues
func (app *Application) DiagnosticMode(ctx context.Context) error {
	logger.LogInfo("🔍 Starting diagnostic mode...")

	// Test 1: MQTT Connectivity
	logger.LogInfo("🔍 Test 1: MQTT Broker Connectivity")
	if !app.gateway.IsConnected() {
		logger.LogError("❌ Gateway is not connected to MQTT broker")
		return fmt.Errorf("gateway not connected to MQTT broker")
	}
	logger.LogInfo("✅ Gateway is connected to MQTT broker")
	// Skip publisher connection check for now - focus on gateway
	logger.LogInfo("✅ Publisher setup complete")

	// Test 2: Gateway Communication
	logger.LogInfo("🔍 Test 2: USR-DR164 Gateway Communication")
	if err := app.gateway.SendDiagnosticCommand(ctx); err != nil {
		logger.LogError("❌ Gateway communication failed: %v", err)
		logger.LogInfo("💡 Possible issues:")
		logger.LogInfo("   - USR-DR164 gateway is not connected to MQTT broker")
		logger.LogInfo("   - USR-DR164 gateway is not configured correctly")
		logger.LogInfo("   - Wrong MAC address in configuration (%s)", app.config.MQTT.Gateway.MAC)
		logger.LogInfo("   - Network connectivity issues")
		return fmt.Errorf("gateway communication failed: %w", err)
	}
	logger.LogInfo("✅ Gateway communication successful")

	// Test 3: Modbus Device Communication
	logger.LogInfo("🔍 Test 3: Modbus Device Communication")

	// Try to execute all strategies to read registers
	results, err := app.executor.ExecuteAll(ctx)
	if err != nil || len(results) == 0 {
		logger.LogError("❌ Modbus device communication failed: %v", err)
		logger.LogInfo("💡 Possible issues:")
		logger.LogInfo("   - Modbus device is not connected to USR-DR164 gateway")
		logger.LogInfo("   - Wrong slave ID in device configuration")
		logger.LogInfo("   - Modbus device is not powered on")
		logger.LogInfo("   - Physical connection issues (RS485 wiring)")
		logger.LogInfo("   - Wrong baud rate or communication parameters")
		if err != nil {
			return fmt.Errorf("modbus device communication failed: %w", err)
		}
		return fmt.Errorf("modbus device communication failed: no results")
	}

	// Log first result as example
	for _, result := range results {
		logger.LogInfo("✅ Modbus device communication successful - %s: %.2f %s", result.Name, result.Value, result.Unit)
		break // Just show one example
	}

	logger.LogInfo("🎉 All diagnostic tests passed!")
	return nil
}
