package secondary

import "context"

// TextEmbedder turns natural language into a float32 vector of a fixed embedding dimension (e.g. Ollama).
type TextEmbedder interface {
	Embed(ctx context.Context, text string, expectedDim int) ([]float32, error)
}
