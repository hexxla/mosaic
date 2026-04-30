package primary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// SeamLifecycle exposes seam discovery and mutation for MCP tools.
type SeamLifecycle interface {
	FindSeams(ctx context.Context, q *domain.FindSeamsQuery) (domain.FindSeamsResponse, error)
	MarkConflict(ctx context.Context, cellA, cellB domain.AxialCoord, reason string) error
	MarkSupersedes(ctx context.Context, superseder, superseded domain.AxialCoord, reason string) error
	ResolveSeam(ctx context.Context, seamID, resolutionStatus, resolutionNote string) error
}
