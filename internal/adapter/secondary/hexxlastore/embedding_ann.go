package hexxlastore

import (
	"context"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
	ollamac "github.com/sploitzberg/mosaic/internal/ollama"
)

// EmbeddingANNAdapter implements [secondary.EmbeddingANN] using Ollama for query vectors and
// [hexxladb.Tx.SearchByEmbedding] for ANN retrieval, then [hexxladb.Tx.GetCell] for payloads.
type EmbeddingANNAdapter struct {
	live         *LiveDB
	ollamaClient *ollamac.Client
}

// NewEmbeddingANNAdapter wires a [LiveDB] and Ollama client (callers own HTTP + URL lifecycle).
func NewEmbeddingANNAdapter(live *LiveDB, oc *ollamac.Client) *EmbeddingANNAdapter {
	return &EmbeddingANNAdapter{live: live, ollamaClient: oc}
}

// Search implements [secondary.EmbeddingANN].
func (a *EmbeddingANNAdapter) Search(ctx context.Context, q domain.EmbeddingSearchQuery) (domain.EmbeddingSearchResponse, error) {
	if a == nil || a.live == nil {
		return domain.EmbeddingSearchResponse{}, fmt.Errorf("hexxlastore embedding: nil database")
	}
	if a.ollamaClient == nil {
		return domain.EmbeddingSearchResponse{}, fmt.Errorf("hexxlastore embedding: nil ollama client")
	}
	var dim uint16
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		dim = db.EmbeddingDimension()
		return nil
	})
	if err != nil {
		return domain.EmbeddingSearchResponse{}, fmt.Errorf("hexxlastore embedding: %w", err)
	}
	if dim == 0 {
		return domain.EmbeddingSearchResponse{}, fmt.Errorf("hexxlastore embedding: database has no configured embedding dimension")
	}
	vec, err := a.ollamaClient.Embed(ctx, q.Text, int(dim))
	if err != nil {
		return domain.EmbeddingSearchResponse{}, fmt.Errorf("query embed: %w", err)
	}
	limit := q.MaxResults
	if limit <= 0 {
		limit = 10
	}

	var matches []domain.EmbeddingMatch
	err = a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			hits, err := tx.SearchByEmbedding(vec, hexxladb.EmbeddingSearchConfig{
				MaxResults: limit,
				MinScore:   q.MinScore,
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
		return domain.EmbeddingSearchResponse{}, fmt.Errorf("hexxlastore embedding search: %w", err)
	}
	return domain.EmbeddingSearchResponse{
		Query:   q.Text,
		Matches: matches,
	}, nil
}

// Ensure EmbeddingANNAdapter implements secondary.EmbeddingANN.
var _ secondary.EmbeddingANN = (*EmbeddingANNAdapter)(nil)
