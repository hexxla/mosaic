package secondary

import (
	"context"

	"github.com/sploitzberg/go-llm-project-structure/internal/core/domain"
)

// TagCatalog provides read-only tag discovery over visible MVCC cells.
type TagCatalog interface {
	ListDistinctTags(ctx context.Context) (domain.DistinctTagsResult, error)
	TagCounts(ctx context.Context) (domain.TagCountsResult, error)
}
