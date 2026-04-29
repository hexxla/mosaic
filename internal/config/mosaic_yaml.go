package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// EnvPolicyFile is the environment variable for the config path when the -policy flag is not set.
const EnvPolicyFile = "MOSAIC_POLICY_FILE"

// MosaicConfigLoaded is the validated result of parsing config YAML (or defaults).
type MosaicConfigLoaded struct {
	Retention       RetentionPolicy
	AllowDeleteCell bool
}

// retentionYAMLFields is the retention subsection (capture_mode, enforcement, notes).
type retentionYAMLFields struct {
	CaptureMode string                    `yaml:"capture_mode"`
	Enforcement retentionEnforcementField `yaml:"enforcement"`
	Notes       string                    `yaml:"notes"`
}

// rawMosaicConfig is the top-level YAML document.
type rawMosaicConfig struct {
	Version int `yaml:"version"`

	Retention *retentionYAMLFields `yaml:"retention"`
	// PersistencePolicy is deprecated; use retention.
	PersistencePolicy *retentionYAMLFields `yaml:"persistence_policy"`

	CaptureMode string                    `yaml:"capture_mode"`
	Enforcement retentionEnforcementField `yaml:"enforcement"`
	Notes       string                    `yaml:"notes"`

	AllowDeleteCell *bool `yaml:"allow_delete_cell"`
}

// DefaultMosaicConfig is the in-process default when no file is loaded.
// Retention matches an omitted capture_mode in YAML (see [DefaultRetentionPolicy]): llm_curates, enforcement off.
func DefaultMosaicConfig() MosaicConfigLoaded {
	return MosaicConfigLoaded{
		Retention:       DefaultRetentionPolicy(),
		AllowDeleteCell: false,
	}
}

// LoadMosaicConfigFromFile reads and validates a Mosaic config YAML file.
func LoadMosaicConfigFromFile(path string) (MosaicConfigLoaded, error) {
	if strings.TrimSpace(path) == "" {
		return MosaicConfigLoaded{}, errors.New("mosaic config: empty path")
	}
	p := filepath.Clean(strings.TrimSpace(path))
	if p == "." {
		return MosaicConfigLoaded{}, errors.New("mosaic config: invalid path")
	}
	//nolint:gosec // G304: path is supplied by the operator (flag or MOSAIC_POLICY_FILE), not remote input.
	data, err := os.ReadFile(p)
	if err != nil {
		return MosaicConfigLoaded{}, fmt.Errorf("mosaic config: read %q: %w", p, err)
	}
	return ParseMosaicConfigYAML(data)
}

// ParseMosaicConfigYAML decodes YAML into [MosaicConfigLoaded].
// Supports retention (preferred), persistence_policy (deprecated), or flat legacy keys at root.
func ParseMosaicConfigYAML(data []byte) (MosaicConfigLoaded, error) {
	var raw rawMosaicConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return MosaicConfigLoaded{}, fmt.Errorf("mosaic config: yaml: %w", err)
	}

	if raw.Version != MosaicConfigVersion {
		return MosaicConfigLoaded{}, fmt.Errorf("mosaic config: version must be %d", MosaicConfigVersion)
	}

	allowDelete := false
	if raw.AllowDeleteCell != nil {
		allowDelete = *raw.AllowDeleteCell
	}

	hasRetKey := raw.Retention != nil
	hasLegKey := raw.PersistencePolicy != nil
	hasRetCM := hasRetKey && strings.TrimSpace(raw.Retention.CaptureMode) != ""
	hasLegCM := hasLegKey && strings.TrimSpace(raw.PersistencePolicy.CaptureMode) != ""
	hasFlat := strings.TrimSpace(raw.CaptureMode) != ""

	if hasRetCM && hasLegCM {
		return MosaicConfigLoaded{}, errors.New("mosaic config: use either retention or persistence_policy, not both")
	}
	if hasFlat && (hasRetCM || hasLegCM) {
		return MosaicConfigLoaded{}, errors.New("mosaic config: use either retention/persistence_policy section or flat capture_mode, not both")
	}

	var fields retentionYAMLFields
	switch {
	case hasRetCM:
		fields = *raw.Retention
	case hasLegCM:
		fields = *raw.PersistencePolicy
	case hasFlat:
		fields = retentionYAMLFields{
			CaptureMode: raw.CaptureMode,
			Enforcement: raw.Enforcement,
			Notes:       raw.Notes,
		}
	case hasRetKey:
		fields = *raw.Retention
	case hasLegKey:
		fields = *raw.PersistencePolicy
	default:
		fields = retentionYAMLFields{}
	}

	rt, err := normalizeRetentionPolicy(fields.CaptureMode, fields.Enforcement, fields.Notes)
	if err != nil {
		return MosaicConfigLoaded{}, err
	}

	return MosaicConfigLoaded{
		Retention:       rt,
		AllowDeleteCell: allowDelete,
	}, nil
}

func normalizeRetentionPolicy(captureMode string, enforcement retentionEnforcementField, notes string) (RetentionPolicy, error) {
	modeStr := strings.TrimSpace(captureMode)
	if modeStr == "" {
		d := DefaultRetentionPolicy()
		d.Notes = strings.TrimSpace(notes)
		if enforcement.present {
			d.Enforcement = enforcement.value
		}
		return d, nil
	}

	mode := CaptureMode(modeStr)
	switch mode {
	case CaptureModeLLMCurates, CaptureModeSaveAllTurns, CaptureModeUserOnly,
		CaptureModeAssistantOnly, CaptureModeNone:
	default:
		return RetentionPolicy{}, fmt.Errorf("mosaic config: invalid capture_mode %q", captureMode)
	}

	enf := enforcement.resolved(PolicyEnforcementOff)

	return RetentionPolicy{
		Version:     MosaicConfigVersion,
		CaptureMode: mode,
		Enforcement: enf,
		Notes:       strings.TrimSpace(notes),
	}, nil
}

// ResolveMosaicConfigPath returns -policy if set, else MOSAIC_POLICY_FILE, else empty.
func ResolveMosaicConfigPath(flagValue string) string {
	flagValue = strings.TrimSpace(flagValue)
	if flagValue != "" {
		return flagValue
	}
	return strings.TrimSpace(os.Getenv(EnvPolicyFile))
}

// ResolvePersistencePolicyPath is an alias for [ResolveMosaicConfigPath].
func ResolvePersistencePolicyPath(flagValue string) string {
	return ResolveMosaicConfigPath(flagValue)
}

// ParsePersistencePolicyYAML parses YAML and returns only retention (backward compat for tests).
func ParsePersistencePolicyYAML(data []byte) (RetentionPolicy, error) {
	cfg, err := ParseMosaicConfigYAML(data)
	if err != nil {
		return RetentionPolicy{}, err
	}
	return cfg.Retention, nil
}

// LoadPersistencePolicyFromFile loads a config file and returns only retention (backward compat).
func LoadPersistencePolicyFromFile(path string) (RetentionPolicy, error) {
	cfg, err := LoadMosaicConfigFromFile(path)
	if err != nil {
		return RetentionPolicy{}, err
	}
	return cfg.Retention, nil
}

// DefaultPersistencePolicy returns default retention (backward compat name).
func DefaultPersistencePolicy() RetentionPolicy {
	return DefaultRetentionPolicy()
}
