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
	// DatabasePassphrase is optional at-rest encryption passphrase from YAML (prefer MOSAIC_DB_PASSPHRASE or -db-passphrase).
	DatabasePassphrase string
	Retrieval          RetrievalBudgetConfig
}

// retentionYAMLFields is the retention subsection (capture_mode, enforcement, notes).
type retentionYAMLFields struct {
	CaptureMode string                    `yaml:"capture_mode"`
	Enforcement retentionEnforcementField `yaml:"enforcement"`
	Notes       string                    `yaml:"notes"`
}

// databaseYAMLFields is optional DB encryption hints (avoid committing real secrets in Git).
type databaseYAMLFields struct {
	Passphrase string `yaml:"passphrase"`
}

// retrievalYAMLFields is optional limits for MCP read-tool payload budgeting (see docs in configs/config.yaml).
type retrievalYAMLFields struct {
	SessionApproxTokenBudget int      `yaml:"session_approx_token_budget"`
	BytesPerApproxToken      *float64 `yaml:"bytes_per_approx_token,omitempty"`
}

// rawMosaicConfig is the top-level YAML document.
type rawMosaicConfig struct {
	Version int `yaml:"version"`

	Database *databaseYAMLFields `yaml:"database"`

	Retention *retentionYAMLFields `yaml:"retention"`
	// PersistencePolicy is deprecated; use retention.
	PersistencePolicy *retentionYAMLFields `yaml:"persistence_policy"`

	CaptureMode string                    `yaml:"capture_mode"`
	Enforcement retentionEnforcementField `yaml:"enforcement"`
	Notes       string                    `yaml:"notes"`

	AllowDeleteCell *bool `yaml:"allow_delete_cell"`

	Retrieval *retrievalYAMLFields `yaml:"retrieval,omitempty"`
}

// DefaultMosaicConfig is the in-process default when no file is loaded.
// Retention matches an omitted capture_mode in YAML (see [DefaultRetentionPolicy]): llm_curates, enforcement off.
func DefaultMosaicConfig() MosaicConfigLoaded {
	return MosaicConfigLoaded{
		Retention:       DefaultRetentionPolicy(),
		AllowDeleteCell: false,
		Retrieval:       DefaultRetrievalBudgetConfig(),
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

	dbPass := ""
	if raw.Database != nil {
		dbPass = strings.TrimSpace(raw.Database.Passphrase)
	}

	retrieval := DefaultRetrievalBudgetConfig()
	if raw.Retrieval != nil {
		if raw.Retrieval.SessionApproxTokenBudget < 0 {
			return MosaicConfigLoaded{}, errors.New("mosaic config: retrieval.session_approx_token_budget must be >= 0")
		}
		retrieval.SessionApproxTokenBudget = raw.Retrieval.SessionApproxTokenBudget
		if raw.Retrieval.BytesPerApproxToken != nil {
			if err := validateRetrievalBytesPerApproxToken(*raw.Retrieval.BytesPerApproxToken); err != nil {
				return MosaicConfigLoaded{}, err
			}
			retrieval.BytesPerApproxToken = *raw.Retrieval.BytesPerApproxToken
		}
	}

	return MosaicConfigLoaded{
		Retention:          rt,
		AllowDeleteCell:    allowDelete,
		DatabasePassphrase: dbPass,
		Retrieval:          retrieval,
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
