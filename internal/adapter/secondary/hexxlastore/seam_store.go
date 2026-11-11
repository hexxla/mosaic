package hexxlastore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// SeamStoreAdapter implements [secondary.SeamStore] against an open HexxlaDB handle.
type SeamStoreAdapter struct {
	live *LiveDB
}

// NewSeamStoreAdapter constructs a SeamStoreAdapter (caller owns [LiveDB.Close]).
func NewSeamStoreAdapter(live *LiveDB) *SeamStoreAdapter {
	return &SeamStoreAdapter{live: live}
}

// FindSeams implements [secondary.SeamStore].
func (a *SeamStoreAdapter) FindSeams(ctx context.Context, q *domain.FindSeamsQuery) ([]domain.SeamHit, error) {
	if a == nil || a.live == nil {
		return nil, fmt.Errorf("hexxlastore seam store: nil database")
	}
	if q == nil {
		return nil, fmt.Errorf("hexxlastore seam store: nil query")
	}
	var out []domain.SeamHit
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			recs, err := tx.FindSeams(ctx, hexxladb.Coord{Q: q.Center.Q, R: q.Center.R}, q.Radius, q.UnresolvedOnly)
			if err != nil {
				return fmt.Errorf("tx FindSeams: %w", err)
			}
			out = make([]domain.SeamHit, 0, len(recs))
			for i := range recs {
				r := recs[i]
				ca, err := hexxladb.Unpack(r.CellA)
				if err != nil {
					return fmt.Errorf("unpack seam cell_a: %w", err)
				}
				cb, err := hexxladb.Unpack(r.CellB)
				if err != nil {
					return fmt.Errorf("unpack seam cell_b: %w", err)
				}
				src := ""
				if r.Provenance.SourceID != "" {
					src = r.Provenance.SourceID
				}
				when := ""
				if r.DetectedAt > 0 {
					when = time.Unix(0, r.DetectedAt).UTC().Format(time.RFC3339Nano)
				}
				out = append(out, domain.SeamHit{
					ID:                r.ID,
					CellA:             domain.AxialCoord{Q: ca.Q, R: ca.R},
					CellB:             domain.AxialCoord{Q: cb.Q, R: cb.R},
					SeamType:          r.SeamType,
					Reason:            r.Reason,
					ConfidenceDelta:   r.ConfidenceDelta,
					DetectedAtRFC3339: when,
					ResolutionStatus:  r.ResolutionStatus,
					ResolutionNote:    r.ResolutionNote,
					SourceID:          src,
				})
			}
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("hexxlastore find seams: %w", err)
	}
	return out, nil
}

// MarkConflict implements [secondary.SeamStore].
func (a *SeamStoreAdapter) MarkConflict(ctx context.Context, cellA, cellB domain.AxialCoord, reason string) error {
	return a.updateSeam(ctx, "mark conflict", "MarkConflict", func(tx *hexxladb.Tx) error {
		return tx.MarkConflict(hexxladb.Coord(cellA), hexxladb.Coord(cellB), reason)
	})
}

// MarkSupersedes implements [secondary.SeamStore].
func (a *SeamStoreAdapter) MarkSupersedes(ctx context.Context, superseder, superseded domain.AxialCoord, reason string) error {
	return a.updateSeam(ctx, "mark supersedes", "MarkSupersedes", func(tx *hexxladb.Tx) error {
		return tx.MarkSupersedes(hexxladb.Coord(superseder), hexxladb.Coord(superseded), reason)
	})
}

// ResolveSeam implements [secondary.SeamStore].
func (a *SeamStoreAdapter) ResolveSeam(ctx context.Context, seamID, resolutionStatus, resolutionNote string) error {
	return a.updateSeam(ctx, "resolve seam", "ResolveSeam", func(tx *hexxladb.Tx) error {
		return tx.ResolveSeam(seamID, resolutionStatus, resolutionNote)
	})
}

func (a *SeamStoreAdapter) updateSeam(ctx context.Context, operation, method string, apply func(*hexxladb.Tx) error) error {
	if a == nil || a.live == nil {
		return fmt.Errorf("hexxlastore seam store: nil database")
	}
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		return updateMutation(db, func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			if err := apply(tx); err != nil {
				return fmt.Errorf("tx %s: %w", method, err)
			}
			return nil
		})
	})
	if err != nil && !errors.Is(err, hexxladb.ErrSeamNotFound) {
		return fmt.Errorf("hexxlastore %s: %w", operation, err)
	}
	return err
}

// Ensure SeamStoreAdapter implements secondary.SeamStore.
var _ secondary.SeamStore = (*SeamStoreAdapter)(nil)
