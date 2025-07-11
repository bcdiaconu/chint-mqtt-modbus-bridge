package modbus

import (
	"context"
	"mqtt-modbus-bridge/internal/config"
)

// CommandStrategy interface for Strategy Pattern - Open/Closed Principle
// Each Modbus command implements this interface
type CommandStrategy interface {
	// Execute executes the Modbus command and returns the raw response
	Execute(ctx context.Context, gateway Gateway) ([]byte, error)

	// ParseData interprets the received data and returns the value
	ParseData(rawData []byte) (float64, error)

	// GetUnit returns the unit of measurement
	GetUnit() string

	// GetTopic returns the MQTT topic for Home Assistant
	GetTopic() string

	// GetName returns the command name
	GetName() string

	// GetDeviceClass returns the device class for Home Assistant
	GetDeviceClass() string

	// GetStateClass returns the state class for Home Assistant
	GetStateClass() string
}

// SelfExecutingStrategy interface for strategies with their own ExecuteCommand implementation
type SelfExecutingStrategy interface {
	CommandStrategy
	ExecuteCommand(ctx context.Context, gateway Gateway) (*CommandResult, error)
}

// Gateway interface for communication with the USR-DR164 gateway
// Interface Segregation Principle - specific interface for gateway
type Gateway interface {
	// SendCommand sends a Modbus command through MQTT
	SendCommand(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16) error

	// WaitForResponse waits for response from gateway
	WaitForResponse(ctx context.Context, timeout int) ([]byte, error)

	// IsConnected checks if gateway is connected
	IsConnected() bool
}

// CommandExecutor responsible for executing commands
// Single Responsibility Principle
type CommandExecutor struct {
	gateway    Gateway
	config     *config.ModbusConfig
	strategies map[string]CommandStrategy
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(gateway Gateway, cfg *config.ModbusConfig) *CommandExecutor {
	return &CommandExecutor{
		gateway:    gateway,
		config:     cfg,
		strategies: make(map[string]CommandStrategy),
	}
}

// RegisterStrategy registers a new strategy
// Dependency Inversion Principle - depends on interface, not implementation
func (e *CommandExecutor) RegisterStrategy(name string, strategy CommandStrategy) {
	e.strategies[name] = strategy
}

// ExecuteCommand executes a command using the specified strategy
func (e *CommandExecutor) ExecuteCommand(ctx context.Context, strategyName string) (*CommandResult, error) {
	strategy, exists := e.strategies[strategyName]
	if !exists {
		return nil, &CommandError{
			Strategy: strategyName,
			Message:  "unknown strategy",
		}
	}

	// Check if strategy implements its own ExecuteCommand
	if selfExecuting, ok := strategy.(SelfExecutingStrategy); ok {
		return selfExecuting.ExecuteCommand(ctx, e.gateway)
	}

	// Execute command using standard flow
	rawData, err := strategy.Execute(ctx, e.gateway)
	if err != nil {
		return nil, &CommandError{
			Strategy: strategyName,
			Message:  "error executing command",
			Cause:    err,
		}
	}

	// Parse data
	value, err := strategy.ParseData(rawData)
	if err != nil {
		return nil, &CommandError{
			Strategy: strategyName,
			Message:  "error parsing data",
			Cause:    err,
		}
	}

	return &CommandResult{
		Strategy:    strategyName,
		Name:        strategy.GetName(),
		Value:       value,
		Unit:        strategy.GetUnit(),
		Topic:       strategy.GetTopic(),
		DeviceClass: strategy.GetDeviceClass(),
		StateClass:  strategy.GetStateClass(),
		RawData:     rawData,
	}, nil
}

// GetAvailableStrategies returns the list of available strategies
func (e *CommandExecutor) GetAvailableStrategies() []string {
	strategies := make([]string, 0, len(e.strategies))
	for name := range e.strategies {
		strategies = append(strategies, name)
	}
	return strategies
}

// CommandResult result of executing a command
type CommandResult struct {
	Strategy    string  `json:"strategy"`
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Topic       string  `json:"topic"`
	DeviceClass string  `json:"device_class"`
	StateClass  string  `json:"state_class"`
	RawData     []byte  `json:"raw_data"`
}

// CommandError custom error for commands
type CommandError struct {
	Strategy string
	Message  string
	Cause    error
}

func (e *CommandError) Error() string {
	if e.Cause != nil {
		return e.Strategy + ": " + e.Message + " - " + e.Cause.Error()
	}
	return e.Strategy + ": " + e.Message
}

func (e *CommandError) Unwrap() error {
	return e.Cause
}
