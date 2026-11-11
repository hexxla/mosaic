package primary

import (
	"context"

	"github.com/hexxla/mosaic/internal/core/domain"
)

// CellRetrieval provides bounded discovery and exact coordinate lookup.
type CellRetrieval interface {
	SearchCells(ctx context.Context, q *domain.CellSearchCommand) (domain.CellHitsResponse, error)
	GetCell(ctx context.Context, coord domain.AxialCoord) (domain.CellLookupResponse, error)
}
