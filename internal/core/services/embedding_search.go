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
	ann       secondary.EmbeddingANN
	embedder  secondary.TextEmbedder
	dimension uint16
}

// NewEmbeddingSearchService constructs an EmbeddingSearchService.
func NewEmbeddingSearchService(ann secondary.EmbeddingANN, embedder secondary.TextEmbedder, dimension uint16) *EmbeddingSearchService {
	return &EmbeddingSearchService{ann: ann, embedder: embedder, dimension: dimension}
}

// Search implements [primary.EmbeddingSearch].
func (s *EmbeddingSearchService) Search(ctx context.Context, q domain.EmbeddingSearchQuery) (domain.EmbeddingSearchResponse, error) {
	if s == nil || s.ann == nil || s.embedder == nil {
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
	if s.dimension == 0 {
		return domain.EmbeddingSearchResponse{}, fmt.Errorf("embedding search: database has no configured embedding dimension")
	}
	vector, err := s.embedder.Embed(ctx, text, int(s.dimension))
	if err != nil {
		return domain.EmbeddingSearchResponse{}, fmt.Errorf("query embed: %w", err)
	}
	matches, err := s.ann.Search(ctx, vector, limit, q.MinScore)
	if err != nil {
		return domain.EmbeddingSearchResponse{}, fmt.Errorf("embedding search: %w", err)
	}
	return domain.EmbeddingSearchResponse{Query: text, Matches: matches, RetrievalHint: domain.RetrievalHintLexicalOrANN}, nil
}

// Ensure EmbeddingSearchService implements primary.EmbeddingSearch.
var _ primary.EmbeddingSearch = (*EmbeddingSearchService)(nil)
