package hexxlastore

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// GetFacet implements [secondary.FacetEdgeReader] via Tx.GetFacet inside View.
func (a *FacetEdgeStoreAdapter) GetFacet(ctx context.Context, coord domain.AxialCoord, facetID uint8) (domain.GetFacetResponse, error) {
	if a == nil || a.live == nil {
		return domain.GetFacetResponse{}, fmt.Errorf("hexxlastore facet reads: nil database")
	}
	pk, err := hexxladb.Pack(hexxladb.Coord{Q: coord.Q, R: coord.R})
	if err != nil {
		return domain.GetFacetResponse{}, fmt.Errorf("hexxlastore get facet pack: %w", err)
	}
	var out domain.GetFacetResponse
	viewErr := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			rec, ok, ierr := tx.GetFacet(pk, facetID)
			if ierr != nil {
				return fmt.Errorf("tx GetFacet: %w", ierr)
			}
			out.Q, out.R = coord.Q, coord.R
			out.FacetID = facetID
			out.Found = ok
			if !ok {
				return nil
			}
			out.DerivedContent = rec.DerivedContent
			out.LastRotatedUnixNano = rec.LastRotated
			out.DerivationHashHex = hexDerivation(rec.DerivationHash)
			return nil
		})
	})
	if viewErr != nil {
		return domain.GetFacetResponse{}, fmt.Errorf("hexxlastore get facet: %w", viewErr)
	}
	return out, nil
}

// ListFacetsForCell implements [secondary.FacetEdgeReader].
func (a *FacetEdgeStoreAdapter) ListFacetsForCell(ctx context.Context, coord domain.AxialCoord) (domain.ListFacetsForCellResponse, error) {
	if a == nil || a.live == nil {
		return domain.ListFacetsForCellResponse{}, fmt.Errorf("hexxlastore facet reads: nil database")
	}
	pk, err := hexxladb.Pack(hexxladb.Coord{Q: coord.Q, R: coord.R})
	if err != nil {
		return domain.ListFacetsForCellResponse{}, fmt.Errorf("hexxlastore list facets pack: %w", err)
	}
	var bullets []domain.FacetSlotBullet
	viewErr := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			return tx.AscendFacetsForCell(pk, func(rec hexxladb.FacetWalkRecord) bool {
				bullets = append(bullets, domain.FacetSlotBullet{
					FacetID:             rec.FacetID,
					DerivedContent:      rec.DerivedContent,
					LastRotatedUnixNano: rec.LastRotated,
					DerivationHashHex:   hexDerivation(rec.DerivationHash),
				})
				return true
			})
		})
	})
	if viewErr != nil {
		return domain.ListFacetsForCellResponse{}, fmt.Errorf("hexxlastore list facets: %w", viewErr)
	}
	if bullets == nil {
		bullets = []domain.FacetSlotBullet{}
	}
	return domain.ListFacetsForCellResponse{Q: coord.Q, R: coord.R, Facets: bullets}, nil
}

// GetEdge implements [secondary.FacetEdgeReader].
func (a *FacetEdgeStoreAdapter) GetEdge(ctx context.Context, from, to domain.AxialCoord, relationType string) (domain.GetEdgeResponse, error) {
	if a == nil || a.live == nil {
		return domain.GetEdgeResponse{}, fmt.Errorf("hexxlastore facet reads: nil database")
	}
	fromPK, err := hexxladb.Pack(hexxladb.Coord{Q: from.Q, R: from.R})
	if err != nil {
		return domain.GetEdgeResponse{}, fmt.Errorf("hexxlastore get edge pack from: %w", err)
	}
	toPK, err := hexxladb.Pack(hexxladb.Coord{Q: to.Q, R: to.R})
	if err != nil {
		return domain.GetEdgeResponse{}, fmt.Errorf("hexxlastore get edge pack to: %w", err)
	}
	var out domain.GetEdgeResponse
	viewErr := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			rec, ok, ierr := tx.GetEdge(fromPK, toPK, relationType)
			if ierr != nil {
				return fmt.Errorf("tx GetEdge: %w", ierr)
			}
			out.FromQ, out.FromR = from.Q, from.R
			out.ToQ, out.ToR = to.Q, to.R
			out.RelationType = relationType
			out.Found = ok
			if !ok {
				return nil
			}
			out.Weight = rec.Weight
			out.SourceID = rec.Provenance.SourceID
			out.Confidence = rec.Provenance.Confidence
			out.CreatedAtNs = rec.Provenance.CreatedAt
			out.UpdatedAtNs = rec.Provenance.UpdatedAt
			return nil
		})
	})
	if viewErr != nil {
		return domain.GetEdgeResponse{}, fmt.Errorf("hexxlastore get edge: %w", viewErr)
	}
	return out, nil
}

// ListEdgesFrom implements [secondary.FacetEdgeReader].
func (a *FacetEdgeStoreAdapter) ListEdgesFrom(ctx context.Context, from domain.AxialCoord, maxEdges int) (domain.ListEdgesFromResponse, error) {
	if a == nil || a.live == nil {
		return domain.ListEdgesFromResponse{}, fmt.Errorf("hexxlastore facet reads: nil database")
	}
	fromPK, err := hexxladb.Pack(hexxladb.Coord{Q: from.Q, R: from.R})
	if err != nil {
		return domain.ListEdgesFromResponse{}, fmt.Errorf("hexxlastore list edges pack: %w", err)
	}
	var edges []domain.EdgeBullet
	truncated := false
	viewErr := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			return tx.AscendEdgesFrom(fromPK, func(rec hexxladb.EdgeWalkRecord) bool {
				if len(edges) >= maxEdges {
					truncated = true
					return false
				}
				toCoord, ierr := hexxladb.Unpack(rec.To)
				if ierr != nil {
					return false
				}
				edges = append(edges, domain.EdgeBullet{
					ToQ:          toCoord.Q,
					ToR:          toCoord.R,
					RelationType: rec.RelationType,
					Weight:       rec.Weight,
					SourceID:     rec.Provenance.SourceID,
					Confidence:   rec.Provenance.Confidence,
					CreatedAtNs:  rec.Provenance.CreatedAt,
					UpdatedAtNs:  rec.Provenance.UpdatedAt,
				})
				return true
			})
		})
	})
	if viewErr != nil {
		return domain.ListEdgesFromResponse{}, fmt.Errorf("hexxlastore list edges from: %w", viewErr)
	}
	if edges == nil {
		edges = []domain.EdgeBullet{}
	}
	return domain.ListEdgesFromResponse{
		FromQ:           from.Q,
		FromR:           from.R,
		Edges:           edges,
		MaxEdgesApplied: maxEdges,
		Truncated:       truncated,
	}, nil
}

func hexDerivation(h [32]byte) string {
	allZero := true
	for _, b := range h {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return ""
	}
	return hex.EncodeToString(h[:])
}

var _ secondary.FacetEdgeReader = (*FacetEdgeStoreAdapter)(nil)
