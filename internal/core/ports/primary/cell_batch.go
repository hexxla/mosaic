package primary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// BatchCellMutation validates and commits a group of idempotent writes.
type BatchCellMutation interface {
	PutCells(context.Context, *domain.PutCellsCommand) (domain.PutCellsResult, error)
}
