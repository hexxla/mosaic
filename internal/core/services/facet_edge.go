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
	maxFacetDerivedBytes = 1 << 20 // aligns with Hexxla wire MaxStringField
	maxRelationTypeBytes = 1024
)

// FacetEdgeService implements [primary.FacetEdge] with lightweight validation before the store.
type FacetEdgeService struct {
	store secondary.FacetEdgeStore
}

// NewFacetEdgeService constructs FacetEdgeService.
func NewFacetEdgeService(store secondary.FacetEdgeStore) *FacetEdgeService {
	return &FacetEdgeService{store: store}
}

// PutFacet implements [primary.FacetEdge].
func (s *FacetEdgeService) PutFacet(ctx context.Context, cmd *domain.PutFacetCommand) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("facet edge: nil dependencies")
	}
	if cmd == nil {
		return fmt.Errorf("facet edge: nil command")
	}
	if cmd.FacetID > domain.MaxFacetID {
		return fmt.Errorf("facet edge: facet_id must be 0..%d", domain.MaxFacetID)
	}
	dc := cmd.DerivedContent
	if len(dc) > maxFacetDerivedBytes {
		return fmt.Errorf("facet edge: derived_content exceeds max byte length")
	}
	if err := s.store.PutFacet(ctx, cmd); err != nil {
		return fmt.Errorf("facet edge put facet: %w", err)
	}
	return nil
}

// LinkCells implements [primary.FacetEdge].
func (s *FacetEdgeService) LinkCells(ctx context.Context, cmd *domain.LinkCellsCommand) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("facet edge: nil dependencies")
	}
	if cmd == nil {
		return fmt.Errorf("facet edge: nil command")
	}
	rt := strings.TrimSpace(cmd.RelationType)
	if rt == "" {
		return fmt.Errorf("facet edge: relation_type is required")
	}
	if len(rt) > maxRelationTypeBytes {
		return fmt.Errorf("facet edge: relation_type too long")
	}
	src := strings.TrimSpace(cmd.SourceID)
	if src == "" {
		return fmt.Errorf("facet edge: source_id is required")
	}
	if len(src) > maxSourceIDBytes {
		return fmt.Errorf("facet edge: source_id too long (max %d)", maxSourceIDBytes)
	}
	conf := cmd.Confidence
	if conf < 0 || conf > 1 {
		return fmt.Errorf("facet edge: confidence must be between 0 and 1")
	}
	stable := &domain.LinkCellsCommand{
		From:         cmd.From,
		To:           cmd.To,
		RelationType: rt,
		Weight:       cmd.Weight,
		SourceID:     src,
		Confidence:   conf,
	}
	if err := s.store.LinkCells(ctx, stable); err != nil {
		return fmt.Errorf("facet edge link cells: %w", err)
	}
	return nil
}

// Ensure FacetEdgeService implements primary.FacetEdge.
var _ primary.FacetEdge = (*FacetEdgeService)(nil)
