package secondary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// CellReader provides bounded discovery and exact coordinate lookup.
type CellReader interface {
	SearchCells(ctx context.Context, q *domain.CellSearchCommand, embedding []float32) (domain.CellHitsResponse, error)
	GetCell(ctx context.Context, coord domain.AxialCoord) (domain.CellLookupResponse, error)
}
