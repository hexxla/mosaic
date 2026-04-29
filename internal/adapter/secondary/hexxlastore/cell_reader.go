package hexxlastore

import (
	"context"
	"errors"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
	"github.com/sploitzberg/go-llm-project-structure/internal/core/ports/secondary"
)

// CellReaderAdapter implements [secondary.CellReader] using (*hexxladb.DB).View and Tx Query/Search APIs.
type CellReaderAdapter struct {
	db *hexxladb.DB
}

// NewCellReaderAdapter wraps an open database (caller owns Open/Close).
func NewCellReaderAdapter(db *hexxladb.DB) *CellReaderAdapter {
	return &CellReaderAdapter{db: db}
}

// QueryCells implements [secondary.CellReader].
func (a *CellReaderAdapter) QueryCells(ctx context.Context, q *domain.CellQueryCommand, embedding []float32) ([]domain.CellHit, error) {
	if a == nil || a.db == nil {
		return nil, fmt.Errorf("hexxlastore cell reader: nil database")
	}
	if q == nil {
		return nil, fmt.Errorf("hexxlastore cell reader: nil query command")
	}
	hq, err := toCellQuery(q)
	if err != nil {
		return nil, err
	}
	if len(embedding) > 0 {
		hq.Embedding = embedding
	}
	var out []domain.CellHit
	err = a.db.View(func(tx *hexxladb.Tx) error {
		res, err := tx.QueryCells(ctx, hq)
		if err != nil {
			return fmt.Errorf("tx QueryCells: %w", err)
		}
		out = make([]domain.CellHit, 0, len(res))
		for i := range res {
			r := &res[i]
			out = append(out, viewToCellHit(&r.Cell, r.Score, r.Explanation))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("hexxlastore query cells: %w", err)
	}
	return out, nil
}

// SearchCells implements [secondary.CellReader].
func (a *CellReaderAdapter) SearchCells(ctx context.Context, q *domain.CellSearchCommand, embedding []float32) ([]domain.CellHit, error) {
	if a == nil || a.db == nil {
		return nil, fmt.Errorf("hexxlastore cell reader: nil database")
	}
	if q == nil {
		return nil, fmt.Errorf("hexxlastore cell reader: nil search command")
	}
	cfg := hexxladb.CellSearchConfig{
		Query:         q.Query,
		RequireTags:   append([]string(nil), q.RequireTags...),
		AnyTags:       append([]string(nil), q.AnyTags...),
		MinConfidence: q.MinConfidence,
		MaxConfidence: q.MaxConfidence,
		SourceID:      q.SourceID,
		MaxResults:    q.MaxResults,
		MaxScanRadius: q.MaxScanRadius,
	}
	if q.Center != nil {
		cfg.Center = hexxladb.Coord{Q: q.Center.Q, R: q.Center.R}
	}
	cfg.Radius = q.Radius
	if len(embedding) > 0 {
		cfg.Embedding = embedding
	}
	var out []domain.CellHit
	err := a.db.View(func(tx *hexxladb.Tx) error {
		res, err := tx.SearchCells(ctx, cfg)
		if err != nil {
			return fmt.Errorf("tx SearchCells: %w", err)
		}
		out = make([]domain.CellHit, 0, len(res))
		for i := range res {
			r := &res[i]
			out = append(out, viewToCellHit(&r.Cell, r.Score, ""))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("hexxlastore search cells: %w", err)
	}
	return out, nil
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
