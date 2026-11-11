package hexxlastore

import (
	"context"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// EmbeddingANNAdapter implements [secondary.EmbeddingANN] with
// [hexxladb.Tx.SearchByEmbedding] for ANN retrieval, then [hexxladb.Tx.GetCell] for payloads.
type EmbeddingANNAdapter struct {
	live *LiveDB
}

// NewEmbeddingANNAdapter wires the vector index to a [LiveDB].
func NewEmbeddingANNAdapter(live *LiveDB) *EmbeddingANNAdapter {
	return &EmbeddingANNAdapter{live: live}
}

// Search implements [secondary.EmbeddingANN].
func (a *EmbeddingANNAdapter) Search(ctx context.Context, vector []float32, limit int, minScore float64) ([]domain.CellHit, error) {
	out, err := a.SearchWithStats(ctx, vector, limit, minScore, 0)
	return out.Hits, err
}

// SearchWithStats returns native execution path and effective ANN breadth.
func (a *EmbeddingANNAdapter) SearchWithStats(ctx context.Context, vector []float32, limit int, minScore float64, efSearch int) (domain.CellHitsResponse, error) {
	if a == nil || a.live == nil {
		return domain.CellHitsResponse{}, fmt.Errorf("hexxlastore embedding: nil database")
	}
	if err := ctx.Err(); err != nil {
		return domain.CellHitsResponse{}, err
	}

	var matches []domain.CellHit
	var stats hexxladb.EmbeddingSearchStats
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			hits, searchStats, err := tx.SearchByEmbeddingWithStats(ctx, vector, hexxladb.EmbeddingSearchConfig{
				MaxResults: limit,
				EfSearch:   efSearch,
				MinScore:   minScore,
			})
			if err != nil {
				return fmt.Errorf("search by embedding: %w", err)
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			stats = searchStats
			matches = make([]domain.CellHit, 0, len(hits))
			for _, hit := range hits {
				if err := ctx.Err(); err != nil {
					return err
				}
				ax, err := hexxladb.Unpack(hit.Coord)
				if err != nil {
					return fmt.Errorf("unpack hit coord: %w", err)
				}
				rec, ok, err := tx.GetCell(hit.Coord)
				if err != nil {
					return fmt.Errorf("get cell: %w", err)
				}
				if !ok {
					continue
				}
				var hint *hexxladb.Coord
				if rec.ClusterHint != nil {
					coord, err := hexxladb.Unpack(*rec.ClusterHint)
					if err != nil {
						return err
					}
					hint = &coord
				}
				matches = append(matches, domain.CellHit{
					ClusterHint: domainCoordPtr(hint),
					Coord:       domain.AxialCoord{Q: ax.Q, R: ax.R},
					Score:       hit.Score,
					RawContent:  rec.RawContent,
					Tags:        append([]string(nil), rec.Tags...),
					SourceID:    rec.Provenance.SourceID,
					Confidence:  rec.Provenance.Confidence,
					CreatedAt:   nanoWallToRFC3339(rec.Provenance.CreatedAt),
					UpdatedAt:   nanoWallToRFC3339(rec.Provenance.UpdatedAt),
					ValidFrom:   nanoWallPtrToRFC3339(rec.Validity.ValidFrom),
					ValidTo:     nanoWallPtrToRFC3339(rec.Validity.ValidTo),
				})
			}
			return ctx.Err()
		})
	})
	if err != nil {
		return domain.CellHitsResponse{}, fmt.Errorf("hexxlastore embedding search: %w", err)
	}
	return domain.CellHitsResponse{Hits: matches, EmbeddingPath: string(stats.Path), EfSearch: stats.EfSearch}, nil
}

// Ensure EmbeddingANNAdapter implements secondary.EmbeddingANN.
var _ secondary.EmbeddingANN = (*EmbeddingANNAdapter)(nil)
