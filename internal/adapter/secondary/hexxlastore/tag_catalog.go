package hexxlastore

import (
	"context"
	"fmt"

	"github.com/hexxla/hexxladb"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// TagCatalogAdapter implements [secondary.TagCatalog] with read-only [hexxladb.DB.View] calls.
type TagCatalogAdapter struct {
	live *LiveDB
}

// NewTagCatalogAdapter wraps a [LiveDB] (caller owns [LiveDB.Close]).
func NewTagCatalogAdapter(live *LiveDB) *TagCatalogAdapter {
	return &TagCatalogAdapter{live: live}
}

// ListDistinctTags implements [secondary.TagCatalog] via Tx.ListExistingTopics.
func (a *TagCatalogAdapter) ListDistinctTags(ctx context.Context) (domain.DistinctTagsResult, error) {
	if a == nil || a.live == nil {
		return domain.DistinctTagsResult{}, fmt.Errorf("hexxlastore tag catalog: nil database")
	}
	var res domain.DistinctTagsResult
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			topics, inner := tx.ListExistingTopics(ctx)
			if inner != nil {
				return fmt.Errorf("tx ListExistingTopics: %w", inner)
			}
			res.Tags = append([]string(nil), topics...)
			return nil
		})
	})
	if err != nil {
		return domain.DistinctTagsResult{}, fmt.Errorf("hexxlastore list distinct tags: %w", err)
	}
	return res, nil
}

// TagCounts implements [secondary.TagCatalog] via Tx.TagCounts.
func (a *TagCatalogAdapter) TagCounts(ctx context.Context) (domain.TagCountsResult, error) {
	if a == nil || a.live == nil {
		return domain.TagCountsResult{}, fmt.Errorf("hexxlastore tag catalog: nil database")
	}
	var res domain.TagCountsResult
	err := a.live.WithRead(func(db *hexxladb.DB) error {
		return db.View(func(tx *hexxladb.Tx) error {
			counts, inner := tx.TagCounts(ctx)
			if inner != nil {
				return fmt.Errorf("tx TagCounts: %w", inner)
			}
			res.Counts = make([]domain.TagCountEntry, len(counts))
			for i := range counts {
				res.Counts[i] = domain.TagCountEntry{Tag: counts[i].Tag, Count: counts[i].Count}
			}
			return nil
		})
	})
	if err != nil {
		return domain.TagCountsResult{}, fmt.Errorf("hexxlastore tag counts: %w", err)
	}
	return res, nil
}

var _ secondary.TagCatalog = (*TagCatalogAdapter)(nil)
