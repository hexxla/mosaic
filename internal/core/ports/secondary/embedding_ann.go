package secondary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// EmbeddingANN searches stored vectors and resolves their application payloads.
type EmbeddingANN interface {
	Search(ctx context.Context, vector []float32, maxResults int, minScore float64) ([]domain.EmbeddingMatch, error)
}
