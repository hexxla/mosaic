package config

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hexxla/mcp-ratchet/pkg/ratchet"
	"github.com/hexxla/mcp-ratchet/pkg/ratchet/adapters"
)

// EnvRatchetConfigFile selects an optional custom workflow policy.
const EnvRatchetConfigFile = "MOSAIC_RATCHET_CONFIG_FILE"

//go:embed ratchet_policy.yaml
var bundledRatchetPolicy []byte

// ResolveRatchetConfigPath gives an explicit flag precedence over the environment.
func ResolveRatchetConfigPath(flagValue string) string {
	if value := strings.TrimSpace(flagValue); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(EnvRatchetConfigFile))
}

// LoadRatchetPolicy loads the bundled policy unless an operator supplies a file.
// Enabling Ratchet is decided separately so a custom file always implies opt-in.
func LoadRatchetPolicy(path string) (ratchet.Policy, error) {
	if strings.TrimSpace(path) == "" {
		return adapters.LoadPolicy(bytes.NewReader(bundledRatchetPolicy))
	}
	p := filepath.Clean(strings.TrimSpace(path))
	//nolint:gosec // The operator selects this startup policy file.
	file, err := os.Open(p)
	if err != nil {
		return ratchet.Policy{}, fmt.Errorf("open ratchet policy: %w", err)
	}
	info, statErr := file.Stat()
	if statErr != nil || !info.Mode().IsRegular() {
		return ratchet.Policy{}, errors.Join(fmt.Errorf("ratchet policy %q must be a regular file", p), statErr, file.Close())
	}
	policy, loadErr := adapters.LoadPolicy(file)
	if err := errors.Join(loadErr, file.Close()); err != nil {
		return ratchet.Policy{}, fmt.Errorf("load ratchet policy: %w", err)
	}
	return policy, nil
}
