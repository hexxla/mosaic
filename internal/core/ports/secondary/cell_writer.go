package secondary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// CellWriter runs HexxlaDB mutations inside DB.Update (PutCell, PutEmbedding, DeleteCell).
type CellWriter interface {
	PutCell(ctx context.Context, cmd *domain.PutCellCommand) error
	PutEmbedding(ctx context.Context, coord domain.AxialCoord, vec []float32) error
	// DeleteCell removes a visible cell when present; CellRemoved reports whether one was removed.
	DeleteCell(ctx context.Context, cmd *domain.DeleteCellCommand) (cellRemoved bool, err error)
}
