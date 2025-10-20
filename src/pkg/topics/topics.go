package topics

import (
	"fmt"
	"strings"
)

// Package-level variables for configuration
// These are set once at startup via Initialize()
var (
	discoveryPrefix string = "homeassistant" // Default HA discovery prefix
)

// Initialize sets up the topics package with configuration from the app
// This should be called once at application startup before any topics are built
func Initialize(prefix string) {
	if prefix != "" {
		discoveryPrefix = prefix
	}
}

// GetDiscoveryPrefix returns the current discovery prefix (useful for testing)
func GetDiscoveryPrefix() string {
	return discoveryPrefix
}

// BuildUniqueID constructs the unique ID for a sensor
// Pattern: {device_id}_{sensor_key}
// This is the foundation for both unique IDs and entity naming in topics
func BuildUniqueID(deviceID, sensorKey string) string {
	return fmt.Sprintf("%s_%s", deviceID, sensorKey)
}

// BuildTopic constructs a complete MQTT topic for Home Assistant
// Pattern: {prefix}/sensor/{device_id}/{device_id}_{sensor_key}/{type}
// Example: homeassistant/sensor/energy_meter_mains/energy_meter_mains_voltage/config
//
// This is the single source of truth for topic construction, ensuring consistency
// across all sensor types (regular, diagnostic, device diagnostic, etc.)
func BuildTopic(deviceID, sensorKey, topicType string) string {
	entityID := BuildUniqueID(deviceID, sensorKey)
	return fmt.Sprintf("%s/sensor/%s/%s/%s", discoveryPrefix, deviceID, entityID, topicType)
}

// ConstructHATopic builds Home Assistant MQTT state topic with configurable prefix
// This is a standalone function to avoid import cycles between mqtt and modbus packages
// Pattern: {prefix}/sensor/{device_id}/{device_id}_{sensor_key}/state
func ConstructHATopic(deviceID, sensorKey, deviceClass string) string {
	return BuildTopic(deviceID, sensorKey, "state")
}

// BuildDiscoveryTopic constructs the discovery config topic for a sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_{sensor_key}/config
func BuildDiscoveryTopic(deviceID, sensorKey string) string {
	return BuildTopic(deviceID, sensorKey, "config")
}

// BuildStateTopic constructs the state topic for a sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_{sensor_key}/state
func BuildStateTopic(deviceID, sensorKey string) string {
	return BuildTopic(deviceID, sensorKey, "state")
}

// BuildDiagnosticDiscoveryTopic constructs discovery topic for diagnostic sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_diagnostic/config
func BuildDiagnosticDiscoveryTopic(deviceID string) string {
	return BuildTopic(deviceID, "diagnostic", "config")
}

// BuildDiagnosticStateTopic constructs state topic for diagnostic sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_diagnostic/state
func BuildDiagnosticStateTopic(deviceID string) string {
	return BuildTopic(deviceID, "diagnostic", "state")
}

// BuildDiagnosticUniqueID constructs unique ID for diagnostic sensor
// Pattern: {device_id}_diagnostic
func BuildDiagnosticUniqueID(deviceID string) string {
	return BuildUniqueID(deviceID, "diagnostic")
}

// BuildDeviceDiagnosticDiscoveryTopic constructs discovery topic for per-device diagnostic sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_device_diagnostic/config
func BuildDeviceDiagnosticDiscoveryTopic(deviceID string) string {
	return BuildTopic(deviceID, "device_diagnostic", "config")
}

// BuildDeviceDiagnosticStateTopic constructs state topic for per-device diagnostic sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_device_diagnostic/state
func BuildDeviceDiagnosticStateTopic(deviceID string) string {
	return BuildTopic(deviceID, "device_diagnostic", "state")
}

// BuildDeviceDiagnosticUniqueID constructs unique ID for per-device diagnostic sensor
// Pattern: {device_id}_device_diagnostic
func BuildDeviceDiagnosticUniqueID(deviceID string) string {
	return BuildUniqueID(deviceID, "device_diagnostic")
}

// BuildStatusTopic constructs the availability/status topic for any device or bridge
// Pattern: {client_id}/status where client_id uses dashes (e.g., modbus-bridge)
// Converts device_id (mqtt_modbus_bridge) to client_id format (modbus-bridge)
// Examples:
//   - mqtt_modbus_bridge -> modbus-bridge/status
//   - energy_meter_lights -> energy-meter-lights/status
//
// This topic is used for Last Will Testament and availability tracking
func BuildStatusTopic(deviceID string) string {
	// Convert underscore to dash for MQTT client compatibility
	clientID := strings.ReplaceAll(deviceID, "_", "-")
	// Remove mqtt- prefix if present (mqtt_modbus_bridge -> modbus-bridge)
	clientID = strings.TrimPrefix(clientID, "mqtt-")
	return fmt.Sprintf("%s/status", clientID)
}

// BuildDiagnosticDataTopic constructs the diagnostic data topic for raw diagnostic information
// Pattern: {client_id}/diagnostic where client_id uses dashes (e.g., modbus-bridge)
// Converts device_id (mqtt_modbus_bridge) to client_id format (modbus-bridge)
// Examples:
//   - mqtt_modbus_bridge -> modbus-bridge/diagnostic
//   - energy_meter_lights -> energy-meter-lights/diagnostic
//
// This is NOT a Home Assistant sensor topic - it's for internal diagnostics data
func BuildDiagnosticDataTopic(deviceID string) string {
	// Convert underscore to dash for MQTT client compatibility
	clientID := strings.ReplaceAll(deviceID, "_", "-")
	// Remove mqtt- prefix if present (mqtt_modbus_bridge -> modbus-bridge)
	clientID = strings.TrimPrefix(clientID, "mqtt-")
	return fmt.Sprintf("%s/diagnostic", clientID)
}
