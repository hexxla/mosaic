package primary

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// EmbeddingSearch is the driving port for semantic (embedding ANN) search over the lattice.
type EmbeddingSearch interface {
	Search(ctx context.Context, q domain.EmbeddingSearchQuery) (domain.EmbeddingSearchResponse, error)
}
