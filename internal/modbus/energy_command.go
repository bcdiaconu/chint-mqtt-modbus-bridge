package modbus

import (
	"encoding/binary"
	"fmt"
	"math"
	"mqtt-modbus-bridge/internal/config"
)

// EnergyCommand Command for reading energy
type EnergyCommand struct {
	*BaseCommand
	lastValidValue float64 // Store last valid value to detect anomalies
}

// NewEnergyCommand creates an energy Command
func NewEnergyCommand(register config.Register, slaveID uint8) *EnergyCommand {
	return &EnergyCommand{
		BaseCommand: NewBaseCommand(register, slaveID),
	}
}

// ParseData parses data for energy with spike detection and validation
func (c *EnergyCommand) ParseData(rawData []byte) (float64, error) {
	if len(rawData) < 4 {
		return 0, fmt.Errorf("not enough data for float32: %d bytes", len(rawData))
	}

	bits := binary.BigEndian.Uint32(rawData[:4])
	value := math.Float32frombits(bits)

	// Convert to float64
	currentValue := float64(value)

	// Validate that the value is reasonable (not NaN, not Inf)
	if math.IsNaN(currentValue) || math.IsInf(currentValue, 0) {
		return 0, fmt.Errorf("invalid energy value: NaN or Inf")
	}

	// For energy meters with state_class "total_increasing", values should not decrease significantly
	// Allow small decreases (meter resets) but prevent large spikes
	if c.lastValidValue > 0 {
		// Calculate the change from last valid value
		change := currentValue - c.lastValidValue
		changeRatio := math.Abs(change) / c.lastValidValue

		// If the change is more than 1000% (10x) of the previous value, it's likely a spike
		// Also reject values that are suspiciously small (like 0.03) if previous was much larger
		if changeRatio > 10.0 {
			return 0, fmt.Errorf("energy value spike detected: current=%.3f, last=%.3f, change=%.1f%%",
				currentValue, c.lastValidValue, changeRatio*100)
		}

		// For exported energy, reject values that decrease more than 1 kWh (meter might reset)
		// but reject dramatic decreases that suggest bad readings
		if change < -1.0 && changeRatio > 0.5 {
			return 0, fmt.Errorf("energy value decreased significantly: current=%.3f, last=%.3f",
				currentValue, c.lastValidValue)
		}
	}

	// Store this as the last valid value
	c.lastValidValue = currentValue

	return currentValue, nil
}
