package ollamaembed

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/secondary"
	ollamac "github.com/sploitzberg/go-llm-project-structure/internal/ollama"
)

type textEmbedder struct {
	c *ollamac.Client
}

// NewTextEmbedder adapts the Ollama HTTP client to [secondary.TextEmbedder]. Returns nil if c is nil.
func NewTextEmbedder(c *ollamac.Client) secondary.TextEmbedder {
	if c == nil {
		return nil
	}
	return textEmbedder{c: c}
}

// Embed implements [secondary.TextEmbedder].
func (e textEmbedder) Embed(ctx context.Context, text string, expectedDim int) ([]float32, error) {
	return e.c.Embed(ctx, text, expectedDim)
}

var _ secondary.TextEmbedder = textEmbedder{}
