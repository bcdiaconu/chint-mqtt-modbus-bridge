package mqtt

import (
	"mqtt-modbus-bridge/pkg/topics"
)

// TopicFactory provides centralized topic construction for Home Assistant MQTT discovery
type TopicFactory struct {
	discoveryPrefix string
}

// NewTopicFactory creates a new topic factory
func NewTopicFactory(discoveryPrefix string) *TopicFactory {
	return &TopicFactory{
		discoveryPrefix: discoveryPrefix,
	}
}

// BuildDiscoveryTopic constructs the discovery config topic for a sensor
func (tf *TopicFactory) BuildDiscoveryTopic(deviceID, sensorKey string) string {
	return topics.BuildDiscoveryTopic(tf.discoveryPrefix, deviceID, sensorKey)
}

// BuildStateTopic constructs the state topic for a sensor
func (tf *TopicFactory) BuildStateTopic(deviceID, sensorKey string) string {
	return topics.BuildStateTopic(tf.discoveryPrefix, deviceID, sensorKey)
}

// BuildUniqueID constructs the unique ID for a sensor
func (tf *TopicFactory) BuildUniqueID(deviceID, sensorKey string) string {
	return topics.BuildUniqueID(deviceID, sensorKey)
}

// BuildDiagnosticDiscoveryTopic constructs discovery topic for diagnostic sensor
func (tf *TopicFactory) BuildDiagnosticDiscoveryTopic(deviceID string) string {
	return topics.BuildDiagnosticDiscoveryTopic(tf.discoveryPrefix, deviceID)
}

// BuildDiagnosticStateTopic constructs state topic for diagnostic sensor
func (tf *TopicFactory) BuildDiagnosticStateTopic(deviceID string) string {
	return topics.BuildDiagnosticStateTopic(tf.discoveryPrefix, deviceID)
}

// BuildDiagnosticUniqueID constructs unique ID for diagnostic sensor
func (tf *TopicFactory) BuildDiagnosticUniqueID(deviceID string) string {
	return topics.BuildDiagnosticUniqueID(deviceID)
}

// BuildDeviceDiagnosticDiscoveryTopic constructs discovery topic for per-device diagnostic sensor
func (tf *TopicFactory) BuildDeviceDiagnosticDiscoveryTopic(deviceID string) string {
	return topics.BuildDeviceDiagnosticDiscoveryTopic(tf.discoveryPrefix, deviceID)
}

// BuildDeviceDiagnosticStateTopic constructs state topic for per-device diagnostic sensor
func (tf *TopicFactory) BuildDeviceDiagnosticStateTopic(deviceID string) string {
	return topics.BuildDeviceDiagnosticStateTopic(tf.discoveryPrefix, deviceID)
}

// BuildDeviceDiagnosticUniqueID constructs unique ID for per-device diagnostic sensor
func (tf *TopicFactory) BuildDeviceDiagnosticUniqueID(deviceID string) string {
	return topics.BuildDeviceDiagnosticUniqueID(deviceID)
}

// ExtractDeviceID extracts the device ID from a DeviceInfo
func ExtractDeviceID(deviceInfo *DeviceInfo) string {
	if deviceInfo != nil && len(deviceInfo.Identifiers) > 0 {
		return deviceInfo.Identifiers[0]
	}
	return ""
}
