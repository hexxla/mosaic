package primary

import (
	"context"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

// TagBrowse exposes Hexxla tag discovery for MCP (reuse tags on writes; broaden search vocabulary).
type TagBrowse interface {
	ListDistinctTags(ctx context.Context) (domain.DistinctTagsResult, error)
	TagCounts(ctx context.Context) (domain.TagCountsResult, error)
}
