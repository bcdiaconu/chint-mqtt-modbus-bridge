package modbus

import (
	"context"
	"mqtt-modbus-bridge/internal/config"
)

// CommandExecutor responsible for executing commands
// Single Responsibility Principle
type CommandExecutor struct {
	gateway  Gateway
	config   *config.ModbusConfig
	commands map[string]ModbusCommand
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(gateway Gateway, cfg *config.ModbusConfig) *CommandExecutor {
	return &CommandExecutor{
		gateway:  gateway,
		config:   cfg,
		commands: make(map[string]ModbusCommand),
	}
}

// RegisterCommand registers a new command
// Dependency Inversion Principle - depends on interface, not implementation
func (e *CommandExecutor) RegisterCommand(name string, command ModbusCommand) {
	e.commands[name] = command
}

// ExecuteCommand executes a command using the specified command name
func (e *CommandExecutor) ExecuteCommand(ctx context.Context, commandName string) (*CommandResult, error) {
	command, exists := e.commands[commandName]
	if !exists {
		return nil, &CommandError{
			Strategy: commandName,
			Message:  "unknown command",
		}
	}

	// Check if command implements its own ExecuteCommand
	if selfExecuting, ok := command.(SelfExecutingCommand); ok {
		return selfExecuting.ExecuteCommand(ctx, e.gateway)
	}

	// Execute command using standard flow
	rawData, err := command.Execute(ctx, e.gateway)
	if err != nil {
		return nil, &CommandError{
			Strategy: commandName,
			Message:  "error executing command",
			Cause:    err,
		}
	}

	// Parse data
	value, err := command.ParseData(rawData)
	if err != nil {
		return nil, &CommandError{
			Strategy: commandName,
			Message:  "error parsing data",
			Cause:    err,
		}
	}

	return &CommandResult{
		Strategy:    commandName,
		Name:        command.GetName(),
		Value:       value,
		Unit:        command.GetUnit(),
		Topic:       command.GetTopic(),
		DeviceClass: command.GetDeviceClass(),
		StateClass:  command.GetStateClass(),
		RawData:     rawData,
	}, nil
}

// GetAvailableCommands returns the list of available commands
func (e *CommandExecutor) GetAvailableCommands() []string {
	commands := make([]string, 0, len(e.commands))
	for name := range e.commands {
		commands = append(commands, name)
	}
	return commands
}
