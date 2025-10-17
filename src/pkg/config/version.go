package config

import "fmt"

// Config file format version constants
const (
	// CurrentVersion is the configuration version this code can parse
	CurrentVersion = "2.1"

	// MinCompatibleVersion is the minimum config version compatible with this code
	// V2.0 (register_groups) and V2.1 (devices) are both supported
	MinCompatibleVersion = "2.0"
)

// VersionInfo contains version metadata from config file
type VersionInfo struct {
	Version string `yaml:"version"`
}

// ValidateVersion checks if the config file version is compatible
func ValidateVersion(fileVersion string) error {
	if fileVersion == "" {
		return fmt.Errorf("configuration file missing 'version' field. Expected version: %s", CurrentVersion)
	}

	// Support both 2.0 and 2.1
	if fileVersion != "2.0" && fileVersion != "2.1" {
		return fmt.Errorf("incompatible configuration version: %s (expected: %s, minimum: %s)",
			fileVersion, CurrentVersion, MinCompatibleVersion)
	}

	return nil
}

// IsCompatible checks if a version string is compatible with current parser
func IsCompatible(version string) bool {
	return version == CurrentVersion
}
