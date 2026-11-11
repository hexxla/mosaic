package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/services"
)

type stubCellReader struct {
	last   *domain.CellSearchCommand
	vector []float32
	err    error
	lookup domain.CellLookupResponse
}

func (s *stubCellReader) SearchCells(_ context.Context, q *domain.CellSearchCommand, v []float32) (domain.CellHitsResponse, error) {
	copy := *q
	s.last = &copy
	s.vector = v
	return domain.CellHitsResponse{Hits: []domain.CellHit{}}, s.err
}
func (s *stubCellReader) GetCell(context.Context, domain.AxialCoord) (domain.CellLookupResponse, error) {
	return s.lookup, s.err
}

type queryEmbedder struct {
	calls int
	text  string
	err   error
}

func (e *queryEmbedder) Embed(_ context.Context, text string, dim int) ([]float32, error) {
	e.calls++
	e.text = text
	return make([]float32, dim), e.err
}

type queryANN struct {
	calls int
	score float64
	err   error
}

func (a *queryANN) Search(context.Context, []float32, int, float64) ([]domain.CellHit, error) {
	a.calls++
	return []domain.CellHit{{Score: a.score}}, a.err
}

func (a *queryANN) SearchWithStats(ctx context.Context, v []float32, limit int, score float64, _ int) (domain.CellHitsResponse, error) {
	hits, err := a.Search(ctx, v, limit, score)
	return domain.CellHitsResponse{Hits: hits}, err
}

func TestUnifiedRetrievalPaths(t *testing.T) {
	for _, mode := range []domain.SearchMode{domain.SearchLexical, domain.SearchSemantic, domain.SearchHybrid} {
		t.Run(string(mode), func(t *testing.T) {
			r := &stubCellReader{}
			e := &queryEmbedder{}
			a := &queryANN{score: 0.87}
			s := services.NewCellRetrievalService(r, a, e, 2)
			cmd := &domain.CellSearchCommand{Query: "concept", Mode: mode}
			if mode == domain.SearchHybrid {
				cmd.LexicalQuery = "literal"
				cmd.RequireTags = []string{"tag"}
			}
			out, err := s.SearchCells(t.Context(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			if out.Mode != mode {
				t.Fatalf("mode=%q", out.Mode)
			}
			if mode == domain.SearchSemantic {
				if a.calls != 1 || r.last != nil || out.Hits[0].Score != 0.87 {
					t.Fatal("semantic path changed pure vector scoring")
				}
			} else if r.last == nil || a.calls != 0 {
				t.Fatal("structured path bypassed reader")
			}
			if mode == domain.SearchLexical {
				if e.calls != 0 || r.last.Radius != 0 || r.last.MaxScanRows != 20_000 {
					t.Fatal("lexical discovery must be global, bounded, and model independent")
				}
			} else if e.calls != 1 || e.text != "concept" {
				t.Fatal("wrong embedding cue")
			}
			if cmd.MaxResults != 0 || cmd.MaxScanRows != 0 {
				t.Fatal("normalization mutated caller")
			}
		})
	}
}

func TestSearchBoundsAndValidationBeforeEmbedding(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  domain.CellSearchCommand
		rows int
		bad  bool
	}{
		{name: "lexical requested", cmd: domain.CellSearchCommand{MaxScanRows: 1}, rows: 1},
		{name: "lexical capped", cmd: domain.CellSearchCommand{MaxScanRows: 30_000}, rows: 20_000},
		{name: "negative", cmd: domain.CellSearchCommand{MaxScanRows: -1}, bad: true},
		{name: "hybrid adaptive", cmd: domain.CellSearchCommand{Mode: domain.SearchHybrid, Query: "cue"}},
		{name: "hybrid exact", cmd: domain.CellSearchCommand{Mode: domain.SearchHybrid, Query: "cue", SourceID: "source", MaxScanRows: 1}, rows: 1},
		{name: "hybrid unbounded exact", cmd: domain.CellSearchCommand{Mode: domain.SearchHybrid, Query: "cue", MaxScanRows: 1}, bad: true},
		{name: "semantic filters", cmd: domain.CellSearchCommand{Mode: domain.SearchSemantic, Query: "cue", RequireTags: []string{"tag"}}, bad: true},
		{name: "bad mode", cmd: domain.CellSearchCommand{Mode: "unknown"}, bad: true},
		{name: "bad sort", cmd: domain.CellSearchCommand{SortBy: "unknown"}, bad: true},
		{name: "missing cue", cmd: domain.CellSearchCommand{Mode: domain.SearchSemantic}, bad: true},
		{name: "literal in lexical", cmd: domain.CellSearchCommand{LexicalQuery: "invalid"}, bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &stubCellReader{}
			e := &queryEmbedder{}
			s := services.NewCellRetrievalService(r, &queryANN{}, e, 2)
			_, err := s.SearchCells(t.Context(), &tc.cmd)
			if tc.bad {
				if err == nil || e.calls != 0 || r.last != nil {
					t.Fatalf("invalid request reached I/O: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if r.last.MaxScanRows != tc.rows || r.last.MaxResults != 20 {
				t.Fatalf("normalized: %+v", r.last)
			}
		})
	}
}

func TestRetrievalDependencyErrorsAndCancellation(t *testing.T) {
	cause := errors.New("source failed")
	r := &stubCellReader{err: cause}
	s := services.NewCellRetrievalService(r, nil, nil, 0)
	if _, err := s.SearchCells(t.Context(), &domain.CellSearchCommand{}); !errors.Is(err, cause) {
		t.Fatalf("reader cause: %v", err)
	}
	if _, err := s.GetCell(t.Context(), domain.AxialCoord{}); !errors.Is(err, cause) {
		t.Fatalf("lookup cause: %v", err)
	}
	if _, err := s.SearchCells(t.Context(), &domain.CellSearchCommand{Mode: domain.SearchSemantic, Query: "cue"}); err == nil {
		t.Fatal("embedding dependencies missing")
	}
	if _, err := s.SearchCells(t.Context(), nil); err == nil {
		t.Fatal("nil command accepted")
	}
	var absent *services.CellRetrievalService
	if _, err := absent.GetCell(t.Context(), domain.AxialCoord{}); err == nil {
		t.Fatal("nil service accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.SearchCells(ctx, &domain.CellSearchCommand{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	r.err = nil
	r.lookup = domain.CellLookupResponse{Found: true, Cell: &domain.CellHit{RawContent: "exact"}}
	out, err := s.GetCell(t.Context(), domain.AxialCoord{Q: 100})
	if err != nil || !out.Found || out.Cell.RawContent != "exact" {
		t.Fatalf("lookup=%+v %v", out, err)
	}
}
