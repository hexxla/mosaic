package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

const (
	defaultEmbeddingMaxResults = 10
	maxEmbeddingMaxResults     = 50
)

// EmbeddingSearchService implements [primary.EmbeddingSearch] by delegating to [secondary.EmbeddingANN].
type EmbeddingSearchService struct {
	ann secondary.EmbeddingANN
}

// NewEmbeddingSearchService constructs an EmbeddingSearchService.
func NewEmbeddingSearchService(ann secondary.EmbeddingANN) *EmbeddingSearchService {
	return &EmbeddingSearchService{ann: ann}
}

// Search implements [primary.EmbeddingSearch].
func (s *EmbeddingSearchService) Search(ctx context.Context, q domain.EmbeddingSearchQuery) (domain.EmbeddingSearchResponse, error) {
	if s == nil || s.ann == nil {
		return domain.EmbeddingSearchResponse{}, fmt.Errorf("embedding search: nil dependencies")
	}
	text := strings.TrimSpace(q.Text)
	if text == "" {
		return domain.EmbeddingSearchResponse{}, fmt.Errorf("embedding search: empty query text")
	}
	limit := q.MaxResults
	if limit <= 0 {
		limit = defaultEmbeddingMaxResults
	}
	if limit > maxEmbeddingMaxResults {
		limit = maxEmbeddingMaxResults
	}
	q2 := domain.EmbeddingSearchQuery{
		Text:       text,
		MaxResults: limit,
		MinScore:   q.MinScore,
	}
	out, err := s.ann.Search(ctx, q2)
	if err != nil {
		return domain.EmbeddingSearchResponse{}, fmt.Errorf("embedding search: %w", err)
	}
	out.RetrievalHint = domain.RetrievalHintLexicalOrANN
	return out, nil
}

// Ensure EmbeddingSearchService implements primary.EmbeddingSearch.
var _ primary.EmbeddingSearch = (*EmbeddingSearchService)(nil)
