package config

import (
	"os"
	"path/filepath"
	"testing"
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

func TestLoadRatchetConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ratchet.yaml")
	if err := os.WriteFile(path, []byte("rules: []\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loaded, err := LoadRatchetConfig(path)
	if err != nil {
		t.Fatalf("load regular config: %v", err)
	}
	if loaded.Path != path {
		t.Fatalf("loaded path = %q, want %q", loaded.Path, path)
	}

	if _, err := LoadRatchetConfig(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Fatal("missing config unexpectedly succeeded")
	}
	if _, err := LoadRatchetConfig(dir); err == nil {
		t.Fatal("directory config unexpectedly succeeded")
	}
}
