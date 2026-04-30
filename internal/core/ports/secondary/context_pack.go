package secondary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// ContextPackLoader runs HexxlaDB Tx.LoadContextPackFrom inside a View.
type ContextPackLoader interface {
	LoadFromSeeds(ctx context.Context, cmd *domain.LoadContextPackCommand) (domain.ContextPackResponse, error)
}
