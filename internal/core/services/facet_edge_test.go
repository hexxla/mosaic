package services

import (
	"context"
	"testing"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

type stubFacetEdge struct {
	putFacetCalls int
	linkCalls     int
}

func (s *stubFacetEdge) PutFacet(context.Context, *domain.PutFacetCommand) error {
	if s != nil {
		s.putFacetCalls++
	}
	return nil
}

func (s *stubFacetEdge) LinkCells(context.Context, *domain.LinkCellsCommand) error {
	if s != nil {
		s.linkCalls++
	}
	return nil
}

func TestFacetEdgeService_PutFacet_facet_id(t *testing.T) {
	t.Parallel()
	st := &stubFacetEdge{}
	svc := NewFacetEdgeService(st)
	err := svc.PutFacet(t.Context(), &domain.PutFacetCommand{
		Coord:          domain.AxialCoord{Q: 0, R: 0},
		FacetID:        6,
		DerivedContent: "x",
	})
	if err == nil {
		t.Fatal("expected error for facet_id > MaxFacetID")
	}
}

func TestFacetEdgeService_LinkCells_relation(t *testing.T) {
	t.Parallel()
	svc := NewFacetEdgeService(&stubFacetEdge{})
	if err := svc.LinkCells(t.Context(), &domain.LinkCellsCommand{
		From:         domain.AxialCoord{Q: 0, R: 0},
		To:           domain.AxialCoord{Q: 1, R: 0},
		RelationType: "   ",
		Weight:       1,
		SourceID:     "edge-src",
		Confidence:   1,
	}); err == nil {
		t.Fatal("expected error for empty relation")
	}
}
