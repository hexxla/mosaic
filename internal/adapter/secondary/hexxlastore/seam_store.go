package hexxlastore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/ports/secondary"
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
			var recs []hexxladb.SeamRecord
			var err error
			if q.AsOf != nil {
				recs, err = tx.FindSeamsAt(ctx, hexxladb.Coord(q.Center), q.Radius, q.UnresolvedOnly, *q.AsOf)
			} else {
				recs, err = tx.FindSeams(ctx, hexxladb.Coord(q.Center), q.Radius, q.UnresolvedOnly)
			}
			if err != nil {
				return fmt.Errorf("tx FindSeams: %w", err)
			}
			out = make([]domain.SeamHit, 0, len(recs))
			for i := range recs {
				r := recs[i]
				if q.SourceID != "" && r.Provenance.SourceID != q.SourceID {
					continue
				}
				hit, err := seamHit(r)
				if err != nil {
					return err
				}
				out = append(out, hit)
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

// PutSeam stores the complete custom seam through the native typed API.
func (a *SeamStoreAdapter) PutSeam(ctx context.Context, q domain.PutSeamCommand) (domain.MutationOK, error) {
	aKey, err := hexxladb.Pack(hexxladb.Coord(q.CellA))
	if err != nil {
		return domain.MutationOK{}, err
	}
	bKey, err := hexxladb.Pack(hexxladb.Coord(q.CellB))
	if err != nil {
		return domain.MutationOK{}, err
	}
	rec := hexxladb.SeamRecord{ID: q.ID, CellA: aKey, CellB: bKey, SeamType: q.SeamType, Reason: q.Reason, ConfidenceDelta: q.ConfidenceDelta, ResolutionStatus: q.ResolutionStatus, ResolutionNote: q.ResolutionNote, Provenance: hexxladb.NewProvenanceWire(q.SourceID, q.Confidence)}
	if q.CreatedAt != nil {
		rec.Provenance.CreatedAt = q.CreatedAt.UnixNano()
	}
	if q.UpdatedAt != nil {
		rec.Provenance.UpdatedAt = q.UpdatedAt.UnixNano()
	}
	if q.DetectedAt != nil {
		rec.DetectedAt = q.DetectedAt.UnixNano()
	} else {
		rec.DetectedAt = time.Now().UTC().UnixNano()
	}
	if q.ValidFrom != nil {
		rec.Validity.ValidFrom = new(q.ValidFrom.UnixNano())
	}
	if q.ValidTo != nil {
		rec.Validity.ValidTo = new(q.ValidTo.UnixNano())
	}
	err = a.live.WithRead(func(db *hexxladb.DB) error {
		return updateMutation(db, func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return tx.PutSeam(ctx, rec)
		})
	})
	return domain.MutationOK{OK: err == nil}, err
}
