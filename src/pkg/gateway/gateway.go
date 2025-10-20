package gateway

import "context"

// Gateway interface for communication with the USR-DR164 gateway
// Interface Segregation Principle - specific interface for gateway operations
type Gateway interface {
	// Connect establishes connection to the gateway
	Connect(ctx context.Context) error

	// Disconnect closes connection to the gateway
	Disconnect()

	// SendCommand sends a Modbus command through MQTT
	SendCommand(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16) error

	// WaitForResponse waits for response from gateway
	WaitForResponse(ctx context.Context, timeout int) ([]byte, error)

	// SendCommandAndWaitForResponse sends a command and waits for response atomically
	SendCommandAndWaitForResponse(ctx context.Context, slaveID uint8, functionCode uint8, address uint16, count uint16, timeoutSeconds int) ([]byte, error)

	// SendDiagnosticCommand sends a diagnostic command to test gateway connectivity
	SendDiagnosticCommand(ctx context.Context) error

	// IsConnected checks if gateway is connected
	IsConnected() bool
}
