package secondary

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// CellReader runs HexxlaDB read transactions for structured cell queries (indexed paths).
// When embedding is non-empty, it is passed to Tx.QueryCells / Tx.SearchCells for hybrid
// ANN + predicate retrieval; nil or empty means no embedding signal.
type CellReader interface {
	QueryCells(ctx context.Context, q *domain.CellQueryCommand, embedding []float32) ([]domain.CellHit, error)
	SearchCells(ctx context.Context, q *domain.CellSearchCommand, embedding []float32) ([]domain.CellHit, error)
}
