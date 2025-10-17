package config

import "fmt"

// Config file format version constants
const (
	// CurrentVersion is the configuration version this code can parse
	CurrentVersion = "2.0"

	// MinCompatibleVersion is the minimum config version compatible with this code
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

	if fileVersion != CurrentVersion {
		// For now, we only support exact version match
		// In the future, we can add semantic versioning logic here
		return fmt.Errorf("incompatible configuration version: %s (expected: %s, minimum: %s)",
			fileVersion, CurrentVersion, MinCompatibleVersion)
	}

	return nil
}

// IsCompatible checks if a version string is compatible with current parser
func IsCompatible(version string) bool {
	return version == CurrentVersion
}
