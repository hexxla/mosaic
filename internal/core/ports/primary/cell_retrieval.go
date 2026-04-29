package primary

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// CellRetrieval exposes structured cell reads for MCP tools (QueryCells / SearchCells).
type CellRetrieval interface {
	QueryCells(ctx context.Context, q *domain.CellQueryCommand) ([]domain.CellHit, error)
	SearchCells(ctx context.Context, q *domain.CellSearchCommand) ([]domain.CellHit, error)
}
