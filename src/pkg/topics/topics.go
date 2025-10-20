package topics

import "fmt"

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
func BuildTopic(prefix, deviceID, sensorKey, topicType string) string {
	entityID := BuildUniqueID(deviceID, sensorKey)
	return fmt.Sprintf("%s/sensor/%s/%s/%s", prefix, deviceID, entityID, topicType)
}

// ConstructHATopic builds Home Assistant MQTT state topic with configurable prefix
// This is a standalone function to avoid import cycles between mqtt and modbus packages
// Pattern: {prefix}/sensor/{device_id}/{device_id}_{sensor_key}/state
func ConstructHATopic(prefix, deviceID, sensorKey, deviceClass string) string {
	return BuildTopic(prefix, deviceID, sensorKey, "state")
}

// BuildDiscoveryTopic constructs the discovery config topic for a sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_{sensor_key}/config
func BuildDiscoveryTopic(prefix, deviceID, sensorKey string) string {
	return BuildTopic(prefix, deviceID, sensorKey, "config")
}

// BuildStateTopic constructs the state topic for a sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_{sensor_key}/state
func BuildStateTopic(prefix, deviceID, sensorKey string) string {
	return BuildTopic(prefix, deviceID, sensorKey, "state")
}

// BuildDiagnosticDiscoveryTopic constructs discovery topic for diagnostic sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_diagnostic/config
func BuildDiagnosticDiscoveryTopic(prefix, deviceID string) string {
	return BuildTopic(prefix, deviceID, "diagnostic", "config")
}

// BuildDiagnosticStateTopic constructs state topic for diagnostic sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_diagnostic/state
func BuildDiagnosticStateTopic(prefix, deviceID string) string {
	return BuildTopic(prefix, deviceID, "diagnostic", "state")
}

// BuildDiagnosticUniqueID constructs unique ID for diagnostic sensor
// Pattern: {device_id}_diagnostic
func BuildDiagnosticUniqueID(deviceID string) string {
	return BuildUniqueID(deviceID, "diagnostic")
}

// BuildDeviceDiagnosticDiscoveryTopic constructs discovery topic for per-device diagnostic sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_device_diagnostic/config
func BuildDeviceDiagnosticDiscoveryTopic(prefix, deviceID string) string {
	return BuildTopic(prefix, deviceID, "device_diagnostic", "config")
}

// BuildDeviceDiagnosticStateTopic constructs state topic for per-device diagnostic sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_device_diagnostic/state
func BuildDeviceDiagnosticStateTopic(prefix, deviceID string) string {
	return BuildTopic(prefix, deviceID, "device_diagnostic", "state")
}

// BuildDeviceDiagnosticUniqueID constructs unique ID for per-device diagnostic sensor
// Pattern: {device_id}_device_diagnostic
func BuildDeviceDiagnosticUniqueID(deviceID string) string {
	return BuildUniqueID(deviceID, "device_diagnostic")
}
