package secondary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// CellWriter runs HexxlaDB mutations inside DB.Update (PutCell, PutEmbedding, DeleteCell).
type CellWriter interface {
	// LookupPutCell checks a keyed request before any embedding work.
	LookupPutCell(ctx context.Context, cmd *domain.PutCellCommand) (domain.PutCellMutationResult, bool, error)
	// PutCell resolves placement and writes atomically, returning the actual coordinate.
	PutCell(ctx context.Context, cmd *domain.PutCellCommand, embedding []float32) (domain.PutCellMutationResult, error)
	// PutEmbedding checks an optional content hash atomically with the write.
	PutEmbedding(ctx context.Context, coord domain.AxialCoord, vec []float32, expectedContentSHA256 string) error
	// DeleteCell removes a visible cell when present; CellRemoved reports whether one was removed.
	DeleteCell(ctx context.Context, cmd *domain.DeleteCellCommand) (cellRemoved bool, err error)
}
