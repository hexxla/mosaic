package config_test

import (
	"testing"

	"github.com/sploitzberg/mosaic/internal/config"
)

func TestParseMosaicConfigYAML_auto_maintain_sets_default_retain_when_pruning(t *testing.T) {
	t.Parallel()
	const raw = `version: 1
database:
  auto_maintain_after_cell_delete:
    enabled: true
allow_delete_cell: false
`
	cfg, err := config.ParseMosaicConfigYAML([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.DeleteAutoMaintain.Enabled {
		t.Fatal("want auto maintain enabled")
	}
	if !cfg.DeleteAutoMaintain.Prune || !cfg.DeleteAutoMaintain.Compact {
		t.Fatalf("want default prune and compact enabled, got prune=%v compact=%v", cfg.DeleteAutoMaintain.Prune, cfg.DeleteAutoMaintain.Compact)
	}
	if cfg.MVCCRetainCommitsBehindHead != config.DefaultAutoMaintainRetainCommits {
		t.Fatalf("retain commits: got %d want %d", cfg.MVCCRetainCommitsBehindHead, config.DefaultAutoMaintainRetainCommits)
	}
	if cfg.DeleteAutoMaintain.DebounceAfterDelete != config.DefaultDebounceAfterDelete {
		t.Fatalf("debounce: got %v want %v", cfg.DeleteAutoMaintain.DebounceAfterDelete, config.DefaultDebounceAfterDelete)
	}
}

func TestParseMosaicConfigYAML_auto_maintain_debounce_zero_immediate(t *testing.T) {
	t.Parallel()
	const raw = `version: 1
database:
  auto_maintain_after_cell_delete:
    enabled: true
    debounce_after_delete_ms: 0
allow_delete_cell: false
`
	cfg, err := config.ParseMosaicConfigYAML([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DeleteAutoMaintain.DebounceAfterDelete != 0 {
		t.Fatalf("debounce: got %v want 0", cfg.DeleteAutoMaintain.DebounceAfterDelete)
	}
}

func TestParseMVCCPruneProfile(t *testing.T) {
	t.Parallel()
	p, err := config.ParseMVCCPruneProfile("low-latency")
	if err != nil {
		t.Fatal(err)
	}
	if string(p) != "low-latency" {
		t.Fatalf("got %q", p)
	}
}
