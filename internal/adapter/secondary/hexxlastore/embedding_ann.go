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
func (a *EmbeddingANNAdapter) Search(ctx context.Context, vector []float32, limit int, minScore float64) ([]domain.EmbeddingMatch, error) {
	if a == nil || a.live == nil {
		return nil, fmt.Errorf("hexxlastore embedding: nil database")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var matches []domain.EmbeddingMatch
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			hits, err := tx.SearchByEmbedding(vector, hexxladb.EmbeddingSearchConfig{
				MaxResults: limit,
				MinScore:   minScore,
			})
			if err != nil {
				return fmt.Errorf("search by embedding: %w", err)
			}
			matches = make([]domain.EmbeddingMatch, 0, len(hits))
			for _, hit := range hits {
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
				matches = append(matches, domain.EmbeddingMatch{
					Coord:      domain.AxialCoord{Q: ax.Q, R: ax.R},
					Score:      hit.Score,
					RawContent: rec.RawContent,
					Tags:       append([]string(nil), rec.Tags...),
					SourceID:   rec.Provenance.SourceID,
					Confidence: rec.Provenance.Confidence,
					CreatedAt:  nanoWallToRFC3339(rec.Provenance.CreatedAt),
					UpdatedAt:  nanoWallToRFC3339(rec.Provenance.UpdatedAt),
					ValidFrom:  nanoWallPtrToRFC3339(rec.Validity.ValidFrom),
					ValidTo:    nanoWallPtrToRFC3339(rec.Validity.ValidTo),
				})
			}
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("hexxlastore embedding search: %w", err)
	}
	return matches, nil
}

// Ensure EmbeddingANNAdapter implements secondary.EmbeddingANN.
var _ secondary.EmbeddingANN = (*EmbeddingANNAdapter)(nil)
