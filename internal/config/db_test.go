package config

import "testing"

func TestLoadDBFromEnv_missing(t *testing.T) {
	t.Setenv(EnvDBPath, "")
	_, err := LoadDBFromEnv()
	if err == nil {
		t.Fatal("expected error when MOSAIC_DB_PATH unset")
	}
}

func TestLoadDBFromEnv_ok(t *testing.T) {
	t.Setenv(EnvDBPath, "/tmp/test.hexxla")
	got, err := LoadDBFromEnv()
	if err != nil {
		t.Fatalf("LoadDBFromEnv: %v", err)
	}
	if got.Path != "/tmp/test.hexxla" {
		t.Fatalf("Path: got %q", got.Path)
	}
}
