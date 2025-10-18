package topics

import "fmt"

// ConstructHATopic builds Home Assistant MQTT state topic
// This is a standalone function to avoid import cycles between mqtt and modbus packages
// Pattern: homeassistant/sensor/{device_id}/{device_id}_{sensor_key}/state
func ConstructHATopic(deviceID, sensorKey, deviceClass string) string {
	return fmt.Sprintf("homeassistant/sensor/%s/%s_%s/state", deviceID, deviceID, sensorKey)
}

// BuildDiscoveryTopic constructs the discovery config topic for a sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_{sensor_key}/config
func BuildDiscoveryTopic(prefix, deviceID, sensorKey string) string {
	return fmt.Sprintf("%s/sensor/%s/%s_%s/config", prefix, deviceID, deviceID, sensorKey)
}

// BuildStateTopic constructs the state topic for a sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_{sensor_key}/state
func BuildStateTopic(prefix, deviceID, sensorKey string) string {
	return ConstructHATopic(deviceID, sensorKey, "")
}

// BuildUniqueID constructs the unique ID for a sensor
// Pattern: {device_id}_{sensor_key}
func BuildUniqueID(deviceID, sensorKey string) string {
	return fmt.Sprintf("%s_%s", deviceID, sensorKey)
}

// BuildDiagnosticDiscoveryTopic constructs discovery topic for diagnostic sensor
// Pattern: {prefix}/sensor/{device_id}_diagnostic/config
func BuildDiagnosticDiscoveryTopic(prefix, deviceID string) string {
	return fmt.Sprintf("%s/sensor/%s_diagnostic/config", prefix, deviceID)
}

// BuildDiagnosticStateTopic constructs state topic for diagnostic sensor
// Pattern: {prefix}/sensor/{device_id}/{device_id}_diagnostic/state
func BuildDiagnosticStateTopic(prefix, deviceID string) string {
	return fmt.Sprintf("%s/sensor/%s/%s_diagnostic/state", prefix, deviceID, deviceID)
}

// BuildDiagnosticUniqueID constructs unique ID for diagnostic sensor
// Pattern: {device_id}_diagnostic
func BuildDiagnosticUniqueID(deviceID string) string {
	return fmt.Sprintf("%s_diagnostic", deviceID)
}
