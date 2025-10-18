package gateway

import "context"

// Gateway interface for communication with the USR-DR164 gateway
// Interface Segregation Principle - specific interface for gateway operations
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
