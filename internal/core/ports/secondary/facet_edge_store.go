package secondary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// FacetEdgeStore applies facet and edge writes inside HexxlaDB transactions.
type FacetEdgeStore interface {
	PutFacet(ctx context.Context, cmd *domain.PutFacetCommand) error
	LinkCells(ctx context.Context, cmd *domain.LinkCellsCommand) error
}
