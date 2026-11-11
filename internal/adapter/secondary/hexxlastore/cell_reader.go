package hexxlastore

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/ports/secondary"
)

// CellReaderAdapter implements [secondary.CellReader] using (*hexxladb.DB).View and Tx Query/Search APIs.
type CellReaderAdapter struct {
	live *LiveDB
}

// NewCellReaderAdapter wraps a [LiveDB] (caller owns [LiveDB.Close]).
func NewCellReaderAdapter(live *LiveDB) *CellReaderAdapter {
	return &CellReaderAdapter{live: live}
}

// SearchCells implements [secondary.CellReader].
func (a *CellReaderAdapter) SearchCells(ctx context.Context, q *domain.CellSearchCommand, embedding []float32) (domain.CellHitsResponse, error) {
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
		return viewSnapshot(db, q.Snapshot, func(tx *hexxladb.Tx) error {
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

// GetCell reads the visible cell at one exact coordinate, including tombstone absence.
func (a *CellReaderAdapter) GetCell(ctx context.Context, coord domain.AxialCoord) (domain.CellLookupResponse, error) {
	if a == nil || a.live == nil {
		return domain.CellLookupResponse{}, fmt.Errorf("hexxlastore cell reader: nil database")
	}
	key, err := hexxladb.Pack(hexxladb.Coord{Q: coord.Q, R: coord.R})
	if err != nil {
		return domain.CellLookupResponse{}, fmt.Errorf("pack coordinate: %w", err)
	}
	var out domain.CellLookupResponse
	err = a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			rec, found, err := tx.GetCell(key)
			if err != nil {
				return err
			}
			if !found {
				return nil
			}
			view := hexxladb.CellView{Coord: hexxladb.Coord{Q: coord.Q, R: coord.R}, RawContent: rec.RawContent, Tags: rec.Tags, Provenance: rec.Provenance, Validity: rec.Validity}
			if rec.ClusterHint != nil {
				hint, err := hexxladb.Unpack(*rec.ClusterHint)
				if err != nil {
					return err
				}
				view.ClusterHint = &hint
			}
			hit := viewToCellHit(&view, 0, "")
			out = domain.CellLookupResponse{Found: true, Cell: &hit}
			return ctx.Err()
		})
	})
	if err != nil {
		return domain.CellLookupResponse{}, fmt.Errorf("hexxlastore get cell: %w", err)
	}
	return out, nil
}

func toCellQuery(q *domain.CellSearchCommand) (hexxladb.CellQuery, error) {
	if q == nil {
		return hexxladb.CellQuery{}, errors.New("hexxlastore: nil query command")
	}
	text := q.Query
	if q.Mode == domain.SearchHybrid {
		text = q.LexicalQuery
	}
	hq := hexxladb.CellQuery{
		Query:                   text,
		RequireTags:             slices.Clone(q.RequireTags),
		AnyTags:                 slices.Clone(q.AnyTags),
		ExcludeTags:             slices.Clone(q.ExcludeTags),
		SourceID:                q.SourceID,
		MinConfidence:           q.MinConfidence,
		MaxConfidence:           q.MaxConfidence,
		Radius:                  q.Radius,
		MaxResults:              q.MaxResults,
		MaxScanRows:             q.MaxScanRows,
		EmbeddingCandidateLimit: q.EmbeddingCandidateLimit,
		Explain:                 q.Explain,
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
		ClusterHint: domainCoordPtr(v.ClusterHint),
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

func domainCoordPtr(coord *hexxladb.Coord) *domain.AxialCoord {
	if coord == nil {
		return nil
	}
	return new(domain.AxialCoord(*coord))
}
