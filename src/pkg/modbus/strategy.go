package modbus

import (
	"context"
	"mqtt-modbus-bridge/pkg/config"
	"mqtt-modbus-bridge/pkg/gateway"
)

// ModbusStrategy defines the interface for all Modbus read/calculation strategies
type ModbusStrategy interface {
	// Execute performs the strategy (read from Modbus or calculate)
	Execute(ctx context.Context) (*CommandResult, error)

	// GetKey returns the unique identifier for this strategy
	GetKey() string

	// GetRegisterInfo returns the register configuration
	GetRegisterInfo() config.Register
}

// BaseStrategy provides common functionality for all strategies
type BaseStrategy struct {
	key      string
	register config.Register
	slaveID  uint8
	gateway  gateway.Gateway
	cache    *ValueCache
}

// NewBaseStrategy creates a new base strategy
func NewBaseStrategy(key string, register config.Register, slaveID uint8, gw gateway.Gateway, cache *ValueCache) *BaseStrategy {
	return &BaseStrategy{
		key:      key,
		register: register,
		slaveID:  slaveID,
		gateway:  gw,
		cache:    cache,
	}
}

// GetKey returns the strategy key
func (b *BaseStrategy) GetKey() string {
	return b.key
}

// GetRegisterInfo returns the register configuration
func (b *BaseStrategy) GetRegisterInfo() config.Register {
	return b.register
}
