package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/internal/config"
)

// BaseStrategy base implementation for all strategies
// Template Method Pattern combined with Strategy Pattern
type BaseStrategy struct {
	register config.Register
	address  uint16
	slaveID  uint8
}

// NewBaseStrategy creates a base strategy
func NewBaseStrategy(register config.Register, slaveID uint8) *BaseStrategy {
	return &BaseStrategy{
		register: register,
		address:  register.Address,
		slaveID:  slaveID,
	}
}

// Execute common implementation for executing Modbus command
func (s *BaseStrategy) Execute(ctx context.Context, gateway Gateway) ([]byte, error) {
	// Function Code 03 - Read Holding Registers
	// Read 2 registers (4 bytes) for float32
	err := gateway.SendCommand(ctx, s.slaveID, 0x03, s.address, 2)
	if err != nil {
		return nil, fmt.Errorf("error sending command: %w", err)
	}

	// Wait for response
	response, err := gateway.WaitForResponse(ctx, 5)
	if err != nil {
		return nil, fmt.Errorf("error waiting for response: %w", err)
	}

	return response, nil
}

// GetUnit returns the unit of measurement
func (s *BaseStrategy) GetUnit() string {
	return s.register.Unit
}

// GetTopic returns the Home Assistant topic
func (s *BaseStrategy) GetTopic() string {
	return s.register.HATopic
}

// GetName returns the register name
func (s *BaseStrategy) GetName() string {
	return s.register.Name
}

// GetDeviceClass returns the device class
func (s *BaseStrategy) GetDeviceClass() string {
	return s.register.DeviceClass
}

// GetStateClass returns the state class
func (s *BaseStrategy) GetStateClass() string {
	return s.register.StateClass
}

// ParseData default implementation for parsing data
func (s *BaseStrategy) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	// Default implementation for float32
	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	return float64(value), nil
}

// VoltageStrategy strategy for reading voltage
type VoltageStrategy struct {
	*BaseStrategy
}

// NewVoltageStrategy creates a voltage strategy
func NewVoltageStrategy(register config.Register, slaveID uint8) *VoltageStrategy {
	return &VoltageStrategy{
		BaseStrategy: NewBaseStrategy(register, slaveID),
	}
}

// ParseData parses data for voltage
func (s *VoltageStrategy) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	// Convert bytes to float32 (IEEE 754)
	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	return float64(value), nil
}

// FrequencyStrategy strategy for reading frequency
type FrequencyStrategy struct {
	*BaseStrategy
}

func NewFrequencyStrategy(register config.Register, slaveID uint8) *FrequencyStrategy {
	return &FrequencyStrategy{
		BaseStrategy: NewBaseStrategy(register, slaveID),
	}
}

func (s *FrequencyStrategy) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	return float64(value), nil
}

// CurrentStrategy strategy for reading current
type CurrentStrategy struct {
	*BaseStrategy
}

func NewCurrentStrategy(register config.Register, slaveID uint8) *CurrentStrategy {
	return &CurrentStrategy{
		BaseStrategy: NewBaseStrategy(register, slaveID),
	}
}

func (s *CurrentStrategy) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	return float64(value), nil
}

// EnergyStrategy strategy for reading energy
type EnergyStrategy struct {
	*BaseStrategy
}

func NewEnergyStrategy(register config.Register, slaveID uint8) *EnergyStrategy {
	return &EnergyStrategy{
		BaseStrategy: NewBaseStrategy(register, slaveID),
	}
}

func (s *EnergyStrategy) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	return float64(value), nil
}

// PowerStrategy strategy for reading power
type PowerStrategy struct {
	*BaseStrategy
}

func NewPowerStrategy(register config.Register, slaveID uint8) *PowerStrategy {
	return &PowerStrategy{
		BaseStrategy: NewBaseStrategy(register, slaveID),
	}
}

func (s *PowerStrategy) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	// Convert from KW/KVA to W/VA (multiply by 1000)
	convertedValue := float64(value) * 1000.0

	return convertedValue, nil
}

// PowerFactorStrategy strategy for reading power factor
type PowerFactorStrategy struct {
	*BaseStrategy
}

func NewPowerFactorStrategy(register config.Register, slaveID uint8) *PowerFactorStrategy {
	return &PowerFactorStrategy{
		BaseStrategy: NewBaseStrategy(register, slaveID),
	}
}

func (s *PowerFactorStrategy) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	return float64(value), nil
}

// ReactivePowerStrategy strategy for calculating reactive power
// Reactive power is calculated as: Q = sqrt(S² - P²) where S = apparent power, P = active power
type ReactivePowerStrategy struct {
	*BaseStrategy
	activePowerName   string
	apparentPowerName string
	executor          *CommandExecutor
}

// NewReactivePowerStrategy creates a new strategy for reactive power
func NewReactivePowerStrategy(register config.Register, slaveID uint8, activePowerName, apparentPowerName string) *ReactivePowerStrategy {
	return &ReactivePowerStrategy{
		BaseStrategy:      NewBaseStrategy(register, slaveID),
		activePowerName:   activePowerName,
		apparentPowerName: apparentPowerName,
	}
}

// SetExecutor sets the executor to allow reading other values
func (s *ReactivePowerStrategy) SetExecutor(executor *CommandExecutor) {
	s.executor = executor
}

// ExecuteCommand implements reading reactive power by calculation
func (s *ReactivePowerStrategy) ExecuteCommand(ctx context.Context, gateway Gateway) (*CommandResult, error) {
	if s.executor == nil {
		return nil, fmt.Errorf("executor not set for reactive power calculation")
	}

	// Read active power
	activePowerResult, err := s.executor.ExecuteCommand(ctx, s.activePowerName)
	if err != nil {
		return nil, fmt.Errorf("error reading active power: %w", err)
	}

	// Read apparent power
	apparentPowerResult, err := s.executor.ExecuteCommand(ctx, s.apparentPowerName)
	if err != nil {
		return nil, fmt.Errorf("error reading apparent power: %w", err)
	}

	// Calculate reactive power: Q = sqrt(S² - P²)
	P := activePowerResult.Value   // Active power in W
	S := apparentPowerResult.Value // Apparent power in VA

	// Validation to avoid sqrt of negative number
	if S*S < P*P {
		// If apparent is less than active (theoretically impossible, but may occur due to measurement errors)
		// Set Q = 0
		Q := 0.0
		return &CommandResult{
			Strategy:    s.register.Name,
			Name:        s.register.Name,
			Value:       Q,
			Unit:        s.register.Unit,
			Topic:       s.register.HATopic,
			DeviceClass: s.register.DeviceClass,
			StateClass:  s.register.StateClass,
		}, nil
	}

	Q := math.Sqrt(S*S - P*P) // Reactive power in VAR

	return &CommandResult{
		Strategy:    s.register.Name,
		Name:        s.register.Name,
		Value:       Q,
		Unit:        s.register.Unit,
		Topic:       s.register.HATopic,
		DeviceClass: s.register.DeviceClass,
		StateClass:  s.register.StateClass,
	}, nil
}

// ParseData is not used for this strategy because we calculate from other values
func (s *ReactivePowerStrategy) ParseData(rawData []byte) (float64, error) {
	return 0, fmt.Errorf("ParseData not used for reactive power - use ExecuteCommand instead")
}

// StrategyFactory factory for creating strategies
// Factory Pattern for creating strategies based on register type
type StrategyFactory struct {
	slaveID uint8
}

// NewStrategyFactory creates a new factory with slave ID
func NewStrategyFactory(slaveID uint8) *StrategyFactory {
	return &StrategyFactory{
		slaveID: slaveID,
	}
}

// CreateStrategy creates the appropriate strategy based on device_class
func (f *StrategyFactory) CreateStrategy(register config.Register) (CommandStrategy, error) {
	switch register.DeviceClass {
	case "voltage":
		return NewVoltageStrategy(register, f.slaveID), nil
	case "frequency":
		return NewFrequencyStrategy(register, f.slaveID), nil
	case "current":
		return NewCurrentStrategy(register, f.slaveID), nil
	case "energy":
		return NewEnergyStrategy(register, f.slaveID), nil
	case "power", "apparent_power":
		return NewPowerStrategy(register, f.slaveID), nil
	case "power_factor":
		return NewPowerFactorStrategy(register, f.slaveID), nil
	case "reactive_power":
		// For reactive power, we need the register names for active and apparent power
		return NewReactivePowerStrategy(register, f.slaveID, "power_active", "power_apparent"), nil
	default:
		// Default strategy for unknown registers
		return NewBaseStrategy(register, f.slaveID), nil
	}
}
