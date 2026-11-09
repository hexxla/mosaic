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
	defaultCellReadMaxResults = 20
	maxCellReadMaxResults     = 100
	maxCellScanRows           = 20_000
)

// CellRetrievalService implements [primary.CellRetrieval] on top of [secondary.CellReader].
type CellRetrievalService struct {
	reader       secondary.CellReader
	embedder     secondary.TextEmbedder
	embeddingDim uint16
}

// NewCellRetrievalService constructs a CellRetrievalService.
// embedder may be nil; hybrid retrieval (EmbedQueryText) fails at runtime unless embedder is set.
// embeddingDim must match [github.com/hexxla/hexxladb.DB.EmbeddingDimension] for hybrid queries.
func NewCellRetrievalService(reader secondary.CellReader, embedder secondary.TextEmbedder, embeddingDim uint16) *CellRetrievalService {
	return &CellRetrievalService{reader: reader, embedder: embedder, embeddingDim: embeddingDim}
}

func (s *CellRetrievalService) resolveHybridEmbedding(ctx context.Context, embedText string) ([]float32, error) {
	t := strings.TrimSpace(embedText)
	if t == "" {
		return nil, nil
	}
	if s.embeddingDim == 0 {
		return nil, fmt.Errorf("cell retrieval: embed_query_text requires a database with non-zero embedding dimension")
	}
	if s.embedder == nil {
		return nil, fmt.Errorf("cell retrieval: embed_query_text requires a text embedder (Ollama)")
	}
	vec, err := s.embedder.Embed(ctx, t, int(s.embeddingDim))
	if err != nil {
		return nil, fmt.Errorf("cell retrieval hybrid embed: %w", err)
	}
	return vec, nil
}

// QueryCells implements [primary.CellRetrieval].
func (s *CellRetrievalService) QueryCells(ctx context.Context, q *domain.CellQueryCommand) (domain.CellHitsResponse, error) {
	if s == nil || s.reader == nil {
		return domain.CellHitsResponse{}, fmt.Errorf("cell retrieval: nil dependencies")
	}
	if q == nil {
		return domain.CellHitsResponse{}, fmt.Errorf("cell retrieval: nil query command")
	}
	q2, err := normalizeCellQuery(q)
	if err != nil {
		return domain.CellHitsResponse{}, err
	}
	emb, err := s.resolveHybridEmbedding(ctx, q.EmbedQueryText)
	if err != nil {
		return domain.CellHitsResponse{}, err
	}
	out, err := s.reader.QueryCells(ctx, q2, emb)
	if err != nil {
		return domain.CellHitsResponse{}, fmt.Errorf("cell retrieval query: %w", err)
	}
	return out, nil
}

// SearchCells implements [primary.CellRetrieval].
func (s *CellRetrievalService) SearchCells(ctx context.Context, q *domain.CellSearchCommand) (domain.CellHitsResponse, error) {
	if s == nil || s.reader == nil {
		return domain.CellHitsResponse{}, fmt.Errorf("cell retrieval: nil dependencies")
	}
	if q == nil {
		return domain.CellHitsResponse{}, fmt.Errorf("cell retrieval: nil search command")
	}
	q2 := normalizeCellSearch(q)
	emb, err := s.resolveHybridEmbedding(ctx, q.EmbedQueryText)
	if err != nil {
		return domain.CellHitsResponse{}, err
	}
	out, err := s.reader.SearchCells(ctx, q2, emb)
	if err != nil {
		return domain.CellHitsResponse{}, fmt.Errorf("cell retrieval search: %w", err)
	}
	return out, nil
}

func normalizeCellQuery(q *domain.CellQueryCommand) (*domain.CellQueryCommand, error) {
	switch q.SortBy {
	case domain.CellQuerySortUnspecified, domain.CellQuerySortScore, domain.CellQuerySortConfidence,
		domain.CellQuerySortRecency, domain.CellQuerySortCoord:
	default:
		return nil, fmt.Errorf("cell retrieval: invalid sort_by %q", q.SortBy)
	}
	out := *q
	if out.MaxScanRows < 0 {
		return nil, fmt.Errorf("cell retrieval: max_scan_rows must be non-negative")
	}
	if strings.TrimSpace(out.EmbedQueryText) == "" {
		if out.MaxScanRows == 0 {
			out.MaxScanRows = maxCellScanRows
		}
	} else if out.MaxScanRows > 0 && out.Radius <= 0 && len(out.RequireTags) == 0 && strings.TrimSpace(out.SourceID) == "" {
		return nil, fmt.Errorf("cell retrieval: max_scan_rows with embed_query_text requires require_tags, source_id, or a positive radius")
	}
	out.MaxScanRows = min(out.MaxScanRows, maxCellScanRows)
	if out.MaxResults <= 0 {
		out.MaxResults = defaultCellReadMaxResults
	}
	if out.MaxResults > maxCellReadMaxResults {
		out.MaxResults = maxCellReadMaxResults
	}
	return &out, nil
}

func normalizeCellSearch(q *domain.CellSearchCommand) *domain.CellSearchCommand {
	out := *q
	if strings.TrimSpace(out.EmbedQueryText) == "" {
		out.MaxScanRows = maxCellScanRows
	}
	if out.MaxResults <= 0 {
		out.MaxResults = defaultCellReadMaxResults
	}
	if out.MaxResults > maxCellReadMaxResults {
		out.MaxResults = maxCellReadMaxResults
	}
	return &out
}

// Ensure CellRetrievalService implements primary.CellRetrieval.
var _ primary.CellRetrieval = (*CellRetrievalService)(nil)
