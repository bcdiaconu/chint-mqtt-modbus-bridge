package logger

// ILogger is an interface for dependency injection
// Allows testing with mock loggers and flexibility in log implementation
type ILogger interface {
	LogInfo(format string, args ...interface{})
	LogWarn(format string, args ...interface{})
	LogError(format string, args ...interface{})
	LogDebug(format string, args ...interface{})
}

// StandardLogger implements ILogger interface using the global logger functions
type StandardLogger struct{}

// NewStandardLogger creates a logger that uses global logger functions
func NewStandardLogger() ILogger {
	return &StandardLogger{}
}

// LogInfo logs an info message
func (l *StandardLogger) LogInfo(format string, args ...interface{}) {
	LogInfo(format, args...)
}

// LogWarn logs a warning message
func (l *StandardLogger) LogWarn(format string, args ...interface{}) {
	LogWarn(format, args...)
}

// LogError logs an error message
func (l *StandardLogger) LogError(format string, args ...interface{}) {
	LogError(format, args...)
}

// LogDebug logs a debug message
func (l *StandardLogger) LogDebug(format string, args ...interface{}) {
	LogDebug(format, args...)
}

// MockLogger is a logger for testing that records log messages
type MockLogger struct {
	InfoMessages  []string
	WarnMessages  []string
	ErrorMessages []string
	DebugMessages []string
}

// NewMockLogger creates a new mock logger for testing
func NewMockLogger() *MockLogger {
	return &MockLogger{
		InfoMessages:  make([]string, 0),
		WarnMessages:  make([]string, 0),
		ErrorMessages: make([]string, 0),
		DebugMessages: make([]string, 0),
	}
}

// LogInfo records an info message
func (l *MockLogger) LogInfo(format string, args ...interface{}) {
	l.InfoMessages = append(l.InfoMessages, format)
}

// LogWarn records a warning message
func (l *MockLogger) LogWarn(format string, args ...interface{}) {
	l.WarnMessages = append(l.WarnMessages, format)
}

// LogError records an error message
func (l *MockLogger) LogError(format string, args ...interface{}) {
	l.ErrorMessages = append(l.ErrorMessages, format)
}

// LogDebug records a debug message
func (l *MockLogger) LogDebug(format string, args ...interface{}) {
	l.DebugMessages = append(l.DebugMessages, format)
}

// Reset clears all recorded messages
func (l *MockLogger) Reset() {
	l.InfoMessages = l.InfoMessages[:0]
	l.WarnMessages = l.WarnMessages[:0]
	l.ErrorMessages = l.ErrorMessages[:0]
	l.DebugMessages = l.DebugMessages[:0]
}

// HasInfoMessage checks if an info message was logged
func (l *MockLogger) HasInfoMessage() bool {
	return len(l.InfoMessages) > 0
}

// HasWarnMessage checks if a warning message was logged
func (l *MockLogger) HasWarnMessage() bool {
	return len(l.WarnMessages) > 0
}

// HasErrorMessage checks if an error message was logged
func (l *MockLogger) HasErrorMessage() bool {
	return len(l.ErrorMessages) > 0
}

// HasDebugMessage checks if a debug message was logged
func (l *MockLogger) HasDebugMessage() bool {
	return len(l.DebugMessages) > 0
}
