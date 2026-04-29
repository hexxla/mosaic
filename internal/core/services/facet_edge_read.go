package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/primary"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/secondary"
)

const (
	defaultEdgesFromCap = 50
	maxEdgesFromCap     = 200
)

// FacetEdgeReadService implements primary.FacetEdgeBrowse with guards on facet id and scan caps.
type FacetEdgeReadService struct {
	reader secondary.FacetEdgeReader
}

// NewFacetEdgeReadService wires read-only facet/edge access.
func NewFacetEdgeReadService(r secondary.FacetEdgeReader) *FacetEdgeReadService {
	return &FacetEdgeReadService{reader: r}
}

// GetFacet implements [primary.FacetEdgeBrowse].
func (s *FacetEdgeReadService) GetFacet(ctx context.Context, coord domain.AxialCoord, facetID uint8) (domain.GetFacetResponse, error) {
	if s == nil || s.reader == nil {
		return domain.GetFacetResponse{}, fmt.Errorf("facet edge read: nil dependencies")
	}
	if facetID > domain.MaxFacetID {
		return domain.GetFacetResponse{}, fmt.Errorf("facet edge read: facet_id must be 0..%d", domain.MaxFacetID)
	}
	out, err := s.reader.GetFacet(ctx, coord, facetID)
	if err != nil {
		return domain.GetFacetResponse{}, fmt.Errorf("facet edge read get facet: %w", err)
	}
	return out, nil
}

// ListFacetsForCell implements [primary.FacetEdgeBrowse].
func (s *FacetEdgeReadService) ListFacetsForCell(ctx context.Context, coord domain.AxialCoord) (domain.ListFacetsForCellResponse, error) {
	if s == nil || s.reader == nil {
		return domain.ListFacetsForCellResponse{}, fmt.Errorf("facet edge read: nil dependencies")
	}
	out, err := s.reader.ListFacetsForCell(ctx, coord)
	if err != nil {
		return domain.ListFacetsForCellResponse{}, fmt.Errorf("facet edge read list facets: %w", err)
	}
	return out, nil
}

// GetEdge implements [primary.FacetEdgeBrowse].
func (s *FacetEdgeReadService) GetEdge(ctx context.Context, from, to domain.AxialCoord, relationType string) (domain.GetEdgeResponse, error) {
	if s == nil || s.reader == nil {
		return domain.GetEdgeResponse{}, fmt.Errorf("facet edge read: nil dependencies")
	}
	rt := strings.TrimSpace(relationType)
	if rt == "" {
		return domain.GetEdgeResponse{}, fmt.Errorf("facet edge read: relation_type is required")
	}
	out, err := s.reader.GetEdge(ctx, from, to, rt)
	if err != nil {
		return domain.GetEdgeResponse{}, fmt.Errorf("facet edge read get edge: %w", err)
	}
	return out, nil
}

// ListEdgesFrom implements [primary.FacetEdgeBrowse].
func (s *FacetEdgeReadService) ListEdgesFrom(ctx context.Context, from domain.AxialCoord, maxEdges int) (domain.ListEdgesFromResponse, error) {
	if s == nil || s.reader == nil {
		return domain.ListEdgesFromResponse{}, fmt.Errorf("facet edge read: nil dependencies")
	}
	if maxEdges <= 0 {
		maxEdges = defaultEdgesFromCap
	}
	if maxEdges > maxEdgesFromCap {
		maxEdges = maxEdgesFromCap
	}
	out, err := s.reader.ListEdgesFrom(ctx, from, maxEdges)
	if err != nil {
		return domain.ListEdgesFromResponse{}, fmt.Errorf("facet edge read list edges: %w", err)
	}
	return out, nil
}

var _ primary.FacetEdgeBrowse = (*FacetEdgeReadService)(nil)
