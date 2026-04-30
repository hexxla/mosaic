package config

import (
	"testing"
)

func TestLoadOllamaFromEnv_defaults(t *testing.T) {
	t.Setenv(EnvOllamaURL, "")
	t.Setenv(EnvEmbedModel, "")
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

func TestResolveOllama_yamlOverridesEnv(t *testing.T) {
	t.Setenv(EnvOllamaURL, "http://env-only:11434")
	t.Setenv(EnvEmbedModel, "env-model")

	got, err := ResolveOllama(OllamaResolveInput{
		YAMLBaseURL:    "http://from-yaml:2345",
		YAMLEmbedModel: "yaml-embed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Base.String() != "http://from-yaml:2345" {
		t.Fatalf("base %s", got.Base.String())
	}
	if got.Model != "yaml-embed" {
		t.Fatalf("model %s", got.Model)
	}
}

func TestResolveOllama_flagOverridesYAML(t *testing.T) {
	t.Setenv(EnvOllamaURL, "")
	t.Setenv(EnvEmbedModel, "")

	got, err := ResolveOllama(OllamaResolveInput{
		FlagBaseURL:    "http://flag:1",
		FlagEmbedModel: "flag-model",
		YAMLBaseURL:    "http://yaml:2",
		YAMLEmbedModel: "yaml-model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Base.String() != "http://flag:1" || got.Model != "flag-model" {
		t.Fatalf("got base=%s model=%s", got.Base.String(), got.Model)
	}
}
