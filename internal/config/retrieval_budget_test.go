package config_test

import (
	"testing"

	"github.com/sploitzberg/mosaic/internal/config"
)

func TestParseMosaicConfigYAML_retrieval_budget(t *testing.T) {
	t.Parallel()
	raw := `version: 1
retention:
  capture_mode: llm_curates
  enforcement: false
retrieval:
  session_approx_token_budget: 50000
  bytes_per_approx_token: 4
`
	cfg, err := config.ParseMosaicConfigYAML([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Retrieval.SessionApproxTokenBudget != 50000 {
		t.Fatalf("SessionApproxTokenBudget: got %d", cfg.Retrieval.SessionApproxTokenBudget)
	}
	if cfg.Retrieval.BytesPerApproxToken != 4 {
		t.Fatalf("BytesPerApproxToken: got %v", cfg.Retrieval.BytesPerApproxToken)
	}
}

func TestParseMosaicConfigYAML_retrieval_bytes_per_token_invalid(t *testing.T) {
	t.Parallel()
	raw := `version: 1
retention:
  capture_mode: llm_curates
retrieval:
  session_approx_token_budget: 1
  bytes_per_approx_token: 1
`
	_, err := config.ParseMosaicConfigYAML([]byte(raw))
	if err == nil {
		t.Fatal("expected error for bytes_per_approx_token out of range")
	}
}

func TestParseMosaicConfigYAML_retrievalRejectsNonFiniteAndNegativeRatios(t *testing.T) {
	t.Parallel()
	for _, value := range []string{".nan", ".inf", "-.inf", "-1"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			_, err := config.ParseMosaicConfigYAML([]byte("version: 1\nretrieval:\n  bytes_per_approx_token: " + value + "\n"))
			if err == nil {
				t.Fatalf("invalid ratio %s was accepted", value)
			}
		})
	}
}

func TestParseMosaicConfigYAML_retrievalPreservesDefaultAndBoundaryRatios(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"0", "2", "4", "16"} {
		if _, err := config.ParseMosaicConfigYAML([]byte("version: 1\nretrieval:\n  bytes_per_approx_token: " + value + "\n")); err != nil {
			t.Fatalf("valid ratio %s: %v", value, err)
		}
	}
}
