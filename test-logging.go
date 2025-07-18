package main

import (
	"fmt"
	"mqtt-modbus-bridge/internal/config"
)

func main() {
	// Test the logging configuration
	cfg := &config.LoggingConfig{
		Level: "debug",
		File:  "",
	}

	// Set global logging
	config.GlobalLogging = cfg

	// Test all logging levels
	fmt.Println("Testing logging system...")

	config.LogError("This is an error message")
	config.LogWarn("This is a warning message")
	config.LogInfo("This is an info message")
	config.LogDebug("This is a debug message")
	config.LogTrace("This is a trace message")

	fmt.Printf("Is debug enabled: %t\n", config.IsDebugEnabled())
	fmt.Printf("Is trace enabled: %t\n", config.IsTraceEnabled())

	// Test with different levels
	fmt.Println("\nTesting with 'warn' level:")
	cfg.Level = "warn"
	config.LogError("Error visible")
	config.LogWarn("Warning visible")
	config.LogInfo("Info not visible")
	config.LogDebug("Debug not visible")
	config.LogTrace("Trace not visible")

	fmt.Println("\nTesting with 'error' level:")
	cfg.Level = "error"
	config.LogError("Error visible")
	config.LogWarn("Warning not visible")
	config.LogInfo("Info not visible")
	config.LogDebug("Debug not visible")
	config.LogTrace("Trace not visible")
}
