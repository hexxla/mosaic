package config

import (
	"testing"
)

func TestLoadOllamaFromEnv_defaults(t *testing.T) {
	t.Setenv(envOllamaURL, "")
	t.Setenv(envEmbedModel, "")
	got, err := LoadOllamaFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if got.Base.String() != "http://127.0.0.1:11434" {
		t.Fatalf("base: %s", got.Base.String())
	}
	if got.Model != defaultEmbedML {
		t.Fatalf("model: %s", got.Model)
	}
}
