package secondary

import (
	"context"

	"github.com/hexxla/mosaic/internal/core/domain"
)

// BatchCellWriter commits a bounded group of individually keyed writes atomically.
type BatchCellWriter interface {
	PutCells(context.Context, *domain.PutCellsCommand, [][]float32) (domain.PutCellsResult, error)
}
