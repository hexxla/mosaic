package services

import (
	"context"
	"fmt"
	"math"
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

// CellRetrievalService owns retrieval mode and bounds; storage owns scoring.
type CellRetrievalService struct {
	reader       secondary.CellReader
	ann          secondary.EmbeddingANN
	embedder     secondary.TextEmbedder
	embeddingDim uint16
}

// NewCellRetrievalService composes structured reads, pure-vector reads and optional text embedding.
func NewCellRetrievalService(reader secondary.CellReader, ann secondary.EmbeddingANN, embedder secondary.TextEmbedder, embeddingDim uint16) *CellRetrievalService {
	return &CellRetrievalService{reader: reader, ann: ann, embedder: embedder, embeddingDim: embeddingDim}
}

// SearchCells keeps lexical, pure-vector, and confidence-aware hybrid scores distinct.
func (s *CellRetrievalService) SearchCells(ctx context.Context, command *domain.CellSearchCommand) (domain.CellHitsResponse, error) {
	if s == nil || s.reader == nil {
		return domain.CellHitsResponse{}, fmt.Errorf("cell retrieval: nil dependencies")
	}
	if err := ctx.Err(); err != nil {
		return domain.CellHitsResponse{}, err
	}
	q, err := normalizeCellSearch(command)
	if err != nil {
		return domain.CellHitsResponse{}, err
	}
	var vector []float32
	if q.Mode != domain.SearchLexical {
		if s.embedder == nil || s.embeddingDim == 0 {
			return domain.CellHitsResponse{}, fmt.Errorf("cell retrieval: %s mode requires an embedder and a configured database dimension", q.Mode)
		}
		if q.Mode == domain.SearchSemantic && s.ann == nil {
			return domain.CellHitsResponse{}, fmt.Errorf("cell retrieval: semantic index is unavailable")
		}
		vector, err = s.embedder.Embed(ctx, q.Query, int(s.embeddingDim))
		if err != nil {
			return domain.CellHitsResponse{}, fmt.Errorf("cell retrieval embed: %w", err)
		}
	}
	var out domain.CellHitsResponse
	if q.Mode == domain.SearchSemantic {
		out.Hits, err = s.ann.Search(ctx, vector, q.MaxResults, q.MinScore)
		if out.Hits == nil && err == nil {
			out.Hits = []domain.CellHit{}
		}
	} else {
		out, err = s.reader.SearchCells(ctx, q, vector)
	}
	if err != nil {
		return domain.CellHitsResponse{}, fmt.Errorf("cell retrieval search: %w", err)
	}
	out.Mode = q.Mode
	out.RetrievalHint = domain.RetrievalHintLexicalOrANN
	return out, nil
}

// GetCell reads one identity without search scope or ranking assumptions.
func (s *CellRetrievalService) GetCell(ctx context.Context, coord domain.AxialCoord) (domain.CellLookupResponse, error) {
	if s == nil || s.reader == nil {
		return domain.CellLookupResponse{}, fmt.Errorf("cell retrieval: nil dependencies")
	}
	if err := ctx.Err(); err != nil {
		return domain.CellLookupResponse{}, err
	}
	out, err := s.reader.GetCell(ctx, coord)
	if err != nil {
		return domain.CellLookupResponse{}, fmt.Errorf("cell retrieval get: %w", err)
	}
	return out, nil
}

func normalizeCellSearch(command *domain.CellSearchCommand) (*domain.CellSearchCommand, error) {
	if command == nil {
		return nil, fmt.Errorf("cell retrieval: nil search command")
	}
	q := *command
	q.Query = strings.TrimSpace(q.Query)
	if q.Mode == "" {
		q.Mode = domain.SearchLexical
	}
	switch q.Mode {
	case domain.SearchLexical, domain.SearchSemantic, domain.SearchHybrid:
	default:
		return nil, fmt.Errorf("cell retrieval: mode must be lexical, semantic, or hybrid")
	}
	switch q.SortBy {
	case domain.CellQuerySortUnspecified, domain.CellQuerySortScore, domain.CellQuerySortConfidence, domain.CellQuerySortRecency, domain.CellQuerySortCoord:
	default:
		return nil, fmt.Errorf("cell retrieval: invalid sort_by %q", q.SortBy)
	}
	if q.Radius < 0 || q.MaxScanRows < 0 {
		return nil, fmt.Errorf("cell retrieval: radius and max_scan_rows must be non-negative")
	}
	for _, v := range []float64{q.MinConfidence, q.MaxConfidence} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
			return nil, fmt.Errorf("cell retrieval: confidence bounds must be finite values in 0..1")
		}
	}
	if math.IsNaN(q.MinScore) || math.IsInf(q.MinScore, 0) {
		return nil, fmt.Errorf("cell retrieval: min_score must be finite")
	}
	if q.Mode != domain.SearchHybrid && q.LexicalQuery != "" {
		return nil, fmt.Errorf("cell retrieval: lexical_query applies only to hybrid mode")
	}
	if q.Mode != domain.SearchSemantic && q.MinScore != 0 {
		return nil, fmt.Errorf("cell retrieval: min_score applies only to semantic mode")
	}
	if q.Mode != domain.SearchLexical && q.Query == "" {
		return nil, fmt.Errorf("cell retrieval: query is required for %s mode", q.Mode)
	}
	if q.Mode == domain.SearchSemantic {
		if len(q.RequireTags)+len(q.AnyTags)+len(q.ExcludeTags) > 0 || q.SourceID != "" || q.Center != nil || q.Radius != 0 || q.MinConfidence != 0 || q.MaxConfidence != 0 || q.After != nil || q.Before != nil || q.MaxScanRows != 0 || q.Explain || (q.SortBy != "" && q.SortBy != domain.CellQuerySortScore) {
			return nil, fmt.Errorf("cell retrieval: semantic mode preserves pure vector scoring; use hybrid mode for filters, scan bounds, or alternate sorting")
		}
	} else {
		if q.Mode == domain.SearchLexical && q.MaxScanRows == 0 {
			q.MaxScanRows = maxCellScanRows
		}
		if q.Mode == domain.SearchHybrid && q.MaxScanRows > 0 && q.Radius == 0 && len(q.RequireTags) == 0 && strings.TrimSpace(q.SourceID) == "" {
			return nil, fmt.Errorf("cell retrieval: hybrid max_scan_rows requires require_tags, source_id, or a positive radius")
		}
		q.MaxScanRows = min(q.MaxScanRows, maxCellScanRows)
	}
	if q.MaxResults <= 0 {
		q.MaxResults = defaultCellReadMaxResults
	}
	q.MaxResults = min(q.MaxResults, maxCellReadMaxResults)
	return &q, nil
}

var _ primary.CellRetrieval = (*CellRetrievalService)(nil)
