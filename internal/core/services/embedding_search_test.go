package services

import (
	"context"
	"errors"
	"testing"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

type stubANN struct {
	out domain.EmbeddingSearchResponse
	err error
}

func (s stubANN) Search(context.Context, domain.EmbeddingSearchQuery) (domain.EmbeddingSearchResponse, error) {
	return s.out, s.err
}

type recordANN struct {
	last   domain.EmbeddingSearchQuery
	result domain.EmbeddingSearchResponse
	err    error
}

func (r *recordANN) Search(_ context.Context, q domain.EmbeddingSearchQuery) (domain.EmbeddingSearchResponse, error) {
	r.last = q
	return r.result, r.err
}

func TestEmbeddingSearchService_Search_success_trim_and_cap(t *testing.T) {
	t.Parallel()

	rec := &recordANN{result: domain.EmbeddingSearchResponse{Query: "x", Matches: []domain.EmbeddingMatch{{Score: 1}}}}
	svc := NewEmbeddingSearchService(rec)
	out, err := svc.Search(t.Context(), domain.EmbeddingSearchQuery{
		Text:       "  hello world  ",
		MaxResults: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.last.Text != "hello world" {
		t.Fatalf("text: %q", rec.last.Text)
	}
	if rec.last.MaxResults != maxEmbeddingMaxResults {
		t.Fatalf("max: got %d want %d", rec.last.MaxResults, maxEmbeddingMaxResults)
	}
	if out.RetrievalHint != domain.RetrievalHintLexicalOrANN {
		t.Fatalf("retrieval hint should steer toward context pack")
	}
}

func TestEmbeddingSearchService_Search_empty_query(t *testing.T) {
	t.Parallel()

	svc := NewEmbeddingSearchService(stubANN{})
	_, err := svc.Search(t.Context(), domain.EmbeddingSearchQuery{Text: "   "})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEmbeddingSearchService_Search_nil_ann(t *testing.T) {
	t.Parallel()

	svc := NewEmbeddingSearchService(nil)
	_, err := svc.Search(t.Context(), domain.EmbeddingSearchQuery{Text: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEmbeddingSearchService_Search_delegate_error(t *testing.T) {
	t.Parallel()

	stub := stubANN{err: errors.New("boom")}
	svc := NewEmbeddingSearchService(stub)
	_, err := svc.Search(t.Context(), domain.EmbeddingSearchQuery{Text: "q"})
	if err == nil {
		t.Fatal("expected error")
	}
}
