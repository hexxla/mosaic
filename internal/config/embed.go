package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	envOllamaURL   = "MOSAIC_OLLAMA_URL"
	envEmbedModel  = "MOSAIC_EMBED_MODEL"
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

// LoadOllamaFromEnv reads MOSAIC_OLLAMA_URL and MOSAIC_EMBED_MODEL with defaults matching cmd/mosaic-seed.
func LoadOllamaFromEnv() (Ollama, error) {
	raw := strings.TrimSpace(os.Getenv(envOllamaURL))
	if raw == "" {
		raw = defaultOllama
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Ollama{}, fmt.Errorf("%s: parse URL: %w", envOllamaURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return Ollama{}, fmt.Errorf("%s must be an absolute URL with scheme and host (e.g. http://127.0.0.1:11434)", envOllamaURL)
	}
	model := strings.TrimSpace(os.Getenv(envEmbedModel))
	if model == "" {
		model = defaultEmbedML
	}
	return Ollama{Base: u, Model: model}, nil
}
