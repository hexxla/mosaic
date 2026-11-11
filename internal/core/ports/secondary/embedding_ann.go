package secondary

import (
	"context"

	"github.com/hexxla/mosaic/internal/core/domain"
)

// EmbeddingANN searches stored vectors and resolves their application payloads.
type EmbeddingANN interface {
	SearchWithStats(ctx context.Context, vector []float32, maxResults int, minScore float64, efSearch int) (domain.CellHitsResponse, error)
	Search(ctx context.Context, vector []float32, maxResults int, minScore float64) ([]domain.CellHit, error)
}
