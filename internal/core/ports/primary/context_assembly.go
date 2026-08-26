package primary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// ContextAssembly loads and byte-budgets lattice neighbourhoods from seed coordinates.
type ContextAssembly interface {
	LoadFromSeeds(ctx context.Context, cmd *domain.LoadContextPackCommand) (domain.ContextPackResponse, error)
}
