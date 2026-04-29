package config_test

import (
	"testing"

	"github.com/sploitzberg/go-llm-project-structure/internal/config"
)

func TestDefaultMosaicConfig_llm_curates_and_gates(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultMosaicConfig()
	if cfg.Retention.CaptureMode != config.CaptureModeLLMCurates {
		t.Fatalf("DefaultMosaicConfig retention capture_mode: got %q want llm_curates", cfg.Retention.CaptureMode)
	}
	if cfg.Retention.Enforcement != config.PolicyEnforcementOff {
		t.Fatalf("DefaultMosaicConfig retention enforcement: got %q want off", cfg.Retention.Enforcement)
	}
	if cfg.AllowDeleteCell {
		t.Fatal("DefaultMosaicConfig AllowDeleteCell: got true want false")
	}
	if cfg.DatabasePassphrase != "" {
		t.Fatal("DefaultMosaicConfig DatabasePassphrase: expected empty")
	}
	if cfg.Retrieval.SessionApproxTokenBudget != 0 {
		t.Fatalf("DefaultMosaicConfig Retrieval.SessionApproxTokenBudget: got %d want 0", cfg.Retrieval.SessionApproxTokenBudget)
	}
}

func TestDefaultRetentionPolicy_llm_curates(t *testing.T) {
	t.Parallel()
	p := config.DefaultRetentionPolicy()
	if p.CaptureMode != config.CaptureModeLLMCurates {
		t.Fatalf("got %q want llm_curates", p.CaptureMode)
	}
	if p.Enforcement != config.PolicyEnforcementOff {
		t.Fatalf("got %q want off", p.Enforcement)
	}
}
