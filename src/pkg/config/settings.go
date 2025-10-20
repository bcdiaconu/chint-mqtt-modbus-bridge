package config

// ModbusSettings contains only Modbus-specific configuration
// Used for dependency injection to avoid coupling to full Config
type ModbusSettings struct {
	SlaveID      uint8
	PollInterval int
	Timeout      int
}

// NewModbusSettings extracts Modbus settings from full config
func NewModbusSettings(cfg *Config) ModbusSettings {
	return ModbusSettings{
		SlaveID:      cfg.Modbus.SlaveID,
		PollInterval: cfg.Modbus.PollInterval,
		Timeout:      cfg.Modbus.Timeout,
	}
}

// MQTTSettings contains only MQTT-specific configuration
// Used for dependency injection to avoid coupling to full Config
type MQTTSettings struct {
	Broker            string
	Port              int
	Username          string
	Password          string
	ClientID          string
	RetryDelay        int
	KeepAlive         int
	HeartbeatInterval int
}

// NewMQTTSettings extracts MQTT settings from full config
func NewMQTTSettings(cfg *Config) MQTTSettings {
	return MQTTSettings{
		Broker:            cfg.MQTT.Broker,
		Port:              cfg.MQTT.Port,
		Username:          cfg.MQTT.Username,
		Password:          cfg.MQTT.Password,
		ClientID:          cfg.MQTT.ClientID,
		RetryDelay:        cfg.MQTT.RetryDelay,
		KeepAlive:         cfg.MQTT.KeepAlive,
		HeartbeatInterval: cfg.MQTT.HeartbeatInterval,
	}
}

// GatewaySettings contains only gateway-specific configuration
// Used for dependency injection to avoid coupling to full Config
type GatewaySettings struct {
	MAC       string
	CmdTopic  string
	DataTopic string
}

// NewGatewaySettings extracts gateway settings from full config
func NewGatewaySettings(cfg *Config) GatewaySettings {
	return GatewaySettings{
		MAC:       cfg.MQTT.Gateway.MAC,
		CmdTopic:  cfg.MQTT.Gateway.CmdTopic,
		DataTopic: cfg.MQTT.Gateway.DataTopic,
	}
}

// PollingSettings contains polling loop configuration
// Used for dependency injection to avoid coupling to full Config
type PollingSettings struct {
	PollInterval               int // Milliseconds
	PerformanceSummaryInterval int // Seconds
	ErrorGracePeriod           int // Seconds
}

// NewPollingSettings extracts polling settings from full config
func NewPollingSettings(cfg *Config) PollingSettings {
	return PollingSettings{
		PollInterval:               cfg.Modbus.PollInterval,
		PerformanceSummaryInterval: 30, // Default 30 seconds
		ErrorGracePeriod:           15, // Default 15 seconds
	}
}

// HomeAssistantSettings contains Home Assistant discovery configuration
// Used for dependency injection to avoid coupling to full Config
type HomeAssistantSettings struct {
	DiscoveryPrefix          string
	DeviceDiagnosticsEnabled bool
	PublishOnStateChange     bool
}

// NewHomeAssistantSettings extracts Home Assistant settings from full config
func NewHomeAssistantSettings(cfg *Config) HomeAssistantSettings {
	return HomeAssistantSettings{
		DiscoveryPrefix:          cfg.HomeAssistant.DiscoveryPrefix,
		DeviceDiagnosticsEnabled: cfg.HomeAssistant.DeviceDiagnostics.Enabled,
		PublishOnStateChange:     cfg.HomeAssistant.DeviceDiagnostics.PublishOnStateChange,
	}
}

// DiagnosticSettings contains diagnostic configuration
// Used for dependency injection to avoid coupling to full Config
type DiagnosticSettings struct {
	Intervals  DiagnosticIntervalsConfig
	Thresholds DiagnosticThresholdsConfig
}

// NewDiagnosticSettings extracts diagnostic settings from full config
func NewDiagnosticSettings(cfg *Config) DiagnosticSettings {
	return DiagnosticSettings{
		Intervals:  cfg.HomeAssistant.DeviceDiagnostics.Intervals,
		Thresholds: cfg.HomeAssistant.DeviceDiagnostics.Thresholds,
	}
}
