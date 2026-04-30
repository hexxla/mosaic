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
	if a == nil || a.live == nil {
		return fmt.Errorf("hexxlastore seam store: nil database")
	}
	updErr := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.Update(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			if err := tx.MarkConflict(
				hexxladb.Coord{Q: cellA.Q, R: cellA.R},
				hexxladb.Coord{Q: cellB.Q, R: cellB.R},
				reason,
			); err != nil {
				return fmt.Errorf("tx MarkConflict: %w", err)
			}
			return nil
		})
	})
	if updErr != nil {
		return fmt.Errorf("hexxlastore mark conflict: %w", updErr)
	}
	return nil
}

// MarkSupersedes implements [secondary.SeamStore].
func (a *SeamStoreAdapter) MarkSupersedes(ctx context.Context, superseder, superseded domain.AxialCoord, reason string) error {
	if a == nil || a.live == nil {
		return fmt.Errorf("hexxlastore seam store: nil database")
	}
	updErr := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.Update(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			if err := tx.MarkSupersedes(
				hexxladb.Coord{Q: superseder.Q, R: superseder.R},
				hexxladb.Coord{Q: superseded.Q, R: superseded.R},
				reason,
			); err != nil {
				return fmt.Errorf("tx MarkSupersedes: %w", err)
			}
			return nil
		})
	})
	if updErr != nil {
		return fmt.Errorf("hexxlastore mark supersedes: %w", updErr)
	}
	return nil
}

// ResolveSeam implements [secondary.SeamStore].
func (a *SeamStoreAdapter) ResolveSeam(ctx context.Context, seamID, resolutionStatus, resolutionNote string) error {
	if a == nil || a.live == nil {
		return fmt.Errorf("hexxlastore seam store: nil database")
	}
	updErr := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.Update(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			if err := tx.ResolveSeam(seamID, resolutionStatus, resolutionNote); err != nil {
				return fmt.Errorf("tx ResolveSeam: %w", err)
			}
			return nil
		})
	})
	if updErr != nil {
		if errors.Is(updErr, hexxladb.ErrSeamNotFound) {
			return updErr
		}
		return fmt.Errorf("hexxlastore resolve seam: %w", updErr)
	}
	return nil
}

// Ensure SeamStoreAdapter implements secondary.SeamStore.
var _ secondary.SeamStore = (*SeamStoreAdapter)(nil)
