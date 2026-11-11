package config

import "testing"

func TestRuntimeEngineOptions(t *testing.T) {
	cfg, err := ParseMosaicConfigYAML([]byte("version: 1\ndatabase:\n  page_cache_bytes: -1\n  use_primary_fdatasync: true\n  changelog_lazy: true\n"))
	if err != nil || cfg.PageCacheBytes != -1 || !cfg.UsePrimaryFdatasync || !cfg.ChangelogLazy {
		t.Fatalf("options=%+v,%v", cfg, err)
	}
	if _, err := ParseMosaicConfigYAML([]byte("version: 1\ndatabase:\n  page_cache_bytes: -2\n")); err == nil {
		t.Fatal("invalid cache size accepted")
	}
}
