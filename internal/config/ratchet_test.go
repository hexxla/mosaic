package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveRatchetConfigPath(t *testing.T) {
	t.Setenv(EnvRatchetConfigFile, " env.yaml ")
	if got := ResolveRatchetConfigPath(" flag.yaml "); got != "flag.yaml" {
		t.Fatalf("flag path = %q, want flag.yaml", got)
	}
	if got := ResolveRatchetConfigPath(""); got != "env.yaml" {
		t.Fatalf("environment path = %q, want env.yaml", got)
	}

	t.Setenv(EnvRatchetConfigFile, " ")
	if got := ResolveRatchetConfigPath(" "); got != "" {
		t.Fatalf("empty path = %q, want empty", got)
	}
}

func TestLoadRatchetPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ratchet.yaml")
	if err := os.WriteFile(path, []byte("rules: []\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loaded, err := LoadRatchetPolicy(path)
	if err != nil {
		t.Fatalf("load regular config: %v", err)
	}
	if len(loaded.Rules) != 0 {
		t.Fatalf("unexpected custom rules: %+v", loaded)
	}

	if _, err := LoadRatchetPolicy(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Fatal("missing config unexpectedly succeeded")
	}
	if _, err := LoadRatchetPolicy(dir); err == nil {
		t.Fatal("directory config unexpectedly succeeded")
	}
}

func TestBundledRatchetPolicy(t *testing.T) {
	policy, err := LoadRatchetPolicy("")
	if err != nil {
		t.Fatal(err)
	}
	if policy.CompletionTTL != 10*time.Minute {
		t.Fatalf("completion TTL=%v", policy.CompletionTTL)
	}
	foundContext := false
	for _, rule := range policy.Rules {
		if rule.Tool == "mosaic_hexxla_load_context_pack" {
			foundContext = rule.TTL == 5*time.Minute
		}
	}
	if !foundContext {
		t.Fatal("context completion TTL must be five minutes")
	}
	if len(policy.Rules) != 13 {
		t.Fatalf("bundled rules=%d", len(policy.Rules))
	}
}
