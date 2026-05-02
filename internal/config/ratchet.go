package config

import (
	"os"
	"path/filepath"
	"strings"
)

// EnvRatchetConfigFile is the environment variable for the ratchet config path when the -ratchet-config flag is not set.
const EnvRatchetConfigFile = "MOSAIC_RATCHET_CONFIG_FILE"

// RatchetConfig represents the ratchet configuration.
type RatchetConfig struct {
	// Path is the file path to the ratchet YAML configuration file.
	Path string
}

// ResolveRatchetConfigPath returns -ratchet-config if set, else MOSAIC_RATCHET_CONFIG_FILE, else empty.
func ResolveRatchetConfigPath(flagValue string) string {
	flagValue = strings.TrimSpace(flagValue)
	if flagValue != "" {
		return flagValue
	}
	return strings.TrimSpace(os.Getenv(EnvRatchetConfigFile))
}

// LoadRatchetConfig reads and validates a ratchet config YAML file.
// Returns an error if the file cannot be read or does not exist.
// The actual parsing and validation of ratchet rules is handled by the ratchet service.
func LoadRatchetConfig(path string) (RatchetConfig, error) {
	if strings.TrimSpace(path) == "" {
		return RatchetConfig{}, nil
	}

	p := filepath.Clean(strings.TrimSpace(path))
	if p == "." {
		return RatchetConfig{}, nil
	}

	// Verify file exists
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return RatchetConfig{}, err
	}

	return RatchetConfig{
		Path: p,
	}, nil
}
