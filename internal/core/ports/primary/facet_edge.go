package primary

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// FacetEdge exposes facet slot and inter-cell edge writes for MCP tools.
type FacetEdge interface {
	PutFacet(ctx context.Context, cmd *domain.PutFacetCommand) error
	LinkCells(ctx context.Context, cmd *domain.LinkCellsCommand) error
}
