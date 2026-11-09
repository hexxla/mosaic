package hexxlastore

import (
	"context"
	"errors"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// CellReaderAdapter implements [secondary.CellReader] using (*hexxladb.DB).View and Tx Query/Search APIs.
type CellReaderAdapter struct {
	live *LiveDB
}

// NewCellReaderAdapter wraps a [LiveDB] (caller owns [LiveDB.Close]).
func NewCellReaderAdapter(live *LiveDB) *CellReaderAdapter {
	return &CellReaderAdapter{live: live}
}

// QueryCells implements [secondary.CellReader].
func (a *CellReaderAdapter) QueryCells(ctx context.Context, q *domain.CellQueryCommand, embedding []float32) (domain.CellHitsResponse, error) {
	if a == nil || a.live == nil {
		return domain.CellHitsResponse{}, fmt.Errorf("hexxlastore cell reader: nil database")
	}
	hq, err := toCellQuery(q)
	if err != nil {
		return domain.CellHitsResponse{}, err
	}
	hq.Embedding = embedding
	var out domain.CellHitsResponse
	err = a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			res, stats, err := tx.QueryCellsWithStats(ctx, hq)
			out.ScanLimitReached = errors.Is(err, hexxladb.ErrQueryScanLimit)
			if err != nil && !out.ScanLimitReached {
				return fmt.Errorf("tx QueryCells: %w", err)
			}
			out.EmbeddingPath = string(stats.EmbeddingPath)
			out.EmbeddingCandidates = stats.EmbeddingCandidates
			out.EmbeddingCandidateLimitReached = stats.EmbeddingCandidateLimitReached
			out.Hits = make([]domain.CellHit, 0, len(res))
			for i := range res {
				r := &res[i]
				out.Hits = append(out.Hits, viewToCellHit(&r.Cell, r.Score, r.Explanation))
			}
			return nil
		})
	})
	if err != nil {
		return domain.CellHitsResponse{}, fmt.Errorf("hexxlastore query cells: %w", err)
	}
	return out, nil
}

// SearchCells preserves the search tool's implicit 32-ring scope.
func (a *CellReaderAdapter) SearchCells(ctx context.Context, q *domain.CellSearchCommand, embedding []float32) (domain.CellHitsResponse, error) {
	if q == nil {
		return domain.CellHitsResponse{}, fmt.Errorf("hexxlastore cell reader: nil search command")
	}
	radius := q.Radius
	if radius <= 0 {
		radius = q.MaxScanRadius
		if radius <= 0 {
			radius = 32
		}
	}
	return a.QueryCells(ctx, &domain.CellQueryCommand{
		Query: q.Query, RequireTags: q.RequireTags, AnyTags: q.AnyTags,
		MinConfidence: q.MinConfidence, MaxConfidence: q.MaxConfidence, SourceID: q.SourceID,
		Center: q.Center, Radius: radius, MaxResults: q.MaxResults, MaxScanRows: q.MaxScanRows,
	}, embedding)
}

func toCellQuery(q *domain.CellQueryCommand) (hexxladb.CellQuery, error) {
	if q == nil {
		return hexxladb.CellQuery{}, errors.New("hexxlastore: nil query command")
	}
	hq := hexxladb.CellQuery{
		Query:         q.Query,
		RequireTags:   append([]string(nil), q.RequireTags...),
		AnyTags:       append([]string(nil), q.AnyTags...),
		ExcludeTags:   append([]string(nil), q.ExcludeTags...),
		SourceID:      q.SourceID,
		MinConfidence: q.MinConfidence,
		MaxConfidence: q.MaxConfidence,
		Radius:        q.Radius,
		MaxResults:    q.MaxResults,
		MaxScanRows:   q.MaxScanRows,
		Explain:       q.Explain,
	}
	if q.Center != nil {
		hq.Center = hexxladb.Coord{Q: q.Center.Q, R: q.Center.R}
	}
	if q.After != nil {
		hq.After = *q.After
	}
	if q.Before != nil {
		hq.Before = *q.Before
	}
	hq.SortBy = sortOrder(q.SortBy)
	return hq, nil
}

func sortOrder(s domain.CellQuerySort) hexxladb.SortOrder {
	switch s {
	case domain.CellQuerySortConfidence:
		return hexxladb.SortByConfidence
	case domain.CellQuerySortRecency:
		return hexxladb.SortByRecency
	case domain.CellQuerySortCoord:
		return hexxladb.SortByCoord
	case domain.CellQuerySortUnspecified, domain.CellQuerySortScore:
		return hexxladb.SortByScore
	default:
		return hexxladb.SortByScore
	}
}

func viewToCellHit(v *hexxladb.CellView, score float64, explanation string) domain.CellHit {
	if v == nil {
		return domain.CellHit{}
	}
	createdAt, updatedAt, validFrom, validTo := timingsFromCellView(v)
	return domain.CellHit{
		Coord:       domain.AxialCoord{Q: v.Coord.Q, R: v.Coord.R},
		RawContent:  v.RawContent,
		Tags:        append([]string(nil), v.Tags...),
		Score:       score,
		Explanation: explanation,
		SourceID:    v.Provenance.SourceID,
		Confidence:  v.Provenance.Confidence,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		ValidFrom:   validFrom,
		ValidTo:     validTo,
	}
}

// Ensure CellReaderAdapter implements secondary.CellReader.
var _ secondary.CellReader = (*CellReaderAdapter)(nil)
