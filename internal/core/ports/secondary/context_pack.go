package secondary

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// ContextPackLoader runs HexxlaDB Tx.LoadContextPackFrom inside a View.
type ContextPackLoader interface {
	LoadFromSeeds(ctx context.Context, cmd *domain.LoadContextPackCommand) (domain.ContextPackResponse, error)
}
