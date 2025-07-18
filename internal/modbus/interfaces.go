package modbus

import "context"

// ModbusCommand interface for Strategy Pattern - Open/Closed Principle
// Each Modbus command implements this interface
type ModbusCommand interface {
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

// SelfExecutingCommand interface for commands with their own ExecuteCommand implementation
type SelfExecutingCommand interface {
	ModbusCommand
	ExecuteCommand(ctx context.Context, gateway Gateway) (*CommandResult, error)
}

// Gateway interface for communication with the USR-DR164 gateway
// Interface Segregation Principle - specific interface for gateway
type Gateway interface {
	// SendCommand sends a Modbus command through MQTT
	SendCommand(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16) error

	// WaitForResponse waits for response from gateway
	WaitForResponse(ctx context.Context, timeout int) ([]byte, error)

	// SendCommandAndWaitForResponse sends a command and waits for response atomically
	SendCommandAndWaitForResponse(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16, timeoutSeconds int) ([]byte, error)

	// IsConnected checks if gateway is connected
	IsConnected() bool
}
