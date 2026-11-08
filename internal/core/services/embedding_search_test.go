package services

import (
	"context"
	"errors"
	"testing"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

type stubANN struct {
	out []domain.EmbeddingMatch
	err error
}

func (s stubANN) Search(context.Context, []float32, int, float64) ([]domain.EmbeddingMatch, error) {
	return s.out, s.err
}

type recordANN struct {
	limit  int
	result []domain.EmbeddingMatch
	err    error
}

func (r *recordANN) Search(_ context.Context, _ []float32, limit int, _ float64) ([]domain.EmbeddingMatch, error) {
	r.limit = limit
	return r.result, r.err
}

func TestEmbeddingSearchService_Search_success_trim_and_cap(t *testing.T) {
	t.Parallel()

	rec := &recordANN{result: []domain.EmbeddingMatch{{Score: 1}}}
	svc := NewEmbeddingSearchService(rec, &stubEmbedder{}, 3)
	out, err := svc.Search(t.Context(), domain.EmbeddingSearchQuery{
		Text:       "  hello world  ",
		MaxResults: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Query != "hello world" {
		t.Fatalf("text: %q", out.Query)
	}
	if rec.limit != maxEmbeddingMaxResults {
		t.Fatalf("max: got %d want %d", rec.limit, maxEmbeddingMaxResults)
	}
	if out.RetrievalHint != domain.RetrievalHintLexicalOrANN {
		t.Fatalf("retrieval hint should steer toward context pack")
	}
}

func TestEmbeddingSearchService_Search_empty_query(t *testing.T) {
	t.Parallel()

	svc := NewEmbeddingSearchService(stubANN{}, &stubEmbedder{}, 3)
	_, err := svc.Search(t.Context(), domain.EmbeddingSearchQuery{Text: "   "})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEmbeddingSearchService_Search_nil_ann(t *testing.T) {
	t.Parallel()

	svc := NewEmbeddingSearchService(nil, &stubEmbedder{}, 3)
	_, err := svc.Search(t.Context(), domain.EmbeddingSearchQuery{Text: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEmbeddingSearchService_Search_delegate_error(t *testing.T) {
	t.Parallel()

	stub := stubANN{err: errors.New("boom")}
	svc := NewEmbeddingSearchService(stub, &stubEmbedder{out: []float32{1, 2, 3}}, 3)
	_, err := svc.Search(t.Context(), domain.EmbeddingSearchQuery{Text: "q"})
	if err == nil {
		t.Fatal("expected error")
	}
}
