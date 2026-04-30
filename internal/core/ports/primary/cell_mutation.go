package primary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// CellMutation exposes write paths for MCP tools (outside mosaic-seed).
type CellMutation interface {
	PutCell(ctx context.Context, cmd *domain.PutCellCommand) error
	PutEmbedding(ctx context.Context, cmd *domain.PutEmbeddingCommand) error
	DeleteCell(ctx context.Context, cmd *domain.DeleteCellCommand) (cellRemoved bool, err error)
}
