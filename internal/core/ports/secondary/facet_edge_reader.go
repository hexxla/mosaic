package secondary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// FacetEdgeReader performs read-only facet and edge lookups (Hexxla View).
type FacetEdgeReader interface {
	GetFacet(ctx context.Context, coord domain.AxialCoord, facetID uint8) (domain.GetFacetResponse, error)
	ListFacetsForCell(ctx context.Context, coord domain.AxialCoord) (domain.ListFacetsForCellResponse, error)

	GetEdge(ctx context.Context, from, to domain.AxialCoord, relationType string) (domain.GetEdgeResponse, error)
	ListEdgesFrom(ctx context.Context, from domain.AxialCoord, maxEdges int) (domain.ListEdgesFromResponse, error)
}
