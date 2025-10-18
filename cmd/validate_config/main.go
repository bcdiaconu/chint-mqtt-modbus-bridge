package main

import (
	"chint-mqtt-modbus-bridge/pkg/config"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run validate_config.go <config-file>")
		os.Exit(1)
	}

	configPath := os.Args[1]
	fmt.Printf("📄 Loading config from: %s\n", configPath)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Config loaded successfully!\n")
	fmt.Printf("   Version: %s\n", cfg.Version)
	fmt.Printf("   MQTT Broker: %s:%d\n", cfg.MQTT.Broker, cfg.MQTT.Port)

	// Debug: print raw devices
	fmt.Printf("\n🔍 DEBUG: Devices map length: %d\n", len(cfg.Devices))
	for key, device := range cfg.Devices {
		fmt.Printf("   Device key '%s':\n", key)
		fmt.Printf("     Metadata.Name: '%s'\n", device.Metadata.Name)
		fmt.Printf("     RTU.SlaveID: %d\n", device.RTU.SlaveID)
		fmt.Printf("     Modbus.RegisterGroups: %d\n", len(device.Modbus.RegisterGroups))
	}

	// Check if V2.1 (device-based)
	if len(cfg.Devices) > 0 {
		fmt.Printf("   Devices: %d\n", len(cfg.Devices))
		for key, device := range cfg.Devices {
			haDeviceID := device.GetHADeviceID(key)
			usesDefaultID := haDeviceID == key

			fmt.Printf("     - %s:\n", key)
			fmt.Printf("         Name: %s\n", device.GetName())
			fmt.Printf("         Slave ID: %d\n", device.GetSlaveID())
			fmt.Printf("         Manufacturer: %s\n", device.GetHAManufacturer())
			fmt.Printf("         Model: %s\n", device.GetHAModel())
			if usesDefaultID {
				fmt.Printf("         HA Device ID: %s (using device key)\n", haDeviceID)
			} else {
				fmt.Printf("         HA Device ID: %s\n", haDeviceID)
			}
			fmt.Printf("         Enabled: %v\n", device.IsEnabled())
			fmt.Printf("         Register Groups: %d\n", len(device.Modbus.RegisterGroups))
			if device.GetPollInterval() > 0 {
				fmt.Printf("         Poll Interval: %d ms\n", device.GetPollInterval())
			}
		}
	} else if len(cfg.RegisterGroups) > 0 {
		fmt.Printf("   Register Groups (V2.0): %d\n", len(cfg.RegisterGroups))
	} else {
		fmt.Printf("   Registers (V1.0): %d\n", len(cfg.Registers))
	}

	fmt.Println("\n✅ Configuration is valid!")
}
