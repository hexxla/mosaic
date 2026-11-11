package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hexxla/mosaic/internal/config"
	"github.com/hexxla/mosaic/internal/core/domain"
)

func TestParseMosaicConfigYAML_ollama_section(t *testing.T) {
	t.Parallel()
	raw := `
version: 1
ollama:
  base_url: http://ollama.local:11434
  embed_model: custom-embed
retention:
  capture_mode: llm_curates
`
	cfg, err := config.ParseMosaicConfigYAML([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OllamaBaseURL != "http://ollama.local:11434" {
		t.Fatalf("base_url: %q", cfg.OllamaBaseURL)
	}
	if cfg.OllamaEmbedModel != "custom-embed" {
		t.Fatalf("embed_model: %q", cfg.OllamaEmbedModel)
	}
}

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

func TestLoadPersistencePolicyFromFile_roundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := []byte(`version: 1
retention:
  capture_mode: none
  enforcement: true
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadMosaicConfigFromFile(path)
	p := cfg.Retention
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

func TestMosaicConfigRejectsMixedRetentionForms(t *testing.T) {
	for _, mixed := range []string{
		"retention: {enforcement: true}\npersistence_policy: {capture_mode: none}",
		"retention: {capture_mode: none}\nenforcement: true",
		"persistence_policy: {capture_mode: none}\nnotes: ''",
	} {
		if _, err := config.ParseMosaicConfigYAML([]byte("version: 1\n" + mixed)); err == nil {
			t.Errorf("mixed forms were accepted: %s", mixed)
		}
	}
}
