package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sploitzberg/mosaic/internal/config"
	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func TestParseMosaicConfigYAML_database_passphrase(t *testing.T) {
	t.Parallel()
	raw := `
version: 1
database:
  passphrase: "from-yaml-only"
retention:
  capture_mode: llm_curates
`
	cfg, err := config.ParseMosaicConfigYAML([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabasePassphrase != "from-yaml-only" {
		t.Fatalf("got %q", cfg.DatabasePassphrase)
	}
}

func TestParseMosaicConfigYAML_retention_section(t *testing.T) {
	t.Parallel()
	raw := `
version: 1
retention:
  capture_mode: user_only
  enforcement: true
  notes: "team workspace"
`
	cfg, err := config.ParseMosaicConfigYAML([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Retention.CaptureMode != config.CaptureModeUserOnly || cfg.Retention.Enforcement != config.PolicyEnforcementReject {
		t.Fatalf("got %+v", cfg)
	}
	if cfg.AllowDeleteCell {
		t.Fatal("expected allow_delete_cell false by default")
	}
}

func TestParseMosaicConfigYAML_persistence_policy_deprecated(t *testing.T) {
	t.Parallel()
	raw := `
version: 1
persistence_policy:
  capture_mode: assistant_only
  enforcement: true
`
	cfg, err := config.ParseMosaicConfigYAML([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Retention.CaptureMode != config.CaptureModeAssistantOnly {
		t.Fatalf("got %+v", cfg.Retention)
	}
}

func TestParseMosaicConfigYAML_flat_and_retention_conflict(t *testing.T) {
	t.Parallel()
	raw := `
version: 1
capture_mode: user_only
retention:
  capture_mode: assistant_only
`
	_, err := config.ParseMosaicConfigYAML([]byte(raw))
	if err == nil {
		t.Fatal("expected error when flat and retention both set")
	}
}

func TestParseMosaicConfigYAML_retention_and_legacy_persistence_both(t *testing.T) {
	t.Parallel()
	raw := `
version: 1
retention:
  capture_mode: user_only
persistence_policy:
  capture_mode: assistant_only
`
	_, err := config.ParseMosaicConfigYAML([]byte(raw))
	if err == nil {
		t.Fatal("expected error when retention and persistence_policy both have capture_mode")
	}
}

func TestLoadMosaicConfigFromFile_allowDelete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`version: 1
retention:
  capture_mode: none
  enforcement: true
allow_delete_cell: true
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadMosaicConfigFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowDeleteCell {
		t.Fatal("expected allow_delete_cell true")
	}
	if err := cfg.Retention.CheckPutCell(domain.CellPutKindUserMessage); err == nil {
		t.Fatal("expected reject for user_message")
	}
}

func TestParseMosaicConfigYAML_enforcement_false(t *testing.T) {
	t.Parallel()
	raw := `
version: 1
retention:
  capture_mode: user_only
  enforcement: false
`
	cfg, err := config.ParseMosaicConfigYAML([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Retention.Enforcement != config.PolicyEnforcementOff {
		t.Fatalf("got %q want off", cfg.Retention.Enforcement)
	}
}

func TestParseMosaicConfigYAML_enforcement_invalidOn(t *testing.T) {
	t.Parallel()
	raw := `
version: 1
retention:
  capture_mode: user_only
  enforcement: on
`
	_, err := config.ParseMosaicConfigYAML([]byte(raw))
	if err == nil {
		t.Fatal("expected error for enforcement: on (use true or false)")
	}
}

func TestParsePersistencePolicyYAML_backwardCompat(t *testing.T) {
	t.Parallel()
	raw := `
version: 1
capture_mode: user_only
enforcement: reject
`
	p, err := config.ParsePersistencePolicyYAML([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if p.CaptureMode != config.CaptureModeUserOnly {
		t.Fatal(p)
	}
}

func TestLoadPersistencePolicyFromFile_roundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := []byte(`version: 1
capture_mode: none
enforcement: true
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := config.LoadPersistencePolicyFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.CheckPutCell(domain.CellPutKindUserMessage); err == nil {
		t.Fatal("expected reject for user_message under capture_mode none")
	}
	if err := p.CheckPutCell(domain.CellPutKindFact); err != nil {
		t.Fatal(err)
	}
}

func TestRetentionPolicy_EnforcementEnabled(t *testing.T) {
	t.Parallel()
	off := config.RetentionPolicy{Enforcement: config.PolicyEnforcementOff}
	if off.EnforcementEnabled() {
		t.Fatal("expected false when enforcement off")
	}
	on := config.RetentionPolicy{Enforcement: config.PolicyEnforcementReject}
	if !on.EnforcementEnabled() {
		t.Fatal("expected true when enforcement on")
	}
}
