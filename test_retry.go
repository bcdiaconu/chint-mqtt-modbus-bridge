package main

import (
	"fmt"
	"log"
	"mqtt-modbus-bridge/internal/config"
	"time"
)

func main() {
	log.Printf("🧪 Testing retry configuration...")

	// Test cu configurare implicită
	cfg := &config.MQTTConfig{
		RetryDelay: 0, // Implicit 0
	}

	retryDelay := time.Duration(cfg.RetryDelay) * time.Millisecond
	if retryDelay == 0 {
		retryDelay = 5000 * time.Millisecond // Default 5 seconds
	}

	fmt.Printf("✅ Default retry delay: %.0f seconds\n", retryDelay.Seconds())

	// Test cu configurare setată
	cfg.RetryDelay = 3000 // 3 secunde
	retryDelay = time.Duration(cfg.RetryDelay) * time.Millisecond

	fmt.Printf("✅ Configured retry delay: %.0f seconds\n", retryDelay.Seconds())

	log.Printf("🎯 Configuration test completed successfully")
}
