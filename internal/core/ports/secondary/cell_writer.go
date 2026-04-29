package secondary

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// CellWriter runs HexxlaDB mutations inside DB.Update (PutCell, PutEmbedding, DeleteCell).
type CellWriter interface {
	PutCell(ctx context.Context, cmd *domain.PutCellCommand) error
	PutEmbedding(ctx context.Context, coord domain.AxialCoord, vec []float32) error
	DeleteCell(ctx context.Context, cmd *domain.DeleteCellCommand) error
}
