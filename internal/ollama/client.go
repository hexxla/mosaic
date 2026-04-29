// Package ollama provides a minimal HTTP client for Ollama's /api/embeddings endpoint (local inference).
// Composition roots (cmd/mosaic-mcp, cmd/mosaic-seed) and adapters construct [Client] with config from env.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultPingTimeout bounds the readiness GET against the Ollama base URL.
	DefaultPingTimeout = 5 * time.Second
	// DefaultEmbedTimeout bounds each embeddings POST (large prompts / cold model load).
	DefaultEmbedTimeout = 120 * time.Second
)

// embeddingsRequest matches Ollama POST /api/embeddings.
type embeddingsRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// embeddingsResponse matches the JSON body from Ollama /api/embeddings.
type embeddingsResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Client calls Ollama over HTTP. Zero values are invalid; set Base, Model, and HTTP (or use [NewClient]).
type Client struct {
	HTTP *http.Client
	Base *url.URL
	// Model is the embedding model name (e.g. all-minilm).
	Model string
}

// NewClient returns a Client with HTTP defaults suitable for local Ollama.
func NewClient(base *url.URL, model string) *Client {
	return &Client{
		HTTP:  &http.Client{Timeout: DefaultEmbedTimeout},
		Base:  base,
		Model: model,
	}
}

// Ping checks that Ollama responds on the base URL (same idea as hexxladb examples).
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.Base == nil {
		return fmt.Errorf("ollama: nil client or base URL")
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	pingCtx, cancel := context.WithTimeout(ctx, DefaultPingTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(pingCtx, http.MethodGet, c.Base.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("ollama ping request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama not reachable at %q: %w", c.Base.String(), err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama ping %q: status %s", c.Base.String(), resp.Status)
	}
	return nil
}

// Embed returns a vector of length expectedDim (validated after decode). Use the database's
// [github.com/hexxla/hexxladb.DB.EmbeddingDimension] as expectedDim when targeting HexxlaDB.
func (c *Client) Embed(ctx context.Context, prompt string, expectedDim int) ([]float32, error) {
	if c == nil || c.Base == nil {
		return nil, fmt.Errorf("ollama: nil client or base URL")
	}
	if expectedDim <= 0 {
		return nil, fmt.Errorf("ollama: expectedDim must be positive")
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	embedURL := strings.TrimSuffix(c.Base.String(), "/") + "/api/embeddings"
	body, err := json.Marshal(embeddingsRequest{Model: c.Model, Prompt: prompt})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	embedCtx, cancel := context.WithTimeout(ctx, DefaultEmbedTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(embedCtx, http.MethodPost, embedURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ollama embed body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embed status %s: %s", resp.Status, bytes.TrimSpace(raw))
	}
	var result embeddingsResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode ollama embed: %w", err)
	}
	if len(result.Embedding) != expectedDim {
		return nil, fmt.Errorf(
			"ollama embedding length %d, want %d for model %q (must match HexxlaDB EmbeddingDimension)",
			len(result.Embedding), expectedDim, c.Model,
		)
	}
	out := make([]float32, len(result.Embedding))
	for i := range result.Embedding {
		out[i] = float32(result.Embedding[i])
	}
	return out, nil
}
