package modbus

import "time"

// CommandResult result of executing a command
type CommandResult struct {
	Strategy    string  `json:"strategy"`
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Topic       string  `json:"topic"`
	DeviceClass string  `json:"device_class"`
	StateClass  string  `json:"state_class"`
	RawData     []byte  `json:"raw_data"`
}

// CachedResult stores a command result with timestamp for cache validation
type CachedResult struct {
	Result    *CommandResult
	Timestamp time.Time
}

// CommandError custom error for commands
type CommandError struct {
	Strategy string
	Message  string
	Cause    error
}

func (e *CommandError) Error() string {
	if e.Cause != nil {
		return e.Strategy + ": " + e.Message + " - " + e.Cause.Error()
	}
	return e.Strategy + ": " + e.Message
}

func (e *CommandError) Unwrap() error {
	return e.Cause
}
