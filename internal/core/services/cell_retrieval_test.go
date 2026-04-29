package services_test

import (
	"context"
	"testing"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/services"
)

type stubCellReader struct {
	lastQuery     *domain.CellQueryCommand
	lastSearch    *domain.CellSearchCommand
	lastEmbedding []float32
	queryErr      error
	searchErr     error
}

func (s *stubCellReader) captureEmb(emb []float32) {
	if len(emb) == 0 {
		s.lastEmbedding = nil
		return
	}
	s.lastEmbedding = append([]float32(nil), emb...)
}

func (s *stubCellReader) QueryCells(_ context.Context, q *domain.CellQueryCommand, emb []float32) ([]domain.CellHit, error) {
	if q != nil {
		c := *q
		s.lastQuery = &c
	}
	s.captureEmb(emb)
	if s.queryErr != nil {
		return nil, s.queryErr
	}
	return nil, nil
}

func (s *stubCellReader) SearchCells(_ context.Context, q *domain.CellSearchCommand, emb []float32) ([]domain.CellHit, error) {
	if q != nil {
		c := *q
		s.lastSearch = &c
	}
	s.captureEmb(emb)
	if s.searchErr != nil {
		return nil, s.searchErr
	}
	return nil, nil
}

type stubEmbedder struct{}

func (stubEmbedder) Embed(_ context.Context, text string, dim int) ([]float32, error) {
	out := make([]float32, dim)
	if dim > 0 {
		out[0] = 1
	}
	return out, nil
}

func TestCellRetrievalService_QueryCells_invalidSort(t *testing.T) {
	t.Parallel()
	svc := services.NewCellRetrievalService(&stubCellReader{}, nil, 0)
	_, err := svc.QueryCells(t.Context(), &domain.CellQueryCommand{
		SortBy: domain.CellQuerySort("invalid"),
	})
	if err == nil {
		t.Fatal("expected error for invalid sort")
	}
}

func TestCellRetrievalService_QueryCells_capsMaxResults(t *testing.T) {
	t.Parallel()
	stub := &stubCellReader{}
	svc := services.NewCellRetrievalService(stub, nil, 0)
	_, err := svc.QueryCells(t.Context(), &domain.CellQueryCommand{
		SortBy:     domain.CellQuerySortScore,
		MaxResults: 500,
	})
	if err != nil {
		t.Fatal(err)
	}
	const capMax = 100
	if stub.lastQuery.MaxResults != capMax {
		t.Fatalf("max results: got %d want %d", stub.lastQuery.MaxResults, capMax)
	}
}

func TestCellRetrievalService_SearchCells_capsMaxResults(t *testing.T) {
	t.Parallel()
	stub := &stubCellReader{}
	svc := services.NewCellRetrievalService(stub, nil, 0)
	_, err := svc.SearchCells(t.Context(), &domain.CellSearchCommand{
		MaxResults: 999,
	})
	if err != nil {
		t.Fatal(err)
	}
	const capMax = 100
	if stub.lastSearch.MaxResults != capMax {
		t.Fatalf("max results: got %d want %d", stub.lastSearch.MaxResults, capMax)
	}
}

func TestCellRetrievalService_nilDeps(t *testing.T) {
	t.Parallel()
	svc := services.NewCellRetrievalService(nil, nil, 0)
	_, err := svc.QueryCells(t.Context(), &domain.CellQueryCommand{SortBy: domain.CellQuerySortScore})
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = svc.SearchCells(t.Context(), &domain.CellSearchCommand{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCellRetrievalService_embedQueryText_requiresDim(t *testing.T) {
	t.Parallel()
	stub := &stubCellReader{}
	svc := services.NewCellRetrievalService(stub, stubEmbedder{}, 0)
	_, err := svc.QueryCells(t.Context(), &domain.CellQueryCommand{
		SortBy:         domain.CellQuerySortScore,
		EmbedQueryText: "hello",
	})
	if err == nil {
		t.Fatal("expected error when embedding dimension is 0")
	}
}

func TestCellRetrievalService_embedQueryText_requiresEmbedder(t *testing.T) {
	t.Parallel()
	stub := &stubCellReader{}
	svc := services.NewCellRetrievalService(stub, nil, 384)
	_, err := svc.QueryCells(t.Context(), &domain.CellQueryCommand{
		SortBy:         domain.CellQuerySortScore,
		EmbedQueryText: "hello",
	})
	if err == nil {
		t.Fatal("expected error when embedder is nil")
	}
}

func TestCellRetrievalService_embedQueryText_passesVectorToReader(t *testing.T) {
	t.Parallel()
	stub := &stubCellReader{}
	svc := services.NewCellRetrievalService(stub, stubEmbedder{}, 4)
	_, err := svc.QueryCells(t.Context(), &domain.CellQueryCommand{
		SortBy:         domain.CellQuerySortScore,
		EmbedQueryText: "hybrid test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if stub.lastEmbedding == nil || len(stub.lastEmbedding) != 4 {
		t.Fatalf("embedding: got %#v want len 4", stub.lastEmbedding)
	}
	if stub.lastEmbedding[0] != 1 {
		t.Fatalf("embedding marker")
	}
}

func TestCellRetrievalService_search_embedQueryText_passesVectorToReader(t *testing.T) {
	t.Parallel()
	stub := &stubCellReader{}
	svc := services.NewCellRetrievalService(stub, stubEmbedder{}, 2)
	_, err := svc.SearchCells(t.Context(), &domain.CellSearchCommand{
		Query:          "lit",
		EmbedQueryText: "semantic cue",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(stub.lastEmbedding) != 2 {
		t.Fatalf("embedding len: got %d want 2", len(stub.lastEmbedding))
	}
}
