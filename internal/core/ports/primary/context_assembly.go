package primary

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// ContextAssembly loads budgeted lattice neighbourhoods from seed coordinates (LoadContextPackFrom).
type ContextAssembly interface {
	LoadFromSeeds(ctx context.Context, cmd *domain.LoadContextPackCommand) (domain.ContextPackResponse, error)
}
