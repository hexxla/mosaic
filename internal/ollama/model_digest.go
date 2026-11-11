package ollama

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ModelIdentity includes an operator-pinned digest when configured. Changing
// this identity requires explicit re-embedding of existing vectors.
func (c *Client) ModelIdentity() string {
	if c.ExpectedDigest == "" {
		return c.Model
	}
	return c.Model + "@sha256:" + c.ExpectedDigest
}

// VerifyModelDigest rejects missing or retagged models when digest pinning is
// enabled. Embed checks before and after inference so a changed provider result
// is discarded before any application write. This trusts the local provider's
// metadata; it is not cryptographic verification of inference execution.
func (c *Client) VerifyModelDigest(ctx context.Context) error {
	if c.ExpectedDigest == "" {
		return nil
	}
	if c.Base == nil {
		return fmt.Errorf("ollama: missing base URL for digest verification")
	}
	endpoint := c.Base.JoinPath("api", "tags")
	endpoint.Fragment, endpoint.RawFragment = "", ""
	checkCtx, cancel := context.WithTimeout(ctx, DefaultPingTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("ollama model identity request: %w", diagnosticRequestError{err})
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama model identity: %w", diagnosticRequestError{err})
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama model identity: status %s", resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxEmbeddingResponseBytes+1))
	if err != nil {
		return fmt.Errorf("ollama model identity body: %w", err)
	}
	if len(raw) > maxEmbeddingResponseBytes {
		return fmt.Errorf("ollama model identity response exceeds size limit")
	}
	var result struct {
		Models []struct {
			Name   string `json:"name"`
			Model  string `json:"model"`
			Digest string `json:"digest"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Errorf("ollama model identity JSON: %w", err)
	}
	for _, model := range result.Models {
		if model.Name != c.Model && model.Name != c.Model+":latest" && model.Model != c.Model && model.Model != c.Model+":latest" {
			continue
		}
		digest := strings.ToLower(strings.TrimPrefix(model.Digest, "sha256:"))
		if digest != c.ExpectedDigest {
			return fmt.Errorf("ollama model digest changed for %q; expected configured SHA256", c.Model)
		}
		return nil
	}
	return fmt.Errorf("ollama model %q not found for digest verification", c.Model)
}
