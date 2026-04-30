package services

import (
	"context"
	"fmt"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/ports/primary"
	"github.com/sploitzberg/mosaic/internal/core/ports/secondary"
)

// TagCatalogService implements primary.TagBrowse by delegating to secondary.TagCatalog.
type TagCatalogService struct {
	catalog secondary.TagCatalog
}

// NewTagCatalogService constructs read-only tag discovery backed by Hexxla.
func NewTagCatalogService(c secondary.TagCatalog) *TagCatalogService {
	return &TagCatalogService{catalog: c}
}

// ListDistinctTags implements [primary.TagBrowse].
func (s *TagCatalogService) ListDistinctTags(ctx context.Context) (domain.DistinctTagsResult, error) {
	if s == nil || s.catalog == nil {
		return domain.DistinctTagsResult{}, fmt.Errorf("tag catalog: nil catalog")
	}
	out, err := s.catalog.ListDistinctTags(ctx)
	if err != nil {
		return domain.DistinctTagsResult{}, fmt.Errorf("tag catalog: list distinct tags: %w", err)
	}
	return out, nil
}

// TagCounts implements [primary.TagBrowse].
func (s *TagCatalogService) TagCounts(ctx context.Context) (domain.TagCountsResult, error) {
	if s == nil || s.catalog == nil {
		return domain.TagCountsResult{}, fmt.Errorf("tag catalog: nil catalog")
	}
	out, err := s.catalog.TagCounts(ctx)
	if err != nil {
		return domain.TagCountsResult{}, fmt.Errorf("tag catalog: tag counts: %w", err)
	}
	return out, nil
}

var _ primary.TagBrowse = (*TagCatalogService)(nil)
