package primary

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// CellMutation exposes write paths for MCP tools (outside mosaic-seed).
type CellMutation interface {
	PutCell(ctx context.Context, cmd *domain.PutCellCommand) error
	PutEmbedding(ctx context.Context, cmd *domain.PutEmbeddingCommand) error
	DeleteCell(ctx context.Context, cmd *domain.DeleteCellCommand) error
}
