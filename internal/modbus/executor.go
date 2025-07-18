package modbus

import (
	"context"
	"mqtt-modbus-bridge/internal/config"
	"sync"
	"time"
)

// CachedResult stores a command result with timestamp for cache validation
type CachedResult struct {
	Result    *CommandResult
	Timestamp time.Time
}

// CommandExecutor responsible for executing commands
// Single Responsibility Principle
type CommandExecutor struct {
	gateway      Gateway
	config       *config.ModbusConfig
	commands     map[string]ModbusCommand
	cache        map[string]*CachedResult // Cache for last valid results
	cacheMutex   sync.RWMutex             // Protect cache access
	cacheTimeout time.Duration            // How long to keep cached values
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(gateway Gateway, cfg *config.ModbusConfig) *CommandExecutor {
	return &CommandExecutor{
		gateway:      gateway,
		config:       cfg,
		commands:     make(map[string]ModbusCommand),
		cache:        make(map[string]*CachedResult),
		cacheTimeout: 5 * time.Minute, // Cache values for 5 minutes
	}
}

// RegisterCommand registers a new command
// Dependency Inversion Principle - depends on interface, not implementation
func (e *CommandExecutor) RegisterCommand(name string, command ModbusCommand) {
	e.commands[name] = command
}

// ExecuteCommand executes a command using the specified command name
// Returns cached result if current execution fails and cache is valid
func (e *CommandExecutor) ExecuteCommand(ctx context.Context, commandName string) (*CommandResult, error) {
	command, exists := e.commands[commandName]
	if !exists {
		return nil, &CommandError{
			Strategy: commandName,
			Message:  "unknown command",
		}
	}

	// Try to execute the command
	var result *CommandResult
	var execErr error

	// Check if command implements its own ExecuteCommand
	if selfExecuting, ok := command.(SelfExecutingCommand); ok {
		result, execErr = selfExecuting.ExecuteCommand(ctx, e.gateway)
	} else {
		// Execute command using standard flow
		rawData, err := command.Execute(ctx, e.gateway)
		if err != nil {
			execErr = &CommandError{
				Strategy: commandName,
				Message:  "error executing command",
				Cause:    err,
			}
		} else {
			// Parse data
			value, err := command.ParseData(rawData)
			if err != nil {
				execErr = &CommandError{
					Strategy: commandName,
					Message:  "error parsing data",
					Cause:    err,
				}
			} else {
				result = &CommandResult{
					Strategy:    commandName,
					Name:        command.GetName(),
					Value:       value,
					Unit:        command.GetUnit(),
					Topic:       command.GetTopic(),
					DeviceClass: command.GetDeviceClass(),
					StateClass:  command.GetStateClass(),
					RawData:     rawData,
				}
			}
		}
	}

	// If execution was successful, cache the result and return it
	if execErr == nil && result != nil {
		e.setCachedResult(commandName, result)
		return result, nil
	}

	// Execution failed - try to return cached result if available and valid
	if cachedResult := e.getCachedResult(commandName); cachedResult != nil {
		// Return cached result but still report the error for logging purposes
		return cachedResult, &CommandError{
			Strategy: commandName,
			Message:  "using cached value due to execution error",
			Cause:    execErr,
		}
	}

	// No valid cache available, return the original error
	return nil, execErr
}

// GetAvailableCommands returns the list of available commands
func (e *CommandExecutor) GetAvailableCommands() []string {
	commands := make([]string, 0, len(e.commands))
	for name := range e.commands {
		commands = append(commands, name)
	}
	return commands
}

// setCachedResult stores a successful command result in cache
func (e *CommandExecutor) setCachedResult(commandName string, result *CommandResult) {
	e.cacheMutex.Lock()
	defer e.cacheMutex.Unlock()

	// Create a copy of the result to avoid any concurrent modification issues
	cachedResult := &CommandResult{
		Strategy:    result.Strategy,
		Name:        result.Name,
		Value:       result.Value,
		Unit:        result.Unit,
		Topic:       result.Topic,
		DeviceClass: result.DeviceClass,
		StateClass:  result.StateClass,
		RawData:     nil, // Don't cache raw data to save memory
	}

	e.cache[commandName] = &CachedResult{
		Result:    cachedResult,
		Timestamp: time.Now(),
	}
}

// getCachedResult retrieves a cached result if it's still valid
func (e *CommandExecutor) getCachedResult(commandName string) *CommandResult {
	e.cacheMutex.RLock()
	defer e.cacheMutex.RUnlock()

	cached, exists := e.cache[commandName]
	if !exists {
		return nil
	}

	// Check if cache is still valid
	if time.Since(cached.Timestamp) > e.cacheTimeout {
		// Cache expired, remove it
		delete(e.cache, commandName)
		return nil
	}

	return cached.Result
}

// ClearCache removes all cached results (useful for testing or manual reset)
func (e *CommandExecutor) ClearCache() {
	e.cacheMutex.Lock()
	defer e.cacheMutex.Unlock()
	e.cache = make(map[string]*CachedResult)
}
