package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/services"
)

type stubFacetEdgeReader struct {
	getFacetResp domain.GetFacetResponse
	listFacets   domain.ListFacetsForCellResponse
	getEdge      domain.GetEdgeResponse
	listEdges    domain.ListEdgesFromResponse

	err error

	listMaxHook func(int)
}

func (s *stubFacetEdgeReader) GetFacet(_ context.Context, _ domain.AxialCoord, _ uint8) (domain.GetFacetResponse, error) {
	if s.err != nil {
		return domain.GetFacetResponse{}, s.err
	}
	return s.getFacetResp, nil
}

func (s *stubFacetEdgeReader) ListFacetsForCell(_ context.Context, _ domain.AxialCoord) (domain.ListFacetsForCellResponse, error) {
	if s.err != nil {
		return domain.ListFacetsForCellResponse{}, s.err
	}
	return s.listFacets, nil
}

func (s *stubFacetEdgeReader) GetEdge(_ context.Context, from, to domain.AxialCoord, _ string) (domain.GetEdgeResponse, error) {
	_ = from
	_ = to
	if s.err != nil {
		return domain.GetEdgeResponse{}, s.err
	}
	return s.getEdge, nil
}

func (s *stubFacetEdgeReader) ListEdgesFrom(_ context.Context, _ domain.AxialCoord, maxEdges int) (domain.ListEdgesFromResponse, error) {
	if s.listMaxHook != nil {
		s.listMaxHook(maxEdges)
	}
	if s.err != nil {
		return domain.ListEdgesFromResponse{}, s.err
	}
	return s.listEdges, nil
}

func TestFacetEdgeReadService_GetFacet_facet_range(t *testing.T) {
	t.Parallel()
	svc := services.NewFacetEdgeReadService(&stubFacetEdgeReader{})
	_, err := svc.GetFacet(t.Context(), domain.AxialCoord{Q: 0, R: 0}, 6)
	if err == nil {
		t.Fatal("expected error for facet_id > MaxFacetID")
	}
}

func TestFacetEdgeReadService_GetEdge_empty_relation(t *testing.T) {
	t.Parallel()
	svc := services.NewFacetEdgeReadService(&stubFacetEdgeReader{})
	_, err := svc.GetEdge(t.Context(), domain.AxialCoord{Q: 0, R: 0}, domain.AxialCoord{Q: 1, R: 0}, "  ")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFacetEdgeReadService_ListEdgesFrom_caps_defaults(t *testing.T) {
	t.Parallel()
	var gotMax int
	stub := &stubFacetEdgeReader{
		listEdges: domain.ListEdgesFromResponse{Edges: []domain.EdgeBullet{}},
		listMaxHook: func(m int) {
			gotMax = m
		},
	}
	svc := services.NewFacetEdgeReadService(stub)
	_, err := svc.ListEdgesFrom(t.Context(), domain.AxialCoord{Q: 0, R: 0}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotMax != 50 {
		t.Fatalf("default max edges: got %d want 50", gotMax)
	}
	_, err = svc.ListEdgesFrom(t.Context(), domain.AxialCoord{Q: 0, R: 0}, 9999)
	if err != nil {
		t.Fatal(err)
	}
	if gotMax != 200 {
		t.Fatalf("cap max edges: got %d want 200", gotMax)
	}
}

func TestFacetEdgeReadService_nil(t *testing.T) {
	t.Parallel()
	svc := services.NewFacetEdgeReadService(nil)
	_, err := svc.GetFacet(t.Context(), domain.AxialCoord{}, 0)
	if err == nil {
		t.Fatal("expected nil reader error")
	}
}

func TestFacetEdgeReadService_propagate(t *testing.T) {
	t.Parallel()
	want := errors.New("x")
	stub := &stubFacetEdgeReader{err: want}
	svc := services.NewFacetEdgeReadService(stub)
	_, err := svc.GetFacet(t.Context(), domain.AxialCoord{Q: 0, R: 0}, 0)
	if !errors.Is(err, want) {
		t.Fatalf("%v", err)
	}
}
