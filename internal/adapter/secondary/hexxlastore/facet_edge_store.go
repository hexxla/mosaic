package hexxlastore

import (
	"context"
	"fmt"
	"time"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// FacetEdgeStoreAdapter implements [secondary.FacetEdgeStore].
type FacetEdgeStoreAdapter struct {
	live *LiveDB
}

// NewFacetEdgeStoreAdapter constructs the adapter (caller owns [LiveDB.Close]).
func NewFacetEdgeStoreAdapter(live *LiveDB) *FacetEdgeStoreAdapter {
	return &FacetEdgeStoreAdapter{live: live}
}

// PutFacet implements [secondary.FacetEdgeStore].
func (a *FacetEdgeStoreAdapter) PutFacet(ctx context.Context, cmd *domain.PutFacetCommand) error {
	if a == nil || a.live == nil {
		return fmt.Errorf("hexxlastore facet edge: nil database")
	}
	if cmd == nil {
		return fmt.Errorf("hexxlastore facet edge: nil command")
	}
	pk, err := hexxladb.Pack(hexxladb.Coord{Q: cmd.Coord.Q, R: cmd.Coord.R})
	if err != nil {
		return fmt.Errorf("hexxlastore put facet: %w", err)
	}
	now := time.Now().UTC().UnixNano()
	rec := hexxladb.NewFacetDerived(pk, cmd.FacetID, cmd.DerivedContent, now)
	updErr := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.Update(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			if err := tx.PutFacet(rec); err != nil {
				return fmt.Errorf("tx PutFacet: %w", err)
			}
			return nil
		})
	})
	if updErr != nil {
		return fmt.Errorf("hexxlastore put facet: %w", updErr)
	}
	return nil
}

// LinkCells implements [secondary.FacetEdgeStore].
func (a *FacetEdgeStoreAdapter) LinkCells(ctx context.Context, cmd *domain.LinkCellsCommand) error {
	if a == nil || a.live == nil {
		return fmt.Errorf("hexxlastore facet edge: nil database")
	}
	if cmd == nil {
		return fmt.Errorf("hexxlastore facet edge: nil command")
	}
	prov := hexxladb.NewProvenanceWire(cmd.SourceID, cmd.Confidence)
	updErr := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.Update(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			if err := tx.LinkCells(
				hexxladb.Coord{Q: cmd.From.Q, R: cmd.From.R},
				hexxladb.Coord{Q: cmd.To.Q, R: cmd.To.R},
				cmd.RelationType,
				cmd.Weight,
				prov,
			); err != nil {
				return fmt.Errorf("tx LinkCells: %w", err)
			}
			return nil
		})
	})
	if updErr != nil {
		return fmt.Errorf("hexxlastore link cells: %w", updErr)
	}
	return nil
}

// Ensure FacetEdgeStoreAdapter implements secondary.FacetEdgeStore.
var _ secondary.FacetEdgeStore = (*FacetEdgeStoreAdapter)(nil)
