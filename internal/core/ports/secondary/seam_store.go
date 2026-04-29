package secondary

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// SeamStore reads and writes seam records via HexxlaDB (FindSeams, MarkConflict, MarkSupersedes, ResolveSeam).
type SeamStore interface {
	FindSeams(ctx context.Context, q *domain.FindSeamsQuery) ([]domain.SeamHit, error)
	MarkConflict(ctx context.Context, a, b domain.AxialCoord, reason string) error
	MarkSupersedes(ctx context.Context, superseder, superseded domain.AxialCoord, reason string) error
	ResolveSeam(ctx context.Context, seamID, resolutionStatus, resolutionNote string) error
}
