package errors

import (
	"fmt"
)

// ErrorSeverity defines the severity level of an error
type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// String returns the string representation of the severity
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// BridgeError is the base error type for all bridge errors
type BridgeError struct {
	Op       string        // Operation that failed
	Err      error         // Underlying error
	Severity ErrorSeverity // Error severity
	Code     int           // Diagnostic code for MQTT
}

// Error implements the error interface
func (e *BridgeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Severity, e.Op, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Severity, e.Op)
}

// Unwrap returns the underlying error
func (e *BridgeError) Unwrap() error {
	return e.Err
}

// GatewayError represents errors from the MQTT gateway
type GatewayError struct {
	BridgeError
	GatewayMAC string
	Topic      string
}

// NewGatewayError creates a new gateway error
func NewGatewayError(op string, err error, gatewayMAC string) *GatewayError {
	return &GatewayError{
		BridgeError: BridgeError{
			Op:       op,
			Err:      err,
			Severity: SeverityError,
			Code:     2, // Gateway error diagnostic code
		},
		GatewayMAC: gatewayMAC,
	}
}

// Error implements the error interface
func (e *GatewayError) Error() string {
	if e.Topic != "" {
		return fmt.Sprintf("[%s] Gateway %s (%s): %s: %v",
			e.Severity, e.GatewayMAC, e.Topic, e.Op, e.Err)
	}
	return fmt.Sprintf("[%s] Gateway %s: %s: %v",
		e.Severity, e.GatewayMAC, e.Op, e.Err)
}

// ModbusError represents errors from Modbus operations
type ModbusError struct {
	BridgeError
	SlaveID      uint8
	FunctionCode uint8
	Address      uint16
	DeviceID     string
}

// NewModbusError creates a new Modbus error
func NewModbusError(op string, err error, slaveID uint8, deviceID string) *ModbusError {
	return &ModbusError{
		BridgeError: BridgeError{
			Op:       op,
			Err:      err,
			Severity: SeverityError,
			Code:     3, // Modbus error diagnostic code
		},
		SlaveID:  slaveID,
		DeviceID: deviceID,
	}
}

// Error implements the error interface
func (e *ModbusError) Error() string {
	if e.DeviceID != "" {
		return fmt.Sprintf("[%s] Modbus device '%s' (slave %d): %s: %v",
			e.Severity, e.DeviceID, e.SlaveID, e.Op, e.Err)
	}
	return fmt.Sprintf("[%s] Modbus slave %d: %s: %v",
		e.Severity, e.SlaveID, e.Op, e.Err)
}

// MQTTError represents errors from MQTT operations
type MQTTError struct {
	BridgeError
	Broker string
	Topic  string
	QoS    byte
}

// NewMQTTError creates a new MQTT error
func NewMQTTError(op string, err error, broker string) *MQTTError {
	return &MQTTError{
		BridgeError: BridgeError{
			Op:       op,
			Err:      err,
			Severity: SeverityError,
			Code:     4, // MQTT error diagnostic code
		},
		Broker: broker,
	}
}

// Error implements the error interface
func (e *MQTTError) Error() string {
	if e.Topic != "" {
		return fmt.Sprintf("[%s] MQTT broker '%s' (topic: %s): %s: %v",
			e.Severity, e.Broker, e.Topic, e.Op, e.Err)
	}
	return fmt.Sprintf("[%s] MQTT broker '%s': %s: %v",
		e.Severity, e.Broker, e.Op, e.Err)
}

// ConfigError represents configuration errors
type ConfigError struct {
	BridgeError
	Field string
	Value interface{}
}

// NewConfigError creates a new configuration error
func NewConfigError(op string, err error, field string) *ConfigError {
	return &ConfigError{
		BridgeError: BridgeError{
			Op:       op,
			Err:      err,
			Severity: SeverityCritical, // Config errors are critical
			Code:     1,                // Config error diagnostic code
		},
		Field: field,
	}
}

// Error implements the error interface
func (e *ConfigError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("[%s] Configuration field '%s': %s: %v",
			e.Severity, e.Field, e.Op, e.Err)
	}
	return fmt.Sprintf("[%s] Configuration: %s: %v",
		e.Severity, e.Op, e.Err)
}

// ValidationError represents validation errors
type ValidationError struct {
	BridgeError
	Field    string
	Expected interface{}
	Actual   interface{}
}

// NewValidationError creates a new validation error
func NewValidationError(field string, expected, actual interface{}) *ValidationError {
	return &ValidationError{
		BridgeError: BridgeError{
			Op:       "validation",
			Err:      fmt.Errorf("validation failed"),
			Severity: SeverityWarning,
			Code:     5, // Validation error diagnostic code
		},
		Field:    field,
		Expected: expected,
		Actual:   actual,
	}
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("[%s] Field '%s': expected %v, got %v",
		e.Severity, e.Field, e.Expected, e.Actual)
}
