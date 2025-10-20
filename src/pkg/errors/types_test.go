package errors

import (
	"errors"
	"fmt"
	"testing"
)

// TestModbusErrorCreation tests creating ModbusError
func TestModbusErrorCreation(t *testing.T) {
	baseErr := fmt.Errorf("timeout reading register")
	modbusErr := NewModbusError("read_register", baseErr, 1, "energy_meter")
	modbusErr.FunctionCode = 0x03
	modbusErr.Address = 0x2000

	if modbusErr.SlaveID != 1 {
		t.Errorf("Expected SlaveID 1, got %d", modbusErr.SlaveID)
	}
	if modbusErr.DeviceID != "energy_meter" {
		t.Errorf("Expected DeviceID 'energy_meter', got '%s'", modbusErr.DeviceID)
	}
	if modbusErr.FunctionCode != 0x03 {
		t.Errorf("Expected FunctionCode 0x03, got 0x%02X", modbusErr.FunctionCode)
	}
	if modbusErr.Address != 0x2000 {
		t.Errorf("Expected Address 0x2000, got 0x%04X", modbusErr.Address)
	}

	errMsg := modbusErr.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}
	t.Logf("ModbusError message: %s", errMsg)
}

// TestMQTTErrorCreation tests creating MQTTError
func TestMQTTErrorCreation(t *testing.T) {
	baseErr := fmt.Errorf("connection timeout")
	mqttErr := NewMQTTError("connect", baseErr, "localhost:1883")
	mqttErr.Topic = "homeassistant/sensor/test/state"
	mqttErr.QoS = 1

	if mqttErr.Broker != "localhost:1883" {
		t.Errorf("Expected Broker 'localhost:1883', got '%s'", mqttErr.Broker)
	}
	if mqttErr.Topic != "homeassistant/sensor/test/state" {
		t.Errorf("Expected Topic 'homeassistant/sensor/test/state', got '%s'", mqttErr.Topic)
	}
	if mqttErr.QoS != 1 {
		t.Errorf("Expected QoS 1, got %d", mqttErr.QoS)
	}

	errMsg := mqttErr.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}
	t.Logf("MQTTError message: %s", errMsg)
}

// TestErrorUnwrapping tests error unwrapping
func TestErrorUnwrapping(t *testing.T) {
	baseErr := fmt.Errorf("base error")
	modbusErr := NewModbusError("test", baseErr, 1, "device")

	unwrapped := errors.Unwrap(modbusErr)
	if unwrapped != baseErr {
		t.Error("Expected to unwrap to base error")
	}
}

// TestErrorTypeAssertion tests type assertion for error handling
func TestErrorTypeAssertion(t *testing.T) {
	baseErr := fmt.Errorf("connection failed")
	modbusErr := NewModbusError("read", baseErr, 5, "meter_1")
	modbusErr.Address = 0x1000

	// Simulate error handling with type switch
	var err error = modbusErr

	switch e := err.(type) {
	case *ModbusError:
		if e.SlaveID != 5 {
			t.Errorf("Expected SlaveID 5, got %d", e.SlaveID)
		}
		if e.DeviceID != "meter_1" {
			t.Errorf("Expected DeviceID 'meter_1', got '%s'", e.DeviceID)
		}
		if e.Address != 0x1000 {
			t.Errorf("Expected Address 0x1000, got 0x%04X", e.Address)
		}
		t.Logf("Successfully identified ModbusError with device: %s", e.DeviceID)
	case *MQTTError:
		t.Error("Expected ModbusError, got MQTTError")
	default:
		t.Error("Expected ModbusError, got unknown type")
	}
}

// TestErrorSeverity tests error severity levels
func TestErrorSeverity(t *testing.T) {
	modbusErr := NewModbusError("test", fmt.Errorf("test error"), 1, "device")
	if modbusErr.Severity != SeverityError {
		t.Errorf("Expected SeverityError, got %s", modbusErr.Severity)
	}

	configErr := NewConfigError("test", fmt.Errorf("test error"), "field")
	if configErr.Severity != SeverityCritical {
		t.Errorf("Expected SeverityCritical, got %s", configErr.Severity)
	}

	validationErr := NewValidationError("field", "expected", "actual")
	if validationErr.Severity != SeverityWarning {
		t.Errorf("Expected SeverityWarning, got %s", validationErr.Severity)
	}
}

// TestErrorCodes tests diagnostic error codes
func TestErrorCodes(t *testing.T) {
	configErr := NewConfigError("test", fmt.Errorf("test"), "field")
	if configErr.Code != 1 {
		t.Errorf("Expected Code 1, got %d", configErr.Code)
	}

	modbusErr := NewModbusError("test", fmt.Errorf("test"), 1, "device")
	if modbusErr.Code != 3 {
		t.Errorf("Expected Code 3, got %d", modbusErr.Code)
	}

	mqttErr := NewMQTTError("test", fmt.Errorf("test"), "broker")
	if mqttErr.Code != 4 {
		t.Errorf("Expected Code 4, got %d", mqttErr.Code)
	}
}
