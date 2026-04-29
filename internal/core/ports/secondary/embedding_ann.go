package secondary

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// EmbeddingANN performs query embedding + [Tx.SearchByEmbedding]-style retrieval against the engine.
// Implemented only at the Hexxla adapter edge (Ollama + hexxladb).
type EmbeddingANN interface {
	Search(ctx context.Context, q domain.EmbeddingSearchQuery) (domain.EmbeddingSearchResponse, error)
}
