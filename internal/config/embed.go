package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	// EnvOllamaURL is the env var for the Ollama HTTP root (e.g. http://127.0.0.1:11434).
	EnvOllamaURL = "MOSAIC_OLLAMA_URL" //nolint:gosec // G101: documented env name, not a secret
	// EnvEmbedModel is the env var for the embedding model name (e.g. all-minilm).
	EnvEmbedModel  = "MOSAIC_EMBED_MODEL"
	defaultOllama  = "http://127.0.0.1:11434"
	defaultEmbedML = "all-minilm"
)

// Ollama holds HTTP embedding endpoints for MCP tools (never stores secrets).
type Ollama struct {
	// Base is the parsed Ollama root URL (e.g. http://127.0.0.1:11434).
	Base *url.URL
	// Model is the embeddings model name (e.g. all-minilm).
	Model string
}

// OllamaResolveInput selects Ollama settings with explicit precedence ([ResolveOllama]).
type OllamaResolveInput struct {
	// FlagBaseURL — non-empty from CLI (e.g. mosaic-seed -ollama).
	FlagBaseURL string
	// FlagEmbedModel — non-empty from CLI (e.g. mosaic-seed -embed-model).
	FlagEmbedModel string
	// YAMLBaseURL / YAMLEmbedModel come from optional policy **`ollama:`** (**`base_url`**, **`embed_model`**).
	YAMLBaseURL    string
	YAMLEmbedModel string
}

// ResolveOllama builds [Ollama] with precedence:
// for URL: CLI flag → YAML base_url → env MOSAIC_OLLAMA_URL → built-in default;
// for model: CLI flag → YAML embed_model → env MOSAIC_EMBED_MODEL → built-in default.
func ResolveOllama(in OllamaResolveInput) (Ollama, error) {
	raw := strings.TrimSpace(in.FlagBaseURL)
	if raw == "" {
		raw = strings.TrimSpace(in.YAMLBaseURL)
	}
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv(EnvOllamaURL))
	}
	if raw == "" {
		raw = defaultOllama
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Ollama{}, fmt.Errorf("ollama URL %q: %w", raw, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return Ollama{}, fmt.Errorf("ollama URL %q must have scheme and host (e.g. http://127.0.0.1:11434)", raw)
	}

	model := strings.TrimSpace(in.FlagEmbedModel)
	if model == "" {
		model = strings.TrimSpace(in.YAMLEmbedModel)
	}
	if model == "" {
		model = strings.TrimSpace(os.Getenv(EnvEmbedModel))
	}
	if model == "" {
		model = defaultEmbedML
	}
	return Ollama{Base: u, Model: model}, nil
}

// LoadOllamaFromEnv reads MOSAIC_OLLAMA_URL and MOSAIC_EMBED_MODEL with defaults (no YAML seed).
//
// Equivalent to ResolveOllama(OllamaResolveInput{}).
func LoadOllamaFromEnv() (Ollama, error) {
	return ResolveOllama(OllamaResolveInput{})
}
