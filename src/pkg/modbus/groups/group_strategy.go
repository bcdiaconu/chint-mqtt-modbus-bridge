package groups

import (
	"context"
	"mqtt-modbus-bridge/pkg/modbus"
)

// GroupStrategy defines the interface for grouped Modbus queries (Strategy Pattern)
type GroupStrategy interface {
	// Execute runs the grouped Modbus query and returns raw data
	Execute(ctx context.Context, gateway modbus.Gateway, slaveID uint8) ([]byte, error)

	// ParseResults interprets the received data and returns a map of values
	ParseResults(rawData []byte) (map[string]float64, error)

	// GetNames returns the names of all registers in the group
	GetNames() []string
}
