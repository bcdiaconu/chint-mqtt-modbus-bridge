package logger

import (
	"log"
	"os"
	"strings"
)

// LogLevel constants
const (
	LogLevelError = "error"
	LogLevelWarn  = "warn"
	LogLevelInfo  = "info"
	LogLevelDebug = "debug"
	LogLevelTrace = "trace"
)

// LoggingConfig represents the logging configuration
type LoggingConfig struct {
	Level   string `yaml:"level"`
	File    string `yaml:"file"`
	MaxSize int    `yaml:"max_size"`
	MaxAge  int    `yaml:"max_age"`
}

// Global logging configuration
var GlobalLogging *LoggingConfig

// Logger wraps the standard logger with verbosity levels
type Logger struct {
	*log.Logger
	level string
}

// NewLogger creates a new logger with verbosity level
func NewLogger(config *LoggingConfig) *Logger {
	level := strings.ToLower(config.Level)
	if level == "" {
		level = LogLevelInfo // Default to INFO
	}

	// Set log output
	var output *os.File
	if config.File != "" {
		var err error
		// Use 0600 permissions (owner read/write only) for security
		output, err = os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			log.Printf("Failed to open log file %s: %v", config.File, err)
			output = os.Stdout
		}
	} else {
		output = os.Stdout
	}

	logger := &Logger{
		Logger: log.New(output, "", log.LstdFlags|log.Lshortfile),
		level:  level,
	}

	// Set global reference
	GlobalLogging = config

	return logger
}

// shouldLog checks if a message should be logged based on current level
func shouldLog(currentLevel, messageLevel string) bool {
	levels := []string{LogLevelError, LogLevelWarn, LogLevelInfo, LogLevelDebug, LogLevelTrace}

	currentIndex := -1
	messageIndex := -1

	for i, level := range levels {
		if level == currentLevel {
			currentIndex = i
		}
		if level == messageLevel {
			messageIndex = i
		}
	}

	// If either level is not found, default to allowing the message
	if currentIndex == -1 || messageIndex == -1 {
		return true
	}

	return messageIndex <= currentIndex
}

// Error logs error messages
func (l *Logger) Error(format string, args ...interface{}) {
	if shouldLog(l.level, LogLevelError) {
		l.Printf("❌ "+format, args...)
	}
}

// Warn logs warning messages
func (l *Logger) Warn(format string, args ...interface{}) {
	if shouldLog(l.level, LogLevelWarn) {
		l.Printf("⚠️ "+format, args...)
	}
}

// Info logs info messages
func (l *Logger) Info(format string, args ...interface{}) {
	if shouldLog(l.level, LogLevelInfo) {
		l.Printf("ℹ️ "+format, args...)
	}
}

// Debug logs debug messages
func (l *Logger) Debug(format string, args ...interface{}) {
	if shouldLog(l.level, LogLevelDebug) {
		l.Printf("🔧 "+format, args...)
	}
}

// Trace logs trace messages
func (l *Logger) Trace(format string, args ...interface{}) {
	if shouldLog(l.level, LogLevelTrace) {
		l.Printf("🔍 "+format, args...)
	}
}

// LogStartup logs startup messages that should always be visible regardless of log level
func LogStartup(format string, args ...interface{}) {
	log.Printf("🔧 "+format, args...)
}

// Helper functions for global logging
func LogError(format string, args ...interface{}) {
	if GlobalLogging != nil && shouldLog(strings.ToLower(GlobalLogging.Level), LogLevelError) {
		log.Printf("❌ "+format, args...)
	}
}

func LogWarn(format string, args ...interface{}) {
	if GlobalLogging != nil && shouldLog(strings.ToLower(GlobalLogging.Level), LogLevelWarn) {
		log.Printf("⚠️ "+format, args...)
	}
}

func LogInfo(format string, args ...interface{}) {
	if GlobalLogging != nil && shouldLog(strings.ToLower(GlobalLogging.Level), LogLevelInfo) {
		log.Printf("ℹ️ "+format, args...)
	}
}

func LogDebug(format string, args ...interface{}) {
	if GlobalLogging != nil && shouldLog(strings.ToLower(GlobalLogging.Level), LogLevelDebug) {
		log.Printf("🔧 "+format, args...)
	}
}

func LogTrace(format string, args ...interface{}) {
	if GlobalLogging != nil && shouldLog(strings.ToLower(GlobalLogging.Level), LogLevelTrace) {
		log.Printf("🔍 "+format, args...)
	}
}

// IsDebugEnabled checks if debug logging is enabled
func IsDebugEnabled() bool {
	return GlobalLogging != nil && shouldLog(strings.ToLower(GlobalLogging.Level), LogLevelDebug)
}

// IsTraceEnabled checks if trace logging is enabled
func IsTraceEnabled() bool {
	return GlobalLogging != nil && shouldLog(strings.ToLower(GlobalLogging.Level), LogLevelTrace)
}
