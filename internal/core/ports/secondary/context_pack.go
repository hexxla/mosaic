package secondary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// ContextPackLoader retrieves provider-neutral HexxlaDB context candidates.
// Application budgeting is applied by the primary service.
type ContextPackLoader interface {
	LoadFromSeeds(ctx context.Context, cmd *domain.LoadContextPackCommand) (domain.ContextPackResponse, error)
}
