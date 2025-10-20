package errors

import (
	"context"
	"fmt"
	"mqtt-modbus-bridge/pkg/logger"
)

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	diagnosticPublisher DiagnosticPublisher
}

// DiagnosticPublisher interface for publishing diagnostics
type DiagnosticPublisher interface {
	PublishDiagnostic(ctx context.Context, code int, message string) error
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(publisher DiagnosticPublisher) *ErrorHandler {
	return &ErrorHandler{
		diagnosticPublisher: publisher,
	}
}

// Handle processes an error with appropriate logging and diagnostics
func (h *ErrorHandler) Handle(ctx context.Context, err error) {
	if err == nil {
		return
	}

	// Type switch on error types
	switch e := err.(type) {
	case *GatewayError:
		h.handleGatewayError(ctx, e)
	case *ModbusError:
		h.handleModbusError(ctx, e)
	case *MQTTError:
		h.handleMQTTError(ctx, e)
	case *ConfigError:
		h.handleConfigError(ctx, e)
	case *ValidationError:
		h.handleValidationError(ctx, e)
	case *BridgeError:
		h.handleBridgeError(ctx, e)
	default:
		h.handleGenericError(ctx, err)
	}
}

// handleGatewayError handles gateway-specific errors
func (h *ErrorHandler) handleGatewayError(ctx context.Context, err *GatewayError) {
	switch err.Severity {
	case SeverityCritical:
		logger.LogError("🔴 CRITICAL Gateway Error: %s", err.Error())
	case SeverityError:
		logger.LogError("❌ Gateway Error: %s", err.Error())
	case SeverityWarning:
		logger.LogWarn("⚠️ Gateway Warning: %s", err.Error())
	default:
		logger.LogInfo("ℹ️ Gateway Info: %s", err.Error())
	}

	// Publish diagnostic if publisher is available
	if h.diagnosticPublisher != nil {
		message := fmt.Sprintf("Gateway %s: %s", err.GatewayMAC, err.Op)
		if publishErr := h.diagnosticPublisher.PublishDiagnostic(ctx, err.Code, message); publishErr != nil {
			logger.LogDebug("Failed to publish gateway error diagnostic: %v", publishErr)
		}
	}
}

// handleModbusError handles Modbus-specific errors
func (h *ErrorHandler) handleModbusError(ctx context.Context, err *ModbusError) {
	switch err.Severity {
	case SeverityCritical:
		logger.LogError("🔴 CRITICAL Modbus Error: %s", err.Error())
	case SeverityError:
		logger.LogError("❌ Modbus Error: %s", err.Error())
	case SeverityWarning:
		logger.LogWarn("⚠️ Modbus Warning: %s", err.Error())
	default:
		logger.LogInfo("ℹ️ Modbus Info: %s", err.Error())
	}

	// Publish diagnostic if publisher is available
	if h.diagnosticPublisher != nil {
		message := fmt.Sprintf("Device '%s' (slave %d): %s", err.DeviceID, err.SlaveID, err.Op)
		if publishErr := h.diagnosticPublisher.PublishDiagnostic(ctx, err.Code, message); publishErr != nil {
			logger.LogDebug("Failed to publish Modbus error diagnostic: %v", publishErr)
		}
	}
}

// handleMQTTError handles MQTT-specific errors
func (h *ErrorHandler) handleMQTTError(ctx context.Context, err *MQTTError) {
	switch err.Severity {
	case SeverityCritical:
		logger.LogError("🔴 CRITICAL MQTT Error: %s", err.Error())
	case SeverityError:
		logger.LogError("❌ MQTT Error: %s", err.Error())
	case SeverityWarning:
		logger.LogWarn("⚠️ MQTT Warning: %s", err.Error())
	default:
		logger.LogInfo("ℹ️ MQTT Info: %s", err.Error())
	}

	// Publish diagnostic if publisher is available
	if h.diagnosticPublisher != nil {
		message := fmt.Sprintf("Broker '%s': %s", err.Broker, err.Op)
		if publishErr := h.diagnosticPublisher.PublishDiagnostic(ctx, err.Code, message); publishErr != nil {
			logger.LogDebug("Failed to publish MQTT error diagnostic: %v", publishErr)
		}
	}
}

// handleConfigError handles configuration errors
func (h *ErrorHandler) handleConfigError(ctx context.Context, err *ConfigError) {
	// Config errors are always critical
	logger.LogError("🔴 CRITICAL Configuration Error: %s", err.Error())

	// Publish diagnostic if publisher is available
	if h.diagnosticPublisher != nil {
		message := fmt.Sprintf("Config field '%s': %s", err.Field, err.Op)
		if publishErr := h.diagnosticPublisher.PublishDiagnostic(ctx, err.Code, message); publishErr != nil {
			logger.LogDebug("Failed to publish config error diagnostic: %v", publishErr)
		}
	}
}

// handleValidationError handles validation errors
func (h *ErrorHandler) handleValidationError(ctx context.Context, err *ValidationError) {
	logger.LogWarn("⚠️ Validation Error: %s", err.Error())

	// Publish diagnostic if publisher is available
	if h.diagnosticPublisher != nil {
		message := fmt.Sprintf("Validation failed for '%s'", err.Field)
		if publishErr := h.diagnosticPublisher.PublishDiagnostic(ctx, err.Code, message); publishErr != nil {
			logger.LogDebug("Failed to publish validation error diagnostic: %v", publishErr)
		}
	}
}

// handleBridgeError handles generic bridge errors
func (h *ErrorHandler) handleBridgeError(ctx context.Context, err *BridgeError) {
	switch err.Severity {
	case SeverityCritical:
		logger.LogError("🔴 CRITICAL Error: %s", err.Error())
	case SeverityError:
		logger.LogError("❌ Error: %s", err.Error())
	case SeverityWarning:
		logger.LogWarn("⚠️ Warning: %s", err.Error())
	default:
		logger.LogInfo("ℹ️ Info: %s", err.Error())
	}

	// Publish diagnostic if publisher is available
	if h.diagnosticPublisher != nil {
		if publishErr := h.diagnosticPublisher.PublishDiagnostic(ctx, err.Code, err.Op); publishErr != nil {
			logger.LogDebug("Failed to publish error diagnostic: %v", publishErr)
		}
	}
}

// handleGenericError handles non-typed errors
func (h *ErrorHandler) handleGenericError(ctx context.Context, err error) {
	logger.LogError("❌ Untyped Error: %v", err)

	// Publish generic diagnostic if publisher is available
	if h.diagnosticPublisher != nil {
		if publishErr := h.diagnosticPublisher.PublishDiagnostic(ctx, 99, err.Error()); publishErr != nil {
			logger.LogDebug("Failed to publish generic error diagnostic: %v", publishErr)
		}
	}
}

// IsRecoverable returns true if the error is recoverable
func IsRecoverable(err error) bool {
	if err == nil {
		return true
	}

	switch e := err.(type) {
	case *ConfigError:
		return false // Config errors are not recoverable
	case *BridgeError:
		return e.Severity != SeverityCritical
	case *GatewayError:
		return e.Severity != SeverityCritical
	case *ModbusError:
		return e.Severity != SeverityCritical
	case *MQTTError:
		return e.Severity != SeverityCritical
	default:
		return true // Unknown errors are assumed recoverable
	}
}

// GetDiagnosticCode extracts the diagnostic code from an error
func GetDiagnosticCode(err error) int {
	if err == nil {
		return 0
	}

	switch e := err.(type) {
	case *GatewayError:
		return e.Code
	case *ModbusError:
		return e.Code
	case *MQTTError:
		return e.Code
	case *ConfigError:
		return e.Code
	case *ValidationError:
		return e.Code
	case *BridgeError:
		return e.Code
	default:
		return 99 // Generic error code
	}
}
