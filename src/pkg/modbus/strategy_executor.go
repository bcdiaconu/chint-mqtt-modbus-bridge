package modbus

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/gateway"
	"mqtt-modbus-bridge/pkg/logger"
	"time"
)

// StrategyExecutor manages and executes all Modbus strategies
type StrategyExecutor struct {
	gateway          gateway.Gateway
	cache            *ValueCache
	singleStrategies map[string]*SingleRegisterStrategy
	groupStrategies  map[string]*GroupRegisterStrategy
	calcStrategies   map[string]*CalculatedRegisterStrategy
	executionOrder   []string // Order: groups first, then calculated
}

// NewStrategyExecutor creates a new strategy executor
func NewStrategyExecutor(gw gateway.Gateway) *StrategyExecutor {
	return &StrategyExecutor{
		gateway:          gw,
		cache:            NewValueCache(5 * time.Minute),
		singleStrategies: make(map[string]*SingleRegisterStrategy),
		groupStrategies:  make(map[string]*GroupRegisterStrategy),
		calcStrategies:   make(map[string]*CalculatedRegisterStrategy),
		executionOrder:   []string{},
	}
}

// RegisterFromDevices registers all strategies from device configuration
func (e *StrategyExecutor) RegisterFromDevices(devices map[string]config.Device) error {
	for deviceKey, device := range devices {
		if !device.Metadata.Enabled {
			logger.LogDebug("Skipping disabled device: %s", deviceKey)
			continue
		}

		slaveID := device.RTU.SlaveID

		// Register group strategies first (for efficient reading)
		for groupKey, group := range device.Modbus.RegisterGroups {
			if !group.Enabled {
				continue
			}

			// Collect all registers for this group
			var registers []RegisterWithKey
			for _, groupReg := range group.Registers {
				// Calculate actual address
				offsetInRegisters := groupReg.Offset / 2
				address := group.StartAddress + uint16(offsetInRegisters)

				// Build full register config
				scaleFactor := groupReg.ScaleFactor
				if scaleFactor == 0 {
					scaleFactor = 1.0
				}

				register := config.Register{
					Name:        groupReg.Name,
					Address:     address,
					Unit:        groupReg.Unit,
					ScaleFactor: scaleFactor,
					ApplyAbs:    groupReg.ApplyAbs, // Copy apply_abs flag
					DeviceClass: groupReg.DeviceClass,
					StateClass:  groupReg.StateClass,
					HATopic:     constructHATopic(deviceKey, groupReg.Key, groupReg.DeviceClass),
				}

				regKey := fmt.Sprintf("%s_%s", deviceKey, groupReg.Key)
				registers = append(registers, RegisterWithKey{
					Key:      regKey,
					Register: register,
				})
			}

			// Create group strategy
			fullGroupKey := fmt.Sprintf("%s_%s", deviceKey, groupKey)
			strategy := NewGroupRegisterStrategy(
				fullGroupKey,
				group,
				registers,
				slaveID,
				e.gateway,
				e.cache,
			)

			e.groupStrategies[fullGroupKey] = strategy
			e.executionOrder = append(e.executionOrder, fullGroupKey)

			logger.LogInfo("✅ Registered group strategy: %s (%d registers)", fullGroupKey, len(registers))
		}

		// Register calculated value strategies (executed after groups)
		for _, calc := range device.CalculatedValues {
			scaleFactor := calc.ScaleFactor
			if scaleFactor == 0 {
				scaleFactor = 1.0
			}

			register := config.Register{
				Name:        calc.Name,
				Unit:        calc.Unit,
				Formula:     calc.Formula,
				ScaleFactor: scaleFactor,
				DeviceClass: calc.DeviceClass,
				StateClass:  calc.StateClass,
				HATopic:     constructHATopic(deviceKey, calc.Key, calc.DeviceClass),
			}

			calcKey := fmt.Sprintf("%s_%s", deviceKey, calc.Key)
			strategy := NewCalculatedRegisterStrategy(
				calcKey,
				register,
				deviceKey, // Device prefix for variable resolution
				e.cache,
			)

			e.calcStrategies[calcKey] = strategy
			e.executionOrder = append(e.executionOrder, calcKey)

			logger.LogInfo("✅ Registered calculated strategy: %s (formula: %s)", calcKey, calc.Formula)
		}
	}

	return nil
}

// ExecuteAll executes all strategies in order (groups first, then calculated)
func (e *StrategyExecutor) ExecuteAll(ctx context.Context) (map[string]*CommandResult, error) {
	results := make(map[string]*CommandResult)

	for _, key := range e.executionOrder {
		// Check if it's a group strategy
		if groupStrategy, exists := e.groupStrategies[key]; exists {
			groupResults, err := groupStrategy.Execute(ctx)
			if err != nil {
				logger.LogError("Failed to execute group strategy '%s': %v", key, err)
				continue
			}

			// Merge group results and log each register value
			for regKey, result := range groupResults {
				results[regKey] = result
				logger.LogDebug("  📊 [Group '%s'] %s = %.2f %s (device_class: %s)",
					key, result.Name, result.Value, result.Unit, result.DeviceClass)
			}

			logger.LogDebug("✅ Group '%s' executed: %d registers", key, len(groupResults))
			continue
		}

		// Check if it's a calculated strategy
		if calcStrategy, exists := e.calcStrategies[key]; exists {
			result, err := calcStrategy.Execute(ctx)
			if err != nil {
				logger.LogError("Failed to execute calculated strategy '%s': %v", key, err)
				continue
			}

			results[key] = result
			logger.LogDebug("  🧮 [Calculated '%s'] %s = %.2f %s (device_class: %s)",
				key, result.Name, result.Value, result.Unit, result.DeviceClass)
			continue
		}
	}

	return results, nil
}

// GetResult fetches a specific result (from cache or executes if needed)
func (e *StrategyExecutor) GetResult(ctx context.Context, key string) (*CommandResult, error) {
	// Try cache first
	if cached, found := e.cache.Get(key); found {
		return cached, nil
	}

	// Try to execute specific strategy
	if groupStrategy, exists := e.groupStrategies[key]; exists {
		results, err := groupStrategy.Execute(ctx)
		if err != nil {
			return nil, err
		}
		// Return the first result (group strategies return multiple)
		for _, result := range results {
			return result, nil
		}
	}

	if calcStrategy, exists := e.calcStrategies[key]; exists {
		return calcStrategy.Execute(ctx)
	}

	return nil, fmt.Errorf("strategy not found for key '%s'", key)
}

// GetAllStrategies returns all individual register strategies (for discovery)
// Note: Group strategies contain multiple registers, so we return their individual registers
func (e *StrategyExecutor) GetAllStrategies() map[string]interface{} {
	allStrategies := make(map[string]interface{})

	// Add single strategies
	for key, strategy := range e.singleStrategies {
		allStrategies[key] = strategy
	}

	// Add calculated strategies
	for key, strategy := range e.calcStrategies {
		allStrategies[key] = strategy
	}

	// Add group strategies as-is (they'll be expanded in the caller if needed)
	for key, strategy := range e.groupStrategies {
		allStrategies[key] = strategy
	}

	return allStrategies
}

// constructHATopic builds Home Assistant MQTT topic (including /state suffix for publishing)
func constructHATopic(deviceID, sensorKey, deviceClass string) string {
	return fmt.Sprintf("homeassistant/sensor/%s/%s_%s/state", deviceID, deviceID, sensorKey)
}
